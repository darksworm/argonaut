package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// buildResourceActionTestModel returns a model pre-set into the resource action
// modal with a fake target and a couple of fake actions loaded. Actions are
// stored sorted alphabetically to match what production sees after the load
// handler runs. Tests mutate the returned model to exercise specific flows.
func buildResourceActionTestModel(t *testing.T) *Model {
	t.Helper()
	m := buildDeleteTestModel(100, 30)
	m.state.Mode = model.ModeResourceAction
	m.state.Modals.ResourceAction = &model.ResourceActionState{
		Target: model.ResourceActionTarget{
			AppName:   "test-app",
			Group:     "argoproj.io",
			Version:   "v1alpha1",
			Kind:      "Rollout",
			Namespace: "test-namespace",
			Name:      "web",
		},
		Actions:     []string{"abort", "promote", "retry"},
		SelectedIdx: 0,
	}
	return m
}

func TestResourceActionKeys_ArrowNavigation(t *testing.T) {
	m := buildResourceActionTestModel(t)

	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("j"))
	newModel := teaModel.(*Model)
	if newModel.state.Modals.ResourceAction.SelectedIdx != 1 {
		t.Fatalf("j should move cursor down to 1, got %d", newModel.state.Modals.ResourceAction.SelectedIdx)
	}

	teaModel, _ = newModel.handleResourceActionKeys(testKeyMsg("j"))
	newModel = teaModel.(*Model)
	teaModel, _ = newModel.handleResourceActionKeys(testKeyMsg("j"))
	newModel = teaModel.(*Model)
	// Should clamp at last index (2).
	if newModel.state.Modals.ResourceAction.SelectedIdx != 2 {
		t.Fatalf("cursor should clamp at last index 2, got %d", newModel.state.Modals.ResourceAction.SelectedIdx)
	}

	teaModel, _ = newModel.handleResourceActionKeys(testKeyMsg("k"))
	newModel = teaModel.(*Model)
	if newModel.state.Modals.ResourceAction.SelectedIdx != 1 {
		t.Fatalf("k should move cursor up to 1, got %d", newModel.state.Modals.ResourceAction.SelectedIdx)
	}

	teaModel, _ = newModel.handleResourceActionKeys(tea.KeyPressMsg{Code: tea.KeyLeft})
	newModel = teaModel.(*Model)
	if newModel.state.Modals.ResourceAction.SelectedIdx != 0 {
		t.Fatalf("left arrow should move cursor to 0, got %d", newModel.state.Modals.ResourceAction.SelectedIdx)
	}

	teaModel, _ = newModel.handleResourceActionKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	newModel = teaModel.(*Model)
	if newModel.state.Modals.ResourceAction.SelectedIdx != 1 {
		t.Fatalf("right arrow should move cursor to 1, got %d", newModel.state.Modals.ResourceAction.SelectedIdx)
	}
}

func TestResourceActionKeys_TypeToFilter_SelectsFirstMatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	// Actions are alphabetical: abort, promote, retry.
	// Typing 'p' should jump from abort to promote.
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "p" {
		t.Fatalf("filter should be 'p', got %q", st.Filter)
	}
	if st.SelectedIdx != 1 {
		t.Fatalf("typing 'p' should select promote (idx 1), got %d", st.SelectedIdx)
	}

	// Typing another char that still matches keeps the same selection.
	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("r"))
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "pr" {
		t.Fatalf("filter should be 'pr', got %q", st.Filter)
	}
	if st.SelectedIdx != 1 {
		t.Fatalf("typing 'pr' should still select promote, got idx %d", st.SelectedIdx)
	}
}

func TestResourceActionKeys_TypeToFilter_RejectsNoMatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	// Type a char that no action starts with — keystroke should be rejected
	// so the user can't get stuck on a filter that selects nothing.
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("z"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "" {
		t.Fatalf("filter should remain empty on no-match, got %q", st.Filter)
	}
	if st.SelectedIdx != 0 {
		t.Fatalf("selection should not move on no-match, got idx %d", st.SelectedIdx)
	}
}

func TestResourceActionKeys_BackspaceShrinksFilter(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("r"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "pr" {
		t.Fatalf("setup: expected filter 'pr', got %q", st.Filter)
	}

	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("backspace"))
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "p" {
		t.Fatalf("backspace should shrink filter to 'p', got %q", st.Filter)
	}
	if st.SelectedIdx != 1 {
		t.Fatalf("filter 'p' should still select promote (idx 1), got %d", st.SelectedIdx)
	}

	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("backspace"))
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "" {
		t.Fatalf("backspace should clear filter, got %q", st.Filter)
	}
}

