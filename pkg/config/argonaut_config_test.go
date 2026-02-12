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

func TestK9sConfigGetters(t *testing.T) {
	tests := []struct {
		name           string
		config         *ArgonautConfig
		expectCommand  string
		expectContext  string
	}{
		{
			name:          "empty config returns defaults",
			config:        &ArgonautConfig{},
			expectCommand: "k9s",
			expectContext: "",
		},
		{
			name: "custom k9s command",
			config: &ArgonautConfig{
				K9s: K9sConfig{
					Command: "/usr/local/bin/k9s",
				},
			},
			expectCommand: "/usr/local/bin/k9s",
			expectContext: "",
		},
		{
			name: "custom k9s context",
			config: &ArgonautConfig{
				K9s: K9sConfig{
					Context: "production-cluster",
				},
			},
			expectCommand: "k9s",
			expectContext: "production-cluster",
		},
		{
			name: "both command and context set",
			config: &ArgonautConfig{
				K9s: K9sConfig{
					Command: "/opt/k9s/bin/k9s",
					Context: "staging",
				},
			},
			expectCommand: "/opt/k9s/bin/k9s",
			expectContext: "staging",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetK9sCommand(); got != tt.expectCommand {
				t.Errorf("GetK9sCommand() = %q, want %q", got, tt.expectCommand)
			}
			if got := tt.config.GetK9sContext(); got != tt.expectContext {
				t.Errorf("GetK9sContext() = %q, want %q", got, tt.expectContext)
			}
		})
	}
}

func TestDiffConfigGetters(t *testing.T) {
	tests := []struct {
		name            string
		config          *ArgonautConfig
		expectViewer    string
		expectFormatter string
	}{
		{
			name:            "empty config returns empty strings",
			config:          &ArgonautConfig{},
			expectViewer:    "",
			expectFormatter: "",
		},
		{
			name: "custom diff viewer",
			config: &ArgonautConfig{
				Diff: DiffConfig{
					Viewer: "code --diff {left} {right}",
				},
			},
			expectViewer:    "code --diff {left} {right}",
			expectFormatter: "",
		},
		{
			name: "custom diff formatter",
			config: &ArgonautConfig{
				Diff: DiffConfig{
					Formatter: "delta --side-by-side",
				},
			},
			expectViewer:    "",
			expectFormatter: "delta --side-by-side",
		},
		{
			name: "both viewer and formatter set",
			config: &ArgonautConfig{
				Diff: DiffConfig{
					Viewer:    "vimdiff",
					Formatter: "colordiff",
				},
			},
			expectViewer:    "vimdiff",
			expectFormatter: "colordiff",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetDiffViewer(); got != tt.expectViewer {
				t.Errorf("GetDiffViewer() = %q, want %q", got, tt.expectViewer)
			}
			if got := tt.config.GetDiffFormatter(); got != tt.expectFormatter {
				t.Errorf("GetDiffFormatter() = %q, want %q", got, tt.expectFormatter)
			}
		})
	}
}

func TestSaveAndLoadK9sAndDiffConfig(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Override config path
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	defer os.Setenv("ARGONAUT_CONFIG", originalConfig)

	configPath := filepath.Join(tempDir, "test_k9s_diff_config.toml")
	os.Setenv("ARGONAUT_CONFIG", configPath)

	// Create test config with K9s and Diff sections
	testConfig := &ArgonautConfig{
		Appearance: AppearanceConfig{
			Theme: "dracula",
		},
		K9s: K9sConfig{
			Command: "/custom/k9s",
			Context: "my-cluster",
		},
		Diff: DiffConfig{
			Viewer:    "meld {left} {right}",
			Formatter: "delta --line-numbers",
		},
	}

	// Save config
	err := SaveArgonautConfig(testConfig)
	if err != nil {
		t.Fatalf("SaveArgonautConfig() failed: %v", err)
	}

	// Load config back
	loadedConfig, err := LoadArgonautConfig()
	if err != nil {
		t.Fatalf("LoadArgonautConfig() failed: %v", err)
	}

	// Verify K9s config
	if loadedConfig.GetK9sCommand() != testConfig.K9s.Command {
		t.Errorf("K9s Command mismatch: expected %q, got %q",
			testConfig.K9s.Command, loadedConfig.GetK9sCommand())
	}
	if loadedConfig.GetK9sContext() != testConfig.K9s.Context {
		t.Errorf("K9s Context mismatch: expected %q, got %q",
			testConfig.K9s.Context, loadedConfig.GetK9sContext())
	}

	// Verify Diff config
	if loadedConfig.GetDiffViewer() != testConfig.Diff.Viewer {
		t.Errorf("Diff Viewer mismatch: expected %q, got %q",
			testConfig.Diff.Viewer, loadedConfig.GetDiffViewer())
	}
	if loadedConfig.GetDiffFormatter() != testConfig.Diff.Formatter {
		t.Errorf("Diff Formatter mismatch: expected %q, got %q",
			testConfig.Diff.Formatter, loadedConfig.GetDiffFormatter())
	}
}

