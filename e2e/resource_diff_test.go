//go:build e2e && unix

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---- Mock Server Helpers ----

// MockArgoServerWithResourceDiffs creates a server with resources that have diffs
func MockArgoServerWithResourceDiffs() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		// Resource tree with Deployment and ConfigMap
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"demo-deploy","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"OutOfSync"},
			{"kind":"ConfigMap","name":"demo-config","namespace":"default","version":"v1","group":"","uid":"cm-1","status":"Synced"}
		]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Deployment has a diff (different replica count)
		deployLive := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"demo-deploy","namespace":"default"},"spec":{"replicas":1}}`
		deployDesired := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"demo-deploy","namespace":"default"},"spec":{"replicas":3}}`
		// ConfigMap is synced (same content)
		cmLive := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"demo-config","namespace":"default"},"data":{"key":"value"}}`
		cmDesired := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"demo-config","namespace":"default"},"data":{"key":"value"}}`
		_, _ = w.Write([]byte(`{"items":[` +
			`{"kind":"Deployment","namespace":"default","name":"demo-deploy","group":"apps","normalizedLiveState":` + jsonEscape(deployLive) + `,"predictedLiveState":` + jsonEscape(deployDesired) + `},` +
			`{"kind":"ConfigMap","namespace":"default","name":"demo-config","normalizedLiveState":` + jsonEscape(cmLive) + `,"predictedLiveState":` + jsonEscape(cmDesired) + `}` +
			`]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		if shouldSendEvent(r, "demo") {
			_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}}}`)))
		}
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// MockArgoServerWithSyncedResources creates a server where all resources are synced (no diffs)
func MockArgoServerWithSyncedResources() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"demo-deploy","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced"},
			{"kind":"ConfigMap","name":"demo-config","namespace":"default","version":"v1","group":"","uid":"cm-1","status":"Synced"}
		]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// All resources are synced (identical live and desired states)
		deployState := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"demo-deploy","namespace":"default"},"spec":{"replicas":1}}`
		cmState := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"demo-config","namespace":"default"},"data":{"key":"value"}}`
		_, _ = w.Write([]byte(`{"items":[` +
			`{"kind":"Deployment","namespace":"default","name":"demo-deploy","group":"apps","normalizedLiveState":` + jsonEscape(deployState) + `,"predictedLiveState":` + jsonEscape(deployState) + `},` +
			`{"kind":"ConfigMap","namespace":"default","name":"demo-config","normalizedLiveState":` + jsonEscape(cmState) + `,"predictedLiveState":` + jsonEscape(cmState) + `}` +
			`]}`))
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

// createMockLess creates a mock less command that captures the input and exits quickly
func createMockLess(t *testing.T, workspace string) (scriptPath, inputFile string) {
	t.Helper()

	binDir := filepath.Join(workspace, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	scriptPath = filepath.Join(binDir, "less")
	inputFile = filepath.Join(workspace, "less_input.txt")

	// Create mock less that captures stdin to a file and displays brief output
	script := fmt.Sprintf(`#!/bin/sh
# Mock less - captures stdin and exits
cat > %q
printf '\033[2J\033[H'
printf 'Mock less diff viewer\n'
sleep 0.05
exit 0
`, inputFile)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create mock less: %v", err)
	}

	return scriptPath, inputFile
}

// ---- Test Cases ----

// TestResourceDiff_SyncedResource_ShowsNoDiffModal verifies that pressing d on a
// synced resource (no changes) shows the "No Diff" modal.
func TestResourceDiff_SyncedResource_ShowsNoDiffModal(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithSyncedResources()
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

	// Wait for initial view
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Wait for tree to load
	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment (synced resource) and request diff in one
	// send — the PTY preserves keystroke order to the app.
	_ = tf.Send("jd")

	// Wait for "No differences found" modal - the modal shows when resource has no changes
	if !tf.WaitForPlain("No differences found", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("no diff modal not shown for synced resource")
	}

	// Dismiss modal and verify we're back in tree view.
	_ = tf.Enter()
	if !tf.WaitForPlain("Application [demo]", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("should be back in tree view after dismissing modal")
	}
}

