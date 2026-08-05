package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/tui/selection"
)

// A selection highlight restyles the frame — it must never change what the
// frame says or how much room it takes. Any line growing past the terminal
// width would hard-wrap on a real terminal, scrolling the screen and
// desyncing every later diff repaint (stale duplicated rows, clipped banner).
func TestSelectionHighlight_NeverChangesFrameGeometryOrText(t *testing.T) {
	m := buildPaneGoldenModel(100, 24)
	openGoldenEventsPane(m)
	frame := m.renderMainLayout()
	lines := strings.Split(frame, "\n")

	for row := 0; row < len(lines); row++ {
		for _, cols := range [][2]int{{0, 10}, {4, 60}, {0, 99}, {20, 999}} {
			m.selection = selection.New()
			m.selection.SetStart(selection.Position{Row: row, Col: cols[0]})
			m.selection.SetEnd(selection.Position{Row: row, Col: cols[1]})

			highlighted := m.applySelectionHighlight(frame)

			if stripANSI(highlighted) != stripANSI(frame) {
				t.Fatalf("row %d cols %v: the highlight changed the frame's text", row, cols)
			}
			for i, hl := range strings.Split(highlighted, "\n") {
				if got, want := lipgloss.Width(hl), lipgloss.Width(lines[i]); got != want {
					t.Fatalf("row %d cols %v: line %d width changed %d → %d", row, cols, i, want, got)
				}
			}
		}
	}
}
