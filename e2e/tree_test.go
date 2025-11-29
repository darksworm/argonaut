//go:build e2e && unix

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTreeViewOpensAndReturns(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServer()
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

	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Open resources (tree) for demo
	_ = tf.Send(":")
	if !tf.WaitForPlain("│ > ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Expect Application root and at least a Deployment entry
	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}
	if !tf.WaitForPlain("Deployment [", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("deployment node not shown")
	}

	// Return to clusters
	_ = tf.Send("q")
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not return to main view")
	}
}

// MockArgoServerWithRichTree creates a server with multiple resource types for filtering
func MockArgoServerWithRichTree() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		// Rich tree with multiple resource types: Deployment, Service, ConfigMap, Pod
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"nginx-deployment","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced","health":{"status":"Healthy"}},
			{"kind":"ReplicaSet","name":"nginx-rs-abc123","namespace":"default","version":"v1","group":"apps","uid":"rs-1","status":"Synced","health":{"status":"Healthy"},"parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"nginx-deployment","namespace":"default","group":"apps","version":"v1"}]},
			{"kind":"Pod","name":"nginx-pod-xyz789","namespace":"default","version":"v1","uid":"pod-1","status":"Running","health":{"status":"Healthy"},"parentRefs":[{"uid":"rs-1","kind":"ReplicaSet","name":"nginx-rs-abc123","namespace":"default","group":"apps","version":"v1"}]},
			{"kind":"Service","name":"nginx-service","namespace":"default","version":"v1","uid":"svc-1","status":"Synced","health":{"status":"Healthy"}},
			{"kind":"ConfigMap","name":"nginx-config","namespace":"default","version":"v1","uid":"cm-1","status":"Synced"}
		]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}` + "\n"))
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

func TestTreeViewFilter(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithRichTree()
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

	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Open resources (tree) for demo
	_ = tf.Send(":")
	if !tf.WaitForPlain("│ > ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Expect Application root
	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}

	// Verify all resources are visible
	if !tf.WaitForPlain("Deployment [", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("deployment not shown")
	}
	if !tf.WaitForPlain("Service [", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("service not shown")
	}

	// Test 1: Enter search mode with /
	_ = tf.Send("/")
	if !tf.WaitForPlain("Search", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("search bar not shown after pressing /")
	}

	// Test 2: Type filter query "pod" and press Enter
	_ = tf.Send("pod")
	time.Sleep(200 * time.Millisecond) // Allow real-time filtering to update
	_ = tf.Enter()

	// Should show match count in status line
	if !tf.WaitForPlain("matches", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("match count not shown in status")
	}

	// Test 3: Verify "Pod" is highlighted with background color
	// The match should have yellow/warning background applied specifically to Pod
	rawSnap := tf.Snapshot()
	// Look for "Pod" with yellow background - the ANSI sequence should be immediately before "Pod"
	// RGB true color format: \x1b[...;48;2;224;175;104m followed by "Pod"
	// 256-color format: \x1b[...;48;5;179m followed by "Pod" (used when COLORTERM != truecolor)
	// Basic 16-color format: \x1b[103m followed by "Pod"
	hasPodHighlighted := strings.Contains(rawSnap, "48;2;224;175;104mPod") ||
		strings.Contains(rawSnap, "48;5;179mPod") ||
		strings.Contains(rawSnap, "[103mPod")
	if !hasPodHighlighted {
		t.Log("Raw snapshot excerpt (looking for Pod highlight):")
		t.Log(rawSnap[max(0, len(rawSnap)-2000):])
		t.Fatal("Pod should have yellow background highlight")
	}

	// Also verify that Service is NOT highlighted (it doesn't match "pod")
	// Service should not have yellow background
	hasServiceHighlighted := strings.Contains(rawSnap, "48;2;224;175;104mService") ||
		strings.Contains(rawSnap, "48;5;179mService") ||
		strings.Contains(rawSnap, "[103mService")
	if hasServiceHighlighted {
		t.Fatal("Service should NOT be highlighted - it doesn't match 'pod'")
	}

	// Test 4: Press n to navigate to next match (should work since we have match)
	_ = tf.Send("n")
	time.Sleep(100 * time.Millisecond)

	// Test 5: Exit tree view
	_ = tf.Send("q")
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not return to main view")
	}
}

func TestTreeViewFilterNoMatches(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithRichTree()
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

	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Open resources (tree) for demo
	_ = tf.Send(":")
	if !tf.WaitForPlain("│ > ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}

	// Enter search mode and type a non-matching query
	_ = tf.Send("/")
	if !tf.WaitForPlain("Search", 2*time.Second) {
		t.Fatal("search bar not shown")
	}

	_ = tf.Send("nonexistent")
	time.Sleep(200 * time.Millisecond)
	_ = tf.Enter()

	// Should show "no matches" in status
	if !tf.WaitForPlain("no matches", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("'no matches' not shown in status")
	}

	// Exit with q
	_ = tf.Send("q")
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("did not return to main view")
	}
}

func TestTreeViewFilterEscapeClearsFilter(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithRichTree()
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

	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Open resources (tree) for demo
	_ = tf.Send(":")
	if !tf.WaitForPlain("│ > ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}

	// Enter search, type query, then Escape without Enter
	_ = tf.Send("/")
	if !tf.WaitForPlain("Search", 2*time.Second) {
		t.Fatal("search bar not shown")
	}
	_ = tf.Send("pod")
	time.Sleep(200 * time.Millisecond)

	// Press Escape to cancel search - should close search bar
	_ = tf.Send("\x1b") // Escape key

	// Wait for search bar to close (search bar gone means filter cancelled)
	// The status should show <tree> without match count after escape
	time.Sleep(300 * time.Millisecond)

	// Verify search bar is closed by checking we're back in tree view mode
	// (typing q should exit tree view, not type 'q' in search)
	_ = tf.Send("q")
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not return to main view - escape may not have closed search")
	}
}
