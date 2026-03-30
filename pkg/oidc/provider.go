package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Endpoints holds the authorization and token endpoint URLs from OIDC discovery.
type Endpoints struct {
	AuthURL  string
	TokenURL string
}

type wellKnownResponse struct {
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
}

// DiscoverEndpoints fetches {issuerURL}/.well-known/openid-configuration and
// returns the auth and token endpoint URLs.
func DiscoverEndpoints(ctx context.Context, issuerURL string, insecure bool) (*Endpoints, error) {
	url := strings.TrimRight(issuerURL, "/") + "/.well-known/openid-configuration"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := buildHTTPClient(insecure).Do(req)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OIDC discovery returned HTTP %d", resp.StatusCode)
	}
	var wk wellKnownResponse
	if err := json.NewDecoder(resp.Body).Decode(&wk); err != nil {
		return nil, fmt.Errorf("decoding OIDC discovery: %w", err)
	}
	if wk.AuthorizationEndpoint == "" || wk.TokenEndpoint == "" {
		return nil, fmt.Errorf("OIDC discovery response missing required endpoints")
	}
	return &Endpoints{AuthURL: wk.AuthorizationEndpoint, TokenURL: wk.TokenEndpoint}, nil
}
