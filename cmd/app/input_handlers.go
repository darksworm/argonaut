package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/kubeconfig"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// Navigation handlers matching TypeScript functionality

// handleNavigationUp moves cursor up with bounds checking
func (m *Model) handleNavigationUp() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View == model.ViewTree {
		return m, nil // Tree view handles its own navigation
	}

	m.listNav.SetItemCount(len(m.getVisibleItemsForCurrentView()))
	m.listNav.SetViewportHeight(m.listViewportHeight())
	m.listNav.MoveUp()
	m.state.Navigation.SelectedIdx = m.listNav.Cursor()

	return m, nil
}

// handleNavigationDown moves cursor down with bounds checking
func (m *Model) handleNavigationDown() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View == model.ViewTree {
		return m, nil // Tree view handles its own navigation
	}

	m.listNav.SetItemCount(len(m.getVisibleItemsForCurrentView()))
	m.listNav.SetViewportHeight(m.listViewportHeight())
	m.listNav.MoveDown()
	m.state.Navigation.SelectedIdx = m.listNav.Cursor()

	return m, nil
}

// treeViewportHeight computes the number of rows available to render the
// tree panel body, mirroring the layout math in renderMainLayout/renderTreePanel.
func (m *Model) treeViewportHeight() int {
	const (
		BORDER_LINES       = 2
		TABLE_HEADER_LINES = 0
		TAG_LINE           = 0
		STATUS_LINES       = 1
	)
	header := m.renderBanner()
	headerLines := countLines(header)
	searchLines := 0
	if m.state.Mode == model.ModeSearch {
		searchLines = 1 // search bar is single-line
	}
	commandLines := 0
	if m.state.Mode == model.ModeCommand {
		commandLines = 1 // command bar is single-line
	}
	overhead := BORDER_LINES + headerLines + searchLines + commandLines + TABLE_HEADER_LINES + TAG_LINE + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)
	return max(0, availableRows)
}

// listViewportHeight computes the number of visible rows in the list view,
// matching the calculation in renderListView: availableRows - 2 (for table header and border)
func (m *Model) listViewportHeight() int {
	const (
		BORDER_LINES       = 2
		TABLE_HEADER_LINES = 0
		TAG_LINE           = 0
		STATUS_LINES       = 1
	)
	header := m.renderBanner()
	headerLines := countLines(header)
	searchLines := 0
	if m.state.Mode == model.ModeSearch {
		searchLines = 1
	}
	commandLines := 0
	if m.state.Mode == model.ModeCommand {
		commandLines = 1
	}
	overhead := BORDER_LINES + headerLines + searchLines + commandLines + TABLE_HEADER_LINES + TAG_LINE + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)
	// Match renderListView: tableHeight = availableRows - 1, visibleRows = tableHeight - 1
	return max(1, availableRows-2)
}

// handleToggleSelection toggles selection of current item (space key)
func (m *Model) handleToggleSelection() (tea.Model, tea.Cmd) {
	visibleItems := m.getVisibleItemsForCurrentView()
	if len(visibleItems) == 0 || m.state.Navigation.SelectedIdx >= len(visibleItems) {
		return m, nil
	}

	selectedItem := visibleItems[m.state.Navigation.SelectedIdx]

	switch m.state.Navigation.View {
	case model.ViewApps:
		if app, ok := selectedItem.(model.App); ok {
			if model.HasInStringSet(m.state.Selections.SelectedApps, app.Name) {
				m.state.Selections.SelectedApps = model.RemoveFromStringSet(m.state.Selections.SelectedApps, app.Name)
			} else {
				m.state.Selections.SelectedApps = model.AddToStringSet(m.state.Selections.SelectedApps, app.Name)
			}
		}
		// For clusters/namespaces/projects views, Space has no effect by design.
	}

	return m, nil
}

// handleDrillDown implements drill-down navigation (enter key)
func (m *Model) handleDrillDown() (tea.Model, tea.Cmd) {
	// In apps view, enter opens the resources/tree view for the selected app
	if m.state.Navigation.View == model.ViewApps {
		return m.handleOpenResourcesForSelection()
	}

	visibleItems := m.getVisibleItemsForCurrentView()
	if len(visibleItems) == 0 || m.state.Navigation.SelectedIdx >= len(visibleItems) {
		return m, nil
	}

	selectedItem := visibleItems[m.state.Navigation.SelectedIdx]

	// Use navigation service to handle drill-down logic
	result := m.navigationService.DrillDown(
		m.state.Navigation.View,
		selectedItem,
		visibleItems,
		m.state.Navigation.SelectedIdx,
	)

	if result == nil {
		return m, nil
	}

	// Apply navigation updates
	var cmds []tea.Cmd
	prevView := m.state.Navigation.View

	if result.NewView != nil {
		m.state.Navigation.View = *result.NewView
	}

	if result.ScopeClusters != nil {
		m.state.Selections.ScopeClusters = result.ScopeClusters
	}

	if result.ScopeNamespaces != nil {
		m.state.Selections.ScopeNamespaces = result.ScopeNamespaces
	}

	if result.ScopeProjects != nil {
		m.state.Selections.ScopeProjects = result.ScopeProjects
	}

	if result.ScopeApplicationSets != nil {
		m.state.Selections.ScopeApplicationSets = result.ScopeApplicationSets
	}

	if result.SelectedApps != nil {
		m.state.Selections.SelectedApps = result.SelectedApps
	}

	if result.ShouldResetNavigation {
		// Reset index and clear transient UI filters similar to TS resetNavigation()
		m.state.Navigation.SelectedIdx = 0
		m.state.UI.ActiveFilter = ""
		m.state.UI.SearchQuery = ""
		// Also reset the list navigator to keep cursor/scroll in sync
		m.listNav.Reset()
	}

	if result.ShouldClearLowerLevelSelections {
		// Clear lower-level selections based on the current view
		cleared := m.navigationService.ClearLowerLevelSelections(prevView)
		if v, ok := cleared["scopeNamespaces"]; ok {
			if set, ok2 := v.(map[string]bool); ok2 {
				m.state.Selections.ScopeNamespaces = set
			}
		}
		if v, ok := cleared["scopeProjects"]; ok {
			if set, ok2 := v.(map[string]bool); ok2 {
				m.state.Selections.ScopeProjects = set
			}
		}
		if v, ok := cleared["selectedApps"]; ok {
			if set, ok2 := v.(map[string]bool); ok2 {
				m.state.Selections.SelectedApps = set
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// Mode switching handlers

// handleEnterSearchMode switches to search mode
func (m *Model) handleEnterSearchMode() (tea.Model, tea.Cmd) {
	return m.handleEnhancedEnterSearchMode()
}

// handleEnterCommandMode switches to command mode
func (m *Model) handleEnterCommandMode() (tea.Model, tea.Cmd) {
	return m.handleEnhancedEnterCommandMode()
}

// handleShowHelp shows the help modal
func (m *Model) handleShowHelp() (tea.Model, tea.Cmd) {
	m.state.Mode = model.ModeHelp
	return m, nil
}

// Action handlers

// handleSyncModal shows sync confirmation modal for selected apps
func (m *Model) handleSyncModal() (tea.Model, tea.Cmd) {
	if len(m.state.Selections.SelectedApps) == 0 {
		// If no apps selected, sync current app
		// Get current app more reliably by checking view and bounds carefully
		if m.state.Navigation.View != model.ViewApps {
			// Not in apps view, cannot sync
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "Navigate to apps view to sync applications"}
			}
		}

		visibleItems := m.getVisibleItemsForCurrentView()
		cblog.With("component", "sync").Debug("handleSyncModal: selecting current app",
			"selectedIdx", m.state.Navigation.SelectedIdx,
			"visibleItemsCount", len(visibleItems),
			"view", m.state.Navigation.View,
			"visibleItemsNames", func() []string {
				names := make([]string, len(visibleItems))
				for i, item := range visibleItems {
					if app, ok := item.(model.App); ok {
						names[i] = app.Name
					} else {
						names[i] = fmt.Sprintf("%v", item)
					}
				}
				return names
			}())

		// Validate bounds and selection more strictly
		if len(visibleItems) == 0 {
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "No applications visible to sync"}
			}
		}

		// Ensure SelectedIdx is within bounds
		selectedIdx := m.state.Navigation.SelectedIdx
		if selectedIdx < 0 || selectedIdx >= len(visibleItems) {
			cblog.With("component", "sync").Warn("SelectedIdx out of bounds, using 0",
				"selectedIdx", selectedIdx,
				"visibleItemsCount", len(visibleItems))
			selectedIdx = 0
		}

		if app, ok := visibleItems[selectedIdx].(model.App); ok {
			target := app.Name
			cblog.With("component", "sync").Info("Setting sync target",
				"targetApp", target,
				"selectedIdx", selectedIdx,
				"correctedIdx", selectedIdx != m.state.Navigation.SelectedIdx)
			m.state.Modals.ConfirmTarget = &target
		} else {
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "Selected item is not an application"}
			}
		}
	} else {
		// Multiple apps selected
		target := "__MULTI__"
		m.state.Modals.ConfirmTarget = &target
	}

	if m.state.Modals.ConfirmTarget != nil {
		m.state.Modals.ConfirmSyncSelected = 0 // default to Yes
		m.state.Mode = model.ModeConfirmSync
	}

	return m, nil
}

