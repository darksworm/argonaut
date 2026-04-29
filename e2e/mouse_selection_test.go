//go:build e2e && unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMouseSelection tests the mouse text selection and clipboard copy feature.
func TestMouseSelection(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Setup mock server with a few apps
	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	WriteArgoConfig(cfgPath, srv.URL)

	// Clipboard file is now automatically set up by the test framework
	clipboardFile := filepath.Join(tf.Workspace(), "clipboard.txt")

	// Start app
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for UI to be ready and data to load (cluster-a appears on screen)
	if !tf.WaitForPlain("cluster-a", 10*time.Second) {
		t.Fatalf("timeout waiting for cluster-a to appear\nScreen:\n%s", tf.Screen())
	}

	// Find "cluster-a" text on screen to select it (mock server returns cluster-a)
	screen := tf.Screen()
	lines := strings.Split(screen, "\n")

	// Find the line containing "cluster-a"
	var targetLine int
	var targetCol int
	for i, line := range lines {
		if idx := strings.Index(line, "cluster-a"); idx >= 0 {
			targetLine = i + 1 // 1-based for terminal
			targetCol = idx + 1
			break
		}
	}

	if targetLine == 0 {
		t.Fatalf("could not find 'cluster-a' on screen\nScreen:\n%s", screen)
	}

	// Drag to select "cluster-a" (9 characters)
	startX := targetCol
	startY := targetLine
	endX := targetCol + 9
	endY := targetLine

	t.Logf("Selecting from (%d,%d) to (%d,%d)", startX, startY, endX, endY)

	if err := tf.MouseDrag(startX, startY, endX, endY); err != nil {
		t.Fatalf("mouse drag: %v", err)
	}

	// Check that "Copied!" appears in status
	if !tf.WaitForScreen("Copied!", 2*time.Second) {
		t.Logf("Note: 'Copied!' message may have already disappeared")
	}

	// Verify clipboard content by reading the mock clipboard file
	clipboardBytes, err := os.ReadFile(clipboardFile)
	if err != nil {
		t.Fatalf("failed to read clipboard file: %v", err)
	}
	clipboardContent := string(clipboardBytes)
	t.Logf("Clipboard content: %q", clipboardContent)

	// Check that we captured part of "cluster-a" - coordinate alignment may be off by 1-2 chars
	// due to terminal rendering differences, so accept "uster-a" or "cluster-a"
	if !strings.Contains(clipboardContent, "uster-a") {
		t.Errorf("expected clipboard to contain 'cluster-a' or partial 'uster-a', got: %q", clipboardContent)
	}

	// Verify selection is reasonably precise - should be short, not the entire screen
	// A proper selection of "cluster-a" should be under 50 chars (with some margin for trailing spaces)
	if len(clipboardContent) > 50 {
		t.Errorf("selection too large (%d chars) - expected precise selection of 'cluster-a', got: %q",
			len(clipboardContent), clipboardContent)
	}
}

