package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// TestStaleAppsLoadedMsgDiscarded verifies that AppsLoadedMsg from a previous
// epoch is silently discarded.
func TestStaleAppsLoadedMsgDiscarded(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.switchEpoch = 5
	m.ready = true

	msg := model.AppsLoadedMsg{
		Apps:        []model.App{{Name: "stale-app"}},
		SwitchEpoch: 3, // old epoch
	}

	result, cmd := m.Update(msg)
	newM := result.(*Model)

	if len(newM.state.Apps) != 0 {
		t.Errorf("expected 0 apps (stale msg discarded), got %d", len(newM.state.Apps))
	}
	if cmd != nil {
		t.Error("expected nil cmd for discarded message")
	}
}

// TestCurrentAppsLoadedMsgApplied verifies that AppsLoadedMsg from the current
// epoch is applied normally.
func TestCurrentAppsLoadedMsgApplied(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.switchEpoch = 5
	m.ready = true

	msg := model.AppsLoadedMsg{
		Apps:        []model.App{{Name: "current-app"}},
		SwitchEpoch: 5, // current epoch
	}

	result, _ := m.Update(msg)
	newM := result.(*Model)

	if len(newM.state.Apps) != 1 || newM.state.Apps[0].Name != "current-app" {
		t.Errorf("expected 1 app 'current-app', got %v", newM.state.Apps)
	}
}

// TestStaleAppsBatchUpdateMsgDiscarded verifies that AppsBatchUpdateMsg from
// a previous epoch is fully discarded (no updates, no immediate re-dispatch).
func TestStaleAppsBatchUpdateMsgDiscarded(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.switchEpoch = 5
	m.ready = true
	m.state.Apps = []model.App{{Name: "existing"}}
	m.state.Index = model.BuildAppIndex(m.state.Apps)

	msg := model.AppsBatchUpdateMsg{
		Updates: []model.AppUpdatedMsg{
			{App: model.App{Name: "stale-update"}},
		},
		Deletes:     []string{"existing"},
		SwitchEpoch: 3, // old epoch
		Generation:  0,
	}

	result, cmd := m.Update(msg)
	newM := result.(*Model)

	// Apps should be unchanged
	if len(newM.state.Apps) != 1 || newM.state.Apps[0].Name != "existing" {
		t.Errorf("expected apps unchanged, got %v", newM.state.Apps)
	}
	if cmd != nil {
		t.Error("expected nil cmd for discarded batch")
	}
}

// TestStaleAuthValidationResultMsgDiscarded verifies that AuthValidationResultMsg
// from a previous epoch is discarded.
func TestStaleAuthValidationResultMsgDiscarded(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.switchEpoch = 5
	m.state.Mode = model.ModeNormal
	m.ready = true

	msg := model.AuthValidationResultMsg{
		Mode:        model.ModeAuthRequired,
		SwitchEpoch: 3, // old epoch
	}

	result, cmd := m.Update(msg)
	newM := result.(*Model)

	if newM.state.Mode != model.ModeNormal {
		t.Errorf("expected mode to remain ModeNormal, got %v", newM.state.Mode)
	}
	if cmd != nil {
		t.Error("expected nil cmd for discarded auth validation")
	}
}

// TestStaleResourceTreeLoadedMsgDiscarded verifies that ResourceTreeLoadedMsg
// from a previous epoch is discarded.
func TestStaleResourceTreeLoadedMsgDiscarded(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.switchEpoch = 5
	m.ready = true

	msg := model.ResourceTreeLoadedMsg{
		AppName:     "stale-app",
		TreeJSON:    []byte(`{"nodes":[]}`),
		SwitchEpoch: 3, // old epoch
	}

	_, cmd := m.Update(msg)
	if cmd != nil {
		t.Error("expected nil cmd for discarded tree loaded msg")
	}
}

// TestStaleAuthErrorMsgDiscarded verifies that AuthErrorMsg from a previous
// epoch is discarded.
func TestStaleAuthErrorMsgDiscarded(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.switchEpoch = 5
	m.state.Mode = model.ModeNormal
	m.ready = true

	msg := model.AuthErrorMsg{
		Error:       nil,
		SwitchEpoch: 3, // old epoch
	}

	result, cmd := m.Update(msg)
	newM := result.(*Model)

	if newM.state.Mode != model.ModeNormal {
		t.Errorf("expected mode to remain ModeNormal, got %v", newM.state.Mode)
	}
	if cmd != nil {
		t.Error("expected nil cmd for discarded auth error")
	}
}

// TestContextSwitchPreservedFields verifies that only expected fields survive
// a context switch (the preserved-fields contract).
func TestContextSwitchPreservedFields(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://old.example.com", Token: "old-token"}
	m.state.Terminal = model.TerminalState{Rows: 50, Cols: 120}
	m.ready = true
	m.argoConfigPath = "/path/to/config"
	m.currentContextName = "old-context"
	m.switchEpoch = 3

	newServer := &model.Server{BaseURL: "https://new.example.com", Token: "new-token"}
	contextNames := []string{"context-a", "context-b"}

	result, _ := m.handleContextSwitchResult(model.ContextSwitchResultMsg{
		Server:       newServer,
		ContextName:  "new-context",
		ContextNames: contextNames,
	})
	newM := result.(*Model)

	// Verify preserved fields
	if newM.state.Terminal.Rows != 50 || newM.state.Terminal.Cols != 120 {
		t.Errorf("terminal size not preserved: %+v", newM.state.Terminal)
	}
	if !newM.ready {
		t.Error("ready flag not preserved")
	}
	if newM.argoConfigPath != "/path/to/config" {
		t.Errorf("argoConfigPath not preserved: %q", newM.argoConfigPath)
	}
	if newM.currentContextName != "new-context" {
		t.Errorf("currentContextName not set: %q", newM.currentContextName)
	}
	if newM.state.Server != newServer {
		t.Error("server not set to new server")
	}
	if len(newM.state.ContextNames) != 2 {
		t.Errorf("context names not set: %v", newM.state.ContextNames)
	}
	if newM.switchEpoch != 4 {
		t.Errorf("switchEpoch not incremented: %d", newM.switchEpoch)
	}

	// Verify old state is NOT carried over
	if len(newM.state.Apps) != 0 {
		t.Errorf("old apps should not be carried: %d apps", len(newM.state.Apps))
	}
	if newM.watchChan != nil {
		t.Error("watchChan should be nil on fresh model")
	}
	if newM.watchCleanup != nil {
		t.Error("watchCleanup should be nil on fresh model")
	}
	if newM.lastResourceVersion != "" {
		t.Errorf("lastResourceVersion should be empty: %q", newM.lastResourceVersion)
	}
}

// TestContextSwitchSameContextNoOp verifies that switching to the same context
// produces a status message without teardown.
func TestContextSwitchSameContextNoOp(t *testing.T) {
	m := NewModel(nil)
	m.state.Server = &model.Server{BaseURL: "https://test.example.com", Token: "tok"}
	m.currentContextName = "my-context"
	m.argoConfigPath = "/tmp/nonexistent" // won't be read for same-context

	cmd := m.performContextSwitch("my-context")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	if statusMsg, ok := msg.(model.StatusChangeMsg); ok {
		if statusMsg.Status != "Already on context: my-context" {
			t.Errorf("unexpected status: %q", statusMsg.Status)
		}
	} else {
		t.Errorf("expected StatusChangeMsg, got %T", msg)
	}
}
