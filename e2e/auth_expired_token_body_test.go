//go:build e2e && unix

package main

import (
    "testing"
    "time"
)

// When API returns 401 with structured body containing "token is expired",
// the UI should show the Authentication Required view (not generic API error)
func TestExpiredTokenStructuredBodyShowsAuthRequired(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, err := MockArgoServerExpiredToken()
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
        t.Fatal("expected AUTHENTICATION REQUIRED for expired token structured body")
    }
}

