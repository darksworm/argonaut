//go:build e2e && unix

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"
)

// MockArgoServerManyApps returns a server with numApps apps named app-01, app-02, ..., app-NN
func MockArgoServerManyApps(numApps int) (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var items []string
		for i := 1; i <= numApps; i++ {
			name := fmt.Sprintf("app-%02d", i)
			items = append(items, fmt.Sprintf(
				`{"metadata":{"name":"%s","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}`,
				name,
			))
		}
		_, _ = w.Write([]byte(wrapListResponse(`[`+strings.Join(items, ",")+`]`, "1000")))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	// Handle resource-tree for any app
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/resource-tree") {
			_, _ = w.Write([]byte(`{"nodes":[]}`))
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(sseEvent(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"app-01","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`)))
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// extractCursorPosition extracts the current cursor position from screen content.
// Expects the screen content from tf.Screen() which properly interprets cursor positioning.
// Returns the current position (1-indexed) or 0 if not found.
func extractCursorPosition(screen string) int {
	// Match "Ready • N/M" pattern - the standard format in apps view
	reWithTotal := regexp.MustCompile(`Ready • (\d+)/(\d+)`)

	// Find the match (should be unique in rendered screen)
	if match := reWithTotal.FindStringSubmatch(screen); match != nil {
		var pos int
		fmt.Sscanf(match[1], "%d", &pos)
		return pos
	}

	// Fallback: try simple "Ready • N" pattern
	reSimple := regexp.MustCompile(`Ready • (\d+)`)
	if match := reSimple.FindStringSubmatch(screen); match != nil {
		var pos int
		fmt.Sscanf(match[1], "%d", &pos)
		return pos
	}

	return 0
}

// waitForCursorPosition waits until the cursor position changes to a value >= minPos
func waitForCursorPosition(tf *TUITestFramework, minPos int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	var lastPos int
	var iterations int
	for time.Now().Before(deadline) {
		screen := tf.Screen()
		pos := extractCursorPosition(screen)
		lastPos = pos
		iterations++
		if pos >= minPos {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	// Debug: log the last position we saw before timing out
	tf.t.Logf("waitForCursorPosition timed out after %d iterations: wanted >= %d, last saw %d", iterations, minPos, lastPos)
	return false
}

// waitForCursorPositionExact waits until the cursor position equals the target
func waitForCursorPositionExact(tf *TUITestFramework, target int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pos := extractCursorPosition(tf.Screen())
		if pos == target {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// navigateToApps is a helper to navigate from initial view to apps list
func navigateToApps(t *testing.T, tf *TUITestFramework) {
	t.Helper()
	// Wait for initial load - increased timeout for CI
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("clusters not ready\nSnapshot:\n%s", snap)
	}
	// Navigate to namespace
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()
	if !tf.WaitForPlain("default", 5*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("namespace not ready\nSnapshot:\n%s", snap)
	}
	// Navigate to apps
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	// Wait for apps to load - app-01 should be visible
	if !tf.WaitForPlain("app-01", 5*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("apps not ready\nSnapshot:\n%s", snap)
	}
}

// TestPageDownFromStart tests PageDown from the beginning of a list
func TestPageDownFromStart(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerManyApps(35)
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

	navigateToApps(t, tf)

	// Verify initial state - cursor should be at position 1
	if !waitForCursorPositionExact(tf, 1, 3*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 initially, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// Send PageDown (escape sequence for PageDown)
	_ = tf.Send("\x1b[6~")

	// After PageDown, cursor should move significantly (at least 10 positions)
	// The exact amount depends on viewport height, but should be substantial
	// Using a lower threshold (10) to be more robust across different environments
	if !waitForCursorPosition(tf, 10, 5*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor position >= 10 after PageDown, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}
}

// TestPageDownAtEnd tests that PageDown at the end of list doesn't go past bounds
func TestPageDownAtEnd(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerManyApps(35)
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

	navigateToApps(t, tf)

	// PageDown four times — last keystroke is past the end, must not crash.
	_ = tf.Send("\x1b[6~\x1b[6~\x1b[6~\x1b[6~")
	if !tf.WaitForPlain("app-35", 2*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("expected app-35 to be visible after PageDown to end\nSnapshot:\n%s", snap)
	}
}

// TestPageUpFromEnd tests PageUp navigation from the end of the list
func TestPageUpFromEnd(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerManyApps(35)
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

	navigateToApps(t, tf)

	// Go to end first using 3 PageDowns, then poll for arrival.
	_ = tf.Send("\x1b[6~\x1b[6~\x1b[6~")
	if !tf.WaitForPlain("app-35", 2*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("expected app-35 to be visible before PageUp\nSnapshot:\n%s", snap)
	}

	// PageUp — cursor should drop back to position <= 10.
	_ = tf.Send("\x1b[5~")
	if !waitUntil(t, func() bool { return extractCursorPosition(tf.Screen()) <= 10 }, 2*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor position <= 10 after PageUp from end, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}
}

// TestPageUpAtStart tests that PageUp from the top of a paged list returns to
// position 1 and doesn't go below 1, even when invoked multiple times.
//
// Strategy: PageDown first to a known-non-1 position (proves the navigator
// is wired and the test setup is correct), then PageUp twice and verify we
// landed back at position 1 — not 0, not negative. A previous version of
// this test only sent two PageUps at the start and checked pos == 1, which
// passed even when PageUp() panicked: the assertion was indistinguishable
// from "cursor never moved".
func TestPageUpAtStart(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerManyApps(35)
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

	navigateToApps(t, tf)

	if !waitForCursorPositionExact(tf, 1, 2*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 initially, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// PageDown to move off position 1 — proves keystroke routing works.
	_ = tf.Send("\x1b[6~")
	if !waitForCursorPosition(tf, 10, 3*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor >= 10 after PageDown (sanity check), got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// PageUp three times. The first PageUp must move the cursor *back toward*
	// position 1; subsequent PageUps must clamp to 1 (not go negative).
	_ = tf.Send("\x1b[5~\x1b[5~\x1b[5~")
	if !waitForCursorPositionExact(tf, 1, 3*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 after PageUp from middle, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}
}

// TestPageUpDownRoundTrip tests that PageDown followed by PageUp returns to original position
func TestPageUpDownRoundTrip(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerManyApps(35)
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

	navigateToApps(t, tf)

	// Verify initial state - cursor at position 1
	if !waitForCursorPositionExact(tf, 1, 3*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 initially, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// PageDown — cursor should move to a higher position (at least 10).
	_ = tf.Send("\x1b[6~")
	if !waitForCursorPosition(tf, 10, 5*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor position >= 10 after PageDown, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// PageUp — should return to start (position 1).
	_ = tf.Send("\x1b[5~")

	// Should be back at position 1
	if !waitForCursorPositionExact(tf, 1, 5*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 after PageDown+PageUp round trip, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}
}
