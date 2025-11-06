package main

import (
	"github.com/darksworm/argonaut/pkg/theme"
)

const warningIndicator = "⚠"

type themeOption struct {
	Name           string
	Display        string
	Warning        bool
	WarningMessage string
}

func buildThemeOptions() []themeOption {
	presetNames := theme.Names()
	options := make([]themeOption, 0, len(presetNames))
	for _, name := range presetNames {
		options = append(options, themeOption{Name: name, Display: name})
	}
	return options
}

func (m *Model) ensureThemeOptionsLoaded() {
	if len(m.themeOptions) == 0 {
		m.themeOptions = buildThemeOptions()
	}
}
