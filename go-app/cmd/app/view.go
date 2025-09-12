package main

import (
    "fmt"
    "net/url"
    "os"
    "regexp"
    "strings"
    "time"

    "github.com/a9s/go-app/pkg/model"
    "github.com/charmbracelet/lipgloss/v2"
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
		return m.renderLoadingView()
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
func (m Model) renderMainLayout() string {
	// Height calculations - dynamic based on rendered section heights
	const (
		BORDER_LINES       = 2 // content border top/bottom
		TABLE_HEADER_LINES = 0 // header is inside the table itself
		TAG_LINE           = 0 // not used
		STATUS_LINES       = 1 // bottom status line
	)

	// Render header and optional bars first to measure their heights
	header := m.renderBanner()
	searchBar := ""
	if m.state.Mode == model.ModeSearch {
		searchBar = m.renderEnhancedSearchBar()
	}
	commandBar := ""
	if m.state.Mode == model.ModeCommand {
		commandBar = m.renderEnhancedCommandBar()
	}

	headerLines := countLines(header)
	searchLines := countLines(searchBar)
	commandLines := countLines(commandBar)

	overhead := BORDER_LINES + headerLines + searchLines + commandLines + TABLE_HEADER_LINES + TAG_LINE + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)
	listRows := max(0, availableRows)

	var sections []string

	// ArgoNaut Banner (matches MainLayout ArgoNautBanner)
	sections = append(sections, header)

	// Search/Command bars
	if searchBar != "" {
		sections = append(sections, searchBar)
	}
	if commandBar != "" {
		sections = append(sections, commandBar)
	}

	// Main content area (matches MainLayout Box with border)
	if m.state.Mode == model.ModeResources && m.state.Server != nil && m.state.Modals.SyncViewApp != nil {
		sections = append(sections, m.renderResourceStream(listRows))
	} else {
		sections = append(sections, m.renderListView(listRows))
	}

	// Status line (matches MainLayout status Box)
	sections = append(sections, m.renderStatusLine())

	// Join with newlines and apply main container style with full width
	content := strings.Join(sections, "\n")
	// Render the full terminal area; padding is handled by the container style
	totalHeight := m.state.Terminal.Rows
	_ = totalHeight

	baseView := mainContainerStyle.Render(content)

    // Modal overlay (render atop existing content using lipgloss v2 layers)
    if m.state.Mode == model.ModeConfirmSync {
        modal := ""
        if m.state.Modals.ConfirmSyncLoading {
            modal = m.renderSyncLoadingModal()
        } else {
            modal = m.renderConfirmSyncModal()
        }
        mw := lipgloss.Width(modal)
        mh := lipgloss.Height(modal)

        // Center modal; desaturate base layer so the modal pops
        grayBase := desaturateANSI(baseView)
        modalX := (m.state.Terminal.Cols - mw) / 2
        modalY := (totalHeight - mh) / 2

        baseLayer := lipgloss.NewLayer(grayBase)
        modalLayer := lipgloss.NewLayer(modal).X(modalX).Y(modalY).Z(1)
        canvas := lipgloss.NewCanvas(baseLayer, modalLayer)
        return canvas.Render()
    }

	// Add diff loading spinner as an overlay if loading using lipgloss v2 layer/canvas system
	if m.state.Diff != nil && m.state.Diff.Loading {
		// Create spinner overlay using lipgloss v2 layer composition
		spinner := m.renderDiffLoadingSpinner()
		
		// Create base layer with the existing view content
		baseLayer := lipgloss.NewLayer(baseView)
		
		// Create spinner layer positioned in center with higher Z-index
		spinnerLayer := lipgloss.NewLayer(spinner).
			X((m.state.Terminal.Cols - lipgloss.Width(spinner)) / 2).
			Y((m.state.Terminal.Rows - lipgloss.Height(spinner)) / 2).
			Z(1) // Place spinner above base content
		
		// Create canvas with both layers
		canvas := lipgloss.NewCanvas(
			baseLayer,      // Base view content at Z=0
			spinnerLayer,   // Spinner overlay at Z=1
		)
		
		return canvas.Render()
	}

	return baseView
}

// countLines returns the number of lines in a rendered string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// renderBanner - 1:1 mapping from Banner.tsx
func (m Model) renderBanner() string {
	// Determine narrow layout threshold similar to TS
	isNarrow := m.state.Terminal.Cols <= 100

	// Cyan badge for very narrow terminals
    if isNarrow {
        badge := lipgloss.NewStyle().
            Background(cyanBright).
            Foreground(whiteBright).
            Bold(true).
            PaddingLeft(1).
            PaddingRight(1).
            Render("Argonaut " + appVersion)

		// Add spacing before and after the badge, and after Project line
		var sections []string
		sections = append(sections, "") // Empty line before badge
		sections = append(sections, badge)
		sections = append(sections, "") // Empty line after badge

		// Context block (stacked)
		ctx := m.renderContextBlock(true)
		sections = append(sections, ctx)
		sections = append(sections, "") // Empty line after Project

		return strings.Join(sections, "\n")
	}

	// Wide layout: left context block, right ASCII logo, bottom-aligned and pushed to right edge
	left := m.renderContextBlock(false)
	right := m.renderAsciiLogo()

	// Normalize heights by padding top of the shorter block
	leftLines := strings.Count(left, "\n") + 1
	rightLines := strings.Count(right, "\n") + 1
	if leftLines < rightLines {
		pad := strings.Repeat("\n", rightLines-leftLines)
		left = pad + left
	} else if rightLines < leftLines {
		pad := strings.Repeat("\n", leftLines-rightLines)
		right = pad + right
	}

	// Compute full row width inside main container (account for main container padding of 1 on each side)
	total := max(0, m.state.Terminal.Cols-2)
	return joinWithRightAlignment(left, right, total)
}

