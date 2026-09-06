//go:build e2e && unix

package main

import (
	"encoding/json"
	"testing"
	"time"
)

// startAppsView boots the TUI against a sync-recording mock and navigates to
// the applications list, which is where the sync modal is opened from.
func startAppsView(t *testing.T) (*TUITestFramework, *SyncRecorder) {
	t.Helper()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, rec, err := MockArgoServerSync("valid-token")
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigWithToken(cfgPath, srv.URL, "valid-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Fatal("clusters not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("ns default")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo", 3*time.Second) {
		t.Fatal("namespaces not ready")
	}
	if err := tf.OpenCommand(); err != nil {
		t.Fatal(err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()
	if !tf.WaitForPlain("demo2", 3*time.Second) {
		t.Fatal("apps not ready")
	}
	return tf, rec
}

// syncBody is the request body argonaut posted to /sync, decoded.
func syncBody(t *testing.T, rec *SyncRecorder, want int) []map[string]any {
	t.Helper()
	if !waitUntil(t, func() bool { return rec.len() == want }, 3*time.Second) {
		t.Fatalf("expected %d sync calls, got %d", want, rec.len())
	}
	bodies := make([]map[string]any, 0, want)
	for _, call := range rec.Calls {
		var body map[string]any
		if err := json.Unmarshal([]byte(call.Body), &body); err != nil {
			t.Fatalf("decoding the sync body for %q: %v (%s)", call.Name, err, call.Body)
		}
		bodies = append(bodies, body)
	}
	return bodies
}

// hookForce reports the force flag Argo CD will read, and whether the request
// used the apply strategy — which would silently skip the app's sync hooks.
func hookForce(t *testing.T, body map[string]any) (force, applyOnly bool) {
	t.Helper()
	strategy, ok := body["strategy"].(map[string]any)
	if !ok {
		return false, false
	}
	if _, applyOnly = strategy["apply"]; applyOnly {
		return false, true
	}
	hook, _ := strategy["hook"].(map[string]any)
	force, _ = hook["force"].(bool)
	return force, false
}

func TestSyncSingleApp_WithForce_SendsHookForce(t *testing.T) {
	t.Parallel()
	tf, rec := startAppsView(t)

	_ = tf.Send("s") // sync modal for the app under the cursor
	if !tf.WaitForScreen("Force", 3*time.Second) {
		t.Fatalf("sync modal never showed the force option:\n%s", tf.Screen())
	}
	_ = tf.Send("f")
	_ = tf.Send("y")
	if !tf.WaitForScreen("Force sync", 3*time.Second) {
		t.Fatalf("force confirmation never appeared:\n%s", tf.Screen())
	}
	_ = tf.Send("y")

	force, applyOnly := hookForce(t, syncBody(t, rec, 1)[0])
	if applyOnly {
		t.Error("a forced sync used the apply strategy, which skips sync hooks")
	}
	if !force {
		t.Errorf("expected strategy.hook.force=true, got %s", rec.Calls[0].Body)
	}
}

func TestSyncMultipleApps_WithForce_SendsHookForceForEach(t *testing.T) {
	t.Parallel()
	tf, rec := startAppsView(t)

	// Select both apps: space, down, space
	_ = tf.Send(" ")
	_ = tf.Send("j")
	_ = tf.Send(" ")

	_ = tf.Send("s")
	if !tf.WaitForScreen("Force", 3*time.Second) {
		t.Fatalf("sync modal never showed the force option:\n%s", tf.Screen())
	}
	_ = tf.Send("f")
	_ = tf.Send("y")
	if !tf.WaitForScreen("Force sync", 3*time.Second) {
		t.Fatalf("force confirmation never appeared:\n%s", tf.Screen())
	}
	_ = tf.Send("y")

	for i, body := range syncBody(t, rec, 2) {
		force, applyOnly := hookForce(t, body)
		if applyOnly {
			t.Errorf("sync call %d used the apply strategy, which skips sync hooks", i)
		}
		if !force {
			t.Errorf("expected strategy.hook.force=true on call %d, got %s", i, rec.Calls[i].Body)
		}
	}
}

func TestSyncApp_WithoutForce_SendsNoStrategy(t *testing.T) {
	t.Parallel()
	tf, rec := startAppsView(t)

	_ = tf.Send("s")
	if !tf.WaitForScreen("Sync", 3*time.Second) {
		t.Fatalf("sync modal never opened:\n%s", tf.Screen())
	}
	_ = tf.Send("y")

	if strategy, ok := syncBody(t, rec, 1)[0]["strategy"]; ok {
		t.Errorf("expected no strategy on an ordinary sync, got %v", strategy)
	}
}
