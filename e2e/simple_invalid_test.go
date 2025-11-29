//go:build e2e && unix

package main

import (
	"testing"
	"time"
)

func TestSimpleInvalidCommand(t *testing.T) {
	// Remove t.Parallel() to avoid race conditions with other tests
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

	// Wait for initial view to be ready
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Enter command mode
	_ = tf.Send(":")
	if !tf.WaitForPlain("> ", 2*time.Second) {
		t.Fatal("command bar not ready")
	}

	// Send an invalid command
	t.Logf("Sending invalid command 'invalidcmd'")
	_ = tf.Send("invalidcmd")
	_ = tf.Enter()

	// Wait longer for the UI to process the invalid command
	time.Sleep(2 * time.Second)

	// Capture current screen state for debugging
	currentScreen := tf.SnapshotPlain()
	t.Logf("Screen after invalid command:\n%s", currentScreen)

	// Should show helpful message for invalid command
	if !tf.WaitForPlain("unknown command", 5*time.Second) {
		t.Errorf("Expected 'unknown command' message for invalid command")
		t.Logf("Final screen state:\n%s", tf.SnapshotPlain())
	}

	if !tf.WaitForPlain("see :help", 3*time.Second) {
		t.Errorf("Expected ':help' suggestion for invalid command")
		if !t.Failed() {
			t.Logf("Current screen state:\n%s", tf.SnapshotPlain())
		}
	}

	// The invalid command should still be visible in the input
	if !tf.WaitForPlain("invalidcmd", 3*time.Second) {
		t.Errorf("Expected invalid command to remain visible")
		if !t.Failed() {
			t.Logf("Current screen state:\n%s", tf.SnapshotPlain())
		}
	}
}