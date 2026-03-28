//go:build e2e && unix

package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func waitUntil(t *testing.T, cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// Ensure single-app sync posts to the correct endpoint with expected body
func TestSyncSingleApp(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerSync("valid-token")
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Navigate deterministically via commands to apps
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("projects not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo2", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Navigate to the second app (demo2) using 'j' to move down
	_ = tf.Send("j")

	// Trigger sync for the highlighted app (now demo2, not the first demo)
	_ = tf.Send("s")  // open confirm
	_ = tf.Send("\r") // Enter to confirm "Yes"

	// Wait for one sync call
	if !waitUntil(t, func() bool { return rec.len() == 1 }, 2*time.Second) {
		t.Fatalf("expected 1 sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}
	call := rec.Calls[0]
	if call.Name != "demo2" {
		t.Fatalf("expected sync for 'demo2', got %q", call.Name)
	}
	// Body should be JSON with prune flag (default false)
	var body map[string]any
	if err := json.Unmarshal([]byte(call.Body), &body); err != nil {
		t.Fatalf("invalid body json: %v", err)
	}
	if v, ok := body["prune"].(bool); !ok || v {
		t.Fatalf("expected prune=false in body, got %v", body["prune"])
	}
}

// Ensure multi-app sync posts for each selected app
func TestSyncMultipleApps(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerSync("valid-token")
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// To apps deterministically
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("projects not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo2", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Select two apps: space (demo), down, space (demo2)
	_ = tf.Send(" ")
	_ = tf.Send("j")
	_ = tf.Send(" ")

	// Open confirm and accept
	_ = tf.Send("s")
	_ = tf.Send("\r")

	// Expect two sync calls to /applications/<name>/sync
	if !waitUntil(t, func() bool { return rec.len() == 2 }, 2*time.Second) {
		t.Fatalf("expected 2 sync calls, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}
	names := map[string]bool{}
	for _, c := range rec.Calls {
		names[c.Name] = true
	}
	if !names["demo"] || !names["demo2"] {
		t.Fatalf("expected sync calls for demo and demo2, got: %+v", names)
	}
}

// Test for the bug where selecting the last app and typing :sync shows a popup asking to sync the first app
func TestSyncLastAppShowsCorrectConfirmation(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerSync("valid-token")
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Navigate deterministically via commands to apps
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("projects not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo2", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Navigate to the last app (demo2) by pressing 'j' to move down from first app (demo)
	_ = tf.Send("j")

	// Type `:sync` instead of using the 's' key
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sync")
	_ = tf.Enter()

	// The modal should show demo2 as the target, not demo
	// Check that the confirmation modal displays the correct app name (demo2)
	if !tf.WaitForPlain("demo2", 500*time.Millisecond) {
		// If we don't see demo2 in the modal, this indicates the bug
		snapshot := tf.SnapshotPlain()
		if strings.Contains(snapshot, "demo") && !strings.Contains(snapshot, "demo2") {
			t.Fatalf("BUG REPRODUCED: Modal shows 'demo' instead of 'demo2' when last app is selected\nSnapshot:\n%s", snapshot)
		}
		t.Fatalf("Expected sync confirmation modal to show 'demo2', but it's not visible\nSnapshot:\n%s", snapshot)
	}

	// Confirm the sync to verify it targets the correct app
	_ = tf.Send("\r") // Enter to confirm "Yes"

	// Wait for one sync call
	if !waitUntil(t, func() bool { return rec.len() == 1 }, 2*time.Second) {
		t.Fatalf("expected 1 sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}
	call := rec.Calls[0]
	if call.Name != "demo2" {
		t.Fatalf("BUG CONFIRMED: expected sync for 'demo2', got %q - the wrong app was synced!", call.Name)
	}
}

// TestSyncSingleApp_WithNamespace_PassesNamespaceToAPI verifies that when an app lives in a
// non-default namespace, the sync POST request includes ?appNamespace= so Argo CD targets
// the correct application.
func TestSyncSingleApp_WithNamespace_PassesNamespaceToAPI(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	var mu sync.Mutex
	var capturedReqURI string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	// Single app in namespace "team-b" (not the default "argocd")
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[
			{"metadata":{"name":"my-app","namespace":"team-b"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}
		]`, "1000")))
	})
	// Capture the sync request URI (path + query string)
	mux.HandleFunc("/api/v1/applications/my-app/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		mu.Lock()
		capturedReqURI = r.URL.RequestURI()
		mu.Unlock()
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	srv := httptest.NewServer(mux)
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

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	// Navigate to apps
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForPlain("my-app", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("my-app not visible in apps view")
	}

	// Trigger sync for my-app
	_ = tf.Send("s")

	if !tf.WaitForPlain("my-app", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("sync confirmation modal did not appear")
	}

	// Confirm sync
	_ = tf.Send("\r")

	// Wait for the sync call to be captured
	if !waitUntil(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return capturedReqURI != ""
	}, 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("no sync API call was made")
	}

	mu.Lock()
	uri := capturedReqURI
	mu.Unlock()

	if !strings.Contains(uri, "appNamespace=team-b") {
		t.Errorf("expected sync URL to contain 'appNamespace=team-b', got: %q", uri)
	}
}
