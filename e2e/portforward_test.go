//go:build e2e && unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestPortForward_HappyPath verifies the complete port-forward flow:
// - Config detection (server: port-forward)
// - kubectl mock finds pod and outputs port
// - App connects to mock ArgoCD server via that port
// - App displays application list normally
func TestPortForward_HappyPath(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Setup mock ArgoCD server (this is what we'll actually connect to)
	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Write port-forward ArgoCD config
	if err := WriteArgoConfigPortForward(cfgPath, "test-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Create mock kubectl - report the mock server's port so traffic routes correctly
	opts := DefaultMockKubectlOptions()
	opts.LocalPort = extractPortFromURL(srv.URL)

	mockKubectl, argsFile := createMockKubectl(t, tf.workspace, opts)

	// Start app with mock kubectl first in PATH
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"PATH="+filepath.Dir(mockKubectl)+":"+os.Getenv("PATH"),
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Should see cluster from mock server (proves tunnel worked)
	// The app initially shows clusters view with "cluster-a" from our mock server
	if !tf.WaitForPlain("cluster-a", 10*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected to see 'cluster-a' after port-forward established")
	}

	// Verify kubectl was called correctly
	args := readMockKubectlArgs(t, argsFile)
	if len(args) < 2 {
		t.Fatalf("expected at least 2 kubectl calls (version + get pods), got %d: %v", len(args), args)
	}

	// Verify get pods was called with correct selector
	foundGetPods := false
	for _, arg := range args {
		if strings.Contains(arg, "get") &&
			strings.Contains(arg, "pods") &&
			strings.Contains(arg, "-n argocd") &&
			strings.Contains(arg, "app.kubernetes.io/name=argocd-server") {
			foundGetPods = true
			break
		}
	}
	if !foundGetPods {
		t.Errorf("expected kubectl get pods with argocd namespace and selector, got: %v", args)
	}

	// Verify port-forward was called
	foundPortForward := false
	for _, arg := range args {
		if strings.Contains(arg, "port-forward") &&
			strings.Contains(arg, "argocd-server") {
			foundPortForward = true
			break
		}
	}
	if !foundPortForward {
		t.Errorf("expected kubectl port-forward call, got: %v", args)
	}
}

// TestPortForward_NoPodFound verifies error handling when no ArgoCD server pod is found.
// The app should show helpful troubleshooting tips.
func TestPortForward_NoPodFound(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	if err := WriteArgoConfigPortForward(cfgPath, "test-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Create mock kubectl that returns empty pod list
	opts := DefaultMockKubectlOptions()
	opts.PodName = "" // No pods found

	mockKubectl, _ := createMockKubectl(t, tf.workspace, opts)

	// Start app with mock kubectl first in PATH
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"PATH="+filepath.Dir(mockKubectl)+":"+os.Getenv("PATH"),
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for error output - the app should exit with error
	// Give it time to fail
	time.Sleep(3 * time.Second)

	snapshot := tf.SnapshotPlain()

	// Should have troubleshooting tips or error about no pods
	if !strings.Contains(snapshot, "Troubleshooting") && !strings.Contains(snapshot, "no ready pods") {
		t.Logf("Output: %s", snapshot)
		t.Error("expected troubleshooting tips or 'no ready pods' error message")
	}
}

// TestPortForward_Reconnection verifies that the app handles disconnection and reconnects.
// The mock kubectl port-forward exits after a delay, simulating a pod restart.
func TestPortForward_Reconnection(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Setup mock ArgoCD server
	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	if err := WriteArgoConfigPortForward(cfgPath, "test-token"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Create mock kubectl that exits after 3 seconds (simulates pod restart)
	opts := DefaultMockKubectlOptions()
	opts.LocalPort = extractPortFromURL(srv.URL)
	opts.PFExitAfter = 3 // Exit after 3 seconds

	mockKubectl, argsFile := createMockKubectl(t, tf.workspace, opts)

	// Start app
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"PATH="+filepath.Dir(mockKubectl)+":"+os.Getenv("PATH"),
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// First, verify app starts successfully (cluster-a appears in clusters view)
	if !tf.WaitForPlain("cluster-a", 10*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected to see 'cluster-a' initially")
	}

	// Wait for reconnection to happen (kubectl exits after 3s, then reconnect attempt)
	// The reconnect delay is 2 seconds, plus time for re-establishing
	time.Sleep(8 * time.Second)

	// After reconnection, the app should still be functional
	// Check that port-forward was called multiple times (initial + reconnection)
	args := readMockKubectlArgs(t, argsFile)

	portForwardCount := 0
	for _, arg := range args {
		if strings.Contains(arg, "port-forward") {
			portForwardCount++
		}
	}

	if portForwardCount < 2 {
		t.Errorf("expected at least 2 port-forward calls (initial + reconnection), got %d. Args: %v", portForwardCount, args)
	}

	// App should still show content (either reconnected successfully or showing error)
	snapshot := tf.SnapshotPlain()
	// Either the app recovered and shows apps, or it shows reconnection-related output
	if len(snapshot) < 10 {
		t.Errorf("expected app to have some output after reconnection attempt, got: %s", snapshot)
	}
}
