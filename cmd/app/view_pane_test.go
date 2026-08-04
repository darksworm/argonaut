package main

import (
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

var paneNow = time.Date(2026, 8, 4, 12, 2, 0, 0, time.UTC)

func TestRenderEventCards_WarningCardLayout(t *testing.T) {
	events := []model.ResourceEvent{{
		Type:     "Warning",
		Reason:   "BackOff",
		Message:  "Back-off restarting failed container web",
		Count:    412,
		LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
	}}

	lines := renderEventCards(events, 46, paneNow)

	head := stripANSI(lines[0])
	if !strings.HasPrefix(head, "! BackOff") {
		t.Errorf("expected the warning marker before the reason, got %q", head)
	}
	if !strings.HasSuffix(head, "x412 · 2m ago") {
		t.Errorf("expected the meta right-aligned at the line end, got %q", head)
	}
	if w := len([]rune(head)); w != 46 {
		t.Errorf("expected the header padded to the full width 46, got %d", w)
	}
	if got := stripANSI(lines[1]); !strings.HasPrefix(got, "  Back-off restarting") {
		t.Errorf("expected the message indented under the reason, got %q", got)
	}
}

func TestRenderEventCards_NormalEventAlignsWithWarnings(t *testing.T) {
	events := []model.ResourceEvent{{
		Type:     "Normal",
		Reason:   "Scheduled",
		Message:  "ok",
		Count:    1,
		LastSeen: paneNow,
	}}

	lines := renderEventCards(events, 46, paneNow)

	if head := stripANSI(lines[0]); !strings.HasPrefix(head, "  Scheduled") {
		t.Errorf("expected normal events indented to align with warning reasons, got %q", head)
	}
}

func TestRenderEventCards_BlankLineBetweenCards(t *testing.T) {
	events := []model.ResourceEvent{
		{Reason: "BackOff", Message: "a", Count: 1, LastSeen: paneNow},
		{Reason: "Pulled", Message: "b", Count: 1, LastSeen: paneNow},
	}

	lines := renderEventCards(events, 46, paneNow)

	joined := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "a\n\n") {
		t.Errorf("expected a blank separator between cards:\n%s", joined)
	}
	if strings.HasSuffix(joined, "\n") || lines[len(lines)-1] == "" {
		t.Errorf("expected no trailing blank line:\n%q", joined)
	}
}

func TestRenderSyncStatusBody_FieldsAndResults(t *testing.T) {
	details := &model.SyncStatusDetails{
		Phase:       "Failed",
		Message:     "one or more objects failed to apply",
		StartedAt:   time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 8, 4, 12, 0, 6, 0, time.UTC),
		Revision:    "a1b2c3d4e5f6789",
		InitiatedBy: "alice",
		Resources: []model.SyncResourceResult{
			{Kind: "Service", Namespace: "demo", Name: "web", Status: "Synced", Message: "service/web unchanged"},
			{Kind: "Deployment", Namespace: "demo", Name: "web", Status: "SyncFailed", Message: "invalid"},
		},
	}

	lines := renderSyncStatusBody(details, 46, paneNow)
	joined := stripANSI(strings.Join(lines, "\n"))

	for _, want := range []string{
		"Operation     Sync",
		"Phase         Failed",
		"Started       2 minutes ago",
		"Duration      6s",
		"Revision      a1b2c3d",
		"Initiated by  alice",
		"RESULT",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q in the sync body:\n%s", want, joined)
		}
	}
	if !strings.Contains(joined, "✔ Service") {
		t.Errorf("expected a success glyph on the synced resource:\n%s", joined)
	}
	if !strings.Contains(joined, "✖ Deployment") {
		t.Errorf("expected a danger glyph on the failed resource:\n%s", joined)
	}
	if !strings.Contains(joined, "Synced") || !strings.Contains(joined, "SyncFailed") {
		t.Errorf("expected per-resource statuses:\n%s", joined)
	}
	if !strings.Contains(joined, "  service/web unchanged") {
		t.Errorf("expected the resource message indented below its row:\n%s", joined)
	}
}

