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
	m.config.Events.RefreshInterval = "0" // no auto-refresh ticks in unit tests
	m.state.Navigation.View = model.ViewTree
	ns := "test-namespace"
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "test-app", AppNamespace: &ns}
	m.treeView = treeview.NewTreeView(100, 30)
	m.treeView.SetAppMeta("test-app", "Healthy", "Synced")
	demoNs := "demo"
	degraded := "Degraded"
	healthMsg := "Back-off restarting container"
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "pod-uid", Version: "v1", Kind: "Pod", Name: "web-1", Namespace: &demoNs,
			Health: &api.ResourceHealth{Status: &degraded, Message: &healthMsg}},
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

// An application row is one lens: the pane loads the sync details for the
// status block on top of the events.
func TestEventsPane_OnApplicationRow_AlsoLoadsSyncDetails(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(0) // synthetic root

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st == nil || !st.Loading || !st.DetailsLoading {
		t.Fatalf("expected both the events and the sync details to load, got %+v", st)
	}
	if cmd == nil {
		t.Fatal("expected fetch commands")
	}
	// The fixture has no server: both producers report errors, one per fetch
	msgs := collectMsgs(t, cmd)
	var gotEvents, gotDetails bool
	for _, msg := range msgs {
		switch msg.(type) {
		case model.EventsErrorMsg:
			gotEvents = true
		case model.SyncStatusErrorMsg:
			gotDetails = true
		}
	}
	if !gotEvents || !gotDetails {
		t.Errorf("expected an events fetch and a details fetch, got %#v", msgs)
	}
}

// Resource rows fetch the app details too — their own row of the last sync
// RESULT belongs in the status block — and snapshot the tree's health/sync.
func TestEventsPane_OnResourceRow_SnapshotsStatusAndLoadsDetails(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	st := m.state.Events
	if !st.DetailsLoading {
		t.Error("expected the app details to load for the resource's last-sync result")
	}
	if st.ResourceStatus == nil || st.ResourceStatus.Health != "Degraded" {
		t.Errorf("expected the resource status snapshot from the tree, got %+v", st.ResourceStatus)
	}
	if st.ResourceStatus.HealthMessage != "Back-off restarting container" {
		t.Errorf("expected the health message in the snapshot, got %q", st.ResourceStatus.HealthMessage)
	}
}

// Moving between rows of the same app must not refetch the app details —
// they are identical for every row.
func TestEventsPane_RetargetWithinApp_KeepsLoadedDetails(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	details := &model.SyncStatusDetails{Phase: "Succeeded"}
	teaModel, _ := m.Update(model.SyncStatusLoadedMsg{
		Target:      model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"},
		Details:     details,
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	m = teaModel.(*Model)

	teaModel, _ = m.handleKeyMsg(testKeyMsg("k")) // onto the root: same app
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st.DetailsLoading || st.Details != details {
		t.Errorf("expected the loaded details carried over within the app, got loading=%v", st.DetailsLoading)
	}
}

// S lost its binding when the sync-status pane merged into the events lens.
func TestTreeKeyS_DoesNothing(t *testing.T) {
	m := buildEventsPaneTestModel()

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("S"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeNormal || mm.state.Events != nil || cmd != nil {
		t.Errorf("expected S to do nothing in tree view, got mode %s", mm.state.Mode)
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

// A Missing resource was never created in the cluster, so it cannot have
// events — the pane opens as a lens with an inline notice, no fetch.
func TestTreeKeyE_OnMissingResource_OpensWithNotice(t *testing.T) {
	m := buildEventsPaneTestModel()
	ns := "argonaut-demo"
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout", Name: "bluegreen-demo", Namespace: &ns},
	}})
	m.treeView.SetSelectedIndex(1)

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)

	if mm.state.Mode != model.ModeEvents || mm.state.Events == nil {
		t.Fatalf("expected the pane to open as a lens, got mode %s", mm.state.Mode)
	}
	st := mm.state.Events
	if st.Notice == "" || st.Loading || st.Error != "" {
		t.Errorf("expected an inline notice and no events fetch, got %+v", st)
	}
	// The app details still load: a Missing resource often has a RESULT row
	// explaining why (failed create, prune skipped)
	if !st.DetailsLoading || cmd == nil {
		t.Error("expected the details fetch to still run for a Missing resource")
	}
}

// collectMsgs executes a command, flattening tea.Batch into its messages.
func collectMsgs(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, collectMsgs(t, c)...)
		}
		return out
	}
	return []tea.Msg{msg}
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
		LoadSeq:     m.state.Events.LoadSeq,
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