// renderContextBlock renders the left-side context (labels + values)
func (m Model) renderContextBlock(isNarrow bool) string {
	if m.state.Server == nil {
		return ""
	}

	label := lipgloss.NewStyle().Bold(true).Foreground(whiteBright)
	cyan := lipgloss.NewStyle().Foreground(cyanBright)
	green := lipgloss.NewStyle().Foreground(syncedColor)

	// Values
	serverHost := hostFromURL(m.state.Server.BaseURL)
	clusterScope := scopeToText(m.state.Selections.ScopeClusters)
	namespaceScope := scopeToText(m.state.Selections.ScopeNamespaces)
	projectScope := scopeToText(m.state.Selections.ScopeProjects)

	var lines []string
	lines = append(lines, fmt.Sprintf("%s %s", label.Render("Context:"), cyan.Render(serverHost)))
	if clusterScope != "—" {
		lines = append(lines, fmt.Sprintf("%s %s", label.Render("Cluster:"), clusterScope))
	}
	if namespaceScope != "—" {
		lines = append(lines, fmt.Sprintf("%s %s", label.Render("Namespace:"), namespaceScope))
	}
	if projectScope != "—" {
		lines = append(lines, fmt.Sprintf("%s %s", label.Render("Project:"), projectScope))
	}
	if !isNarrow && m.state.APIVersion != "" {
		lines = append(lines, fmt.Sprintf("%s %s", label.Render("ArgoCD:"), green.Render(m.state.APIVersion)))
	}

	// Right padding between context and logo
	block := strings.Join(lines, "\n")
	return lipgloss.NewStyle().PaddingRight(2).Render(block)
}

// renderAsciiLogo renders the right-side Argonaut ASCII logo like TS component
func (m Model) renderAsciiLogo() string {
	cyan := lipgloss.NewStyle().Foreground(cyanBright)
	white := lipgloss.NewStyle().Foreground(whiteBright)
	dim := lipgloss.NewStyle().Foreground(dimColor)

	// Last line version (Argonaut version from build)
	version := appVersion
	versionPadded := fmt.Sprintf("%13s", version)

	l1 := cyan.Render("   _____") + strings.Repeat(" ", 43) + white.Render(" __   ")
	l2 := cyan.Render("  /  _  \\_______  ____   ____") + white.Render("   ____ _____   __ ___/  |_ ")
	l3 := cyan.Render(" /  /_\\  \\_  __ \\/ ___\\ /  _ \\ ") + white.Render("/    \\\\__  \\ |  |  \\   __\\")
	l4 := cyan.Render(" /    |    \\  | \\/ /_/  >  <_> )  ") + white.Render(" |  \\/ __ \\|  |  /|  |  ")
	l5 := cyan.Render("\\____|__  /__|  \\___  / \\____/") + white.Render("|___|  (____  /____/ |__|  ")
	l6 := cyan.Render("        \\/     /_____/             ") + white.Render("\\/     \\/") + dim.Render(versionPadded)

	return strings.Join([]string{l1, l2, l3, l4, l5, l6}, "\n")
}

// scopeToText formats a selection set for display
func scopeToText(set map[string]bool) string {
	if len(set) == 0 {
		return "—"
	}
	vals := make([]string, 0, len(set))
	for k := range set {
		vals = append(vals, k)
	}
	sortStrings(vals)
	return strings.Join(vals, ",")
}

// hostFromURL extracts host from URL (similar to TS hostFromUrl)
func hostFromURL(s string) string {
	if s == "" {
		return "—"
	}
	if u, err := url.Parse(s); err == nil && u.Host != "" {
		return u.Host
	}
	return s
}

// joinWithRightAlignment composes two multi-line blocks with the right block flush to the given width
func joinWithRightAlignment(left, right string, totalWidth int) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")
	n := len(leftLines)
	if len(rightLines) > n {
		n = len(rightLines)
	}

	// Ensure equal length slices
	if len(leftLines) < n {
		pad := make([]string, n-len(leftLines))
		leftLines = append(pad, leftLines...)
	}
	if len(rightLines) < n {
		pad := make([]string, n-len(rightLines))
		rightLines = append(pad, rightLines...)
	}

	var out []string
	for i := 0; i < n; i++ {
		l := leftLines[i]
		r := rightLines[i]
		lw := lipgloss.Width(l)
		rw := lipgloss.Width(r)
		filler := totalWidth - lw - rw
		if filler < 1 {
			filler = 1
		}
		out = append(out, l+strings.Repeat(" ", filler)+r)
	}
	return strings.Join(out, "\n")
}

// contentInnerWidth computes inner content width inside the bordered box
func (m Model) contentInnerWidth() int {
	// Subtract: main padding (2) + border (2) + inner padding (2)
	// Reduced slack to use more available space
	return max(0, m.state.Terminal.Cols-6)
}

