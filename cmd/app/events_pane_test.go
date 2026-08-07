package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/config"
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

	if mm.state.Mode != model.ModeNormal {
		t.Fatalf("expected the pane to open without a mode change, got %s", mm.state.Mode)
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

	if mm.state.Events == nil {
		t.Fatal("expected enter on a resource row to open events")
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

	if mm.state.Events == nil {
		t.Fatal("expected enter on the root to open app events")
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

	if mm.state.Events == nil {
		t.Fatal("expected the pane to open as a lens")
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

	teaModel, _ := m.handleKeyMsg(testKeyMsg("e")) // toggle closed
	m = openEventsPane(t, teaModel.(*Model))       // reopen: a fresh load supersedes the stale one

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

	teaModel, _ = m.handleKeyMsg(testKeyMsg("e"))                 // toggle closed
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
	if mm.state.Events == nil {
		t.Error("a failed fetch must not close the pane")
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

// e toggles the pane: the one escape hatch for hiding an auto-opened pane,
// and the way back in.
func TestTreeKeyE_TogglesThePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	cursorBefore := m.treeView.SelectedIndex()

	teaModel, _ := m.handleKeyMsg(testKeyMsg("e"))
	mm := teaModel.(*Model)

	if mm.state.Events != nil {
		t.Fatal("expected e to close the open pane")
	}
	if mm.state.Navigation.View != model.ViewTree || mm.treeView.SelectedIndex() != cursorBefore {
		t.Error("toggling the pane must not leave the tree or move the cursor")
	}

	teaModel, _ = mm.handleKeyMsg(testKeyMsg("e"))
	mm = teaModel.(*Model)
	if mm.state.Events == nil {
		t.Error("expected e to reopen the pane")
	}
}

// The pane is part of the view, not a layer over it: esc leaves the tree
// view exactly as it did before the pane existed, taking the pane with it.
func TestEventsPane_EscLeavesTreeViewLikeAlways(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("esc"))
	mm := teaModel.(*Model)

	if mm.state.Navigation.View != model.ViewApps {
		t.Fatalf("expected esc to leave the tree for the apps view, got %s", mm.state.Navigation.View)
	}
	if mm.state.Events != nil {
		t.Error("expected the pane gone with the tree session")
	}
}

func TestEventsPane_QLeavesTreeViewLikeAlways(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("q"))
	mm := teaModel.(*Model)

	if mm.state.Navigation.View != model.ViewApps || mm.state.Events != nil {
		t.Errorf("expected q to return to the apps view with no pane state, got view %s events %+v",
			mm.state.Navigation.View, mm.state.Events)
	}
}

// The pane is a lens, not a modal: action hotkeys act on the selected tree
// row and the pane stays open through the flow.
func TestEventsPane_ActionKeysActWithThePaneOpen(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("d")) // diff the selected resource
	mm := teaModel.(*Model)

	if mm.state.Events == nil {
		t.Error("expected the pane to stay open")
	}
	if mm.state.Diff == nil || !mm.state.Diff.Loading || cmd == nil {
		t.Errorf("expected d to start the resource diff, got %+v", mm.state.Diff)
	}
}

func TestEventsPane_CtrlDOpensDeleteConfirmationOverThePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("ctrl+d"))
	mm := teaModel.(*Model)

	if mm.state.Events == nil {
		t.Error("expected the pane to stay open behind the modal")
	}
	if mm.state.Mode != model.ModeConfirmResourceDelete {
		t.Fatalf("expected the delete confirmation, got mode %s", mm.state.Mode)
	}
}

