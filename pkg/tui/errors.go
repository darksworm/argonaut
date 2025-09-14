package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
)

// ErrorHandler manages TUI error state and responses
type ErrorHandler struct {
	handler *apperrors.ErrorHandlerImpl
}

// NewErrorHandler creates a new TUI error handler
func NewErrorHandler() (*ErrorHandler, error) {
	handler, err := apperrors.NewErrorHandler(apperrors.ErrorHandlerConfig{
		LogFilePath: "logs/tui-errors.log",
		MaxHistory:  50,
		NotifyCallback: func(err *apperrors.ArgonautError) {
			// This will be called when errors occur
			// Can be used for additional TUI notifications
		},
	})
	if err != nil {
		return nil, err
	}

	return &ErrorHandler{
		handler: handler,
	}, nil
}

// HandleError processes an error and returns appropriate TUI messages
func (h *ErrorHandler) HandleError(err *apperrors.ArgonautError) []tea.Msg {
	if err == nil {
		return nil
	}

	// Handle the error through the centralized handler
	response := h.handler.Handle(err)
	if response == nil {
		return nil
	}

	var messages []tea.Msg

	// Create structured error message for TUI
	structuredMsg := model.StructuredErrorMsg{
		Error:   err,
		Context: make(map[string]interface{}),
		Retry:   h.handler.ShouldRetry(err),
	}

	// Add auto-hide for low severity errors
	if err.Severity == apperrors.SeverityLow {
		structuredMsg.AutoHide = true
	}

	messages = append(messages, structuredMsg)

	// If the error should trigger a retry, schedule it
	if response.RetryAfter != nil {
		retryMsg := model.RetryOperationMsg{
			Operation: "last_operation", // This could be more specific
			Context:   err.Context,
			Attempt:   1,
		}

		// Schedule the retry after the specified delay
		messages = append(messages, tea.Tick(*response.RetryAfter, func(time.Time) tea.Msg {
			return retryMsg
		}))
	}

	return messages
}

// ConvertLegacyError converts old-style errors to structured errors
func (h *ErrorHandler) ConvertLegacyError(msg interface{}) *apperrors.ArgonautError {
	switch m := msg.(type) {
	case model.AuthErrorMsg:
		return apperrors.AuthError("AUTHENTICATION_REQUIRED", "Authentication required").
			WithCause(m.Error).
			WithUserAction("Please run 'argocd login' to authenticate")

	case model.ApiErrorMsg:
		category := apperrors.ErrorAPI
		code := "API_ERROR"

		// Determine category based on status code
		if m.StatusCode >= 401 && m.StatusCode < 403 {
			category = apperrors.ErrorAuth
			code = "UNAUTHORIZED"
		} else if m.StatusCode == 403 {
			category = apperrors.ErrorPermission
			code = "FORBIDDEN"
		} else if m.StatusCode == 404 {
			code = "NOT_FOUND"
		} else if m.StatusCode >= 500 {
			code = "SERVER_ERROR"
		}

		err := apperrors.New(category, code, m.Message).
			WithContext("statusCode", m.StatusCode).
			WithContext("errorCode", m.ErrorCode)

		if m.Details != "" {
			err = err.WithDetails(m.Details)
		}

		// Mark as recoverable for most API errors
		if m.StatusCode != 401 && m.StatusCode != 403 {
			err = err.AsRecoverable()
		}

		return err

	default:
		return nil
	}
}

// UpdateErrorState updates the error state in the app state
func (h *ErrorHandler) UpdateErrorState(state *model.AppState, err *apperrors.ArgonautError) {
	if state.ErrorState == nil {
		state.ErrorState = &model.ErrorState{
			History: make([]apperrors.ArgonautError, 0),
		}
	}

	// Set current error
	state.ErrorState.Current = err

	// Add to history
	if err != nil {
		state.ErrorState.History = append(state.ErrorState.History, *err)

		// Limit history size
		maxHistory := 20
		if len(state.ErrorState.History) > maxHistory {
			state.ErrorState.History = state.ErrorState.History[1:]
		}

		// Set auto-hide time for low severity errors
		if err.Severity == apperrors.SeverityLow {
			autoHide := time.Now().Add(5 * time.Second)
			state.ErrorState.AutoHideAt = &autoHide
		}
	}
}

