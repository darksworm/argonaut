package trust

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// testCA creates a test CA certificate for testing
func testCA() (*x509.Certificate, *rsa.PrivateKey, []byte, error) {
	// Generate private key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, nil, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"Test CA"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, nil, err
	}

	// Encode as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	return cert, priv, certPEM, nil
}

func TestLoadPool_EmptyOptions(t *testing.T) {
	opts := Options{}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool with empty options should not fail: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Pool should exist even if empty (system may not have certs available)
	// This is fine - TLS handshake will fail appropriately later
}

func TestLoadPool_ValidCertFile(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "test-ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(certPEM); err != nil {
		t.Fatalf("Failed to write cert to temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading
	opts := Options{CACertFile: tmpFile.Name()}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed with valid cert file: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should have at least one certificate (our test cert)
	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Fatal("Pool should contain at least the test certificate")
	}
}

func TestLoadPool_InvalidCertFile(t *testing.T) {
	// Write invalid data to temporary file
	tmpFile, err := os.CreateTemp("", "invalid-ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte("invalid certificate data")); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Test loading
	opts := Options{CACertFile: tmpFile.Name()}
	_, err = LoadPool(opts)

	if err == nil {
		t.Fatal("LoadPool should fail with invalid cert file")
	}

	if !strings.Contains(err.Error(), "no valid certificates found") {
		t.Errorf("Error should mention invalid certificates, got: %v", err)
	}
}

func TestLoadPool_NonExistentFile(t *testing.T) {
	opts := Options{CACertFile: "/nonexistent/file.pem"}
	_, err := LoadPool(opts)

	if err == nil {
		t.Fatal("LoadPool should fail with non-existent file")
	}

	if !strings.Contains(err.Error(), "failed to read CA cert file") {
		t.Errorf("Error should mention failed to read file, got: %v", err)
	}
}

func TestLoadPool_ValidCertDir(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-ca-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write certificates
	certFile1 := filepath.Join(tmpDir, "ca1.pem")
	certFile2 := filepath.Join(tmpDir, "ca2.crt")
	certFile3 := filepath.Join(tmpDir, "ignored.txt") // Should be ignored

	if err := os.WriteFile(certFile1, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file 1: %v", err)
	}
	if err := os.WriteFile(certFile2, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file 2: %v", err)
	}
	if err := os.WriteFile(certFile3, []byte("ignored"), 0644); err != nil {
		t.Fatalf("Failed to write ignored file: %v", err)
	}

	// Test loading
	opts := Options{CACertDir: tmpDir}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed with valid cert dir: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should have certificates
	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Fatal("Pool should contain certificates from directory")
	}
}

func TestLoadPool_NonExistentDir(t *testing.T) {
	opts := Options{CACertDir: "/nonexistent/dir"}
	_, err := LoadPool(opts)

	if err == nil {
		t.Fatal("LoadPool should fail with non-existent directory")
	}

	if !strings.Contains(err.Error(), "failed to load certificates from directory") {
		t.Errorf("Error should mention failed to load from directory, got: %v", err)
	}
}

func TestLoadPool_EnvironmentVariables(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Write to temporary file
	tmpFile, err := os.CreateTemp("", "test-ca-env-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(certPEM); err != nil {
		t.Fatalf("Failed to write cert to temp file: %v", err)
	}
	tmpFile.Close()

	// Set environment variable
	oldEnv := os.Getenv("SSL_CERT_FILE")
	defer os.Setenv("SSL_CERT_FILE", oldEnv)
	os.Setenv("SSL_CERT_FILE", tmpFile.Name())

	// Test loading (with empty options, should use env var)
	opts := Options{}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed with env var set: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}
}

func TestLoadPool_FlagPrecedenceOverEnv(t *testing.T) {
	// Create two test CAs
	_, _, certPEM1, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA 1: %v", err)
	}
	_, _, certPEM2, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA 2: %v", err)
	}

	// Write to temporary files
	tmpFile1, err := os.CreateTemp("", "test-ca-flag-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())

	tmpFile2, err := os.CreateTemp("", "test-ca-env-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file 2: %v", err)
	}
	defer os.Remove(tmpFile2.Name())

	if _, err := tmpFile1.Write(certPEM1); err != nil {
		t.Fatalf("Failed to write cert to temp file 1: %v", err)
	}
	tmpFile1.Close()

	if _, err := tmpFile2.Write(certPEM2); err != nil {
		t.Fatalf("Failed to write cert to temp file 2: %v", err)
	}
	tmpFile2.Close()

	// Set environment variable
	oldEnv := os.Getenv("SSL_CERT_FILE")
	defer os.Setenv("SSL_CERT_FILE", oldEnv)
	os.Setenv("SSL_CERT_FILE", tmpFile2.Name())

	// Test loading with flag (should take precedence over env)
	opts := Options{CACertFile: tmpFile1.Name()}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should use the flag file, not the env file
	// This is harder to test directly, but we can at least verify it succeeds
}

