//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

// When token is invalid, app should show auth-required screen
func TestInvalidTokenShowsAuthRequired(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Mock server requiring a token for userinfo (auth gate)
	srv, err := MockArgoServerAuth("valid-token")
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	// Write config with an invalid token
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "invalid-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Expect authentication required banner and instructions
	if !tf.WaitForPlain("AUTHENTICATION REQUIRED", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected AUTHENTICATION REQUIRED with invalid token")
	}

	// Check login instructions in the same snapshot to avoid sequential wait
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "argocd login") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected login instructions")
	}
}
