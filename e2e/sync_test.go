//go:build e2e && unix

package main

import (
    "encoding/json"
    "testing"
    "time"
)

func waitUntil(t *testing.T, cond func() bool, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if cond() { return true }
        time.Sleep(25 * time.Millisecond)
    }
    return false
}

// Ensure single-app sync posts to the correct endpoint with expected body
func TestSyncSingleApp(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, rec, err := MockArgoServerSync("valid-token")
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil { t.Fatalf("write config: %v", err) }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil { t.Fatalf("start app: %v", err) }

    // Navigate deterministically via commands to apps
    if !tf.WaitForPlain("cluster-a", 3*time.Second) { t.Fatal("clusters not ready") }
    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("ns default")
    _ = tf.Enter()
    if !tf.WaitForPlain("demo", 3*time.Second) { t.Fatal("projects not ready") }
    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("apps")
    _ = tf.Enter()
    if !tf.WaitForPlain("demo2", 3*time.Second) { t.Fatal("apps not ready") }

    // Trigger sync for the highlighted app (default is the first: demo)
    _ = tf.Send("s")               // open confirm
    _ = tf.Send("\r")              // Enter to confirm "Yes"

    // Wait for one sync call
    if !waitUntil(t, func() bool { return rec.len() == 1 }, 2*time.Second) {
        t.Fatalf("expected 1 sync call, got %d\n%s", rec.len(), tf.SnapshotPlain())
    }
    call := rec.Calls[0]
    if call.Name != "demo" {
        t.Fatalf("expected sync for 'demo', got %q", call.Name)
    }
    // Body should be JSON with prune flag (default false)
    var body map[string]any
    if err := json.Unmarshal([]byte(call.Body), &body); err != nil {
        t.Fatalf("invalid body json: %v", err)
    }
    if v, ok := body["prune"].(bool); !ok || v { t.Fatalf("expected prune=false in body, got %v", body["prune"]) }
}

// Ensure multi-app sync posts for each selected app
func TestSyncMultipleApps(t *testing.T) {
    t.Parallel()
    tf := NewTUITest(t)
    t.Cleanup(tf.Cleanup)

    srv, rec, err := MockArgoServerSync("valid-token")
    if err != nil { t.Fatalf("mock server: %v", err) }
    t.Cleanup(srv.Close)

    cfgPath, err := tf.SetupWorkspace()
    if err != nil { t.Fatalf("setup workspace: %v", err) }
    if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil { t.Fatalf("write config: %v", err) }

    if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil { t.Fatalf("start app: %v", err) }

    // To apps deterministically
    if !tf.WaitForPlain("cluster-a", 3*time.Second) { t.Fatal("clusters not ready") }
    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("ns default")
    _ = tf.Enter()
    if !tf.WaitForPlain("demo", 3*time.Second) { t.Fatal("projects not ready") }
    _ = tf.Send(":")
    if !tf.WaitForPlain("> ", 2*time.Second) { t.Fatal("command bar not ready") }
    _ = tf.Send("apps")
    _ = tf.Enter()
    if !tf.WaitForPlain("demo2", 3*time.Second) { t.Fatal("apps not ready") }

    // Select two apps: space (demo), down, space (demo2)
    _ = tf.Send(" ")
    _ = tf.Send("j")
    _ = tf.Send(" ")

    // Open confirm and accept
    _ = tf.Send("s")
    _ = tf.Send("\r")

    // Expect two sync calls to /applications/<name>/sync
    if !waitUntil(t, func() bool { return rec.len() == 2 }, 2*time.Second) {
        t.Fatalf("expected 2 sync calls, got %d\n%s", rec.len(), tf.SnapshotPlain())
    }
    names := map[string]bool{}
    for _, c := range rec.Calls { names[c.Name] = true }
    if !names["demo"] || !names["demo2"] {
        t.Fatalf("expected sync calls for demo and demo2, got: %+v", names)
    }
}
