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

// ResourceSyncCall captures a resource sync API call
type ResourceSyncCall struct {
	AppName   string
	Body      string
	Resources []map[string]interface{} // Parsed resources array from body
}

// ResourceSyncRecorder records resource sync API calls
type ResourceSyncRecorder struct {
	mu    sync.Mutex
	Calls []ResourceSyncCall
}

func (r *ResourceSyncRecorder) add(call ResourceSyncCall) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Calls = append(r.Calls, call)
}

func (r *ResourceSyncRecorder) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.Calls)
}

func (r *ResourceSyncRecorder) getCall(i int) ResourceSyncCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	if i < len(r.Calls) {
		return r.Calls[i]
	}
	return ResourceSyncCall{}
}

// MockArgoServerForResourceSync creates a mock server for testing resource-level sync
func MockArgoServerForResourceSync(validToken string) (*httptest.Server, *ResourceSyncRecorder, error) {
	rec := &ResourceSyncRecorder{}
	mux := http.NewServeMux()

	requireAuth := func(w http.ResponseWriter, r *http.Request) bool {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+validToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return false
		}
		return true
	}

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// One app for testing resource sync
		_, _ = w.Write([]byte(`{"items":[
			{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"},"resources":[
				{"group":"apps","version":"v1","kind":"Deployment","namespace":"default","name":"nginx-deployment","status":"OutOfSync","health":{"status":"Healthy"}},
				{"group":"","version":"v1","kind":"Service","namespace":"default","name":"nginx-service","status":"Synced","health":{"status":"Healthy"}}
			]}}
		]}`))
	})

	// Resource tree with actual resources
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Tree with a Deployment and Service
		_, _ = w.Write([]byte(`{
			"nodes": [
				{
					"group": "apps",
					"version": "v1",
					"kind": "Deployment",
					"namespace": "default",
					"name": "nginx-deployment",
					"uid": "uid-deployment-1",
					"health": {"status": "Healthy"},
					"resourceRef": {"group": "apps", "version": "v1", "kind": "Deployment", "namespace": "default", "name": "nginx-deployment", "uid": "uid-deployment-1"}
				},
				{
					"group": "",
					"version": "v1",
					"kind": "Service",
					"namespace": "default",
					"name": "nginx-service",
					"uid": "uid-service-1",
					"health": {"status": "Healthy"},
					"resourceRef": {"group": "", "version": "v1", "kind": "Service", "namespace": "default", "name": "nginx-service", "uid": "uid-service-1"}
				}
			]
		}`))
	})

	// Diffs
	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	})

	// Sync endpoint - capture all sync calls
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if !requireAuth(w, r) {
			return
		}
		// Expect path like /api/v1/applications/<name>/sync
		p := r.URL.Path
		if !strings.HasSuffix(p, "/sync") {
			http.NotFound(w, r)
			return
		}
		segs := strings.Split(p, "/")
		if len(segs) < 6 {
			http.NotFound(w, r)
			return
		}
		appName := segs[4]
		body, _ := io.ReadAll(r.Body)

		// Parse body to extract resources
		var bodyMap map[string]interface{}
		var resources []map[string]interface{}
		if err := json.Unmarshal(body, &bodyMap); err == nil {
			if res, ok := bodyMap["resources"].([]interface{}); ok {
				for _, r := range res {
					if rm, ok := r.(map[string]interface{}); ok {
						resources = append(resources, rm)
					}
				}
			}
		}

		rec.add(ResourceSyncCall{
			AppName:   appName,
			Body:      string(body),
			Resources: resources,
		})
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	srv := httptest.NewServer(mux)
	return srv, rec, nil
}

