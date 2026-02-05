package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "argonaut"
	tokenKey    = "token"
	usernameKey = "username"
	passwordKey = "password"
)

// ErrKeychainUnavailable indicates that the system keychain is not available
var ErrKeychainUnavailable = errors.New("keychain is not available on this system")

// ErrNotFound indicates that the requested credential was not found
var ErrNotFound = errors.New("credential not found in keychain")

// KeychainStore provides cross-platform credential storage
type KeychainStore interface {
	StoreToken(serverURL, token string) error
	LoadToken(serverURL string) (string, error)
	DeleteToken(serverURL string) error
	StoreCredentials(serverURL, username, password string) error
	LoadCredentials(serverURL string) (username, password string, err error)
	DeleteCredentials(serverURL string) error
}

// DefaultKeychainStore uses the system keychain via go-keyring
type DefaultKeychainStore struct{}

// NewKeychainStore creates a new keychain store
func NewKeychainStore() KeychainStore {
	return &DefaultKeychainStore{}
}

// serverKey creates a unique key for a server URL using a truncated hash
func serverKey(serverURL string) string {
	hash := sha256.Sum256([]byte(serverURL))
	return hex.EncodeToString(hash[:8]) // First 16 chars of hex = 8 bytes
}

// StoreToken stores an auth token for a server
func (k *DefaultKeychainStore) StoreToken(serverURL, token string) error {
	key := fmt.Sprintf("%s:%s:%s", serviceName, serverKey(serverURL), tokenKey)
	err := keyring.Set(serviceName, key, token)
	if err != nil {
		if isKeychainUnavailable(err) {
			return ErrKeychainUnavailable
		}
		return fmt.Errorf("failed to store token in keychain: %w", err)
	}
	return nil
}

// LoadToken retrieves an auth token for a server
func (k *DefaultKeychainStore) LoadToken(serverURL string) (string, error) {
	key := fmt.Sprintf("%s:%s:%s", serviceName, serverKey(serverURL), tokenKey)
	token, err := keyring.Get(serviceName, key)
	if err != nil {
		if isNotFound(err) {
			return "", ErrNotFound
		}
		if isKeychainUnavailable(err) {
			return "", ErrKeychainUnavailable
		}
		return "", fmt.Errorf("failed to load token from keychain: %w", err)
	}
	return token, nil
}

// DeleteToken removes an auth token for a server
func (k *DefaultKeychainStore) DeleteToken(serverURL string) error {
	key := fmt.Sprintf("%s:%s:%s", serviceName, serverKey(serverURL), tokenKey)
	err := keyring.Delete(serviceName, key)
	if err != nil {
		if isNotFound(err) {
			return nil // Already deleted, not an error
		}
		if isKeychainUnavailable(err) {
			return ErrKeychainUnavailable
		}
		return fmt.Errorf("failed to delete token from keychain: %w", err)
	}
	return nil
}

// StoreCredentials stores username and password for a server
func (k *DefaultKeychainStore) StoreCredentials(serverURL, username, password string) error {
	urlKey := serverKey(serverURL)

	// Store username
	usernameFullKey := fmt.Sprintf("%s:%s:%s", serviceName, urlKey, usernameKey)
	if err := keyring.Set(serviceName, usernameFullKey, username); err != nil {
		if isKeychainUnavailable(err) {
			return ErrKeychainUnavailable
		}
		return fmt.Errorf("failed to store username in keychain: %w", err)
	}

	// Store password
	passwordFullKey := fmt.Sprintf("%s:%s:%s", serviceName, urlKey, passwordKey)
	if err := keyring.Set(serviceName, passwordFullKey, password); err != nil {
		if isKeychainUnavailable(err) {
			return ErrKeychainUnavailable
		}
		return fmt.Errorf("failed to store password in keychain: %w", err)
	}

	return nil
}

// LoadCredentials retrieves username and password for a server
func (k *DefaultKeychainStore) LoadCredentials(serverURL string) (string, string, error) {
	urlKey := serverKey(serverURL)

	// Load username
	usernameFullKey := fmt.Sprintf("%s:%s:%s", serviceName, urlKey, usernameKey)
	username, err := keyring.Get(serviceName, usernameFullKey)
	if err != nil {
		if isNotFound(err) {
			return "", "", ErrNotFound
		}
		if isKeychainUnavailable(err) {
			return "", "", ErrKeychainUnavailable
		}
		return "", "", fmt.Errorf("failed to load username from keychain: %w", err)
	}

	// Load password
	passwordFullKey := fmt.Sprintf("%s:%s:%s", serviceName, urlKey, passwordKey)
	password, err := keyring.Get(serviceName, passwordFullKey)
	if err != nil {
		if isNotFound(err) {
			return "", "", ErrNotFound
		}
		if isKeychainUnavailable(err) {
			return "", "", ErrKeychainUnavailable
		}
		return "", "", fmt.Errorf("failed to load password from keychain: %w", err)
	}

	return username, password, nil
}

// DeleteCredentials removes username and password for a server
func (k *DefaultKeychainStore) DeleteCredentials(serverURL string) error {
	urlKey := serverKey(serverURL)

	// Delete username
	usernameFullKey := fmt.Sprintf("%s:%s:%s", serviceName, urlKey, usernameKey)
	if err := keyring.Delete(serviceName, usernameFullKey); err != nil && !isNotFound(err) {
		if isKeychainUnavailable(err) {
			return ErrKeychainUnavailable
		}
		return fmt.Errorf("failed to delete username from keychain: %w", err)
	}

	// Delete password
	passwordFullKey := fmt.Sprintf("%s:%s:%s", serviceName, urlKey, passwordKey)
	if err := keyring.Delete(serviceName, passwordFullKey); err != nil && !isNotFound(err) {
		if isKeychainUnavailable(err) {
			return ErrKeychainUnavailable
		}
		return fmt.Errorf("failed to delete password from keychain: %w", err)
	}

	return nil
}

// isNotFound checks if the error indicates the credential was not found
func isNotFound(err error) bool {
	return errors.Is(err, keyring.ErrNotFound)
}

// isKeychainUnavailable checks if the error indicates the keychain is unavailable
func isKeychainUnavailable(err error) bool {
	// go-keyring doesn't have a specific error for unavailable,
	// but certain errors indicate the keychain isn't accessible
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Check for common indicators of unavailable keychain
	return errors.Is(err, keyring.ErrUnsupportedPlatform) ||
		containsAny(errStr, "secret service", "dbus", "keychain", "credential manager")
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
