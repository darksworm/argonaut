package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/portforward"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/theme"
	"github.com/darksworm/argonaut/pkg/trust"
)

// CoreModeError indicates that ArgoCD is running in core mode
type CoreModeError struct{}

func (e *CoreModeError) Error() string {
	return "ArgoCD is running in core mode"
}

// PortForwardModeError indicates that ArgoCD is configured for port-forward mode
type PortForwardModeError struct {
	Token string
}

func (e *PortForwardModeError) Error() string {
	return "ArgoCD is configured for port-forward mode"
}

// appVersion is the Argonaut version shown in the ASCII banner.
// Override at build time: go build -ldflags "-X main.appVersion=1.16.0"
var appVersion = "dev"

// Color definitions for help output (updated by theme system)
var (
	helpTitleColor     = lipgloss.Color("14") // Cyan (fallback)
	helpSectionColor   = lipgloss.Color("11") // Yellow (fallback)
	helpHighlightColor = lipgloss.Color("10") // Green (fallback)
	helpTextColor      = lipgloss.Color("15") // Bright white (fallback)
	helpDimColor       = lipgloss.Color("8")  // Dim (fallback)
	helpUrlColor       = lipgloss.Color("12") // Blue (fallback)
)

// renderColorfulHelp creates a beautifully styled help output
func renderColorfulHelp(fs *flag.FlagSet) string {
	var help strings.Builder

	// Title with styling
	titleStyle := lipgloss.NewStyle().Foreground(helpTitleColor).Bold(true)
	help.WriteString(titleStyle.Render("argonaut"))
	help.WriteString(" - Interactive terminal UI for Argo CD\n\n")

	// Usage section
	sectionStyle := lipgloss.NewStyle().Foreground(helpSectionColor).Bold(true)
	help.WriteString(sectionStyle.Render("USAGE"))
	help.WriteString("\n  ")
	help.WriteString(lipgloss.NewStyle().Foreground(helpTextColor).Render("argonaut"))
	help.WriteString(lipgloss.NewStyle().Foreground(helpDimColor).Render(" [options]"))
	help.WriteString("\n\n")

	// Options section
	help.WriteString(sectionStyle.Render("OPTIONS"))
	help.WriteString("\n")

	// Capture flag defaults to a buffer
	var flagBuf strings.Builder
	fs.SetOutput(&flagBuf)
	fs.PrintDefaults()
	flagsOutput := flagBuf.String()

	// Style the flags output
	lines := strings.Split(flagsOutput, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "  -") {
			// Flag line - format can be "  -flagname" or "  -flagname type"
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				help.WriteString("  ")
				help.WriteString(lipgloss.NewStyle().Foreground(helpHighlightColor).Render(parts[0])) // -flagname
				if len(parts) > 1 {
					// Has type (like "string")
					help.WriteString(" " + lipgloss.NewStyle().Foreground(helpTextColor).Render(strings.Join(parts[1:], " ")))
				}
				help.WriteString("\n")
			}
		} else if strings.HasPrefix(line, "    \t") {
			// Description line (indented with tab)
			help.WriteString(lipgloss.NewStyle().Foreground(helpDimColor).Render(line))
			help.WriteString("\n")
		}
	}

	// Prerequisites section
	help.WriteString("\n")
	help.WriteString(sectionStyle.Render("PREREQUISITES"))
	help.WriteString("\n")
	help.WriteString("  • ")
	help.WriteString(lipgloss.NewStyle().Foreground(helpHighlightColor).Render("ArgoCD CLI"))
	help.WriteString(lipgloss.NewStyle().Foreground(helpTextColor).Render(" must be installed and configured"))
	help.WriteString("\n  • Run ")
	help.WriteString(lipgloss.NewStyle().Foreground(helpHighlightColor).Render("'argocd login <server>'"))
	help.WriteString(lipgloss.NewStyle().Foreground(helpTextColor).Render(" to authenticate before using argonaut"))
	help.WriteString("\n\n")

	// Optional dependencies section
	help.WriteString(sectionStyle.Render("OPTIONAL DEPENDENCIES"))
	help.WriteString("\n  • ")
	help.WriteString(lipgloss.NewStyle().Foreground(helpHighlightColor).Render("delta"))
	help.WriteString(lipgloss.NewStyle().Foreground(helpTextColor).Render(" - Enhanced diff viewer for better syntax highlighting"))
	help.WriteString("\n    Install: ")
	help.WriteString(lipgloss.NewStyle().Foreground(helpUrlColor).Underline(true).Render("https://github.com/dandavison/delta"))
	help.WriteString("\n\n")

	// Footer
	help.WriteString("For more information, visit: ")
	help.WriteString(lipgloss.NewStyle().Foreground(helpUrlColor).Underline(true).Render("https://github.com/darksworm/argonaut"))
	help.WriteString("\n")

	return help.String()
}

