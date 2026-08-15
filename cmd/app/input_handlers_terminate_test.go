package main

import (
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func buildTerminateTestModel(phase string) *Model {
	m := buildDeleteTestModel(80, 24)
	if phase != "" {
		m.state.Apps[0].SyncOp = &model.SyncOpSummary{Phase: phase, StartedAt: time.Now()}
	}
	return m
}

func TestTerminate_OpensConfirmationForRunningOperation(t *testing.T) {
	m := buildTerminateTestModel("Running")

	m.handleTerminateOperation()

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Fatalf("Expected mode %q, got %q", model.ModeConfirmTerminate, m.state.Mode)
	}
	st := m.state.Modals.Terminate
	if st == nil {
		t.Fatal("Expected terminate modal state, got nil")
	}
	if st.AppName != "test-app" {
		t.Errorf("Expected the modal to target test-app, got %q", st.AppName)
	}
	if st.AppNamespace == nil || *st.AppNamespace != "test-namespace" {
		t.Errorf("Expected the modal to carry the app namespace, got %v", st.AppNamespace)
	}
}

func TestTerminate_TKeyOpensConfirmationInAppsView(t *testing.T) {
	m := buildTerminateTestModel("Running")

	m.handleKeyMsg(testKeyMsg("t"))

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Errorf("Expected t to open the terminate confirmation, got mode %q", m.state.Mode)
	}
}

func TestTerminate_TKeyTargetsTheTreesAppInTreeView(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.state.Navigation.View = model.ViewTree
	// The tree carries a snapshot of the app; the running phase is only
	// current in the watched app list.
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "test-app"}

	m.handleKeyMsg(testKeyMsg("t"))

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Fatalf("Expected t to open the terminate confirmation, got mode %q", m.state.Mode)
	}
	if st := m.state.Modals.Terminate; st == nil || st.AppName != "test-app" {
		t.Errorf("Expected the modal to target the tree's app, got %+v", st)
	}
}

// The pane fetches the app directly while the list waits on a watch event, so
// just after a sync starts the pane can show Running before the list does.
// Whatever the pane advertises the hint for is what t must act on.
func TestTerminate_TrustsThePaneOverAStaleAppList(t *testing.T) {
	m := buildTerminateTestModel("Succeeded")
	m.state.Events = &model.EventsState{
		Target:  model.EventsTarget{AppName: "test-app", AppNamespace: "test-namespace"},
		Details: &model.SyncStatusDetails{Phase: "Running"},
	}

	m.handleKeyMsg(testKeyMsg("t"))

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Fatalf("Expected the pane's Running phase to allow terminating, got mode %q", m.state.Mode)
	}
	if st := m.state.Modals.Terminate; st == nil || st.AppName != "test-app" {
		t.Errorf("Expected the modal to target the pane's app, got %+v", st)
	}
}

// Sorting is by health and sync status, so a starting sync can move rows out
// from under the cursor while the pane keeps showing the app it was opened for.
func TestTerminate_TargetsThePanesAppNotTheCursor(t *testing.T) {
	m := buildTerminateTestModel("")
	m.state.Apps[1].SyncOp = &model.SyncOpSummary{Phase: "Running"}
	m.state.Navigation.SelectedIdx = 0 // cursor sits on the app that is not syncing
	m.state.Events = &model.EventsState{
		Target:  model.EventsTarget{AppName: "zzz-other-app"},
		Details: &model.SyncStatusDetails{Phase: "Running"},
	}

	m.handleKeyMsg(testKeyMsg("t"))

	if st := m.state.Modals.Terminate; st == nil || st.AppName != "zzz-other-app" {
		t.Errorf("Expected the modal to target the pane's app, got %+v", st)
	}
}

func TestTerminate_CommandOpensConfirmation(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.state.Mode = model.ModeCommand
	m.inputComponents.SetCommandValue("terminate")

	m.handleEnhancedCommandModeKeys(testKeyMsg("enter"))

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Errorf("Expected :terminate to open the confirmation, got mode %q", m.state.Mode)
	}
}

func TestTerminate_CancellingClearsTheWholeModal(t *testing.T) {
	for _, key := range []string{"esc", "q", "enter"} {
		t.Run("key="+key, func(t *testing.T) {
			m := buildTerminateTestModel("Running")
			m.handleTerminateOperation()
			m.state.Modals.Terminate.Error = "a previous attempt failed"
			if key == "enter" {
				m.state.Modals.Terminate.ConfirmSelected = 1 // Cancel
			}

			m.handleKeyMsg(testKeyMsg(key))

			if m.state.Mode != model.ModeNormal {
				t.Errorf("Expected to return to normal mode, got %q", m.state.Mode)
			}
			if m.state.Modals.Terminate != nil {
				t.Errorf("Expected the modal state to be cleared, got %+v", m.state.Modals.Terminate)
			}
		})
	}
}

