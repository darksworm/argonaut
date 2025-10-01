package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// Navigation message handlers

// handleSetView changes the current view
func (m *Model) handleSetView(msg tea.Msg) (tea.Model, tea.Cmd) {
	setViewMsg := msg.(model.SetViewMsg)
	m.state.Navigation.View = setViewMsg.View
	return m, nil
}

// handleSetSelectedIdx updates the selected index
func (m *Model) handleSetSelectedIdx(msg tea.Msg) (tea.Model, tea.Cmd) {
	setIdxMsg := msg.(model.SetSelectedIdxMsg)
	// Keep selection within bounds of currently visible items
	m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
		setIdxMsg.SelectedIdx,
		len(m.getVisibleItems()),
	)
	return m, nil
}

// handleResetNavigation resets navigation state
func (m *Model) handleResetNavigation(msg tea.Msg) (tea.Model, tea.Cmd) {
	resetMsg := msg.(model.ResetNavigationMsg)
	m.state.Navigation.SelectedIdx = 0
	if resetMsg.View != nil {
		m.state.Navigation.View = *resetMsg.View
	}
	return m, nil
}

// handleSetSelectedApps updates the selected applications
func (m *Model) handleSetSelectedApps(msg tea.Msg) (tea.Model, tea.Cmd) {
	selectedAppsMsg := msg.(model.SetSelectedAppsMsg)
	m.state.Selections.SelectedApps = selectedAppsMsg.Apps
	return m, nil
}

// handleNavigationUpdate processes comprehensive navigation updates
func (m *Model) handleNavigationUpdate(msg tea.Msg) (tea.Model, tea.Cmd) {
	navMsg := msg.(model.NavigationUpdateMsg)

	if navMsg.NewView != nil {
		m.state.Navigation.View = *navMsg.NewView
	}
	if navMsg.ScopeClusters != nil {
		m.state.Selections.ScopeClusters = navMsg.ScopeClusters
	}
	if navMsg.ScopeNamespaces != nil {
		m.state.Selections.ScopeNamespaces = navMsg.ScopeNamespaces
	}
	if navMsg.ScopeProjects != nil {
		m.state.Selections.ScopeProjects = navMsg.ScopeProjects
	}
	if navMsg.SelectedApps != nil {
		m.state.Selections.SelectedApps = navMsg.SelectedApps
	}
	if navMsg.ShouldResetNavigation {
		m.state.Navigation.SelectedIdx = 0
	}
	return m, nil
}