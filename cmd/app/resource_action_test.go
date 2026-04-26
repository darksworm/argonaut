package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

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

	advance := func(model *Model, msg tea.KeyMsg, want int, label string) *Model {
		teaModel, _ := model.handleResourceActionKeys(msg)
		next := teaModel.(*Model)
		if next.state.Modals.ResourceAction.SelectedIdx != want {
			t.Fatalf("%s should select idx %d, got %d", label, want, next.state.Modals.ResourceAction.SelectedIdx)
		}
		return next
	}

	m = advance(m, tea.KeyPressMsg{Code: tea.KeyRight}, 1, "right arrow")
	m = advance(m, tea.KeyPressMsg{Code: tea.KeyRight}, 2, "right arrow")
	m = advance(m, tea.KeyPressMsg{Code: tea.KeyRight}, 2, "right arrow clamps at last")
	m = advance(m, tea.KeyPressMsg{Code: tea.KeyLeft}, 1, "left arrow")
	m = advance(m, tea.KeyPressMsg{Code: tea.KeyDown}, 2, "down arrow")
	m = advance(m, tea.KeyPressMsg{Code: tea.KeyUp}, 1, "up arrow")
	_ = advance(m, tea.KeyPressMsg{Code: tea.KeyLeft}, 0, "left arrow")
}

// Letters now type into the type-ahead buffer instead of being navigation
// shortcuts (no '/' prefix needed).
func TestResourceActionKeys_TypeAheadJumpsToFirstMatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "p" || st.SelectedIdx != 1 {
		t.Fatalf("typing 'p' should select promote, got filter=%q idx=%d", st.Filter, st.SelectedIdx)
	}

	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("r"))
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "pr" || st.SelectedIdx != 1 {
		t.Fatalf("typing 'pr' should keep promote selected, got filter=%q idx=%d", st.Filter, st.SelectedIdx)
	}
}

// hjkl no longer navigates — they're typeable buffer chars like any letter.
func TestResourceActionKeys_HJKLAreTypeAheadChars(t *testing.T) {
	cases := []struct {
		key  string
		want string
	}{
		{"h", "halt"},
		{"k", "kill"},
		{"l", "list"},
	}
	for _, tc := range cases {
		m := buildResourceActionTestModel(t)
		m.state.Modals.ResourceAction.Actions = []string{"abort", "halt", "kill", "list", "promote"}

		teaModel, _ := m.handleResourceActionKeys(testKeyMsg(tc.key))
		st := teaModel.(*Model).state.Modals.ResourceAction
		if st.Filter != tc.key {
			t.Errorf("%q should extend type-ahead buffer, got filter=%q", tc.key, st.Filter)
			continue
		}
		if got := st.Actions[st.SelectedIdx]; got != tc.want {
			t.Errorf("buffer %q should select %q, got %q", tc.key, tc.want, got)
		}
	}
}

// A keystroke that doesn't extend any prefix is dropped silently.
func TestResourceActionKeys_NoMatchKeystrokeDropped(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("z"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "" {
		t.Fatalf("non-matching keystroke must not enter the buffer, got %q", st.Filter)
	}
}

// Backspace clears the entire buffer (Explorer-style reset, not per-char shrink).
func TestResourceActionKeys_BackspaceClearsBuffer(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("r"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "pr" {
		t.Fatalf("setup: expected filter 'pr', got %q", st.Filter)
	}

	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("backspace"))
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "" {
		t.Fatalf("backspace should clear the buffer, got %q", st.Filter)
	}
}

// Arrow navigation is an explicit override and clears the buffer so the
// highlight goes away the moment the user takes manual control.
func TestResourceActionKeys_ArrowClearsBuffer(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter == "" {
		t.Fatalf("setup: filter should be non-empty after typing")
	}

	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(tea.KeyPressMsg{Code: tea.KeyRight})
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "" {
		t.Fatalf("arrow nav must clear the type-ahead buffer, got %q", st.Filter)
	}
}

// The decay tick clears the buffer only when no newer keypress has happened.
func TestResourceActionKeys_FilterDecayClearsOnIdle(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	st := teaModel.(*Model).state.Modals.ResourceAction
	seq := st.FilterSeq
	if st.Filter == "" || seq == 0 {
		t.Fatalf("setup: filter and seq should be set after typing")
	}

	// Decay matching the current seq clears the buffer.
	teaModel, _ = teaModel.(*Model).Update(model.ResourceActionFilterDecayMsg{Seq: seq})
	st = teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "" {
		t.Fatalf("decay with matching seq should clear filter, got %q", st.Filter)
	}
}

// 'q' closes the modal whenever no enabled action starts with 'q' — the
// common case for argo-rollouts. The action list is the only authority.
func TestResourceActionKeys_QClosesModalWhenNoActionStartsWithQ(t *testing.T) {
	m := buildResourceActionTestModel(t)
	// Default fixture: actions = abort/promote/retry — none start with 'q'.
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("q"))
	mm := teaModel.(*Model)
	if mm.state.Mode != model.ModeNormal {
		t.Fatalf("q should close modal when no action starts with q, got mode %s", mm.state.Mode)
	}
	if mm.state.Modals.ResourceAction != nil {
		t.Fatalf("q should clear ResourceAction state when closing")
	}
}

