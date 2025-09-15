package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/autocomplete"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/tui"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
	"github.com/noborus/ov/oviewer"
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
	resourcesTable  table.Model
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
}

// NewModel creates a new Model with default state and services
func NewModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Create a function to get standard table styles
	getTableStyle := func() table.Styles {
		s := table.DefaultStyles()
		s.Header = s.Header.
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
		return s
	}

	// Initialize resources table
	resourcesColumns := []table.Column{
		{Title: "KIND", Width: 20},
		{Title: "NAME", Width: 40},
		{Title: "STATUS", Width: 15},
	}
	resourcesTable := table.New(
		table.WithColumns(resourcesColumns),
		table.WithFocused(false),
		table.WithHeight(10),
	)
	resourcesTable.SetStyles(getTableStyle())

	// Initialize apps table
	appsColumns := []table.Column{
		{Title: "NAME", Width: 40},
		{Title: "SYNC", Width: 12},
		{Title: "HEALTH", Width: 15},
	}
	appsTable := table.New(
		table.WithColumns(appsColumns),
		table.WithFocused(true), // Apps table should be focused for navigation
		table.WithHeight(10),
	)
	appsTable.SetStyles(getTableStyle())

	// Initialize simple tables for other views
	simpleColumns := []table.Column{
		{Title: "NAME", Width: 60},
	}

	clustersTable := table.New(
		table.WithColumns(simpleColumns),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	clustersTable.SetStyles(getTableStyle())

	namespacesTable := table.New(
		table.WithColumns(simpleColumns),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	namespacesTable.SetStyles(getTableStyle())

	projectsTable := table.New(
		table.WithColumns(simpleColumns),
		table.WithFocused(true),
		table.WithHeight(10),
	)
	projectsTable.SetStyles(getTableStyle())

	return &Model{
		state:             model.NewAppState(),
		argoService:       services.NewArgoApiService(nil), // Will be configured when server is available
		navigationService: services.NewNavigationService(),
		statusService: services.NewStatusService(services.StatusServiceConfig{
			Handler:      createFileStatusHandler(), // Log to file instead of stdout
			DebugEnabled: true,
		}),
		inputComponents:    NewInputComponents(),
		autocompleteEngine: autocomplete.NewAutocompleteEngine(),
		ready:              false,
		err:             nil,
		spinner:         s,
		resourcesTable:  resourcesTable,
		appsTable:       appsTable,
		clustersTable:   clustersTable,
		namespacesTable: namespacesTable,
		projectsTable:   projectsTable,
		program:         nil,
        inPager:         false,
        treeView:        treeview.NewTreeView(0, 0),
        treeStream:      make(chan model.ResourceTreeStreamMsg, 64),
    }
}

// SetProgram stores the Bubble Tea program pointer for terminal hand-off
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// pagerDoneMsg signals that an external pager has closed
type pagerDoneMsg struct{ Err error }
type pauseRenderingMsg struct{}
type resumeRenderingMsg struct{}

// Init implements tea.Model.Init
func (m Model) Init() tea.Cmd {
	// Initialize with terminal size request and startup commands
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen, m.spinner.Tick)

    // Show initial loading modal immediately if server is configured
	if m.state.Server != nil {
		cmds = append(cmds, func() tea.Msg {
			return model.SetInitialLoadingMsg{Loading: true}
		})
	}

	cmds = append(cmds,
		func() tea.Msg {
			return model.StatusChangeMsg{Status: "Initializing..."}
		},
		// Validate authentication if server is configured
		m.validateAuthentication(),
	)

	return tea.Batch(cmds...)
}

// watchTreeDeliver is used by the watcher goroutine to send messages into Bubble Tea
func (m Model) watchTreeDeliver(msg model.ResourceTreeStreamMsg) {
    // Non-blocking send; if full, drop to avoid blocking
    select {
    case m.treeStream <- msg:
    default:
    }
}

// consumeTreeEvent reads a single tree stream event and returns it as a tea message
func (m Model) consumeTreeEvent() tea.Cmd {
    return func() tea.Msg {
        if m.treeStream == nil {
            return nil
        }
        ev, ok := <-m.treeStream
        if !ok {
            return nil
        }
        return ev
    }
}

