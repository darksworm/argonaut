package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/a9s/go-app/pkg/model"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
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
	case model.ModeDiffLoading:
		return m.renderLoadingView()
	case model.ModeDiff:
		return m.renderDiffView()
	case model.ModeRulerLine:
		return m.renderOfficeSupplyManager()
	default:
		return m.renderMainLayout()
	}
}

// renderMainLayout - 1:1 mapping from MainLayout.tsx
func (m Model) renderMainLayout() string {
	// Height calculations - dynamic based on rendered section heights
	const (
		BORDER_LINES       = 2 // content border top/bottom
		TABLE_HEADER_LINES = 1 // list header row inside content
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
	totalHeight := m.state.Terminal.Rows - 1
	totalWidth := m.state.Terminal.Cols

	baseView := mainContainerStyle.Height(totalHeight).Width(totalWidth).Render(content)

	// Modal overlay (should overlay the base view, not push content down)
	if m.state.Mode == model.ModeConfirmSync {
		modal := m.renderConfirmSyncModal()
		// Center the modal in the available space
		centeredModal := lipgloss.Place(totalWidth, totalHeight, lipgloss.Center, lipgloss.Center, modal)
		return centeredModal
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
			Bold(true).Render(" Argonaut " + appVersion)

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

// renderListView - uses bubbles table for consistent styling across all views
func (m Model) renderListView(availableRows int) string {
	visibleItems := m.getVisibleItems()
	
	// Calculate content dimensions
	contentWidth := max(0, m.state.Terminal.Cols-12) // Aggressive padding to prevent overflow
	tableHeight := max(3, availableRows-2) // Reserve space for title if needed

	// Handle empty state
	if len(visibleItems) == 0 {
		emptyContent := statusStyle.Render("No items.")
		return contentBorderStyle.Width(contentWidth+12).Height(availableRows).Render(emptyContent)
	}

	// Prepare data and update the appropriate table directly
	var tableView string
	
	switch m.state.Navigation.View {
	case model.ViewApps:
		// Calculate responsive column widths for apps
		nameWidth, syncWidth, healthWidth := calculateColumnWidths(contentWidth)
		
		// Update column widths and headers based on available width
		var nameHeaderText, syncHeaderText, healthHeaderText string
		if contentWidth < 60 {
			nameHeaderText = "📱 NAME"
			syncHeaderText = "🔄"
			healthHeaderText = "💚"
		} else {
			nameHeaderText = "NAME"
			syncHeaderText = "SYNC"
			healthHeaderText = "HEALTH"
		}
		
		columns := []table.Column{
			{Title: nameHeaderText, Width: nameWidth},
			{Title: syncHeaderText, Width: syncWidth},
			{Title: healthHeaderText, Width: healthWidth},
		}
		m.appsTable.SetColumns(columns)
		
		// Convert apps to table rows with proper styling
		rows := make([]table.Row, len(visibleItems))
		for i, item := range visibleItems {
			app := item.(model.App)
			
			// Get icons and format text based on width
			syncIcon := m.getSyncIcon(app.Sync)
			healthIcon := m.getHealthIcon(app.Health)
			
			var syncText, healthText string
			if contentWidth < 45 {
				syncText = syncIcon
				healthText = healthIcon
			} else {
				syncText = fmt.Sprintf("%s %s", syncIcon, app.Sync)
				healthText = fmt.Sprintf("%s %s", healthIcon, app.Health)
			}
			
			rows[i] = table.Row{
				truncateWithEllipsis(app.Name, nameWidth),
				syncText,
				healthText,
			}
		}
		
		m.appsTable.SetRows(rows)
		m.appsTable.SetHeight(tableHeight)
		m.appsTable.SetWidth(contentWidth)
		
		// Set cursor with bounds checking
		if len(rows) > 0 {
			cursor := m.state.Navigation.SelectedIdx
			if cursor < 0 {
				cursor = 0
			}
			if cursor >= len(rows) {
				cursor = len(rows) - 1
			}
			m.appsTable.SetCursor(cursor)
		}
		tableView = m.appsTable.View()
		
	case model.ViewClusters:
		// Update column width to be responsive
		columns := []table.Column{
			{Title: "NAME", Width: contentWidth - 4}, // Full width minus some padding
		}
		m.clustersTable.SetColumns(columns)
		
		rows := make([]table.Row, len(visibleItems))
		for i, item := range visibleItems {
			rows[i] = table.Row{fmt.Sprintf("%v", item)}
		}
		
		m.clustersTable.SetRows(rows)
		m.clustersTable.SetHeight(tableHeight)
		m.clustersTable.SetWidth(contentWidth)
		
		// Set cursor with bounds checking
		if len(rows) > 0 {
			cursor := m.state.Navigation.SelectedIdx
			if cursor < 0 {
				cursor = 0
			}
			if cursor >= len(rows) {
				cursor = len(rows) - 1
			}
			m.clustersTable.SetCursor(cursor)
		}
		tableView = m.clustersTable.View()
		
	case model.ViewNamespaces:
		// Update column width to be responsive
		columns := []table.Column{
			{Title: "NAME", Width: contentWidth - 4}, // Full width minus some padding
		}
		m.namespacesTable.SetColumns(columns)
		
		rows := make([]table.Row, len(visibleItems))
		for i, item := range visibleItems {
			rows[i] = table.Row{fmt.Sprintf("%v", item)}
		}
		
		m.namespacesTable.SetRows(rows)
		m.namespacesTable.SetHeight(tableHeight)
		m.namespacesTable.SetWidth(contentWidth)
		
		// Set cursor with bounds checking
		if len(rows) > 0 {
			cursor := m.state.Navigation.SelectedIdx
			if cursor < 0 {
				cursor = 0
			}
			if cursor >= len(rows) {
				cursor = len(rows) - 1
			}
			m.namespacesTable.SetCursor(cursor)
		}
		tableView = m.namespacesTable.View()
		
	case model.ViewProjects:
		// Update column width to be responsive
		columns := []table.Column{
			{Title: "NAME", Width: contentWidth - 4}, // Full width minus some padding
		}
		m.projectsTable.SetColumns(columns)
		
		rows := make([]table.Row, len(visibleItems))
		for i, item := range visibleItems {
			rows[i] = table.Row{fmt.Sprintf("%v", item)}
		}
		
		m.projectsTable.SetRows(rows)
		m.projectsTable.SetHeight(tableHeight)
		m.projectsTable.SetWidth(contentWidth)
		
		// Set cursor with bounds checking
		if len(rows) > 0 {
			cursor := m.state.Navigation.SelectedIdx
			if cursor < 0 {
				cursor = 0
			}
			if cursor >= len(rows) {
				cursor = len(rows) - 1
			}
			m.projectsTable.SetCursor(cursor)
		}
		tableView = m.projectsTable.View()
		
	default:
		// Fallback to apps table
		m.appsTable.SetRows([]table.Row{})
		m.appsTable.SetHeight(tableHeight)
		m.appsTable.SetWidth(contentWidth)
		tableView = m.appsTable.View()
	}

	// Render the table
	var content strings.Builder
	content.WriteString(tableView)

	// Apply border style with proper dimensions
	return contentBorderStyle.Width(contentWidth+12).Height(availableRows).Render(content.String())
}

// renderListHeader - matches ListView header row with responsive widths
func (m Model) renderListHeader() string {
	if m.state.Navigation.View == model.ViewApps {
		// Calculate responsive column widths to match content box width
		// contentBorderStyle has left/right padding (2 chars) + border chars (2 chars) + main container padding (2 chars)
		contentWidth := max(0, m.state.Terminal.Cols-6) // Account for all padding and borders
		nameWidth, syncWidth, healthWidth := calculateColumnWidths(contentWidth)

		// Use emojis for narrow displays, full text for wide displays
		var nameHeaderText, syncHeaderText, healthHeaderText string
		if contentWidth < 60 {
			// Use emojis to save space
			nameHeaderText = "📱 NAME"
			syncHeaderText = "🔄"
			healthHeaderText = "💚"
		} else {
			// Full text for wide displays
			nameHeaderText = "NAME"
			syncHeaderText = "SYNC"
			healthHeaderText = "HEALTH"
		}

		nameHeader := headerStyle.Render(nameHeaderText)
		syncHeader := headerStyle.Render(syncHeaderText)
		healthHeader := headerStyle.Render(healthHeaderText)

		nameCell := padRight(nameHeader, nameWidth)
		syncCell := padLeft(syncHeader, syncWidth)
		healthCell := padLeft(healthHeader, healthWidth)

		return fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)
	}

	// Simple header for other views
	return headerStyle.Render("NAME")
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

	contentWidth := max(0, m.state.Terminal.Cols-6) // Account for all padding and borders to match header
	nameWidth, syncWidth, healthWidth := calculateColumnWidths(contentWidth)

	// Generate text based on available width (either full text or icons only)
	var syncText, healthText string
	if contentWidth < 45 {
		// Very narrow: use single character icons only
		syncText = syncIcon     // Just icon
		healthText = healthIcon // Just icon
	} else {
		// Wide: use full text (skip abbreviated step)
		syncText = fmt.Sprintf("%s %s", syncIcon, app.Sync)
		healthText = fmt.Sprintf("%s %s", healthIcon, app.Health)
	}

	// Truncate app name with ellipsis if it's too long
	truncatedName := truncateWithEllipsis(app.Name, nameWidth)

	var nameCell, syncCell, healthCell string
	if isCursor || isSelected {
		// Active row: avoid inner color styles so background highlight spans the whole row
		nameCell = padRight(truncatedName, nameWidth)
		syncCell = padLeft(syncText, syncWidth)
		healthCell = padLeft(healthText, healthWidth)
	} else {
		// Inactive row: apply color styles to sync/health
		syncStyled := m.getColorForStatus(app.Sync).Render(syncText)
		healthStyled := m.getColorForStatus(app.Health).Render(healthText)
		nameCell = padRight(truncatedName, nameWidth)
		syncCell = padLeft(syncStyled, syncWidth)
		healthCell = padLeft(healthStyled, healthWidth)
	}

	row := fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)

	// Ensure row spans full content width for proper highlighting
	fullRowWidth := nameWidth + syncWidth + healthWidth + 2 // +2 for separators
	if lipgloss.Width(row) < fullRowWidth {
		// Pad the row to full width so selection highlighting spans completely
		row = padRight(row, fullRowWidth)
	}

	// Apply selection highlight (matches ListView backgroundColor)
	if active {
		row = selectedStyle.Render(row)
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
	contentWidth := max(0, m.state.Terminal.Cols-6)

	// Truncate label if too long
	truncatedLabel := truncateWithEllipsis(label, contentWidth)

	// Apply selection highlight if active
	if active {
		return selectedStyle.Render(truncatedLabel)
	}

	return truncatedLabel
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

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		leftStyled,
		strings.Repeat(" ", max(0, m.state.Terminal.Cols-len(leftText)-len(rightText)-2)),
		rightStyled,
	)
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
	totalHeight := m.state.Terminal.Rows - 1
	totalWidth := max(0, m.state.Terminal.Cols-4) // Account for main container padding

	return contentBorderStyle.Width(totalWidth).Height(totalHeight).Render(content)
}

func (m Model) renderAuthRequiredView() string {
	serverText := "—"
	if m.state.Server != nil {
		serverText = m.state.Server.BaseURL
	}

	// Header message (matches AuthRequiredView.tsx)
	headerMsg := fmt.Sprintf("View: AUTH REQUIRED • Context: %s", serverText)

	// Instructions (matches AuthRequiredView.tsx instructions array)
	instructions := []string{
		"1. Run: argocd login <your-argocd-server>",
		"2. Follow prompts to authenticate",
		"3. Re-run argonaut",
	}

	var sections []string

	// ArgoNaut Banner (matches AuthRequiredView ArgoNautBanner)
	sections = append(sections, m.renderBanner())

	// Main content area with auth message (matches AuthRequiredView main Box)
	var contentSections []string

	// Center the content vertically
	contentSections = append(contentSections, "")
	contentSections = append(contentSections, lipgloss.NewStyle().
		Background(outOfSyncColor).
		Foreground(lipgloss.Color("15")).
		Bold(true).
		Render(" AUTHENTICATION REQUIRED "))
	contentSections = append(contentSections, "")
	contentSections = append(contentSections, lipgloss.NewStyle().
		Foreground(outOfSyncColor).
		Bold(true).
		Render("Please login to ArgoCD before running argonaut."))
	contentSections = append(contentSections, "")

	// Add instructions (matches AuthRequiredView instructions map)
	for _, instruction := range instructions {
		contentSections = append(contentSections, statusStyle.Render("- "+instruction))
	}
	contentSections = append(contentSections, "")
	contentSections = append(contentSections, statusStyle.Render("Current context: "+serverText))
	contentSections = append(contentSections, statusStyle.Render("Press l to view logs, q to quit."))

	// Apply border with red color (matches AuthRequiredView borderColor="red")
	authBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outOfSyncColor).
		PaddingLeft(2).
		PaddingRight(2).
		PaddingTop(1).
		PaddingBottom(1)

	authContent := authBoxStyle.Render(strings.Join(contentSections, "\n"))
	sections = append(sections, authContent)

	// Status line (matches AuthRequiredView bottom Box)
	statusLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		statusStyle.Render(headerMsg),
		strings.Repeat(" ", max(0, m.state.Terminal.Cols-len(headerMsg)-6)),
		statusStyle.Render("Ready"),
	)
	sections = append(sections, statusLine)

	// Join with newlines and apply main container style with full width
	content := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	totalWidth := m.state.Terminal.Cols

	return mainContainerStyle.Height(totalHeight).Width(totalWidth).Render(content)
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
	actionsContent := ":diff [app] • :sync [app] • :rollback [app]\n:up go up level\ns sync modal (apps view)"
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

