//go:build e2e && unix

package main

import (
    "testing"
    "time"
)

// Drill down: clusters -> namespaces -> projects -> apps, then back up
func TestNavigateDownAndUpAllLevels(t *testing.T) {
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, err := MockArgoServerAuth("valid-token")
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil {
        t.Fatalf("write config: %v", err)
    }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
        t.Fatalf("start app: %v", err)
    }

    // Clusters view should show cluster-a
    if !tf.WaitForPlain("cluster-a", 4*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected cluster-a in clusters view")
    }

    // Enter to namespaces
    _ = tf.Enter()
    if !tf.WaitForPlain("default", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected namespace 'default'")
    }

    // Enter to projects
    _ = tf.Enter()
    if !tf.WaitForPlain("demo", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected project 'demo'")
    }

    // Enter to apps; expect app row with name 'demo'
    _ = tf.Enter()
    if !tf.WaitForPlain("demo", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected app 'demo' in apps view")
    }

    // Back up to projects
    _ = tf.Send("\x1b") // Esc
    if !tf.WaitForPlain("demo", 3*time.Second) { // project name still 'demo'
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected project 'demo' after Esc")
    }
    // Back up to namespaces
    _ = tf.Send("\x1b")
    if !tf.WaitForPlain("default", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected namespace 'default' after Esc")
    }
    // Back up to clusters
    _ = tf.Send("\x1b")
    if !tf.WaitForPlain("cluster-a", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected cluster-a after Esc")
    }
}