// validateAuthentication checks if authentication is valid (matches TypeScript app-orchestrator.ts)
func (m Model) validateAuthentication() tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			log.Printf("No server configured - showing auth required")
			return model.SetModeMsg{Mode: model.ModeAuthRequired}
		}

		// Create API service to validate authentication
		appService := api.NewApplicationService(m.state.Server)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Validate user info (similar to TypeScript getUserInfo call)
		if err := appService.GetUserInfo(ctx); err != nil {
			log.Printf("Authentication validation failed: %v", err)

			// Check if this is a connection error rather than authentication error
			errStr := err.Error()
			if strings.Contains(errStr, "connection refused") ||
				strings.Contains(errStr, "no such host") ||
				strings.Contains(errStr, "network is unreachable") ||
				strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "dial tcp") {
				return model.SetModeMsg{Mode: model.ModeConnectionError}
			}

			// Otherwise, it's likely an authentication issue
			return model.SetModeMsg{Mode: model.ModeAuthRequired}
		}

		log.Printf("Authentication validated successfully")
		return model.SetModeMsg{Mode: model.ModeLoading}
	}
}

// openTextPager releases the terminal and runs an oviewer pager with the given text
func (m Model) openTextPager(title, text string) tea.Cmd {
    return func() tea.Msg {
        if m.program != nil {
            m.program.Send(pauseRenderingMsg{})
            _ = m.program.ReleaseTerminal()
        }
		defer func() {
			// Clear screen and restore terminal to Bubble Tea
			fmt.Print("\x1b[2J\x1b[H")
			time.Sleep(150 * time.Millisecond)
			if m.program != nil {
				_ = m.program.RestoreTerminal()
				m.program.Send(resumeRenderingMsg{})
			}
		}()

		// Prepare pager root
		r := strings.NewReader(text)
		root, err := oviewer.NewRoot(r)
		if err != nil {
			log.Printf("ERROR: Failed to create oviewer root: %v", err)
			return pagerDoneMsg{Err: err}
		}

		cfg := oviewer.NewConfig()
		cfg.IsWriteOnExit = false
		cfg.IsWriteOriginal = false

		// Ensure ov starts in normal view mode, not input/command mode
		cfg.ViewMode = "" // Use default normal view mode

		// Try to configure keybindings and catch any errors
		configErr := configureVimKeyBindings(&cfg)
		if configErr != nil {
			log.Printf("ERROR: Failed to configure vim keybindings: %v", configErr)
			return pagerDoneMsg{Err: configErr}
		}

		root.SetConfig(cfg)
		// Don't set FileName as it might trigger input mode
		// root.Doc.FileName = title

		// Capture any error from Run()
		runErr := root.Run()
		if runErr != nil {
			log.Printf("ERROR: Failed to run oviewer: %v", runErr)
			return pagerDoneMsg{Err: runErr}
		}

		return pagerDoneMsg{Err: nil}
	}
}

// openExternalDiffPager runs an external diff viewer/pager. It supports two modes:
//  1. Command string with placeholders {left} and {right} for file paths (e.g. "vimdiff {left} {right}")
//  2. Pager that reads unified diff from stdin (e.g. "delta --side-by-side"). In that case we pipe
//     the diff text to the process.
//
// openInteractiveDiffViewer replaces the terminal with an interactive diff tool
// configured via ARGONAUT_DIFF_VIEWER. The command may include {left} and {right}
// placeholders for file paths.
func (m Model) openInteractiveDiffViewer(leftFile, rightFile, cmdStr string) tea.Msg {
	if m.program != nil {
		m.program.Send(pauseRenderingMsg{})
		_ = m.program.ReleaseTerminal()
	}
	defer func() {
		fmt.Print("\x1b[2J\x1b[H")
		time.Sleep(150 * time.Millisecond)
		if m.program != nil {
			_ = m.program.RestoreTerminal()
			m.program.Send(resumeRenderingMsg{})
		}
	}()

	cmdStr = strings.ReplaceAll(cmdStr, "{left}", shellEscape(leftFile))
	cmdStr = strings.ReplaceAll(cmdStr, "{right}", shellEscape(rightFile))
	c := exec.Command("sh", "-lc", cmdStr)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		log.Printf("interactive diff viewer failed: %v", err)
		return pagerDoneMsg{Err: err}
	}
	return pagerDoneMsg{Err: nil}
}

// runDiffFormatter runs a non-interactive diff formatter on diffText and returns its output.
// Priority: ARGONAUT_DIFF_FORMATTER if set; else delta (if present); else return input.
func (m Model) runDiffFormatter(diffText string) (string, error) {
	cmdStr := os.Getenv("ARGONAUT_DIFF_FORMATTER")
	cols := 0
	if m.state != nil {
		cols = m.state.Terminal.Cols
	}
	if cmdStr == "" && inPath("delta") {
		// Ensure delta does not page; we want raw output to feed our pager.
		// Also force width so piping to OV uses full terminal width.
		if cols > 0 {
			cmdStr = fmt.Sprintf("delta --side-by-side --line-numbers --navigate --paging=never --width=%d", cols)
		} else {
			cmdStr = "delta --side-by-side --line-numbers --navigate --paging=never"
		}
	}
	if cmdStr == "" {
		return diffText, nil
	}
	c := exec.Command("sh", "-lc", cmdStr)
	// Help tools detect width when stdout is a pipe
	if cols > 0 {
		c.Env = append(os.Environ(), fmt.Sprintf("COLUMNS=%d", cols))
	} else {
		c.Env = os.Environ()
	}
	c.Stdin = strings.NewReader(diffText)
	out, err := c.CombinedOutput()
	if err != nil {
		return diffText, err
	}
	return string(out), nil
}

func inPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func shellEscape(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

// configureVimKeyBindings adds vim-like key bindings to the oviewer config
func configureVimKeyBindings(cfg *oviewer.Config) error {
	// Disable OV's default key bindings so our map is authoritative.
	cfg.DefaultKeyBind = "disable"
	// Hard reset keybindings to avoid duplicates from library defaults.
	// We'll assign a minimal, deterministic set.
	cfg.Keybind = make(map[string][]string)

	// Helper to set a binding list without duplicates
	set := func(action string, keys ...string) {
		uniq := make(map[string]struct{})
		out := make([]string, 0, len(keys))
		for _, k := range keys {
			if _, ok := uniq[k]; ok {
				continue
			}
			uniq[k] = struct{}{}
			out = append(out, k)
		}
		cfg.Keybind[action] = out
	}

	// Vim-like navigation
	set("left", "h")
	set("right", "l")
	set("up", "k")
	set("down", "j")
	set("top", "g")
	set("bottom", "G")
	set("search", "/")
	set("exit", "q")

	// Additional quality-of-life bindings that don't conflict
	// Page navigation (space/down: page down, b: page up) if supported
	// These actions might be ignored if oviewer doesn't map them; harmless otherwise.
	set("page_down", " ")
	set("page_up", "b")

	return nil
}

// Update implements tea.Model.Update
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

    // Tree stream messages from watcher goroutine
    case model.ResourceTreeStreamMsg:
        if m.state.Navigation.View == model.ViewTree && m.treeView != nil && len(msg.TreeJSON) > 0 {
            var tree api.ResourceTree
            if err := json.Unmarshal(msg.TreeJSON, &tree); err == nil {
                m.treeView.SetData(&tree)
            }
        }
        return m, m.consumeTreeEvent()

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
		// [MODE] Switching from %s to %s - removed printf to avoid TUI interference

		// Handle mode transitions
		if msg.Mode == model.ModeLoading && oldMode != model.ModeLoading {
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
		// m.ui.UpdateListItems(m.state)
		return m, nil

	case model.SetServerMsg:
		m.state.Server = msg.Server
		// Also fetch API version and start watching
		return m, tea.Batch(m.startWatchingApplications(), m.fetchAPIVersion())

	// API Event messages
	case model.AppsLoadedMsg:
		m.state.Apps = msg.Apps
		// Turn off initial loading modal if it was active
		m.state.Modals.InitialLoading = false
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

		// Clear diff loading state for diff-related status messages
		if (msg.Status == "No diffs" || msg.Status == "No differences") && m.state.Diff != nil {
			m.state.Diff.Loading = false
		}

		return m, m.consumeWatchEvent()

    case model.ResourceTreeLoadedMsg:
		// Populate tree view with loaded data
		if m.treeView != nil && len(msg.TreeJSON) > 0 {
			var tree api.ResourceTree
			if err := json.Unmarshal(msg.TreeJSON, &tree); err == nil {
				m.treeView.SetAppMeta(msg.AppName, msg.Health, msg.Sync)
				m.treeView.SetData(&tree)
			}
			// Reset cursor for tree view
			m.state.Navigation.SelectedIdx = 0
			m.statusService.Set("Tree loaded")
		}
		return m, nil

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

	case model.StructuredErrorMsg:
		// Handle structured errors with proper error state management
		if msg.Error != nil {
			errorMsg := fmt.Sprintf("Error: %s", msg.Error.Message)
			if msg.Error.UserAction != "" {
				errorMsg += fmt.Sprintf(" - %s", msg.Error.UserAction)
			}
			m.statusService.Error(errorMsg)

			// Debug: Log structured error details
			log.Printf("StructuredErrorMsg received: Category=%s, Code=%s, Message=%s",
				msg.Error.Category, msg.Error.Code, msg.Error.Message)

			// Update error state in TUI
			if tuiHandler := tui.GetDefaultTUIHandler(); tuiHandler != nil {
				tuiMessages := tuiHandler.HandleError(msg.Error)
				// Process TUI messages if needed
				_ = tuiMessages
			}
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
        // Log error to file and store structured error in state for display
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
			log.Printf("PAGER ERROR: %v", msg.Err)
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
		// Log error to file and store in model for display
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
				m.state.Navigation.View = model.ViewTree
				m.state.UI.TreeAppName = &msg.AppName
				// find app
				var appObj model.App
				found := false
				for _, a := range m.state.Apps {
					if a.Name == msg.AppName { appObj = a; found = true; break }
				}
				if !found { appObj = model.App{Name: msg.AppName} }
                return m, tea.Batch(m.startLoadingResourceTree(appObj), m.startWatchingResourceTree(appObj), m.consumeTreeEvent())
			}
		} else {
			m.statusService.Set("Sync cancelled")
		}
		// Close confirm modal/loading state if open
		m.state.Modals.ConfirmTarget = nil
		m.state.Modals.ConfirmSyncLoading = false
		if m.state.Mode == model.ModeConfirmSync && !m.state.Modals.ConfirmSyncWatch {
			m.state.Mode = model.ModeNormal
		}
		return m, nil

	case model.MultiSyncCompletedMsg:
		// Handle multiple app sync completion
		if msg.Success {
			m.statusService.Set(fmt.Sprintf("Sync initiated for %d app(s)", msg.AppCount))
			// Clear selections after multi-sync (matching TypeScript behavior)
			m.state.Selections.SelectedApps = model.NewStringSet()
		}
		// Close confirm modal/loading state if open
		m.state.Modals.ConfirmTarget = nil
		m.state.Modals.ConfirmSyncLoading = false
		if m.state.Mode == model.ModeConfirmSync {
			m.state.Mode = model.ModeNormal
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
				m.state.Navigation.View = model.ViewTree
				m.state.UI.TreeAppName = &msg.AppName
				var appObj model.App
				found := false
				for _, a := range m.state.Apps { if a.Name == msg.AppName { appObj = a; found = true; break } }
				if !found { appObj = model.App{Name: msg.AppName} }
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
		// Control the initial loading modal display
		m.state.Modals.InitialLoading = msg.Loading

		// If turning on initial loading, also trigger the API load
		if msg.Loading && m.state.Server != nil {
			return m, m.startLoadingApplications()
		}

		return m, nil
	}

	return m, nil
}

