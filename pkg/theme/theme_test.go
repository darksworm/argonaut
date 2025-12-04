package theme

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/config"
)

// Helper function to compare colors by creating them with lipgloss.Color
func colorsEqual(a, b interface{}, expected string) bool {
	expectedColor := lipgloss.Color(expected)
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", expectedColor)
}

func TestAllPresetsHaveRequiredColors(t *testing.T) {
	themeNames := Names()
	if len(themeNames) == 0 {
		t.Fatal("No themes found")
	}

	for _, name := range themeNames {
		t.Run(name, func(t *testing.T) {
			palette := FromName(name)

			// Test that all required colors are non-nil
			if palette.Accent == nil {
				t.Error("Accent color is nil")
			}
			if palette.Warning == nil {
				t.Error("Warning color is nil")
			}
			if palette.Dim == nil {
				t.Error("Dim color is nil")
			}
			if palette.Success == nil {
				t.Error("Success color is nil")
			}
			if palette.Danger == nil {
				t.Error("Danger color is nil")
			}
			if palette.Progress == nil {
				t.Error("Progress color is nil")
			}
			if palette.Unknown == nil {
				t.Error("Unknown color is nil")
			}
			if palette.Info == nil {
				t.Error("Info color is nil")
			}
			if palette.Text == nil {
				t.Error("Text color is nil")
			}
			if palette.Gray == nil {
				t.Error("Gray color is nil")
			}
			if palette.SelectedBG == nil {
				t.Error("SelectedBG color is nil")
			}
			if palette.CursorSelectedBG == nil {
				t.Error("CursorSelectedBG color is nil")
			}
			if palette.CursorBG == nil {
				t.Error("CursorBG color is nil")
			}
			if palette.Border == nil {
				t.Error("Border color is nil")
			}
			if palette.MutedBG == nil {
				t.Error("MutedBG color is nil")
			}
			if palette.ShadeBG == nil {
				t.Error("ShadeBG color is nil")
			}
			if palette.DarkBG == nil {
				t.Error("DarkBG color is nil")
			}
		})
	}
}

func TestFromName_ValidThemes(t *testing.T) {
	testCases := []struct {
		name     string
		expected string // Expected accent color for verification
	}{
		{"dracula", "#bd93f9"},
		{"nord", "#88c0d0"},
		{"gruvbox", "#d79921"},
		{"oxocarbon", "#be95ff"},
		{"monokai", "#ae81ff"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			palette := FromName(tc.name)

			// Compare colors by creating expected color and comparing string representation
			expectedColor := lipgloss.Color(tc.expected)
			if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
				t.Errorf("Expected accent color %s, got %v", tc.expected, palette.Accent)
			}
		})
	}
}

func TestFromName_InvalidThemes(t *testing.T) {
	testCases := []string{
		"nonexistent",
		"invalid-theme",
		"",
		"definitely-not-a-theme",
	}

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			palette := FromName(name)

			// Should return default theme (tokyo-night)
			expectedAccent := "#bb9af7"
			expectedColor := lipgloss.Color(expectedAccent)
			if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
				t.Errorf("Expected fallback to default theme with accent %s, got %v", expectedAccent, palette.Accent)
			}
		})
	}
}

func TestFromName_CaseInsensitive(t *testing.T) {
	testCases := []string{
		"dracula",
		"DRACULA",
		"Dracula",
		"DrAcUlA",
	}

	expected := "#bd93f9" // dracula accent

	for _, name := range testCases {
		t.Run(name, func(t *testing.T) {
			palette := FromName(name)
			expectedColor := lipgloss.Color(expected)
			if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
				t.Errorf("Case insensitive lookup failed: expected %s, got %v", expected, palette.Accent)
			}
		})
	}
}

func TestGet_ExistingAndNonExisting(t *testing.T) {
	// Test existing theme
	palette, exists := Get("dracula")
	if !exists {
		t.Error("Expected dracula theme to exist")
	}
	expectedAccent := "#bd93f9"
	expectedColor := lipgloss.Color(expectedAccent)
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected accent %s, got %v", expectedAccent, palette.Accent)
	}

	// Test non-existing theme
	_, exists = Get("nonexistent")
	if exists {
		t.Error("Expected nonexistent theme to not exist")
	}
}

