package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/a9s/go-app/pkg/model"
	"github.com/a9s/go-app/pkg/services"
)

// startLoadingApplications initiates loading applications from ArgoCD API
func (m Model) startLoadingApplications() tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.AuthErrorMsg{Error: fmt.Errorf("no server configured")}
		}
	}

	return tea.Cmd(func() tea.Msg {
		// Log the API call attempt
		fmt.Printf("[API] Starting to load applications from %s\n", m.state.Server.BaseURL)
		
		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)
		
		// Load applications
		fmt.Printf("[API] Calling ListApplications...\n")
		apps, err := apiService.ListApplications(ctx, m.state.Server)
		if err != nil {
			fmt.Printf("[API] Error loading applications: %v\n", err)
			// Check if it's an auth error
			errMsg := err.Error()
			if isAuthenticationError(errMsg) {
				return model.AuthErrorMsg{Error: err}
			}
			return model.ApiErrorMsg{Message: err.Error()}
		}

		// Successfully loaded applications
		fmt.Printf("[API] Successfully loaded %d applications\n", len(apps))
		for i, app := range apps {
			fmt.Printf("[API] App %d: %s (sync: %s, health: %s)\n", i+1, app.Name, app.Sync, app.Health)
		}
		return model.AppsLoadedMsg{Apps: apps}
	})
}

// startWatchingApplications starts the real-time watch stream
func (m Model) startWatchingApplications() tea.Cmd {
	if m.state.Server == nil {
		return nil
	}

	return tea.Cmd(func() tea.Msg {
		// Create context for the watch stream
		ctx := context.Background()
		
		// Create a new ArgoApiService with the current server
		apiService := services.NewArgoApiService(m.state.Server)
		
		// Start watching applications
		eventChan, cleanup, err := apiService.WatchApplications(ctx, m.state.Server)
		if err != nil {
			return model.ApiErrorMsg{Message: "Failed to start watch: " + err.Error()}
		}

		// Process events in a goroutine and send them as tea messages
		go func() {
			defer cleanup()
			
			for event := range eventChan {
				// Convert ArgoApiEvent to appropriate tea message
				switch event.Type {
				case "apps-loaded":
					if event.Apps != nil {
						// Send the loaded apps (this might be sent via a channel in a real implementation)
						// For now, we'll just log it since we can't send tea messages from goroutines
					}
				case "app-updated":
					if event.App != nil {
						// Handle individual app updates
						// In a real implementation, you'd need to send this through a channel
					}
				case "app-deleted":
					if event.AppName != "" {
						// Handle app deletion
					}
				case "status-change":
					if event.Status != "" {
						// Handle status changes
					}
				case "auth-error":
					if event.Error != nil {
						// Handle auth errors
					}
				case "api-error":
					if event.Error != nil {
						// Handle API errors
					}
				}
			}
		}()

		return model.StatusChangeMsg{Status: "Watching for changes..."}
	})
}

// syncSelectedApplications syncs the currently selected applications
func (m Model) syncSelectedApplications(prune bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	selectedApps := make([]string, 0, len(m.state.Selections.SelectedApps))
	for appName := range m.state.Selections.SelectedApps {
		selectedApps = append(selectedApps, appName)
	}

	if len(selectedApps) == 0 {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No applications selected"}
		}
	}

	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		
		for _, appName := range selectedApps {
			err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
			if err != nil {
				return model.ApiErrorMsg{Message: fmt.Sprintf("Failed to sync %s: %v", appName, err)}
			}
		}

		// Clear selections after successful sync
		return model.ClearAllSelectionsMsg{}
	})
}

// syncSingleApplication syncs a specific application
func (m Model) syncSingleApplication(appName string, prune bool) tea.Cmd {
	if m.state.Server == nil {
		return func() tea.Msg {
			return model.ApiErrorMsg{Message: "No server configured"}
		}
	}

	return tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		apiService := services.NewArgoApiService(m.state.Server)
		
		err := apiService.SyncApplication(ctx, m.state.Server, appName, prune)
		if err != nil {
			return model.ApiErrorMsg{Message: fmt.Sprintf("Failed to sync %s: %v", appName, err)}
		}

		return model.StatusChangeMsg{Status: fmt.Sprintf("Synced %s successfully", appName)}
	})
}

// isAuthenticationError checks if an error is related to authentication
func isAuthenticationError(errMsg string) bool {
	authIndicators := []string{
		"401", "403", "unauthorized", "forbidden", "authentication", "auth",
		"login", "token", "invalid credentials", "access denied",
	}
	
	for _, indicator := range authIndicators {
		if strings.Contains(strings.ToLower(errMsg), indicator) {
			return true
		}
	}
	return false
}