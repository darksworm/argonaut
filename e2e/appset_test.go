//go:build e2e && unix

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockArgoServerWithAppSets creates a mock server with apps that have ApplicationSet ownerReferences
func MockArgoServerWithAppSets() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// Return apps with ownerReferences linking them to ApplicationSets
		_, _ = w.Write([]byte(wrapListResponse(`[
			{
				"metadata":{
					"name":"app-from-appset-1",
					"namespace":"argocd",
					"ownerReferences":[{"apiVersion":"argoproj.io/v1alpha1","kind":"ApplicationSet","name":"nerdy-demo","uid":"123"}]
				},
				"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},
				"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}
			},
			{
				"metadata":{
					"name":"app-from-appset-2",
					"namespace":"argocd",
					"ownerReferences":[{"apiVersion":"argoproj.io/v1alpha1","kind":"ApplicationSet","name":"nerdy-demo","uid":"123"}]
				},
				"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},
				"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}
			},
			{
				"metadata":{
					"name":"prod-app",
					"namespace":"argocd",
					"ownerReferences":[{"apiVersion":"argoproj.io/v1alpha1","kind":"ApplicationSet","name":"production-apps","uid":"456"}]
				},
				"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},
				"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Degraded"}}
			},
			{
				"metadata":{"name":"standalone-app","namespace":"argocd"},
				"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},
				"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}
			}
		]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		if shouldSendEvent(r, "default") {
			_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"app-from-appset-1","namespace":"argocd","ownerReferences":[{"kind":"ApplicationSet","name":"nerdy-demo"}]},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`)))
		}
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

func TestAppSetsCommand_ShowsApplicationSetsList(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithAppSets()
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

	// Wait for initial clusters view
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("initial view not ready")
	}

	// Navigate to ApplicationSets view
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("appsets")
	_ = tf.Enter()

	// Verify ApplicationSets list shows unique ApplicationSet names
	if !tf.WaitForPlain("nerdy-demo", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected to see nerdy-demo ApplicationSet")
	}
	if !tf.WaitForPlain("production-apps", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected to see production-apps ApplicationSet")
	}

	// Verify standalone apps don't appear in the list
	snapshot := tf.SnapshotPlain()
	if strings.Contains(snapshot, "standalone-app") {
		t.Fatal("standalone-app should not appear in ApplicationSets view")
	}
}

func TestAppSetCommand_NavigatesToFilteredApps(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithAppSets()
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
		t.Fatal("initial view not ready")
	}

	// Navigate via :appset nerdy-demo to filter apps
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("appset nerdy-demo")
	_ = tf.Enter()

	// Verify filtered apps view shows only apps from nerdy-demo ApplicationSet
	if !tf.WaitForPlain("app-from-appset-1", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected to see app-from-appset-1")
	}
	if !tf.WaitForPlain("app-from-appset-2", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected to see app-from-appset-2")
	}

	// Verify apps from other ApplicationSets and standalone apps are NOT shown
	snapshot := tf.SnapshotPlain()
	if strings.Contains(snapshot, "prod-app") {
		t.Fatal("prod-app should not appear when filtered by nerdy-demo")
	}
	if strings.Contains(snapshot, "standalone-app") {
		t.Fatal("standalone-app should not appear when filtered by ApplicationSet")
	}
}

func TestAppSetCommand_DrillDown(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithAppSets()
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
		t.Fatal("initial view not ready")
	}

	// Navigate to ApplicationSets view
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("appsets")
	_ = tf.Enter()

	// Wait for ApplicationSets to appear
	if !tf.WaitForPlain("nerdy-demo", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("ApplicationSets view not ready")
	}

	// Drill down with Enter on selected ApplicationSet
	_ = tf.Enter()

	// Should now be in apps view filtered by the selected ApplicationSet
	if !tf.WaitForPlain("app-from-appset", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected drill-down to show apps from ApplicationSet")
	}
}

func TestAppSetCommand_InvalidAppSet(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithAppSets()
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
		t.Fatal("initial view not ready")
	}

	// Try to navigate to non-existent ApplicationSet
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("appset nonexistent")
	_ = tf.Enter()

	// The validateCommand() function marks unknown arguments as invalid,
	// so the command bar shows "unknown command" visual indicator.
	// The command is still sent but validation prevents execution.
	// We verify the UI rejects invalid input.
	if !tf.WaitForPlain("unknown command", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected unknown command indicator for invalid ApplicationSet")
	}
}
