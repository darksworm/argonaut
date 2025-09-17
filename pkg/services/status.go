package services

import (
	"fmt"
	cblog "github.com/charmbracelet/log"
)

// StatusLevel represents the level of a status message
type StatusLevel string

const (
	StatusLevelInfo  StatusLevel = "info"
	StatusLevelWarn  StatusLevel = "warn"
	StatusLevelError StatusLevel = "error"
	StatusLevelDebug StatusLevel = "debug"
)

// StatusMessage represents a status message
type StatusMessage struct {
	Level   StatusLevel `json:"level"`
	Message string      `json:"message"`
}

// StatusChangeHandler is called when status changes
type StatusChangeHandler func(message StatusMessage)

// StatusService interface defines operations for status logging
type StatusService interface {
	// Info logs an info message
	Info(message string)

	// Warn logs a warning message
	Warn(message string)

	// Error logs an error message
	Error(message string)

	// Debug logs a debug message
	Debug(message string)

	// Set sets a status message (typically for current status display)
	Set(message string)

	// Clear clears the current status
	Clear()

	// SetHandler sets the status change handler
	SetHandler(handler StatusChangeHandler)
}

// StatusServiceImpl provides a concrete implementation of StatusService
type StatusServiceImpl struct {
	handler       StatusChangeHandler
	currentStatus string
	debugEnabled  bool
}

// StatusServiceConfig holds configuration for StatusService
type StatusServiceConfig struct {
	Handler      StatusChangeHandler
	DebugEnabled bool
}

// NewStatusService creates a new StatusService implementation
func NewStatusService(config StatusServiceConfig) StatusService {
	return &StatusServiceImpl{
		handler:      config.Handler,
		debugEnabled: config.DebugEnabled,
	}
}

// Info implements StatusService.Info
func (s *StatusServiceImpl) Info(message string) {
	msg := StatusMessage{
		Level:   StatusLevelInfo,
		Message: message,
	}
	s.handleMessage(msg)
}

// Warn implements StatusService.Warn
func (s *StatusServiceImpl) Warn(message string) {
	msg := StatusMessage{
		Level:   StatusLevelWarn,
		Message: message,
	}
	s.handleMessage(msg)
}

// Error implements StatusService.Error
func (s *StatusServiceImpl) Error(message string) {
	msg := StatusMessage{
		Level:   StatusLevelError,
		Message: message,
	}
	s.handleMessage(msg)
}

// Debug implements StatusService.Debug
func (s *StatusServiceImpl) Debug(message string) {
	if !s.debugEnabled {
		return
	}

	msg := StatusMessage{
		Level:   StatusLevelDebug,
		Message: message,
	}
	s.handleMessage(msg)
}

// Set implements StatusService.Set
func (s *StatusServiceImpl) Set(message string) {
	s.currentStatus = message
	s.Info(message)
}

// Clear implements StatusService.Clear
func (s *StatusServiceImpl) Clear() {
	s.currentStatus = ""
}

// SetHandler implements StatusService.SetHandler
func (s *StatusServiceImpl) SetHandler(handler StatusChangeHandler) {
	s.handler = handler
}

// GetCurrentStatus returns the current status message
func (s *StatusServiceImpl) GetCurrentStatus() string {
	return s.currentStatus
}

// handleMessage processes a status message
func (s *StatusServiceImpl) handleMessage(msg StatusMessage) {
	// Log via charmbracelet/log
	logger := cblog.With("component", "status")
	switch msg.Level {
	case StatusLevelError:
		logger.Error(msg.Message)
	case StatusLevelWarn:
		logger.Warn(msg.Message)
	case StatusLevelInfo:
		logger.Info(msg.Message)
	case StatusLevelDebug:
		logger.Debug(msg.Message)
	}

	// Call custom handler if provided
	if s.handler != nil {
		s.handler(msg)
	}
}

// DefaultStatusChangeHandler provides a default handler that just prints to stdout
func DefaultStatusChangeHandler(msg StatusMessage) {
	switch msg.Level {
	case StatusLevelError:
		fmt.Printf("❌ %s\n", msg.Message)
	case StatusLevelWarn:
		fmt.Printf("⚠️  %s\n", msg.Message)
	case StatusLevelInfo:
		fmt.Printf("ℹ️  %s\n", msg.Message)
	case StatusLevelDebug:
		fmt.Printf("🐛 %s\n", msg.Message)
	}
}

// NullStatusChangeHandler provides a handler that does nothing (for testing)
func NullStatusChangeHandler(msg StatusMessage) {
	// Do nothing
}
