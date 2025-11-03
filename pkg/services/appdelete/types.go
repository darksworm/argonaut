package appdelete

import (
	"context"

	"github.com/darksworm/argonaut/pkg/model"
)

// AppDeleteRequest represents a request to delete an application
type AppDeleteRequest struct {
	AppName           string
	AppNamespace      *string
	Cascade           bool
	PropagationPolicy string // foreground, background, or orphan
	Force             bool
}

// AppDeleteResponse represents the response from an application delete operation
type AppDeleteResponse struct {
	Success bool
	Error   *AppDeleteError
}

// AppDeleteError represents an error during application deletion with recovery hints
type AppDeleteError struct {
	Code       string // NOT_FOUND, FORBIDDEN, CONFLICT, UNKNOWN
	Message    string
	Retryable  bool
	StatusCode int
}

// AppDeleteModalState manages the state of the application delete confirmation modal
type AppDeleteModalState struct {
	Active           bool
	AppName          string
	AppNamespace     *string
	ConfirmationKey  string // Track what user has typed
	Loading          bool
	Error            *AppDeleteError
	Options          AppDeleteOptions
	ConfirmSelected  int // 0 = Delete, 1 = Cancel
}

// AppDeleteOptions configures the application deletion behavior
type AppDeleteOptions struct {
	Cascade           bool
	PropagationPolicy string
}

// NewAppDeleteModalState creates a new application delete modal state with defaults
func NewAppDeleteModalState() *AppDeleteModalState {
	return &AppDeleteModalState{
		Active:          false,
		ConfirmationKey: "",
		Loading:         false,
		Options: AppDeleteOptions{
			Cascade:           true, // Default to cascade for safety
			PropagationPolicy: "foreground",
		},
		ConfirmSelected: 1, // Default to Cancel button
	}
}

// AppDeleteService interface defines operations for application deletion
type AppDeleteService interface {
	// DeleteApplication orchestrates the deletion of an application
	DeleteApplication(ctx context.Context, server *model.Server, req AppDeleteRequest) (*AppDeleteResponse, error)

	// ValidateDeleteRequest validates a delete request before execution
	ValidateDeleteRequest(req AppDeleteRequest) error
}