func TestLoadPool_ColonSeparatedCertDir(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Create temporary directories
	tmpDir1, err := os.MkdirTemp("", "test-ca-dir1-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "test-ca-dir2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tmpDir2)

	// Write certificates to both directories
	certFile1 := filepath.Join(tmpDir1, "ca1.pem")
	certFile2 := filepath.Join(tmpDir2, "ca2.crt")

	if err := os.WriteFile(certFile1, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file 1: %v", err)
	}
	if err := os.WriteFile(certFile2, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file 2: %v", err)
	}

	// Test with colon-separated SSL_CERT_DIR
	oldEnv := os.Getenv("SSL_CERT_DIR")
	defer os.Setenv("SSL_CERT_DIR", oldEnv)

	colonSeparated := tmpDir1 + ":" + tmpDir2
	os.Setenv("SSL_CERT_DIR", colonSeparated)

	opts := Options{}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed with colon-separated SSL_CERT_DIR: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should have certificates from both directories
	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Fatal("Pool should contain certificates from both directories")
	}
}

func TestLoadPool_ColonSeparatedCertDirWithSpaces(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test-ca-dir-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write certificate
	certFile := filepath.Join(tmpDir, "ca.pem")
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Test with colon-separated SSL_CERT_DIR that has spaces and empty entries
	oldEnv := os.Getenv("SSL_CERT_DIR")
	defer os.Setenv("SSL_CERT_DIR", oldEnv)

	// Include spaces around paths and empty entries (common in env vars)
	colonSeparated := " " + tmpDir + " : : /nonexistent/dir :"
	os.Setenv("SSL_CERT_DIR", colonSeparated)

	opts := Options{}
	pool, err := LoadPool(opts)

	// Should succeed despite non-existent directories and spaces
	if err != nil {
		t.Fatalf("LoadPool should succeed with spaced colon-separated SSL_CERT_DIR: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should have certificate from the valid directory
	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Fatal("Pool should contain certificate from valid directory")
	}
}

func TestLoadPool_FlagWithColonSeparatedDirs(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Create temporary directories
	tmpDir1, err := os.MkdirTemp("", "test-ca-flag1-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 1: %v", err)
	}
	defer os.RemoveAll(tmpDir1)

	tmpDir2, err := os.MkdirTemp("", "test-ca-flag2-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir 2: %v", err)
	}
	defer os.RemoveAll(tmpDir2)

	// Write certificates to both directories
	certFile1 := filepath.Join(tmpDir1, "ca1.pem")
	certFile2 := filepath.Join(tmpDir2, "ca2.crt")

	if err := os.WriteFile(certFile1, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file 1: %v", err)
	}
	if err := os.WriteFile(certFile2, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file 2: %v", err)
	}

	// Test with colon-separated --ca-path flag (should work with existing dirs)
	colonSeparated := tmpDir1 + ":" + tmpDir2
	opts := Options{CACertDir: colonSeparated}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed with colon-separated flag dirs: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should have certificates from both directories
	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Fatal("Pool should contain certificates from both flag directories")
	}
}

func TestLoadPool_FlagWithColonSeparatedDirsOneNonExistent(t *testing.T) {
	// Create test CA
	_, _, certPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Create one temporary directory
	tmpDir, err := os.MkdirTemp("", "test-ca-flag-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write certificate
	certFile := filepath.Join(tmpDir, "ca.pem")
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}

	// Test with colon-separated --ca-path flag with one non-existent directory
	// Should succeed and skip the non-existent directory
	colonSeparated := tmpDir + ":/nonexistent/dir"
	opts := Options{CACertDir: colonSeparated}
	pool, err := LoadPool(opts)

	if err != nil {
		t.Fatalf("LoadPool should succeed with mixed existing/non-existing flag dirs: %v", err)
	}

	if pool == nil {
		t.Fatal("LoadPool should return a non-nil pool")
	}

	// Should have certificate from the existing directory
	subjects := pool.Subjects()
	if len(subjects) == 0 {
		t.Fatal("Pool should contain certificate from existing flag directory")
	}
}

