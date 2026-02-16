package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	cblog "github.com/charmbracelet/log"
	appcontext "github.com/darksworm/argonaut/pkg/context"
	apperrors "github.com/darksworm/argonaut/pkg/errors"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/retry"
)

// Client represents an HTTP client for ArgoCD API
type Client struct {
	baseURL          string
	token            string
	httpClient       *http.Client
	streamHTTPClient *http.Client // Separate client for SSE streams (no ResponseHeaderTimeout)
	insecure         bool
	grpcWebRootPath  string
}

var customHTTPClient *http.Client

// SetHTTPClient sets a custom HTTP client to be used by all new Client instances
func SetHTTPClient(client *http.Client) {
	customHTTPClient = client
}

// NewClient creates a new ArgoCD API client
func NewClient(server *model.Server) *Client {
	var httpClient *http.Client

	// Use custom HTTP client if available
	if customHTTPClient != nil {
		// Clone the custom client to avoid modifying the shared instance
		httpClient = &http.Client{
			Transport:     customHTTPClient.Transport,
			CheckRedirect: customHTTPClient.CheckRedirect,
			Jar:           customHTTPClient.Jar,
			Timeout:       customHTTPClient.Timeout,
		}

		// If insecure flag is set, we need to modify the transport
		if server.Insecure {
			// Clone the transport to avoid modifying the shared one
			if transport, ok := httpClient.Transport.(*http.Transport); ok {
				clonedTransport := transport.Clone()
				if clonedTransport.TLSClientConfig == nil {
					clonedTransport.TLSClientConfig = &tls.Config{}
				} else {
					// Clone TLS config to avoid modifying shared config
					clonedTLSConfig := clonedTransport.TLSClientConfig.Clone()
					clonedTransport.TLSClientConfig = clonedTLSConfig
				}
				clonedTransport.TLSClientConfig.InsecureSkipVerify = true
				httpClient.Transport = clonedTransport
			}
		}
	} else {
		// Create HTTP transport with reasonable timeouts for problematic TLS
		transport := &http.Transport{
			// Connection establishment timeout - should be very fast
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   10 * time.Second, // Increased for slow certificate validation
			ResponseHeaderTimeout: 10 * time.Second, // Increased to match TLS timeout
			// Keep connections alive for efficiency
			IdleConnTimeout:     30 * time.Second,
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
		}

		// If insecure flag is set, skip TLS verification
		if server.Insecure {
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: true,
			}
		}

		// Create HTTP client without a default timeout (we'll use context timeouts)
		httpClient = &http.Client{
			Transport: transport,
			// No timeout here - we use context timeouts for request-specific timing
		}
	}

	// Create a separate HTTP client for SSE streams.
	// Streams must not have ResponseHeaderTimeout because the server may
	// not send headers until the first event (which could be minutes away
	// when using resourceVersion).
	streamTransport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second,
		// No ResponseHeaderTimeout for streams
		IdleConnTimeout:     30 * time.Second,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 2,
	}
	if server.Insecure {
		streamTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}
	// If we cloned a custom transport for regular requests, also apply its TLS config to stream
	if customHTTPClient != nil {
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			streamTransport = transport.Clone()
			// Remove ResponseHeaderTimeout for streams
			streamTransport.ResponseHeaderTimeout = 0
		}
	}
	streamHTTPClient := &http.Client{
		Transport: streamTransport,
	}

	return &Client{
		baseURL:          server.BaseURL,
		token:            server.Token,
		httpClient:       httpClient,
		streamHTTPClient: streamHTTPClient,
		insecure:         server.Insecure,
		grpcWebRootPath:  server.GrpcWebRootPath,
	}
}

// sanitizeURL removes any embedded credentials from a URL for safe logging
func sanitizeURL(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.User != nil {
		u.User = nil
		return u.String()
	}
	return rawURL
}

