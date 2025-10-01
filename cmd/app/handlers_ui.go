package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// UI state message handlers

// handleSetSearchQuery updates the search query
func (m *Model) handleSetSearchQuery(msg tea.Msg) (tea.Model, tea.Cmd) {
	searchMsg := msg.(model.SetSearchQueryMsg)
	m.state.UI.SearchQuery = searchMsg.Query
	return m, nil
}

// handleSetActiveFilter updates the active filter
func (m *Model) handleSetActiveFilter(msg tea.Msg) (tea.Model, tea.Cmd) {
	filterMsg := msg.(model.SetActiveFilterMsg)
	m.state.UI.ActiveFilter = filterMsg.Filter
	return m, nil
}

// handleSetCommand updates the command input
func (m *Model) handleSetCommand(msg tea.Msg) (tea.Model, tea.Cmd) {
	commandMsg := msg.(model.SetCommandMsg)
	m.state.UI.Command = commandMsg.Command
	return m, nil
}

// handleClearFilters clears all active filters
func (m *Model) handleClearFilters(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.state.UI.SearchQuery = ""
	m.state.UI.ActiveFilter = ""
	return m, nil
}

// handleClearAllSelections clears all user selections
func (m *Model) handleClearAllSelections(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.state.Selections = *model.NewSelectionState()
	return m, nil
}

// handleSetAPIVersion updates the API version information
func (m *Model) handleSetAPIVersion(msg tea.Msg) (tea.Model, tea.Cmd) {
	apiMsg := msg.(model.SetAPIVersionMsg)
	m.state.APIVersion = apiMsg.Version
	return m, nil
}