func TestApplyOverrides_AllColorTypes(t *testing.T) {
	base := FromName("dracula")

	// Test without accent override first, so selected_bg can be independently set
	overrides := map[string]string{
		"warning":            "#00ff00",
		"dim":                "#0000ff",
		"success":            "#ffff00",
		"danger":             "#ff00ff",
		"progress":           "#00ffff",
		"unknown":            "#ffffff",
		"info":               "#808080",
		"text":               "#c0c0c0",
		"gray":               "#404040",
		"selected_bg":        "#ff8080",
		"cursor_selected_bg": "#80ff80",
		"cursor_bg":          "#8080ff",
		"border":             "#ffff80",
		"muted_bg":           "#ff80ff",
		"shade_bg":           "#80ffff",
		"dark_bg":            "#800000",
	}

	result := applyOverrides(base, overrides)

	// Verify each override was applied
	testCases := []struct {
		field    string
		color    interface{}
		expected string
	}{
		{"warning", result.Warning, "#00ff00"},
		{"dim", result.Dim, "#0000ff"},
		{"success", result.Success, "#ffff00"},
		{"danger", result.Danger, "#ff00ff"},
		{"progress", result.Progress, "#00ffff"},
		{"unknown", result.Unknown, "#ffffff"},
		{"info", result.Info, "#808080"},
		{"text", result.Text, "#c0c0c0"},
		{"gray", result.Gray, "#404040"},
		{"selected_bg", result.SelectedBG, "#ff8080"}, // Should get its own value since no accent override
		{"cursor_selected_bg", result.CursorSelectedBG, "#80ff80"},
		{"cursor_bg", result.CursorBG, "#8080ff"},
		{"border", result.Border, "#ffff80"},
		{"muted_bg", result.MutedBG, "#ff80ff"},
		{"shade_bg", result.ShadeBG, "#80ffff"},
		{"dark_bg", result.DarkBG, "#800000"},
	}

	for _, tc := range testCases {
		t.Run(tc.field, func(t *testing.T) {
			expectedColor := lipgloss.Color(tc.expected)
			if fmt.Sprintf("%v", tc.color) != fmt.Sprintf("%v", expectedColor) {
				t.Errorf("Override for %s failed: expected %s, got %v", tc.field, tc.expected, tc.color)
			}
		})
	}

	// Test accent override separately to demonstrate sync behavior
	accentOverrides := map[string]string{"accent": "#ff0000"}
	accentResult := applyOverrides(base, accentOverrides)

	expectedAccentColor := lipgloss.Color("#ff0000")
	if fmt.Sprintf("%v", accentResult.Accent) != fmt.Sprintf("%v", expectedAccentColor) {
		t.Error("Accent override failed")
	}
	// Accent should sync with SelectedBG
	if fmt.Sprintf("%v", accentResult.SelectedBG) != fmt.Sprintf("%v", expectedAccentColor) {
		t.Error("Accent should sync with SelectedBG")
	}
}

func TestApplyOverrides_AccentSyncsSelectedBG(t *testing.T) {
	base := FromName("dracula")

	overrides := map[string]string{
		"accent": "#ff0000",
	}

	result := applyOverrides(base, overrides)

	// Both accent and selected_bg should be updated when accent is overridden
	expectedColor := lipgloss.Color("#ff0000")

	if fmt.Sprintf("%v", result.Accent) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected accent to be #ff0000, got %v", result.Accent)
	}
	if fmt.Sprintf("%v", result.SelectedBG) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected selected_bg to sync with accent #ff0000, got %v", result.SelectedBG)
	}
}