// renderListView - custom list/table rendering with fixed inner width
func (m Model) renderListView(availableRows int) string {
	visibleItems := m.getVisibleItems()

	contentWidth := max(0, m.contentInnerWidth())
	// Leave room for the table header row inside the bordered area
	tableHeight := max(3, availableRows-1)

	// Handle empty state
	if len(visibleItems) == 0 {
		emptyContent := statusStyle.Render("No items.")
		return contentBorderStyle.Render(emptyContent)
	}

	// Prepare data and update the appropriate table directly
	var tableView string

	switch m.state.Navigation.View {
	case model.ViewApps:
		// Custom-render apps list to restore full-row selection highlight and per-cell colors
		// Determine viewport to keep selection visible
		total := len(visibleItems)
		visibleRows := max(0, tableHeight-1) // leave 1 line for header
		if visibleRows <= 0 {
			visibleRows = 0
		}
		cursor := m.state.Navigation.SelectedIdx
		if cursor < 0 {
			cursor = 0
		}
		if cursor >= total {
			cursor = max(0, total-1)
		}
		start := cursor - visibleRows/2
		if start < 0 {
			start = 0
		}
		if start > max(0, total-visibleRows) {
			start = max(0, total-visibleRows)
		}
		end := min(total, start+visibleRows)

		// Build header + rows
		var b strings.Builder
		b.WriteString(m.renderListHeader())
		b.WriteString("\n")
		for i := start; i < end; i++ {
			app := visibleItems[i].(model.App)
			isCursor := (i == cursor)
			b.WriteString(m.renderAppRow(app, isCursor))
			if i < end-1 {
				b.WriteString("\n")
			}
		}
		// Pad remaining lines to maintain fixed height inside border
		for pad := end - start; pad < visibleRows; pad++ {
			b.WriteString("\n")
		}
		tableView = b.String()

	case model.ViewClusters, model.ViewNamespaces, model.ViewProjects:
		// Custom-render single-column lists with full-row highlight
		total := len(visibleItems)
		visibleRows := max(0, tableHeight-1)
		cursor := m.state.Navigation.SelectedIdx
		if cursor < 0 {
			cursor = 0
		}
		if cursor >= total {
			cursor = max(0, total-1)
		}
		start := cursor - visibleRows/2
		if start < 0 {
			start = 0
		}
		if start > max(0, total-visibleRows) {
			start = max(0, total-visibleRows)
		}
		end := min(total, start+visibleRows)

		var b strings.Builder
		b.WriteString(m.renderListHeader())
		b.WriteString("\n")
		for i := start; i < end; i++ {
			label := fmt.Sprintf("%v", visibleItems[i])
			isCursor := (i == cursor)
			b.WriteString(m.renderSimpleRow(label, isCursor))
			if i < end-1 {
				b.WriteString("\n")
			}
		}
		for pad := end - start; pad < visibleRows; pad++ {
			b.WriteString("\n")
		}
		tableView = b.String()

	default:
		// Fallback to simple empty content to avoid bubbles table
		tableView = ""
	}

	// Render the table/content ensuring each line fits the content width
	var content strings.Builder
	content.WriteString(normalizeLinesToWidth(tableView, contentWidth))

	// Apply border style with proper width. Let height auto-size to content
	// to avoid tmux line-wrapping issues.
	return contentBorderStyle.Render(content.String())
}

// renderListHeader - matches ListView header row with responsive widths
func (m Model) renderListHeader() string {
	if m.state.Navigation.View == model.ViewApps {
		// Fixed-width columns with full text headers
		contentWidth := m.contentInnerWidth()
		syncWidth := 12
		healthWidth := 15
		nameWidth := max(10, contentWidth-syncWidth-healthWidth-2)

		nameHeader := headerStyle.Render("NAME")
		syncHeader := headerStyle.Render("SYNC")
		healthHeader := headerStyle.Render("HEALTH")

		nameCell := padRight(clipAnsiToWidth(nameHeader, nameWidth), nameWidth)
		syncCell := padLeft(clipAnsiToWidth(syncHeader, syncWidth), syncWidth)
		healthCell := padLeft(clipAnsiToWidth(healthHeader, healthWidth), healthWidth)

		header := fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)
		// Guarantee exact width to prevent underline overflow
		if lipgloss.Width(header) < contentWidth {
			header = padRight(header, contentWidth)
		} else if lipgloss.Width(header) > contentWidth {
			header = clipAnsiToWidth(header, contentWidth)
		}
		return header
	}

	// Simple header for other views padded to full content width
	contentWidth := m.contentInnerWidth()
	hdr := headerStyle.Render("NAME")
	if lipgloss.Width(hdr) < contentWidth {
		hdr = padRight(hdr, contentWidth)
	} else if lipgloss.Width(hdr) > contentWidth {
		hdr = clipAnsiToWidth(hdr, contentWidth)
	}
	return hdr
}

// clipAnsiToWidth trims a styled string to the given display width (ANSI-aware)
func clipAnsiToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		candidate := b.String() + string(r)
		if lipgloss.Width(candidate) > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
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

// renderAppRow - matches ListView app row rendering
func (m Model) renderAppRow(app model.App, isCursor bool) string {
	// Selection checking (matches ListView isChecked logic)
	isSelected := m.state.Selections.HasSelectedApp(app.Name)
	active := isCursor || isSelected

	// Get project name (for future use)
	_ = "default"
	if app.Project != nil {
		_ = *app.Project
	}

	// Prepare texts and widths using same responsive logic as header
	syncIcon := m.getSyncIcon(app.Sync)
	healthIcon := m.getHealthIcon(app.Health)

	contentWidth := m.contentInnerWidth() // Match header/content inner width
	nameWidth, syncWidth, healthWidth := calculateColumnWidths(contentWidth)

	// Generate text based on available width (either full text or icons only)
	// Colored status strings with icons (as before)
	syncText := fmt.Sprintf("%s %s", syncIcon, app.Sync)
	healthText := fmt.Sprintf("%s %s", healthIcon, app.Health)

	// Truncate app name with ellipsis if it's too long
	truncatedName := truncateWithEllipsis(app.Name, nameWidth)

	var nameCell, syncCell, healthCell string
	// Build cells with clipping to assigned widths to prevent wrapping
	nameCell = padRight(truncateWithEllipsis(truncatedName, nameWidth), nameWidth)

	if isCursor || isSelected {
		// Active row: avoid inner color styles so background highlight spans the whole row
		if lipgloss.Width(syncText) > syncWidth {
			syncText = clipAnsiToWidth(syncText, syncWidth)
		}
		if lipgloss.Width(healthText) > healthWidth {
			healthText = clipAnsiToWidth(healthText, healthWidth)
		}
		syncCell = padLeft(syncText, syncWidth)
		healthCell = padLeft(healthText, healthWidth)
	} else {
		// Inactive row: apply color styles to sync/health then clip if needed
		syncStyled := m.getColorForStatus(app.Sync).Render(syncText)
		healthStyled := m.getColorForStatus(app.Health).Render(healthText)
		if lipgloss.Width(syncStyled) > syncWidth {
			syncStyled = clipAnsiToWidth(syncStyled, syncWidth)
		}
		if lipgloss.Width(healthStyled) > healthWidth {
			healthStyled = clipAnsiToWidth(healthStyled, healthWidth)
		}
		syncCell = padLeft(syncStyled, syncWidth)
		healthCell = padLeft(healthStyled, healthWidth)
	}

	row := fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)

	// Ensure row is exactly the content width to avoid wrapping
	fullRowWidth := nameWidth + syncWidth + healthWidth + 2 // +2 for separators
	if lipgloss.Width(row) < fullRowWidth {
		row = padRight(row, fullRowWidth)
	} else if lipgloss.Width(row) > fullRowWidth {
		row = clipAnsiToWidth(row, fullRowWidth)
	}

	// Apply selection highlight (matches ListView backgroundColor)
	if active {
		row = selectedStyle.Render(row)
		// After styling, clip again defensively (some terminals render bold differently)
		if lipgloss.Width(row) > fullRowWidth {
			row = clipAnsiToWidth(row, fullRowWidth)
		}
	}

	return row
}

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
func (m Model) renderSimpleRow(label string, isCursor bool) string {
	// Check if selected based on view (matches ListView isChecked logic)
	isSelected := false
	switch m.state.Navigation.View {
	case model.ViewClusters:
		isSelected = m.state.Selections.HasCluster(label)
	case model.ViewNamespaces:
		isSelected = m.state.Selections.HasNamespace(label)
	case model.ViewProjects:
		isSelected = m.state.Selections.HasProject(label)
	}

	active := isCursor || isSelected

	// Calculate available width for simple rows (full content width minus padding)
	contentWidth := m.contentInnerWidth()

	// Truncate and pad label to full width
	truncatedLabel := truncateWithEllipsis(label, contentWidth)
	row := padRight(truncatedLabel, contentWidth)

	// Apply selection highlight if active
	if active {
		return selectedStyle.Render(row)
	}
	return row
}

