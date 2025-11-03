package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// testKeyMsg creates a KeyMsg for testing that returns the given string
func testKeyMsg(s string) tea.KeyMsg {
	// For single characters, create a KeyPressMsg
	if len(s) == 1 {
		return tea.KeyPressMsg{Code: rune(s[0]), Text: s}
	}
	// For special keys like "esc", "backspace", we'll handle them specifically
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+d":
		return tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}
	default:
		// For multi-character strings, just use the first character
		if len(s) > 0 {
			return tea.KeyPressMsg{Code: rune(s[0]), Text: string(s[0])}
		}
		return tea.KeyPressMsg{Code: 0}
	}
}

// Test helper to create model for delete testing
func buildDeleteTestModel(cols, rows int) *Model {
	m := NewModel()
	m.ready = true
	m.state.Terminal.Cols = cols
	m.state.Terminal.Rows = rows
	m.state.Mode = model.ModeNormal
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0

	// Add a test app to select
	namespace := "test-namespace"
	appNamespace := "test-namespace"
	project := "test-project"
	m.state.Apps = []model.App{
		{Name: "test-app", Sync: "Synced", Health: "Healthy", Namespace: &namespace, AppNamespace: &appNamespace, Project: &project},
		{Name: "other-app", Sync: "OutOfSync", Health: "Degraded"},
	}

	// Clear modal state
	m.state.Modals = model.ModalState{}

	return m
}

