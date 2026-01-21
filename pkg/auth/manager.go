package auth

import (
	"context"
	"errors"
	"fmt"

	cblog "github.com/charmbracelet/log"
)

// AuthManager coordinates authentication flow
type AuthManager struct {
	keychain   KeychainStore
	sessionSvc *SessionService
	serverURL  string
	insecure   bool
}

// AuthManagerConfig holds configuration for AuthManager
type AuthManagerConfig struct {
	ServerURL string
	Insecure  bool
	Keychain  KeychainStore // Optional, uses default if nil
}

// AuthResult represents the result of an authentication attempt
type AuthResult struct {
	Token    string
	Username string
}

// NewAuthManager creates a new auth manager
func NewAuthManager(cfg AuthManagerConfig) *AuthManager {
	keychain := cfg.Keychain
	if keychain == nil {
		keychain = NewKeychainStore()
	}

	sessionSvc := NewSessionService(SessionServiceConfig{
		BaseURL:  cfg.ServerURL,
		Insecure: cfg.Insecure,
	})

	return &AuthManager{
		keychain:   keychain,
		sessionSvc: sessionSvc,
		serverURL:  cfg.ServerURL,
		insecure:   cfg.Insecure,
	}
}

// TryAutoLogin attempts to authenticate using stored credentials from keychain
// Returns the token if successful, or an error if auto-login failed
func (m *AuthManager) TryAutoLogin(ctx context.Context) (*AuthResult, error) {
	log := cblog.With("component", "auth-manager")

	// First, try to load a stored token
	token, err := m.keychain.LoadToken(m.serverURL)
	if err == nil && token != "" {
		log.Debug("Found stored token, validating...")
		// Validate the token
		if err := m.sessionSvc.ValidateToken(ctx, token); err == nil {
			log.Info("Auto-login successful using stored token")
			return &AuthResult{Token: token}, nil
		}
		log.Debug("Stored token is invalid, will try credentials")
	}

	// Token not found or invalid, try stored credentials
	username, password, err := m.keychain.LoadCredentials(m.serverURL)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			log.Debug("No stored credentials found")
			return nil, fmt.Errorf("no stored credentials: %w", ErrNotFound)
		}
		if errors.Is(err, ErrKeychainUnavailable) {
			log.Debug("Keychain unavailable")
			return nil, fmt.Errorf("keychain unavailable: %w", ErrKeychainUnavailable)
		}
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	log.Debug("Found stored credentials, attempting login...")

	// Try to login with stored credentials
	newToken, err := m.sessionSvc.Login(ctx, username, password)
	if err != nil {
		log.Debug("Auto-login with stored credentials failed", "err", err)
		return nil, fmt.Errorf("login with stored credentials failed: %w", err)
	}

	// Store the new token
	if storeErr := m.keychain.StoreToken(m.serverURL, newToken); storeErr != nil {
		log.Warn("Failed to store new token in keychain", "err", storeErr)
		// Not fatal, continue
	}

	log.Info("Auto-login successful using stored credentials")
	return &AuthResult{Token: newToken, Username: username}, nil
}

// LoginWithCredentials authenticates with the provided credentials
// If saveCredentials is true, stores them in the keychain for future auto-login
func (m *AuthManager) LoginWithCredentials(ctx context.Context, username, password string, saveCredentials bool) (*AuthResult, error) {
	log := cblog.With("component", "auth-manager")

	log.Debug("Attempting login with provided credentials", "username", username)

	token, err := m.sessionSvc.Login(ctx, username, password)
	if err != nil {
		return nil, fmt.Errorf("login failed: %w", err)
	}

	log.Info("Login successful", "username", username)

	// Store the token in keychain
	if storeErr := m.keychain.StoreToken(m.serverURL, token); storeErr != nil {
		log.Warn("Failed to store token in keychain", "err", storeErr)
		// Not fatal, continue
	}

	// Optionally store credentials for future auto-login
	if saveCredentials {
		if storeErr := m.keychain.StoreCredentials(m.serverURL, username, password); storeErr != nil {
			log.Warn("Failed to store credentials in keychain", "err", storeErr)
			// Not fatal, continue
		} else {
			log.Debug("Stored credentials in keychain for future auto-login")
		}
	}

	return &AuthResult{Token: token, Username: username}, nil
}

// GetStoredUsername retrieves the stored username for pre-filling the login form
func (m *AuthManager) GetStoredUsername() string {
	username, _, err := m.keychain.LoadCredentials(m.serverURL)
	if err != nil {
		return ""
	}
	return username
}

// ClearStoredCredentials removes all stored credentials for the server
func (m *AuthManager) ClearStoredCredentials() error {
	log := cblog.With("component", "auth-manager")

	if err := m.keychain.DeleteToken(m.serverURL); err != nil {
		log.Warn("Failed to delete token from keychain", "err", err)
	}

	if err := m.keychain.DeleteCredentials(m.serverURL); err != nil {
		log.Warn("Failed to delete credentials from keychain", "err", err)
		return err
	}

	log.Info("Cleared stored credentials from keychain")
	return nil
}

// ValidateToken checks if a token is still valid
func (m *AuthManager) ValidateToken(ctx context.Context, token string) error {
	return m.sessionSvc.ValidateToken(ctx, token)
}
