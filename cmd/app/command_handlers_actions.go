package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// Action command handlers

// handleLogsCommand handles the "logs" command
func (m *Model) handleLogsCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	// Open logs using the configured log file (via ARGONAUT_LOG_FILE) with a sensible fallback.
	// Reuse the view helper so behavior matches the Logs view.
	body := m.readLogContent()
	return m, m.openTextPager(body)
}

// handleSyncCommand handles the "sync" command
func (m *Model) handleSyncCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	return m.handleSyncModal()
}

// handleRollbackCommand handles the "rollback" command
func (m *Model) handleRollbackCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	target := arg
	if target == "" {
		// Only try to get current selection if we're in the apps view
		if m.state.Navigation.View == model.ViewApps {
			items := m.getVisibleItemsForCurrentView()
			if len(items) > 0 && m.state.Navigation.SelectedIdx < len(items) {
				if app, ok := items[m.state.Navigation.SelectedIdx].(model.App); ok {
					target = app.Name
				}
			}
		} else {
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for rollback"}
			}
		}
	}
	if target == "" {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for rollback"} }
	}

	// Use the same rollback logic as the R key
	cblog.With("component", "rollback").Debug(":rollback command invoked", "app", target)
	m.state.Modals.RollbackAppName = &target
	m.state.Mode = model.ModeRollback

	// Initialize rollback state with loading
	m.state.Rollback = &model.RollbackState{
		AppName: target,
		Loading: true,
		Mode:    "list",
	}

	// Start loading rollback history using the same function as R key
	return m, m.startRollbackSession(target)
}

// handleResourcesCommand handles the "resources", "res", "r" commands
func (m *Model) handleResourcesCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	target := arg

	// If no explicit target provided, check for multiple selections first (like 'r' key does)
	if target == "" {
		sel := m.state.Selections.SelectedApps
		names := make([]string, 0, len(sel))
		for name, ok := range sel {
			if ok {
				names = append(names, name)
			}
		}

		if len(names) > 1 {
			// Clean up any existing tree watchers before starting new ones
			m.cleanupTreeWatchers()
			// Multiple apps selected - open multi tree view with live updates
			m.treeView = treeview.NewTreeView(0, 0)
			m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)
			m.treeScrollOffset = 0 // Reset scroll position
			m.state.SaveNavigationState()
			m.state.Navigation.View = model.ViewTree
			m.state.UI.TreeAppName = nil
			m.treeLoading = true
			var cmds []tea.Cmd
			for _, n := range names {
				var appObj *model.App
				for i := range m.state.Apps {
					if m.state.Apps[i].Name == n {
						appObj = &m.state.Apps[i]
						break
					}
				}
				if appObj == nil {
					tmp := model.App{Name: n}
					appObj = &tmp
				}
				cmds = append(cmds, m.startLoadingResourceTree(*appObj))
				cmds = append(cmds, m.startWatchingResourceTree(*appObj))
			}
			cmds = append(cmds, m.consumeTreeEvent())
			return m, tea.Batch(cmds...)
		} else if len(names) == 1 {
			// Single app selected via checkbox
			target = names[0]
		} else {
			// No apps selected via checkbox, try cursor position
			if m.state.Navigation.View == model.ViewApps {
				items := m.getVisibleItemsForCurrentView()
				if len(items) > 0 && m.state.Navigation.SelectedIdx < len(items) {
					if app, ok := items[m.state.Navigation.SelectedIdx].(model.App); ok {
						target = app.Name
					}
				}
			} else {
				return m, func() tea.Msg {
					return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for resources"}
				}
			}
		}
	}

	if target == "" {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app specified for resources view"} }
	}

	var targetApp *model.App
	for i := range m.state.Apps {
		if m.state.Apps[i].Name == target {
			targetApp = &m.state.Apps[i]
			break
		}
	}

	if targetApp == nil {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown app: " + target} }
	}

	// Clean up any existing tree watchers before starting new ones
	m.cleanupTreeWatchers()

	// Create tree view
	m.treeView = treeview.NewTreeView(0, 0)
	m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)
	m.treeScrollOffset = 0 // Reset scroll position

	// Set tree-specific state
	m.state.SaveNavigationState()
	m.state.Navigation.View = model.ViewTree
	m.state.UI.TreeAppName = &target
	m.treeLoading = true

	// Start loading the tree and watching for live updates
	cmds := []tea.Cmd{
		m.startLoadingResourceTree(*targetApp),
		m.startWatchingResourceTree(*targetApp),
		m.consumeTreeEvent(),
	}
	return m, tea.Batch(cmds...)
}