package services

import (
	"context"
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/darksworm/argonaut/pkg/api"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	appcontext "github.com/darksworm/argonaut/pkg/context"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/retry"
)

// ArgoApiService interface defines operations for interacting with ArgoCD API
type ArgoApiService interface {
	// ListApplications retrieves all applications from ArgoCD
	ListApplications(ctx context.Context, server *model.Server) ([]model.App, error)

	// WatchApplications starts watching for application changes
	// Returns a channel for events and a cleanup function
	WatchApplications(ctx context.Context, server *model.Server) (<-chan ArgoApiEvent, func(), error)

	// SyncApplication syncs a specific application
	SyncApplication(ctx context.Context, server *model.Server, appName string, prune bool) error

	// GetResourceDiffs gets resource diffs for an application
    GetResourceDiffs(ctx context.Context, server *model.Server, appName string) ([]ResourceDiff, error)

    // GetAPIVersion fetches the ArgoCD API server version string
    GetAPIVersion(ctx context.Context, server *model.Server) (string, error)

    // GetApplication fetches a single application with full details including history
    GetApplication(ctx context.Context, server *model.Server, appName string, appNamespace *string) (*api.ArgoApplication, error)

    // GetRevisionMetadata fetches git metadata for a specific revision
    GetRevisionMetadata(ctx context.Context, server *model.Server, appName string, revision string, appNamespace *string) (*model.RevisionMetadata, error)

    // RollbackApplication performs a rollback operation
    RollbackApplication(ctx context.Context, server *model.Server, request model.RollbackRequest) error

    // GetResourceTree fetches the resource tree for an application
    GetResourceTree(ctx context.Context, server *model.Server, appName string, appNamespace string) (*api.ResourceTree, error)

    // WatchResourceTree streams resource tree updates for an application
    WatchResourceTree(ctx context.Context, server *model.Server, appName string, appNamespace string) (<-chan *api.ResourceTree, func(), error)

    // Cleanup stops all watchers and cleans up resources
    Cleanup()
}

// ArgoApiEvent represents events from the ArgoCD API
type ArgoApiEvent struct {
	Type    string       `json:"type"`
	Apps    []model.App  `json:"apps,omitempty"`
	App     *model.App   `json:"app,omitempty"`
	AppName string       `json:"appName,omitempty"`
	Error   error        `json:"error,omitempty"`
	Status  string       `json:"status,omitempty"`
}

// ResourceDiff represents a resource difference
type ResourceDiff struct {
    Kind       string `json:"kind"`
    Name       string `json:"name"`
    Namespace  string `json:"namespace"`
    LiveState  string `json:"liveState,omitempty"`
    TargetState string `json:"targetState,omitempty"`
}

// ArgoApiServiceImpl provides a concrete implementation of ArgoApiService
type ArgoApiServiceImpl struct {
	appService  *api.ApplicationService
	watchCancel context.CancelFunc
	mu          sync.RWMutex
}

// NewArgoApiService creates a new ArgoApiService implementation
func NewArgoApiService(server *model.Server) ArgoApiService {
	impl := &ArgoApiServiceImpl{}
	if server != nil {
		impl.appService = api.NewApplicationService(server)
	}
	return impl
}

