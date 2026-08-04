package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// buildEventsPaneTestModel returns a model in tree view showing "test-app"
// (AppNamespace "test-namespace") with a single Pod row under the root.
func buildEventsPaneTestModel() *Model {
	m := buildDeleteTestModel(120, 40)
	m.state.Navigation.View = model.ViewTree
	ns := "test-namespace"
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "test-app", AppNamespace: &ns}
	m.treeView = treeview.NewTreeView(100, 30)
	m.treeView.SetAppMeta("test-app", "Healthy", "Synced")
	demoNs := "demo"
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "pod-uid", Version: "v1", Kind: "Pod", Name: "web-1", Namespace: &demoNs},
	}})
	return m
}

func TestTreeKeyE_OnSyntheticRoot_OpensApplicationEvents(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(0) // synthetic application root

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeEvents {
		t.Fatalf("expected ModeEvents, got %s", mm.state.Mode)
	}
	st := mm.state.Events
	if st == nil {
		t.Fatal("expected EventsState to be set")
	}
	want := model.EventsTarget{AppName: "test-app", AppNamespace: "test-namespace"}
	if st.Target != want {
		t.Errorf("expected app-level target %+v, got %+v", want, st.Target)
	}
	if cmd == nil {
		t.Error("expected a command to fetch events")
	}
}