// handleRollback initiates rollback for selected or current app
func (m *Model) handleRollback() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View != model.ViewApps {
		// Rollback only available in apps view
		return m, nil
	}

	var appName string

	// Check if we have a single app selected
	if len(m.state.Selections.SelectedApps) == 1 {
		// Use the selected app
		for name := range m.state.Selections.SelectedApps {
			appName = name
			break
		}
	} else if len(m.state.Selections.SelectedApps) == 0 {
		// No selection, use current app under cursor
		visibleItems := m.getVisibleItemsForCurrentView()
		if len(visibleItems) > 0 && m.state.Navigation.SelectedIdx < len(visibleItems) {
			if app, ok := visibleItems[m.state.Navigation.SelectedIdx].(model.App); ok {
				appName = app.Name
			}
		}
	} else {
		// Multiple apps selected - rollback not supported for multiple apps
		m.statusService.Set("Rollback not supported for multiple apps")
		return m, nil
	}

	if appName == "" {
		m.statusService.Set("No app selected for rollback")
		return m, nil
	}

	// Set rollback app name and switch to rollback mode
	m.state.Modals.RollbackAppName = &appName
	m.state.Mode = model.ModeRollback

	// Initialize rollback state with loading
	m.state.Rollback = &model.RollbackState{
		AppName: appName,
		Loading: true,
		Mode:    "list",
	}

	// Log rollback start
	cblog.With("component", "rollback").Info("Starting rollback session", "app", appName)

	// Start loading rollback history
	return m, m.startRollbackSession(appName)
}

// handleEscape handles escape key (clear filters, exit modes)
func (m *Model) handleEscape() (tea.Model, tea.Cmd) {
	// Note: Global escape debounce is now handled in handleKeyMsg
	switch m.state.Mode {
	case model.ModeSearch, model.ModeCommand, model.ModeTheme, model.ModeHelp, model.ModeConfirmSync, model.ModeRollback, model.ModeDiff, model.ModeNoDiff:
		m.state.Mode = model.ModeNormal
		return m, nil
	default:
		curr := m.state.Navigation.View
		// Edge case: in apps view with an applied filter, first Esc only clears the filter
		if curr == model.ViewApps && (m.state.UI.ActiveFilter != "" || m.state.UI.SearchQuery != "") {
			m.state.UI.SearchQuery = ""
			m.state.UI.ActiveFilter = ""
			return m, nil
		}

		// Drill up one level and clear current and prior scope selections
		// Clear transient UI inputs as we navigate up
		m.state.UI.SearchQuery = ""
		m.state.UI.ActiveFilter = ""
		m.state.UI.Command = ""

		switch curr {
		case model.ViewTree:
			// Return to apps view from tree/resources view
			if m.treeView != nil {
				m.treeView.ClearFilter()
			}
			m = m.safeChangeView(model.ViewApps)
			m.state.UI.TreeAppName = nil
			m.state.Navigation.SelectedIdx = 0
		case model.ViewApps:
			// Check if scoped by ApplicationSet (separate hierarchy)
			if len(m.state.Selections.ScopeApplicationSets) > 0 {
				m.state.Selections.SelectedApps = model.NewStringSet()
				m.state.Selections.ScopeApplicationSets = model.NewStringSet()
				m = m.safeChangeView(model.ViewApplicationSets)
				m.state.Navigation.SelectedIdx = 0
				return m, nil
			}
			// Clear current level (selected apps) and prior (projects), go up to Projects
			m.state.Selections.SelectedApps = model.NewStringSet()
			m.state.Selections.ScopeProjects = model.NewStringSet()
			m = m.safeChangeView(model.ViewProjects)
			m.state.Navigation.SelectedIdx = 0
		case model.ViewApplicationSets:
			// At ApplicationSets view, escape just clears the scope (stay in place)
			m.state.Selections.ScopeApplicationSets = model.NewStringSet()
			m.state.Navigation.SelectedIdx = 0
		case model.ViewProjects:
			// Clear current (projects) and prior (namespaces), go up to Namespaces
			m.state.Selections.ScopeProjects = model.NewStringSet()
			m.state.Selections.ScopeNamespaces = model.NewStringSet()
			m = m.safeChangeView(model.ViewNamespaces)
			m.state.Navigation.SelectedIdx = 0
		case model.ViewNamespaces:
			// Clear current (namespaces) and prior (clusters), go up to Clusters
			m.state.Selections.ScopeNamespaces = model.NewStringSet()
			m.state.Selections.ScopeClusters = model.NewStringSet()
			m = m.safeChangeView(model.ViewClusters)
			m.state.Navigation.SelectedIdx = 0
		case model.ViewClusters:
			// At top level: clear current scope only; stay on Clusters
			m.state.Selections.ScopeClusters = model.NewStringSet()
			m.state.Navigation.SelectedIdx = 0
		}
		return m, nil
	}
}