// renderStatusLine - 1:1 mapping from MainLayout status Box
func (m Model) renderStatusLine() string {
	visibleItems := m.getVisibleItems()

	// Left side: view and filter info (matches MainLayout left Box)
	leftText := fmt.Sprintf("<%s>", m.state.Navigation.View)
	if m.state.UI.ActiveFilter != "" && m.state.Navigation.View == model.ViewApps {
		leftText = fmt.Sprintf("<%s:%s>", m.state.Navigation.View, m.state.UI.ActiveFilter)
	}

	// Right side: status and position (matches MainLayout right Box)
	position := "0/0"
	if len(visibleItems) > 0 {
		position = fmt.Sprintf("%d/%d", m.state.Navigation.SelectedIdx+1, len(visibleItems))
	}

	rightText := fmt.Sprintf("Ready • %s", position)
	if m.state.UI.IsVersionOutdated {
		rightText += " • Update available!"
	}

	// Layout matching MainLayout justifyContent="space-between"
	leftStyled := statusStyle.Render(leftText)
	rightStyled := statusStyle.Render(rightText)

	// Available width inside main container (accounts for its padding)
	available := max(0, m.state.Terminal.Cols-2)
	// Use lipgloss.Width for accurate spacing
	gap := max(0, available-lipgloss.Width(leftText)-lipgloss.Width(rightText))
	line := lipgloss.JoinHorizontal(
		lipgloss.Center,
		leftStyled,
		strings.Repeat(" ", gap),
		rightStyled,
	)
	// Ensure the status line exactly fits the available width
	w := lipgloss.Width(line)
	if w < available {
		line = padRight(line, available)
	} else if w > available {
		line = clipAnsiToWidth(line, available)
	}
	return line
}

// Helper functions matching TypeScript utilities

func (m Model) getSyncIcon(sync string) string {
	switch sync {
	case "Synced":
		return checkIcon
	case "OutOfSync":
		return deltaIcon
	case "Unknown":
		return questIcon
	default:
		return warnIcon
	}
}

func (m Model) getHealthIcon(health string) string {
	switch health {
	case "Healthy":
		return checkIcon
	case "Missing":
		return questIcon
	case "Degraded":
		return warnIcon
	case "Progressing":
		return dotIcon
	default:
		return questIcon
	}
}

func (m Model) getColorForStatus(status string) lipgloss.Style {
	switch status {
	case "Synced", "Healthy":
		return lipgloss.NewStyle().Foreground(syncedColor)
	case "OutOfSync", "Degraded":
		return lipgloss.NewStyle().Foreground(outOfSyncColor)
	case "Progressing":
		return lipgloss.NewStyle().Foreground(progressColor)
	default:
		return lipgloss.NewStyle().Foreground(unknownColor)
	}
}

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
	loadingHeader := fmt.Sprintf("View: LOADING • Context: %s", serverText)

	// Main content with bubbles spinner (matches LoadingView center box)
	loadingMessage := fmt.Sprintf("%s Connecting & fetching applications…", m.spinner.View())

	var sections []string

	// Header section
	sections = append(sections, headerStyle.Render(loadingHeader))

	// Center loading message with proper spacing
	centerPadding := max(0, (m.state.Terminal.Rows-6)/2)
	for i := 0; i < centerPadding; i++ {
		sections = append(sections, "")
	}
	sections = append(sections, lipgloss.NewStyle().
		Foreground(progressColor).
		Render(loadingMessage))

	// Fill remaining space
	for i := 0; i < centerPadding; i++ {
		sections = append(sections, "")
	}

	// Status section (matches LoadingView bottom)
	sections = append(sections, statusStyle.Render("Starting…"))

	// Join content and apply border (matches LoadingView Box with border)
	content := strings.Join(sections, "\n")
	// Use inner content width for bordered area
	totalWidth := m.contentInnerWidth()
	return contentBorderStyle.Width(totalWidth).Render(content)
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

	var sections []string

	// Calculate widths: banner needs full width, auth box needs constrained width
	containerWidth := max(0, m.state.Terminal.Cols-2)
	contentWidth := max(0, containerWidth-1) // Account for auth box padding

	// ArgoNaut Banner needs full container width to render properly
	banner := m.renderBanner()
	sections = append(sections, banner)

	// Main content area with auth message (matches AuthRequiredView main Box)
	var contentSections []string

	// Center the content vertically
	contentSections = append(contentSections, "")

	// Apply background only to text, then center within full width
	authHeaderStyled := lipgloss.NewStyle().
		Background(outOfSyncColor).
		Foreground(lipgloss.Color("15")).
		Bold(true).
		Render(" AUTHENTICATION REQUIRED ")
	authHeaderCentered := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(authHeaderStyled)
	contentSections = append(contentSections, authHeaderCentered)

	contentSections = append(contentSections, "")
	contentSections = append(contentSections, lipgloss.NewStyle().
		Foreground(outOfSyncColor).
		Bold(true).
		Width(contentWidth).
		Align(lipgloss.Center).
		Render("Please login to ArgoCD before running argonaut."))
	contentSections = append(contentSections, "")

	// Add instructions (matches AuthRequiredView instructions map)
	for _, instruction := range instructions {
		contentSections = append(contentSections, statusStyle.Width(contentWidth).Render("- "+instruction))
	}
	contentSections = append(contentSections, "")
	if serverText != "—" {
		contentSections = append(contentSections, statusStyle.Width(contentWidth).Render("Current context: "+serverText))
	}
	contentSections = append(contentSections, statusStyle.Width(contentWidth).Render("Press l to view logs, q to quit."))

	// Calculate available height for auth box (total - banner - status line)
	bannerHeight := strings.Count(banner, "\n") + 1
	statusHeight := 1                                                                // status line is always 1 line
	availableAuthHeight := max(5, m.state.Terminal.Rows-bannerHeight-statusHeight-2) // -2 for some padding

	// Apply border with red color, full width and height (matches AuthRequiredView borderColor="red")
	authBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outOfSyncColor).
		Width(contentWidth).
		Height(availableAuthHeight).
		PaddingLeft(2).
		PaddingRight(2).
		PaddingTop(1).
		PaddingBottom(1).
		AlignVertical(lipgloss.Center) // Center content vertically in the full-height box

	authContent := authBoxStyle.Render(strings.Join(contentSections, "\n"))
	sections = append(sections, authContent)

	// Join with newlines and apply main container style with full width
	content := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1

	return mainContainerStyle.Height(totalHeight).Render(content)
}

