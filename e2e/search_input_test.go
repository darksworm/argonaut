//go:build e2e && unix

package main

import (
	"testing"
	"time"
)

func TestSearchInputAcceptsAllCharacters(t *testing.T) {
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

	// Wait for apps view to load
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Fatal("clusters not ready")
	}

	// Enter search mode
	if err := tf.OpenSearch(); err != nil {
		t.Fatal(err)
	}

	// Type "jk" — these characters should appear in the search input,
	// not be swallowed by vim-style navigation
	_ = tf.Send("jk")

	if !tf.WaitForPlain("jk", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'jk' to appear in search input, but it was intercepted as navigation")
	}
}
