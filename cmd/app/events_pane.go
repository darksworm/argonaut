package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// resolveAppNamespace resolves an app's ArgoCD namespace ("" = unset).
// Prefers the tree-scoped app (ADR-0004) before falling back to a name scan
// over the app list.
func (m *Model) resolveAppNamespace(appName string) string {
	if treeApp := m.state.UI.TreeApp; treeApp != nil && treeApp.Name == appName && treeApp.AppNamespace != nil {
		return *treeApp.AppNamespace
	}
	for i := range m.state.Apps {
		if m.state.Apps[i].Name == appName {
			if m.state.Apps[i].AppNamespace != nil {
				return *m.state.Apps[i].AppNamespace
			}
			return ""
		}
	}
	return ""
}

// panePageSize returns the number of rows a page scroll moves in a side
// pane — its rendered body height, from the same geometry the renderer uses.
func (m *Model) panePageSize() int {
	return max(1, m.paneLayout(m.viewportRowBudget()).paneBodyRows)
}

// modeBehindCommandBar returns the mode a dismissed command bar falls back
// to — a live side pane resumes owning the input.
func (m *Model) modeBehindCommandBar() model.Mode {
	if m.state.Events != nil {
		return model.ModeEvents
	}
	if m.state.SyncStatus != nil {
		return model.ModeSyncStatus
	}
	return model.ModeNormal
}

// closePanes closes whichever side pane is open and returns input to the tree.
func (m *Model) closePanes() {
	m.state.Events = nil
	m.state.SyncStatus = nil
	m.state.Mode = model.ModeNormal
}

// handlePaneModeKeys handles input while the events or sync-status pane is
// open. The pane owns the input: close (esc/q), pane switch (e/S), command
// mode (:) and scroll (nav router) are recognized; everything else is
// swallowed — the pane is a reading surface, not an action surface.
func (m *Model) handlePaneModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.closePanes()
		return m, nil
	case "e":
		if m.state.Mode == model.ModeSyncStatus {
			return m.handleShowEvents()
		}
		return m, nil
	case "S":
		if m.state.Mode == model.ModeEvents {
			return m.handleShowSyncStatus()
		}
		return m, nil
	case ":":
		return m.handleEnterCommandMode()
	}
	return m, nil
}

// handleShowSyncStatus opens the sync-status pane for the app that owns the
// selected tree row; unlike events it is app-scoped from any row.
func (m *Model) handleShowSyncStatus() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View != model.ViewTree || m.treeView == nil {
		return m, nil
	}
	appName := m.treeView.SelectedNodeApp()
	target := model.SyncStatusTarget{AppName: appName, AppNamespace: m.resolveAppNamespace(appName)}
	m.paneLoadSeq++
	m.state.Mode = model.ModeSyncStatus
	m.state.Events = nil
	m.state.SyncStatus = &model.SyncStatusState{Target: target, Loading: true, LoadSeq: m.paneLoadSeq}
	return m, m.loadSyncStatus(target, m.paneLoadSeq)
}

// handleShowEvents opens the events pane for the selected tree row: the
// application's own events on the synthetic root, the resource's events
// otherwise.
func (m *Model) handleShowEvents() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View != model.ViewTree || m.treeView == nil {
		return m, nil
	}
	appName := m.treeView.SelectedNodeApp()
	target := model.EventsTarget{AppName: appName, AppNamespace: m.resolveAppNamespace(appName)}
	if detail, ok := m.treeView.SelectedResourceDetail(); ok {
		target.Resource = model.EventsResource{
			Kind:      detail.Kind,
			Namespace: detail.Namespace,
			Name:      detail.Name,
			UID:       detail.UID,
		}
	}
	m.paneLoadSeq++
	m.state.Mode = model.ModeEvents
	m.state.SyncStatus = nil
	m.state.Events = &model.EventsState{Target: target, Loading: true, LoadSeq: m.paneLoadSeq}
	return m, m.loadEvents(target, m.paneLoadSeq)
}
