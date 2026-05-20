package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// TestRace_StartLoadingApplications_ServerReadInGoroutine verifies Bug 1:
// startLoadingApplications reads m.state.Server inside the returned closure
// (which Bubble Tea runs as a goroutine), not at call time. A concurrent
// SetServerMsg or context switch can mutate m.state.Server between call and
// execution. Run with -race to observe the data race.
func TestRace_StartLoadingApplications_ServerReadInGoroutine(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	// m.state.Server is non-nil (set by buildSyncTestModel)

	cmd := m.startLoadingApplications()

	started := make(chan struct{})
	go func() {
		close(started)
		cmd() // reads m.state.Server at line ~45 inside goroutine
	}()
	<-started
	// Concurrent write — races with the read inside cmd().
	// The race detector fires here before the fix.
	m.state.Server = &model.Server{BaseURL: "https://new.example.com", Token: "tok2"}
}

// TestRace_StartDiffSession_ServerReadInGoroutine verifies Bug 2 (server-read aspect):
// startDiffSession reads m.state.Server inside the returned closure (goroutine),
// not at call time. Additionally, on certain code paths the closure writes
// m.state.Diff.Loading = false and m.state.Diff = &model.DiffState{} while
// Update concurrently reads m.state.Diff — that Diff-write race is confirmed
// by code inspection (requires an HTTP mock to trigger the no-diff path;
// the server-read race below is demonstrable without one).
func TestRace_StartDiffSession_ServerReadInGoroutine(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	m.state.Diff = &model.DiffState{Loading: true}

	cmd := m.startDiffSession("my-app", nil)

	started := make(chan struct{})
	go func() {
		close(started)
		cmd() // reads m.state.Server inside goroutine (lines ~425, ~432)
	}()
	<-started
	// Concurrent write — races with the read inside cmd().
	m.state.Server = &model.Server{BaseURL: "https://other.example.com", Token: "tok2"}
}
