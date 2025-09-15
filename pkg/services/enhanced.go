package services

import (
    "context"
    "encoding/json"
    "sync"

    "github.com/darksworm/argonaut/pkg/api"
    apperrors "github.com/darksworm/argonaut/pkg/errors"
    appcontext "github.com/darksworm/argonaut/pkg/context"
    "github.com/darksworm/argonaut/pkg/model"
    "github.com/darksworm/argonaut/pkg/retry"
    cblog "github.com/charmbracelet/log"
)

// EnhancedArgoApiService provides enhanced ArgoApiService with recovery and degradation
type EnhancedArgoApiService struct {
	appService      *api.ApplicationService
	watchCancel     context.CancelFunc
	mu              sync.RWMutex
	recoveryManager *StreamRecoveryManager
	degradationMgr  *GracefulDegradationManager
}

// NewEnhancedArgoApiService creates a new enhanced ArgoApiService implementation
func NewEnhancedArgoApiService(server *model.Server) *EnhancedArgoApiService {
	impl := &EnhancedArgoApiService{
		recoveryManager: NewStreamRecoveryManager(DefaultStreamRecoveryConfig),
		degradationMgr:  NewGracefulDegradationManager(),
	}
	if server != nil {
		impl.appService = api.NewApplicationService(server)
	}

	// Register degradation callback
    impl.degradationMgr.RegisterCallback(func(oldMode, newMode DegradationMode) {
        cblog.With("component", "services").Info("Service degradation mode changed", "from", oldMode, "to", newMode)
    })

	return impl
}

// restartWatch is the recovery function for watch streams
func (s *EnhancedArgoApiService) restartWatch(ctx context.Context, server *model.Server, eventChan chan<- ArgoApiEvent) error {
	// This is a simplified restart - in a real implementation, this would
	// re-establish the watch connection
	apps, err := s.ListApplications(ctx, server)
	if err != nil {
		return err
	}

	eventChan <- ArgoApiEvent{
		Type: "apps-loaded",
		Apps: apps,
	}

	return nil
}

// ListApplications implements ArgoApiService.ListApplications with degradation support
func (s *EnhancedArgoApiService) ListApplications(ctx context.Context, server *model.Server) ([]model.App, error) {
	if server == nil {
		return nil, apperrors.ConfigError("SERVER_MISSING",
			"Server configuration is required").
			WithUserAction("Please run 'argocd login' to configure the server")
	}

	// Check if operation is allowed in current degradation mode
	if allowed, err := s.degradationMgr.CanPerformOperation("ListApplications"); !allowed {
		// Try to serve from cache in offline mode
		if s.degradationMgr.GetCurrentMode() == DegradationOffline {
			if cachedApps, found := s.degradationMgr.GetCachedApps(); found {
				return cachedApps, nil
			}
		}
		return nil, err
	}

	// Use the real API service with resource timeout
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	ctx, cancel := appcontext.WithResourceTimeout(ctx)
	defer cancel()

	// Use retry mechanism for network operations
	var apps []model.App
	err := retry.RetryAPIOperation(ctx, "ListApplications", func(attempt int) error {
		var opErr error
		apps, opErr = s.appService.ListApplications(ctx)
		return opErr
	})

	if err != nil {
		// Report health status
		s.degradationMgr.ReportAPIHealth(false, err)

		// Convert API errors to structured format if needed
		if argErr, ok := err.(*apperrors.ArgonautError); ok {
			return nil, argErr.WithContext("operation", "ListApplications")
		}

		return nil, apperrors.Wrap(err, apperrors.ErrorAPI, "LIST_APPS_FAILED",
			"Failed to list applications").
			WithContext("server", server.BaseURL).
			AsRecoverable().
			WithUserAction("Check your ArgoCD server connection and try again")
	}

	// Report successful operation and update cache
	s.degradationMgr.ReportAPIHealth(true, nil)
	s.degradationMgr.UpdateCache(apps, server, "")

	return apps, nil
}

