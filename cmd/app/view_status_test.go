package main

import (
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

// With the pane hidden, the tree advertises how to bring it back.
func TestStatusLine_TreeView_AdvertisesThePaneToggle(t *testing.T) {
	m := buildEventsPaneTestModel()

	line := stripANSI(m.renderStatusLine())

	if !strings.Contains(line, "e: events") {
		t.Errorf("expected the tree status line to advertise the pane toggle, got %q", line)
	}
	if !strings.Contains(line, "Ready") {
		t.Errorf("expected Ready to remain in the status line, got %q", line)
	}
}

// The pane is part of the view: the status line keeps the tree segment and
// position, and only the scroll hint changes.
func TestStatusLine_OpenPane_KeepsTreeSegmentAndShowsScrollHint(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	line := stripANSI(m.renderStatusLine())

	if !strings.Contains(line, "<tree>") {
		t.Errorf("expected the tree segment to stay, got %q", line)
	}
	if !strings.Contains(line, "u/i: scroll events") {
		t.Errorf("expected the scroll hint, got %q", line)
	}
	if !strings.Contains(line, "2/2") {
		t.Errorf("expected the tree position to stay visible, got %q", line)
	}
	for _, gone := range []string{"<events>", "esc: close", "j/k: select"} {
		if strings.Contains(line, gone) {
			t.Errorf("expected %q gone from the status line, got %q", gone, line)
		}
	}
}

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