// Close + reopen the same target: the first request's late completion (e.g.
// a timeout error) carries the same epoch and target but must not paint
// over the fresh load.
func TestEventsErrorMsg_FromSupersededLoad_IsDropped(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	staleTarget := m.state.Events.Target
	staleSeq := m.state.Events.LoadSeq

	teaModel, _ := m.handleKeyMsg(testKeyMsg("esc"))
	m = openEventsPane(t, teaModel.(*Model))

	teaModel, _ = m.Update(model.EventsErrorMsg{
		Target:      staleTarget,
		Error:       "context deadline exceeded",
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     staleSeq,
	})
	mm := teaModel.(*Model)

	if mm.state.Events.Error != "" {
		t.Errorf("a superseded load's error must be dropped, got %q", mm.state.Events.Error)
	}
	if !mm.state.Events.Loading {
		t.Error("the fresh load must still be pending")
	}
}

func TestSyncStatusLoadedMsg_FromSupersededLoad_IsDropped(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(0)
	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	m = teaModel.(*Model)
	staleSeq := m.state.Events.LoadSeq

	teaModel, _ = m.handleKeyMsg(testKeyMsg("esc"))
	teaModel, _ = teaModel.(*Model).handleKeyMsg(testKeyMsg("e")) // reopen, same target
	m = teaModel.(*Model)

	teaModel, _ = m.Update(model.SyncStatusLoadedMsg{
		Target:      model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"},
		Details:     &model.SyncStatusDetails{Phase: "Failed"},
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     staleSeq,
	})
	mm := teaModel.(*Model)

	if mm.state.Events.Details != nil {
		t.Error("a superseded load's details must be dropped")
	}
}

