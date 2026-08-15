package main

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestTerminateModal_AsksAboutTheAppUnderTheCursor(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate
	m.state.Modals.Terminate = &model.TerminateState{AppName: "demo-app"}

	out := stripANSI(m.renderTerminateConfirmModal())

	if !strings.Contains(out, "demo-app") {
		t.Errorf("Expected the modal to name the app, got:\n%s", out)
	}
	if !strings.Contains(out, "Terminate") {
		t.Errorf("Expected a Terminate button, got:\n%s", out)
	}
	if !strings.Contains(out, "Cancel") {
		t.Errorf("Expected a Cancel button, got:\n%s", out)
	}
}

func TestTerminateModal_IsTheActiveOverlayWhileConfirming(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate
	m.state.Modals.Terminate = &model.TerminateState{AppName: "demo-app"}

	overlay := m.activeOverlay()
	if overlay == nil {
		t.Fatal("Expected the terminate modal to be the active overlay, got none")
	}
	if !strings.Contains(stripANSI(overlay.modal), "demo-app") {
		t.Errorf("Expected the confirmation on screen, got:\n%s", stripANSI(overlay.modal))
	}

	m.state.Modals.Terminate.Loading = true
	overlay = m.activeOverlay()
	if overlay == nil || !strings.Contains(stripANSI(overlay.modal), "Terminating operation") {
		t.Errorf("Expected the in-flight modal while loading, got:\n%v", overlay)
	}
}

func TestGolden_TerminateModal_Confirm(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate
	m.state.Modals.Terminate = &model.TerminateState{AppName: "demo-app"}
	compareWithGolden(t, "modal_terminate_confirm", stripANSI(m.renderTerminateConfirmModal()))
}

// stripANSI would erase the highlight that is the entire difference, so this
// compares the styled output instead.
func TestTerminateModal_HighlightsTheSelectedButton(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate

	m.state.Modals.Terminate = &model.TerminateState{AppName: "demo-app"}
	terminateSelected := m.renderTerminateConfirmModal()

	m.state.Modals.Terminate = &model.TerminateState{AppName: "demo-app", ConfirmSelected: 1}
	cancelSelected := m.renderTerminateConfirmModal()

	if terminateSelected == cancelSelected {
		t.Error("Expected the highlight to move with ConfirmSelected, got identical renders")
	}
	if stripANSI(terminateSelected) != stripANSI(cancelSelected) {
		t.Error("Expected only styling to differ between the two selections")
	}
}

func TestGolden_TerminateModal_Error(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate
	m.state.Modals.Terminate = &model.TerminateState{
		AppName: "demo-app",
		Error:   "Unable to terminate operation. No operation is in progress",
	}
	compareWithGolden(t, "modal_terminate_error", stripANSI(m.renderTerminateConfirmModal()))
}

func TestTerminateModal_ShowsWhyTheAttemptFailed(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate
	m.state.Modals.Terminate = &model.TerminateState{
		AppName: "demo-app",
		Error:   "no operation is in progress",
	}

	out := stripANSI(m.renderTerminateConfirmModal())

	if !strings.Contains(out, "no operation is in progress") {
		t.Errorf("Expected the failure reason in the modal, got:\n%s", out)
	}
}
