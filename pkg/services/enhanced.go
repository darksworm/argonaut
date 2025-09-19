package services

import (
	"context"
	"sync"

	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
	appcontext "github.com/darksworm/argonaut/pkg/context"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/retry"
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

// isAuthError function already exists in argo.go, reusing it
