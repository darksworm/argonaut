//go:build e2e && unix

package main

import (
	"strings"
	"testing"
	"time"
)

// ---- Error Scenario Tests ----

// TestK9s_NotFound_ShowsErrorModal verifies that when k9s is not found in PATH,
// an error modal is displayed to the user.
func TestK9s_NotFound_ShowsErrorModal(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Setup mock ArgoCD server with resources
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

	// Setup kubeconfig with context matching the ArgoCD cluster name for exact match
	kubeconfigPath := setupSingleContextKubeconfig(t, tf.workspace, "cluster-a")

	// Start app with non-existent k9s command
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"ARGONAUT_K9S_COMMAND=/nonexistent/k9s",
		"KUBECONFIG="+kubeconfigPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for clusters to load first (initial view is clusters)
	if !tf.WaitForPlain("cluster-a", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	// Navigate to tree view: open resources for the demo app
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Wait for tree view to load
	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate down to a Kubernetes resource (skip Application root)
	_ = tf.Send("j") // Move to first child
	time.Sleep(200 * time.Millisecond)

	// Press K to trigger k9s
	_ = tf.Send("K")

	// Wait for error modal
	if !tf.WaitForPlain("k9s Error", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("k9s error modal not shown")
	}

	// Verify error message content
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "k9s not found") {
		t.Log(snapshot)
		t.Fatal("error message about k9s not found is missing")
	}

	// Dismiss modal with Enter
	_ = tf.Enter()

	// Wait for modal to close - verify by checking that k9s Error is no longer shown
	// and tree view is back
	time.Sleep(500 * time.Millisecond)

	// Take snapshot and verify modal is closed
	snapshot = tf.SnapshotPlain()
	// The modal may still be rendered in the buffer history - check the most recent state
	// by looking for the tree view without the error overlay
	if !strings.Contains(snapshot, "Application [demo]") {
		t.Log(snapshot)
		t.Fatal("should be back in tree view after dismissing error modal")
	}
}

// TestK9s_OpenApplicationCR verifies that pressing K on the Application root node
// in tree view opens k9s for the ArgoCD Application CR with context picker.
func TestK9s_OpenApplicationCR(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
	kubeconfigPath := setupMultipleContextsKubeconfigNoCurrent(t, tf.workspace, []string{
		"mgmt-cluster",
		"workload-cluster",
	})

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

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Application [demo]", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded with Application root")
	}

	// Stay on Application root node (don't navigate down)
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "Ready • 1/5") {
		t.Log(snapshot)
		t.Fatal("expected cursor at position 1 (Application node)")
	}

	// Press K on Application node - should show context picker (always for Application CRs)
	_ = tf.Send("K")

	if !tf.WaitForPlain("Select Kubernetes Context", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context picker should appear for Application CR")
	}

	// Select first context
	_ = tf.Enter()

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched")
	}

	time.Sleep(500 * time.Millisecond)

	// Verify k9s was called with Application CR arguments
	args := readMockK9sArgs(t, argsFile)
	if args == "" {
		t.Fatal("k9s was not invoked")
	}

	// Should use the full CRD name for Application
	if !strings.Contains(args, "-c applications.argoproj.io /demo") {
		t.Errorf("expected args to contain '-c applications.argoproj.io /demo', got: %s", args)
	}

	// Should use the ArgoCD namespace (from metadata.namespace)
	if !strings.Contains(args, "-n argocd") {
		t.Errorf("expected args to contain '-n argocd', got: %s", args)
	}

	// Should have context from the picker
	if !strings.Contains(args, "--context mgmt-cluster") {
		t.Errorf("expected args to contain '--context mgmt-cluster', got: %s", args)
	}
}

