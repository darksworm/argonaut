package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestTerminateOperation_DeletesTheRunningOperation(t *testing.T) {
	var gotMethod, gotPath string
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	if err := svc.TerminateOperation(context.Background(), "test-app", nil); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !called {
		t.Fatal("Expected a request to the ArgoCD API, got none")
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("Expected DELETE request, got %s", gotMethod)
	}
	if gotPath != "/api/v1/applications/test-app/operation" {
		t.Errorf("Expected path /api/v1/applications/test-app/operation, got %s", gotPath)
	}
}

func TestTerminateOperation_ScopesToTheApplicationNamespace(t *testing.T) {
	var gotNamespace string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotNamespace = r.URL.Query().Get("appNamespace")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	appNamespace := "team-a"
	if err := svc.TerminateOperation(context.Background(), "test-app", &appNamespace); err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if gotNamespace != "team-a" {
		t.Errorf("Expected appNamespace=team-a, got %q", gotNamespace)
	}
}

func TestTerminateOperation_ReportsServerRejection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"Unable to terminate operation. No operation is in progress"}`))
	}))
	defer server.Close()

	svc := NewApplicationService(&model.Server{BaseURL: server.URL, Token: "test-token"})

	err := svc.TerminateOperation(context.Background(), "test-app", nil)
	if err == nil {
		t.Fatal("Expected an error when the server rejects the termination, got nil")
	}
	if !strings.Contains(err.Error(), "test-app") {
		t.Errorf("Expected the error to name the application, got %v", err)
	}
	// The modal shows this text, so the server's reason has to survive the trip.
	if !strings.Contains(err.Error(), "No operation is in progress") {
		t.Errorf("Expected Argo CD's reason to reach the caller, got %v", err)
	}
}
