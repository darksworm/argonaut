//go:build e2e && unix

package main

import (
    "testing"
    "time"
)

// When server responds 403, app should show auth-required screen
func TestForbiddenShowsAuthRequired(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, err := MockArgoServerForbidden()
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
        t.Fatalf("write config: %v", err)
    }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
        t.Fatalf("start app: %v", err)
    }

    if !tf.WaitForPlain("AUTHENTICATION REQUIRED", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected AUTHENTICATION REQUIRED on 403")
    }
}