// TestK9s_OpenApplicationFromAppsList verifies that pressing K in the app list view
// opens k9s for the ArgoCD Application CR with context picker.
func TestK9s_OpenApplicationFromAppsList(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
	kubeconfigPath := setupMultipleContextsKubeconfigNoCurrent(t, tf.workspace, []string{
		"mgmt-cluster",
		"workload-cluster",
	})

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

	// Navigate to apps view via command
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("apps")
	_ = tf.Enter()

	if !tf.WaitForPlain("demo", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("apps not visible")
	}

	// Press K on the app in the list view - should show context picker
	_ = tf.Send("K")

	if !tf.WaitForPlain("Select Kubernetes Context", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context picker should appear for Application CR from app list")
	}

	// Select first context
	_ = tf.Enter()

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched")
	}

	time.Sleep(500 * time.Millisecond)

	// Verify k9s was called with Application CR arguments
	args := readMockK9sArgs(t, argsFile)
	if args == "" {
		t.Fatal("k9s was not invoked")
	}

	// Should use the full CRD name for Application
	if !strings.Contains(args, "-c applications.argoproj.io /demo") {
		t.Errorf("expected args to contain '-c applications.argoproj.io /demo', got: %s", args)
	}

	// Should use the ArgoCD namespace
	if !strings.Contains(args, "-n argocd") {
		t.Errorf("expected args to contain '-n argocd', got: %s", args)
	}

	// Should have context from the picker
	if !strings.Contains(args, "--context mgmt-cluster") {
		t.Errorf("expected args to contain '--context mgmt-cluster', got: %s", args)
	}
}

// ---- Happy Path Tests ----

// TestK9s_OpenDeploymentResource verifies that pressing K on a Deployment
// opens k9s with the correct resource alias (-c deploy).
func TestK9s_OpenDeploymentResource(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
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

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Wait for tree to load and show Deployment
	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment (first child after Application root)
	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)

	// Press K to open k9s
	_ = tf.Send("K")

	// Wait for mock k9s to output something (indicates it was launched)
	// The mock outputs "Mock k9s" after the clear screen sequence
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		// Check if context picker appeared instead
		snapshot := tf.SnapshotPlain()
		if strings.Contains(snapshot, "Select Kubernetes Context") {
			t.Fatal("context picker appeared - kubeconfig may have multiple contexts")
		}
		t.Log(snapshot)
		t.Fatal("mock k9s was not launched")
	}

	// Wait for k9s to exit and return to tree view
	// The mock exits after 0.2s
	time.Sleep(500 * time.Millisecond)

	// Verify k9s was called with correct arguments
	args := readMockK9sArgs(t, argsFile)
	if args == "" {
		t.Fatal("k9s was not invoked")
	}

	// Should contain -c 'deploy /demo-deploy' (resource alias + filter for Deployment)
	if !strings.Contains(args, "-c deploy /demo-deploy") {
		t.Errorf("expected args to contain '-c deploy /demo-deploy', got: %s", args)
	}

	// Should contain -n default (namespace)
	if !strings.Contains(args, "-n default") {
		t.Errorf("expected args to contain '-n default', got: %s", args)
	}

	// Should contain --context cluster-a (matches ArgoCD cluster name)
	if !strings.Contains(args, "--context cluster-a") {
		t.Errorf("expected args to contain '--context cluster-a', got: %s", args)
	}
}

// TestK9s_OpenServiceResource verifies that pressing K on a Service
// opens k9s with the correct resource alias (-c svc).
func TestK9s_OpenServiceResource(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
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

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	// Wait for Service to appear in tree
	if !tf.WaitForPlain("Service", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded with Service")
	}

	// Navigate to Service - tree structure is:
	// Application [demo] (pos 1)
	// ├── Deployment (pos 2)
	// │   └── ReplicaSet (pos 3)
	// │       └── Pod (pos 4)
	// └── Service (pos 5)
	// Navigate down 4 times to reach Service
	for i := 0; i < 4; i++ {
		_ = tf.Send("j")
		time.Sleep(100 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)

	// Press K to open k9s
	_ = tf.Send("K")

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		snapshot := tf.SnapshotPlain()
		if strings.Contains(snapshot, "Select Kubernetes Context") {
			t.Fatal("context picker appeared - kubeconfig may have multiple contexts")
		}
		t.Log(snapshot)
		t.Fatal("mock k9s was not launched")
	}

	// Wait for k9s to exit
	time.Sleep(500 * time.Millisecond)

	args := readMockK9sArgs(t, argsFile)
	if args == "" {
		t.Fatal("k9s was not invoked")
	}

	// Should contain -c 'svc /demo-svc' (resource alias + filter for Service)
	if !strings.Contains(args, "-c svc /demo-svc") {
		t.Errorf("expected args to contain '-c svc /demo-svc', got: %s", args)
	}
}