func TestResourceActionKeys_EscClearsFilterBeforeClosing(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter == "" {
		t.Fatalf("setup: filter should be set after typing 'p'")
	}

	// First Esc clears the filter but keeps the modal open.
	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("esc"))
	newModel := teaModel.(*Model)
	if newModel.state.Mode != model.ModeResourceAction {
		t.Fatalf("first Esc should keep modal open while filter is active, got mode %s", newModel.state.Mode)
	}
	if newModel.state.Modals.ResourceAction == nil {
		t.Fatalf("first Esc should not destroy the modal state")
	}
	if newModel.state.Modals.ResourceAction.Filter != "" {
		t.Fatalf("first Esc should clear the filter, got %q", newModel.state.Modals.ResourceAction.Filter)
	}

	// Second Esc closes the modal.
	teaModel, _ = newModel.handleResourceActionKeys(testKeyMsg("esc"))
	newModel = teaModel.(*Model)
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("second Esc should close the modal, got mode %s", newModel.state.Mode)
	}
}

func TestResourceActionKeys_EmptyActionsEnterClosesModal(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Actions = nil
	m.state.Modals.ResourceAction.Error = "No actions available for this resource"

	teaModel, _ := m.handleResourceActionKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	newModel := teaModel.(*Model)
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Enter on the empty/error modal should close it, got mode %s", newModel.state.Mode)
	}
	if newModel.state.Modals.ResourceAction != nil {
		t.Fatalf("Enter on the empty/error modal should clear the modal state")
	}
}

func TestResourceActionKeys_EscClosesModal(t *testing.T) {
	m := buildResourceActionTestModel(t)

	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("esc"))
	newModel := teaModel.(*Model)

	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Esc should return to ModeNormal, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.ResourceAction != nil {
		t.Fatalf("Esc should clear ResourceAction state")
	}
}

func TestResourceActionKeys_LoadingIgnoresNavigation(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Loading = true
	m.state.Modals.ResourceAction.Actions = nil
	m.state.Modals.ResourceAction.SelectedIdx = 0

	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("j"))
	newModel := teaModel.(*Model)
	if newModel.state.Modals.ResourceAction.SelectedIdx != 0 {
		t.Fatalf("navigation should be ignored while loading, got idx %d",
			newModel.state.Modals.ResourceAction.SelectedIdx)
	}
	if newModel.state.Mode != model.ModeResourceAction {
		t.Fatalf("mode should remain ModeResourceAction while loading, got %s", newModel.state.Mode)
	}
}

func TestResourceActionKeys_EnterWithNoServerReturnsErrorMsg(t *testing.T) {
	m := buildResourceActionTestModel(t)

	teaModel, cmd := m.handleResourceActionKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	newModel := teaModel.(*Model)

	if !newModel.state.Modals.ResourceAction.Executing {
		t.Fatalf("Enter should set Executing=true")
	}
	if cmd == nil {
		t.Fatalf("Enter should return a command")
	}
	msg := cmd()
	errMsg, ok := msg.(model.ResourceActionExecuteErrorMsg)
	if !ok {
		t.Fatalf("expected ResourceActionExecuteErrorMsg (no server configured), got %T", msg)
	}
	if errMsg.Error != "No server configured" {
		t.Fatalf("unexpected error message: %q", errMsg.Error)
	}
}

