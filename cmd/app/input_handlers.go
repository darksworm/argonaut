package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// Navigation handlers matching TypeScript functionality

// handleNavigationUp moves cursor up with bounds checking
func (m *Model) handleNavigationUp() (tea.Model, tea.Cmd) {
	// Only update navigation state - table cursor will be synced in render
	newIdx := m.state.Navigation.SelectedIdx - 1
	if newIdx < 0 {
		newIdx = 0
	}
	m.state.Navigation.SelectedIdx = newIdx

	// Update list scroll offset for non-tree views to keep cursor visible
	if m.state.Navigation.View != model.ViewTree {
		if newIdx < m.listScrollOffset {
			m.listScrollOffset = newIdx
		}
	}

	return m, nil
}

// handleNavigationDown moves cursor down with bounds checking
func (m *Model) handleNavigationDown() (tea.Model, tea.Cmd) {
	visibleItems := m.getVisibleItemsForCurrentView()
	newIdx := m.state.Navigation.SelectedIdx + 1
	maxItems := len(visibleItems)
	if maxItems == 0 {
		return m, nil
	}
	if newIdx >= maxItems {
		newIdx = maxItems - 1
	}
	m.state.Navigation.SelectedIdx = newIdx

	// Update list scroll offset for non-tree views to keep cursor visible
	if m.state.Navigation.View != model.ViewTree {
		// Calculate viewport height for list views (similar to tree view calculation)
		availableRows := m.state.Terminal.Rows - 10 // Approximate overhead
		visibleRows := max(0, availableRows-1)      // Leave room for header

		if newIdx >= m.listScrollOffset+visibleRows {
			m.listScrollOffset = newIdx - visibleRows + 1
		}
	}

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

	if result.SelectedApps != nil {
		m.state.Selections.SelectedApps = result.SelectedApps
	}

	if result.ShouldResetNavigation {
		// Reset index and clear transient UI filters similar to TS resetNavigation()
		m.state.Navigation.SelectedIdx = 0
		m.state.UI.ActiveFilter = ""
		m.state.UI.SearchQuery = ""
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
	case model.ModeSearch, model.ModeCommand, model.ModeHelp, model.ModeConfirmSync, model.ModeRollback, model.ModeDiff, model.ModeNoDiff:
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
		case model.ViewApps:
			// Clear current level (selected apps) and prior (projects), go up to Projects
			m.state.Selections.SelectedApps = model.NewStringSet()
			m.state.Selections.ScopeProjects = model.NewStringSet()
			m = m.safeChangeView(model.ViewProjects)
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

// handleGoToTop moves to first item (double-g)
func (m *Model) handleGoToTop() (tea.Model, tea.Cmd) {
	// Special handling for tree view - scroll to top
	if m.state.Navigation.View == model.ViewTree {
		m.treeScrollOffset = 0
		m.state.Navigation.LastGPressed = 0 // Reset double-g state
		return m, nil
	}

	m.state.Navigation.SelectedIdx = 0
	m.listScrollOffset = 0              // Reset scroll to top for list views
	m.state.Navigation.LastGPressed = 0 // Reset double-g state
	return m, nil
}

// handleGoToBottom moves to last item (G key)
func (m *Model) handleGoToBottom() (tea.Model, tea.Cmd) {
	// Special handling for tree view - scroll to bottom
	if m.state.Navigation.View == model.ViewTree {
		// Set to a large value, will be clamped in renderTreePanel
		m.treeScrollOffset = 1 << 30
		return m, nil
	}

	visibleItems := m.getVisibleItemsForCurrentView()
	if len(visibleItems) > 0 {
		m.state.Navigation.SelectedIdx = len(visibleItems) - 1

		// Update scroll offset to show the last item
		availableRows := m.state.Terminal.Rows - 10 // Approximate overhead
		visibleRows := max(0, availableRows-1)      // Leave room for header
		m.listScrollOffset = max(0, len(visibleItems)-visibleRows)
	}
	return m, nil
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

// removed: resources list mode

// handleDiffModeKeys handles navigation and search in diff mode
func (m *Model) handleDiffModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.state.Diff == nil {
		return m, nil
	}
	switch msg.String() {
	case "q", "esc":
		m.state.Mode = model.ModeNormal
		m.state.Diff = nil
		return m, nil
	case "up", "k":
		m.state.Diff.Offset = max(0, m.state.Diff.Offset-1)
		return m, nil
	case "down", "j":
		m.state.Diff.Offset = m.state.Diff.Offset + 1
		return m, nil
	case "g":
		m.state.Diff.Offset = 0
		return m, nil
	case "G":
		// set to large; clamped on render
		m.state.Diff.Offset = 1 << 30
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

// handleRollbackModeKeys handles input when in rollback mode
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
	case "j", "down":
		// Navigate down in rollback history
		if len(m.state.Rollback.Rows) > 0 {
			newIdx := m.state.Rollback.SelectedIdx + 1
			if newIdx >= len(m.state.Rollback.Rows) {
				newIdx = len(m.state.Rollback.Rows) - 1
			}
			m.state.Rollback.SelectedIdx = newIdx
		}
		return m, nil
	case "k", "up":
		// Navigate up in rollback history
		if len(m.state.Rollback.Rows) > 0 {
			newIdx := m.state.Rollback.SelectedIdx - 1
			if newIdx < 0 {
				newIdx = 0
			}
			m.state.Rollback.SelectedIdx = newIdx
		}
		return m, nil
	case "g":
		// Double-g check for go to top
		now := time.Now().UnixMilli()
		if now-m.state.Navigation.LastGPressed < 500 {
			// Go to top
			m.state.Rollback.SelectedIdx = 0
			m.state.Navigation.LastGPressed = 0
		} else {
			m.state.Navigation.LastGPressed = now
		}
		return m, nil
	case "G":
		// Go to bottom
		if len(m.state.Rollback.Rows) > 0 {
			m.state.Rollback.SelectedIdx = len(m.state.Rollback.Rows) - 1
		}
		return m, nil
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

	// Mode-specific handling first
	switch m.state.Mode {
	case model.ModeSearch:
		return m.handleSearchModeKeys(msg)
	case model.ModeCommand:
		return m.handleCommandModeKeys(msg)
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
	case model.ModeDiff:
		return m.handleDiffModeKeys(msg)
	case model.ModeAuthRequired:
		return m.handleAuthRequiredModeKeys(msg)
	case model.ModeError:
		return m.handleErrorModeKeys(msg)
	case model.ModeConnectionError:
		return m.handleConnectionErrorModeKeys(msg)
	case model.ModeUpgrade:
		return m.handleUpgradeModeKeys(msg)
	case model.ModeUpgradeError:
		return m.handleUpgradeErrorModeKeys(msg)
	case model.ModeUpgradeSuccess:
		return m.handleUpgradeSuccessModeKeys(msg)
	}

	// Tree view keys when in normal mode
	if m.state.Navigation.View == model.ViewTree {
		switch msg.String() {
		case "q", "esc":
			// Stop active tree watchers and return to list
			m = m.safeChangeView(model.ViewApps)
			visibleItems := m.getVisibleItemsForCurrentView()
			m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
				m.state.Navigation.SelectedIdx,
				len(visibleItems),
			)
			return m, nil
		case "up", "k", "down", "j":
			if m.treeView != nil {
				// Use line-based indices to account for blank separators between app roots
				oldLine := m.treeView.SelectedIndex()
				if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
					oldLine = s.SelectedLineIndex()
				}
				updatedModel, _ := m.treeView.Update(msg)
				m.treeView = updatedModel.(*treeview.TreeView)
				newLine := m.treeView.SelectedIndex()
				if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
					newLine = s.SelectedLineIndex()
				}

				// Adjust scroll based on cursor movement
				if msg.String() == "up" || msg.String() == "k" {
					// Moving up - ensure cursor is visible at top
					if newLine < oldLine && newLine < m.treeScrollOffset {
						m.treeScrollOffset = newLine
					}
				} else {
					// Moving down - ensure cursor is visible at bottom
					viewportHeight := m.treeViewportHeight()
					if newLine > oldLine && newLine >= m.treeScrollOffset+viewportHeight {
						m.treeScrollOffset = newLine - viewportHeight + 1
					}
				}
			}
			return m, nil
		case "left", "h", "right", "l", "enter":
			if m.treeView != nil {
				oldLine := m.treeView.SelectedIndex()
				if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
					oldLine = s.SelectedLineIndex()
				}
				oldCount := m.treeView.VisibleCount()
				updatedModel, _ := m.treeView.Update(msg)
				m.treeView = updatedModel.(*treeview.TreeView)
				newLine := m.treeView.SelectedIndex()
				if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
					newLine = s.SelectedLineIndex()
				}
				newCount := m.treeView.VisibleCount()

				// If the visible count changed (expand/collapse) or cursor moved,
				// ensure cursor stays in viewport
				if oldCount != newCount || oldLine != newLine {
					viewportHeight := m.treeViewportHeight()
					// Ensure cursor is visible
					if newLine < m.treeScrollOffset {
						m.treeScrollOffset = newLine
					} else if newLine >= m.treeScrollOffset+viewportHeight {
						m.treeScrollOffset = newLine - viewportHeight + 1
					}
					// Clamp scroll to valid range
					// Use total rendered lines if available to account for separators
					maxScroll := max(0, newCount-viewportHeight)
					if v, ok := interface{}(m.treeView).(interface{ VisibleLineCount() int }); ok {
						maxScroll = max(0, v.VisibleLineCount()-viewportHeight)
					}
					if m.treeScrollOffset > maxScroll {
						m.treeScrollOffset = maxScroll
					}
				}
			}
			return m, nil
		case "g":
			// Handle double-g for go to top
			now := time.Now().UnixMilli()
			if m.state.Navigation.LastGPressed > 0 && now-m.state.Navigation.LastGPressed < 500 {
				// Double-g: go to top
				if m.treeView != nil {
					// Send multiple "up" keys to move to top
					for i := 0; i < m.treeView.SelectedIndex(); i++ {
						upMsg := tea.KeyPressMsg{Text: "k", Code: 'k'}
						updatedModel, _ := m.treeView.Update(upMsg)
						m.treeView = updatedModel.(*treeview.TreeView)
					}
					m.treeScrollOffset = 0
					m.state.Navigation.LastGPressed = 0
				}
			} else {
				m.state.Navigation.LastGPressed = now
			}
			return m, nil
		case "G":
			// Go to bottom
			if m.treeView != nil {
				// Use rendered line count if available
				totalItems := m.treeView.VisibleCount()
				if v, ok := interface{}(m.treeView).(interface{ VisibleLineCount() int }); ok {
					totalItems = v.VisibleLineCount()
				}
				currentIdx := m.treeView.SelectedIndex()
				// Send multiple "down" keys to move to bottom
				for i := currentIdx; i < totalItems-1; i++ {
					downMsg := tea.KeyPressMsg{Text: "j", Code: 'j'}
					updatedModel, _ := m.treeView.Update(downMsg)
					m.treeView = updatedModel.(*treeview.TreeView)
				}
				// Scroll to show the bottom item
				viewportHeight := m.treeViewportHeight()
				m.treeScrollOffset = max(0, totalItems-viewportHeight)
			}
			return m, nil
		default:
			if m.treeView != nil {
				_, cmd := m.treeView.Update(msg)
				return m, cmd
			}
			return m, nil
		}
	}

	// Normal-mode global keys
	switch msg.String() {
	case "ctrl+c":
		return m, func() tea.Msg { return model.QuitMsg{} }
	case "up", "k":
		return m.handleNavigationUp()
	case "down", "j":
		return m.handleNavigationDown()
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
		// Open delete confirmation for selected app (apps view)
		if m.state.Navigation.View == model.ViewApps {
			return m.handleAppDelete()
		}
		return m, nil
	case "esc":
		return m.handleEscape()
	case "g":
		now := time.Now().UnixMilli()
		if m.state.Navigation.LastGPressed > 0 && now-m.state.Navigation.LastGPressed < 500 {
			return m.handleGoToTop()
		}
		m.state.Navigation.LastGPressed = now
		return m, nil
	case "G":
		return m.handleGoToBottom()
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
		m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)
		m.treeScrollOffset = 0 // Reset scroll position
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
	m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)
	m.treeScrollOffset = 0 // Reset scroll position
	m.state.SaveNavigationState()
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeAppName = &app.Name
	m.treeLoading = true
	return m, tea.Batch(m.startLoadingResourceTree(app), m.startWatchingResourceTree(app), m.consumeTreeEvent())
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
