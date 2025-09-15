//go:build e2e && unix

package main

import (
    "os"
    "testing"
    "time"
)

// When config is empty, app should show the auth-required screen with instructions
func TestEmptyConfigShowsAuthRequired(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    // Create empty config file
    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
        t.Fatalf("write empty config: %v", err)
    }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
        t.Fatalf("start app: %v", err)
    }

    // Expect authentication required banner and instructions
    if !tf.WaitForPlain("AUTHENTICATION REQUIRED", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected AUTHENTICATION REQUIRED banner")
    }
    if !tf.WaitForPlain("argocd login", 2*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected login instructions")
    }

    // Exit cleanly
    _ = tf.CtrlC()
}
