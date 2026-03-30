package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/oidc"
	"golang.org/x/term"
)

// RunLogin implements the `argonaut login <server> [flags]` subcommand.
// Returns an exit code (0 = success).
func RunLogin(args []string) int {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	var (
		ssoFlag             bool
		insecureFlag        bool
		plaintextFlag       bool
		grpcWebFlag         bool
		grpcWebRootPathFlag string
		nameFlag            string
		portFlag            int
		noBrowserFlag       bool
		cfgPathFlag         string
	)
	fs.BoolVar(&ssoFlag, "sso", false, "Use SSO/OIDC browser authentication")
	fs.BoolVar(&insecureFlag, "insecure", false, "Skip TLS certificate verification")
	fs.BoolVar(&plaintextFlag, "plaintext", false, "Use HTTP instead of HTTPS")
	fs.BoolVar(&grpcWebFlag, "grpc-web", false, "Enable gRPC-web protocol")
	fs.StringVar(&grpcWebRootPathFlag, "grpc-web-root-path", "", "Enable gRPC-web protocol with this root path prefix")
	fs.StringVar(&nameFlag, "name", "", "Context name (default: server hostname)")
	fs.IntVar(&portFlag, "port", 8085, "Local port for SSO callback server")
	fs.BoolVar(&noBrowserFlag, "no-browser", false, "Print SSO URL instead of opening browser")
	fs.StringVar(&cfgPathFlag, "config", "", "Config file path (default: ~/.config/argonaut/session.yaml)")

	if err := fs.Parse(args); err != nil {
		return 1
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: argonaut login <server> [flags]")
		fs.PrintDefaults()
		return 1
	}

	var serverURL string
	if plaintextFlag {
		serverURL = "http://" + stripScheme(fs.Arg(0))
	} else {
		serverURL = ensureHTTPSScheme(fs.Arg(0))
	}
	contextName := nameFlag
	if contextName == "" {
		contextName = stripScheme(serverURL)
	}

	cfgPath := cfgPathFlag
	if cfgPath == "" {
		cfgPath = config.GetArgonautSessionPath()
	}

	var (
		authToken    string
		refreshToken string
		oidcIssuer   string
		isSSO        bool
		err          error
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if ssoFlag {
		authToken, refreshToken, oidcIssuer, err = doSSOLogin(ctx, serverURL, insecureFlag, portFlag, noBrowserFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "SSO login failed: %v\n", err)
			return 1
		}
		isSSO = true
	} else {
		authToken, err = doPasswordLogin(ctx, serverURL, insecureFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Password login failed: %v\n", err)
			return 1
		}
	}

	if err := saveSession(cfgPath, serverURL, contextName, authToken, refreshToken, oidcIssuer, insecureFlag, plaintextFlag, grpcWebFlag, grpcWebRootPathFlag, isSSO); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to save config: %v\n", err)
		return 1
	}

	fmt.Printf("Context %q saved to %s\n", contextName, cfgPath)
	return 0
}

func doSSOLogin(ctx context.Context, serverURL string, insecure bool, port int, noBrowser bool) (authToken, refreshToken, issuer string, err error) {
	oidcCfg, err := oidc.FetchOIDCConfig(ctx, serverURL, insecure)
	if err != nil {
		return "", "", "", fmt.Errorf("fetching OIDC config: %w", err)
	}

	endpoints, err := oidc.DiscoverEndpoints(ctx, oidcCfg.Issuer, insecure)
	if err != nil {
		return "", "", "", fmt.Errorf("OIDC discovery: %w", err)
	}

	redirectURI, resultCh, cleanup, err := oidc.StartCallbackServer(ctx, port)
	if err != nil {
		return "", "", "", fmt.Errorf("starting callback server: %w", err)
	}
	defer cleanup()

	state := fmt.Sprintf("%d", time.Now().UnixNano())
	verifier, authURL, err := oidc.AuthCodeURL(endpoints, oidcCfg, redirectURI, state)
	if err != nil {
		return "", "", "", err
	}

	if noBrowser {
		fmt.Printf("Open this URL in your browser to authenticate:\n%s\n\n", authURL)
	} else {
		fmt.Println("Opening browser for SSO authentication...")
		if openErr := oidc.OpenBrowser(authURL); openErr != nil {
			fmt.Printf("Could not open browser automatically. Open this URL:\n%s\n\n", authURL)
		}
	}
	fmt.Println("Waiting for callback...")

	select {
	case result := <-resultCh:
		if result.Err != nil {
			return "", "", "", result.Err
		}
		if result.State != state {
			return "", "", "", fmt.Errorf("state mismatch: CSRF check failed")
		}
		tokens, err := oidc.ExchangeCode(ctx, endpoints, oidcCfg, result.Code, verifier, redirectURI, insecure)
		if err != nil {
			return "", "", "", err
		}
		fmt.Println("SSO authentication successful.")
		return tokens.AuthToken, tokens.RefreshToken, oidcCfg.Issuer, nil
	case <-ctx.Done():
		return "", "", "", fmt.Errorf("timed out waiting for SSO callback")
	}
}

