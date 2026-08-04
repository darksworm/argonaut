package main

import (
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestStatusLine_TreeView_AdvertisesPaneHotkeys(t *testing.T) {
	m := buildEventsPaneTestModel()

	line := stripANSI(m.renderStatusLine())

	if !strings.Contains(line, "e: events • S: sync status") {
		t.Errorf("expected the tree status line to advertise the pane hotkeys, got %q", line)
	}
	if !strings.Contains(line, "Ready") {
		t.Errorf("expected Ready to remain in the status line, got %q", line)
	}
}

func TestStatusLine_EventsPane_ShowsModeAndScrollHints(t *testing.T) {
	m := openEventsPane(t, buildEventsPaneTestModel())

	line := stripANSI(m.renderStatusLine())

	if !strings.Contains(line, "<events>") {
		t.Errorf("expected the mode segment <events>, got %q", line)
	}
	if !strings.Contains(line, "j/k: select • J/K: scroll • esc: close") {
		t.Errorf("expected the pane hints, got %q", line)
	}
	if strings.Contains(line, "e: events") {
		t.Errorf("tree hints must yield while the pane is open, got %q", line)
	}
}

func TestStatusLine_SyncStatusPane_ShowsModeSegment(t *testing.T) {
	m := buildEventsPaneTestModel()
	teaModel, _ := m.handleKeyMsg(testKeyMsg("S"))
	m = teaModel.(*Model)

	line := stripANSI(m.renderStatusLine())

	if !strings.Contains(line, "<sync-status>") {
		t.Errorf("expected the mode segment <sync-status>, got %q", line)
	}
	if !strings.Contains(line, "j/k: select • J/K: scroll • esc: close") {
		t.Errorf("expected the pane hints, got %q", line)
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