// TestK9s_TerminalRestoredAfterExit verifies that the UI is functional
// after k9s exits.
func TestK9s_TerminalRestoredAfterExit(t *testing.T) {
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

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment and open k9s
	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)
	_ = tf.Send("K")

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched")
	}

	// Wait for k9s to exit
	time.Sleep(500 * time.Millisecond)

	// Verify UI is functional - try navigating
	_ = tf.Send("j") // Move down
	time.Sleep(200 * time.Millisecond)
	_ = tf.Send("k") // Move up
	time.Sleep(200 * time.Millisecond)

	// Open command mode to verify input works
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("command mode failed after k9s exit: %v", err)
	}

	// Press Escape to exit command mode
	_ = tf.Send("\x1b")
	time.Sleep(200 * time.Millisecond)

	// Verify we're still in tree view
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "Application [demo]") {
		t.Log(snapshot)
		t.Fatal("UI not functional after k9s exit")
	}
}

// ---- Context Selection Tests ----

// TestK9s_ContextPicker_SingleContext_StillShowsPicker verifies that even with a single
// kubeconfig context, the context picker is shown for user confirmation.
// This is intentionally strict to prevent accidentally opening the wrong cluster.
func TestK9s_ContextPicker_SingleContext_StillShowsPicker(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
	kubeconfigPath := setupSingleContextKubeconfig(t, tf.workspace, "my-single-context")

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

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment
	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)

	// Press K - should show context picker even with single context (safety feature)
	_ = tf.Send("K")

	// Context picker should appear
	if !tf.WaitForPlain("Select Kubernetes Context", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context picker should appear even with single context")
	}

	// Verify the single context is shown
	if !tf.WaitForPlain("my-single-context", 2*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context 'my-single-context' not shown in picker")
	}

	// User must explicitly select with Enter
	_ = tf.Enter()

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched")
	}

	// Wait for k9s to exit
	time.Sleep(500 * time.Millisecond)

	// Verify k9s was called with the single context
	args := readMockK9sArgs(t, argsFile)
	if !strings.Contains(args, "--context my-single-context") {
		t.Errorf("expected context 'my-single-context', got args: %s", args)
	}
}

// TestK9s_ContextPicker_MultipleContexts verifies that with multiple kubeconfig
// contexts and NO current-context set, the context picker modal is shown.
func TestK9s_ContextPicker_MultipleContexts(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
	// Use NoCurrent variant to ensure context picker appears
	kubeconfigPath := setupMultipleContextsKubeconfigNoCurrent(t, tf.workspace, []string{
		"dev-cluster",
		"staging-cluster",
		"prod-cluster",
	})

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

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)

	// Press K - should show context picker
	_ = tf.Send("K")

	// Wait for context picker modal
	if !tf.WaitForPlain("Select Kubernetes Context", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context picker modal not shown")
	}

	// Verify all contexts are visible
	snapshot := tf.SnapshotPlain()
	for _, ctx := range []string{"dev-cluster", "staging-cluster", "prod-cluster"} {
		if !strings.Contains(snapshot, ctx) {
			t.Errorf("context %q not visible in picker", ctx)
		}
	}

	// Select staging-cluster (second option) - press j then Enter
	_ = tf.Send("j") // Move to staging-cluster
	time.Sleep(100 * time.Millisecond)
	_ = tf.Enter()

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched after context selection")
	}

	// Wait for k9s to exit
	time.Sleep(500 * time.Millisecond)

	// Verify k9s was called with staging-cluster
	args := readMockK9sArgs(t, argsFile)
	if !strings.Contains(args, "--context staging-cluster") {
		t.Errorf("expected context 'staging-cluster', got args: %s", args)
	}
}

