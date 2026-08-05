package main

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
)

var paneNow = time.Date(2026, 8, 4, 12, 2, 0, 0, time.UTC)

// withRealPalette gives the package palette real colors for tests that
// assert styling — the zero palette renders everything unstyled, which made
// style assertions pass vacuously.
func withRealPalette(t *testing.T) {
	t.Helper()
	old := currentPalette
	currentPalette = theme.Default()
	t.Cleanup(func() { currentPalette = old })
}

// Reasons sit flush against the frame padding — warnings are distinguished
// by color alone, not a marker column.
func TestRenderEventCards_ReasonsAreFlushLeft(t *testing.T) {
	events := []model.ResourceEvent{
		{
			Type:     "Warning",
			Reason:   "BackOff",
			Message:  "Back-off restarting failed container web",
			Count:    412,
			LastSeen: time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC),
		},
		{
			Type:     "Normal",
			Reason:   "Scheduled",
			Message:  "ok",
			Count:    1,
			LastSeen: paneNow,
		},
	}

	lines := renderEventCards(events, 46, paneNow, "")

	head := stripANSI(lines[0])
	if !strings.HasPrefix(head, "BackOff") {
		t.Errorf("expected the warning reason flush left, got %q", head)
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
	if normalHead := stripANSI(lines[3]); !strings.HasPrefix(normalHead, "Scheduled") {
		t.Errorf("expected the normal reason flush left too, got %q", normalHead)
	}
}

// The session's own username reads as "you" in event messages.
func TestRenderEventCards_CurrentUserBecomesYou(t *testing.T) {
	events := []model.ResourceEvent{{
		Reason:   "OperationStarted",
		Message:  "admin initiated sync to HEAD (4caae2c0)",
		Count:    1,
		LastSeen: paneNow,
	}}

	joined := stripANSI(strings.Join(renderEventCards(events, 46, paneNow, "admin"), "\n"))

	if !strings.Contains(joined, "you initiated sync") {
		t.Errorf("expected the current user replaced with 'you': %s", joined)
	}
	if strings.Contains(joined, "admin initiated") {
		t.Errorf("expected the username gone from the message: %s", joined)
	}
}

// Shas get their identity color so the same sha is spottable across cards.
func TestRenderEventCards_ShasGetIdentityColors(t *testing.T) {
	events := []model.ResourceEvent{{
		Reason:   "OperationStarted",
		Message:  "sync to 4caae2c0 done",
		Count:    1,
		LastSeen: paneNow,
	}}

	withRealPalette(t)
	lines := renderEventCards(events, 46, paneNow, "")

	shaStyled := lipgloss.NewStyle().Foreground(currentPalette.ShaColor("4caae2c0")).Render("4caae2c0")
	if !strings.Contains(strings.Join(lines, "\n"), shaStyled) {
		t.Errorf("expected the sha rendered in its identity color, got: %q", lines)
	}
}

// Pod-template hashes (8-10 hex) deserve identity colors too, not only
// shortened git shas.
func TestRenderEventCards_TemplateHashesGetIdentityColors(t *testing.T) {
	events := []model.ResourceEvent{{
		Reason:   "SwitchService",
		Message:  "Switched selector from '' to '595d8d78b6'",
		Count:    2,
		LastSeen: paneNow,
	}}

	withRealPalette(t)
	lines := renderEventCards(events, 46, paneNow, "")

	hashStyled := lipgloss.NewStyle().Foreground(currentPalette.ShaColor("595d8d78b6")).Render("595d8d78b6")
	if !strings.Contains(strings.Join(lines, "\n"), hashStyled) {
		t.Errorf("expected the template hash in its identity color, got: %q", lines)
	}
}

// The substituted "you" stands out from the dim message text.
func TestRenderEventCards_YouIsHighlighted(t *testing.T) {
	events := []model.ResourceEvent{{
		Reason:   "OperationStarted",
		Message:  "admin initiated sync to HEAD",
		Count:    1,
		LastSeen: paneNow,
	}}

	withRealPalette(t)
	lines := renderEventCards(events, 46, paneNow, "admin")

	youStyled := lipgloss.NewStyle().Foreground(currentPalette.Text).Bold(true).Render("you")
	if !strings.Contains(strings.Join(lines, "\n"), youStyled) {
		t.Errorf("expected 'you' highlighted, got: %q", lines)
	}
}

