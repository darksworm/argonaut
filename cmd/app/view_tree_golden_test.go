package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
)

func TestGolden_TreeView_SelectionStyle(t *testing.T) {
	m := buildBaseModel(100, 24)
	// Switch to tree view
	m.state.Navigation.View = "tree"
	// Load a simple tree with two roots that will attach under the synthetic app root
	ns := "ns-a"
	healthy := "Healthy"
	tree := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "s1", Kind: "Service", Name: "web", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("demo-app", "Healthy", "Synced")
	m.treeView.SetData(&tree)

	// Render only the bordered tree panel area; default selection is the synthetic root (Application)
	out := m.renderTreePanel(12)
	// Keep ANSI to capture highlight background codes in golden
	compareWithGolden(t, "tree_view_selection", out)
}

// TestGolden_TreeView_CtrlD_SingleResource tests the tree view with Ctrl+D pressed on a single resource.
// It verifies that:
// - The tree view highlight stays visible (in desaturated mode)
// - The delete confirmation popup shows the correct single resource info
func TestGolden_TreeView_CtrlD_SingleResource(t *testing.T) {
	m := buildBaseModel(100, 24)
	m.state.Navigation.View = model.ViewTree
	m.state.Mode = model.ModeConfirmResourceDelete

	// Load tree with resources
	ns := "production"
	healthy := "Healthy"
	synced := "Synced"
	tree := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "api-server", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "s1", Kind: "Service", Name: "api-server", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "cm1", Kind: "ConfigMap", Name: "app-config", Namespace: &ns, Health: &api.ResourceHealth{Status: &synced}},
	}}
	m.treeView.SetAppMeta("my-app", "Healthy", "Synced")
	m.treeView.SetData(&tree)

	// Move cursor to Deployment (index 1 after Application root)
	m.treeView.SetSelectedIndex(1)

	// Set up resource delete modal state (single resource - the cursor position)
	appName := "my-app"
	m.state.Modals.ResourceDeleteAppName = &appName
	m.state.Modals.ResourceDeleteTargets = []model.ResourceDeleteTarget{
		{AppName: "my-app", Kind: "Deployment", Namespace: "production", Name: "api-server"},
	}
	m.state.Modals.ResourceDeleteCascade = true
	m.state.Modals.ResourceDeletePropagationPolicy = "foreground"
	m.state.Modals.ResourceDeleteConfirmationKey = ""
	m.state.Modals.ResourceDeleteLoading = false

	// Render full view to capture overlay composition (use renderMainLayout which returns string)
	out := m.renderMainLayout()
	// Strip ANSI for stable golden - the key test is logical content
	compareWithGolden(t, "tree_view_ctrld_single_resource", stripANSI(out))
}

// TestGolden_TreeView_CtrlD_MultiSelect tests the tree view with Ctrl+D pressed after multi-selecting resources.
// It verifies that:
// - The tree view shows multiple selected items highlighted
// - The delete confirmation popup shows the count of selected resources
func TestGolden_TreeView_CtrlD_MultiSelect(t *testing.T) {
	m := buildBaseModel(100, 24)
	m.state.Navigation.View = model.ViewTree
	m.state.Mode = model.ModeConfirmResourceDelete

	// Load tree with resources
	ns := "staging"
	healthy := "Healthy"
	tree := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "frontend", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "d2", Kind: "Deployment", Name: "backend", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "s1", Kind: "Service", Name: "frontend-svc", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "s2", Kind: "Service", Name: "backend-svc", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("multi-app", "Healthy", "Synced")
	m.treeView.SetData(&tree)

	// Simulate multi-selection by toggling multiple resources
	m.treeView.SetSelectedIndex(1) // Move to first Deployment
	m.treeView.ToggleSelection()
	m.treeView.SetSelectedIndex(2) // Move to second Deployment
	m.treeView.ToggleSelection()
	m.treeView.SetSelectedIndex(3) // Move to first Service
	m.treeView.ToggleSelection()

	// Set up resource delete modal state (multiple resources)
	appName := "multi-app"
	m.state.Modals.ResourceDeleteAppName = &appName
	m.state.Modals.ResourceDeleteTargets = []model.ResourceDeleteTarget{
		{AppName: "multi-app", Kind: "Deployment", Namespace: "staging", Name: "frontend"},
		{AppName: "multi-app", Kind: "Deployment", Namespace: "staging", Name: "backend"},
		{AppName: "multi-app", Kind: "Service", Namespace: "staging", Name: "frontend-svc"},
	}
	m.state.Modals.ResourceDeleteCascade = true
	m.state.Modals.ResourceDeletePropagationPolicy = "background"
	m.state.Modals.ResourceDeleteConfirmationKey = ""
	m.state.Modals.ResourceDeleteLoading = false

	// Render full view to capture overlay composition
	out := m.renderMainLayout()
	// Strip ANSI for stable golden
	compareWithGolden(t, "tree_view_ctrld_multi_select", stripANSI(out))
}

// TestGolden_TreeView_CtrlD_HighlightPreserved tests that the highlight styling is preserved
// when the delete modal is shown. This test keeps ANSI codes to verify the highlight background.
func TestGolden_TreeView_CtrlD_HighlightPreserved(t *testing.T) {
	m := buildBaseModel(100, 24)
	m.state.Navigation.View = model.ViewTree
	m.state.Mode = model.ModeConfirmResourceDelete

	// Load tree with resources
	ns := "default"
	healthy := "Healthy"
	tree := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
		{UID: "s1", Kind: "Service", Name: "web", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("highlight-test", "Healthy", "Synced")
	m.treeView.SetData(&tree)

	// Move to Deployment and toggle selection
	m.treeView.SetSelectedIndex(1)
	m.treeView.ToggleSelection()

	// Set up resource delete modal state
	appName := "highlight-test"
	m.state.Modals.ResourceDeleteAppName = &appName
	m.state.Modals.ResourceDeleteTargets = []model.ResourceDeleteTarget{
		{AppName: "highlight-test", Kind: "Deployment", Namespace: "default", Name: "web"},
	}
	m.state.Modals.ResourceDeleteCascade = true
	m.state.Modals.ResourceDeletePropagationPolicy = "foreground"

	// Render tree panel only with desaturate mode enabled (as it would be during delete modal)
	m.treeView.SetDesaturateMode(true)
	out := m.renderTreePanel(12)
	// Keep ANSI to verify highlight styling is preserved
	compareWithGolden(t, "tree_view_ctrld_highlight_preserved", out)
}
