package main

import (
	"context"
	"time"
)

// contextWithTimeout creates a context with the specified timeout duration.
// The caller is responsible for calling the returned cancel function.
// Returns (ctx, cancel) where cancel must be deferred by the caller.
func contextWithTimeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}
