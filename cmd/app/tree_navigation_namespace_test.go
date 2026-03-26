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
	if m.state.SavedNavigation == nil || m.state.SavedNavigation.TreeAppNamespace == nil || *m.state.SavedNavigation.TreeAppNamespace != parentNS {
		t.Fatalf("expected saved parent namespace %q, got %#v", parentNS, m.state.SavedNavigation)
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
	m.state.SavedNavigation = &model.NavigationState{
		View:             model.ViewTree,
		TreeAppName:      &parentName,
		TreeAppNamespace: &parentNSB,
	}
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
	if m.state.SavedNavigation != nil {
		t.Fatalf("expected saved navigation to be cleared after restore, got %#v", m.state.SavedNavigation)
	}
}