func main() {
	// Set up logging to file
	setupLogging()

	// Flags: allow overriding ArgoCD config path and TLS trust settings
	var (
		cfgPathFlag    string
		caCertFlag     string
		caPathFlag     string
		clientCertFlag string
		clientKeyFlag  string
		themeFlag      string
		showVersion    bool
		showHelp       bool
	)
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&showVersion, "version", false, "Show version information and exit")
	fs.BoolVar(&showHelp, "help", false, "Show help information and exit")
	fs.StringVar(&cfgPathFlag, "argocd-config", "", "Path to ArgoCD CLI config file")
	// Alias
	fs.StringVar(&cfgPathFlag, "config", "", "Path to ArgoCD CLI config file (alias)")
	// TLS trust flags (unified naming)
	fs.StringVar(&caCertFlag, "ca-cert", "", "Path to CA certificate bundle (PEM format)")
	fs.StringVar(&caPathFlag, "ca-path", "", "Directory containing CA certificates (*.pem, *.crt)")
	// Backward-compatible aliases
	fs.StringVar(&caCertFlag, "cacert", "", "Path to CA certificate bundle (alias)")
	fs.StringVar(&caPathFlag, "capath", "", "Directory containing CA certificates (alias)")
	// Client certificate authentication flags
	fs.StringVar(&clientCertFlag, "client-cert", "", "Path to client certificate file (PEM format)")
	fs.StringVar(&clientKeyFlag, "client-cert-key", "", "Path to client certificate private key file (PEM format)")
	// Theme selection flag
	fs.StringVar(&themeFlag, "theme", "", fmt.Sprintf("UI theme preset (%s)", strings.Join(theme.Names(), ", ")))

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			showHelp = true
		} else {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}
	}

	// Handle --version flag
	if showVersion {
		fmt.Println(appVersion)
		return
	}

	// Handle --help flag
	if showHelp {
		fmt.Print(renderColorfulHelp(fs))
		return
	}

	// Set up TLS trust configuration
	setupTLSTrust(caCertFlag, caPathFlag, clientCertFlag, clientKeyFlag)

	// Check if config file exists before loading (for "what's new" logic)
	configExisted := config.ConfigFileExists()

	// Load and apply theme
	argonautConfig, err := config.LoadArgonautConfig()
	if err != nil {
		cblog.With("component", "app").Warn("Could not load config, using defaults", "err", err)
		argonautConfig = config.GetDefaultConfig()
	}

	// Override theme from CLI flag if provided
	if themeFlag != "" {
		argonautConfig.Appearance.Theme = themeFlag
	}

	// Apply theme colors
	palette := theme.FromConfig(argonautConfig)
	applyTheme(palette)

	// Create the initial model
	m := NewModel(argonautConfig)

	// Check if this is a new version (for "what's new" notification)
	if appVersion != "dev" {
		lastSeen := argonautConfig.LastSeenVersion
		if !configExisted {
			// Fresh install - no config file existed, save version, no notification
			argonautConfig.LastSeenVersion = appVersion
			if err := config.SaveArgonautConfig(argonautConfig); err != nil {
				cblog.With("component", "app").Warn("Could not save last seen version", "err", err)
			}
		} else if lastSeen == "" {
			// Config exists but no last_seen_version - existing user upgrading to version with this feature
			// Show notification!
			m.state.UI.ShowWhatsNew = true
			now := time.Now()
			m.state.UI.WhatsNewShownAt = &now
			argonautConfig.LastSeenVersion = appVersion
			if err := config.SaveArgonautConfig(argonautConfig); err != nil {
				cblog.With("component", "app").Warn("Could not save last seen version", "err", err)
			}
		} else if lastSeen != appVersion {
			// User upgraded to a new version - show notification
			m.state.UI.ShowWhatsNew = true
			now := time.Now()
			m.state.UI.WhatsNewShownAt = &now
			argonautConfig.LastSeenVersion = appVersion
			if err := config.SaveArgonautConfig(argonautConfig); err != nil {
				cblog.With("component", "app").Warn("Could not save last seen version", "err", err)
			}
		}
	}

	// Apply saved sort preference from config
	if argonautConfig.Sort.Field != "" {
		m.state.UI.Sort = model.SortConfig{
			Field:     model.SortField(argonautConfig.Sort.Field),
			Direction: model.SortDirection(argonautConfig.Sort.Direction),
		}
	}

	// Load Argo CD CLI configuration (matches TypeScript app-orchestrator.ts)
	cblog.With("component", "app").Info("Loading Argo CD config…")

	// Port-forward manager (if used)
	var pfManager *portforward.Manager

	// Try to read the ArgoCD CLI config file
	server, err := loadArgoConfig(cfgPathFlag)
	if err != nil {
		// Check if it's a port-forward mode error
		if pfErr, isPortForward := err.(*PortForwardModeError); isPortForward {
			cblog.With("component", "app").Info("Port-forward mode detected, starting kubectl port-forward")

			// Check kubectl availability
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if kubectlErr := portforward.CheckKubectl(ctx); kubectlErr != nil {
				cancel()
				fmt.Fprintf(os.Stderr, "Error: kubectl is required for port-forward mode but was not found.\n")
				fmt.Fprintf(os.Stderr, "Please install kubectl and ensure it's in your PATH.\n")
				os.Exit(1)
			}
			cancel()

			// Get namespace from Argonaut config
			namespace := argonautConfig.GetPortForwardNamespace()

			// Create port-forward manager
			pfManager = portforward.NewManager(portforward.Options{
				Namespace: namespace,
				OnDisconnect: func(pfDisconnectErr error) {
					// Port-forward failed permanently - exit with error
					fmt.Fprintf(os.Stderr, "Error: port-forward connection lost: %v\n", pfDisconnectErr)
					os.Exit(1)
				},
			})

			// Start port-forward
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			localPort, pfStartErr := pfManager.Start(ctx)
			cancel()
			if pfStartErr != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to establish port-forward: %v\n", pfStartErr)
				fmt.Fprintf(os.Stderr, "\nTroubleshooting tips:\n")
				fmt.Fprintf(os.Stderr, "  • Ensure ArgoCD is installed in namespace '%s'\n", namespace)
				fmt.Fprintf(os.Stderr, "  • Check that you have kubectl access to the cluster\n")
				fmt.Fprintf(os.Stderr, "  • Verify the argocd-server pod is running\n")
				os.Exit(1)
			}

			cblog.With("component", "app").Info("Port-forward established", "localPort", localPort, "namespace", namespace)

			// Create server config using local port-forward address
			server = &model.Server{
				BaseURL:  fmt.Sprintf("http://127.0.0.1:%d", localPort),
				Token:    pfErr.Token,
				Insecure: true, // Local connection doesn't need TLS verification
			}
			m.state.Server = server

		} else if _, isCoreError := err.(*CoreModeError); isCoreError {
			// Check if it's a core mode error
			cblog.With("component", "app").Info("ArgoCD core installation detected")
			// Set mode to show core detection view
			m.state.Mode = model.ModeCoreDetected
			m.state.Server = nil
		} else {
			cblog.With("component", "app").Error("Could not load Argo CD config", "err", err)
			cblog.With("component", "app").Info("Please run 'argocd login' to configure and authenticate")
			// Set to nil - the app will show auth-required mode
			m.state.Server = nil
		}
	} else {
		cblog.With("component", "app").Info("Loaded Argo CD config", "server", server.BaseURL)
		m.state.Server = server
		// Server is configured - the Init() method will handle showing loading screen
	}

	// Ensure port-forward is cleaned up on exit
	if pfManager != nil {
		defer pfManager.Stop()
	}

	// Start with empty apps - they will be loaded from API
	m.state.Apps = []model.App{}

	// Create the Bubbletea program
    p := tea.NewProgram(
        m,
    )

	// Store program pointer for terminal hand-off (pager integration)
	m.SetProgram(p)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// setupLogging configures logging to write to a file instead of stdout