// handleKeyMsg handles keyboard input with 1:1 mapping to TypeScript functionality
func (m Model) handleKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
    // Global kill: always quit on Ctrl+C
    if msg.String() == "ctrl+c" {
        return m, func() tea.Msg { return model.QuitMsg{} }
    }
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
	case model.ModeResources:
		return m.handleResourcesModeKeys(msg)
	case model.ModeLogs:
		return m.handleLogsModeKeys(msg)
	case model.ModeAuthRequired:
		return m.handleAuthRequiredModeKeys(msg)
	case model.ModeError:
		return m.handleErrorModeKeys(msg)
	case model.ModeConnectionError:
		return m.handleConnectionErrorModeKeys(msg)
    }

    // Tree view key handling (operates in normal mode but separate view)
    if m.state.Navigation.View == model.ViewTree {
        switch msg.String() {
        case "q", "esc":
            // Return to apps view, preserve previous cursor
            m.state.Navigation.View = model.ViewApps
            // Stop any ongoing tree watch by ignoring future messages (goroutine continues harmlessly)
            // Validate bounds in case list changed
            visibleItems := m.getVisibleItemsForCurrentView()
            m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
                m.state.Navigation.SelectedIdx,
                len(visibleItems),
            )
            return m, nil
        default:
            if m.treeView != nil {
                // Forward to tree view
                _, cmd := m.treeView.Update(msg)
                return m, cmd
            }
            return m, nil
        }
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
	case "R":
		log.Printf("R key pressed, current view: %v", m.state.Navigation.View)
		if m.state.Navigation.View == model.ViewApps {
			log.Printf("Calling handleRollback()")
			return m.handleRollback()
		} else {
			log.Printf("Rollback not available in view: %v", m.state.Navigation.View)
		}

	// Clear/escape functionality
	case "esc":
		return m.handleEscape()

	// Quick navigation (matching TypeScript app)
	case "g":
		// Double-g check for go to top
		now := time.Now().UnixMilli()
		if m.state.Navigation.LastGPressed > 0 && now-m.state.Navigation.LastGPressed < 500 {
			// Double-g: go to top
			return m.handleGoToTop()
		} else {
			// Single g: record timestamp
			m.state.Navigation.LastGPressed = now
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
	}
}

// Duplicate sync functions removed - using existing ones from api_integration.go

// Helper functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
