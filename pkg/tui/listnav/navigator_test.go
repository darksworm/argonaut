package listnav

import (
	"testing"
)

func TestNew(t *testing.T) {
	n := New()
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0, got %d", n.ScrollOffset())
	}
}

func TestMoveDown(t *testing.T) {
	n := New()
	n.SetItemCount(10)
	n.SetViewportHeight(5)

	// Move down once
	changed := n.MoveDown()
	if !changed {
		t.Error("expected state change")
	}
	if n.Cursor() != 1 {
		t.Errorf("expected cursor 1, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0, got %d", n.ScrollOffset())
	}

	// Move to position 4 (still visible)
	for i := 0; i < 3; i++ {
		n.MoveDown()
	}
	if n.Cursor() != 4 {
		t.Errorf("expected cursor 4, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0, got %d", n.ScrollOffset())
	}

	// Move to position 5 (should scroll)
	n.MoveDown()
	if n.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 1 {
		t.Errorf("expected scroll offset 1, got %d", n.ScrollOffset())
	}
}

func TestMoveUp(t *testing.T) {
	n := New()
	n.SetItemCount(10)
	n.SetViewportHeight(5)

	// Start at position 5
	n.SetCursor(5)
	if n.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", n.Cursor())
	}

	// Move up
	changed := n.MoveUp()
	if !changed {
		t.Error("expected state change")
	}
	if n.Cursor() != 4 {
		t.Errorf("expected cursor 4, got %d", n.Cursor())
	}

	// Move up to scroll boundary
	for i := 0; i < 4; i++ {
		n.MoveUp()
	}
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", n.Cursor())
	}

	// Can't move up past 0
	changed = n.MoveUp()
	if changed {
		t.Error("expected no state change at boundary")
	}
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", n.Cursor())
	}
}

func TestMoveDownAtEnd(t *testing.T) {
	n := New()
	n.SetItemCount(10)
	n.SetViewportHeight(5)
	n.SetCursor(9)

	changed := n.MoveDown()
	if changed {
		t.Error("expected no state change at end")
	}
	if n.Cursor() != 9 {
		t.Errorf("expected cursor 9, got %d", n.Cursor())
	}
}

func TestPageDown(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)

	// Page down from start
	changed := n.PageDown()
	if !changed {
		t.Error("expected state change")
	}
	if n.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 5 {
		t.Errorf("expected scroll offset 5, got %d", n.ScrollOffset())
	}

	// Page down again
	n.PageDown()
	if n.Cursor() != 10 {
		t.Errorf("expected cursor 10, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 10 {
		t.Errorf("expected scroll offset 10, got %d", n.ScrollOffset())
	}

	// Page down near end (should clamp)
	n.PageDown()
	n.PageDown()
	if n.Cursor() != 19 {
		t.Errorf("expected cursor 19, got %d", n.Cursor())
	}
	// Max scroll is 20-5=15
	if n.ScrollOffset() != 15 {
		t.Errorf("expected scroll offset 15, got %d", n.ScrollOffset())
	}
}

func TestPageUp(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)

	// Start at bottom
	n.GoToBottom()
	if n.Cursor() != 19 {
		t.Errorf("expected cursor 19, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 15 {
		t.Errorf("expected scroll offset 15, got %d", n.ScrollOffset())
	}

	// Page up
	changed := n.PageUp()
	if !changed {
		t.Error("expected state change")
	}
	if n.Cursor() != 10 {
		t.Errorf("expected cursor 10, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 10 {
		t.Errorf("expected scroll offset 10, got %d", n.ScrollOffset())
	}

	// Page up again
	n.PageUp()
	if n.Cursor() != 5 {
		t.Errorf("expected cursor 5, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 5 {
		t.Errorf("expected scroll offset 5, got %d", n.ScrollOffset())
	}

	// Page up to start
	n.PageUp()
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0, got %d", n.ScrollOffset())
	}
}

func TestGoToTop(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)
	n.SetCursor(15)

	changed := n.GoToTop()
	if !changed {
		t.Error("expected state change")
	}
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0, got %d", n.ScrollOffset())
	}

	// Already at top
	changed = n.GoToTop()
	if changed {
		t.Error("expected no state change when already at top")
	}
}

