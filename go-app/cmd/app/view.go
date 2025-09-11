package main

import (
    "fmt"
    "net/url"
    "strings"

    "github.com/charmbracelet/lipgloss"
    "github.com/a9s/go-app/pkg/model"
)

// Color mappings from TypeScript colorFor() function
var (
	// Color scheme matching React+Ink app
	magentaBright = lipgloss.Color("13")  // Selection highlight
	yellowBright  = lipgloss.Color("11")  // Headers
	dimColor      = lipgloss.Color("8")   // Dimmed text
	
	// Status colors (matching TypeScript colorFor function)
	syncedColor     = lipgloss.Color("10")  // Green for Synced/Healthy
	outOfSyncColor  = lipgloss.Color("9")   // Red for OutOfSync/Degraded  
	progressColor   = lipgloss.Color("11")  // Yellow for Progressing
    unknownColor    = lipgloss.Color("8")   // Dim for Unknown
    cyanBright      = lipgloss.Color("14")  // Cyan accents
    whiteBright     = lipgloss.Color("15")  // Bright white
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
	checkIcon  = "V"
	warnIcon   = "!"
	questIcon  = "?"
	deltaIcon  = "^"
	dotIcon    = "."
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
		return ""  // External mode returns null in React
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
    if searchBar != "" { sections = append(sections, searchBar) }
    if commandBar != "" { sections = append(sections, commandBar) }

	// Modal (matches MainLayout modal)
	if m.state.Mode == model.ModeConfirmSync {
		sections = append(sections, m.renderConfirmSyncModal())
	}

	// Main content area (matches MainLayout Box with border)
	if m.state.Mode == model.ModeResources && m.state.Server != nil && m.state.Modals.SyncViewApp != nil {
		sections = append(sections, m.renderResourceStream())
	} else {
		sections = append(sections, m.renderListView(listRows))
	}

	// Status line (matches MainLayout status Box)
	sections = append(sections, m.renderStatusLine())

	// Join with newlines and apply main container style with full width
	content := strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	totalWidth := m.state.Terminal.Cols
	
	return mainContainerStyle.Height(totalHeight).Width(totalWidth).Render(content)
}

// countLines returns the number of lines in a rendered string
func countLines(s string) int {
    if s == "" { return 0 }
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

        content := badge

        // Context block (stacked)
        ctx := m.renderContextBlock(true)
        return lipgloss.JoinVertical(lipgloss.Left, content, ctx)
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

    l1 := cyan.Render("   _____") + strings.Repeat(" ",43) + white.Render(" __   ")
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
    if len(rightLines) > n { n = len(rightLines) }

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
        if filler < 1 { filler = 1 }
        out = append(out, l+strings.Repeat(" ", filler)+r)
    }
    return strings.Join(out, "\n")
}

// renderListView - 1:1 mapping from ListView.tsx  
func (m Model) renderListView(availableRows int) string {
	visibleItems := m.getVisibleItems()
	
	// Calculate visible slice (exact copy from ListView.tsx)
	listRows := max(0, availableRows)
	selectedIdx := m.state.Navigation.SelectedIdx
	
	start := max(0, min(
		max(0, selectedIdx - listRows/2),
		max(0, len(visibleItems) - listRows),
	))
	end := min(len(visibleItems), start + listRows)
	
	if start >= len(visibleItems) {
		start = 0
		end = 0
	}
	
	rowsSlice := visibleItems[start:end]

	var content strings.Builder

	// Header row (matches ListView header)
	headerRow := m.renderListHeader()
	content.WriteString(headerRow)
	content.WriteString("\n")

	// Data rows (matches ListView map function)
	for i, item := range rowsSlice {
		actualIndex := start + i
		isCursor := actualIndex == selectedIdx
		
		if m.state.Navigation.View == model.ViewApps {
			row := m.renderAppRow(item.(model.App), isCursor)
			content.WriteString(row)
		} else {
			row := m.renderSimpleRow(fmt.Sprintf("%v", item), isCursor)
			content.WriteString(row)
		}
		
		if i < len(rowsSlice)-1 {
			content.WriteString("\n")
		}
	}

	// No items message (matches ListView empty state)
	if len(visibleItems) == 0 {
		content.WriteString(statusStyle.Render("No items."))
	}

	// Pad the list to consume all available rows so the table fills the space
	usedRows := len(rowsSlice)
	if len(visibleItems) == 0 {
		usedRows = 1 // the "No items." line uses one row
	}
	pad := max(0, listRows-usedRows)
	for i := 0; i < pad; i++ {
		content.WriteString("\n")
	}

	// Apply border style with full width (matches MainLayout content Box)
	contentWidth := max(0, m.state.Terminal.Cols - 4) // Account for main container padding
	// Set height to header (1) + listRows so border takes full vertical space
	return contentBorderStyle.Width(contentWidth).Height(1+listRows).Render(content.String())
}

