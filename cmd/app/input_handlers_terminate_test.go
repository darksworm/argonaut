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
