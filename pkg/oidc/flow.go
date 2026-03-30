package oidc

import (
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// TokenSet holds the tokens returned from a successful OIDC exchange or refresh.
type TokenSet struct {
	// AuthToken is the token used as ArgoCD's auth-token.
	// It is the id_token when the OIDC provider issues one, otherwise the access_token.
	AuthToken    string
	AccessToken  string
	RefreshToken string
}

// AuthCodeURL generates a PKCE verifier and returns (verifier, authURL, error).
// The caller must pass verifier to ExchangeCode when handling the callback.
func AuthCodeURL(endpoints *Endpoints, cfg *OIDCConfig, redirectURI, state string) (verifier, authURL string, err error) {
	verifier = oauth2.GenerateVerifier()
	o2cfg := oauth2Config(endpoints, cfg, redirectURI)
	authURL = o2cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	return verifier, authURL, nil
}

// ExchangeCode exchanges the authorization code for tokens using the PKCE verifier.
func ExchangeCode(ctx context.Context, endpoints *Endpoints, cfg *OIDCConfig, code, verifier, redirectURI string, insecure bool) (*TokenSet, error) {
	o2cfg := oauth2Config(endpoints, cfg, redirectURI)
	ctx = contextWithHTTPClient(ctx, insecure)
	tok, err := o2cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("OIDC code exchange: %w", err)
	}
	return tokenSetFromOAuth2(tok), nil
}

// RefreshTokens silently refreshes the session using a refresh token.
// Returns new tokens on success.
func RefreshTokens(ctx context.Context, endpoints *Endpoints, cfg *OIDCConfig, refreshToken string, insecure bool) (*TokenSet, error) {
	o2cfg := oauth2Config(endpoints, cfg, "")
	ctx = contextWithHTTPClient(ctx, insecure)
	src := o2cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken})
	tok, err := src.Token()
	if err != nil {
		return nil, fmt.Errorf("OIDC token refresh: %w", err)
	}
	return tokenSetFromOAuth2(tok), nil
}

func oauth2Config(endpoints *Endpoints, cfg *OIDCConfig, redirectURI string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  endpoints.AuthURL,
			TokenURL: endpoints.TokenURL,
		},
		RedirectURL: redirectURI,
		Scopes:      cfg.Scopes,
	}
}

func tokenSetFromOAuth2(tok *oauth2.Token) *TokenSet {
	idToken, _ := tok.Extra("id_token").(string)
	// Use id_token as the ArgoCD auth-token; fall back to access_token if absent.
	authToken := idToken
	if authToken == "" {
		authToken = tok.AccessToken
	}
	return &TokenSet{
		AuthToken:    authToken,
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
	}
}

// contextWithHTTPClient injects a custom http.Client for insecure requests.
func contextWithHTTPClient(ctx context.Context, insecure bool) context.Context {
	if !insecure {
		return ctx
	}
	return context.WithValue(ctx, oauth2.HTTPClient, buildHTTPClient(insecure))
}