func (m Model) renderRollbackModal() string {
	// 1:1 mapping from RollbackModal.tsx
	if m.state.Modals.RollbackAppName == nil {
		return ""
	}

	var sections []string

	// ArgoNaut Banner (matches RollbackModal ArgoNautBanner)
	sections = append(sections, m.renderBanner())

	// Rollback content placeholder (would integrate with Rollback component)
	rollbackContent := fmt.Sprintf("Rollback Application: %s\n\nThis would show rollback history and options.\n\nPress Esc to close.", *m.state.Modals.RollbackAppName)
	sections = append(sections, contentBorderStyle.Render(rollbackContent))

	content := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1

	return mainContainerStyle.Height(totalHeight).Render(content)
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
	// 1:1 mapping from ConfirmSyncModal.tsx
	if m.state.Modals.ConfirmTarget == nil {
		return ""
	}

	target := *m.state.Modals.ConfirmTarget
	isMulti := target == "__MULTI__"

	// Modal content matching ConfirmationBox
	var title, message, targetText string
	if isMulti {
		title = "Sync applications?"
		message = "Do you want to sync"
		targetText = fmt.Sprintf("%d selected apps", len(m.state.Selections.SelectedApps))
	} else {
		title = "Sync application?"
		message = "Do you want to sync"
		targetText = target
	}

	// Options (matches ConfirmSyncModal options)
	pruneStatus := "[ ]"
	if m.state.Modals.ConfirmSyncPrune {
		pruneStatus = "[×]"
	}

	watchStatus := "[ ]"
	if m.state.Modals.ConfirmSyncWatch {
		watchStatus = "[×]"
	}
	watchDisabled := isMulti

	// Create modal style that matches TypeScript version
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		Background(lipgloss.Color("0")).
		Foreground(whiteBright).
		PaddingLeft(2).
		PaddingRight(2).
		PaddingTop(1).
		PaddingBottom(1).
		Width(50). // Fixed width like the TypeScript version
		Align(lipgloss.Center)

	var content strings.Builder

	// Title styling
	titleStyled := lipgloss.NewStyle().
		Bold(true).
		Foreground(yellowBright).
		Align(lipgloss.Center).
		Render(title)
	content.WriteString(titleStyled)
	content.WriteString("\n\n")

	// Message and target
	messageStyled := lipgloss.NewStyle().
		Foreground(whiteBright).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("%s %s", message, targetText))
	content.WriteString(messageStyled)
	content.WriteString("\n\n")

	// Options with better styling
	optionStyle := lipgloss.NewStyle().Foreground(cyanBright)
	content.WriteString(optionStyle.Render(fmt.Sprintf("p) %s Prune", pruneStatus)))
	content.WriteString("\n")

	if !watchDisabled {
		content.WriteString(optionStyle.Render(fmt.Sprintf("w) %s Watch", watchStatus)))
	} else {
		dimStyle := lipgloss.NewStyle().Foreground(dimColor)
		content.WriteString(dimStyle.Render(fmt.Sprintf("w) %s Watch (disabled for multi)", watchStatus)))
	}
	content.WriteString("\n\n")

	// Instructions
	instructionStyle := lipgloss.NewStyle().
		Foreground(dimColor).
		Align(lipgloss.Center)
	content.WriteString(instructionStyle.Render("Enter to confirm • Esc to cancel"))

	return modalStyle.Render(content.String())
}

