package main

import (
	"encoding/json"
	"fmt"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
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

// handleResourceTreeLoaded processes resource tree loading completion
func (m *Model) handleResourceTreeLoaded(msg tea.Msg) (tea.Model, tea.Cmd) {
	resourceTreeMsg := msg.(model.ResourceTreeLoadedMsg)

	// Populate tree view with loaded data (single or multi-app)
	if m.treeView != nil && len(resourceTreeMsg.TreeJSON) > 0 {
		var tree api.ResourceTree
		if err := json.Unmarshal(resourceTreeMsg.TreeJSON, &tree); err == nil {
			m.treeView.SetAppMeta(resourceTreeMsg.AppName, resourceTreeMsg.Health, resourceTreeMsg.Sync)
			m.treeView.UpsertAppTree(resourceTreeMsg.AppName, &tree)
		}
		// Reset cursor for tree view
		m.state.Navigation.SelectedIdx = 0
		m.statusService.Set("Tree loaded")
	}
	// Loading is done (even if tree is empty)
	m.treeLoading = false

	return m, nil
}

// handleRollbackHistoryLoaded processes rollback deployment history loading
func (m *Model) handleRollbackHistoryLoaded(msg tea.Msg) (tea.Model, tea.Cmd) {
	rollbackMsg := msg.(model.RollbackHistoryLoadedMsg)

	// Initialize rollback state with deployment history
	m.state.Rollback = &model.RollbackState{
		AppName:         rollbackMsg.AppName,
		Rows:            rollbackMsg.Rows,
		CurrentRevision: rollbackMsg.CurrentRevision,
		Loading:         false,
		Mode:            "list",
		SelectedIdx:     0,
	}

	// Load metadata for first visible chunk (up to 10) asynchronously
	var cmds []tea.Cmd
	preload := minInt(10, len(rollbackMsg.Rows))
	for i := 0; i < preload; i++ {
		cmds = append(cmds, m.loadRevisionMetadata(rollbackMsg.AppName, i, rollbackMsg.Rows[i].Revision))
	}

	return m, tea.Batch(cmds...)
}

// handleRollbackMetadataLoaded processes revision metadata loading
func (m *Model) handleRollbackMetadataLoaded(msg tea.Msg) (tea.Model, tea.Cmd) {
	metadataMsg := msg.(model.RollbackMetadataLoadedMsg)

	// Update rollback row with loaded metadata
	if m.state.Rollback != nil && metadataMsg.RowIndex < len(m.state.Rollback.Rows) {
		row := &m.state.Rollback.Rows[metadataMsg.RowIndex]
		row.Author = &metadataMsg.Metadata.Author
		row.Date = &metadataMsg.Metadata.Date
		row.Message = &metadataMsg.Metadata.Message
	}
	return m, nil
}

// handleRollbackMetadataError processes revision metadata loading errors
func (m *Model) handleRollbackMetadataError(msg tea.Msg) (tea.Model, tea.Cmd) {
	errorMsg := msg.(model.RollbackMetadataErrorMsg)

	// Update rollback row with error
	if m.state.Rollback != nil && errorMsg.RowIndex < len(m.state.Rollback.Rows) {
		row := &m.state.Rollback.Rows[errorMsg.RowIndex]
		row.MetaError = &errorMsg.Error
	}
	return m, nil
}

// handleRollbackExecuted processes rollback execution completion
func (m *Model) handleRollbackExecuted(msg tea.Msg) (tea.Model, tea.Cmd) {
	rollbackMsg := msg.(model.RollbackExecutedMsg)

	// Handle rollback completion
	if rollbackMsg.Success {
		m.statusService.Set(fmt.Sprintf("Rollback initiated for %s", rollbackMsg.AppName))

		// Clear rollback state and return to normal mode
		m.state.Rollback = nil
		m.state.Mode = model.ModeNormal

		// Optionally start watching for updates
		if rollbackMsg.Watch {
			return m, m.startWatchingApplications()
		}
	} else {
		m.statusService.Set(fmt.Sprintf("Rollback failed for %s", rollbackMsg.AppName))
	}
	return m, nil
}

// handleTreeWatchStarted processes tree watch initialization
func (m *Model) handleTreeWatchStarted(msg tea.Msg) (tea.Model, tea.Cmd) {
	treeWatchMsg := msg.(treeWatchStartedMsg)

	if treeWatchMsg.cleanup != nil {
		m.treeWatchCleanups = append(m.treeWatchCleanups, treeWatchMsg.cleanup)
		m.statusService.Set("Watching tree…")
	}
	return m, nil
}

// handleRollbackNavigation processes rollback view navigation
func (m *Model) handleRollbackNavigation(msg tea.Msg) (tea.Model, tea.Cmd) {
	rollbackNavMsg := msg.(model.RollbackNavigationMsg)

	// Handle rollback navigation
	if m.state.Rollback != nil {
		switch rollbackNavMsg.Direction {
		case "up":
			if m.state.Rollback.SelectedIdx > 0 {
				m.state.Rollback.SelectedIdx--
				// Load metadata for newly selected row if not loaded
				row := m.state.Rollback.Rows[m.state.Rollback.SelectedIdx]
				if row.Author == nil && row.MetaError == nil {
					return m, m.loadRevisionMetadata(m.state.Rollback.AppName, m.state.Rollback.SelectedIdx, row.Revision)
				}
			}
		case "down":
			if m.state.Rollback.SelectedIdx < len(m.state.Rollback.Rows)-1 {
				m.state.Rollback.SelectedIdx++
				// Load metadata for newly selected row if not loaded
				row := m.state.Rollback.Rows[m.state.Rollback.SelectedIdx]
				var cmds []tea.Cmd
				if row.Author == nil && row.MetaError == nil {
					cmds = append(cmds, m.loadRevisionMetadata(m.state.Rollback.AppName, m.state.Rollback.SelectedIdx, row.Revision))
				}
				// Opportunistically preload the next two rows' metadata to reduce "loading" gaps
				for j := 1; j <= 2; j++ {
					idx := m.state.Rollback.SelectedIdx + j
					if idx < len(m.state.Rollback.Rows) {
						r := m.state.Rollback.Rows[idx]
						if r.Author == nil && r.MetaError == nil {
							cmds = append(cmds, m.loadRevisionMetadata(m.state.Rollback.AppName, idx, r.Revision))
						}
					}
				}
				if len(cmds) > 0 {
					return m, tea.Batch(cmds...)
				}
			}
		case "top":
			m.state.Rollback.SelectedIdx = 0
			// Load metadata for top row if not loaded
			if len(m.state.Rollback.Rows) > 0 {
				row := m.state.Rollback.Rows[0]
				if row.Author == nil && row.MetaError == nil {
					return m, m.loadRevisionMetadata(m.state.Rollback.AppName, 0, row.Revision)
				}
			}
		case "bottom":
			if len(m.state.Rollback.Rows) > 0 {
				m.state.Rollback.SelectedIdx = len(m.state.Rollback.Rows) - 1
				// Load metadata for bottom row if not loaded
				row := m.state.Rollback.Rows[m.state.Rollback.SelectedIdx]
				if row.Author == nil && row.MetaError == nil {
					return m, m.loadRevisionMetadata(m.state.Rollback.AppName, m.state.Rollback.SelectedIdx, row.Revision)
				}
			}
		}
	}
	return m, nil
}

// handleRollbackToggleOption processes rollback option toggling
func (m *Model) handleRollbackToggleOption(msg tea.Msg) (tea.Model, tea.Cmd) {
	toggleMsg := msg.(model.RollbackToggleOptionMsg)

	if m.state.Rollback != nil {
		switch toggleMsg.Option {
		case "prune":
			m.state.Rollback.Prune = !m.state.Rollback.Prune
		case "watch":
			m.state.Rollback.Watch = !m.state.Rollback.Watch
		case "dryrun":
			m.state.Rollback.DryRun = !m.state.Rollback.DryRun
		}
	}
	return m, nil
}

// handleRollbackConfirm processes rollback confirmation
func (m *Model) handleRollbackConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle rollback confirmation
	if m.state.Rollback != nil && m.state.Rollback.SelectedIdx < len(m.state.Rollback.Rows) {
		// Switch to confirmation mode
		m.state.Rollback.Mode = "confirm"
	}
	return m, nil
}

// handleRollbackCancel processes rollback cancellation
func (m *Model) handleRollbackCancel(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle rollback cancellation
	m.state.Rollback = nil
	m.state.Modals.RollbackAppName = nil
	m.state.Mode = model.ModeNormal
	return m, nil
}

// handleRollbackShowDiff processes rollback diff request
func (m *Model) handleRollbackShowDiff(msg tea.Msg) (tea.Model, tea.Cmd) {
	diffMsg := msg.(model.RollbackShowDiffMsg)

	// Handle rollback diff request
	if m.state.Rollback != nil {
		return m, m.startRollbackDiffSession(m.state.Rollback.AppName, diffMsg.Revision)
	}
	return m, nil
}

// Helper function for min calculation
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}