// TestK9s_ContextPicker_NavigateWithJK verifies that j/k keys work for
// navigating in the context picker.
func TestK9s_ContextPicker_NavigateWithJK(t *testing.T) {
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
	// Use NoCurrent variant to ensure context picker appears
	kubeconfigPath := setupMultipleContextsKubeconfigNoCurrent(t, tf.workspace, []string{
		"ctx-1",
		"ctx-2",
		"ctx-3",
	})

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

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)
	_ = tf.Send("K")

	if !tf.WaitForPlain("Select Kubernetes Context", 3*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context picker not shown")
	}

	// Navigate: j, j to ctx-3, then k back to ctx-2
	_ = tf.Send("j") // ctx-2
	time.Sleep(100 * time.Millisecond)
	_ = tf.Send("j") // ctx-3
	time.Sleep(100 * time.Millisecond)
	_ = tf.Send("k") // back to ctx-2
	time.Sleep(100 * time.Millisecond)
	_ = tf.Enter()

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched after context selection")
	}

	// Wait for k9s to exit
	time.Sleep(500 * time.Millisecond)

	args := readMockK9sArgs(t, argsFile)
	if !strings.Contains(args, "--context ctx-2") {
		t.Errorf("expected context 'ctx-2', got args: %s", args)
	}
}

// TestK9s_InCluster_ShowsContextPicker verifies that when an app uses "in-cluster"
// as destination, the context picker is always shown instead of auto-detecting.
// Also verifies the current kubeconfig context is pre-selected.
func TestK9s_InCluster_ShowsContextPicker(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServerWithInCluster()
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

	mockK9s, argsFile := createMockK9s(t, tf.workspace, 0)
	// Set up kubeconfig with current-context=staging-cluster
	// The old code would blindly use staging-cluster; the fix shows a picker instead
	kubeconfigPath := setupMultipleContextsKubeconfigWithCurrent(t, tf.workspace,
		[]string{"dev-cluster", "staging-cluster", "prod-cluster"}, "staging-cluster")

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath},
		"ARGONAUT_K9S_COMMAND="+mockK9s,
		"KUBECONFIG="+kubeconfigPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("in-cluster", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("clusters not visible")
	}

	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment
	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)

	// Press K - should show context picker (not auto-detect)
	_ = tf.Send("K")

	if !tf.WaitForPlain("Select Kubernetes Context", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("context picker should appear for in-cluster apps")
	}

	// All contexts should be visible
	snapshot := tf.SnapshotPlain()
	for _, ctx := range []string{"dev-cluster", "staging-cluster", "prod-cluster"} {
		if !strings.Contains(snapshot, ctx) {
			t.Errorf("context %q not visible in picker", ctx)
		}
	}

	// Current context (staging-cluster) should be pre-selected.
	// Just press Enter to confirm the pre-selected context.
	_ = tf.Enter()

	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched")
	}

	time.Sleep(500 * time.Millisecond)

	// Verify k9s was called with staging-cluster (the pre-selected current context)
	args := readMockK9sArgs(t, argsFile)
	if !strings.Contains(args, "--context staging-cluster") {
		t.Errorf("expected pre-selected context 'staging-cluster', got args: %s", args)
	}
}

// ---- Keyboard Input Tests ----

// TestK9s_KeyboardInputForwarded verifies that keyboard input from the user
// is correctly forwarded to k9s through the PTY.
func TestK9s_KeyboardInputForwarded(t *testing.T) {
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

	// Use interactive mock that captures stdin
	mockK9s, _, inputFile := createInteractiveMockK9s(t, tf.workspace)
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

	// Navigate to tree view
	if err := tf.OpenCommand(); err != nil {
		t.Fatalf("open command: %v", err)
	}
	_ = tf.Send("resources demo")
	_ = tf.Enter()

	if !tf.WaitForPlain("Deployment", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("tree view not loaded")
	}

	// Navigate to Deployment and open k9s
	_ = tf.Send("j")
	time.Sleep(200 * time.Millisecond)
	_ = tf.Send("K")

	// Wait for mock k9s to be launched
	if !tf.WaitForPlain("Mock k9s", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("mock k9s was not launched")
	}

	// Send keystrokes that should be forwarded to k9s
	_ = tf.Send("hello")

	// Wait for mock k9s to exit (2s timeout + buffer)
	time.Sleep(3 * time.Second)

	// Verify the keystrokes were received by k9s
	input := readMockK9sInput(t, inputFile)
	if input != "hello" {
		t.Errorf("expected k9s to receive 'hello', got: %q", input)
	}
}

