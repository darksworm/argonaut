package main

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestSyncModal_DKeyTogglesDryRun(t *testing.T) {
	m := syncModalModel(t)

	m = press(t, m, "d")

	if !m.state.Modals.ConfirmSyncDryRun {
		t.Error("expected d to turn dry run on")
	}
}

func TestSyncModal_DryRunSkipsTheForceConfirmationBecauseNothingIsApplied(t *testing.T) {
	m := syncModalModel(t)
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncDryRun = true

	m = press(t, m, "y")

	if m.state.Modals.ConfirmSyncForcePending {
		t.Error("expected no force confirmation during a dry run — nothing is applied")
	}
	if !m.state.Modals.ConfirmSyncLoading {
		t.Error("expected the dry run to start immediately")
	}
}

func TestSyncModal_DryRunMarksForceInert(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmSync
	target := "demo-app"
	m.state.Modals.ConfirmTarget = &target
	m.state.Modals.ConfirmSyncForce = true
	m.state.Modals.ConfirmSyncDryRun = true

	out := stripANSI(m.renderConfirmSyncModal())

	force := optionLine(t, out, "Force")
	if !strings.Contains(force, "inert in dry run") {
		t.Errorf("expected force marked inert during a dry run, got %q", force)
	}
	dryRun := optionLine(t, out, "Dry run")
	if !strings.Contains(dryRun, "validates only") {
		t.Errorf("expected the dry run row to say what it does, got %q", dryRun)
	}
}
