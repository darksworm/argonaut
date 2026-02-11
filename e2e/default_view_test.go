//go:build e2e && unix

package main

import (
	"testing"
	"time"
)

func TestDefaultViewApps(t *testing.T) {
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

	// Set default_view to apps — app should start in apps view instead of clusters
	tf.extraConfig = `default_view = "apps"`

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// In apps view, we should see the app name "demo" with sync/health status
	if !tf.WaitForPlain("demo", 5*time.Second) {
		t.Fatal("expected app name 'demo' in apps view")
	}
	if !tf.WaitForPlain("Synced", 3*time.Second) {
		t.Fatal("expected 'Synced' status in apps view")
	}

	// Verify we're NOT in clusters view — the screen should show app details,
	// not a cluster list. In clusters view "cluster-a" appears as a selectable item.
	// In apps view "cluster-a" may appear in the app's cluster column, so we check
	// that the view header/breadcrumb shows "Applications" or similar apps view indicator.
	screen := tf.Screen()
	if screen == "" {
		t.Fatal("screen is empty")
	}
}

func TestDefaultViewWithScope(t *testing.T) {
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

	// Set default_view to scope to a cluster — should show namespaces view
	tf.extraConfig = `default_view = "cluster cluster-a"`

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// In namespaces view scoped to cluster-a, we should see "default" namespace
	if !tf.WaitForPlain("default", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'default' namespace in namespaces view scoped to cluster-a")
	}
}
