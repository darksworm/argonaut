//go:build e2e && unix

package main

import (
	"testing"
	"time"
)

// TestPasteKeyBinding tests that paste key bindings don't crash the app
func TestPasteKeyBinding(t *testing.T) {
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Setup mock server
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

	// Wait for initial load - look for any content that indicates the app loaded
	if !tf.WaitForPlain("Ready", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("app not ready after 5 seconds")
	}

	// Test search mode paste
	t.Run("search_mode_paste", func(t *testing.T) {
		// Enter search mode
		if err := tf.Send("/"); err != nil {
			t.Fatalf("enter search mode: %v", err)
		}

		// Wait for search prompt
		if !tf.WaitForPlain("Search", 2*time.Second) {
			t.Log(tf.SnapshotPlain())
			t.Fatal("search bar not ready")
		}

		// Test paste key binding - Ctrl+V
		if err := tf.Send("\x16"); err != nil {
			t.Fatalf("send Ctrl+V: %v", err)
		}

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Exit search mode
		if err := tf.Send("\x1b"); err != nil { // ESC
			t.Fatalf("exit search mode: %v", err)
		}

		// Verify we're still responsive
		if !tf.WaitForPlain("Ready", 2*time.Second) {
			t.Log(tf.SnapshotPlain())
			t.Fatal("app became unresponsive after paste")
		}
	})

	// Test command mode paste
	t.Run("command_mode_paste", func(t *testing.T) {
		// Enter command mode
		if err := tf.Send(":"); err != nil {
			t.Fatalf("enter command mode: %v", err)
		}

		// Wait for command prompt
		if !tf.WaitForPlain(">", 2*time.Second) {
			t.Log(tf.SnapshotPlain())
			t.Fatal("command bar not ready")
		}

		// Test paste key binding - Ctrl+V
		if err := tf.Send("\x16"); err != nil {
			t.Fatalf("send Ctrl+V: %v", err)
		}

		// Give it a moment to process
		time.Sleep(100 * time.Millisecond)

		// Exit command mode
		if err := tf.Send("\x1b"); err != nil { // ESC
			t.Fatalf("exit command mode: %v", err)
		}

		// Verify we're still responsive
		if !tf.WaitForPlain("Ready", 2*time.Second) {
			t.Log(tf.SnapshotPlain())
			t.Fatal("app became unresponsive after paste")
		}
	})

	_ = tf.CtrlC()
}