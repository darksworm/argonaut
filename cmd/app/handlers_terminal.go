package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/darksworm/argonaut/pkg/model"
)

// Terminal and system message handlers

// handleWindowSize processes terminal resize events
func (m *Model) handleWindowSize(msg tea.Msg) (tea.Model, tea.Cmd) {
	windowMsg := msg.(tea.WindowSizeMsg)

	m.state.Terminal.Rows = windowMsg.Height
	m.state.Terminal.Cols = windowMsg.Width
	if m.treeView != nil {
		m.treeView.SetSize(windowMsg.Width, windowMsg.Height)
	}
	if !m.ready {
		m.ready = true
		return m, func() tea.Msg {
			return model.StatusChangeMsg{Status: "Ready"}
		}
	}
	return m, nil
}

// handleKeyMsgRegistry processes keyboard input for the message registry
func (m *Model) handleKeyMsgRegistry(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg := msg.(tea.KeyMsg)
	// Delegate to the existing handleKeyMsg function in input_handlers.go
	return m.handleKeyMsg(keyMsg)
}

// handleSpinnerTick processes spinner animation updates
func (m *Model) handleSpinnerTick(msg tea.Msg) (tea.Model, tea.Cmd) {
	spinnerMsg := msg.(spinner.TickMsg)

	if m.inPager {
		// Suspend spinner updates while pager owns the terminal
		return m, nil
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(spinnerMsg)
	return m, cmd
}