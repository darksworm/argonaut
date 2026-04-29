//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

// Parallelized basic command navigation tests

// TestCommandCtxFromProjects verifies that `:context` from projects view
// switches to the contexts picker view.
//
// Earlier this test sent `:ctx cluster-a` and asserted "default" appeared,
// but `:ctx <name>` switches Argo CD context (not cluster scope), the test
// passed an unknown context name, and "default" was already in the
// cumulative buffer from the earlier `:ns default` step — so the test
// passed even when `:ctx` was a no-op.
func TestCommandCtxFromProjects(t *testing.T) {
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

	// Drill into projects view first.
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo", 5*time.Second) {
		t.Fatal("projects not ready")
	}

	// `:context` (no arg) must open the contexts picker even from a
	// non-clusters view.
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("context")
	_ = tf.Enter()

	if !tf.WaitForScreen("<contexts>", 3*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("expected `<contexts>` breadcrumb after `:context` from projects view")
	}
}

func TestCommandNsFromClusters(t *testing.T) {
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

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()

	// With namespace arg, we advance to projects view
	if !tf.WaitForPlain("demo", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected projects view after :ns default")
	}
}

func TestCommandAppFromAnywhere(t *testing.T) {
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

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("app demo")
	_ = tf.Enter()

	// Expect apps view. Assert on the `<apps>` breadcrumb in the rendered
	// status line — `WaitForPlain("demo", ...)` would match the demo
	// substring in the cluster-view's REST JSON payload (project: demo)
	// and pass even when `:app` is a no-op.
	if !tf.WaitForScreen("<apps>", 5*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("expected `<apps>` status breadcrumb after :app demo")
	}
	if !strings.Contains(tf.Screen(), "demo") {
		t.Log(tf.Screen())
		t.Fatal("expected `demo` in apps view rows")
	}
}

// TestStreamingAppliesUpdates verifies that an SSE-delivered status change
// is applied to the apps view. The mock server returns the demo app as
// Synced via REST and then sends a single OutOfSync event over SSE — so
// the only way the apps view can display "OutOfSync" is if the streaming
// update was wired through to the model. No transition-history assertion
// (which would be timing-sensitive); just final-state assertion.
func TestStreamingAppliesUpdates(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerStreaming()
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

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	// REST returns Synced, SSE sends one OutOfSync event. If "OutOfSync"
	// ever appears, streaming worked.
	if !tf.WaitForPlain("OutOfSync", 5*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected OutOfSync status (delivered via SSE) to appear in apps view")
	}
}
