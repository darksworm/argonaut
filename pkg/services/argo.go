package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"sync"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
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
		return nil, errors.New("server configuration is required")
	}

	// Use the real API service
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	apps, err := s.appService.ListApplications(ctx)
	if err != nil {
		return nil, err
	}

	return apps, nil
}

// WatchApplications implements ArgoApiService.WatchApplications
func (s *ArgoApiServiceImpl) WatchApplications(ctx context.Context, server *model.Server) (<-chan ArgoApiEvent, func(), error) {
	if server == nil {
		return nil, nil, errors.New("server configuration is required")
	}

	// Use the real API service
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	eventChan := make(chan ArgoApiEvent, 100)
	watchCtx, cancel := context.WithCancel(ctx)
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

		// Send initial apps loaded event
		apps, err := s.ListApplications(watchCtx, server)
		if err != nil {
			if isAuthError(err.Error()) {
				eventChan <- ArgoApiEvent{
					Type:  "auth-error",
					Error: err,
				}
				eventChan <- ArgoApiEvent{
					Type:   "status-change",
					Status: "Auth required",
				}
				return
			}
			eventChan <- ArgoApiEvent{
				Type:  "api-error",
				Error: err,
			}
			eventChan <- ArgoApiEvent{
				Type:   "status-change",
				Status: "Error: " + err.Error(),
			}
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
		return errors.New("server configuration is required")
	}
	if appName == "" {
		return errors.New("application name is required")
	}

	// Use the real API service
	if s.appService == nil {
		s.appService = api.NewApplicationService(server)
	}

	opts := &api.SyncOptions{
		Prune: prune,
	}

	return s.appService.SyncApplication(ctx, appName, opts)
}

// GetResourceDiffs implements ArgoApiService.GetResourceDiffs
func (s *ArgoApiServiceImpl) GetResourceDiffs(ctx context.Context, server *model.Server, appName string) ([]ResourceDiff, error) {
	if server == nil {
		return nil, errors.New("server configuration is required")
	}
	if appName == "" {
		return nil, errors.New("application name is required")
	}

    // Use the real API service
    if s.appService == nil {
        s.appService = api.NewApplicationService(server)
    }

    diffs, err := s.appService.GetManagedResourceDiffs(ctx, appName)
    if err != nil {
        return nil, err
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
    if server == nil { return "", errors.New("server configuration is required") }
    client := api.NewClient(server)
    data, err := client.Get(ctx, "/api/version")
    if err != nil { return "", err }
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
