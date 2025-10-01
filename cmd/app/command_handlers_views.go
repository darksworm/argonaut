package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// View command handlers for switching between different application views

// handleAllCommand handles the "all" command - clears all filters and selections
func (m *Model) handleAllCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	m.state.Selections = *model.NewSelectionState()
	m.state.UI.SearchQuery = ""
	m.state.UI.ActiveFilter = ""
	return m, func() tea.Msg { return model.StatusChangeMsg{Status: "All filtering cleared."} }
}

// handleUpCommand handles the "up" command - equivalent to escape
func (m *Model) handleUpCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	return m.handleEscape()
}

// handleDiffCommand handles the "diff" command
func (m *Model) handleDiffCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	// :diff [app]
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
				return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for diff"}
			}
		}
	}
	if target == "" {
		return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for diff"} }
	}
	// Initialize diff state with loading
	if m.state.Diff == nil {
		m.state.Diff = &model.DiffState{}
	}
	m.state.Diff.Loading = true
	return m, m.startDiffSession(target)
}

// handleClusterCommand handles "cluster", "clusters", "cls", "context", "ctx" commands
func (m *Model) handleClusterCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	// Exit deep views and clear lower-level scopes
	m.state.UI.TreeAppName = nil
	m.treeLoading = false
	m.state.Selections.SelectedApps = model.NewStringSet()
	m = m.safeChangeView(model.ViewClusters)
	if arg != "" {
		// Validate cluster exists
		all := m.autocompleteEngine.GetArgumentSuggestions("cluster", "", m.state)
		names := make([]string, 0, len(all))
		for _, s := range all {
			names = append(names, strings.TrimPrefix(s, ":cluster "))
		}
		matched := false
		for _, n := range names {
			if strings.EqualFold(n, arg) {
				arg = n
				matched = true
				break
			}
		}
		if !matched {
			return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown cluster: " + arg} }
		}
		m.state.Selections.ScopeClusters = model.StringSetFromSlice([]string{arg})
		m.state.Selections.ScopeNamespaces = model.NewStringSet()
		m.state.Selections.ScopeProjects = model.NewStringSet()
		m = m.safeChangeView(model.ViewNamespaces)
	} else {
		m.state.Selections.ScopeClusters = model.NewStringSet()
		m.state.Selections.ScopeNamespaces = model.NewStringSet()
		m.state.Selections.ScopeProjects = model.NewStringSet()
	}
	return m, nil
}

// handleNamespaceCommand handles "namespace", "namespaces", "ns" commands
func (m *Model) handleNamespaceCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	m.state.UI.TreeAppName = nil
	m.treeLoading = false
	m = m.safeChangeView(model.ViewNamespaces)
	m.state.Selections.SelectedApps = model.NewStringSet()
	if arg != "" {
		all := m.autocompleteEngine.GetArgumentSuggestions("namespace", "", m.state)
		names := make([]string, 0, len(all))
		for _, s := range all {
			names = append(names, strings.TrimPrefix(s, ":namespace "))
		}
		matched := false
		for _, n := range names {
			if strings.EqualFold(n, arg) {
				arg = n
				matched = true
				break
			}
		}
		if !matched {
			return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown namespace: " + arg} }
		}
		m.state.Selections.ScopeNamespaces = model.StringSetFromSlice([]string{arg})
		m.state.Selections.ScopeProjects = model.NewStringSet()
		m = m.safeChangeView(model.ViewProjects)
	} else {
		m.state.Selections.ScopeNamespaces = model.NewStringSet()
		m.state.Selections.ScopeProjects = model.NewStringSet()
	}
	return m, nil
}

// handleProjectCommand handles "project", "projects", "proj" commands
func (m *Model) handleProjectCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	m.state.UI.TreeAppName = nil
	m.treeLoading = false
	m = m.safeChangeView(model.ViewProjects)
	m.state.Selections.SelectedApps = model.NewStringSet()
	if arg != "" {
		all := m.autocompleteEngine.GetArgumentSuggestions("project", "", m.state)
		names := make([]string, 0, len(all))
		for _, s := range all {
			names = append(names, strings.TrimPrefix(s, ":project "))
		}
		matched := false
		for _, n := range names {
			if strings.EqualFold(n, arg) {
				arg = n
				matched = true
				break
			}
		}
		if !matched {
			return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown project: " + arg} }
		}
		m.state.Selections.ScopeProjects = model.StringSetFromSlice([]string{arg})
		m = m.safeChangeView(model.ViewApps)
	} else {
		m.state.Selections.ScopeProjects = model.NewStringSet()
	}
	return m, nil
}

// handleAppCommand handles "app", "apps" commands
func (m *Model) handleAppCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	m.state.UI.TreeAppName = nil
	m.treeLoading = false
	m = m.safeChangeView(model.ViewApps)
	if arg != "" {
		// Select the app and move cursor to it if found
		m.state.Selections.SelectedApps = model.StringSetFromSlice([]string{arg})
		idx := -1
		for i, a := range m.state.Apps {
			if a.Name == arg {
				idx = i
				break
			}
		}
		if idx >= 0 {
			m.state.Navigation.SelectedIdx = idx
		}
	} else {
		m.state.Selections.SelectedApps = model.NewStringSet()
	}
	return m, nil
}