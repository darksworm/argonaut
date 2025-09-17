package main

import "github.com/darksworm/argonaut/pkg/model"

// cleanupTreeWatchers stops all active tree watchers and clears the list.
func (m *Model) cleanupTreeWatchers() *Model {
	if len(m.treeWatchCleanups) > 0 {
		for _, c := range m.treeWatchCleanups {
			if c != nil {
				c()
			}
		}
	}
	m.treeWatchCleanups = nil
	return m
}

// safeChangeView changes navigation view and cleans up tree watchers if leaving tree view.
func (m *Model) safeChangeView(newView model.View) *Model {
	if m.state.Navigation.View == model.ViewTree && newView != model.ViewTree {
		m = m.cleanupTreeWatchers()
	}
	m.state.Navigation.View = newView
	return m
}
