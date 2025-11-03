package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

// TestDeleteIntegration_FullFlow tests the complete delete flow from user input to state changes
func TestDeleteIntegration_FullFlow(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Step 1: User presses Ctrl+D in apps view
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0 // "test-app"

	// Simulate Ctrl+D key press - this should call handleAppDelete
	teaModel, cmd := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Verify delete confirmation modal is shown
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode ModeConfirmAppDelete, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.DeleteAppName == nil || *newModel.state.Modals.DeleteAppName != "test-app" {
		t.Fatalf("Expected DeleteAppName 'test-app', got %v", newModel.state.Modals.DeleteAppName)
	}
	if cmd != nil {
		t.Fatalf("Expected no command until confirmation, got %v", cmd)
	}

	// Step 2: User types 'y' to confirm deletion
	keyMsg := testKeyMsg("y")
	teaModel, cmd = newModel.handleConfirmAppDeleteKeys(keyMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Verify that delete command is generated
	if cmd == nil {
		t.Fatalf("Expected delete command after typing 'y'")
	}

	// Execute the command to get the message
	// With the new architecture, this will try to delete and return an error since no server is configured
	msg := cmd()

	// Should get an error message since no server is configured in test
	deleteErrorMsg, ok := msg.(model.AppDeleteErrorMsg)
	if !ok {
		t.Fatalf("Expected AppDeleteErrorMsg (no server configured), got %T", msg)
	}

	// Verify error content
	if deleteErrorMsg.AppName != "test-app" {
		t.Fatalf("Expected AppName 'test-app', got %s", deleteErrorMsg.AppName)
	}
	if deleteErrorMsg.Error != "No server configured" {
		t.Fatalf("Expected error 'No server configured', got %s", deleteErrorMsg.Error)
	}

	// Step 3: Simulate processing the error message through Update method
	teaModel, errorCmd := newModel.Update(deleteErrorMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Verify error handling - modal should still be open with error displayed
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode to remain ModeConfirmAppDelete after error, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.DeleteError == nil || *newModel.state.Modals.DeleteError != "No server configured" {
		t.Fatalf("Expected DeleteError to be set with server error")
	}
	if newModel.state.Modals.DeleteLoading {
		t.Fatalf("Expected DeleteLoading to be false after error")
	}

	// No additional command should be returned for error handling
	if errorCmd != nil {
		t.Fatalf("Expected no command from error handling, got %v", errorCmd)
	}

	// Step 4: Test successful flow by simulating a success message
	successMsg := model.AppDeleteSuccessMsg{AppName: "test-app"}
	teaModel, successCmd := newModel.Update(successMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Verify that app is removed from the apps list
	appFound := false
	for _, app := range newModel.state.Apps {
		if app.Name == "test-app" {
			appFound = true
			break
		}
	}
	if appFound {
		t.Fatalf("Expected app 'test-app' to be removed from apps list")
	}

	// Verify that modal is cleared and mode is reset
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to be ModeNormal after successful delete, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.DeleteAppName != nil {
		t.Fatalf("Expected DeleteAppName to be nil after successful delete, got %v", newModel.state.Modals.DeleteAppName)
	}
	if newModel.state.Modals.DeleteLoading {
		t.Fatalf("Expected DeleteLoading to be false after successful delete")
	}

	// Should not return any command after successful completion
	if successCmd != nil {
		t.Fatalf("Expected no command after successful delete, got %v", successCmd)
	}
}

// TestDeleteIntegration_ErrorHandling tests error handling in the delete flow
func TestDeleteIntegration_ErrorHandling(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Set up delete confirmation mode
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteLoading = true

	// Step 1: Simulate delete error response
	errorMsg := model.AppDeleteErrorMsg{
		AppName: "test-app",
		Error:   "Failed to delete application: permission denied",
	}
	teaModel, errorCmd := m.Update(errorMsg)
	newModel := teaModel.(*Model) // Cast back to *Model

	// Verify error handling
	if newModel.state.Modals.DeleteLoading {
		t.Fatalf("Expected DeleteLoading to be false after error")
	}
	if newModel.state.Modals.DeleteError == nil {
		t.Fatalf("Expected DeleteError to be set")
	}
	if *newModel.state.Modals.DeleteError != "Failed to delete application: permission denied" {
		t.Fatalf("Expected error message to match, got %s", *newModel.state.Modals.DeleteError)
	}

	// Mode should still be confirm delete to show error
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode to remain ModeConfirmAppDelete to show error, got %s", newModel.state.Mode)
	}

	// App should still be in the list (not deleted)
	appFound := false
	for _, app := range newModel.state.Apps {
		if app.Name == "test-app" {
			appFound = true
			break
		}
	}
	if !appFound {
		t.Fatalf("Expected app 'test-app' to remain in apps list after error")
	}

	// Should not return any command after error
	if errorCmd != nil {
		t.Fatalf("Expected no command after error, got %v", errorCmd)
	}

	// Step 2: User presses Escape to dismiss error modal
	escKey := testKeyMsg("esc")
	teaModel, escCmd := newModel.handleConfirmAppDeleteKeys(escKey)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Verify modal is dismissed
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to be ModeNormal after escape, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.DeleteError != nil {
		t.Fatalf("Expected DeleteError to be cleared after escape, got %v", *newModel.state.Modals.DeleteError)
	}
	if newModel.state.Modals.DeleteAppName != nil {
		t.Fatalf("Expected DeleteAppName to be cleared after escape, got %v", *newModel.state.Modals.DeleteAppName)
	}

	if escCmd != nil {
		t.Fatalf("Expected no command after escape, got %v", escCmd)
	}
}

// TestDeleteIntegration_CancelBeforeConfirm tests canceling delete before confirmation
func TestDeleteIntegration_CancelBeforeConfirm(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Step 1: Initiate delete
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0
	teaModel, _ := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Verify modal is shown
	if newModel.state.Mode != model.ModeConfirmAppDelete {
		t.Fatalf("Expected mode ModeConfirmAppDelete, got %s", newModel.state.Mode)
	}

	// Step 2: User presses Escape to cancel
	escKey := testKeyMsg("esc")
	teaModel, cmd := newModel.handleConfirmAppDeleteKeys(escKey)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Verify cancel behavior
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to be ModeNormal after cancel, got %s", newModel.state.Mode)
	}
	if newModel.state.Modals.DeleteAppName != nil {
		t.Fatalf("Expected DeleteAppName to be cleared after cancel, got %v", newModel.state.Modals.DeleteAppName)
	}

	// App should still be in the list
	appFound := false
	for _, app := range newModel.state.Apps {
		if app.Name == "test-app" {
			appFound = true
			break
		}
	}
	if !appFound {
		t.Fatalf("Expected app 'test-app' to remain in apps list after cancel")
	}

	// Should not return any command
	if cmd != nil {
		t.Fatalf("Expected no command after cancel, got %v", cmd)
	}
}

// TestDeleteIntegration_InputValidation tests various input scenarios
func TestDeleteIntegration_InputValidation(t *testing.T) {
	m := buildDeleteTestModel(100, 30)
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName

	testCases := []struct {
		name           string
		inputRune      rune
		expectedKey    string
		shouldTrigger  bool
		description    string
	}{
		{"type_y", 'y', "y", true, "typing 'y' should trigger delete"},
		{"type_Y", 'Y', "Y", true, "typing 'Y' should trigger delete"},
		{"type_n", 'n', "n", false, "typing 'n' should not trigger delete"},
		{"type_x", 'x', "x", false, "typing 'x' should not trigger delete"},
		{"type_space", ' ', " ", false, "typing space should not trigger delete"},
		{"type_1", '1', "1", false, "typing '1' should not trigger delete"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset state
			testModel := buildDeleteTestModel(100, 30)
			testModel.state.Mode = model.ModeConfirmAppDelete
			testModel.state.Modals.DeleteAppName = &appName

			// Simulate key press
			keyMsg := testKeyMsg(string(tc.inputRune))
			teaModel, cmd := testModel.handleConfirmAppDeleteKeys(keyMsg)
			newModel := teaModel.(*Model) // Cast back to *Model

			// Check confirmation key recording
			if newModel.state.Modals.DeleteConfirmationKey != tc.expectedKey {
				t.Fatalf("Expected DeleteConfirmationKey '%s', got '%s'", tc.expectedKey, newModel.state.Modals.DeleteConfirmationKey)
			}

			// Check command generation
			if tc.shouldTrigger && cmd == nil {
				t.Fatalf("Expected command for %s, but got nil", tc.description)
			}
			if !tc.shouldTrigger && cmd != nil {
				t.Fatalf("Expected no command for %s, but got %v", tc.description, cmd)
			}
		})
	}
}

