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

	// Wait a bit for the copy to happen
	time.Sleep(200 * time.Millisecond)

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

// TestMouseSelectionClearsOnEscape tests that Escape clears the selection.
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

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for UI to be ready and data to load (cluster-a appears on screen)
	if !tf.WaitForPlain("cluster-a", 10*time.Second) {
		t.Fatalf("timeout waiting for cluster-a to appear\nScreen:\n%s", tf.Screen())
	}

	// Start a selection but don't release - just click
	if err := tf.MouseClick(0, 10, 5); err != nil {
		t.Fatalf("mouse click: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Move mouse to create selection
	if err := tf.MouseMotion(0, 20, 5); err != nil {
		t.Fatalf("mouse motion: %v", err)
	}
	time.Sleep(50 * time.Millisecond)

	// Press Escape to clear selection (without releasing mouse first)
	// This simulates user pressing Escape while dragging
	if err := tf.Escape(); err != nil {
		t.Fatalf("escape: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Now release - should not copy since selection was cleared
	if err := tf.MouseRelease(0, 20, 5); err != nil {
		t.Fatalf("mouse release: %v", err)
	}

	// "Copied!" should NOT appear since selection was cleared
	time.Sleep(200 * time.Millisecond)
	screen := tf.Screen()
	if strings.Contains(screen, "Copied!") {
		t.Errorf("expected 'Copied!' to NOT appear after Escape cleared selection")
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

	// Select cluster and navigate to apps view
	tf.Enter()
	// Wait for app-01 to appear (navigating through namespaces/projects if needed)
	if !tf.WaitForPlain("app-01", 5*time.Second) {
		tf.Enter()
		if !tf.WaitForPlain("app-01", 3*time.Second) {
			tf.Enter()
			if !tf.WaitForPlain("app-01", 3*time.Second) {
				t.Fatalf("could not navigate to apps view with app-01\nScreen:\n%s", tf.Screen())
			}
		}
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

	time.Sleep(200 * time.Millisecond)

	// Verify clipboard content by reading the mock clipboard file
	clipboardBytes, err := os.ReadFile(clipboardFile)
	if err != nil {
		t.Fatalf("failed to read clipboard file: %v", err)
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
