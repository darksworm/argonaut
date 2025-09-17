package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
)

// appVersion is the Argonaut version shown in the ASCII banner.
// Override at build time: go build -ldflags "-X main.appVersion=1.16.0"
var appVersion = "dev"

func main() {
	// Set up logging to file
	setupLogging()

	// Flags: allow overriding ArgoCD config path for tests and custom setups
	var cfgPathFlag string
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&cfgPathFlag, "argocd-config", "", "Path to ArgoCD CLI config file")
	// Alias
	fs.StringVar(&cfgPathFlag, "config", "", "Path to ArgoCD CLI config file (alias)")
	_ = fs.Parse(os.Args[1:])

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
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Store program pointer for terminal hand-off (pager integration)
	m.SetProgram(p)

	// Run the program
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running program: %v\n", err)
		os.Exit(1)
	}
}

// stringPtr is a helper function to create string pointers
func stringPtr(s string) *string {
	return &s
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
