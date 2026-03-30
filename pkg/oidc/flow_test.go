package oidc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/oidc"
)

func TestAuthCodeURL_ContainsPKCEAndState(t *testing.T) {
	cfg := &oidc.OIDCConfig{
		ClientID: "test-client",
		Scopes:   []string{"openid"},
	}
	endpoints := &oidc.Endpoints{
		AuthURL:  "https://dex.example.com/auth",
		TokenURL: "https://dex.example.com/token",
	}
	verifier, url, err := oidc.AuthCodeURL(endpoints, cfg, "http://localhost:8085/auth/callback", "test-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if verifier == "" {
		t.Error("expected non-empty verifier")
	}
	if !strings.Contains(url, "code_challenge") {
		t.Errorf("expected code_challenge in auth URL, got: %s", url)
	}
	if !strings.Contains(url, "test-state") {
		t.Errorf("expected state in auth URL, got: %s", url)
	}
	if !strings.Contains(url, "code_challenge_method=S256") {
		t.Errorf("expected S256 challenge method in auth URL, got: %s", url)
	}
}

func TestExchangeCode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "access-tok",
			"id_token":      "id-tok",
			"refresh_token": "refresh-tok",
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	endpoints := &oidc.Endpoints{AuthURL: srv.URL + "/auth", TokenURL: srv.URL + "/token"}
	cfg := &oidc.OIDCConfig{ClientID: "c", Scopes: []string{"openid"}}

	tokens, err := oidc.ExchangeCode(context.Background(), endpoints, cfg, "auth-code", "verifier", "http://localhost:8085/auth/callback", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.IDToken != "id-tok" {
		t.Errorf("IDToken: got %q", tokens.IDToken)
	}
	if tokens.RefreshToken != "refresh-tok" {
		t.Errorf("RefreshToken: got %q", tokens.RefreshToken)
	}
}

func TestExchangeCode_FallsBackToAccessToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "access-only",
			// no id_token
			"token_type": "Bearer",
		})
	}))
	defer srv.Close()

	endpoints := &oidc.Endpoints{AuthURL: srv.URL + "/auth", TokenURL: srv.URL + "/token"}
	cfg := &oidc.OIDCConfig{ClientID: "c", Scopes: []string{"openid"}}

	tokens, err := oidc.ExchangeCode(context.Background(), endpoints, cfg, "auth-code", "verifier", "http://localhost:8085/auth/callback", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.IDToken != "access-only" {
		t.Errorf("expected fallback to access_token, got: %q", tokens.IDToken)
	}
}

func TestRefreshTokens_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token":  "new-access",
			"id_token":      "new-id-tok",
			"refresh_token": "new-refresh",
			"token_type":    "Bearer",
		})
	}))
	defer srv.Close()

	endpoints := &oidc.Endpoints{AuthURL: srv.URL + "/auth", TokenURL: srv.URL + "/token"}
	cfg := &oidc.OIDCConfig{ClientID: "c", Scopes: []string{"openid"}}

	tokens, err := oidc.RefreshTokens(context.Background(), endpoints, cfg, "old-refresh", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens.IDToken != "new-id-tok" {
		t.Errorf("IDToken: got %q", tokens.IDToken)
	}
}
