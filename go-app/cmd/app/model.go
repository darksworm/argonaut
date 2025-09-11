package main

import (
	"fmt"
	"time"
	
	tea "github.com/charmbracelet/bubbletea"
	"github.com/a9s/go-app/pkg/model"
	"github.com/a9s/go-app/pkg/services"
)

// Model represents the main Bubbletea model containing all application state
type Model struct {
	// Core application state
	state *model.AppState

	// Services
	argoService       services.ArgoApiService
	navigationService services.NavigationService
	statusService     services.StatusService

	// Interactive input components using bubbles
	inputComponents *InputComponentState

	// Internal flags
	ready bool
    err   error

    // Watch channel for Argo events
    watchChan chan services.ArgoApiEvent
}

// NewModel creates a new Model with default state and services
func NewModel() *Model {
	return &Model{
		state: model.NewAppState(),
		argoService: services.NewArgoApiService(nil), // Will be configured when server is available
		navigationService: services.NewNavigationService(),
		statusService: services.NewStatusService(services.StatusServiceConfig{
			Handler:      createFileStatusHandler(), // Log to file instead of stdout
			DebugEnabled: true,
		}),
		inputComponents: NewInputComponents(),
		ready: false,
		err:   nil,
	}
}

// Init implements tea.Model.Init
func (m Model) Init() tea.Cmd {
	// Initialize with terminal size request and startup commands
	return tea.Batch(
		tea.EnterAltScreen,
		func() tea.Msg {
			return model.StatusChangeMsg{Status: "Initializing..."}
		},
		// Start loading applications if server is configured
		func() tea.Msg {
			if m.state.Server != nil {
				fmt.Printf("[INIT] Server configured: %s, triggering loading mode\n", m.state.Server.BaseURL)
				return model.SetModeMsg{Mode: model.ModeLoading}
			}
			fmt.Printf("[INIT] No server configured\n")
			return model.StatusChangeMsg{Status: "No server configured"}
		},
	)
}

// Update implements tea.Model.Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	// Terminal/System messages
	case tea.WindowSizeMsg:
		m.state.Terminal.Rows = msg.Height
		m.state.Terminal.Cols = msg.Width
		if !m.ready {
			m.ready = true
			return m, func() tea.Msg {
				return model.StatusChangeMsg{Status: "Ready"}
			}
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

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
		fmt.Printf("[MODE] Switching from %s to %s\n", oldMode, msg.Mode)
		
		// Handle mode transitions
		if msg.Mode == model.ModeLoading && oldMode != model.ModeLoading {
			// Start loading applications from API
			fmt.Printf("[MODE] Triggering API load for loading mode\n")
			return m, m.startLoadingApplications()
		}
		
		return m, nil

	// Data messages
	case model.SetAppsMsg:
		m.state.Apps = msg.Apps
		// m.ui.UpdateListItems(m.state)
		return m, nil

	case model.SetServerMsg:
		m.state.Server = msg.Server
		// Also fetch API version and start watching
		return m, tea.Batch(m.startWatchingApplications(), m.fetchAPIVersion())

	// API Event messages
	case model.AppsLoadedMsg:
		m.state.Apps = msg.Apps
		// m.ui.UpdateListItems(m.state)
		return m, tea.Batch(func() tea.Msg { return model.SetModeMsg{Mode: model.ModeNormal} }, m.consumeWatchEvent())

	case model.AppUpdatedMsg:
		// upsert app
		updated := msg.App
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
		return m, m.consumeWatchEvent()

	case model.AppDeletedMsg:
		name := msg.AppName
		filtered := m.state.Apps[:0]
		for _, a := range m.state.Apps {
			if a.Name != name { filtered = append(filtered, a) }
		}
		m.state.Apps = filtered
		return m, m.consumeWatchEvent()

	case model.StatusChangeMsg:
		// Now safe to log since we're using file logging
		m.statusService.Set(msg.Status)
		return m, m.consumeWatchEvent()

	case model.ApiErrorMsg:
		// Log error to file and store in model for display
		m.statusService.Error(msg.Message)
		m.err = fmt.Errorf(msg.Message)
		return m, func() tea.Msg {
			return model.SetModeMsg{Mode: model.ModeError}
		}

	case model.AuthErrorMsg:
		// Log error to file and store in model for display
		m.statusService.Error(msg.Error.Error())
		m.err = msg.Error
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

	case model.QuitMsg:
		return m, tea.Quit
	}

	return m, nil
}

// handleKeyMsg handles keyboard input with 1:1 mapping to TypeScript functionality
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Handle mode-specific input first
	switch m.state.Mode {
	case model.ModeSearch:
		return m.handleSearchModeKeys(msg)
	case model.ModeCommand:
		return m.handleCommandModeKeys(msg)
	case model.ModeHelp:
		return m.handleHelpModeKeys(msg)
	case model.ModeConfirmSync:
		return m.handleConfirmSyncKeys(msg)
	case model.ModeRollback:
		return m.handleRollbackModeKeys(msg)
	case model.ModeDiff:
		return m.handleDiffModeKeys(msg)
	}

	// Global key handling for normal mode
	switch msg.String() {
	case "q", "ctrl+c":
		return m, func() tea.Msg { return model.QuitMsg{} }

	// Navigation keys (j/k, up/down)
	case "up", "k":
		return m.handleNavigationUp()
	case "down", "j":
		return m.handleNavigationDown()

	// Selection and interaction
	case " ": // Space for selection
		return m.handleToggleSelection()
	case "enter":
		return m.handleDrillDown()

	// Mode switching keys
	case "/":
		return m.handleEnterSearchMode()
	case ":":
		return m.handleEnterCommandMode()
	case "?":
		return m.handleShowHelp()

	// Quick actions
	case "s":
		if m.state.Navigation.View == model.ViewApps {
			return m.handleSyncModal()
		}
	case "r":
		return m.handleRefresh()

	// Clear/escape functionality
	case "esc":
		return m.handleEscape()

	// Quick navigation (matching TypeScript app)
	case "g":
		if m.state.Navigation.LastGPressed > 0 && 
		   time.Since(time.Unix(m.state.Navigation.LastGPressed, 0)) < 500*time.Millisecond {
			// Double-g: go to top
			return m.handleGoToTop()
		} else {
			// Single g: record timestamp
			m.state.Navigation.LastGPressed = time.Now().Unix()
			return m, nil
		}
	case "G":
		return m.handleGoToBottom()
	}

	return m, nil
}
