package main

import (
	"fmt"
	"image/color"
	"os"
	"regexp"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/sort"
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

	// Additional colors for modals
	black    = lipgloss.Color("0")  // Black
	white    = lipgloss.Color("15") // White (alias for whiteBright)
	redColor = lipgloss.Color("9")  // Red
)

// HighlightLogLine applies syntax highlighting to a single log line
func HighlightLogLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}

	// Use a more sophisticated parser that handles quoted strings
	parts, err := parseLogLineParts(line)
	if err != nil || len(parts) < 3 {
		return line // Fallback to original line if parsing fails
	}

	var highlighted strings.Builder
	partIndex := 0

	// Try to identify timestamp (first part that looks like a timestamp)
	if partIndex < len(parts) && looksLikeTimestamp(parts[partIndex]) {
		highlighted.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render(parts[partIndex]))
		highlighted.WriteString(" ")
		partIndex++
	}

	// Try to identify time (second part that looks like HH:MM:SS)
	if partIndex < len(parts) && looksLikeTime(parts[partIndex]) {
		highlighted.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render(parts[partIndex]))
		highlighted.WriteString(" ")
		partIndex++
	}

	// Try to identify log level
	if partIndex < len(parts) && looksLikeLogLevel(parts[partIndex]) {
		var style lipgloss.Style
		switch strings.ToUpper(parts[partIndex]) {
		case "DEBUG", "TRACE":
			style = lipgloss.NewStyle().Foreground(magentaBright).Bold(true) // magenta
		case "INFO":
			style = lipgloss.NewStyle().Foreground(cyanBright).Bold(true) // blue
		case "WARN", "WARNING":
			style = lipgloss.NewStyle().Foreground(yellowBright).Bold(true) // yellow
		case "ERROR", "FATAL":
			style = lipgloss.NewStyle().Foreground(outOfSyncColor).Bold(true) // red
		default:
			style = lipgloss.NewStyle().Foreground(whiteBright).Bold(true) // white
		}
		highlighted.WriteString(style.Render(parts[partIndex]))
		highlighted.WriteString(" ")
		partIndex++
	}

	// Process remaining parts
	for partIndex < len(parts) {
		part := parts[partIndex]

		// Check if it's a key=value pair
		if strings.Contains(part, "=") {
			// Split on first = only
			eqIndex := strings.Index(part, "=")
			if eqIndex > 0 {
				key := part[:eqIndex]
				value := part[eqIndex+1:]

				// Remove quotes from value if present
				value = strings.Trim(value, `"`)

				highlighted.WriteString(lipgloss.NewStyle().Foreground(cyanBright).Render(key))   // cyan for field names
				highlighted.WriteString(lipgloss.NewStyle().Foreground(dimColor).Render("="))    // dim for equals
				highlighted.WriteString(lipgloss.NewStyle().Foreground(whiteBright).Render(value)) // white for values
			} else {
				// Not a proper key=value, just render as is
				highlighted.WriteString(part)
			}
		} else {
			// Check if this looks like a component name (no spaces, no special chars)
			if isLikelyComponent(part) {
				highlighted.WriteString(lipgloss.NewStyle().Foreground(syncedColor).Render(part)) // green for components
			} else {
				// Regular text
				highlighted.WriteString(lipgloss.NewStyle().Foreground(whiteBright).Render(part)) // white for regular text
			}
		}

		if partIndex < len(parts)-1 {
			highlighted.WriteString(" ")
		}
		partIndex++
	}

	return highlighted.String()
}

// parseLogLineParts parses a log line into parts, properly handling quoted strings
func parseLogLineParts(line string) ([]string, error) {
	var parts []string
	var current strings.Builder
	inQuotes := false
	quoteChar := byte(0)

	for i := 0; i < len(line); i++ {
		char := line[i]

		switch {
		case !inQuotes && (char == '"' || char == '\''):
			// Start of quoted string
			inQuotes = true
			quoteChar = char
			current.WriteByte(char)
		case inQuotes && char == quoteChar:
			// End of quoted string
			inQuotes = false
			current.WriteByte(char)
		case inQuotes:
			// Inside quoted string, include everything
			current.WriteByte(char)
		case !inQuotes && char == ' ':
			// Space separator
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			// Regular character
			current.WriteByte(char)
		}
	}

	// Add the last part if any
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts, nil
}

