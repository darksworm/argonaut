package main

import (
	"testing"
	"time"
)

// TestConsumeTreeEvent_TerminatesAfterCleanup verifies Bug 6:
// consumeTreeEvent reads m.treeStream inside the closure at goroutine
// execution time rather than capturing the channel at call time. After
// cleanupTreeWatchers runs (on a view change or context switch), nothing
// writes to treeStream and the channel is never closed — so any in-flight
// consumeTreeEvent goroutine blocks forever. This test should time out
// before the fix (Bug 7) closes the channel in cleanupTreeWatchers.
func TestConsumeTreeEvent_TerminatesAfterCleanup(t *testing.T) {
	m := NewModel(nil)

	cmd := m.consumeTreeEvent()

	done := make(chan struct{})
	go func() {
		cmd()
		close(done)
	}()

	m.cleanupTreeWatchers()

	select {
	case <-done:
		// pass: goroutine exited after the channel was closed
	case <-time.After(300 * time.Millisecond):
		t.Error("Bug 6/7: consumeTreeEvent goroutine leaked — did not exit after cleanupTreeWatchers")
	}
}

// TestCleanupTreeWatchers_ClosesTreeStream verifies Bug 7:
// cleanupTreeWatchers must signal any waiting consumeTreeEvent goroutines to exit.
// After the fix, treeStreamDone is closed (unblocking goroutines) and then
// re-created (non-nil) for the next time the tree view is entered.
func TestCleanupTreeWatchers_ClosesTreeStream(t *testing.T) {
	m := NewModel(nil)

	doneBefore := m.treeStreamDone
	m.cleanupTreeWatchers()

	// treeStreamDone must be non-nil (re-created) after cleanup
	if m.treeStreamDone == nil {
		t.Error("Bug 7: treeStreamDone is nil after cleanupTreeWatchers — not re-created")
	}
	// The old channel must have been closed (select should not block on default)
	select {
	case <-doneBefore:
		// closed — correct
	default:
		t.Error("Bug 7: treeStreamDone was not closed by cleanupTreeWatchers")
	}
	// The new channel must be open (not yet closed)
	select {
	case <-m.treeStreamDone:
		t.Error("treeStreamDone was prematurely closed after re-creation")
	default:
		// open — correct
	}
}