// WatchApplications implements ArgoApiService.WatchApplications with stream recovery
func (s *EnhancedArgoApiService) WatchApplications(ctx context.Context, server *model.Server) (<-chan ArgoApiEvent, func(), error) {
	if server == nil {
		return nil, nil, apperrors.ConfigError("SERVER_MISSING",
			"Server configuration is required").
			WithUserAction("Please run 'argocd login' to configure the server")
	}

	// Use the real API service
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	eventChan := make(chan ArgoApiEvent, 100)
	watchCtx, cancel := appcontext.WithCancel(ctx) // No timeout for streams
	s.mu.Lock()
	s.watchCancel = cancel
	s.mu.Unlock()

	// Register stream for recovery
	streamID := "watch-applications-" + server.BaseURL
	_ = s.recoveryManager.RegisterStream(streamID, server, func(recoveryCtx context.Context) error {
		// Recovery function - restart the watch
		return s.restartWatch(recoveryCtx, server, eventChan)
	})

	// Start watching in a goroutine
	go func() {
		defer close(eventChan)
		defer s.recoveryManager.UnregisterStream(streamID)

		// Send initial status
		eventChan <- ArgoApiEvent{
			Type:   "status-change",
			Status: "Loading…",
		}

		// Send initial apps loaded event (no retry for initial watch load to avoid delays)
		apps, err := s.ListApplications(watchCtx, server)
		if err != nil {
			if isAuthError(err.Error()) {
				s.degradationMgr.ReportAuthHealth(false, err)
				eventChan <- ArgoApiEvent{
					Type:  "auth-error",
					Error: err,
				}
				eventChan <- ArgoApiEvent{
					Type:   "status-change",
					Status: "Auth required",
				}
				s.recoveryManager.ReportStreamFailure(streamID, err)
				return
			}
			s.degradationMgr.ReportAPIHealth(false, err)
			eventChan <- ArgoApiEvent{
				Type:  "api-error",
				Error: err,
			}
			eventChan <- ArgoApiEvent{
				Type:   "status-change",
				Status: "Error: " + err.Error(),
			}
			s.recoveryManager.ReportStreamFailure(streamID, err)
			return
		}

		// Report healthy stream
		s.recoveryManager.ReportStreamHealthy(streamID)

		eventChan <- ArgoApiEvent{
			Type: "apps-loaded",
			Apps: apps,
		}

		eventChan <- ArgoApiEvent{
			Type:   "status-change",
			Status: "Live",
		}

		// Start real watch stream
		s.startWatchStreamWithRecovery(watchCtx, eventChan, streamID)
	}()

	cleanup := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.watchCancel != nil {
			s.watchCancel()
			s.watchCancel = nil
		}
		// Unregister stream from recovery
		s.recoveryManager.UnregisterStream(streamID)
	}

	return eventChan, cleanup, nil
}

// SyncApplication implements ArgoApiService.SyncApplication with degradation check
func (s *EnhancedArgoApiService) SyncApplication(ctx context.Context, server *model.Server, appName string, prune bool) error {
	if server == nil {
		return apperrors.ConfigError("SERVER_MISSING",
			"Server configuration is required").
			WithUserAction("Please run 'argocd login' to configure the server")
	}
	if appName == "" {
		return apperrors.ValidationError("APP_NAME_MISSING",
			"Application name is required").
			WithUserAction("Specify an application name for the sync operation")
	}

	// Check if operation is allowed in current degradation mode
	if allowed, err := s.degradationMgr.CanPerformOperation("SyncApplication"); !allowed {
		return err
	}

	// Use the real API service with sync timeout
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	ctx, cancel := appcontext.WithSyncTimeout(ctx)
	defer cancel()

	opts := &api.SyncOptions{
		Prune: prune,
	}

	// Use retry mechanism for sync operations
	err := retry.RetryAPIOperation(ctx, "SyncApplication", func(attempt int) error {
		return s.appService.SyncApplication(ctx, appName, opts)
	})

	if err != nil {
		// Report API health status
		s.degradationMgr.ReportAPIHealth(false, err)

		// Convert API errors to structured format if needed
		if argErr, ok := err.(*apperrors.ArgonautError); ok {
			return argErr.WithContext("operation", "SyncApplication").
				WithContext("appName", appName).
				WithContext("prune", prune)
		}

		return apperrors.Wrap(err, apperrors.ErrorAPI, "SYNC_FAILED",
			"Failed to sync application").
			WithContext("server", server.BaseURL).
			WithContext("appName", appName).
			WithContext("prune", prune).
			AsRecoverable().
			WithUserAction("Check the application status and try syncing again")
	}

	// Report successful operation
	s.degradationMgr.ReportAPIHealth(true, nil)
	return nil
}

// GetResourceDiffs implements ArgoApiService.GetResourceDiffs
func (s *EnhancedArgoApiService) GetResourceDiffs(ctx context.Context, server *model.Server, appName string) ([]ResourceDiff, error) {
	if server == nil {
		return nil, apperrors.ConfigError("SERVER_MISSING",
			"Server configuration is required").
			WithUserAction("Please run 'argocd login' to configure the server")
	}
	if appName == "" {
		return nil, apperrors.ValidationError("APP_NAME_MISSING",
			"Application name is required").
			WithUserAction("Specify an application name to get resource diffs")
	}

	// Use the real API service
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	// Use retry mechanism for API calls
	var diffs []api.ManagedResourceDiff
	err := retry.RetryAPIOperation(ctx, "GetManagedResourceDiffs", func(attempt int) error {
		var opErr error
		diffs, opErr = s.appService.GetManagedResourceDiffs(ctx, appName)
		return opErr
	})
	if err != nil {
		if argErr, ok := err.(*apperrors.ArgonautError); ok {
			return nil, argErr.WithContext("operation", "GetManagedResourceDiffs").
				WithContext("appName", appName)
		}
		return nil, apperrors.Wrap(err, apperrors.ErrorAPI, "GET_DIFFS_FAILED",
			"Failed to get resource diffs").
			WithContext("appName", appName).
			AsRecoverable().
			WithUserAction("Check the application exists and try again")
	}
	// Map to service layer struct
	out := make([]ResourceDiff, len(diffs))
	for i, d := range diffs {
		out[i] = ResourceDiff{
			Kind: d.Kind, Name: d.Name, Namespace: d.Namespace,
			LiveState: d.LiveState, TargetState: d.TargetState,
		}
	}
	return out, nil
}

