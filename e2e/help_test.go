//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

func TestHelpModalOpensAndQuits(t *testing.T) {
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

	// Wait for app to fully load - need to see clusters loaded, not just "Ready"
	if ok := tf.WaitForPlain("cluster-a", 3*time.Second); !ok {
		t.Log(tf.SnapshotPlain())
		t.Fatal("did not see cluster data loaded")
	}

	// Enter help. (Earlier WaitForPlain on cluster-a is enough — by the
	// time clusters render, the connecting overlay has been dismissed.)
	if err := tf.Send("?"); err != nil {
		t.Fatalf("send ?: %v", err)
	}
	if !tf.WaitForPlain("Press ?, q or Esc to close", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("help not shown")
	}

	// Verify that theme command shows up in help
	helpSnapshot := tf.SnapshotPlain()
	if !strings.Contains(helpSnapshot, ":theme") {
		t.Log(helpSnapshot)
		t.Fatal("theme command not found in help")
	}

	// Quit help and exit
	_ = tf.Send("q")
	_ = tf.CtrlC()
}
