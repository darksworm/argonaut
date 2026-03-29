//go:build e2e && unix

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeFakeArgocd writes a shell script that acts as a fake `argocd` binary.
// When invoked as `argocd login <server> --sso --config <path>`, it writes a
// fresh token to the config file at <path>.
// serverURL must be the full URL (e.g. "http://127.0.0.1:PORT") to match
// the format used by WriteArgoConfigWithToken.
func writeFakeArgocd(t *testing.T, dir, freshToken, serverURL string) string {
	t.Helper()
	scriptPath := filepath.Join(dir, "argocd")
	script := fmt.Sprintf(`#!/bin/sh
# Fake argocd for SSO reauth E2E tests.
# Finds --config <path> in args, writes a fresh config with a known token.
CONFIG_PATH=""
prev=""
for arg in "$@"; do
    if [ "$prev" = "--config" ]; then CONFIG_PATH="$arg"; fi
    prev="$arg"
done
if [ -n "$CONFIG_PATH" ]; then
    cat > "$CONFIG_PATH" << 'ARGOYAML'
contexts:
  - name: default
    server: %s
    user: default-user
servers:
  - server: %s
    insecure: true
users:
  - name: default-user
    auth-token: %s
current-context: default
ARGOYAML
fi
exit 0
`, serverURL, serverURL, freshToken)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("writeFakeArgocd: %v", err)
	}
	return scriptPath
}

// WriteArgoConfigNoToken writes an ArgoCD CLI config with an empty auth token.
// Uses the full serverURL (e.g. "http://127.0.0.1:PORT") as the server key,
// matching the format used by WriteArgoConfigWithToken.
func WriteArgoConfigNoToken(path, serverURL string) error {
	var y bytes.Buffer
	y.WriteString("contexts:\n")
	y.WriteString("  - name: default\n    server: " + serverURL + "\n    user: default-user\n")
	y.WriteString("servers:\n")
	y.WriteString("  - server: " + serverURL + "\n    insecure: true\n")
	y.WriteString("users:\n")
	y.WriteString("  - name: default-user\n    auth-token: \"\"\n")
	y.WriteString("current-context: default\n")
	return os.WriteFile(path, y.Bytes(), 0o644)
}

// TestSSOReauthOnStartup verifies the startup flow when no token is present:
//  1. Config has an empty token → argonaut emits TriggerReauthMsg → shows "Re-authenticating via SSO"
//  2. Fake argocd writes a fresh token to the config file
//  3. Argonaut resumes, validates auth, and loads apps
func TestSSOReauthOnStartup(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	const freshToken = "sso-fresh-token-startup"

	appsJSON := `[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]`

	mux := http.NewServeMux()

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

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}

	// Write config with empty token → triggers reauth on startup
	if err := WriteArgoConfigNoToken(cfgPath, srv.URL); err != nil {
		t.Fatalf("write no-token config: %v", err)
	}

	// Create fake argocd in a temp bin dir; it will write the fresh token on invocation
	binDir := t.TempDir()
	writeFakeArgocd(t, binDir, freshToken, srv.URL)

	// Start in apps view so "demo" is immediately visible after load
	tf.extraConfig = `default_view = "apps"`

	origPath := os.Getenv("PATH")
	if err := tf.StartAppArgs(
		[]string{"-argocd-config=" + cfgPath},
		"PATH="+binDir+":"+origPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Step 1: reauth pending view
	if !tf.WaitForPlain("Re-authenticating via SSO", 5*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'Re-authenticating via SSO' message during startup reauth")
	}

	// Step 2+3: after fake argocd exits with fresh token, app loads and shows apps
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
//  3. Fake argocd writes a fresh token; argonaut reloads apps via validateAuthentication
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

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfigWithToken(cfgPath, srv.URL, initialToken); err != nil {
		t.Fatalf("write initial config: %v", err)
	}

	binDir := t.TempDir()
	writeFakeArgocd(t, binDir, freshToken, srv.URL)

	// Start in apps view so "demo" is immediately visible after load
	tf.extraConfig = `default_view = "apps"`

	origPath := os.Getenv("PATH")
	if err := tf.StartAppArgs(
		[]string{"-argocd-config=" + cfgPath},
		"PATH="+binDir+":"+origPath,
	); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Step 1+2: apps load (list succeeds), then stream setup returns 401 →
	// AuthErrorMsg → TriggerReauthMsg → ModeReauthPending
	if !tf.WaitForPlain("Re-authenticating via SSO", 10*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'Re-authenticating via SSO' after stream setup 401")
	}

	// Step 3: after fake argocd writes fresh token, apps reload
	if !tf.WaitForPlain("demo", 12*time.Second) {
		t.Log(tf.SnapshotPlain())
		t.Fatal("expected 'demo' app to reload after reauth")
	}

	_ = tf.CtrlC()
}
