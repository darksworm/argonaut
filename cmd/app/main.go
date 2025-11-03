package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
	"github.com/darksworm/argonaut/pkg/trust"
)

// appVersion is the Argonaut version shown in the ASCII banner.
// Override at build time: go build -ldflags "-X main.appVersion=1.16.0"
var appVersion = "dev"

// Color definitions for help output (matching app theme)
var (
	helpTitleColor     = lipgloss.Color("14") // Cyan
	helpSectionColor   = lipgloss.Color("11") // Yellow
	helpHighlightColor = lipgloss.Color("10") // Green
	helpTextColor      = lipgloss.Color("15") // Bright white
	helpDimColor       = lipgloss.Color("8")  // Dim
	helpUrlColor       = lipgloss.Color("12") // Blue
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

	// Create the initial model
	m := NewModel()

	// Load Argo CD CLI configuration (matches TypeScript app-orchestrator.ts)
	cblog.With("component", "app").Info("Loading Argo CD config…")

	// Try to read the ArgoCD CLI config file
	server, err := loadArgoConfig(cfgPathFlag)
	if err != nil {
		cblog.With("component", "app").Error("Could not load Argo CD config", "err", err)
		cblog.With("component", "app").Info("Please run 'argocd login' to configure and authenticate")
		// Set to nil - the app will show auth-required mode
		m.state.Server = nil
	} else {
		cblog.With("component", "app").Info("Loaded Argo CD config", "server", server.BaseURL)
		m.state.Server = server
		// Server is configured - the Init() method will handle showing loading screen
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

	// Convert to server config
	server, err := cfg.ToServerConfig()
	if err != nil {
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