// Mode-specific key handlers

// handleSearchModeKeys handles input when in search mode
func (m *Model) handleSearchModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleEnhancedSearchModeKeys(msg)
}

// handleCommandModeKeys handles input when in command mode
func (m *Model) handleCommandModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m.handleEnhancedCommandModeKeys(msg)
}

// handleThemeModeKeys handles input when in theme selection mode.
// Navigation keys (up/k, down/j, pgup, pgdown, g, G) are handled by the centralized router.
func (m *Model) handleThemeModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.ensureThemeOptionsLoaded()

	if len(m.themeOptions) == 0 {
		m.state.Mode = model.ModeNormal
		return m, nil
	}

	switch msg.String() {
	case "esc", "q":
		// Restore original theme when cancelled
		if m.state.UI.ThemeOriginalName != "" {
			m.applyThemePreview(m.state.UI.ThemeOriginalName)
		}
		// Clear command state if any
		m.inputComponents.BlurInputs()
		m.inputComponents.ClearCommandInput()
		m.state.UI.Command = ""
		m.state.Mode = model.ModeNormal
		return m, nil
	case "enter":
		selectedTheme := m.themeOptions[m.themeNav.Cursor()].Name
		newModel, cmd := m.handleThemeCommand(selectedTheme)
		// Clear command state if any
		newModel.inputComponents.BlurInputs()
		newModel.inputComponents.ClearCommandInput()
		newModel.state.UI.Command = ""
		newModel.state.Mode = model.ModeNormal
		return newModel, cmd
	}
	return m, nil
}

// syncThemeNavToState syncs themeNav cursor/scroll to the legacy state fields for rendering
func (m *Model) syncThemeNavToState() {
	m.state.UI.ThemeSelectedIndex = m.themeNav.Cursor()
	m.state.UI.ThemeScrollOffset = m.themeNav.ScrollOffset()
}

// applyThemePreview applies a theme temporarily for preview without saving to config
func (m *Model) applyThemePreview(themeName string) {
	// Create a temporary config with the preview theme
	tempConfig := &config.ArgonautConfig{
		Appearance: config.AppearanceConfig{
			Theme: themeName,
		},
	}

	// Apply the theme temporarily
	palette := theme.FromConfig(tempConfig)
	applyTheme(palette)
	m.applyThemeToModel()
}

// themePageSize returns the number of visible theme rows for page scrolling
func (m *Model) themePageSize() int {
	headerLines := 2      // title + blank line
	borderLines := 4      // top + bottom border + padding
	footerLines := 2      // potential warning message + blank line
	statusLine := 1       // status line at bottom
	maxAvailableHeight := m.state.Terminal.Rows - headerLines - borderLines - footerLines - statusLine
	maxThemeLines := max(5, maxAvailableHeight)
	return max(1, maxThemeLines-2) // Reserve 2 lines for scroll indicators
}


// handleHelpModeKeys handles input when in help mode
func (m *Model) handleHelpModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.state.Mode = model.ModeNormal
		return m, nil
	}
	return m, nil
}

// handleNoDiffModeKeys handles input when in no-diff modal mode
func (m *Model) handleNoDiffModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key press closes the modal
	m.state.Mode = model.ModeNormal
	return m, nil
}

// handleK9sErrorModeKeys handles input when k9s error modal is shown
func (m *Model) handleK9sErrorModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc", "q":
		m.state.Mode = model.ModeNormal
		m.state.Modals.K9sError = nil
		return m, nil
	}
	return m, nil
}

// removed: resources list mode

// handleDiffModeKeys handles non-navigation input in diff mode.
// Navigation keys (up/k, down/j, pgup, pgdown, g, G) are handled by the centralized router.
func (m *Model) handleDiffModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.Diff == nil {
		return m, nil
	}
	switch msg.String() {
	case "q", "esc":
		m.state.Mode = model.ModeNormal
		m.state.Diff = nil
		return m, nil
	case "/":
		// Reuse search input for diff filtering
		m.inputComponents.ClearSearchInput()
		m.inputComponents.FocusSearchInput()
		m.state.Mode = model.ModeSearch
		return m, nil
	default:
		return m, nil
	}
}

// diffPageSize returns the number of visible rows for page scrolling in diff mode
func (m *Model) diffPageSize() int {
	// Diff view uses most of the terminal height minus borders and header
	overhead := 6 // header, footer, borders
	return max(1, m.state.Terminal.Rows-overhead)
}

// handleConfirmSyncKeys handles input when in sync confirmation mode
func (m *Model) handleConfirmSyncKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state.Mode = model.ModeNormal
		m.state.Modals.ConfirmTarget = nil
		return m, nil
	case "left", "h":
		if m.state.Modals.ConfirmSyncSelected > 0 {
			m.state.Modals.ConfirmSyncSelected = 0
		}
		return m, nil
	case "right", "l":
		if m.state.Modals.ConfirmSyncSelected < 1 {
			m.state.Modals.ConfirmSyncSelected = 1
		}
		return m, nil
	case "enter":
		if m.state.Modals.ConfirmSyncSelected == 1 {
			// Cancel
			m.state.Mode = model.ModeNormal
			m.state.Modals.ConfirmTarget = nil
			return m, nil
		}
		fallthrough
	case "y":
		// Confirm sync - keep modal open and show loading overlay
		target := m.state.Modals.ConfirmTarget
		prune := m.state.Modals.ConfirmSyncPrune
		m.state.Modals.ConfirmSyncLoading = true
		m.state.Mode = model.ModeConfirmSync

		if target != nil {
			cblog.With("component", "sync").Info("Executing sync confirmation",
				"target", *target,
				"isMulti", *target == "__MULTI__")
			if *target == "__MULTI__" {
				return m, m.syncSelectedApplications(prune)
			} else {
				return m, m.syncSingleApplication(*target, prune)
			}
		}
		return m, nil
	case "p":
		// Toggle prune option
		m.state.Modals.ConfirmSyncPrune = !m.state.Modals.ConfirmSyncPrune
		return m, nil
	case "w":
		// Toggle watch option (single or multi)
		m.state.Modals.ConfirmSyncWatch = !m.state.Modals.ConfirmSyncWatch
		return m, nil
	}
	return m, nil
}

// rollbackPageSize returns the number of visible rows for page scrolling in rollback mode
func (m *Model) rollbackPageSize() int {
	// Approximate: modal takes ~60% of terminal height, minus header/footer
	modalHeight := m.state.Terminal.Rows * 60 / 100
	overhead := 6 // header, footer, borders
	return max(1, modalHeight-overhead)
}

