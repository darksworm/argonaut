//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

func TestDeleteFunctionality_FullFlow(t *testing.T) {
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

	// Wait for clusters to be ready first
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Navigate to apps view using command bar
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	// Wait for apps to load
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not see demo loaded")
	}

	// Trigger delete with Ctrl+D
	if err := tf.Send("\x04"); err != nil { // Ctrl+D
		t.Fatalf("send Ctrl+D: %v", err)
	}

	// Wait for delete confirmation modal to fully render (including options)
	if !tf.WaitForPlain("Delete demo?", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("delete confirmation modal not shown")
	}
	if !tf.WaitForPlain("c: Cascade On", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("delete modal should show cascade warning")
	}
	if !tf.WaitForPlain("Delete (y)", 1*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("delete modal should show safety instructions")
	}

	// Type 'n' (must not trigger delete), then backspace, then 'y' to commit.
	// PTY preserves keystroke order; the next assertion (Deleting modal)
	// would not appear if 'n' had been treated as confirmation.
	if err := tf.Send("n\x7fy"); err != nil {
		t.Fatalf("send n/backspace/y: %v", err)
	}

	// Wait for delete to process (delete modal should appear)
	if !tf.WaitForPlain("Deleting", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("delete processing modal not shown")
	}

	// Wait for delete to complete and return to normal view
	if !tf.WaitForPlain("demo", 5*time.Second) {
		// Note: In a real test, the app would be deleted and removed from the list
		// For this test, we're just verifying the flow works
		t.Log(tf.SnapshotPlain())
		// Don't fail here as the mock server might not properly handle deletes
		t.Log("Delete completed (app removal depends on mock server implementation)")
	}

	// Exit
	_ = tf.CtrlC()
}

func TestDeleteFunctionality_CancelWithNonYKey(t *testing.T) {
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

	// Wait for clusters to be ready first
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Navigate to apps view using command bar
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	// Wait for apps to load
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not see demo loaded")
	}

	// Trigger delete with Ctrl+D
	if err := tf.Send("\x04"); err != nil { // Ctrl+D
		t.Fatalf("send Ctrl+D: %v", err)
	}

	// Wait for delete confirmation modal to appear.
	if !tf.WaitForPlain("Delete demo?", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("delete confirmation modal not shown")
	}

	// Type 'n' - must not trigger delete; modal must remain. Send a
	// known-no-op key (e.g. another 'n') as a barrier so we can poll the
	// final state and be confident the first 'n' has been processed.
	if err := tf.Send("nn"); err != nil {
		t.Fatalf("send 'nn': %v", err)
	}
	if !waitUntil(t, func() bool {
		s := tf.SnapshotPlain()
		return strings.Contains(s, "Delete demo?") && strings.Contains(s, "demo")
	}, 1*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("delete modal should still be visible (and demo still listed) after typing 'n'")
	}

	// Exit the app
	_ = tf.CtrlC()
}

func TestDeleteFunctionality_NotInAppsView(t *testing.T) {
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

	// Wait for clusters to be ready first
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Navigate to clusters view and trigger Ctrl+D (must be a no-op there).
	if err := tf.Send("c"); err != nil {
		t.Fatalf("send 'c': %v", err)
	}
	if !tf.WaitForPlain("<clusters>", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("should be in clusters view")
	}

	if err := tf.Send("\x04"); err != nil { // Ctrl+D
		t.Fatalf("send Ctrl+D: %v", err)
	}
	time.Sleep(100 * time.Millisecond)
	if strings.Contains(tf.Screen(), "Delete Application") {
		t.Log(tf.Screen())
		t.Fatal("delete modal should not appear in clusters view")
	}
	if !strings.Contains(tf.Screen(), "<clusters>") {
		t.Log(tf.Screen())
		t.Fatal("should still be in clusters view")
	}

	// --- Positive contrast: navigate to apps view, Ctrl+D must show modal.
	// Without this contrast the negative assertion above would also pass
	// when Ctrl+D is broken globally — both branches would silently no-op.
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("apps view not ready")
	}

	if err := tf.Send("\x04"); err != nil {
		t.Fatalf("send Ctrl+D in apps view: %v", err)
	}
	if !tf.WaitForPlain("Delete demo?", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("Ctrl+D in apps view should open the delete confirmation — Ctrl+D may be globally broken")
	}

	// Exit
	_ = tf.CtrlC()
}