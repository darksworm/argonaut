//go:build e2e && unix

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockArgoServerMultipleApps creates a server with multiple apps for sorting tests
func MockArgoServerMultipleApps() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		// Return multiple apps with different sync/health statuses for sorting tests
		// Names: app-charlie, app-alpha, app-bravo (out of alphabetical order)
		// Sync: OutOfSync, Synced, Unknown
		// Health: Degraded, Healthy, Progressing
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[
			{"metadata":{"name":"app-charlie","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Degraded"}}},
			{"metadata":{"name":"app-alpha","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
			{"metadata":{"name":"app-bravo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Unknown"},"health":{"status":"Progressing"}}}
		]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"app-charlie","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Degraded"}}}}}` + "\n"))
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// getFirstAppInList returns the name of the first app found in the snapshot
func getFirstAppInList(snapshot string) string {
	lines := strings.Split(snapshot, "\n")
	for _, line := range lines {
		// Look for app names in the output
		if strings.Contains(line, "app-alpha") {
			// Check position relative to other apps
			alphaPos := strings.Index(line, "app-alpha")
			bravoPos := strings.Index(line, "app-bravo")
			charliePos := strings.Index(line, "app-charlie")

			if alphaPos >= 0 && (bravoPos < 0 || alphaPos < bravoPos) && (charliePos < 0 || alphaPos < charliePos) {
				return "app-alpha"
			}
		}
		if strings.Contains(line, "app-bravo") {
			bravoPos := strings.Index(line, "app-bravo")
			alphaPos := strings.Index(line, "app-alpha")
			charliePos := strings.Index(line, "app-charlie")

			if bravoPos >= 0 && (alphaPos < 0 || bravoPos < alphaPos) && (charliePos < 0 || bravoPos < charliePos) {
				return "app-bravo"
			}
		}
		if strings.Contains(line, "app-charlie") {
			charliePos := strings.Index(line, "app-charlie")
			alphaPos := strings.Index(line, "app-alpha")
			bravoPos := strings.Index(line, "app-bravo")

			if charliePos >= 0 && (alphaPos < 0 || charliePos < alphaPos) && (bravoPos < 0 || charliePos < bravoPos) {
				return "app-charlie"
			}
		}
	}
	return ""
}

func TestSortCommand(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerMultipleApps()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

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

	// Wait for app to load
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Navigate to apps view
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	// Wait for apps to load
	if !tf.WaitForPlain("app-alpha", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("apps not loaded")
	}

	// Default sort is by name ascending - verify ascending indicator exists
	time.Sleep(500 * time.Millisecond)
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "▲") {
		t.Log(snapshot)
		t.Fatal("expected ascending sort indicator (▲)")
	}

	// Sort by name descending
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort name desc")
	_ = tf.Enter()

	// Wait and verify sort indicator changes to descending
	time.Sleep(500 * time.Millisecond)
	snapshot = tf.SnapshotPlain()
	if !strings.Contains(snapshot, "▼") {
		t.Log(snapshot)
		t.Fatal("expected descending sort indicator (▼)")
	}

	// Sort by sync status
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort sync asc")
	_ = tf.Enter()

	// Wait for sort to apply - should show ascending indicator
	time.Sleep(500 * time.Millisecond)
	snapshot = tf.SnapshotPlain()
	if !strings.Contains(snapshot, "▲") {
		t.Log(snapshot)
		t.Fatal("expected ascending sort indicator after sorting by sync")
	}

	// Sort by health status descending
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort health desc")
	_ = tf.Enter()

	// Wait for sort to apply - should show descending indicator
	time.Sleep(500 * time.Millisecond)
	snapshot = tf.SnapshotPlain()
	if !strings.Contains(snapshot, "▼") {
		t.Log(snapshot)
		t.Fatal("expected descending sort indicator after sorting by health")
	}
}

// TestSortRequiresDirection verifies that :sort requires both field and direction
func TestSortRequiresDirection(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerMultipleApps()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

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

	// Wait for app to load and navigate to apps
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForPlain("app-alpha", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("apps not loaded")
	}

	// Default is name asc - should have ▲
	time.Sleep(500 * time.Millisecond)
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "▲") {
		t.Log(snapshot)
		t.Fatal("expected ascending indicator initially")
	}

	// Try to sort without direction - should show autocomplete suggestions
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort name")

	// Wait a moment for autocomplete to render
	time.Sleep(300 * time.Millisecond)
	snapshot = tf.SnapshotPlain()

	// Should show autocomplete suggestion for direction (asc or desc)
	// The autocomplete should suggest "sort name asc" or similar
	if !strings.Contains(snapshot, "asc") && !strings.Contains(snapshot, "desc") {
		t.Log(snapshot)
		t.Fatal("expected autocomplete to suggest direction (asc/desc)")
	}

	// Press Escape to cancel and verify sort unchanged
	_ = tf.Send("\x1b") // Escape
	time.Sleep(300 * time.Millisecond)

	// Sort should still be ascending (unchanged)
	snapshot = tf.SnapshotPlain()
	if !strings.Contains(snapshot, "▲") {
		t.Log(snapshot)
		t.Fatal("expected ascending indicator to remain unchanged after cancelled incomplete command")
	}
}
