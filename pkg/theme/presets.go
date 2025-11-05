package theme

import (
	"image/color"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/config"
)

// Preset palettes inspired by popular themes.
var presets = map[string]Palette{
	"dracula": {
		Accent:           lipgloss.Color("#bd93f9"),
		Warning:          lipgloss.Color("#f1fa8c"),
		Dim:              lipgloss.Color("#6272a4"),
		Success:          lipgloss.Color("#50fa7b"),
		Danger:           lipgloss.Color("#ff5555"),
		Progress:         lipgloss.Color("#f1fa8c"),
		Unknown:          lipgloss.Color("#6272a4"),
		Info:             lipgloss.Color("#8be9fd"),
		Text:             lipgloss.Color("#f8f8f2"),
		Gray:             lipgloss.Color("#6272a4"),
		SelectedBG:       lipgloss.Color("#bd93f9"),
		CursorSelectedBG: lipgloss.Color("#8be9fd"),
		CursorBG:         lipgloss.Color("#8be9fd"),
		Border:           lipgloss.Color("#bd93f9"),
		MutedBG:          lipgloss.Color("#44475a"),
		ShadeBG:          lipgloss.Color("#3a3c4e"),
		DarkBG:           lipgloss.Color("#282a36"),
	},
	"nord": {
		Accent:           lipgloss.Color("#81a1c1"),
		Warning:          lipgloss.Color("#ebcb8b"),
		Dim:              lipgloss.Color("#4c566a"),
		Success:          lipgloss.Color("#a3be8c"),
		Danger:           lipgloss.Color("#bf616a"),
		Progress:         lipgloss.Color("#ebcb8b"),
		Unknown:          lipgloss.Color("#4c566a"),
		Info:             lipgloss.Color("#88c0d0"),
		Text:             lipgloss.Color("#eceff4"),
		Gray:             lipgloss.Color("#4c566a"),
		SelectedBG:       lipgloss.Color("#81a1c1"),
		CursorSelectedBG: lipgloss.Color("#88c0d0"),
		CursorBG:         lipgloss.Color("#88c0d0"),
		Border:           lipgloss.Color("#81a1c1"),
		MutedBG:          lipgloss.Color("#3b4252"),
		ShadeBG:          lipgloss.Color("#434c5e"),
		DarkBG:           lipgloss.Color("#2e3440"),
	},
	"gruvbox": {
		Accent:           lipgloss.Color("#d3869b"),
		Warning:          lipgloss.Color("#fabd2f"),
		Dim:              lipgloss.Color("#928374"),
		Success:          lipgloss.Color("#b8bb26"),
		Danger:           lipgloss.Color("#fb4934"),
		Progress:         lipgloss.Color("#fabd2f"),
		Unknown:          lipgloss.Color("#928374"),
		Info:             lipgloss.Color("#83a598"),
		Text:             lipgloss.Color("#ebdbb2"),
		Gray:             lipgloss.Color("#928374"),
		SelectedBG:       lipgloss.Color("#d3869b"),
		CursorSelectedBG: lipgloss.Color("#83a598"),
		CursorBG:         lipgloss.Color("#83a598"),
		Border:           lipgloss.Color("#d3869b"),
		MutedBG:          lipgloss.Color("#3c3836"),
		ShadeBG:          lipgloss.Color("#504945"),
		DarkBG:           lipgloss.Color("#282828"),
	},
	"solarized-dark": {
		Accent:           lipgloss.Color("#6c71c4"),
		Warning:          lipgloss.Color("#b58900"),
		Dim:              lipgloss.Color("#586e75"),
		Success:          lipgloss.Color("#859900"),
		Danger:           lipgloss.Color("#dc322f"),
		Progress:         lipgloss.Color("#b58900"),
		Unknown:          lipgloss.Color("#586e75"),
		Info:             lipgloss.Color("#2aa198"),
		Text:             lipgloss.Color("#93a1a1"),
		Gray:             lipgloss.Color("#586e75"),
		SelectedBG:       lipgloss.Color("#6c71c4"),
		CursorSelectedBG: lipgloss.Color("#2aa198"),
		CursorBG:         lipgloss.Color("#2aa198"),
		Border:           lipgloss.Color("#6c71c4"),
		MutedBG:          lipgloss.Color("#073642"),
		ShadeBG:          lipgloss.Color("#0a3942"),
		DarkBG:           lipgloss.Color("#002b36"),
	},
	"one-dark": {
		Accent:           lipgloss.Color("#c678dd"),
		Warning:          lipgloss.Color("#e5c07b"),
		Dim:              lipgloss.Color("#5c6370"),
		Success:          lipgloss.Color("#98c379"),
		Danger:           lipgloss.Color("#e06c75"),
		Progress:         lipgloss.Color("#e5c07b"),
		Unknown:          lipgloss.Color("#5c6370"),
		Info:             lipgloss.Color("#56b6c2"),
		Text:             lipgloss.Color("#abb2bf"),
		Gray:             lipgloss.Color("#5c6370"),
		SelectedBG:       lipgloss.Color("#c678dd"),
		CursorSelectedBG: lipgloss.Color("#56b6c2"),
		CursorBG:         lipgloss.Color("#56b6c2"),
		Border:           lipgloss.Color("#c678dd"),
		MutedBG:          lipgloss.Color("#3e4451"),
		ShadeBG:          lipgloss.Color("#2c313a"),
		DarkBG:           lipgloss.Color("#282c34"),
	},
	"oxocarbon": {
		Accent:           lipgloss.Color("#be95ff"),
		Warning:          lipgloss.Color("#f1c21b"),
		Dim:              lipgloss.Color("#8d8d8d"),
		Success:          lipgloss.Color("#42be65"),
		Danger:           lipgloss.Color("#fa4d56"),
		Progress:         lipgloss.Color("#f1c21b"),
		Unknown:          lipgloss.Color("#8d8d8d"),
		Info:             lipgloss.Color("#3ddbd9"),
		Text:             lipgloss.Color("#f2f4f8"),
		Gray:             lipgloss.Color("#8d8d8d"),
		SelectedBG:       lipgloss.Color("#be95ff"),
		CursorSelectedBG: lipgloss.Color("#3ddbd9"),
		CursorBG:         lipgloss.Color("#3ddbd9"),
		Border:           lipgloss.Color("#be95ff"),
		MutedBG:          lipgloss.Color("#262626"),
		ShadeBG:          lipgloss.Color("#393939"),
		DarkBG:           lipgloss.Color("#161616"),
	},
	"catppuccin-mocha": {
		Accent:           lipgloss.Color("#cba6f7"),
		Warning:          lipgloss.Color("#f9e2af"),
		Dim:              lipgloss.Color("#7f849c"),
		Success:          lipgloss.Color("#a6e3a1"),
		Danger:           lipgloss.Color("#f38ba8"),
		Progress:         lipgloss.Color("#f9e2af"),
		Unknown:          lipgloss.Color("#7f849c"),
		Info:             lipgloss.Color("#94e2d5"),
		Text:             lipgloss.Color("#cdd6f4"),
		Gray:             lipgloss.Color("#7f849c"),
		SelectedBG:       lipgloss.Color("#cba6f7"),
		CursorSelectedBG: lipgloss.Color("#94e2d5"),
		CursorBG:         lipgloss.Color("#94e2d5"),
		Border:           lipgloss.Color("#cba6f7"),
		MutedBG:          lipgloss.Color("#313244"),
		ShadeBG:          lipgloss.Color("#45475a"),
		DarkBG:           lipgloss.Color("#1e1e2e"),
	},
}

