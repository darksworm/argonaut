package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darksworm/argonaut/pkg/oidc"
)

func TestDiscoverEndpoints_Success(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"issuer":                  srv.URL,
			"authorization_endpoint": srv.URL + "/oauth2/authorize",
			"token_endpoint":         srv.URL + "/oauth2/token",
		})
	}))
	defer srv.Close()

	endpoints, err := oidc.DiscoverEndpoints(context.Background(), srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if endpoints.AuthURL != srv.URL+"/oauth2/authorize" {
		t.Errorf("authURL: got %q", endpoints.AuthURL)
	}
	if endpoints.TokenURL != srv.URL+"/oauth2/token" {
		t.Errorf("tokenURL: got %q", endpoints.TokenURL)
	}
}

func TestDiscoverEndpoints_MissingFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{
			"issuer": "https://dex.example.com",
			// missing authorization_endpoint and token_endpoint
		})
	}))
	defer srv.Close()

	_, err := oidc.DiscoverEndpoints(context.Background(), srv.URL, false)
	if err == nil {
		t.Fatal("expected error for missing endpoints")
	}
}
