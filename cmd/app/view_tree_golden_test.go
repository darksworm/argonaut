package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
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
