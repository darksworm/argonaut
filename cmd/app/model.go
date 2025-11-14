package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/autocomplete"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/tui"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// Model represents the main Bubbletea model containing all application state
type Model struct {
	// Core application state
	state *model.AppState

	// Services
	argoService       services.ArgoApiService
	navigationService services.NavigationService
	statusService     services.StatusService
	updateService     services.UpdateService

	// Interactive input components using bubbles
	inputComponents *InputComponentState

	// Autocomplete engine for command suggestions
	autocompleteEngine *autocomplete.AutocompleteEngine

	// Internal flags
	ready bool
	err   error

	// Watch channel for Argo events
	watchChan chan services.ArgoApiEvent

	// bubbles spinner for loading
	spinner spinner.Model

	// bubbles tables for all views
	appsTable       table.Model
	clustersTable   table.Model
	namespacesTable table.Model
	projectsTable   table.Model

	// Bubble Tea program reference for terminal hand-off (pager integration)
	program *tea.Program
	inPager bool

	// Tree view component
	treeView *treeview.TreeView

	// Tree watch internal channel delivery
	treeStream chan model.ResourceTreeStreamMsg

	// Tree loading overlay state
	treeLoading bool

	// Tree view scroll offset
	treeScrollOffset int

	// List view scroll offset (for apps, clusters, namespaces, projects)
	listScrollOffset int

	// Cleanup callbacks for active tree watchers
	treeWatchCleanups []func()

	// Debug: render counter
	renderCount int

	// Theme selection helpers
	themeOptions []themeOption
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Terminal/System messages
	case tea.WindowSizeMsg:
		m.state.Terminal.Rows = msg.Height
		m.state.Terminal.Cols = msg.Width
		if m.treeView != nil {
			m.treeView.SetSize(msg.Width, msg.Height)
		}
		if !m.ready {
			m.ready = true
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "Ready"}
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.PasteMsg:
		// Handle clipboard paste events

		// Handle based on current mode
		if m.state.Mode == model.ModeSearch {
			// For search mode, append pasted text to current search
			currentValue := m.inputComponents.GetSearchValue()
			newValue := currentValue + string(msg)
			m.inputComponents.SetSearchValue(newValue)
			m.state.UI.SearchQuery = newValue
			// Clamp selection within new filtered results
			m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
				m.state.Navigation.SelectedIdx,
				len(m.getVisibleItems()),
			)
			return m, nil
		} else if m.state.Mode == model.ModeCommand {
			// For command mode, append pasted text to current command
			currentValue := m.inputComponents.GetCommandValue()
			newValue := currentValue + string(msg)
			m.inputComponents.SetCommandValue(newValue)
			m.state.UI.Command = newValue
			m.state.UI.CommandInvalid = false
			return m, nil
		}
		return m, nil

	// Tree stream messages from watcher goroutine
	case model.ResourceTreeStreamMsg:
		cblog.With("component", "ui").Debug("Processing tree stream message", "app", msg.AppName, "hasData", len(msg.TreeJSON) > 0)
		if len(msg.TreeJSON) > 0 && m.treeView != nil && m.state.Navigation.View == model.ViewTree {
			var tree api.ResourceTree
			if err := json.Unmarshal(msg.TreeJSON, &tree); err == nil {
				cblog.With("component", "ui").Debug("Updating tree view", "app", msg.AppName, "nodes", len(tree.Nodes))
				m.treeView.UpsertAppTree(msg.AppName, &tree)
			} else {
				cblog.With("component", "ui").Error("Failed to unmarshal tree", "err", err, "app", msg.AppName)
			}
		}
		// Any tree stream activity implies data is arriving; clear loading overlay
		m.treeLoading = false
		return m, m.consumeTreeEvent()

	// Tree watch started (store cleanup)
	case treeWatchStartedMsg:
		if msg.cleanup != nil {
			m.treeWatchCleanups = append(m.treeWatchCleanups, msg.cleanup)
			m.statusService.Set("Watching tree…")
		}
		return m, nil

		// Spinner messages
	case spinner.TickMsg:
		if m.inPager {
			// Suspend spinner updates while pager owns the terminal
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	// Navigation messages
	case model.SetViewMsg:
		m.state.Navigation.View = msg.View
		return m, nil

	case model.SetSelectedIdxMsg:
		// Keep selection within bounds of currently visible items
		m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
			msg.SelectedIdx,
			len(m.getVisibleItems()),
		)
		return m, nil

	case model.ResetNavigationMsg:
		m.state.Navigation.SelectedIdx = 0
		if msg.View != nil {
			m.state.Navigation.View = *msg.View
		}
		return m, nil

	// Selection messages
	case model.SetSelectedAppsMsg:
		m.state.Selections.SelectedApps = msg.Apps
		return m, nil

	case model.ClearAllSelectionsMsg:
		m.state.Selections = *model.NewSelectionState()
		return m, nil

	// UI messages
	case model.SetSearchQueryMsg:
		m.state.UI.SearchQuery = msg.Query
		return m, nil

	case model.SetActiveFilterMsg:
		m.state.UI.ActiveFilter = msg.Filter
		return m, nil

	case model.SetCommandMsg:
		m.state.UI.Command = msg.Command
		return m, nil

	case model.ClearFiltersMsg:
		m.state.UI.SearchQuery = ""
		m.state.UI.ActiveFilter = ""
		return m, nil

	case model.SetAPIVersionMsg:
		m.state.APIVersion = msg.Version
		return m, nil

		// Mode messages
	case model.SetModeMsg:
		oldMode := m.state.Mode
		m.state.Mode = msg.Mode
		cblog.With("component", "model").Info("SetModeMsg received",
			"old_mode", oldMode,
			"new_mode", msg.Mode)
		// [MODE] Switching from %s to %s - removed printf to avoid TUI interference

		// Handle mode transitions
		if msg.Mode == model.ModeLoading && oldMode != model.ModeLoading {
			cblog.With("component", "model").Info("Triggering initial load for ModeLoading")
			// Start loading applications from API when transitioning to loading mode
			// [MODE] Triggering API load for loading mode - removed printf to avoid TUI interference
			return m, m.startLoadingApplications()
		}

		// If entering diff mode with content available, show in external pager
		if msg.Mode == model.ModeDiff && m.state.Diff != nil && len(m.state.Diff.Content) > 0 && !m.state.Diff.Loading {
			title := m.state.Diff.Title
			body := strings.Join(m.state.Diff.Content, "\n")
			return m, m.openTextPager(title, body)
		}

		return m, nil

	// Data messages
	case model.SetAppsMsg:
		m.state.Apps = msg.Apps
		return m, nil

	case model.SetServerMsg:
		m.state.Server = msg.Server
		// Also fetch API version and start watching
		return m, tea.Batch(m.startWatchingApplications(), m.fetchAPIVersion())

	case watchStartedMsg:
		// Set up the watch channel with proper forwarding
		m.watchChan = make(chan services.ArgoApiEvent, 100)
		cblog.With("component", "watch").Debug("watchStartedMsg: setting up watch channel forwarding")
		go func() {
			cblog.With("component", "watch").Debug("watchStartedMsg: goroutine started")
			eventCount := 0
			for ev := range msg.eventChan {
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
		)

	// API Event messages
	case model.AppsLoadedMsg:
		cblog.With("component", "model").Info("AppsLoadedMsg received",
			"apps_count", len(msg.Apps),
			"watchChan_nil", m.watchChan == nil)
		m.state.Apps = msg.Apps
		// Turn off initial loading modal if it was active
		m.state.Modals.InitialLoading = false
		// m.ui.UpdateListItems(m.state)

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

	case model.AppUpdatedMsg:
		// upsert app
		updated := msg.App
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

	case model.AppDeletedMsg:
		name := msg.AppName
		// Remove app while preserving order
		for i, a := range m.state.Apps {
			if a.Name == name {
				// Remove the app at index i by combining slices before and after
				m.state.Apps = append(m.state.Apps[:i], m.state.Apps[i+1:]...)
				break
			}
		}
		// Keep selection at the same index position
		// Only adjust if selection is now beyond the list bounds
		visibleItems := m.getVisibleItemsForCurrentView()
		if m.state.Navigation.SelectedIdx >= len(visibleItems) && len(visibleItems) > 0 {
			m.state.Navigation.SelectedIdx = len(visibleItems) - 1
		}
		return m, m.consumeWatchEvent()

	case model.StatusChangeMsg:
		// Now safe to log since we're using file logging
		m.statusService.Set(msg.Status)

		// Clear diff loading state for diff-related status messages
		if (msg.Status == "No diffs" || msg.Status == "No differences") && m.state.Diff != nil {
			m.state.Diff.Loading = false
		}

		return m, m.consumeWatchEvent()

	case model.ResourceTreeLoadedMsg:
		// Populate tree view with loaded data (single or multi-app)
		if m.treeView != nil && len(msg.TreeJSON) > 0 {
			var tree api.ResourceTree
			if err := json.Unmarshal(msg.TreeJSON, &tree); err == nil {
				m.treeView.SetAppMeta(msg.AppName, msg.Health, msg.Sync)
				m.treeView.UpsertAppTree(msg.AppName, &tree)
			}
			// Reset cursor for tree view
			m.state.Navigation.SelectedIdx = 0
			m.statusService.Set("Tree loaded")
		}
		// Clear loading overlay once initial tree is loaded
		m.treeLoading = false
		return m, nil

		// removed: resources list loader

		// Old spinner TickMsg removed - now using bubbles spinner

	case model.StructuredErrorMsg:
		// Handle structured errors with proper error state management
		if msg.Error != nil {
			errorMsg := fmt.Sprintf("Error: %s", msg.Error.Message)
			if msg.Error.UserAction != "" {
				errorMsg += fmt.Sprintf(" - %s", msg.Error.UserAction)
			}
			m.statusService.Error(errorMsg)

			// Debug: Log structured error details
			cblog.With("component", "tui").Debug("StructuredErrorMsg",
				"category", msg.Error.Category, "code", msg.Error.Code, "message", msg.Error.Message)

			// Update error state so the error view can show full details
			tui.UpdateAppErrorState(m.state, msg.Error)
		}

		// Clear any loading states that might be active
		if m.state.Diff != nil {
			m.state.Diff.Loading = false
		}
		if m.state.Modals.ConfirmSyncLoading {
			m.state.Modals.ConfirmSyncLoading = false
			m.state.Modals.ConfirmTarget = nil
			// Set mode to error to show the error immediately
			m.state.Mode = model.ModeError
		}
		// Turn off initial loading modal if it was active
		m.state.Modals.InitialLoading = false

		// If we were in the loading mode when the structured error arrived, switch to error view
		if msg.Error != nil {
			if msg.Error.Category == apperrors.ErrorAuth {
				m.state.Mode = model.ModeAuthRequired
			} else if m.state.Mode == model.ModeLoading {
				m.state.Mode = model.ModeError
			}
		}

		// If we have a structured error with high severity, switch to error mode
		if msg.Error != nil && msg.Error.Severity == apperrors.SeverityHigh {
			m.state.Mode = model.ModeError
		}

		return m, nil

	case model.ApiErrorMsg:
		// If we're already in auth-required mode, suppress generic API errors to avoid
		// overriding the auth-required view with a generic error panel.
		if m.state.Mode == model.ModeAuthRequired {
			return m, nil
		}
		// Log error and store structured error in state for display
		fullErrorMsg := fmt.Sprintf("API Error: %s", msg.Message)
		if msg.StatusCode > 0 {
			fullErrorMsg = fmt.Sprintf("API Error (%d): %s", msg.StatusCode, msg.Message)
		}
		m.statusService.Error(fullErrorMsg)

		// Clear any loading states that might be active
		if m.state.Diff != nil {
			m.state.Diff.Loading = false
		}
		if m.state.Modals.ConfirmSyncLoading {
			m.state.Modals.ConfirmSyncLoading = false
			m.state.Modals.ConfirmTarget = nil
			if m.state.Mode == model.ModeConfirmSync {
				m.state.Mode = model.ModeNormal
			}
		}
		// Turn off initial loading modal if it was active
		m.state.Modals.InitialLoading = false

		// If we were loading tree view, return to apps view
		if m.state.Navigation.View == model.ViewTree {
			m = m.cleanupTreeWatchers()
			m.state.Navigation.View = model.ViewApps
		}

		// Handle rollback-specific errors
		if m.state.Mode == model.ModeRollback {
			// If we're not in an active rollback execution (i.e., not loading), keep error in modal
			if m.state.Rollback != nil && !m.state.Rollback.Loading {
				// Initialize rollback state with error if not exists
				if m.state.Rollback == nil && m.state.Modals.RollbackAppName != nil {
					m.state.Rollback = &model.RollbackState{
						AppName: *m.state.Modals.RollbackAppName,
						Loading: false,
						Error:   msg.Message,
						Mode:    "list",
					}
				} else {
					// Update existing rollback state with error
					m.state.Rollback.Loading = false
					m.state.Rollback.Error = msg.Message
				}
				// Stay in rollback mode to show the error inline
				return m, nil
			}
			// else: in active rollback execution, fall through to generic error screen below
		}

		// Store structured error information in state
		m.state.CurrentError = &model.ApiError{
			Message:    msg.Message,
			StatusCode: msg.StatusCode,
			ErrorCode:  msg.ErrorCode,
			Details:    msg.Details,
			Timestamp:  time.Now().Unix(),
		}

		return m, func() tea.Msg {
			return model.SetModeMsg{Mode: model.ModeError}
		}

	case pauseRenderingMsg:
		m.inPager = true
		return m, nil

	case resumeRenderingMsg:
		m.inPager = false
		return m, nil

	case pagerDoneMsg:
		// Restore pager state
		m.inPager = false

		// If there was an error, display it
		if msg.Err != nil {
			cblog.With("component", "pager").Error("Pager error", "err", msg.Err)
			// Set error state and display the error on screen
			m.state.CurrentError = &model.ApiError{
				Message:    "Pager Error: " + msg.Err.Error(),
				StatusCode: 0,
				ErrorCode:  1001, // Custom error code for pager errors
				Details:    "Failed to open text pager",
				Timestamp:  time.Now().Unix(),
			}
			return m, func() tea.Msg {
				return model.SetModeMsg{Mode: model.ModeError}
			}
		}

		// No error, go back to normal mode
		m.state.Mode = model.ModeNormal
		return m, nil

	case model.AuthErrorMsg:
		// Log error and store in model for display
		m.statusService.Error(msg.Error.Error())
		m.err = msg.Error

		// Turn off initial loading modal if it was active
		m.state.Modals.InitialLoading = false

		// Handle rollback-specific auth errors
		if m.state.Mode == model.ModeRollback {
			// Initialize rollback state with error if not exists
			if m.state.Rollback == nil && m.state.Modals.RollbackAppName != nil {
				m.state.Rollback = &model.RollbackState{
					AppName: *m.state.Modals.RollbackAppName,
					Loading: false,
					Error:   "Authentication required: " + msg.Error.Error(),
					Mode:    "list",
				}
			} else if m.state.Rollback != nil {
				// Update existing rollback state with auth error
				m.state.Rollback.Loading = false
				m.state.Rollback.Error = "Authentication required: " + msg.Error.Error()
			}
			// Stay in rollback mode to show the error
			return m, nil
		}

		return m, tea.Batch(func() tea.Msg { return model.SetModeMsg{Mode: model.ModeAuthRequired} })

	// Navigation update messages
	case model.NavigationUpdateMsg:
		if msg.NewView != nil {
			m.state.Navigation.View = *msg.NewView
		}
		if msg.ScopeClusters != nil {
			m.state.Selections.ScopeClusters = msg.ScopeClusters
		}
		if msg.ScopeNamespaces != nil {
			m.state.Selections.ScopeNamespaces = msg.ScopeNamespaces
		}
		if msg.ScopeProjects != nil {
			m.state.Selections.ScopeProjects = msg.ScopeProjects
		}
		if msg.SelectedApps != nil {
			m.state.Selections.SelectedApps = msg.SelectedApps
		}
		if msg.ShouldResetNavigation {
			m.state.Navigation.SelectedIdx = 0
		}
		// m.ui.UpdateListItems(m.state)
		return m, nil

	case model.SyncCompletedMsg:
		// Handle single app sync completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Sync initiated for %s", msg.AppName))

			// Show tree view if watch is enabled
			if m.state.Modals.ConfirmSyncWatch {
				// Close confirm modal/loading state before switching views
				m.state.Modals.ConfirmTarget = nil
				m.state.Modals.ConfirmSyncLoading = false
				if m.state.Mode == model.ModeConfirmSync {
					m.state.Mode = model.ModeNormal
				}
				// Clean up any existing tree watchers before starting new one
				m.cleanupTreeWatchers()
				m.state.Navigation.View = model.ViewTree
				m.state.UI.TreeAppName = &msg.AppName
				// find app
				var appObj model.App
				found := false
				for _, a := range m.state.Apps {
					if a.Name == msg.AppName {
						appObj = a
						found = true
						break
					}
				}
				if !found {
					appObj = model.App{Name: msg.AppName}
				}
				return m, tea.Batch(m.startLoadingResourceTree(appObj), m.startWatchingResourceTree(appObj), m.consumeTreeEvent())
			}
		} else {
			m.statusService.Set("Sync cancelled")
		}
		// Close confirm modal/loading state if open (non-watch path)
		m.state.Modals.ConfirmTarget = nil
		m.state.Modals.ConfirmSyncLoading = false
		if m.state.Mode == model.ModeConfirmSync && !m.state.Modals.ConfirmSyncWatch {
			m.state.Mode = model.ModeNormal
		}
		return m, nil

	case model.AppDeleteRequestMsg:
		// Handle application delete request
		m.state.Modals.DeleteLoading = true
		return m, m.deleteApplication(msg)

	case model.AppDeleteSuccessMsg:
		// Handle successful application deletion
		m.statusService.Set(fmt.Sprintf("Application %s deleted successfully", msg.AppName))

		// Remove app from local state while preserving order
		for i, app := range m.state.Apps {
			if app.Name == msg.AppName {
				// Remove the app at index i by combining slices before and after
				m.state.Apps = append(m.state.Apps[:i], m.state.Apps[i+1:]...)
				break
			}
		}

		// Clear modal state and return to normal mode
		m.state.Mode = model.ModeNormal
		m.state.Modals.DeleteAppName = nil
		m.state.Modals.DeleteAppNamespace = nil
		m.state.Modals.DeleteConfirmationKey = ""
		m.state.Modals.DeleteError = nil
		m.state.Modals.DeleteLoading = false

		// Keep selection at the same index position
		// Only adjust if selection is now beyond the list bounds
		visibleItems := m.getVisibleItemsForCurrentView()
		if m.state.Navigation.SelectedIdx >= len(visibleItems) && len(visibleItems) > 0 {
			m.state.Navigation.SelectedIdx = len(visibleItems) - 1
		}

		return m, nil

	case model.AppDeleteErrorMsg:
		// Handle application deletion error
		m.statusService.Set(fmt.Sprintf("Failed to delete %s: %s", msg.AppName, msg.Error))
		m.state.Modals.DeleteError = &msg.Error
		m.state.Modals.DeleteLoading = false
		// Keep modal open to show error
		return m, nil

	case model.MultiSyncCompletedMsg:
		// Handle multiple app sync completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Sync initiated for %d app(s)", msg.AppCount))
			if m.state.Modals.ConfirmSyncWatch && len(m.state.Selections.SelectedApps) > 1 {
				// Snapshot selected names before clearing
				sel := m.state.Selections.SelectedApps
				names := make([]string, 0, len(sel))
				for name, ok := range sel {
					if ok {
						names = append(names, name)
					}
				}
				if len(names) > 0 {
					var cmds []tea.Cmd
					// Clean up any existing tree watchers first
					m.cleanupTreeWatchers()
					// Reset tree view for multi-app session
					m.treeView = treeview.NewTreeView(0, 0)
					m.treeView.ApplyTheme(currentPalette)
					m.treeScrollOffset = 0 // Reset scroll position
					m.state.SaveNavigationState()
					m.state.Navigation.View = model.ViewTree
					// Clear single-app tracker
					m.state.UI.TreeAppName = nil
					m.treeLoading = true
					for _, n := range names {
						var appObj *model.App
						for i := range m.state.Apps {
							if m.state.Apps[i].Name == n {
								appObj = &m.state.Apps[i]
								break
							}
						}
						if appObj == nil {
							tmp := model.App{Name: n}
							appObj = &tmp
						}
						cmds = append(cmds, m.startLoadingResourceTree(*appObj))
						cmds = append(cmds, m.startWatchingResourceTree(*appObj))
					}
					// Close modal before switching
					m.state.Modals.ConfirmTarget = nil
					m.state.Modals.ConfirmSyncLoading = false
					if m.state.Mode == model.ModeConfirmSync {
						m.state.Mode = model.ModeNormal
					}
					// Clear selections after queueing
					m.state.Selections.SelectedApps = model.NewStringSet()
					cmds = append(cmds, m.consumeTreeEvent())
					return m, tea.Batch(cmds...)
				}
			}
			// Clear selections when not opening multi tree
			m.state.Selections.SelectedApps = model.NewStringSet()
		}
		// Close confirm modal/loading state if open
		m.state.Modals.ConfirmTarget = nil
		m.state.Modals.ConfirmSyncLoading = false
		if m.state.Mode == model.ModeConfirmSync {
			m.state.Mode = model.ModeNormal
		}
		return m, nil

	case model.MultiDeleteCompletedMsg:
		// Handle multiple app delete completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Successfully deleted %d app(s)", msg.AppCount))
			// Clear selections after successful multi-delete
			m.state.Selections.SelectedApps = model.NewStringSet()
		}
		// Close confirm delete modal/loading state if open
		m.state.Modals.DeleteAppName = nil
		m.state.Modals.DeleteAppNamespace = nil
		m.state.Modals.DeleteConfirmationKey = ""
		m.state.Modals.DeleteError = nil
		m.state.Modals.DeleteLoading = false
		if m.state.Mode == model.ModeConfirmAppDelete {
			m.state.Mode = model.ModeNormal
		}
		// Keep selection at the same index position
		// Only adjust if selection is now beyond the list bounds
		visibleItems := m.getVisibleItemsForCurrentView()
		if m.state.Navigation.SelectedIdx >= len(visibleItems) && len(visibleItems) > 0 {
			m.state.Navigation.SelectedIdx = len(visibleItems) - 1
		}
		return m, nil

	// Rollback Messages
	case model.RollbackHistoryLoadedMsg:
		// Initialize rollback state with deployment history
		m.state.Rollback = &model.RollbackState{
			AppName:         msg.AppName,
			Rows:            msg.Rows,
			CurrentRevision: msg.CurrentRevision,
			SelectedIdx:     0,
			Loading:         false,
			Mode:            "list",
			Prune:           false,
			Watch:           true,
			DryRun:          false,
		}

		// Start loading metadata for the first visible chunk (up to 10)
		var cmds []tea.Cmd
		preload := min(10, len(msg.Rows))
		for i := 0; i < preload; i++ {
			cmds = append(cmds, m.loadRevisionMetadata(msg.AppName, i, msg.Rows[i].Revision))
		}

		return m, tea.Batch(cmds...)

	case model.RollbackMetadataLoadedMsg:
		// Update rollback row with loaded metadata
		if m.state.Rollback != nil && msg.RowIndex < len(m.state.Rollback.Rows) {
			row := &m.state.Rollback.Rows[msg.RowIndex]
			row.Author = &msg.Metadata.Author
			row.Date = &msg.Metadata.Date
			row.Message = &msg.Metadata.Message
		}
		return m, nil

	case model.RollbackMetadataErrorMsg:
		// Handle metadata loading error
		if m.state.Rollback != nil && msg.RowIndex < len(m.state.Rollback.Rows) {
			row := &m.state.Rollback.Rows[msg.RowIndex]
			row.MetaError = &msg.Error
		}
		return m, nil

	case model.RollbackExecutedMsg:
		// Handle rollback completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Rollback initiated for %s", msg.AppName))

			// Clear rollback state and return to normal mode
			m.state.Rollback = nil
			m.state.Modals.RollbackAppName = nil
			m.state.Mode = model.ModeNormal

			// Start watching tree if requested
			if msg.Watch {
				// Clean up any existing tree watchers before starting new one
				m.cleanupTreeWatchers()
				m.state.Navigation.View = model.ViewTree
				m.state.UI.TreeAppName = &msg.AppName
				var appObj model.App
				found := false
				for _, a := range m.state.Apps {
					if a.Name == msg.AppName {
						appObj = a
						found = true
						break
					}
				}
				if !found {
					appObj = model.App{Name: msg.AppName}
				}
				return m, tea.Batch(m.startLoadingResourceTree(appObj), m.startWatchingResourceTree(appObj), m.consumeTreeEvent())
			}
		} else {
			m.statusService.Error(fmt.Sprintf("Rollback failed for %s", msg.AppName))
		}
		return m, nil

	case model.RollbackNavigationMsg:
		// Handle rollback navigation
		if m.state.Rollback != nil {
			switch msg.Direction {
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
					return m, tea.Batch(cmds...)
				}
			case "top":
				m.state.Rollback.SelectedIdx = 0
			case "bottom":
				m.state.Rollback.SelectedIdx = len(m.state.Rollback.Rows) - 1
			}
		}
		return m, nil

	case model.RollbackToggleOptionMsg:
		// Handle rollback option toggling
		if m.state.Rollback != nil {
			switch msg.Option {
			case "prune":
				m.state.Rollback.Prune = !m.state.Rollback.Prune
			case "watch":
				m.state.Rollback.Watch = !m.state.Rollback.Watch
			case "dryrun":
				m.state.Rollback.DryRun = !m.state.Rollback.DryRun
			}
		}
		return m, nil

	case model.RollbackConfirmMsg:
		// Handle rollback confirmation
		if m.state.Rollback != nil && m.state.Rollback.SelectedIdx < len(m.state.Rollback.Rows) {
			// Switch to confirmation mode
			m.state.Rollback.Mode = "confirm"
		}
		return m, nil

	case model.RollbackCancelMsg:
		// Handle rollback cancellation
		m.state.Rollback = nil
		m.state.Modals.RollbackAppName = nil
		m.state.Mode = model.ModeNormal
		return m, nil

	case model.RollbackShowDiffMsg:
		// Handle rollback diff request
		if m.state.Rollback != nil {
			return m, m.startRollbackDiffSession(m.state.Rollback.AppName, msg.Revision)
		}
		return m, nil

	case model.QuitMsg:
		return m, tea.Quit

	case model.SetInitialLoadingMsg:
		cblog.With("component", "model").Info("SetInitialLoadingMsg received", "loading", msg.Loading)
		// Control the initial loading modal display
		m.state.Modals.InitialLoading = msg.Loading
		// Don't trigger load here - let SetModeMsg handle it to avoid duplicates

		return m, nil

	// Update Messages
	case model.UpdateCheckCompletedMsg:
		if msg.Error != nil {
			cblog.With("component", "update").Error("Update check failed", "err", msg.Error)
			return m, nil
		}
		if msg.UpdateInfo != nil {
			// Check if this is a new update notification (different version or first time)
			isNewNotification := m.state.UI.UpdateInfo == nil ||
				!m.state.UI.UpdateInfo.Available ||
				m.state.UI.UpdateInfo.LatestVersion != msg.UpdateInfo.LatestVersion

			m.state.UI.UpdateInfo = msg.UpdateInfo
			m.state.UI.IsVersionOutdated = msg.UpdateInfo.Available

			if msg.UpdateInfo.Available {
				// Set notification timestamp for new notifications
				if isNewNotification && msg.UpdateInfo.NotificationShownAt == nil {
					now := time.Now()
					msg.UpdateInfo.NotificationShownAt = &now
					m.state.UI.UpdateInfo = msg.UpdateInfo
				}

				m.state.UI.LatestVersion = &msg.UpdateInfo.LatestVersion
				cblog.With("component", "update").Info("Update available",
					"current", msg.UpdateInfo.CurrentVersion,
					"latest", msg.UpdateInfo.LatestVersion,
					"install_method", msg.UpdateInfo.InstallMethod)
			}
		}
		return m, nil

	case model.UpgradeRequestedMsg:
		return m, m.handleUpgradeRequest()

	case model.UpgradeProgressMsg:
		m.statusService.Set(msg.Message)
		return m, nil

	case model.UpgradeCompletedMsg:
		if msg.Success {
			// Show upgrade success modal
			m.state.Mode = model.ModeUpgradeSuccess
			m.state.Modals.UpgradeLoading = false
		} else {
			// Show upgrade error modal with detailed instructions
			errorMsg := msg.Error.Error()
			m.state.Modals.UpgradeError = &errorMsg
			m.state.Mode = model.ModeUpgradeError
			m.state.Modals.UpgradeLoading = false
		}
		return m, nil
	}

	return m, nil
}
