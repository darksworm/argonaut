package services

import (
	"sync"
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/logging"
	"github.com/darksworm/argonaut/pkg/model"
)

// DegradationMode represents different levels of service degradation
type DegradationMode string

const (
	DegradationNone     DegradationMode = "none"     // Full functionality
	DegradationPartial  DegradationMode = "partial"  // Some features disabled
	DegradationOffline  DegradationMode = "offline"  // Offline/cached mode
	DegradationReadOnly DegradationMode = "readonly" // Read-only operations
)

// ServiceHealth represents the health status of various services
type ServiceHealth struct {
	ArgoAPI      HealthStatus `json:"argoAPI"`
	Authentication HealthStatus `json:"authentication"`
	Connectivity HealthStatus `json:"connectivity"`
	LastCheck    time.Time    `json:"lastCheck"`
	Mode         DegradationMode `json:"mode"`
}

// HealthStatus represents the status of a service component
type HealthStatus struct {
	Status    string    `json:"status"`    // healthy, degraded, unavailable
	LastSeen  time.Time `json:"lastSeen"`
	Failures  int       `json:"failures"`
	Message   string    `json:"message,omitempty"`
}

// GracefulDegradationManager handles service degradation scenarios
type GracefulDegradationManager struct {
	health          ServiceHealth
	mu              sync.RWMutex
	logger          logging.Logger
	cache           *ServiceCache
	healthCheckTicker *time.Ticker
	shutdown        chan struct{}
	callbacks       []DegradationCallback
}

// DegradationCallback is called when degradation mode changes
type DegradationCallback func(oldMode, newMode DegradationMode)

// ServiceCache provides cached data for offline mode
type ServiceCache struct {
	Apps         []model.App   `json:"apps"`
	LastUpdated  time.Time     `json:"lastUpdated"`
	Server       *model.Server `json:"server"`
	APIVersion   string        `json:"apiVersion"`
}

// NewGracefulDegradationManager creates a new degradation manager
func NewGracefulDegradationManager() *GracefulDegradationManager {
	manager := &GracefulDegradationManager{
		health: ServiceHealth{
			ArgoAPI:      HealthStatus{Status: "unknown", LastSeen: time.Now()},
			Authentication: HealthStatus{Status: "unknown", LastSeen: time.Now()},
			Connectivity: HealthStatus{Status: "unknown", LastSeen: time.Now()},
			LastCheck:    time.Now(),
			Mode:         DegradationNone,
		},
		logger:   logging.GetDefaultLogger().WithComponent("degradation"),
		cache:    &ServiceCache{},
		shutdown: make(chan struct{}),
	}

	// Start health monitoring
	manager.startHealthMonitoring()

	return manager
}

// RegisterCallback registers a callback for degradation mode changes
func (m *GracefulDegradationManager) RegisterCallback(callback DegradationCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

// ReportAPIHealth reports the health status of the ArgoCD API
func (m *GracefulDegradationManager) ReportAPIHealth(healthy bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if healthy {
		m.health.ArgoAPI.Status = "healthy"
		m.health.ArgoAPI.LastSeen = time.Now()
		m.health.ArgoAPI.Failures = 0
		m.health.ArgoAPI.Message = ""
	} else {
		m.health.ArgoAPI.Failures++
		m.health.ArgoAPI.Message = ""

		if err != nil {
			if argErr, ok := err.(*apperrors.ArgonautError); ok {
				switch argErr.Category {
				case apperrors.ErrorAuth:
					m.health.Authentication.Status = "unavailable"
					m.health.ArgoAPI.Status = "degraded"
					m.health.ArgoAPI.Message = "Authentication required"
				case apperrors.ErrorNetwork, apperrors.ErrorTimeout:
					m.health.Connectivity.Status = "unavailable"
					m.health.ArgoAPI.Status = "unavailable"
					m.health.ArgoAPI.Message = "Network connectivity issues"
				default:
					m.health.ArgoAPI.Status = "degraded"
					m.health.ArgoAPI.Message = argErr.Message
				}
			} else {
				m.health.ArgoAPI.Status = "unavailable"
				m.health.ArgoAPI.Message = err.Error()
			}
		}
	}

	m.health.LastCheck = time.Now()
	m.updateDegradationMode()
}

// ReportAuthHealth reports authentication health status
func (m *GracefulDegradationManager) ReportAuthHealth(healthy bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if healthy {
		m.health.Authentication.Status = "healthy"
		m.health.Authentication.LastSeen = time.Now()
		m.health.Authentication.Failures = 0
		m.health.Authentication.Message = ""
	} else {
		m.health.Authentication.Status = "unavailable"
		m.health.Authentication.Failures++
		if err != nil {
			m.health.Authentication.Message = err.Error()
		}
	}

	m.updateDegradationMode()
}

// ReportConnectivityHealth reports network connectivity health
func (m *GracefulDegradationManager) ReportConnectivityHealth(healthy bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if healthy {
		m.health.Connectivity.Status = "healthy"
		m.health.Connectivity.LastSeen = time.Now()
		m.health.Connectivity.Failures = 0
		m.health.Connectivity.Message = ""
	} else {
		m.health.Connectivity.Status = "unavailable"
		m.health.Connectivity.Failures++
		if err != nil {
			m.health.Connectivity.Message = err.Error()
		}
	}

	m.updateDegradationMode()
}

// updateDegradationMode determines the appropriate degradation mode based on health
func (m *GracefulDegradationManager) updateDegradationMode() {
	oldMode := m.health.Mode
	var newMode DegradationMode

	// Determine degradation mode based on service health
	if m.health.Authentication.Status == "unavailable" {
		newMode = DegradationOffline
	} else if m.health.Connectivity.Status == "unavailable" {
		newMode = DegradationOffline
	} else if m.health.ArgoAPI.Status == "unavailable" {
		newMode = DegradationOffline
	} else if m.health.ArgoAPI.Status == "degraded" {
		newMode = DegradationPartial
	} else {
		newMode = DegradationNone
	}

	if newMode != oldMode {
		m.health.Mode = newMode
		m.logger.Info("Degradation mode changed: %s -> %s", oldMode, newMode)

		// Notify callbacks
		for _, callback := range m.callbacks {
			go callback(oldMode, newMode)
		}
	}
}

// GetCurrentMode returns the current degradation mode
func (m *GracefulDegradationManager) GetCurrentMode() DegradationMode {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health.Mode
}

// GetServiceHealth returns the current service health status
func (m *GracefulDegradationManager) GetServiceHealth() ServiceHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.health
}

