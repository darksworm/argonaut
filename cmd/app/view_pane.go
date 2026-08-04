package main

// Geometry and framing for the tree view's side panes (events, sync status).
// All pane measurements live here so the layout math cannot drift between
// the renderers and the scroll/viewport calculations.

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/humantime"
	"github.com/darksworm/argonaut/pkg/model"
)

// paneSideMinCols is the narrowest terminal that fits the tree and the pane
// side by side; below it the pane drops to the bottom, spending height
// instead of width.
const paneSideMinCols = 100

// The pane's outer width in side-by-side mode scales with the terminal:
// tree width has diminishing returns past ~80 rendered columns, so growth
// beyond that goes to the pane until it reaches its own useful maximum.
const (
	paneSideMinBoxWidth   = 50 // matches the design mock at 100 cols
	paneSideMaxBoxWidth   = 100
	treePreferredBoxWidth = 82 // tree renders 2 narrower: ~80 columns
)

// paneLayout describes how the tree box and the pane box share the screen.
type paneLayout struct {
	side          bool // side-by-side (wide) vs stacked bottom pane (narrow)
	paneBoxWidth  int  // pane outer width, borders included
	paneBodyWidth int  // pane content columns inside the frame
	treeBoxWidth  int  // tree box outer width
	paneBodyRows  int  // pane content rows inside the frame
	treeBodyRows  int  // tree content rows
}

// paneContentRows is how many body lines fit below the frame's fixed top
// padding row (see renderPaneFrame).
func (l paneLayout) paneContentRows() int {
	return max(0, l.paneBodyRows-1)
}

// paneLayout computes the split for the given row budget (the rows the tree
// body alone would get with no pane open).
func (m *Model) paneLayout(availableRows int) paneLayout {
	cols := m.state.Terminal.Cols
	if cols >= paneSideMinCols {
		// renderTreePanel renders 2 cells narrower than the width it is
		// given (the full-width call passes cols and renders cols-2), so
		// the tree box gets cols-pane for a rendered row of exactly
		// cols-2 — flush with the status line and command bar.
		paneBox := min(max(cols-treePreferredBoxWidth, paneSideMinBoxWidth), paneSideMaxBoxWidth)
		return paneLayout{
			side:          true,
			paneBoxWidth:  paneBox,
			paneBodyWidth: paneBox - 4, // borders + one space padding each side
			treeBoxWidth:  cols - paneBox,
			paneBodyRows:  max(0, availableRows-1), // frame total == tree box total height
			treeBodyRows:  availableRows,
		}
	}
	// Bottom pane: spend height instead of width; the tree keeps full width.
	// (treeBoxWidth carries the legacy renderTreePanel convention where the
	// rendered box comes out 2 cells narrower than the given width.)
	paneRows := min(max(availableRows-6, 3), 12)
	paneBox := max(0, cols-2)
	return paneLayout{
		side:          false,
		paneBoxWidth:  paneBox,
		paneBodyWidth: paneBox - 4,
		treeBoxWidth:  cols,
		paneBodyRows:  paneRows,
		treeBodyRows:  max(0, availableRows-paneRows-2), // pane borders come out of the budget
	}
}

// statusGlyph maps a phase or sync-result code to its glyph and color.
func statusGlyph(status string) (string, color.Color) {
	switch status {
	case "Failed", "Error", "SyncFailed", "Terminating":
		return "✖", currentPalette.Danger
	case "Running":
		return "◌", currentPalette.Progress
	case "Pruned", "PruneSkipped":
		return "–", currentPalette.Dim
	default: // Succeeded, Synced, Healthy
		return "✔", currentPalette.Success
	}
}

