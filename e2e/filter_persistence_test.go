//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

// TestFilterPersistsAfterResourcesAndK9s verifies that an apps-view filter
// survives a round-trip into the resources (tree) view and through k9s.
//
// Reproduction:
//  1. apps view, filter "dem"
//  2. Enter on the filtered app → tree view
//  3. K on a resource → k9s
//  4. exit k9s, Esc back to apps view
//  5. filter must still be visible
func TestFilterPersistsAfterResourcesAndK9s(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithResources()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	mockK9s, _ := createMockK9s(t, tf.workspace, 0)
	kubeconfigPath := setupSingleContextKubeconfig(t, tf.workspace, "cluster-a")

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"ARGONAUT_K9S_COMMAND="+mockK9s,
		"KUBECONFIG="+kubeconfigPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	// Navigate to apps view.
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForPlain("demo", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("apps view did not load")
	}

	// Apply filter "dem".
	if err := tf.OpenSearch(); err != nil {
		t.Fatalf("open search: %v", err)
	}
	_ = tf.Send("dem")
	_ = tf.Enter()

	// Filter chip in the status bar is rendered as "<apps:dem>".
	if !tf.WaitForScreen("<apps:dem>", 3*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("filter chip <apps:dem> not visible after applying filter")
	}

	// Open resources for the (now filtered) app.
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view did not load")
	}

	// Drill into a resource and launch k9s. Kubeconfig context name matches
	// the ArgoCD cluster name ("cluster-a"), so the picker is skipped.
	_ = tf.Send("jK")

	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s did not launch")
	}

	// Mock k9s exits after ~0.2s. Wait for argonaut to regain input routing
	// (confirmed by the command bar opening) before sending Esc. OpenCommand
	// is not usable here: it scans the cumulative output ring, which already
	// contains a command-bar frame from line 55, so it can return before the
	// app is actually accepting input again. Poll the live screen instead.
	// Keystrokes sent before the PTY handoff completes are dropped, so keep
	// re-sending ":" until the bar shows up. Extra ":" presses once the bar
	// is open just type into it and are discarded by the Esc below.
	openDeadline := time.Now().Add(10 * time.Second)
	for {
		_ = tf.Send(":")
		time.Sleep(100 * time.Millisecond)
		if strings.Contains(tf.Screen(), "│ > ") {
			break
		}
		if time.Now().After(openDeadline) {
			t.Log(tf.Screen())
			t.Fatal("argonaut input not restored after k9s: command bar not on live screen")
		}
	}
	_ = tf.Send("\x1b") // close command bar

	// The app debounces esc presses 100ms apart by *its* clock, so sleeping
	// between sends is only safe once we've seen the app process the first
	// esc. Wait for the command bar to disappear from the live screen, then
	// the debounce window is anchored: the app handled esc #1 no later than
	// the frame we just observed.
	closeDeadline := time.Now().Add(10 * time.Second)
	for strings.Contains(tf.Screen(), "│ > ") {
		if time.Now().After(closeDeadline) {
			t.Fatal("command bar did not close after esc")
		}
		time.Sleep(50 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)

	// Esc back to apps view.
	_ = tf.Send("\x1b")

	// We're back in apps view AND filter should still be applied.
	// WaitForPlain scans the cumulative output ring, which includes stale
	// "<apps:dem>" frames from before the tree view, so assert against the
	// live screen instead.
	deadline := time.Now().Add(3 * time.Second)
	var screen string
	for time.Now().Before(deadline) {
		screen = tf.Screen()
		if strings.Contains(screen, "<apps:dem>") {
			return // pass
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Log(screen)
	t.Fatal("filter <apps:dem> was lost after returning from tree view")
}
