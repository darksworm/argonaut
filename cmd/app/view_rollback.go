package main

// The rollback view: a full-width deployment history list (capped at 10
// rows) over a detail pane that tracks the row under the cursor. Both panes
// reuse the tree view's frame chrome.

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/humantime"
	"github.com/darksworm/argonaut/pkg/model"
)

// rollbackListMaxRows caps the history list's height; Argo CD's default
// revisionHistoryLimit is 10, and the detail pane needs the rest.
const rollbackListMaxRows = 10

// renderRollbackLayout composes banner + history list + detail pane + status
// line. Always a horizontal split: the list is as tall as the history
// (capped), the full-width detail pane gets the remaining height.
func (m *Model) renderRollbackLayout() string {
	header := m.renderBanner()
	l := m.rollbackPaneLayout(header)

	sections := []string{header}
	if m.state.Terminal.Cols > 100 {
		sections = append(sections, "")
	}
	if bar := m.renderEnhancedCommandBar(); bar != "" {
		sections = append(sections, bar)
	}
	sections = append(sections, m.renderRollbackList(l), m.renderRollbackDetail(l))
	sections = append(sections, m.renderStatusLine())
	base := mainContainerStyle.Render(strings.Join(sections, "\n"))

	rb := m.state.Rollback
	if rb == nil || rb.Mode != "confirm" || rb.Loading {
		return base
	}
	modal := m.renderRollbackConfirmModal()
	return m.composeOverlay(
		lipgloss.NewLayer(desaturateANSI(base)),
		lipgloss.NewLayer(modal).
			X((m.state.Terminal.Cols-lipgloss.Width(modal))/2).
			Y((m.state.Terminal.Rows-lipgloss.Height(modal))/2).
			Z(1),
	)
}

// renderRollbackConfirmModal is the centered confirmation box shown over
// the desaturated rollback view, in the same visual language as the sync
// confirmation modal.
func (m *Model) renderRollbackConfirmModal() string {
	rb := m.state.Rollback
	row := rb.SelectedRow()

	dim := lipgloss.NewStyle().Foreground(dimColor)

	// Title: de-emphasize the verb, highlight the app
	verb := "Rollback"
	if rb.IsRedeploy() {
		verb = "Redeploy"
	}
	title := dim.Render(verb+" ") +
		lipgloss.NewStyle().Foreground(whiteBright).Bold(true).Render(rb.AppName) +
		dim.Render("?")

	// What moves where
	sha := func(revision string) string {
		s := shortRevision(revision)
		return lipgloss.NewStyle().Foreground(currentPalette.ShaColor(s)).Render(s)
	}
	transition := sha(rb.CurrentRevision) + dim.Render(" (current) → ") + sha(row.Revision)
	if row.DeployedAt != nil {
		transition += dim.Render(fmt.Sprintf(" (%s)", humantime.Ago(*row.DeployedAt, m.now())))
	}

	// Size the modal to its content: the widest line plus generous side
	// margins, clamped to the terminal (long commit subjects get truncated
	// first). The margin also hides the centering rounding, which drops
	// its spare cell on one side.
	maxSubject := min(56, m.state.Terminal.Cols-18)
	subject := truncateWithEllipsis(commitSubject(*row), maxSubject)
	naturalWidth := max(lipgloss.Width(title), lipgloss.Width(transition))
	naturalWidth = max(naturalWidth, lipgloss.Width(subject))
	modalWidth := min(max(56, naturalWidth+12), m.state.Terminal.Cols-6)
	innerWidth := max(0, modalWidth-6) // borders(2) + padding(2+2); Width includes the borders
	// Center by hand: lipgloss puts the odd spare cell on the left, which
	// reads as uneven padding; the wrapper pads the right edge for us
	center := func(s string) string {
		return strings.Repeat(" ", max(0, (innerWidth-lipgloss.Width(s))/2)) + s
	}

	lines := []string{center(title), "", center(transition)}
	if subject != "" {
		lines = append(lines, center(dim.Render(subject)))
	}
	if rb.AutoSyncEnabled {
		warn := lipgloss.NewStyle().Foreground(yellowBright)
		lines = append(lines, "", center(warn.Render("⚠ Auto-sync on — will be disabled first")))
	}

	// Buttons: strong contrast on the selected action, like the sync modal
	inactiveFG := ensureContrastingForeground(inactiveBG, whiteBright)
	active := lipgloss.NewStyle().Background(magentaBright).Foreground(textOnAccent).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(inactiveBG).Foreground(inactiveFG).Padding(0, 2)
	actionBtn := inactive.Render(verb)
	cancelBtn := inactive.Render("Cancel")
	if rb.ConfirmSelected == 0 {
		actionBtn = active.Render(verb)
	} else {
		cancelBtn = active.Render("Cancel")
	}
	lines = append(lines, "", center(lipgloss.JoinHorizontal(lipgloss.Center, actionBtn, strings.Repeat(" ", 4), cancelBtn)))

	// Options, rendered piecewise like the sync modal
	on := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
	onOff := func(v bool) string {
		if v {
			return on.Render("On")
		}
		return dim.Render("Off")
	}
	lines = append(lines, "", center(dim.Render("p: Prune ")+onOff(rb.Prune)+dim.Render(" • w: Watch ")+onOff(rb.Watch)))

	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2).
		Width(modalWidth)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(strings.Join(lines, "\n")))
}

