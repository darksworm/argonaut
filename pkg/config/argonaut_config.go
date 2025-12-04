package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pelletier/go-toml/v2"
)

// Default theme constant - easy to change
const DefaultThemeName = "tokyo-night"

// ArgonautConfig represents the complete Argonaut configuration
type ArgonautConfig struct {
	Appearance      AppearanceConfig  `toml:"appearance"`
	Sort            SortConfig        `toml:"sort,omitempty"`
	K9s             K9sConfig         `toml:"k9s,omitempty"`
	Diff            DiffConfig        `toml:"diff,omitempty"`
	PortForward     PortForwardConfig `toml:"port_forward,omitempty"`
	LastSeenVersion string            `toml:"last_seen_version,omitempty"`
}

// AppearanceConfig holds theme and visual settings
type AppearanceConfig struct {
	Theme     string            `toml:"theme"`
	Overrides map[string]string `toml:"overrides,omitempty"`
}

// SortConfig holds sort preferences
type SortConfig struct {
	Field     string `toml:"field"`
	Direction string `toml:"direction"`
}

// K9sConfig holds k9s integration settings
type K9sConfig struct {
	Command string `toml:"command,omitempty"` // Path to k9s executable (default: "k9s")
	Context string `toml:"context,omitempty"` // Override Kubernetes context for k9s
}

// DiffConfig holds diff viewer/formatter settings
type DiffConfig struct {
	Viewer    string `toml:"viewer,omitempty"`    // External diff viewer command (e.g., "code --diff {left} {right}")
	Formatter string `toml:"formatter,omitempty"` // Diff formatter command (e.g., "delta")
}

// PortForwardConfig holds settings for kubectl port-forward mode
type PortForwardConfig struct {
	Namespace string `toml:"namespace,omitempty"` // Kubernetes namespace where ArgoCD is installed (default: "argocd")
}


// GetArgonautConfigPath returns the path to the Argonaut configuration file
func GetArgonautConfigPath() string {
	if configPath := os.Getenv("ARGONAUT_CONFIG"); configPath != "" {
		return configPath
	}

	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			// Fallback for Windows
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "argonaut", "config.toml")
	default:
		// Unix-like systems (Linux, macOS, BSD)
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, "argonaut", "config.toml")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "argonaut", "config.toml")
	}
}

// EnsureArgonautConfigDir creates the config directory if it doesn't exist
func EnsureArgonautConfigDir() error {
	configPath := GetArgonautConfigPath()
	configDir := filepath.Dir(configPath)

	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return os.MkdirAll(configDir, 0755)
	}
	return nil
}

// ConfigFileExists returns true if the config file exists on disk
func ConfigFileExists() bool {
	configPath := GetArgonautConfigPath()
	_, err := os.Stat(configPath)
	return err == nil
}

// GetDefaultConfig returns a config with sensible defaults
func GetDefaultConfig() *ArgonautConfig {
	return &ArgonautConfig{
		Appearance: AppearanceConfig{
			Theme: DefaultThemeName,
		},
		Sort: SortConfig{
			Field:     "name",
			Direction: "asc",
		},
	}
}

// LoadArgonautConfig loads the Argonaut configuration with fallback to defaults
func LoadArgonautConfig() (*ArgonautConfig, error) {
	configPath := GetArgonautConfigPath()

	// If config file doesn't exist, return defaults
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return GetDefaultConfig(), nil
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from %s: %w", configPath, err)
	}

	var config ArgonautConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Apply defaults for missing fields
	if config.Appearance.Theme == "" {
		config.Appearance.Theme = DefaultThemeName
	}
	if config.Sort.Field == "" {
		config.Sort.Field = "name"
	}
	if config.Sort.Direction == "" {
		config.Sort.Direction = "asc"
	}

	return &config, nil
}

// SaveArgonautConfig saves the configuration to the config file
func SaveArgonautConfig(config *ArgonautConfig) error {
	if err := EnsureArgonautConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := GetArgonautConfigPath()

	// Marshal to TOML with nice formatting
	data, err := toml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config to %s: %w", configPath, err)
	}

	return nil
}

// GetConfigPathForHelp returns the config path for display in help text
func GetConfigPathForHelp() string {
	return GetArgonautConfigPath()
}

// GetK9sCommand returns the k9s command path, defaulting to "k9s" if not configured.
// Priority: ARGONAUT_K9S_COMMAND env var > config file > default "k9s"
func (c *ArgonautConfig) GetK9sCommand() string {
	if envCmd := os.Getenv("ARGONAUT_K9S_COMMAND"); envCmd != "" {
		return envCmd
	}
	if c.K9s.Command != "" {
		return c.K9s.Command
	}
	return "k9s"
}

// GetK9sContext returns the k9s context override, or empty string if not configured
func (c *ArgonautConfig) GetK9sContext() string {
	return c.K9s.Context
}

// GetDiffViewer returns the external diff viewer command, or empty string if not configured
func (c *ArgonautConfig) GetDiffViewer() string {
	return c.Diff.Viewer
}

// GetDiffFormatter returns the diff formatter command, or empty string if not configured
func (c *ArgonautConfig) GetDiffFormatter() string {
	return c.Diff.Formatter
}

// GetPortForwardNamespace returns the namespace for kubectl port-forward, defaulting to "argocd"
func (c *ArgonautConfig) GetPortForwardNamespace() string {
	if c.PortForward.Namespace != "" {
		return c.PortForward.Namespace
	}
	return "argocd"
}