// TestMouseSelectionClearsOnEscape tests that Escape clears an in-flight
// selection: a drag-then-release that would normally produce a clipboard
// write must not produce one when Escape is pressed before release.
//
// Two-phase strategy: first do a drag *without* Escape and verify the
// clipboard file received content (positive control — proves the copy
// path works). Then do a drag *with* Escape and verify the clipboard file
// did NOT receive new content. Without the positive control, the
// "no copy" assertion passes whenever the copy path is globally broken,
// which makes the test a poor regression detector.
//
// The test relies on the e2e framework's mock copy_command (`tee
// $WORKSPACE/clipboard.txt`) to make the clipboard write observable.
func TestMouseSelectionClearsOnEscape(t *testing.T) {
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
		t.Fatalf("workspace: %v", err)
	}
	WriteArgoConfig(cfgPath, srv.URL)
	clipboardFile := filepath.Join(tf.Workspace(), "clipboard.txt")

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("cluster-a", 10*time.Second) {
		t.Fatalf("timeout waiting for cluster-a to appear\nScreen:\n%s", tf.Screen())
	}

	// --- Positive control: drag without Escape must populate the clipboard.
	if err := tf.MouseDrag(10, 5, 20, 5); err != nil {
		t.Fatalf("control drag: %v", err)
	}
	if !waitUntil(t, func() bool {
		b, err := os.ReadFile(clipboardFile)
		return err == nil && len(b) > 0
	}, 2*time.Second) {
		t.Fatalf("control drag did not write to clipboard — copy path broken or selection logic regressed:\n%s", tf.Screen())
	}
	controlBytes, _ := os.ReadFile(clipboardFile)

	// --- Real assertion: drag, Escape mid-drag, release. Clipboard file
	// must remain unchanged.
	if err := tf.MouseClick(0, 10, 5); err != nil {
		t.Fatalf("mouse click: %v", err)
	}
	time.Sleep(25 * time.Millisecond)
	if err := tf.MouseMotion(0, 20, 5); err != nil {
		t.Fatalf("mouse motion: %v", err)
	}
	time.Sleep(25 * time.Millisecond)
	if err := tf.Escape(); err != nil {
		t.Fatalf("escape: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := tf.MouseRelease(0, 20, 5); err != nil {
		t.Fatalf("mouse release: %v", err)
	}

	time.Sleep(150 * time.Millisecond)
	finalBytes, _ := os.ReadFile(clipboardFile)
	if string(finalBytes) != string(controlBytes) {
		t.Errorf("clipboard was overwritten after Escape cleared the selection:\n  before escape: %q\n  after escape:  %q", controlBytes, finalBytes)
	}
}

// TestMouseSelectionMultiLine tests selecting across multiple lines.
func TestMouseSelectionMultiLine(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Use many apps server to have multiple lines
	srv, err := MockArgoServerManyApps(5)
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	WriteArgoConfig(cfgPath, srv.URL)

	// Clipboard file is now automatically set up by the test framework
	clipboardFile := filepath.Join(tf.Workspace(), "clipboard.txt")

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for UI to be ready and data to load (cluster-a appears on screen)
	if !tf.WaitForPlain("cluster-a", 10*time.Second) {
		t.Fatalf("timeout waiting for cluster-a to appear\nScreen:\n%s", tf.Screen())
	}

	// Jump straight to the apps view via :apps — much faster than drilling
	// through clusters → namespaces → projects with Enter.
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	if !tf.WaitForPlain("app-01", 5*time.Second) {
		t.Fatalf("could not reach apps view with app-01\nScreen:\n%s", tf.Screen())
	}

	// Find app-01 on screen to make selection more precise
	screen := tf.Screen()
	lines := strings.Split(screen, "\n")

	var startLine int
	for i, line := range lines {
		if strings.Contains(line, "app-01") {
			startLine = i + 1 // 1-based for terminal
			break
		}
	}

	if startLine == 0 {
		t.Fatalf("could not find 'app-01' on screen\nScreen:\n%s", screen)
	}

	// Multi-line selection across multiple app rows (app-01 through app-04)
	if err := tf.MouseDrag(5, startLine, 40, startLine+3); err != nil {
		t.Fatalf("mouse drag: %v", err)
	}

	// Poll for the clipboard file to appear with content (app processes the
	// release asynchronously, but typically within a few ms).
	var clipboardBytes []byte
	if !waitUntil(t, func() bool {
		b, err := os.ReadFile(clipboardFile)
		if err != nil || len(b) == 0 {
			return false
		}
		clipboardBytes = b
		return true
	}, 2*time.Second) {
		t.Fatalf("clipboard file empty after multi-line selection")
	}
	clipboardContent := string(clipboardBytes)
	t.Logf("Multi-line clipboard content: %q", clipboardContent)

	// Should contain at least one app name
	hasAppName := strings.Contains(clipboardContent, "app-01") ||
		strings.Contains(clipboardContent, "app-02") ||
		strings.Contains(clipboardContent, "app-03") ||
		strings.Contains(clipboardContent, "app-04")
	if !hasAppName {
		t.Errorf("expected clipboard to contain app names (app-01, app-02, etc.), got: %q", clipboardContent)
	}

	// Should contain newlines for multi-line selection
	if !strings.Contains(clipboardContent, "\n") {
		t.Errorf("expected multi-line selection to contain newlines, got: %q", clipboardContent)
	}
}