func TestPortForwardConfigGetters(t *testing.T) {
	tests := []struct {
		name            string
		config          *ArgonautConfig
		expectNamespace string
	}{
		{
			name:            "empty config returns default argocd",
			config:          &ArgonautConfig{},
			expectNamespace: "argocd",
		},
		{
			name: "custom namespace from config",
			config: &ArgonautConfig{
				PortForward: PortForwardConfig{
					Namespace: "custom-ns",
				},
			},
			expectNamespace: "custom-ns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.GetPortForwardNamespace(); got != tt.expectNamespace {
				t.Errorf("GetPortForwardNamespace() = %q, want %q", got, tt.expectNamespace)
			}
		})
	}
}

func TestHTTPTimeoutConfigGetters(t *testing.T) {
	tests := []struct {
		name           string
		config         *ArgonautConfig
		expectTimeout  string
		expectDuration string
	}{
		{
			name:           "empty config returns default 10s",
			config:         &ArgonautConfig{},
			expectTimeout:  "10s",
			expectDuration: "10s",
		},
		{
			name: "custom timeout 30s",
			config: &ArgonautConfig{
				HTTPTimeouts: HTTPTimeoutConfig{
					RequestTimeout: "30s",
				},
			},
			expectTimeout:  "30s",
			expectDuration: "30s",
		},
		{
			name: "custom timeout 1m",
			config: &ArgonautConfig{
				HTTPTimeouts: HTTPTimeoutConfig{
					RequestTimeout: "1m",
				},
			},
			expectTimeout:  "1m",
			expectDuration: "1m0s",
		},
		{
			name: "custom timeout 90s",
			config: &ArgonautConfig{
				HTTPTimeouts: HTTPTimeoutConfig{
					RequestTimeout: "90s",
				},
			},
			expectTimeout:  "90s",
			expectDuration: "1m30s",
		},
		{
			name: "invalid timeout returns default",
			config: &ArgonautConfig{
				HTTPTimeouts: HTTPTimeoutConfig{
					RequestTimeout: "invalid",
				},
			},
			expectTimeout:  "invalid", // Raw value is returned
			expectDuration: "10s",     // But parsed duration is default
		},
		{
			name: "zero timeout returns default",
			config: &ArgonautConfig{
				HTTPTimeouts: HTTPTimeoutConfig{
					RequestTimeout: "0s",
				},
			},
			expectTimeout:  "0s",  // Raw value is returned
			expectDuration: "10s", // But parsed duration is default
		},
		{
			name: "negative timeout returns default",
			config: &ArgonautConfig{
				HTTPTimeouts: HTTPTimeoutConfig{
					RequestTimeout: "-5s",
				},
			},
			expectTimeout:  "-5s", // Raw value is returned
			expectDuration: "10s", // But parsed duration is default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test raw string getter
			if got := tt.config.GetRequestTimeoutString(); got != tt.expectTimeout {
				t.Errorf("GetRequestTimeoutString() = %q, want %q", got, tt.expectTimeout)
			}

			// Test parsed duration getter
			gotDuration := tt.config.GetRequestTimeout()
			if gotDuration.String() != tt.expectDuration {
				t.Errorf("GetRequestTimeout() = %q, want %q", gotDuration.String(), tt.expectDuration)
			}
		})
	}
}

