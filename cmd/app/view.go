package main

import (
    "fmt"
    "image/color"
    "os"
    "regexp"
    "strings"
    "time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// Color mappings from TypeScript colorFor() function
var (
	// Color scheme matching React+Ink app
	magentaBright = lipgloss.Color("13") // Selection highlight
	yellowBright  = lipgloss.Color("11") // Headers
	dimColor      = lipgloss.Color("8")  // Dimmed text

	// Status colors (matching TypeScript colorFor function)
	syncedColor    = lipgloss.Color("10") // Green for Synced/Healthy
	outOfSyncColor = lipgloss.Color("9")  // Red for OutOfSync/Degraded
	progressColor  = lipgloss.Color("11") // Yellow for Progressing
	unknownColor   = lipgloss.Color("8")  // Dim for Unknown
	cyanBright     = lipgloss.Color("14") // Cyan accents
	whiteBright    = lipgloss.Color("15") // Bright white
)

// Styles matching React+Ink components
var (
	// Main container style (matches MainLayout Box)
	mainContainerStyle = lipgloss.NewStyle().
		PaddingLeft(1).
		PaddingRight(1)

	// Border style for main content area (matches ListView container)
	// Add inner padding for readability; width calculations account for it
	contentBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(magentaBright).
		PaddingLeft(1).
		PaddingRight(1)

	// Header styles (matches ListView header)
	headerStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(yellowBright)

	// Selection highlight style (matches ListView active items)
	selectedStyle = lipgloss.NewStyle().
		Background(magentaBright)

	// Status bar style (matches MainLayout status line)
	statusStyle = lipgloss.NewStyle().
		Foreground(dimColor)
)

// ASCII icons matching React ListView
const (
	checkIcon = "V"
	warnIcon  = "!"
	questIcon = "?"
	deltaIcon = "^"
	dotIcon   = "."
)

// View implements tea.Model.View - 1:1 mapping from React App.tsx
func (m Model) View() string {
	if !m.ready {
		return statusStyle.Render("Starting…")
	}

	// Map React App.tsx switch statement exactly
	switch m.state.Mode {
    case model.ModeLoading:
        // Show regular layout with the initial loading modal overlay instead of a separate loading view
        return m.renderMainLayout()
	case model.ModeAuthRequired:
		return m.renderAuthRequiredView()
	case model.ModeHelp:
		return m.renderHelpModal()
	case model.ModeRollback:
		return m.renderRollbackModal()
	case model.ModeExternal:
		return "" // External mode returns null in React
	case model.ModeDiff:
		return m.renderDiffView()
	case model.ModeRulerLine:
		return m.renderOfficeSupplyManager()
	case model.ModeLogs:
		return m.renderLogsView()
	case model.ModeError:
		return m.renderErrorView()
	case model.ModeConnectionError:
		return m.renderConnectionErrorView()
	default:
		return m.renderMainLayout()
	}
}

// renderMainLayout - 1:1 mapping from MainLayout.tsx
// moved to view_layout.go

// countLines returns the number of lines in a rendered string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// renderBanner - 1:1 mapping from Banner.tsx
// moved to view_banner.go

// renderSmallBadge renders the compact "Argonaut <version>" badge used in narrow terminals.
// If grayscale is true, it uses a gray background with dark text so it stays readable when
// the base layer is desaturated for modal backdrops.
// moved to view_banner.go

// renderContextBlock renders the left-side context (labels + values)
// moved to view_banner.go

// renderAsciiLogo renders the right-side Argonaut ASCII logo like TS component
// moved to view_banner.go

// scopeToText formats a selection set for display
// moved to view_banner.go

// hostFromURL extracts host from URL (similar to TS hostFromUrl)
// moved to view_banner.go

// joinWithRightAlignment composes two multi-line blocks with the right block flush to the given width
// moved to view_banner.go

// contentInnerWidth computes inner content width inside the bordered box
// moved to view_layout.go

// moved to view_lists.go

// renderTreePanel renders the resource tree view inside a bordered container
// moved to view_layout.go

// renderListHeader - matches ListView header row with responsive widths
// moved to view_lists.go

// clipAnsiToWidth trims a styled string to the given display width (ANSI-aware)
// moved to view_utils.go

// Layout Helper Functions - Centralized layout management to ensure consistency

// FullScreenViewOptions configures the full-screen layout
type FullScreenViewOptions struct {
	ContentBordered bool
	BorderColor     color.Color // Optional: override border color (defaults to magentaBright)
}

// renderFullScreenView provides the standard full-terminal layout used by most views:
// header + content (optionally bordered) + status, with consistent height management
func (m Model) renderFullScreenView(header, content, status string, contentBordered bool) string {
	return m.renderFullScreenViewWithOptions(header, content, status, FullScreenViewOptions{
		ContentBordered: contentBordered,
		BorderColor:     magentaBright, // default
	})
}

// renderFullScreenViewWithOptions provides the full-screen layout with customizable options
func (m Model) renderFullScreenViewWithOptions(header, content, status string, opts FullScreenViewOptions) string {
	var sections []string

	// Header section
	if header != "" {
		sections = append(sections, header)
	}

	// Content section - apply border if requested
	if opts.ContentBordered {
		// Calculate available space for bordered content
		const (
			BORDER_LINES = 2 // content border top/bottom
			STATUS_LINES = 1 // bottom status line
		)

		headerLines := countLines(header)
		statusLines := countLines(status)
		overhead := BORDER_LINES + headerLines + statusLines
		availableRows := max(1, m.state.Terminal.Rows-overhead)

		// Apply bordered styling with custom color if specified
		contentWidth := max(0, m.state.Terminal.Cols-2) // Adjusted to fill space properly
        borderStyle := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(opts.BorderColor).
            Width(contentWidth).
            Height(availableRows + 1). // Add 1 to properly fill vertical space
            PaddingLeft(1).
            PaddingRight(1).
            AlignVertical(lipgloss.Top) // Align content to top for help/everywhere

		content = borderStyle.Render(content)
	}

	sections = append(sections, content)

	// Status section
	if status != "" {
		sections = append(sections, status)
	}

	// Apply main container with full height
	finalContent := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	return mainContainerStyle.Height(totalHeight).Render(finalContent)
}

// renderModalContent provides simple modal content styling (used by help modal)
// Returns only the styled content without full-screen layout
func (m Model) renderModalContent(content string) string {
	return contentBorderStyle.PaddingTop(1).PaddingBottom(1).Render(content)
}

// renderBackdropBlock returns a patterned, dim block for modal backdrops
func (m Model) renderBackdropBlock(width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	// Render a soft drop shadow using spaces with a dark background.
	line := strings.Repeat(" ", width)
	var b strings.Builder
	for y := 0; y < height; y++ {
		b.WriteString(line)
		if y < height-1 {
			b.WriteByte('\n')
		}
	}
	// Slightly dark background to suggest depth; keep foreground default
	style := lipgloss.NewStyle().Background(lipgloss.Color("236"))
	return style.Render(b.String())
}

// clipAnsiToLines trims the string to at most maxLines lines (ANSI-safe).
func clipAnsiToLines(s string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n")
}

// normalizeLinesToWidth pads or trims each line to an exact width (ANSI-aware)
func normalizeLinesToWidth(s string, width int) string {
	if width <= 0 || s == "" {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		w := lipgloss.Width(line)
		if w < width {
			lines[i] = padRight(line, width)
		} else if w > width {
			lines[i] = clipAnsiToWidth(line, width)
		}
	}
	return strings.Join(lines, "\n")
}

// ANSI escape sequence regex for colors/styles
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

// desaturateANSI strips ANSI color/style codes and recolors text dim gray
func desaturateANSI(s string) string {
	plain := ansiRE.ReplaceAllString(s, "")
	return lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render(plain)
}

// wrapAnsiToWidth wraps a string into visual lines that fit the given width (ANSI-aware)
// moved to view_utils.go

// renderAppRow - matches ListView app row rendering
// moved to view_lists.go

// padLeft returns s left-padded with spaces to the given visible width (ANSI-aware)
func padLeft(s string, width int) string {
	n := width - lipgloss.Width(s)
	if n > 0 {
		return strings.Repeat(" ", n) + s
	}
	return s
}

// padRight returns s right-padded with spaces to the given visible width (ANSI-aware)
func padRight(s string, width int) string {
	n := width - lipgloss.Width(s)
	if n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// renderSimpleRow - matches ListView non-app row rendering
// moved to view_lists.go

// renderStatusLine - 1:1 mapping from MainLayout status Box
// moved to view_status.go

// Helper functions matching TypeScript utilities

// moved to view_status.go

// moved to view_status.go

// moved to view_status.go

func (m Model) getVisibleItems() []interface{} {
	// Derive unique groups and filtered apps from current state, mirroring TS useVisibleItems
	// 1) Gather filtered apps through selected scopes
	apps := m.state.Apps

	// Filter by clusters scope
	if len(m.state.Selections.ScopeClusters) > 0 {
		filtered := make([]model.App, 0, len(apps))
		for _, a := range apps {
			var cl string
			if a.ClusterLabel != nil {
				cl = *a.ClusterLabel
			}
			if model.HasInStringSet(m.state.Selections.ScopeClusters, cl) {
				filtered = append(filtered, a)
			}
		}
		apps = filtered
	}

	// Compute all namespaces after cluster filtering
	// and optionally filter by namespace scope
	if len(m.state.Selections.ScopeNamespaces) > 0 {
		filtered := make([]model.App, 0, len(apps))
		for _, a := range apps {
			var ns string
			if a.Namespace != nil {
				ns = *a.Namespace
			}
			if model.HasInStringSet(m.state.Selections.ScopeNamespaces, ns) {
				filtered = append(filtered, a)
			}
		}
		apps = filtered
	}

	// Filter by project scope
	if len(m.state.Selections.ScopeProjects) > 0 {
		filtered := make([]model.App, 0, len(apps))
		for _, a := range apps {
			var prj string
			if a.Project != nil {
				prj = *a.Project
			}
			if model.HasInStringSet(m.state.Selections.ScopeProjects, prj) {
				filtered = append(filtered, a)
			}
		}
		apps = filtered
	}

	// 2) Build base list depending on current view
	var base []interface{}
	switch m.state.Navigation.View {
	case model.ViewClusters:
		// Unique cluster labels from all apps
		clusters := make([]string, 0)
		seen := map[string]bool{}
		for _, a := range m.state.Apps { // all apps (unscoped) define cluster list
			var cl string
			if a.ClusterLabel != nil {
				cl = *a.ClusterLabel
			}
			if cl == "" {
				continue
			}
			if !seen[cl] {
				seen[cl] = true
				clusters = append(clusters, cl)
			}
		}
		sortStrings(clusters)
		for _, c := range clusters {
			base = append(base, c)
		}
	case model.ViewNamespaces:
		// Unique namespaces from apps filtered by clusters scope
		nss := make([]string, 0)
		seen := map[string]bool{}
		for _, a := range apps {
			var ns string
			if a.Namespace != nil {
				ns = *a.Namespace
			}
			if ns == "" {
				continue
			}
			if !seen[ns] {
				seen[ns] = true
				nss = append(nss, ns)
			}
		}
		sortStrings(nss)
		for _, ns := range nss {
			base = append(base, ns)
		}
	case model.ViewProjects:
		// Unique projects from apps filtered by cluster+namespace scopes
		projs := make([]string, 0)
		seen := map[string]bool{}
		for _, a := range apps {
			var pj string
			if a.Project != nil {
				pj = *a.Project
			}
			if pj == "" {
				continue
			}
			if !seen[pj] {
				seen[pj] = true
				projs = append(projs, pj)
			}
		}
		sortStrings(projs)
		for _, pj := range projs {
			base = append(base, pj)
		}
	case model.ViewApps:
		for _, app := range apps {
			base = append(base, app)
		}
	default:
		// No-op
	}

	// 3) Apply text filter or search
	filter := m.state.UI.ActiveFilter
	if m.state.Mode == model.ModeSearch {
		filter = m.state.UI.SearchQuery
	}
	f := strings.ToLower(filter)
	if f == "" {
		return base
	}

	filtered := make([]interface{}, 0, len(base))
	if m.state.Navigation.View == model.ViewApps {
		for _, it := range base {
			app := it.(model.App)
			name := strings.ToLower(app.Name)
			sync := strings.ToLower(app.Sync)
			health := strings.ToLower(app.Health)
			var ns, prj string
			if app.Namespace != nil {
				ns = strings.ToLower(*app.Namespace)
			}
			if app.Project != nil {
				prj = strings.ToLower(*app.Project)
			}
			if strings.Contains(name, f) || strings.Contains(sync, f) || strings.Contains(health, f) || strings.Contains(ns, f) || strings.Contains(prj, f) {
				filtered = append(filtered, it)
			}
		}
	} else {
		for _, it := range base {
			s := strings.ToLower(fmt.Sprintf("%v", it))
			if strings.Contains(s, f) {
				filtered = append(filtered, it)
			}
		}
	}
	return filtered
}

// sortStrings sorts a slice of strings in-place (lexicographically)
func sortStrings(items []string) {
	// Simple insertion sort to avoid pulling extra deps; lists are small
	for i := 1; i < len(items); i++ {
		j := i
		for j > 0 && items[j-1] > items[j] {
			items[j-1], items[j] = items[j], items[j-1]
			j--
		}
	}
}

// Placeholder functions for other components (to be implemented)
func (m Model) renderLoadingView() string {
	serverText := "—"
	if m.state.Server != nil {
		serverText = m.state.Server.BaseURL
	}

	// Header matching LoadingView.tsx
	header := headerStyle.Render(fmt.Sprintf("View: LOADING • Context: %s", serverText))

	// Main content with bubbles spinner - let the layout helper handle centering
	loadingMessage := fmt.Sprintf("%s Loading...", m.spinner.View())
	content := lipgloss.NewStyle().Foreground(progressColor).Render(loadingMessage)

	// Status section
	status := statusStyle.Render("Starting…")

	// Use the new layout helper with bordered content
	return m.renderFullScreenView(header, content, status, true)
}

func (m Model) renderAuthRequiredView() string {
	serverText := "—"
	if m.state.Server != nil {
		serverText = m.state.Server.BaseURL
	}

	// Instructions (matches AuthRequiredView.tsx instructions array)
	instructions := []string{
		"1. Run: argocd login <your-argocd-server>",
		"2. Follow prompts to authenticate",
		"3. Re-run argonaut",
	}

	// Header - ArgoNaut Banner
	header := m.renderBanner()

	// Build content sections
	var contentSections []string
	contentSections = append(contentSections, "")

	// Auth header with background styling
	authHeaderStyled := lipgloss.NewStyle().
		Background(outOfSyncColor).
		Foreground(lipgloss.Color("15")).
		Bold(true).
		Render(" AUTHENTICATION REQUIRED ")
	authHeaderCentered := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Render(authHeaderStyled)
	contentSections = append(contentSections, authHeaderCentered)

	contentSections = append(contentSections, "")
	contentSections = append(contentSections, lipgloss.NewStyle().
		Foreground(outOfSyncColor).
		Bold(true).
		Align(lipgloss.Center).
		Render("Please login to ArgoCD before running argonaut."))
	contentSections = append(contentSections, "")

	// Add instructions
	for _, instruction := range instructions {
		contentSections = append(contentSections, statusStyle.Render("- "+instruction))
	}
	contentSections = append(contentSections, "")
	if serverText != "—" {
		contentSections = append(contentSections, statusStyle.Render("Current context: "+serverText))
	}

	// Join content sections
	content := strings.Join(contentSections, "\n")

	// Status
	status := statusStyle.Render("Press l to view logs, q to quit.")

	// Use the new layout helper with red border (matches AuthRequiredView borderColor="red")
	return m.renderFullScreenViewWithOptions(header, content, status, FullScreenViewOptions{
		ContentBordered: true,
		BorderColor:     outOfSyncColor, // red border for auth error
	})
}

// moved to view_modals.go

func (m Model) renderOfficeSupplyManager() string {
	return statusStyle.Render("Office supply manager - TODO: implement 1:1")
}

func (m Model) renderSearchBar() string {
	// 1:1 mapping from SearchBar.tsx
	if m.state.Mode != model.ModeSearch {
		return ""
	}

	// Search bar with border (matches SearchBar Box with borderStyle="round" borderColor="yellow")
	searchBarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		PaddingLeft(1).
		PaddingRight(1).
		// Ensure width matches the main bordered content box
		Width(m.contentInnerWidth())

	// Content matching SearchBar layout
	searchLabel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("Search")
	searchValue := m.state.UI.SearchQuery

	helpText := "Enter "
	if m.state.Navigation.View == model.ViewApps {
		helpText += "keeps filter"
	} else {
		helpText += "opens first result"
	}
	helpText += ", Esc cancels"

	content := fmt.Sprintf("%s %s  %s", searchLabel, searchValue, statusStyle.Render("("+helpText+")"))

	// Clip content to inner width to avoid stretching the box
	content = clipAnsiToWidth(content, m.contentInnerWidth())
	return searchBarStyle.Render(content)
}

func (m Model) renderCommandBar() string {
	// 1:1 mapping from CommandBar.tsx
	if m.state.Mode != model.ModeCommand {
		return ""
	}

	// Command bar with border (matches CommandBar Box with borderStyle="round" borderColor="yellow")
	commandBarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		PaddingLeft(1).
		PaddingRight(1).
		// Match main content width
		Width(m.contentInnerWidth())

	// Content matching CommandBar layout
	cmdLabel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("CMD")
	commandValue := ":" + m.state.UI.Command

	helpText := "(Enter to run, Esc to cancel)"
	if m.state.UI.Command != "" {
		helpText = "(Command entered)"
	}

	content := fmt.Sprintf("%s %s  %s", cmdLabel, commandValue, statusStyle.Render(helpText))

	content = clipAnsiToWidth(content, m.contentInnerWidth())
	return commandBarStyle.Render(content)
}

func (m Model) renderConfirmSyncModal() string {
	if m.state.Modals.ConfirmTarget == nil {
		return ""
	}

	target := *m.state.Modals.ConfirmTarget
	isMulti := target == "__MULTI__"

	// Modal width: compact and centered
	half := m.state.Terminal.Cols / 2
	modalWidth := min(max(36, half), m.state.Terminal.Cols-6)
	innerWidth := max(0, modalWidth-4) // border(2)+padding(2)

	// Message: de-emphasize the "Sync" verb and highlight the subject
	var titleLine string
	{
		// Build parts with different emphasis, then center as a whole
		syncPart := statusStyle.Render("Sync ") // dim
		var subject string
		if isMulti {
			subject = fmt.Sprintf("%d application(s)", len(m.state.Selections.SelectedApps))
		} else {
			subject = target
		}
		subjectStyled := lipgloss.NewStyle().Foreground(whiteBright).Bold(true).Render(subject)
		qmark := statusStyle.Render("?")
		titleLine = syncPart + subjectStyled + qmark
	}

	// Buttons: highlight selected using stronger contrast
	active := lipgloss.NewStyle().Background(magentaBright).Foreground(whiteBright).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(whiteBright).Padding(0, 2)
	yesBtn := inactive.Render("Yes")
	cancelBtn := inactive.Render("Cancel")
	if m.state.Modals.ConfirmSyncSelected == 0 {
		yesBtn = active.Render("Yes")
	}
	if m.state.Modals.ConfirmSyncSelected == 1 {
		cancelBtn = active.Render("Cancel")
	}

	// Options line (prune/watch) rendered below piecewise; no prebuilt string

	// Simple rounded border; cyan accent
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2).
		Width(modalWidth)

	// Center helpers
	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)

	title := center.Render(titleLine)

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, strings.Repeat(" ", 4), cancelBtn)
	buttons = center.Render(buttons)

	// Options line rendered piecewise to avoid ANSI resets affecting following text
	dim := lipgloss.NewStyle().Foreground(dimColor)
	on := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
	var optsLine strings.Builder
	optsLine.WriteString(dim.Render("p: Prune "))
	if m.state.Modals.ConfirmSyncPrune {
		optsLine.WriteString(on.Render("On"))
	} else {
		optsLine.WriteString(dim.Render("Off"))
	}
    // Always show watch toggle (single and multi)
    optsLine.WriteString(dim.Render(" • w: Watch "))
    if m.state.Modals.ConfirmSyncWatch {
        optsLine.WriteString(on.Render("On"))
    } else {
        optsLine.WriteString(dim.Render("Off"))
    }
	aux := center.Render(optsLine.String())

	// Lines are already centered to innerWidth; avoid re-normalizing which can
	// introduce asymmetric trailing padding.
	body := strings.Join([]string{title, "", buttons, "", aux}, "\n")

	// Add outer whitespace so the modal doesn't sit directly on top of content
	outer := lipgloss.NewStyle().Padding(1, 1) // 1 blank line top/bottom, 1 space left/right
	return outer.Render(wrapper.Render(body))
}

