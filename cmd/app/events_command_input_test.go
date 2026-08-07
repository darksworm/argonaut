package main

import (
	"strings"
	"testing"
)

func TestValidateCommand_BareEventsIsInvalid(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	if m.validateCommand("events") {
		t.Error("bare :events should be invalid, it needs on|off")
	}
	if !m.validateCommand("events on") {
		t.Error(":events on should be valid")
	}
	if !m.validateCommand("events off always") {
		t.Error(":events off always should be valid")
	}
}

func TestValidateCommand_EventsRejectsUnknownFlags(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	invalid := []string{"events foo", "events on alwayss", "events on always extra"}
	for _, cmd := range invalid {
		if m.validateCommand(cmd) {
			t.Errorf(":%s should be invalid", cmd)
		}
	}
	if !m.validateCommand("events ON") {
		t.Error(":events ON should be valid, flags are case-insensitive")
	}
}

func TestValidateCommand_SingularEventIsUnknown(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	if m.validateCommand("event on") {
		t.Error(":event should not be accepted, only :events")
	}
}

func TestCommandBar_EventsHintsBothFlagOptions(t *testing.T) {
	tests := []struct {
		typed    string
		wantHint string
	}{
		{"ev", "ents on|off"},
		{"events", " on|off"},
		{"events ", "on|off"},
	}

	for _, tc := range tests {
		m := buildSyncTestModel(100, 30)
		m.inputComponents.SetCommandValue(tc.typed)

		rendered := m.renderCommandInputWithAutocomplete(80)

		if !strings.Contains(rendered, tc.wantHint) {
			t.Errorf("typed %q: expected hint %q, got %q", tc.typed, tc.wantHint, rendered)
		}
	}
}
