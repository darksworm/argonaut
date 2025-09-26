package main

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/theme"
)

// applyTheme updates global color variables and derived styles used
// throughout the TUI. Call this early at startup and whenever the theme
// changes.
func applyTheme(p theme.Palette) {
	// Base colors
	magentaBright = p.Accent
	yellowBright = p.Warning
	dimColor = p.Dim
	syncedColor = p.Success
	outOfSyncColor = p.Danger
	progressColor = p.Progress
	unknownColor = p.Unknown
	cyanBright = p.Info
	whiteBright = p.Text

	// Rebuild frequently used styles so they pick up new colors
	contentBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(yellowBright)
	selectedStyle = lipgloss.NewStyle().Background(p.SelectedBG)
	cursorOnSelectedStyle = lipgloss.NewStyle().Background(p.CursorSelectedBG)
	statusStyle = lipgloss.NewStyle().Foreground(dimColor)
}
