package main

import (
	"charm.land/lipgloss/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
)

func buildRollbackSessionServer(t *testing.T, appJSON string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(appJSON))
	}))
}

func TestStartRollbackSession_ReportsAutoSyncAndNewestFirstRows(t *testing.T) {
	server := buildRollbackSessionServer(t, `{
		"metadata": {"name": "demo-app"},
		"spec": {"syncPolicy": {"automated": {}}},
		"status": {
			"sync": {"revision": "ccc333"},
			"history": [
				{"id": 1, "revision": "aaa111", "deployedAt": "2026-08-01T10:00:00Z"},
				{"id": 2, "revision": "bbb222", "deployedAt": "2026-08-02T10:00:00Z"},
				{"id": 3, "revision": "ccc333", "deployedAt": "2026-08-03T10:00:00Z"}
			]
		}
	}`)
	defer server.Close()

	m := buildSyncTestModel(100, 30)
	m.switchEpoch = 42
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}

	msg := m.startRollbackSession("demo-app", nil)()
	loaded, ok := msg.(model.RollbackHistoryLoadedMsg)
	if !ok {
		t.Fatalf("expected RollbackHistoryLoadedMsg, got %T", msg)
	}

	if loaded.SwitchEpoch != 42 {
		t.Errorf("SwitchEpoch = %d, want 42", loaded.SwitchEpoch)
	}
	if !loaded.AutoSyncEnabled {
		t.Error("AutoSyncEnabled = false, want true for automated sync policy")
	}
	if len(loaded.Rows) != 3 || loaded.Rows[0].ID != 3 || loaded.Rows[2].ID != 1 {
		ids := make([]int, len(loaded.Rows))
		for i, r := range loaded.Rows {
			ids[i] = r.ID
		}
		t.Errorf("row IDs = %v, want newest-first [3 2 1]", ids)
	}
}

func TestStartRollbackSession_AutoSyncOffWhenPolicyDisabledOrAbsent(t *testing.T) {
	cases := []struct {
		name string
		spec string
	}{
		{"no sync policy", `{}`},
		{"no automated policy", `{"syncPolicy": {}}`},
		{"automated explicitly disabled", `{"syncPolicy": {"automated": {"enabled": false}}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := buildRollbackSessionServer(t, `{
				"metadata": {"name": "demo-app"},
				"spec": `+tc.spec+`,
				"status": {"sync": {"revision": "aaa111"}, "history": []}
			}`)
			defer server.Close()

			m := buildSyncTestModel(100, 30)
			m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}

			msg := m.startRollbackSession("demo-app", nil)()
			loaded, ok := msg.(model.RollbackHistoryLoadedMsg)
			if !ok {
				t.Fatalf("expected RollbackHistoryLoadedMsg, got %T", msg)
			}
			if loaded.AutoSyncEnabled {
				t.Error("AutoSyncEnabled = true, want false")
			}
		})
	}
}

func rollbackHistoryRows(n int) []model.RollbackRow {
	rows := make([]model.RollbackRow, n)
	for i := range rows {
		rows[i] = model.RollbackRow{ID: n - i, Revision: "rev" + string(rune('a'+i))}
	}
	return rows
}

func TestRollbackHistoryLoaded_FetchesOnlySelectedRowMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"author": "jane", "date": "2026-08-07T13:58:00Z", "message": "fix"}`))
	}))
	defer server.Close()

	m := buildSyncTestModel(100, 30)
	m.switchEpoch = 42
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}

	teaModel, cmd := m.Update(model.RollbackHistoryLoadedMsg{
		AppName:         "demo-app",
		Rows:            rollbackHistoryRows(15),
		CurrentRevision: "reva",
		AutoSyncEnabled: true,
		SwitchEpoch:     42,
	})
	m = teaModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("expected rollback state to be initialized")
	}
	if !m.state.Rollback.AutoSyncEnabled {
		t.Error("AutoSyncEnabled not carried into rollback state")
	}

	// The list shows commit subjects for every visible row, so the whole
	// visible window loads up front — not just the row under the cursor.
	fetched := map[int]bool{}
	for _, msg := range collectMsgs(t, cmd) {
		if meta, ok := msg.(model.RollbackMetadataLoadedMsg); ok {
			fetched[meta.RowIndex] = true
		}
	}
	window := m.rollbackPageSize()
	for i := 0; i < min(window, 15); i++ {
		if !fetched[i] {
			t.Errorf("row %d is visible but its metadata was not fetched", i)
		}
	}
}

func TestRollbackHistoryLoaded_StaleEpochIgnored(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	m.switchEpoch = 42

	teaModel, cmd := m.Update(model.RollbackHistoryLoadedMsg{
		AppName:     "demo-app",
		Rows:        rollbackHistoryRows(3),
		SwitchEpoch: 41,
	})
	m = teaModel.(*Model)

	if m.state.Rollback != nil {
		t.Error("stale history load must not initialize rollback state")
	}
	if cmd != nil {
		t.Error("stale history load must not dispatch metadata fetches")
	}
}

