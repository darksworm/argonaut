package main

import (
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestShouldShowWhatsNewNotification(t *testing.T) {
	tests := []struct {
		name       string
		shownAt    *time.Time
		wantResult bool
	}{
		{
			name:       "returns true when shownAt is nil",
			shownAt:    nil,
			wantResult: true,
		},
		{
			name: "returns true when notification was just shown",
			shownAt: func() *time.Time {
				t := time.Now()
				return &t
			}(),
			wantResult: true,
		},
		{
			name: "returns true when notification was shown 10 seconds ago",
			shownAt: func() *time.Time {
				t := time.Now().Add(-10 * time.Second)
				return &t
			}(),
			wantResult: true,
		},
		{
			name: "returns true when notification was shown 29 seconds ago",
			shownAt: func() *time.Time {
				t := time.Now().Add(-29 * time.Second)
				return &t
			}(),
			wantResult: true,
		},
		{
			name: "returns false when notification was shown 31 seconds ago",
			shownAt: func() *time.Time {
				t := time.Now().Add(-31 * time.Second)
				return &t
			}(),
			wantResult: false,
		},
		{
			name: "returns false when notification was shown 1 minute ago",
			shownAt: func() *time.Time {
				t := time.Now().Add(-60 * time.Second)
				return &t
			}(),
			wantResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Model{
				state: model.NewAppState(),
			}
			m.state.UI.WhatsNewShownAt = tt.shownAt

			got := m.shouldShowWhatsNewNotification()
			if got != tt.wantResult {
				t.Errorf("shouldShowWhatsNewNotification() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