// removed entire implementation; stubbed for compatibility
func (m Model) renderResourceStream(availableRows int) string { return "" }

// buildGroupedResourceLines builds visual lines for multi-app resources with blank lines between apps
func (m Model) buildGroupedResourceLines(tableContentWidth int) []string { return nil }

// renderDiffView - simple pager for diff content
func (m Model) renderDiffView() string {
	if m.state.Diff == nil {
		return contentBorderStyle.Render("No diff loaded")
	}
	lines := m.state.Diff.Content
	// Apply filter if present
	if q := strings.ToLower(strings.TrimSpace(m.state.Diff.SearchQuery)); q != "" {
		filtered := make([]string, 0, len(lines))
		for _, ln := range lines {
			if strings.Contains(strings.ToLower(ln), q) {
				filtered = append(filtered, ln)
			}
		}
		lines = filtered
	}

	// Compute viewport height: account for all UI elements like main layout does
	// The diff view structure: title + bordered_content + status
	// contentBorderStyle adds 2 lines (top+bottom border), no vertical padding
	const (
		TITLE_LINES            = 1 // diff title line
		STATUS_LINES           = 1 // diff status line
		BORDER_LINES           = 2 // contentBorderStyle border top+bottom
		MAIN_CONTAINER_PADDING = 1 // main container has some margin
	)
	overhead := TITLE_LINES + STATUS_LINES + BORDER_LINES + MAIN_CONTAINER_PADDING
	contentHeight := max(3, m.state.Terminal.Rows-overhead)

	// Clamp offset - the content area height should be used for pagination
	if m.state.Diff.Offset < 0 {
		m.state.Diff.Offset = 0
	}
	if m.state.Diff.Offset > max(0, len(lines)-contentHeight) {
		m.state.Diff.Offset = max(0, len(lines)-contentHeight)
	}
	start := m.state.Diff.Offset
	end := min(len(lines), start+contentHeight)
	body := strings.Join(lines[start:end], "\n")

	title := headerStyle.Render(m.state.Diff.Title)
	status := statusStyle.Render(fmt.Sprintf("%d-%d/%d  j/k, g/G, / search, esc/q back", start+1, end, len(lines)))

	// Width should account for main container padding (2) and content border padding (2)
	contentWidth := max(0, m.state.Terminal.Cols-4)

	// Don't set a fixed height on the content border - let it size naturally
	content := contentBorderStyle.Width(contentWidth).Render(body)

	// Build sections ensuring header and status are always visible
	// Don't use fixed height container which can clip the header
	var sections []string
	sections = append(sections, title)
	sections = append(sections, content)
	sections = append(sections, status)

	// Join sections and apply main container style WITHOUT fixed height
	// This ensures title and status are always visible
	viewContent := strings.Join(sections, "\n")
	totalWidth := m.state.Terminal.Cols

	return mainContainerStyle.Width(totalWidth).Render(viewContent)
}

