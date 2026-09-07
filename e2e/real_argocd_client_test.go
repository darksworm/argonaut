//go:build e2e && unix

package main

import (
	"github.com/darksworm/argonaut/pkg/config"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func TestRealArgoClientRejectsRemoteTargets(t *testing.T) {
	for _, target := range []string{"https://argo.example.com", "http://argo.example.com", "https://localhost.example.com", "https://127.0.0.1@argo.example.com", "ftp://localhost", "://bad"} {
		t.Run(target, func(t *testing.T) {
			if _, err := newRealArgo(&model.Server{BaseURL: target, Insecure: true}); err == nil {
				t.Fatalf("accepted non-local or invalid target %q", target)
			}
		})
	}
}

func TestRealArgoClientHonorsTLSSetting(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	for _, insecure := range []bool{false, true} {
		r, err := newRealArgo(&model.Server{BaseURL: srv.URL, Insecure: insecure})
		if err != nil {
			t.Fatal(err)
		}
		resp, err := r.client.Get(srv.URL)
		if resp != nil {
			resp.Body.Close()
		}
		if insecure && err != nil {
			t.Fatalf("explicit local insecure connection failed: %v", err)
		}
		if !insecure && err == nil {
			t.Fatal("accepted an untrusted certificate despite insecure=false")
		}
	}
}

func TestRealArgoClientDoesNotFollowRedirects(t *testing.T) {
	var reached atomic.Bool
	dst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { reached.Store(true) }))
	defer dst.Close()
	src := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, dst.URL, http.StatusFound) }))
	defer src.Close()
	r, err := newRealArgo(&model.Server{BaseURL: src.URL})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := r.client.Get(src.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if reached.Load() {
		t.Fatal("followed a redirect outside the configured endpoint")
	}
}

func TestRealArgoClientLocalConfigPreservesTLS(t *testing.T) {
	for _, target := range []string{"https://localhost:8080", "https://127.0.0.1:8080", "https://[::1]:8080"} {
		for _, insecure := range []bool{false, true} {
			r, err := newRealArgo(&model.Server{BaseURL: target, Token: "fixture-token", Insecure: insecure})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "config")
			if err := writeArgoConfigWithTLS(path, r.baseURL, r.token, r.insecure); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.ReadCLIConfigFromPath(path)
			if err != nil {
				t.Fatal(err)
			}
			server, err := cfg.ToServerConfig()
			if err != nil {
				t.Fatal(err)
			}
			if server.Insecure != insecure || server.Token != r.token || server.BaseURL != target {
				t.Fatal("generated TUI config changed the endpoint, token, or TLS setting")
			}
		}
	}
}