func (m Model) renderHelpModal() string {
	// 1:1 mapping from HelpModal.tsx + Help.tsx
	isWide := m.state.Terminal.Cols >= 60

	var sections []string

	// GENERAL section
	generalContent := ": command • / search • ? help"
	sections = append(sections, m.renderHelpSection("GENERAL", generalContent, isWide))

	// NAV section
	navContent := "j/k up/down • Space select • Enter drill down • Esc clear/up"
	sections = append(sections, m.renderHelpSection("NAV", navContent, isWide))

	// VIEWS section
	viewsContent := ":cls|:clusters|:cluster • :ns|:namespaces|:namespace\n:proj|:projects|:project • :apps"
	sections = append(sections, m.renderHelpSection("VIEWS", viewsContent, isWide))

	// ACTIONS section
	actionsContent := ":diff [app] • :sync [app] • :rollback [app]\n:up go up level\ns sync modal • R rollback modal (apps view)"
	sections = append(sections, m.renderHelpSection("ACTIONS", actionsContent, isWide))

	// MISC section
	miscContent := ":all • :licenses\n:logs • :q"
	sections = append(sections, m.renderHelpSection("MISC", miscContent, isWide))

	// Close instruction
	sections = append(sections, "")
	sections = append(sections, statusStyle.Render("Press ?, q or Esc to close"))

	content := strings.Join(sections, "\n")
	return contentBorderStyle.PaddingTop(1).PaddingBottom(1).Render(content)
}


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
		PaddingRight(1)

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
		PaddingRight(1)

	// Content matching CommandBar layout
	cmdLabel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("CMD")
	commandValue := ":" + m.state.UI.Command

	helpText := "(Enter to run, Esc to cancel)"
	if m.state.UI.Command != "" {
		helpText = "(Command entered)"
	}

	content := fmt.Sprintf("%s %s  %s", cmdLabel, commandValue, statusStyle.Render(helpText))

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
    if m.state.Modals.ConfirmSyncSelected == 0 { yesBtn = active.Render("Yes") }
    if m.state.Modals.ConfirmSyncSelected == 1 { cancelBtn = active.Render("Cancel") }

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
    if !isMulti {
        optsLine.WriteString(dim.Render(" • w: Watch "))
        if m.state.Modals.ConfirmSyncWatch {
            optsLine.WriteString(on.Render("On"))
        } else {
            optsLine.WriteString(dim.Render("Off"))
        }
    }
    aux := center.Render(optsLine.String())

    // Lines are already centered to innerWidth; avoid re-normalizing which can
    // introduce asymmetric trailing padding.
    body := strings.Join([]string{title, "", buttons, "", aux}, "\n")

    return wrapper.Render(body)
}

