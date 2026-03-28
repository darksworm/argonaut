package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

func TestHandleNavigateToChildApp_UsesNamespaceDisambiguation(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	parentNS := "argocd"
	childNSA := "team-a"
	childNSB := "team-b"

	m.state.Apps = []model.App{
		{Name: "parent", AppNamespace: &parentNS},
		{Name: "child", AppNamespace: &childNSA},
		{Name: "child", AppNamespace: &childNSB},
	}
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeAppName = &m.state.Apps[0].Name
	m.state.UI.TreeAppNamespace = m.state.Apps[0].AppNamespace
	m.treeView = treeview.NewTreeView(0, 0)

	_, _ = m.handleNavigateToChildApp("child", childNSB)

	if m.state.UI.TreeAppName == nil || *m.state.UI.TreeAppName != "child" {
		t.Fatalf("expected child app to be opened, got %v", m.state.UI.TreeAppName)
	}
	if m.state.UI.TreeAppNamespace == nil || *m.state.UI.TreeAppNamespace != childNSB {
		t.Fatalf("expected child namespace %q, got %v", childNSB, m.state.UI.TreeAppNamespace)
	}
	if len(m.state.SavedNavigation) == 0 || m.state.SavedNavigation[0].TreeAppNamespace == nil || *m.state.SavedNavigation[0].TreeAppNamespace != parentNS {
		t.Fatalf("expected saved parent namespace %q, got %#v", parentNS, m.state.SavedNavigation)
	}
}

// TestHandleNavigateToChildApp_ThreeLevels_SupportsDeepNesting verifies that navigating
// A→B→C and then pressing Escape twice returns through B then A (not directly to apps list).
func TestHandleNavigateToChildApp_ThreeLevels_SupportsDeepNesting(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	nsA := "ns-a"
	nsB := "ns-b"
	nsC := "ns-c"

	m.state.Apps = []model.App{
		{Name: "app-a", AppNamespace: &nsA},
		{Name: "app-b", AppNamespace: &nsB},
		{Name: "app-c", AppNamespace: &nsC},
	}

	// Start in app-a's tree
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeAppName = &m.state.Apps[0].Name
	m.state.UI.TreeAppNamespace = m.state.Apps[0].AppNamespace
	m.treeView = treeview.NewTreeView(0, 0)

	// Navigate from app-a to app-b
	newModel, _ := m.handleNavigateToChildApp("app-b", nsB)
	m = newModel.(*Model)

	// Navigate from app-b to app-c
	newModel, _ = m.handleNavigateToChildApp("app-c", nsC)
	m = newModel.(*Model)

	// First Escape: C → B
	newModel, _ = m.handleEscape()
	m = newModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("first escape: expected ViewTree, got %s", m.state.Navigation.View)
	}
	if m.state.UI.TreeAppName == nil || *m.state.UI.TreeAppName != "app-b" {
		t.Fatalf("first escape: expected app-b's tree, got %v", m.state.UI.TreeAppName)
	}
	if m.state.UI.TreeAppNamespace == nil || *m.state.UI.TreeAppNamespace != nsB {
		t.Fatalf("first escape: expected namespace %q, got %v", nsB, m.state.UI.TreeAppNamespace)
	}

	// Second Escape: B → A
	newModel, _ = m.handleEscape()
	m = newModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("second escape: expected ViewTree (back to app-a), got %s", m.state.Navigation.View)
	}
	if m.state.UI.TreeAppName == nil || *m.state.UI.TreeAppName != "app-a" {
		t.Fatalf("second escape: expected app-a's tree, got %v", m.state.UI.TreeAppName)
	}
	if m.state.UI.TreeAppNamespace == nil || *m.state.UI.TreeAppNamespace != nsA {
		t.Fatalf("second escape: expected namespace %q, got %v", nsA, m.state.UI.TreeAppNamespace)
	}

	// Third Escape: A → apps list
	newModel, _ = m.handleEscape()
	m = newModel.(*Model)

	if m.state.Navigation.View != model.ViewApps {
		t.Fatalf("third escape: expected ViewApps, got %s", m.state.Navigation.View)
	}
}

func TestHandleEscape_ReturnsToParentTreeUsingNamespace(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	parentNSA := "team-a"
	parentNSB := "team-b"
	childNS := "child-ns"

	parentName := "parent"
	childName := "child"

	m.state.Apps = []model.App{
		{Name: parentName, AppNamespace: &parentNSA},
		{Name: parentName, AppNamespace: &parentNSB},
		{Name: childName, AppNamespace: &childNS},
	}
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeAppName = &childName
	m.state.UI.TreeAppNamespace = &childNS
	m.state.SavedNavigation = []model.NavigationState{{
		View:             model.ViewTree,
		TreeAppName:      &parentName,
		TreeAppNamespace: &parentNSB,
	}}
	m.treeView = treeview.NewTreeView(0, 0)
	m.treeView.SetAppMeta(childName, "Healthy", "Synced")
	tree := api.ResourceTree{Nodes: []api.ResourceNode{{UID: "root", Kind: "Deployment", Name: "demo"}}}
	m.treeView.UpsertAppTree(childName, &tree)

	newModel, _ := m.handleEscape()
	m = newModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("expected to remain in tree view, got %s", m.state.Navigation.View)
	}
	if m.state.UI.TreeAppName == nil || *m.state.UI.TreeAppName != parentName {
		t.Fatalf("expected parent app name %q, got %v", parentName, m.state.UI.TreeAppName)
	}
	if m.state.UI.TreeAppNamespace == nil || *m.state.UI.TreeAppNamespace != parentNSB {
		t.Fatalf("expected parent namespace %q, got %v", parentNSB, m.state.UI.TreeAppNamespace)
	}
	if len(m.state.SavedNavigation) != 0 {
		t.Fatalf("expected saved navigation to be cleared after restore, got %#v", m.state.SavedNavigation)
	}
}
