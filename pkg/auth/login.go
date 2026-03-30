package auth

import (
	"context"
	"strings"

	"github.com/darksworm/argonaut/pkg/model"
)

// ReauthProvider performs SSO re-authentication.
// On success it writes the new token(s) to configPath and returns the new auth token.
type ReauthProvider interface {
	Reauth(ctx context.Context, server *model.Server, configPath, contextName string) (string, error)
}

// StripProtocol removes the http:// or https:// scheme from a URL.
func StripProtocol(serverURL string) string {
	serverURL = strings.TrimPrefix(serverURL, "https://")
	serverURL = strings.TrimPrefix(serverURL, "http://")
	return serverURL
}
