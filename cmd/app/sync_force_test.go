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

func TestSyncModal_CancelButtonOnTheForceConfirmationKeepsTheOptions(t *testing.T) {
	m := syncModalModel(t)
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncForcePending = true
	m.state.Modals.ConfirmSyncSelected = 1 // Cancel

	updated, _ := m.handleConfirmSyncKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(*Model)

	if m.state.Mode != model.ModeConfirmSync {
		t.Errorf("expected cancelling the force confirmation to return to the options, got mode %q", m.state.Mode)
	}
	if m.state.Modals.ConfirmSyncForcePending {
		t.Error("expected the force confirmation to close")
	}
	if !m.state.Modals.ConfirmSyncForce {
		t.Error("expected force to survive cancelling its confirmation")
	}
}

func TestSyncModal_OptionKeysDoNothingWhileTheForceConfirmationIsUp(t *testing.T) {
	for _, key := range []string{"p", "f", "w", "d"} {
		t.Run(key, func(t *testing.T) {
			m := syncModalModel(t)
			m.state.Modals.ConfirmSyncPrune = true
			m.state.Modals.ConfirmSyncForce = true
			m.state.Modals.ConfirmSyncWatch = true
			m.state.Modals.ConfirmSyncForcePending = true
			before := m.state.Modals

			m = press(t, m, key)

			if m.state.Modals.ConfirmSyncPrune != before.ConfirmSyncPrune ||
				m.state.Modals.ConfirmSyncForce != before.ConfirmSyncForce ||
				m.state.Modals.ConfirmSyncWatch != before.ConfirmSyncWatch {
				t.Errorf("expected %q to be ignored while the force confirmation is up", key)
			}
		})
	}
}

func TestSyncModal_ConfirmingTheForceDialogAlwaysForces(t *testing.T) {
	m := syncModalModel(t)
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncForcePending = true

	// A stray f must not disarm the force the dialog is asking about.
	m = press(t, m, "f")
	m = press(t, m, "y")

	if !m.state.Modals.ConfirmSyncForce {
		t.Error("expected the sync to force after confirming a dialog that said Force sync")
	}
}
