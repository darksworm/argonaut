package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/logging"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts   int                          `json:"maxAttempts"`
	InitialDelay  time.Duration                `json:"initialDelay"`
	MaxDelay      time.Duration                `json:"maxDelay"`
	Multiplier    float64                      `json:"multiplier"`
	Jitter        bool                         `json:"jitter"`
	ShouldRetry   func(*apperrors.ArgonautError) bool `json:"-"`
}

// DefaultConfig provides sensible retry defaults
var DefaultConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     30 * time.Second,
	Multiplier:   2.0,
	Jitter:       true,
	ShouldRetry:  DefaultShouldRetry,
}

// NetworkConfig is optimized for network operations
var NetworkConfig = RetryConfig{
	MaxAttempts:  5,
	InitialDelay: 500 * time.Millisecond,
	MaxDelay:     10 * time.Second,
	Multiplier:   1.5,
	Jitter:       true,
	ShouldRetry:  NetworkShouldRetry,
}

// APIConfig is optimized for API operations
var APIConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     15 * time.Second,
	Multiplier:   2.0,
	Jitter:       true,
	ShouldRetry:  APIShouldRetry,
}

// RetryFunc is a function that can be retried
type RetryFunc func(attempt int) error

// RetryWithBackoff executes a function with exponential backoff retry logic
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn RetryFunc) error {
	logger := logging.GetDefaultLogger().WithComponent("retry")

	var lastErr error

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		startTime := time.Now()
		err := fn(attempt)
		duration := time.Since(startTime)

		if err == nil {
			if attempt > 1 {
				logger.Info("Operation succeeded after %d attempts (took %v)", attempt, duration)
			}
			return nil
		}

		lastErr = err

		// Convert to structured error if needed
		var argErr *apperrors.ArgonautError
		if ae, ok := err.(*apperrors.ArgonautError); ok {
			argErr = ae
		} else {
			argErr = apperrors.Wrap(err, apperrors.ErrorInternal, "RETRY_OPERATION_FAILED", "Operation failed during retry")
		}

		// Log the attempt
		logger.Warn("Attempt %d/%d failed (took %v): %s",
			attempt, config.MaxAttempts, duration, argErr.Error())

		// Check if we should retry
		if !config.ShouldRetry(argErr) {
			logger.Info("Not retrying due to error type: %s", argErr.Category)
			return argErr
		}

		// Don't sleep after the last attempt
		if attempt >= config.MaxAttempts {
			break
		}

		// Check if context is cancelled
		if ctx.Err() != nil {
			logger.Info("Context cancelled, stopping retry attempts")
			return apperrors.Wrap(ctx.Err(), apperrors.ErrorTimeout, "RETRY_CANCELLED", "Retry cancelled due to context")
		}

		// Calculate delay for next attempt
		delay := calculateDelay(attempt, config)
		logger.Debug("Waiting %v before attempt %d", delay, attempt+1)

		// Wait with context cancellation support
		select {
		case <-ctx.Done():
			return apperrors.Wrap(ctx.Err(), apperrors.ErrorTimeout, "RETRY_CANCELLED", "Retry cancelled due to context")
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	if argErr, ok := lastErr.(*apperrors.ArgonautError); ok {
		return argErr.WithContext("retryAttempts", config.MaxAttempts)
	}

	return apperrors.Wrap(lastErr, apperrors.ErrorInternal, "RETRY_EXHAUSTED",
		"All retry attempts failed").
		WithContext("maxAttempts", config.MaxAttempts).
		WithUserAction("The operation failed after multiple attempts. Check your connection and try again")
}

// calculateDelay calculates the delay for the next retry attempt
func calculateDelay(attempt int, config RetryConfig) time.Duration {
	// Calculate exponential backoff
	delay := time.Duration(float64(config.InitialDelay) * math.Pow(config.Multiplier, float64(attempt-1)))

	// Apply maximum delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}

	// Apply jitter to prevent thundering herd
	if config.Jitter {
		jitter := time.Duration(rand.Float64() * float64(delay) * 0.1) // 10% jitter
		delay = delay + jitter
	}

	return delay
}

// DefaultShouldRetry is the default retry predicate
func DefaultShouldRetry(err *apperrors.ArgonautError) bool {
	if err == nil {
		return false
	}

	// Don't retry authentication or validation errors
	switch err.Category {
	case apperrors.ErrorAuth, apperrors.ErrorValidation, apperrors.ErrorPermission:
		return false
	case apperrors.ErrorNetwork, apperrors.ErrorTimeout, apperrors.ErrorAPI:
		return true
	default:
		// Use the error's recoverable flag
		return err.Recoverable
	}
}

// NetworkShouldRetry determines if network errors should be retried
func NetworkShouldRetry(err *apperrors.ArgonautError) bool {
	if err == nil {
		return false
	}

	// Retry most network and timeout errors
	switch err.Category {
	case apperrors.ErrorNetwork, apperrors.ErrorTimeout:
		return true
	case apperrors.ErrorAuth, apperrors.ErrorValidation, apperrors.ErrorPermission:
		return false
	case apperrors.ErrorAPI:
		// Only retry certain API errors
		return err.IsCode("CONNECTION_REFUSED") ||
			   err.IsCode("TIMEOUT") ||
			   err.IsCode("SERVICE_UNAVAILABLE") ||
			   err.IsCode("RATE_LIMITED") ||
			   err.IsCode("SERVER_ERROR")
	default:
		return err.Recoverable
	}
}

// APIShouldRetry determines if API errors should be retried
func APIShouldRetry(err *apperrors.ArgonautError) bool {
	if err == nil {
		return false
	}

	// Don't retry client errors (4xx) except for specific cases
	switch err.Category {
	case apperrors.ErrorAuth, apperrors.ErrorValidation, apperrors.ErrorPermission:
		return false
	case apperrors.ErrorNetwork, apperrors.ErrorTimeout:
		return true
	case apperrors.ErrorAPI:
		// Retry server errors and rate limits, but not client errors
		return err.IsCode("SERVER_ERROR") ||
			   err.IsCode("RATE_LIMITED") ||
			   err.IsCode("SERVICE_UNAVAILABLE") ||
			   err.IsCode("TIMEOUT")
	default:
		return err.Recoverable
	}
}

// RetryableOperation wraps an operation with retry logic
type RetryableOperation struct {
	Name     string
	Config   RetryConfig
	Logger   logging.Logger
	Context  context.Context
}

// NewRetryableOperation creates a new retryable operation
func NewRetryableOperation(name string, config RetryConfig) *RetryableOperation {
	return &RetryableOperation{
		Name:    name,
		Config:  config,
		Logger:  logging.GetDefaultLogger().WithOperation(name),
		Context: context.Background(),
	}
}

// WithContext sets the context for the operation
func (ro *RetryableOperation) WithContext(ctx context.Context) *RetryableOperation {
	ro.Context = ctx
	return ro
}

// WithLogger sets a custom logger
func (ro *RetryableOperation) WithLogger(logger logging.Logger) *RetryableOperation {
	ro.Logger = logger
	return ro
}

// Execute executes the operation with retry logic
func (ro *RetryableOperation) Execute(fn RetryFunc) error {
	ro.Logger.Info("Starting retryable operation: %s", ro.Name)

	startTime := time.Now()
	err := RetryWithBackoff(ro.Context, ro.Config, fn)
	duration := time.Since(startTime)

	if err != nil {
		ro.Logger.Error("Operation %s failed after %v: %v", ro.Name, duration, err)
	} else {
		ro.Logger.Info("Operation %s completed successfully in %v", ro.Name, duration)
	}

	return err
}

// Convenience functions for common retry patterns

// RetryNetworkOperation retries a network operation with appropriate config
func RetryNetworkOperation(ctx context.Context, name string, fn RetryFunc) error {
	op := NewRetryableOperation(name, NetworkConfig).WithContext(ctx)
	return op.Execute(fn)
}

// RetryAPIOperation retries an API operation with appropriate config
func RetryAPIOperation(ctx context.Context, name string, fn RetryFunc) error {
	op := NewRetryableOperation(name, APIConfig).WithContext(ctx)
	return op.Execute(fn)
}

// RetryOperation retries an operation with default config
func RetryOperation(ctx context.Context, name string, fn RetryFunc) error {
	op := NewRetryableOperation(name, DefaultConfig).WithContext(ctx)
	return op.Execute(fn)
}

// Quick retry functions for immediate use

// DoWithRetry executes a function with default retry logic
func DoWithRetry(ctx context.Context, fn func() error) error {
	return RetryWithBackoff(ctx, DefaultConfig, func(attempt int) error {
		return fn()
	})
}

// DoNetworkWithRetry executes a network function with network retry logic
func DoNetworkWithRetry(ctx context.Context, fn func() error) error {
	return RetryWithBackoff(ctx, NetworkConfig, func(attempt int) error {
		return fn()
	})
}

// DoAPIWithRetry executes an API function with API retry logic
func DoAPIWithRetry(ctx context.Context, fn func() error) error {
	return RetryWithBackoff(ctx, APIConfig, func(attempt int) error {
		return fn()
	})
}