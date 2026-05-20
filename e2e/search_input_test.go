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
	// not be swallowed by vim-style navigation. Use WaitForScreen, not
	// WaitForPlain: the latter polls the cumulative raw output ring
	// buffer and may see ANSI cursor-move sequences interleaved between
	// the "j" and "k" renders even though the rendered screen has them
	// adjacent ("Search > jk"). WaitForScreen reads the actual terminal
	// state from the emulator, which is what the user sees.
	_ = tf.Send("jk")
	if !tf.WaitForScreen("Search > jk", 3*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("expected 'jk' to appear in search input, but it was intercepted as navigation")
	}
}
