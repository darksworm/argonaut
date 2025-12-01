package main

import (
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func buildBaseModel(cols, rows int) *Model {
	m := NewModel(nil)
	m.ready = true
	m.state.Terminal.Cols = cols
	m.state.Terminal.Rows = rows
	// Provide server + version so banner/context is stable
	m.state.Server = &model.Server{BaseURL: "https://argo.example.com"}
	m.state.APIVersion = "v2.10.3"
	return m
}

// renderFullScreenView helper for tests - wraps renderFullScreenViewWithOptions
func (m *Model) renderFullScreenView(header, content, status string, contentBordered bool) string {
	return m.renderFullScreenViewWithOptions(header, content, status, FullScreenViewOptions{
		ContentBordered: contentBordered,
		BorderColor:     magentaBright,
	})
}

func TestGolden_HelpModal(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeHelp
	out := stripANSI(m.renderHelpModal())
	compareWithGolden(t, "modal_help", out)
}

func TestGolden_DiffLoadingSpinner(t *testing.T) {
	m := buildBaseModel(80, 24)
	// Use spinner initial frame deterministically
	out := stripANSI(m.renderDiffLoadingSpinner())
	compareWithGolden(t, "modal_diff_loading_spinner", out)
}

func TestGolden_SyncLoadingModal(t *testing.T) {
	m := buildBaseModel(80, 24)
	out := stripANSI(m.renderSyncLoadingModal())
	compareWithGolden(t, "modal_sync_loading", out)
}

func TestGolden_InitialLoadingModal(t *testing.T) {
	m := buildBaseModel(100, 30)
	out := stripANSI(m.renderInitialLoadingModal())
	compareWithGolden(t, "modal_initial_loading", out)
}

func sampleRollbackState() *model.RollbackState {
	// Fixed timestamps for determinism
	ts := time.Date(2024, 7, 1, 12, 34, 0, 0, time.UTC)
	author := "Jane Doe"
	msg := "Refactor and optimize"
	rev1 := "a1b2c3d4e5f6"
	rev2 := "112233445566"
	rev3 := "deadbeefcafebabe"
	rows := []model.RollbackRow{
		{ID: 30, Revision: rev1, DeployedAt: &ts, Author: &author, Message: &msg},
		{ID: 29, Revision: rev2, DeployedAt: &ts},
		{ID: 28, Revision: rev3},
	}
	cur := "cafebabedeadbeef"
	return &model.RollbackState{
		AppName:         "demo-app",
		Rows:            rows,
		SelectedIdx:     0,
		CurrentRevision: cur,
		Loading:         false,
		Error:           "",
		Mode:            "list",
		Prune:           false,
		Watch:           false,
		DryRun:          false,
		ConfirmSelected: 1,
	}
}

func TestGolden_RollbackModal_List(t *testing.T) {
	m := buildBaseModel(100, 32)
	app := "demo-app"
	m.state.Modals.RollbackAppName = &app
	m.state.Rollback = sampleRollbackState()
	out := stripANSI(m.renderRollbackModal())
	compareWithGolden(t, "modal_rollback_list", out)
}

func TestGolden_RollbackModal_Confirm(t *testing.T) {
	m := buildBaseModel(100, 32)
	app := "demo-app"
	m.state.Modals.RollbackAppName = &app
	rb := sampleRollbackState()
	rb.Mode = "confirm"
	rb.ConfirmSelected = 0
	m.state.Rollback = rb
	out := stripANSI(m.renderRollbackModal())
	compareWithGolden(t, "modal_rollback_confirm", out)
}

func TestGolden_Banner(t *testing.T) {
	m := buildBaseModel(120, 24)
	// Add some scope selections to exercise context lines
	m.state.Selections.AddCluster("prod")
	m.state.Selections.AddNamespace("payments")
	m.state.Selections.AddProject("billing")
	out := stripANSI(m.renderBanner())
	compareWithGolden(t, "banner", out)
}

func TestGolden_FullScreenLayout_Sample(t *testing.T) {
	m := buildBaseModel(80, 20)
	header := "Header Title"
	body := "Sample body line 1\nSample body line 2"
	status := "Status footer"
	out := stripANSI(m.renderFullScreenView(header, body, status, true))
	compareWithGolden(t, "layout_fullscreen_sample", out)
}

// App Delete Modal Tests

func TestGolden_AppDeleteConfirmModal(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmAppDelete

	// Set up delete modal state
	appName := "test-app"
	namespace := "production"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteAppNamespace = &namespace
	m.state.Modals.DeleteCascade = true
	m.state.Modals.DeletePropagationPolicy = "foreground"
	m.state.Modals.DeleteConfirmationKey = ""
	m.state.Modals.DeleteLoading = false
	m.state.Modals.DeleteError = nil

	out := stripANSI(m.renderAppDeleteConfirmModal())
	compareWithGolden(t, "modal_app_delete_confirm", out)
}

func TestGolden_AppDeleteConfirmModal_WithInput(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmAppDelete

	// Set up delete modal state with partial input
	appName := "my-service"
	namespace := "staging"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteAppNamespace = &namespace
	m.state.Modals.DeleteCascade = true
	m.state.Modals.DeletePropagationPolicy = "background"
	m.state.Modals.DeleteConfirmationKey = "y" // User has typed 'y'
	m.state.Modals.DeleteLoading = false
	m.state.Modals.DeleteError = nil

	out := stripANSI(m.renderAppDeleteConfirmModal())
	compareWithGolden(t, "modal_app_delete_confirm_with_input", out)
}

func TestGolden_AppDeleteLoadingModal(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmAppDelete

	// Set up delete loading state
	appName := "delete-test-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteLoading = true
	m.state.Modals.DeleteError = nil

	out := stripANSI(m.renderAppDeleteLoadingModal())
	compareWithGolden(t, "modal_app_delete_loading", out)
}

func TestGolden_AppDeleteErrorModal(t *testing.T) {
	m := buildBaseModel(100, 30)
	m.state.Mode = model.ModeConfirmAppDelete

	// Set up delete error state
	appName := "error-app"
	errorMsg := "Failed to delete application: resource not found"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteLoading = false
	m.state.Modals.DeleteError = &errorMsg

	out := stripANSI(m.renderAppDeleteConfirmModal())
	compareWithGolden(t, "modal_app_delete_error", out)
}

func TestGolden_AppDeleteConfirmModal_NoCascade(t *testing.T) {
	m := buildBaseModel(80, 24)
	m.state.Mode = model.ModeConfirmAppDelete

	// Set up delete modal with cascade disabled
	appName := "non-cascade-app"
	m.state.Modals.DeleteAppName = &appName
	m.state.Modals.DeleteAppNamespace = nil // No namespace
	m.state.Modals.DeleteCascade = false
	m.state.Modals.DeletePropagationPolicy = "orphan"
	m.state.Modals.DeleteConfirmationKey = ""
	m.state.Modals.DeleteLoading = false
	m.state.Modals.DeleteError = nil

	out := stripANSI(m.renderAppDeleteConfirmModal())
	compareWithGolden(t, "modal_app_delete_confirm_no_cascade", out)
}