// ListApplications implements ArgoApiService.ListApplications
func (s *ArgoApiServiceImpl) ListApplications(ctx context.Context, server *model.Server) ([]model.App, error) {
	if server == nil {
		return nil, apperrors.ConfigError("SERVER_MISSING",
			"Server configuration is required").
			WithUserAction("Please run 'argocd login' to configure the server")
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

	return apps, nil
}

// WatchApplications implements ArgoApiService.WatchApplications
func (s *ArgoApiServiceImpl) WatchApplications(ctx context.Context, server *model.Server) (<-chan ArgoApiEvent, func(), error) {
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

	// Start watching in a goroutine
	go func() {
		defer close(eventChan)

		// Send initial status
		eventChan <- ArgoApiEvent{
			Type:   "status-change",
			Status: "Loading…",
		}

		// Send initial apps loaded event (no retry for initial watch load to avoid delays)
        apps, err := s.ListApplications(watchCtx, server)
        if err != nil {
            // Prefer structured error inspection
            if argErr, ok := err.(*apperrors.ArgonautError); ok {
                if argErr.IsCategory(apperrors.ErrorAuth) {
                    eventChan <- ArgoApiEvent{Type: "auth-error", Error: err}
                    eventChan <- ArgoApiEvent{Type: "status-change", Status: "Auth required"}
                    return
                }
            }
            // Fallback string check
            if isAuthError(err.Error()) {
                eventChan <- ArgoApiEvent{Type: "auth-error", Error: err}
                eventChan <- ArgoApiEvent{Type: "status-change", Status: "Auth required"}
                return
            }
            eventChan <- ArgoApiEvent{Type: "api-error", Error: err}
            eventChan <- ArgoApiEvent{Type: "status-change", Status: "Error: " + err.Error()}
            return
        }

		eventChan <- ArgoApiEvent{
			Type: "apps-loaded",
			Apps: apps,
		}

		eventChan <- ArgoApiEvent{
			Type:   "status-change",
			Status: "Live",
		}

		// Start real watch stream
		s.startWatchStream(watchCtx, eventChan)
	}()

	cleanup := func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.watchCancel != nil {
			s.watchCancel()
			s.watchCancel = nil
		}
	}

	return eventChan, cleanup, nil
}

// SyncApplication implements ArgoApiService.SyncApplication
func (s *ArgoApiServiceImpl) SyncApplication(ctx context.Context, server *model.Server, appName string, prune bool) error {
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

	return nil
}

// GetResourceDiffs implements ArgoApiService.GetResourceDiffs
func (s *ArgoApiServiceImpl) GetResourceDiffs(ctx context.Context, server *model.Server, appName string) ([]ResourceDiff, error) {
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
func (s *ArgoApiServiceImpl) GetAPIVersion(ctx context.Context, server *model.Server) (string, error) {
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
        if v, ok := anyMap["Version"].(string); ok && v != "" { return v, nil }
        if v, ok := anyMap["version"].(string); ok && v != "" { return v, nil }
    }
    return string(data), nil
}

// Cleanup implements ArgoApiService.Cleanup
func (s *ArgoApiServiceImpl) Cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.watchCancel != nil {
		s.watchCancel()
		s.watchCancel = nil
	}
}