// buildURL constructs the full URL including the gRPC-web root path if configured
func (c *Client) buildURL(path string) string {
	if c.grpcWebRootPath != "" {
		// Trim leading and trailing slashes from root path, similar to ArgoCD implementation
		rootPath := strings.TrimRight(strings.TrimLeft(c.grpcWebRootPath, "/"), "/")
		return fmt.Sprintf("%s/%s%s", c.baseURL, rootPath, path)
	}
	return c.baseURL + path
}

// Get performs a GET request with retry logic.
// Callers are responsible for setting a timeout on ctx (e.g. via appcontext.WithAPITimeout
// or WithMinAPITimeout). Do not add a timeout here — it would undercut WithMinAPITimeout
// callers that need a longer deadline for slow operations like diffs and rollbacks.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	var result []byte
	err := retry.RetryNetworkOperation(ctx, fmt.Sprintf("GET %s", path), func(attempt int) error {
		var opErr error
		result, opErr = c.request(ctx, "GET", path, nil)
		return opErr
	})

	return result, err
}

// Post performs a POST request with retry logic.
// See Get for timeout responsibility.
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	var result []byte
	err := retry.RetryNetworkOperation(ctx, fmt.Sprintf("POST %s", path), func(attempt int) error {
		var opErr error
		result, opErr = c.request(ctx, "POST", path, body)
		return opErr
	})

	return result, err
}

// Put performs a PUT request with retry logic.
// See Get for timeout responsibility.
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	var result []byte
	err := retry.RetryNetworkOperation(ctx, fmt.Sprintf("PUT %s", path), func(attempt int) error {
		var opErr error
		result, opErr = c.request(ctx, "PUT", path, body)
		return opErr
	})

	return result, err
}

// Delete performs a DELETE request with retry logic.
// See Get for timeout responsibility.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	var result []byte
	err := retry.RetryNetworkOperation(ctx, fmt.Sprintf("DELETE %s", path), func(attempt int) error {
		var opErr error
		result, opErr = c.request(ctx, "DELETE", path, nil)
		return opErr
	})

	return result, err
}

// Stream performs a streaming GET request for Server-Sent Events
// Returns both the stream body and response headers for potential future use
func (c *Client) Stream(ctx context.Context, path string) (*StreamResponse, error) {
	// No timeout for streams - managed by caller context
	url := c.buildURL(path)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorNetwork, "REQUEST_CREATE_FAILED",
			"Failed to create stream request").
			WithContext("url", url).
			WithUserAction("Check the server URL and try again")
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	// Log the stream request for debugging
	cblog.With("component", "api", "op", "stream").Debug("Making stream request",
		"method", "GET",
		"url", sanitizeURL(url),
		"type", "SSE",
	)

	// Use the stream-specific HTTP client (no ResponseHeaderTimeout)
	resp, err := c.streamHTTPClient.Do(req)
	if err != nil {
		// Check for timeout
		if timeoutErr := appcontext.HandleTimeout(ctx, appcontext.OpStream); timeoutErr != nil {
			return nil, timeoutErr.WithContext("url", url)
		}

		// Log the actual error at warn level so users can see what went wrong
		cblog.With("component", "api", "op", "http").Warn("HTTP request failed",
			"method", "GET",
			"url", sanitizeURL(url),
			"error", err.Error(),
		)

		return nil, apperrors.Wrap(err, apperrors.ErrorNetwork, "STREAM_REQUEST_FAILED",
			fmt.Sprintf("Stream request failed: %v", err)).
			WithContext("url", url).
			AsRecoverable().
			WithUserAction("Check your network connection and ArgoCD server status")
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		return nil, c.createAPIError(resp.StatusCode, string(body), url).
			WithContext("method", "GET").
			WithContext("path", path)
	}

	return &StreamResponse{
		Body:    resp.Body,
		Headers: resp.Header,
	}, nil
}