// When some action does start with 'q', q is treated as a typeable character
// and the modal stays open. This trades q-as-close for the ability to reach
// custom actions like "quarantine" via type-ahead.
func TestResourceActionKeys_QIsTypeableWhenAnActionStartsWithQ(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Actions = []string{"abort", "promote", "quarantine"}
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("q"))
	mm := teaModel.(*Model)
	st := mm.state.Modals.ResourceAction
	if st == nil || mm.state.Mode != model.ModeResourceAction {
		t.Fatalf("q should not close when an action starts with q")
	}
	if st.Filter != "q" {
		t.Fatalf("q should extend the type-ahead buffer, got filter=%q", st.Filter)
	}
	if got := st.Actions[st.SelectedIdx]; got != "quarantine" {
		t.Fatalf("typing q should select quarantine, got %q", got)
	}
}

func TestResourceActionKeys_FilterDecayIgnoredWhenStale(t *testing.T) {
	m := buildResourceActionTestModel(t)
	teaModel, _ := m.handleResourceActionKeys(testKeyMsg("p"))
	staleSeq := teaModel.(*Model).state.Modals.ResourceAction.FilterSeq
	// New keypress bumps the seq.
	teaModel, _ = teaModel.(*Model).handleResourceActionKeys(testKeyMsg("r"))

	teaModel, _ = teaModel.(*Model).Update(model.ResourceActionFilterDecayMsg{Seq: staleSeq})
	st := teaModel.(*Model).state.Modals.ResourceAction
	if st.Filter != "pr" {
		t.Fatalf("stale decay must not clear a freshly-extended buffer, got %q", st.Filter)
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
		Target:      m.state.Modals.ResourceAction.Target,
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

// While idle (no type-ahead buffer) the bottom hint nudges the user toward
// typing. Verifies the hint text is "type to select" not "Esc clear".
func TestRenderResourceActionModal_IdleHint(t *testing.T) {
	m := buildResourceActionTestModel(t)
	out := stripANSI(m.renderResourceActionModal())
	if !strings.Contains(out, "type to select") {
		t.Errorf("idle modal should hint 'type to select', got:\n%s", out)
	}
	if strings.Contains(out, "Esc clear") {
		t.Errorf("idle modal should not show 'Esc clear' yet, got:\n%s", out)
	}
}

// Once the user has typed at least one matching char, the hint flips to
// "Esc clear" so the user knows how to wipe the buffer fast.
func TestRenderResourceActionModal_TypingHint(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Filter = "p"
	out := stripANSI(m.renderResourceActionModal())
	if !strings.Contains(out, "Esc clear") {
		t.Errorf("typing modal should hint 'Esc clear', got:\n%s", out)
	}
	if strings.Contains(out, "type to select") {
		t.Errorf("typing modal should not show 'type to select', got:\n%s", out)
	}
}


// The "no actions available" overlay was bumped from a dim border to the
// same bright green border the "no differences" modal uses, so it catches
// the eye. Compare the leading ANSI escape on the top-border row of both
// modals; they must match.
func TestRenderResourceActionInfoModal_BorderMatchesNoDiffModal(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Actions = nil
	m.state.Modals.ResourceAction.Error = "No actions available for this resource"

	infoOut := m.renderResourceActionInfoModal()
	noDiffOut := m.renderNoDiffModal()

	infoBorder := extractTopBorderEscape(t, infoOut, "no-actions modal")
	noDiffBorder := extractTopBorderEscape(t, noDiffOut, "no-diff modal")
	if infoBorder != noDiffBorder {
		t.Errorf("no-actions border should match no-diff border\n  no-actions: %q\n  no-diff:    %q", infoBorder, noDiffBorder)
	}
}

// extractTopBorderEscape returns the ANSI escape preceding the rounded-
// border top-left character ╭, which corresponds to the border style.
// A late ResourceActionsErrorMsg from a previously-targeted load must not
// overwrite the modal state of a different (newer) target.
func TestUpdate_ResourceActionsErrorMsg_IgnoredOnTargetMismatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Modals.ResourceAction.Loading = true

	stale := m.state.Modals.ResourceAction.Target
	stale.Name = "previous-rollout"

	msg := model.ResourceActionsErrorMsg{
		Target:      stale,
		Error:       "stale list error",
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	st := newModel.state.Modals.ResourceAction
	if st == nil {
		t.Fatalf("modal should still be open")
	}
	if !st.Loading {
		t.Fatalf("Loading should remain true; stale error must not clear it")
	}
	if st.Error != "" {
		t.Fatalf("Error should remain empty; stale error must not be surfaced, got %q", st.Error)
	}
}

// A late ResourceActionExecutedMsg from a previously-targeted execute must not
// close the modal of a different (newer) target.
func TestUpdate_ResourceActionExecutedMsg_IgnoredOnTargetMismatch(t *testing.T) {
	m := buildResourceActionTestModel(t)
	current := m.state.Modals.ResourceAction.Target

	stale := current
	stale.Name = "previous-rollout"

	msg := model.ResourceActionExecutedMsg{
		Target:      stale,
		Action:      "promote",
		AppName:     stale.AppName,
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	if newModel.state.Mode != model.ModeResourceAction {
		t.Fatalf("mode should stay on modal; stale exec must not return to Normal, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.ResourceAction == nil {
		t.Fatalf("ResourceAction state must remain; stale exec msg should not clear it")
	}
	if newModel.state.Modals.ResourceAction.Target != current {
		t.Fatalf("Target should be untouched on stale exec")
	}
}

// The nil-server fast path in loadResourceActions must carry the current
// switch epoch, otherwise the model.go gate drops the error and the modal
// stays stuck on Loading forever after a context switch.
func TestLoadResourceActions_NoServer_PreservesSwitchEpoch(t *testing.T) {
	m := buildDeleteTestModel(100, 30)
	m.state.Server = nil
	m.switchEpoch = 7

	cmd := m.loadResourceActions(model.ResourceActionTarget{AppName: "x", Kind: "Rollout", Name: "y"})
	msg := cmd().(model.ResourceActionsErrorMsg)
	if msg.SwitchEpoch != 7 {
		t.Fatalf("expected SwitchEpoch=7, got %d", msg.SwitchEpoch)
	}
}

// Same as above for executeResourceAction.
func TestExecuteResourceAction_NoServer_PreservesSwitchEpoch(t *testing.T) {
	m := buildDeleteTestModel(100, 30)
	m.state.Server = nil
	m.switchEpoch = 11

	cmd := m.executeResourceAction(model.ResourceActionTarget{AppName: "x", Kind: "Rollout", Name: "y"}, "promote")
	msg := cmd().(model.ResourceActionExecuteErrorMsg)
	if msg.SwitchEpoch != 11 {
		t.Fatalf("expected SwitchEpoch=11, got %d", msg.SwitchEpoch)
	}
}

// handleResourceAction must resolve AppNamespace using the tree's currently
// scoped app (UI.TreeApp), not the first name-match in m.state.Apps. When two
// apps share a name across different ArgoCD namespaces, the wrong namespace
// would otherwise be sent to the API.
func TestHandleResourceAction_DisambiguatesAppByNamespace(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	nsArgocd := "argocd"
	nsTeamA := "team-a"
	// "wrong" app first in slice; tree is scoped to the second one.
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd},
		{Name: "my-app", AppNamespace: &nsTeamA},
	}
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "my-app", AppNamespace: &nsTeamA}

	m.treeView = treeview.NewTreeView(0, 0)
	deployNS := "default"
	healthy := "Healthy"
	tree := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web", Namespace: &deployNS, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("my-app", "Healthy", "Synced")
	m.treeView.UpsertAppTree("my-app", &tree)
	m.treeView.SetSelectedIndex(1) // Deployment node (0 is synthetic Application root)

	newModel, _ := m.handleResourceAction()
	m = newModel.(*Model)

	st := m.state.Modals.ResourceAction
	if st == nil {
		t.Fatalf("ResourceAction modal should be open")
	}
	if st.Target.AppNamespace == nil || *st.Target.AppNamespace != nsTeamA {
		t.Fatalf("expected target AppNamespace %q (the tree-scoped app), got %v",
			nsTeamA, st.Target.AppNamespace)
	}
}

// Closures returned by load/executeResourceAction must dereference the server
// they were created against, not whatever m.state.Server points to at the
// moment of execution. Otherwise a context switch between cmd creation and
// execution causes a destructive action to fire on the *new* context's server.
func TestExecuteResourceAction_CapturesServerAtCmdCreation(t *testing.T) {
	var serverAHits, serverBHits int32
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverAHits, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverBHits, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer serverB.Close()

	m := buildDeleteTestModel(100, 30)
	m.state.Server = &model.Server{BaseURL: serverA.URL, Token: "tok"}

	cmd := m.executeResourceAction(model.ResourceActionTarget{
		AppName: "x", Kind: "Rollout", Name: "y", Version: "v1alpha1", Group: "argoproj.io",
	}, "promote")
	if cmd == nil {
		t.Fatalf("expected a cmd")
	}

	// Simulate a context switch between cmd creation and execution.
	m.state.Server = &model.Server{BaseURL: serverB.URL, Token: "tok2"}

	_ = cmd()

	if got := atomic.LoadInt32(&serverBHits); got != 0 {
		t.Fatalf("post-switch server B must NOT receive the request; got %d hits", got)
	}
	if got := atomic.LoadInt32(&serverAHits); got != 1 {
		t.Fatalf("pre-switch server A must receive the request; got %d hits", got)
	}
}

// Same as above for loadResourceActions — a stale list shouldn't hit the new
// server (less catastrophic than a mutation, but still wrong).
func TestLoadResourceActions_CapturesServerAtCmdCreation(t *testing.T) {
	var serverAHits, serverBHits int32
	serverA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverAHits, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"actions":[]}`))
	}))
	defer serverA.Close()
	serverB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&serverBHits, 1)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"actions":[]}`))
	}))
	defer serverB.Close()

	m := buildDeleteTestModel(100, 30)
	m.state.Server = &model.Server{BaseURL: serverA.URL, Token: "tok"}

	cmd := m.loadResourceActions(model.ResourceActionTarget{
		AppName: "x", Kind: "Rollout", Name: "y", Version: "v1alpha1", Group: "argoproj.io",
	})

	m.state.Server = &model.Server{BaseURL: serverB.URL, Token: "tok2"}

	_ = cmd()

	if got := atomic.LoadInt32(&serverBHits); got != 0 {
		t.Fatalf("post-switch server B must NOT receive the request; got %d hits", got)
	}
	if got := atomic.LoadInt32(&serverAHits); got != 1 {
		t.Fatalf("pre-switch server A must receive the request; got %d hits", got)
	}
}

