package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	// Save original environment variables
	origArgoConfig := os.Getenv("ARGOCD_CONFIG")
	origArgoConfigDir := os.Getenv("ARGOCD_CONFIG_DIR")
	origXDGConfigHome := os.Getenv("XDG_CONFIG_HOME")
	defer func() {
		os.Setenv("ARGOCD_CONFIG", origArgoConfig)
		os.Setenv("ARGOCD_CONFIG_DIR", origArgoConfigDir)
		os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)
	}()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name              string
		envARGOCD_CONFIG  string
		envARGOCD_CONFIG_DIR string
		envXDG_CONFIG_HOME string
		createLegacyPath  bool
		expected          string
	}{
		{
			name:             "ARGOCD_CONFIG takes precedence",
			envARGOCD_CONFIG: "/custom/path/to/config",
			expected:         "/custom/path/to/config",
		},
		{
			name:                 "ARGOCD_CONFIG_DIR is used",
			envARGOCD_CONFIG_DIR: "/custom/config/dir",
			expected:             "/custom/config/dir/config",
		},
		{
			name:             "Legacy path ~/.argocd/config if exists",
			createLegacyPath: true,
			expected:         filepath.Join(homeDir, ".argocd", "config"),
		},
		{
			name:               "XDG_CONFIG_HOME is used",
			envXDG_CONFIG_HOME: "/custom/xdg",
			expected:           "/custom/xdg/argocd/config",
		},
		{
			name:     "Default to ~/.config/argocd/config",
			expected: filepath.Join(homeDir, ".config", "argocd", "config"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment variables
			os.Unsetenv("ARGOCD_CONFIG")
			os.Unsetenv("ARGOCD_CONFIG_DIR")
			os.Unsetenv("XDG_CONFIG_HOME")

			// Set test environment variables
			if tt.envARGOCD_CONFIG != "" {
				os.Setenv("ARGOCD_CONFIG", tt.envARGOCD_CONFIG)
			}
			if tt.envARGOCD_CONFIG_DIR != "" {
				os.Setenv("ARGOCD_CONFIG_DIR", tt.envARGOCD_CONFIG_DIR)
			}
			if tt.envXDG_CONFIG_HOME != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.envXDG_CONFIG_HOME)
			}

			// Create legacy path if needed for testing
			if tt.createLegacyPath {
				legacyDir := filepath.Join(homeDir, ".argocd")
				os.MkdirAll(legacyDir, 0755)
				legacyFile := filepath.Join(legacyDir, "config")
				os.WriteFile(legacyFile, []byte("test"), 0644)
				defer os.RemoveAll(legacyDir)
			}

			result := GetConfigPath()
			if result != tt.expected {
				t.Errorf("GetConfigPath() = %v, want %v", result, tt.expected)
			}
		})
	}
}