// rollbackPaneLayout computes the stacked split under the given banner.
func (m *Model) rollbackPaneLayout(header string) paneLayout {
	cols := m.state.Terminal.Cols
	gap := 0
	if cols > 100 {
		gap = 1
	}
	// banner + gap + command bar + list frame (body+2) + detail frame
	// (body+2) + status
	barLines := 0
	if bar := m.renderEnhancedCommandBar(); bar != "" {
		barLines = countLines(bar)
	}
	availableBody := m.state.Terminal.Rows - countLines(header) - gap - barLines - 1 - 4
	listEntries := 1
	if rb := m.state.Rollback; rb != nil && len(rb.Rows) > 0 {
		listEntries = min(len(rb.Rows), rollbackListMaxRows)
	}
	return paneLayout{
		paneBoxWidth:  max(0, cols-2),
		paneBodyWidth: max(1, cols-6),
		treeBoxWidth:  cols,
		treeBodyRows:  listEntries, // flush frame: one body row per entry
		paneBodyRows:  max(3, availableBody-listEntries),
	}
}

// rollbackListGeometry returns the list frame's outer width and body rows.
// The box follows the tree panel convention of rendering 2 cells narrower
// than its allotted width so rows stay flush with the status line.
func rollbackListGeometry(l paneLayout) (width, bodyRows int) {
	return l.treeBoxWidth - 2, l.treeBodyRows
}

// rollbackInitiator names who triggered a deployment; the viewing user
// reads as "you", like the events pane.
func rollbackInitiator(row model.RollbackRow, selfUser string) string {
	switch {
	case row.InitiatedBy != "" && row.InitiatedBy == selfUser:
		return "you"
	case row.InitiatedBy != "":
		return row.InitiatedBy
	case row.Automated:
		return "automated"
	}
	return ""
}

// commitSubject is the first line of a commit message, or "" while the
// metadata is still loading.
func commitSubject(row model.RollbackRow) string {
	if row.Message == nil {
		return ""
	}
	return strings.SplitN(*row.Message, "\n", 2)[0]
}

