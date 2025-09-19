package tui

import (
	"os"
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
)

// ErrorHandler manages TUI error state and responses
type ErrorHandler struct {
	handler *apperrors.ErrorHandlerImpl
}

// NewErrorHandler creates a new TUI error handler
func NewErrorHandler() (*ErrorHandler, error) {
	// Use temp file for TUI errors
	logFile, err := os.CreateTemp("", "a9s-tui-errors-*.log")
	if err != nil {
		// Fallback to basic handler if temp file creation fails
		return &ErrorHandler{
			handler: nil,
		}, nil
	}

	handler, err := apperrors.NewErrorHandler(apperrors.ErrorHandlerConfig{
		LogFilePath: logFile.Name(),
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

// UpdateAppErrorState updates error state in app state
func UpdateAppErrorState(state *model.AppState, err *apperrors.ArgonautError) {
	GetDefaultTUIHandler().UpdateErrorState(state, err)
}