// renderEventCards formats normalized events as display lines: a header row
// (warning marker + reason left, "xN · age" right) and the wrapped message
// indented beneath, with a blank line between cards.
func renderEventCards(events []model.ResourceEvent, width int, now time.Time) []string {
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)

	var lines []string
	for i, e := range events {
		if i > 0 {
			lines = append(lines, "")
		}
		// Warnings are marked by color alone; reasons stay flush left
		reasonStyle := lipgloss.NewStyle().Foreground(currentPalette.Text)
		if e.Type == "Warning" {
			reasonStyle = lipgloss.NewStyle().Foreground(currentPalette.Danger)
		}
		meta := fmt.Sprintf("x%d · %s", e.Count, humantime.Ago(e.LastSeen, now))
		gap := max(1, width-lipgloss.Width(e.Reason)-lipgloss.Width(meta))
		lines = append(lines, reasonStyle.Render(e.Reason)+strings.Repeat(" ", gap)+dim.Render(meta))
		for _, part := range wrapAnsiToWidth(e.Message, max(1, width-2)) {
			lines = append(lines, "  "+dim.Render(part))
		}
	}
	return lines
}

// renderSyncStatusBody formats the last operation's state as display lines:
// a label/value block followed by the per-resource RESULT rows.
func renderSyncStatusBody(details *model.SyncStatusDetails, width int, now time.Time) []string {
	const labelWidth = 14
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)
	text := lipgloss.NewStyle().Foreground(currentPalette.Text)

	var lines []string
	field := func(name, value string, style lipgloss.Style) {
		for i, part := range wrapAnsiToWidth(value, max(1, width-labelWidth)) {
			if i == 0 {
				lines = append(lines, dim.Render(fmt.Sprintf("%-*s", labelWidth, name))+style.Render(part))
			} else {
				lines = append(lines, strings.Repeat(" ", labelWidth)+style.Render(part))
			}
		}
	}

	_, phaseColor := statusGlyph(details.Phase)
	field("Operation", "Sync", text)
	field("Phase", details.Phase, lipgloss.NewStyle().Foreground(phaseColor))
	field("Started", humantime.AgoLong(details.StartedAt, now), text)
	duration := details.FinishedAt.Sub(details.StartedAt)
	if details.FinishedAt.IsZero() {
		duration = now.Sub(details.StartedAt)
	}
	field("Duration", humantime.Duration(duration), text)
	if details.Revision != "" {
		revision := details.Revision
		if len(revision) > 7 {
			revision = revision[:7]
		}
		field("Revision", revision, text)
	}
	if details.InitiatedBy != "" {
		field("Initiated by", details.InitiatedBy, text)
	} else if details.Automated {
		field("Initiated by", "automated sync policy", text)
	}
	if details.Message != "" {
		field("Message", details.Message, dim)
	}

	if len(details.Resources) > 0 {
		lines = append(lines, "", dim.Render("RESULT"))
		for _, r := range details.Resources {
			glyph, glyphColor := statusGlyph(r.Status)
			glyphStyle := lipgloss.NewStyle().Foreground(glyphColor)
			name := r.Name
			if r.Namespace != "" {
				name = r.Namespace + "/" + r.Name
			}
			// The status must survive at the row's end; the kind stays in
			// its 13-cell column and the name yields the rest
			kind := r.Kind
			if lipgloss.Width(kind) > 13 {
				kind = clipAnsiToWidth(kind, 12) + "…"
			}
			maxName := width - lipgloss.Width(fmt.Sprintf("%s %-13s", glyph, kind)) - lipgloss.Width(r.Status) - 1
			if lipgloss.Width(name) > maxName {
				name = clipAnsiToWidth(name, max(0, maxName-1)) + "…"
			}
			left := fmt.Sprintf("%s %-13s%s", glyph, kind, name)
			gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(r.Status))
			lines = append(lines,
				glyphStyle.Render(glyph)+" "+text.Render(fmt.Sprintf("%-13s", kind))+dim.Render(name)+
					strings.Repeat(" ", gap)+glyphStyle.Render(r.Status))
			for _, part := range wrapAnsiToWidth(r.Message, max(1, width-2)) {
				if part != "" {
					lines = append(lines, "  "+dim.Render(part))
				}
			}
		}
	}
	return lines
}

// severityColor mirrors the tree rows' status coloring for the pane's
// status fields.
func severityColor(s string) color.Color {
	switch strings.ToLower(s) {
	case "healthy", "running", "synced", "succeeded":
		return currentPalette.Success
	case "progressing", "pending":
		return currentPalette.Progress
	case "degraded", "error", "crashloop", "failed", "syncfailed":
		return currentPalette.Danger
	default:
		return currentPalette.Unknown
	}
}

