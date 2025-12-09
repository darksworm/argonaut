package main

import (
	"image/color"
	"math"

	"charm.land/lipgloss/v2"
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

	// Derived foreground colors for themed backgrounds
	textOnSelected       color.Color
	textOnCursorSelected color.Color
	textOnAccent         color.Color
	textOnInfo           color.Color
	textOnDanger         color.Color

	// Modal-specific colors
	keycapBG   color.Color
	spinnerBG  color.Color
	neutralBG  color.Color
	inactiveBG color.Color

	// Table colors
	tableBorderColor  color.Color
	tableSelectedFG   color.Color
	tableSelectedBG   color.Color
)

// applyTheme updates global color variables and derived styles used
// throughout the TUI. Call this early at startup and whenever the theme
// changes.
func applyTheme(p theme.Palette) {
	// Store the current palette globally
	currentPalette = p

	// Defensive defaults for optional fields
	if p.CursorSelectedBG == nil {
		p.CursorSelectedBG = p.Accent
	}
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
	white = whiteBright
	redColor = outOfSyncColor

	// Update help colors
	helpTitleColor = p.Info
	helpSectionColor = p.Warning
	helpHighlightColor = p.Success
	helpTextColor = p.Text
	helpDimColor = p.Dim
	helpUrlColor = p.Accent

	// Store background colors
	mutedBG = p.MutedBG
	shadeBG = p.ShadeBG
	darkBG = p.DarkBG

	// Calculate high-contrast foregrounds for key backgrounds
	textOnSelected = ensureContrastingForeground(p.SelectedBG, p.Text)
	textOnCursorSelected = ensureContrastingForeground(p.CursorSelectedBG, textOnSelected)

	// Update modal colors
	keycapBG = p.MutedBG
	spinnerBG = p.DarkBG
	neutralBG = p.ShadeBG
	inactiveBG = p.MutedBG

	// Update table colors
	tableBorderColor = p.Border
	tableSelectedFG = textOnSelected
	tableSelectedBG = p.SelectedBG
	textOnAccent = ensureContrastingForeground(p.Accent, p.Text)
	textOnInfo = ensureContrastingForeground(p.Info, p.Text)
	textOnDanger = ensureContrastingForeground(p.Danger, p.Text)

	// Rebuild frequently used styles so they pick up new colors
	contentBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(p.Border).
		PaddingLeft(1).
		PaddingRight(1)

	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(yellowBright)
	selectedStyle = lipgloss.NewStyle().
		Background(p.SelectedBG).
		Foreground(textOnSelected)
	statusStyle = lipgloss.NewStyle().Foreground(dimColor)

	// TODO: Update other styles that depend on theme colors
	cursorOnSelectedStyle = lipgloss.NewStyle().
		Background(p.CursorSelectedBG).
		Foreground(textOnCursorSelected)
	// cursorStyle = lipgloss.NewStyle().Background(p.CursorBG)

	// Refresh flash uses a success/green-ish highlight
	refreshFlashStyle = lipgloss.NewStyle().
		Background(p.Success).
		Foreground(textOnSelected)
}

// applyThemeToModel applies the current theme to model components that need it
func (m *Model) applyThemeToModel() {
	if m.treeView != nil {
		m.treeView.ApplyTheme(currentPalette)
	}
}

const wcagAAContrast = 4.5

var (
	lightFallback = lipgloss.Color("#ffffff")
	darkFallback  = lipgloss.Color("#000000")
)

func ensureContrastingForeground(bg color.Color, desired color.Color) color.Color {
	if desired == nil {
		desired = lightFallback
	}
	if bg == nil {
		return desired
	}

	if contrastRatio(bg, desired) >= wcagAAContrast {
		return desired
	}

	lightRatio := contrastRatio(bg, lightFallback)
	darkRatio := contrastRatio(bg, darkFallback)
	if lightRatio >= darkRatio {
		return lightFallback
	}
	return darkFallback
}

func contrastRatio(a, b color.Color) float64 {
	la := relativeLuminance(a)
	lb := relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c color.Color) float64 {
	if c == nil {
		return 0
	}

	r, g, b, _ := c.RGBA()
	rf := srgbToLinear(float64(r) / 65535.0)
	gf := srgbToLinear(float64(g) / 65535.0)
	bf := srgbToLinear(float64(b) / 65535.0)
	return 0.2126*rf + 0.7152*gf + 0.0722*bf
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}