func (m Model) renderResourceStream(availableRows int) string {
	// Calculate dimensions for consistent full-height layout
	containerWidth := max(0, m.state.Terminal.Cols-8)
	contentWidth := max(0, containerWidth-4) // Account for border and padding
	contentHeight := max(3, availableRows)

	if m.state.Resources == nil {
		return m.renderFullHeightContent("Loading resources...", contentWidth, contentHeight, containerWidth)
	}

	if m.state.Resources.Error != "" {
		errorContent := fmt.Sprintf("Error loading resources:\n%s\n\nPress q to return", m.state.Resources.Error)
		return m.renderFullHeightContent(errorContent, contentWidth, contentHeight, containerWidth)
	}

	if m.state.Resources.Loading {
		loadingContent := fmt.Sprintf("Loading resources for %s...\n\nPress q to return", m.state.Resources.AppName)
		return m.renderFullHeightContent(loadingContent, contentWidth, contentHeight, containerWidth)
	}

	resources := m.state.Resources.Resources
    if len(resources) == 0 {
        // Create single bordered box with standard inner padding (1 char each side)
        resourcesStyle := lipgloss.NewStyle().
            Border(lipgloss.RoundedBorder()).
            BorderForeground(magentaBright).
            Width(contentWidth).
            Height(contentHeight).
            AlignVertical(lipgloss.Top).
            PaddingLeft(1).
            PaddingRight(1)

		// Highlight the app name in cyan
		appNameStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
		highlightedAppName := appNameStyle.Render(m.state.Resources.AppName)

		emptyContent := fmt.Sprintf("No resources found for application: %s\n\nPress q to return", highlightedAppName)
		return resourcesStyle.Render(emptyContent)
	}

    // Calculate widths so the inner content fills the bordered box exactly
    // Outer container width inside main container
    tableContainerWidth := max(0, m.state.Terminal.Cols-2) // account for mainContainer left/right padding
    // Inner width = outer width - borders(2) - padding(2)
    tableContentWidth := max(0, tableContainerWidth-4)
	// Leave one line for the table header
	tableHeight := max(3, availableRows-1)

	// Column widths calculation is now handled by calculateResourceColumnWidths
	// Remove unused leftWidth variable since we're using proper column widths

	// Determine viewport based on Offset
	total := len(resources)
	cursor := m.state.Resources.Offset
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = max(0, total-1)
	}
	visibleRows := max(0, tableHeight)
	start := cursor - visibleRows/2
	if start < 0 {
		start = 0
	}
	if start > max(0, total-visibleRows) {
		start = max(0, total-visibleRows)
	}
	end := min(total, start+visibleRows)

    // Calculate proper column widths for a single-line table format
    // Derive kind and status widths from visible rows to give more room to name
    // 2 separators (spaces) between columns
    const sep = 2
    // Find maximum kind length among visible rows
    maxKind := 0
    maxStatus := 0
    for i := start; i < end; i++ {
        r := resources[i]
        if w := lipgloss.Width(r.Kind); w > maxKind { maxKind = w }
        hs := "Unknown"
        if r.Health != nil && r.Health.Status != nil { hs = *r.Health.Status }
        st := fmt.Sprintf("%s %s", m.getHealthIcon(hs), hs)
        if w := lipgloss.Width(st); w > maxStatus { maxStatus = w }
    }
    // Kind is exactly its widest value + 1 char
    kindWidth := min(maxKind+1, max(6, tableContentWidth/3))
    // Status width capped; right-aligned at end
    statusWidthCalc := min(max(6, maxStatus), 18)
    // Name gets the remainder
    nameWidth := max(10, tableContentWidth-kindWidth-statusWidthCalc-sep)

	// Build single-line header with proper column alignment
    kindHeader := padRight(headerStyle.Render("KIND"), kindWidth)
    nameHeader := padRight(headerStyle.Render("NAME"), nameWidth)
    statusHeader := padLeft(headerStyle.Render("STATUS"), statusWidthCalc)
    headerLine := fmt.Sprintf("%s %s%s", kindHeader, nameHeader, statusHeader)
	headerLine = clipAnsiToWidth(headerLine, tableContentWidth)

	// Build rows
	var b strings.Builder
	b.WriteString(headerLine)
	b.WriteString("\n")

	for i := start; i < end; i++ {
		r := resources[i]
		name := r.Name
		if r.Namespace != nil && *r.Namespace != "" {
			name = fmt.Sprintf("%s.%s", *r.Namespace, r.Name)
		}

		healthStatus := "Unknown"
		if r.Health != nil && r.Health.Status != nil {
			healthStatus = *r.Health.Status
		}

		// Single-line row: kind + name + status in proper columns
		kindText := truncateWithEllipsis(r.Kind, kindWidth)
		nameText := truncateWithEllipsis(name, nameWidth)
		statusText := fmt.Sprintf("%s %s", m.getHealthIcon(healthStatus), healthStatus)
		statusText = truncateWithEllipsis(statusText, statusWidthCalc)

		// Build the row with proper column alignment
		kindCell := padRight(kindText, kindWidth)
		nameCell := padRight(nameText, nameWidth)
        statusCell := m.getColorForStatus(healthStatus).Render(padLeft(statusText, statusWidthCalc))

        rowLine := fmt.Sprintf("%s %s%s", kindCell, nameCell, statusCell)
		rowLine = clipAnsiToWidth(rowLine, tableContentWidth)

		if i == cursor {
			rowLine = selectedStyle.Render(rowLine)
		}

		b.WriteString(rowLine)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Pad remaining lines to keep fixed height (1 line per item)
	// Only pad if we have space - don't exceed available rows
	usedLines := (end - start) + 1                              // +1 for header line
	if usedLines < visibleRows && usedLines < availableRows-4 { // Reserve space for title and footer
		for pad := usedLines; pad < min(visibleRows, availableRows-4); pad++ {
			b.WriteString("\n")
		}
	}

	// Footer
	visibleStart := start + 1
	visibleEnd := end
	footerText := fmt.Sprintf(
		"Showing %d-%d of %d resources • j/k to scroll • g/G jump • q to return",
		visibleStart, visibleEnd, total,
	)

	// Compose content; clip each section to inner content width
	var content strings.Builder
	title := fmt.Sprintf("Resources for %s", m.state.Resources.AppName)
	titleLine := clipAnsiToWidth(headerStyle.Render(title), tableContentWidth)
	tableBody := b.String()
	footerLine := clipAnsiToWidth(statusStyle.Render(footerText), tableContentWidth)

	content.WriteString(titleLine)
	content.WriteString("\n\n")
	content.WriteString(normalizeLinesToWidth(tableBody, tableContentWidth))
	content.WriteString("\n")
	content.WriteString(footerLine)

    // Render with standard inner padding so content matches other views visually
    resourcesBorder := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(magentaBright).
        PaddingLeft(1).
        PaddingRight(1).
        Width(tableContainerWidth)
    normalized := normalizeLinesToWidth(content.String(), tableContentWidth)
    return resourcesBorder.Render(normalized)
}

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
		// Wide layout: title on left (12 chars), content on right
		titlePadded := fmt.Sprintf("%-12s", titleStyled)
		return titlePadded + content
	} else {
		// Narrow layout: title above, content below
		return titleStyled + "\n" + content
	}
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}