func TestEventsErrorMsg_ShowsInlineError(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.Update(model.EventsErrorMsg{
		Target:      m.state.Events.Target,
		Error:       "connection refused",
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
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

func TestSyncStatusLoadedMsg_FillsThePaneDetails(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(0)
	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	m = teaModel.(*Model)

	details := &model.SyncStatusDetails{Phase: "Failed", Revision: "a1b2c3d"}
	teaModel, _ = m.Update(model.SyncStatusLoadedMsg{
		Target:      model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"},
		Details:     details,
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st.DetailsLoading || st.Details == nil || st.Details.Phase != "Failed" {
		t.Errorf("expected the loaded details in the pane, got %+v", st)
	}
}

func TestSyncStatusErrorMsg_StaleEpoch_IsDropped(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.treeView.SetSelectedIndex(0)
	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	m = teaModel.(*Model)

	teaModel, _ = m.Update(model.SyncStatusErrorMsg{
		Target:      model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"},
		Error:       "boom",
		SwitchEpoch: m.switchEpoch - 1,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	mm := teaModel.(*Model)

	if mm.state.Events.DetailsError != "" || !mm.state.Events.DetailsLoading {
		t.Errorf("stale-epoch error must not touch the pane, got %+v", mm.state.Events)
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

// The pane is a lens, not a modal: action hotkeys close it and act on the
// selected tree row as if the pane were not there.
func TestEventsPane_ActionKeysCloseThePaneAndAct(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("d")) // diff the selected resource
	mm := teaModel.(*Model)

	if mm.state.Events != nil {
		t.Error("expected the pane to close when an action key fires")
	}
	if mm.state.Diff == nil || !mm.state.Diff.Loading || cmd == nil {
		t.Errorf("expected d to start the resource diff, got %+v", mm.state.Diff)
	}
}

func TestEventsPane_CtrlDOpensDeleteConfirmation(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("ctrl+d"))
	mm := teaModel.(*Model)

	if mm.state.Events != nil {
		t.Error("expected the pane to close when an action key fires")
	}
	if mm.state.Mode != model.ModeConfirmResourceDelete {
		t.Fatalf("expected the delete confirmation, got mode %s", mm.state.Mode)
	}
}

func TestEventsPane_JKNavigateTheTreeAndRetargetThePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel()) // opened on the Pod row (index 1)
	if m.state.Events.Target.Resource.UID != "pod-uid" {
		t.Fatalf("setup: expected the pod target, got %+v", m.state.Events.Target)
	}

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("k"))
	mm := teaModel.(*Model)

	if mm.treeView.SelectedIndex() != 0 {
		t.Fatalf("expected k to move the tree cursor to the root, got %d", mm.treeView.SelectedIndex())
	}
	if got := mm.state.Events.Target.Resource; got != (model.EventsResource{}) {
		t.Errorf("expected the pane to retarget to app-level events, got %+v", got)
	}
	if !mm.state.Events.Loading {
		t.Error("expected the retargeted pane to be loading")
	}
	if cmd == nil {
		t.Error("expected a debounced fetch to be scheduled")
	}

	teaModel, _ = mm.handleKeyMsg(testKeyMsg("j"))
	mm = teaModel.(*Model)
	if mm.state.Events.Target.Resource.UID != "pod-uid" {
		t.Errorf("expected j to retarget back to the pod, got %+v", mm.state.Events.Target)
	}
}

func TestEventsPane_ShiftedKeysScrollTheViewport(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	for _, tc := range []struct {
		key    tea.KeyMsg
		want   int
		label  string
	}{
		{testKeyMsg("J"), 1, "J scrolls down"},
		{testKeyMsg("K"), 0, "K scrolls up"},
		{tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}, 1, "shift+down scrolls down"},
		{tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}, 0, "shift+up scrolls up"},
	} {
		teaModel, _ := m.handleKeyMsg(tc.key)
		m = teaModel.(*Model)
		if m.state.Events.Offset != tc.want {
			t.Fatalf("%s: expected offset %d, got %d", tc.label, tc.want, m.state.Events.Offset)
		}
	}
}

func TestEventsPane_PaneFetchDue_DispatchesOnlyForCurrentSeq(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	teaModel, _ := m.handleKeyMsg(testKeyMsg("k")) // retarget to the root
	m = teaModel.(*Model)
	seq := m.state.Events.LoadSeq

	// A stale tick (an earlier, superseded retarget) must not fetch
	_, cmd := m.Update(model.PaneFetchDueMsg{SwitchEpoch: m.switchEpoch, LoadSeq: seq - 1})
	if cmd != nil {
		t.Error("expected a stale fetch tick to be dropped")
	}

	// The current tick dispatches the fetch for the pane's target
	_, cmd = m.Update(model.PaneFetchDueMsg{SwitchEpoch: m.switchEpoch, LoadSeq: seq})
	if cmd == nil {
		t.Fatal("expected the due fetch to dispatch")
	}
	// The fixture has no server, so the producers error — with gating intact
	var found bool
	for _, msg := range collectMsgs(t, cmd) {
		if errMsg, ok := msg.(model.EventsErrorMsg); ok {
			found = true
			if errMsg.Target != m.state.Events.Target || errMsg.LoadSeq != seq {
				t.Errorf("expected the fetch to carry the current target and seq, got %+v", errMsg)
			}
		}
	}
	if !found {
		t.Error("expected an events fetch to dispatch")
	}
}

func TestEventsPane_RetargetSkipsWhenTargetUnchanged(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel()) // pod is the last row
	teaModel, _ := m.Update(model.EventsLoadedMsg{
		Target:      m.state.Events.Target,
		Items:       []model.ResourceEvent{{Reason: "BackOff"}},
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	m = teaModel.(*Model)

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("j")) // cursor cannot move further
	mm := teaModel.(*Model)

	if mm.state.Events.Loading || len(mm.state.Events.Items) != 1 {
		t.Errorf("expected no refetch when the selection did not change, got %+v", mm.state.Events)
	}
	if cmd != nil {
		t.Error("expected no fetch command when the target is unchanged")
	}
}

func TestEventsPane_CursorOntoMissingResource_ShowsNotice(t *testing.T) {
	m := buildEventsPaneTestModel()
	ns := "argonaut-demo"
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "pod-uid", Version: "v1", Kind: "Pod", Name: "web-1", Namespace: &ns},
		{Group: "argoproj.io", Version: "v1alpha1", Kind: "Rollout", Name: "bluegreen-demo", Namespace: &ns},
	}})
	// Siblings sort by name: index 1 = Rollout (no UID), index 2 = Pod.
	// Open on the Pod, then move up onto the UID-less Rollout row.
	m.treeView.SetSelectedIndex(2)
	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	m = teaModel.(*Model)
	if m.state.Events.Target.Resource.UID != "pod-uid" {
		t.Fatalf("setup: expected to open on the pod, got %+v", m.state.Events.Target)
	}

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("k"))
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st.Notice == "" || st.Loading {
		t.Errorf("expected an inline notice without an events fetch, got %+v", st)
	}
	if !st.DetailsLoading || cmd == nil {
		t.Error("expected the details fetch to still run for a Missing resource")
	}
}