func TestTerminate_ClosesTheModalOnceAccepted(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.handleTerminateOperation()
	m.state.Modals.Terminate.Loading = true

	m.Update(model.TerminateCompletedMsg{AppName: "test-app"})

	if m.state.Mode != model.ModeNormal {
		t.Errorf("Expected to return to normal mode, got %q", m.state.Mode)
	}
	if m.state.Modals.Terminate != nil {
		t.Errorf("Expected the modal state to be cleared, got %+v", m.state.Modals.Terminate)
	}
	if status := m.statusService.GetCurrentStatus(); !strings.Contains(status, "test-app") {
		t.Errorf("Expected the status line to name the app, got %q", status)
	}
}

func TestTerminate_KeepsTheModalOpenWithTheFailureReason(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.handleTerminateOperation()
	m.state.Modals.Terminate.Loading = true

	m.Update(model.TerminateCompletedMsg{AppName: "test-app", Error: "no operation is in progress"})

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Errorf("Expected the modal to stay open, got mode %q", m.state.Mode)
	}
	st := m.state.Modals.Terminate
	if st == nil {
		t.Fatal("Expected the modal state to survive the failure, got nil")
	}
	if st.Error != "no operation is in progress" {
		t.Errorf("Expected the failure reason on the modal, got %q", st.Error)
	}
	if st.Loading {
		t.Error("Expected loading to clear once the attempt finished")
	}
}

func TestTerminate_IgnoredWhenNoOperationIsRunning(t *testing.T) {
	for _, phase := range []string{"", "Succeeded", "Failed", "Error"} {
		t.Run("phase="+phase, func(t *testing.T) {
			m := buildTerminateTestModel(phase)

			m.handleTerminateOperation()

			if m.state.Mode != model.ModeNormal {
				t.Errorf("Expected to stay in normal mode, got %q", m.state.Mode)
			}
			if m.state.Modals.Terminate != nil {
				t.Errorf("Expected no terminate modal state, got %+v", m.state.Modals.Terminate)
			}
		})
	}
}

func TestTerminate_IgnoresRepeatedConfirmationWhileInFlight(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.handleTerminateOperation()

	if _, cmd := m.handleConfirmTerminateKeys(testKeyMsg("y")); cmd == nil {
		t.Fatal("Expected the first confirmation to fire a termination")
	}
	if _, cmd := m.handleConfirmTerminateKeys(testKeyMsg("y")); cmd != nil {
		t.Error("Expected no second termination while the first is in flight")
	}
	if _, cmd := m.handleConfirmTerminateKeys(testKeyMsg("enter")); cmd != nil {
		t.Error("Expected enter to be inert while a termination is in flight")
	}
}

// esc closes the modal while the request is still in flight, so its completion
// can land after the user has moved on.
func TestTerminate_LateCompletionLeavesANewerModalAlone(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.handleTerminateOperation()
	m.handleConfirmTerminateKeys(testKeyMsg("y"))
	m.handleConfirmTerminateKeys(testKeyMsg("esc"))

	// The user has moved on to a different confirmation
	m.state.Mode = model.ModeConfirmSync

	m.Update(model.TerminateCompletedMsg{AppName: "test-app"})

	if m.state.Mode != model.ModeConfirmSync {
		t.Errorf("Expected the newer modal to stay open, got mode %q", m.state.Mode)
	}
}

func TestTerminate_LateFailureDoesNotLandOnAnotherApp(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.handleTerminateOperation()
	m.state.Modals.Terminate = &model.TerminateState{AppName: "zzz-other-app", Loading: true}

	m.Update(model.TerminateCompletedMsg{AppName: "test-app", Error: "no operation is in progress"})

	st := m.state.Modals.Terminate
	if st.Error != "" {
		t.Errorf("Expected the error to stay off another app's modal, got %q", st.Error)
	}
	if !st.Loading {
		t.Error("Expected the newer attempt to still be in flight")
	}
}

// Switching Argo CD context reuses the modal state, so a completion from the
// previous context must not close a modal belonging to the new one.
func TestTerminate_CompletionFromAPreviousContextIsIgnored(t *testing.T) {
	m := buildTerminateTestModel("Running")
	m.handleTerminateOperation()
	m.state.Modals.Terminate.Loading = true
	m.switchEpoch++

	m.Update(model.TerminateCompletedMsg{AppName: "test-app", SwitchEpoch: m.switchEpoch - 1})

	if m.state.Mode != model.ModeConfirmTerminate {
		t.Errorf("Expected the modal to survive a stale completion, got mode %q", m.state.Mode)
	}
	if st := m.state.Modals.Terminate; st == nil || !st.Loading {
		t.Errorf("Expected the current attempt to stay in flight, got %+v", st)
	}
}