// renderRollbackList renders the deployment history list pane, windowed
// around the cursor.
func (m *Model) renderRollbackList(l paneLayout) string {
	rb := m.state.Rollback
	width, bodyRows := rollbackListGeometry(l)
	bodyWidth := max(1, width-4)
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)
	text := lipgloss.NewStyle().Foreground(currentPalette.Text)

	var body []string
	moreAbove, moreBelow := false, false
	switch {
	case rb == nil || rb.Loading:
		body = append(body, dim.Render("Loading deployment history…"))
	case rb.Error != "":
		for _, part := range wrapAnsiToWidth(rb.Error, bodyWidth) {
			body = append(body, lipgloss.NewStyle().Foreground(currentPalette.Danger).Render(part))
		}
	case len(rb.Rows) == 0:
		body = append(body, dim.Render("No deployment history."))
	default:
		start, end := rollbackVisibleWindow(rb.SelectedIdx, len(rb.Rows), bodyRows)
		moreAbove, moreBelow = start > 0, end < len(rb.Rows)

		// Left-packed table: the subject and initiator columns are sized to
		// the widest visible value, so the metadata sits right next to the
		// text it describes instead of drifting to the far edge of a wide
		// terminal.
		const idShaWidth, ageWidth, byWidth, badgeWidth = 14, 8, 3, 8
		maxSubject, maxInitiator := 0, 0
		for i := start; i < end; i++ {
			maxSubject = max(maxSubject, lipgloss.Width(commitSubject(rb.Rows[i])))
			maxInitiator = max(maxInitiator, lipgloss.Width(rollbackInitiator(rb.Rows[i], m.currentUsername)))
		}
		subjectWidth := min(maxSubject, max(0, bodyWidth-idShaWidth-ageWidth-byWidth-maxInitiator-badgeWidth-2))
		if subjectWidth < 8 {
			subjectWidth = 0 // a sliver of subject is just noise
		}

		for i := start; i < end; i++ {
			row := rb.Rows[i]
			age := ""
			if row.DeployedAt != nil {
				age = humantime.Ago(*row.DeployedAt, m.now())
			}
			initiator := rollbackInitiator(row, m.currentUsername)
			isYou := initiator == "you"
			by := "by "
			if initiator == "" {
				by = "   "
			}
			subject := ""
			if subjectWidth > 0 {
				subject = truncateWithEllipsis(commitSubject(row), subjectWidth)
			}
			left := fmt.Sprintf("#%-3d %-8s %-*s  %-*s %s",
				row.ID, shortRevision(row.Revision), subjectWidth, subject, ageWidth, age, by)
			initiatorPadded := fmt.Sprintf("%-*s", maxInitiator, initiator)
			badge := ""
			if i == 0 {
				badge = "  current"
			}
			pad := strings.Repeat(" ", max(0, bodyWidth-lipgloss.Width(left)-maxInitiator-lipgloss.Width(badge)))

			if i == rb.SelectedIdx {
				body = append(body, selectedStyle.Render(left)+selectedStyle.Bold(isYou).Render(initiatorPadded)+selectedStyle.Render(badge+pad))
				continue
			}
			style := dim
			if i == 0 {
				style = text
			}
			// "you" stays bright even on dim rows, like the events pane
			initiatorStyle := style
			if isYou {
				initiatorStyle = text.Bold(true)
			}
			body = append(body, style.Render(left)+initiatorStyle.Render(initiatorPadded)+
				lipgloss.NewStyle().Foreground(currentPalette.Success).Render(badge))
		}
	}

	title := "Deployment history"
	if rb != nil {
		title += " " + rb.AppName
	}
	return renderPaneFrame(paneFrame{
		Title:     title,
		Width:     width,
		BodyRows:  bodyRows,
		Flush:     true,
		MoreAbove: moreAbove,
		MoreBelow: moreBelow,
	}, body)
}

// renderRollbackDetail renders the right pane: details for the row under the
// cursor, or the confirmation when one is pending.
func (m *Model) renderRollbackDetail(l paneLayout) string {
	rb := m.state.Rollback
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)

	frame := paneFrame{Width: l.paneBoxWidth, BodyRows: l.paneBodyRows}
	var body []string
	switch {
	case rb == nil || rb.Loading || rb.Error != "" || len(rb.Rows) == 0:
		frame.Title = "Deployment"
	default:
		row := rb.SelectedRow()
		frame.Title = fmt.Sprintf("#%d %s", row.ID, shortRevision(row.Revision))
		if row.DeployedAt != nil {
			frame.Status = "deployed " + humantime.Ago(*row.DeployedAt, m.now())
		}
		if rb.Notice != "" {
			warn := lipgloss.NewStyle().Foreground(currentPalette.Warning)
			for _, part := range wrapAnsiToWidth(rb.Notice, max(1, l.paneBodyWidth)) {
				body = append(body, warn.Render(part))
			}
			body = append(body, "")
		}
		body = append(body, m.renderRollbackDetailBody(l.paneBodyWidth)...)
	}
	if len(body) == 0 {
		body = append(body, dim.Render("Nothing to show."))
	}
	if rb != nil {
		capacity := max(0, frame.BodyRows-1)
		rb.DetailOffset = min(max(0, rb.DetailOffset), max(0, len(body)-capacity))
		frame.MoreAbove = rb.DetailOffset > 0
		frame.MoreBelow = rb.DetailOffset+capacity < len(body)
		body = body[rb.DetailOffset:min(rb.DetailOffset+capacity, len(body))]
	}
	return renderPaneFrame(frame, body)
}

