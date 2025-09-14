//go:build e2e && unix

package main

import (
    "testing"
    "time"
)

func TestHelpModalOpensAndQuits(t *testing.T) {
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, err := MockArgoServer()
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfig(cfgPath, srv.URL); err != nil { t.Fatalf("write config: %v", err) }

    if err := tf.StartApp("ARGOCD_CONFIG="+cfgPath); err != nil {
        t.Fatalf("start app: %v", err)
    }

    // Wait for banner
    if ok := tf.WaitForPlain("Argo", 3*time.Second); !ok {
        t.Log(tf.SnapshotPlain())
        t.Fatal("did not see banner")
    }

    // Enter help
    if err := tf.Send("?"); err != nil { t.Fatalf("send ?: %v", err) }
    if !tf.WaitForPlain("Help", 2*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("help not shown")
    }

    // Quit help and exit
    _ = tf.Send("q")
    _ = tf.CtrlC()
}