// looksLikeTimestamp checks if a string resembles a timestamp
func looksLikeTimestamp(s string) bool {
	// Match patterns like: 2024/01/15, 10:30:45, 2024-01-15T10:30:45, etc.
	timestampPatterns := []string{
		`^\d{4}/\d{2}/\d{2}$`,                  // 2024/01/15
		`^\d{2}:\d{2}:\d{2}`,                   // 10:30:45
		`^\d{4}-\d{2}-\d{2}`,                   // 2024-01-15
		`^\d{4}/\d{2}/\d{2}T\d{2}:\d{2}:\d{2}`, // 2024/01/15T10:30:45
	}

	for _, pattern := range timestampPatterns {
		if matched, _ := regexp.MatchString(pattern, s); matched {
			return true
		}
	}
	return false
}

// looksLikeLogLevel checks if a string is a log level
func looksLikeLogLevel(s string) bool {
	levels := []string{"DEBUG", "INFO", "WARN", "WARNING", "ERROR", "FATAL", "TRACE"}
	s = strings.ToUpper(s)
	for _, level := range levels {
		if s == level {
			return true
		}
	}
	return false
}

// looksLikeTime checks if a string resembles a time (HH:MM:SS)
func looksLikeTime(s string) bool {
	timePattern := `^\d{2}:\d{2}:\d{2}$`
	matched, _ := regexp.MatchString(timePattern, s)
	return matched
}

// isLikelyComponent checks if a string looks like a component name
func isLikelyComponent(s string) bool {
	// Component names typically contain letters, numbers, underscores, dots
	// No spaces, no special characters except underscore and dot
	if strings.ContainsAny(s, " \t\n\r\"'()[]{}<>,;:!@#$%^&*+-=|\\") {
		return false
	}
	// Should have at least one letter
	hasLetter := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			hasLetter = true
			break
		}
	}
	return hasLetter && len(s) > 1 && len(s) < 50
}

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
	// Cursor sitting on a selected row should stand out
	cursorOnSelectedStyle = lipgloss.NewStyle().
				Background(cyanBright)
	// Flash highlight for refresh feedback
	refreshFlashStyle = lipgloss.NewStyle().
			Background(syncedColor)

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
func (m *Model) View() tea.View {
	m.renderCount++
	cblog.With("component", "view").Debug("View() called",
		"render_count", m.renderCount,
		"mode", m.state.Mode,
		"view", m.state.Navigation.View,
		"apps_count", len(m.state.Apps))

	var content string
	// Don't show plain "Starting..." - let renderMainLayout handle the loading modal
	if !m.ready && m.state.Mode != model.ModeNormal {
		content = statusStyle.Render("Starting…")
	} else {
		// Map React App.tsx switch statement exactly
		switch m.state.Mode {
		case model.ModeLoading:
			// Show regular layout with the initial loading modal overlay instead of a separate loading view
			content = m.renderMainLayout()
		case model.ModeAuthRequired:
			content = m.renderAuthRequiredView()
		case model.ModeHelp:
			content = m.renderHelpModal()
		case model.ModeRollback:
			content = m.renderRollbackModal()
		case model.ModeConfirmAppDelete:
			content = m.renderMainLayout()
		case model.ModeExternal:
			content = ""
		case model.ModeDiff:
			content = m.renderDiffView()
		case model.ModeRulerLine:
			content = m.renderOfficeSupplyManager()
		case model.ModeError:
			content = m.renderErrorView()
		case model.ModeConnectionError:
			content = m.renderConnectionErrorView()
		case model.ModeCoreDetected:
			content = m.renderCoreDetectedView()
		case model.ModeLogin, model.ModeLoginLoading:
			content = m.renderLoginView()
		default:
			content = m.renderMainLayout()
		}
	}

	// Store plain text content for text selection extraction
	m.storeRenderedContent(content)

	// Apply selection highlighting if there's an active selection
	content = m.applySelectionHighlight(content)

	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion // Track all motion for better selection support
	return v
}

// countLines returns the number of lines in a rendered string
func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// FullScreenViewOptions configures the full-screen layout
type FullScreenViewOptions struct {
	ContentBordered bool
	BorderColor     color.Color // Optional: override border color (defaults to magentaBright)
}