// Names returns sorted preset names.
func Names() []string {
	out := make([]string, 0, len(presets))
	for k := range presets {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// FromName returns a preset by name, or the default theme (oxocarbon) if unknown.
func FromName(name string) Palette {
	if p, ok := presets[strings.ToLower(name)]; ok {
		return p
	}
	// Use the configured default theme instead of fallback
	if p, ok := presets[config.DefaultThemeName]; ok {
		return p
	}
	// Ultimate fallback to hardcoded default
	return Default()
}

// Get returns a preset and whether it exists.
func Get(name string) (Palette, bool) {
	p, ok := presets[strings.ToLower(name)]
	return p, ok
}

// Colors exposes palette colors in case external packages require image/color.
func Colors(p Palette) map[string]color.Color {
	return map[string]color.Color{
		"accent":    p.Accent,
		"warning":   p.Warning,
		"dim":       p.Dim,
		"success":   p.Success,
		"danger":    p.Danger,
		"progress":  p.Progress,
		"unknown":   p.Unknown,
		"info":      p.Info,
		"text":      p.Text,
		"gray":      p.Gray,
		"selected":  p.SelectedBG,
		"cursorSel": p.CursorSelectedBG,
		"border":    p.Border,
		"muted":     p.MutedBG,
		"shade":     p.ShadeBG,
		"dark":      p.DarkBG,
	}
}