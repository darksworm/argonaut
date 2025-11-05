package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetArgonautConfigPath(t *testing.T) {
	// Save original env vars
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	originalXDG := os.Getenv("XDG_CONFIG_HOME")
	originalAppData := os.Getenv("APPDATA")

	defer func() {
		// Restore original env vars
		os.Setenv("ARGONAUT_CONFIG", originalConfig)
		os.Setenv("XDG_CONFIG_HOME", originalXDG)
		os.Setenv("APPDATA", originalAppData)
	}()

	tests := []struct {
		name     string
		envVars  map[string]string
		expected string
	}{
		{
			name: "ARGONAUT_CONFIG override",
			envVars: map[string]string{
				"ARGONAUT_CONFIG": "/custom/path/config.toml",
			},
			expected: "/custom/path/config.toml",
		},
		{
			name: "XDG_CONFIG_HOME on Unix",
			envVars: map[string]string{
				"ARGONAUT_CONFIG":  "",
				"XDG_CONFIG_HOME": "/home/user/.config",
			},
			expected: func() string {
				if runtime.GOOS == "windows" {
					home, _ := os.UserHomeDir()
					return filepath.Join(home, "AppData", "Roaming", "argonaut", "config.toml")
				}
				return "/home/user/.config/argonaut/config.toml"
			}(),
		},
		{
			name: "Default Unix path",
			envVars: map[string]string{
				"ARGONAUT_CONFIG":  "",
				"XDG_CONFIG_HOME": "",
			},
			expected: func() string {
				if runtime.GOOS == "windows" {
					home, _ := os.UserHomeDir()
					return filepath.Join(home, "AppData", "Roaming", "argonaut", "config.toml")
				}
				home, _ := os.UserHomeDir()
				return filepath.Join(home, ".config", "argonaut", "config.toml")
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for key, value := range tt.envVars {
				if value == "" {
					os.Unsetenv(key)
				} else {
					os.Setenv(key, value)
				}
			}

			result := GetArgonautConfigPath()
			if result != tt.expected {
				t.Errorf("GetArgonautConfigPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	config := GetDefaultConfig()

	if config == nil {
		t.Fatal("GetDefaultConfig() returned nil")
	}

	if config.Appearance.Theme != DefaultThemeName {
		t.Errorf("Expected default theme %q, got %q", DefaultThemeName, config.Appearance.Theme)
	}
}

func TestLoadArgonautConfig_NoFile(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Override config path to non-existent location
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	defer os.Setenv("ARGONAUT_CONFIG", originalConfig)

	configPath := filepath.Join(tempDir, "nonexistent", "config.toml")
	os.Setenv("ARGONAUT_CONFIG", configPath)

	config, err := LoadArgonautConfig()
	if err != nil {
		t.Errorf("LoadArgonautConfig() should not error when config file doesn't exist, got: %v", err)
	}

	if config == nil {
		t.Fatal("LoadArgonautConfig() returned nil config")
	}

	if config.Appearance.Theme != DefaultThemeName {
		t.Errorf("Expected default theme %q when no config file exists, got %q", DefaultThemeName, config.Appearance.Theme)
	}
}

func TestSaveAndLoadArgonautConfig(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Override config path
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	defer os.Setenv("ARGONAUT_CONFIG", originalConfig)

	configPath := filepath.Join(tempDir, "test_config.toml")
	os.Setenv("ARGONAUT_CONFIG", configPath)

	// Create test config
	testConfig := &ArgonautConfig{
		Appearance: AppearanceConfig{
			Theme: "dracula",
			Overrides: map[string]string{
				"accent": "#ff0000",
			},
		},
		Custom: CustomTheme{
			Accent:  "#bd93f9",
			Warning: "#f1fa8c",
		},
	}

	// Save config
	err := SaveArgonautConfig(testConfig)
	if err != nil {
		t.Fatalf("SaveArgonautConfig() failed: %v", err)
	}

	// Check file was created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loadedConfig, err := LoadArgonautConfig()
	if err != nil {
		t.Fatalf("LoadArgonautConfig() failed: %v", err)
	}

	// Verify loaded config matches saved config
	if loadedConfig.Appearance.Theme != testConfig.Appearance.Theme {
		t.Errorf("Theme mismatch: expected %q, got %q", testConfig.Appearance.Theme, loadedConfig.Appearance.Theme)
	}

	if loadedConfig.Appearance.Overrides["accent"] != testConfig.Appearance.Overrides["accent"] {
		t.Errorf("Override mismatch: expected %q, got %q",
			testConfig.Appearance.Overrides["accent"],
			loadedConfig.Appearance.Overrides["accent"])
	}

	if loadedConfig.Custom.Accent != testConfig.Custom.Accent {
		t.Errorf("Custom theme accent mismatch: expected %q, got %q",
			testConfig.Custom.Accent, loadedConfig.Custom.Accent)
	}
}

func TestLoadArgonautConfig_InvalidTOML(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Override config path
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	defer os.Setenv("ARGONAUT_CONFIG", originalConfig)

	configPath := filepath.Join(tempDir, "invalid_config.toml")
	os.Setenv("ARGONAUT_CONFIG", configPath)

	// Write invalid TOML
	err := os.WriteFile(configPath, []byte("invalid toml content [[["), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Load config should fail with invalid TOML
	_, err = LoadArgonautConfig()
	if err == nil {
		t.Error("LoadArgonautConfig() should fail with invalid TOML")
	}
}

func TestEnsureArgonautConfigDir(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Override config path
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	defer os.Setenv("ARGONAUT_CONFIG", originalConfig)

	configPath := filepath.Join(tempDir, "nested", "dirs", "config.toml")
	os.Setenv("ARGONAUT_CONFIG", configPath)

	// Ensure directory doesn't exist yet
	configDir := filepath.Dir(configPath)
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Fatal("Config directory should not exist initially")
	}

	// Create directory
	err := EnsureArgonautConfigDir()
	if err != nil {
		t.Fatalf("EnsureArgonautConfigDir() failed: %v", err)
	}

	// Check directory was created
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		t.Fatal("Config directory was not created")
	}

	// Second call should not error
	err = EnsureArgonautConfigDir()
	if err != nil {
		t.Errorf("EnsureArgonautConfigDir() should not error when directory exists: %v", err)
	}
}