func doPasswordLogin(ctx context.Context, serverURL string, insecure bool) (string, error) {
	fmt.Print("Username: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	username := strings.TrimSpace(scanner.Text())

	fmt.Print("Password: ")
	var password string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		raw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("reading password: %w", err)
		}
		password = string(raw)
	} else {
		// Non-interactive (e.g., tests piping stdin)
		scanner.Scan()
		password = strings.TrimSpace(scanner.Text())
	}

	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(serverURL, "/")+"/api/v1/session",
		bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.DefaultClient
	if insecure {
		client = buildLoginHTTPClient()
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed (HTTP %d): %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding login response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("server returned empty token")
	}
	return result.Token, nil
}

func saveSession(cfgPath, serverURL, contextName, authToken, refreshToken, oidcIssuer string, insecure, plainText, grpcWeb bool, grpcWebRootPath string, sso bool) error {
	existing, err := config.ReadCLIConfigFromPath(cfgPath)
	if err != nil {
		existing = &config.ArgoCLIConfig{}
	}

	serverHost := stripScheme(serverURL)

	// Upsert server
	serverFound := false
	for i, s := range existing.Servers {
		if s.Server == serverHost {
			existing.Servers[i].Insecure = insecure
			existing.Servers[i].PlainText = plainText
			existing.Servers[i].GrpcWeb = grpcWeb
			existing.Servers[i].GrpcWebRootPath = grpcWebRootPath
			serverFound = true
			break
		}
	}
	if !serverFound {
		existing.Servers = append(existing.Servers, config.ArgoServer{
			Server:          serverHost,
			Insecure:        insecure,
			PlainText:       plainText,
			GrpcWeb:         grpcWeb,
			GrpcWebRootPath: grpcWebRootPath,
		})
	}

	// Upsert user
	userName := contextName
	userFound := false
	for i, u := range existing.Users {
		if u.Name == userName {
			existing.Users[i].AuthToken = authToken
			existing.Users[i].RefreshToken = refreshToken
			existing.Users[i].OIDCIssuer = oidcIssuer
			existing.Users[i].SSO = sso
			userFound = true
			break
		}
	}
	if !userFound {
		existing.Users = append(existing.Users, config.ArgoUser{
			Name:         userName,
			AuthToken:    authToken,
			RefreshToken: refreshToken,
			OIDCIssuer:   oidcIssuer,
			SSO:          sso,
		})
	}

	// Upsert context
	ctxFound := false
	for i, c := range existing.Contexts {
		if c.Name == contextName {
			existing.Contexts[i].Server = serverHost
			existing.Contexts[i].User = userName
			ctxFound = true
			break
		}
	}
	if !ctxFound {
		existing.Contexts = append(existing.Contexts, config.ArgoContext{
			Name:   contextName,
			Server: serverHost,
			User:   userName,
		})
	}

	existing.CurrentContext = contextName
	return config.WriteCLIConfig(cfgPath, existing)
}

func ensureHTTPSScheme(server string) string {
	if strings.HasPrefix(server, "http://") || strings.HasPrefix(server, "https://") {
		return server
	}
	return "https://" + server
}

func stripScheme(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return strings.TrimRight(url, "/")
}

// buildLoginHTTPClient returns an http.Client with TLS verification disabled.
// Used only for --insecure password login to handle self-signed certs on the ArgoCD server.
func buildLoginHTTPClient() *http.Client {
	tr := &http.Transport{
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
	}
	return &http.Client{Transport: tr}
}