func TestTreeKeyS_OpensSyncStatusFromAnyRow(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(1) // resource row: S still targets the app

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("S"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeSyncStatus {
		t.Fatalf("expected ModeSyncStatus, got %s", mm.state.Mode)
	}
	st := mm.state.SyncStatus
	if st == nil {
		t.Fatal("expected SyncStatusState to be set")
	}
	want := model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"}
	if st.Target != want {
		t.Errorf("expected target %+v, got %+v", want, st.Target)
	}
	if !st.Loading {
		t.Error("expected the pane to open in loading state")
	}
	if cmd == nil {
		t.Error("expected a command to fetch the sync status")
	}
}

func TestTreeKeyEnter_OnResourceRow_OpensResourceEvents(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(1) // the Pod row

	teaModel, _ := m.handleKeyMsg(testKeyMsg("enter"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeEvents {
		t.Fatalf("expected enter on a resource row to open events, got mode %s", mm.state.Mode)
	}
	if mm.state.Events == nil || mm.state.Events.Target.Resource.UID != "pod-uid" {
		t.Errorf("expected resource-scoped events target, got %+v", mm.state.Events)
	}
}

func TestTreeKeyEnter_OnSyntheticRoot_OpensApplicationEvents(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(0)

	teaModel, _ := m.handleKeyMsg(testKeyMsg("enter"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeEvents {
		t.Fatalf("expected enter on the root to open app events, got mode %s", mm.state.Mode)
	}
	if mm.state.Events == nil || mm.state.Events.Target.Resource != (model.EventsResource{}) {
		t.Errorf("expected app-level events target, got %+v", mm.state.Events)
	}
}

func TestTreeKeyEnter_OnChildApplicationRow_KeepsDrillIn(t *testing.T) {
	m := buildEventsPaneTestModel()
	childNs := "argocd"
	m.state.Apps = append(m.state.Apps, model.App{Name: "child-app", Sync: "Synced", Health: "Healthy", AppNamespace: &childNs})
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "child-app-uid", Group: "argoproj.io", Version: "v1alpha1", Kind: "Application", Name: "child-app", Namespace: &childNs},
	}})
	m.treeView.SetSelectedIndex(1) // the child Application row

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("enter"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeNormal {
		t.Fatalf("expected drill-in to keep normal mode, got %s", mm.state.Mode)
	}
	if mm.state.Events != nil {
		t.Error("expected no events pane on child Application drill-in")
	}
	if mm.state.UI.TreeApp == nil || mm.state.UI.TreeApp.Name != "child-app" {
		t.Errorf("expected drill-in to target child-app, got %+v", mm.state.UI.TreeApp)
	}
	if cmd == nil {
		t.Error("expected drill-in to start loading the child tree")
	}
}

func openEventsPane(t *testing.T, m *Model) *Model {
	t.Helper()
	m.treeView.SetSelectedIndex(1)
	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)
	if mm.state.Events == nil {
		t.Fatal("setup: events pane should be open")
	}
	return mm
}

func TestEventsLoadedMsg_FillsThePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	items := []model.ResourceEvent{{Reason: "BackOff", Message: "Back-off restarting failed container", Type: "Warning", Count: 412}}
	teaModel, _ := m.Update(model.EventsLoadedMsg{
		Target:      m.state.Events.Target,
		Items:       items,
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st.Loading {
		t.Error("expected loading to clear")
	}
	if len(st.Items) != 1 || st.Items[0].Reason != "BackOff" {
		t.Errorf("expected the loaded events in the pane, got %+v", st.Items)
	}
}

func TestEventsLoadedMsg_StaleEpoch_IsDropped(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.Update(model.EventsLoadedMsg{
		Target:      m.state.Events.Target,
		Items:       []model.ResourceEvent{{Reason: "BackOff"}},
		SwitchEpoch: m.switchEpoch - 1,
	})
	mm := teaModel.(*Model)

	if !mm.state.Events.Loading || len(mm.state.Events.Items) != 0 {
		t.Errorf("stale-epoch message must not touch the pane, got %+v", mm.state.Events)
	}
}

func TestEventsLoadedMsg_WrongTarget_IsDropped(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	other := m.state.Events.Target
	other.Resource.UID = "some-other-uid"
	teaModel, _ := m.Update(model.EventsLoadedMsg{
		Target:      other,
		Items:       []model.ResourceEvent{{Reason: "BackOff"}},
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	if !mm.state.Events.Loading || len(mm.state.Events.Items) != 0 {
		t.Errorf("wrong-target message must not touch the pane, got %+v", mm.state.Events)
	}
}

func TestEventsLoadedMsg_PaneAlreadyClosed_IsIgnored(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	target := m.state.Events.Target
	m.state.Events = nil
	m.state.Mode = model.ModeNormal

	teaModel, _ := m.Update(model.EventsLoadedMsg{Target: target, SwitchEpoch: m.switchEpoch})
	mm := teaModel.(*Model)

	if mm.state.Events != nil {
		t.Error("a late load must not reopen a closed pane")
	}
}

func TestEventsErrorMsg_ShowsInlineError(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.Update(model.EventsErrorMsg{
		Target:      m.state.Events.Target,
		Error:       "connection refused",
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st == nil || st.Error != "connection refused" || st.Loading {
		t.Errorf("expected a pane-scoped inline error, got %+v", st)
	}
	if mm.state.Mode != model.ModeEvents {
		t.Errorf("a failed fetch must not yank the user off the pane, got mode %s", mm.state.Mode)
	}
}

func TestSyncStatusLoadedMsg_FillsThePane(t *testing.T) {
	m := buildEventsPaneTestModel()
	teaModel, _ := m.handleKeyMsg(testKeyMsg("S"))
	m = teaModel.(*Model)

	details := &model.SyncStatusDetails{Phase: "Failed", Revision: "a1b2c3d"}
	teaModel, _ = m.Update(model.SyncStatusLoadedMsg{
		Target:      m.state.SyncStatus.Target,
		Details:     details,
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	st := mm.state.SyncStatus
	if st.Loading || st.Details == nil || st.Details.Phase != "Failed" {
		t.Errorf("expected loaded details in the pane, got %+v", st)
	}
}

func TestSyncStatusErrorMsg_StaleEpoch_IsDropped(t *testing.T) {
	m := buildEventsPaneTestModel()
	teaModel, _ := m.handleKeyMsg(testKeyMsg("S"))
	m = teaModel.(*Model)

	teaModel, _ = m.Update(model.SyncStatusErrorMsg{
		Target:      m.state.SyncStatus.Target,
		Error:       "boom",
		SwitchEpoch: m.switchEpoch - 1,
	})
	mm := teaModel.(*Model)

	if mm.state.SyncStatus.Error != "" || !mm.state.SyncStatus.Loading {
		t.Errorf("stale-epoch error must not touch the pane, got %+v", mm.state.SyncStatus)
	}
}

func TestEventsPane_EscClosesBackToTree(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	cursorBefore := m.treeView.SelectedIndex()

	teaModel, _ := m.handleKeyMsg(testKeyMsg("esc"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeNormal {
		t.Fatalf("expected ModeNormal after esc, got %s", mm.state.Mode)
	}
	if mm.state.Events != nil {
		t.Error("expected events state to be cleared on close")
	}
	if mm.treeView.SelectedIndex() != cursorBefore {
		t.Error("closing the pane must not move the tree cursor")
	}
}

func TestEventsPane_QCloses(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("q"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeNormal || mm.state.Events != nil {
		t.Errorf("expected q to close the pane, got mode %s events %+v", mm.state.Mode, mm.state.Events)
	}
}

func TestEventsPane_ActionKeysAreSwallowed(t *testing.T) {
	for _, key := range []string{"d", "s", "a", "ctrl+d", " ", "K"} {
		m := openEventsPane(t, buildEventsPaneTestModel())

		teaModel, cmd := m.handleKeyMsg(testKeyMsg(key))
		mm := teaModel.(*Model)

		if mm.state.Mode != model.ModeEvents || mm.state.Events == nil {
			t.Errorf("key %q must be swallowed by the pane, got mode %s", key, mm.state.Mode)
		}
		if cmd != nil {
			t.Errorf("key %q must not trigger a command from the pane", key)
		}
	}
}

func TestEventsPane_SSwitchesToSyncStatus(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("S"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeSyncStatus {
		t.Fatalf("expected S to switch to the sync-status pane, got %s", mm.state.Mode)
	}
	if mm.state.Events != nil {
		t.Error("expected events state to be cleared on switch")
	}
	if mm.state.SyncStatus == nil || cmd == nil {
		t.Error("expected sync-status state and a fetch command")
	}
}

func TestSyncStatusPane_ESwitchesToEvents(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(1)
	teaModel, _ := m.handleKeyMsg(testKeyMsg("S"))
	m = teaModel.(*Model)

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeEvents {
		t.Fatalf("expected e to switch to the events pane, got %s", mm.state.Mode)
	}
	if mm.state.SyncStatus != nil {
		t.Error("expected sync-status state to be cleared on switch")
	}
	if mm.state.Events == nil || cmd == nil {
		t.Error("expected events state and a fetch command")
	}
}

func TestEventsPane_JKScrollTheViewport(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("j"))
	mm := teaModel.(*Model)
	if mm.state.Events.Offset != 1 {
		t.Fatalf("expected j to scroll down to offset 1, got %d", mm.state.Events.Offset)
	}

	teaModel, _ = mm.handleKeyMsg(testKeyMsg("k"))
	mm = teaModel.(*Model)
	if mm.state.Events.Offset != 0 {
		t.Fatalf("expected k to scroll back to offset 0, got %d", mm.state.Events.Offset)
	}

	teaModel, _ = mm.handleKeyMsg(testKeyMsg("G"))
	mm = teaModel.(*Model)
	if mm.state.Events.Offset == 0 {
		t.Error("expected G to jump towards the bottom (clamped at render time)")
	}
}

func TestEventsPane_ColonOpensCommandAndEscReturnsToPane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg(":"))
	mm := teaModel.(*Model)
	if mm.state.Mode != model.ModeCommand {
		t.Fatalf("expected ':' to enter command mode, got %s", mm.state.Mode)
	}
	if mm.state.Events == nil {
		t.Fatal("expected the pane to stay alive behind the command bar")
	}

	teaModel, _ = mm.handleKeyMsg(testKeyMsg("esc"))
	mm = teaModel.(*Model)
	if mm.state.Mode != model.ModeEvents {
		t.Fatalf("expected esc to return to the events pane, got %s", mm.state.Mode)
	}
	if mm.state.Events == nil {
		t.Error("expected the pane state to survive the command-mode round trip")
	}
}

func runCommand(t *testing.T, m *Model, command string) *Model {
	t.Helper()
	m.state.Mode = model.ModeCommand
	m.inputComponents.SetCommandValue(command)
	m.state.UI.Command = command
	teaModel, _ := m.handleEnhancedCommandModeKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	return teaModel.(*Model)
}

func TestEventsCommand_InTreeView_OpensEventsPane(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(1)

	m = runCommand(t, m, "events")

	if m.state.Mode != model.ModeEvents {
		t.Fatalf("expected :events to open the events pane, got mode %s", m.state.Mode)
	}
	if m.state.Events == nil || m.state.Events.Target.Resource.UID != "pod-uid" {
		t.Errorf("expected events for the selected row, got %+v", m.state.Events)
	}
}

func TestSyncStatusCommand_InTreeView_OpensSyncStatusPane(t *testing.T) {
	m := buildEventsPaneTestModel()

	m = runCommand(t, m, "syncstatus")

	if m.state.Mode != model.ModeSyncStatus {
		t.Fatalf("expected :syncstatus to open the sync-status pane, got mode %s", m.state.Mode)
	}
	if m.state.SyncStatus == nil || m.state.SyncStatus.Target.AppName != "test-app" {
		t.Errorf("expected sync status for the tree app, got %+v", m.state.SyncStatus)
	}
}

func TestViewJumpingCommand_ClosesTheOpenPane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	// ':' from the pane, then a command that navigates elsewhere
	teaModel, _ := m.handleKeyMsg(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "cluster")

	if m.state.Events != nil {
		t.Error("expected a view-jumping command to close the events pane")
	}
	if m.state.Mode == model.ModeEvents {
		t.Errorf("expected to leave the pane mode, got %s", m.state.Mode)
	}
}

func TestAppsBatchUpdate_RefreshesTreeSyncSummaryLive(t *testing.T) {
	m := buildEventsPaneTestModel()
	updated := m.state.Apps[0] // test-app
	updated.SyncOp = &model.SyncOpSummary{
		Phase:      "Failed",
		StartedAt:  time.Now().Add(-2 * time.Minute),
		FinishedAt: time.Now().Add(-2 * time.Minute),
		Revision:   "a1b2c3d",
	}

	teaModel, _ := m.Update(model.AppsBatchUpdateMsg{
		Updates:     []model.AppUpdatedMsg{{App: updated}},
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	if !strings.Contains(mm.treeView.Render(), "last sync:") {
		t.Error("expected the tree to show the sync summary line after a live app update")
	}
}

func TestResourceTreeLoaded_SetsSyncSummaryFromAppList(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Apps[0].SyncOp = &model.SyncOpSummary{
		Phase:      "Succeeded",
		FinishedAt: time.Now().Add(-time.Hour),
	}
	treeJSON, err := json.Marshal(api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	teaModel, _ := m.Update(model.ResourceTreeLoadedMsg{
		AppName:     "test-app",
		Health:      "Healthy",
		Sync:        "Synced",
		TreeJSON:    treeJSON,
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	if !strings.Contains(mm.treeView.Render(), "last sync:") {
		t.Error("expected the freshly loaded tree to carry the app's sync summary line")
	}
}

// Two apps share a name in different ArgoCD namespaces; the pane target must
// use the tree-scoped app's namespace, not the first name-match in the list.
func TestPaneTargets_DisambiguateAppByNamespace(t *testing.T) {
	m := buildEventsPaneTestModel()
	nsArgocd := "argocd"
	nsTeamA := "team-a"
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd}, // wrong app first in the slice
		{Name: "my-app", AppNamespace: &nsTeamA},
	}
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "my-app", AppNamespace: &nsTeamA}
	m.treeView = treeview.NewTreeView(100, 30)
	m.treeView.SetAppMeta("my-app", "Healthy", "Synced")
	m.treeView.UpsertAppTree("my-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web"},
	}})

	teaModel, _ := m.handleKeyMsg(testKeyMsg("S"))
	mm := teaModel.(*Model)
	if got := mm.state.SyncStatus.Target.AppNamespace; got != nsTeamA {
		t.Errorf("sync-status target should use the tree app's namespace %q, got %q", nsTeamA, got)
	}

	mm.closePanes()
	teaModel, _ = mm.handleKeyMsg(testKeyMsg("e"))
	mm = teaModel.(*Model)
	if got := mm.state.Events.Target.AppNamespace; got != nsTeamA {
		t.Errorf("events target should use the tree app's namespace %q, got %q", nsTeamA, got)
	}
}

func TestLoadEvents_ProducesGatedMessageWithData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications/test-app/events" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Write([]byte(`{"items":[{"reason": "BackOff", "type": "Warning", "count": 3, "lastTimestamp": "2026-08-04T12:00:00Z"}]}`))
	}))
	defer server.Close()

	m := buildEventsPaneTestModel()
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	target := model.EventsTarget{AppName: "test-app", AppNamespace: "test-namespace"}

	msg := m.loadEvents(target)()

	loaded, ok := msg.(model.EventsLoadedMsg)
	if !ok {
		t.Fatalf("expected EventsLoadedMsg, got %T: %+v", msg, msg)
	}
	if loaded.Target != target || loaded.SwitchEpoch != m.switchEpoch {
		t.Errorf("expected gating fields carried through, got %+v", loaded)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].Reason != "BackOff" {
		t.Errorf("expected the fetched events, got %+v", loaded.Items)
	}
}

func TestLoadSyncStatus_FetchesFullOperationState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications/test-app" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("appNamespace"); got != "test-namespace" {
			t.Errorf("expected appNamespace=test-namespace, got %q", got)
		}
		w.Write([]byte(`{"metadata": {"name": "test-app"}, "status": {"operationState": {
			"phase": "Failed",
			"message": "one or more objects failed to apply",
			"syncResult": {"revision": "a1b2c3d", "resources": [
				{"kind": "Deployment", "namespace": "demo", "name": "web", "status": "SyncFailed", "message": "invalid"}
			]}
		}}}`))
	}))
	defer server.Close()

	m := buildEventsPaneTestModel()
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	target := model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"}

	msg := m.loadSyncStatus(target)()

	loaded, ok := msg.(model.SyncStatusLoadedMsg)
	if !ok {
		t.Fatalf("expected SyncStatusLoadedMsg, got %T: %+v", msg, msg)
	}
	if loaded.Details == nil || loaded.Details.Phase != "Failed" || len(loaded.Details.Resources) != 1 {
		t.Errorf("expected full operation state with resource results, got %+v", loaded.Details)
	}
}

func TestTreeKeyE_OnResourceRow_OpensResourceEvents(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(1) // row 0 is the synthetic root, row 1 the Pod

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeEvents {
		t.Fatalf("expected ModeEvents, got %s", mm.state.Mode)
	}
	st := mm.state.Events
	if st == nil {
		t.Fatal("expected EventsState to be set")
	}
	want := model.EventsTarget{
		AppName:      "test-app",
		AppNamespace: "test-namespace",
		Resource:     model.EventsResource{Kind: "Pod", Namespace: "demo", Name: "web-1", UID: "pod-uid"},
	}
	if st.Target != want {
		t.Errorf("expected target %+v, got %+v", want, st.Target)
	}
	if !st.Loading {
		t.Error("expected the pane to open in loading state")
	}
	if cmd == nil {
		t.Error("expected a command to fetch events")
	}
}
