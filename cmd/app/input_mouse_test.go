package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

func TestHandleMouseWheelListViewScrollsSelection(t *testing.T) {
	m := NewModel()
	m.state.Navigation.View = model.ViewApps
	m.state.Apps = []model.App{
		{Name: "app-1", Sync: "Synced", Health: "Healthy"},
		{Name: "app-2", Sync: "Synced", Health: "Healthy"},
		{Name: "app-3", Sync: "Synced", Health: "Healthy"},
		{Name: "app-4", Sync: "Synced", Health: "Healthy"},
		{Name: "app-5", Sync: "Synced", Health: "Healthy"},
	}

	initialIdx := m.state.Navigation.SelectedIdx
	if initialIdx != 0 {
		t.Fatalf("expected initial selection 0, got %d", initialIdx)
	}

	_, _ = m.handleMouseWheelMsg(tea.MouseWheelMsg{Button: tea.MouseWheelDown})

	expected := min(mouseScrollAmount, len(m.state.Apps)-1)
	if m.state.Navigation.SelectedIdx != expected {
		t.Fatalf("expected selection %d after scroll, got %d", expected, m.state.Navigation.SelectedIdx)
	}
}

func TestHandleMouseWheelListViewBlockedByOverlay(t *testing.T) {
	m := NewModel()
	m.state.Navigation.View = model.ViewApps
	m.state.Apps = []model.App{{Name: "app-1", Sync: "Synced", Health: "Healthy"}, {Name: "app-2", Sync: "Synced", Health: "Healthy"}}
	m.state.Mode = model.ModeConfirmSync

	_, _ = m.handleMouseWheelMsg(tea.MouseWheelMsg{Button: tea.MouseWheelDown})

	if m.state.Navigation.SelectedIdx != 0 {
		t.Fatalf("expected selection to remain 0 when overlay active, got %d", m.state.Navigation.SelectedIdx)
	}
}

func TestHandleMouseWheelThemeOverlay(t *testing.T) {
	m := NewModel()
	m.state.Mode = model.ModeTheme
	defer func() {
		applyTheme(theme.Default())
		m.applyThemeToModel()
	}()
	m.ensureThemeOptionsLoaded()

	if len(m.themeOptions) == 0 {
		t.Fatal("expected theme options to be loaded")
	}

	_, _ = m.handleMouseWheelMsg(tea.MouseWheelMsg{Button: tea.MouseWheelDown})

	if m.state.UI.ThemeSelectedIndex <= 0 {
		t.Fatalf("expected theme selection to advance, got %d", m.state.UI.ThemeSelectedIndex)
	}
}

func TestHandleMouseWheelDiffViewNoop(t *testing.T) {
	m := NewModel()
	m.state.Mode = model.ModeDiff
	m.state.Diff = &model.DiffState{
		Title:  "Example",
		Offset: 5,
	}

	_, _ = m.handleMouseWheelMsg(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.state.Diff.Offset != 5 {
		t.Fatalf("expected diff offset to remain unchanged, got %d", m.state.Diff.Offset)
	}

	_, _ = m.handleMouseWheelMsg(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.state.Diff.Offset != 5 {
		t.Fatalf("expected diff offset to remain unchanged after scroll up, got %d", m.state.Diff.Offset)
	}
}

func TestHandleMouseWheelTreeView(t *testing.T) {
	m := NewModel()
	m.state.Navigation.View = model.ViewTree
	m.treeView = treeview.NewTreeView(0, 0)
	m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)

	ns := "default"
	tree := &api.ResourceTree{
		Nodes: []api.ResourceNode{
			{
				UID:         "root",
				Kind:        "Application",
				Name:        "app",
				Group:       "argoproj.io",
				Status:      "Synced",
				Version:     "v1",
				ResourceRef: api.ResourceRef{UID: "root", Kind: "Application", Name: "app", Group: "argoproj.io", Version: "v1"},
			},
			{
				UID:         "deploy",
				Kind:        "Deployment",
				Name:        "app",
				Group:       "apps",
				Status:      "Synced",
				Version:     "v1",
				ResourceRef: api.ResourceRef{UID: "deploy", Kind: "Deployment", Name: "app", Group: "apps", Version: "v1"},
				ParentRefs:  []api.ResourceRef{{UID: "root", Kind: "Application", Name: "app", Group: "argoproj.io", Version: "v1"}},
			},
			{
				UID:         "rs",
				Kind:        "ReplicaSet",
				Name:        "app",
				Group:       "apps",
				Status:      "Synced",
				Version:     "v1",
				ResourceRef: api.ResourceRef{UID: "rs", Kind: "ReplicaSet", Name: "app", Group: "apps", Version: "v1"},
				ParentRefs:  []api.ResourceRef{{UID: "deploy", Kind: "Deployment", Name: "app", Group: "apps", Version: "v1"}},
			},
			{
				UID:         "pod",
				Kind:        "Pod",
				Name:        "app-123",
				Group:       "",
				Status:      "Running",
				Version:     "v1",
				Namespace:   &ns,
				ResourceRef: api.ResourceRef{UID: "pod", Kind: "Pod", Name: "app-123", Version: "v1", Namespace: &ns},
				ParentRefs:  []api.ResourceRef{{UID: "rs", Kind: "ReplicaSet", Name: "app", Group: "apps", Version: "v1"}},
			},
		},
	}

	m.treeView.UpsertAppTree("app", tree)

	_, _ = m.handleMouseWheelMsg(tea.MouseWheelMsg{Button: tea.MouseWheelDown})

	if m.treeView.SelectedIndex() == 0 {
		t.Fatal("expected tree selection to advance after scrolling")
	}
}
