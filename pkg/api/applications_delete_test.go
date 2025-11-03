package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestDeleteApplication_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/applications/test-app" {
			t.Errorf("Expected path /api/v1/applications/test-app, got %s", r.URL.Path)
		}

		// Check query parameters
		cascade := r.URL.Query().Get("cascade")
		if cascade != "true" {
			t.Errorf("Expected cascade=true, got %s", cascade)
		}

		propagationPolicy := r.URL.Query().Get("propagationPolicy")
		if propagationPolicy != "foreground" {
			t.Errorf("Expected propagationPolicy=foreground, got %s", propagationPolicy)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	req := DeleteRequest{
		AppName:           "test-app",
		Cascade:           true,
		PropagationPolicy: "foreground",
	}

	err := svc.DeleteApplication(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDeleteApplication_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		errorResp := map[string]interface{}{
			"error":   "Not Found",
			"message": "applications.argoproj.io \"test-app\" not found",
		}
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	req := DeleteRequest{
		AppName: "test-app",
		Cascade: true,
	}

	err := svc.DeleteApplication(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}

	// Check that the error contains appropriate context
	if err.Error() == "" {
		t.Error("Expected error message to be non-empty")
	}
}

func TestDeleteApplication_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		errorResp := map[string]interface{}{
			"error":   "Forbidden",
			"message": "User does not have delete permissions",
		}
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	req := DeleteRequest{
		AppName: "test-app",
		Cascade: true,
	}

	err := svc.DeleteApplication(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for 403 response")
	}
}

func TestDeleteApplication_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		errorResp := map[string]interface{}{
			"error":   "Conflict",
			"message": "Application has finalizers",
		}
		json.NewEncoder(w).Encode(errorResp)
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	req := DeleteRequest{
		AppName: "test-app",
		Cascade: true,
	}

	err := svc.DeleteApplication(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error for 409 response")
	}
}

func TestDeleteApplication_WithNamespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications/test-app" {
			t.Errorf("Expected path /api/v1/applications/test-app, got %s", r.URL.Path)
		}

		appNamespace := r.URL.Query().Get("appNamespace")
		if appNamespace != "test-namespace" {
			t.Errorf("Expected appNamespace=test-namespace, got %s", appNamespace)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	namespace := "test-namespace"
	req := DeleteRequest{
		AppName:      "test-app",
		AppNamespace: &namespace,
		Cascade:      true,
	}

	err := svc.DeleteApplication(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestDeleteApplication_WithPropagationPolicy(t *testing.T) {
	policies := []string{"foreground", "background", "orphan"}

	for _, policy := range policies {
		t.Run(policy, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				propagationPolicy := r.URL.Query().Get("propagationPolicy")
				if propagationPolicy != policy {
					t.Errorf("Expected propagationPolicy=%s, got %s", policy, propagationPolicy)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte("{}"))
			}))
			defer server.Close()

			svc := NewApplicationService(&model.Server{
				BaseURL: server.URL,
				Token:   "test-token",
			})

			req := DeleteRequest{
				AppName:           "test-app",
				Cascade:           true,
				PropagationPolicy: policy,
			}

			err := svc.DeleteApplication(context.Background(), req)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		})
	}
}

func TestDeleteApplication_NoCascade(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cascade := r.URL.Query().Get("cascade")
		if cascade != "false" {
			t.Errorf("Expected cascade=false, got %s", cascade)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	req := DeleteRequest{
		AppName: "test-app",
		Cascade: false,
	}

	err := svc.DeleteApplication(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}