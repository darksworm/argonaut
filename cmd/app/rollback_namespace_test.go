package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestRollbackBottomNavigation_EmptyRows_DoesNotSetNegativeIndex(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	m.state.Rollback = &model.RollbackState{
		Rows:        []model.RollbackRow{},
		SelectedIdx: 0,
	}

	newModel, _ := m.Update(model.RollbackNavigationMsg{Direction: "bottom"})
	m = newModel.(*Model)

	if m.state.Rollback.SelectedIdx < 0 {
		t.Fatalf("expected SelectedIdx >= 0 with empty Rows, got %d", m.state.Rollback.SelectedIdx)
	}
}

func TestHandleRollback_CapturesAppNamespaceFromCursor(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	ns := "team-b"
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &ns},
	}
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 0

	newModel, _ := m.handleRollback()
	m = newModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("expected RollbackState to be set")
	}
	if m.state.Rollback.AppNamespace == nil {
		t.Fatalf("expected AppNamespace %q, got nil", ns)
	}
	if *m.state.Rollback.AppNamespace != ns {
		t.Fatalf("expected AppNamespace %q, got %q", ns, *m.state.Rollback.AppNamespace)
	}
}
