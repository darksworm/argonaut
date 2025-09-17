package services

import (
	"context"
	"sync"
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/logging"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/retry"
)

// StreamRecoveryConfig configures stream recovery behavior
type StreamRecoveryConfig struct {
	MaxReconnectAttempts int           `json:"maxReconnectAttempts"`
	InitialBackoff       time.Duration `json:"initialBackoff"`
	MaxBackoff           time.Duration `json:"maxBackoff"`
	BackoffMultiplier    float64       `json:"backoffMultiplier"`
	HealthCheckInterval  time.Duration `json:"healthCheckInterval"`
}

// DefaultStreamRecoveryConfig provides sensible defaults
var DefaultStreamRecoveryConfig = StreamRecoveryConfig{
	MaxReconnectAttempts: 10,
	InitialBackoff:       1 * time.Second,
	MaxBackoff:           60 * time.Second,
	BackoffMultiplier:    2.0,
	HealthCheckInterval:  30 * time.Second,
}

// StreamRecoveryManager handles stream connection recovery
type StreamRecoveryManager struct {
	config        StreamRecoveryConfig
	logger        logging.Logger
	mu            sync.RWMutex
	activeStreams map[string]*StreamConnection
	shutdown      chan struct{}
	wg            sync.WaitGroup
}

// StreamConnection represents an active stream connection
type StreamConnection struct {
	ID           string
	Server       *model.Server
	LastSeen     time.Time
	Failures     int
	Status       StreamStatus
	RecoveryFunc func(ctx context.Context) error
	Context      context.Context
	Cancel       context.CancelFunc
}

// StreamStatus represents the status of a stream
type StreamStatus string

const (
	StreamStatusHealthy      StreamStatus = "healthy"
	StreamStatusRecovering   StreamStatus = "recovering"
	StreamStatusFailed       StreamStatus = "failed"
	StreamStatusDisconnected StreamStatus = "disconnected"
)

// NewStreamRecoveryManager creates a new stream recovery manager
func NewStreamRecoveryManager(config StreamRecoveryConfig) *StreamRecoveryManager {
	manager := &StreamRecoveryManager{
		config:        config,
		logger:        logging.GetDefaultLogger().WithComponent("stream-recovery"),
		activeStreams: make(map[string]*StreamConnection),
		shutdown:      make(chan struct{}),
	}

	// Start health check goroutine
	manager.wg.Add(1)
	go manager.healthCheckLoop()

	return manager
}

// ReportStreamFailure reports a failure for a specific stream
func (m *StreamRecoveryManager) ReportStreamFailure(id string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	stream, exists := m.activeStreams[id]
	if !exists {
		return
	}

	stream.Failures++
	stream.Status = StreamStatusFailed
	stream.LastSeen = time.Now()

	m.logger.Warn("Stream failure reported for %s (failures: %d): %v", id, stream.Failures, err)

	// Start recovery process
	m.wg.Add(1)
	go m.recoverStream(stream, err)
}

// recoverStream attempts to recover a failed stream
func (m *StreamRecoveryManager) recoverStream(stream *StreamConnection, originalErr error) {
	defer m.wg.Done()

	m.logger.Info("Starting recovery for stream %s", stream.ID)

	// Create recovery context with timeout
	recoveryCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create retry configuration for stream recovery
	retryConfig := retry.RetryConfig{
		MaxAttempts:  m.config.MaxReconnectAttempts,
		InitialDelay: m.config.InitialBackoff,
		MaxDelay:     m.config.MaxBackoff,
		Multiplier:   m.config.BackoffMultiplier,
		Jitter:       true,
		ShouldRetry: func(err *apperrors.ArgonautError) bool {
			// Retry most errors except authentication and permission issues
			return err.Category != apperrors.ErrorAuth &&
				err.Category != apperrors.ErrorPermission &&
				err.Category != apperrors.ErrorConfig
		},
	}

	m.mu.Lock()
	stream.Status = StreamStatusRecovering
	m.mu.Unlock()

	err := retry.RetryWithBackoff(recoveryCtx, retryConfig, func(attempt int) error {
		m.logger.Debug("Recovery attempt %d for stream %s", attempt, stream.ID)

		select {
		case <-m.shutdown:
			return apperrors.New(apperrors.ErrorInternal, "SHUTDOWN", "Recovery cancelled due to shutdown")
		case <-stream.Context.Done():
			return apperrors.New(apperrors.ErrorInternal, "CANCELLED", "Stream context cancelled")
		default:
			return stream.RecoveryFunc(recoveryCtx)
		}
	})

	m.mu.Lock()
	defer m.mu.Unlock()

	if err != nil {
		stream.Status = StreamStatusFailed
		stream.Failures++
		m.logger.Error("Failed to recover stream %s after %d attempts: %v",
			stream.ID, m.config.MaxReconnectAttempts, err)

		// If we've exceeded max attempts, mark as disconnected
		if stream.Failures >= m.config.MaxReconnectAttempts {
			stream.Status = StreamStatusDisconnected
			m.logger.Error("Stream %s marked as disconnected after %d failures",
				stream.ID, stream.Failures)
		}
	} else {
		stream.Status = StreamStatusHealthy
		stream.Failures = 0
		stream.LastSeen = time.Now()
		m.logger.Info("Successfully recovered stream %s", stream.ID)
	}
}

// healthCheckLoop periodically checks stream health
func (m *StreamRecoveryManager) healthCheckLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.shutdown:
			return
		case <-ticker.C:
			m.performHealthCheck()
		}
	}
}

// performHealthCheck checks all registered streams for health
func (m *StreamRecoveryManager) performHealthCheck() {
	m.mu.RLock()
	streams := make([]*StreamConnection, 0, len(m.activeStreams))
	for _, stream := range m.activeStreams {
		streams = append(streams, stream)
	}
	m.mu.RUnlock()

	now := time.Now()
	for _, stream := range streams {
		// Check if stream hasn't been seen for too long
		if now.Sub(stream.LastSeen) > m.config.HealthCheckInterval*2 {
			if stream.Status == StreamStatusHealthy {
				m.logger.Warn("Stream %s appears stale, marking for recovery", stream.ID)
				m.ReportStreamFailure(stream.ID,
					apperrors.TimeoutError("STREAM_STALE", "Stream has been inactive"))
			}
		}
	}
}

// StreamRecoveryStats provides statistics about stream recovery
type StreamRecoveryStats struct {
	TotalStreams        int                     `json:"totalStreams"`
	HealthyStreams      int                     `json:"healthyStreams"`
	RecoveringStreams   int                     `json:"recoveringStreams"`
	FailedStreams       int                     `json:"failedStreams"`
	DisconnectedStreams int                     `json:"disconnectedStreams"`
	StreamDetails       map[string]StreamDetail `json:"streamDetails"`
}

// StreamDetail provides detailed information about a stream
type StreamDetail struct {
	ID       string       `json:"id"`
	Status   StreamStatus `json:"status"`
	LastSeen time.Time    `json:"lastSeen"`
	Failures int          `json:"failures"`
	Server   string       `json:"server"`
}