func (m Model) renderResourceStream(availableRows int) string {
	if m.state.Resources == nil {
		return contentBorderStyle.Render("Loading resources...")
	}

	if m.state.Resources.Error != "" {
		errorContent := fmt.Sprintf("Error loading resources:\n%s\n\nPress q to return", m.state.Resources.Error)
		return contentBorderStyle.Render(errorContent)
	}

	if m.state.Resources.Loading {
		loadingContent := fmt.Sprintf("Loading resources for %s...\n\nPress q to return", m.state.Resources.AppName)
		return contentBorderStyle.Render(loadingContent)
	}

	resources := m.state.Resources.Resources
	if len(resources) == 0 {
		emptyContent := fmt.Sprintf("No resources found for application: %s\n\nPress q to return", m.state.Resources.AppName)
		return contentBorderStyle.Render(emptyContent)
	}

	// Calculate content dimensions matching main layout pattern
	contentWidth := max(0, m.state.Terminal.Cols-4) // Account for main container padding

	// Create table rows from resources data
	rows := make([]table.Row, len(resources))
	for i, resource := range resources {
		name := resource.Name
		if resource.Namespace != nil && *resource.Namespace != "" {
			name = fmt.Sprintf("%s.%s", *resource.Namespace, resource.Name)
		}

		healthStatus := "Unknown"
		if resource.Health != nil && resource.Health.Status != nil {
			healthStatus = *resource.Health.Status
		}

		rows[i] = table.Row{
			resource.Kind,
			name,
			healthStatus,
		}
	}

	// Create a local copy of the resources table to modify
	resourcesTable := m.resourcesTable
	
	// Update the table with new data
	resourcesTable.SetRows(rows)
	
	// Calculate column widths based on available width - account for table borders and padding
	// Bubbles table adds its own border/padding, so give it much less width to prevent overflow
	tableWidth := max(0, contentWidth-12) // Very aggressive padding to prevent header overflow
	kindWidth, nameWidth, statusWidth := calculateResourceColumnWidths(tableWidth)
	
	// Update table column widths
	columns := []table.Column{
		{Title: "KIND", Width: kindWidth},
		{Title: "NAME", Width: nameWidth},
		{Title: "STATUS", Width: statusWidth},
	}
	resourcesTable.SetColumns(columns)

	// Set table dimensions to match available space
	tableHeight := max(3, availableRows-4) // Reserve space for title and footer
	resourcesTable.SetHeight(tableHeight)
	resourcesTable.SetWidth(tableWidth)

	// Apply table styles with header border (should fit now with reduced width)
	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true). // Re-enable header border with reduced table width
		Bold(false)
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	resourcesTable.SetStyles(tableStyle)

	// Handle scrolling by moving the table cursor to the correct position
	if m.state.Resources.Offset > 0 && len(rows) > 0 {
		// Set cursor position to simulate scrolling
		targetPos := min(m.state.Resources.Offset, len(rows)-1)
		resourcesTable.SetCursor(targetPos)
	}

	// Create content with title and table
	var content strings.Builder
	title := fmt.Sprintf("Resources for %s", m.state.Resources.AppName)
	content.WriteString(headerStyle.Render(title))
	content.WriteString("\n\n")
	
	// Render the bubbles table
	content.WriteString(resourcesTable.View())
	content.WriteString("\n")

	// Footer with navigation info - calculate visible range properly
	totalResources := len(resources)
	currentCursor := resourcesTable.Cursor()
	visibleStart := currentCursor + 1
	visibleEnd := min(totalResources, currentCursor + tableHeight)
	
	footerText := fmt.Sprintf("Showing %d-%d of %d resources • j/k to scroll • g/G jump • q to return",
		visibleStart, visibleEnd, totalResources)
	content.WriteString(statusStyle.Render(footerText))

	// Apply border style with proper dimensions
	return contentBorderStyle.Width(contentWidth).Height(availableRows).Render(content.String())
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