func TestRenderSyncStatusBody_LongResourceName_TruncatesNameNotStatus(t *testing.T) {
	details := &model.SyncStatusDetails{
		Phase:     "Failed",
		StartedAt: paneNow,
		Resources: []model.SyncResourceResult{
			{Kind: "Deployment", Namespace: "default", Name: "nginx-deployment-with-a-very-long-name", Status: "SyncFailed"},
		},
	}

	lines := renderSyncStatusBody(details, 46, paneNow)

	var row string
	for _, l := range lines {
		if strings.Contains(stripANSI(l), "Deployment") {
			row = stripANSI(l)
			break
		}
	}
	if row == "" {
		t.Fatal("result row not found")
	}
	if !strings.HasSuffix(row, "SyncFailed") {
		t.Errorf("the status must survive at the row's end, got %q", row)
	}
	if w := len([]rune(row)); w > 46 {
		t.Errorf("the row must fit the pane width 46, got %d: %q", w, row)
	}
	if !strings.Contains(row, "…") {
		t.Errorf("expected the name truncated with an ellipsis, got %q", row)
	}
}

func TestRenderSyncStatusBody_LongKind_TruncatesKindNotStatus(t *testing.T) {
	details := &model.SyncStatusDetails{
		Phase:     "Failed",
		StartedAt: paneNow,
		Resources: []model.SyncResourceResult{
			// A CRD-length kind: longer than the row can absorb by
			// truncating the name alone
			{Kind: "SomeVeryLongCustomResourceDefinitionKind", Namespace: "default", Name: "web", Status: "SyncFailed"},
		},
	}

	lines := renderSyncStatusBody(details, 46, paneNow)

	var row string
	for _, l := range lines {
		if strings.Contains(stripANSI(l), "SomeVeryLong") {
			row = stripANSI(l)
			break
		}
	}
	if row == "" {
		t.Fatal("result row not found")
	}
	if !strings.HasSuffix(row, "SyncFailed") {
		t.Errorf("the status must survive at the row's end, got %q", row)
	}
	if w := len([]rune(row)); w > 46 {
		t.Errorf("the row must fit the pane width 46, got %d: %q", w, row)
	}
}

func TestRenderSyncStatusBody_RunningOperationDurationTicksFromNow(t *testing.T) {
	details := &model.SyncStatusDetails{
		Phase:     "Running",
		StartedAt: paneNow.Add(-42 * time.Second),
	}

	joined := stripANSI(strings.Join(renderSyncStatusBody(details, 46, paneNow), "\n"))

	if !strings.Contains(joined, "Duration      42s") {
		t.Errorf("expected the running duration measured against now:\n%s", joined)
	}
}

func TestPaneLayout_WideTerminal_SplitsSideBySide(t *testing.T) {
	m := buildEventsPaneTestModel() // 120 cols
	m.state.Terminal.Cols = 100

	l := m.paneLayout(16)

	if !l.side {
		t.Fatal("expected a side-by-side layout at 100 cols")
	}
	if l.paneBoxWidth != 50 {
		t.Errorf("expected pane box width 50, got %d", l.paneBoxWidth)
	}
	if l.treeBoxWidth != 48 {
		t.Errorf("expected tree box width 48 at 100 cols, got %d", l.treeBoxWidth)
	}
	if l.paneBodyWidth != 46 {
		t.Errorf("expected pane body width 46, got %d", l.paneBodyWidth)
	}
}

func TestPaneLayout_NarrowTerminal_FallsBackToBottomPane(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Terminal.Cols = 99

	l := m.paneLayout(18)

	if l.side {
		t.Fatal("expected a bottom pane below 100 cols")
	}
	if l.paneBodyRows != 12 {
		t.Errorf("expected pane body rows clamp(18-6,3,12)=12, got %d", l.paneBodyRows)
	}
	// Total height must not change: tree body + pane body + pane borders = budget
	if got := l.treeBodyRows + l.paneBodyRows + 2; got != 18 {
		t.Errorf("expected the stacked layout to spend exactly the 18-row budget, got %d", got)
	}
}

