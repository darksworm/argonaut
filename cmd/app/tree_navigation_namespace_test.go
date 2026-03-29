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
	m.setTreeApp(m.state.Apps[0])
	m.treeView = treeview.NewTreeView(0, 0)

	_, _ = m.handleNavigateToChildApp("child", childNSB)

	if m.state.UI.TreeApp == nil || m.state.UI.TreeApp.Name != "child" {
		t.Fatalf("expected child app to be opened, got %v", m.state.UI.TreeApp)
	}
	if m.state.UI.TreeApp.AppNamespace == nil || *m.state.UI.TreeApp.AppNamespace != childNSB {
		t.Fatalf("expected child namespace %q, got %v", childNSB, m.state.UI.TreeApp.AppNamespace)
	}
	if len(m.state.SavedNavigation) == 0 || m.state.SavedNavigation[0].TreeApp == nil || m.state.SavedNavigation[0].TreeApp.AppNamespace == nil || *m.state.SavedNavigation[0].TreeApp.AppNamespace != parentNS {
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
	m.setTreeApp(m.state.Apps[0])
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
	if m.state.UI.TreeApp == nil || m.state.UI.TreeApp.Name != "app-b" {
		t.Fatalf("first escape: expected app-b's tree, got %v", m.state.UI.TreeApp)
	}
	if m.state.UI.TreeApp.AppNamespace == nil || *m.state.UI.TreeApp.AppNamespace != nsB {
		t.Fatalf("first escape: expected namespace %q, got %v", nsB, m.state.UI.TreeApp.AppNamespace)
	}

	// Second Escape: B → A
	newModel, _ = m.handleEscape()
	m = newModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("second escape: expected ViewTree (back to app-a), got %s", m.state.Navigation.View)
	}
	if m.state.UI.TreeApp == nil || m.state.UI.TreeApp.Name != "app-a" {
		t.Fatalf("second escape: expected app-a's tree, got %v", m.state.UI.TreeApp)
	}
	if m.state.UI.TreeApp.AppNamespace == nil || *m.state.UI.TreeApp.AppNamespace != nsA {
		t.Fatalf("second escape: expected namespace %q, got %v", nsA, m.state.UI.TreeApp.AppNamespace)
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
	m.setTreeApp(model.App{Name: childName, AppNamespace: &childNS})
	m.state.SavedNavigation = []model.NavigationState{{
		View:    model.ViewTree,
		TreeApp: &model.TreeAppInfo{Name: parentName, AppNamespace: &parentNSB},
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
	if m.state.UI.TreeApp == nil || m.state.UI.TreeApp.Name != parentName {
		t.Fatalf("expected parent app name %q, got %v", parentName, m.state.UI.TreeApp)
	}
	if m.state.UI.TreeApp.AppNamespace == nil || *m.state.UI.TreeApp.AppNamespace != parentNSB {
		t.Fatalf("expected parent namespace %q, got %v", parentNSB, m.state.UI.TreeApp.AppNamespace)
	}
	if len(m.state.SavedNavigation) != 0 {
		t.Fatalf("expected saved navigation to be cleared after restore, got %#v", m.state.SavedNavigation)
	}
}