// renderHelpSection - helper for HelpModal (matches Help.tsx HelpSection)
func (m Model) renderHelpSection(title, content string, isWide bool) string {
    titleStyled := lipgloss.NewStyle().Foreground(syncedColor).Bold(true).Render(title)
    if isWide {
        // Two-column layout: 12-char title column + 1 space gap
        const col = 12
        // Pad the title visually to width 'col'
        padRightVisual := func(s string, w int) string {
            diff := w - lipgloss.Width(s)
            if diff > 0 { return s + strings.Repeat(" ", diff) }
            return s
        }
        lines := strings.Split(content, "\n")
        // Indent wrapped lines by title width + 1 space gap
        indent := strings.Repeat(" ", col+1)
        for i := 1; i < len(lines); i++ {
            lines[i] = indent + lines[i]
        }
        contentAligned := strings.Join(lines, "\n")
        titlePadded := padRightVisual(titleStyled, col)
        return titlePadded + " " + contentAligned
    }
    // Narrow layout: title above, content below
    return titleStyled + "\n" + content
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// abbreviateStatus shortens status text for narrow displays
// moved to view_utils.go

// truncateWithEllipsis truncates text to fit width, adding ellipsis if needed
func truncateWithEllipsis(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if maxWidth <= 3 {
		// Too narrow even for ellipsis
		return text[:min(len(text), maxWidth)]
	}

	// Use lipgloss.Width to handle ANSI sequences properly
	if lipgloss.Width(text) <= maxWidth {
		return text
	}

	// Need to truncate - reserve 3 characters for "..."
	targetWidth := maxWidth - 3
	if targetWidth <= 0 {
		return "..."
	}

	// Truncate character by character until we fit
	for i := len(text); i > 0; i-- {
		truncated := text[:i]
		if lipgloss.Width(truncated) <= targetWidth {
			return truncated + "..."
		}
	}

	return "..."
}

// calculateColumnWidths returns responsive column widths based on available space
// moved to view_utils.go

// calculateResourceColumnWidths returns responsive column widths for resources table
func calculateResourceColumnWidths(availableWidth int) (kindWidth, nameWidth, statusWidth int) {
	// Account for separators between the 3 columns (2 separators, 1 char each)
	const sep = 2

	switch {
	case availableWidth <= 0:
		return 0, 0, 0
	case availableWidth < 30:
		// Ultra-narrow: icon-only status, tiny kind
		kindWidth = 6
		statusWidth = 2
		nameWidth = max(10, availableWidth-kindWidth-statusWidth-sep)
	case availableWidth < 45:
		// Narrow: minimized columns
		kindWidth = 8
		statusWidth = 6
		nameWidth = max(12, availableWidth-kindWidth-statusWidth-sep)
	default:
		// Wide: full widths
		kindWidth = 20
		statusWidth = 15
		nameWidth = max(15, availableWidth-kindWidth-statusWidth-sep)
	}

	// Ensure exact fit including separators
	totalUsed := kindWidth + nameWidth + statusWidth + sep
	if totalUsed < availableWidth {
		nameWidth += (availableWidth - totalUsed)
	} else if totalUsed > availableWidth {
		overflow := totalUsed - availableWidth
		// Take overflow from name first, then kind if needed
		if nameWidth > overflow {
			nameWidth -= overflow
		} else {
			overflow -= nameWidth
			nameWidth = 1
			if kindWidth > overflow {
				kindWidth -= overflow
			} else {
				kindWidth = max(1, kindWidth-overflow)
			}
		}
	}

	return kindWidth, nameWidth, statusWidth
}

// renderLogsView renders the logs view with full-height layout
func (m Model) renderLogsView() string {
	// Dimensions
	containerWidth := max(0, m.state.Terminal.Cols-2)
	contentWidth := max(0, containerWidth-4)
	contentHeight := max(10, m.state.Terminal.Rows-6)

	// Build wrapped log lines (header + file content), clipped to viewport using offset
	wrapped := m.buildWrappedLogLines(contentWidth)
	// Ensure diff state exists for offset bookkeeping
	if m.state.Diff == nil {
		m.state.Diff = &model.DiffState{Title: "Logs", Content: []string{}, Offset: 0}
	}
	// Clamp offset to available range
	maxStart := max(0, len(wrapped)-contentHeight)
	if m.state.Diff.Offset < 0 {
		m.state.Diff.Offset = 0
	}
	if m.state.Diff.Offset > maxStart {
		m.state.Diff.Offset = maxStart
	}
	start := m.state.Diff.Offset
	end := min(len(wrapped), start+contentHeight)
	body := strings.Join(wrapped[start:end], "\n")

	// Render in a fixed-height bordered box; body is already clipped to avoid overflow
	logStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(magentaBright).
		Width(contentWidth).
		Height(contentHeight).
		AlignVertical(lipgloss.Top).
		PaddingLeft(1).
		PaddingRight(1)

	return logStyle.Render(body)
}

// readLogContent reads the actual log file content
func (m Model) readLogContent() string {
    // Try to read the log file path from environment (set by setupLogging)
    logFile := os.Getenv("ARGONAUT_LOG_FILE")
    if strings.TrimSpace(logFile) == "" {
        // Fallback to legacy location if env not set
        logFile = "logs/a9s.log"
    }
    content, err := os.ReadFile(logFile)
	if err != nil {
		return fmt.Sprintf("ArgoCD Application Logs\n\nError reading log file: %v\n\nPress q to return to main view.", err)
	}

	// Convert to string and add instructions
	logText := string(content)
	if logText == "" {
		return "ArgoCD Application Logs\n\nNo log entries found.\n\nPress q to return to main view."
	}

	// Add header and instructions
	header := "ArgoCD Application Logs\n\nPress q to return to main view.\n\n"
	return header + "--- Log Content ---\n\n" + logText
}

// buildWrappedLogLines returns header + log content lines wrapped to contentWidth
func (m Model) buildWrappedLogLines(contentWidth int) []string {
	text := m.readLogContent()
	// Split into logical lines, then wrap into visual lines
	logical := strings.Split(text, "\n")
	visual := make([]string, 0, len(logical))
	for _, ln := range logical {
		parts := wrapAnsiToWidth(ln, contentWidth)
		for _, p := range parts {
			// Ensure each visual line fits exactly (avoid residual wrap)
			visual = append(visual, clipAnsiToWidth(p, contentWidth))
		}
	}
	// Guarantee we have at least contentHeight lines to keep the box height consistent
	return visual
}

// renderFullHeightContent renders content with consistent full-height layout
func (m Model) renderFullHeightContent(content string, contentWidth, contentHeight, containerWidth int) string {
	// Create a full-height bordered box with vertically centered content
	fullHeightStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(magentaBright).
		Width(contentWidth).
		Height(contentHeight).
		AlignVertical(lipgloss.Center).
		AlignHorizontal(lipgloss.Center).
		PaddingLeft(1).
		PaddingRight(1)

	styledContent := fullHeightStyle.Render(content)

	// Apply container width for consistency with other views
	return contentBorderStyle.Width(containerWidth).Render(styledContent)
}

// renderErrorView displays API errors in a user-friendly format
func (m Model) renderErrorView() string {
	// Header
	header := m.renderBanner()

	// Build error content
	errorContent := ""

	// Check modern error structure first (structured errors)
	if m.state.ErrorState != nil && m.state.ErrorState.Current != nil {
		err := m.state.ErrorState.Current

		// Title with error category styling
		titleStyle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true)
		errorTitle := string(err.Category)
		if errorTitle == "" {
			errorTitle = "Error"
		}
		errorContent += titleStyle.Render(strings.Title(strings.ReplaceAll(errorTitle, "_", " "))) + "\n\n"

		// Error code/type
		if err.Code != "" {
			codeStyle := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
			errorContent += fmt.Sprintf("Code: %s\n", codeStyle.Render(err.Code))
		}

		// Main error message
		messageStyle := lipgloss.NewStyle().Foreground(whiteBright)
		errorContent += fmt.Sprintf("\nMessage:\n%s\n", messageStyle.Render(err.Message))

		// User action suggestion (if available)
		if err.UserAction != "" {
			actionStyle := lipgloss.NewStyle().Foreground(cyanBright)
			errorContent += fmt.Sprintf("\nSuggestion:\n%s\n", actionStyle.Render(err.UserAction))
		}

		// Additional context (if available)
		if err.Context != nil && len(err.Context) > 0 {
			contextStyle := lipgloss.NewStyle().Foreground(unknownColor)
			errorContent += "\nContext:\n"
			for key, value := range err.Context {
				errorContent += fmt.Sprintf("  %s: %s\n", contextStyle.Render(key), contextStyle.Render(fmt.Sprintf("%v", value)))
			}
		}

		// Timestamp
		timeStyle := lipgloss.NewStyle().Foreground(unknownColor)
		errorContent += fmt.Sprintf("\nTime: %s\n", timeStyle.Render(err.Timestamp.Format("2006-01-02 15:04:05")))

	} else if m.state.CurrentError != nil {
		// Fallback to legacy error structure
		err := m.state.CurrentError

		// Title with error type styling
		titleStyle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true)
		errorContent += titleStyle.Render("API Error") + "\n\n"

		// Status code (if available)
		if err.StatusCode > 0 {
			codeStyle := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
			errorContent += fmt.Sprintf("Status Code: %s\n", codeStyle.Render(fmt.Sprintf("%d", err.StatusCode)))
		}

		// Error code (if available)
		if err.ErrorCode > 0 {
			codeStyle := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
			errorContent += fmt.Sprintf("Error Code: %s\n", codeStyle.Render(fmt.Sprintf("%d", err.ErrorCode)))
		}

		// Main error message
		messageStyle := lipgloss.NewStyle().Foreground(whiteBright)
		errorContent += fmt.Sprintf("\nMessage:\n%s\n", messageStyle.Render(err.Message))

		// Additional details (if available)
		if err.Details != "" {
			detailStyle := lipgloss.NewStyle().Foreground(unknownColor)
			errorContent += fmt.Sprintf("\nDetails:\n%s\n", detailStyle.Render(err.Details))
		}

		// Timestamp
		timeStyle := lipgloss.NewStyle().Foreground(unknownColor)
		timeStr := time.Unix(err.Timestamp, 0).Format("2006-01-02 15:04:05")
		errorContent += fmt.Sprintf("\nTime: %s\n", timeStyle.Render(timeStr))
	} else {
		// Fallback error message
		errorContent = "An unknown error occurred."
	}

	// Instructions
	instructStyle := lipgloss.NewStyle().Foreground(cyanBright)
	errorContent += fmt.Sprintf("\n%s", instructStyle.Render("Press Esc to return to main view"))

	// Status (empty for error views)
	status := ""

	// Use the new layout helper with red border (matching error styling)
	return m.renderFullScreenViewWithOptions(header, errorContent, status, FullScreenViewOptions{
		ContentBordered: true,
		BorderColor:     outOfSyncColor, // red border for errors
	})
}

