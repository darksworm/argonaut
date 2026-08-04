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
		return m.schedulePaneRefresh(m.paneLoadSeq)
	}
	return tea.Batch(m.schedulePaneFetch(m.paneLoadSeq), m.schedulePaneRefresh(m.paneLoadSeq))
}

// modeBehindCommandBar returns the mode a dismissed command bar falls back
// to — a live side pane resumes owning the input.
func (m *Model) modeBehindCommandBar() model.Mode {
	if m.state.Events != nil {
		return model.ModeEvents
	}
	return model.ModeNormal
}

// closePanes closes the side pane and returns input to the tree.
func (m *Model) closePanes() {
	m.state.Events = nil
	m.state.Mode = model.ModeNormal
}

// handlePaneModeKeys handles input while the side pane is open. The pane is
// a lens over the tree: navigation keys keep moving the tree selection (the
// pane follows it), the shifted variants scroll the pane, esc/q close it,
// ':' opens the command bar — and any other tree hotkey closes the lens and
// acts on the selected row as if the pane were not there.
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
		}
		return m, nil
	case "K", "shift+up":
		if st := m.state.Events; st != nil {
			st.Offset = max(0, st.Offset-1)
		}
		return m, nil
	case "esc", "q":
		m.closePanes()
		return m, nil
	case ":":
		return m.handleEnterCommandMode()
	}
	// Anything else is a tree hotkey: the lens closes and the key acts on
	// the selected row as if the pane were not there.
	m.closePanes()
	return m.handleTreeViewKeys(msg)
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
	m.state.Mode = model.ModeEvents
	m.state.Events = &model.EventsState{
		Target:         target,
		Notice:         notice,
		Loading:        notice == "",
		DetailsLoading: true,
		ResourceStatus: resourceStatus,
		LoadSeq:        m.paneLoadSeq,
	}
	return m, tea.Batch(m.paneFetchCmds(), m.schedulePaneRefresh(m.paneLoadSeq))
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
