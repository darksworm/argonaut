package main

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// Starting a mouse selection forces a full repaint. If the terminal and the
// renderer's model of it have drifted apart (e.g. after a resize glitch),
// partial repaints show stale rows exactly where the user is about to
// highlight — a full rewrite at selection start makes that impossible.
func TestMouseSelectionStart_ForcesFullRepaint(t *testing.T) {
	m := buildEventsPaneTestModel()

	_, cmd := m.handleMouseClickMsg(tea.MouseClickMsg{Button: tea.MouseLeft, X: 3, Y: 2})

	if cmd == nil {
		t.Fatal("expected a command forcing the repaint")
	}
	if got, want := reflect.TypeOf(cmd()), reflect.TypeOf(tea.ClearScreen()); got != want {
		t.Errorf("expected a ClearScreen message, got %v", got)
	}
}

func TestMouseRightClick_DoesNotRepaint(t *testing.T) {
	m := buildEventsPaneTestModel()

	_, cmd := m.handleMouseClickMsg(tea.MouseClickMsg{Button: tea.MouseRight, X: 3, Y: 2})

	if cmd != nil {
		t.Error("expected no repaint on non-selection clicks")
	}
}
