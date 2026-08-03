package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

func TestHandleResourceSync_ChildApplicationTargetsChildApp(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	parentNamespace := "argocd"
	otherChildNamespace := "team-a"
	childNamespace := "team-b"
	m.state.Apps = []model.App{
		{Name: "parent", AppNamespace: &parentNamespace},
		{Name: "child", AppNamespace: &otherChildNamespace},
		{Name: "child", AppNamespace: &childNamespace},
	}
	m.state.Navigation.View = model.ViewTree
	m.setTreeApp(m.state.Apps[0])
	m.treeView = treeview.NewTreeView(0, 0)
	m.treeView.SetAppMeta("parent", "Healthy", "OutOfSync")
	m.treeView.UpsertAppTree("parent", &api.ResourceTree{Nodes: []api.ResourceNode{
		{
			UID:       "child-application",
			Group:     "argoproj.io",
			Version:   "v1alpha1",
			Kind:      "Application",
			Namespace: &childNamespace,
			Name:      "child",
			Status:    "OutOfSync",
		},
	}})
	m.treeView.SetSelectedIndex(1)

	updated, cmd := m.handleKeyMsg(testKeyMsg("s"))
	m = updated.(*Model)

	if cmd != nil {
		t.Fatal("opening the sync confirmation should not execute a command")
	}
	if m.state.Mode != model.ModeConfirmSync {
		t.Fatalf("expected application sync confirmation, got mode %s", m.state.Mode)
	}
	if m.state.Modals.ConfirmTarget == nil || *m.state.Modals.ConfirmTarget != "child" {
		t.Fatalf("expected child sync target, got %v", m.state.Modals.ConfirmTarget)
	}
	if m.state.Modals.ConfirmTargetNamespace == nil || *m.state.Modals.ConfirmTargetNamespace != childNamespace {
		t.Fatalf("expected child app namespace %q, got %v", childNamespace, m.state.Modals.ConfirmTargetNamespace)
	}
	if m.state.Modals.ResourceSyncAppName != nil || len(m.state.Modals.ResourceSyncTargets) != 0 {
		t.Fatalf("expected no parent resource sync target, got app=%v resources=%v", m.state.Modals.ResourceSyncAppName, m.state.Modals.ResourceSyncTargets)
	}
}

func TestHandleResourceSync_SyntheticRootTargetsParentApp(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	parentNamespace := "argocd"
	m.state.Apps = []model.App{{Name: "parent", AppNamespace: &parentNamespace}}
	m.state.Navigation.View = model.ViewTree
	m.setTreeApp(m.state.Apps[0])
	m.treeView = treeview.NewTreeView(0, 0)
	m.treeView.SetAppMeta("parent", "Healthy", "OutOfSync")
	m.treeView.UpsertAppTree("parent", &api.ResourceTree{})
	m.treeView.SetSelectedIndex(0)

	updated, cmd := m.handleKeyMsg(testKeyMsg("s"))
	m = updated.(*Model)

	if cmd != nil {
		t.Fatal("opening the sync confirmation should not execute a command")
	}
	if m.state.Mode != model.ModeConfirmSync {
		t.Fatalf("expected application sync confirmation, got mode %s", m.state.Mode)
	}
	if m.state.Modals.ConfirmTarget == nil || *m.state.Modals.ConfirmTarget != "parent" {
		t.Fatalf("expected parent sync target, got %v", m.state.Modals.ConfirmTarget)
	}
	if m.state.Modals.ConfirmTargetNamespace == nil || *m.state.Modals.ConfirmTargetNamespace != parentNamespace {
		t.Fatalf("expected parent app namespace %q, got %v", parentNamespace, m.state.Modals.ConfirmTargetNamespace)
	}
}