// handleRollbackModeKeys handles input when in rollback mode.
// Navigation keys (up/k, down/j, pgup, pgdown, g, G) are handled by the centralized router.
func (m *Model) handleRollbackModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "ctrl+c":
		// Allow exit even during loading
		m.state.Mode = model.ModeNormal
		m.state.Modals.RollbackAppName = nil
		m.state.Rollback = nil
		return m, nil
	}

	// If still loading or no rollback state, only handle exit keys above
	if m.state.Rollback == nil || m.state.Rollback.Loading {
		return m, nil
	}

	switch msg.String() {
	case "p":
		// Toggle prune option in confirmation view
		if m.state.Rollback.Mode == "confirm" {
			m.state.Rollback.Prune = !m.state.Rollback.Prune
		}
		return m, nil
	case "w":
		// Toggle watch option in confirmation view
		if m.state.Rollback.Mode == "confirm" {
			m.state.Rollback.Watch = !m.state.Rollback.Watch
		}
		return m, nil
	case "left", "h":
		if m.state.Rollback.Mode == "confirm" {
			m.state.Rollback.ConfirmSelected = 0
		}
		return m, nil
	case "right", "l":
		if m.state.Rollback.Mode == "confirm" {
			m.state.Rollback.ConfirmSelected = 1
		}
		return m, nil
	case "enter":
		// Confirm rollback or execute rollback
		if m.state.Rollback.Mode == "list" {
			// Switch to confirmation mode
			m.state.Rollback.Mode = "confirm"
			m.state.Rollback.ConfirmSelected = 0
		} else if m.state.Rollback.Mode == "confirm" {
			if m.state.Rollback.ConfirmSelected == 1 {
				// Cancel
				m.state.Rollback = nil
				m.state.Modals.RollbackAppName = nil
				m.state.Mode = model.ModeNormal
				return m, nil
			}
			// Execute rollback
			if len(m.state.Rollback.Rows) > 0 && m.state.Rollback.SelectedIdx < len(m.state.Rollback.Rows) {
				selectedRow := m.state.Rollback.Rows[m.state.Rollback.SelectedIdx]
				request := model.RollbackRequest{
					ID:           selectedRow.ID,
					Name:         m.state.Rollback.AppName,
					AppNamespace: m.state.Rollback.AppNamespace,
					Prune:        m.state.Rollback.Prune,
					DryRun:       m.state.Rollback.DryRun,
				}
				// Set loading state
				m.state.Rollback.Loading = true
				m.state.Rollback.Error = ""
				return m, m.executeRollback(request)
			}
		}
		return m, nil
	case "d":
		// Show diff for selected revision (if we want to implement this later)
		if m.state.Rollback.Mode == "list" && len(m.state.Rollback.Rows) > 0 && m.state.Rollback.SelectedIdx < len(m.state.Rollback.Rows) {
			selectedRow := m.state.Rollback.Rows[m.state.Rollback.SelectedIdx]
			// Could implement diff viewing here later
			_ = selectedRow
		}
		return m, nil
	}
	return m, nil
}

// handleConfirmAppDeleteKeys handles input when in app delete confirmation mode
func (m *Model) handleConfirmAppDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		// Cancel deletion and return to normal mode
		m.state.Mode = model.ModeNormal
		m.state.Modals.DeleteAppName = nil
		m.state.Modals.DeleteAppNamespace = nil
		m.state.Modals.DeleteConfirmationKey = ""
		m.state.Modals.DeleteError = nil
		m.state.Modals.DeleteLoading = false
		return m, nil
	case "backspace":
		// Remove the last character from confirmation key
		if len(m.state.Modals.DeleteConfirmationKey) > 0 {
			m.state.Modals.DeleteConfirmationKey = m.state.Modals.DeleteConfirmationKey[:len(m.state.Modals.DeleteConfirmationKey)-1]
		}
		return m, nil
	case "c":
		// Toggle cascade option
		m.state.Modals.DeleteCascade = !m.state.Modals.DeleteCascade
		return m, nil
	case "p":
		// Cycle through propagation policies: foreground -> background -> orphan -> foreground
		switch m.state.Modals.DeletePropagationPolicy {
		case "foreground":
			m.state.Modals.DeletePropagationPolicy = "background"
		case "background":
			m.state.Modals.DeletePropagationPolicy = "orphan"
		case "orphan":
			m.state.Modals.DeletePropagationPolicy = "foreground"
		default:
			m.state.Modals.DeletePropagationPolicy = "foreground"
		}
		return m, nil
	default:
		// Record the key press, normalizing space handling for Bubble Tea v2
		// where space may stringify as "space" instead of " ".
		keyStr := msg.String()
		if len(keyStr) == 1 {
			m.state.Modals.DeleteConfirmationKey = keyStr
			if keyStr == "y" || keyStr == "Y" {
				return m.executeAppDeletion()
			}
		} else {
			// Handle space explicitly
			if msg.Key().Code == tea.KeySpace || keyStr == "space" {
				m.state.Modals.DeleteConfirmationKey = " "
			}
		}
		return m, nil
	}
}

// handleAppDelete initiates the app deletion confirmation
func (m *Model) handleAppDelete() (tea.Model, tea.Cmd) {
	// Only work in apps view
	if m.state.Navigation.View != model.ViewApps {
		return m, nil
	}

	if len(m.state.Selections.SelectedApps) == 0 {
		// If no apps selected, delete current app
		visibleItems := m.getVisibleItemsForCurrentView()
		if len(visibleItems) > 0 && m.state.Navigation.SelectedIdx < len(visibleItems) {
			if app, ok := visibleItems[m.state.Navigation.SelectedIdx].(model.App); ok {
				// Single app deletion
				m.state.Mode = model.ModeConfirmAppDelete
				m.state.Modals.DeleteAppName = &app.Name
				m.state.Modals.DeleteAppNamespace = app.AppNamespace
				m.state.Modals.DeleteConfirmationKey = ""
				m.state.Modals.DeleteError = nil
				m.state.Modals.DeleteLoading = false
				m.state.Modals.DeleteCascade = true // Default to cascade
				m.state.Modals.DeletePropagationPolicy = "foreground"

				cblog.With("component", "app-delete").Debug("Opening delete confirmation", "app", app.Name)
			}
		}
	} else {
		// Multiple apps selected
		multiTarget := "__MULTI__"
		m.state.Mode = model.ModeConfirmAppDelete
		m.state.Modals.DeleteAppName = &multiTarget
		m.state.Modals.DeleteAppNamespace = nil // Not applicable for multi-delete
		m.state.Modals.DeleteConfirmationKey = ""
		m.state.Modals.DeleteError = nil
		m.state.Modals.DeleteLoading = false
		m.state.Modals.DeleteCascade = true // Default to cascade
		m.state.Modals.DeletePropagationPolicy = "foreground"

		cblog.With("component", "app-delete").Debug("Opening multi-delete confirmation", "count", len(m.state.Selections.SelectedApps))
	}

	return m, nil
}