// request performs the actual HTTP request
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	// Retrieve the original timeout duration for accurate error messages.
	// Uses the value stored by WithAPITimeout/WithMinAPITimeout at context
	// creation time, avoiding time.Until(deadline) which drifts on retries.
	var timeoutStr string
	if d, ok := appcontext.GetTimeoutDuration(ctx); ok {
		timeoutStr = d.Round(time.Second).String()
	} else {
		timeoutStr = appcontext.DefaultTimeouts.API.String()
	}

	url := c.buildURL(path)

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, apperrors.Wrap(err, apperrors.ErrorValidation, "JSON_MARSHAL_FAILED",
				"Failed to marshal request body").
				WithContext("method", method).
				WithContext("path", path).
				WithUserAction("Check the request data format")
		}
		reqBody = bytes.NewReader(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorNetwork, "REQUEST_CREATE_FAILED",
			"Failed to create HTTP request").
			WithContext("method", method).
			WithContext("url", url).
			WithUserAction("Check the server URL and try again")
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Log the request for debugging
	cblog.With("component", "api", "op", "http").Debug("Making HTTP request",
		"method", method,
		"url", sanitizeURL(url),
		"timeout", timeoutStr,
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Check for timeout first - context errors have priority
		if ctx.Err() == context.DeadlineExceeded {
			// Log the timeout at warn level for visibility
			cblog.With("component", "api", "op", "http").Warn("Request timed out",
				"method", method,
				"url", sanitizeURL(url),
				"timeout", "context deadline exceeded",
				"error", err.Error(),
			)
			return nil, apperrors.TimeoutError("REQUEST_TIMEOUT",
				fmt.Sprintf("Request timed out after %s - server did not respond in time", timeoutStr)).
				WithContext("method", method).
				WithContext("url", url).
				WithContext("timeout", timeoutStr).
				WithUserAction("Check network connection and server status. For slow servers, increase request_timeout in config")
		}

		if ctx.Err() == context.Canceled {
			// Log the cancellation for debugging
			cblog.With("component", "api", "op", "http").Warn("Request was canceled",
				"method", method,
				"url", sanitizeURL(url),
				"error", err.Error(),
			)
			return nil, apperrors.New(apperrors.ErrorInternal, "REQUEST_CANCELLED",
				"Request was cancelled").
				WithContext("method", method).
				WithContext("url", url)
		}

		// Check if it's a network timeout from the transport layer
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Log network timeout for visibility
			cblog.With("component", "api", "op", "http").Warn("Network timeout",
				"method", method,
				"url", sanitizeURL(url),
				"timeout", "network/transport layer",
				"error", err.Error(),
			)
			return nil, apperrors.TimeoutError("NETWORK_TIMEOUT",
				fmt.Sprintf("Network connection timed out after %s", timeoutStr)).
				WithContext("method", method).
				WithContext("url", url).
				WithContext("timeout", timeoutStr).
				WithUserAction("Check network/firewall settings and TLS configuration. Increase request_timeout if needed")
		}

		// Log the actual error at warn level so users can see what went wrong
		// This will show TLS certificate errors, connection refused, etc.
		cblog.With("component", "api", "op", "http").Warn("HTTP request failed",
			"method", method,
			"url", sanitizeURL(url),
			"error", err.Error(),
		)

		return nil, apperrors.Wrap(err, apperrors.ErrorNetwork, "HTTP_REQUEST_FAILED",
			fmt.Sprintf("HTTP request failed: %v", err)).
			WithContext("method", method).
			WithContext("url", url).
			AsRecoverable().
			WithUserAction("Check your network connection and ArgoCD server status")
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrorNetwork, "RESPONSE_READ_FAILED",
			"Failed to read response body").
			WithContext("method", method).
			WithContext("url", url).
			WithUserAction("Try the request again")
	}

	if resp.StatusCode >= 400 {
		// Log request/response metadata at error level
		cblog.With("component", "api", "op", "http").Error("http error",
			"method", method,
			"url", sanitizeURL(url),
			"status", resp.StatusCode,
			"len", len(respBody),
		)
		// Log body content at debug level (may contain details); truncate to avoid huge logs
		body := string(respBody)
		const maxLen = 2048
		if len(body) > maxLen {
			body = body[:maxLen] + "…"
		}
		cblog.With("component", "api").Debug("response body", "body", body)

		return nil, c.createAPIError(resp.StatusCode, string(respBody), url).
			WithContext("method", method).
			WithContext("path", path)
	}

	return respBody, nil
}

