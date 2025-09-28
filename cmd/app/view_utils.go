package main

import (
	"github.com/charmbracelet/lipgloss/v2"
	"strings"
)

// clipAnsiToWidth trims a styled string to the given display width (ANSI-aware)
func clipAnsiToWidth(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		candidate := b.String() + string(r)
		if lipgloss.Width(candidate) > width {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

// wrapAnsiToWidth wraps a string into visual lines that fit the given width (ANSI-aware)
func wrapAnsiToWidth(s string, width int) []string {
	if width <= 0 {
		return []string{""}
	}
	// Fast path if it already fits
	if lipgloss.Width(s) <= width {
		return []string{s}
	}
	var lines []string
	var b strings.Builder
	for _, r := range s {
		ch := string(r)
		next := b.String() + ch
		if lipgloss.Width(next) > width {
			lines = append(lines, b.String())
			b.Reset()
			b.WriteString(ch)
		} else {
			b.WriteString(ch)
		}
	}
	if b.Len() > 0 {
		lines = append(lines, b.String())
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
}

// calculateColumnWidths returns responsive column widths based on available space
func calculateColumnWidths(availableWidth int) (nameWidth, syncWidth, healthWidth int) {
	// Account for separators between the 3 columns (2 separators, 1 char each)
	const sep = 2

	if availableWidth < 45 {
		// Very narrow: minimal widths (icons only)
		syncWidth = 2   // Just icon
		healthWidth = 2 // Just icon
		// Ensure minimum viable name width
		nameWidth = max(1, availableWidth-syncWidth-healthWidth-sep)
	} else {
		// Wide: full widths
		syncWidth = 12   // SYNC column
		healthWidth = 15 // HEALTH column
		nameWidth = max(10, availableWidth-syncWidth-healthWidth-sep)
	}

	// Make sure columns exactly fill the available width including separators
	totalUsed := nameWidth + syncWidth + healthWidth + sep
	if totalUsed < availableWidth {
		nameWidth += (availableWidth - totalUsed)
	} else if totalUsed > availableWidth {
		overflow := totalUsed - availableWidth
		nameWidth = max(1, nameWidth-overflow)
	}

	return nameWidth, syncWidth, healthWidth
}
