// Package selection provides mouse-based text selection for TUI applications.
package selection

// Position represents a screen coordinate (0-based).
type Position struct {
	Row int
	Col int
}

// Selection tracks the state of a text selection.
type Selection struct {
	Start      Position
	End        Position
	Active     bool // Currently dragging
	HasContent bool // Has finalized selection
}

// New creates a new empty Selection.
func New() *Selection {
	return &Selection{}
}

// Clear resets the selection state.
func (s *Selection) Clear() {
	s.Start = Position{}
	s.End = Position{}
	s.Active = false
	s.HasContent = false
}

// SetStart begins a new selection at the given position.
func (s *Selection) SetStart(pos Position) {
	s.Start = pos
	s.End = pos
	s.Active = true
	s.HasContent = false
}

// SetEnd updates the end position of the selection.
func (s *Selection) SetEnd(pos Position) {
	s.End = pos
}

// Finalize marks the selection as complete.
// Returns true if the selection has actual content (not just a click).
func (s *Selection) Finalize() bool {
	s.Active = false
	// Only mark as having content if start != end
	s.HasContent = s.Start != s.End
	return s.HasContent
}

// Normalize returns the selection bounds in order (start <= end).
func (s *Selection) Normalize() (start, end Position) {
	start, end = s.Start, s.End

	// If end is before start, swap them
	if end.Row < start.Row || (end.Row == start.Row && end.Col < start.Col) {
		start, end = end, start
	}

	return start, end
}

// GetBounds returns the normalized selection boundaries.
func (s *Selection) GetBounds() (startRow, startCol, endRow, endCol int) {
	start, end := s.Normalize()
	return start.Row, start.Col, end.Row, end.Col
}

// Contains checks if a given position is within the selection.
func (s *Selection) Contains(row, col int) bool {
	if !s.HasContent && !s.Active {
		return false
	}

	startRow, startCol, endRow, endCol := s.GetBounds()

	// Before selection starts
	if row < startRow || (row == startRow && col < startCol) {
		return false
	}

	// After selection ends
	if row > endRow || (row == endRow && col > endCol) {
		return false
	}

	return true
}

// IsEmpty returns true if there's no selection.
func (s *Selection) IsEmpty() bool {
	return !s.HasContent && !s.Active
}

// ExtractText extracts the selected text from rendered content lines.
// The lines should be the plain text content (ANSI stripped).
func (s *Selection) ExtractText(lines []string) string {
	if s.IsEmpty() {
		return ""
	}

	startRow, startCol, endRow, endCol := s.GetBounds()

	// Clamp to available lines
	if startRow >= len(lines) {
		return ""
	}
	if endRow >= len(lines) {
		endRow = len(lines) - 1
	}

	var result []byte

	for row := startRow; row <= endRow; row++ {
		if row >= len(lines) {
			break
		}
		line := lines[row]
		lineRunes := []rune(line)

		var colStart, colEnd int

		if row == startRow {
			colStart = startCol
		} else {
			colStart = 0
		}

		if row == endRow {
			colEnd = endCol
		} else {
			colEnd = len(lineRunes)
		}

		// Clamp to line length
		if colStart > len(lineRunes) {
			colStart = len(lineRunes)
		}
		if colEnd > len(lineRunes) {
			colEnd = len(lineRunes)
		}

		if colStart < colEnd {
			result = append(result, string(lineRunes[colStart:colEnd])...)
		}

		// Add newline between lines (but not after the last line)
		if row < endRow {
			result = append(result, '\n')
		}
	}

	return string(result)
}