// CanPerformOperation checks if an operation is allowed in current degradation mode
func (m *GracefulDegradationManager) CanPerformOperation(operation string) (bool, *apperrors.ArgonautError) {
	mode := m.GetCurrentMode()

	switch mode {
	case DegradationNone:
		return true, nil

	case DegradationPartial:
		// Allow read operations, restrict write operations
		readOps := []string{"ListApplications", "GetApplication", "GetResourceDiffs", "GetAPIVersion"}
		for _, op := range readOps {
			if op == operation {
				return true, nil
			}
		}
		return false, apperrors.New(apperrors.ErrorUnavailable, "OPERATION_RESTRICTED",
			"Operation restricted due to service degradation").
			WithUserAction("Some features are temporarily unavailable due to service issues")

	case DegradationReadOnly:
		// Only allow read operations
		readOps := []string{"ListApplications", "GetApplication", "GetResourceDiffs", "GetAPIVersion"}
		for _, op := range readOps {
			if op == operation {
				return true, nil
			}
		}
		return false, apperrors.New(apperrors.ErrorUnavailable, "READONLY_MODE",
			"System is in read-only mode").
			WithUserAction("Write operations are disabled. Please try again later")

	case DegradationOffline:
		// Only allow cached operations
		cachedOps := []string{"ListApplications"} // We can serve from cache
		for _, op := range cachedOps {
			if op == operation {
				return true, nil
			}
		}
		return false, apperrors.New(apperrors.ErrorUnavailable, "OFFLINE_MODE",
			"System is offline - limited functionality available").
			WithUserAction("Check your connection and try again. Some cached data may be available")

	default:
		return false, apperrors.New(apperrors.ErrorInternal, "UNKNOWN_MODE",
			"Unknown degradation mode")
	}
}

// UpdateCache updates the service cache with fresh data
func (m *GracefulDegradationManager) UpdateCache(apps []model.App, server *model.Server, apiVersion string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cache.Apps = apps
	m.cache.Server = server
	m.cache.APIVersion = apiVersion
	m.cache.LastUpdated = time.Now()

	m.logger.Debug("Updated service cache with %d apps", len(apps))
}

// GetCachedApps returns cached application data
func (m *GracefulDegradationManager) GetCachedApps() ([]model.App, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Return cached data if it's less than 5 minutes old
	if time.Since(m.cache.LastUpdated) < 5*time.Minute && len(m.cache.Apps) > 0 {
		return m.cache.Apps, true
	}

	return nil, false
}

// GetCacheAge returns the age of the cached data
func (m *GracefulDegradationManager) GetCacheAge() time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return time.Since(m.cache.LastUpdated)
}

// startHealthMonitoring starts periodic health checks
func (m *GracefulDegradationManager) startHealthMonitoring() {
	m.healthCheckTicker = time.NewTicker(30 * time.Second)

	go func() {
		for {
			select {
			case <-m.shutdown:
				return
			case <-m.healthCheckTicker.C:
				m.performHealthCheck()
			}
		}
	}()
}

// performHealthCheck performs periodic health validation
func (m *GracefulDegradationManager) performHealthCheck() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Check if services haven't been seen for too long
	if now.Sub(m.health.ArgoAPI.LastSeen) > 2*time.Minute && m.health.ArgoAPI.Status == "healthy" {
		m.health.ArgoAPI.Status = "degraded"
		m.health.ArgoAPI.Message = "Service hasn't been seen recently"
		m.logger.Warn("ArgoCD API marked as degraded due to inactivity")
	}

	if now.Sub(m.health.Connectivity.LastSeen) > 2*time.Minute && m.health.Connectivity.Status == "healthy" {
		m.health.Connectivity.Status = "degraded"
		m.health.Connectivity.Message = "Connectivity check overdue"
		m.logger.Warn("Connectivity marked as degraded due to inactivity")
	}

	m.health.LastCheck = now
	m.updateDegradationMode()
}

// Shutdown gracefully shuts down the degradation manager
func (m *GracefulDegradationManager) Shutdown() {
	if m.healthCheckTicker != nil {
		m.healthCheckTicker.Stop()
	}
	close(m.shutdown)
	m.logger.Info("Graceful degradation manager shutdown complete")
}

// GetDegradationSummary returns a human-readable summary of the current degradation status
func (m *GracefulDegradationManager) GetDegradationSummary() string {
	health := m.GetServiceHealth()

	switch health.Mode {
	case DegradationNone:
		return "All systems operational"
	case DegradationPartial:
		return "Some features may be limited due to service issues"
	case DegradationReadOnly:
		return "System in read-only mode - write operations disabled"
	case DegradationOffline:
		cacheAge := m.GetCacheAge()
		if cacheAge < 5*time.Minute {
			return "Offline mode - showing cached data from " + cacheAge.Round(time.Second).String() + " ago"
		}
		return "Offline mode - no cached data available"
	default:
		return "System status unknown"
	}
}