// renderListHeader - matches ListView header row with responsive widths
func (m Model) renderListHeader() string {
    if m.state.Navigation.View == model.ViewApps {
        // Calculate responsive column widths based on terminal size
        contentWidth := max(0, m.state.Terminal.Cols-8) // Account for padding and borders
        syncWidth := 12   // Fixed width for SYNC column
        healthWidth := 15 // Fixed width for HEALTH column
        nameWidth := max(10, contentWidth-syncWidth-healthWidth-2) // Remaining space for NAME, minimum 10

        nameHeader := headerStyle.Render("NAME")
        syncHeader := headerStyle.Render("SYNC")
        healthHeader := headerStyle.Render("HEALTH")

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

    // Prepare texts and widths
    syncIcon := m.getSyncIcon(app.Sync)
    syncText := fmt.Sprintf("%s %s", syncIcon, app.Sync)
    healthIcon := m.getHealthIcon(app.Health)
    healthText := fmt.Sprintf("%s %s", healthIcon, app.Health)

    contentWidth := max(0, m.state.Terminal.Cols-8) // Account for padding and borders
    syncWidth := 12   // Fixed width for SYNC column
    healthWidth := 15 // Fixed width for HEALTH column
    nameWidth := max(10, contentWidth-syncWidth-healthWidth-2) // Remaining space for NAME, minimum 10

    var nameCell, syncCell, healthCell string
    if isCursor || isSelected {
        // Active row: avoid inner color styles so background highlight spans the whole row
        nameCell = padRight(app.Name, nameWidth)
        syncCell = padLeft(syncText, syncWidth)
        healthCell = padLeft(healthText, healthWidth)
    } else {
        // Inactive row: apply color styles to sync/health
        syncStyled := m.getColorForStatus(app.Sync).Render(syncText)
        healthStyled := m.getColorForStatus(app.Health).Render(healthText)
        nameCell = padRight(app.Name, nameWidth)
        syncCell = padLeft(syncStyled, syncWidth)
        healthCell = padLeft(healthStyled, healthWidth)
    }

    row := fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)

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
	
	// Apply selection highlight if active
	if active {
		return selectedStyle.Render(label)
	}
	
	return label
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
            if a.ClusterLabel != nil { cl = *a.ClusterLabel }
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
            if a.Namespace != nil { ns = *a.Namespace }
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
            if a.Project != nil { prj = *a.Project }
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
            if a.ClusterLabel != nil { cl = *a.ClusterLabel }
            if cl == "" { continue }
            if !seen[cl] {
                seen[cl] = true
                clusters = append(clusters, cl)
            }
        }
        sortStrings(clusters)
        for _, c := range clusters { base = append(base, c) }
    case model.ViewNamespaces:
        // Unique namespaces from apps filtered by clusters scope
        nss := make([]string, 0)
        seen := map[string]bool{}
        for _, a := range apps {
            var ns string
            if a.Namespace != nil { ns = *a.Namespace }
            if ns == "" { continue }
            if !seen[ns] { seen[ns] = true; nss = append(nss, ns) }
        }
        sortStrings(nss)
        for _, ns := range nss { base = append(base, ns) }
    case model.ViewProjects:
        // Unique projects from apps filtered by cluster+namespace scopes
        projs := make([]string, 0)
        seen := map[string]bool{}
        for _, a := range apps {
            var pj string
            if a.Project != nil { pj = *a.Project }
            if pj == "" { continue }
            if !seen[pj] { seen[pj] = true; projs = append(projs, pj) }
        }
        sortStrings(projs)
        for _, pj := range projs { base = append(base, pj) }
    case model.ViewApps:
        for _, app := range apps { base = append(base, app) }
    default:
        // No-op
    }

    // 3) Apply text filter or search
    filter := m.state.UI.ActiveFilter
    if m.state.Mode == model.ModeSearch {
        filter = m.state.UI.SearchQuery
    }
    f := strings.ToLower(filter)
    if f == "" { return base }

    filtered := make([]interface{}, 0, len(base))
    if m.state.Navigation.View == model.ViewApps {
        for _, it := range base {
            app := it.(model.App)
            name := strings.ToLower(app.Name)
            sync := strings.ToLower(app.Sync)
            health := strings.ToLower(app.Health)
            var ns, prj string
            if app.Namespace != nil { ns = strings.ToLower(*app.Namespace) }
            if app.Project != nil { prj = strings.ToLower(*app.Project) }
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
	
	// Main content with spinner (matches LoadingView center box)
    frames := []string{"⠋","⠙","⠹","⠸","⠼","⠴","⠦","⠧"}
    idx := m.spinnerFrame % len(frames)
    spinChar := frames[idx]
	loadingMessage := fmt.Sprintf("%s Connecting & fetching applications…", spinChar)
	
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
	
	return contentBorderStyle.Height(totalHeight).Render(content)
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
		targetText = fmt.Sprintf("%d", len(m.state.Selections.SelectedApps))
	} else {
		title = "Sync application?"
		message = "Do you want to sync"
		targetText = target
	}
	
	// Options (matches ConfirmSyncModal options)
	pruneStatus := "[ ]"
	if m.state.Modals.ConfirmSyncPrune {
		pruneStatus = "[x]"
	}
	
	watchStatus := "[ ]"
	if m.state.Modals.ConfirmSyncWatch {
		watchStatus = "[x]"
	}
	watchDisabled := isMulti
	
	var content strings.Builder
	content.WriteString(headerStyle.Render(title))
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("%s %s\n\n", message, targetText))
	content.WriteString(fmt.Sprintf("p) %s Prune\n", pruneStatus))
	if !watchDisabled {
		content.WriteString(fmt.Sprintf("w) %s Watch\n", watchStatus))
	} else {
		content.WriteString(fmt.Sprintf("w) %s Watch (disabled for multi)\n", watchStatus))
	}
	content.WriteString("\nEnter to confirm, Esc to cancel")
	
	return contentBorderStyle.Render(content.String())
}

func (m Model) renderResourceStream() string {
    return contentBorderStyle.Render("Resource stream - TODO: implement 1:1")
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

    // Compute viewport height: fill remaining space (reuse main container padding assumptions)
    totalHeight := m.state.Terminal.Rows - 2 // leave one row for title and one for status
    if totalHeight < 5 { totalHeight = 5 }
    // Clamp offset
    if m.state.Diff.Offset < 0 { m.state.Diff.Offset = 0 }
    if m.state.Diff.Offset > max(0, len(lines)-totalHeight) {
        m.state.Diff.Offset = max(0, len(lines)-totalHeight)
    }
    start := m.state.Diff.Offset
    end := min(len(lines), start+totalHeight)
    body := strings.Join(lines[start:end], "\n")

    title := headerStyle.Render(m.state.Diff.Title)
    status := statusStyle.Render(fmt.Sprintf("%d-%d/%d  j/k, g/G, / search, esc/q back", start+1, end, len(lines)))
    width := max(0, m.state.Terminal.Cols-4)
    return lipgloss.JoinVertical(lipgloss.Left,
        title,
        contentBorderStyle.Width(width).Height(totalHeight).Render(body),
        status,
    )
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
