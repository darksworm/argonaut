package appdelete

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
)

// ApplicationAPI interface for dependency injection
type ApplicationAPI interface {
	DeleteApplication(ctx context.Context, req api.DeleteRequest) error
}

// appDeleteServiceImpl provides the implementation of AppDeleteService
type appDeleteServiceImpl struct {
	appService ApplicationAPI
}

// NewAppDeleteService creates a new application delete service with the default API service
func NewAppDeleteService(server *model.Server) AppDeleteService {
	return &appDeleteServiceImpl{
		appService: api.NewApplicationService(server),
	}
}

// NewAppDeleteServiceWithAPI creates a new application delete service with a custom API service (for testing)
func NewAppDeleteServiceWithAPI(appService ApplicationAPI) AppDeleteService {
	return &appDeleteServiceImpl{
		appService: appService,
	}
}

// DeleteApplication orchestrates the deletion of an application
func (s *appDeleteServiceImpl) DeleteApplication(ctx context.Context, server *model.Server, req AppDeleteRequest) (*AppDeleteResponse, error) {
	// Validate the request first
	if err := s.ValidateDeleteRequest(req); err != nil {
		return nil, err
	}

	// Convert to API request
	apiReq := api.DeleteRequest{
		AppName:           req.AppName,
		AppNamespace:      req.AppNamespace,
		Cascade:           req.Cascade,
		PropagationPolicy: req.PropagationPolicy,
	}

	// Call the API to delete the application
	err := s.appService.DeleteApplication(ctx, apiReq)
	if err != nil {
		// Handle API error and return structured response
		deleteErr := s.handleAPIError(err)
		return &AppDeleteResponse{
			Success: false,
			Error:   deleteErr,
		}, nil // Return error in response, not as function error
	}

	// Success
	return &AppDeleteResponse{
		Success: true,
		Error:   nil,
	}, nil
}

// ValidateDeleteRequest validates a delete request before execution
func (s *appDeleteServiceImpl) ValidateDeleteRequest(req AppDeleteRequest) error {
	if req.AppName == "" {
		return fmt.Errorf("application name is required")
	}

	// Validate propagation policy if specified
	if req.PropagationPolicy != "" {
		validPolicies := []string{"foreground", "background", "orphan"}
		valid := false
		for _, policy := range validPolicies {
			if req.PropagationPolicy == policy {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid propagation policy")
		}
	}

	return nil
}

// handleAPIError converts API errors to structured DeleteError
func (s *appDeleteServiceImpl) handleAPIError(err error) *AppDeleteError {
	errStr := err.Error()

	// Try to extract status code from error message
	statusCode := 0
	if strings.Contains(errStr, "404") {
		statusCode = 404
		return &AppDeleteError{
			Code:       "NOT_FOUND",
			Message:    "Application no longer exists",
			Retryable:  false,
			StatusCode: statusCode,
		}
	}

	if strings.Contains(errStr, "403") || strings.Contains(errStr, "Forbidden") {
		statusCode = 403
		return &AppDeleteError{
			Code:       "FORBIDDEN",
			Message:    "Permission denied to delete application",
			Retryable:  false,
			StatusCode: statusCode,
		}
	}

	if strings.Contains(errStr, "409") || strings.Contains(errStr, "Conflict") {
		statusCode = 409
		return &AppDeleteError{
			Code:       "CONFLICT",
			Message:    "Application has finalizers or cannot be deleted",
			Retryable:  true,
			StatusCode: statusCode,
		}
	}

	if strings.Contains(errStr, "500") || strings.Contains(errStr, "Internal Server Error") {
		statusCode = 500
		return &AppDeleteError{
			Code:       "SERVER_ERROR",
			Message:    "Server error occurred during deletion",
			Retryable:  true,
			StatusCode: statusCode,
		}
	}

	// Try to extract status code from error string
	parts := strings.Fields(errStr)
	for _, part := range parts {
		if code, err := strconv.Atoi(part); err == nil && code >= 400 && code < 600 {
			statusCode = code
			break
		}
	}

	// Default unknown error
	return &AppDeleteError{
		Code:       "UNKNOWN",
		Message:    err.Error(),
		Retryable:  true,
		StatusCode: statusCode,
	}
}

// removeAppFromList removes an application from the list by name
func (s *appDeleteServiceImpl) removeAppFromList(apps []model.App, appName string) []model.App {
	result := make([]model.App, 0, len(apps))
	for _, app := range apps {
		if app.Name != appName {
			result = append(result, app)
		}
	}
	return result
}