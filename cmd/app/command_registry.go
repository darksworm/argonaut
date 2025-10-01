package main

import (
	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
)

// CommandHandler defines the signature for command handling functions.
// Each handler receives a command and optional argument, returning updated model and command.
type CommandHandler func(cmd string, arg string) (tea.Model, tea.Cmd)

// KeyHandler defines the signature for key handling functions.
// Each handler receives a key string and returns model, command, and whether it was handled.
type KeyHandler func(key string) (tea.Model, tea.Cmd, bool)

// CommandRegistry provides type-safe registration and lookup of command and key handlers.
// This implements the Observer/Event Listener pattern for command handling.
type CommandRegistry struct {
	keyHandlers     map[string]KeyHandler
	commandHandlers map[string]CommandHandler
}

// NewCommandRegistry creates a new empty command registry.
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		keyHandlers:     make(map[string]KeyHandler),
		commandHandlers: make(map[string]CommandHandler),
	}
}

// RegisterKey associates a key string with its handler function.
func (r *CommandRegistry) RegisterKey(key string, handler KeyHandler) {
	r.keyHandlers[key] = handler
	cblog.With("component", "command-registry").Debug("Registered key handler",
		"key", key)
}

// RegisterCommand associates a command string with its handler function.
func (r *CommandRegistry) RegisterCommand(cmd string, handler CommandHandler) {
	r.commandHandlers[cmd] = handler
	cblog.With("component", "command-registry").Debug("Registered command handler",
		"command", cmd)
}

// GetKeyHandler looks up the handler for a given key.
// Returns the handler function and true if found, nil and false otherwise.
func (r *CommandRegistry) GetKeyHandler(key string) (KeyHandler, bool) {
	handler, exists := r.keyHandlers[key]
	return handler, exists
}

// GetCommandHandler looks up the handler for a given command.
// Returns the handler function and true if found, nil and false otherwise.
func (r *CommandRegistry) GetCommandHandler(cmd string) (CommandHandler, bool) {
	handler, exists := r.commandHandlers[cmd]
	return handler, exists
}

// ListRegisteredKeys returns all currently registered key handlers.
// Useful for debugging and documentation purposes.
func (r *CommandRegistry) ListRegisteredKeys() []string {
	keys := make([]string, 0, len(r.keyHandlers))
	for key := range r.keyHandlers {
		keys = append(keys, key)
	}
	return keys
}

// ListRegisteredCommands returns all currently registered command handlers.
// Useful for debugging and documentation purposes.
func (r *CommandRegistry) ListRegisteredCommands() []string {
	commands := make([]string, 0, len(r.commandHandlers))
	for cmd := range r.commandHandlers {
		commands = append(commands, cmd)
	}
	return commands
}

// HandlersCount returns the total number of registered handlers.
func (r *CommandRegistry) HandlersCount() int {
	return len(r.keyHandlers) + len(r.commandHandlers)
}