// renderRollbackDetailBody formats the selected deployment as DEPLOYMENT /
// COMMIT / SOURCE label-value sections.
func (m *Model) renderRollbackDetailBody(width int) []string {
	rb := m.state.Rollback
	row := rb.SelectedRow()
	now := m.now()

	const labelWidth = 15
	dim := lipgloss.NewStyle().Foreground(currentPalette.Dim)
	text := lipgloss.NewStyle().Foreground(currentPalette.Text)
	danger := lipgloss.NewStyle().Foreground(currentPalette.Danger)

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

	lines = append(lines, dim.Render("DEPLOYMENT"))
	if row.DeployedAt != nil {
		field("Deployed", fmt.Sprintf("%s · %s", humantime.Ago(*row.DeployedAt, now), row.DeployedAt.Format("2006-01-02 15:04")), text)
	}
	if row.DeployStartedAt != nil && row.DeployedAt != nil {
		field("Time to deploy", humantime.Duration(row.DeployedAt.Sub(*row.DeployStartedAt)), text)
	}
	switch {
	case row.InitiatedBy != "" && row.InitiatedBy == m.currentUsername:
		field("Initiated by", "you", text.Bold(true))
	case row.InitiatedBy != "":
		field("Initiated by", row.InitiatedBy, text)
	case row.Automated:
		field("Initiated by", "automated sync policy", text)
	}
	if activeFor := rollbackActiveFor(rb, now); activeFor != "" {
		field("Active for", activeFor, text)
	}

	lines = append(lines, "")
	sha := shortRevision(row.Revision)
	lines = append(lines, dim.Render("COMMIT         ")+lipgloss.NewStyle().Foreground(currentPalette.ShaColor(sha)).Render(row.Revision))
	switch {
	case row.MetaError != nil:
		field("Metadata", *row.MetaError, danger)
	case row.Author == nil:
		lines = append(lines, dim.Render("Loading commit metadata…"))
	default:
		field("Author", *row.Author, text)
		if row.Date != nil {
			field("Date", fmt.Sprintf("%s · %s", humantime.Ago(*row.Date, now), row.Date.Format("2006-01-02 15:04")), text)
		}
		if row.Message != nil {
			lines = append(lines, "")
			lines = append(lines, wrapMessageLines(*row.Message, width, text)...)
		}
	}

	if row.Source != nil {
		lines = append(lines, "", dim.Render("SOURCE"))
		field("Repo", row.Source.RepoURL, text)
		field("Path", row.Source.Path, text)
		field("Target rev", row.Source.TargetRevision, text)
	}
	return lines
}

// wrapMessageLines renders a possibly multi-line commit message as one body
// row per terminal line — a row with an embedded newline breaks the pane
// frame's height math.
func wrapMessageLines(message string, width int, style lipgloss.Style) []string {
	var out []string
	for _, raw := range strings.Split(message, "\n") {
		if raw == "" {
			out = append(out, "")
			continue
		}
		for _, part := range wrapAnsiToWidth(raw, max(1, width)) {
			out = append(out, style.Render(part))
		}
	}
	return out
}

// rollbackActiveFor is how long the selected deployment was (or has been)
// the live one: until the next-newer deployment, or until now for the newest.
func rollbackActiveFor(rb *model.RollbackState, now time.Time) string {
	row := rb.SelectedRow()
	if row.DeployedAt == nil {
		return ""
	}
	if rb.SelectedIdx == 0 {
		return humantime.Duration(now.Sub(*row.DeployedAt)) + " — still active"
	}
	newer := rb.Rows[rb.SelectedIdx-1]
	if newer.DeployedAt == nil {
		return ""
	}
	return humantime.Duration(newer.DeployedAt.Sub(*row.DeployedAt))
}
