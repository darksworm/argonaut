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

func TestIsCurrentServerCore(t *testing.T) {
	tests := []struct {
		name     string
		config   *ArgoCLIConfig
		expected bool
		hasError bool
	}{
		{
			name: "Server with core true",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "kubernetes", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "kubernetes", Core: true},
				},
			},
			expected: true,
			hasError: false,
		},
		{
			name: "Server with core false",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "https://argocd.example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "https://argocd.example.com", Core: false},
				},
			},
			expected: false,
			hasError: false,
		},
		{
			name: "Server without core field (defaults to false)",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "https://argocd.example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "https://argocd.example.com"},
				},
			},
			expected: false,
			hasError: false,
		},
		{
			name: "No current context",
			config: &ArgoCLIConfig{
				Servers: []ArgoServer{
					{Server: "kubernetes", Core: true},
				},
			},
			expected: false,
			hasError: true,
		},
		{
			name: "Context not found in servers",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "missing-server", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "kubernetes", Core: true},
				},
			},
			expected: false,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.IsCurrentServerCore()

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("IsCurrentServerCore() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToServerConfig_GrpcWebRootPath(t *testing.T) {
	tests := []struct {
		name             string
		config           *ArgoCLIConfig
		expectedRootPath string
		hasError         bool
	}{
		{
			name: "Config without grpc-web-root-path",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "example.com"},
				},
				Users: []ArgoUser{
					{Name: "admin", AuthToken: "test-token"},
				},
			},
			expectedRootPath: "",
			hasError:         false,
		},
		{
			name: "Config with grpc-web-root-path",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "example.com", GrpcWebRootPath: "argocd"},
				},
				Users: []ArgoUser{
					{Name: "admin", AuthToken: "test-token"},
				},
			},
			expectedRootPath: "argocd",
			hasError:         false,
		},
		{
			name: "Config with grpc-web-root-path with slashes",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "example.com", GrpcWebRootPath: "/argocd/"},
				},
				Users: []ArgoUser{
					{Name: "admin", AuthToken: "test-token"},
				},
			},
			expectedRootPath: "/argocd/",
			hasError:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverConfig, err := tt.config.ToServerConfig()

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if serverConfig.GrpcWebRootPath != tt.expectedRootPath {
				t.Errorf("GrpcWebRootPath = %v, want %v", serverConfig.GrpcWebRootPath, tt.expectedRootPath)
			}
		})
	}
}