// renderConnectionErrorView displays connection error in a user-friendly format
func (m Model) renderConnectionErrorView() string {
    // Header
    header := m.renderBanner()

    // Build connection error content
    errorContent := ""

	// Title with connection error styling
	titleStyle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true)
	errorContent += titleStyle.Render("Connection Error") + "\n\n"

    // Main error message
    messageStyle := lipgloss.NewStyle().Foreground(whiteBright)
    errorContent += messageStyle.Render("Unable to connect to Argo CD server.\n\nPlease check that:\n• Argo CD server is running\n• Network connection is available\n• Server URL and port are correct") + "\n\n"

    // Tip: encourage checking the current context and re-auth
    tipStyle := lipgloss.NewStyle().Foreground(cyanBright)
    tip := "Tip: Ensure you are using the correct Argo CD context. You can switch or re-authenticate with: argocd login <server>"
    errorContent += tipStyle.Render(tip) + "\n\n"

	// Instructions
	instructStyle := lipgloss.NewStyle().Foreground(cyanBright)
	errorContent += instructStyle.Render("Press q to exit")

	// Status (empty for error views)
	status := ""

	// Use the new layout helper with red border (matching connection error styling)
	return m.renderFullScreenViewWithOptions(header, errorContent, status, FullScreenViewOptions{
		ContentBordered: true,
		BorderColor:     outOfSyncColor, // red border for connection errors
	})
}