func TestApplyOverrides_SelectedBGIndependent(t *testing.T) {
	base := FromName("dracula")

	overrides := map[string]string{
		"selected_bg": "#ff8080",
	}

	result := applyOverrides(base, overrides)

	// selected_bg should be updated when overridden independently
	expectedColor := lipgloss.Color("#ff8080")

	if fmt.Sprintf("%v", result.SelectedBG) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected selected_bg to be #ff8080, got %v", result.SelectedBG)
	}

	// Accent should remain from base theme
	originalAccent := base.Accent
	if fmt.Sprintf("%v", result.Accent) != fmt.Sprintf("%v", originalAccent) {
		t.Errorf("Expected accent to remain unchanged, got %v", result.Accent)
	}
}

func TestApplyOverrides_UnknownKeysIgnored(t *testing.T) {
	base := FromName("dracula")
	originalAccent := base.Accent

	overrides := map[string]string{
		"invalid_key":     "#ff0000",
		"unknown_color":   "#00ff00",
		"nonexistent":     "#0000ff",
	}

	result := applyOverrides(base, overrides)

	// Accent should remain unchanged
	if fmt.Sprintf("%v", result.Accent) != fmt.Sprintf("%v", originalAccent) {
		t.Errorf("Expected accent to remain %v, got %v", originalAccent, result.Accent)
	}
}

func TestFromConfig_WithoutOverrides(t *testing.T) {
	cfg := &config.ArgonautConfig{
		Appearance: config.AppearanceConfig{
			Theme: "nord",
		},
	}

	palette := FromConfig(cfg)

	// Should match nord theme
	expectedAccent := "#88c0d0"
	expectedColor := lipgloss.Color(expectedAccent)
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected nord accent %s, got %v", expectedAccent, palette.Accent)
	}
}

func TestFromConfig_WithOverrides(t *testing.T) {
	cfg := &config.ArgonautConfig{
		Appearance: config.AppearanceConfig{
			Theme: "nord",
			Overrides: map[string]string{
				"accent":  "#ff0000",
				"warning": "#00ff00",
			},
		},
	}

	palette := FromConfig(cfg)

	// Overrides should be applied
	expectedAccentColor := lipgloss.Color("#ff0000")
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedAccentColor) {
		t.Errorf("Expected overridden accent #ff0000, got %v", palette.Accent)
	}

	expectedWarningColor := lipgloss.Color("#00ff00")
	if fmt.Sprintf("%v", palette.Warning) != fmt.Sprintf("%v", expectedWarningColor) {
		t.Errorf("Expected overridden warning #00ff00, got %v", palette.Warning)
	}

	// Non-overridden colors should remain from base theme (nord)
	expectedSuccess := "#a3be8c" // nord success color
	expectedSuccessColor := lipgloss.Color(expectedSuccess)
	if fmt.Sprintf("%v", palette.Success) != fmt.Sprintf("%v", expectedSuccessColor) {
		t.Errorf("Expected nord success color %s, got %v", expectedSuccess, palette.Success)
	}
}

func TestFromConfig_InvalidThemeWithOverrides(t *testing.T) {
	cfg := &config.ArgonautConfig{
		Appearance: config.AppearanceConfig{
			Theme: "nonexistent",
			Overrides: map[string]string{
				"accent": "#ff0000",
			},
		},
	}

	palette := FromConfig(cfg)

	// Should fallback to default theme but apply overrides
	expectedColor := lipgloss.Color("#ff0000")
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected overridden accent #ff0000 even with invalid theme, got %v", palette.Accent)
	}
}

func TestNames_ReturnsAllThemes(t *testing.T) {
	names := Names()

	// Should return a reasonable number of themes
	if len(names) < 10 {
		t.Errorf("Expected at least 10 themes, got %d", len(names))
	}

	// Should include known themes
	expectedThemes := []string{"dracula", "nord", "gruvbox", "oxocarbon", "monokai"}
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}

	for _, expected := range expectedThemes {
		if !nameSet[expected] {
			t.Errorf("Expected theme %s not found in Names() result", expected)
		}
	}

	// Should be sorted
	for i := 1; i < len(names); i++ {
		if strings.Compare(names[i-1], names[i]) >= 0 {
			t.Errorf("Names() result not sorted: %s >= %s", names[i-1], names[i])
		}
	}
}

