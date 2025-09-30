package main

import (
	"context"
	"time"
)

// Timeout constants for various API operations
const (
	timeoutQuick    = 5 * time.Second  // Quick operations: sync triggers, app list loading
	timeoutStandard = 10 * time.Second // Standard operations: API version, resource tree, revision metadata, auth validation
	timeoutMedium   = 30 * time.Second // Medium operations: rollback session loading
	timeoutLong     = 45 * time.Second // Long operations: diff sessions, rollback diffs
	timeoutExtended = 60 * time.Second // Extended operations: rollback execution
)

// contextWithTimeout creates a context with the specified timeout duration.
// The caller is responsible for calling the returned cancel function.
// Returns (ctx, cancel) where cancel must be deferred by the caller.
func contextWithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