// TestDeleteIntegration_KeyboardInputHandling tests various keyboard inputs
func TestDeleteIntegration_KeyboardInputHandling(t *testing.T) {
	m := buildDeleteTestModel(100, 30)
	m.state.Mode = model.ModeConfirmAppDelete
	appName := "test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteConfirmationKey = "test"

	// Test backspace
	backspaceKey := testKeyMsg("backspace")
	teaModel, cmd := m.handleConfirmAppDeleteKeys(backspaceKey)
	newModel := teaModel.(*Model) // Cast back to *Model

	if newModel.state.Modals.DeleteConfirmationKey != "tes" {
		t.Fatalf("Expected backspace to remove last character, got %s", newModel.state.Modals.DeleteConfirmationKey)
	}
	if cmd != nil {
		t.Fatalf("Expected no command for backspace, got %v", cmd)
	}

	// Test backspace on empty string
	m.state.Modals.DeleteConfirmationKey = ""
	teaModel, cmd = m.handleConfirmAppDeleteKeys(backspaceKey)
	newModel = teaModel.(*Model) // Cast back to *Model
	if newModel.state.Modals.DeleteConfirmationKey != "" {
		t.Fatalf("Expected empty string to remain empty after backspace, got %s", newModel.state.Modals.DeleteConfirmationKey)
	}
	if cmd != nil {
		t.Fatalf("Expected no command for backspace on empty, got %v", cmd)
	}
}