// After a successful execute, the post-action tree refresh must look the app
// up by both Name and AppNamespace. Otherwise, in a multi-tenant ArgoCD where
// two apps share a name across ArgoCD namespaces, the wrong app's tree gets
// reloaded and the user is left looking at stale data for the actioned app.
func TestUpdate_ResourceActionExecutedMsg_RefreshUsesAppNamespace(t *testing.T) {
	m := buildResourceActionTestModel(t)
	m.state.Navigation.View = model.ViewTree

	nsArgocd := "argocd"
	nsTeamA := "team-a"
	// "wrong" app first in slice; the actioned app is the team-a one.
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd},
		{Name: "my-app", AppNamespace: &nsTeamA},
	}

	target := model.ResourceActionTarget{
		AppName: "my-app", AppNamespace: &nsTeamA,
		Kind: "Rollout", Name: "web", Namespace: "team-a",
		Group: "argoproj.io", Version: "v1alpha1",
	}
	m.state.Modals.ResourceAction.Target = target

	msg := model.ResourceActionExecutedMsg{
		Target: target, Action: "promote", AppName: "my-app",
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ := m.Update(msg)
	newModel := teaModel.(*Model)

	// The refresh cmd needs the right app object. We can't easily inspect the
	// closure, but we can verify the Apps slice still has both entries and
	// that the refresh logic in the handler picks the team-a app by checking
	// the loadingApp helper (or a side-effect trace).
	//
	// Practical check: the handler must NOT have replaced apps[0] for the
	// refresh — if it did first-name-match, it would have built model.App from
	// apps[0] (nsArgocd). We instead verify by retrieving the resolved app via
	// a small helper exposed for this purpose.
	got := newModel.resolveActionRefreshApp(msg)
	if got == nil {
		t.Fatalf("expected a resolved app for refresh")
	}
	if got.AppNamespace == nil || *got.AppNamespace != nsTeamA {
		t.Fatalf("refresh should target the team-a app; got AppNamespace=%v", got.AppNamespace)
	}
}

func extractTopBorderEscape(t *testing.T, rendered, label string) string {
	t.Helper()
	for _, line := range strings.Split(rendered, "\n") {
		idx := strings.Index(line, "╭")
		if idx < 0 {
			continue
		}
		return line[:idx]
	}
	t.Fatalf("%s: no top-border row found in:\n%s", label, rendered)
	return ""
}