// GetAPIVersion fetches /api/version and returns a version string
func (s *EnhancedArgoApiService) GetAPIVersion(ctx context.Context, server *model.Server) (string, error) {
	if server == nil {
		return "", apperrors.ConfigError("SERVER_MISSING",
			"Server configuration is required").
			WithUserAction("Please run 'argocd login' to configure the server")
	}
	client := api.NewClient(server)
	var data []byte
	err := retry.RetryAPIOperation(ctx, "GetAPIVersion", func(attempt int) error {
		var opErr error
		data, opErr = client.Get(ctx, "/api/version")
		return opErr
	})
	if err != nil {
		if argErr, ok := err.(*apperrors.ArgonautError); ok {
			return "", argErr.WithContext("operation", "GetAPIVersion")
		}
		return "", apperrors.Wrap(err, apperrors.ErrorAPI, "GET_VERSION_FAILED",
			"Failed to get API version").
			WithContext("server", server.BaseURL).
			AsRecoverable().
			WithUserAction("Check ArgoCD server connectivity")
	}
	// Accept {Version:"..."} or {version:"..."}
	var anyMap map[string]interface{}
	if err := json.Unmarshal(data, &anyMap); err == nil {
		if v, ok := anyMap["Version"].(string); ok && v != "" {
			return v, nil
		}
		if v, ok := anyMap["version"].(string); ok && v != "" {
			return v, nil
		}
	}
	return string(data), nil
}

// Cleanup implements ArgoApiService.Cleanup
func (s *EnhancedArgoApiService) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.watchCancel != nil {
		s.watchCancel()
		s.watchCancel = nil
	}

	// Shutdown recovery and degradation managers
	if s.recoveryManager != nil {
		s.recoveryManager.Shutdown()
	}
	if s.degradationMgr != nil {
		s.degradationMgr.Shutdown()
	}
}

// startWatchStreamWithRecovery starts the application watch stream with recovery support
func (s *EnhancedArgoApiService) startWatchStreamWithRecovery(ctx context.Context, eventChan chan<- ArgoApiEvent, streamID string) {
	watchEventChan := make(chan api.ApplicationWatchEvent, 100)

	go func() {
		defer close(watchEventChan)
		err := s.appService.WatchApplications(ctx, watchEventChan)
		if err != nil && ctx.Err() == nil {
            cblog.With("component", "services").Error("Watch stream error", "err", err)
			s.recoveryManager.ReportStreamFailure(streamID, err)
			s.degradationMgr.ReportAPIHealth(false, err)
			eventChan <- ArgoApiEvent{
				Type:  "api-error",
				Error: err,
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watchEventChan:
			if !ok {
				return
			}
			// Report stream activity
			s.recoveryManager.ReportStreamHealthy(streamID)
			s.handleWatchEvent(event, eventChan)
		}
	}
}

// handleWatchEvent processes watch events from the API stream
func (s *EnhancedArgoApiService) handleWatchEvent(event api.ApplicationWatchEvent, eventChan chan<- ArgoApiEvent) {
	appName := event.Application.Metadata.Name
	if appName == "" {
		return
	}

	switch event.Type {
	case "DELETED":
		eventChan <- ArgoApiEvent{
			Type:    "app-deleted",
			AppName: appName,
		}
	default:
		// Convert to our model
		app := s.appService.ConvertToApp(event.Application)
		eventChan <- ArgoApiEvent{
			Type: "app-updated",
			App:  &app,
		}
	}
}

// GetApplication fetches a single application with full details including history
func (s *EnhancedArgoApiService) GetApplication(ctx context.Context, server *model.Server, appName string, appNamespace *string) (*api.ArgoApplication, error) {
	return s.appService.GetApplication(ctx, appName, appNamespace)
}

// GetRevisionMetadata fetches git metadata for a specific revision
func (s *EnhancedArgoApiService) GetRevisionMetadata(ctx context.Context, server *model.Server, appName string, revision string, appNamespace *string) (*model.RevisionMetadata, error) {
	return s.appService.GetRevisionMetadata(ctx, appName, revision, appNamespace)
}

// RollbackApplication performs a rollback operation
func (s *EnhancedArgoApiService) RollbackApplication(ctx context.Context, server *model.Server, request model.RollbackRequest) error {
	return s.appService.RollbackApplication(ctx, request)
}

// GetRecoveryStats returns recovery statistics
func (s *EnhancedArgoApiService) GetRecoveryStats() StreamRecoveryStats {
	return s.recoveryManager.GetRecoveryStats()
}

// GetServiceHealth returns service health status
func (s *EnhancedArgoApiService) GetServiceHealth() ServiceHealth {
	return s.degradationMgr.GetServiceHealth()
}

// GetDegradationSummary returns a human-readable degradation summary
func (s *EnhancedArgoApiService) GetDegradationSummary() string {
	return s.degradationMgr.GetDegradationSummary()
}

// isAuthError function already exists in argo.go, reusing it
