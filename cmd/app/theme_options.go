package main

import (
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/theme"
)

const warningIndicator = "⚠"

type themeOption struct {
	Name           string
	Display        string
	Warning        bool
	WarningMessage string
}

func buildThemeOptions(custom config.CustomTheme) []themeOption {
	presetNames := theme.Names()
	options := make([]themeOption, 0, len(presetNames)+1)
	for _, name := range presetNames {
		options = append(options, themeOption{Name: name, Display: name})
	}

	analysis := theme.AnalyzeCustomTheme(custom)
	if !analysis.HasAny() {
		return options
	}

	if analysis.Complete() {
		return append(options, themeOption{Name: "custom", Display: "custom"})
	}

	return append(options, themeOption{
		Name:           "custom",
		Display:        "custom " + warningIndicator,
		Warning:        true,
		WarningMessage: warningIndicator + " some colors missing from custom theme",
	})
}

func (m *Model) ensureThemeOptionsLoaded() {
	if len(m.themeOptions) == 0 {
		m.themeOptions = buildThemeOptions(m.customTheme)
	}
}
