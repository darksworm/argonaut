package theme

import (
	"image/color"

	"github.com/charmbracelet/lipgloss/v2"
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

// Default returns the stock palette matching the previous hardcoded colors.
// This is kept for backward compatibility and as a fallback.
func Default() Palette {
	return Palette{
		Accent:           lipgloss.Color("13"), // magentaBright
		Warning:          lipgloss.Color("11"),
		Dim:              lipgloss.Color("8"),
		Success:          lipgloss.Color("10"),
		Danger:           lipgloss.Color("9"),
		Progress:         lipgloss.Color("11"),
		Unknown:          lipgloss.Color("8"),
		Info:             lipgloss.Color("14"),
		Text:             lipgloss.Color("15"),
		Gray:             lipgloss.Color("8"),
		SelectedBG:       lipgloss.Color("13"), // same as Accent by default
		CursorSelectedBG: lipgloss.Color("14"), // cyan
		CursorBG:         lipgloss.Color("14"), // cyan by default
		Border:           lipgloss.Color("13"),
		MutedBG:          lipgloss.Color("238"),
		ShadeBG:          lipgloss.Color("240"),
		DarkBG:           lipgloss.Color("0"),
	}
}

// FromConfig creates a palette from the Argonaut configuration.
// It handles built-in presets, custom themes, and overrides.
func FromConfig(cfg *config.ArgonautConfig) Palette {
	var base Palette

	// Start with the configured theme
	switch cfg.Appearance.Theme {
	case "custom":
		base = fromCustomTheme(cfg.Custom)
	default:
		base = FromName(cfg.Appearance.Theme)
	}

	// Apply any overrides
	if cfg.Appearance.Overrides != nil {
		base = applyOverrides(base, cfg.Appearance.Overrides)
	}

	return base
}

// fromCustomTheme creates a palette from custom theme colors
func fromCustomTheme(custom config.CustomTheme) Palette {
	return Palette{
		Accent:           lipgloss.Color(custom.Accent),
		Warning:          lipgloss.Color(custom.Warning),
		Dim:              lipgloss.Color(custom.Dim),
		Success:          lipgloss.Color(custom.Success),
		Danger:           lipgloss.Color(custom.Danger),
		Progress:         lipgloss.Color(custom.Progress),
		Unknown:          lipgloss.Color(custom.Unknown),
		Info:             lipgloss.Color(custom.Info),
		Text:             lipgloss.Color(custom.Text),
		Gray:             lipgloss.Color(custom.Gray),
		SelectedBG:       lipgloss.Color(custom.SelectedBG),
		CursorSelectedBG: lipgloss.Color(custom.CursorSelectedBG),
		CursorBG:         lipgloss.Color(custom.CursorBG),
		Border:           lipgloss.Color(custom.Border),
		MutedBG:          lipgloss.Color(custom.MutedBG),
		ShadeBG:          lipgloss.Color(custom.ShadeBG),
		DarkBG:           lipgloss.Color(custom.DarkBG),
	}
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

// ValidateCustomTheme checks if a custom theme has all required colors
func ValidateCustomTheme(custom config.CustomTheme) error {
	// For now, we'll be lenient and allow missing colors to fallback
	// In the future, we could add stricter validation
	return nil
}