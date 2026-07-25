package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
)

// Two apps sharing a name in different ArgoCD namespaces (ADR-0004): a watch
// delete for one must not remove the other.
func TestBatchAppDelete_SameName_RemovesOnlyMatchingNamespace(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	nsArgocd := "argocd"
	nsTeamA := "team-a"
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd},
		{Name: "my-app", AppNamespace: &nsTeamA},
	}
	m.state.Index = model.BuildAppIndex(m.state.Apps)

	msg := model.AppsBatchUpdateMsg{
		Operations: []model.AppBatchOperation{{
			Type:   model.AppBatchOperationDelete,
			Delete: &model.AppDeletedMsg{AppName: "my-app", AppNamespace: &nsTeamA},
		}},
	}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if len(m.state.Apps) != 1 {
		t.Fatalf("expected 1 app left, got %d", len(m.state.Apps))
	}
	if got := *m.state.Apps[0].AppNamespace; got != nsArgocd {
		t.Errorf("wrong app deleted: survivor is in %q, want %q", got, nsArgocd)
	}
}

// A watch update for one of two same-named apps must patch that app's row,
// not the first name-match.
func TestBatchAppUpdate_SameName_UpdatesOnlyMatchingNamespace(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	nsArgocd := "argocd"
	nsTeamA := "team-a"
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd, Health: "Healthy"},
		{Name: "my-app", AppNamespace: &nsTeamA, Health: "Healthy"},
	}
	m.state.Index = model.BuildAppIndex(m.state.Apps)

	msg := model.AppsBatchUpdateMsg{
		Operations: []model.AppBatchOperation{{
			Type:   model.AppBatchOperationUpdate,
			Update: &model.AppUpdatedMsg{App: model.App{Name: "my-app", AppNamespace: &nsTeamA, Health: "Degraded"}},
		}},
	}
	newModel, _ := m.Update(msg)
	m = newModel.(*Model)

	if len(m.state.Apps) != 2 {
		t.Fatalf("expected 2 apps, got %d", len(m.state.Apps))
	}
	if m.state.Apps[0].Health != "Healthy" {
		t.Errorf("argocd app was patched but the event targeted team-a")
	}
	if m.state.Apps[1].Health != "Degraded" {
		t.Errorf("team-a app not patched: health %q, want Degraded", m.state.Apps[1].Health)
	}
}

// The DELETED watch event carries the CR namespace; classification must not
// drop it.
func TestClassifyWatchEvent_Delete_CarriesAppNamespace(t *testing.T) {
	ns := "team-a"
	ev := services.ArgoApiEvent{Type: "app-deleted", AppName: "my-app", AppNamespace: &ns}
	result := classifyWatchEvent(ev, 0)

	if result.delete == nil {
		t.Fatal("expected delete to be set for app-deleted")
	}
	if result.delete.AppNamespace == nil || *result.delete.AppNamespace != ns {
		t.Errorf("delete lost the app namespace: %v", result.delete.AppNamespace)
	}
}
