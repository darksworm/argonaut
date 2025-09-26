//go:build e2e && unix

package main

import (
    "fmt"
    "strings"
    "testing"
    "time"
)

// When server is down, show connection error screen with helpful tip
func TestConnectionErrorViewCopy(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    // Point to a port that is definitely unreachable to trigger immediate connection error
    baseURL := "http://127.0.0.1:1"
    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfig(cfgPath, baseURL); err != nil {
        t.Fatalf("write config: %v", err)
    }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
        t.Fatalf("start app: %v", err)
    }

    // Expect connection error view with all expected content
    if !tf.WaitForPlain("Connection Error", 6*time.Second) {
        t.Log(tf.SnapshotPlain())
        t.Fatal("expected Connection Error header")
    }

    // Check all expected content in a single snapshot to avoid sequential waits
    snapshot := tf.SnapshotPlain()

    if !strings.Contains(snapshot, "Tip: Ensure you are using the correct Argo CD context") {
        t.Log("Snapshot:", snapshot)
        t.Fatal("expected context tip")
    }

    if !strings.Contains(snapshot, "argocd login") {
        t.Log("Snapshot:", snapshot)
        t.Fatal("expected argocd login hint")
    }

    notExpected := fmt.Sprintf("ArgoCD Server: %s", baseURL)
    if strings.Contains(snapshot, notExpected) {
        t.Log("Snapshot:", snapshot)
        t.Fatal("did not expect explicit server line in connection error view")
    }
}

