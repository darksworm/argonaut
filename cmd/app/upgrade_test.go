package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/config"
)

// TestScheduleInitialUpdateCheck_RespectsDisableConfig verifies that the
// initial update-check scheduler short-circuits to nil when the
// `[updates] check_enabled = false` config is set, and returns a real
// command otherwise.
func TestScheduleInitialUpdateCheck_RespectsDisableConfig(t *testing.T) {
	tt := false
	tr := true

	tests := []struct {
		name      string
		cfg       *config.ArgonautConfig
		wantNoCmd bool
	}{
		{"nil config → enabled by default → cmd returned", nil, false},
		{"empty config → enabled by default → cmd returned", &config.ArgonautConfig{}, false},
		{"check_enabled=true → cmd returned", &config.ArgonautConfig{Updates: config.UpdatesConfig{CheckEnabled: &tr}}, false},
		{"check_enabled=false → no cmd", &config.ArgonautConfig{Updates: config.UpdatesConfig{CheckEnabled: &tt}}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := &Model{config: tc.cfg}
			m.initializeUpdateService() // sets m.updateService

			gotInitial := m.scheduleInitialUpdateCheck()
			gotPeriodic := m.schedulePeriodicUpdateCheck()
			if tc.wantNoCmd {
				if gotInitial != nil {
					t.Errorf("scheduleInitialUpdateCheck: expected nil with check_enabled=false, got non-nil")
				}
				if gotPeriodic != nil {
					t.Errorf("schedulePeriodicUpdateCheck: expected nil with check_enabled=false, got non-nil")
				}
			} else {
				if gotInitial == nil {
					t.Errorf("scheduleInitialUpdateCheck: expected non-nil with check enabled, got nil")
				}
				if gotPeriodic == nil {
					t.Errorf("schedulePeriodicUpdateCheck: expected non-nil with check enabled, got nil")
				}
			}
		})
	}
}
