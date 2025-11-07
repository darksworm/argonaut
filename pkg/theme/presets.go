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
		Accent: lipgloss.Color("#bd93f9"), Warning: lipgloss.Color("#f1fa8c"), Dim: lipgloss.Color("#6272a4"), Success: lipgloss.Color("#50fa7b"), Danger: lipgloss.Color("#ff5555"),
		Progress: lipgloss.Color("#f1fa8c"), Unknown: lipgloss.Color("#6272a4"), Info: lipgloss.Color("#8be9fd"), Text: lipgloss.Color("#f8f8f2"), Gray: lipgloss.Color("#6272a4"),
		SelectedBG: lipgloss.Color("#bd93f9"), CursorSelectedBG: lipgloss.Color("#8be9fd"), CursorBG: lipgloss.Color("#8be9fd"), Border: lipgloss.Color("#bd93f9"),
		MutedBG: lipgloss.Color("#44475a"), ShadeBG: lipgloss.Color("#3a3c4e"), DarkBG: lipgloss.Color("#282a36"),
	},

	// --- Solarized ---
	"solarized-dark": {
		Accent: lipgloss.Color("#6c71c4"), Warning: lipgloss.Color("#b58900"), Dim: lipgloss.Color("#586e75"), Success: lipgloss.Color("#859900"), Danger: lipgloss.Color("#dc322f"),
		Progress: lipgloss.Color("#b58900"), Unknown: lipgloss.Color("#586e75"), Info: lipgloss.Color("#2aa198"), Text: lipgloss.Color("#93a1a1"), Gray: lipgloss.Color("#586e75"),
		SelectedBG: lipgloss.Color("#073642"), CursorSelectedBG: lipgloss.Color("#2aa198"), CursorBG: lipgloss.Color("#2aa198"), Border: lipgloss.Color("#073642"),
		MutedBG: lipgloss.Color("#073642"), ShadeBG: lipgloss.Color("#002b36"), DarkBG: lipgloss.Color("#002b36"),
	},
	"solarized-light": {
		Accent: lipgloss.Color("#6c71c4"), Warning: lipgloss.Color("#b58900"), Dim: lipgloss.Color("#93a1a1"), Success: lipgloss.Color("#859900"), Danger: lipgloss.Color("#dc322f"),
		Progress: lipgloss.Color("#b58900"), Unknown: lipgloss.Color("#93a1a1"), Info: lipgloss.Color("#2aa198"), Text: lipgloss.Color("#657b83"), Gray: lipgloss.Color("#93a1a1"),
		SelectedBG: lipgloss.Color("#eee8d5"), CursorSelectedBG: lipgloss.Color("#2aa198"), CursorBG: lipgloss.Color("#2aa198"), Border: lipgloss.Color("#93a1a1"),
		MutedBG: lipgloss.Color("#eee8d5"), ShadeBG: lipgloss.Color("#fdf6e3"), DarkBG: lipgloss.Color("#fdf6e3"),
	},

	// --- Gruvbox ---
	"gruvbox-dark": {
		Accent: lipgloss.Color("#d79921"), Warning: lipgloss.Color("#d79921"), Dim: lipgloss.Color("#a89984"), Success: lipgloss.Color("#98971a"), Danger: lipgloss.Color("#cc241d"),
		Progress: lipgloss.Color("#d79921"), Unknown: lipgloss.Color("#928374"), Info: lipgloss.Color("#458588"), Text: lipgloss.Color("#ebdbb2"), Gray: lipgloss.Color("#928374"),
		SelectedBG: lipgloss.Color("#3c3836"), CursorSelectedBG: lipgloss.Color("#83a598"), CursorBG: lipgloss.Color("#83a598"), Border: lipgloss.Color("#504945"),
		MutedBG: lipgloss.Color("#3c3836"), ShadeBG: lipgloss.Color("#32302f"), DarkBG: lipgloss.Color("#282828"),
	},
	"gruvbox-light": {
		Accent: lipgloss.Color("#b57614"), Warning: lipgloss.Color("#b57614"), Dim: lipgloss.Color("#7c6f64"), Success: lipgloss.Color("#79740e"), Danger: lipgloss.Color("#9d0006"),
		Progress: lipgloss.Color("#b57614"), Unknown: lipgloss.Color("#928374"), Info: lipgloss.Color("#076678"), Text: lipgloss.Color("#3c3836"), Gray: lipgloss.Color("#928374"),
		SelectedBG: lipgloss.Color("#d5c4a1"), CursorSelectedBG: lipgloss.Color("#076678"), CursorBG: lipgloss.Color("#076678"), Border: lipgloss.Color("#bdae93"),
		MutedBG: lipgloss.Color("#f2e5bc"), ShadeBG: lipgloss.Color("#fbf1c7"), DarkBG: lipgloss.Color("#fbf1c7"),
	},

	// --- Nord ---
	"nord": {
		Accent: lipgloss.Color("#88c0d0"), Warning: lipgloss.Color("#ebcb8b"), Dim: lipgloss.Color("#4c566a"), Success: lipgloss.Color("#a3be8c"), Danger: lipgloss.Color("#bf616a"),
		Progress: lipgloss.Color("#ebcb8b"), Unknown: lipgloss.Color("#4c566a"), Info: lipgloss.Color("#81a1c1"), Text: lipgloss.Color("#e5e9f0"), Gray: lipgloss.Color("#4c566a"),
		SelectedBG: lipgloss.Color("#3b4252"), CursorSelectedBG: lipgloss.Color("#88c0d0"), CursorBG: lipgloss.Color("#88c0d0"), Border: lipgloss.Color("#434c5e"),
		MutedBG: lipgloss.Color("#3b4252"), ShadeBG: lipgloss.Color("#2e3440"), DarkBG: lipgloss.Color("#2e3440"),
	},

	// --- One (Atom/VS Code) ---
	"one-dark": {
		Accent: lipgloss.Color("#61afef"), Warning: lipgloss.Color("#e5c07b"), Dim: lipgloss.Color("#5c6370"), Success: lipgloss.Color("#98c379"), Danger: lipgloss.Color("#e06c75"),
		Progress: lipgloss.Color("#e5c07b"), Unknown: lipgloss.Color("#5c6370"), Info: lipgloss.Color("#56b6c2"), Text: lipgloss.Color("#abb2bf"), Gray: lipgloss.Color("#5c6370"),
		SelectedBG: lipgloss.Color("#3e4451"), CursorSelectedBG: lipgloss.Color("#528bff"), CursorBG: lipgloss.Color("#528bff"), Border: lipgloss.Color("#3e4451"),
		MutedBG: lipgloss.Color("#2c313c"), ShadeBG: lipgloss.Color("#21252b"), DarkBG: lipgloss.Color("#282c34"),
	},
	"one-light": {
		Accent: lipgloss.Color("#4078f2"), Warning: lipgloss.Color("#c18401"), Dim: lipgloss.Color("#a0a1a7"), Success: lipgloss.Color("#50a14f"), Danger: lipgloss.Color("#e45649"),
		Progress: lipgloss.Color("#c18401"), Unknown: lipgloss.Color("#a0a1a7"), Info: lipgloss.Color("#0184bc"), Text: lipgloss.Color("#383a42"), Gray: lipgloss.Color("#a0a1a7"),
		SelectedBG: lipgloss.Color("#e5eaf0"), CursorSelectedBG: lipgloss.Color("#4078f2"), CursorBG: lipgloss.Color("#4078f2"), Border: lipgloss.Color("#d0d0d0"),
		MutedBG: lipgloss.Color("#f3f3f3"), ShadeBG: lipgloss.Color("#f7f7f7"), DarkBG: lipgloss.Color("#fafafa"),
	},

	// --- Monokai ---
	"monokai": {
		Accent: lipgloss.Color("#ae81ff"), Warning: lipgloss.Color("#e6db74"), Dim: lipgloss.Color("#75715e"), Success: lipgloss.Color("#a6e22e"), Danger: lipgloss.Color("#f92672"),
		Progress: lipgloss.Color("#e6db74"), Unknown: lipgloss.Color("#75715e"), Info: lipgloss.Color("#66d9ef"), Text: lipgloss.Color("#f8f8f2"), Gray: lipgloss.Color("#75715e"),
		SelectedBG: lipgloss.Color("#49483e"), CursorSelectedBG: lipgloss.Color("#66d9ef"), CursorBG: lipgloss.Color("#66d9ef"), Border: lipgloss.Color("#75715e"),
		MutedBG: lipgloss.Color("#3e3d32"), ShadeBG: lipgloss.Color("#2d2e2a"), DarkBG: lipgloss.Color("#272822"),
	},

	// --- Tokyo Night ---
	"tokyo-night": {
		Accent: lipgloss.Color("#bb9af7"), Warning: lipgloss.Color("#e0af68"), Dim: lipgloss.Color("#565f89"), Success: lipgloss.Color("#9ece6a"), Danger: lipgloss.Color("#f7768e"),
		Progress: lipgloss.Color("#e0af68"), Unknown: lipgloss.Color("#565f89"), Info: lipgloss.Color("#7dcfff"), Text: lipgloss.Color("#c0caf5"), Gray: lipgloss.Color("#565f89"),
		SelectedBG: lipgloss.Color("#33467c"), CursorSelectedBG: lipgloss.Color("#7dcfff"), CursorBG: lipgloss.Color("#7dcfff"), Border: lipgloss.Color("#3b4261"),
		MutedBG: lipgloss.Color("#24283b"), ShadeBG: lipgloss.Color("#1f2335"), DarkBG: lipgloss.Color("#1a1b26"),
	},
	"tokyo-storm": {
		Accent: lipgloss.Color("#7aa2f7"), Warning: lipgloss.Color("#e0af68"), Dim: lipgloss.Color("#565f89"), Success: lipgloss.Color("#9ece6a"), Danger: lipgloss.Color("#f7768e"),
		Progress: lipgloss.Color("#e0af68"), Unknown: lipgloss.Color("#565f89"), Info: lipgloss.Color("#7dcfff"), Text: lipgloss.Color("#c0caf5"), Gray: lipgloss.Color("#565f89"),
		SelectedBG: lipgloss.Color("#3b4261"), CursorSelectedBG: lipgloss.Color("#7dcfff"), CursorBG: lipgloss.Color("#7dcfff"), Border: lipgloss.Color("#3b4261"),
		MutedBG: lipgloss.Color("#2f3449"), ShadeBG: lipgloss.Color("#24283b"), DarkBG: lipgloss.Color("#24283b"),
	},

	// --- Catppuccin ---
	"catppuccin-mocha": {
		Accent: lipgloss.Color("#cba6f7"), Warning: lipgloss.Color("#f9e2af"), Dim: lipgloss.Color("#6c7086"), Success: lipgloss.Color("#a6e3a1"), Danger: lipgloss.Color("#f38ba8"),
		Progress: lipgloss.Color("#f9e2af"), Unknown: lipgloss.Color("#6c7086"), Info: lipgloss.Color("#89dceb"), Text: lipgloss.Color("#cdd6f4"), Gray: lipgloss.Color("#6c7086"),
		SelectedBG: lipgloss.Color("#313244"), CursorSelectedBG: lipgloss.Color("#89b4fa"), CursorBG: lipgloss.Color("#89dceb"), Border: lipgloss.Color("#585b70"),
		MutedBG: lipgloss.Color("#313244"), ShadeBG: lipgloss.Color("#181825"), DarkBG: lipgloss.Color("#1e1e2e"),
	},
	"catppuccin-latte": {
		Accent: lipgloss.Color("#8839ef"), Warning: lipgloss.Color("#df8e1d"), Dim: lipgloss.Color("#8c8fa1"), Success: lipgloss.Color("#40a02b"), Danger: lipgloss.Color("#d20f39"),
		Progress: lipgloss.Color("#df8e1d"), Unknown: lipgloss.Color("#8c8fa1"), Info: lipgloss.Color("#04a5e5"), Text: lipgloss.Color("#4c4f69"), Gray: lipgloss.Color("#8c8fa1"),
		SelectedBG: lipgloss.Color("#ccd0da"), CursorSelectedBG: lipgloss.Color("#1e66f5"), CursorBG: lipgloss.Color("#1e66f5"), Border: lipgloss.Color("#acb0be"),
		MutedBG: lipgloss.Color("#e6e9ef"), ShadeBG: lipgloss.Color("#ccd0da"), DarkBG: lipgloss.Color("#eff1f5"),
	},

	// --- Tomorrow / One Half Light (simple, readable) ---
	"onehalf-light": {
		Accent: lipgloss.Color("#4078f2"), Warning: lipgloss.Color("#c18401"), Dim: lipgloss.Color("#a0a1a7"), Success: lipgloss.Color("#50a14f"), Danger: lipgloss.Color("#e45649"),
		Progress: lipgloss.Color("#c18401"), Unknown: lipgloss.Color("#a0a1a7"), Info: lipgloss.Color("#0184bc"), Text: lipgloss.Color("#383a42"), Gray: lipgloss.Color("#a0a1a7"),
		SelectedBG: lipgloss.Color("#e5eaf0"), CursorSelectedBG: lipgloss.Color("#4078f2"), CursorBG: lipgloss.Color("#4078f2"), Border: lipgloss.Color("#d0d0d0"),
		MutedBG: lipgloss.Color("#f2f2f2"), ShadeBG: lipgloss.Color("#f7f7f7"), DarkBG: lipgloss.Color("#fafafa"),
	},

	// --- Accessibility / Utility ---
	"high-contrast": {
		Accent: lipgloss.Color("#00ffff"), Warning: lipgloss.Color("#ffff00"), Dim: lipgloss.Color("#bfbfbf"), Success: lipgloss.Color("#00ff00"), Danger: lipgloss.Color("#ff0033"),
		Progress: lipgloss.Color("#ffff00"), Unknown: lipgloss.Color("#9e9e9e"), Info: lipgloss.Color("#00ffff"), Text: lipgloss.Color("#ffffff"), Gray: lipgloss.Color("#9e9e9e"),
		SelectedBG: lipgloss.Color("#333333"), CursorSelectedBG: lipgloss.Color("#00ffff"), CursorBG: lipgloss.Color("#ffffff"), Border: lipgloss.Color("#ffffff"),
		MutedBG: lipgloss.Color("#1a1a1a"), ShadeBG: lipgloss.Color("#0d0d0d"), DarkBG: lipgloss.Color("#000000"),
	},
	"colorblind-safe": { // Okabe–Ito palette on dark neutral
		Accent: lipgloss.Color("#cc79a7"), Warning: lipgloss.Color("#f0e442"), Dim: lipgloss.Color("#a8a8a8"), Success: lipgloss.Color("#009e73"), Danger: lipgloss.Color("#d55e00"),
		Progress: lipgloss.Color("#f0e442"), Unknown: lipgloss.Color("#8d8d8d"), Info: lipgloss.Color("#56b4e9"), Text: lipgloss.Color("#eaeaea"), Gray: lipgloss.Color("#8d8d8d"),
		SelectedBG: lipgloss.Color("#303030"), CursorSelectedBG: lipgloss.Color("#56b4e9"), CursorBG: lipgloss.Color("#56b4e9"), Border: lipgloss.Color("#a8a8a8"),
		MutedBG: lipgloss.Color("#252525"), ShadeBG: lipgloss.Color("#1e1e1e"), DarkBG: lipgloss.Color("#161616"),
	},
	"grayscale-lowchroma": {
		Accent: lipgloss.Color("#bdbdbd"), Warning: lipgloss.Color("#d0d0d0"), Dim: lipgloss.Color("#a8a8a8"), Success: lipgloss.Color("#c8c8c8"), Danger: lipgloss.Color("#afafaf"),
		Progress: lipgloss.Color("#d0d0d0"), Unknown: lipgloss.Color("#8e8e8e"), Info: lipgloss.Color("#bdbdbd"), Text: lipgloss.Color("#e0e0e0"), Gray: lipgloss.Color("#8e8e8e"),
		SelectedBG: lipgloss.Color("#333333"), CursorSelectedBG: lipgloss.Color("#bdbdbd"), CursorBG: lipgloss.Color("#e0e0e0"), Border: lipgloss.Color("#4d4d4d"),
		MutedBG: lipgloss.Color("#262626"), ShadeBG: lipgloss.Color("#1a1a1a"), DarkBG: lipgloss.Color("#121212"),
	},

	// Keep legacy themes for compatibility
	"gruvbox": {
		Accent: lipgloss.Color("#d79921"), Warning: lipgloss.Color("#d79921"), Dim: lipgloss.Color("#a89984"), Success: lipgloss.Color("#98971a"), Danger: lipgloss.Color("#cc241d"),
		Progress: lipgloss.Color("#d79921"), Unknown: lipgloss.Color("#928374"), Info: lipgloss.Color("#458588"), Text: lipgloss.Color("#ebdbb2"), Gray: lipgloss.Color("#928374"),
		SelectedBG: lipgloss.Color("#3c3836"), CursorSelectedBG: lipgloss.Color("#83a598"), CursorBG: lipgloss.Color("#83a598"), Border: lipgloss.Color("#504945"),
		MutedBG: lipgloss.Color("#3c3836"), ShadeBG: lipgloss.Color("#32302f"), DarkBG: lipgloss.Color("#282828"),
	},
	"oxocarbon": {
		Accent: lipgloss.Color("#be95ff"), Warning: lipgloss.Color("#f1c21b"), Dim: lipgloss.Color("#8d8d8d"), Success: lipgloss.Color("#42be65"), Danger: lipgloss.Color("#fa4d56"),
		Progress: lipgloss.Color("#f1c21b"), Unknown: lipgloss.Color("#8d8d8d"), Info: lipgloss.Color("#3ddbd9"), Text: lipgloss.Color("#f2f4f8"), Gray: lipgloss.Color("#8d8d8d"),
		SelectedBG: lipgloss.Color("#be95ff"), CursorSelectedBG: lipgloss.Color("#3ddbd9"), CursorBG: lipgloss.Color("#3ddbd9"), Border: lipgloss.Color("#be95ff"),
		MutedBG: lipgloss.Color("#262626"), ShadeBG: lipgloss.Color("#393939"), DarkBG: lipgloss.Color("#161616"),
	},


	// Special: use ANSI colors to honor the user's terminal palette
	"inherit-terminal": {
		Accent: lipgloss.Color("13"), Warning: lipgloss.Color("11"), Dim: lipgloss.Color("8"), Success: lipgloss.Color("10"), Danger: lipgloss.Color("9"),
		Progress: lipgloss.Color("11"), Unknown: lipgloss.Color("8"), Info: lipgloss.Color("14"), Text: lipgloss.Color("15"), Gray: lipgloss.Color("8"),
		SelectedBG: lipgloss.Color("13"), CursorSelectedBG: lipgloss.Color("14"), CursorBG: lipgloss.Color("14"), Border: lipgloss.Color("7"),
		MutedBG: lipgloss.Color("238"), ShadeBG: lipgloss.Color("240"), DarkBG: lipgloss.Color("0"),
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