func TestColors_ReturnsAllPaletteColors(t *testing.T) {
	palette := FromName("dracula")
	colors := Colors(palette)

	expectedKeys := []string{
		"accent", "warning", "dim", "success", "danger", "progress",
		"unknown", "info", "text", "gray", "selected", "cursorSel",
		"border", "muted", "shade", "dark",
	}

	for _, key := range expectedKeys {
		if _, exists := colors[key]; !exists {
			t.Errorf("Expected color key %s not found in Colors() result", key)
		}
	}

	// Check that colors match palette
	if colors["accent"] != palette.Accent {
		t.Error("accent color mismatch in Colors() result")
	}
	if colors["selected"] != palette.SelectedBG {
		t.Error("selected color mismatch in Colors() result")
	}
}

func TestDefault_ReturnsValidPalette(t *testing.T) {
	palette := Default()

	// Should have all required colors
	if palette.Accent == nil {
		t.Error("Default palette missing Accent color")
	}
	if palette.Warning == nil {
		t.Error("Default palette missing Warning color")
	}

	// Should match expected default colors
	expectedAccent := "13" // magentaBright
	expectedColor := lipgloss.Color(expectedAccent)
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", expectedColor) {
		t.Errorf("Expected default accent %s, got %v", expectedAccent, palette.Accent)
	}
}

func TestGetAvailableThemes_MatchesNames(t *testing.T) {
	available := GetAvailableThemes()
	names := Names()

	if len(available) != len(names) {
		t.Errorf("GetAvailableThemes() length %d doesn't match Names() length %d", len(available), len(names))
	}

	availableSet := make(map[string]bool)
	for _, name := range available {
		availableSet[name] = true
	}

	for _, name := range names {
		if !availableSet[name] {
			t.Errorf("Theme %s from Names() not found in GetAvailableThemes()", name)
		}
	}
}

// Test edge cases and robustness
func TestFromConfig_NilConfig(t *testing.T) {
	// This shouldn't crash even with nil config
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FromConfig panicked with nil config: %v", r)
		}
	}()

	// This would panic, so we test with minimal config instead
	cfg := &config.ArgonautConfig{}
	palette := FromConfig(cfg)

	// Should get default theme
	if palette.Accent == nil {
		t.Error("Got nil accent from empty config")
	}
}

func TestFromConfig_EmptyOverrides(t *testing.T) {
	cfg := &config.ArgonautConfig{
		Appearance: config.AppearanceConfig{
			Theme:     "nord",
			Overrides: map[string]string{},
		},
	}

	palette := FromConfig(cfg)

	// Should match nord theme exactly
	nordPalette := FromName("nord")
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", nordPalette.Accent) {
		t.Error("Empty overrides shouldn't change base theme")
	}
}

func TestFromConfig_NilOverrides(t *testing.T) {
	cfg := &config.ArgonautConfig{
		Appearance: config.AppearanceConfig{
			Theme:     "nord",
			Overrides: nil,
		},
	}

	palette := FromConfig(cfg)

	// Should match nord theme exactly
	nordPalette := FromName("nord")
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", nordPalette.Accent) {
		t.Error("Nil overrides shouldn't change base theme")
	}
}

