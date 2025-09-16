package main

import (
    "github.com/charmbracelet/bubbles/v2/spinner"
    "github.com/charmbracelet/bubbles/v2/table"
    tea "github.com/charmbracelet/bubbletea/v2"
    "github.com/charmbracelet/lipgloss/v2"
    "github.com/darksworm/argonaut/pkg/autocomplete"
    "github.com/darksworm/argonaut/pkg/model"
    "github.com/darksworm/argonaut/pkg/services"
    "github.com/darksworm/argonaut/pkg/tui/treeview"
)

// NewModel creates a new Model with default state and services
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
        state:             model.NewAppState(),
        argoService:       services.NewArgoApiService(nil),
        navigationService: services.NewNavigationService(),
        statusService: services.NewStatusService(services.StatusServiceConfig{ Handler: createFileStatusHandler(), DebugEnabled: true }),
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

