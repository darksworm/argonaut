package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
)

// Data loading message handlers

// handleSetApps updates the applications list
func (m *Model) handleSetApps(msg tea.Msg) (tea.Model, tea.Cmd) {
	setAppsMsg := msg.(model.SetAppsMsg)
	m.state.Apps = setAppsMsg.Apps
	return m, nil
}

// handleSetServer updates the server configuration
func (m *Model) handleSetServer(msg tea.Msg) (tea.Model, tea.Cmd) {
	setServerMsg := msg.(model.SetServerMsg)
	m.state.Server = setServerMsg.Server
	// Also fetch API version and start watching
	return m, tea.Batch(m.startWatchingApplications(), m.fetchAPIVersion())
}

// handleAppsLoaded processes the completion of initial app loading
func (m *Model) handleAppsLoaded(msg tea.Msg) (tea.Model, tea.Cmd) {
	appsLoadedMsg := msg.(model.AppsLoadedMsg)

	cblog.With("component", "model").Info("AppsLoadedMsg received",
		"apps_count", len(appsLoadedMsg.Apps),
		"watchChan_nil", m.watchChan == nil)
	m.state.Apps = appsLoadedMsg.Apps
	// Turn off initial loading modal if it was active
	m.state.Modals.InitialLoading = false

	// Only start watching if we haven't already started
	// (watchChan is set when watch starts)
	if m.watchChan == nil {
		cblog.With("component", "model").Info("Starting watch as watchChan is nil")
		// Start watching for app updates after initial load
		return m, tea.Batch(
			func() tea.Msg { return model.SetModeMsg{Mode: model.ModeNormal} },
			m.startWatchingApplications(),
		)
	}
	// If watch is already running, just switch to normal mode and keep consuming
	return m, tea.Batch(
		func() tea.Msg { return model.SetModeMsg{Mode: model.ModeNormal} },
		m.consumeWatchEvent(),
	)
}

// handleAppUpdated processes real-time application updates
func (m *Model) handleAppUpdated(msg tea.Msg) (tea.Model, tea.Cmd) {
	appUpdatedMsg := msg.(model.AppUpdatedMsg)

	// upsert app
	updated := appUpdatedMsg.App
	cblog.With("component", "watch").Debug("AppUpdatedMsg received",
		"app", updated.Name,
		"health", updated.Health,
		"sync", updated.Sync)
	found := false
	for i, a := range m.state.Apps {
		if a.Name == updated.Name {
			m.state.Apps[i] = updated
			found = true
			break
		}
	}
	if !found {
		m.state.Apps = append(m.state.Apps, updated)
	}
	cblog.With("component", "watch").Debug("Apps list updated",
		"total_apps", len(m.state.Apps),
		"updated_app", updated.Name)
	return m, m.consumeWatchEvent()
}

// handleAppDeleted processes application deletion events
func (m *Model) handleAppDeleted(msg tea.Msg) (tea.Model, tea.Cmd) {
	appDeletedMsg := msg.(model.AppDeletedMsg)

	name := appDeletedMsg.AppName
	filtered := m.state.Apps[:0]
	for _, a := range m.state.Apps {
		if a.Name != name {
			filtered = append(filtered, a)
		}
	}
	m.state.Apps = filtered
	return m, m.consumeWatchEvent()
}

// handleStatusChange processes status message updates
func (m *Model) handleStatusChange(msg tea.Msg) (tea.Model, tea.Cmd) {
	statusMsg := msg.(model.StatusChangeMsg)

	// Now safe to log since we're using file logging
	m.statusService.Set(statusMsg.Status)

	// Clear diff loading state for diff-related status messages
	if (statusMsg.Status == "No diffs" || statusMsg.Status == "No differences") && m.state.Diff != nil {
		m.state.Diff.Loading = false
	}

	return m, m.consumeWatchEvent()
}

// handleWatchStarted processes watch initialization
func (m *Model) handleWatchStarted(msg tea.Msg) (tea.Model, tea.Cmd) {
	watchMsg := msg.(watchStartedMsg)

	// Set up the watch channel with proper forwarding
	m.watchChan = make(chan services.ArgoApiEvent, 100)
	cblog.With("component", "watch").Debug("watchStartedMsg: setting up watch channel forwarding")
	go func() {
		cblog.With("component", "watch").Debug("watchStartedMsg: goroutine started")
		eventCount := 0
		for ev := range watchMsg.eventChan {
			eventCount++
			cblog.With("component", "watch").Debug("watchStartedMsg: forwarding event",
				"event_number", eventCount,
				"type", ev.Type)
			m.watchChan <- ev
		}
		cblog.With("component", "watch").Debug("watchStartedMsg: eventChan closed, closing watchChan")
		close(m.watchChan)
	}()
	// Start consuming events
	return m, tea.Batch(
		m.consumeWatchEvent(),
		func() tea.Msg { return model.StatusChangeMsg{Status: "Watching for changes..."} },
	)
}