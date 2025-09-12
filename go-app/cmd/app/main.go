package main

import (
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/a9s/go-app/pkg/config"
	"github.com/a9s/go-app/pkg/model"
	"github.com/a9s/go-app/pkg/services"
)

// appVersion is the Argonaut version shown in the ASCII banner.
// Override at build time: go build -ldflags "-X main.appVersion=1.16.0"
var appVersion = "dev"

func main() {
	// Set up logging to file
	setupLogging()

	// Create the initial model
	m := NewModel()

	// Load ArgoCD CLI configuration (matches TypeScript app-orchestrator.ts)
	log.Println("Loading ArgoCD config…")
	
	// Try to read the ArgoCD CLI config file
	server, err := loadArgoConfig()
	if err != nil {
		log.Printf("Could not load ArgoCD config: %v", err)
		log.Println("Please run 'argocd login' to configure and authenticate")
		// Set to nil - the app will show auth-required mode
		m.state.Server = nil
	} else {
		log.Printf("Successfully loaded ArgoCD config for server: %s", server.BaseURL)
		m.state.Server = server
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
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logs directory: %v\n", err)
		return
	}

	// Open log file
	logFile, err := os.OpenFile("logs/a9s.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		return
	}

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	
	log.Println("ArgoCD Apps started")
}

// loadArgoConfig loads ArgoCD CLI configuration (matches TypeScript app-orchestrator.ts)
func loadArgoConfig() (*model.Server, error) {
	// Read CLI config file
	cfg, err := config.ReadCLIConfig()
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
		switch msg.Level {
		case services.StatusLevelError:
			log.Printf("ERROR: %s", msg.Message)
		case services.StatusLevelWarn:
			log.Printf("WARN: %s", msg.Message)
		case services.StatusLevelInfo:
			log.Printf("INFO: %s", msg.Message)
		case services.StatusLevelDebug:
			log.Printf("DEBUG: %s", msg.Message)
		}
	}
}
