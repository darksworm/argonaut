package main

import (
	"context"
	"strings"
	"time"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/autocomplete"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/tui/listnav"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

func NewModel(cfg *config.ArgonautConfig) *Model {
	// Use default config if none provided
	if cfg == nil {
		cfg = config.GetDefaultConfig()
	}

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(magentaBright)

	// Initialize tables using helpers
	appsTable := newAppsTable()
	clustersTable := newSimpleTable()
	namespacesTable := newSimpleTable()
	projectsTable := newSimpleTable()

	// Initialize update service
	updateService := services.NewUpdateService(services.UpdateServiceConfig{
		HTTPClient:       nil, // Use default HTTP client
		GitHubRepo:       "darksworm/argonaut",
		CheckIntervalMin: 60, // Check every hour
	})

	return &Model{
		state:              model.NewAppState(),
		argoService:        services.NewArgoApiService(nil),
		navigationService:  services.NewNavigationService(),
		statusService:      services.NewStatusService(services.StatusServiceConfig{Handler: createFileStatusHandler(), DebugEnabled: true}),
		updateService:      updateService,
		config:             cfg,
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
		listNav:            listnav.New(),
		treeNav:            listnav.New(),
		themeNav:           listnav.New(),
		rollbackNav:        listnav.New(),
	}
}

// preserve imports used by other files in this package
var _ table.Model
var _ tea.Msg

// Init implements tea.Model.Init
func (m *Model) Init() tea.Cmd {
    // Initialize with terminal size request and startup commands
    var cmds []tea.Cmd
    cmds = append(cmds, m.spinner.Tick)

	// Apply theme to model components
	m.applyThemeToModel()

	// Show initial loading modal immediately if server is configured
	if m.state.Server != nil {
		cmds = append(cmds, func() tea.Msg { return model.SetInitialLoadingMsg{Loading: true} })
	}

	cmds = append(cmds,
		// Validate authentication if server is configured
		m.validateAuthentication(),
		// Start periodic update check (delayed)
		m.scheduleInitialUpdateCheck(),
	)

	_ = context.TODO() // keep import stable if unused on some builds
	_ = time.Second
	return tea.Batch(cmds...)
}

func (m *Model) validateAuthentication() tea.Cmd {
	return func() tea.Msg {
		if m.state.Server == nil {
			// Check if we're already in core detected mode (set during config loading)
			if m.state.Mode == model.ModeCoreDetected {
				return model.SetModeMsg{Mode: model.ModeCoreDetected}
			}
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
