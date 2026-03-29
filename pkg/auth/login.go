package auth

import (
	"os/exec"
	"strings"
)

// LoginParams holds the parameters needed to reconstruct an argocd login command.
// argocd login does NOT merge existing settings — every flag must be re-specified.
type LoginParams struct {
	ServerURL       string // bare hostname (no https:// prefix), e.g. "argocd.example.com"
	ContextName     string // passed as --name; identifies the context entry to update
	Insecure        bool
	GrpcWeb         bool
	GrpcWebRootPath string
	ConfigPath      string // empty = use argocd default (~/.config/argocd/config)
}

// JWTAuthProvider builds the CLI command used for SSO re-authentication.
// Injected into Model so tests can swap in a fake.
type JWTAuthProvider interface {
	LoginCmd(params LoginParams) *exec.Cmd
}

// ArgocdCLIAuthProvider implements JWTAuthProvider using the real argocd CLI.
type ArgocdCLIAuthProvider struct{}

// LoginCmd builds: argocd login <ServerURL> --sso [flags...]
func (a ArgocdCLIAuthProvider) LoginCmd(params LoginParams) *exec.Cmd {
	args := []string{"login", params.ServerURL, "--sso"}
	if params.ContextName != "" {
		args = append(args, "--name", params.ContextName)
	}
	if params.Insecure {
		args = append(args, "--insecure")
	}
	if params.GrpcWeb {
		args = append(args, "--grpc-web")
	}
	if params.GrpcWebRootPath != "" {
		args = append(args, "--grpc-web-root-path", params.GrpcWebRootPath)
	}
	if params.ConfigPath != "" {
		args = append(args, "--config", params.ConfigPath)
	}
	return exec.Command("argocd", args...)
}

// StripProtocol removes the http:// or https:// scheme from a URL, returning
// the bare hostname (and optional port). Bare hostnames are returned unchanged.
func StripProtocol(url string) string {
	if strings.HasPrefix(url, "https://") {
		return url[len("https://"):]
	}
	if strings.HasPrefix(url, "http://") {
		return url[len("http://"):]
	}
	return url
}