func TestUpdate_ResourceActionsLoadedMsg_PopulatesModal(t *testing.T) {
	m := buildResourceActionTestModel(t)
	target := m.state.Modals.ResourceAction.Target
	m.state.Modals.ResourceAction.Loading = true
	m.state.Modals.ResourceAction.Actions = nil

	msg := model.ResourceActionsLoadedMsg{
		Target:      target,
		Actions:     []string{"promote-full", "pause"},
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	st := newModel.state.Modals.ResourceAction
	if st == nil {
		t.Fatalf("ResourceAction state should still exist after load")
	}
	if st.Loading {
		t.Fatalf("Loading should be false after load")
	}
	// Actions should be sorted alphabetically so the order is deterministic.
	if len(st.Actions) != 2 || st.Actions[0] != "pause" || st.Actions[1] != "promote-full" {
		t.Fatalf("Actions should be sorted alphabetically, got: %v", st.Actions)
	}
	if st.SelectedIdx != 0 {
		t.Fatalf("SelectedIdx should reset to 0, got %d", st.SelectedIdx)
	}
}

func TestUpdate_ResourceActionsLoadedMsg_EmptyShowsInlineError(t *testing.T) {
	m := buildResourceActionTestModel(t)
	target := m.state.Modals.ResourceAction.Target
	m.state.Modals.ResourceAction.Loading = true
	m.state.Modals.ResourceAction.Actions = nil

	msg := model.ResourceActionsLoadedMsg{
		Target:      target,
		Actions:     []string{},
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	st := newModel.state.Modals.ResourceAction
	if st.Error == "" {
		t.Fatalf("expected inline error when no actions are available")
	}
}

func TestUpdate_ResourceActionsLoadedMsg_IgnoredOnEpochMismatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	target := m.state.Modals.ResourceAction.Target
	m.state.Modals.ResourceAction.Loading = true

	msg := model.ResourceActionsLoadedMsg{
		Target:      target,
		Actions:     []string{"promote"},
		SwitchEpoch: m.switchEpoch + 1, // stale
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	if !newModel.state.Modals.ResourceAction.Loading {
		t.Fatalf("stale message must not apply; loading should remain true")
	}
}

func TestUpdate_ResourceActionsErrorMsg_SurfacesError(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Loading = true

	msg := model.ResourceActionsErrorMsg{
		Error:       "Forbidden: user cannot list actions",
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	st := newModel.state.Modals.ResourceAction
	if st.Loading {
		t.Fatalf("Loading should clear on error")
	}
	if st.Error == "" {
		t.Fatalf("Error should be set")
	}
	if newModel.state.Mode != model.ModeResourceAction {
		t.Fatalf("mode should stay on modal to display the error")
	}
}

func TestUpdate_ResourceActionExecutedMsg_ClosesModal(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Executing = true

	msg := model.ResourceActionExecutedMsg{
		Target:      m.state.Modals.ResourceAction.Target,
		Action:      "promote",
		AppName:     m.state.Modals.ResourceAction.Target.AppName,
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("mode should return to Normal after action executed, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.ResourceAction != nil {
		t.Fatalf("ResourceAction state should be cleared after success")
	}
}

func TestUpdate_ResourceActionExecuteErrorMsg_KeepsModalOpen(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Executing = true

	msg := model.ResourceActionExecuteErrorMsg{
		Target:      m.state.Modals.ResourceAction.Target,
		Error:       "rpc error: code = InvalidArgument",
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	st := newModel.state.Modals.ResourceAction
	if st == nil {
		t.Fatalf("modal should stay open to display error")
	}
	if st.Executing {
		t.Fatalf("Executing should clear after error")
	}
	if st.Error == "" {
		t.Fatalf("Error should be set")
	}
	if newModel.state.Mode != model.ModeResourceAction {
		t.Fatalf("mode should remain ModeResourceAction after error, got %s", newModel.state.Mode)
	}
}

func TestUpdate_ResourceActionExecuteErrorMsg_IgnoredOnTargetMismatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Executing = true

	stale := m.state.Modals.ResourceAction.Target
	stale.Name = "some-other-rollout"

	msg := model.ResourceActionExecuteErrorMsg{
		Target:      stale,
		Error:       "stale error from previous modal",
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	st := newModel.state.Modals.ResourceAction
	if st == nil {
		t.Fatalf("modal should still be open")
	}
	if !st.Executing {
		t.Fatalf("Executing should remain true; stale error must not clear it")
	}
	if st.Error != "" {
		t.Fatalf("Error should remain empty; stale error must not be surfaced")
	}
}

func TestLoadResourceActions_NoServer_ReturnsError(t *testing.T) {
	m := buildDeleteTestModel(100, 30)
	m.state.Server = nil

	cmd := m.loadResourceActions(model.ResourceActionTarget{AppName: "x", Kind: "Rollout", Name: "y"})
	if cmd == nil {
		t.Fatalf("expected a command returning an error msg")
	}
	msg := cmd()
	errMsg, ok := msg.(model.ResourceActionsErrorMsg)
	if !ok {
		t.Fatalf("expected ResourceActionsErrorMsg, got %T", msg)
	}
	if errMsg.Error != "No server configured" {
		t.Fatalf("unexpected error: %q", errMsg.Error)
	}
}

func TestExecuteResourceAction_NoServer_ReturnsError(t *testing.T) {
	m := buildDeleteTestModel(100, 30)
	m.state.Server = nil

	cmd := m.executeResourceAction(model.ResourceActionTarget{AppName: "x", Kind: "Rollout", Name: "y"}, "promote")
	if cmd == nil {
		t.Fatalf("expected a command returning an error msg")
	}
	msg := cmd()
	errMsg, ok := msg.(model.ResourceActionExecuteErrorMsg)
	if !ok {
		t.Fatalf("expected ResourceActionExecuteErrorMsg, got %T", msg)
	}
	if errMsg.Error != "No server configured" {
		t.Fatalf("unexpected error: %q", errMsg.Error)
	}
}

func TestRenderResourceActionModal_Smoke(t *testing.T) {
	m := buildResourceActionTestModel(t)
	// Need a spinner for the loading branch to render without panic; cover list path.
	out := m.renderResourceActionModal()
	if out == "" {
		t.Fatalf("modal render returned empty output")
	}
	// The selected action should be in the output.
	for _, name := range []string{"promote", "abort", "retry", "Rollout"} {
		if !stringContains(out, name) {
			t.Errorf("modal output missing %q", name)
		}
	}
}

func stringContains(s, sub string) bool {
	if sub == "" {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
