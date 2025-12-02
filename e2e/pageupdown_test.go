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
		_, _ = w.Write([]byte(`{"items":[` + strings.Join(items, ",") + `]}`))
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
		w.Header().Set("Content-Type", "application/json")
		// Send initial data for the first app
		_, _ = w.Write([]byte(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"app-01","namespace":"argocd"},"spec":{"project":"default","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}` + "\n"))
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

	// Wait for UI to stabilize before sending PageDown
	time.Sleep(200 * time.Millisecond)

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

	// Press PageDown multiple times to ensure we reach the end
	for i := 0; i < 3; i++ {
		_ = tf.Send("\x1b[6~")
		time.Sleep(150 * time.Millisecond)
	}

	// After multiple PageDowns, we should be able to see app-35 (last item)
	// and the app should still be responsive (not crashed)
	if !tf.WaitForPlain("app-35", 2*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("expected app-35 to be visible after PageDown to end\nSnapshot:\n%s", snap)
	}

	// Press PageDown one more time - should not crash, app should still work
	_ = tf.Send("\x1b[6~")
	time.Sleep(200 * time.Millisecond)

	// Verify app-35 is still visible (we haven't gone past it)
	if !tf.WaitForPlain("app-35", 1*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("expected app-35 to still be visible after extra PageDown at end\nSnapshot:\n%s", snap)
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

	// Go to end first using multiple PageDowns
	for i := 0; i < 3; i++ {
		_ = tf.Send("\x1b[6~")
		time.Sleep(150 * time.Millisecond)
	}

	// Verify we're at/near the end - app-35 should be visible
	if !tf.WaitForPlain("app-35", 2*time.Second) {
		snap := tf.SnapshotPlain()
		t.Fatalf("expected app-35 to be visible before PageUp\nSnapshot:\n%s", snap)
	}

	// Press PageUp
	_ = tf.Send("\x1b[5~")
	time.Sleep(200 * time.Millisecond)

	// After PageUp from the end, we should see earlier apps (around app-01 to app-10)
	// Check cursor position moved to a lower value
	screen := tf.Screen()
	pos := extractCursorPosition(screen)
	if pos > 10 {
		t.Fatalf("expected cursor position <= 10 after PageUp from end, got %d\nScreen:\n%s", pos, screen)
	}
}

// TestPageUpAtStart tests that PageUp at the start doesn't go negative
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

	// Verify we start at position 1
	if !waitForCursorPositionExact(tf, 1, 2*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 initially, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// Press PageUp at the start - should not crash or go negative, should stay at 1
	_ = tf.Send("\x1b[5~")
	time.Sleep(200 * time.Millisecond)

	screen := tf.Screen()
	pos := extractCursorPosition(screen)
	if pos != 1 {
		t.Fatalf("expected cursor to stay at position 1 after PageUp at start, got %d\nScreen:\n%s", pos, screen)
	}

	// Press PageUp again - still should stay at 1
	_ = tf.Send("\x1b[5~")
	time.Sleep(200 * time.Millisecond)

	screen = tf.Screen()
	pos = extractCursorPosition(screen)
	if pos != 1 {
		t.Fatalf("expected cursor to stay at position 1 after multiple PageUps at start, got %d\nScreen:\n%s", pos, screen)
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

	// Wait for UI to stabilize
	time.Sleep(200 * time.Millisecond)

	// PageDown
	_ = tf.Send("\x1b[6~")

	// Should move to a higher position (at least 10)
	if !waitForCursorPosition(tf, 10, 5*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor position >= 10 after PageDown, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}

	// Wait before PageUp
	time.Sleep(200 * time.Millisecond)

	// PageUp - should return to start (position 1)
	_ = tf.Send("\x1b[5~")

	// Should be back at position 1
	if !waitForCursorPositionExact(tf, 1, 5*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected cursor at position 1 after PageDown+PageUp round trip, got %d\nScreen:\n%s", extractCursorPosition(screen), screen)
	}
}