// renderResourceStatusBody formats a resource row's status block: the tree's
// local knowledge (health, sync, age, health message) plus the resource's
// own row from the app's last sync RESULT. Unknown fields are skipped.
func renderResourceStatusBody(status *model.ResourceStatusSummary, details *model.SyncStatusDetails, target model.EventsResource, width int, now time.Time) []string {
	const labelWidth = 14
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)

	var lines []string
	field := func(name, value string, style lipgloss.Style) {
		for i, part := range wrapAnsiToWidth(value, max(1, width-labelWidth)) {
			if i == 0 {
				lines = append(lines, dim.Render(fmt.Sprintf("%-*s", labelWidth, name))+style.Render(part))
			} else {
				lines = append(lines, strings.Repeat(" ", labelWidth)+style.Render(part))
			}
		}
	}

	if status != nil {
		if status.Health != "" {
			field("Health", status.Health, lipgloss.NewStyle().Foreground(severityColor(status.Health)))
		}
		if status.Sync != "" {
			field("Sync", status.Sync, lipgloss.NewStyle().Foreground(severityColor(status.Sync)))
		}
		if !status.CreatedAt.IsZero() {
			field("Age", humantime.Age(status.CreatedAt, now), lipgloss.NewStyle().Foreground(currentPalette.Text))
		}
		if status.HealthMessage != "" {
			field("Message", status.HealthMessage, dim)
		}
	}
	if details != nil {
		for _, r := range details.Resources {
			if r.Kind == target.Kind && r.Namespace == target.Namespace && r.Name == target.Name {
				field("Last sync", r.Status, lipgloss.NewStyle().Foreground(severityColor(r.Status)))
				for _, part := range wrapAnsiToWidth(r.Message, max(1, width-2)) {
					if part != "" {
						lines = append(lines, "  "+dim.Render(part))
					}
				}
				break
			}
		}
	}
	return lines
}

// paneOpen reports whether the side pane is open.
func (m *Model) paneOpen() bool {
	return m.state.Events != nil
}

// renderSidePane renders the open pane at the given geometry, clamping its
// scroll offset to the content (the diff pager pattern). Application rows
// show the last sync operation's status block above the events.
func (m *Model) renderSidePane(l paneLayout) string {
	st := m.state.Events
	if st == nil {
		return ""
	}
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)
	danger := lipgloss.NewStyle().Foreground(currentPalette.Danger)
	wrapWith := func(body []string, text string, style lipgloss.Style) []string {
		for _, part := range wrapAnsiToWidth(text, max(1, l.paneBodyWidth)) {
			body = append(body, style.Render(part))
		}
		return body
	}

	appLevel := st.Target.Resource == (model.EventsResource{})
	title := "Application " + st.Target.AppName
	if !appLevel {
		title = fmt.Sprintf("%s %s", st.Target.Resource.Kind, st.Target.Resource.Name)
	}

	var body []string
	if appLevel {
		switch {
		case st.DetailsLoading:
			body = append(body, dim.Render("Loading sync status…"))
		case st.DetailsError != "":
			body = wrapWith(body, st.DetailsError, danger)
		case st.Details == nil:
			body = append(body, dim.Render("This application has never been synced."))
		default:
			body = append(body, renderSyncStatusBody(st.Details, l.paneBodyWidth, m.now())...)
		}
	} else {
		body = append(body, renderResourceStatusBody(st.ResourceStatus, st.Details, st.Target.Resource, l.paneBodyWidth, m.now())...)
	}
	if len(body) > 0 {
		body = append(body, "")
	}
	body = append(body, dim.Render("EVENTS"))
	switch {
	case st.Notice != "":
		body = wrapWith(body, st.Notice, dim)
	case st.Loading:
		body = append(body, dim.Render("Loading events…"))
	case st.Error != "":
		body = wrapWith(body, st.Error, danger)
	case len(st.Items) == 0:
		body = append(body, dim.Render("No events."))
	default:
		body = append(body, renderEventCards(st.Items, l.paneBodyWidth, m.now())...)
	}

	frameStatus := ""
	if interval := m.config.GetEventsRefreshInterval(); interval > 0 {
		frameStatus = "⟳ " + interval.String()
		if st.Refreshing {
			frameStatus = "⟳ refreshing"
		}
	}

	capacity := l.paneContentRows()
	st.Offset = min(max(0, st.Offset), max(0, len(body)-capacity))
	visible := body[st.Offset:min(st.Offset+capacity, len(body))]
	return renderPaneFrame(paneFrame{
		Title:        title,
		Width:        l.paneBoxWidth,
		BodyRows:     l.paneBodyRows,
		MoreAbove:    st.Offset > 0,
		MoreBelow:    st.Offset+capacity < len(body),
		Status:       frameStatus,
		StatusActive: st.Refreshing,
	}, visible)
}

