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
		_, _ = w.Write([]byte(wrapListResponse(`[
			{"metadata":{"name":"app-charlie","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Degraded"}}},
			{"metadata":{"name":"app-alpha","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
			{"metadata":{"name":"app-bravo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Unknown"},"health":{"status":"Progressing"}}}
		]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		if shouldSendEvent(r, "demo") {
			_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"app-charlie","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Degraded"}}}}}`)))
		}
		if fl != nil {
			fl.Flush()
		}
	})
	// Resource tree for app-charlie: Deployment (Degraded/OutOfSync), Service (Healthy/Synced), ConfigMap (Progressing/Unknown)
	mux.HandleFunc("/api/v1/applications/app-charlie/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"broken-deploy","namespace":"default","version":"v1","group":"apps","uid":"dep-c","health":{"status":"Degraded"},"status":"OutOfSync"},
			{"kind":"Service","name":"stable-svc","namespace":"default","version":"v1","uid":"svc-c","health":{"status":"Healthy"},"status":"Synced"},
			{"kind":"ConfigMap","name":"mid-cfg","namespace":"default","version":"v1","uid":"cm-c","health":{"status":"Progressing"},"status":"Unknown"}
		]}`))
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

// linePosition returns the line index of the first line containing the substring,
// or -1 if not found.
func linePosition(snapshot, substr string) int {
	for i, line := range strings.Split(snapshot, "\n") {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

// TestSortInTreeView verifies that :sort health/sync commands have visual effect
// on resources inside the resource tree view (ViewTree).
func TestSortInTreeView(t *testing.T) {
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

	// Wait for initial load
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Navigate to apps view
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForPlain("app-alpha", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("apps not loaded")
	}

	// Open the resource tree for app-charlie
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("resources app-charlie")
	_ = tf.Enter()

	// Wait for tree view to load
	if !tf.WaitForPlain("Application [app-charlie]", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}
	if !tf.WaitForPlain("broken-deploy", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree resources not visible")
	}

	// --- sort health asc: Degraded < Progressing < Healthy ---
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort health asc")
	_ = tf.Enter()

	time.Sleep(500 * time.Millisecond)
	snapshot := tf.SnapshotPlain()

	degradedPos := linePosition(snapshot, "broken-deploy")
	progressingPos := linePosition(snapshot, "mid-cfg")
	healthyPos := linePosition(snapshot, "stable-svc")

	if degradedPos < 0 || progressingPos < 0 || healthyPos < 0 {
		t.Logf("snapshot:\n%s", snapshot)
		t.Fatal("expected all three resources in tree snapshot")
	}
	if degradedPos >= progressingPos {
		t.Logf("snapshot:\n%s", snapshot)
		t.Errorf("sort health asc: expected Degraded (line %d) before Progressing (line %d)", degradedPos, progressingPos)
	}
	if progressingPos >= healthyPos {
		t.Logf("snapshot:\n%s", snapshot)
		t.Errorf("sort health asc: expected Progressing (line %d) before Healthy (line %d)", progressingPos, healthyPos)
	}

	// --- sort sync asc: OutOfSync < Unknown < Synced ---
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort sync asc")
	_ = tf.Enter()

	time.Sleep(500 * time.Millisecond)
	snapshot = tf.SnapshotPlain()

	outOfSyncPos := linePosition(snapshot, "broken-deploy")  // OutOfSync
	unknownPos := linePosition(snapshot, "mid-cfg")          // Unknown
	syncedPos := linePosition(snapshot, "stable-svc")        // Synced

	if outOfSyncPos < 0 || unknownPos < 0 || syncedPos < 0 {
		t.Logf("snapshot:\n%s", snapshot)
		t.Fatal("expected all three resources in tree snapshot")
	}
	if outOfSyncPos >= unknownPos {
		t.Logf("snapshot:\n%s", snapshot)
		t.Errorf("sort sync asc: expected OutOfSync (line %d) before Unknown (line %d)", outOfSyncPos, unknownPos)
	}
	if unknownPos >= syncedPos {
		t.Logf("snapshot:\n%s", snapshot)
		t.Errorf("sort sync asc: expected Unknown (line %d) before Synced (line %d)", unknownPos, syncedPos)
	}

	// --- sort health desc: Healthy < Progressing < Degraded ---
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort health desc")
	_ = tf.Enter()

	time.Sleep(500 * time.Millisecond)
	snapshot = tf.SnapshotPlain()

	healthyPos = linePosition(snapshot, "stable-svc")
	progressingPos = linePosition(snapshot, "mid-cfg")
	degradedPos = linePosition(snapshot, "broken-deploy")

	if healthyPos < 0 || progressingPos < 0 || degradedPos < 0 {
		t.Logf("snapshot:\n%s", snapshot)
		t.Fatal("expected all three resources in tree snapshot")
	}
	if healthyPos >= progressingPos {
		t.Logf("snapshot:\n%s", snapshot)
		t.Errorf("sort health desc: expected Healthy (line %d) before Progressing (line %d)", healthyPos, progressingPos)
	}
	if progressingPos >= degradedPos {
		t.Logf("snapshot:\n%s", snapshot)
		t.Errorf("sort health desc: expected Progressing (line %d) before Degraded (line %d)", progressingPos, degradedPos)
	}

	// Exit tree view — confirm apps list is still functional
	_ = tf.Send("q")
	if !tf.WaitForPlain("app-alpha", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not return to apps view")
	}
}