// argocdRuntimeError represents the standard ArgoCD error response format
// See: runtimeError in ArgoCD swagger docs
type argocdRuntimeError struct {
	Code    int    `json:"code"`
	Error   string `json:"error"`
	Message string `json:"message"`
}

// parseArgocdError attempts to parse an ArgoCD runtimeError from JSON response body
// Returns the parsed message if successful, or empty string if parsing fails
func parseArgocdError(responseBody string) *argocdRuntimeError {
	if responseBody == "" {
		return nil
	}
	var runtimeErr argocdRuntimeError
	if err := json.Unmarshal([]byte(responseBody), &runtimeErr); err != nil {
		return nil
	}
	// ArgoCD typically uses "message" for the human-readable error
	if runtimeErr.Message != "" || runtimeErr.Error != "" {
		return &runtimeErr
	}
	return nil
}

// createAPIError creates a structured API error based on status code and response
func (c *Client) createAPIError(statusCode int, responseBody, url string) *apperrors.ArgonautError {
	var category apperrors.ErrorCategory
	var code string
	var message string
	var userAction string
	var recoverable bool

	// Try to parse ArgoCD's runtimeError format first
	argoErr := parseArgocdError(responseBody)

	switch statusCode {
	case 401:
		category = apperrors.ErrorAuth
		code = "UNAUTHORIZED"
		message = "Authentication required or token expired"
		userAction = "Please run 'argocd login' to authenticate"
		recoverable = false

	case 403:
		category = apperrors.ErrorPermission
		code = "FORBIDDEN"
		message = "Insufficient permissions for this operation"
		userAction = "Check your ArgoCD user permissions"
		recoverable = false

	case 404:
		category = apperrors.ErrorAPI
		code = "NOT_FOUND"
		message = "Requested resource not found"
		userAction = "Verify the resource exists and the path is correct"
		recoverable = false

	case 409:
		category = apperrors.ErrorValidation
		code = "CONFLICT"
		message = "Request conflicts with current state"
		userAction = "Check the current state and adjust your request"
		recoverable = true

	case 429:
		category = apperrors.ErrorAPI
		code = "RATE_LIMITED"
		message = "Too many requests - rate limited"
		userAction = "Wait a moment and try again"
		recoverable = true

	case 500, 502, 503, 504:
		category = apperrors.ErrorAPI
		code = "SERVER_ERROR"
		message = "ArgoCD server error"
		userAction = "Check ArgoCD server status and try again"
		recoverable = true

	default:
		category = apperrors.ErrorAPI
		code = "API_ERROR"
		message = fmt.Sprintf("API request failed with status %d", statusCode)
		userAction = "Check the request and try again"
		recoverable = true
	}

	// Try to extract more specific error from ArgoCD's runtimeError response
	if argoErr != nil {
		// Prefer "message" field, fall back to "error" field
		argoMessage := argoErr.Message
		if argoMessage == "" {
			argoMessage = argoErr.Error
		}
		if argoMessage != "" {
			// Use ArgoCD's message as it's typically more informative
			message = argoMessage
			cblog.With("component", "api").Debug("Parsed ArgoCD error",
				"code", argoErr.Code, "message", argoMessage, "statusCode", statusCode)
		}
	} else if responseBody != "" && len(responseBody) < 500 {
		// Fallback: check for common error patterns in raw body
		if strings.Contains(strings.ToLower(responseBody), "unauthorized") ||
			strings.Contains(strings.ToLower(responseBody), "invalid token") ||
			strings.Contains(strings.ToLower(responseBody), "authentication") {
			category = apperrors.ErrorAuth
			code = "AUTHENTICATION_FAILED"
			userAction = "Please run 'argocd login' to authenticate"
		}
	}

	err := apperrors.New(category, code, message).
		WithSeverity(apperrors.SeverityMedium).
		WithDetails(responseBody).
		WithContext("statusCode", statusCode).
		WithContext("url", url).
		WithUserAction(userAction)

	if recoverable {
		err.AsRecoverable()
	}

	return err
}
