package main

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// syncModal renders the confirm-sync modal for a single app, with the
// toggles set as given.
func syncModal(t *testing.T, prune, watch bool) string {
	t.Helper()
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmSync
	target := "demo-app"
	m.state.Modals.ConfirmTarget = &target
	m.state.Modals.ConfirmSyncPrune = prune
	m.state.Modals.ConfirmSyncWatch = watch
	return stripANSI(m.renderConfirmSyncModal())
}

// optionLine returns the single rendered line carrying the named option.
func optionLine(t *testing.T, out, label string) string {
	t.Helper()
	var found []string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, label) {
			found = append(found, line)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected exactly one line mentioning %q, got %d:\n%s", label, len(found), out)
	}
	return found[0]
}

func TestSyncModal_GivesEachOptionItsOwnLine(t *testing.T) {
	out := syncModal(t, false, true)

	prune := optionLine(t, out, "Prune")
	if strings.Contains(prune, "Watch") {
		t.Errorf("expected Prune and Watch on separate lines, both were on:\n%s", prune)
	}
}

func TestSyncModal_AlignsOptionValuesInOneColumn(t *testing.T) {
	out := syncModal(t, false, true)

	prune := optionLine(t, out, "Prune")
	watch := optionLine(t, out, "Watch")

	pruneValue := strings.Index(prune, "Off")
	watchValue := strings.Index(watch, "On")
	if pruneValue != watchValue {
		t.Errorf("expected option values to start in the same column, got Prune at %d and Watch at %d:\n%s",
			pruneValue, watchValue, out)
	}
}

func TestSyncModal_LeadsEachOptionWithItsKey(t *testing.T) {
	out := syncModal(t, false, true)

	for _, tc := range []struct{ key, label string }{
		{"p", "Prune"},
		{"w", "Watch"},
	} {
		line := strings.TrimLeft(optionLine(t, out, tc.label), " │")
		if !strings.HasPrefix(line, tc.key+"  ") {
			t.Errorf("expected the %s row to lead with the key %q, got %q", tc.label, tc.key, line)
		}
		if strings.Contains(line, tc.key+": ") {
			t.Errorf("expected no %q prefix on the label, got %q", tc.key+":", line)
		}
	}
}

func TestSyncModal_MarksEnabledOptionsOn(t *testing.T) {
	out := syncModal(t, true, false)

	if !strings.Contains(optionLine(t, out, "Prune"), "On") {
		t.Errorf("expected Prune to read On when enabled:\n%s", out)
	}
	if !strings.Contains(optionLine(t, out, "Watch"), "Off") {
		t.Errorf("expected Watch to read Off when disabled:\n%s", out)
	}
}

func TestGolden_ConfirmSyncModal(t *testing.T) {
	out := syncModal(t, false, true)
	compareWithGolden(t, "modal_confirm_sync", out)
}

func TestGolden_ConfirmSyncModal_PruneOn(t *testing.T) {
	out := syncModal(t, true, true)
	compareWithGolden(t, "modal_confirm_sync_prune_on", out)
}

func TestSyncModal_NamesTheActionOnTheButton(t *testing.T) {
	out := syncModal(t, false, true)

	if !strings.Contains(out, "Sync") {
		t.Errorf("expected a Sync button, got:\n%s", out)
	}
	if strings.Contains(out, "Yes") {
		t.Errorf("expected the verb on the button rather than %q, got:\n%s", "Yes", out)
	}
}

func TestSyncModal_PutsOptionsAboveTheButtons(t *testing.T) {
	out := syncModal(t, false, true)

	options := strings.Index(out, "Prune")
	buttons := strings.LastIndex(out, "Cancel")
	if options > buttons {
		t.Errorf("expected the options above the buttons, options at %d and buttons at %d:\n%s",
			options, buttons, out)
	}
}

func TestSyncModal_ForceConfirmationSpellsOutWhatForceDoes(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmSync
	target := "demo-app"
	m.state.Modals.ConfirmTarget = &target
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncForcePending = true

	out := stripANSI(m.renderConfirmSyncModal())

	for _, want := range []string{"demo-app", "recreates", "Force sync", "Cancel"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected the force confirmation to mention %q, got:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Prune") {
		t.Errorf("expected the options replaced by the confirmation, got:\n%s", out)
	}
}

func TestSyncModal_ShowsForceAmongTheOptions(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmSync
	target := "demo-app"
	m.state.Modals.ConfirmTarget = &target
	m.state.Modals.ConfirmSyncForce = true

	out := stripANSI(m.renderConfirmSyncModal())

	line := optionLine(t, out, "Force")
	if !strings.Contains(line, "On") {
		t.Errorf("expected Force to read On, got %q", line)
	}
	if !strings.Contains(line, "delete") {
		t.Errorf("expected Force to spell out its consequence, got %q", line)
	}
}