// TestResourceDiff_ApplicationNode_AllSyncedShowsNoDiff verifies that
// pressing `d` on the Application root node when all resources are
// in sync shows the "No differences found" modal — the application-level
// diff aggregates over all child resources, and with everything synced
// the aggregate has no diff content.
//
// (Renamed from `_ShowsFullAppDiff` — the original name suggested this
// test exercised the full-diff path, but the mock has all-synced
// resources, so the only path actually tested is the no-diff modal.)
func TestResourceDiff_ApplicationNode_AllSyncedShowsNoDiff(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithSyncedResources()
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

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Cursor on Application root; press d to request a full app-level diff.
	_ = tf.Send("d")

	if !tf.WaitForPlain("No differences found", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("no diff modal not shown for synced application")
	}
}

// TestResourceDiff_OutOfSyncResource_OpensDiffViewer verifies that pressing d on
// a resource with changes opens the diff viewer.
func TestResourceDiff_OutOfSyncResource_OpensDiffViewer(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithResourceDiffs()
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

	// Create mock less to capture diff output
	mockLess, inputFile := createMockLess(t, tf.workspace)

	// Prepend mock bin dir to PATH so our mock less is found first
	binDir := filepath.Dir(mockLess)
	origPath := os.Getenv("PATH")

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"PATH="+binDir+":"+origPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Wait for tree to load
	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Tree structure is:
	// 1. Application [demo]
	// 2. ConfigMap [default/demo-config] (Synced)
	// 3. Deployment [default/demo-deploy] (OutOfSync)
	// Navigate to Deployment (j j) and request diff (d) — PTY preserves order.
	_ = tf.Send("jjd")

	// Wait for mock less to display something
	if !tf.WaitForPlain("Mock less", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock less was not launched for diff")
	}

	// Verify diff content was passed to less
	diffContent, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("failed to read less input: %v", err)
	}

	// The diff should contain the replica change
	diffStr := string(diffContent)
	if !strings.Contains(diffStr, "replicas") {
		t.Errorf("diff should contain 'replicas' change, got: %s", diffStr)
	}
}

// TestResourceDiff_LoadingSpinner_Appears verifies that the loading spinner
// appears when fetching diff data.
func TestResourceDiff_LoadingSpinner_Appears(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Create a slow server to give time to see the loading spinner
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[{"kind":"Deployment","name":"demo-deploy","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"OutOfSync"}]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		// Slow response to ensure loading spinner is visible. 150ms is
		// plenty for at least one spinner frame (~80ms cadence), and we
		// poll for it below rather than sleeping a fixed amount.
		// httptest.Server.Close blocks on in-flight handlers, so this
		// also caps the test's worst-case duration.
		time.Sleep(150 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		deployState := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"demo-deploy","namespace":"default"},"spec":{"replicas":1}}`
		_, _ = w.Write([]byte(`{"items":[{"kind":"Deployment","namespace":"default","name":"demo-deploy","group":"apps","normalizedLiveState":` + jsonEscape(deployState) + `,"predictedLiveState":` + jsonEscape(deployState) + `}]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}}}`)))
		if fl != nil {
			fl.Flush()
		}
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

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment, then trigger the diff.
	_ = tf.Send("j")
	_ = tf.Send("d")

	// Poll for the diff-loading marker on the *current* screen (not the
	// cumulative SnapshotPlain — that buffer also contains the
	// "Connecting to Argo CD..." startup-spinner chars from earlier in
	// the run, which would make this assertion pass even if pressing 'd'
	// produced no spinner at all). The managed-resources mock above sleeps
	// 150ms, so the diff-loading spinner has a window to render.
	if !waitUntil(t, func() bool {
		s := tf.Screen()
		return strings.Contains(s, "Loading") ||
			strings.Contains(s, "⠋") ||
			strings.Contains(s, "⠙") ||
			strings.Contains(s, "⠹") ||
			strings.Contains(s, "⠸")
	}, 500*time.Millisecond) {
		t.Log(tf.Screen())
		t.Fatal("loading spinner did not appear before managed-resources response arrived")
	}
}

// TestResourceDiff_ResourceNotInDiffList_ShowsNoDiff verifies that pressing d on
// a resource that doesn't exist in the managed-resources response shows no diff.
func TestResourceDiff_ResourceNotInDiffList_ShowsNoDiff(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Create server where resource tree has resources but managed-resources is empty
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		// Tree has a Pod
		_, _ = w.Write([]byte(`{"nodes":[{"kind":"Pod","name":"demo-pod","namespace":"default","version":"v1","uid":"pod-1","status":"Synced"}]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		// But managed-resources doesn't include it (e.g., it's a child resource not directly managed)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`)))
		if fl != nil {
			fl.Flush()
		}
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

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Pod", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Pod and request diff (PTY preserves keystroke order).
	_ = tf.Send("jd")

	// Should show "No differences found" since resource isn't in managed-resources
	if !tf.WaitForPlain("No differences found", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("no diff modal not shown for unmanaged resource")
	}
}

