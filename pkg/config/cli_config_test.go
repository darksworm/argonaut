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
	origHome := os.Getenv("HOME")
	defer func() {
		os.Setenv("ARGOCD_CONFIG", origArgoConfig)
		os.Setenv("ARGOCD_CONFIG_DIR", origArgoConfigDir)
		os.Setenv("XDG_CONFIG_HOME", origXDGConfigHome)
		os.Setenv("HOME", origHome)
	}()

	tests := []struct {
		name                 string
		envARGOCD_CONFIG     string
		envARGOCD_CONFIG_DIR string
		envXDG_CONFIG_HOME   string
		createLegacyPath     bool
		expectedPath         func(tmpDir string) string
	}{
		{
			name:             "ARGOCD_CONFIG takes precedence",
			envARGOCD_CONFIG: "/custom/path/to/config",
			expectedPath: func(tmpDir string) string {
				return "/custom/path/to/config"
			},
		},
		{
			name:                 "ARGOCD_CONFIG_DIR is used",
			envARGOCD_CONFIG_DIR: "/custom/config/dir",
			expectedPath: func(tmpDir string) string {
				return "/custom/config/dir/config"
			},
		},
		{
			name:             "Legacy path ~/.argocd/config if exists",
			createLegacyPath: true,
			expectedPath: func(tmpDir string) string {
				return filepath.Join(tmpDir, ".argocd", "config")
			},
		},
		{
			name:               "XDG_CONFIG_HOME is used",
			envXDG_CONFIG_HOME: "custom-xdg", // Will be made absolute in test
			expectedPath: func(tmpDir string) string {
				return filepath.Join(tmpDir, "custom-xdg", "argocd", "config")
			},
		},
		{
			name: "Default to ~/.config/argocd/config",
			expectedPath: func(tmpDir string) string {
				return filepath.Join(tmpDir, ".config", "argocd", "config")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a separate temp directory for each test
			tmpDir := t.TempDir()

			// Set HOME to our test-specific temp directory
			os.Setenv("HOME", tmpDir)

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
				// Make XDG_CONFIG_HOME absolute if it's relative
				if !filepath.IsAbs(tt.envXDG_CONFIG_HOME) {
					os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, tt.envXDG_CONFIG_HOME))
				} else {
					os.Setenv("XDG_CONFIG_HOME", tt.envXDG_CONFIG_HOME)
				}
			}

			// Create legacy path in temp directory if needed for testing
			if tt.createLegacyPath {
				legacyDir := filepath.Join(tmpDir, ".argocd")
				os.MkdirAll(legacyDir, 0755)
				legacyFile := filepath.Join(legacyDir, "config")
				os.WriteFile(legacyFile, []byte("test"), 0644)
			}

			expected := tt.expectedPath(tmpDir)
			result := GetConfigPath()
			if result != expected {
				t.Errorf("GetConfigPath() = %v, want %v", result, expected)
			}
		})
	}
}