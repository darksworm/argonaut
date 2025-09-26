//go:build e2e && unix

package main

import (
	"path/filepath"
	"testing"
	"time"
)

// TestTLSInvalidCertFile tests that argonaut shows proper error when cert file is invalid
func TestTLSInvalidCertFile(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Set up workspace and ArgoCD config
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Use HTTPS config with any server URL (we won't connect due to cert error)
	if err := WriteArgoConfigHTTPS(cfgPath, "https://localhost:9999"); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Start argonaut with invalid --cacert path - this should fail during TLS setup
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath, "--cacert=/nonexistent/cert.pem"}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Expect TLS configuration failure message
	if !tf.WaitForPlain("TLS configuration failed", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected 'TLS configuration failed' error message")
	}

	// Expect hint about using --cacert or --capath flags
	if !tf.WaitForPlain("--cacert", 2*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected hint about --cacert flag")
	}

	if !tf.WaitForPlain("--capath", 2*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected hint about --capath flag")
	}
}

// TestTLSUntrustedCert tests that argonaut shows connection error when encountering untrusted certificate
func TestTLSUntrustedCert(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Get absolute paths to test certificates
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	serverCertPath := filepath.Join(cwd, "testdata", "certs", "server.crt")
	serverKeyPath := filepath.Join(cwd, "testdata", "certs", "server.key")

	// Create HTTPS server with self-signed certificate
	srv, err := MockArgoServerHTTPS(serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("failed to create HTTPS server: %v", err)
	}
	defer srv.Close()

	// Set up workspace and ArgoCD config
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Use HTTPS config (insecure: false) to trigger TLS validation
	if err := WriteArgoConfigHTTPS(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Start argonaut without --cacert flag - this should show connection error due to untrusted cert
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Expect connection error (since TLS handshake will fail with untrusted cert)
	if !tf.WaitForPlain("Connection Error", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected 'Connection Error' message when using untrusted certificate")
	}

	// Should show general connection troubleshooting tips
	if !tf.WaitForPlain("Unable to connect to Argo CD server", 2*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected connection error details")
	}
}

// TestTLSTrustedCert tests that argonaut works correctly when provided with trusted certificate
func TestTLSTrustedCert(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Get absolute paths to test certificates
	cwd, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	serverCertPath := filepath.Join(cwd, "testdata", "certs", "server.crt")
	serverKeyPath := filepath.Join(cwd, "testdata", "certs", "server.key")
	caCertPath := filepath.Join(cwd, "testdata", "certs", "ca.pem")

	// Create HTTPS server with self-signed certificate
	srv, err := MockArgoServerHTTPS(serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("failed to create HTTPS server: %v", err)
	}
	defer srv.Close()

	// Set up workspace and ArgoCD config
	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Use HTTPS config (insecure: false) to trigger TLS validation
	if err := WriteArgoConfigHTTPS(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Start argonaut WITH --cacert flag pointing to our CA certificate
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath, "--cacert=" + caCertPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Should NOT see TLS configuration failure
	if tf.WaitForPlain("TLS configuration failed", 2*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("should not see 'TLS configuration failed' when using valid CA certificate")
	}

	// Should see successful TLS trust configuration (this appears in logs, but might not be visible in TUI)
	// Instead, let's check that the app starts normally and shows expected content

	// Wait for the app to load and show normal UI state
	// The app shows cluster-a from our mock server, indicating successful TLS connection
	if !tf.WaitForPlain("cluster-a", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see cluster-a, indicating successful API communication via trusted TLS connection")
	}

	// Verify we're not stuck at "Connecting to Argo CD..." which would indicate connection issues
	snapshot := tf.SnapshotPlain()
	if !tf.WaitForPlain("Ready", 2*time.Second) {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected to see 'Ready' status, indicating successful connection")
	}
}