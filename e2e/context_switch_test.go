//go:build e2e && unix

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// WriteArgoConfigMultiContext writes an ArgoCD CLI config with two named contexts
// pointing to two different servers.
func WriteArgoConfigMultiContext(path, serverAURL, serverBURL string) error {
	var y bytes.Buffer
	y.WriteString("contexts:\n")
	y.WriteString("  - name: serverA\n    server: " + serverAURL + "\n    user: userA\n")
	y.WriteString("  - name: serverB\n    server: " + serverBURL + "\n    user: userB\n")
	y.WriteString("servers:\n")
	y.WriteString("  - server: " + serverAURL + "\n    insecure: true\n")
	y.WriteString("  - server: " + serverBURL + "\n    insecure: true\n")
	y.WriteString("users:\n")
	y.WriteString("  - name: userA\n    auth-token: token-a\n")
	y.WriteString("  - name: userB\n    auth-token: token-b\n")
	y.WriteString("current-context: serverA\n")
	return os.WriteFile(path, y.Bytes(), 0o644)
}

// MockArgoServerContextA creates a mock server with apps ["alpha-app", "bravo-app"].
// The SSE stream sends MODIFIED events for both apps in multiple rounds before returning.
func MockArgoServerContextA() (*httptest.Server, *StreamRecorder, error) {
	rec := &StreamRecorder{}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	appsJSON := `[
		{"metadata":{"name":"alpha-app","namespace":"argocd"},"spec":{"project":"projA","destination":{"name":"cluster-a","namespace":"ns-a"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
		{"metadata":{"name":"bravo-app","namespace":"argocd"},"spec":{"project":"projA","destination":{"name":"cluster-a","namespace":"ns-a"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}
	]`

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(appsJSON, "100")))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

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
		{"alpha-app", "projA"},
		{"bravo-app", "projA"},
	}

	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		rec.add(StreamRequest{
			Projects:        r.URL.Query()["projects"],
			ResourceVersion: r.URL.Query().Get("resourceVersion"),
		})

		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")

		// Send multiple rounds of events to simulate a busy server.
		// Per memory notes: send events and return, never block on r.Context().Done().
		for round := 0; round < 6; round++ {
			for _, app := range apps {
				if shouldSendEvent(r, app.project) {
					event := fmt.Sprintf(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"%s","namespace":"argocd"},"spec":{"project":"%s","destination":{"name":"cluster-a","namespace":"ns-a"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`, app.name, app.project)
					_, _ = w.Write([]byte(sseEvent(event)))
				}
			}
			if fl != nil {
				fl.Flush()
			}
			time.Sleep(500 * time.Millisecond)
		}
	})

	srv := httptest.NewServer(mux)
	return srv, rec, nil
}

// MockArgoServerContextB creates a mock server with apps ["charlie-app", "delta-app"].
func MockArgoServerContextB() (*httptest.Server, *StreamRecorder, error) {
	rec := &StreamRecorder{}
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	appsJSON := `[
		{"metadata":{"name":"charlie-app","namespace":"argocd"},"spec":{"project":"projB","destination":{"name":"cluster-b","namespace":"ns-b"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}},
		{"metadata":{"name":"delta-app","namespace":"argocd"},"spec":{"project":"projB","destination":{"name":"cluster-b","namespace":"ns-b"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}
	]`

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(appsJSON, "200")))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

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
		{"charlie-app", "projB"},
		{"delta-app", "projB"},
	}

	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		rec.add(StreamRequest{
			Projects:        r.URL.Query()["projects"],
			ResourceVersion: r.URL.Query().Get("resourceVersion"),
		})

		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")

		for _, app := range apps {
			if shouldSendEvent(r, app.project) {
				event := fmt.Sprintf(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"%s","namespace":"argocd"},"spec":{"project":"%s","destination":{"name":"cluster-b","namespace":"ns-b"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`, app.name, app.project)
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

// TestContextSwitchDiscardsOldSSEEvents verifies the core safety property:
// after switching from Server A to Server B, SSE events from Server A's stream
// are discarded and never leak into Server B's view.
func TestContextSwitchDiscardsOldSSEEvents(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// 1. Create two mock servers with StreamRecorders
	srvA, recA, err := MockArgoServerContextA()
	if err != nil {
		t.Fatalf("mock server A: %v", err)
	}
	t.Cleanup(srvA.Close)

	srvB, recB, err := MockArgoServerContextB()
	if err != nil {
		t.Fatalf("mock server B: %v", err)
	}
	t.Cleanup(srvB.Close)

	// 2. Write multi-context ArgoCD config (current-context: serverA)
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigMultiContext(cfgPath, srvA.URL, srvB.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Start app with -argocd-config flag
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// 4. Wait for Server A to load (default view is clusters)
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatalf("Server A not loaded\n%s", tf.SnapshotPlain())
	}

	// 5. Navigate to apps view to see individual app names
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForScreen("alpha-app", 3*time.Second) {
		t.Fatalf("alpha-app not visible\nScreen:\n%s", tf.Screen())
	}
	if !tf.WaitForScreen("bravo-app", 3*time.Second) {
		t.Fatalf("bravo-app not visible\nScreen:\n%s", tf.Screen())
	}

	// 6. Verify SSE stream connected to Server A
	if !waitUntil(t, func() bool { return recA.len() > 0 }, 3*time.Second) {
		t.Fatal("expected at least one SSE stream request to Server A")
	}

	// 7. Switch context: open command, type "context serverB", press Enter
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("context serverB")
	_ = tf.Enter()

	// 8. Wait for Server B to load (default view is clusters, so wait for cluster-b)
	if !tf.WaitForScreen("cluster-b", 8*time.Second) {
		t.Fatalf("cluster-b not visible after context switch\nScreen:\n%s", tf.Screen())
	}

	// 9. Navigate to apps view on Server B (context switch resets to default clusters view)
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForScreen("charlie-app", 3*time.Second) {
		t.Fatalf("charlie-app not visible in apps view\nScreen:\n%s", tf.Screen())
	}

	// 10. Verify Server A's apps are NOT on screen
	screen := tf.Screen()
	if strings.Contains(screen, "alpha-app") {
		t.Fatalf("alpha-app should NOT be visible after switching to Server B\nScreen:\n%s", screen)
	}
	if strings.Contains(screen, "bravo-app") {
		t.Fatalf("bravo-app should NOT be visible after switching to Server B\nScreen:\n%s", screen)
	}

	// 11. Let Server A's SSE keep emitting (it sends events every 500ms for 3s total)
	time.Sleep(2 * time.Second)

	// 12. Final assertion: screen still shows ONLY Server B's apps
	screen = tf.Screen()
	if strings.Contains(screen, "alpha-app") {
		t.Fatalf("alpha-app leaked onto screen after waiting — old SSE events not discarded\nScreen:\n%s", screen)
	}
	if strings.Contains(screen, "bravo-app") {
		t.Fatalf("bravo-app leaked onto screen after waiting — old SSE events not discarded\nScreen:\n%s", screen)
	}
	if !strings.Contains(screen, "charlie-app") {
		t.Fatalf("charlie-app should still be visible\nScreen:\n%s", screen)
	}
	if !strings.Contains(screen, "delta-app") {
		t.Fatalf("delta-app should still be visible\nScreen:\n%s", screen)
	}

	// 13. Verify SSE stream connected to Server B
	if !waitUntil(t, func() bool { return recB.len() > 0 }, 3*time.Second) {
		t.Fatal("expected at least one SSE stream request to Server B")
	}
}

// TestViewContextsDismissesLoadingState verifies that when the user switches to
// an unreachable context (which shows "Connecting to Argo CD...") and then opens
// the :context picker view, the loading overlay is dismissed and the context list
// is visible.
func TestViewContextsDismissesLoadingState(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// 1. Create server A (reachable)
	srvA, _, err := MockArgoServerContextA()
	if err != nil {
		t.Fatalf("mock server A: %v", err)
	}
	t.Cleanup(srvA.Close)

	// Server B is unreachable — port 1 will always refuse connections
	unreachableURL := "http://127.0.0.1:1"

	// 2. Write multi-context config (current-context: serverA)
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigMultiContext(cfgPath, srvA.URL, unreachableURL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// 3. Start app with -argocd-config flag
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// 4. Wait for Server A to load (default view is clusters)
	if !tf.WaitForScreen("cluster-a", 5*time.Second) {
		t.Fatalf("Server A not loaded\n%s", tf.Screen())
	}

	// 5. Switch to serverB (unreachable): open command, type "context serverB", Enter
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("context serverB")
	_ = tf.Enter()

	// 6. Wait for "Connecting to Argo CD" spinner to appear
	if !tf.WaitForScreen("Connecting to Argo CD", 5*time.Second) {
		t.Fatalf("expected 'Connecting to Argo CD' after switching to unreachable context\n%s", tf.Screen())
	}

	// 7. Brief pause, then open :context view (no arg = browse contexts)
	time.Sleep(300 * time.Millisecond)
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("context")
	_ = tf.Enter()

	// 8. Wait for context names to appear in the list
	if !tf.WaitForScreen("serverA", 3*time.Second) {
		t.Fatalf("expected 'serverA' in contexts view\n%s", tf.Screen())
	}

	// 9. Assert the other context is also visible
	screen := tf.Screen()
	if !strings.Contains(screen, "serverB") {
		t.Fatalf("expected 'serverB' in contexts view\nScreen:\n%s", screen)
	}

	// 10. Assert the loading/connecting overlay is gone
	if strings.Contains(screen, "Connecting to Argo CD") {
		t.Fatalf("'Connecting to Argo CD' should be dismissed when contexts view is active\nScreen:\n%s", screen)
	}

	// 11. Wait a moment and re-check — the loading state should not reappear
	time.Sleep(1 * time.Second)
	screen = tf.Screen()
	if strings.Contains(screen, "Connecting to Argo CD") {
		t.Fatalf("'Connecting to Argo CD' reappeared after waiting — loading state leaked into contexts view\nScreen:\n%s", screen)
	}
	if !strings.Contains(screen, "serverA") {
		t.Fatalf("contexts view should still show serverA after waiting\nScreen:\n%s", screen)
	}
}
