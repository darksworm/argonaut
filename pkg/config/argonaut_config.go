package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

// Default theme constant - easy to change
const DefaultThemeName = "tokyo-night"

// ArgonautConfig represents the complete Argonaut configuration
type ArgonautConfig struct {
	Appearance      AppearanceConfig   `toml:"appearance"`
	Sort            SortConfig         `toml:"sort,omitempty"`
	K9s             K9sConfig          `toml:"k9s,omitempty"`
	Diff            DiffConfig         `toml:"diff,omitempty"`
	PortForward     PortForwardConfig  `toml:"port_forward,omitempty"`
	Clipboard       ClipboardConfig    `toml:"clipboard,omitempty"`
	HTTPTimeouts    HTTPTimeoutConfig  `toml:"http_timeouts,omitempty"`
	DefaultView     string             `toml:"default_view,omitempty"`
	LastSeenVersion string             `toml:"last_seen_version,omitempty"`
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

// ClipboardConfig holds settings for clipboard operations
type ClipboardConfig struct {
	// CopyCommand is the command to copy text to clipboard.
	// Text is passed via stdin. Examples: "pbcopy", "xclip -selection clipboard", "wl-copy"
	CopyCommand string `toml:"copy_command,omitempty"`
	// PasteCommand is the command to paste text from clipboard.
	// Text is read from stdout. Examples: "pbpaste", "xclip -selection clipboard -o", "wl-paste"
	PasteCommand string `toml:"paste_command,omitempty"`
}

// HTTPTimeoutConfig holds HTTP request timeout settings.
// This configuration is essential for large deployments where API operations
// may take longer due to the volume of data being processed.
type HTTPTimeoutConfig struct {
	// RequestTimeout is the timeout for HTTP requests (e.g., "30s", "1m", "90s")
	// Default is "10s". Increase for large deployments with thousands of applications.
	// Zero or negative values are ignored and default timeout is used.
	RequestTimeout string `toml:"request_timeout,omitempty"`
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

// GetClipboardCopyCommand returns the configured clipboard copy command, or empty for auto-detect
func (c *ArgonautConfig) GetClipboardCopyCommand() string {
	return c.Clipboard.CopyCommand
}

// GetClipboardPasteCommand returns the configured clipboard paste command, or empty for auto-detect
func (c *ArgonautConfig) GetClipboardPasteCommand() string {
	return c.Clipboard.PasteCommand
}

// GetRequestTimeoutString returns the raw string value of the request timeout configuration.
// If no timeout is configured, returns the default value of "10s".
// This method returns the raw string without validation.
func (c *ArgonautConfig) GetRequestTimeoutString() string {
	if c.HTTPTimeouts.RequestTimeout != "" {
		return c.HTTPTimeouts.RequestTimeout
	}
	return "10s"
}

// ParseDefaultView parses the default_view config value into a view, scope type, and scope value.
// Returns zero values if the input is empty. Returns an error message if the input is invalid.
// The view is returned as a string matching model.View constants (e.g. "apps", "clusters").
// When an argument is provided, drill-down logic applies:
//   - cluster+arg → namespaces view scoped to cluster
//   - namespace+arg → projects view scoped to namespace
//   - project+arg → apps view scoped to project
//   - appset+arg → apps view scoped to appset
//   - app+arg → apps view (no scope)
func (c *ArgonautConfig) ParseDefaultView() (view string, scopeType string, scopeValue string, errMsg string) {
	input := strings.TrimSpace(c.DefaultView)
	if input == "" {
		return "", "", "", ""
	}

	// Split on whitespace: command + optional arg
	parts := strings.Fields(input)
	cmd := parts[0]
	var arg string
	if len(parts) > 1 {
		arg = parts[1]
	}

	// Alias lookup: maps all aliases to a canonical command name
	type viewDef struct {
		view      string // view to show (without arg)
		drillView string // view to show when arg is provided
		scopeType string // scope type when arg is provided
	}

	aliases := map[string]viewDef{
		"app":             {view: "apps"},
		"apps":            {view: "apps"},
		"application":     {view: "apps"},
		"applications":    {view: "apps"},
		"cluster":         {view: "clusters", drillView: "namespaces", scopeType: "cluster"},
		"clusters":        {view: "clusters", drillView: "namespaces", scopeType: "cluster"},
		"cls":             {view: "clusters", drillView: "namespaces", scopeType: "cluster"},
		"namespace":       {view: "namespaces", drillView: "projects", scopeType: "namespace"},
		"namespaces":      {view: "namespaces", drillView: "projects", scopeType: "namespace"},
		"ns":              {view: "namespaces", drillView: "projects", scopeType: "namespace"},
		"project":         {view: "projects", drillView: "apps", scopeType: "project"},
		"projects":        {view: "projects", drillView: "apps", scopeType: "project"},
		"proj":            {view: "projects", drillView: "apps", scopeType: "project"},
		"appset":          {view: "applicationsets", drillView: "apps", scopeType: "appset"},
		"appsets":         {view: "applicationsets", drillView: "apps", scopeType: "appset"},
		"applicationset":  {view: "applicationsets", drillView: "apps", scopeType: "appset"},
		"applicationsets": {view: "applicationsets", drillView: "apps", scopeType: "appset"},
		"as":              {view: "applicationsets", drillView: "apps", scopeType: "appset"},
	}

	def, ok := aliases[cmd]
	if !ok {
		return "", "", "", fmt.Sprintf("Malformed default_view in config: %q\nValid options: apps, clusters, ns, proj, appset\nExample: default_view = \"cluster production\"", input)
	}

	if arg == "" || def.drillView == "" {
		return def.view, "", "", ""
	}

	return def.drillView, def.scopeType, arg, ""
}

// GetRequestTimeout returns the parsed duration for request timeout, defaulting to 10s
// Validates that the timeout is positive and returns default if invalid
func (c *ArgonautConfig) GetRequestTimeout() time.Duration {
	if c.HTTPTimeouts.RequestTimeout == "" {
		return 10 * time.Second
	}
	
	duration, err := time.ParseDuration(c.HTTPTimeouts.RequestTimeout)
	if err != nil {
		// If parsing fails, return default
		return 10 * time.Second
	}
	
	// Validate that timeout is positive
	if duration <= 0 {
		// Log warning and return default for zero or negative durations
		return 10 * time.Second
	}
	
	return duration
}