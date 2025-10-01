package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// Mode and application state message handlers

// handleSetMode processes application mode changes
func (m *Model) handleSetMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	setModeMsg := msg.(model.SetModeMsg)

	oldMode := m.state.Mode
	m.state.Mode = setModeMsg.Mode
	cblog.With("component", "model").Info("SetModeMsg received",
		"old_mode", oldMode,
		"new_mode", setModeMsg.Mode)

	// Handle mode transitions
	if setModeMsg.Mode == model.ModeLoading && oldMode != model.ModeLoading {
		cblog.With("component", "model").Info("Triggering initial load for ModeLoading")
		// Start loading applications from API when transitioning to loading mode
		return m, m.startLoadingApplications()
	}

	// If entering diff mode with content available, show in external pager
	if setModeMsg.Mode == model.ModeDiff && m.state.Diff != nil && len(m.state.Diff.Content) > 0 && !m.state.Diff.Loading {
		body := strings.Join(m.state.Diff.Content, "\n")
		return m, m.openTextPager(body)
	}

	return m, nil
}

// handleQuit processes application quit requests
func (m *Model) handleQuit(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, tea.Quit
}

// handleSetInitialLoading sets the initial loading state
func (m *Model) handleSetInitialLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	setLoadingMsg := msg.(model.SetInitialLoadingMsg)
	m.state.Modals.InitialLoading = setLoadingMsg.Loading
	return m, nil
}