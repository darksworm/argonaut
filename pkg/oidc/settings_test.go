package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/darksworm/argonaut/pkg/oidc"
)

func TestFetchOIDCConfig_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/settings" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"oidcConfig": map[string]interface{}{
				"issuer":      "https://dex.example.com",
				"clientID":    "argo-cd",
				"cliClientID": "argo-cd-cli",
				"requestedScopes": []string{"openid", "profile"},
			},
		})
	}))
	defer srv.Close()

	cfg, err := oidc.FetchOIDCConfig(context.Background(), srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Issuer != "https://dex.example.com" {
		t.Errorf("issuer: got %q, want %q", cfg.Issuer, "https://dex.example.com")
	}
	if cfg.ClientID != "argo-cd-cli" { // prefers cliClientID
		t.Errorf("clientID: got %q, want %q", cfg.ClientID, "argo-cd-cli")
	}
}

func TestFetchOIDCConfig_FallsBackToClientID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"oidcConfig": map[string]interface{}{
				"issuer":   "https://dex.example.com",
				"clientID": "argo-cd",
				// no cliClientID
			},
		})
	}))
	defer srv.Close()

	cfg, err := oidc.FetchOIDCConfig(context.Background(), srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ClientID != "argo-cd" {
		t.Errorf("clientID: got %q, want %q", cfg.ClientID, "argo-cd")
	}
}

func TestFetchOIDCConfig_DefaultScopes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"oidcConfig": map[string]interface{}{
				"issuer":   "https://dex.example.com",
				"clientID": "argo-cd",
				// no requestedScopes
			},
		})
	}))
	defer srv.Close()

	cfg, err := oidc.FetchOIDCConfig(context.Background(), srv.URL, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Scopes) == 0 {
		t.Error("expected default scopes when none configured")
	}
}

func TestFetchOIDCConfig_NoOIDC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{}) // no oidcConfig
	}))
	defer srv.Close()

	_, err := oidc.FetchOIDCConfig(context.Background(), srv.URL, false)
	if err == nil {
		t.Fatal("expected error when no oidcConfig present")
	}
}
