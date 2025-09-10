package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/a9s/go-app/pkg/model"
)

// Client represents an HTTP client for ArgoCD API
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	insecure   bool
}

// NewClient creates a new ArgoCD API client
func NewClient(server *model.Server) *Client {
	// Create HTTP client with optional TLS config
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// If insecure flag is set, skip TLS verification
	if server.Insecure {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &Client{
		baseURL:    server.BaseURL,
		token:      server.Token,
		httpClient: httpClient,
		insecure:   server.Insecure,
	}
}

// Get performs a GET request
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "GET", path, nil)
}

// Post performs a POST request
func (c *Client) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.request(ctx, "POST", path, body)
}

// Put performs a PUT request
func (c *Client) Put(ctx context.Context, path string, body interface{}) ([]byte, error) {
	return c.request(ctx, "PUT", path, body)
}

// Delete performs a DELETE request
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "DELETE", path, nil)
}

// Stream performs a streaming GET request for Server-Sent Events
func (c *Client) Stream(ctx context.Context, path string) (io.ReadCloser, error) {
	url := c.baseURL + path
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}
	
	return resp.Body, nil
}

// request performs the actual HTTP request
func (c *Client) request(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.baseURL + path
	
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}