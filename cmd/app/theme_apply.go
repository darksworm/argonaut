package main

import (
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/theme"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// applyTheme updates global color variables and derived styles used
// throughout the TUI. Call this early at startup and whenever the theme
// changes.
func applyTheme(p theme.Palette) {
	// Defensive defaults for optional fields
	if p.CursorBG == nil {
		p.CursorBG = p.CursorSelectedBG
	}
	if p.Border == nil {
		p.Border = p.Accent
	}
	if p.SelectedBG == nil {
		p.SelectedBG = p.Accent
	}
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
	cursorStyle = lipgloss.NewStyle().Background(p.CursorBG)
	statusStyle = lipgloss.NewStyle().Foreground(dimColor)

	// Neutral backgrounds
	mutedBG = p.MutedBG
	shadeBG = p.ShadeBG
	darkBG = p.DarkBG

	// Propagate to tree view package
	treeview.ApplyTheme(p)
}