// TestResourceSync_SingleResource tests syncing a single resource from tree view
func TestResourceSync_SingleResource(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerForResourceSync("valid-token")
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

	// Navigate to apps via commands
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
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Enter tree view by pressing 'r'
	_ = tf.Send("r")
	if !tf.WaitForPlain("Deployment", 3*time.Second) {
		t.Fatalf("tree view not loaded\n%s", tf.SnapshotPlain())
	}

	// Navigate down to first resource (Deployment)
	_ = tf.Send("j") // Move to first resource (down from app node)

	// Trigger resource sync with 's'
	_ = tf.Send("s")

	// Wait for the sync confirmation modal
	if !tf.WaitForPlain("Sync", 2*time.Second) {
		t.Fatalf("sync modal not shown\n%s", tf.SnapshotPlain())
	}

	// Confirm sync with 'y'
	_ = tf.Send("y")

	// Wait for sync call
	if !waitUntil(t, func() bool { return rec.len() >= 1 }, 3*time.Second) {
		t.Fatalf("expected at least 1 sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}

	call := rec.getCall(0)
	if call.AppName != "demo" {
		t.Fatalf("expected sync for 'demo', got %q", call.AppName)
	}

	// Verify resources array is in the body
	if len(call.Resources) != 1 {
		t.Fatalf("expected 1 resource in sync request, got %d. Body: %s", len(call.Resources), call.Body)
	}

	// Verify the resource details
	res := call.Resources[0]
	if res["kind"] != "Deployment" {
		t.Fatalf("expected kind=Deployment, got %v", res["kind"])
	}
	if res["name"] != "nginx-deployment" {
		t.Fatalf("expected name=nginx-deployment, got %v", res["name"])
	}
}

// TestResourceSync_Cancel tests cancelling a resource sync
func TestResourceSync_Cancel(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerForResourceSync("valid-token")
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

	// Navigate to apps and then tree view
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
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Enter tree view
	_ = tf.Send("r")
	if !tf.WaitForPlain("Deployment", 3*time.Second) {
		t.Fatalf("tree view not loaded\n%s", tf.SnapshotPlain())
	}

	// Navigate to resource
	_ = tf.Send("j")

	// Trigger resource sync
	_ = tf.Send("s")

	// Wait for modal
	if !tf.WaitForPlain("Sync", 2*time.Second) {
		t.Fatalf("sync modal not shown\n%s", tf.SnapshotPlain())
	}

	// Cancel with escape
	_ = tf.Send("\x1b") // Escape

	// Wait a bit and verify no sync calls were made
	time.Sleep(500 * time.Millisecond)
	if rec.len() != 0 {
		t.Fatalf("expected 0 sync calls after cancel, got %d", rec.len())
	}
}

// TestResourceSync_WithPrune tests syncing with prune option enabled
func TestResourceSync_WithPrune(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerForResourceSync("valid-token")
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

	// Navigate to tree view
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
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	_ = tf.Send("r")
	if !tf.WaitForPlain("Deployment", 3*time.Second) {
		t.Fatalf("tree view not loaded\n%s", tf.SnapshotPlain())
	}

	_ = tf.Send("j")
	_ = tf.Send("s")

	if !tf.WaitForPlain("Sync", 2*time.Second) {
		t.Fatalf("sync modal not shown\n%s", tf.SnapshotPlain())
	}

	// Toggle prune option with 'p'
	_ = tf.Send("p")

	// Confirm sync
	_ = tf.Send("y")

	if !waitUntil(t, func() bool { return rec.len() >= 1 }, 3*time.Second) {
		t.Fatalf("expected sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}

	call := rec.getCall(0)

	// Parse body and verify prune=true
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(call.Body), &body); err != nil {
		t.Fatalf("invalid body json: %v", err)
	}
	if prune, ok := body["prune"].(bool); !ok || !prune {
		t.Fatalf("expected prune=true in body, got %v. Body: %s", body["prune"], call.Body)
	}
}

// TestResourceSync_AppNodeFullSync tests that pressing 's' on app root triggers full app sync
func TestResourceSync_AppNodeFullSync(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerForResourceSync("valid-token")
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

	// Navigate to tree view
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
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	_ = tf.Send("r")
	if !tf.WaitForPlain("Deployment", 3*time.Second) {
		t.Fatalf("tree view not loaded\n%s", tf.SnapshotPlain())
	}

	// Stay on app node (don't navigate down) and trigger sync
	_ = tf.Send("s")

	// Wait for the app-level sync modal (not resource sync modal)
	if !tf.WaitForPlain("Sync demo", 2*time.Second) {
		t.Fatalf("app sync modal not shown\n%s", tf.SnapshotPlain())
	}

	// Confirm sync
	_ = tf.Send("\r")

	if !waitUntil(t, func() bool { return rec.len() >= 1 }, 3*time.Second) {
		t.Fatalf("expected sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}

	call := rec.getCall(0)

	// Verify this is a full app sync (no resources array)
	var body map[string]interface{}
	if err := json.Unmarshal([]byte(call.Body), &body); err != nil {
		t.Fatalf("invalid body json: %v", err)
	}
	if _, hasResources := body["resources"]; hasResources {
		t.Fatalf("expected full app sync without resources array, but got resources: %s", call.Body)
	}
}

// MockArgoServerForResourceSyncError creates a mock server that returns sync errors
// with ArgoCD's runtimeError format
func MockArgoServerForResourceSyncError(validToken string, errorCode int, errorMessage string) (*httptest.Server, error) {
	mux := http.NewServeMux()

	requireAuth := func(w http.ResponseWriter, r *http.Request) bool {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+validToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return false
		}
		return true
	}

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[
			{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"},"resources":[
				{"group":"apps","version":"v1","kind":"Deployment","namespace":"default","name":"nginx-deployment","status":"OutOfSync","health":{"status":"Healthy"}}
			]}}
		]}`))
	})

	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"nodes": [
				{
					"group": "apps",
					"version": "v1",
					"kind": "Deployment",
					"namespace": "default",
					"name": "nginx-deployment",
					"uid": "uid-deployment-1",
					"health": {"status": "Healthy"},
					"resourceRef": {"group": "apps", "version": "v1", "kind": "Deployment", "namespace": "default", "name": "nginx-deployment", "uid": "uid-deployment-1"}
				}
			]
		}`))
	})

	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[]}`))
	})

	// Sync endpoint - returns error with ArgoCD runtimeError format
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if !requireAuth(w, r) {
			return
		}
		p := r.URL.Path
		if !strings.HasSuffix(p, "/sync") {
			http.NotFound(w, r)
			return
		}
		// Return ArgoCD runtimeError format
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(errorCode)
		errResp := map[string]interface{}{
			"code":    errorCode,
			"error":   "sync_failed",
			"message": errorMessage,
		}
		resp, _ := json.Marshal(errResp)
		_, _ = w.Write(resp)
	})

	srv := httptest.NewServer(mux)
	return srv, nil
}

// TestResourceSync_ErrorDisplayed tests that sync errors are displayed to the user
func TestResourceSync_ErrorDisplayed(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Mock server returns a 403 Forbidden with a descriptive ArgoCD error message
	errorMessage := "permission denied: applications, sync, demo/demo, sub: proj:demo:admin, iat: 2024-01-01"
	srv, err := MockArgoServerForResourceSyncError("valid-token", 403, errorMessage)
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

	// Navigate to tree view
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
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Enter tree view
	_ = tf.Send("r")
	if !tf.WaitForPlain("Deployment", 3*time.Second) {
		t.Fatalf("tree view not loaded\n%s", tf.SnapshotPlain())
	}

	// Navigate to resource
	_ = tf.Send("j")

	// Trigger resource sync
	_ = tf.Send("s")

	// Wait for modal
	if !tf.WaitForPlain("Sync", 2*time.Second) {
		t.Fatalf("sync modal not shown\n%s", tf.SnapshotPlain())
	}

	// Confirm sync
	_ = tf.Send("y")

	// Wait for error message to appear in the UI
	// The error message should contain key parts of the ArgoCD error
	if !tf.WaitForPlain("permission denied", 3*time.Second) {
		snapshot := tf.SnapshotPlain()
		// Check if any error is displayed
		if !strings.Contains(snapshot, "Error") {
			t.Fatalf("expected error to be displayed, but no error visible\n%s", snapshot)
		}
		t.Fatalf("expected 'permission denied' in error message\n%s", snapshot)
	}

	// Verify the error modal shows the ArgoCD error message (not a generic message)
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "applications, sync") {
		t.Fatalf("expected full ArgoCD error details in modal\n%s", snapshot)
	}

	// User can dismiss the error modal
	_ = tf.Send("\x1b") // Escape

	// Wait for modal to close - verify we're back to tree view mode
	time.Sleep(300 * time.Millisecond)

	// The key test assertion: ArgoCD's runtimeError message was parsed and displayed
	// This confirms the error handling chain works end-to-end
	t.Log("Successfully verified: ArgoCD error message was parsed and displayed to user")
}

// TestResourceSync_CommandInTreeView tests that :sync command works in tree view
// This is a regression test for the bug where :sync only worked in apps view
func TestResourceSync_CommandInTreeView(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerForResourceSync("valid-token")
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

	// Navigate to apps via commands
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
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps not ready")
	}

	// Enter tree view by pressing 'r'
	_ = tf.Send("r")
	if !tf.WaitForPlain("Deployment", 3*time.Second) {
		t.Fatalf("tree view not loaded\n%s", tf.SnapshotPlain())
	}

	// Navigate down to first resource (Deployment)
	_ = tf.Send("j") // Move to first resource (down from app node)

	// Use :sync command instead of 's' key - this is what we're testing
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("sync")
	_ = tf.Enter()

	// Wait for the sync confirmation modal
	if !tf.WaitForPlain("Sync", 2*time.Second) {
		t.Fatalf(":sync command did not show sync modal in tree view\n%s", tf.SnapshotPlain())
	}

	// Confirm sync with 'y'
	_ = tf.Send("y")

	// Wait for sync call
	if !waitUntil(t, func() bool { return rec.len() >= 1 }, 3*time.Second) {
		t.Fatalf("expected at least 1 sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}

	call := rec.getCall(0)
	if call.AppName != "demo" {
		t.Fatalf("expected sync for 'demo', got %q", call.AppName)
	}

	// Verify resources array is in the body (resource-level sync, not full app sync)
	if len(call.Resources) != 1 {
		t.Fatalf("expected 1 resource in sync request (resource-level sync), got %d. Body: %s", len(call.Resources), call.Body)
	}

	// Verify the resource details
	res := call.Resources[0]
	if res["kind"] != "Deployment" {
		t.Fatalf("expected kind=Deployment, got %v", res["kind"])
	}
	if res["name"] != "nginx-deployment" {
		t.Fatalf("expected name=nginx-deployment, got %v", res["name"])
	}
}