func TestRenderSyncStatusBody_CurrentUserBecomesYou(t *testing.T) {
	details := &model.SyncStatusDetails{Phase: "Succeeded", StartedAt: paneNow, InitiatedBy: "admin"}

	joined := stripANSI(strings.Join(renderSyncStatusBody(details, 46, paneNow, "admin"), "\n"))

	if !strings.Contains(joined, "Initiated by  you") {
		t.Errorf("expected 'Initiated by  you': %s", joined)
	}
}

func TestRenderEventCards_BlankLineBetweenCards(t *testing.T) {
	events := []model.ResourceEvent{
		{Reason: "BackOff", Message: "a", Count: 1, LastSeen: paneNow},
		{Reason: "Pulled", Message: "b", Count: 1, LastSeen: paneNow},
	}

	lines := renderEventCards(events, 46, paneNow, "")

	joined := stripANSI(strings.Join(lines, "\n"))
	if !strings.Contains(joined, "a\n\n") {
		t.Errorf("expected a blank separator between cards:\n%s", joined)
	}
	if strings.HasSuffix(joined, "\n") || lines[len(lines)-1] == "" {
		t.Errorf("expected no trailing blank line:\n%q", joined)
	}
}

func TestRenderResourceStatusBody_FieldsAndLastSyncResult(t *testing.T) {
	status := &model.ResourceStatusSummary{
		Health:        "Degraded",
		Sync:          "OutOfSync",
		HealthMessage: "Deployment does not have minimum availability",
		CreatedAt:     paneNow.Add(-72 * time.Hour),
	}
	details := &model.SyncStatusDetails{Resources: []model.SyncResourceResult{
		{Kind: "Service", Namespace: "demo", Name: "web", Status: "Synced", Message: "unchanged"},
		{Kind: "Deployment", Namespace: "demo", Name: "web", Status: "SyncFailed", Message: "apply failed: image required"},
	}}
	target := model.EventsResource{Kind: "Deployment", Namespace: "demo", Name: "web", UID: "d1"}

	lines := renderResourceStatusBody(status, details, target, 46, paneNow)
	joined := stripANSI(strings.Join(lines, "\n"))

	for _, want := range []string{
		"Health        Degraded",
		"Sync          OutOfSync",
		"Age           3d",
		"Deployment does not have minimum",
		"Last sync     SyncFailed",
		"  apply failed: image required",
	} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected %q in the resource status block:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "unchanged") {
		t.Errorf("only the targeted resource's RESULT row belongs in the block:\n%s", joined)
	}
}