// abbreviateStatus shortens status text for narrow displays
func abbreviateStatus(status string) string {
	switch status {
	case "Synced":
		return "Sync"
	case "OutOfSync":
		return "Out"
	case "Healthy":
		return "OK"
	case "Degraded":
		return "Bad"
	case "Progressing":
		return "Prog"
	case "Unknown":
		return "?"
	default:
		// If status is short already, return as-is
		if len(status) <= 4 {
			return status
		}
		// Otherwise truncate to 4 characters
		return status[:4]
	}
}

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
func calculateColumnWidths(availableWidth int) (nameWidth, syncWidth, healthWidth int) {
	// Account for separators between the 3 columns (2 separators, 1 char each)
	const sep = 2

	if availableWidth < 45 {
		// Very narrow: minimal widths (icons only)
		syncWidth = 2   // Just icon
		healthWidth = 2 // Just icon
		nameWidth = max(8, availableWidth-syncWidth-healthWidth-sep)
	} else {
		// Wide: full widths
		syncWidth = 12   // SYNC column
		healthWidth = 15 // HEALTH column
		nameWidth = max(10, availableWidth-syncWidth-healthWidth-sep)
	}

	// Make sure columns exactly fill the available width including separators
	totalUsed := nameWidth + syncWidth + healthWidth + sep
	if totalUsed < availableWidth {
		nameWidth += (availableWidth - totalUsed)
	} else if totalUsed > availableWidth {
		overflow := totalUsed - availableWidth
		nameWidth = max(1, nameWidth-overflow)
	}

	return nameWidth, syncWidth, healthWidth
}

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
	// Calculate dimensions for consistent full-height layout
	containerWidth := max(0, m.state.Terminal.Cols-2)
	contentWidth := max(0, containerWidth-4)          // Account for border and padding
	contentHeight := max(10, m.state.Terminal.Rows-6) // Reserve space for header/footer

	// Read actual log file content
	logContent := m.readLogContent()

	// Create single bordered box (no double border)
	logStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(magentaBright).
		Width(contentWidth).
		Height(contentHeight).
		AlignVertical(lipgloss.Top). // Align to top for log content
		PaddingLeft(1).
		PaddingRight(1)

	return logStyle.Render(logContent)
}

// readLogContent reads the actual log file content
func (m Model) readLogContent() string {
	// Try to read the log file that we write to in main.go
	logFile := "logs/a9s.log"
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
	// Calculate available space using the same pattern as other views
	header := m.renderBanner()
	headerLines := countLines(header)

	// Error view doesn't have search/command bars, so overhead is just banner + borders + status
	const BORDER_LINES = 2
	const STATUS_LINES = 1
	overhead := BORDER_LINES + headerLines + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)

    // Calculate dimensions for consistent full-height layout
    // Use full available width inside the main container
    containerWidth := max(0, m.state.Terminal.Cols-2) // main container has 1 char padding on each side
    contentHeight := max(3, availableRows)

	// Build error content
	errorContent := ""
	if m.state.CurrentError != nil {
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

	// Create a full-height bordered box directly to avoid double borders
    errorStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(outOfSyncColor).
        Width(containerWidth).
        Height(contentHeight).
        AlignVertical(lipgloss.Top).
        PaddingLeft(1).
        PaddingRight(1)

	styledErrorContent := errorStyle.Render(errorContent)

	// Combine with header
	var sections []string
	sections = append(sections, header)
	sections = append(sections, styledErrorContent)

	content := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	return mainContainerStyle.Height(totalHeight).Render(content)
}

// renderConnectionErrorView displays connection error in a user-friendly format
func (m Model) renderConnectionErrorView() string {
	// Calculate available space using the same pattern as other views
	header := m.renderBanner()
	headerLines := countLines(header)

	// Connection error view doesn't have search/command bars, so overhead is just banner + borders + status
	const BORDER_LINES = 2
	const STATUS_LINES = 1
	overhead := BORDER_LINES + headerLines + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)

    // Calculate dimensions for consistent full-height layout
    containerWidth := max(0, m.state.Terminal.Cols-2)
    contentHeight := max(3, availableRows)

	// Build connection error content
	errorContent := ""

	// Title with connection error styling
	titleStyle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true)
	errorContent += titleStyle.Render("Connection Error") + "\n\n"

	// Server info if available
	if m.state.Server != nil {
		serverStyle := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
		errorContent += fmt.Sprintf("ArgoCD Server: %s\n\n", serverStyle.Render(m.state.Server.BaseURL))
	}

	// Main error message
	messageStyle := lipgloss.NewStyle().Foreground(whiteBright)
	errorContent += messageStyle.Render("Unable to connect to ArgoCD server.\n\nPlease check that:\n• ArgoCD server is running\n• Network connection is available\n• Server URL and port are correct") + "\n\n"

	// Instructions
	instructStyle := lipgloss.NewStyle().Foreground(cyanBright)
	errorContent += instructStyle.Render("Press q to exit • Press Esc to retry")

    // Create a full-height bordered box directly to avoid double borders
    errorStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(outOfSyncColor).
        Width(containerWidth).
        Height(contentHeight).
        AlignVertical(lipgloss.Top).
        PaddingLeft(1).
        PaddingRight(1)

	styledErrorContent := errorStyle.Render(errorContent)

	// Combine with header
	var sections []string
	sections = append(sections, header)
	sections = append(sections, styledErrorContent)

	content := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	return mainContainerStyle.Height(totalHeight).Render(content)
}

// renderDiffLoadingSpinner displays a centered loading spinner for diff operations
func (m Model) renderDiffLoadingSpinner() string {
	// Create spinner content with message
	spinnerContent := fmt.Sprintf("%s Loading diff...", m.spinner.View())

	// Style the spinner with a small bordered box and semi-transparent background
	spinnerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		Background(lipgloss.Color("0")). // Dark background
		Foreground(whiteBright).
		Padding(1, 2).
		Bold(true).
		Align(lipgloss.Center)

	return spinnerStyle.Render(spinnerContent)
}

// renderSyncLoadingModal displays a compact centered modal with a spinner during sync start
func (m Model) renderSyncLoadingModal() string {
    msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Syncing…"))
    content := msg
    // Compact wrapper with cyan border to match confirm modal theme
    wrapper := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
        Padding(1, 2)
    // Width based on content, with a small minimum
    minW := 24
    w := max(minW, lipgloss.Width(content)+4)
    wrapper = wrapper.Width(w)
    return wrapper.Render(content)
}

