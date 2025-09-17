package errors

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrorHandler interface defines methods for handling errors consistently
type ErrorHandler interface {
	Handle(err *ArgonautError) *ErrorResponse
	Log(err *ArgonautError)
	Notify(err *ArgonautError) // Send to UI/notification system
	ShouldRetry(err *ArgonautError) bool
	GetRetryDelay(err *ArgonautError) time.Duration
}

// ErrorResponse represents how the application should respond to an error
type ErrorResponse struct {
	ShouldExit     bool           `json:"shouldExit"`
	DisplayMessage string         `json:"displayMessage"`
	Mode           string         `json:"mode"` // error, connection-error, auth-required
	RetryAfter     *time.Duration `json:"retryAfter,omitempty"`
	UserActions    []UserAction   `json:"userActions,omitempty"`
}

// UserAction represents an action the user can take to resolve an error
type UserAction struct {
	Label       string `json:"label"`
	Description string `json:"description"`
	Command     string `json:"command,omitempty"`
}

// ErrorHandlerImpl provides a concrete implementation of ErrorHandler
type ErrorHandlerImpl struct {
	logger       *log.Logger
	logFile      *os.File
	errorHistory []ArgonautError
	historyMu    sync.RWMutex
	maxHistory   int
	notifyFunc   func(*ArgonautError) // Callback for UI notifications
}

// ErrorHandlerConfig configures the error handler
type ErrorHandlerConfig struct {
	LogFilePath    string
	MaxHistory     int
	NotifyCallback func(*ArgonautError)
}

// NewErrorHandler creates a new error handler with the given configuration
func NewErrorHandler(config ErrorHandlerConfig) (*ErrorHandlerImpl, error) {
	handler := &ErrorHandlerImpl{
		maxHistory: config.MaxHistory,
		notifyFunc: config.NotifyCallback,
	}

	if handler.maxHistory <= 0 {
		handler.maxHistory = 100 // Default history size
	}

	// Set up logging
	if config.LogFilePath != "" {
		// Ensure log directory exists
		logDir := filepath.Dir(config.LogFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		// Open log file
		logFile, err := os.OpenFile(config.LogFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		handler.logFile = logFile
		handler.logger = log.New(logFile, "", log.LstdFlags|log.Lshortfile)
	} else {
		handler.logger = log.Default()
	}

	return handler, nil
}

// Default error handler instance
var defaultHandler *ErrorHandlerImpl
var defaultHandlerOnce sync.Once

// GetDefaultHandler returns the default error handler instance
func GetDefaultHandler() *ErrorHandlerImpl {
	defaultHandlerOnce.Do(func() {
		// Use temp file for error logs
		logFile, err := os.CreateTemp("", "a9s-errors-*.log")
		logFilePath := "logs/errors.log" // fallback
		if err == nil {
			logFilePath = logFile.Name()
		}

		config := ErrorHandlerConfig{
			LogFilePath: logFilePath,
			MaxHistory:  100,
		}

		handler, err := NewErrorHandler(config)
		if err != nil {
			// Fallback to basic handler
			handler = &ErrorHandlerImpl{
				logger:     log.Default(),
				maxHistory: 100,
			}
		}

		defaultHandler = handler
	})

	return defaultHandler
}