func buildRollbackListModel(rows int) *Model {
	m := buildSyncTestModel(100, 30)
	m.switchEpoch = 42
	m.state.Mode = model.ModeRollback
	m.state.Rollback = &model.RollbackState{
		AppName: "demo-app",
		Rows:    rollbackHistoryRows(rows),
		Mode:    "list",
		Watch:   true,
	}
	return m
}

func TestRollbackNav_SchedulesDebouncedMetadataFetch(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("j"))
	m = teaModel.(*Model)

	if m.state.Rollback.SelectedIdx != 1 {
		t.Fatalf("SelectedIdx = %d, want 1", m.state.Rollback.SelectedIdx)
	}
	if m.state.Rollback.LoadSeq != 1 {
		t.Errorf("LoadSeq = %d, want 1 after first cursor move", m.state.Rollback.LoadSeq)
	}
	if cmd == nil {
		t.Fatal("expected a debounce tick command")
	}
	due, ok := cmd().(rollbackMetaDueMsg)
	if !ok || due.loadSeq != 1 || due.switchEpoch != 42 {
		t.Errorf("tick msg = %+v, want rollbackMetaDueMsg{switchEpoch: 42, loadSeq: 1}", due)
	}
}

func TestRollbackMetaDue_StaleSeqIgnored(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.LoadSeq = 3

	_, cmd := m.Update(rollbackMetaDueMsg{switchEpoch: 42, loadSeq: 2})
	if cmd != nil {
		t.Error("expected a stale debounce tick to be dropped")
	}
	_, cmd = m.Update(rollbackMetaDueMsg{switchEpoch: 41, loadSeq: 3})
	if cmd != nil {
		t.Error("expected a stale-epoch debounce tick to be dropped")
	}
}

