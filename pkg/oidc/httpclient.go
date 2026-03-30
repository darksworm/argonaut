package oidc

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// buildHTTPClient returns an http.Client for OIDC requests.
// The insecure branch skips TLS verification (for self-signed certs).
// Both branches set connection-level timeouts as a safety net
// in addition to the context deadlines applied at the call site.
func buildHTTPClient(insecure bool) *http.Client {
	tr := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	if insecure {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &http.Client{Transport: tr}
}
