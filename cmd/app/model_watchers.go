package main

import "github.com/darksworm/argonaut/pkg/model"

// cleanupAppWatcher stops the active applications watcher if present.
func (m *Model) cleanupAppWatcher() *Model {
	if m.appWatchCleanup != nil {
		m.appWatchCleanup()
		m.appWatchCleanup = nil
	}
	return m
}

// cleanupTreeWatchers stops all active tree watchers and clears the list.
// It marks the end of a tree session (leaving the view, or rebuilding for
// another app), so the side pane — a lens over that session — closes with it;
// auto-open reopens it against whatever tree loads next.
func (m *Model) cleanupTreeWatchers() *Model {
	if len(m.treeWatchCleanups) > 0 {
		for _, c := range m.treeWatchCleanups {
			if c != nil {
				c()
			}
		}
	}
	m.treeWatchCleanups = nil
	if m.treeStreamDone != nil {
		close(m.treeStreamDone)
		m.treeStreamDone = make(chan struct{})
	}
	m.closePane()
	return m
}

// safeChangeView changes navigation view and cleans up tree watchers if leaving tree view.
func (m *Model) safeChangeView(newView model.View) *Model {
	if m.state.Navigation.View == model.ViewTree && newView != model.ViewTree {
		m = m.cleanupTreeWatchers()
	}
	m.state.Navigation.View = newView
	// Reset list navigator when changing views
	m.listNav.Reset()
	return m
}