func min(a, b int) int {
	if a < b {
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
	if availableWidth < 45 {
		// Very narrow: minimal widths (icons only)
		syncWidth = 2                                              // Just icon
		healthWidth = 2                                            // Just icon
		nameWidth = max(8, availableWidth-syncWidth-healthWidth-2) // -2 for column separators
	} else {
		// Wide: full widths (keep full text, skip abbreviated step)
		syncWidth = 12   // Full width for SYNC column
		healthWidth = 15 // Full width for HEALTH column
		nameWidth = max(10, availableWidth-syncWidth-healthWidth-2)
	}

	// Ensure we use the full width - distribute any remaining space to nameWidth
	totalUsed := nameWidth + syncWidth + healthWidth + 2 // +2 for separators
	if totalUsed < availableWidth {
		nameWidth += (availableWidth - totalUsed)
	}

	return nameWidth, syncWidth, healthWidth
}

// calculateResourceColumnWidths returns responsive column widths for resources table
func calculateResourceColumnWidths(availableWidth int) (kindWidth, nameWidth, statusWidth int) {
	if availableWidth < 45 {
		// Very narrow: minimal widths
		kindWidth = 8                                             // KIND column
		statusWidth = 8                                           // STATUS column
		nameWidth = max(10, availableWidth-kindWidth-statusWidth-4) // -4 for separators and padding
	} else {
		// Wide: full widths
		kindWidth = 20    // KIND column
		statusWidth = 15  // STATUS column
		nameWidth = max(15, availableWidth-kindWidth-statusWidth-4)
	}

	// Ensure we use the full width - distribute any remaining space to nameWidth
	totalUsed := kindWidth + nameWidth + statusWidth + 4 // +4 for separators and padding
	if totalUsed < availableWidth {
		nameWidth += (availableWidth - totalUsed)
	}

	return kindWidth, nameWidth, statusWidth
}
