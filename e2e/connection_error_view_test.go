//go:build e2e && unix

package main

import (
    "fmt"
    "testing"
    "time"
)

// When server is down, show connection error screen with helpful tip
func TestConnectionErrorViewCopy(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    // Point to a port that is likely closed to trigger a quick connection error
    baseURL := "http://127.0.0.1:9"
    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfig(cfgPath, baseURL); err != nil {
        t.Fatalf("write config: %v", err)
    }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
        t.Fatalf("start app: %v", err)
    }

    // Expect connection error view, and the tip about context/login; and not to include redundant server line
    if !tf.WaitForPlain("Connection Error", 4*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected Connection Error header")
    }
    if !tf.WaitForPlain("Tip: Ensure you are using the correct Argo CD context", 2*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected context tip")
    }
    if !tf.WaitForPlain("argocd login", 2*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected argocd login hint")
    }
    notExpected := fmt.Sprintf("ArgoCD Server: %s", baseURL)
    if tf.WaitForPlain(notExpected, 1*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("did not expect explicit server line in connection error view")
    }
}

