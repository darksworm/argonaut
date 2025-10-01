package main

import (
	"reflect"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
)

// MessageHandler defines the signature for message handling functions.
// Each handler receives a message, returning updated model and command.
type MessageHandler func(tea.Msg) (tea.Model, tea.Cmd)

// MessageRegistry provides type-safe registration and lookup of message handlers.
// This implements the Observer/Event Listener pattern for Bubble Tea messages.
type MessageRegistry struct {
	handlers map[reflect.Type]MessageHandler
}

// NewMessageRegistry creates a new empty message registry.
func NewMessageRegistry() *MessageRegistry {
	return &MessageRegistry{
		handlers: make(map[reflect.Type]MessageHandler),
	}
}

// Register associates a message type with its handler function.
// The msgExample parameter is used only for type inference - its value is ignored.
func (r *MessageRegistry) Register(msgExample tea.Msg, handler MessageHandler) {
	msgType := reflect.TypeOf(msgExample)
	r.handlers[msgType] = handler

	cblog.With("component", "registry").Debug("Registered message handler",
		"type", msgType.String())
}

// GetHandler looks up the handler for a given message.
// Returns the handler function and true if found, nil and false otherwise.
func (r *MessageRegistry) GetHandler(msg tea.Msg) (MessageHandler, bool) {
	msgType := reflect.TypeOf(msg)
	handler, exists := r.handlers[msgType]
	return handler, exists
}

// ListRegisteredTypes returns all currently registered message types.
// Useful for debugging and documentation purposes.
func (r *MessageRegistry) ListRegisteredTypes() []reflect.Type {
	types := make([]reflect.Type, 0, len(r.handlers))
	for msgType := range r.handlers {
		types = append(types, msgType)
	}
	return types
}

// HandlersCount returns the number of registered message handlers.
func (r *MessageRegistry) HandlersCount() int {
	return len(r.handlers)
}