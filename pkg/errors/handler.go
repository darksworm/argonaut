package errors

import (
	"context"
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

// Close closes the error handler and any associated resources
func (h *ErrorHandlerImpl) Close() error {
	if h.logFile != nil {
		return h.logFile.Close()
	}
	return nil
}

// Handle processes an error and returns the appropriate response
func (h *ErrorHandlerImpl) Handle(err *ArgonautError) *ErrorResponse {
	if err == nil {
		return nil
	}

	// Log the error
	h.Log(err)

	// Add to history
	h.addToHistory(*err)

	// Notify UI if callback is set
	h.Notify(err)

	// Determine response based on error category and severity
	response := h.determineResponse(err)

	return response
}

// Log logs the error with structured information
func (h *ErrorHandlerImpl) Log(err *ArgonautError) {
	if err == nil {
		return
	}

	// Create structured log entry
	logEntry := fmt.Sprintf("[%s] [%s:%s] %s",
		err.Severity, err.Category, err.Code, err.Message)

	if err.Details != "" {
		logEntry += fmt.Sprintf(" | Details: %s", err.Details)
	}

	if err.Cause != nil {
		logEntry += fmt.Sprintf(" | Cause: %v", err.Cause)
	}

	if len(err.Context) > 0 {
		logEntry += fmt.Sprintf(" | Context: %+v", err.Context)
	}

	// Log based on severity
	switch err.Severity {
	case SeverityCritical:
		h.logger.Printf("CRITICAL: %s", logEntry)
	case SeverityHigh:
		h.logger.Printf("ERROR: %s", logEntry)
	case SeverityMedium:
		h.logger.Printf("WARN: %s", logEntry)
	case SeverityLow:
		h.logger.Printf("INFO: %s", logEntry)
	default:
		h.logger.Printf("UNKNOWN: %s", logEntry)
	}
}

// Notify sends the error to the UI notification system
func (h *ErrorHandlerImpl) Notify(err *ArgonautError) {
	if err == nil || h.notifyFunc == nil {
		return
	}

	h.notifyFunc(err)
}

// ShouldRetry determines if an error should be retried
func (h *ErrorHandlerImpl) ShouldRetry(err *ArgonautError) bool {
	if err == nil || !err.Recoverable {
		return false
	}

	// Retry logic based on error category
	switch err.Category {
	case ErrorNetwork, ErrorTimeout:
		return true
	case ErrorAPI:
		// Only retry on specific API error codes
		return err.Code == "CONNECTION_REFUSED" || err.Code == "TIMEOUT" || err.Code == "SERVICE_UNAVAILABLE"
	case ErrorAuth:
		return false // Don't auto-retry auth errors
	case ErrorValidation:
		return false // Don't retry validation errors
	default:
		return false
	}
}

// GetRetryDelay returns the appropriate delay before retrying an operation
func (h *ErrorHandlerImpl) GetRetryDelay(err *ArgonautError) time.Duration {
	if err == nil {
		return 0
	}

	// Base delay based on error category
	baseDelay := 1 * time.Second

	switch err.Category {
	case ErrorNetwork:
		baseDelay = 2 * time.Second
	case ErrorTimeout:
		baseDelay = 3 * time.Second
	case ErrorAPI:
		baseDelay = 1 * time.Second
	}

	// TODO: Implement exponential backoff based on retry count
	// This would require tracking retry attempts per error

	return baseDelay
}

// GetErrorHistory returns a copy of the recent error history
func (h *ErrorHandlerImpl) GetErrorHistory() []ArgonautError {
	h.historyMu.RLock()
	defer h.historyMu.RUnlock()

	history := make([]ArgonautError, len(h.errorHistory))
	copy(history, h.errorHistory)
	return history
}

// ClearErrorHistory clears the error history
func (h *ErrorHandlerImpl) ClearErrorHistory() {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	h.errorHistory = nil
}

// addToHistory adds an error to the internal history, maintaining the max size
func (h *ErrorHandlerImpl) addToHistory(err ArgonautError) {
	h.historyMu.Lock()
	defer h.historyMu.Unlock()

	h.errorHistory = append(h.errorHistory, err)

	// Maintain max history size
	if len(h.errorHistory) > h.maxHistory {
		h.errorHistory = h.errorHistory[1:]
	}
}

// determineResponse determines the appropriate response for an error
func (h *ErrorHandlerImpl) determineResponse(err *ArgonautError) *ErrorResponse {
	response := &ErrorResponse{
		DisplayMessage: err.Message,
		ShouldExit:     false,
	}

	// Set mode based on error category
	switch err.Category {
	case ErrorAuth:
		response.Mode = "auth-required"
		response.UserActions = []UserAction{
			{
				Label:       "Login",
				Description: "Authenticate with ArgoCD server",
				Command:     "argocd login",
			},
		}
	case ErrorNetwork:
		response.Mode = "connection-error"
		if h.ShouldRetry(err) {
			retryDelay := h.GetRetryDelay(err)
			response.RetryAfter = &retryDelay
		}
		response.UserActions = []UserAction{
			{
				Label:       "Retry",
				Description: "Try the operation again",
			},
			{
				Label:       "Check Connection",
				Description: "Verify network connectivity to ArgoCD server",
			},
		}
	case ErrorConfig:
		response.Mode = "error"
		response.UserActions = []UserAction{
			{
				Label:       "Check Config",
				Description: "Verify ArgoCD configuration",
				Command:     "argocd config",
			},
		}
	default:
		response.Mode = "error"
		if err.UserAction != "" {
			response.UserActions = []UserAction{
				{
					Label:       "Suggested Action",
					Description: err.UserAction,
				},
			}
		}
	}

	// Set exit condition for critical errors
	if err.Severity == SeverityCritical {
		response.ShouldExit = true
	}

	return response
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

// Convenience functions for common error handling patterns

// HandleWithContext handles an error with additional context
func HandleWithContext(ctx context.Context, err *ArgonautError, contextData map[string]interface{}) *ErrorResponse {
	if err != nil && contextData != nil {
		for k, v := range contextData {
			err.WithContext(k, v)
		}
	}

	return GetDefaultHandler().Handle(err)
}

// LogAndHandle logs an error and returns the response
func LogAndHandle(err *ArgonautError) *ErrorResponse {
	return GetDefaultHandler().Handle(err)
}

// ConvertError converts a standard error to an ArgonautError
func ConvertError(err error, category ErrorCategory, code string) *ArgonautError {
	if err == nil {
		return nil
	}

	// Check if it's already an ArgonautError
	if argErr, ok := err.(*ArgonautError); ok {
		return argErr
	}

	return Wrap(err, category, code, err.Error())
}