// renderDiffLoadingSpinner displays a centered loading spinner for diff operations
// moved to view_modals.go

// renderSyncLoadingModal displays a compact centered modal with a spinner during sync start
// moved to view_modals.go

// renderInitialLoadingModal displays a compact centered modal with a spinner during initial app load
// moved to view_modals.go

// renderRollbackModal displays the rollback modal with deployment history
// moved to view_modals.go

// renderRollbackHistory renders the deployment history list
func (m Model) renderRollbackHistory(rollback *model.RollbackState) string {
	titleStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
	content := titleStyle.Render(fmt.Sprintf("Rollback %s", rollback.AppName)) + "\n\n"

	if len(rollback.Rows) == 0 {
		content += "No deployment history available"
		return content
	}

	// Show current revision info
	if rollback.CurrentRevision != "" {
		currentStyle := lipgloss.NewStyle().Foreground(syncedColor)
		content += currentStyle.Render(fmt.Sprintf("Current: %s", rollback.CurrentRevision[:min(8, len(rollback.CurrentRevision))])) + "\n\n"
	}

	// Show deployment history table
	content += "Deployment History:\n\n"

	// Compute how many rows we can show to avoid overflowing the modal.
	// This mirrors the height math used in renderRollbackModal.
	header := m.renderBanner()
	headerLines := countLines(header)
	const BORDER_LINES = 2
	const STATUS_LINES = 1
	availableRows := max(0, m.state.Terminal.Rows-(BORDER_LINES+headerLines+STATUS_LINES))

	// Inside the modal we render the following fixed lines when in list mode:
	// 2 (title + blank) + optional 2 for current revision + 2 (section header + blank)
	// + 2 (blank + options) added below in this function
	// + 3 (two blanks + instructions) appended by renderRollbackModal.
	fixedTop := 2
	if rollback.CurrentRevision != "" {
		fixedTop += 2
	}
	fixedBottom := 2 + 3
	rowsViewport := max(1, availableRows-fixedTop-fixedBottom)

	// Window the rows around the selection
	total := len(rollback.Rows)
	start := max(0, min(rollback.SelectedIdx-rowsViewport/2, total-rowsViewport))
	end := min(start+rowsViewport, total)

	// Indicators for clipped content
	if start > 0 {
		content += lipgloss.NewStyle().Foreground(dimColor).Render("… older entries above …") + "\n"
	}

	// Calculate the maximum line width inside the modal so rows never wrap
	containerWidth := max(0, m.state.Terminal.Cols-2)
	rowMaxWidth := max(0, containerWidth-4) // inner width (2 border + 2 padding)

	for i := start; i < end; i++ {
		row := rollback.Rows[i]
		var line string

		// Build single-line summary: id, short rev, date, author, and message
		idStyle := lipgloss.NewStyle().Foreground(whiteBright)
		revisionStyle := lipgloss.NewStyle().Foreground(cyanBright)
		line += fmt.Sprintf("%s %s",
			idStyle.Render(fmt.Sprintf("#%d", row.ID)),
			revisionStyle.Render(row.Revision[:min(8, len(row.Revision))]))

		if row.DeployedAt != nil {
			dateStyle := lipgloss.NewStyle().Foreground(unknownColor)
			line += " " + dateStyle.Render(row.DeployedAt.Format("2006-01-02 15:04"))
		}

		if row.Author != nil && row.Message != nil {
			authorStyle := lipgloss.NewStyle().Foreground(yellowBright)
			messageStyle := lipgloss.NewStyle().Foreground(whiteBright)
			// Leave message uncapped here; we clip to rowMaxWidth below.
			line += fmt.Sprintf(" %s: %s",
				authorStyle.Render(*row.Author),
				messageStyle.Render(*row.Message))
		} else if row.MetaError != nil {
			errorStyle := lipgloss.NewStyle().Foreground(outOfSyncColor)
			line += " " + errorStyle.Render("(metadata unavailable)")
		} else {
			loadingStyle := lipgloss.NewStyle().Foreground(unknownColor)
			line += " " + loadingStyle.Render("(loading metadata...)")
		}

		// Ensure single visual line within the modal width
		line = clipAnsiToWidth(line, rowMaxWidth)
		line = padRight(line, rowMaxWidth)

		// Highlight entire row when selected
		if i == rollback.SelectedIdx {
			content += selectedStyle.Render(line) + "\n"
		} else {
			content += line + "\n"
		}
	}

	if end < total {
		content += lipgloss.NewStyle().Foreground(dimColor).Render("… newer entries below …") + "\n"
	}

	// No options in list view; options are configured in confirmation view
	return content
}

