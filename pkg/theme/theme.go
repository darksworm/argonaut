package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/config"
)

// Palette defines the core colors used by the TUI. It uses
// lipgloss.TerminalColor so themes can be ANSI indices, truecolor hex values,
// or AdaptiveColor depending on the terminal.
type Palette struct {
	// Accents and roles
	Accent  color.Color // primary accent (selection/background)
	Warning color.Color // headings, hints
	Dim     color.Color // subtle text

	// Status colors
	Success  color.Color // Healthy/Synced
	Danger   color.Color // OutOfSync/Degraded
	Progress color.Color // Progressing
	Unknown  color.Color // Unknown/neutral

	// Additional accents
	Info color.Color // cyan accents
	Text color.Color // bright white text
	Gray color.Color // gray

	// Specific backgrounds
	SelectedBG       color.Color // selected row bg (usually == Accent)
	CursorSelectedBG color.Color // cursor on selected row bg
	CursorBG         color.Color // cursor on unselected row bg
	Border           color.Color // border color

	// Neutrals/backgrounds
	MutedBG color.Color // low-contrast background (e.g., inactive buttons)
	ShadeBG color.Color // subtle row highlight background
	DarkBG  color.Color // dark panel background when needed
}

// NewPalette creates a new palette with all required colors.
// This function enforces that all color fields are provided at compile time.
func NewPalette(
	accent, warning, dim color.Color,
	success, danger, progress, unknown color.Color,
	info, text, gray color.Color,
	selectedBG, cursorSelectedBG, cursorBG, border color.Color,
	mutedBG, shadeBG, darkBG color.Color,
) Palette {
	return Palette{
		Accent:           accent,
		Warning:          warning,
		Dim:             dim,
		Success:         success,
		Danger:          danger,
		Progress:        progress,
		Unknown:         unknown,
		Info:            info,
		Text:            text,
		Gray:            gray,
		SelectedBG:      selectedBG,
		CursorSelectedBG: cursorSelectedBG,
		CursorBG:        cursorBG,
		Border:          border,
		MutedBG:         mutedBG,
		ShadeBG:         shadeBG,
		DarkBG:          darkBG,
	}
}

// Default returns the stock palette matching the previous hardcoded colors.
// This is kept for backward compatibility and as a fallback.
func Default() Palette {
	return NewPalette(
		lipgloss.Color("13"), lipgloss.Color("11"), lipgloss.Color("8"), // accent, warning, dim
		lipgloss.Color("10"), lipgloss.Color("9"), lipgloss.Color("11"), lipgloss.Color("8"), // success, danger, progress, unknown
		lipgloss.Color("14"), lipgloss.Color("15"), lipgloss.Color("8"), // info, text, gray
		lipgloss.Color("13"), lipgloss.Color("14"), lipgloss.Color("14"), lipgloss.Color("13"), // selectedBG, cursorSelectedBG, cursorBG, border
		lipgloss.Color("238"), lipgloss.Color("240"), lipgloss.Color("0"), // mutedBG, shadeBG, darkBG
	)
}

// FromConfig creates a palette from the Argonaut configuration.
// It handles built-in presets, custom themes, and overrides.
func FromConfig(cfg *config.ArgonautConfig) Palette {
	var base Palette

	// Start with the configured theme
	base = FromName(cfg.Appearance.Theme)

	// Apply any overrides
	if cfg.Appearance.Overrides != nil {
		base = applyOverrides(base, cfg.Appearance.Overrides)
	}

	return base
}


// applyOverrides applies color overrides to a palette
func applyOverrides(base Palette, overrides map[string]string) Palette {
	for key, value := range overrides {
		color := lipgloss.Color(value)
		switch key {
		case "accent":
			base.Accent = color
			base.SelectedBG = color // Keep them in sync by default
		case "warning":
			base.Warning = color
		case "dim":
			base.Dim = color
		case "success":
			base.Success = color
		case "danger":
			base.Danger = color
		case "progress":
			base.Progress = color
		case "unknown":
			base.Unknown = color
		case "info":
			base.Info = color
		case "text":
			base.Text = color
		case "gray":
			base.Gray = color
		case "selected_bg":
			base.SelectedBG = color
		case "cursor_selected_bg":
			base.CursorSelectedBG = color
		case "cursor_bg":
			base.CursorBG = color
		case "border":
			base.Border = color
		case "muted_bg":
			base.MutedBG = color
		case "shade_bg":
			base.ShadeBG = color
		case "dark_bg":
			base.DarkBG = color
		}
	}
	return base
}

// GetAvailableThemes returns all available preset theme names
func GetAvailableThemes() []string {
	return Names()
}