func TestParseDefaultView(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantView   string
		wantScope  string
		wantValue  string
		wantErr    bool
	}{
		// Empty / zero
		{name: "empty string", input: "", wantView: "", wantScope: "", wantValue: ""},

		// All view aliases without arguments
		{name: "apps", input: "apps", wantView: "apps", wantScope: "", wantValue: ""},
		{name: "app", input: "app", wantView: "apps", wantScope: "", wantValue: ""},
		{name: "application", input: "application", wantView: "apps", wantScope: "", wantValue: ""},
		{name: "applications", input: "applications", wantView: "apps", wantScope: "", wantValue: ""},
		{name: "clusters", input: "clusters", wantView: "clusters", wantScope: "", wantValue: ""},
		{name: "cluster", input: "cluster", wantView: "clusters", wantScope: "", wantValue: ""},
		{name: "cls", input: "cls", wantView: "clusters", wantScope: "", wantValue: ""},
		{name: "namespaces", input: "namespaces", wantView: "namespaces", wantScope: "", wantValue: ""},
		{name: "namespace", input: "namespace", wantView: "namespaces", wantScope: "", wantValue: ""},
		{name: "ns", input: "ns", wantView: "namespaces", wantScope: "", wantValue: ""},
		{name: "projects", input: "projects", wantView: "projects", wantScope: "", wantValue: ""},
		{name: "project", input: "project", wantView: "projects", wantScope: "", wantValue: ""},
		{name: "proj", input: "proj", wantView: "projects", wantScope: "", wantValue: ""},
		{name: "appsets", input: "appsets", wantView: "applicationsets", wantScope: "", wantValue: ""},
		{name: "appset", input: "appset", wantView: "applicationsets", wantScope: "", wantValue: ""},
		{name: "applicationsets", input: "applicationsets", wantView: "applicationsets", wantScope: "", wantValue: ""},
		{name: "applicationset", input: "applicationset", wantView: "applicationsets", wantScope: "", wantValue: ""},
		{name: "as", input: "as", wantView: "applicationsets", wantScope: "", wantValue: ""},

		// View + scope argument drill-down
		{name: "cluster with arg", input: "cluster production", wantView: "namespaces", wantScope: "cluster", wantValue: "production"},
		{name: "cls with arg", input: "cls production", wantView: "namespaces", wantScope: "cluster", wantValue: "production"},
		{name: "ns with arg", input: "ns my-namespace", wantView: "projects", wantScope: "namespace", wantValue: "my-namespace"},
		{name: "namespace with arg", input: "namespace my-namespace", wantView: "projects", wantScope: "namespace", wantValue: "my-namespace"},
		{name: "project with arg", input: "project myproj", wantView: "apps", wantScope: "project", wantValue: "myproj"},
		{name: "proj with arg", input: "proj myproj", wantView: "apps", wantScope: "project", wantValue: "myproj"},
		{name: "appset with arg", input: "appset myset", wantView: "apps", wantScope: "appset", wantValue: "myset"},
		{name: "as with arg", input: "as myset", wantView: "apps", wantScope: "appset", wantValue: "myset"},
		{name: "apps with arg (no scope)", input: "apps myapp", wantView: "apps", wantScope: "", wantValue: ""},

		// Invalid inputs — should return error
		{name: "invalid tree", input: "tree", wantView: "", wantScope: "", wantValue: "", wantErr: true},
		{name: "invalid unknown", input: "unknown", wantView: "", wantScope: "", wantValue: "", wantErr: true},
		{name: "invalid sync", input: "sync", wantView: "", wantScope: "", wantValue: "", wantErr: true},

		// Whitespace handling
		{name: "leading/trailing whitespace", input: "  apps  ", wantView: "apps", wantScope: "", wantValue: ""},
		{name: "extra whitespace between", input: "ns   my-namespace", wantView: "projects", wantScope: "namespace", wantValue: "my-namespace"},
		{name: "only whitespace", input: "   ", wantView: "", wantScope: "", wantValue: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ArgonautConfig{DefaultView: tt.input}
			view, scopeType, scopeValue, errMsg := cfg.ParseDefaultView()
			if view != tt.wantView {
				t.Errorf("view = %q, want %q", view, tt.wantView)
			}
			if scopeType != tt.wantScope {
				t.Errorf("scopeType = %q, want %q", scopeType, tt.wantScope)
			}
			if scopeValue != tt.wantValue {
				t.Errorf("scopeValue = %q, want %q", scopeValue, tt.wantValue)
			}
			if tt.wantErr && errMsg == "" {
				t.Error("expected error message, got empty")
			}
			if !tt.wantErr && errMsg != "" {
				t.Errorf("unexpected error message: %q", errMsg)
			}
		})
	}
}

func TestSaveAndLoadHTTPTimeoutConfig(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Override config path
	originalConfig := os.Getenv("ARGONAUT_CONFIG")
	defer os.Setenv("ARGONAUT_CONFIG", originalConfig)

	configPath := filepath.Join(tempDir, "test_timeout_config.toml")
	os.Setenv("ARGONAUT_CONFIG", configPath)

	// Create test config with HTTPTimeouts
	testConfig := &ArgonautConfig{
		Appearance: AppearanceConfig{
			Theme: "dracula",
		},
		HTTPTimeouts: HTTPTimeoutConfig{
			RequestTimeout: "45s",
		},
	}

	// Save config
	err := SaveArgonautConfig(testConfig)
	if err != nil {
		t.Fatalf("SaveArgonautConfig() failed: %v", err)
	}

	// Load config back
	loadedConfig, err := LoadArgonautConfig()
	if err != nil {
		t.Fatalf("LoadArgonautConfig() failed: %v", err)
	}

	// Verify HTTPTimeouts config
	if loadedConfig.GetRequestTimeoutString() != "45s" {
		t.Errorf("RequestTimeout mismatch: expected %q, got %q",
			"45s", loadedConfig.GetRequestTimeoutString())
	}

	// Verify parsed duration
	expectedDuration := "45s"
	if loadedConfig.GetRequestTimeout().String() != expectedDuration {
		t.Errorf("Parsed timeout mismatch: expected %q, got %q",
			expectedDuration, loadedConfig.GetRequestTimeout().String())
	}
}