func TestIsPortForwardMode(t *testing.T) {
	tests := []struct {
		name     string
		config   *ArgoCLIConfig
		expected bool
		hasError bool
	}{
		{
			name: "Server is port-forward",
			config: &ArgoCLIConfig{
				CurrentContext: "port-forward",
				Contexts: []ArgoContext{
					{Name: "port-forward", Server: "port-forward", User: "port-forward"},
				},
				Servers: []ArgoServer{
					{Server: "port-forward", PlainText: true},
				},
			},
			expected: true,
			hasError: false,
		},
		{
			name: "Server is regular URL",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "https://argocd.example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "https://argocd.example.com"},
				},
			},
			expected: false,
			hasError: false,
		},
		{
			name: "Server contains port-forward but is not exact match",
			config: &ArgoCLIConfig{
				CurrentContext: "test-context",
				Contexts: []ArgoContext{
					{Name: "test-context", Server: "https://port-forward.example.com", User: "admin"},
				},
				Servers: []ArgoServer{
					{Server: "https://port-forward.example.com"},
				},
			},
			expected: false,
			hasError: false,
		},
		{
			name: "No current context",
			config: &ArgoCLIConfig{
				Servers: []ArgoServer{
					{Server: "port-forward"},
				},
			},
			expected: false,
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.IsPortForwardMode()

			if tt.hasError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result != tt.expected {
				t.Errorf("IsPortForwardMode() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetContextNames(t *testing.T) {
	tests := []struct {
		name     string
		config   *ArgoCLIConfig
		expected []string
	}{
		{
			name:     "Empty config",
			config:   &ArgoCLIConfig{},
			expected: []string{},
		},
		{
			name: "Single context",
			config: &ArgoCLIConfig{
				Contexts: []ArgoContext{
					{Name: "production", Server: "https://prod.example.com", User: "admin"},
				},
			},
			expected: []string{"production"},
		},
		{
			name: "Multiple contexts sorted",
			config: &ArgoCLIConfig{
				Contexts: []ArgoContext{
					{Name: "staging", Server: "https://staging.example.com", User: "admin"},
					{Name: "production", Server: "https://prod.example.com", User: "admin"},
					{Name: "dev", Server: "https://dev.example.com", User: "admin"},
				},
			},
			expected: []string{"dev", "production", "staging"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetContextNames()
			if len(result) != len(tt.expected) {
				t.Fatalf("GetContextNames() returned %d items, want %d", len(result), len(tt.expected))
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("GetContextNames()[%d] = %q, want %q", i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestToServerConfigForContext(t *testing.T) {
	multiContextConfig := &ArgoCLIConfig{
		CurrentContext: "production",
		Contexts: []ArgoContext{
			{Name: "production", Server: "prod.example.com", User: "prod-admin"},
			{Name: "staging", Server: "staging.example.com", User: "staging-admin"},
		},
		Servers: []ArgoServer{
			{Server: "prod.example.com", Insecure: false},
			{Server: "staging.example.com", Insecure: true, GrpcWebRootPath: "argocd"},
		},
		Users: []ArgoUser{
			{Name: "prod-admin", AuthToken: "prod-token"},
			{Name: "staging-admin", AuthToken: "staging-token"},
		},
	}

	tests := []struct {
		name        string
		config      *ArgoCLIConfig
		contextName string
		wantBaseURL string
		wantToken   string
		wantErr     bool
	}{
		{
			name:        "Resolve production context",
			config:      multiContextConfig,
			contextName: "production",
			wantBaseURL: "https://prod.example.com",
			wantToken:   "prod-token",
		},
		{
			name:        "Resolve staging context",
			config:      multiContextConfig,
			contextName: "staging",
			wantBaseURL: "https://staging.example.com",
			wantToken:   "staging-token",
		},
		{
			name:        "Unknown context",
			config:      multiContextConfig,
			contextName: "nonexistent",
			wantErr:     true,
		},
		{
			name: "Context with missing user token",
			config: &ArgoCLIConfig{
				Contexts: []ArgoContext{
					{Name: "notoken", Server: "example.com", User: "nouser"},
				},
				Servers: []ArgoServer{
					{Server: "example.com"},
				},
				Users: []ArgoUser{
					{Name: "nouser", AuthToken: ""},
				},
			},
			contextName: "notoken",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := tt.config.ToServerConfigForContext(tt.contextName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if server.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", server.BaseURL, tt.wantBaseURL)
			}
			if server.Token != tt.wantToken {
				t.Errorf("Token = %q, want %q", server.Token, tt.wantToken)
			}
		})
	}
}

func TestIsContextPortForward(t *testing.T) {
	config := &ArgoCLIConfig{
		Contexts: []ArgoContext{
			{Name: "pf-context", Server: "port-forward", User: "admin"},
			{Name: "normal-context", Server: "https://argocd.example.com", User: "admin"},
		},
	}

	tests := []struct {
		name        string
		contextName string
		expected    bool
		wantErr     bool
	}{
		{"Port-forward context", "pf-context", true, false},
		{"Normal context", "normal-context", false, false},
		{"Unknown context", "nonexistent", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.IsContextPortForward(tt.contextName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("IsContextPortForward(%q) = %v, want %v", tt.contextName, result, tt.expected)
			}
		})
	}
}

func TestIsContextCore(t *testing.T) {
	config := &ArgoCLIConfig{
		Contexts: []ArgoContext{
			{Name: "core-context", Server: "kubernetes", User: "admin"},
			{Name: "normal-context", Server: "https://argocd.example.com", User: "admin"},
		},
		Servers: []ArgoServer{
			{Server: "kubernetes", Core: true},
			{Server: "https://argocd.example.com", Core: false},
		},
	}

	tests := []struct {
		name        string
		contextName string
		expected    bool
		wantErr     bool
	}{
		{"Core context", "core-context", true, false},
		{"Normal context", "normal-context", false, false},
		{"Unknown context", "nonexistent", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := config.IsContextCore(tt.contextName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("IsContextCore(%q) = %v, want %v", tt.contextName, result, tt.expected)
			}
		})
	}
}