// executeAppDeletion performs the actual deletion after confirmation
func (m *Model) executeAppDeletion() (tea.Model, tea.Cmd) {
	if m.state.Modals.DeleteAppName == nil {
		return m, nil
	}

	appName := *m.state.Modals.DeleteAppName
	isMulti := appName == "__MULTI__"
	m.state.Modals.DeleteLoading = true

	if isMulti {
		// Multi-app deletion
		return m, m.deleteSelectedApplications(m.state.Modals.DeleteCascade, m.state.Modals.DeletePropagationPolicy)
	} else {
		// Single app deletion
		return m, m.deleteSingleApplication(appName, m.state.Modals.DeleteAppNamespace, m.state.Modals.DeleteCascade, m.state.Modals.DeletePropagationPolicy)
	}
}

// handleResourceDelete initiates the resource deletion confirmation
func (m *Model) handleResourceDelete() (tea.Model, tea.Cmd) {
	// Only work in tree view
	if m.state.Navigation.View != model.ViewTree {
		return m, nil
	}

	if m.treeView == nil {
		return m, nil
	}

	// Early check: if no multi-selection and cursor is on Missing resource, silently reject
	if !m.treeView.HasSelection() && m.treeView.CurrentResourceIsMissing() {
		return m, nil
	}

	// Get selected resources
	selections := m.treeView.GetSelectedResources()
	if len(selections) == 0 {
		// No resources to delete (e.g., Application node selected)
		return m, nil
	}

	// Filter out Missing resources (already deleted)
	var validSelections []treeview.ResourceSelection
	for _, sel := range selections {
		if sel.IsMissing() {
			continue
		}
		validSelections = append(validSelections, sel)
	}

	// If all selections were Missing, silently reject
	if len(validSelections) == 0 {
		return m, nil
	}

	// Convert to ResourceDeleteTarget
	targets := make([]model.ResourceDeleteTarget, 0, len(validSelections))
	for _, sel := range validSelections {
		targets = append(targets, model.ResourceDeleteTarget{
			AppName:   sel.AppName,
			Group:     sel.Group,
			Version:   sel.Version,
			Kind:      sel.Kind,
			Namespace: sel.Namespace,
			Name:      sel.Name,
		})
	}

	// Get app name for the modal
	appName := m.treeView.GetAppName()
	if len(targets) > 0 {
		appName = targets[0].AppName
	}

	// Set up modal state
	m.state.Mode = model.ModeConfirmResourceDelete
	m.state.Modals.ResourceDeleteAppName = &appName
	m.state.Modals.ResourceDeleteAppNamespace = nil // TODO: Get from app if needed
	m.state.Modals.ResourceDeleteTargets = targets
	m.state.Modals.ResourceDeleteConfirmationKey = ""
	m.state.Modals.ResourceDeleteError = nil
	m.state.Modals.ResourceDeleteLoading = false
	m.state.Modals.ResourceDeleteCascade = true        // Default to cascade
	m.state.Modals.ResourceDeletePropagationPolicy = "foreground"
	m.state.Modals.ResourceDeleteForce = false

	cblog.With("component", "resource-delete").Debug("Opening resource delete confirmation", "count", len(targets))

	return m, nil
}

// handleConfirmResourceDeleteKeys handles input when in resource delete confirmation mode
func (m *Model) handleConfirmResourceDeleteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		// Cancel deletion and return to normal mode
		m.state.Mode = model.ModeNormal
		m.state.Modals.ResourceDeleteAppName = nil
		m.state.Modals.ResourceDeleteTargets = nil
		m.state.Modals.ResourceDeleteConfirmationKey = ""
		m.state.Modals.ResourceDeleteError = nil
		m.state.Modals.ResourceDeleteLoading = false
		return m, nil
	case "backspace":
		// Remove the last character from confirmation key
		if len(m.state.Modals.ResourceDeleteConfirmationKey) > 0 {
			m.state.Modals.ResourceDeleteConfirmationKey = m.state.Modals.ResourceDeleteConfirmationKey[:len(m.state.Modals.ResourceDeleteConfirmationKey)-1]
		}
		return m, nil
	case "c":
		// Toggle cascade option
		m.state.Modals.ResourceDeleteCascade = !m.state.Modals.ResourceDeleteCascade
		return m, nil
	case "p":
		// Cycle through propagation policies: foreground -> background -> orphan -> foreground
		switch m.state.Modals.ResourceDeletePropagationPolicy {
		case "foreground":
			m.state.Modals.ResourceDeletePropagationPolicy = "background"
		case "background":
			m.state.Modals.ResourceDeletePropagationPolicy = "orphan"
		case "orphan":
			m.state.Modals.ResourceDeletePropagationPolicy = "foreground"
		default:
			m.state.Modals.ResourceDeletePropagationPolicy = "foreground"
		}
		return m, nil
	case "f":
		// Toggle force option
		m.state.Modals.ResourceDeleteForce = !m.state.Modals.ResourceDeleteForce
		return m, nil
	default:
		// Record the key press
		keyStr := msg.String()
		if len(keyStr) == 1 {
			m.state.Modals.ResourceDeleteConfirmationKey = keyStr
			if keyStr == "y" || keyStr == "Y" {
				return m.executeResourceDeletion()
			}
		}
		return m, nil
	}
}

// executeResourceDeletion performs the actual resource deletion after confirmation
func (m *Model) executeResourceDeletion() (tea.Model, tea.Cmd) {
	if m.state.Modals.ResourceDeleteAppName == nil || len(m.state.Modals.ResourceDeleteTargets) == 0 {
		return m, nil
	}

	m.state.Modals.ResourceDeleteLoading = true

	return m, m.deleteSelectedResources(
		m.state.Modals.ResourceDeleteTargets,
		m.state.Modals.ResourceDeleteCascade,
		m.state.Modals.ResourceDeletePropagationPolicy,
		m.state.Modals.ResourceDeleteForce,
	)
}

// handleAuthRequiredModeKeys handles input when authentication is required
func (m *Model) handleAuthRequiredModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, func() tea.Msg { return model.QuitMsg{} }
	case "l":
		// Open logs pager with syntax highlighting
		logFile := os.Getenv("ARGONAUT_LOG_FILE")
		if strings.TrimSpace(logFile) == "" {
			return m, func() tea.Msg { return model.ApiErrorMsg{Message: "No logs available"} }
		}
		data, err := os.ReadFile(logFile)
		if err != nil {
			return m, func() tea.Msg { return model.ApiErrorMsg{Message: "No logs available"} }
		}

		// Apply syntax highlighting to each log line
		lines := strings.Split(string(data), "\n")
		var highlightedLines []string
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				highlightedLines = append(highlightedLines, HighlightLogLine(line))
			} else {
				highlightedLines = append(highlightedLines, line)
			}
		}
		highlightedContent := strings.Join(highlightedLines, "\n")

		return m, m.openTextPager("Logs", highlightedContent)
	}
	return m, nil
}