// TestDeleteIntegration_StateConsistency tests that state remains consistent through the delete flow
func TestDeleteIntegration_StateConsistency(t *testing.T) {
	m := buildDeleteTestModel(100, 30)

	// Verify initial state
	initialAppCount := len(m.state.Apps)
	if initialAppCount != 2 {
		t.Fatalf("Expected 2 initial apps, got %d", initialAppCount)
	}

	// Initiate delete for first app
	m.state.Navigation.SelectedIdx = 0
	teaModel, _ := m.handleAppDelete()
	newModel := teaModel.(*Model) // Cast back to *Model

	// Verify app list is unchanged during confirmation
	if len(newModel.state.Apps) != initialAppCount {
		t.Fatalf("Expected app count to remain %d during confirmation, got %d", initialAppCount, len(newModel.state.Apps))
	}

	// Verify selected app details are correctly captured
	expectedApp := m.state.Apps[0]
	if newModel.state.Modals.DeleteAppName == nil || *newModel.state.Modals.DeleteAppName != expectedApp.Name {
		t.Fatalf("Expected DeleteAppName to match selected app '%s', got %v", expectedApp.Name, newModel.state.Modals.DeleteAppName)
	}
	if newModel.state.Modals.DeleteAppNamespace == nil || *newModel.state.Modals.DeleteAppNamespace != *expectedApp.Namespace {
		t.Fatalf("Expected DeleteAppNamespace to match selected app '%s', got %v", *expectedApp.Namespace, newModel.state.Modals.DeleteAppNamespace)
	}

	// Complete delete confirmation
	keyMsg := testKeyMsg("y")
	teaModel, cmd := newModel.handleConfirmAppDeleteKeys(keyMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	if cmd == nil {
		t.Fatalf("Expected delete command")
	}

	// Process delete attempt (will get error since no server configured)
	deleteMsg := cmd()
	if _, ok := deleteMsg.(model.AppDeleteErrorMsg); !ok {
		t.Fatalf("Expected AppDeleteErrorMsg due to no server, got %T", deleteMsg)
	}
	teaModel, _ = newModel.Update(deleteMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Process successful delete response (simulating what would happen with real server)
	successMsg := model.AppDeleteSuccessMsg{AppName: expectedApp.Name}
	teaModel, _ = newModel.Update(successMsg)
	newModel = teaModel.(*Model) // Cast back to *Model

	// Verify final state consistency
	if len(newModel.state.Apps) != initialAppCount-1 {
		t.Fatalf("Expected app count to be %d after delete, got %d", initialAppCount-1, len(newModel.state.Apps))
	}

	// Verify the correct app was removed
	for _, app := range newModel.state.Apps {
		if app.Name == expectedApp.Name {
			t.Fatalf("Expected app '%s' to be removed, but it's still in the list", expectedApp.Name)
		}
	}

	// Verify modal state is completely cleared
	if newModel.state.Modals.DeleteAppName != nil ||
		newModel.state.Modals.DeleteAppNamespace != nil ||
		newModel.state.Modals.DeleteConfirmationKey != "" ||
		newModel.state.Modals.DeleteLoading ||
		newModel.state.Modals.DeleteError != nil {
		t.Fatalf("Expected all delete modal state to be cleared after successful delete")
	}

	// Verify UI state is back to normal
	if newModel.state.Mode != model.ModeNormal {
		t.Fatalf("Expected mode to be ModeNormal after delete, got %s", newModel.state.Mode)
	}
}