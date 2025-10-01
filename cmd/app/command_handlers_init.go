package main

import (
	cblog "github.com/charmbracelet/log"
)

// initCommandHandlers registers all command handlers with the command registry.
// This implements the Observer/Event Listener pattern for type-safe command handling.
func (m *Model) initCommandHandlers() {
	registry := m.commandRegistry

	// Key handlers (input control)
	registry.RegisterKey("ctrl+c", m.handleCtrlCKey)
	registry.RegisterKey("esc", m.handleEscapeKey)
	registry.RegisterKey("tab", m.handleTabKey)
	registry.RegisterKey("enter", m.handleEnterKey)

	// Action commands
	registry.RegisterCommand("logs", m.handleLogsCommand)
	registry.RegisterCommand("sync", m.handleSyncCommand)
	registry.RegisterCommand("rollback", m.handleRollbackCommand)
	registry.RegisterCommand("resources", m.handleResourcesCommand)
	registry.RegisterCommand("res", m.handleResourcesCommand)
	registry.RegisterCommand("r", m.handleResourcesCommand)

	// View commands
	registry.RegisterCommand("all", m.handleAllCommand)
	registry.RegisterCommand("up", m.handleUpCommand)
	registry.RegisterCommand("diff", m.handleDiffCommand)
	registry.RegisterCommand("cluster", m.handleClusterCommand)
	registry.RegisterCommand("clusters", m.handleClusterCommand)
	registry.RegisterCommand("cls", m.handleClusterCommand)
	registry.RegisterCommand("context", m.handleClusterCommand)
	registry.RegisterCommand("ctx", m.handleClusterCommand)
	registry.RegisterCommand("namespace", m.handleNamespaceCommand)
	registry.RegisterCommand("namespaces", m.handleNamespaceCommand)
	registry.RegisterCommand("ns", m.handleNamespaceCommand)
	registry.RegisterCommand("project", m.handleProjectCommand)
	registry.RegisterCommand("projects", m.handleProjectCommand)
	registry.RegisterCommand("proj", m.handleProjectCommand)
	registry.RegisterCommand("app", m.handleAppCommand)
	registry.RegisterCommand("apps", m.handleAppCommand)

	// System commands
	registry.RegisterCommand("help", m.handleHelpCommand)
	registry.RegisterCommand("quit", m.handleQuitCommand)
	registry.RegisterCommand("q", m.handleQuitCommand)
	registry.RegisterCommand("q!", m.handleQuitCommand)
	registry.RegisterCommand("wq", m.handleQuitCommand)
	registry.RegisterCommand("wq!", m.handleQuitCommand)
	registry.RegisterCommand("exit", m.handleQuitCommand)
	registry.RegisterCommand("upgrade", m.handleUpgradeCommand)
	registry.RegisterCommand("update", m.handleUpgradeCommand)

	cblog.With("component", "command-registry").Info("Command handlers initialized",
		"handler_count", registry.HandlersCount())
}