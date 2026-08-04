package main

import (
	"time"

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

// paneFetchDebounce is how long the pane waits after a retarget before
// fetching, so holding j/k fetches once for the row the user settles on.
const paneFetchDebounce = 200 * time.Millisecond

// eventsTargetForSelection derives the events target from the tree cursor:
// app-level on the synthetic root, resource-scoped otherwise. A non-empty
// notice means the target cannot have events (resource Missing → no UID).
func (m *Model) eventsTargetForSelection() (model.EventsTarget, string) {
	appName := m.treeView.SelectedNodeApp()
	target := model.EventsTarget{AppName: appName, AppNamespace: m.resolveAppNamespace(appName)}
	if detail, ok := m.treeView.SelectedResourceDetail(); ok {
		target.Resource = model.EventsResource{
			Kind:      detail.Kind,
			Namespace: detail.Namespace,
			Name:      detail.Name,
			UID:       detail.UID,
		}
		if detail.UID == "" {
			return target, "No events: resource does not exist in the cluster."
		}
	}
	return target, ""
}

// schedulePaneFetch arms the retarget debounce for the given load.
func (m *Model) schedulePaneFetch(loadSeq int) tea.Cmd {
	epoch := m.switchEpoch
	return tea.Tick(paneFetchDebounce, func(time.Time) tea.Msg {
		return model.PaneFetchDueMsg{SwitchEpoch: epoch, LoadSeq: loadSeq}
	})
}

// retargetOpenPane points the open pane at the current tree selection,
// scheduling a debounced fetch when the target actually changed.
func (m *Model) retargetOpenPane() tea.Cmd {
	switch {
	case m.state.Events != nil:
		target, notice := m.eventsTargetForSelection()
		if target == m.state.Events.Target {
			return nil
		}
		m.paneLoadSeq++
		m.state.Events = &model.EventsState{
			Target:  target,
			Notice:  notice,
			Loading: notice == "",
			LoadSeq: m.paneLoadSeq,
		}
		if notice != "" {
			return nil
		}
		return m.schedulePaneFetch(m.paneLoadSeq)
	case m.state.SyncStatus != nil:
		appName := m.treeView.SelectedNodeApp()
		target := model.SyncStatusTarget{AppName: appName, AppNamespace: m.resolveAppNamespace(appName)}
		if target == m.state.SyncStatus.Target {
			return nil
		}
		m.paneLoadSeq++
		m.state.SyncStatus = &model.SyncStatusState{Target: target, Loading: true, LoadSeq: m.paneLoadSeq}
		return m.schedulePaneFetch(m.paneLoadSeq)
	}
	return nil
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
// open. The pane is a lens over the tree: navigation keys keep moving the
// tree selection (the pane follows it), the shifted variants scroll the
// pane, esc/q close it, e/S switch panes, ':' opens the command bar.
// Everything else is swallowed — the pane is a reading surface, not an
// action surface.
func (m *Model) handlePaneModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k", "down", "j", "pgup", "pgdown", "g", "G":
		ctx := m.treeNavigatorContext()
		if !ctx.SupportsNavigation {
			return m, nil
		}
		// The selection may have been set outside the navigator (search
		// jumps, direct SetSelectedIndex) — start from where the tree is
		m.treeNav.SetItemCount(m.treeView.VisibleCount())
		m.treeNav.SetCursor(m.treeView.SelectedIndex())
		_, _ = m.executeNavigation(ctx, msg)
		return m, m.retargetOpenPane()
	case "J", "shift+down":
		if st := m.state.Events; st != nil {
			st.Offset++
		} else if st := m.state.SyncStatus; st != nil {
			st.Offset++
		}
		return m, nil
	case "K", "shift+up":
		if st := m.state.Events; st != nil {
			st.Offset = max(0, st.Offset-1)
		} else if st := m.state.SyncStatus; st != nil {
			st.Offset = max(0, st.Offset-1)
		}
		return m, nil
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
// otherwise. A Missing resource opens with an inline notice instead of a
// fetch — the lens must show something wherever the cursor lands.
func (m *Model) handleShowEvents() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View != model.ViewTree || m.treeView == nil {
		return m, nil
	}
	target, notice := m.eventsTargetForSelection()
	m.paneLoadSeq++
	m.state.Mode = model.ModeEvents
	m.state.SyncStatus = nil
	m.state.Events = &model.EventsState{
		Target:  target,
		Notice:  notice,
		Loading: notice == "",
		LoadSeq: m.paneLoadSeq,
	}
	if notice != "" {
		return m, nil
	}
	return m, m.loadEvents(target, m.paneLoadSeq)
}