func setupLogging() {
	// Create temp log file and expose path via env for the logs view
	f, err := os.CreateTemp("", "a9s-*.log")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create temp log file: %v\n", err)
		return
	}
	_ = os.Setenv("ARGONAUT_LOG_FILE", f.Name())

	// Standard library log to same file (for any remaining log.Printf)
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Charmbracelet/log to same file
	logger := cblog.NewWithOptions(f, cblog.Options{ReportTimestamp: true})
	switch strings.ToUpper(os.Getenv("ARGONAUT_LOG_LEVEL")) {
	case "DEBUG":
		logger.SetLevel(cblog.DebugLevel)
	case "WARN":
		logger.SetLevel(cblog.WarnLevel)
	case "ERROR":
		logger.SetLevel(cblog.ErrorLevel)
	case "FATAL":
		logger.SetLevel(cblog.FatalLevel)
	default:
		logger.SetLevel(cblog.InfoLevel)
	}
	cblog.SetDefault(logger)

	cblog.With("component", "app").Info("Argo CD Apps started", "logFile", f.Name())
}

// loadArgoConfig loads ArgoCD CLI configuration (matches TypeScript app-orchestrator.ts)
func loadArgoConfig(overridePath string) (*model.Server, error) {
	// Read CLI config file (override path if specified)
	var (
		cfg *config.ArgoCLIConfig
		err error
	)
	if overridePath != "" {
		cfg, err = config.ReadCLIConfigFromPath(overridePath)
	} else {
		// Still respect ARGOCD_CONFIG environment variable via ReadCLIConfig()
		cfg, err = config.ReadCLIConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read CLI config: %w", err)
	}

	// Check if port-forward mode is configured
	if isPortForward, pfErr := cfg.IsPortForwardMode(); pfErr == nil && isPortForward {
		// Get the auth token for port-forward mode
		token, tokenErr := cfg.GetCurrentToken()
		if tokenErr != nil {
			return nil, fmt.Errorf("port-forward mode requires authentication: %w", tokenErr)
		}
		return nil, &PortForwardModeError{Token: token}
	}

	// Convert to server config
	server, err := cfg.ToServerConfig()
	if err != nil {
		// Check if it's an auth error AND we're in core mode
		if isCore, coreErr := cfg.IsCurrentServerCore(); coreErr == nil && isCore {
			// Check if the error is about missing auth token
			if strings.Contains(err.Error(), "auth token") || strings.Contains(err.Error(), "no token") {
				return nil, &CoreModeError{}
			}
		}
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}

	return server, nil
}

