package theme

import (
	"image/color"
	"os"

	"github.com/charmbracelet/lipgloss/v2"
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
	Border           color.Color // border color

	// Neutrals/backgrounds
	MutedBG color.Color // low-contrast background (e.g., inactive buttons)
	ShadeBG color.Color // subtle row highlight background
	DarkBG  color.Color // dark panel background when needed
}

// Default returns the stock palette matching the previous hardcoded colors.
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
		Border:           lipgloss.Color("13"),
		MutedBG:          lipgloss.Color("238"),
		ShadeBG:          lipgloss.Color("240"),
		DarkBG:           lipgloss.Color("0"),
	}
}

// FromEnv overlays the provided base palette with environment-provided colors.
// Hex values like "#88c0d0" or ANSI numbers like "33" are both supported.
//
// Supported variables:
//
//	ARGONAUT_COLOR_ACCENT
//	ARGONAUT_COLOR_WARNING
//	ARGONAUT_COLOR_DIM
//	ARGONAUT_COLOR_SUCCESS
//	ARGONAUT_COLOR_DANGER
//	ARGONAUT_COLOR_PROGRESS
//	ARGONAUT_COLOR_UNKNOWN
//	ARGONAUT_COLOR_INFO
//	ARGONAUT_COLOR_TEXT
//	ARGONAUT_COLOR_GRAY
//	ARGONAUT_BG_SELECTED
//	ARGONAUT_BG_CURSOR_SELECTED
//	ARGONAUT_COLOR_BORDER
func FromEnv(base Palette) Palette {
	set := func(env string, apply func(color.Color)) {
		if v := os.Getenv(env); v != "" {
			apply(lipgloss.Color(v))
		}
	}

	set("ARGONAUT_COLOR_ACCENT", func(c color.Color) { base.Accent = c; base.SelectedBG = c })
	set("ARGONAUT_COLOR_WARNING", func(c color.Color) { base.Warning = c })
	set("ARGONAUT_COLOR_DIM", func(c color.Color) { base.Dim = c })
	set("ARGONAUT_COLOR_SUCCESS", func(c color.Color) { base.Success = c })
	set("ARGONAUT_COLOR_DANGER", func(c color.Color) { base.Danger = c })
	set("ARGONAUT_COLOR_PROGRESS", func(c color.Color) { base.Progress = c })
	set("ARGONAUT_COLOR_UNKNOWN", func(c color.Color) { base.Unknown = c })
	set("ARGONAUT_COLOR_INFO", func(c color.Color) { base.Info = c })
	set("ARGONAUT_COLOR_TEXT", func(c color.Color) { base.Text = c })
	set("ARGONAUT_COLOR_GRAY", func(c color.Color) { base.Gray = c })
	set("ARGONAUT_BG_SELECTED", func(c color.Color) { base.SelectedBG = c })
	set("ARGONAUT_BG_CURSOR_SELECTED", func(c color.Color) { base.CursorSelectedBG = c })
	set("ARGONAUT_COLOR_BORDER", func(c color.Color) { base.Border = c })
	set("ARGONAUT_BG_MUTED", func(c color.Color) { base.MutedBG = c })
	set("ARGONAUT_BG_SHADE", func(c color.Color) { base.ShadeBG = c })
	set("ARGONAUT_BG_DARK", func(c color.Color) { base.DarkBG = c })
	return base
}
