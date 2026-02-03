//go:build e2e && unix

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTimeoutErrorDisplay tests that timeout errors show the actual timeout value and config hint
func TestTimeoutErrorDisplay(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Create a mock server that delays response to trigger timeout
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			// Version endpoint works normally
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"Version":"v2.8.0+test"}`)
		} else if r.URL.Path == "/api/v1/applications" {
			// Applications endpoint delays to trigger timeout
			time.Sleep(3 * time.Second) // Sleep longer than our 2s test timeout
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)

	// Set up workspace and ArgoCD config
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Write config pointing to our mock server
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Start argonaut (will use 2s timeout from E2E test framework)
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for timeout error to appear
	if !tf.WaitForPlain("Request timed out after 2s", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see 'Request timed out after 2s' message")
	}

	// Verify config hint is shown
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "increase timeout in") || !strings.Contains(snapshot, "config.toml") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected to see configuration hint about increasing timeout")
	}

	// Test passed
	t.Log("Timeout error display test passed")
}

// TestHTTPErrorDisplay tests that HTTP errors show status code and server message
func TestHTTPErrorDisplay(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Create a mock server that returns various HTTP errors
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			// Version endpoint works
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"Version":"v2.8.0+test"}`)
		} else if r.URL.Path == "/api/v1/applications" {
			// Return 502 Bad Gateway
			w.WriteHeader(http.StatusBadGateway)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error":"Bad Gateway","message":"The server received an invalid response from the upstream server","code":502}`)
		}
	}))
	t.Cleanup(srv.Close)

	// Set up workspace and ArgoCD config
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Write config pointing to our mock server
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Start argonaut
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for error to appear
	if !tf.WaitForPlain("502", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see '502' status code")
	}

	// Verify error message is shown
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "Bad Gateway") || !strings.Contains(snapshot, "invalid response") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected to see server error message")
	}

	// Test passed
	t.Log("HTTP 502 error display test passed")
}

// Test503ServiceUnavailable tests 503 error display
func TestHTTP503ErrorDisplay(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Create a mock server that returns 503
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/version" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"Version":"v2.8.0+test"}`)
		} else {
			// Return 503 Service Unavailable
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"error":"Service Unavailable","message":"ArgoCD server is temporarily unavailable. Please try again later.","code":503}`)
		}
	}))
	t.Cleanup(srv.Close)

	// Set up workspace and ArgoCD config
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Start argonaut
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for error to appear
	if !tf.WaitForPlain("503", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see '503' status code")
	}

	// Verify error details
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "Service Unavailable") || !strings.Contains(snapshot, "temporarily unavailable") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected to see service unavailable message")
	}

	// Test passed
	t.Log("HTTP 503 error display test passed")
}