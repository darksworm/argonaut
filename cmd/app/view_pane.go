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

// paneSideBoxWidth is the pane's fixed outer width in side-by-side mode —
// event text doesn't get more useful past ~46 content columns.
const paneSideBoxWidth = 50

// paneLayout describes how the tree box and the pane box share the screen.
type paneLayout struct {
	side          bool // side-by-side (wide) vs stacked bottom pane (narrow)
	paneBoxWidth  int  // pane outer width, borders included
	paneBodyWidth int  // pane content columns inside the frame
	treeBoxWidth  int  // tree box outer width
	paneBodyRows  int  // pane content rows inside the frame
	treeBodyRows  int  // tree content rows
}

// paneLayout computes the split for the given row budget (the rows the tree
// body alone would get with no pane open).
func (m *Model) paneLayout(availableRows int) paneLayout {
	cols := m.state.Terminal.Cols
	if cols >= paneSideMinCols {
		return paneLayout{
			side:          true,
			paneBoxWidth:  paneSideBoxWidth,
			paneBodyWidth: paneSideBoxWidth - 4, // borders + one space padding each side
			treeBoxWidth:  (cols - 2) - paneSideBoxWidth,
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
		marker := "  "
		reasonStyle := lipgloss.NewStyle().Foreground(currentPalette.Text)
		if e.Type == "Warning" {
			marker = "! "
			reasonStyle = lipgloss.NewStyle().Foreground(currentPalette.Danger)
		}
		meta := fmt.Sprintf("x%d · %s", e.Count, humantime.Ago(e.LastSeen, now))
		gap := max(1, width-lipgloss.Width(marker+e.Reason)-lipgloss.Width(meta))
		lines = append(lines, reasonStyle.Render(marker+e.Reason)+strings.Repeat(" ", gap)+dim.Render(meta))
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

// paneOpen reports whether a side pane (events or sync status) is open.
func (m *Model) paneOpen() bool {
	return m.state.Events != nil || m.state.SyncStatus != nil
}

// renderSidePane renders whichever pane is open at the given geometry,
// clamping its scroll offset to the content (the diff pager pattern).
func (m *Model) renderSidePane(l paneLayout) string {
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)
	danger := lipgloss.NewStyle().Foreground(currentPalette.Danger)

	var title string
	var body []string
	var offset *int
	switch {
	case m.state.Events != nil:
		st := m.state.Events
		offset = &st.Offset
		title = "Events · Application " + st.Target.AppName
		if st.Target.Resource != (model.EventsResource{}) {
			title = fmt.Sprintf("Events · %s %s", st.Target.Resource.Kind, st.Target.Resource.Name)
		}
		switch {
		case st.Loading:
			body = []string{dim.Render("Loading events…")}
		case st.Error != "":
			for _, part := range wrapAnsiToWidth(st.Error, max(1, l.paneBodyWidth)) {
				body = append(body, danger.Render(part))
			}
		case len(st.Items) == 0:
			body = []string{dim.Render("No events.")}
		default:
			body = renderEventCards(st.Items, l.paneBodyWidth, m.now())
		}
	case m.state.SyncStatus != nil:
		st := m.state.SyncStatus
		offset = &st.Offset
		title = "Sync Status · " + st.Target.AppName
		switch {
		case st.Loading:
			body = []string{dim.Render("Loading sync status…")}
		case st.Error != "":
			for _, part := range wrapAnsiToWidth(st.Error, max(1, l.paneBodyWidth)) {
				body = append(body, danger.Render(part))
			}
		case st.Details == nil:
			body = []string{dim.Render("This application has never been synced.")}
		default:
			body = renderSyncStatusBody(st.Details, l.paneBodyWidth, m.now())
		}
	default:
		return ""
	}

	*offset = min(max(0, *offset), max(0, len(body)-l.paneBodyRows))
	visible := body[*offset:min(*offset+l.paneBodyRows, len(body))]
	return renderPaneFrame(paneFrame{
		Title:     title,
		Width:     l.paneBoxWidth,
		BodyRows:  l.paneBodyRows,
		MoreAbove: *offset > 0,
		MoreBelow: *offset+l.paneBodyRows < len(body),
	}, visible)
}

// paneFrame describes the frame around a pane body.
type paneFrame struct {
	Title     string
	Width     int // outer width, borders included
	BodyRows  int // body rows to render (padded with blanks)
	MoreAbove bool
	MoreBelow bool
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

	// Body rows, padded to the frame's height and width
	for i := 0; i < f.BodyRows; i++ {
		line := strings.Repeat(" ", bodyWidth)
		if i < len(body) && body[i] != "" {
			line = normalizeLinesToWidth(body[i], bodyWidth)
		}
		b.WriteString(borderStyle.Render("│") + " " + line + " " + borderStyle.Render("│"))
		b.WriteString("\n")
	}

	// Bottom border: ╰────[ ▼ more below ─]╯
	bottomFill := f.Width - 2
	bottomRight := ""
	if f.MoreBelow {
		bottomFill -= lipgloss.Width(" ▼ more below ─")
		bottomRight = markerStyle.Render(" ▼ more below ") + borderStyle.Render("─")
	}
	b.WriteString(borderStyle.Render("╰" + strings.Repeat("─", max(0, bottomFill))))
	b.WriteString(bottomRight)
	b.WriteString(borderStyle.Render("╯"))
	return b.String()
}