// MockArgoServerWithAppOfApps creates a mock server simulating an app-of-apps setup.
// parent-app manages child-app as an Application CR resource.
// child-app's Application CR is OutOfSync (live vs desired configs differ).
func MockArgoServerWithAppOfApps() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	// Both parent-app and child-app are in the apps list
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(`[
			{"metadata":{"name":"parent-app","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}},
			{"metadata":{"name":"child-app","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}
		]`, "1000")))
	})
	// parent-app resource tree: contains child-app as an Application CR node
	mux.HandleFunc("/api/v1/applications/parent-app/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Application","name":"child-app","namespace":"argocd","version":"v1","group":"argoproj.io","uid":"child-1","status":"OutOfSync"}
		]}`))
	})
	// parent-app managed-resources: child-app Application CR has a diff
	mux.HandleFunc("/api/v1/applications/parent-app/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		childLive := `{"apiVersion":"argoproj.io/v1alpha1","kind":"Application","metadata":{"name":"child-app","namespace":"argocd"},"spec":{"source":{"repoURL":"https://github.com/org/repo","targetRevision":"main"}}}`
		childDesired := `{"apiVersion":"argoproj.io/v1alpha1","kind":"Application","metadata":{"name":"child-app","namespace":"argocd"},"spec":{"source":{"repoURL":"https://github.com/org/repo","targetRevision":"v2.0.0"}}}`
		_, _ = w.Write([]byte(`{"items":[` +
			`{"kind":"Application","group":"argoproj.io","namespace":"argocd","name":"child-app","normalizedLiveState":` + jsonEscape(childLive) + `,"predictedLiveState":` + jsonEscape(childDesired) + `}` +
			`]}`))
	})
	// child-app resource tree: contains a Deployment
	mux.HandleFunc("/api/v1/applications/child-app/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"child-deploy","namespace":"default","version":"v1","group":"apps","uid":"cdep-1","status":"Synced"}
		]}`))
	})
	// child-app managed-resources (synced, no diff needed for navigation test)
	mux.HandleFunc("/api/v1/applications/child-app/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		deployState := `{"apiVersion":"apps/v1","kind":"Deployment","metadata":{"name":"child-deploy","namespace":"default"},"spec":{"replicas":1}}`
		_, _ = w.Write([]byte(`{"items":[` +
			`{"kind":"Deployment","namespace":"default","name":"child-deploy","group":"apps","normalizedLiveState":` + jsonEscape(deployState) + `,"predictedLiveState":` + jsonEscape(deployState) + `}` +
			`]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"parent-app","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}}}`)))
		_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"child-app","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}}}`)))
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// TestAppOfApps_ChildAppDiff_ShowsApplicationCRDiff verifies that pressing d on a
// child Application node in an app-of-apps tree shows the Application CR diff
// (not "No differences found" from the child's internal resources).
func TestAppOfApps_ChildAppDiff_ShowsApplicationCRDiff(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithAppOfApps()
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

	mockLess, inputFile := createMockLess(t, tf.workspace)
	binDir := filepath.Dir(mockLess)
	origPath := os.Getenv("PATH")

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"PATH="+binDir+":"+origPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	// Navigate to parent-app's tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources parent-app")
	_ = tf.Enter()

	// Wait for tree to load with the child Application node
	if !tf.WaitForPlain("Application [parent-app]", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("parent-app tree view not loaded")
	}

	// Navigate to child Application node (j) and request diff (d).
	_ = tf.Send("jd")

	// Should open diff viewer with the Application CR targetRevision diff (not "No differences found")
	if !tf.WaitForPlain("Mock less", 5*time.Second) {
		snapshot := tf.SnapshotPlain()
		if strings.Contains(snapshot, "No differences found") {
			t.Log(snapshot)
			t.Fatal("diff shows 'No differences found' — child app's internal resources were diffed instead of the Application CR")
		}
		t.Log(snapshot)
		t.Fatal("mock less was not launched for Application CR diff")
	}

	// Verify diff content contains the targetRevision change
	diffContent, err := os.ReadFile(inputFile)
	if err != nil {
		t.Fatalf("failed to read less input: %v", err)
	}

	diffStr := string(diffContent)
	if !strings.Contains(diffStr, "targetRevision") {
		t.Errorf("diff should contain 'targetRevision' change, got: %s", diffStr)
	}
}
