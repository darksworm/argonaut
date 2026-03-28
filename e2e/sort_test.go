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
	// Resource tree for app-charlie:
	//   broken-deploy  Deployment  Degraded/OutOfSync
	//   mid-cfg        ConfigMap   Progressing/Unknown
	//   alpha-svc      Service     Healthy/Synced   (same health+sync as stable-svc; name "alpha" < "stable" for tiebreak)
	//   stable-svc     Service     Healthy/Synced
	mux.HandleFunc("/api/v1/applications/app-charlie/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"broken-deploy","namespace":"default","version":"v1","group":"apps","uid":"dep-c","health":{"status":"Degraded"},"status":"OutOfSync"},
			{"kind":"Service","name":"stable-svc","namespace":"default","version":"v1","uid":"svc-c","health":{"status":"Healthy"},"status":"Synced"},
			{"kind":"ConfigMap","name":"mid-cfg","namespace":"default","version":"v1","uid":"cm-c","health":{"status":"Progressing"},"status":"Unknown"},
			{"kind":"Service","name":"alpha-svc","namespace":"default","version":"v1","uid":"svc-a","health":{"status":"Healthy"},"status":"Synced"}
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

	if !tf.WaitForScreen("Application [app-charlie]", 5*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("tree view not loaded")
	}
	if !tf.WaitForScreen("broken-deploy", 5*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("broken-deploy not visible in tree")
	}
	if !tf.WaitForScreen("alpha-svc", 5*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("alpha-svc not visible in tree")
	}

	// screenSnapshot captures the current rendered screen and returns a helper
	// that finds the first line containing a given substring.
	// Using Screen() here is essential: SnapshotPlain() returns accumulated
	// ring-buffer output, so linePosition would find nodes from earlier renders
	// rather than the current (sorted) frame.
	screenSnapshot := func() (string, func(string) int) {
		snap := tf.Screen()
		return snap, func(substr string) int { return linePosition(snap, substr) }
	}

	// --- sort health asc ---
	// Expected order: broken-deploy (Degraded) < mid-cfg (Progressing) <
	//                 alpha-svc (Healthy, name tiebreak asc) < stable-svc (Healthy)
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort health asc")
	_ = tf.Enter()

	time.Sleep(500 * time.Millisecond)
	snap, pos := screenSnapshot()

	degradedPos := pos("broken-deploy")
	progressingPos := pos("mid-cfg")
	alphaAscPos := pos("alpha-svc")
	stableAscPos := pos("stable-svc")

	if degradedPos < 0 || progressingPos < 0 || alphaAscPos < 0 || stableAscPos < 0 {
		t.Logf("screen:\n%s", snap)
		t.Fatal("sort health asc: expected all four resources on screen")
	}
	if degradedPos >= progressingPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort health asc: expected Degraded/broken-deploy (line %d) before Progressing/mid-cfg (line %d)", degradedPos, progressingPos)
	}
	if progressingPos >= alphaAscPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort health asc: expected Progressing/mid-cfg (line %d) before Healthy/alpha-svc (line %d)", progressingPos, alphaAscPos)
	}
	// Name tiebreak (asc): alpha-svc < stable-svc because "alpha" < "stable"
	if alphaAscPos >= stableAscPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort health asc name tiebreak: expected alpha-svc (line %d) before stable-svc (line %d)", alphaAscPos, stableAscPos)
	}

	// --- sort sync asc ---
	// Expected order: broken-deploy (OutOfSync) < mid-cfg (Unknown) <
	//                 alpha-svc (Synced, name tiebreak asc) < stable-svc (Synced)
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort sync asc")
	_ = tf.Enter()

	time.Sleep(500 * time.Millisecond)
	snap, pos = screenSnapshot()

	outOfSyncPos := pos("broken-deploy") // OutOfSync
	unknownPos := pos("mid-cfg")         // Unknown
	alphaSyncPos := pos("alpha-svc")     // Synced (name tiebreak asc: alpha < stable)
	stableSyncPos := pos("stable-svc")   // Synced

	if outOfSyncPos < 0 || unknownPos < 0 || alphaSyncPos < 0 || stableSyncPos < 0 {
		t.Logf("screen:\n%s", snap)
		t.Fatal("sort sync asc: expected all four resources on screen")
	}
	if outOfSyncPos >= unknownPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort sync asc: expected OutOfSync/broken-deploy (line %d) before Unknown/mid-cfg (line %d)", outOfSyncPos, unknownPos)
	}
	if unknownPos >= alphaSyncPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort sync asc: expected Unknown/mid-cfg (line %d) before Synced/alpha-svc (line %d)", unknownPos, alphaSyncPos)
	}
	// Name tiebreak (asc): alpha-svc < stable-svc because "alpha" < "stable"
	if alphaSyncPos >= stableSyncPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort sync asc name tiebreak: expected alpha-svc (line %d) before stable-svc (line %d)", alphaSyncPos, stableSyncPos)
	}

	// --- sort health desc ---
	// Expected order: stable-svc (Healthy, desc name tiebreak: "stable" > "alpha") <
	//                 alpha-svc (Healthy) < mid-cfg (Progressing) < broken-deploy (Degraded)
	// The name tiebreak follows the sort direction: desc ⇒ larger name first.
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sort health desc")
	_ = tf.Enter()

	time.Sleep(500 * time.Millisecond)
	snap, pos = screenSnapshot()

	stableDescPos := pos("stable-svc") // Healthy (desc name tiebreak: "stable" > "alpha" → first)
	alphaDescPos := pos("alpha-svc")   // Healthy (second)
	progressingPos = pos("mid-cfg")    // Progressing
	degradedPos = pos("broken-deploy") // Degraded (last)

	if stableDescPos < 0 || alphaDescPos < 0 || progressingPos < 0 || degradedPos < 0 {
		t.Logf("screen:\n%s", snap)
		t.Fatal("sort health desc: expected all four resources on screen")
	}
	// Name tiebreak (desc): stable-svc before alpha-svc because "stable" > "alpha"
	if stableDescPos >= alphaDescPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort health desc name tiebreak: expected stable-svc (line %d) before alpha-svc (line %d)", stableDescPos, alphaDescPos)
	}
	if alphaDescPos >= progressingPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort health desc: expected Healthy/alpha-svc (line %d) before Progressing/mid-cfg (line %d)", alphaDescPos, progressingPos)
	}
	if progressingPos >= degradedPos {
		t.Logf("screen:\n%s", snap)
		t.Errorf("sort health desc: expected Progressing/mid-cfg (line %d) before Degraded/broken-deploy (line %d)", progressingPos, degradedPos)
	}

	// Exit tree view — confirm apps list is still functional
	_ = tf.Send("q")
	if !tf.WaitForScreen("app-alpha", 5*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("did not return to apps view")
	}
}
