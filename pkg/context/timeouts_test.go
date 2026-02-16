package context

import (
	"context"
	"testing"
	"time"
)

func TestSetRequestTimeout(t *testing.T) {
	// Store original timeouts to restore after test
	originalTimeouts := DefaultTimeouts

	tests := []struct {
		name              string
		timeout           time.Duration
		expectAPI         time.Duration
		expectSync        time.Duration
		expectResource    time.Duration
		expectAuth        time.Duration
		expectUI          time.Duration // Should remain unchanged
		expectStream      time.Duration // Should remain unchanged
	}{
		{
			name:           "set 30 second timeout",
			timeout:        30 * time.Second,
			expectAPI:      30 * time.Second,
			expectSync:     30 * time.Second,
			expectResource: 30 * time.Second,
			expectAuth:     30 * time.Second,
			expectUI:       2 * time.Second, // Should not change
			expectStream:   0,                // Should not change
		},
		{
			name:           "set 1 minute timeout",
			timeout:        1 * time.Minute,
			expectAPI:      1 * time.Minute,
			expectSync:     1 * time.Minute,
			expectResource: 1 * time.Minute,
			expectAuth:     1 * time.Minute,
			expectUI:       2 * time.Second, // Should not change
			expectStream:   0,                // Should not change
		},
		{
			name:           "set 5 second timeout",
			timeout:        5 * time.Second,
			expectAPI:      5 * time.Second,
			expectSync:     5 * time.Second,
			expectResource: 5 * time.Second,
			expectAuth:     5 * time.Second,
			expectUI:       2 * time.Second, // Should not change
			expectStream:   0,                // Should not change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply the timeout
			SetRequestTimeout(tt.timeout)

			// Check all timeout values
			if DefaultTimeouts.API != tt.expectAPI {
				t.Errorf("API timeout = %v, want %v", DefaultTimeouts.API, tt.expectAPI)
			}
			if DefaultTimeouts.Sync != tt.expectSync {
				t.Errorf("Sync timeout = %v, want %v", DefaultTimeouts.Sync, tt.expectSync)
			}
			if DefaultTimeouts.Resource != tt.expectResource {
				t.Errorf("Resource timeout = %v, want %v", DefaultTimeouts.Resource, tt.expectResource)
			}
			if DefaultTimeouts.Auth != tt.expectAuth {
				t.Errorf("Auth timeout = %v, want %v", DefaultTimeouts.Auth, tt.expectAuth)
			}
			if DefaultTimeouts.UI != tt.expectUI {
				t.Errorf("UI timeout = %v, want %v", DefaultTimeouts.UI, tt.expectUI)
			}
			if DefaultTimeouts.Stream != tt.expectStream {
				t.Errorf("Stream timeout = %v, want %v", DefaultTimeouts.Stream, tt.expectStream)
			}
		})
	}

	// Restore original timeouts
	DefaultTimeouts = originalTimeouts
}

func TestTimeoutOperations(t *testing.T) {
	// Store original timeouts to restore after test
	originalTimeouts := DefaultTimeouts
	defer func() {
		DefaultTimeouts = originalTimeouts
	}()

	// Set a custom timeout
	SetRequestTimeout(15 * time.Second)

	tests := []struct {
		name      string
		opType    OperationType
		expectTimeout time.Duration
	}{
		{
			name:      "API operation uses configured timeout",
			opType:    OpAPI,
			expectTimeout: 15 * time.Second,
		},
		{
			name:      "Sync operation uses configured timeout",
			opType:    OpSync,
			expectTimeout: 15 * time.Second,
		},
		{
			name:      "Resource operation uses configured timeout",
			opType:    OpResource,
			expectTimeout: 15 * time.Second,
		},
		{
			name:      "Auth operation uses configured timeout",
			opType:    OpAuth,
			expectTimeout: 15 * time.Second,
		},
		{
			name:      "UI operation keeps its own timeout",
			opType:    OpUI,
			expectTimeout: 2 * time.Second,
		},
		{
			name:      "Stream operation has no timeout",
			opType:    OpStream,
			expectTimeout: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := WithTimeout(context.Background(), tt.opType)
			defer cancel()

			// For operations with timeout, check the deadline
			if tt.expectTimeout > 0 {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Error("Context should have a deadline")
				} else {
					// Check that deadline is approximately correct (within 100ms tolerance)
					expectedDeadline := time.Now().Add(tt.expectTimeout)
					diff := deadline.Sub(expectedDeadline)
					if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
						t.Errorf("Deadline off by %v, expected ~%v from now", diff, tt.expectTimeout)
					}
				}
			} else {
				// Stream operations should not have a deadline
				_, ok := ctx.Deadline()
				if ok {
					t.Error("Stream context should not have a deadline")
				}
			}
		})
	}
}

