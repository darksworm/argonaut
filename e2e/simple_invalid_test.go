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

	// Wait for ready state
	if !tf.WaitForPlain("> ", 5*time.Second) {
		t.Fatal("command bar not ready")
	}

	// Enter command mode and send an invalid command
	_ = tf.Send(":invalidcmd")
	_ = tf.Enter()

	// Wait longer for the UI to process the invalid command
	time.Sleep(2 * time.Second)

	// Should show helpful message for invalid command
	if !tf.WaitForPlain("unknown command", 5*time.Second) {
		t.Errorf("Expected 'unknown command' message for invalid command")
	}

	if !tf.WaitForPlain("see :help", 3*time.Second) {
		t.Errorf("Expected ':help' suggestion for invalid command")
	}

	// The invalid command should still be visible in the input
	if !tf.WaitForPlain("invalidcmd", 3*time.Second) {
		t.Errorf("Expected invalid command to remain visible")
	}
}