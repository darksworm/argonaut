package oidc

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

// CallbackResult is the result of the OAuth2 redirect callback.
type CallbackResult struct {
	Code  string
	State string
	Err   error
}

// StartCallbackServer starts a local HTTP server that handles the OAuth2 callback.
// Pass port=0 to pick a random available port.
// Returns the redirect URI (http://localhost:{port}/auth/callback), a channel
// that receives one result, and a cleanup function to shut down the server.
func StartCallbackServer(port int) (redirectURI string, resultCh <-chan CallbackResult, cleanup func(), err error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return "", nil, nil, fmt.Errorf("starting callback server: %w", err)
	}

	addr := ln.Addr().(*net.TCPAddr)
	redirectURI = fmt.Sprintf("http://localhost:%d/auth/callback", addr.Port)

	ch := make(chan CallbackResult, 1)
	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		errParam := r.URL.Query().Get("error")
		if errParam != "" {
			desc := r.URL.Query().Get("error_description")
			fmt.Fprintf(w, "Authentication failed: %s — %s. You may close this tab.", errParam, desc)
			ch <- CallbackResult{Err: fmt.Errorf("OIDC callback error: %s: %s", errParam, desc)}
			return
		}
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		fmt.Fprintf(w, "Authentication successful. You may close this tab.")
		ch <- CallbackResult{Code: code, State: state}
	})

	go srv.Serve(ln) //nolint:errcheck

	return redirectURI, ch, func() {
		srv.Shutdown(context.Background()) //nolint:errcheck
	}, nil
}
