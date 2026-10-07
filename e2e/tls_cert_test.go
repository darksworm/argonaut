//go:build e2e && unix

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func generateTLSCertificates(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	now := time.Now()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ca := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Argonaut E2E CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	writePEM := func(name, kind string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: data}), 0600); err != nil {
			t.Fatal(err)
		}
	}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}
	writePEM("ca.pem", "CERTIFICATE", caDER)
	for index, name := range []string{"server", "client"} {
		key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		certificate := &x509.Certificate{
			SerialNumber: big.NewInt(int64(index + 2)),
			Subject:      pkix.Name{CommonName: name},
			NotBefore:    ca.NotBefore,
			NotAfter:     ca.NotAfter,
			KeyUsage:     x509.KeyUsageDigitalSignature,
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		if name == "server" {
			certificate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
			certificate.DNSNames = []string{"localhost"}
			certificate.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
		}
		certificateDER, err := x509.CreateCertificate(rand.Reader, certificate, ca, &key.PublicKey, caKey)
		if err != nil {
			t.Fatal(err)
		}
		keyDER, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		writePEM(name+".crt", "CERTIFICATE", certificateDER)
		writePEM(name+".key", "PRIVATE KEY", keyDER)
	}
	return directory
}

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

	// Start argonaut with invalid --ca-cert path - this should fail during TLS setup
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath, "--ca-cert=/nonexistent/cert.pem"}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Expect TLS configuration failure message and hints
	if !tf.WaitForPlain("TLS configuration failed", 3*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected 'TLS configuration failed' error message")
	}

	// Should already contain hints in the snapshot
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "--ca-cert") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected hint about --ca-cert flag")
	}

	if !strings.Contains(snapshot, "--ca-path") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected hint about --ca-path flag")
	}
}

// TestTLSUntrustedCert tests that argonaut shows connection error when encountering untrusted certificate
func TestTLSUntrustedCert(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Get absolute paths to test certificates
	cwd := generateTLSCertificates(t)
	serverCertPath := filepath.Join(cwd, "server.crt")
	serverKeyPath := filepath.Join(cwd, "server.key")

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

	// Start argonaut without --ca-cert flag - this should show connection error due to untrusted cert
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Expect connection error (since TLS handshake will fail with untrusted cert)
	// With 2s timeout, we can wait for 4s instead of 12s
	if !tf.WaitForPlain("Connection Error", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected 'Connection Error' message when using untrusted certificate")
	}

	// Should show general connection troubleshooting tips in the same snapshot
	snapshot := tf.SnapshotPlain()
	if !strings.Contains(snapshot, "Unable to connect to Argo CD server") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected connection error details")
	}
}

// TestTLSTrustedCert tests that argonaut works correctly when provided with trusted certificate
func TestTLSTrustedCert(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Get absolute paths to test certificates
	cwd := generateTLSCertificates(t)
	serverCertPath := filepath.Join(cwd, "server.crt")
	serverKeyPath := filepath.Join(cwd, "server.key")
	caCertPath := filepath.Join(cwd, "ca.pem")

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

	// Start argonaut WITH --ca-cert flag pointing to our CA certificate
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath, "--ca-cert=" + caCertPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Wait for the app to load and show normal UI state
	// The app shows cluster-a from our mock server, indicating successful TLS connection
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see cluster-a, indicating successful API communication via trusted TLS connection")
	}

	// Verify we're connected successfully and not stuck at error state
	snapshot := tf.SnapshotPlain()
	if !tf.WaitForPlain("Ready", 2*time.Second) {
		t.Log("Snapshot:", snapshot)
		t.Fatal("expected to see 'Ready' status, indicating successful connection")
	}

	// Should not contain error messages
	if strings.Contains(snapshot, "TLS configuration failed") {
		t.Log("Snapshot:", snapshot)
		t.Fatal("should not see 'TLS configuration failed' when using valid CA certificate")
	}
}

// TestTLSClientCertAuthFails tests that argonaut shows connection error when client cert is required but not provided
func TestTLSClientCertAuthFails(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Get absolute paths to test certificates
	cwd := generateTLSCertificates(t)
	serverCertPath := filepath.Join(cwd, "server.crt")
	serverKeyPath := filepath.Join(cwd, "server.key")
	caCertPath := filepath.Join(cwd, "ca.pem")

	// Create HTTPS server that requires client certificates
	srv, err := MockArgoServerHTTPSWithClientAuth(serverCertPath, serverKeyPath, caCertPath)
	if err != nil {
		t.Fatalf("failed to create HTTPS server with client auth: %v", err)
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

	// Start argonaut without client cert - should fail with TLS handshake error
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath, "--ca-cert=" + caCertPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Should see connection error due to missing client certificate
	// With 2s timeout, we can wait for 4s instead of 12s
	if !tf.WaitForPlain("Connection Error", 4*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected 'Connection Error' when client cert is required but not provided")
	}
}

// TestTLSClientCertAuthSucceeds tests that argonaut works correctly with client certificate authentication
func TestTLSClientCertAuthSucceeds(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	// Get absolute paths to test certificates
	cwd := generateTLSCertificates(t)
	serverCertPath := filepath.Join(cwd, "server.crt")
	serverKeyPath := filepath.Join(cwd, "server.key")
	caCertPath := filepath.Join(cwd, "ca.pem")
	clientCertPath := filepath.Join(cwd, "client.crt")
	clientKeyPath := filepath.Join(cwd, "client.key")

	// Create HTTPS server that requires client certificates
	srv, err := MockArgoServerHTTPSWithClientAuth(serverCertPath, serverKeyPath, caCertPath)
	if err != nil {
		t.Fatalf("failed to create HTTPS server with client auth: %v", err)
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

	// Start argonaut WITH client cert and CA cert
	if err := tf.StartAppArgs([]string{
		"-argocd-config=" + cfgPath,
		"--ca-cert=" + caCertPath,
		"--client-cert=" + clientCertPath,
		"--client-cert-key=" + clientKeyPath,
	}); err != nil {
		t.Fatalf("start app with client cert: %v", err)
	}

	// Wait for the app to load and show normal UI state with successful client cert auth
	if !tf.WaitForPlain("cluster-a", 3*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see cluster-a, indicating successful API communication via client cert auth")
	}

	// Verify we're connected successfully
	if !tf.WaitForPlain("Ready", 2*time.Second) {
		t.Log("Snapshot:", tf.SnapshotPlain())
		t.Fatal("expected to see 'Ready' status, indicating successful client cert authentication")
	}
}
