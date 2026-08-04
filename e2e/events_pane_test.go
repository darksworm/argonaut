//go:build e2e && unix

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// MockArgoServerWithEvents serves an app whose Pod has warning events and
// whose last sync operation failed, exercising the events and sync-status
// panes plus the summary line under the tree root.
func MockArgoServerWithEvents() (*httptest.Server, error) {
	startedAt := time.Now().Add(-2 * time.Minute).UTC().Format(time.RFC3339)
	finishedAt := time.Now().Add(-2*time.Minute + 6*time.Second).UTC().Format(time.RFC3339)
	operationState := fmt.Sprintf(`{
		"phase": "Failed",
		"message": "one or more objects failed to apply",
		"startedAt": %q,
		"finishedAt": %q,
		"operation": {"sync": {"revision": "HEAD"}, "initiatedBy": {"username": "alice"}},
		"syncResult": {"revision": "a1b2c3d4e5f6789", "resources": [
			{"kind": "Service", "namespace": "default", "name": "nginx-service", "status": "Synced", "message": "service/nginx-service unchanged"},
			{"kind": "Deployment", "namespace": "default", "name": "nginx-deployment", "status": "SyncFailed", "message": "Deployment.apps \"nginx-deployment\" is invalid"}
		]}
	}`, startedAt, finishedAt)
	appJSON := fmt.Sprintf(`{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Degraded"},"operationState":%s}}`, operationState)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse("["+appJSON+"]", "1000")))
	})
	mux.HandleFunc("/api/v1/applications/demo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(appJSON))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"nginx-deployment","namespace":"default","version":"v1","group":"apps","uid":"dep-1","health":{"status":"Degraded"}},
			{"kind":"Pod","name":"nginx-pod-xyz789","namespace":"default","version":"v1","uid":"pod-1","health":{"status":"Degraded"},"parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"nginx-deployment","namespace":"default","group":"apps","version":"v1"}]}
		]}`))
	})
	lastSeen := time.Now().Add(-90 * time.Second).UTC().Format(time.RFC3339)
	mux.HandleFunc("/api/v1/applications/demo/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("resourceUID") == "pod-1" {
			_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[
				{"type":"Warning","reason":"BackOff","message":"Back-off restarting failed container nginx","count":42,"lastTimestamp":%q}
			]}`, lastSeen)))
			return
		}
		_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[
			{"type":"Warning","reason":"OperationCompleted","message":"Sync operation to a1b2c3d failed","count":1,"lastTimestamp":%q}
		]}`, lastSeen)))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		if shouldSendEvent(r, "demo") {
			_, _ = w.Write([]byte(sseEvent(fmt.Sprintf(`{"result":{"type":"MODIFIED","application":%s}}`, appJSON))))
		}
		if fl != nil {
			fl.Flush()
		}
	})
	return httptest.NewServer(mux), nil
}

func openDemoTree(t *testing.T, tf *TUITestFramework) {
	t.Helper()
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()
	if !tf.WaitForPlain("Application [demo]", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application root not shown")
	}
}

func TestEventsPane_ShowsResourceEvents(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithEvents()
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

	openDemoTree(t, tf)

	// The app root carries the last-sync summary line
	if !tf.WaitForPlain("last sync:", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("sync summary line not shown under the app root")
	}

	// Move to the Pod row and open its events
	_ = tf.Send("j")
	time.Sleep(50 * time.Millisecond)
	_ = tf.Send("j")
	time.Sleep(50 * time.Millisecond)
	_ = tf.Send("e")

	if !tf.WaitForPlain("─ Pod nginx-pod-xyz789 ", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("events pane title not shown")
	}
	if !tf.WaitForPlain("BackOff", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("pod warning event not shown")
	}
	if !tf.WaitForPlain("esc: close", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("pane hints not shown in status line")
	}

	// Escape lands back on the intact tree
	_ = tf.Escape()
	if !waitUntil(t, func() bool {
		s := tf.SnapshotPlain()
		return strings.Contains(s, "Application [demo]") && !strings.Contains(tf.Screen(), "esc: close")
	}, 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("esc did not land back on the tree")
	}
}

// Opening the pane on the application row shows the last operation's status
// block above the app's events — one lens for both.
func TestEventsPane_AppRow_ShowsOperationStateAboveEvents(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithEvents()
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

	openDemoTree(t, tf)

	_ = tf.Send("e") // cursor starts on the application root

	if !tf.WaitForPlain("─ Application demo ", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("app lens title not shown")
	}
	if !tf.WaitForPlain("Phase         Failed", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("operation phase not shown")
	}
	if !tf.WaitForPlain("SyncFailed", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("per-resource result not shown")
	}
	if !tf.WaitForPlain("EVENTS", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("events section not shown below the status block")
	}
	if !tf.WaitForPlain("OperationCompleted", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("application events not shown")
	}

	_ = tf.Escape()
	if !waitUntil(t, func() bool {
		s := tf.SnapshotPlain()
		return strings.Contains(s, "Application [demo]") && !strings.Contains(tf.Screen(), "esc: close")
	}, 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("esc did not land back on the tree")
	}
}
