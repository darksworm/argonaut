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
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
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
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`, "1000")))
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
		w.Header().Set("Content-Type", "text/event-stream")
		if shouldSendEvent(r, "demo") {
			_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`)))
		}
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
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
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
	if err := tf.OpenSearch(); err != nil {
		t.Log(tf.SnapshotPlain())
		t.Fatal(err)
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
	// The match should have yellow/warning background applied to the row containing Pod
	rawSnap := tf.Snapshot()

	// Find the line containing "Pod" and verify it has yellow background
	// Lipgloss v2 optimizes ANSI sequences, so the background may be set at the start
	// of the line rather than immediately before "Pod"
	var podLineHighlighted bool
	var serviceLineHighlighted bool
	for _, line := range strings.Split(rawSnap, "\n") {
		// Check for yellow background (warning color) on lines with Pod/Service
		hasYellowBG := strings.Contains(line, "48;2;224;175;104") ||
			strings.Contains(line, "48;5;179") ||
			strings.Contains(line, "[103m")

		if strings.Contains(line, "Pod") && strings.Contains(line, "nginx-pod") {
			podLineHighlighted = hasYellowBG
		}
		if strings.Contains(line, "Service") && strings.Contains(line, "nginx-service") {
			serviceLineHighlighted = hasYellowBG
		}
	}

	if !podLineHighlighted {
		t.Log("Raw snapshot excerpt (looking for Pod highlight):")
		t.Log(rawSnap[max(0, len(rawSnap)-2000):])
		t.Fatal("Pod row should have yellow background highlight")
	}

	// Also verify that Service is NOT highlighted (it doesn't match "pod")
	if serviceLineHighlighted {
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
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}

	// Enter search mode and type a non-matching query
	if err := tf.OpenSearch(); err != nil {
		t.Fatal(err)
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

// MockArgoServerWithResourceSyncStatus creates a server with resource sync status in tree view.
func MockArgoServerWithResourceSyncStatus() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// App with resources array containing sync statuses
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"},"resources":[{"group":"apps","kind":"Deployment","name":"nginx-deployment","namespace":"default","status":"OutOfSync","version":"v1"},{"group":"","kind":"Service","name":"nginx-service","namespace":"default","status":"Synced","version":"v1"}]}}]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications/demo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Single app endpoint returns resources for tree view loading
		_, _ = w.Write([]byte(`{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"},"resources":[{"group":"apps","kind":"Deployment","name":"nginx-deployment","namespace":"default","status":"OutOfSync","version":"v1"},{"group":"","kind":"Service","name":"nginx-service","namespace":"default","status":"Synced","version":"v1"}]}}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		// Tree with Deployment and Service - sync status comes from resources array
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"nginx-deployment","namespace":"default","version":"v1","group":"apps","uid":"dep-1","health":{"status":"Healthy"}},
			{"kind":"Service","name":"nginx-service","namespace":"default","version":"v1","uid":"svc-1","health":{"status":"Healthy"}}
		]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")

		// Send initial state (SSE format)
		if shouldSendEvent(r, "demo") {
			_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"},"resources":[{"group":"apps","kind":"Deployment","name":"nginx-deployment","namespace":"default","status":"OutOfSync","version":"v1"},{"group":"","kind":"Service","name":"nginx-service","namespace":"default","status":"Synced","version":"v1"}]}}}}`)))
		}
		if fl != nil {
			fl.Flush()
		}

		// Keep stream open for a bit
		time.Sleep(3 * time.Second)
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// TestTreeViewResourceSyncStatus verifies that individual resources (Deployments, Services)
// display their sync status from Application.status.resources in the tree view.
func TestTreeViewResourceSyncStatus(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithResourceSyncStatus()
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

	// Open resources (tree) for demo app
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Wait for tree view to load
	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}

	// Verify Deployment shows OutOfSync status from resources array
	if !tf.WaitForPlain("Deployment [default/nginx-deployment] (Healthy, OutOfSync)", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Deployment sync status not shown correctly")
	}

	// Verify Service shows Synced status from resources array
	if !tf.WaitForPlain("Service [default/nginx-service] (Healthy, Synced)", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Service sync status not shown correctly")
	}

	// Exit tree view
	_ = tf.Send("q")
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
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
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}

	// Enter search, type query, then Escape without Enter
	if err := tf.OpenSearch(); err != nil {
		t.Fatal(err)
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