func TestPaneLayout_BottomPane_ClampsToMinimumRows(t *testing.T) {
	m := buildEventsPaneTestModel()
	m.state.Terminal.Cols = 80

	l := m.paneLayout(7)

	if l.paneBodyRows != 3 {
		t.Errorf("expected the pane to clamp at 3 body rows, got %d", l.paneBodyRows)
	}
}

func TestPaneLayout_TinyBudgets_NeverGoNegative(t *testing.T) {
	m := buildEventsPaneTestModel()
	for _, cols := range []int{80, 120} {
		m.state.Terminal.Cols = cols
		for budget := 0; budget <= 4; budget++ {
			l := m.paneLayout(budget)
			if l.paneBodyRows < 0 || l.treeBodyRows < 0 {
				t.Errorf("at %d cols budget %d: paneBodyRows=%d treeBodyRows=%d must not be negative",
					cols, budget, l.paneBodyRows, l.treeBodyRows)
			}
		}
	}
}

func TestRenderSidePane_ZeroRowBudget_DoesNotPanic(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel()) // 120 cols → side layout
	m.state.Events.Loading = false
	m.state.Events.Items = []model.ResourceEvent{{Reason: "BackOff", Message: "x", Count: 1, LastSeen: paneNow}}

	_ = m.renderSidePane(m.paneLayout(0))
}

func TestRenderPaneFrame_MoreAboveMarker_LeavesRoomForTitleByCells(t *testing.T) {
	// 29-cell title in a 50-cell frame: with the 15-cell marker it fits
	// exactly; measuring the marker in bytes truncated it wrongly.
	title := "Events · Pod web-6f7d9b-x4k2m"
	frame := paneFrame{Title: title, Width: 50, BodyRows: 1, MoreAbove: true}

	top := strings.Split(stripANSI(renderPaneFrame(frame, []string{"x"})), "\n")[0]

	if !strings.Contains(top, title) {
		t.Errorf("expected the full title to remain visible, got %q", top)
	}
	if w := len([]rune(top)); w != 50 {
		t.Errorf("expected the top border to span exactly 50 cells, got %d", w)
	}
}

func TestRenderPaneFrame_TitleAndBody(t *testing.T) {
	frame := paneFrame{Title: "Events · Pod web-1", Width: 30, BodyRows: 2}

	out := stripANSI(renderPaneFrame(frame, []string{"hello"}))
	lines := strings.Split(out, "\n")

	if len(lines) != 4 {
		t.Fatalf("expected top border + 2 body rows + bottom border, got %d lines:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "╭─ Events · Pod web-1 ") || !strings.HasSuffix(lines[0], "╮") {
		t.Errorf("expected the title embedded in the top border, got %q", lines[0])
	}
	if lines[1] != "│ hello"+strings.Repeat(" ", 30-2-2-len("hello"))+" │" {
		t.Errorf("unexpected body row: %q", lines[1])
	}
	if lines[2] != "│"+strings.Repeat(" ", 28)+"│" {
		t.Errorf("expected a blank padded row, got %q", lines[2])
	}
	for _, line := range lines {
		if w := len([]rune(line)); w != 30 {
			t.Errorf("every frame line must be exactly 30 cells, got %d in %q", w, line)
		}
	}
}

func TestRenderPaneFrame_ScrollMarkersAnchorInBorders(t *testing.T) {
	frame := paneFrame{Title: "Events", Width: 40, BodyRows: 1, MoreAbove: true, MoreBelow: true}

	out := stripANSI(renderPaneFrame(frame, []string{"x"}))
	lines := strings.Split(out, "\n")

	if !strings.Contains(lines[0], "▲ more above ─╮") {
		t.Errorf("expected the top border to carry the ▲ marker at its right edge, got %q", lines[0])
	}
	if !strings.Contains(lines[len(lines)-1], "▼ more below ─╯") {
		t.Errorf("expected the bottom border to carry the ▼ marker at its right edge, got %q", lines[len(lines)-1])
	}

	// Markers only when clipped in that direction
	plain := stripANSI(renderPaneFrame(paneFrame{Title: "Events", Width: 40, BodyRows: 1}, []string{"x"}))
	if strings.Contains(plain, "▲") || strings.Contains(plain, "▼") {
		t.Errorf("expected no markers when nothing is clipped:\n%s", plain)
	}
}