// paneFrame describes the frame around a pane body.
type paneFrame struct {
	Title     string
	Width     int // outer width, borders included
	BodyRows  int // body rows to render (padded with blanks)
	MoreAbove bool
	MoreBelow bool
	// Status is anchored at the bottom border's left edge (auto-refresh
	// cadence / activity); StatusActive brightens it while work is in flight
	Status       string
	StatusActive bool
}

// renderPaneFrame draws a manually-composed rounded frame: the title lives
// in the top border, scroll markers anchor at fixed spots in the borders
// (▲ top-right, ▼ bottom-right) so users learn where to look.
func renderPaneFrame(f paneFrame, body []string) string {
	borderStyle := lipgloss.NewStyle().Foreground(currentPalette.Border)
	titleStyle := lipgloss.NewStyle().Foreground(currentPalette.Text)
	markerStyle := lipgloss.NewStyle().Foreground(currentPalette.Info)
	bodyWidth := max(0, f.Width-4)

	// Top border: ╭─ Title ────[▲ more above ─]╮
	title := f.Title
	maxTitle := f.Width - 6 // "╭─ " + " " + filler + "╮"
	if f.MoreAbove {
		maxTitle -= lipgloss.Width(" ▲ more above ─")
	}
	if lipgloss.Width(title) > maxTitle {
		title = clipAnsiToWidth(title, max(0, maxTitle-1)) + "…"
	}
	fill := f.Width - lipgloss.Width(title) - 5 // "╭─ " + " " + "╮"
	topRight := ""
	if f.MoreAbove {
		fill -= lipgloss.Width(" ▲ more above ─")
		topRight = markerStyle.Render(" ▲ more above ") + borderStyle.Render("─")
	}
	var b strings.Builder
	b.WriteString(borderStyle.Render("╭─ "))
	b.WriteString(titleStyle.Render(title))
	b.WriteString(borderStyle.Render(" " + strings.Repeat("─", max(0, fill))))
	b.WriteString(topRight)
	b.WriteString(borderStyle.Render("╮"))
	b.WriteString("\n")

	// Body rows, padded to the frame's height and width. The first row is
	// always blank — breathing room between the title and the content.
	for i := 0; i < f.BodyRows; i++ {
		line := strings.Repeat(" ", bodyWidth)
		if i > 0 && i-1 < len(body) && body[i-1] != "" {
			line = normalizeLinesToWidth(body[i-1], bodyWidth)
		}
		b.WriteString(borderStyle.Render("│") + " " + line + " " + borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Bottom border: ╰[─ status ]────[ ▼ more below ─]╯
	bottomFill := f.Width - 2
	bottomLeft := ""
	if f.Status != "" {
		statusStyle := lipgloss.NewStyle().Foreground(currentPalette.Dim)
		if f.StatusActive {
			statusStyle = markerStyle
		}
		bottomFill -= lipgloss.Width("─ " + f.Status + " ")
		bottomLeft = borderStyle.Render("─ ") + statusStyle.Render(f.Status) + borderStyle.Render(" ")
	}
	bottomRight := ""
	if f.MoreBelow {
		bottomFill -= lipgloss.Width(" ▼ more below ─")
		bottomRight = markerStyle.Render(" ▼ more below ") + borderStyle.Render("─")
	}
	b.WriteString(borderStyle.Render("╰"))
	b.WriteString(bottomLeft)
	b.WriteString(borderStyle.Render(strings.Repeat("─", max(0, bottomFill))))
	b.WriteString(bottomRight)
	b.WriteString(borderStyle.Render("╯"))
	return b.String()
}
