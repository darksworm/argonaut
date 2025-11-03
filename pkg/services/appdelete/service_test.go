package appdelete

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
)

// MockApplicationService implements a mock for testing
type MockApplicationService struct {
	DeleteFunc func(ctx context.Context, req api.DeleteRequest) error
	Apps       []model.App
}

func (m *MockApplicationService) DeleteApplication(ctx context.Context, req api.DeleteRequest) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, req)
	}
	return nil
}

func TestAppDeleteService_ValidateRequest(t *testing.T) {
	service := NewAppDeleteServiceWithAPI(&MockApplicationService{})

	tests := []struct {
		name    string
		req     AppDeleteRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid request with cascade",
			req: AppDeleteRequest{
				AppName:           "test-app",
				Cascade:           true,
				PropagationPolicy: "foreground",
			},
			wantErr: false,
		},
		{
			name: "Valid request without cascade",
			req: AppDeleteRequest{
				AppName:           "test-app",
				Cascade:           false,
				PropagationPolicy: "orphan",
			},
			wantErr: false,
		},
		{
			name: "Invalid - missing app name",
			req: AppDeleteRequest{
				AppName: "",
				Cascade: true,
			},
			wantErr: true,
			errMsg:  "application name is required",
		},
		{
			name: "Invalid propagation policy",
			req: AppDeleteRequest{
				AppName:           "test-app",
				Cascade:           true,
				PropagationPolicy: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid propagation policy",
		},
		{
			name: "Valid with namespace",
			req: AppDeleteRequest{
				AppName:      "test-app",
				AppNamespace: stringPtr("test-namespace"),
				Cascade:      true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidateDeleteRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDeleteRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateDeleteRequest() error = %v, want %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

func TestAppDeleteService_HandleAPIErrors(t *testing.T) {
	tests := []struct {
		name          string
		apiError      error
		expectedError *AppDeleteError
	}{
		{
			name:     "404 Not Found",
			apiError: fmt.Errorf("404 Not Found"),
			expectedError: &AppDeleteError{
				Code:       "NOT_FOUND",
				Message:    "Application no longer exists",
				Retryable:  false,
				StatusCode: 404,
			},
		},
		{
			name:     "403 Forbidden",
			apiError: fmt.Errorf("403 Forbidden"),
			expectedError: &AppDeleteError{
				Code:       "FORBIDDEN",
				Message:    "Permission denied to delete application",
				Retryable:  false,
				StatusCode: 403,
			},
		},
		{
			name:     "409 Conflict",
			apiError: fmt.Errorf("409 Conflict"),
			expectedError: &AppDeleteError{
				Code:       "CONFLICT",
				Message:    "Application has finalizers or cannot be deleted",
				Retryable:  true,
				StatusCode: 409,
			},
		},
		{
			name:     "500 Server Error",
			apiError: fmt.Errorf("500 Internal Server Error"),
			expectedError: &AppDeleteError{
				Code:       "SERVER_ERROR",
				Message:    "Server error occurred during deletion",
				Retryable:  true,
				StatusCode: 500,
			},
		},
		{
			name:     "Unknown error",
			apiError: errors.New("some unknown error"),
			expectedError: &AppDeleteError{
				Code:       "UNKNOWN",
				Message:    "some unknown error",
				Retryable:  true,
				StatusCode: 0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAppDeleteServiceWithAPI(&MockApplicationService{})
			deleteErr := service.(*appDeleteServiceImpl).handleAPIError(tt.apiError)

			if deleteErr.Code != tt.expectedError.Code {
				t.Errorf("Code = %v, want %v", deleteErr.Code, tt.expectedError.Code)
			}
			if deleteErr.Retryable != tt.expectedError.Retryable {
				t.Errorf("Retryable = %v, want %v", deleteErr.Retryable, tt.expectedError.Retryable)
			}
		})
	}
}

func TestAppDeleteService_DeleteApplication(t *testing.T) {
	tests := []struct {
		name       string
		req        AppDeleteRequest
		mockFunc   func(ctx context.Context, req api.DeleteRequest) error
		wantResp   *AppDeleteResponse
		wantErr    bool
	}{
		{
			name: "Successful deletion",
			req: AppDeleteRequest{
				AppName: "test-app",
				Cascade: true,
			},
			mockFunc: func(ctx context.Context, req api.DeleteRequest) error {
				return nil
			},
			wantResp: &AppDeleteResponse{
				Success: true,
				Error:   nil,
			},
			wantErr: false,
		},
		{
			name: "Failed deletion - not found",
			req: AppDeleteRequest{
				AppName: "test-app",
				Cascade: true,
			},
			mockFunc: func(ctx context.Context, req api.DeleteRequest) error {
				return fmt.Errorf("404 Not Found")
			},
			wantResp: &AppDeleteResponse{
				Success: false,
				Error: &AppDeleteError{
					Code:       "NOT_FOUND",
					Message:    "Application no longer exists",
					Retryable:  false,
					StatusCode: 404,
				},
			},
			wantErr: false, // We return error in response, not as error
		},
		{
			name: "Invalid request",
			req: AppDeleteRequest{
				AppName: "",
				Cascade: true,
			},
			mockFunc: nil,
			wantResp: nil,
			wantErr:  true, // Validation error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAPI := &MockApplicationService{
				DeleteFunc: tt.mockFunc,
			}

			service := NewAppDeleteServiceWithAPI(mockAPI)
			resp, err := service.DeleteApplication(context.Background(), &model.Server{}, tt.req)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteApplication() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && resp != nil {
				if resp.Success != tt.wantResp.Success {
					t.Errorf("Success = %v, want %v", resp.Success, tt.wantResp.Success)
				}
				if tt.wantResp.Error != nil && resp.Error != nil {
					if resp.Error.Code != tt.wantResp.Error.Code {
						t.Errorf("Error.Code = %v, want %v", resp.Error.Code, tt.wantResp.Error.Code)
					}
				}
			}
		})
	}
}

func TestAppDeleteService_UpdateLocalState(t *testing.T) {
	apps := []model.App{
		{Name: "app1"},
		{Name: "app2"},
		{Name: "app3"},
	}

	tests := []struct {
		name         string
		appToDelete  string
		expectedApps []model.App
	}{
		{
			name:        "Delete existing app",
			appToDelete: "app2",
			expectedApps: []model.App{
				{Name: "app1"},
				{Name: "app3"},
			},
		},
		{
			name:        "Delete non-existing app",
			appToDelete: "app4",
			expectedApps: []model.App{
				{Name: "app1"},
				{Name: "app2"},
				{Name: "app3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewAppDeleteServiceWithAPI(&MockApplicationService{})
			updatedApps := service.(*appDeleteServiceImpl).removeAppFromList(apps, tt.appToDelete)

			if len(updatedApps) != len(tt.expectedApps) {
				t.Errorf("Got %d apps, want %d", len(updatedApps), len(tt.expectedApps))
			}

			for i, app := range updatedApps {
				if app.Name != tt.expectedApps[i].Name {
					t.Errorf("App[%d].Name = %v, want %v", i, app.Name, tt.expectedApps[i].Name)
				}
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}