func TestEventsPane_SyncConfirmationOverThePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg("s"))
	mm := teaModel.(*Model)

	if mm.state.Events == nil {
		t.Error("expected the pane to stay open behind the sync confirmation")
	}
	if mm.state.Mode != model.ModeConfirmResourceSync {
		t.Fatalf("expected the sync confirmation, got mode %s", mm.state.Mode)
	}
	// Cancelling the modal lands back on the tree with the pane intact
	teaModel, _ = mm.handleKeyMsg(testKeyMsg("esc"))
	mm = teaModel.(*Model)
	if mm.state.Mode != model.ModeNormal || mm.state.Events == nil {
		t.Errorf("expected the pane to survive the modal round trip, got mode %s", mm.state.Mode)
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

// u/i sit right above j/k: scroll the pane without leaving the tree's home
// row. Shift arrows do the same for non-vimmers.
func TestEventsPane_UIAndShiftArrowsScrollTheViewport(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	for _, tc := range []struct {
		key   tea.KeyMsg
		want  int
		label string
	}{
		{testKeyMsg("u"), 1, "u scrolls down"},
		{testKeyMsg("i"), 0, "i scrolls up"},
		{tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}, 1, "shift+down scrolls down"},
		{tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}, 0, "shift+up scrolls up"},
		{tea.KeyPressMsg{Code: 'e', Mod: tea.ModCtrl}, 0, "ctrl+e is retired"},
		{tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl}, 0, "ctrl+y is retired"},
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
	if mm.state.Mode != model.ModeNormal {
		t.Fatalf("expected esc to land back on the tree, got %s", mm.state.Mode)
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

// :events off hides the pane for this session: it closes now and stays away
// on the next tree load.
func TestEventsCommand_Off_ClosesPaneAndStopsAutoOpen(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "events off")

	if m.state.Events != nil {
		t.Fatal("expected :events off to close the pane")
	}
	treeJSON, err := json.Marshal(api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	teaModel, _ = m.Update(model.ResourceTreeLoadedMsg{
		AppName: "test-app", Health: "Healthy", Sync: "Synced",
		TreeJSON: treeJSON, SwitchEpoch: m.switchEpoch,
	})
	if teaModel.(*Model).state.Events != nil {
		t.Error("expected auto-open to stay off for the session")
	}
}

// :events on undoes a session off — reopening immediately in the tree view.
func TestEventsCommand_On_ReopensThePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	teaModel, _ := m.handleKeyMsg(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "events off")

	teaModel, _ = m.handleKeyMsg(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "events on")

	if m.state.Events == nil {
		t.Error("expected :events on to open the pane in the tree view")
	}
}

// A bare :events (or junk argument) explains itself instead of guessing.
func TestEventsCommand_WithoutOnOff_ExplainsUsage(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	m.state.Mode = model.ModeCommand
	m.inputComponents.SetCommandValue("events")
	m.state.UI.Command = "events"
	teaModel, cmd := m.handleEnhancedCommandModeKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := teaModel.(*Model)

	if mm.state.Events == nil {
		t.Error("expected the open pane left untouched")
	}
	var usage string
	for _, msg := range collectMsgs(t, cmd) {
		if sc, ok := msg.(model.StatusChangeMsg); ok {
			usage = sc.Status
		}
	}
	if !strings.Contains(usage, "on|off") {
		t.Errorf("expected a usage hint, got %q", usage)
	}
}

// "always" makes the choice outlive the session: it lands in the config file.
func TestEventsCommand_OffAlways_PersistsToConfig(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("ARGONAUT_CONFIG", cfgPath)
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "events off always")

	if m.state.Events != nil {
		t.Error("expected the pane closed")
	}
	loaded, err := config.LoadArgonautConfig()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.IsEventsAutoOpenEnabled() {
		t.Error("expected auto_open = false persisted to the config file")
	}
}

