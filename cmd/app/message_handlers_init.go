package main

import (
	"github.com/charmbracelet/bubbles/v2/spinner"
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// initMessageHandlers registers all message handlers with the message registry.
// This implements the Observer/Event Listener pattern for type-safe message handling.
func (m *Model) initMessageHandlers() {
	registry := m.messageRegistry

	// Terminal/System message handlers
	registry.Register(tea.WindowSizeMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleWindowSize(msg) })

	// Create a dummy KeyMsg for type registration (KeyMsg is an interface)
	keyExample := tea.KeyPressMsg(tea.Key{})
	registry.Register(keyExample, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleKeyMsgRegistry(msg) })

	registry.Register(spinner.TickMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSpinnerTick(msg) })

	// Navigation message handlers
	registry.Register(model.SetViewMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetView(msg) })
	registry.Register(model.SetSelectedIdxMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetSelectedIdx(msg) })
	registry.Register(model.ResetNavigationMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleResetNavigation(msg) })
	registry.Register(model.SetSelectedAppsMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetSelectedApps(msg) })
	registry.Register(model.NavigationUpdateMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleNavigationUpdate(msg) })

	// UI state message handlers
	registry.Register(model.SetSearchQueryMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetSearchQuery(msg) })
	registry.Register(model.SetActiveFilterMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetActiveFilter(msg) })
	registry.Register(model.SetCommandMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetCommand(msg) })
	registry.Register(model.ClearFiltersMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleClearFilters(msg) })
	registry.Register(model.ClearAllSelectionsMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleClearAllSelections(msg) })
	registry.Register(model.SetAPIVersionMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetAPIVersion(msg) })

	// Mode and state message handlers
	registry.Register(model.SetModeMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetMode(msg) })
	registry.Register(model.QuitMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleQuit(msg) })
	registry.Register(model.SetInitialLoadingMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetInitialLoading(msg) })

	// Data loading message handlers
	registry.Register(model.SetAppsMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetApps(msg) })
	registry.Register(model.SetServerMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleSetServer(msg) })
	registry.Register(model.AppsLoadedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleAppsLoaded(msg) })
	registry.Register(model.AppUpdatedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleAppUpdated(msg) })
	registry.Register(model.AppDeletedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleAppDeleted(msg) })
	registry.Register(model.StatusChangeMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleStatusChange(msg) })
	registry.Register(model.ResourceTreeLoadedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleResourceTreeLoaded(msg) })
	registry.Register(watchStartedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleWatchStarted(msg) })
	registry.Register(treeWatchStartedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleTreeWatchStarted(msg) })

	// Rollback message handlers
	registry.Register(model.RollbackHistoryLoadedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackHistoryLoaded(msg) })
	registry.Register(model.RollbackMetadataLoadedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackMetadataLoaded(msg) })
	registry.Register(model.RollbackMetadataErrorMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackMetadataError(msg) })
	registry.Register(model.RollbackExecutedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackExecuted(msg) })
	registry.Register(model.RollbackNavigationMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackNavigation(msg) })
	registry.Register(model.RollbackToggleOptionMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackToggleOption(msg) })
	registry.Register(model.RollbackConfirmMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackConfirm(msg) })
	registry.Register(model.RollbackCancelMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackCancel(msg) })
	registry.Register(model.RollbackShowDiffMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleRollbackShowDiff(msg) })

	// Update and upgrade message handlers
	registry.Register(model.UpdateCheckCompletedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleUpdateCheckCompleted(msg) })
	registry.Register(model.UpgradeRequestedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleUpgradeRequested(msg) })
	registry.Register(model.UpgradeProgressMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleUpgradeProgress(msg) })
	registry.Register(model.UpgradeCompletedMsg{}, func(msg tea.Msg) (tea.Model, tea.Cmd) { return m.handleUpgradeCompleted(msg) })

	cblog.With("component", "registry").Info("Message handlers initialized",
		"handler_count", registry.HandlersCount())
}