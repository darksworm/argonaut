//go:build e2e && unix

package main

import (
    "testing"
    "time"
)

func TestTreeViewOpensAndReturns(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, err := MockArgoServer()
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfig(cfgPath, srv.URL); err != nil { t.Fatalf("write config: %v", err) }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil { t.Fatalf("start app: %v", err) }

    if !tf.WaitForPlain("cluster-a", 3*time.Second) { t.Fatal("clusters not ready") }

    // Open resources (tree) for demo
    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("resources demo")
    _ = tf.Enter()

    // Expect Application root and at least a Deployment entry
    if !tf.WaitForPlain("Application [demo]", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("application root not shown")
    }
    if !tf.WaitForPlain("Deployment [", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("deployment node not shown")
    }

    // Return to clusters
    _ = tf.Send("q")
    if !tf.WaitForPlain("cluster-a", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("did not return to main view")
    }
}
