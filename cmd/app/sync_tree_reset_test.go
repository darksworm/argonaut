package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// TestSyncCompletedMsg_ResetsTreeView verifies that when a single app sync completes
// with watch enabled, the tree view is reset to a fresh instance.
// This prevents the bug where syncing app B after watching app A would show both apps.
func TestSyncCompletedMsg_ResetsTreeView(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	// Set up existing tree view with app A's data
	ns := "default"
	healthy := "Healthy"
	treeA := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "a1", Kind: "Deployment", Name: "app-a-deploy", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("app-a", "Healthy", "Synced")
	m.treeView.UpsertAppTree("app-a", &treeA)

	// Verify app A is in tree view
	if m.treeView.VisibleCount() == 0 {
		t.Fatal("Expected tree view to have app A data before sync")
	}

	// Enable watch mode
	m.state.Modals.ConfirmSyncWatch = true

	// Add app B to state so it can be found
	m.state.Apps = append(m.state.Apps, model.App{Name: "app-b"})

	// Process SyncCompletedMsg for app B
	msg := model.SyncCompletedMsg{AppName: "app-b", Success: true}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	// The tree view should be reset - a new instance with no data yet
	// (the actual app B data would be loaded asynchronously via commands)
	if m.treeView == nil {
		t.Fatal("Expected tree view to exist after sync")
	}

	// Check that the tree view is fresh (no app A data)
	// After reset, UpsertAppTree hasn't been called yet, so VisibleCount should be 0
	if m.treeView.VisibleCount() != 0 {
		t.Errorf("Expected fresh tree view with no data, got %d visible nodes", m.treeView.VisibleCount())
	}

	// Verify view changed to tree
	if m.state.Navigation.View != model.ViewTree {
		t.Errorf("Expected view to be ViewTree, got %s", m.state.Navigation.View)
	}

	// Verify TreeAppName is set to app B
	if m.state.UI.TreeAppName == nil || *m.state.UI.TreeAppName != "app-b" {
		t.Errorf("Expected TreeAppName to be 'app-b', got %v", m.state.UI.TreeAppName)
	}
}

// TestSyncCompletedMsg_WithoutWatch_DoesNotAffectTreeView verifies that sync
// without watch enabled doesn't change the tree view.
func TestSyncCompletedMsg_WithoutWatch_DoesNotAffectTreeView(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	// Set up existing tree view with app A's data
	ns := "default"
	healthy := "Healthy"
	treeA := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "a1", Kind: "Deployment", Name: "app-a-deploy", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("app-a", "Healthy", "Synced")
	m.treeView.UpsertAppTree("app-a", &treeA)

	initialCount := m.treeView.VisibleCount()

	// Watch mode disabled
	m.state.Modals.ConfirmSyncWatch = false

	// Process SyncCompletedMsg for app B
	msg := model.SyncCompletedMsg{AppName: "app-b", Success: true}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	// Tree view should still have app A's data (unchanged)
	if m.treeView.VisibleCount() != initialCount {
		t.Errorf("Expected tree view to be unchanged, had %d nodes, now has %d", initialCount, m.treeView.VisibleCount())
	}

	// View should not change to tree
	if m.state.Navigation.View == model.ViewTree {
		t.Error("Expected view to NOT change to tree when watch is disabled")
	}
}

// TestMultiSyncCompletedMsg_ResetsTreeView verifies that multi-app sync also
// resets the tree view (this was already working, but let's ensure it stays that way).
func TestMultiSyncCompletedMsg_ResetsTreeView(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	// Set up existing tree view with app A's data
	ns := "default"
	healthy := "Healthy"
	treeA := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "a1", Kind: "Deployment", Name: "app-a-deploy", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("app-a", "Healthy", "Synced")
	m.treeView.UpsertAppTree("app-a", &treeA)

	// Verify app A is in tree view
	if m.treeView.VisibleCount() == 0 {
		t.Fatal("Expected tree view to have app A data before sync")
	}

	// Enable watch mode and select multiple apps
	m.state.Modals.ConfirmSyncWatch = true
	m.state.Selections.SelectedApps["app-b"] = true
	m.state.Selections.SelectedApps["app-c"] = true

	// Add apps to state
	m.state.Apps = append(m.state.Apps,
		model.App{Name: "app-b"},
		model.App{Name: "app-c"},
	)

	// Process MultiSyncCompletedMsg
	msg := model.MultiSyncCompletedMsg{AppCount: 2, Success: true}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	// The tree view should be reset
	if m.treeView == nil {
		t.Fatal("Expected tree view to exist after multi-sync")
	}

	// Check that the tree view is fresh (no app A data)
	if m.treeView.VisibleCount() != 0 {
		t.Errorf("Expected fresh tree view with no data, got %d visible nodes", m.treeView.VisibleCount())
	}

	// Verify view changed to tree
	if m.state.Navigation.View != model.ViewTree {
		t.Errorf("Expected view to be ViewTree, got %s", m.state.Navigation.View)
	}
}

// TestRollbackExecutedMsg_ResetsTreeView verifies that when a rollback completes
// with watch enabled, the tree view is reset to a fresh instance.
func TestRollbackExecutedMsg_ResetsTreeView(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	// Set up existing tree view with app A's data
	ns := "default"
	healthy := "Healthy"
	treeA := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "a1", Kind: "Deployment", Name: "app-a-deploy", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("app-a", "Healthy", "Synced")
	m.treeView.UpsertAppTree("app-a", &treeA)

	// Verify app A is in tree view
	if m.treeView.VisibleCount() == 0 {
		t.Fatal("Expected tree view to have app A data before rollback")
	}

	// Add app B to state so it can be found
	m.state.Apps = append(m.state.Apps, model.App{Name: "app-b"})

	// Process RollbackExecutedMsg for app B with watch enabled
	msg := model.RollbackExecutedMsg{AppName: "app-b", Success: true, Watch: true}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	// The tree view should be reset
	if m.treeView == nil {
		t.Fatal("Expected tree view to exist after rollback")
	}

	// Check that the tree view is fresh (no app A data)
	if m.treeView.VisibleCount() != 0 {
		t.Errorf("Expected fresh tree view with no data, got %d visible nodes", m.treeView.VisibleCount())
	}

	// Verify view changed to tree
	if m.state.Navigation.View != model.ViewTree {
		t.Errorf("Expected view to be ViewTree, got %s", m.state.Navigation.View)
	}

	// Verify TreeAppName is set to app B
	if m.state.UI.TreeAppName == nil || *m.state.UI.TreeAppName != "app-b" {
		t.Errorf("Expected TreeAppName to be 'app-b', got %v", m.state.UI.TreeAppName)
	}
}

// buildSyncTestModel creates a model suitable for sync tests
func buildSyncTestModel(cols, rows int) *Model {
	m := NewModel(nil)
	m.ready = true
	m.state.Terminal.Cols = cols
	m.state.Terminal.Rows = rows
	m.state.Mode = model.ModeNormal
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0

	// Initialize tree view (simulating it was opened before)
	m.treeView = treeview.NewTreeView(0, 0)

	// Provide server so commands don't fail on nil checks
	m.state.Server = &model.Server{BaseURL: "https://argo.example.com"}
	m.state.APIVersion = "v2.10.3"

	return m
}
