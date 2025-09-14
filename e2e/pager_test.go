//go:build e2e && unix

package main

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

// Validate that :logs opens OV and we can quit
func TestLogsPagerOpensAndQuits(t *testing.T) {
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    // Pre-create logs file in the working directory for the app (e2e dir)
    if err := os.MkdirAll("logs", 0o755); err != nil { t.Fatalf("mkdir logs: %v", err) }
    if err := os.WriteFile(filepath.Join("logs", "a9s.log"), []byte("hello log\nline 2\n"), 0o644); err != nil { t.Fatalf("write log: %v", err) }

    // Use simple no-auth server so app can start
    srv, err := MockArgoServer()
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfig(cfgPath, srv.URL); err != nil { t.Fatalf("write config: %v", err) }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil { t.Fatalf("start app: %v", err) }

    // Enter command mode and run :logs
    _ = tf.Send(":")
    _ = tf.Send("logs")
    _ = tf.Enter()

    // Expect to see log content in OV
    if !tf.WaitForPlain("hello log", 2*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("did not see logs content in OV")
    }

    // Quit pager with 'q'
    _ = tf.Send("q")
}

// Validate that :diff opens OV and we can quit
func TestDiffPagerOpensAndQuits(t *testing.T) {
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, rec, err := MockArgoServerSync("valid-token")
    _ = rec
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil { t.Fatalf("write config: %v", err) }

    // Force formatter to 'cat' so output is predictable regardless of delta presence
    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}, "ARGONAUT_DIFF_FORMATTER=cat"); err != nil { t.Fatalf("start app: %v", err) }

    // To apps
    if !tf.WaitForPlain("cluster-a", 3*time.Second) { t.Fatal("clusters not ready") }
    _ = tf.Enter()
    if !tf.WaitForPlain("default", 3*time.Second) { t.Fatal("namespaces not ready") }
    _ = tf.Enter()
    if !tf.WaitForPlain("demo", 3*time.Second) { t.Fatal("projects not ready") }
    _ = tf.Enter()
    if !tf.WaitForPlain("demo2", 3*time.Second) { t.Fatal("apps not ready") }

    // Enter command mode and run :diff demo
    _ = tf.Send(":")
    _ = tf.Send("diff demo")
    _ = tf.Enter()

    // Expect a diff sign '+' for desired
    if !tf.WaitForPlain("+", 3*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("did not see diff content in OV")
    }

    // Quit pager with 'q'
    _ = tf.Send("q")
}

