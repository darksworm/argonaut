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

// eventsAutoOpenEnabled reports whether the pane opens by itself in the tree
// view, honoring a session-scoped ":events on|off" before the config default.
func (m *Model) eventsAutoOpenEnabled() bool {
	if m.eventsAutoOpenOverride != nil {
		return *m.eventsAutoOpenOverride
	}
	return m.config.IsEventsAutoOpenEnabled()
}

// eventsTargetForSelection derives the events target from the tree cursor:
// app-level on the synthetic root, resource-scoped otherwise. A non-empty
// notice means the target cannot have events (resource Missing → no UID).
func (m *Model) eventsTargetForSelection() (model.EventsTarget, string, *model.ResourceStatusSummary) {
	appName := m.treeView.SelectedNodeApp()
	target := model.EventsTarget{AppName: appName, AppNamespace: m.resolveAppNamespace(appName)}
	if detail, ok := m.treeView.SelectedResourceDetail(); ok {
		target.Resource = model.EventsResource{
			Kind:      detail.Kind,
			Namespace: detail.Namespace,
			Name:      detail.Name,
			UID:       detail.UID,
		}
		status := &model.ResourceStatusSummary{
			Health:        detail.Health,
			Sync:          detail.Status,
			HealthMessage: detail.HealthMessage,
			CreatedAt:     detail.CreatedAt,
		}
		if detail.UID == "" {
			return target, "No events: resource does not exist in the cluster.", status
		}
		return target, "", status
	}
	return target, "", nil
}

// schedulePaneRefresh arms the auto-refresh interval for the given load;
// nil when auto-refresh is disabled.
func (m *Model) schedulePaneRefresh(loadSeq int) tea.Cmd {
	interval := m.config.GetEventsRefreshInterval()
	if interval <= 0 {
		return nil
	}
	epoch := m.switchEpoch
	return tea.Tick(interval, func(time.Time) tea.Msg {
		return model.PaneRefreshDueMsg{SwitchEpoch: epoch, LoadSeq: loadSeq}
	})
}

// paneRefreshCmds returns the background refetches for the open pane:
// details always, events unless the target cannot have any.
func (m *Model) paneRefreshCmds() tea.Cmd {
	st := m.state.Events
	cmds := []tea.Cmd{m.loadSyncStatus(model.SyncStatusTarget{
		AppName:      st.Target.AppName,
		AppNamespace: st.Target.AppNamespace,
	}, st.LoadSeq)}
	if st.Notice == "" {
		cmds = append(cmds, m.loadEvents(st.Target, st.LoadSeq))
	}
	return tea.Batch(cmds...)
}

// paneAgeTickMsg re-renders the open pane once a second so the border's
// "updated Ns ago" stays honest between refreshes. Display only.
type paneAgeTickMsg struct {
	switchEpoch int
	loadSeq     int
}

// schedulePaneAgeTick arms the next display tick; nil when the indicator is
// hidden (auto-refresh disabled).
func (m *Model) schedulePaneAgeTick(loadSeq int) tea.Cmd {
	if m.config.GetEventsRefreshInterval() <= 0 {
		return nil
	}
	epoch := m.switchEpoch
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return paneAgeTickMsg{switchEpoch: epoch, loadSeq: loadSeq}
	})
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
	prev := m.state.Events
	if prev == nil {
		return nil
	}
	target, notice, resourceStatus := m.eventsTargetForSelection()
	if target == prev.Target {
		return nil
	}
	m.paneLoadSeq++
	st := &model.EventsState{
		Target:         target,
		Notice:         notice,
		Loading:        notice == "",
		DetailsLoading: true,
		ResourceStatus: resourceStatus,
		LoadSeq:        m.paneLoadSeq,
	}
	// The app details are identical for every row of the same app —
	// carry them over instead of refetching on each cursor move
	if prev.Target.AppName == target.AppName && prev.Target.AppNamespace == target.AppNamespace &&
		!prev.DetailsLoading && prev.DetailsError == "" {
		st.Details = prev.Details
		st.DetailsLoading = false
	}
	m.state.Events = st
	if !st.Loading && !st.DetailsLoading {
		return tea.Batch(m.schedulePaneRefresh(m.paneLoadSeq), m.schedulePaneAgeTick(m.paneLoadSeq))
	}
	return tea.Batch(m.schedulePaneFetch(m.paneLoadSeq), m.schedulePaneRefresh(m.paneLoadSeq), m.schedulePaneAgeTick(m.paneLoadSeq))
}

// closePane closes the side pane; the tree keeps the input focus it already
// had — the pane is state, not a mode.
func (m *Model) closePane() {
	m.state.Events = nil
}

// handleShowEvents opens the events pane for the selected tree row: the
// application's own events on the synthetic root, the resource's events
// otherwise. A Missing resource opens with an inline notice instead of a
// fetch — the lens must show something wherever the cursor lands.
func (m *Model) handleShowEvents() (tea.Model, tea.Cmd) {
	if m.state.Navigation.View != model.ViewTree || m.treeView == nil {
		return m, nil
	}
	target, notice, resourceStatus := m.eventsTargetForSelection()
	m.paneLoadSeq++
	m.state.Events = &model.EventsState{
		Target:         target,
		Notice:         notice,
		Loading:        notice == "",
		DetailsLoading: true,
		ResourceStatus: resourceStatus,
		LoadSeq:        m.paneLoadSeq,
	}
	return m, tea.Batch(m.paneFetchCmds(), m.schedulePaneRefresh(m.paneLoadSeq), m.schedulePaneAgeTick(m.paneLoadSeq))
}

// paneFetchCmds returns the fetches the open pane still needs.
func (m *Model) paneFetchCmds() tea.Cmd {
	st := m.state.Events
	var cmds []tea.Cmd
	if st.Loading {
		cmds = append(cmds, m.loadEvents(st.Target, st.LoadSeq))
	}
	if st.DetailsLoading {
		cmds = append(cmds, m.loadSyncStatus(model.SyncStatusTarget{
			AppName:      st.Target.AppName,
			AppNamespace: st.Target.AppNamespace,
		}, st.LoadSeq))
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}
