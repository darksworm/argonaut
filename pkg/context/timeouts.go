package context

import (
	"context"
	"fmt"
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
)

// TimeoutConfig holds timeout configuration for different operations
type TimeoutConfig struct {
	Default  time.Duration
	API      time.Duration
	Stream   time.Duration
	Auth     time.Duration
	Sync     time.Duration
	Resource time.Duration
	UI       time.Duration
}

// DefaultTimeouts provides sensible defaults for different operation types
var DefaultTimeouts = TimeoutConfig{
	Default:  5 * time.Second,  // Most operations should be fast
	API:      3 * time.Second,  // API calls should be quick
	Stream:   0,                // No timeout for streams (handled by parent context)
	Auth:     5 * time.Second,  // Auth should be reasonably fast
	Sync:     10 * time.Second, // Sync operations - max 10 seconds
	Resource: 3 * time.Second,  // Resource queries should be fast
	UI:       2 * time.Second,  // UI operations must be very fast
}

// OperationType represents different types of operations that need timeouts
type OperationType string

const (
	OpAPI      OperationType = "api"
	OpStream   OperationType = "stream"
	OpAuth     OperationType = "auth"
	OpSync     OperationType = "sync"
	OpResource OperationType = "resource"
	OpUI       OperationType = "ui"
)

// WithTimeout creates a context with timeout based on operation type
func WithTimeout(parent context.Context, opType OperationType) (context.Context, context.CancelFunc) {
	timeout := getTimeoutForOperation(opType)
	if timeout == 0 {
		// For operations with no timeout (like streams), return the parent context
		// but still provide a cancel function for cleanup
		ctx, cancel := context.WithCancel(parent)
		return ctx, cancel
	}
	return context.WithTimeout(parent, timeout)
}

// WithCancel creates a cancellable context
func WithCancel(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(parent)
}

// getTimeoutForOperation returns the appropriate timeout for the given operation type
func getTimeoutForOperation(opType OperationType) time.Duration {
	switch opType {
	case OpAPI:
		return DefaultTimeouts.API
	case OpStream:
		return DefaultTimeouts.Stream
	case OpAuth:
		return DefaultTimeouts.Auth
	case OpSync:
		return DefaultTimeouts.Sync
	case OpResource:
		return DefaultTimeouts.Resource
	case OpUI:
		return DefaultTimeouts.UI
	default:
		return DefaultTimeouts.Default
	}
}

// HandleTimeout converts a context timeout error to a structured error
func HandleTimeout(ctx context.Context, opType OperationType) *apperrors.ArgonautError {
	if ctx.Err() == context.DeadlineExceeded {
		return apperrors.TimeoutError(
			"OPERATION_TIMEOUT",
			fmt.Sprintf("Operation timed out after %v", getTimeoutForOperation(opType)),
		).WithDetails(fmt.Sprintf("Operation type: %s", opType))
	}

	if ctx.Err() == context.Canceled {
		return apperrors.New(
			apperrors.ErrorInternal,
			"OPERATION_CANCELED",
			"Operation was canceled",
		).WithDetails(fmt.Sprintf("Operation type: %s", opType))
	}

	return nil
}

// WithTimeoutAndRetry creates a context with timeout and provides retry information
type RetryableContext struct {
	Context    context.Context
	Cancel     context.CancelFunc
	Attempt    int
	MaxRetries int
	OpType     OperationType
}

// Convenience functions for common timeout patterns

// WithAPITimeout creates a context specifically for API operations
func WithAPITimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, OpAPI)
}

// WithSyncTimeout creates a context specifically for sync operations
func WithSyncTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, OpSync)
}

// WithResourceTimeout creates a context specifically for resource operations
func WithResourceTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	return WithTimeout(parent, OpResource)
}
