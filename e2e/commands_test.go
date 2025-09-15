//go:build e2e && unix

package main

import (
    "testing"
    "time"
)

// Parallelized basic command navigation tests

func TestCommandCtxFromProjects(t *testing.T) {
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

    // Go to projects deterministically via commands
    if !tf.WaitForPlain("cluster-a", 3*time.Second) { t.Fatal("clusters not ready") }
    _ = tf.Send(":")
    _ = tf.Send("ns default")
    _ = tf.Enter()
    if !tf.WaitForPlain("demo", 3*time.Second) { t.Fatal("projects not ready") }

    // Use :ctx to jump back to clusters (with arg -> advance to namespaces)
    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("ctx cluster-a")
    _ = tf.Enter()

    // Expect namespaces view after applying cluster scope
    if !tf.WaitForPlain("default", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected namespaces view after :ctx cluster-a")
    }
}

func TestCommandNsFromClusters(t *testing.T) {
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

    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("ns default")
    _ = tf.Enter()

    // With namespace arg, we advance to projects view
    if !tf.WaitForPlain("demo", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected projects view after :ns default")
    }
}

func TestCommandAppFromAnywhere(t *testing.T) {
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

    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("app demo")
    _ = tf.Enter()

    // Expect apps list showing demo
    if !tf.WaitForPlain("demo", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected apps view after :app demo")
    }
}
