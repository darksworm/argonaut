package oidc_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/oidc"
)

func TestStartCallbackServer_ReceivesCode(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redirectURI, resultCh, cleanup, err := oidc.StartCallbackServer(ctx, 0) // port 0 = random
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	redirectURI, resultCh, cleanup, err := oidc.StartCallbackServer(ctx, 0)
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

func TestStartCallbackServer_ContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	_, resultCh, cleanup, err := oidc.StartCallbackServer(ctx, 0)
	if err != nil {
		t.Fatalf("start server: %v", err)
	}
	defer cleanup()

	// Cancel the context — should deliver an error on the channel
	cancel()

	select {
	case result := <-resultCh:
		if result.Err == nil {
			t.Fatal("expected error when context cancelled")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for cancellation result")
	}
}