// renderFullScreenViewWithOptions provides the full-screen layout with customizable options
func (m *Model) renderFullScreenViewWithOptions(header, content, status string, opts FullScreenViewOptions) string {
	var sections []string

	// Header section
	if header != "" {
		sections = append(sections, header)
	}

	// Content section - apply border if requested
	if opts.ContentBordered {
		// Calculate available space for bordered content
		// lipgloss Height() sets total visual height including borders
		headerLines := countLines(header)
		statusLines := countLines(status)
		overhead := headerLines + statusLines
		availableRows := max(1, m.state.Terminal.Rows-overhead)

		// Apply bordered styling with custom color if specified
		contentWidth := max(0, m.state.Terminal.Cols-2) // Adjusted to fill space properly
		borderStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(opts.BorderColor).
			Width(contentWidth).
			Height(availableRows).
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
	totalHeight := m.state.Terminal.Rows
	return mainContainerStyle.Height(totalHeight).Render(finalContent)
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

// Regex to detect background color codes in ANSI sequences
// Matches: 4X (basic bg), 10X (bright bg), 48;5;X (256-color bg), 48;2;R;G;B (truecolor bg)
var bgColorRE = regexp.MustCompile("\x1b\\[(?:[0-9;]*;)?(?:4[0-7]|10[0-7]|48;[25];[0-9;]+)m")

// desaturateANSI strips ANSI color/style codes and recolors text.
// Lines with background colors are preserved as-is (they represent selected items).
// Other lines are dimmed.
func desaturateANSI(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if bgColorRE.MatchString(line) {
			// Line has background color - preserve it as-is (selected item)
			// Don't modify - keep the original styling from tree view
			continue
		}
		// Regular line - dim it
		plain := ansiRE.ReplaceAllString(line, "")
		lines[i] = lipgloss.NewStyle().Foreground(dimColor).Render(plain)
	}
	return strings.Join(lines, "\n")
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

func (m *Model) getVisibleItems() []interface{} {
	// Derive unique groups and filtered apps from current state, mirroring TS useVisibleItems
	// 1) Gather filtered apps through selected scopes
	apps := m.state.Apps
	cblog.With("component", "view").Debug("getVisibleItems called",
		"total_apps", len(apps),
		"first_app", func() string {
			if len(apps) > 0 {
				return fmt.Sprintf("%s (health=%s, sync=%s)", apps[0].Name, apps[0].Health, apps[0].Sync)
			}
			return "no apps"
		}())

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

	// Filter by ApplicationSet scope
	if len(m.state.Selections.ScopeApplicationSets) > 0 {
		filtered := make([]model.App, 0, len(apps))
		for _, a := range apps {
			if a.ApplicationSet != nil && model.HasInStringSet(m.state.Selections.ScopeApplicationSets, *a.ApplicationSet) {
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
	case model.ViewApplicationSets:
		// Unique ApplicationSet names from all apps
		appsets := make([]string, 0)
		seen := map[string]bool{}
		for _, a := range m.state.Apps {
			if a.ApplicationSet == nil || *a.ApplicationSet == "" {
				continue
			}
			if !seen[*a.ApplicationSet] {
				seen[*a.ApplicationSet] = true
				appsets = append(appsets, *a.ApplicationSet)
			}
		}
		sortStrings(appsets)
		for _, as := range appsets {
			base = append(base, as)
		}
	case model.ViewApps:
		// Ensure consistent, stable ordering without mutating state
		appsCopy := make([]model.App, len(apps))
		copy(appsCopy, apps)
		sort.SortApps(appsCopy, m.state.UI.Sort)
		for _, app := range appsCopy {
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

func (m *Model) renderAuthRequiredView() string {
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
		Foreground(textOnDanger).
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

func (m *Model) renderOfficeSupplyManager() string {
	return statusStyle.Render("Office supply manager - TODO: implement 1:1")
}
func (m *Model) renderConfirmSyncModal() string {
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
	inactiveFG := ensureContrastingForeground(inactiveBG, whiteBright)
	active := lipgloss.NewStyle().Background(magentaBright).Foreground(textOnAccent).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(inactiveBG).Foreground(inactiveFG).Padding(0, 2)
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

// renderDiffView - simple pager for diff content
func (m *Model) renderDiffView() string {
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
func (m *Model) renderHelpSection(title, content string, isWide bool) string {
	titleStyled := lipgloss.NewStyle().Foreground(syncedColor).Bold(true).Render(title)
	if isWide {
		// Two-column layout: 12-char title column + 1 space gap
		const col = 12
		// Pad the title visually to width 'col'
		padRightVisual := func(s string, w int) string {
			diff := w - lipgloss.Width(s)
			if diff > 0 {
				return s + strings.Repeat(" ", diff)
			}
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

// readLogContent reads the actual log file content
func (m *Model) readLogContent() string {
	// Try to read the log file path from environment (set by setupLogging)
	logFile := os.Getenv("ARGONAUT_LOG_FILE")
	if strings.TrimSpace(logFile) == "" {
		return "ArgoCD Application Logs\n\nNo log file available.\n\nPress q to return to main view."
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

	// Apply syntax highlighting to each log line
	lines := strings.Split(logText, "\n")
	var highlightedLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			highlightedLines = append(highlightedLines, HighlightLogLine(line))
		} else {
			highlightedLines = append(highlightedLines, line)
		}
	}
	highlightedLogText := strings.Join(highlightedLines, "\n")

	// Add header and instructions
	header := "ArgoCD Application Logs\n\nPress q to return to main view.\n\n"
	return header + "--- Log Content ---\n\n" + highlightedLogText
}

// renderErrorView displays API errors in a user-friendly format
func (m *Model) renderErrorView() string {
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
	instructions := []string{
		"Press Esc to return to main view",
		"Press 'l' to view system logs",
	}
	errorContent += fmt.Sprintf("\n%s", instructStyle.Render(strings.Join(instructions, " | ")))

	// Status (empty for error views)
	status := ""

	// Use the new layout helper with red border (matching error styling)
	return m.renderFullScreenViewWithOptions(header, errorContent, status, FullScreenViewOptions{
		ContentBordered: true,
		BorderColor:     outOfSyncColor, // red border for errors
	})
}

// renderConnectionErrorView displays connection error in a user-friendly format
func (m *Model) renderConnectionErrorView() string {
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
	instructions := []string{
		"Press 'q' to exit",
		"Press 'l' to view system logs",
		"Press Esc to retry",
	}
	errorContent += instructStyle.Render(strings.Join(instructions, " | "))

	// Status (empty for error views)
	status := ""

	// Use the new layout helper with red border (matching connection error styling)
	return m.renderFullScreenViewWithOptions(header, errorContent, status, FullScreenViewOptions{
		ContentBordered: true,
		BorderColor:     outOfSyncColor, // red border for connection errors
	})
}

// renderCoreDetectedView displays helpful instructions for ArgoCD core installations
func (m *Model) renderCoreDetectedView() string {
	// Header
	header := m.renderBanner()

	// Build core detection content - more compact version
	var contentSections []string

	// Title with warning styling
	titleStyle := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
	contentSections = append(contentSections, titleStyle.Render("ArgoCD Core Installation Detected"))
	contentSections = append(contentSections, "")

	// Main explanation message (shortened)
	messageStyle := lipgloss.NewStyle().Foreground(whiteBright)
	contentSections = append(contentSections, messageStyle.Render("Core mode doesn't include the API server required by argonaut. As a workaround, you can run the dashboard locally:"))
	contentSections = append(contentSections, "")

	// Step-by-step commands (more compact)
	codeStyle := lipgloss.NewStyle().Foreground(syncedColor)
	commentStyle := lipgloss.NewStyle().Foreground(dimColor)

	// Combine steps more compactly
	contentSections = append(contentSections, commentStyle.Render("# 1. Get admin password"))
	contentSections = append(contentSections, codeStyle.Render(`ADMIN_PASS="$(kubectl -n argocd get secret argocd-initial-admin-secret \`))
	contentSections = append(contentSections, codeStyle.Render(`  -o jsonpath="{.data.password}" | base64 -d)"`))
	contentSections = append(contentSections, "")

	contentSections = append(contentSections, commentStyle.Render("# 2. Start dashboard & login"))
	contentSections = append(contentSections, codeStyle.Render("argocd admin dashboard --namespace argocd --port 8080 &"))
	contentSections = append(contentSections, codeStyle.Render(`argocd login localhost:8080 --insecure --username admin --password $ADMIN_PASS`))
	contentSections = append(contentSections, "")

	contentSections = append(contentSections, commentStyle.Render("# 3. Run argonaut"))
	contentSections = append(contentSections, codeStyle.Render("argonaut"))

	// Join content
	errorContent := strings.Join(contentSections, "\n")

	// Status with instructions
	status := statusStyle.Render("Press q or Esc to exit")

	// Use the new layout helper with yellow border (warning styling)
	return m.renderFullScreenViewWithOptions(header, errorContent, status, FullScreenViewOptions{
		ContentBordered: true,
		BorderColor:     yellowBright, // yellow border for warnings
	})
}

// renderRollbackHistory renders the deployment history list
func (m *Model) renderRollbackHistory(rollback *model.RollbackState) string {
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
	const MARGIN_TOP_LINES = 1 // blank line between header and box
	availableRows := max(0, m.state.Terminal.Rows-(BORDER_LINES+headerLines+STATUS_LINES+MARGIN_TOP_LINES))

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
			// Truncate commit message to single line to prevent overflow
			message := strings.ReplaceAll(*row.Message, "\n", " ")
			message = strings.ReplaceAll(message, "\r", " ")
			message = truncateWithEllipsis(message, 60)
			line += fmt.Sprintf(" %s: %s",
				authorStyle.Render(*row.Author),
				messageStyle.Render(message))
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
func (m *Model) renderRollbackConfirmation(rollback *model.RollbackState, innerHeight int, innerWidth int) string {
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
	inactiveFG := ensureContrastingForeground(inactiveBG, whiteBright)
	active := lipgloss.NewStyle().Background(magentaBright).Foreground(textOnAccent).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(inactiveBG).Foreground(inactiveFG).Padding(0, 2)
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