// handleErrorModeKeys handles input when in error mode
func (m *Model) handleErrorModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		// If no apps have been loaded (initial load failed), exit the application
		// Otherwise, clear error state and return to normal mode
		if len(m.state.Apps) == 0 {
			return m, func() tea.Msg { return model.QuitMsg{} }
		}

		// Clear error state and return to normal mode
		m.state.CurrentError = nil
		if m.state.ErrorState != nil {
			m.state.ErrorState.Current = nil
		}
		m.state.Mode = model.ModeNormal
		return m, nil
	case "l":
		// Open system logs view to help debug the error
		// Clear error state and open logs in pager
		m.state.CurrentError = nil
		if m.state.ErrorState != nil {
			m.state.ErrorState.Current = nil
		}
		// Open logs in ov pager with syntax highlighting
		logContent := m.readLogContent()
		// Apply syntax highlighting
		lines := strings.Split(logContent, "\n")
		highlightedLines := make([]string, 0, len(lines))
		for _, line := range lines {
			highlightedLines = append(highlightedLines, HighlightLogLine(line))
		}
		highlightedContent := strings.Join(highlightedLines, "\n")
		return m, m.openTextPager("Logs", highlightedContent)
	}
	return m, nil
}

// handleConnectionErrorModeKeys handles input when in connection error mode
func (m *Model) handleConnectionErrorModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		// Exit application when there's no connection
		return m, func() tea.Msg { return model.QuitMsg{} }
	case "esc":
		// Return to normal mode from connection error (for retry attempts)
		m.state.Mode = model.ModeNormal
		return m, nil
	case "l":
		// Open system logs view to help debug connection issues
		// Open logs in ov pager with syntax highlighting
		logContent := m.readLogContent()
		// Apply syntax highlighting
		lines := strings.Split(logContent, "\n")
		highlightedLines := make([]string, 0, len(lines))
		for _, line := range lines {
			highlightedLines = append(highlightedLines, HighlightLogLine(line))
		}
		highlightedContent := strings.Join(highlightedLines, "\n")
		return m, m.openTextPager("Logs", highlightedContent)
	}
	return m, nil
}

// handleCoreDetectedModeKeys handles input when core mode is detected
func (m *Model) handleCoreDetectedModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		// Exit application
		return m, func() tea.Msg { return model.QuitMsg{} }
	}
	// Ignore other keys including ":" to prevent command mode
	return m, nil
}

// Helper function to get visible items for current view
func (m *Model) getVisibleItemsForCurrentView() []interface{} {
	// Delegate to shared computation used by the view
	return m.getVisibleItems()
}

// handleKeyMsg centralizes keyboard handling and delegates to mode/view handlers
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global kill: always quit on Ctrl+C
	if msg.String() == "ctrl+c" {
		return m, func() tea.Msg { return model.QuitMsg{} }
	}

	// Global escape debounce to prevent rapid consecutive escape key presses
	if msg.String() == "esc" {
		now := time.Now().UnixMilli()
		const GLOBAL_ESCAPE_DEBOUNCE_MS = 100 // 100ms debounce

		if now-m.state.Navigation.LastEscPressed < GLOBAL_ESCAPE_DEBOUNCE_MS {
			// Too soon, ignore this escape
			return m, nil
		}

		// Update last escape timestamp
		m.state.Navigation.LastEscPressed = now
	}

	// Centralized navigation interception
	// Navigation keys (up/k, down/j, pgup, pgdown, g, G) are handled here for all modes
	// that support list navigation. Mode-specific handlers only handle non-navigation keys.
	if isNavigationKey(msg) {
		ctx := m.getNavigatorContext()
		if ctx.SupportsNavigation {
			return m.executeNavigation(ctx, msg)
		}
		// If navigation not supported, fall through to mode-specific handler
	}

	// Mode-specific handling (non-navigation keys only for modes that support navigation)
	switch m.state.Mode {
	case model.ModeSearch:
		return m.handleSearchModeKeys(msg)
	case model.ModeCommand:
		return m.handleCommandModeKeys(msg)
	case model.ModeTheme:
		return m.handleThemeModeKeys(msg)
	case model.ModeHelp:
		return m.handleHelpModeKeys(msg)
	case model.ModeNoDiff:
		return m.handleNoDiffModeKeys(msg)
	case model.ModeConfirmSync:
		return m.handleConfirmSyncKeys(msg)
	case model.ModeRollback:
		return m.handleRollbackModeKeys(msg)
	case model.ModeConfirmAppDelete:
		return m.handleConfirmAppDeleteKeys(msg)
	case model.ModeConfirmResourceDelete:
		return m.handleConfirmResourceDeleteKeys(msg)
	case model.ModeDiff:
		return m.handleDiffModeKeys(msg)
	case model.ModeAuthRequired:
		return m.handleAuthRequiredModeKeys(msg)
	case model.ModeError:
		return m.handleErrorModeKeys(msg)
	case model.ModeConnectionError:
		return m.handleConnectionErrorModeKeys(msg)
	case model.ModeCoreDetected:
		return m.handleCoreDetectedModeKeys(msg)
	case model.ModeUpgrade:
		return m.handleUpgradeModeKeys(msg)
	case model.ModeUpgradeError:
		return m.handleUpgradeErrorModeKeys(msg)
	case model.ModeUpgradeSuccess:
		return m.handleUpgradeSuccessModeKeys(msg)
	case model.ModeK9sContextSelect:
		return m.handleK9sContextSelectKeys(msg)
	case model.ModeK9sError:
		return m.handleK9sErrorModeKeys(msg)
	}

	// Tree view keys when in normal mode.
	// Navigation keys (up/k, down/j, pgup, pgdown, g, G) are handled by the centralized router.
	if m.state.Navigation.View == model.ViewTree {
		switch msg.String() {
		case "q", "esc":
			// Clear filter and stop active tree watchers, return to list
			if m.treeView != nil {
				m.treeView.ClearFilter()
			}
			m = m.safeChangeView(model.ViewApps)
			visibleItems := m.getVisibleItemsForCurrentView()
			m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
				m.state.Navigation.SelectedIdx,
				len(visibleItems),
			)
			return m, nil
		case "/":
			// Enter search mode for tree filtering
			return m.handleEnterSearchMode()
		case "n":
			// Next match in tree filter
			if m.treeView != nil && m.treeView.MatchCount() > 0 {
				m.treeView.NextMatch()
				m.treeNav.SetCursor(m.treeView.SelectedIndex())
			}
			return m, nil
		case "N":
			// Previous match in tree filter
			if m.treeView != nil && m.treeView.MatchCount() > 0 {
				m.treeView.PrevMatch()
				m.treeNav.SetCursor(m.treeView.SelectedIndex())
			}
			return m, nil
		case "left", "h", "right", "l", "enter":
			// Expand/collapse handled by tree view, then sync treeNav
			if m.treeView != nil {
				updatedModel, _ := m.treeView.Update(msg)
				m.treeView = updatedModel.(*treeview.TreeView)
				// After expand/collapse, item count may change - sync treeNav
				newLine := m.treeView.SelectedIndex()
				if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
					newLine = s.SelectedLineIndex()
				}
				m.treeNav.SetItemCount(m.treeView.VisibleCount())
				m.treeNav.SetViewportHeight(m.treeViewportHeight())
				m.treeNav.SetCursor(newLine)
			}
			return m, nil
		case "K":
			// Open k9s for the selected resource
			return m.handleOpenK9s()
		case "d":
			// Show diff for the selected resource
			return m.handleResourceDiff()
		case " ", "space":
			// Toggle selection for delete
			if m.treeView != nil {
				if !m.treeView.ToggleSelection() && m.treeView.CurrentResourceIsMissing() {
					return m, func() tea.Msg {
						return model.StatusChangeMsg{Status: "Cannot select: resource is missing"}
					}
				}
			}
			return m, nil
		case "ctrl+d":
			// Open delete confirmation for selected resource(s)
			return m.handleResourceDelete()
		case ":":
			// Enter command mode
			return m.handleEnterCommandMode()
		case "?":
			// Show help
			return m.handleShowHelp()
		default:
			if m.treeView != nil {
				_, cmd := m.treeView.Update(msg)
				return m, cmd
			}
			return m, nil
		}
	}

	// Normal-mode global keys.
	// Navigation keys (up/k, down/j, pgup, pgdown, g, G) are handled by the centralized router.
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return model.QuitMsg{} }
	case " ", "space":
		return m.handleToggleSelection()
	case "enter":
		return m.handleDrillDown()
	case "/":
		return m.handleEnterSearchMode()
	case ":":
		return m.handleEnterCommandMode()
	case "?":
		return m.handleShowHelp()
	case "s":
		if m.state.Navigation.View == model.ViewApps {
			return m.handleSyncModal()
		}
	case "r":
		// Open resources for selected app (apps view)
		if m.state.Navigation.View == model.ViewApps {
			return m.handleOpenResourcesForSelection()
		}
		return m, nil
	case "d":
		// Open diff for selected app (apps view)
		if m.state.Navigation.View == model.ViewApps {
			return m.handleOpenDiffForSelection()
		}
		return m, nil
	case "R":
		cblog.With("component", "tui").Debug("R key pressed", "view", m.state.Navigation.View)
		if m.state.Navigation.View == model.ViewApps {
			cblog.With("component", "rollback").Debug("Calling handleRollback()")
			return m.handleRollback()
		} else {
			cblog.With("component", "rollback").Debug("Rollback not available in view", "view", m.state.Navigation.View)
		}
	case "ctrl+d":
		// Open delete confirmation for selected app (apps view) or resource (tree view)
		if m.state.Navigation.View == model.ViewApps {
			return m.handleAppDelete()
		}
		if m.state.Navigation.View == model.ViewTree {
			return m.handleResourceDelete()
		}
		return m, nil
	case "esc":
		return m.handleEscape()
	case "Z":
		now := time.Now().UnixMilli()
		if m.state.Navigation.LastZPressed > 0 && now-m.state.Navigation.LastZPressed < 500 {
			// ZZ: save and quit (like vim)
			return m, func() tea.Msg { return model.QuitMsg{} }
		}
		m.state.Navigation.LastZPressed = now
		return m, nil
	case "Q":
		// Check if this is ZQ (quit without saving)
		now := time.Now().UnixMilli()
		if m.state.Navigation.LastZPressed > 0 && now-m.state.Navigation.LastZPressed < 500 {
			// ZQ: quit without saving (like vim)
			m.state.Navigation.LastZPressed = 0 // Reset Z state
			return m, func() tea.Msg { return model.QuitMsg{} }
		}
		return m, nil
	}
	return m, nil
}

