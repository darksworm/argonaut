//go:build e2e && unix

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/config"
)

// oidcTokenDelay is added to /oauth2/token responses in the startup test
// so the TUI has time to render "Re-authenticating via SSO" before reauth completes.
const oidcTokenDelay = 300 * time.Millisecond

// WriteArgoConfigNoToken writes an ArgoCD CLI config with an empty auth token
// but with sso:true and a refresh-token, simulating a user whose SSO token has
// expired before the first use. serverURL must include the scheme (e.g.
// "http://127.0.0.1:PORT") so that ensureHTTPS in the config layer preserves
// the http:// scheme for test servers.
func WriteArgoConfigNoToken(path, serverURL string) error {
	cfg := &config.ArgoCLIConfig{
		CurrentContext: "default",
		Contexts:       []config.ArgoContext{{Name: "default", Server: serverURL, User: "default-user"}},
		Servers:        []config.ArgoServer{{Server: serverURL, Insecure: true}},
		Users: []config.ArgoUser{{
			Name:         "default-user",
			AuthToken:    "",
			RefreshToken: "e2e-refresh-token",
			SSO:          true,
			OIDCIssuer:   serverURL,
		}},
	}
	return config.WriteCLIConfig(path, cfg)
}

// TestSSOReauthOnStartup verifies the startup flow when no token is present:
//  1. Config has an empty token → argonaut emits TriggerReauthMsg → shows "Re-authenticating via SSO"
//  2. NativeOIDCReauthProvider performs silent refresh via mock OIDC endpoints → gets fresh token
//  3. Argonaut resumes, validates auth, and loads apps
func TestSSOReauthOnStartup(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	const freshToken = "sso-fresh-token-startup"

	appsJSON := `[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`

	mux := http.NewServeMux()

	// srv is captured after creation; handlers reference it via closure.
	var srv *httptest.Server

	mux.HandleFunc("/api/v1/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"oidcConfig": map[string]interface{}{
				"issuer":          srv.URL,
				"cliClientID":     "argo-cd-cli",
				"requestedScopes": []string{"openid"},
			},
		})
	})

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                  srv.URL,
			"authorization_endpoint": srv.URL + "/oauth2/authorize",
			"token_endpoint":         srv.URL + "/oauth2/token",
		})
	})

	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		// Brief delay so the TUI can render "Re-authenticating via SSO" before reauth completes.
		time.Sleep(oidcTokenDelay)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id_token":      freshToken,
			"access_token":  freshToken,
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
		})
	})

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok != freshToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok != freshToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(wrapListResponse(appsJSON, "1000")))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok != freshToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		time.Sleep(200 * time.Millisecond)
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Write config with empty token → triggers reauth on startup.
	// OIDCIssuer points at srv.URL so the provider can discover OIDC endpoints.
	if err := WriteArgoConfigNoToken(cfgPath, srv.URL); err != nil {
		t.Fatalf("write no-token config: %v", err)
	}

	// Start in apps view so "demo" is immediately visible after load
	tf.extraConfig = `default_view = "apps"`

	if err := tf.StartAppArgs(
		[]string{"-argocd-config=" + cfgPath},
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Step 1: reauth pending view
	if !tf.WaitForPlain("Re-authenticating via SSO", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'Re-authenticating via SSO' message during startup reauth")
	}

	// Step 2+3: after silent refresh obtains fresh token, app loads and shows apps
	if !tf.WaitForPlain("demo", 10*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected app 'demo' to appear after successful startup reauth")
	}

	_ = tf.CtrlC()
}

// TestSSOReauthOnExpiredToken verifies the runtime expiry flow:
//  1. App starts with a valid token, list endpoint serves apps, stream setup returns 401
//     (simulating a token that is valid for REST but expired for streaming — triggers AuthErrorMsg)
//  2. Argonaut emits TriggerReauthMsg → shows "Re-authenticating via SSO"
//  3. NativeOIDCReauthProvider performs silent refresh via mock OIDC endpoints
//  4. Argonaut reloads apps via validateAuthentication
//
// Design note: there is no automatic watch reconnect in argonaut — when the stream
// setup (startWatchingApplications) returns 401, that is the point at which AuthErrorMsg
// is emitted. The list endpoint succeeds so "demo" is shown briefly before the stream
// setup fails and reauth is triggered.
func TestSSOReauthOnExpiredToken(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	const initialToken = "initial-token"
	const freshToken = "sso-fresh-token-runtime"

	appsJSON := `[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`

	mux := http.NewServeMux()

	var srv *httptest.Server

	mux.HandleFunc("/api/v1/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"oidcConfig": map[string]interface{}{
				"issuer":          srv.URL,
				"cliClientID":     "argo-cd-cli",
				"requestedScopes": []string{"openid"},
			},
		})
	})

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"issuer":                  srv.URL,
			"authorization_endpoint": srv.URL + "/oauth2/authorize",
			"token_endpoint":         srv.URL + "/oauth2/token",
		})
	})

	mux.HandleFunc("/oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"id_token":      freshToken,
			"access_token":  freshToken,
			"refresh_token": "new-refresh-token",
			"token_type":    "Bearer",
		})
	})

	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == initialToken || tok == freshToken {
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	})

	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == initialToken || tok == freshToken {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(wrapListResponse(appsJSON, "1000")))
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	})

	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})

	// Stream handler: reject initialToken with 401 (simulating expired token on stream setup),
	// accept freshToken and keep the stream open briefly.
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		tok := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if tok == initialToken {
			// Simulate expired token: stream setup fails with 401
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid session: token has invalid claims: token is expired","code":16,"message":"invalid session: token has invalid claims: token is expired"}`))
			return
		}
		if tok != freshToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		// Fresh token: open stream and hold briefly so test can confirm app loads
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if fl, ok := w.(http.Flusher); ok {
			fl.Flush()
		}
		time.Sleep(200 * time.Millisecond)
	})

	srv = httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	// issuerURL == srv.URL: mock OIDC endpoints live on the same server
	if err := WriteArgoConfigWithTokenSSO(cfgPath, srv.URL, initialToken, srv.URL); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	// Start in apps view so "demo" is immediately visible after load
	tf.extraConfig = `default_view = "apps"`

	if err := tf.StartAppArgs(
		[]string{"-argocd-config=" + cfgPath},
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Step 1+2: apps load (list succeeds), then stream setup returns 401 →
	// AuthErrorMsg → TriggerReauthMsg → ModeReauthPending
	if !tf.WaitForPlain("Re-authenticating via SSO", 10*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'Re-authenticating via SSO' after stream setup 401")
	}

	// Step 3+4: silent refresh obtains fresh token → apps reload
	if !tf.WaitForPlain("demo", 12*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'demo' app to reload after reauth")
	}

	_ = tf.CtrlC()
}