func TestGoToBottom(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)

	changed := n.GoToBottom()
	if !changed {
		t.Error("expected state change")
	}
	if n.Cursor() != 19 {
		t.Errorf("expected cursor 19, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 15 {
		t.Errorf("expected scroll offset 15, got %d", n.ScrollOffset())
	}
}

func TestSetCursor(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)

	// Set cursor in middle
	n.SetCursor(10)
	if n.Cursor() != 10 {
		t.Errorf("expected cursor 10, got %d", n.Cursor())
	}
	// Should scroll to show cursor
	if n.ScrollOffset() > 10 || n.ScrollOffset()+n.viewportHeight <= 10 {
		t.Errorf("cursor should be visible, scroll=%d viewport=%d", n.ScrollOffset(), n.viewportHeight)
	}

	// Set cursor out of bounds (negative)
	n.SetCursor(-5)
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0 after negative set, got %d", n.Cursor())
	}

	// Set cursor out of bounds (too high)
	n.SetCursor(100)
	if n.Cursor() != 19 {
		t.Errorf("expected cursor 19 after overflow set, got %d", n.Cursor())
	}
}

func TestReset(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)
	n.SetCursor(15)

	n.Reset()
	if n.Cursor() != 0 {
		t.Errorf("expected cursor 0 after reset, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0 after reset, got %d", n.ScrollOffset())
	}
}

func TestEmptyList(t *testing.T) {
	n := New()
	n.SetItemCount(0)
	n.SetViewportHeight(5)

	changed := n.MoveDown()
	if changed {
		t.Error("expected no state change on empty list")
	}

	changed = n.MoveUp()
	if changed {
		t.Error("expected no state change on empty list")
	}

	changed = n.PageDown()
	if changed {
		t.Error("expected no state change on empty list")
	}

	changed = n.PageUp()
	if changed {
		t.Error("expected no state change on empty list")
	}

	changed = n.GoToBottom()
	if changed {
		t.Error("expected no state change on empty list")
	}
}

func TestSmallList(t *testing.T) {
	n := New()
	n.SetItemCount(3)
	n.SetViewportHeight(10) // Viewport larger than list

	// PageDown should go to last item
	n.PageDown()
	if n.Cursor() != 2 {
		t.Errorf("expected cursor 2, got %d", n.Cursor())
	}
	if n.ScrollOffset() != 0 {
		t.Errorf("expected scroll offset 0 (no scroll needed), got %d", n.ScrollOffset())
	}
}

func TestScrollAdjustment(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)

	// Move cursor to position 10
	n.SetCursor(10)
	// Scroll should adjust to show cursor
	if n.Cursor() < n.ScrollOffset() || n.Cursor() >= n.ScrollOffset()+n.viewportHeight {
		t.Error("cursor should be visible in viewport")
	}

	// Now if we change item count to shrink list
	n.SetItemCount(8)
	if n.Cursor() != 7 {
		t.Errorf("expected cursor clamped to 7, got %d", n.Cursor())
	}
}

func TestViewportResizing(t *testing.T) {
	n := New()
	n.SetItemCount(20)
	n.SetViewportHeight(5)
	n.GoToBottom() // cursor=19, scroll=15

	// Shrink viewport
	n.SetViewportHeight(3)
	// Max scroll is now 20-3=17
	if n.ScrollOffset() > 17 {
		t.Errorf("scroll should be clamped to max, got %d", n.ScrollOffset())
	}

	// Expand viewport
	n.SetViewportHeight(10)
	// Max scroll is now 20-10=10
	if n.ScrollOffset() > 10 {
		t.Errorf("scroll should be clamped to max, got %d", n.ScrollOffset())
	}
}
