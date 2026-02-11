package main

import (
	"errors"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
)

func TestSetServerMsg_CleansUpExistingAppWatcher(t *testing.T) {
	m := NewModel(nil)

	cleanupCalls := 0
	m.appWatchCleanup = func() { cleanupCalls++ }

	server := &model.Server{BaseURL: "https://argo.example.com"}
	newModel, _ := m.Update(model.SetServerMsg{Server: server})
	m = newModel.(*Model)

	if cleanupCalls != 1 {
		t.Fatalf("expected existing watcher cleanup to be called once, got %d", cleanupCalls)
	}
	if m.appWatchCleanup != nil {
		t.Fatal("expected appWatchCleanup to be cleared until new watch starts")
	}
	if m.state.Server != server {
		t.Fatal("expected server to be updated")
	}
}

func TestWatchStartedMsg_ReplacesCleanupAndForwardsEvents(t *testing.T) {
	m := NewModel(nil)

	oldCleanupCalls := 0
	m.appWatchCleanup = func() { oldCleanupCalls++ }

	newCleanupCalls := 0
	eventChan := make(chan services.ArgoApiEvent, 1)
	eventChan <- services.ArgoApiEvent{Type: "status-change", Status: "watching"}
	close(eventChan)

	newModel, cmd := m.Update(watchStartedMsg{
		eventChan: eventChan,
		cleanup:   func() { newCleanupCalls++ },
	})
	m = newModel.(*Model)

	if oldCleanupCalls != 1 {
		t.Fatalf("expected previous watcher cleanup to be called once, got %d", oldCleanupCalls)
	}

	if cmd == nil {
		t.Fatal("expected consumeWatchEvents command from watchStartedMsg")
	}
	result := cmd()
	// The batch system wraps non-batchable events in AppsBatchUpdateMsg.Immediate
	batchMsg, ok := result.(model.AppsBatchUpdateMsg)
	if !ok {
		t.Fatalf("expected AppsBatchUpdateMsg, got %T", result)
	}
	statusMsg, ok := batchMsg.Immediate.(model.StatusChangeMsg)
	if !ok {
		t.Fatalf("expected StatusChangeMsg in Immediate, got %T", batchMsg.Immediate)
	}
	if statusMsg.Status != "watching" {
		t.Fatalf("expected forwarded status 'watching', got %q", statusMsg.Status)
	}

	m.cleanupAppWatcher()
	if newCleanupCalls != 1 {
		t.Fatalf("expected new watcher cleanup to be called once, got %d", newCleanupCalls)
	}
}

func TestApiErrorMsg_CleansUpAppWatcher(t *testing.T) {
	m := NewModel(nil)

	cleanupCalls := 0
	m.appWatchCleanup = func() { cleanupCalls++ }

	newModel, _ := m.Update(model.ApiErrorMsg{Message: "boom"})
	m = newModel.(*Model)

	if cleanupCalls != 1 {
		t.Fatalf("expected app watcher cleanup on API error, got %d", cleanupCalls)
	}
	if m.appWatchCleanup != nil {
		t.Fatal("expected appWatchCleanup to be cleared after API error")
	}
}

func TestAuthErrorMsg_CleansUpAppWatcher(t *testing.T) {
	m := NewModel(nil)

	cleanupCalls := 0
	m.appWatchCleanup = func() { cleanupCalls++ }

	newModel, _ := m.Update(model.AuthErrorMsg{Error: errors.New("auth required")})
	m = newModel.(*Model)

	if cleanupCalls != 1 {
		t.Fatalf("expected app watcher cleanup on auth error, got %d", cleanupCalls)
	}
	if m.appWatchCleanup != nil {
		t.Fatal("expected appWatchCleanup to be cleared after auth error")
	}
}

func TestWatchStartedMsg_CleanupStopsForwarderWithoutUpstreamClose(t *testing.T) {
	m := NewModel(nil)

	cleanupCalls := 0
	eventChan := make(chan services.ArgoApiEvent)

	newModel, cmd := m.Update(watchStartedMsg{
		eventChan: eventChan,
		cleanup:   func() { cleanupCalls++ },
	})
	m = newModel.(*Model)

	if cmd == nil {
		t.Fatal("expected consumeWatchEvent command from watchStartedMsg")
	}

	done := make(chan any, 1)
	go func() {
		done <- cmd()
	}()

	m.cleanupAppWatcher()

	select {
	case msg := <-done:
		if msg != nil {
			t.Fatalf("expected nil message when watcher is stopped, got %T", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("consumeWatchEvent did not unblock after cleanup")
	}

	if cleanupCalls != 1 {
		t.Fatalf("expected upstream cleanup to be called once, got %d", cleanupCalls)
	}
}
