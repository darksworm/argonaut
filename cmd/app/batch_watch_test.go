package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
)

func TestClassifyWatchEvent_AppUpdated(t *testing.T) {
	app := &model.App{Name: "test-app", Health: "Healthy", Sync: "Synced"}
	ev := services.ArgoApiEvent{Type: "app-updated", App: app}
	result := classifyWatchEvent(ev)

	if result.update == nil {
		t.Fatal("expected update to be non-nil")
	}
	if result.update.App.Name != "test-app" {
		t.Errorf("expected app name 'test-app', got %q", result.update.App.Name)
	}
	if result.immediate != nil {
		t.Error("expected immediate to be nil for batchable event")
	}
	if result.deleteName != "" {
		t.Error("expected deleteName to be empty")
	}
}

func TestClassifyWatchEvent_AppDeleted(t *testing.T) {
	ev := services.ArgoApiEvent{Type: "app-deleted", AppName: "removed-app"}
	result := classifyWatchEvent(ev)

	if result.deleteName != "removed-app" {
		t.Errorf("expected deleteName 'removed-app', got %q", result.deleteName)
	}
	if result.update != nil {
		t.Error("expected update to be nil")
	}
	if result.immediate != nil {
		t.Error("expected immediate to be nil for batchable event")
	}
}

func TestClassifyWatchEvent_AuthError(t *testing.T) {
	ev := services.ArgoApiEvent{Type: "auth-error", Error: fmt.Errorf("unauthorized")}
	result := classifyWatchEvent(ev)

	if result.immediate == nil {
		t.Fatal("expected immediate to be non-nil for auth-error")
	}
	if _, ok := result.immediate.(model.AuthErrorMsg); !ok {
		t.Errorf("expected AuthErrorMsg, got %T", result.immediate)
	}
	if result.update != nil || result.deleteName != "" {
		t.Error("expected update/delete fields to be empty for non-batchable event")
	}
}

func TestClassifyWatchEvent_StatusChange(t *testing.T) {
	ev := services.ArgoApiEvent{Type: "status-change", Status: "Connected"}
	result := classifyWatchEvent(ev)

	if result.immediate == nil {
		t.Fatal("expected immediate to be non-nil for status-change")
	}
	msg, ok := result.immediate.(model.StatusChangeMsg)
	if !ok {
		t.Fatalf("expected StatusChangeMsg, got %T", result.immediate)
	}
	if msg.Status != "Connected" {
		t.Errorf("expected status 'Connected', got %q", msg.Status)
	}
}

func TestConsumeWatchEvents_BatchesMultipleUpdates(t *testing.T) {
	m := &Model{
		watchChan: make(chan services.ArgoApiEvent, 10),
	}

	// Send multiple events quickly
	apps := []string{"app-1", "app-2", "app-3"}
	for _, name := range apps {
		m.watchChan <- services.ArgoApiEvent{
			Type: "app-updated",
			App:  &model.App{Name: name, Health: "Healthy"},
		}
	}

	cmd := m.consumeWatchEvents()
	msg := cmd()

	batch, ok := msg.(model.AppsBatchUpdateMsg)
	if !ok {
		t.Fatalf("expected AppsBatchUpdateMsg, got %T", msg)
	}
	if len(batch.Updates) != 3 {
		t.Errorf("expected 3 updates in batch, got %d", len(batch.Updates))
	}
	if len(batch.Deletes) != 0 {
		t.Errorf("expected 0 deletes, got %d", len(batch.Deletes))
	}
}

func TestConsumeWatchEvents_MixedUpdatesAndDeletes(t *testing.T) {
	m := &Model{
		watchChan: make(chan services.ArgoApiEvent, 10),
	}

	m.watchChan <- services.ArgoApiEvent{
		Type: "app-updated",
		App:  &model.App{Name: "app-1", Health: "Healthy"},
	}
	m.watchChan <- services.ArgoApiEvent{
		Type:    "app-deleted",
		AppName: "app-2",
	}
	m.watchChan <- services.ArgoApiEvent{
		Type: "app-updated",
		App:  &model.App{Name: "app-3", Health: "Degraded"},
	}

	cmd := m.consumeWatchEvents()
	msg := cmd()

	batch, ok := msg.(model.AppsBatchUpdateMsg)
	if !ok {
		t.Fatalf("expected AppsBatchUpdateMsg, got %T", msg)
	}
	if len(batch.Updates) != 2 {
		t.Errorf("expected 2 updates, got %d", len(batch.Updates))
	}
	if len(batch.Deletes) != 1 {
		t.Errorf("expected 1 delete, got %d", len(batch.Deletes))
	}
	if batch.Deletes[0] != "app-2" {
		t.Errorf("expected delete of 'app-2', got %q", batch.Deletes[0])
	}
}

