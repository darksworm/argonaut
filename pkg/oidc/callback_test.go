package oidc_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/oidc"
)

func TestStartCallbackServer_ReceivesCode(t *testing.T) {
	redirectURI, resultCh, cleanup, err := oidc.StartCallbackServer(0) // port 0 = random
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer cleanup()

	if redirectURI == "" {
		t.Fatal("expected non-empty redirect URI")
	}

	// Simulate browser redirect
	go func() {
		http.Get(redirectURI + "?code=test-code&state=test-state") //nolint:errcheck
	}()

	select {
	case result := <-resultCh:
		if result.Err != nil {
			t.Fatalf("unexpected error: %v", result.Err)
		}
		if result.Code != "test-code" {
			t.Errorf("code: got %q", result.Code)
		}
		if result.State != "test-state" {
			t.Errorf("state: got %q", result.State)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}

func TestStartCallbackServer_ErrorCallback(t *testing.T) {
	redirectURI, resultCh, cleanup, err := oidc.StartCallbackServer(0)
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer cleanup()

	go func() {
		http.Get(redirectURI + "?error=access_denied&error_description=user+denied") //nolint:errcheck
	}()

	select {
	case result := <-resultCh:
		if result.Err == nil {
			t.Fatal("expected error for error callback")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for callback")
	}
}
