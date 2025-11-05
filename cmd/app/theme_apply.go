package main

import (
	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/theme"
)

// Global variables for storing current theme colors
var (
	// Theme colors (these will be set by applyTheme)
	currentPalette theme.Palette

	// Background colors for special use cases
	mutedBG color.Color
	shadeBG color.Color
	darkBG  color.Color
)

// applyTheme updates global color variables and derived styles used
// throughout the TUI. Call this early at startup and whenever the theme
// changes.
func applyTheme(p theme.Palette) {
	// Store the current palette globally
	currentPalette = p

	// Defensive defaults for optional fields
	if p.CursorBG == nil {
		if p.CursorSelectedBG != nil {
			p.CursorBG = p.CursorSelectedBG
		} else {
			p.CursorBG = p.Info
		}
	}
	if p.Border == nil {
		p.Border = p.Accent
	}
	if p.SelectedBG == nil {
		p.SelectedBG = p.Accent
	}

	// Update base color variables in view.go
	magentaBright = p.Accent
	yellowBright = p.Warning
	dimColor = p.Dim
	syncedColor = p.Success
	outOfSyncColor = p.Danger
	progressColor = p.Progress
	unknownColor = p.Unknown
	cyanBright = p.Info
	whiteBright = p.Text

	// Store background colors
	mutedBG = p.MutedBG
	shadeBG = p.ShadeBG
	darkBG = p.DarkBG

	// Rebuild frequently used styles so they pick up new colors
	contentBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(yellowBright)
	selectedStyle = lipgloss.NewStyle().Background(p.SelectedBG)
	statusStyle = lipgloss.NewStyle().Foreground(dimColor)

	// TODO: Update other styles that depend on theme colors
	// cursorOnSelectedStyle = lipgloss.NewStyle().Background(p.CursorSelectedBG)
	// cursorStyle = lipgloss.NewStyle().Background(p.CursorBG)

	// TODO: Propagate to tree view package when it supports themes
	// treeview.ApplyTheme(p)
}