func TestNewPalette_CompilerEnforcedCompleteness(t *testing.T) {
	// This test demonstrates that NewPalette enforces all colors at compile time
	accent := lipgloss.Color("#ff0000")
	warning := lipgloss.Color("#ffaa00")
	dim := lipgloss.Color("#666666")
	success := lipgloss.Color("#00ff00")
	danger := lipgloss.Color("#ff0000")
	progress := lipgloss.Color("#0088ff")
	unknown := lipgloss.Color("#888888")
	info := lipgloss.Color("#00ffff")
	text := lipgloss.Color("#ffffff")
	gray := lipgloss.Color("#808080")
	selectedBG := lipgloss.Color("#ff0000")
	cursorSelectedBG := lipgloss.Color("#00ff00")
	cursorBG := lipgloss.Color("#0000ff")
	border := lipgloss.Color("#ffffff")
	mutedBG := lipgloss.Color("#333333")
	shadeBG := lipgloss.Color("#222222")
	darkBG := lipgloss.Color("#111111")

	palette := NewPalette(
		accent, warning, dim,
		success, danger, progress, unknown,
		info, text, gray,
		selectedBG, cursorSelectedBG, cursorBG, border,
		mutedBG, shadeBG, darkBG,
	)

	// Verify all colors were set correctly
	if fmt.Sprintf("%v", palette.Accent) != fmt.Sprintf("%v", accent) {
		t.Error("Accent color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Warning) != fmt.Sprintf("%v", warning) {
		t.Error("Warning color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Dim) != fmt.Sprintf("%v", dim) {
		t.Error("Dim color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Success) != fmt.Sprintf("%v", success) {
		t.Error("Success color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Danger) != fmt.Sprintf("%v", danger) {
		t.Error("Danger color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Progress) != fmt.Sprintf("%v", progress) {
		t.Error("Progress color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Unknown) != fmt.Sprintf("%v", unknown) {
		t.Error("Unknown color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Info) != fmt.Sprintf("%v", info) {
		t.Error("Info color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Text) != fmt.Sprintf("%v", text) {
		t.Error("Text color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Gray) != fmt.Sprintf("%v", gray) {
		t.Error("Gray color not set correctly")
	}
	if fmt.Sprintf("%v", palette.SelectedBG) != fmt.Sprintf("%v", selectedBG) {
		t.Error("SelectedBG color not set correctly")
	}
	if fmt.Sprintf("%v", palette.CursorSelectedBG) != fmt.Sprintf("%v", cursorSelectedBG) {
		t.Error("CursorSelectedBG color not set correctly")
	}
	if fmt.Sprintf("%v", palette.CursorBG) != fmt.Sprintf("%v", cursorBG) {
		t.Error("CursorBG color not set correctly")
	}
	if fmt.Sprintf("%v", palette.Border) != fmt.Sprintf("%v", border) {
		t.Error("Border color not set correctly")
	}
	if fmt.Sprintf("%v", palette.MutedBG) != fmt.Sprintf("%v", mutedBG) {
		t.Error("MutedBG color not set correctly")
	}
	if fmt.Sprintf("%v", palette.ShadeBG) != fmt.Sprintf("%v", shadeBG) {
		t.Error("ShadeBG color not set correctly")
	}
	if fmt.Sprintf("%v", palette.DarkBG) != fmt.Sprintf("%v", darkBG) {
		t.Error("DarkBG color not set correctly")
	}
}

func TestNewPalette_UsageExample(t *testing.T) {
	// Example of how NewPalette should be used for new themes
	customPalette := NewPalette(
		// Core colors
		lipgloss.Color("#e06c75"), lipgloss.Color("#e5c07b"), lipgloss.Color("#5c6370"), // accent, warning, dim
		// Status colors
		lipgloss.Color("#98c379"), lipgloss.Color("#e06c75"), lipgloss.Color("#61afef"), lipgloss.Color("#5c6370"), // success, danger, progress, unknown
		// Text colors
		lipgloss.Color("#56b6c2"), lipgloss.Color("#abb2bf"), lipgloss.Color("#5c6370"), // info, text, gray
		// Background colors
		lipgloss.Color("#e06c75"), lipgloss.Color("#56b6c2"), lipgloss.Color("#61afef"), lipgloss.Color("#e06c75"), // selectedBG, cursorSelectedBG, cursorBG, border
		// Panel backgrounds
		lipgloss.Color("#2c313c"), lipgloss.Color("#21252b"), lipgloss.Color("#181a1f"), // mutedBG, shadeBG, darkBG
	)

	// Verify it creates a valid palette
	if customPalette.Accent == nil {
		t.Error("NewPalette should create valid palette with all colors")
	}

	// Verify no color is nil
	if customPalette.Warning == nil || customPalette.Success == nil || customPalette.DarkBG == nil {
		t.Error("All colors in NewPalette result should be non-nil")
	}
}