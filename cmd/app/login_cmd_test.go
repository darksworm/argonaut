package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/darksworm/argonaut/pkg/config"
)

func TestRunLogin_PasswordFlow(t *testing.T) {
	const tok = "password-auth-token"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/session" && r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(map[string]string{"token": tok})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "session.yaml")
	t.Setenv("ARGONAUT_SESSION_CONFIG", cfgPath)

	// Pipe username + password to stdin
	r, w, _ := os.Pipe()
	w.WriteString("testuser\ntestpass\n")
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	code := RunLogin([]string{srv.URL})
	if code != 0 {
		t.Fatalf("RunLogin returned %d", code)
	}

	cfg, err := config.ReadCLIConfigFromPath(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if cfg.CurrentContext == "" {
		t.Fatal("expected current-context to be set")
	}
	gotTok, err := cfg.GetCurrentToken()
	if err != nil {
		t.Fatalf("get token: %v", err)
	}
	if gotTok != tok {
		t.Errorf("token: got %q, want %q", gotTok, tok)
	}
}

func TestRunLogin_MissingServer(t *testing.T) {
	code := RunLogin([]string{})
	if code == 0 {
		t.Fatal("expected non-zero exit when no server provided")
	}
}

func TestRunLogin_SavesContextName(t *testing.T) {
	const tok = "named-tok"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"token": tok})
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "session.yaml")

	r, w, _ := os.Pipe()
	w.WriteString("user\npass\n")
	w.Close()
	oldStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = oldStdin }()

	code := RunLogin([]string{"--name", "my-context", "--config", cfgPath, srv.URL})
	if code != 0 {
		t.Fatalf("RunLogin returned %d", code)
	}

	cfg, err := config.ReadCLIConfigFromPath(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if cfg.CurrentContext != "my-context" {
		t.Errorf("current-context: got %q, want %q", cfg.CurrentContext, "my-context")
	}
}