// ShouldHideError checks if an error should be automatically hidden
func (h *ErrorHandler) ShouldHideError(state *model.ErrorState) bool {
	if state == nil || state.AutoHideAt == nil {
		return false
	}

	return time.Now().After(*state.AutoHideAt)
}

// ClearError clears the current error from the state
func (h *ErrorHandler) ClearError(state *model.AppState) {
	if state.ErrorState != nil {
		state.ErrorState.Current = nil
		state.ErrorState.AutoHideAt = nil
	}

	// Also clear legacy error for compatibility
	state.CurrentError = nil
}

// IncrementRetryCount increments the retry count for the current error
func (h *ErrorHandler) IncrementRetryCount(state *model.ErrorState) {
	if state != nil {
		state.RetryCount++
		now := time.Now()
		state.LastRetryAt = &now
	}
}

// GetErrorDisplayInfo returns formatted error information for display
func (h *ErrorHandler) GetErrorDisplayInfo(err *apperrors.ArgonautError) ErrorDisplayInfo {
	if err == nil {
		return ErrorDisplayInfo{}
	}

	info := ErrorDisplayInfo{
		Title:       string(err.Category),
		Message:     err.Message,
		Details:     err.Details,
		UserAction:  err.UserAction,
		Recoverable: err.Recoverable,
		Severity:    string(err.Severity),
	}

	// Format title based on category
	switch err.Category {
	case apperrors.ErrorAuth:
		info.Title = "Authentication Required"
		info.Icon = "🔐"
	case apperrors.ErrorNetwork:
		info.Title = "Network Error"
		info.Icon = "🌐"
	case apperrors.ErrorAPI:
		info.Title = "API Error"
		info.Icon = "⚠️"
	case apperrors.ErrorTimeout:
		info.Title = "Timeout"
		info.Icon = "⏱️"
	case apperrors.ErrorValidation:
		info.Title = "Validation Error"
		info.Icon = "❌"
	case apperrors.ErrorConfig:
		info.Title = "Configuration Error"
		info.Icon = "⚙️"
	default:
		info.Title = "Error"
		info.Icon = "❌"
	}

	return info
}

// ErrorDisplayInfo holds formatted error information for display
type ErrorDisplayInfo struct {
	Title       string `json:"title"`
	Message     string `json:"message"`
	Details     string `json:"details,omitempty"`
	UserAction  string `json:"userAction,omitempty"`
	Recoverable bool   `json:"recoverable"`
	Severity    string `json:"severity"`
	Icon        string `json:"icon"`
}

// CreateRetryCommand creates a tea.Cmd for retrying an operation
func (h *ErrorHandler) CreateRetryCommand(operation string, context map[string]interface{}, attempt int) tea.Cmd {
	return func() tea.Msg {
		return model.RetryOperationMsg{
			Operation: operation,
			Context:   context,
			Attempt:   attempt,
		}
	}
}

// CreateAutoHideCommand creates a tea.Cmd for auto-hiding errors
func (h *ErrorHandler) CreateAutoHideCommand(delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return AutoHideErrorMsg{}
	})
}

// AutoHideErrorMsg triggers automatic hiding of errors
type AutoHideErrorMsg struct{}

// Default TUI error handler instance
var defaultTUIHandler *ErrorHandler

// GetDefaultTUIHandler returns the default TUI error handler
func GetDefaultTUIHandler() *ErrorHandler {
	if defaultTUIHandler == nil {
		handler, err := NewErrorHandler()
		if err != nil {
			// Fallback to basic handler
			handler = &ErrorHandler{
				handler: apperrors.GetDefaultHandler(),
			}
		}
		defaultTUIHandler = handler
	}
	return defaultTUIHandler
}

// Package-level convenience functions

// HandleTUIError handles an error and returns TUI messages
func HandleTUIError(err *apperrors.ArgonautError) []tea.Msg {
	return GetDefaultTUIHandler().HandleError(err)
}

// ConvertError converts legacy error messages to structured errors
func ConvertError(msg interface{}) *apperrors.ArgonautError {
	return GetDefaultTUIHandler().ConvertLegacyError(msg)
}

// UpdateAppErrorState updates error state in app state
func UpdateAppErrorState(state *model.AppState, err *apperrors.ArgonautError) {
	GetDefaultTUIHandler().UpdateErrorState(state, err)
}