// renderRollbackModal displays the rollback modal with deployment history
func (m Model) renderRollbackModal() string {
	// Calculate available space using the same pattern as other modals
	header := m.renderBanner()
	headerLines := countLines(header)

	// Rollback modal doesn't have search/command bars, so overhead is just banner + borders + status
	const BORDER_LINES = 2
	const STATUS_LINES = 1
	overhead := BORDER_LINES + headerLines + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)

    // Calculate dimensions for consistent full-height layout
    containerWidth := max(0, m.state.Terminal.Cols-2)
    contentHeight := max(3, availableRows)
    innerWidth := max(0, containerWidth-4) // 2 borders + 2 padding
    innerHeight := max(0, contentHeight-2) // no vertical padding set

	// Check if rollback state is available
	if m.state.Rollback == nil || m.state.Modals.RollbackAppName == nil {
		var content string
		if m.state.Modals.RollbackAppName == nil {
			content = "No app selected for rollback"
		} else {
			content = fmt.Sprintf("Loading deployment history for %s...\n\n%s", *m.state.Modals.RollbackAppName, m.spinner.View())
		}
		return m.renderSimpleModal("Rollback", content)
	}

	rollback := m.state.Rollback
	var modalContent string

    if rollback.Loading {
        // Loading state: if confirming, we're executing the rollback; otherwise we're loading history
        if rollback.Mode == "confirm" {
            modalContent = fmt.Sprintf("%s Executing rollback for %s...", m.spinner.View(), rollback.AppName)
        } else {
            modalContent = fmt.Sprintf("%s Loading deployment history for %s...", m.spinner.View(), *m.state.Modals.RollbackAppName)
        }
	} else if rollback.Error != "" {
		// Error state
		errorStyle := lipgloss.NewStyle().Foreground(outOfSyncColor)
		modalContent = errorStyle.Render(fmt.Sprintf("Error loading rollback history:\n%s", rollback.Error))
    } else if rollback.Mode == "confirm" {
        // Confirmation mode - render with bottom-aligned confirmation block
        modalContent = m.renderRollbackConfirmation(rollback, innerHeight, innerWidth)
	} else {
		// List mode - show deployment history
		modalContent = m.renderRollbackHistory(rollback)
	}

    // Add instructions only for list mode; confirmation view has inline keys
    if rollback.Mode != "confirm" {
        instructionStyle := lipgloss.NewStyle().Foreground(cyanBright)
        instructions := "j/k: Navigate • Enter: Select • Esc: Cancel"
        modalContent += "\n\n" + instructionStyle.Render(instructions)
    }

    // Create a full-height bordered box
    modalStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
        // Occupy full width of the main container
        Width(containerWidth).
        Height(contentHeight).
        AlignVertical(lipgloss.Top).
        PaddingLeft(1).
        PaddingRight(1)

    // Normalize each modal line to the inner content width to avoid wrapping,
    // then clip to available inner height to prevent vertical overflow.
    modalContent = normalizeLinesToWidth(modalContent, innerWidth)
    // Clip modal content to available inner height to prevent overflow.
    // Height() does not clip content; it only pads. Inner height is total minus borders.
    modalContent = clipAnsiToLines(modalContent, innerHeight)
    styledContent := modalStyle.Render(modalContent)

	// Combine with header
	var sections []string
	sections = append(sections, header)
	sections = append(sections, styledContent)

    content := strings.Join(sections, "\n")
    totalHeight := m.state.Terminal.Rows - 1
    // Clip final composed content to terminal height to ensure no overflow.
    content = clipAnsiToLines(content, totalHeight)
    return mainContainerStyle.Height(totalHeight).Render(content)
}

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
    titleStyle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true)
    content := titleStyle.Render("Confirm Rollback") + "\n\n"

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
    if innerWidth < 20 { innerWidth = 20 }
    center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)
    dim := lipgloss.NewStyle().Foreground(dimColor)
    on := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
    var opts strings.Builder
    opts.WriteString(dim.Render("[p] Prune: "))
    if rollback.Prune { opts.WriteString(on.Render("Yes")) } else { opts.WriteString(dim.Render("No")) }
    opts.WriteString(dim.Render("   [w] Watch: "))
    if rollback.Watch { opts.WriteString(on.Render("Yes")) } else { opts.WriteString(dim.Render("No")) }
    // Build inner confirmation modal (bordered) with title
    active := lipgloss.NewStyle().Background(magentaBright).Foreground(whiteBright).Bold(true).Padding(0, 2)
    inactive := lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(whiteBright).Padding(0, 2)
    yesBtn := inactive.Render("Yes")
    noBtn := inactive.Render("No")
    if rollback.ConfirmSelected == 0 { yesBtn = active.Render("Yes") }
    if rollback.ConfirmSelected == 1 { noBtn = active.Render("No") }
    buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, strings.Repeat(" ", 4), noBtn)

    confirmTitle := lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true).Render("Confirm Rollback")
    confirmInner := strings.Join([]string{
        center.Render(confirmTitle),
        "",
        center.Render(opts.String()),
        "",
        center.Render(buttons),
    }, "\n")

    // Make confirmation modal narrower than the outer box and center it
    confirmWidth := innerWidth - 8
    if confirmWidth < 30 {
        confirmWidth = max(24, innerWidth-4)
    }
    confirmStyled := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
        Padding(1, 2).
        Width(confirmWidth)
    confirmBox := center.Render(confirmStyled.Render(confirmInner))

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
func (m Model) renderSimpleModal(title, content string) string {
	header := m.renderBanner()
	headerLines := countLines(header)

	const BORDER_LINES = 2
	const STATUS_LINES = 1
	overhead := BORDER_LINES + headerLines + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)

	containerWidth := max(0, m.state.Terminal.Cols-2)
	contentWidth := max(0, containerWidth-4)
	contentHeight := max(3, availableRows)

	titleStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
	modalContent := titleStyle.Render(title) + "\n\n" + content

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Width(contentWidth).
		Height(contentHeight).
		AlignVertical(lipgloss.Top).
		PaddingLeft(1).
		PaddingRight(1)

	styledContent := modalStyle.Render(modalContent)

	var sections []string
	sections = append(sections, header)
	sections = append(sections, styledContent)

	content = strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	return mainContainerStyle.Height(totalHeight).Render(content)
}

// truncateString truncates a string to the specified length with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}
