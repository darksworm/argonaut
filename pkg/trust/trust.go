package trust

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// Options configures certificate loading and HTTP client behavior
type Options struct {
	CACertFile string        // Path to PEM bundle file
	CACertDir  string        // Directory containing *.pem or *.crt files
	Timeout    time.Duration // HTTP client timeout
	MinTLS     uint16        // Minimum TLS version
}

// LoadPool creates a certificate pool with system roots plus optional extras
func LoadPool(opts Options) (*x509.CertPool, error) {
	// Start with system certificate pool
	pool, err := x509.SystemCertPool()
	if err != nil {
		// On some systems (Windows without CGO), SystemCertPool may fail
		// Fall back to empty pool
		pool = x509.NewCertPool()
	}
	if pool == nil {
		pool = x509.NewCertPool()
	}

	// Helper to add certificates from PEM data
	add := func(src string, pem []byte) error {
		if ok := pool.AppendCertsFromPEM(pem); !ok {
			return fmt.Errorf("no valid certificates found in %s", src)
		}
		return nil
	}

	// Add certificates from file (--cacert or SSL_CERT_FILE)
	if f := first(opts.CACertFile, os.Getenv("SSL_CERT_FILE")); f != "" {
		b, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA cert file %s: %w", f, err)
		}
		if err := add(f, b); err != nil {
			return nil, err
		}
	}

	// Add certificates from directory (--capath or SSL_CERT_DIR)
	if d := first(opts.CACertDir, os.Getenv("SSL_CERT_DIR")); d != "" {
		err := filepath.WalkDir(d, func(p string, e fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			if e.IsDir() {
				return nil
			}
			if !hasSuffix(p, ".pem", ".crt") {
				return nil
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return fmt.Errorf("failed to read CA cert file %s: %w", p, err)
			}
			return add(p, b)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to load certificates from directory %s: %w", d, err)
		}
	}

	// Note: We don't verify pool has subjects here because:
	// 1. On some systems (Windows without CGO), system pool may be empty but still functional
	// 2. User may be providing all certs via flags/env vars
	// 3. Better to let TLS handshake fail with specific error than fail early

	return pool, nil
}

// NewHTTP creates an HTTP client with the given certificate pool and TLS settings
func NewHTTP(pool *x509.CertPool, minTLS uint16, timeout time.Duration) (*http.Client, context.Context) {
	// Create transport with TLS configuration
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs:    pool,
			MinVersion: minTLS,
		},
	}

	// Create HTTP client
	hc := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	// Create context with HTTP client for oauth2 compatibility
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, hc)

	return hc, ctx
}

// first returns the first non-empty, non-whitespace string
func first(vs ...string) string {
	for _, v := range vs {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// hasSuffix checks if string has any of the given suffixes (case insensitive)
func hasSuffix(s string, suff ...string) bool {
	s = strings.ToLower(s)
	for _, x := range suff {
		if strings.HasSuffix(s, x) {
			return true
		}
	}
	return false
}