//go:build e2e && unix

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockArgoServerSlow creates a mock server where /api/v1/applications
// responds after the given delay. Used to verify that request_timeout
// config actually controls the deadline.
func MockArgoServerSlow(appDelay time.Duration) (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(appDelay):
			// Respond after delay
		case <-r.Context().Done():
			// Client gave up
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, _ := w.(http.Flusher)
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// TestConfiguredTimeoutRespected verifies that request_timeout config
// actually controls the API call deadline. A slow server that responds
// after 3s should:
// - Time out when request_timeout = "1s"
// - Succeed when request_timeout = "5s"
func TestConfiguredTimeoutRespected(t *testing.T) {
	t.Parallel()

	// Server takes 3s to respond to /api/v1/applications
	srv, err := MockArgoServerSlow(3 * time.Second)
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	t.Run("short timeout causes error", func(t *testing.T) {
		t.Parallel()
		tf := NewTUITest(t)
		t.Cleanup(tf.Cleanup)
		tf.requestTimeout = "1s"

		cfgPath, err := tf.SetupWorkspace()
		if err != nil {
			t.Fatalf("setup workspace: %v", err)
		}
		if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
			t.Fatalf("write config: %v", err)
		}

		if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
			t.Fatalf("start app: %v", err)
		}

		// With 1s timeout and 3s server delay, should see a timeout/error
		if !tf.WaitForPlain("timed out", 6*time.Second) {
			snap := tf.SnapshotPlain()
			// Also accept connection-related errors since the timeout might
			// manifest as different error types depending on timing
			if !strings.Contains(snap, "Error") && !strings.Contains(snap, "error") {
				t.Log("Snapshot:", snap)
				t.Fatal("expected timeout error with 1s request_timeout and 3s server delay")
			}
		}

		// Verify the error message shows the actual configured timeout (1s), not a hardcoded value
		snap := tf.SnapshotPlain()
		if strings.Contains(snap, "after 10s") || strings.Contains(snap, "after 5s") {
			t.Log("Snapshot:", snap)
			t.Fatal("error message shows hardcoded timeout instead of configured 1s")
		}
	})

	t.Run("sufficient timeout loads apps", func(t *testing.T) {
		t.Parallel()
		tf := NewTUITest(t)
		t.Cleanup(tf.Cleanup)
		tf.requestTimeout = "5s"

		cfgPath, err := tf.SetupWorkspace()
		if err != nil {
			t.Fatalf("setup workspace: %v", err)
		}
		if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
			t.Fatalf("write config: %v", err)
		}

		if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
			t.Fatalf("start app: %v", err)
		}

		// With 5s timeout and 3s server delay, app should load successfully.
		// Default view is clusters, so look for the cluster name from app data.
		if !tf.WaitForPlain("cluster-a", 8*time.Second) {
			snap := tf.SnapshotPlain()
			t.Log("Snapshot:", snap)
			t.Fatal("expected apps to load with 5s timeout")
		}
	})
}