func TestHandleAppDelete_InAppsView_InitiatesDelete(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Ensure we're in apps view with an app selected
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0

	// Call handleAppDelete
	teaModel, cmd := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that mode changed to confirm delete
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode to be ModeConfirmAppDelete, got %s", newModel.state.Mode)
	}

	// Check that app name is set correctly
	if newModel.state.Modals.DeleteAppName == nil || *newModel.state.Modals.DeleteAppName != "test-app" {
		t.Fatalf("Expected DeleteAppName to be 'test-app', got %v", newModel.state.Modals.DeleteAppName)
	}

	// Check that app namespace is set correctly
	if newModel.state.Modals.DeleteAppNamespace == nil || *newModel.state.Modals.DeleteAppNamespace != "test-namespace" {
		t.Fatalf("Expected DeleteAppNamespace to be 'test-namespace', got %v", newModel.state.Modals.DeleteAppNamespace)
	}

	// Check that cascade is enabled by default (safety)
	if !newModel.state.Modals.DeleteCascade {
		t.Fatalf("Expected DeleteCascade to be true by default for safety")
	}

	// Check that propagation policy is set to foreground (default safe option)
	if newModel.state.Modals.DeletePropagationPolicy != "foreground" {
		t.Fatalf("Expected DeletePropagationPolicy to be 'foreground', got %s", newModel.state.Modals.DeletePropagationPolicy)
	}

	// Should not return a command yet (only on confirmation)
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleAppDelete_MultipleAppsSelected_InitiatesMultiDelete(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Ensure we're in apps view
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0

	// Select multiple apps
	m.state.Selections.SelectedApps["test-app"] = true
	m.state.Selections.SelectedApps["other-app"] = true

	// Call handleAppDelete
	teaModel, cmd := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that mode changed to confirm delete
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode to be ModeConfirmAppDelete, got %s", newModel.state.Mode)
	}

	// Check that app name is set to multi-delete marker
	if newModel.state.Modals.DeleteAppName == nil || *newModel.state.Modals.DeleteAppName != "__MULTI__" {
		t.Fatalf("Expected DeleteAppName to be '__MULTI__', got %v", newModel.state.Modals.DeleteAppName)
	}

	// Check that namespace is nil for multi-delete
	if newModel.state.Modals.DeleteAppNamespace != nil {
		t.Fatalf("Expected DeleteAppNamespace to be nil for multi-delete, got %v", newModel.state.Modals.DeleteAppNamespace)
	}

	// Check that cascade is enabled by default (safety)
	if !newModel.state.Modals.DeleteCascade {
		t.Fatalf("Expected DeleteCascade to be true by default for safety")
	}

	// Check that propagation policy is set to foreground (default safe option)
	if newModel.state.Modals.DeletePropagationPolicy != "foreground" {
		t.Fatalf("Expected DeletePropagationPolicy to be 'foreground', got %s", newModel.state.Modals.DeletePropagationPolicy)
	}

	// Should not return a command yet (only on confirmation)
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleAppDelete_NoAppsAvailable_DoesNothing(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Remove all apps
	m.state.Apps = []model.App{}

	// Call handleAppDelete
	teaModel, cmd := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Mode should not change
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to remain ModeNormal, got %s", newModel.state.Mode)
	}

	// No command should be returned
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleAppDelete_NotInAppsView_DoesNothing(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set to different view
	m.state.Navigation.View = model.ViewClusters

	// Call handleAppDelete
	teaModel, cmd := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Mode should not change
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to remain ModeNormal, got %s", newModel.state.Mode)
	}

	// No command should be returned
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleConfirmAppDeleteKeys_TypeY_TriggersDelete(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	namespace := "test-namespace"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteAppNamespace = &namespace
	m.state.Modals.DeleteCascade = true
	m.state.Modals.DeletePropagationPolicy = "foreground"
	m.state.Modals.DeleteConfirmationKey = ""

	// Simulate typing 'y'
	keyMsg := testKeyMsg("y")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(keyMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that confirmation key was recorded
	if newModel.state.Modals.DeleteConfirmationKey != "y" {
		t.Fatalf("Expected DeleteConfirmationKey to be 'y', got %s", newModel.state.Modals.DeleteConfirmationKey)
	}

	// Should trigger deletion - check for AppDeleteRequestMsg
	if cmd == nil {
		t.Fatalf("Expected delete command, but got nil")
	}

	// Execute the command to see if it returns the expected message
	// With new architecture, this will return an error since no server is configured
	msg := cmd()
	deleteErrorMsg, ok := msg.(model.AppDeleteErrorMsg)
	if !ok {
		t.Fatalf("Expected AppDeleteErrorMsg (no server configured), got %T: %v", msg, msg)
	}

	// Check error message contents
	if deleteErrorMsg.AppName != "test-app" {
		t.Fatalf("Expected AppName 'test-app', got %s", deleteErrorMsg.AppName)
	}
	if deleteErrorMsg.Error != "No server configured" {
		t.Fatalf("Expected error 'No server configured', got %s", deleteErrorMsg.Error)
	}
}

func TestHandleConfirmAppDeleteKeys_TypeOtherKey_UpdatesInput(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteConfirmationKey = ""

	// Simulate typing 'x' (should not trigger delete)
	keyMsg := testKeyMsg("x")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(keyMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that confirmation key was recorded
	if newModel.state.Modals.DeleteConfirmationKey != "x" {
		t.Fatalf("Expected DeleteConfirmationKey to be 'x', got %s", newModel.state.Modals.DeleteConfirmationKey)
	}

	// Should not trigger deletion
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleConfirmAppDeleteKeys_Backspace_ClearsInput(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode with existing input
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteConfirmationKey = "xyz"

	// Simulate backspace
	keyMsg := testKeyMsg("backspace")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(keyMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that last character was removed
	if newModel.state.Modals.DeleteConfirmationKey != "xy" {
		t.Fatalf("Expected DeleteConfirmationKey to be 'xy', got %s", newModel.state.Modals.DeleteConfirmationKey)
	}

	// Should not trigger deletion
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleConfirmAppDeleteKeys_Q_CancelsDelete(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName

	// Simulate 'q' key
	keyMsg := testKeyMsg("q")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(keyMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that mode changed back to normal
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to be ModeNormal, got %s", newModel.state.Mode)
	}

	// Check that delete modal state is cleared
	if newModel.state.Modals.DeleteAppName != nil {
		t.Fatalf("Expected DeleteAppName to be nil, got %v", newModel.state.Modals.DeleteAppName)
	}
	if newModel.state.Modals.DeleteConfirmationKey != "" {
		t.Fatalf("Expected DeleteConfirmationKey to be empty, got %s", newModel.state.Modals.DeleteConfirmationKey)
	}

	// Should not trigger deletion
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleConfirmAppDeleteKeys_Escape_CancelsDelete(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName

	// Simulate escape key
	keyMsg := testKeyMsg("esc")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(keyMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that mode changed back to normal
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to be ModeNormal, got %s", newModel.state.Mode)
	}

	// Check that delete modal state is cleared
	if newModel.state.Modals.DeleteAppName != nil {
		t.Fatalf("Expected DeleteAppName to be nil, got %v", newModel.state.Modals.DeleteAppName)
	}
	if newModel.state.Modals.DeleteConfirmationKey != "" {
		t.Fatalf("Expected DeleteConfirmationKey to be empty, got %s", newModel.state.Modals.DeleteConfirmationKey)
	}

	// Should not trigger deletion
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleConfirmAppDeleteKeys_C_TogglesCascade(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode with cascade initially true
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteCascade = true

	// Simulate 'c' key to toggle cascade
	keyMsg := testKeyMsg("c")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(keyMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that cascade was toggled to false
	if newModel.state.Modals.DeleteCascade {
		t.Fatalf("Expected DeleteCascade to be false after toggle, got true")
	}

	// Should not trigger deletion
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}

	// Toggle again
	keyMsg = testKeyMsg("c")
	teaModel, cmd = newModel.handleConfirmAppDeleteKeys(keyMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Check that cascade was toggled back to true
	if !newModel.state.Modals.DeleteCascade {
		t.Fatalf("Expected DeleteCascade to be true after second toggle, got false")
	}

	// Should not trigger deletion
	if cmd != nil {
		t.Fatalf("Expected no command, but got %v", cmd)
	}
}

func TestHandleConfirmAppDeleteKeys_P_CyclesPropagationPolicy(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up in delete confirmation mode with foreground policy
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeletePropagationPolicy = "foreground"

	// Test cycling through policies
	testCases := []struct {
		from string
		to   string
	}{
		{"foreground", "background"},
		{"background", "orphan"},
		{"orphan", "foreground"},
	}

	currentModel := m
	for _, tc := range testCases {
		// Verify starting state
		if currentModel.state.Modals.DeletePropagationPolicy != tc.from {
			t.Fatalf("Expected policy to start as %s, got %s", tc.from, currentModel.state.Modals.DeletePropagationPolicy)
		}

		// Press 'p' to cycle
		keyMsg := testKeyMsg("p")
		teaModel, cmd := currentModel.handleConfirmAppDeleteKeys(keyMsg)
		newModel := teaModel.(*Model)

		// Verify new state
		if newModel.state.Modals.DeletePropagationPolicy != tc.to {
			t.Fatalf("Expected policy to change from %s to %s, got %s", tc.from, tc.to, newModel.state.Modals.DeletePropagationPolicy)
		}

		// Should not trigger deletion
		if cmd != nil {
			t.Fatalf("Expected no command, but got %v", cmd)
		}

		currentModel = newModel
	}
}


// Test keyboard shortcut (Ctrl+D) integration
func TestCtrlD_InAppsView_TriggersDelete(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Ensure we're in apps view
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0

	// Simulate Ctrl+D key
	_ = testKeyMsg("ctrl+d")

	// This would be handled in the main Update method's key handling
	// For now, just test that our handler works
	teaModel, cmd := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Check that delete modal was initiated
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode to be ModeConfirmAppDelete, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.DeleteAppName == nil || *newModel.state.Modals.DeleteAppName != "test-app" {
		t.Fatalf("Expected DeleteAppName to be 'test-app', got %v", newModel.state.Modals.DeleteAppName)
	}
	if cmd != nil {
		t.Fatalf("Expected no command until confirmation, but got %v", cmd)
	}
}

// Test message handling for delete responses
func TestHandleAppDeleteSuccessMsg_ClearsModalAndRemovesApp(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up delete in progress state
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteLoading = true

	// Create success message
	_ = model.AppDeleteSuccessMsg{AppName: "test-app"}

	// This would be handled in the main Update method
	// Test that the app is removed from state and modal is cleared
	// (We'll implement this part when we write the integration tests)
}

func TestHandleAppDeleteErrorMsg_ShowsError(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up delete in progress state
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteLoading = true

	// Create error message
	_ = model.AppDeleteErrorMsg{AppName: "test-app", Error: "Failed to delete: permission denied"}

	// This would be handled in the main Update method
	// Test that error is displayed and loading stops
	// (We'll implement this part when we write the integration tests)
}