// :sync stays in the tree view, so — like the s key — it must not eat the pane.
func TestSyncCommand_InTreeView_KeepsThePaneOpen(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	teaModel, _ := m.handleKeyMsg(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "sync")

	if m.state.Mode != model.ModeConfirmResourceSync {
		t.Fatalf("expected the sync confirmation, got mode %s", m.state.Mode)
	}
	if m.state.Events == nil {
		t.Error("expected the pane to stay open behind the sync confirmation")
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

	if !strings.Contains(stripANSI(mm.treeView.Render()), "· ✖ sync failed") {
		t.Error("expected the root row's sync summary after a live app update")
	}
}

// The watch stream is the pane's freshness signal: an update for the app the
// pane is showing arms a debounced refresh, so a sync or health change shows
// up without waiting for the polling interval.
func TestAppsBatchUpdate_ForPaneApp_ArmsWatchRefresh(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.watchGeneration = 7 // a foreign generation keeps the consume chain out of cmd
	updated := m.state.Apps[0]
	updated.SyncOp = &model.SyncOpSummary{Phase: "Succeeded", FinishedAt: time.Now()}

	_, cmd := m.Update(model.AppsBatchUpdateMsg{
		Updates:     []model.AppUpdatedMsg{{App: updated}},
		SwitchEpoch: m.switchEpoch,
	})

	if cmd == nil {
		t.Fatal("expected a watch update for the pane's app to arm a refresh")
	}
}

// A sync emits a burst of watch updates; one armed tick absorbs the burst
// instead of refetching per update.
func TestAppsBatchUpdate_WhileRefreshArmed_Coalesces(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.watchGeneration = 7
	updated := m.state.Apps[0]
	batch := model.AppsBatchUpdateMsg{
		Updates:     []model.AppUpdatedMsg{{App: updated}},
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, cmd := m.Update(batch)
	if cmd == nil {
		t.Fatal("setup: the first update must arm the refresh")
	}

	_, cmd = teaModel.(*Model).Update(batch)

	if cmd != nil {
		t.Error("expected updates within the armed window to coalesce into the pending tick")
	}
}

// Only the app the pane is showing is a freshness signal; churn on other
// apps must not refetch it.
func TestAppsBatchUpdate_ForOtherApp_DoesNotArmWatchRefresh(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.watchGeneration = 7
	other := m.state.Apps[1] // zzz-other-app

	_, cmd := m.Update(model.AppsBatchUpdateMsg{
		Updates:     []model.AppUpdatedMsg{{App: other}},
		SwitchEpoch: m.switchEpoch,
	})

	if cmd != nil {
		t.Error("expected an unrelated app's update to leave the pane alone")
	}
}

// When the coalescing window closes, the pane refetches in the background —
// no loading placeholder — and the next watch update can arm again.
func TestPaneWatchRefreshDue_RefetchesAndDisarms(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.watchGeneration = 7
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
	batch := model.AppsBatchUpdateMsg{
		Updates:     []model.AppUpdatedMsg{{App: m.state.Apps[0]}},
		SwitchEpoch: m.switchEpoch,
	}
	teaModel, _ = m.Update(batch)
	m = teaModel.(*Model)

	teaModel, cmd := m.Update(paneWatchRefreshDueMsg{switchEpoch: m.switchEpoch, loadSeq: m.state.Events.LoadSeq})
	m = teaModel.(*Model)

	if m.state.Events.Loading || len(m.state.Events.Items) != 1 {
		t.Errorf("a watch-triggered refresh must not blank the pane, got %+v", m.state.Events)
	}
	if cmd == nil {
		t.Fatal("expected the refresh fetches to dispatch")
	}
	var gotEvents bool
	for _, msg := range collectMsgs(t, cmd) {
		if _, ok := msg.(model.EventsErrorMsg); ok {
			gotEvents = true
		}
	}
	if !gotEvents {
		t.Error("expected an events refetch to dispatch")
	}

	if _, cmd = m.Update(batch); cmd == nil {
		t.Error("expected the fired tick to disarm, letting the next update arm again")
	}
}

// A retarget between the arm and the tick supersedes the refresh: the new
// target's own load is already running.
func TestPaneWatchRefreshDue_FromSupersededLoad_IsDropped(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
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

	_, cmd := m.Update(paneWatchRefreshDueMsg{switchEpoch: m.switchEpoch, loadSeq: m.state.Events.LoadSeq - 1})

	if cmd != nil {
		t.Error("expected a superseded watch-refresh tick to be dropped")
	}
}

// Two apps sharing a name in different ArgoCD namespaces: only the pane's
// own app is a freshness signal (ADR-0004).
func TestAppsBatchUpdate_SameNameOtherNamespace_DoesNotArmWatchRefresh(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.watchGeneration = 7
	otherNs := "team-a"
	imposter := m.state.Apps[0]
	imposter.AppNamespace = &otherNs

	_, cmd := m.Update(model.AppsBatchUpdateMsg{
		Updates:     []model.AppUpdatedMsg{{App: imposter}},
		SwitchEpoch: m.switchEpoch,
	})

	if cmd != nil {
		t.Error("expected the same-named app from another namespace to leave the pane alone")
	}
}

// The pane is part of the tree view: it opens by itself once the tree loads,
// targeting the selected row, and starts its fetches.
func TestResourceTreeLoaded_AutoOpensThePane(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Events = nil
	treeJSON, err := json.Marshal(api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	teaModel, cmd := m.Update(model.ResourceTreeLoadedMsg{
		AppName: "test-app", Health: "Healthy", Sync: "Synced",
		TreeJSON: treeJSON, SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	st := mm.state.Events
	if st == nil {
		t.Fatal("expected the pane to auto-open once the tree loaded")
	}
	if st.Target.AppName != "test-app" {
		t.Errorf("expected the pane targeted at the loaded app, got %+v", st.Target)
	}
	if cmd == nil {
		t.Error("expected the auto-opened pane to start its fetches")
	}
}

func TestResourceTreeLoaded_AutoOpenDisabled_KeepsPaneClosed(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Events = nil
	off := false
	m.config.Events.AutoOpen = &off
	treeJSON, err := json.Marshal(api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "web"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	teaModel, _ := m.Update(model.ResourceTreeLoadedMsg{
		AppName: "test-app", Health: "Healthy", Sync: "Synced",
		TreeJSON: treeJSON, SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	if mm.state.Events != nil {
		t.Error("expected the pane to stay closed with auto_open = false")
	}
}

// A later tree load (multi-app streams, watch reloads) must not reset a pane
// that is already open and possibly retargeted.
func TestResourceTreeLoaded_PaneAlreadyOpen_IsLeftAlone(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel()) // opened on the Pod row
	targetBefore := m.state.Events.Target
	seqBefore := m.state.Events.LoadSeq
	treeJSON, err := json.Marshal(api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "pod-uid", Version: "v1", Kind: "Pod", Name: "web-1"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	teaModel, _ := m.Update(model.ResourceTreeLoadedMsg{
		AppName: "test-app", Health: "Healthy", Sync: "Synced",
		TreeJSON: treeJSON, SwitchEpoch: m.switchEpoch,
	})
	mm := teaModel.(*Model)

	if mm.state.Events.Target != targetBefore || mm.state.Events.LoadSeq != seqBefore {
		t.Errorf("expected the open pane untouched by a tree reload, got %+v", mm.state.Events)
	}
}

// Drilling into a child app replaces the tree session; the pane must not
// keep pointing at the old app's resources while the child tree loads.
func TestDrillIntoChildApp_ClosesTheStalePane(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	childNs := "argocd"
	m.state.Apps = append(m.state.Apps, model.App{Name: "child-app", Sync: "Synced", Health: "Healthy", AppNamespace: &childNs})
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "child-app-uid", Group: "argoproj.io", Version: "v1alpha1", Kind: "Application", Name: "child-app", Namespace: &childNs},
	}})
	m.treeView.SetSelectedIndex(1) // the child Application row

	teaModel, _ := m.handleKeyMsg(testKeyMsg("enter"))
	mm := teaModel.(*Model)

	if mm.state.Events != nil {
		t.Error("expected the stale pane closed while the child tree loads (auto-open retargets it)")
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
	if !strings.Contains(rendered, "· ✔ synced") {
		t.Error("expected the tree-scoped app's summary (synced)")
	}
	if strings.Contains(rendered, "sync failed") {
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

	if !strings.Contains(stripANSI(mm.treeView.Render()), "· ✔ synced") {
		t.Error("expected the freshly loaded tree to carry the app's sync summary")
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

	if mm.state.Mode != model.ModeNormal {
		t.Fatalf("expected the pane to open without a mode change, got %s", mm.state.Mode)
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
