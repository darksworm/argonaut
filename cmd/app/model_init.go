package main

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/autocomplete"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

func NewModel() *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	// Initialize tables using helpers
	appsTable := newAppsTable()
	clustersTable := newSimpleTable()
	namespacesTable := newSimpleTable()
	projectsTable := newSimpleTable()

	return &Model{
		state:              model.NewAppState(),
		argoService:        services.NewArgoApiService(nil),
		navigationService:  services.NewNavigationService(),
		statusService:      services.NewStatusService(services.StatusServiceConfig{Handler: createFileStatusHandler(), DebugEnabled: true}),
		inputComponents:    NewInputComponents(),
		autocompleteEngine: autocomplete.NewAutocompleteEngine(),
		ready:              false,
		err:                nil,
		spinner:            s,
		appsTable:          appsTable,
		clustersTable:      clustersTable,
		namespacesTable:    namespacesTable,
		projectsTable:      projectsTable,
		program:            nil,
		inPager:            false,
		treeView:           treeview.NewTreeView(0, 0),
		treeStream:         make(chan model.ResourceTreeStreamMsg, 64),
	}
}

// preserve imports used by other files in this package
var _ table.Model
var _ tea.Msg

// Init implements tea.Model.Init
func (m Model) Init() tea.Cmd {
	// Initialize with terminal size request and startup commands
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen, m.spinner.Tick)

	// Show initial loading modal immediately if server is configured
	if m.state.Server != nil {
		cmds = append(cmds, func() tea.Msg { return model.SetInitialLoadingMsg{Loading: true} })
	}

	cmds = append(cmds,
		func() tea.Msg { return model.StatusChangeMsg{Status: "Initializing..."} },
		// Validate authentication if server is configured
		m.validateAuthentication(),
	)

	_ = context.TODO() // keep import stable if unused on some builds
	_ = time.Second
	return tea.Batch(cmds...)
}

func (m Model) validateAuthentication() tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			cblog.With("component", "auth").Info("No server configured - showing auth required")
			return model.SetModeMsg{Mode: model.ModeAuthRequired}
		}

		// Create API service to validate authentication
		appService := api.NewApplicationService(m.state.Server)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Validate user info (similar to TypeScript getUserInfo call)
		if err := appService.GetUserInfo(ctx); err != nil {
			cblog.With("component", "auth").Error("Authentication validation failed", "err", err)

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

		cblog.With("component", "auth").Info("Authentication validated successfully")
		return model.SetModeMsg{Mode: model.ModeLoading}
	}
}