// handleOpenResourcesForSelection opens the resources (tree) view for the selected app
func (m *Model) handleOpenResourcesForSelection() (tea.Model, tea.Cmd) {
	// If multiple apps selected, open tree view and stream all
	sel := m.state.Selections.SelectedApps
	selected := make([]string, 0, len(sel))
	for name, ok := range sel {
		if ok {
			selected = append(selected, name)
		}
	}
	if len(selected) > 1 {
		// Clean up any existing tree watchers before starting new ones
		m.cleanupTreeWatchers()
		// Reset tree view to a fresh multi-app instance
		m.treeView = treeview.NewTreeView(0, 0)
		m.treeView.ApplyTheme(currentPalette)
		m.treeView.SetSize(m.contentInnerWidth(), m.state.Terminal.Rows)
		m.treeNav.Reset() // Reset scroll position
		m.state.SaveNavigationState()
		m.state.Navigation.View = model.ViewTree
		// Clear single-app tracker
		m.state.UI.TreeAppName = nil
		m.treeLoading = true
		var cmds []tea.Cmd
		for _, name := range selected {
			// start initial load + watch stream for the tree view
			var appObj *model.App
			for i := range m.state.Apps {
				if m.state.Apps[i].Name == name {
					appObj = &m.state.Apps[i]
					break
				}
			}
			if appObj == nil {
				tmp := model.App{Name: name}
				appObj = &tmp
			}
			cmds = append(cmds, m.startLoadingResourceTree(*appObj))
			cmds = append(cmds, m.startWatchingResourceTree(*appObj))
		}
		cmds = append(cmds, m.consumeTreeEvent())
		// Include consumeWatchEvent to continue receiving app updates (for resource sync status)
		cmds = append(cmds, m.consumeWatchEvent())
		return m, tea.Batch(cmds...)
	}
	// Fallback to single app tree view
	items := m.getVisibleItemsForCurrentView()
	if len(items) == 0 || m.state.Navigation.SelectedIdx >= len(items) {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for resources"} }
	}
	app, ok := items[m.state.Navigation.SelectedIdx].(model.App)
	if !ok {
		return m, func() tea.Msg {
			return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for resources"}
		}
	}
	// Clean up any existing tree watchers before starting new one
	m.cleanupTreeWatchers()
	// Reset tree view to a fresh single-app instance
	m.treeView = treeview.NewTreeView(0, 0)
	m.treeView.ApplyTheme(currentPalette)
	m.treeView.SetSize(m.contentInnerWidth(), m.state.Terminal.Rows)
	m.treeNav.Reset() // Reset scroll position
	m.state.SaveNavigationState()
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeAppName = &app.Name
	m.treeLoading = true
	// Include consumeWatchEvent to continue receiving app updates (for resource sync status)
	return m, tea.Batch(m.startLoadingResourceTree(app), m.startWatchingResourceTree(app), m.consumeTreeEvent(), m.consumeWatchEvent())
}

