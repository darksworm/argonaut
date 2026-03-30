package oidc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// OIDCConfig holds the OIDC client config fetched from the ArgoCD server.
type OIDCConfig struct {
	Issuer       string
	ClientID     string // CLIClientID if present, else ClientID
	ClientSecret string // empty for public clients
	Scopes       []string
}

type settingsResponse struct {
	OIDCConfig *oidcConfigRaw `json:"oidcConfig"`
}

type oidcConfigRaw struct {
	Issuer       string   `json:"issuer"`
	ClientID     string   `json:"clientID"`
	CLIClientID  string   `json:"cliClientID"`
	ClientSecret string   `json:"clientSecret"`
	Scopes       []string `json:"requestedScopes"`
}

// FetchOIDCConfig fetches OIDC client config from /api/v1/settings.
// Returns error if no OIDC is configured on the server.
func FetchOIDCConfig(ctx context.Context, serverURL string, insecure bool) (*OIDCConfig, error) {
	url := strings.TrimRight(serverURL, "/") + "/api/v1/settings"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := buildHTTPClient(insecure)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching ArgoCD settings: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ArgoCD settings returned HTTP %d", resp.StatusCode)
	}
	var settings settingsResponse
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return nil, fmt.Errorf("decoding settings response: %w", err)
	}
	if settings.OIDCConfig == nil || settings.OIDCConfig.Issuer == "" {
		return nil, fmt.Errorf("ArgoCD server has no OIDC configured")
	}
	raw := settings.OIDCConfig
	clientID := raw.ClientID
	if raw.CLIClientID != "" {
		clientID = raw.CLIClientID
	}
	scopes := raw.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "profile", "email", "groups"}
	}
	return &OIDCConfig{
		Issuer:       raw.Issuer,
		ClientID:     clientID,
		ClientSecret: raw.ClientSecret,
		Scopes:       scopes,
	}, nil
}

// buildHTTPClient returns an http.Client. If insecure, TLS verification is skipped.
// Otherwise uses http.DefaultClient (which inherits the globally-configured trust store).
func buildHTTPClient(insecure bool) *http.Client {
	if !insecure {
		return http.DefaultClient
	}
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		},
	}
}
