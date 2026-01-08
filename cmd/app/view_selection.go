package main

import (
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
)

// ANSI escape sequence pattern
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes all ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

// splitIntoLines splits content into lines and returns them.
func splitIntoLines(content string) []string {
	return strings.Split(content, "\n")
}

// storeRenderedContent extracts plain text lines from rendered content
// and stores them for text selection extraction.
func (m *Model) storeRenderedContent(content string) {
	plainContent := stripANSI(content)
	m.lastRenderedLines = splitIntoLines(plainContent)
}

// Selection highlight style
var selectionStyle = lipgloss.NewStyle().Reverse(true)

// applySelectionHighlight overlays selection highlighting on rendered content.
func (m *Model) applySelectionHighlight(content string) string {
	if m.selection == nil || m.selection.IsEmpty() {
		return content
	}

	lines := strings.Split(content, "\n")
	startRow, startCol, endRow, endCol := m.selection.GetBounds()

	for row := startRow; row <= endRow && row < len(lines); row++ {
		line := lines[row]

		// Get plain text to calculate proper positions
		plainLine := stripANSI(line)
		plainRunes := []rune(plainLine)

		// Determine selection bounds for this line
		var colStart, colEnd int
		if row == startRow {
			colStart = startCol
		} else {
			colStart = 0
		}
		if row == endRow {
			colEnd = endCol
		} else {
			colEnd = len(plainRunes)
		}

		// Clamp to line length
		if colStart > len(plainRunes) {
			colStart = len(plainRunes)
		}
		if colEnd > len(plainRunes) {
			colEnd = len(plainRunes)
		}
		if colStart >= colEnd {
			continue
		}

		// For simple highlighting, we replace the line with a version
		// that has the selection portion highlighted.
		// This is a simplified approach that works with plain text.
		// For fully styled content, we'd need more complex ANSI parsing.

		before := string(plainRunes[:colStart])
		selected := string(plainRunes[colStart:colEnd])
		after := string(plainRunes[colEnd:])

		// Apply highlight to selected portion
		highlighted := selectionStyle.Render(selected)

		lines[row] = before + highlighted + after
	}

	return strings.Join(lines, "\n")
}