// handleResourceDiff shows the diff for the currently selected resource in tree view
func (m *Model) handleResourceDiff() (*Model, tea.Cmd) {
	if m.treeView == nil {
		return m, nil
	}

	group, kind, namespace, name, ok := m.treeView.SelectedResource()
	if !ok {
		return m, nil
	}

	// For Application nodes, show the full app diff
	if kind == "Application" {
		// Show loading spinner
		if m.state.Diff == nil {
			m.state.Diff = &model.DiffState{}
		}
		m.state.Diff.Loading = true
		return m, m.startDiffSession(name)
	}

	// Get the app name for resource-level diff
	appName := ""
	if m.state.UI.TreeAppName != nil {
		appName = *m.state.UI.TreeAppName
	} else if m.treeView != nil {
		appName = m.treeView.GetAppName()
	}
	if appName == "" {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Could not determine application name"} }
	}

	// Show loading spinner
	if m.state.Diff == nil {
		m.state.Diff = &model.DiffState{}
	}
	m.state.Diff.Loading = true

	return m, m.startResourceDiffSession(appName, group, kind, namespace, name)
}

// handleOpenK9s opens k9s for the currently selected resource in tree view
func (m *Model) handleOpenK9s() (tea.Model, tea.Cmd) {
	if m.treeView == nil {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No resource selected"} }
	}

	_, kind, namespace, name, ok := m.treeView.SelectedResource()
	if !ok {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No resource selected"} }
	}

	// Skip Application nodes - they're synthetic roots
	if kind == "Application" {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Select a Kubernetes resource to open in k9s"} }
	}

	cblog.With("component", "k9s").Debug("Opening k9s for resource",
		"kind", kind,
		"namespace", namespace,
		"name", name)

	// Try to find the context from the current app's cluster info
	var context string
	var contextFound bool
	if m.state.UI.TreeAppName != nil {
		for i := range m.state.Apps {
			if m.state.Apps[i].Name == *m.state.UI.TreeAppName {
				app := m.state.Apps[i]
				if app.ClusterID != nil {
					clusterID := *app.ClusterID
					// Try to find matching context
					ctx, err := m.findK9sContext(clusterID)
					if err == nil {
						context = ctx
						contextFound = true
					} else {
						cblog.With("component", "k9s").Debug("Could not find context for cluster",
							"clusterID", clusterID, "err", err)
					}
				}
				break
			}
		}
	}

	// If we couldn't auto-detect the context, show the context picker
	// IMPORTANT: Always prompt user to select - never auto-select to prevent
	// accidentally operating on the wrong cluster
	if !contextFound {
		contexts, err := kubeconfig.ListContextNames()
		if err != nil || len(contexts) == 0 {
			// No kubeconfig or no contexts - just launch k9s without context
			cblog.With("component", "k9s").Warn("Could not load kubeconfig contexts", "err", err)
			return m, m.openK9s(kind, namespace, "")
		}

		// Always show picker - even with single context, user must confirm
		m.k9sContextOptions = contexts
		m.k9sContextSelected = 0
		m.k9sPendingKind = kind
		m.k9sPendingNamespace = namespace
		m.state.Mode = model.ModeK9sContextSelect
		return m, nil
	}

	return m, m.openK9s(kind, namespace, context)
}

// handleK9sContextSelectKeys handles input when selecting a kubeconfig context for k9s
func (m *Model) handleK9sContextSelectKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if len(m.k9sContextOptions) == 0 {
		m.state.Mode = model.ModeNormal
		return m, nil
	}

	switch msg.String() {
	case "q", "esc":
		// Cancel context selection
		m.state.Mode = model.ModeNormal
		m.k9sContextOptions = nil
		return m, nil
	case "up", "k":
		if m.k9sContextSelected > 0 {
			m.k9sContextSelected--
		}
		return m, nil
	case "down", "j":
		if m.k9sContextSelected < len(m.k9sContextOptions)-1 {
			m.k9sContextSelected++
		}
		return m, nil
	case "enter":
		// Select context and launch k9s
		selectedContext := m.k9sContextOptions[m.k9sContextSelected]
		kind := m.k9sPendingKind
		namespace := m.k9sPendingNamespace

		// Clear state
		m.k9sContextOptions = nil
		m.k9sPendingKind = ""
		m.k9sPendingNamespace = ""
		m.state.Mode = model.ModeNormal

		return m, m.openK9s(kind, namespace, selectedContext)
	}

	return m, nil
}

// findK9sContext tries to find a kubeconfig context for the given cluster ID.
// It only returns a match for EXACT matches to prevent accidentally opening
// the wrong cluster. If no exact match is found, the caller should prompt the user.
func (m *Model) findK9sContext(clusterID string) (string, error) {
	kc, err := kubeconfig.Load()
	if err != nil {
		return "", err
	}

	// Special case: in-cluster means use current context
	if clusterID == "in-cluster" {
		if current := kc.GetCurrentContext(); current != "" {
			return current, nil
		}
		return "", fmt.Errorf("in-cluster specified but no current context set")
	}

	// Only accept exact name match - no fuzzy or partial matching
	// This is intentionally strict to prevent opening wrong clusters
	if ctx, found := kc.FindContextByName(clusterID); found {
		return ctx, nil
	}

	return "", fmt.Errorf("no exact context match found for cluster: %s", clusterID)
}

// handleOpenDiffForSelection opens the diff for the selected app
func (m *Model) handleOpenDiffForSelection() (tea.Model, tea.Cmd) {
	// Check if there are multiple selected apps first
	sel := m.state.Selections.SelectedApps
	selected := make([]string, 0, len(sel))
	for name, ok := range sel {
		if ok {
			selected = append(selected, name)
		}
	}

	cblog.With("component", "diff").Debug("handleOpenDiffForSelection",
		"selected_apps", selected,
		"selected_count", len(selected),
		"cursor_idx", m.state.Navigation.SelectedIdx)

	var appName string
	if len(selected) == 1 {
		// Use the single selected app
		appName = selected[0]
		cblog.With("component", "diff").Debug("Using single selected app", "app", appName)
	} else if len(selected) > 1 {
		// Multiple apps selected - cannot show diff for multiple apps
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Cannot show diff for multiple apps"} }
	} else {
		// No apps selected via checkbox, use cursor position
		items := m.getVisibleItemsForCurrentView()
		if len(items) == 0 || m.state.Navigation.SelectedIdx >= len(items) {
			return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for diff"} }
		}
		app, ok := items[m.state.Navigation.SelectedIdx].(model.App)
		if !ok {
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for diff"}
			}
		}
		appName = app.Name
		cblog.With("component", "diff").Debug("Using cursor position", "app", appName, "idx", m.state.Navigation.SelectedIdx)
	}

	cblog.With("component", "diff").Debug("Starting diff session", "app", appName)
	if m.state.Diff == nil {
		m.state.Diff = &model.DiffState{}
	}
	m.state.Diff.Loading = true
	return m, m.startDiffSession(appName)
}
