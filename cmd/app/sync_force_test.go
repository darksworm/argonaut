package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

func syncModalModel(t *testing.T) *Model {
	t.Helper()
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmSync
	target := "demo-app"
	m.state.Modals.ConfirmTarget = &target
	return m
}

func press(t *testing.T, m *Model, key string) *Model {
	t.Helper()
	msg := tea.KeyPressMsg{Code: rune(key[0]), Text: key}
	if key == "esc" {
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	updated, _ := m.handleConfirmSyncKeys(msg)
	next, ok := updated.(*Model)
	if !ok {
		t.Fatalf("expected the handler to return a *Model, got %T", updated)
	}
	return next
}

func TestSyncModal_FKeyTogglesForce(t *testing.T) {
	m := syncModalModel(t)

	m = press(t, m, "f")
	if !m.state.Modals.ConfirmSyncForce {
		t.Error("expected f to turn force on")
	}

	m = press(t, m, "f")
	if m.state.Modals.ConfirmSyncForce {
		t.Error("expected f to turn force off again")
	}
}

func TestSyncModal_ConfirmingWithForceAsksAgainBeforeSyncing(t *testing.T) {
	m := syncModalModel(t)
	m.state.Modals.ConfirmSyncForce = true

	m = press(t, m, "y")

	if !m.state.Modals.ConfirmSyncForcePending {
		t.Error("expected a force sync to ask for confirmation first")
	}
	if m.state.Modals.ConfirmSyncLoading {
		t.Error("expected the sync not to start until the force confirmation is answered")
	}
}

func TestSyncModal_ConfirmingWithoutForceSyncsStraightAway(t *testing.T) {
	m := syncModalModel(t)

	m = press(t, m, "y")

	if m.state.Modals.ConfirmSyncForcePending {
		t.Error("expected no force confirmation when force is off")
	}
	if !m.state.Modals.ConfirmSyncLoading {
		t.Error("expected the sync to start immediately")
	}
}

func TestSyncModal_AnsweringTheForceConfirmationStartsTheSync(t *testing.T) {
	m := syncModalModel(t)
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncForcePending = true

	m = press(t, m, "y")

	if m.state.Modals.ConfirmSyncForcePending {
		t.Error("expected the force confirmation to close once answered")
	}
	if !m.state.Modals.ConfirmSyncLoading {
		t.Error("expected the sync to start after confirming the force")
	}
}

func TestSyncModal_CancellingTheForceConfirmationKeepsTheOptions(t *testing.T) {
	m := syncModalModel(t)
	m.state.Modals.ConfirmSyncPrune = true
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncForcePending = true

	m = press(t, m, "esc")

	if m.state.Modals.ConfirmSyncForcePending {
		t.Error("expected esc to leave the force confirmation")
	}
	if m.state.Mode != model.ModeConfirmSync {
		t.Errorf("expected to return to the sync modal, got mode %q", m.state.Mode)
	}
	if !m.state.Modals.ConfirmSyncForce || !m.state.Modals.ConfirmSyncPrune {
		t.Error("expected the chosen options to survive cancelling the force confirmation")
	}
}