// startWatchStream starts the application watch stream
func (s *ArgoApiServiceImpl) startWatchStream(ctx context.Context, eventChan chan<- ArgoApiEvent) {
    watchEventChan := make(chan api.ApplicationWatchEvent, 100)
    
    go func() {
        defer close(watchEventChan)
        err := s.appService.WatchApplications(ctx, watchEventChan)
        if err != nil && ctx.Err() == nil {
            log.Printf("Watch stream error: %v", err)
            // Map auth-related errors to a dedicated event so the TUI can switch to auth-required
            if isAuthError(err.Error()) {
                eventChan <- ArgoApiEvent{
                    Type:  "auth-error",
                    Error: err,
                }
                eventChan <- ArgoApiEvent{Type: "status-change", Status: "Auth required"}
            } else {
                eventChan <- ArgoApiEvent{
                    Type:  "api-error",
                    Error: err,
                }
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
			s.handleWatchEvent(event, eventChan)
		}
	}
}

// handleWatchEvent processes watch events from the API stream
func (s *ArgoApiServiceImpl) handleWatchEvent(event api.ApplicationWatchEvent, eventChan chan<- ArgoApiEvent) {
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
func (s *ArgoApiServiceImpl) GetApplication(ctx context.Context, server *model.Server, appName string, appNamespace *string) (*api.ArgoApplication, error) {
	return s.appService.GetApplication(ctx, appName, appNamespace)
}

// GetRevisionMetadata fetches git metadata for a specific revision
func (s *ArgoApiServiceImpl) GetRevisionMetadata(ctx context.Context, server *model.Server, appName string, revision string, appNamespace *string) (*model.RevisionMetadata, error) {
	return s.appService.GetRevisionMetadata(ctx, appName, revision, appNamespace)
}

// RollbackApplication performs a rollback operation
func (s *ArgoApiServiceImpl) RollbackApplication(ctx context.Context, server *model.Server, request model.RollbackRequest) error {
    return s.appService.RollbackApplication(ctx, request)
}

// GetResourceTree implements ArgoApiService.GetResourceTree
func (s *ArgoApiServiceImpl) GetResourceTree(ctx context.Context, server *model.Server, appName string, appNamespace string) (*api.ResourceTree, error) {
    if server == nil {
        return nil, apperrors.ConfigError("SERVER_MISSING",
            "Server configuration is required").
            WithUserAction("Please run 'argocd login' to configure the server")
    }
    if appName == "" {
        return nil, apperrors.ValidationError("APP_NAME_MISSING",
            "Application name is required").
            WithUserAction("Specify an application name to load the resource tree")
    }

    if s.appService == nil {
        s.appService = api.NewApplicationService(server)
    }

    // Respect resource timeout
    ctx, cancel := appcontext.WithResourceTimeout(ctx)
    defer cancel()

    var tree *api.ResourceTree
    err := retry.RetryAPIOperation(ctx, "GetResourceTree", func(attempt int) error {
        var opErr error
        tree, opErr = s.appService.GetResourceTree(ctx, appName, appNamespace)
        return opErr
    })
    if err != nil {
        if argErr, ok := err.(*apperrors.ArgonautError); ok {
            return nil, argErr.WithContext("operation", "GetResourceTree").
                WithContext("appName", appName).
                WithContext("appNamespace", appNamespace)
        }
        return nil, apperrors.Wrap(err, apperrors.ErrorAPI, "GET_RESOURCE_TREE_FAILED",
            "Failed to load resource tree").
            WithContext("appName", appName).
            WithContext("appNamespace", appNamespace).
            AsRecoverable().
            WithUserAction("Check the application exists and try again")
    }
    return tree, nil
}

// WatchResourceTree implements ArgoApiService.WatchResourceTree
func (s *ArgoApiServiceImpl) WatchResourceTree(ctx context.Context, server *model.Server, appName string, appNamespace string) (<-chan *api.ResourceTree, func(), error) {
    if server == nil {
        return nil, nil, apperrors.ConfigError("SERVER_MISSING",
            "Server configuration is required").
            WithUserAction("Please run 'argocd login' to configure the server")
    }
    if appName == "" {
        return nil, nil, apperrors.ValidationError("APP_NAME_MISSING",
            "Application name is required").
            WithUserAction("Specify an application name to watch the resource tree")
    }
    if s.appService == nil { s.appService = api.NewApplicationService(server) }

    out := make(chan *api.ResourceTree, 32)
    watchCtx, cancel := appcontext.WithCancel(ctx)

    go func() {
        defer close(out)
        // internal channel of plain ResourceTree values from api
        ch := make(chan api.ResourceTree, 32)
        go func() {
            defer close(ch)
            _ = s.appService.WatchResourceTree(watchCtx, appName, appNamespace, ch)
        }()
        for {
            select {
            case <-watchCtx.Done():
                return
            case t, ok := <-ch:
                if !ok { return }
                // copy to heap pointer
                tt := t
                out <- &tt
            }
        }
    }()

    cleanup := func() { cancel() }
    return out, cleanup, nil
}

// isAuthError checks if an error indicates authentication issues
func isAuthError(errMsg string) bool {
	authIndicators := []string{
		"401", "403", "unauthorized", "forbidden", "auth", "login",
	}
	
	errLower := strings.ToLower(errMsg)
	for _, indicator := range authIndicators {
		if strings.Contains(errLower, indicator) {
			return true
		}
	}
	return false
}
