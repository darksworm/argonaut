package main

import (
	"github.com/a9s/go-app/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

// Navigation handlers matching TypeScript functionality

// handleNavigationUp moves cursor up with bounds checking
func (m Model) handleNavigationUp() (Model, tea.Cmd) {
	// Update both our navigation state and the appropriate table cursor
	newIdx := m.state.Navigation.SelectedIdx - 1
	if newIdx < 0 {
		newIdx = 0
	}
	m.state.Navigation.SelectedIdx = newIdx
	
	// Update the appropriate table cursor to match
	switch m.state.Navigation.View {
	case model.ViewApps:
		m.appsTable.MoveUp(1)
	case model.ViewClusters:
		m.clustersTable.MoveUp(1)
	case model.ViewNamespaces:
		m.namespacesTable.MoveUp(1)
	case model.ViewProjects:
		m.projectsTable.MoveUp(1)
	}
	
	return m, nil
}

// handleNavigationDown moves cursor down with bounds checking
func (m Model) handleNavigationDown() (Model, tea.Cmd) {
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
	
	// Update the appropriate table cursor to match
	switch m.state.Navigation.View {
	case model.ViewApps:
		m.appsTable.MoveDown(1)
	case model.ViewClusters:
		m.clustersTable.MoveDown(1)
	case model.ViewNamespaces:
		m.namespacesTable.MoveDown(1)
	case model.ViewProjects:
		m.projectsTable.MoveDown(1)
	}
	
	return m, nil
}

// handleToggleSelection toggles selection of current item (space key)
func (m Model) handleToggleSelection() (Model, tea.Cmd) {
	visibleItems := m.getVisibleItemsForCurrentView()
	if len(visibleItems) == 0 || m.state.Navigation.SelectedIdx >= len(visibleItems) {
		return m, nil
	}

	selectedItem := visibleItems[m.state.Navigation.SelectedIdx]

	switch m.state.Navigation.View {
	case model.ViewApps:
		if app, ok := selectedItem.(model.App); ok {
			if model.HasInStringSet(m.state.Selections.SelectedApps, app.Name) {
				model.RemoveFromStringSet(m.state.Selections.SelectedApps, app.Name)
			} else {
				model.AddToStringSet(m.state.Selections.SelectedApps, app.Name)
			}
		}
	case model.ViewClusters:
		if cluster, ok := selectedItem.(string); ok {
			if model.HasInStringSet(m.state.Selections.ScopeClusters, cluster) {
				model.RemoveFromStringSet(m.state.Selections.ScopeClusters, cluster)
			} else {
				model.AddToStringSet(m.state.Selections.ScopeClusters, cluster)
			}
		}
	case model.ViewNamespaces:
		if namespace, ok := selectedItem.(string); ok {
			if model.HasInStringSet(m.state.Selections.ScopeNamespaces, namespace) {
				model.RemoveFromStringSet(m.state.Selections.ScopeNamespaces, namespace)
			} else {
				model.AddToStringSet(m.state.Selections.ScopeNamespaces, namespace)
			}
		}
	case model.ViewProjects:
		if project, ok := selectedItem.(string); ok {
			if model.HasInStringSet(m.state.Selections.ScopeProjects, project) {
				model.RemoveFromStringSet(m.state.Selections.ScopeProjects, project)
			} else {
				model.AddToStringSet(m.state.Selections.ScopeProjects, project)
			}
		}
	}

	return m, nil
}

// handleDrillDown implements drill-down navigation (enter key)
func (m Model) handleDrillDown() (Model, tea.Cmd) {
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
func (m Model) handleEnterSearchMode() (Model, tea.Cmd) {
	return m.handleEnhancedEnterSearchMode()
}

// handleEnterCommandMode switches to command mode
func (m Model) handleEnterCommandMode() (Model, tea.Cmd) {
	return m.handleEnhancedEnterCommandMode()
}

// handleShowHelp shows the help modal
func (m Model) handleShowHelp() (Model, tea.Cmd) {
	m.state.Mode = model.ModeHelp
	return m, nil
}

// Action handlers

// handleSyncModal shows sync confirmation modal for selected apps
func (m Model) handleSyncModal() (Model, tea.Cmd) {
	if len(m.state.Selections.SelectedApps) == 0 {
		// If no apps selected, sync current app
		visibleItems := m.getVisibleItemsForCurrentView()
		if len(visibleItems) > 0 && m.state.Navigation.SelectedIdx < len(visibleItems) {
			if app, ok := visibleItems[m.state.Navigation.SelectedIdx].(model.App); ok {
				target := app.Name
				m.state.Modals.ConfirmTarget = &target
			}
		}
	} else {
		// Multiple apps selected
		target := "__MULTI__"
		m.state.Modals.ConfirmTarget = &target
	}

	if m.state.Modals.ConfirmTarget != nil {
		m.state.Mode = model.ModeConfirmSync
	}

	return m, nil
}

// handleRefresh refreshes the current view data
func (m Model) handleRefresh() (Model, tea.Cmd) {
	if m.state.Server != nil {
		m.state.Mode = model.ModeLoading
		return m, m.startLoadingApplications()
	}
	return m, func() tea.Msg {
		return model.StatusChangeMsg{Status: "No server configured"}
	}
}

// handleEscape handles escape key (clear filters, exit modes) with debounce
func (m Model) handleEscape() (Model, tea.Cmd) {
	// Debounce escape key to prevent rapid multiple exits
	now := time.Now().UnixMilli()
	const ESCAPE_DEBOUNCE_MS = 100 // 100ms debounce (reduced from 200ms)

	if now-m.state.Navigation.LastEscPressed < ESCAPE_DEBOUNCE_MS {
		// Too soon, ignore this escape
		return m, nil
	}

	// Update last escape timestamp
	m.state.Navigation.LastEscPressed = now

	switch m.state.Mode {
	case model.ModeSearch, model.ModeCommand, model.ModeHelp, model.ModeConfirmSync, model.ModeRollback, model.ModeResources, model.ModeDiff:
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
			m.state.Navigation.View = model.ViewProjects
			m.state.Navigation.SelectedIdx = 0
		case model.ViewProjects:
			// Clear current (projects) and prior (namespaces), go up to Namespaces
			m.state.Selections.ScopeProjects = model.NewStringSet()
			m.state.Selections.ScopeNamespaces = model.NewStringSet()
			m.state.Navigation.View = model.ViewNamespaces
			m.state.Navigation.SelectedIdx = 0
		case model.ViewNamespaces:
			// Clear current (namespaces) and prior (clusters), go up to Clusters
			m.state.Selections.ScopeNamespaces = model.NewStringSet()
			m.state.Selections.ScopeClusters = model.NewStringSet()
			m.state.Navigation.View = model.ViewClusters
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
func (m Model) handleGoToTop() (Model, tea.Cmd) {
	m.state.Navigation.SelectedIdx = 0
	m.state.Navigation.LastGPressed = 0 // Reset double-g state
	
	// Update the appropriate table cursor to match
	switch m.state.Navigation.View {
	case model.ViewApps:
		m.appsTable.GotoTop()
	case model.ViewClusters:
		m.clustersTable.GotoTop()
	case model.ViewNamespaces:
		m.namespacesTable.GotoTop()
	case model.ViewProjects:
		m.projectsTable.GotoTop()
	}
	
	return m, nil
}

// handleGoToBottom moves to last item (G key)
func (m Model) handleGoToBottom() (Model, tea.Cmd) {
	visibleItems := m.getVisibleItemsForCurrentView()
	if len(visibleItems) > 0 {
		m.state.Navigation.SelectedIdx = len(visibleItems) - 1
	}
	
	// Update the appropriate table cursor to match
	switch m.state.Navigation.View {
	case model.ViewApps:
		m.appsTable.GotoBottom()
	case model.ViewClusters:
		m.clustersTable.GotoBottom()
	case model.ViewNamespaces:
		m.namespacesTable.GotoBottom()
	case model.ViewProjects:
		m.projectsTable.GotoBottom()
	}
	
	return m, nil
}

// Mode-specific key handlers

// handleSearchModeKeys handles input when in search mode
func (m Model) handleSearchModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	return m.handleEnhancedSearchModeKeys(msg)
}

// handleCommandModeKeys handles input when in command mode
func (m Model) handleCommandModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	return m.handleEnhancedCommandModeKeys(msg)
}

// handleHelpModeKeys handles input when in help mode
func (m Model) handleHelpModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "?":
		m.state.Mode = model.ModeNormal
		return m, nil
	}
	return m, nil
}

