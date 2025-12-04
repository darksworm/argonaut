package main

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/listnav"
)

// NavigatorContext provides navigation configuration for the current mode/view.
// It encapsulates the navigator instance, sizing functions, and post-navigation
// callbacks that handle view-specific side effects.
type NavigatorContext struct {
	// Navigator to use (nil for direct offset modes like Diff)
	Navigator *listnav.ListNavigator

	// Item count function - called before each navigation operation
	GetItemCount func() int

	// Viewport height function - called before each navigation operation
	GetViewportHeight func() int

	// Post-navigation callback - called after navigation to handle side effects
	// The 'changed' parameter indicates whether the cursor actually moved
	OnNavigate func(changed bool)

	// Whether this context supports navigation (false for Search, Command, Help modes)
	SupportsNavigation bool

	// For direct offset modes (Diff view) that don't use a navigator
	DirectOffset *int
	PageSize     func() int
}

// isNavigationKey returns true if the key is a list navigation key.
// These keys are handled centrally by the navigation router.
func isNavigationKey(msg tea.KeyMsg) bool {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "g", "G":
		return true
	default:
		return false
	}
}

// getNavigatorContext returns the appropriate NavigatorContext based on current mode and view.
// This is the single source of truth for "which navigator handles navigation in this state".
func (m *Model) getNavigatorContext() *NavigatorContext {
	switch m.state.Mode {

	case model.ModeTheme:
		m.ensureThemeOptionsLoaded()
		if len(m.themeOptions) == 0 {
			return &NavigatorContext{SupportsNavigation: false}
		}
		return &NavigatorContext{
			Navigator:         m.themeNav,
			GetItemCount:      func() int { return len(m.themeOptions) },
			GetViewportHeight: m.themePageSize,
			OnNavigate: func(changed bool) {
				if changed {
					m.syncThemeNavToState()
					m.applyThemePreview(m.themeOptions[m.themeNav.Cursor()].Name)
				}
			},
			SupportsNavigation: true,
		}

	case model.ModeRollback:
		if m.state.Rollback == nil || m.state.Rollback.Loading {
			return &NavigatorContext{SupportsNavigation: false}
		}
		return &NavigatorContext{
			Navigator:         m.rollbackNav,
			GetItemCount:      func() int { return len(m.state.Rollback.Rows) },
			GetViewportHeight: m.rollbackPageSize,
			OnNavigate: func(changed bool) {
				if changed {
					m.state.Rollback.SelectedIdx = m.rollbackNav.Cursor()
				}
			},
			SupportsNavigation: true,
		}

	case model.ModeDiff:
		if m.state.Diff == nil {
			return &NavigatorContext{SupportsNavigation: false}
		}
		return &NavigatorContext{
			SupportsNavigation: true,
			DirectOffset:       &m.state.Diff.Offset,
			PageSize:           m.diffPageSize,
		}

	case model.ModeNormal:
		// Check for tree view first
		if m.state.Navigation.View == model.ViewTree {
			if m.treeView == nil {
				return &NavigatorContext{SupportsNavigation: false}
			}
			return &NavigatorContext{
				Navigator:         m.treeNav,
				GetItemCount:      func() int { return m.treeView.VisibleCount() },
				GetViewportHeight: m.treeViewportHeight,
				OnNavigate: func(changed bool) {
					if changed {
						m.treeView.SetSelectedIndex(m.treeNav.Cursor())
					}
				},
				SupportsNavigation: true,
			}
		}
		// Default: list navigation (apps, clusters, namespaces, projects)
		return &NavigatorContext{
			Navigator:         m.listNav,
			GetItemCount:      func() int { return len(m.getVisibleItemsForCurrentView()) },
			GetViewportHeight: m.listViewportHeight,
			OnNavigate: func(changed bool) {
				if changed {
					m.state.Navigation.SelectedIdx = m.listNav.Cursor()
				}
			},
			SupportsNavigation: true,
		}

	default:
		// ModeSearch, ModeCommand, ModeHelp, ModeConfirmSync, etc.
		// These modes don't support list navigation
		return &NavigatorContext{SupportsNavigation: false}
	}
}

// executeNavigation handles navigation key presses using the provided context.
// It updates the navigator state and invokes the OnNavigate callback for side effects.
func (m *Model) executeNavigation(ctx *NavigatorContext, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if !ctx.SupportsNavigation {
		return m, nil
	}

	// Handle DirectOffset mode (Diff view)
	if ctx.DirectOffset != nil {
		return m.executeDirectOffsetNavigation(ctx, msg)
	}

	// Sync navigator with current item count and viewport
	ctx.Navigator.SetItemCount(ctx.GetItemCount())
	ctx.Navigator.SetViewportHeight(ctx.GetViewportHeight())

	var changed bool

	switch msg.String() {
	case "up", "k":
		changed = ctx.Navigator.MoveUp()
	case "down", "j":
		changed = ctx.Navigator.MoveDown()
	case "pgup":
		changed = ctx.Navigator.PageUp()
	case "pgdown":
		changed = ctx.Navigator.PageDown()
	case "g":
		// Handle double-g timing for go-to-top
		now := time.Now().UnixMilli()
		if m.state.Navigation.LastGPressed > 0 && now-m.state.Navigation.LastGPressed < 500 {
			changed = ctx.Navigator.GoToTop()
			m.state.Navigation.LastGPressed = 0
		} else {
			m.state.Navigation.LastGPressed = now
			// Don't invoke OnNavigate for first 'g' press
			return m, nil
		}
	case "G":
		changed = ctx.Navigator.GoToBottom()
	}

	// Invoke post-navigation callback for side effects
	if ctx.OnNavigate != nil {
		ctx.OnNavigate(changed)
	}

	return m, nil
}

// executeDirectOffsetNavigation handles navigation for views using direct offset (Diff mode).
func (m *Model) executeDirectOffsetNavigation(ctx *NavigatorContext, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		*ctx.DirectOffset = max(0, *ctx.DirectOffset-1)
	case "down", "j":
		*ctx.DirectOffset = *ctx.DirectOffset + 1
	case "pgup":
		*ctx.DirectOffset = max(0, *ctx.DirectOffset-ctx.PageSize())
	case "pgdown":
		*ctx.DirectOffset = *ctx.DirectOffset + ctx.PageSize()
	case "g":
		now := time.Now().UnixMilli()
		if m.state.Navigation.LastGPressed > 0 && now-m.state.Navigation.LastGPressed < 500 {
			*ctx.DirectOffset = 0
			m.state.Navigation.LastGPressed = 0
		} else {
			m.state.Navigation.LastGPressed = now
		}
	case "G":
		// Set to large value; clamped on render
		*ctx.DirectOffset = 1 << 30
	}
	return m, nil
}
