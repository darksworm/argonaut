package main

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/tui/clipboard"
	"github.com/darksworm/argonaut/pkg/tui/selection"
)

// handleMouseClickMsg processes mouse click (press) events for text selection.
func (m *Model) handleMouseClickMsg(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button == tea.MouseLeft {
		// Start a new selection, clearing any previous one
		m.selection.SetStart(selection.Position{Row: msg.Y, Col: msg.X})
	}
	return m, nil
}

// handleMouseMotionMsg processes mouse motion events for text selection.
func (m *Model) handleMouseMotionMsg(msg tea.MouseMotionMsg) (tea.Model, tea.Cmd) {
	// Only update selection if we're actively selecting
	if m.selection.Active {
		m.selection.SetEnd(selection.Position{Row: msg.Y, Col: msg.X})

		// Check if button was released (motion with no button pressed)
		// Some terminals don't send MouseReleaseMsg but do send motion with MouseNone
		if msg.Button == tea.MouseNone {
			return m.finalizeSelection()
		}
	}
	return m, nil
}

// handleMouseReleaseMsg processes mouse release events for text selection.
func (m *Model) handleMouseReleaseMsg(msg tea.MouseReleaseMsg) (tea.Model, tea.Cmd) {
	if m.selection.Active {
		m.selection.SetEnd(selection.Position{Row: msg.Y, Col: msg.X})
		return m.finalizeSelection()
	}
	return m, nil
}

// finalizeSelection completes the selection and copies to clipboard.
func (m *Model) finalizeSelection() (tea.Model, tea.Cmd) {
	// Finalize the selection
	if m.selection.Finalize() {
		// Extract selected text and copy to clipboard
		text := m.selection.ExtractText(m.lastRenderedLines)

		// Clear selection
		m.selection.Clear()

		if text != "" {
			// Return command to copy to clipboard and show status
			return m, tea.Batch(
				clipboard.CopyCmd(text),
				m.showCopiedStatus(),
			)
		}
	}
	// Clear selection
	m.selection.Clear()
	return m, nil
}

// showCopiedStatus displays a brief "Copied!" message in the status bar.
func (m *Model) showCopiedStatus() tea.Cmd {
	m.state.UI.SelectionCopied = true

	// Clear the copied status after a short delay (1.5 seconds)
	return tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
		return clearCopiedStatusMsg{}
	})
}

// clearCopiedStatusMsg is sent to clear the "Copied!" status message.
type clearCopiedStatusMsg struct{}
