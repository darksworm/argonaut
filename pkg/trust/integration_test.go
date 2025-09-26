package trust

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// createTestCertificateAndKey creates a certificate signed by the given CA
func createTestCertificateAndKey(caCert *x509.Certificate, caKey *rsa.PrivateKey, hosts []string) (tls.Certificate, error) {
	// Generate private key for server
	serverKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization:  []string{"Test Server"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{"Test"},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     hosts,
	}

	// Create certificate signed by CA
	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create tls.Certificate
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  serverKey,
	}

	return cert, nil
}

func TestIntegration_HTTPSServerWithCustomCA(t *testing.T) {
	// Create test CA
	caCert, caKey, caPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	// Create server certificate signed by our CA
	serverCert, err := createTestCertificateAndKey(caCert, caKey, []string{"localhost", "127.0.0.1"})
	if err != nil {
		t.Fatalf("Failed to create server certificate: %v", err)
	}

	// Create HTTPS test server
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"message": "Hello from HTTPS server"}`)
	}))

	// Configure server with our certificate
	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	server.StartTLS()
	defer server.Close()

	// Test 1: Request should fail without CA
	t.Run("FailsWithoutCA", func(t *testing.T) {
		// Create trust options without our CA
		opts := Options{
			Timeout: 5 * time.Second,
			MinTLS:  tls.VersionTLS12,
		}

		pool, err := LoadPool(opts)
		if err != nil {
			t.Skipf("Cannot create pool without system certs: %v", err)
		}

		client, _ := NewHTTP(pool, opts.MinTLS, opts.Timeout)

		// This should fail because server cert is not trusted
		resp, err := client.Get(server.URL)
		if err == nil {
			resp.Body.Close()
			t.Fatal("Expected TLS verification to fail, but request succeeded")
		}

		// Check that it's a TLS error
		if !isTLSError(err) {
			t.Errorf("Expected TLS verification error, got: %v", err)
		}
	})

	// Test 2: Request should succeed with CA certificate
	t.Run("SucceedsWithCA", func(t *testing.T) {
		// Write CA certificate to temporary file
		tmpFile, err := os.CreateTemp("", "test-ca-*.pem")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(caPEM); err != nil {
			t.Fatalf("Failed to write CA cert: %v", err)
		}
		tmpFile.Close()

		// Create trust options with our CA
		opts := Options{
			CACertFile: tmpFile.Name(),
			Timeout:    5 * time.Second,
			MinTLS:     tls.VersionTLS12,
		}

		pool, err := LoadPool(opts)
		if err != nil {
			t.Fatalf("Failed to load pool: %v", err)
		}

		client, _ := NewHTTP(pool, opts.MinTLS, opts.Timeout)

		// This should succeed because server cert is now trusted
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Expected request to succeed with CA, got error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Test 3: Test with environment variable
	t.Run("SucceedsWithEnvVar", func(t *testing.T) {
		// Write CA certificate to temporary file
		tmpFile, err := os.CreateTemp("", "test-ca-env-*.pem")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer os.Remove(tmpFile.Name())

		if _, err := tmpFile.Write(caPEM); err != nil {
			t.Fatalf("Failed to write CA cert: %v", err)
		}
		tmpFile.Close()

		// Set environment variable
		oldEnv := os.Getenv("SSL_CERT_FILE")
		defer os.Setenv("SSL_CERT_FILE", oldEnv)
		os.Setenv("SSL_CERT_FILE", tmpFile.Name())

		// Create trust options (empty, should use env var)
		opts := Options{
			Timeout: 5 * time.Second,
			MinTLS:  tls.VersionTLS12,
		}

		pool, err := LoadPool(opts)
		if err != nil {
			t.Fatalf("Failed to load pool: %v", err)
		}

		client, _ := NewHTTP(pool, opts.MinTLS, opts.Timeout)

		// This should succeed because CA is loaded from env var
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Expected request to succeed with env var, got error: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestIntegration_OAuth2Context(t *testing.T) {
	// Create test CA and server as above
	caCert, caKey, caPEM, err := testCA()
	if err != nil {
		t.Fatalf("Failed to create test CA: %v", err)
	}

	serverCert, err := createTestCertificateAndKey(caCert, caKey, []string{"localhost", "127.0.0.1"})
	if err != nil {
		t.Fatalf("Failed to create server certificate: %v", err)
	}

	// Create mock OAuth2 endpoint
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate OAuth2 userinfo endpoint
		if r.URL.Path == "/userinfo" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"sub": "test-user", "name": "Test User"}`)
			return
		}
		// Simulate OAuth2 token endpoint
		if r.URL.Path == "/token" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"access_token": "test-token", "token_type": "Bearer"}`)
			return
		}
		w.WriteHeader(404)
	}))

	server.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
	}
	server.StartTLS()
	defer server.Close()

	// Write CA certificate to temporary file
	tmpFile, err := os.CreateTemp("", "test-oauth-ca-*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(caPEM); err != nil {
		t.Fatalf("Failed to write CA cert: %v", err)
	}
	tmpFile.Close()

	// Create trust options with our CA
	opts := Options{
		CACertFile: tmpFile.Name(),
		Timeout:    5 * time.Second,
		MinTLS:     tls.VersionTLS12,
	}

	pool, err := LoadPool(opts)
	if err != nil {
		t.Fatalf("Failed to load pool: %v", err)
	}

	client, ctx := NewHTTP(pool, opts.MinTLS, opts.Timeout)

	// Test that the context contains the HTTP client for oauth2 usage
	t.Run("ContextContainsHTTPClient", func(t *testing.T) {
		// Create a request using the context
		req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/userinfo", nil)
		if err != nil {
			t.Fatalf("Failed to create request: %v", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Test simulating oauth2 library usage
	t.Run("OAuth2LibrarySimulation", func(t *testing.T) {
		// Simulate how oauth2 library would extract HTTP client from context
		contextClient, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
		if !ok {
			t.Fatal("Context should contain oauth2.HTTPClient")
		}

		if contextClient != client {
			t.Error("Context HTTP client should be the same as returned client")
		}

		// Use the context client for OAuth2-style request
		resp, err := contextClient.Get(server.URL + "/token")
		if err != nil {
			t.Fatalf("OAuth2-style request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

// isTLSError checks if an error is related to TLS verification
func isTLSError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common TLS error patterns
	errStr := err.Error()
	return containsAny(errStr,
		"certificate verify failed",
		"certificate signed by unknown authority",
		"certificate is not trusted",
		"x509: certificate",
		"tls: bad certificate",
		"tls: handshake failure",
		"tls: failed to verify certificate",
	)
}

// containsAny checks if string contains any of the given substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}