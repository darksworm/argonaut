package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pelletier/go-toml/v2"
)

// Default theme constant - easy to change
const DefaultThemeName = "oxocarbon"

// ArgonautConfig represents the complete Argonaut configuration
type ArgonautConfig struct {
	Appearance AppearanceConfig `toml:"appearance"`
}

// AppearanceConfig holds theme and visual settings
type AppearanceConfig struct {
	Theme     string            `toml:"theme"`
	Overrides map[string]string `toml:"overrides,omitempty"`
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

// GetDefaultConfig returns a config with sensible defaults
func GetDefaultConfig() *ArgonautConfig {
	return &ArgonautConfig{
		Appearance: AppearanceConfig{
			Theme: DefaultThemeName,
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