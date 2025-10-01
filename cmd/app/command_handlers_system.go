package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// System command handlers

// handleHelpCommand handles the "help" command
func (m *Model) handleHelpCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	// Show help modal
	m.state.Mode = model.ModeHelp
	return m, nil
}

// handleQuitCommand handles "quit", "q", "q!", "wq", "wq!", "exit" commands
func (m *Model) handleQuitCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	// Exit the application
	return m, func() tea.Msg { return model.QuitMsg{} }
}

// handleUpgradeCommand handles "upgrade", "update" commands
func (m *Model) handleUpgradeCommand(cmd string, arg string) (tea.Model, tea.Cmd) {
	// Trigger upgrade process
	return m, func() tea.Msg { return model.UpgradeRequestedMsg{} }
}