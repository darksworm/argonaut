package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

const mouseScrollAmount = 3

func (m *Model) handleMouseWheelMsg(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	direction := 0
	switch msg.Button {
	case tea.MouseWheelUp:
		direction = -1
	case tea.MouseWheelDown:
		direction = 1
	default:
		return m, nil
	}

	if direction == 0 {
		return m, nil
	}

	// Theme overlay consumes scrolling events
	if m.state.Mode == model.ModeTheme {
		return m.handleThemeMouseWheel(direction)
	}

	// Block scrolling when non-scrollable overlays are active
	if m.hasBlockingOverlay() {
		return m, nil
	}

	// Diff view is rendered through an external pager (less), so Bubble Tea
	// should not attempt to adjust any offsets here.
	if m.state.Mode == model.ModeDiff {
		return m, nil
	}

	// When tree loading overlay is active, ignore scrolling
	if m.treeLoading {
		return m, nil
	}

	if m.state.Navigation.View == model.ViewTree {
		m.scrollTreeBy(direction * mouseScrollAmount)
	} else {
		m.scrollListBy(direction * mouseScrollAmount)
	}

	return m, nil
}

func (m *Model) handleThemeMouseWheel(direction int) (tea.Model, tea.Cmd) {
	m.ensureThemeOptionsLoaded()
	if len(m.themeOptions) == 0 {
		return m, nil
	}

	steps := mouseScrollAmount
	for i := 0; i < steps; i++ {
		if direction < 0 {
			if m.state.UI.ThemeSelectedIndex <= 0 {
				break
			}
			m.state.UI.ThemeSelectedIndex--
		} else {
			if m.state.UI.ThemeSelectedIndex >= len(m.themeOptions)-1 {
				break
			}
			m.state.UI.ThemeSelectedIndex++
		}
		m.adjustThemeScrollOffset()
		selectedTheme := m.themeOptions[m.state.UI.ThemeSelectedIndex].Name
		m.applyThemePreview(selectedTheme)
	}

	return m, nil
}

func (m *Model) scrollListBy(lines int) {
	if lines == 0 {
		return
	}

	steps := lines
	direction := 1
	if lines < 0 {
		direction = -1
		steps = -lines
	}

	for i := 0; i < steps; i++ {
		if direction < 0 {
			m.handleNavigationUp()
		} else {
			m.handleNavigationDown()
		}
	}
}

func (m *Model) scrollTreeBy(lines int) {
	if lines == 0 || m.treeView == nil {
		return
	}

	steps := lines
	direction := 1
	if lines < 0 {
		direction = -1
		steps = -lines
	}

	for i := 0; i < steps; i++ {
		var key tea.KeyPressMsg
		if direction < 0 {
			key = tea.KeyPressMsg{Text: "k", Code: 'k'}
		} else {
			key = tea.KeyPressMsg{Text: "j", Code: 'j'}
		}

		oldLine := m.treeView.SelectedIndex()
		if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
			oldLine = s.SelectedLineIndex()
		}

		updatedModel, _ := m.treeView.Update(key)
		m.treeView = updatedModel.(*treeview.TreeView)

		newLine := m.treeView.SelectedIndex()
		if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
			newLine = s.SelectedLineIndex()
		}

		if newLine == oldLine {
			break
		}

		viewportHeight := m.treeViewportHeight()
		if viewportHeight <= 0 {
			m.treeScrollOffset = 0
			continue
		}

		if direction < 0 {
			if newLine < m.treeScrollOffset {
				m.treeScrollOffset = newLine
			}
		} else {
			if newLine >= m.treeScrollOffset+viewportHeight {
				m.treeScrollOffset = newLine - viewportHeight + 1
			}
		}
	}
}

func (m *Model) hasBlockingOverlay() bool {
	switch m.state.Mode {
	case model.ModeHelp,
		model.ModeConfirmSync,
		model.ModeRollback,
		model.ModeConfirmAppDelete,
		model.ModeUpgrade,
		model.ModeUpgradeError,
		model.ModeUpgradeSuccess,
		model.ModeNoDiff,
		model.ModeLoading,
		model.ModeAuthRequired,
		model.ModeError,
		model.ModeConnectionError:
		return true
	}
	if m.state.Diff != nil && m.state.Diff.Loading {
		return true
	}
	return false
}