func TestRollbackMetaDue_FetchesRowUnderCursor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"author": "jane", "date": "2026-08-07T13:58:00Z", "message": "fix"}`))
	}))
	defer server.Close()

	m := buildRollbackListModel(5)
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	m.state.Rollback.SelectedIdx = 2
	m.state.Rollback.LoadSeq = 1

	_, cmd := m.Update(rollbackMetaDueMsg{switchEpoch: 42, loadSeq: 1})
	if cmd == nil {
		t.Fatal("expected the due tick to dispatch a metadata fetch")
	}
	fetched := map[int]bool{}
	for _, msg := range collectMsgs(t, cmd) {
		if meta, ok := msg.(model.RollbackMetadataLoadedMsg); ok {
			fetched[meta.RowIndex] = true
		}
	}
	if !fetched[2] {
		t.Errorf("fetched rows = %v, want the row under the cursor (2) included", fetched)
	}
}

func TestRollbackMetaDue_SkipsWhenAllVisibleRowsLoaded(t *testing.T) {
	m := buildRollbackListModel(5)
	author := "jane"
	for i := range m.state.Rollback.Rows {
		m.state.Rollback.Rows[i].Author = &author
	}
	m.state.Rollback.LoadSeq = 1

	_, cmd := m.Update(rollbackMetaDueMsg{switchEpoch: 42, loadSeq: 1})
	if cmd != nil {
		t.Error("expected no fetch when every visible row's metadata is loaded")
	}
}

func TestRollbackEnter_OpensConfirmWithCancelPreselected(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.SelectedIdx = 2
	m.state.Rollback.ConfirmSelected = 0

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("enter"))
	m = teaModel.(*Model)

	// Cancel is the safe default — confirming a rollback takes a deliberate move
	if m.state.Rollback.Mode != "confirm" || m.state.Rollback.ConfirmSelected != 1 {
		t.Errorf("state = (%s, confirmSelected=%d), want (confirm, 1=Cancel)",
			m.state.Rollback.Mode, m.state.Rollback.ConfirmSelected)
	}
}

func TestRollbackConfirmY_ExecutesEvenWithCancelPreselected(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.Mode = "confirm"
	m.state.Rollback.ConfirmSelected = 1

	teaModel, cmd := m.handleRollbackModeKeys(testKeyMsg("y"))
	m = teaModel.(*Model)

	if !m.state.Rollback.Loading || cmd == nil {
		t.Errorf("state = (loading=%v, cmd=%v), want rollback executing", m.state.Rollback.Loading, cmd)
	}
}

func TestRollbackYInList_DoesNotExecute(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, cmd := m.handleRollbackModeKeys(testKeyMsg("y"))
	m = teaModel.(*Model)

	if m.state.Rollback.Loading || cmd != nil {
		t.Error("y outside the confirm modal must not trigger a rollback")
	}
}

func TestRollbackEscInConfirm_ReturnsToListKeepingSession(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.Mode = "confirm"
	m.state.Rollback.SelectedIdx = 2
	m.state.Rollback.Prune = true

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("esc"))
	m = teaModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("esc in confirm must keep the rollback session")
	}
	if m.state.Rollback.Mode != "list" || m.state.Rollback.SelectedIdx != 2 {
		t.Errorf("state = (%s, idx=%d), want back to (list, 2)",
			m.state.Rollback.Mode, m.state.Rollback.SelectedIdx)
	}
	if m.state.Mode != model.ModeRollback {
		t.Errorf("mode = %s, want still rollback", m.state.Mode)
	}
}

func TestRollbackEscInList_ClosesSession(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("esc"))
	m = teaModel.(*Model)

	if m.state.Rollback != nil || m.state.Mode != model.ModeNormal {
		t.Errorf("state = (rollback=%v, mode=%s), want session closed and ModeNormal",
			m.state.Rollback != nil, m.state.Mode)
	}
}

func TestRollbackConfirmCancel_ReturnsToList(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.Mode = "confirm"
	m.state.Rollback.ConfirmSelected = 1

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("enter"))
	m = teaModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("cancel must keep the rollback session")
	}
	if m.state.Rollback.Mode != "list" {
		t.Errorf("mode = %s, want list", m.state.Rollback.Mode)
	}
}

func TestRollbackPruneWatchToggles_OnlyInConfirm(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("p"))
	m = teaModel.(*Model)
	if m.state.Rollback.Prune {
		t.Error("p in list mode must not toggle prune")
	}

	m.state.Rollback.Mode = "confirm"
	teaModel, _ = m.handleRollbackModeKeys(testKeyMsg("p"))
	m = teaModel.(*Model)
	teaModel, _ = m.handleRollbackModeKeys(testKeyMsg("w"))
	m = teaModel.(*Model)
	if !m.state.Rollback.Prune || m.state.Rollback.Watch {
		t.Errorf("toggles = (prune=%v, watch=%v), want (true, false)",
			m.state.Rollback.Prune, m.state.Rollback.Watch)
	}
}

func TestRollbackExecute_DisablesAutoSyncBeforeRollingBack(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	m := buildRollbackListModel(5)
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	m.state.Rollback.Mode = "confirm"
	m.state.Rollback.AutoSyncEnabled = true

	_, cmd := m.handleRollbackModeKeys(testKeyMsg("enter"))
	if cmd == nil {
		t.Fatal("expected the rollback command to dispatch")
	}
	msg := cmd()
	executed, ok := msg.(model.RollbackExecutedMsg)
	if !ok || !executed.Success {
		t.Fatalf("result = %+v, want successful RollbackExecutedMsg", msg)
	}

	want := []string{
		"PATCH /api/v1/applications/demo-app",
		"POST /api/v1/applications/demo-app/rollback",
	}
	if len(calls) != 2 || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("calls = %v, want %v", calls, want)
	}
}

func TestRollbackExecute_SkipsPatchWhenAutoSyncOff(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	m := buildRollbackListModel(5)
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	m.state.Rollback.Mode = "confirm"

	_, cmd := m.handleRollbackModeKeys(testKeyMsg("enter"))
	if cmd == nil {
		t.Fatal("expected the rollback command to dispatch")
	}
	cmd()

	if len(calls) != 1 || calls[0] != "POST /api/v1/applications/demo-app/rollback" {
		t.Errorf("calls = %v, want only the rollback POST", calls)
	}
}

func TestRollbackDiff_ComparesSelectedAgainstCurrentRevision(t *testing.T) {
	var revisions []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/manifests") {
			revisions = append(revisions, r.URL.Query().Get("revision"))
		}
		w.Write([]byte(`{"manifests": ["{\"kind\":\"ConfigMap\",\"metadata\":{\"name\":\"cm\"}}"]}`))
	}))
	defer server.Close()

	m := buildRollbackListModel(5)
	m.state.Server = &model.Server{BaseURL: server.URL, Token: "test-token"}
	m.state.Rollback.CurrentRevision = "reva"
	m.state.Rollback.SelectedIdx = 2 // revision "revc"

	_, cmd := m.handleRollbackModeKeys(testKeyMsg("d"))
	if cmd == nil {
		t.Fatal("expected d to dispatch a diff command")
	}
	msg := cmd()

	noDiff, ok := msg.(rollbackDiffNoChangesMsg)
	if !ok {
		t.Fatalf("result = %+v, want rollbackDiffNoChangesMsg for identical manifests", msg)
	}
	if noDiff.switchEpoch != 42 || noDiff.revision != "revc" {
		t.Errorf("msg = %+v, want epoch 42 and revision revc", noDiff)
	}
	if len(revisions) != 2 || revisions[0] != "revc" || revisions[1] != "reva" {
		t.Errorf("manifest fetches = %v, want [revc reva] (selected, then current)", revisions)
	}
}

// A no-changes diff must not leave the rollback view (a NoDiff modal would
// dismiss back to the apps list) — it surfaces as a notice in the pane.
func TestRollbackDiffNoChanges_ShowsNoticeAndStaysInRollback(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, _ := m.Update(rollbackDiffNoChangesMsg{switchEpoch: 42, revision: "revc"})
	m = teaModel.(*Model)

	if m.state.Mode != model.ModeRollback {
		t.Errorf("mode = %s, want still rollback", m.state.Mode)
	}
	if m.state.Rollback.Notice == "" {
		t.Error("expected a no-differences notice on the rollback state")
	}

	l := m.rollbackPaneLayout(m.renderBanner())
	if !strings.Contains(stripANSI(m.renderRollbackDetail(l)), "No differences") {
		t.Error("detail pane should show the no-differences notice")
	}
}

func TestRollbackDiffNoChanges_StaleEpochIgnored(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, _ := m.Update(rollbackDiffNoChangesMsg{switchEpoch: 41, revision: "revc"})
	m = teaModel.(*Model)

	if m.state.Rollback.Notice != "" {
		t.Error("stale no-changes result must not set a notice")
	}
}

func TestRollbackNotice_ClearsOnCursorMove(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.Notice = "No differences vs revc"

	teaModel, _ := m.handleKeyMsg(testKeyMsg("j"))
	m = teaModel.(*Model)

	if m.state.Rollback.Notice != "" {
		t.Error("notice should clear when the cursor moves to another row")
	}
}

func TestRollbackPageSize_MatchesVisibleListRows(t *testing.T) {
	// The list frame is as tall as the history, capped at 10 rows — the
	// rest of the screen belongs to the detail pane.
	m := buildRollbackGoldenModel(100, 24) // 4 history entries
	if got := m.rollbackPageSize(); got != 4 {
		t.Errorf("rollbackPageSize = %d, want 4 (one row per history entry)", got)
	}

	m.state.Rollback.Rows = rollbackHistoryRows(25)
	if got := m.rollbackPageSize(); got != 10 {
		t.Errorf("rollbackPageSize = %d, want the 10-row cap", got)
	}
}

func TestRollbackDetailBody_MultilineCommitMessageStaysInFrame(t *testing.T) {
	m := buildRollbackListModel(2)
	message := "feat: add apps (#476)\n\n* fix: invalid image\n\nSigned-off-by: Alexandre Gaudreault <alexandre_gaudreault@intuit.com>"
	author := "Alexandre Gaudreault"
	m.state.Rollback.Rows[0].Author = &author
	m.state.Rollback.Rows[0].Message = &message

	for _, line := range m.renderRollbackDetailBody(46) {
		if strings.Contains(line, "\n") {
			t.Fatalf("body row contains a raw newline, breaking the pane frame: %q", line)
		}
	}
}

func TestRollbackConfirmModal_ShowsOnlyCommitSubject(t *testing.T) {
	m := buildRollbackListModel(2)
	message := "feat: one\n\n* two\n\nSigned-off-by: someone"
	m.state.Rollback.Rows[0].Message = &message
	m.state.Rollback.Mode = "confirm"

	out := stripANSI(m.renderRollbackConfirmModal())
	if !strings.Contains(out, "feat: one") {
		t.Errorf("modal should show the commit subject:\n%s", out)
	}
	if strings.Contains(out, "Signed-off-by") {
		t.Errorf("modal must not spill the commit body — the detail pane has it:\n%s", out)
	}
}

func TestRollbackDetailScroll_UIKeysMoveAndClamp(t *testing.T) {
	m := buildRollbackListModel(3)

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("u"))
	m = teaModel.(*Model)
	if m.state.Rollback.DetailOffset != 1 {
		t.Errorf("DetailOffset after u = %d, want 1", m.state.Rollback.DetailOffset)
	}

	teaModel, _ = m.handleRollbackModeKeys(testKeyMsg("i"))
	m = teaModel.(*Model)
	teaModel, _ = m.handleRollbackModeKeys(testKeyMsg("i"))
	m = teaModel.(*Model)
	if m.state.Rollback.DetailOffset != 0 {
		t.Errorf("DetailOffset after i past top = %d, want clamped to 0", m.state.Rollback.DetailOffset)
	}
}

func TestRollbackDetailScroll_ResetsOnCursorMove(t *testing.T) {
	m := buildRollbackListModel(3)
	m.state.Rollback.DetailOffset = 5

	teaModel, _ := m.handleKeyMsg(testKeyMsg("j"))
	m = teaModel.(*Model)
	if m.state.Rollback.DetailOffset != 0 {
		t.Errorf("DetailOffset after cursor move = %d, want 0 (pane retargeted)", m.state.Rollback.DetailOffset)
	}
}

func TestRollbackDetailScroll_ShowsMoreMarkers(t *testing.T) {
	m := buildRollbackGoldenModel(100, 14) // short terminal so the detail overflows
	long := strings.Repeat("line\n", 30)
	m.state.Rollback.Rows[1].Message = &long

	l := m.rollbackPaneLayout(m.renderBanner())
	if !strings.Contains(stripANSI(m.renderRollbackDetail(l)), "▼ more below") {
		t.Error("overflowing detail pane should show the ▼ more below marker")
	}

	m.state.Rollback.DetailOffset = 3
	out := stripANSI(m.renderRollbackDetail(l))
	if !strings.Contains(out, "▲ more above") {
		t.Error("scrolled detail pane should show the ▲ more above marker")
	}
}

func TestRollbackList_ShowsCommitSubjectBetweenShaAndAge(t *testing.T) {
	m := buildRollbackGoldenModel(100, 24)
	long := "fix: a very long commit subject that cannot possibly fit into the list column\n\nbody"
	m.state.Rollback.Rows[1].Message = &long

	l := m.rollbackPaneLayout(m.renderBanner())
	out := stripANSI(m.renderRollbackList(l))

	line := ""
	for _, candidate := range strings.Split(out, "\n") {
		if strings.Contains(candidate, "11223344") {
			line = candidate
		}
	}
	if line == "" {
		t.Fatal("row #29 not rendered")
	}
	subject := strings.Index(line, "fix: a")
	if subject == -1 {
		t.Fatalf("commit subject missing from list row: %q", line)
	}
	if !strings.Contains(line, "...") {
		t.Errorf("overflowing subject should be truncated with an ellipsis: %q", line)
	}
	if sha := strings.Index(line, "11223344"); sha > subject {
		t.Errorf("subject should come after the sha: %q", line)
	}
	if age := strings.Index(line, "21h ago"); age != -1 && age < subject {
		t.Errorf("subject should come before the deploy age: %q", line)
	}
	if strings.Contains(line, "body") {
		t.Errorf("only the subject line belongs in the list: %q", line)
	}
}

func TestRollbackInitiator_ViewingUserShownAsYou(t *testing.T) {
	m := buildRollbackGoldenModel(100, 24)
	m.currentUsername = "jane.doe" // row #30 was initiated by jane.doe
	m.state.Rollback.SelectedIdx = 0

	l := m.rollbackPaneLayout(m.renderBanner())
	listOut := stripANSI(m.renderRollbackList(l))
	if strings.Contains(listOut, "jane.doe") || !strings.Contains(listOut, "by you") {
		t.Errorf("list should read '<age> ago by you' for the viewing user:\n%s", listOut)
	}
	if !strings.Contains(listOut, "by bob.smith") {
		t.Errorf("other users' rows should read 'by <user>':\n%s", listOut)
	}
	detailOut := stripANSI(m.renderRollbackDetail(l))
	if strings.Contains(detailOut, "jane.doe") {
		t.Errorf("detail pane should name the viewing user 'you':\n%s", detailOut)
	}
}

func TestRollbackNav_FreshSessionStartsFromTop(t *testing.T) {
	m := buildRollbackListModel(8)

	// Move the cursor deep into the list in a first session…
	for range 5 {
		teaModel, _ := m.handleKeyMsg(testKeyMsg("j"))
		m = teaModel.(*Model)
	}
	if m.state.Rollback.SelectedIdx != 5 {
		t.Fatalf("setup: SelectedIdx = %d, want 5", m.state.Rollback.SelectedIdx)
	}

	// …the session ends (rollback executed / esc), and a new one loads
	m.state.Rollback = nil
	m.state.Mode = model.ModeNormal
	teaModel, _ := m.Update(model.RollbackHistoryLoadedMsg{
		AppName:     "demo-app",
		Rows:        rollbackHistoryRows(8),
		SwitchEpoch: 42,
	})
	m = teaModel.(*Model)
	m.state.Mode = model.ModeRollback

	teaModel, _ = m.handleKeyMsg(testKeyMsg("j"))
	m = teaModel.(*Model)
	if m.state.Rollback.SelectedIdx != 1 {
		t.Errorf("SelectedIdx after one j in a fresh session = %d, want 1 (stale navigator cursor)", m.state.Rollback.SelectedIdx)
	}
}

func TestRollbackColon_OpensCommandBarKeepingSession(t *testing.T) {
	m := buildRollbackListModel(5)

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg(":"))
	m = teaModel.(*Model)

	if m.state.Mode != model.ModeCommand {
		t.Errorf("mode = %s, want command", m.state.Mode)
	}
	if m.state.Rollback == nil {
		t.Error("entering command mode must keep the rollback session")
	}
}

func TestRollbackCommandBarEsc_ReturnsToRollbackView(t *testing.T) {
	m := buildRollbackListModel(5)
	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg(":"))
	m = teaModel.(*Model)

	teaModel, _ = m.handleEnhancedCommandModeKeys(testKeyMsg("esc"))
	m = teaModel.(*Model)

	if m.state.Mode != model.ModeRollback {
		t.Errorf("mode after esc = %s, want back in rollback", m.state.Mode)
	}
	if m.state.Rollback == nil {
		t.Error("rollback session must survive an abandoned command")
	}
}

// A command is a jump somewhere else — executing one ends the rollback
// session instead of leaving it dangling under another view.
func TestRollbackCommandExecution_ClosesSession(t *testing.T) {
	m := buildRollbackListModel(5)
	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg(":"))
	m = runCommand(t, teaModel.(*Model), "apps")

	if m.state.Rollback != nil {
		t.Error("executing a command must close the rollback session")
	}
	if m.state.Mode == model.ModeRollback {
		t.Errorf("mode = %s, want to have left the rollback view", m.state.Mode)
	}
}

func TestRollbackCommandBar_RendersOverRollbackLayout(t *testing.T) {
	m := buildRollbackGoldenModel(100, 24)
	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg(":"))
	m = teaModel.(*Model)

	out := stripANSI(m.renderRollbackLayout())
	if !strings.Contains(out, "Deployment history") {
		t.Fatal("rollback layout should still render under the command bar")
	}
	// list frame + detail frame + command bar = three rounded boxes
	if boxes := strings.Count(out, "╭"); boxes != 3 {
		t.Errorf("rendered %d boxes, want 3 (command bar missing):\n%s", boxes, out)
	}
}

// The command executor clears the list filter before dispatching; the
// selection must keep pointing at the same app or :rollback (and friends)
// target whatever happens to sit at that index unfiltered.
func TestRollbackCommand_TargetsSelectedAppUnderActiveFilter(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	nsB := "team-b"
	m.state.Apps = []model.App{
		{Name: "aardvark", Sync: "Synced", Health: "Healthy"},
		{Name: "zebra", AppNamespace: &nsB, Sync: "Synced", Health: "Healthy"},
	}
	m.state.Navigation.View = model.ViewApps
	m.state.UI.ActiveFilter = "zebra"
	m.state.Navigation.SelectedIdx = 0 // first (and only) filtered row = zebra

	m = runCommand(t, m, "rollback")

	if m.state.Rollback == nil {
		t.Fatal("expected :rollback to start a session")
	}
	if m.state.Rollback.AppName != "zebra" {
		t.Errorf("rollback target = %q, want the filtered selection zebra", m.state.Rollback.AppName)
	}
}

func TestRollbackList_ShowsScrollMarkersWhenHistoryOverflows(t *testing.T) {
	m := buildRollbackListModel(25)
	l := m.rollbackPaneLayout(m.renderBanner())

	top := stripANSI(m.renderRollbackList(l))
	if strings.Contains(top, "▲ more above") || !strings.Contains(top, "▼ more below") {
		t.Errorf("at the top, want only the ▼ marker:\n%s", top)
	}

	m.state.Rollback.SelectedIdx = 12
	middle := stripANSI(m.renderRollbackList(l))
	if !strings.Contains(middle, "▲ more above") || !strings.Contains(middle, "▼ more below") {
		t.Errorf("mid-list, want both markers:\n%s", middle)
	}

	m.state.Rollback.SelectedIdx = 24
	bottom := stripANSI(m.renderRollbackList(l))
	if !strings.Contains(bottom, "▲ more above") || strings.Contains(bottom, "▼ more below") {
		t.Errorf("at the bottom, want only the ▲ marker:\n%s", bottom)
	}
}

func TestRollbackKey_FromTreeView_TargetsTreeApp(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	ns := "team-b"
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "tree-app", AppNamespace: &ns}

	teaModel, _ := m.handleRollback()
	m = teaModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("expected R in tree view to start a rollback session")
	}
	if m.state.Rollback.AppName != "tree-app" ||
		m.state.Rollback.AppNamespace == nil || *m.state.Rollback.AppNamespace != ns {
		t.Errorf("target = (%s, %v), want the tree app (tree-app, team-b)",
			m.state.Rollback.AppName, m.state.Rollback.AppNamespace)
	}
	if m.state.Mode != model.ModeRollback {
		t.Errorf("mode = %s, want rollback", m.state.Mode)
	}
}

func TestRollbackCommand_FromTreeView_TargetsTreeApp(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	ns := "team-b"
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeApp = &model.TreeAppInfo{Name: "tree-app", AppNamespace: &ns}

	m = runCommand(t, m, "rollback")

	if m.state.Rollback == nil {
		t.Fatal("expected :rollback in tree view to start a session")
	}
	if m.state.Rollback.AppName != "tree-app" {
		t.Errorf("target = %q, want tree-app", m.state.Rollback.AppName)
	}
}

func TestRollbackConfirm_JKDoesNotChangeTargetRevision(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.SelectedIdx = 2
	m.state.Rollback.Mode = "confirm"

	teaModel, _ := m.handleKeyMsg(testKeyMsg("j"))
	m = teaModel.(*Model)

	if m.state.Rollback.SelectedIdx != 2 {
		t.Errorf("SelectedIdx = %d, want 2 — the confirm modal must pin its target", m.state.Rollback.SelectedIdx)
	}
}

func TestRollbackKey_FromAppsView_TargetsCursorApp(t *testing.T) {
	m := buildSyncTestModel(100, 30)
	m.state.Apps = []model.App{{Name: "app-a"}, {Name: "app-b"}}
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 1

	teaModel, _ := m.handleKeyMsg(testKeyMsg("R"))
	m = teaModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("expected R in apps view to start a rollback session")
	}
	if m.state.Rollback.AppName != "app-b" {
		t.Errorf("target = %q, want app-b", m.state.Rollback.AppName)
	}
}

func buildMultiSelectedAppsModel() *Model {
	m := buildSyncTestModel(100, 30)
	m.state.Apps = []model.App{{Name: "app-a"}, {Name: "app-b"}, {Name: "app-c"}}
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 2
	m.state.Selections.SelectedApps = model.AddToStringSet(m.state.Selections.SelectedApps, "app-a")
	m.state.Selections.SelectedApps = model.AddToStringSet(m.state.Selections.SelectedApps, "app-b")
	return m
}

// Rollback is single-app: with several apps deliberately selected there is
// no unambiguous target, so the view must refuse to open.
func TestRollbackKey_RefusesWithMultipleAppsSelected(t *testing.T) {
	m := buildMultiSelectedAppsModel()

	teaModel, _ := m.handleKeyMsg(testKeyMsg("R"))
	m = teaModel.(*Model)

	if m.state.Rollback != nil || m.state.Mode == model.ModeRollback {
		t.Errorf("R with multi-select must not open the rollback view (target %v)", m.state.Rollback)
	}
}

func TestRollbackCommand_RefusesWithMultipleAppsSelected(t *testing.T) {
	m := buildMultiSelectedAppsModel()

	m = runCommand(t, m, "rollback")

	if m.state.Rollback != nil || m.state.Mode == model.ModeRollback {
		t.Errorf(":rollback with multi-select must not open the rollback view (target %v)", m.state.Rollback)
	}
}

func TestRollbackListR_OpensConfirmLikeEnter(t *testing.T) {
	m := buildRollbackListModel(5)
	m.state.Rollback.SelectedIdx = 2

	teaModel, _ := m.handleRollbackModeKeys(testKeyMsg("R"))
	m = teaModel.(*Model)

	if m.state.Rollback.Mode != "confirm" || m.state.Rollback.ConfirmSelected != 1 {
		t.Errorf("state = (%s, confirmSelected=%d), want (confirm, 1=Cancel)",
			m.state.Rollback.Mode, m.state.Rollback.ConfirmSelected)
	}
}

func TestRollbackConfirmModal_WidthHugsContent(t *testing.T) {
	m := buildRollbackGoldenModel(200, 40)
	m.state.Rollback.Mode = "confirm"

	modal := m.renderRollbackConfirmModal()
	if w := lipgloss.Width(modal); w > 70 {
		t.Errorf("modal width = %d on a 200-col terminal, want content-sized (≤70)", w)
	}

	long := strings.Repeat("long subject ", 20)
	m.state.Rollback.Rows[1].Message = &long
	modal = m.renderRollbackConfirmModal()
	if w := lipgloss.Width(modal); w > 200-6 {
		t.Errorf("modal width = %d, must stay within the terminal", w)
	}
}

// A refusal the user can't see is a no-op: the multi-select hint must show
// up in the footer (status messages only go to the log file).
func TestRollbackMultiSelectRefusal_ShowsFooterHint(t *testing.T) {
	m := buildMultiSelectedAppsModel()

	teaModel, _ := m.handleKeyMsg(testKeyMsg("R"))
	m = teaModel.(*Model)

	footer := stripANSI(m.renderStatusLine())
	if !strings.Contains(footer, "one app at a time") {
		t.Errorf("footer = %q, want the one-app-at-a-time hint", footer)
	}
}

func TestRollbackKey_OnHoveredChildApplication_TargetsThatApp(t *testing.T) {
	m := buildEventsPaneTestModel()
	childNs := "argocd"
	m.treeView.UpsertAppTree("test-app", &api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "child-app-uid", Group: "argoproj.io", Version: "v1alpha1", Kind: "Application", Name: "child-app", Namespace: &childNs},
	}})
	m.treeView.SetSelectedIndex(1) // hover the child Application row

	teaModel, _ := m.handleKeyMsg(testKeyMsg("R"))
	m = teaModel.(*Model)

	if m.state.Rollback == nil {
		t.Fatal("expected R on a child Application to start a rollback session")
	}
	if m.state.Rollback.AppName != "child-app" ||
		m.state.Rollback.AppNamespace == nil || *m.state.Rollback.AppNamespace != childNs {
		t.Errorf("target = (%s, %v), want (child-app, argocd)",
			m.state.Rollback.AppName, m.state.Rollback.AppNamespace)
	}
}

// Rolling back from inside an app's tree and watching must not push a
// duplicate navigation entry — esc would "return" to the same tree and the
// user needs two presses to actually leave.
func TestRollbackWatch_FromSameAppsTree_DoesNotStackDuplicateNav(t *testing.T) {
	m := buildEventsPaneTestModel() // tree view of test-app
	m.state.SaveNavigationState()   // the drill-in from apps already saved one entry
	m.state.Navigation.View = model.ViewTree
	saved := len(m.state.SavedNavigation)

	// The rollback target carries the tree app's identity, namespace included
	ns := "test-namespace"
	teaModel, _ := m.Update(model.RollbackExecutedMsg{AppName: "test-app", AppNamespace: &ns, Success: true, Watch: true})
	m = teaModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("view = %s, want the watch tree", m.state.Navigation.View)
	}
	if len(m.state.SavedNavigation) != saved {
		t.Errorf("saved nav entries = %d, want still %d (no duplicate push)", len(m.state.SavedNavigation), saved)
	}
}

func TestRollbackWatch_FromAppsView_SavesReturnPoint(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Navigation.View = model.ViewApps
	m.state.UI.TreeApp = nil
	saved := len(m.state.SavedNavigation)

	teaModel, _ := m.Update(model.RollbackExecutedMsg{AppName: "test-app", Success: true, Watch: true})
	m = teaModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("view = %s, want the watch tree", m.state.Navigation.View)
	}
	if len(m.state.SavedNavigation) != saved+1 {
		t.Errorf("saved nav entries = %d, want %d (apps view saved for esc)", len(m.state.SavedNavigation), saved+1)
	}
}

// esc from the post-rollback watch tree reopens the app's deployment
// history (over the apps view), not the bare apps list.
func TestRollbackWatchEsc_ReopensDeploymentHistory(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Navigation.View = model.ViewApps
	m.state.UI.TreeApp = nil
	ns := "team-b"

	teaModel, _ := m.Update(model.RollbackExecutedMsg{AppName: "test-app", AppNamespace: &ns, Success: true, Watch: true})
	m = teaModel.(*Model)

	teaModel, cmd := m.handleKeyMsg(testKeyMsg("esc"))
	m = teaModel.(*Model)

	if m.state.Mode != model.ModeRollback || m.state.Rollback == nil {
		t.Fatalf("mode = %s, want the rollback view reopened", m.state.Mode)
	}
	if m.state.Rollback.AppName != "test-app" ||
		m.state.Rollback.AppNamespace == nil || *m.state.Rollback.AppNamespace != ns {
		t.Errorf("target = (%s, %v), want (test-app, team-b)", m.state.Rollback.AppName, m.state.Rollback.AppNamespace)
	}
	if m.state.Navigation.View != model.ViewApps {
		t.Errorf("view beneath = %s, want apps so the next esc lands there", m.state.Navigation.View)
	}
	if cmd == nil {
		t.Error("expected the history to start loading")
	}
}

// The sync flow uses the same watch machinery but esc there keeps its
// normal meaning — straight back to where the sync started.
func TestSyncWatchEsc_DoesNotReopenRollback(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Navigation.View = model.ViewApps
	m.state.UI.TreeApp = nil
	m.state.Modals.ConfirmSyncWatch = true
	target := "test-app"
	m.state.Modals.ConfirmTarget = &target

	teaModel, _ := m.Update(model.SyncCompletedMsg{AppName: "test-app", Success: true, SwitchEpoch: m.switchEpoch})
	m = teaModel.(*Model)
	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("setup: expected the sync watch tree, got %s", m.state.Navigation.View)
	}

	teaModel, _ = m.handleKeyMsg(testKeyMsg("esc"))
	m = teaModel.(*Model)

	if m.state.Mode == model.ModeRollback {
		t.Error("esc after a sync watch must not open the rollback view")
	}
	if m.state.Navigation.View != model.ViewApps {
		t.Errorf("view = %s, want apps", m.state.Navigation.View)
	}
}

// Confirm must not open over an empty history — the modal would
// dereference a selected row that does not exist.
func TestRollbackConfirm_NotOpenedForEmptyHistory(t *testing.T) {
	m := buildRollbackListModel(0)

	for _, key := range []string{"enter", "R"} {
		teaModel, _ := m.handleRollbackModeKeys(testKeyMsg(key))
		m = teaModel.(*Model)
		if m.state.Rollback.Mode != "list" {
			t.Errorf("%s on empty history switched to %q, want to stay in list", key, m.state.Rollback.Mode)
		}
	}
}

// A metadata reply from a previous session (other app, other epoch, or a
// reshuffled history) must not be written into the current rows.
func TestRollbackMetadataLoaded_StaleRepliesIgnored(t *testing.T) {
	m := buildRollbackListModel(3)
	meta := model.RevisionMetadata{Author: "stale", Message: "stale"}

	cases := []struct {
		name string
		msg  model.RollbackMetadataLoadedMsg
	}{
		{"wrong epoch", model.RollbackMetadataLoadedMsg{RowIndex: 0, Metadata: meta, AppName: "demo-app", Revision: "reva", SwitchEpoch: 41}},
		{"wrong app", model.RollbackMetadataLoadedMsg{RowIndex: 0, Metadata: meta, AppName: "other-app", Revision: "reva", SwitchEpoch: 42}},
		{"reshuffled history", model.RollbackMetadataLoadedMsg{RowIndex: 0, Metadata: meta, AppName: "demo-app", Revision: "oldrev", SwitchEpoch: 42}},
	}
	for _, tc := range cases {
		teaModel, _ := m.Update(tc.msg)
		m = teaModel.(*Model)
		if m.state.Rollback.Rows[0].Author != nil {
			t.Errorf("%s: stale metadata was applied to the current session", tc.name)
		}
	}

	teaModel, _ := m.Update(model.RollbackMetadataLoadedMsg{RowIndex: 0, Metadata: meta, AppName: "demo-app", Revision: "reva", SwitchEpoch: 42})
	m = teaModel.(*Model)
	if m.state.Rollback.Rows[0].Author == nil {
		t.Error("a matching reply must still be applied")
	}
}
