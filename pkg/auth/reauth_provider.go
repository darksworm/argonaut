package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/oidc"
)

// NativeOIDCReauthProvider implements ReauthProvider using the native OIDC flow.
// It tries silent token refresh first; falls back to browser-based auth if refresh fails.
type NativeOIDCReauthProvider struct{}

func (p *NativeOIDCReauthProvider) Reauth(ctx context.Context, server *model.Server, configPath, contextName string) (string, error) {
	// Attempt silent refresh if we have a refresh token.
	if server.RefreshToken != "" {
		if tok, err := p.tryRefresh(ctx, server); err == nil {
			if writeErr := updateConfigToken(configPath, contextName, tok.AuthToken, tok.RefreshToken); writeErr == nil {
				return tok.AuthToken, nil
			}
		}
		// Silent refresh failed; fall through to browser flow.
	}
	return p.browserFlow(ctx, server, configPath, contextName)
}

func (p *NativeOIDCReauthProvider) tryRefresh(ctx context.Context, server *model.Server) (*oidc.TokenSet, error) {
	oidcCfg, err := oidc.FetchOIDCConfig(ctx, server.BaseURL, server.Insecure)
	if err != nil {
		return nil, err
	}
	endpoints, err := oidc.DiscoverEndpoints(ctx, oidcCfg.Issuer, server.Insecure)
	if err != nil {
		return nil, err
	}
	return oidc.RefreshTokens(ctx, endpoints, oidcCfg, server.RefreshToken, server.Insecure)
}

func (p *NativeOIDCReauthProvider) browserFlow(ctx context.Context, server *model.Server, configPath, contextName string) (string, error) {
	oidcCfg, err := oidc.FetchOIDCConfig(ctx, server.BaseURL, server.Insecure)
	if err != nil {
		return "", fmt.Errorf("fetching OIDC config: %w", err)
	}
	endpoints, err := oidc.DiscoverEndpoints(ctx, oidcCfg.Issuer, server.Insecure)
	if err != nil {
		return "", fmt.Errorf("OIDC discovery: %w", err)
	}
	redirectURI, resultCh, cleanup, err := oidc.StartCallbackServer(ctx, 8085)
	if err != nil {
		return "", fmt.Errorf("starting callback server: %w", err)
	}
	defer cleanup()

	state := fmt.Sprintf("%d", time.Now().UnixNano())
	verifier, authURL, err := oidc.AuthCodeURL(endpoints, oidcCfg, redirectURI, state)
	if err != nil {
		return "", err
	}

	// Best-effort browser open; if it fails the UI should show the URL.
	_ = oidc.OpenBrowser(authURL)

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return "", result.Err
		}
		if result.State != state {
			return "", fmt.Errorf("state mismatch: possible CSRF")
		}
		tokens, err := oidc.ExchangeCode(ctx, endpoints, oidcCfg, result.Code, verifier, redirectURI, server.Insecure)
		if err != nil {
			return "", err
		}
		if err := updateConfigToken(configPath, contextName, tokens.AuthToken, tokens.RefreshToken); err != nil {
			return "", err
		}
		return tokens.AuthToken, nil
	case <-ctx.Done():
		return "", fmt.Errorf("timed out waiting for SSO callback")
	}
}

// updateConfigToken reads configPath, updates the named context's user auth-token and refresh-token, writes back.
func updateConfigToken(configPath, contextName, authToken, refreshToken string) error {
	cfg, err := config.ReadCLIConfigFromPath(configPath)
	if err != nil {
		return fmt.Errorf("re-reading config after reauth: %w", err)
	}
	var userName string
	for _, ctx := range cfg.Contexts {
		if ctx.Name == contextName {
			userName = ctx.User
			break
		}
	}
	if userName == "" {
		userName = contextName
	}
	for i, u := range cfg.Users {
		if u.Name == userName {
			cfg.Users[i].AuthToken = authToken
			cfg.Users[i].RefreshToken = refreshToken
			return config.WriteCLIConfig(configPath, cfg)
		}
	}
	return fmt.Errorf("user %q not found in config", userName)
}