func TestConsumeWatchEvents_ImmediateEventStopsBatching(t *testing.T) {
	m := &Model{
		watchChan: make(chan services.ArgoApiEvent, 10),
	}

	m.watchChan <- services.ArgoApiEvent{
		Type: "app-updated",
		App:  &model.App{Name: "app-1"},
	}
	m.watchChan <- services.ArgoApiEvent{
		Type:  "auth-error",
		Error: fmt.Errorf("token expired"),
	}
	// This event should NOT be in the batch (comes after immediate)
	m.watchChan <- services.ArgoApiEvent{
		Type: "app-updated",
		App:  &model.App{Name: "app-2"},
	}

	cmd := m.consumeWatchEvents()
	msg := cmd()

	batch, ok := msg.(model.AppsBatchUpdateMsg)
	if !ok {
		t.Fatalf("expected AppsBatchUpdateMsg, got %T", msg)
	}
	if len(batch.Updates) != 1 {
		t.Errorf("expected 1 update (before auth-error), got %d", len(batch.Updates))
	}
	if batch.Immediate == nil {
		t.Fatal("expected Immediate to carry the auth-error")
	}
	if _, ok := batch.Immediate.(model.AuthErrorMsg); !ok {
		t.Errorf("expected AuthErrorMsg in Immediate, got %T", batch.Immediate)
	}
}

func TestConsumeWatchEvents_NonBatchableFirstEvent(t *testing.T) {
	m := &Model{
		watchChan: make(chan services.ArgoApiEvent, 10),
	}

	m.watchChan <- services.ArgoApiEvent{
		Type:  "auth-error",
		Error: fmt.Errorf("forbidden"),
	}

	cmd := m.consumeWatchEvents()
	msg := cmd()

	// When the first event is non-batchable, it should be returned directly
	if _, ok := msg.(model.AuthErrorMsg); !ok {
		t.Fatalf("expected AuthErrorMsg returned directly, got %T", msg)
	}
}

func TestConsumeWatchEvents_TimerFlushes(t *testing.T) {
	m := &Model{
		watchChan: make(chan services.ArgoApiEvent, 10),
	}

	// Send one event, then nothing — the 500ms timer should flush it
	m.watchChan <- services.ArgoApiEvent{
		Type: "app-updated",
		App:  &model.App{Name: "lonely-app"},
	}

	start := time.Now()
	cmd := m.consumeWatchEvents()
	msg := cmd()
	elapsed := time.Since(start)

	batch, ok := msg.(model.AppsBatchUpdateMsg)
	if !ok {
		t.Fatalf("expected AppsBatchUpdateMsg, got %T", msg)
	}
	if len(batch.Updates) != 1 {
		t.Errorf("expected 1 update, got %d", len(batch.Updates))
	}
	// Should have waited roughly 500ms for the timer
	if elapsed < 400*time.Millisecond {
		t.Errorf("expected ~500ms wait for timer flush, but only waited %v", elapsed)
	}
	if elapsed > 1*time.Second {
		t.Errorf("expected ~500ms wait, but waited %v (too long)", elapsed)
	}
}

func TestConsumeWatchEvents_NilChannel(t *testing.T) {
	m := &Model{watchChan: nil}

	cmd := m.consumeWatchEvents()
	msg := cmd()

	if msg != nil {
		t.Errorf("expected nil for nil watchChan, got %T", msg)
	}
}

func TestConsumeWatchEvents_ClosedChannel(t *testing.T) {
	ch := make(chan services.ArgoApiEvent)
	close(ch)
	m := &Model{watchChan: ch}

	cmd := m.consumeWatchEvents()
	msg := cmd()

	if msg != nil {
		t.Errorf("expected nil for closed watchChan, got %T", msg)
	}
}