func TestWithMinAPITimeout(t *testing.T) {
	originalTimeouts := DefaultTimeouts
	defer func() {
		DefaultTimeouts = originalTimeouts
	}()

	tolerance := 100 * time.Millisecond

	t.Run("min floor wins when config is shorter", func(t *testing.T) {
		SetRequestTimeout(15 * time.Second)
		ctx, cancel := WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("Context should have a deadline")
		}
		got := time.Until(deadline)
		if got < 45*time.Second-tolerance || got > 45*time.Second+tolerance {
			t.Errorf("Expected ~45s deadline, got %v", got)
		}
	})

	t.Run("config wins when longer than min", func(t *testing.T) {
		SetRequestTimeout(120 * time.Second)
		ctx, cancel := WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("Context should have a deadline")
		}
		got := time.Until(deadline)
		if got < 120*time.Second-tolerance || got > 120*time.Second+tolerance {
			t.Errorf("Expected ~120s deadline, got %v", got)
		}
	})

	t.Run("config equals min", func(t *testing.T) {
		SetRequestTimeout(45 * time.Second)
		ctx, cancel := WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Fatal("Context should have a deadline")
		}
		got := time.Until(deadline)
		if got < 45*time.Second-tolerance || got > 45*time.Second+tolerance {
			t.Errorf("Expected ~45s deadline, got %v", got)
		}
	})
}

func TestGetTimeoutDuration(t *testing.T) {
	originalTimeouts := DefaultTimeouts
	defer func() {
		DefaultTimeouts = originalTimeouts
	}()

	t.Run("WithAPITimeout stores duration", func(t *testing.T) {
		SetRequestTimeout(42 * time.Second)
		ctx, cancel := WithAPITimeout(context.Background())
		defer cancel()

		d, ok := GetTimeoutDuration(ctx)
		if !ok {
			t.Fatal("Expected timeout duration in context")
		}
		if d != 42*time.Second {
			t.Errorf("Expected 42s, got %v", d)
		}
	})

	t.Run("WithMinAPITimeout stores effective duration", func(t *testing.T) {
		SetRequestTimeout(15 * time.Second)
		ctx, cancel := WithMinAPITimeout(context.Background(), 45*time.Second)
		defer cancel()

		d, ok := GetTimeoutDuration(ctx)
		if !ok {
			t.Fatal("Expected timeout duration in context")
		}
		if d != 45*time.Second {
			t.Errorf("Expected 45s (min floor), got %v", d)
		}
	})

	t.Run("bare context returns false", func(t *testing.T) {
		_, ok := GetTimeoutDuration(context.Background())
		if ok {
			t.Error("Expected no timeout duration on bare context")
		}
	})
}

func TestBackwardCompatibility(t *testing.T) {
	// Store original timeouts to restore after test
	originalTimeouts := DefaultTimeouts
	defer func() {
		DefaultTimeouts = originalTimeouts
	}()

	// Test that existing timeout functions still work after configuration
	SetRequestTimeout(20 * time.Second)

	// Test convenience functions
	t.Run("WithAPITimeout", func(t *testing.T) {
		ctx, cancel := WithAPITimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Context should have a deadline")
		} else {
			expectedDeadline := time.Now().Add(20 * time.Second)
			diff := deadline.Sub(expectedDeadline)
			if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
				t.Errorf("API timeout not using configured value, off by %v", diff)
			}
		}
	})

	t.Run("WithSyncTimeout", func(t *testing.T) {
		ctx, cancel := WithSyncTimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Context should have a deadline")
		} else {
			expectedDeadline := time.Now().Add(20 * time.Second)
			diff := deadline.Sub(expectedDeadline)
			if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
				t.Errorf("Sync timeout not using configured value, off by %v", diff)
			}
		}
	})

	t.Run("WithResourceTimeout", func(t *testing.T) {
		ctx, cancel := WithResourceTimeout(context.Background())
		defer cancel()

		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("Context should have a deadline")
		} else {
			expectedDeadline := time.Now().Add(20 * time.Second)
			diff := deadline.Sub(expectedDeadline)
			if diff < -100*time.Millisecond || diff > 100*time.Millisecond {
				t.Errorf("Resource timeout not using configured value, off by %v", diff)
			}
		}
	})
}