// handleResourcesModeKeys handles navigation in resources mode
func (m Model) handleResourcesModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.state.Resources == nil {
		return m, nil
	}
	switch msg.String() {
	case "q", "esc":
		m.state.Mode = model.ModeNormal
		return m, nil
	case "j", "down":
		// Scroll down in resources list using table cursor
		if m.state.Resources != nil && len(m.state.Resources.Resources) > 0 {
			maxOffset := len(m.state.Resources.Resources) - 1
			if m.state.Resources.Offset < maxOffset {
				m.state.Resources.Offset++
			}
		}
		return m, nil
	case "k", "up":
		// Scroll up in resources list using table cursor
		if m.state.Resources != nil && m.state.Resources.Offset > 0 {
			m.state.Resources.Offset--
		}
		return m, nil
	case "g":
		// Go to top of resources
		if m.state.Resources != nil {
			m.state.Resources.Offset = 0
		}
		return m, nil
	case "G":
		// Go to bottom of resources
		if m.state.Resources != nil && len(m.state.Resources.Resources) > 0 {
			m.state.Resources.Offset = len(m.state.Resources.Resources) - 1
		}
		return m, nil
	}
	return m, nil
}

// handleDiffModeKeys handles navigation and search in diff mode
func (m Model) handleDiffModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
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
func (m Model) handleConfirmSyncKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state.Mode = model.ModeNormal
		m.state.Modals.ConfirmTarget = nil
		return m, nil
	case "enter", "y":
		// Confirm sync
		m.state.Mode = model.ModeNormal
		target := m.state.Modals.ConfirmTarget
		prune := m.state.Modals.ConfirmSyncPrune
		m.state.Modals.ConfirmTarget = nil

		if target != nil {
			if *target == "__MULTI__" {
				// Sync multiple selected applications
				return m, m.syncSelectedApplications(prune)
			} else {
				// Sync single application
				return m, m.syncSingleApplication(*target, prune)
			}
		}
		return m, nil
	case "p":
		// Toggle prune option
		m.state.Modals.ConfirmSyncPrune = !m.state.Modals.ConfirmSyncPrune
		return m, nil
	case "w":
		// Toggle watch option (only for single app)
		if m.state.Modals.ConfirmTarget != nil && *m.state.Modals.ConfirmTarget != "__MULTI__" {
			m.state.Modals.ConfirmSyncWatch = !m.state.Modals.ConfirmSyncWatch
		}
		return m, nil
	}
	return m, nil
}

// handleRollbackModeKeys handles input when in rollback mode
func (m Model) handleRollbackModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state.Mode = model.ModeNormal
		m.state.Modals.RollbackAppName = nil
		return m, nil
	}
	return m, nil
}

// Helper function to get visible items for current view
func (m Model) getVisibleItemsForCurrentView() []interface{} {
	// Delegate to shared computation used by the view
	return m.getVisibleItems()
}