func TestRenderResourceStatusBody_SkipsAbsentFields(t *testing.T) {
	status := &model.ResourceStatusSummary{Sync: "Synced"} // no health, no age, no message

	lines := renderResourceStatusBody(status, nil, model.EventsResource{Kind: "ConfigMap", Name: "cfg"}, 46, paneNow)
	joined := stripANSI(strings.Join(lines, "\n"))

	for _, absent := range []string{"Health", "Age", "Message", "Last sync"} {
		if strings.Contains(joined, absent) {
			t.Errorf("expected %q to be skipped when unknown:\n%s", absent, joined)
		}
	}
	if !strings.Contains(joined, "Sync          Synced") {
		t.Errorf("expected the sync field:\n%s", joined)
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

	lines := renderSyncStatusBody(details, 46, paneNow, "")
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

	lines := renderSyncStatusBody(details, 46, paneNow, "")

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

	lines := renderSyncStatusBody(details, 46, paneNow, "")

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

	joined := stripANSI(strings.Join(renderSyncStatusBody(details, 46, paneNow, ""), "\n"))

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
	// renderTreePanel renders 2 narrower than given: 50 → the 48-wide box
	// of the design mock, flush against the 50-wide pane
	if l.treeBoxWidth != 50 {
		t.Errorf("expected tree box width 50 at 100 cols, got %d", l.treeBoxWidth)
	}
	if l.paneBodyWidth != 46 {
		t.Errorf("expected pane body width 46, got %d", l.paneBodyWidth)
	}
}

// Tree width has diminishing returns: once it has ~80 rendered columns,
// additional terminal width goes to the pane (up to its own cap).
func TestPaneLayout_WideTerminal_GivesGrowthToThePane(t *testing.T) {
	m := buildEventsPaneTestModel()

	m.state.Terminal.Cols = 180
	l := m.paneLayout(16)
	if l.paneBoxWidth != 98 {
		t.Errorf("at 180 cols expected pane box 98 (tree keeps 82), got %d", l.paneBoxWidth)
	}
	if l.treeBoxWidth != 82 {
		t.Errorf("at 180 cols expected tree box 82, got %d", l.treeBoxWidth)
	}

	// Beyond the pane's cap the tree resumes growing
	m.state.Terminal.Cols = 240
	l = m.paneLayout(16)
	if l.paneBoxWidth != 100 {
		t.Errorf("at 240 cols expected the pane capped at 100, got %d", l.paneBoxWidth)
	}
	if l.treeBoxWidth != 140 {
		t.Errorf("at 240 cols expected the tree to take the rest (140), got %d", l.treeBoxWidth)
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

// The side-by-side row must end flush with the other full-width elements
// (status line, command bar, the pane-closed tree box) — no right-edge gap.
func TestSideBySideLayout_FillsTheFullContentWidth(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel()) // 120 cols → side layout
	m.state.Events.Loading = false
	m.state.Events.Items = []model.ResourceEvent{{Reason: "BackOff", Message: "x", Count: 1, LastSeen: paneNow}}

	fullWidth := lipgloss.Width(m.renderTreePanel(10, m.state.Terminal.Cols)) // pane-closed box width

	l := m.paneLayout(10)
	row := lipgloss.JoinHorizontal(lipgloss.Top,
		m.renderTreePanel(l.treeBodyRows, l.treeBoxWidth), m.renderSidePane(l))

	if got := lipgloss.Width(row); got != fullWidth {
		t.Errorf("side-by-side row is %d cells wide, want %d (flush with the full-width box)", got, fullWidth)
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
	frame := paneFrame{Title: "Events · Pod web-1", Width: 30, BodyRows: 3}

	out := stripANSI(renderPaneFrame(frame, []string{"hello"}))
	lines := strings.Split(out, "\n")

	if len(lines) != 5 {
		t.Fatalf("expected top border + 3 body rows + bottom border, got %d lines:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[0], "╭─ Events · Pod web-1 ") || !strings.HasSuffix(lines[0], "╮") {
		t.Errorf("expected the title embedded in the top border, got %q", lines[0])
	}
	// The first body row is always blank: breathing room under the title
	if lines[1] != "│"+strings.Repeat(" ", 28)+"│" {
		t.Errorf("expected the padding row under the title, got %q", lines[1])
	}
	if lines[2] != "│ hello"+strings.Repeat(" ", 30-2-2-len("hello"))+" │" {
		t.Errorf("unexpected first content row: %q", lines[2])
	}
	if lines[3] != "│"+strings.Repeat(" ", 28)+"│" {
		t.Errorf("expected a blank padded row, got %q", lines[3])
	}
	for _, line := range lines {
		if w := len([]rune(line)); w != 30 {
			t.Errorf("every frame line must be exactly 30 cells, got %d in %q", w, line)
		}
	}
}

// Content may use every row below the title padding — a pane that fits its
// content exactly shows all of it, with no phantom scroll marker.
func TestRenderSidePane_ContentFillingTheBodyExactly_ShowsItAll(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.state.Events.Loading = false
	m.state.Events.DetailsLoading = false
	m.state.Events.ResourceStatus = nil // no status block: the body is EVENTS + cards
	// heading + 3 two-line cards + 2 separators = exactly 9 content lines
	var events []model.ResourceEvent
	for _, r := range []string{"A", "B", "C"} {
		events = append(events, model.ResourceEvent{Reason: r, Message: "m", Count: 1, LastSeen: paneNow})
	}
	m.state.Events.Items = events

	l := m.paneLayout(11) // paneBodyRows 10: padding row + all 9 content lines
	out := stripANSI(m.renderSidePane(l))

	if !strings.Contains(out, "C") {
		t.Errorf("expected the last card visible:\n%s", out)
	}
	if strings.Contains(out, "▼ more below") {
		t.Errorf("nothing is clipped, so no marker belongs on the frame:\n%s", out)
	}
}

// The padding row consumes one physical row, so the scrollable capacity is
// BodyRows-1 — the viewport math and markers must account for it.
func TestRenderSidePane_PaddingRowReducesScrollCapacity(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())
	m.state.Events.Loading = false
	// 3 two-line cards + 2 separators = exactly 8 content lines
	var events []model.ResourceEvent
	for _, r := range []string{"A", "B", "C"} {
		events = append(events, model.ResourceEvent{Reason: r, Message: "m", Count: 1, LastSeen: paneNow})
	}
	m.state.Events.Items = events

	l := m.paneLayout(8) // paneBodyRows 7: the padding row leaves capacity 6
	// for 8 content lines → clipped
	out := stripANSI(m.renderSidePane(l))

	if !strings.Contains(out, "▼ more below") {
		t.Errorf("expected the below marker: the padding row costs one row of capacity:\n%s", out)
	}
	lines := strings.Split(out, "\n")
	if got := len(lines); got != l.paneBodyRows+2 {
		t.Errorf("frame height must stay %d rows, got %d", l.paneBodyRows+2, got)
	}
	if content := strings.Trim(stripANSI(lines[1]), "│ "); content != "" {
		t.Errorf("expected the padding row directly under the title, got %q", lines[1])
	}
}

func TestRenderPaneFrame_StatusAnchorsInTopBorderRight(t *testing.T) {
	frame := paneFrame{Title: "Events", Width: 40, BodyRows: 1, Status: "⟳ 10s"}

	out := stripANSI(renderPaneFrame(frame, []string{"x"}))
	lines := strings.Split(out, "\n")

	if top := lines[0]; !strings.HasSuffix(top, " ⟳ 10s ─╮") {
		t.Errorf("expected the status anchored at the top border's right edge, got %q", top)
	}
	if bottom := lines[len(lines)-1]; strings.Contains(bottom, "⟳") {
		t.Errorf("expected no status in the bottom border, got %q", bottom)
	}

	// With the ▲ marker up too, the marker keeps the corner
	frame.MoreAbove = true
	top := strings.Split(stripANSI(renderPaneFrame(frame, []string{"x"})), "\n")[0]
	if !strings.HasSuffix(top, " ⟳ 10s ─ ▲ more above ─╮") {
		t.Errorf("expected the status to coexist with the scroll marker, got %q", top)
	}
	if w := len([]rune(top)); w != 40 {
		t.Errorf("expected the top border to span exactly 40 cells, got %d", w)
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

// The layout must never exceed the terminal height — an extra line scrolls
// the banner's top row off the screen.
func TestMainLayout_WithPaneOpen_FitsTheTerminalHeight(t *testing.T) {
	for _, size := range []struct{ cols, rows int }{
		{100, 24}, {120, 30}, {140, 40}, {80, 24}, {101, 24}, {200, 50}, {170, 45},
	} {
		m := buildEventsPaneTestModel()
		// The full-size banner (ASCII logo) renders with a server configured
		m.state.Server = &model.Server{BaseURL: "https://argo.example.com"}
		m.state.APIVersion = "v2.10.3"
		m.state.Terminal.Cols, m.state.Terminal.Rows = size.cols, size.rows

		check := func(state string) {
			lines := strings.Split(m.renderMainLayout(), "\n")
			if len(lines) > size.rows {
				t.Errorf("%dx%d pane %s: layout is %d lines, terminal has %d",
					size.cols, size.rows, state, len(lines), size.rows)
			}
			for i, line := range lines {
				if w := lipgloss.Width(line); w > size.cols {
					t.Errorf("%dx%d pane %s: line %d is %d cells wide (terminal %d) — it wraps and clips the top: %q",
						size.cols, size.rows, state, i, w, size.cols, stripANSI(line))
				}
			}
		}
		check("closed")
		m = openEventsPane(t, m)
		check("open")
	}
}
