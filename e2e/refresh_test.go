//go:build e2e && unix

package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- Refresh capturing mock server ----

type RefreshCall struct {
	Name string
	Hard bool // true if refresh=hard, false if refresh=true
}

type RefreshRecorder struct {
	mu    sync.Mutex
	Calls []RefreshCall
}

func (rr *RefreshRecorder) add(call RefreshCall) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.Calls = append(rr.Calls, call)
}

func (rr *RefreshRecorder) len() int {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	return len(rr.Calls)
}

// MockArgoServerRefresh returns an auth-checking server and a recorder for refresh calls
func MockArgoServerRefresh(validToken string) (*httptest.Server, *RefreshRecorder, error) {
	rec := &RefreshRecorder{}
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
		_, _ = w.Write([]byte(wrapListResponse(`[
            {"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}},
            {"metadata":{"name":"demo2","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}
        ]`, "1000")))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"nodes":[]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo2/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"nodes":[]}`))
	})

	// Handler for individual application GET with refresh query param
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}

		// Extract app name from path: /api/v1/applications/<name>
		p := r.URL.Path
		segs := strings.Split(strings.TrimPrefix(p, "/api/v1/applications/"), "/")
		if len(segs) == 0 || segs[0] == "" {
			http.NotFound(w, r)
			return
		}
		name := segs[0]

		// Check for refresh query param
		refreshParam := r.URL.Query().Get("refresh")
		if refreshParam != "" {
			hard := refreshParam == "hard"
			rec.add(RefreshCall{Name: name, Hard: hard})
		}

		// Return app JSON
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"metadata":{"name":"` + name + `","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}`))
	})

	// SSE stream endpoint (required for app to start properly)
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		// Send one event and return (don't block)
		if shouldSendEvent(r, "demo") {
			_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}}}`)))
		}
		if fl != nil {
			fl.Flush()
		}
	})

	srv := httptest.NewServer(mux)
	return srv, rec, nil
}

// TestRefreshCommand tests both :refresh and :refresh! commands
func TestRefreshCommand(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerRefresh("valid-token")
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

	// Navigate to apps view
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

	// Test 1: Normal refresh with :refresh command
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("refresh")
	_ = tf.Enter()

	// Wait for refresh call
	if !waitUntil(t, func() bool { return rec.len() >= 1 }, 2*time.Second) {
		t.Fatalf("expected at least 1 refresh call, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}

	call := rec.Calls[0]
	if call.Name != "demo" {
		t.Fatalf("expected refresh for 'demo', got %q", call.Name)
	}
	if call.Hard {
		t.Fatal("expected normal refresh (hard=false), got hard refresh")
	}

	// Test 2: Hard refresh with :refresh! command
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("refresh!")
	_ = tf.Enter()

	// Wait for second refresh call
	if !waitUntil(t, func() bool { return rec.len() >= 2 }, 2*time.Second) {
		t.Fatalf("expected at least 2 refresh calls, got %d\n%s", rec.len(), tf.SnapshotPlain())
	}

	call2 := rec.Calls[1]
	if call2.Name != "demo" {
		t.Fatalf("expected hard refresh for 'demo', got %q", call2.Name)
	}
	if !call2.Hard {
		t.Fatal("expected hard refresh (hard=true), got normal refresh")
	}
}
