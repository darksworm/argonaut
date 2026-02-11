//go:build e2e && unix

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- Stream request recording ----

type StreamRequest struct {
	Projects        []string
	ResourceVersion string
}

type StreamRecorder struct {
	mu       sync.Mutex
	Requests []StreamRequest
}

func (sr *StreamRecorder) add(req StreamRequest) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.Requests = append(sr.Requests, req)
}

func (sr *StreamRecorder) len() int {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return len(sr.Requests)
}

func (sr *StreamRecorder) last() StreamRequest {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	if len(sr.Requests) == 0 {
		return StreamRequest{}
	}
	return sr.Requests[len(sr.Requests)-1]
}

// MockArgoServerMultiProject creates a server with 4 apps across 2 projects (frontend, backend)
// and records stream requests via StreamRecorder.
func MockArgoServerMultiProject() (*httptest.Server, *StreamRecorder, error) {
	rec := &StreamRecorder{}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	appsJSON := `[
		{"metadata":{"name":"web-frontend","namespace":"argocd"},"spec":{"project":"frontend","destination":{"name":"cluster-a","namespace":"web"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
		{"metadata":{"name":"web-backend","namespace":"argocd"},"spec":{"project":"frontend","destination":{"name":"cluster-a","namespace":"web"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
		{"metadata":{"name":"db-primary","namespace":"argocd"},"spec":{"project":"backend","destination":{"name":"cluster-a","namespace":"data"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
		{"metadata":{"name":"db-replica","namespace":"argocd"},"spec":{"project":"backend","destination":{"name":"cluster-a","namespace":"data"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}
	]`

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(appsJSON, "1000")))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

	// Resource tree for each app (minimal)
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/resource-tree") {
			_, _ = w.Write([]byte(`{"nodes":[]}`))
			return
		}
		http.NotFound(w, r)
	})

	type appInfo struct {
		name    string
		project string
	}
	apps := []appInfo{
		{"web-frontend", "frontend"},
		{"web-backend", "frontend"},
		{"db-primary", "backend"},
		{"db-replica", "backend"},
	}

	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		// Record request
		rec.add(StreamRequest{
			Projects:        r.URL.Query()["projects"],
			ResourceVersion: r.URL.Query().Get("resourceVersion"),
		})

		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")

		for _, app := range apps {
			if shouldSendEvent(r, app.project) {
				event := fmt.Sprintf(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"%s","namespace":"argocd"},"spec":{"project":"%s","destination":{"name":"cluster-a","namespace":"web"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`, app.name, app.project)
				_, _ = w.Write([]byte(sseEvent(event)))
			}
		}
		if fl != nil {
			fl.Flush()
		}
	})

	srv := httptest.NewServer(mux)
	return srv, rec, nil
}

// TestScopedStreamingFiltersEvents verifies that drilling down to a specific project
// triggers a watch restart with the ?projects= filter.
func TestScopedStreamingFiltersEvents(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerMultiProject()
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
		t.Fatalf("initial view not ready\n%s", tf.SnapshotPlain())
	}

	// Navigate to apps view to verify all 4 apps are loaded
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForPlain("web-frontend", 3*time.Second) {
		t.Fatalf("apps not loaded\n%s", tf.SnapshotPlain())
	}
	if !tf.WaitForPlain("db-primary", 3*time.Second) {
		t.Fatalf("all apps not loaded\n%s", tf.SnapshotPlain())
	}

	// Navigate to projects view
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("projects")
	_ = tf.Enter()

	// Wait for projects to appear
	if !tf.WaitForPlain("backend", 3*time.Second) {
		t.Fatalf("projects not shown\n%s", tf.SnapshotPlain())
	}
	if !tf.WaitForPlain("frontend", 3*time.Second) {
		t.Fatalf("frontend project not shown\n%s", tf.SnapshotPlain())
	}

	// Record how many stream requests we have before drilling down
	initialRequests := rec.len()

	// Drill down into the first project by pressing Enter
	// Projects are sorted alphabetically, so "backend" should be first
	_ = tf.Enter()

	// Wait for the scoped view — should show backend apps
	if !tf.WaitForPlain("db-primary", 3*time.Second) {
		t.Fatalf("backend apps not shown after drill-down\n%s", tf.SnapshotPlain())
	}

	// Wait for watch restart (500ms debounce + some margin)
	if !waitUntil(t, func() bool { return rec.len() > initialRequests }, 3*time.Second) {
		t.Fatalf("expected additional stream request after scope change, had %d requests before drill-down, still %d",
			initialRequests, rec.len())
	}

	// Verify the latest stream request has projects=["backend"]
	last := rec.last()
	if len(last.Projects) != 1 || last.Projects[0] != "backend" {
		t.Fatalf("expected stream request with projects=[backend], got %v", last.Projects)
	}

	// Verify resourceVersion is passed (non-empty since we got list response with rv)
	if last.ResourceVersion == "" {
		t.Log("Note: resourceVersion was empty in scoped stream request")
	}

	// Verify that only backend apps are visible on the rendered screen (not frontend apps)
	// Use Screen() not SnapshotPlain() — SnapshotPlain() contains raw history including
	// earlier output when all apps were visible.
	screen := tf.Screen()
	if strings.Contains(screen, "web-frontend") {
		t.Fatalf("web-frontend should not be visible when scoped to backend project\nScreen:\n%s", screen)
	}
	if strings.Contains(screen, "web-backend") {
		t.Fatalf("web-backend should not be visible when scoped to backend project\nScreen:\n%s", screen)
	}
	if !strings.Contains(screen, "db-primary") {
		t.Fatalf("db-primary should be visible when scoped to backend project\nScreen:\n%s", screen)
	}
}
