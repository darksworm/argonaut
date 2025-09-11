package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/a9s/go-app/pkg/api"
	"github.com/a9s/go-app/pkg/model"
	"github.com/a9s/go-app/pkg/services"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	// bubbles spinner for loading
	spinner spinner.Model
	
	// bubbles table for resources view
	resourcesTable table.Model
}

// NewModel creates a new Model with default state and services
func NewModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	
	// Initialize resources table
	columns := []table.Column{
		{Title: "KIND", Width: 20},
		{Title: "NAME", Width: 40},
		{Title: "STATUS", Width: 15},
	}
	
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(false),
		table.WithHeight(10),
	)
	
	// Style the table to match the app's theme
	tableStyle := table.DefaultStyles()
	tableStyle.Header = tableStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	tableStyle.Selected = tableStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(tableStyle)
	
	return &Model{
		state:             model.NewAppState(),
		argoService:       services.NewArgoApiService(nil), // Will be configured when server is available
		navigationService: services.NewNavigationService(),
		statusService: services.NewStatusService(services.StatusServiceConfig{
			Handler:      createFileStatusHandler(), // Log to file instead of stdout
			DebugEnabled: true,
		}),
		inputComponents: NewInputComponents(),
		ready:           false,
		err:             nil,
		spinner:          s,
		resourcesTable:   t,
	}
}

// Init implements tea.Model.Init
func (m Model) Init() tea.Cmd {
	// Initialize with terminal size request and startup commands
	return tea.Batch(
		tea.EnterAltScreen,
		m.spinner.Tick,
		func() tea.Msg {
			return model.StatusChangeMsg{Status: "Initializing..."}
		},
		// Start loading applications if server is configured
		func() tea.Msg {
			if m.state.Server != nil {
				// [INIT] Server configured - removed printf to avoid TUI interference
				return model.SetModeMsg{Mode: model.ModeLoading}
			}
			// [INIT] No server configured - removed printf to avoid TUI interference
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
	
	// Spinner messages
	case spinner.TickMsg:
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
		// [MODE] Switching from %s to %s - removed printf to avoid TUI interference

		// Handle mode transitions
		if msg.Mode == model.ModeLoading && oldMode != model.ModeLoading {
			// Start loading applications from API
			// [MODE] Triggering API load for loading mode - removed printf to avoid TUI interference
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
			if a.Name != name {
				filtered = append(filtered, a)
			}
		}
		m.state.Apps = filtered
		return m, m.consumeWatchEvent()

	case model.StatusChangeMsg:
		// Now safe to log since we're using file logging
		m.statusService.Set(msg.Status)
		return m, m.consumeWatchEvent()

	case ResourcesLoadedMsg:
		log.Printf("Received ResourcesLoadedMsg for app: %s", msg.AppName)
		if m.state.Resources != nil && m.state.Resources.AppName == msg.AppName {
			if msg.Error != "" {
				log.Printf("ERROR: Resource loading failed for %s: %s", msg.AppName, msg.Error)
				m.state.Resources.Loading = false
				m.state.Resources.Error = msg.Error
			} else {
				log.Printf("SUCCESS: Loaded %d resources for app %s", len(msg.Resources), msg.AppName)
				m.state.Resources.Loading = false
				m.state.Resources.Resources = msg.Resources
				m.state.Resources.Error = ""
			}
		}
		return m, nil

	// Old spinner TickMsg removed - now using bubbles spinner

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

	case model.SyncCompletedMsg:
		// Handle single app sync completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Sync initiated for %s", msg.AppName))
			
			// Show resource stream if watch is enabled (matching TypeScript behavior)
			if m.state.Modals.ConfirmSyncWatch {
				m.state.Modals.SyncViewApp = &msg.AppName
				m.state.Mode = model.ModeResources
				
				// Initialize resource state and start loading
				m.state.Resources = &model.ResourceState{
					AppName:   msg.AppName,
					Resources: nil,
					Loading:   true,
					Error:     "",
					Offset:    0,
				}
				
				return m, m.loadResourcesForApp(msg.AppName)
			}
		} else {
			m.statusService.Set("Sync cancelled")
		}
		return m, nil

	case model.MultiSyncCompletedMsg:
		// Handle multiple app sync completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Sync initiated for %d app(s)", msg.AppCount))
			// Clear selections after multi-sync (matching TypeScript behavior)
			m.state.Selections.SelectedApps = model.NewStringSet()
		}
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

// loadResourcesForApp creates a command to load resources for the given app
func (m Model) loadResourcesForApp(appName string) tea.Cmd {
	log.Printf("Loading resources for app: %s", appName)

	return func() tea.Msg {
		if m.state.Server == nil {
			log.Printf("ERROR: server not configured when loading resources for %s", appName)
			return ResourcesLoadedMsg{
				AppName: appName,
				Error:   "Server not configured",
			}
		}

		log.Printf("Creating ApplicationService for server: %s", m.state.Server.BaseURL)
		appService := api.NewApplicationService(m.state.Server)

		log.Printf("Calling GetResourceTree API for app: %s", appName)
		tree, err := appService.GetResourceTree(context.Background(), appName, "")
		if err != nil {
			log.Printf("ERROR: Failed to load resources for app %s: %v", appName, err)
			return ResourcesLoadedMsg{
				AppName: appName,
				Error:   err.Error(),
			}
		}

		log.Printf("Successfully loaded %d resources for app %s", len(tree.Nodes), appName)

		// Convert api.ResourceNode to model.ResourceNode
		modelResources := make([]model.ResourceNode, len(tree.Nodes))
		for i, node := range tree.Nodes {
			modelResources[i] = convertApiToModelResourceNode(node)
		}

		return ResourcesLoadedMsg{
			AppName:   appName,
			Resources: modelResources,
		}
	}
}

// ResourcesLoadedMsg represents the result of loading resources
type ResourcesLoadedMsg struct {
	AppName   string
	Resources []model.ResourceNode
	Error     string
}

// convertApiToModelResourceNode converts api.ResourceNode to model.ResourceNode
func convertApiToModelResourceNode(apiNode api.ResourceNode) model.ResourceNode {
	var health *model.ResourceHealth
	if apiNode.Health != nil {
		health = &model.ResourceHealth{
			Status:  apiNode.Health.Status,
			Message: apiNode.Health.Message,
		}
	}

	var networkingInfo *model.NetworkingInfo
	if apiNode.NetworkingInfo != nil {
		targetRefs := make([]model.ResourceRef, len(apiNode.NetworkingInfo.TargetRefs))
		for i, ref := range apiNode.NetworkingInfo.TargetRefs {
			targetRefs[i] = model.ResourceRef{
				Group:     ref.Group,
				Kind:      ref.Kind,
				Name:      ref.Name,
				Namespace: ref.Namespace,
			}
		}
		networkingInfo = &model.NetworkingInfo{
			TargetLabels: apiNode.NetworkingInfo.TargetLabels,
			TargetRefs:   targetRefs,
		}
	}

	return model.ResourceNode{
		Group:          apiNode.Group,
		Kind:           apiNode.Kind,
		Name:           apiNode.Name,
		Namespace:      apiNode.Namespace,
		Version:        apiNode.Version,
		Health:         health,
		NetworkingInfo: networkingInfo,
		Images:         apiNode.Images,
	}
}

// Duplicate sync functions removed - using existing ones from api_integration.go