// While the pane is open it refetches on the configured interval — silently,
// without flipping into the loading placeholder.
func TestPaneRefreshDue_RefetchesInBackground(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.config.Events.RefreshInterval = "10s"
	teaModel, _ := m.Update(model.EventsLoadedMsg{
		Target:      m.state.Events.Target,
		Items:       []model.ResourceEvent{{Reason: "BackOff"}},
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	m = teaModel.(*Model)
	teaModel, _ = m.Update(model.SyncStatusLoadedMsg{
		Target:      model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"},
		Details:     &model.SyncStatusDetails{Phase: "Succeeded"},
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	m = teaModel.(*Model)

	teaModel, cmd := m.Update(model.PaneRefreshDueMsg{SwitchEpoch: m.switchEpoch, LoadSeq: m.state.Events.LoadSeq})
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st.Loading || len(st.Items) != 1 {
		t.Errorf("a background refresh must not blank the pane, got %+v", st)
	}
	if cmd == nil {
		t.Fatal("expected the refresh fetches (and the next tick) to be scheduled")
	}
}

func TestPaneRefreshDue_StaleSeq_IsDropped(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.config.Events.RefreshInterval = "10s"

	teaModel, cmd := m.Update(model.PaneRefreshDueMsg{SwitchEpoch: m.switchEpoch, LoadSeq: m.state.Events.LoadSeq - 1})
	_ = teaModel
	if cmd != nil {
		t.Error("a stale refresh tick must be dropped")
	}
}

// A landed refresh stamps the pane so the border can show when the data was
// last updated — a blink is not a signal.
func TestEventsLoadedMsg_StampsLastRefreshed(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	stamp := time.Date(2026, 8, 4, 18, 35, 12, 0, time.UTC)
	m.now = func() time.Time { return stamp }

	teaModel, _ := m.Update(model.EventsLoadedMsg{
		Target:      m.state.Events.Target,
		Items:       []model.ResourceEvent{{Reason: "Pulled"}},
		SwitchEpoch: m.switchEpoch,
		LoadSeq:     m.state.Events.LoadSeq,
	})
	mm := teaModel.(*Model)

	if !mm.state.Events.LastRefreshed.Equal(stamp) {
		t.Errorf("expected the pane stamped with the refresh time, got %v", mm.state.Events.LastRefreshed)
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

// Two apps share a name; the summary under the tree root must come from the
// tree-scoped app, not the first name-match in the list.
func TestResourceTreeLoaded_SyncSummary_DisambiguatesAppByNamespace(t *testing.T) {
	m := buildEventsPaneTestModel()
	nsArgocd := "argocd"
	nsTeamA := "team-a"
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd, SyncOp: &model.SyncOpSummary{
			Phase: "Failed", FinishedAt: time.Now().Add(-time.Hour),
		}},
		{Name: "my-app", AppNamespace: &nsTeamA, SyncOp: &model.SyncOpSummary{
			Phase: "Succeeded", FinishedAt: time.Now().Add(-time.Minute),
		}},
	}
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "my-app", AppNamespace: &nsTeamA}
	m.treeView = treeview.NewTreeView(100, 30)
	treeJSON, err := json.Marshal(api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	teaModel, _ := m.Update(model.ResourceTreeLoadedMsg{
		AppName:     "my-app",
		Health:      "Healthy",
		Sync:        "Synced",
		TreeJSON:    treeJSON,
		SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	rendered := mm.treeView.Render()
	if !strings.Contains(rendered, "Succeeded") {
		t.Error("expected the tree-scoped app's summary (Succeeded)")
	}
	if strings.Contains(rendered, "Failed") {
		t.Error("the same-named app from another namespace must not leak its summary")
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

	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)
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
	m.switchEpoch = 42 // nonzero so an omitted SwitchEpoch cannot pass vacuously
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	target := model.EventsTarget{AppName: "test-app", AppNamespace: "test-namespace"}

	msg := m.loadEvents(target, 1)()

	loaded, ok := msg.(model.EventsLoadedMsg)
	if !ok {
		t.Fatalf("expected EventsLoadedMsg, got %T: %+v", msg, msg)
	}
	if loaded.Target != target || loaded.SwitchEpoch != m.switchEpoch || loaded.LoadSeq != 1 {
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
	m.switchEpoch = 42 // nonzero so an omitted SwitchEpoch cannot pass vacuously
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	target := model.SyncStatusTarget{AppName: "test-app", AppNamespace: "test-namespace"}

	msg := m.loadSyncStatus(target, 1)()

	loaded, ok := msg.(model.SyncStatusLoadedMsg)
	if !ok {
		t.Fatalf("expected SyncStatusLoadedMsg, got %T: %+v", msg, msg)
	}
	if loaded.Target != target || loaded.SwitchEpoch != m.switchEpoch || loaded.LoadSeq != 1 {
		t.Errorf("expected gating fields carried through, got %+v", loaded)
	}
	if loaded.Details == nil || loaded.Details.Phase != "Failed" ||
		loaded.Details.Revision != "a1b2c3d" || len(loaded.Details.Resources) != 1 {
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