// renderRollbackConfirmation renders the confirmation screen
func (m Model) renderRollbackConfirmation(rollback *model.RollbackState, innerHeight int, innerWidth int) string {
	// Top details section (no title here)
	content := ""

	if len(rollback.Rows) == 0 || rollback.SelectedIdx >= len(rollback.Rows) {
		return content + "Invalid selection"
	}

	selectedRow := rollback.Rows[rollback.SelectedIdx]

	// App info
	appStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
	content += fmt.Sprintf("Application: %s\n", appStyle.Render(rollback.AppName))

	// Current revision
	currentStyle := lipgloss.NewStyle().Foreground(syncedColor)
	content += fmt.Sprintf("Current: %s\n", currentStyle.Render(rollback.CurrentRevision[:min(8, len(rollback.CurrentRevision))]))

	// Target revision
	targetStyle := lipgloss.NewStyle().Foreground(yellowBright)
	content += fmt.Sprintf("Rollback to: %s\n", targetStyle.Render(selectedRow.Revision[:min(8, len(selectedRow.Revision))]))

	// Git metadata if available
	if selectedRow.Author != nil && selectedRow.Message != nil {
		content += fmt.Sprintf("Author: %s\n", *selectedRow.Author)
		content += fmt.Sprintf("Message: %s\n", *selectedRow.Message)
		if selectedRow.Date != nil {
			content += fmt.Sprintf("Date: %s\n", selectedRow.Date.Format("2006-01-02 15:04:05"))
		}
	}

	// Prepare bottom-aligned confirmation block
	if innerWidth < 20 {
		innerWidth = 20
	}
	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	on := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
	var opts strings.Builder
	opts.WriteString(dim.Render("[p] Prune: "))
	if rollback.Prune {
		opts.WriteString(on.Render("Yes"))
	} else {
		opts.WriteString(dim.Render("No"))
	}
	opts.WriteString(dim.Render("   [w] Watch: "))
	if rollback.Watch {
		opts.WriteString(on.Render("Yes"))
	} else {
		opts.WriteString(dim.Render("No"))
	}
	// Build inner confirmation modal (bordered) with title
	active := lipgloss.NewStyle().Background(magentaBright).Foreground(whiteBright).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(whiteBright).Padding(0, 2)
	yesBtn := inactive.Render("Yes")
	noBtn := inactive.Render("No")
	if rollback.ConfirmSelected == 0 {
		yesBtn = active.Render("Yes")
	}
	if rollback.ConfirmSelected == 1 {
		noBtn = active.Render("No")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, strings.Repeat(" ", 4), noBtn)

	confirmTitle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true).Render("Confirm Rollback")
	confirmInner := strings.Join([]string{
		center.Render(confirmTitle),
		"",
		center.Render(opts.String()),
		"",
		center.Render(buttons),
	}, "\n")

	// Render confirmation content centered without an inner box
	confirmBox := center.Render(confirmInner)

	bottomBlock := strings.Builder{}
	// Add a bit of top padding for the confirmation area
	bottomBlock.WriteString("\n")
	bottomBlock.WriteString(confirmBox)

	// Now bottom-align the confirmation block by inserting filler lines
	topLines := countLines(content)
	bottomLines := countLines(bottomBlock.String())
	filler := max(0, innerHeight-topLines-bottomLines)
	if filler > 0 {
		content += strings.Repeat("\n", filler)
	}
	content += bottomBlock.String()

	return content
}

// renderSimpleModal renders a simple modal with title and content
// moved to view_modals.go

// truncateString truncates a string to the specified length with ellipsis
// moved to view_utils.go
