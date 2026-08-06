package main

import (
	"testing"

	"github.com/darksworm/argonaut/pkg/config"
)

func TestShouldPinClustersView(t *testing.T) {
	tests := []struct {
		name          string
		configExisted bool
		defaultView   string
		lastSeen      string
		want          bool
	}{
		{
			name:          "fresh install keeps the new apps default",
			configExisted: false,
			want:          false,
		},
		{
			name:          "existing user from before version tracking is pinned to clusters",
			configExisted: true,
			lastSeen:      "",
			want:          true,
		},
		{
			name:          "user who already chose a default view is left alone",
			configExisted: true,
			defaultView:   "apps",
			lastSeen:      "",
			want:          false,
		},
		{
			name:          "user upgrading from the last clusters-default version is pinned",
			configExisted: true,
			lastSeen:      "2.17.2",
			want:          true,
		},
		{
			name:          "user upgrading from an older clusters-default version is pinned",
			configExisted: true,
			lastSeen:      "2.16.0",
			want:          true,
		},
		{
			name:          "user who started on an apps-default version keeps the apps default",
			configExisted: true,
			lastSeen:      "2.18.0",
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.GetDefaultConfig()
			cfg.DefaultView = tt.defaultView
			cfg.LastSeenVersion = tt.lastSeen

			if got := shouldPinClustersView(cfg, tt.configExisted); got != tt.want {
				t.Errorf("shouldPinClustersView() = %v, want %v", got, tt.want)
			}
		})
	}
}
