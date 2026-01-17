package auth

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// SessionService handles ArgoCD authentication API calls
type SessionService struct {
	baseURL    string
	httpClient *http.Client
	insecure   bool
}

// SessionServiceConfig holds configuration for SessionService
type SessionServiceConfig struct {
	BaseURL  string
	Insecure bool
}

// LoginRequest represents the ArgoCD session create request
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the ArgoCD session create response
type LoginResponse struct {
	Token string `json:"token"`
}

// UserInfo represents the ArgoCD user info response
type UserInfo struct {
	LoggedIn bool   `json:"loggedIn"`
	Username string `json:"username"`
	Iss      string `json:"iss"`
}

// NewSessionService creates a new session service
func NewSessionService(cfg SessionServiceConfig) *SessionService {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   5 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
	}

	if cfg.Insecure {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	return &SessionService{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		insecure: cfg.Insecure,
	}
}

// Login authenticates with ArgoCD and returns a JWT token
func (s *SessionService) Login(ctx context.Context, username, password string) (string, error) {
	reqBody := LoginRequest{
		Username: username,
		Password: password,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	url := s.baseURL + "/api/v1/session"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("login request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read login response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Try to parse error message from response
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if json.Unmarshal(body, &errResp) == nil && (errResp.Error != "" || errResp.Message != "") {
			msg := errResp.Error
			if msg == "" {
				msg = errResp.Message
			}
			return "", fmt.Errorf("login failed: %s", msg)
		}
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to parse login response: %w", err)
	}

	if loginResp.Token == "" {
		return "", fmt.Errorf("login succeeded but no token returned")
	}

	return loginResp.Token, nil
}

// ValidateToken checks if a token is still valid by calling the userinfo endpoint
func (s *SessionService) ValidateToken(ctx context.Context, token string) error {
	url := s.baseURL + "/api/v1/session/userinfo"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("userinfo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrTokenInvalid
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("userinfo request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// ErrTokenInvalid indicates the token is invalid or expired
var ErrTokenInvalid = fmt.Errorf("token is invalid or expired")

// IsConnectionError checks if an error is a network/connection error
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "i/o timeout")
}
