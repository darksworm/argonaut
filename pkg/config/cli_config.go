package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
	"github.com/darksworm/argonaut/pkg/model"
)

// ArgoContext represents an ArgoCD context configuration
type ArgoContext struct {
	Name   string `yaml:"name"`
	Server string `yaml:"server"`
	User   string `yaml:"user"`
}

// ArgoServer represents an ArgoCD server configuration
type ArgoServer struct {
	Server           string `yaml:"server"`
	GrpcWeb          bool   `yaml:"grpc-web,omitempty"`
	GrpcWebRootPath  string `yaml:"grpc-web-root-path,omitempty"`
	Insecure         bool   `yaml:"insecure,omitempty"`
	PlainText        bool   `yaml:"plain-text,omitempty"`
}

// ArgoUser represents an ArgoCD user configuration
type ArgoUser struct {
	Name      string `yaml:"name"`
	AuthToken string `yaml:"auth-token,omitempty"`
}

// ArgoCLIConfig represents the complete ArgoCD CLI configuration
type ArgoCLIConfig struct {
	Contexts       []ArgoContext `yaml:"contexts,omitempty"`
	Servers        []ArgoServer  `yaml:"servers,omitempty"`
	Users          []ArgoUser    `yaml:"users,omitempty"`
	CurrentContext string        `yaml:"current-context,omitempty"`
	PromptsEnabled bool          `yaml:"prompts-enabled,omitempty"`
}

// GetConfigPath returns the path to the ArgoCD CLI configuration file
func GetConfigPath() string {
	if configPath := os.Getenv("ARGOCD_CONFIG"); configPath != "" {
		return configPath
	}
	
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	
	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "argocd", "config")
	}
	
	// Default to ~/.config/argocd/config
	return filepath.Join(homeDir, ".config", "argocd", "config")
}

// ReadCLIConfig reads and parses the ArgoCD CLI configuration
func ReadCLIConfig() (*ArgoCLIConfig, error) {
	configPath := GetConfigPath()
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read ArgoCD config from %s: %w", configPath, err)
	}
	
	var config ArgoCLIConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse ArgoCD config: %w", err)
	}
	
	return &config, nil
}

// GetCurrentServer returns the server URL for the current context
func (c *ArgoCLIConfig) GetCurrentServer() (string, error) {
	if c.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in ArgoCD config")
	}
	
	for _, ctx := range c.Contexts {
		if ctx.Name == c.CurrentContext {
			if ctx.Server == "" {
				return "", fmt.Errorf("no server specified for context %s", c.CurrentContext)
			}
			return ctx.Server, nil
		}
	}
	
	return "", fmt.Errorf("context %s not found in ArgoCD config", c.CurrentContext)
}

// GetCurrentServerConfig returns the server configuration for the current context
func (c *ArgoCLIConfig) GetCurrentServerConfig() (*ArgoServer, error) {
	serverURL, err := c.GetCurrentServer()
	if err != nil {
		return nil, err
	}
	
	for _, server := range c.Servers {
		if server.Server == serverURL {
			return &server, nil
		}
	}
	
	return nil, fmt.Errorf("server configuration not found for %s", serverURL)
}

// GetCurrentToken returns the auth token for the current context
func (c *ArgoCLIConfig) GetCurrentToken() (string, error) {
	if c.CurrentContext == "" {
		return "", fmt.Errorf("no current context set in ArgoCD config")
	}
	
	// Find the current context
	var currentUser string
	for _, ctx := range c.Contexts {
		if ctx.Name == c.CurrentContext {
			currentUser = ctx.User
			break
		}
	}
	
	if currentUser == "" {
		return "", fmt.Errorf("no user specified for context %s", c.CurrentContext)
	}
	
	// Find the user and their token
	for _, user := range c.Users {
		if user.Name == currentUser {
			if user.AuthToken == "" {
				return "", fmt.Errorf("no auth token found for user %s. Please run 'argocd login' to authenticate", currentUser)
			}
			return user.AuthToken, nil
		}
	}
	
	return "", fmt.Errorf("user %s not found in ArgoCD config", currentUser)
}

// ToServerConfig converts the ArgoCD CLI config to our internal Server model
func (c *ArgoCLIConfig) ToServerConfig() (*model.Server, error) {
	serverConfig, err := c.GetCurrentServerConfig()
	if err != nil {
		return nil, err
	}
	
	token, err := c.GetCurrentToken()
	if err != nil {
		return nil, err
	}
	
	baseURL := ensureHTTPS(serverConfig.Server, serverConfig.PlainText)
	
	return &model.Server{
		BaseURL:  baseURL,
		Token:    token,
		Insecure: serverConfig.Insecure,
	}, nil
}

// ensureHTTPS ensures the URL has the correct protocol
func ensureHTTPS(baseURL string, plainText bool) string {
	if len(baseURL) == 0 {
		return baseURL
	}
	
	// If already has protocol, return as-is
	if len(baseURL) >= 7 && (baseURL[:7] == "http://" || baseURL[:8] == "https://") {
		return baseURL
	}
	
	// Add appropriate protocol
	if plainText {
		return "http://" + baseURL
	}
	return "https://" + baseURL
}