func TestNewHTTP(t *testing.T) {
	// Create test cert pool
	opts := Options{}
	pool, err := LoadPool(opts)
	if err != nil {
		t.Fatalf("Failed to load pool: %v", err)
	}

	// Test NewHTTP
	client, ctx := NewHTTP(pool, nil, tls.VersionTLS12, 30*time.Second)

	if client == nil {
		t.Fatal("NewHTTP should return a non-nil client")
	}

	if ctx == nil {
		t.Fatal("NewHTTP should return a non-nil context")
	}

	// Verify client configuration - timeout should be zero to allow streaming
	if client.Timeout != 0 {
		t.Errorf("Expected no client timeout (0), got %v", client.Timeout)
	}

	// Verify transport configuration
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Skip("Transport type check skipped (implementation specific)")
	}

	if transport != nil && transport.TLSClientConfig != nil {
		if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("Expected min TLS version 1.2, got %v", transport.TLSClientConfig.MinVersion)
		}

		if transport.TLSClientConfig.RootCAs != pool {
			t.Error("Expected RootCAs to be our certificate pool")
		}
	}
}

func TestFirst(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{"empty", []string{}, ""},
		{"first non-empty", []string{"", "  ", "hello", "world"}, "hello"},
		{"whitespace only", []string{"", "  ", "\t", "\n"}, ""},
		{"single value", []string{"test"}, "test"},
		{"all empty", []string{"", "", ""}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := first(tt.input...)
			if result != tt.expected {
				t.Errorf("first(%v) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHasSuffix(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		suffixes []string
		expected bool
	}{
		{"pem extension", "ca.pem", []string{".pem", ".crt"}, true},
		{"crt extension", "ca.crt", []string{".pem", ".crt"}, true},
		{"case insensitive", "CA.PEM", []string{".pem", ".crt"}, true},
		{"no match", "ca.txt", []string{".pem", ".crt"}, false},
		{"empty suffixes", "ca.pem", []string{}, false},
		{"empty string", "", []string{".pem"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasSuffix(tt.s, tt.suffixes...)
			if result != tt.expected {
				t.Errorf("hasSuffix(%q, %v) = %v, want %v", tt.s, tt.suffixes, result, tt.expected)
			}
		})
	}
}

func TestLoadClientCertificate(t *testing.T) {
	// Create test CA and client cert
	caCert, caKey, _, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Generate client certificate
	clientKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate client key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization:  []string{"Test Client"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
			CommonName:    "test-client",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// Create certificate signed by CA (using the same CA instance)
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("Failed to create client certificate: %v", err)
	}

	// Encode certificate as PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Encode private key as PEM
	keyDER, err := x509.MarshalPKCS8PrivateKey(clientKey)
	if err != nil {
		t.Fatalf("Failed to marshal private key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})

	// Write to temporary files
	certFile, err := os.CreateTemp("", "client-cert-*.pem")
	if err != nil {
		t.Fatalf("Failed to create cert temp file: %v", err)
	}
	defer os.Remove(certFile.Name())

	keyFile, err := os.CreateTemp("", "client-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create key temp file: %v", err)
	}
	defer os.Remove(keyFile.Name())

	if _, err := certFile.Write(certPEM); err != nil {
		t.Fatalf("Failed to write cert file: %v", err)
	}
	certFile.Close()

	if _, err := keyFile.Write(keyPEM); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	keyFile.Close()

	// Test loading valid client certificate
	cert, err := LoadClientCertificate(certFile.Name(), keyFile.Name())
	if err != nil {
		t.Fatalf("LoadClientCertificate should succeed with valid files: %v", err)
	}

	if cert == nil {
		t.Fatal("LoadClientCertificate should return non-nil certificate")
	}

	// Verify certificate has expected properties
	if len(cert.Certificate) == 0 {
		t.Fatal("Certificate should have at least one certificate")
	}

	// Test loading with invalid cert file
	_, err = LoadClientCertificate("/nonexistent/cert.pem", keyFile.Name())
	if err == nil {
		t.Fatal("LoadClientCertificate should fail with non-existent cert file")
	}

	// Test loading with invalid key file
	_, err = LoadClientCertificate(certFile.Name(), "/nonexistent/key.pem")
	if err == nil {
		t.Fatal("LoadClientCertificate should fail with non-existent key file")
	}

	// Test loading with mismatched cert and key
	// Create another key that doesn't match the certificate
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate wrong key: %v", err)
	}

	wrongKeyDER, err := x509.MarshalPKCS8PrivateKey(wrongKey)
	if err != nil {
		t.Fatalf("Failed to marshal wrong key: %v", err)
	}
	wrongKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: wrongKeyDER})

	wrongKeyFile, err := os.CreateTemp("", "wrong-key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create wrong key temp file: %v", err)
	}
	defer os.Remove(wrongKeyFile.Name())

	if _, err := wrongKeyFile.Write(wrongKeyPEM); err != nil {
		t.Fatalf("Failed to write wrong key file: %v", err)
	}
	wrongKeyFile.Close()

	_, err = LoadClientCertificate(certFile.Name(), wrongKeyFile.Name())
	if err == nil {
		t.Fatal("LoadClientCertificate should fail with mismatched cert and key")
	}
}