// createFileStatusHandler creates a status handler that logs to file
func createFileStatusHandler() services.StatusChangeHandler {
	return func(msg services.StatusMessage) {
		logger := cblog.With("component", "status")
		switch msg.Level {
		case services.StatusLevelError:
			logger.Error(msg.Message)
		case services.StatusLevelWarn:
			logger.Warn(msg.Message)
		case services.StatusLevelInfo:
			logger.Info(msg.Message)
		case services.StatusLevelDebug:
			logger.Debug(msg.Message)
		}
	}
}

// setupTLSTrust configures TLS trust using the trust package
func setupTLSTrust(caCertFile, caCertDir, clientCertFile, clientKeyFile string) {
	// Only configure custom TLS trust if flags or environment variables are provided
	if caCertFile == "" && caCertDir == "" && clientCertFile == "" && clientKeyFile == "" &&
		os.Getenv("SSL_CERT_FILE") == "" && os.Getenv("SSL_CERT_DIR") == "" {
		return
	}

	// Configure trust options
	opts := trust.Options{
		CACertFile:     caCertFile,
		CACertDir:      caCertDir,
		ClientCertFile: clientCertFile,
		ClientKeyFile:  clientKeyFile,
		Timeout:        30 * time.Second, // Default HTTP timeout
		MinTLS:         tls.VersionTLS12, // Minimum TLS 1.2
	}

	// Load certificate pool
    pool, err := trust.LoadPool(opts)
    if err != nil {
        cblog.With("component", "tls").Error("Failed to load certificate pool", "err", err)
        // Print hint in the same line to avoid race with PTY readers in CI
        fmt.Fprintf(os.Stderr, "TLS configuration failed: %v. Hint: Use --ca-cert or --ca-path to add trusted CAs, or install your CA in the OS trust store\n", err)
        os.Exit(1)
    }

	// Load client certificate if provided
	var clientCert *tls.Certificate

	if clientCertFile != "" && clientKeyFile != "" {
		cblog.With("component", "tls").Info("Loading client certificate for mutual TLS authentication",
			"cert", clientCertFile, "key", clientKeyFile)
		var err error
		clientCert, err = trust.LoadClientCertificate(clientCertFile, clientKeyFile)
        if err != nil {
            cblog.With("component", "tls").Error("Failed to load client certificate", "err", err)
            // Include hint inline to avoid PTY read races
            fmt.Fprintf(os.Stderr, "Client certificate configuration failed: %v. Hint: Ensure --client-cert and --client-cert-key point to valid certificate files\n", err)
            os.Exit(1)
        }
		cblog.With("component", "tls").Info("Client certificate loaded successfully")
	} else if clientCertFile != "" || clientKeyFile != "" {
		cblog.With("component", "tls").Warn("Incomplete client certificate configuration - both --client-cert and --client-cert-key are required")
		fmt.Fprintf(os.Stderr, "Warning: Both --client-cert and --client-cert-key must be provided for client certificate authentication\n")
	}

	// Create HTTP client with trust configuration
	httpClient, _ := trust.NewHTTP(pool, clientCert, opts.MinTLS, opts.Timeout)

	// Set the HTTP client globally for all API operations
	api.SetHTTPClient(httpClient)

	// Log successful trust setup
	var certSources []string
	if caCertFile != "" {
		certSources = append(certSources, "1 file")
	}
	if caCertDir != "" {
		certSources = append(certSources, "dir certs")
	}

	sourceStr := "system roots"
	if len(certSources) > 0 {
		sourceStr += " + " + strings.Join(certSources, " + ")
	}

	var authMethod string
	if clientCert != nil {
		authMethod = " + client cert auth"
	}

	cblog.With("component", "tls").Info("TLS trust configured", "sources", sourceStr+authMethod)
}
