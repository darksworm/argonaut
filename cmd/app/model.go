package main

import (
	"fmt"

	"github.com/charmbracelet/bubbles/v2/spinner"
	"github.com/charmbracelet/bubbles/v2/table"
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/autocomplete"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
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

	// Message handler registry for type-safe message handling
	messageRegistry *MessageRegistry

	// Command handler registry for type-safe command handling
	commandRegistry *CommandRegistry
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Use message registry for type-safe, extensible message handling
	if handler, exists := m.messageRegistry.GetHandler(msg); exists {
		return handler(msg)
	}

	// Log unhandled message types for debugging
	cblog.With("component", "model").Warn("Unhandled message type",
		"type", fmt.Sprintf("%T", msg))
	return m, nil

}
