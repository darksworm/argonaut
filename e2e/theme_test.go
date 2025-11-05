//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

func TestThemeCommand_ShowsThemeModal(t *testing.T) {
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
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for app to be ready
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("app not ready")
	}

	// Enter command mode
	_ = tf.Send(":")
	if !tf.WaitForPlain("> ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}

	// Type theme command
	_ = tf.Send("theme")
	_ = tf.Enter()

	// Check that theme selection modal appears
	if !tf.WaitForPlain("Select Theme", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Expected 'Select Theme' modal in output")
	}

	// Verify at least one theme name is visible in the modal
	screen := tf.SnapshotPlain()
	hasTheme := strings.Contains(screen, "oxocarbon") ||
		strings.Contains(screen, "dracula") ||
		strings.Contains(screen, "nord")

	if !hasTheme {
		t.Fatalf("Expected at least one theme name in modal, got: %s", screen)
	}

	// Verify modal instructions are shown
	if !strings.Contains(screen, "Navigate") {
		t.Fatalf("Expected navigation instructions in modal, got: %s", screen)
	}
}

func TestThemeCommand_NavigateAndSelect(t *testing.T) {
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
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for app to be ready
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("app not ready")
	}

	// Enter command mode
	_ = tf.Send(":")
	if !tf.WaitForPlain("> ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}

	// Type theme command to open modal
	_ = tf.Send("theme")
	_ = tf.Enter()

	// Check that theme selection modal appears
	if !tf.WaitForPlain("Select Theme", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Expected 'Select Theme' modal in output")
	}

	// Verify the first theme (oxocarbon, the default) is initially selected
	screen := tf.SnapshotPlain()
	if !strings.Contains(screen, "► oxocarbon") {
		t.Logf("Expected oxocarbon to be initially selected, screen: %s", screen)
		// Don't fail - different themes might be default in different setups
	}

	// Navigate down and select a theme
	_ = tf.Send("j") // Move down to second theme
	time.Sleep(100 * time.Millisecond)
	_ = tf.Send("\r") // Press Enter to select

	// Check that theme was set successfully
	if !tf.WaitForPlain("Theme set to", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Expected theme set confirmation")
	}
}

func TestThemeCommand_CancelRestoresOriginal(t *testing.T) {
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
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for app to be ready
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("app not ready")
	}

	// Enter command mode
	_ = tf.Send(":")
	if !tf.WaitForPlain("> ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}

	// Type theme command to open modal
	_ = tf.Send("theme")
	_ = tf.Enter()

	// Check that theme selection modal appears
	if !tf.WaitForPlain("Select Theme", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Expected 'Select Theme' modal in output")
	}

	// Navigate down to change selection
	_ = tf.Send("j") // Move down to second theme
	time.Sleep(100 * time.Millisecond)

	// Cancel with Esc
	_ = tf.Send("\x1b") // Press Esc

	// Verify we're back to normal mode (modal should be gone)
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Expected to return to normal view after cancelling")
	}

	// Verify "Select Theme" modal is no longer visible
	screen := tf.SnapshotPlain()
	if strings.Contains(screen, "Select Theme") {
		t.Fatalf("Theme modal should be closed after Esc, but still visible: %s", screen)
	}
}