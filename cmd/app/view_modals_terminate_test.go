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

func TestGolden_TerminateModal_CancelSelected(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmTerminate
	m.state.Modals.Terminate = &model.TerminateState{AppName: "demo-app", ConfirmSelected: 1}
	compareWithGolden(t, "modal_terminate_cancel_selected", stripANSI(m.renderTerminateConfirmModal()))
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
