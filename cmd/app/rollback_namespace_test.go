package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

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
