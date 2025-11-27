package listnav

// ListNavigator encapsulates cursor and scroll state for any scrollable list.
// It does NOT render - it only manages navigation state.
type ListNavigator struct {
	cursor         int // Currently selected item index (0-based)
	scrollOffset   int // First visible item index
	itemCount      int // Total items (set externally before navigation)
	viewportHeight int // Visible rows (set externally before navigation)
}

// New creates a ListNavigator with default values.
func New() *ListNavigator {
	return &ListNavigator{
		cursor:         0,
		scrollOffset:   0,
		itemCount:      0,
		viewportHeight: 10, // sensible default
	}
}

// Cursor returns the currently selected item index.
func (n *ListNavigator) Cursor() int {
	return n.cursor
}

// ScrollOffset returns the index of the first visible item.
func (n *ListNavigator) ScrollOffset() int {
	return n.scrollOffset
}

// SetItemCount updates the total item count and clamps cursor/scroll.
// Call this before any navigation operation.
func (n *ListNavigator) SetItemCount(count int) {
	n.itemCount = count
	n.clampCursor()
	n.clampScrollOffset()
}

// SetViewportHeight updates the visible row count.
// Call this before any navigation operation.
func (n *ListNavigator) SetViewportHeight(h int) {
	if h < 1 {
		h = 1
	}
	n.viewportHeight = h
	n.clampScrollOffset()
}

// MoveUp moves the cursor up by one item, adjusting scroll as needed.
// Returns true if state changed.
func (n *ListNavigator) MoveUp() bool {
	if n.cursor <= 0 {
		return false
	}
	n.cursor--
	if n.cursor < n.scrollOffset {
		n.scrollOffset = n.cursor
	}
	return true
}

// MoveDown moves the cursor down by one item, adjusting scroll as needed.
// Returns true if state changed.
func (n *ListNavigator) MoveDown() bool {
	if n.itemCount == 0 || n.cursor >= n.itemCount-1 {
		return false
	}
	n.cursor++
	if n.cursor >= n.scrollOffset+n.viewportHeight {
		n.scrollOffset = n.cursor - n.viewportHeight + 1
	}
	return true
}

// PageUp moves the cursor up by one page (viewport-based).
// The cursor moves to the first item of the previous viewport.
// Returns true if state changed.
func (n *ListNavigator) PageUp() bool {
	if n.itemCount == 0 {
		return false
	}
	oldCursor := n.cursor
	oldScroll := n.scrollOffset

	// Move to first item of previous page
	newIdx := n.scrollOffset - n.viewportHeight
	if newIdx < 0 {
		newIdx = 0
	}
	n.cursor = newIdx
	n.scrollOffset = newIdx

	return n.cursor != oldCursor || n.scrollOffset != oldScroll
}

// PageDown moves the cursor down by one page (viewport-based).
// The cursor moves to the first item after the current viewport.
// Returns true if state changed.
func (n *ListNavigator) PageDown() bool {
	if n.itemCount == 0 {
		return false
	}
	oldCursor := n.cursor
	oldScroll := n.scrollOffset

	maxIdx := n.itemCount - 1

	// Move to first item after current viewport
	newIdx := n.scrollOffset + n.viewportHeight
	if newIdx > maxIdx {
		newIdx = maxIdx
	}
	n.cursor = newIdx
	n.scrollOffset = newIdx

	// Clamp scroll offset so we don't scroll past the end
	n.clampScrollOffset()

	return n.cursor != oldCursor || n.scrollOffset != oldScroll
}

// GoToTop moves the cursor to the first item.
// Returns true if state changed.
func (n *ListNavigator) GoToTop() bool {
	if n.cursor == 0 && n.scrollOffset == 0 {
		return false
	}
	n.cursor = 0
	n.scrollOffset = 0
	return true
}

// GoToBottom moves the cursor to the last item.
// Returns true if state changed.
func (n *ListNavigator) GoToBottom() bool {
	if n.itemCount == 0 {
		return false
	}
	lastIdx := n.itemCount - 1
	if n.cursor == lastIdx {
		// Already at bottom, but we might need to adjust scroll
		oldScroll := n.scrollOffset
		n.scrollOffset = max(0, n.itemCount-n.viewportHeight)
		return n.scrollOffset != oldScroll
	}
	n.cursor = lastIdx
	n.scrollOffset = max(0, n.itemCount-n.viewportHeight)
	return true
}

// SetCursor directly sets the cursor position with bounds checking.
// Adjusts scroll to keep cursor visible.
func (n *ListNavigator) SetCursor(idx int) {
	n.cursor = idx
	n.clampCursor()
	n.ensureCursorVisible()
}

// Reset clears state to initial values.
func (n *ListNavigator) Reset() {
	n.cursor = 0
	n.scrollOffset = 0
}

// clampCursor ensures cursor is within valid bounds.
func (n *ListNavigator) clampCursor() {
	if n.itemCount == 0 {
		n.cursor = 0
		return
	}
	if n.cursor < 0 {
		n.cursor = 0
	}
	if n.cursor >= n.itemCount {
		n.cursor = n.itemCount - 1
	}
}

// clampScrollOffset ensures scroll offset is within valid bounds.
func (n *ListNavigator) clampScrollOffset() {
	if n.itemCount == 0 || n.viewportHeight <= 0 {
		n.scrollOffset = 0
		return
	}
	maxScroll := max(0, n.itemCount-n.viewportHeight)
	if n.scrollOffset < 0 {
		n.scrollOffset = 0
	}
	if n.scrollOffset > maxScroll {
		n.scrollOffset = maxScroll
	}
}

// ensureCursorVisible adjusts scroll to keep cursor visible.
func (n *ListNavigator) ensureCursorVisible() {
	if n.cursor < n.scrollOffset {
		n.scrollOffset = n.cursor
	}
	if n.cursor >= n.scrollOffset+n.viewportHeight {
		n.scrollOffset = n.cursor - n.viewportHeight + 1
	}
	n.clampScrollOffset()
}
