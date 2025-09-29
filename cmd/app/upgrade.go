package main

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/services"
)

// handleUpgradeRequest handles the :upgrade command
func (m *Model) handleUpgradeRequest() tea.Cmd {
	cblog.With("component", "upgrade").Info("Upgrade request received")

	// Check if we have update info
	if m.state.UI.UpdateInfo == nil {
		cblog.With("component", "upgrade").Debug("No update info available, checking for updates first")
		return m.checkForUpdates()
	}

	// If no update is available, inform user
	if !m.state.UI.UpdateInfo.Available {
		return func() tea.Msg {
			return model.StatusChangeMsg{Status: "Already running the latest version"}
		}
	}

	// Show upgrade confirmation modal
	m.state.Mode = model.ModeUpgrade
	m.state.Modals.UpgradeSelected = 0
	m.state.Modals.UpgradeLoading = false

	return func() tea.Msg {
		return model.StatusChangeMsg{Status: "Upgrade confirmation"}
	}
}

// checkForUpdates initiates a background update check
func (m *Model) checkForUpdates() tea.Cmd {
	return func() tea.Msg {
		logger := cblog.With("component", "update")
		logger.Info("Checking for updates")

		updateInfo, err := m.updateService.CheckForUpdates(appVersion)
		if err != nil {
			logger.Error("Update check failed", "err", err)
			return model.UpdateCheckCompletedMsg{
				UpdateInfo: nil,
				Error:      err,
			}
		}

		logger.Info("Update check completed",
			"available", updateInfo.Available,
			"current", updateInfo.CurrentVersion,
			"latest", updateInfo.LatestVersion,
			"install_method", updateInfo.InstallMethod)

		return model.UpdateCheckCompletedMsg{
			UpdateInfo: updateInfo,
			Error:      nil,
		}
	}
}

// executeUpgrade performs the actual upgrade process
func (m *Model) executeUpgrade() tea.Cmd {
	if m.state.UI.UpdateInfo == nil {
		return func() tea.Msg {
			return model.UpgradeCompletedMsg{
				Success: false,
				Error:   fmt.Errorf("no update information available"),
			}
		}
	}

	updateInfo := m.state.UI.UpdateInfo

	return func() tea.Msg {
		logger := cblog.With("component", "upgrade")
		logger.Info("Starting upgrade process",
			"from", updateInfo.CurrentVersion,
			"to", updateInfo.LatestVersion,
			"install_method", updateInfo.InstallMethod)

		// Show progress for downloading
		go func() {
			time.Sleep(100 * time.Millisecond) // Small delay to ensure UI updates
		}()

		// Handle different install methods
		switch updateInfo.InstallMethod {
		case model.InstallMethodBrew:
			// User chose to proceed despite Homebrew warning - attempt binary upgrade
			logger.Warn("User proceeding with binary upgrade on Homebrew installation")
			fallthrough
		case model.InstallMethodAUR:
			// User chose to proceed despite AUR warning - attempt binary upgrade
			logger.Warn("User proceeding with binary upgrade on AUR installation")
			fallthrough
		case model.InstallMethodDocker:
			// User chose to proceed despite Docker warning - attempt binary upgrade
			logger.Warn("User proceeding with binary upgrade on Docker installation")
			fallthrough
		case model.InstallMethodManual:
			// Proceed with manual binary replacement
			logger.Info("Performing manual binary upgrade")

			// Download and replace binary
			if err := m.updateService.DownloadAndReplace(updateInfo); err != nil {
				logger.Error("Binary upgrade failed", "err", err)

				// Create context-aware error message based on install method
				var errorMsg error
				switch updateInfo.InstallMethod {
				case model.InstallMethodBrew:
					errorMsg = fmt.Errorf("binary upgrade failed: %v\n\nThis may be due to Homebrew file protection.\nRecommended solution: brew upgrade argonaut\n\nOr download manually from:\nhttps://github.com/darksworm/argonaut/releases/latest", err)
				case model.InstallMethodAUR:
					errorMsg = fmt.Errorf("binary upgrade failed: %v\n\nThis may be due to pacman file protection.\nRecommended solution: yay -Syu argonaut\n\nOr download manually from:\nhttps://github.com/darksworm/argonaut/releases/latest", err)
				case model.InstallMethodDocker:
					errorMsg = fmt.Errorf("binary upgrade failed: %v\n\nContainer filesystem may be read-only or temporary.\nRecommended solution: docker pull ghcr.io/darksworm/argonaut:latest\n\nOr use a persistent volume mount", err)
				default:
					errorMsg = fmt.Errorf("automatic upgrade failed: %v\n\nPlease upgrade manually by downloading the latest release from:\nhttps://github.com/darksworm/argonaut/releases/latest\n\nRefer to the installation instructions in the README for your platform", err)
				}

				return model.UpgradeCompletedMsg{
					Success: false,
					Error:   errorMsg,
				}
			}

			// Show success message and exit cleanly
			logger.Info("Binary upgraded successfully")

			return model.UpgradeCompletedMsg{
				Success: true,
				Error:   nil,
			}
		default:
			return model.UpgradeCompletedMsg{
				Success: false,
				Error:   fmt.Errorf("unknown install method: %s", updateInfo.InstallMethod),
			}
		}
	}
}

// handleUpgradeModeKeys handles input when in upgrade confirmation mode
func (m *Model) handleUpgradeModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.state.Mode = model.ModeNormal
		return m, nil
	case "left", "h":
		if m.state.Modals.UpgradeSelected > 0 {
			m.state.Modals.UpgradeSelected = 0
		}
		return m, nil
	case "right", "l":
		if m.state.Modals.UpgradeSelected < 1 {
			m.state.Modals.UpgradeSelected = 1
		}
		return m, nil
	case "enter":
		if m.state.Modals.UpgradeSelected == 1 {
			// Cancel
			m.state.Mode = model.ModeNormal
			return m, nil
		}
		// Confirm upgrade - show loading and start upgrade
		m.state.Modals.UpgradeLoading = true
		return m, m.executeUpgrade()
	case "y":
		// Quick yes
		m.state.Modals.UpgradeLoading = true
		return m, m.executeUpgrade()
	case "n":
		// Quick no
		m.state.Mode = model.ModeNormal
		return m, nil
	}
	return m, nil
}

// handleUpgradeErrorModeKeys handles input when in upgrade error mode
func (m *Model) handleUpgradeErrorModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "enter":
		// Clear error state and return to normal mode
		m.state.Mode = model.ModeNormal
		m.state.Modals.UpgradeError = nil
		m.state.Modals.UpgradeLoading = false
		return m, nil
	}
	return m, nil
}

// handleUpgradeSuccessModeKeys handles input when in upgrade success mode
func (m *Model) handleUpgradeSuccessModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "enter":
		// Exit the application after successful upgrade
		return m, func() tea.Msg { return model.QuitMsg{} }
	}
	return m, nil
}

// initializeUpdateService initializes the update service in the model
func (m *Model) initializeUpdateService() {
	config := services.UpdateServiceConfig{
		HTTPClient:       nil, // Use default HTTP client
		GitHubRepo:       "darksworm/argonaut",
		CheckIntervalMin: 60, // Check every hour
	}
	m.updateService = services.NewUpdateService(config)
}

// scheduleInitialUpdateCheck performs an initial update check after app startup
func (m *Model) scheduleInitialUpdateCheck() tea.Cmd {
	return func() tea.Msg {
		// Wait a bit after app startup to not interfere with initial loading
		time.Sleep(5 * time.Second)

		logger := cblog.With("component", "update")
		logger.Debug("Performing initial update check")

		updateInfo, err := m.updateService.CheckForUpdates(appVersion)
		if err != nil {
			logger.Debug("Initial update check failed", "err", err)
			return nil
		}

		if updateInfo.Available {
			logger.Info("Update available during initial check",
				"current", updateInfo.CurrentVersion,
				"latest", updateInfo.LatestVersion)
		}

		return model.UpdateCheckCompletedMsg{
			UpdateInfo: updateInfo,
			Error:      nil,
		}
	}
}

// schedulePeriodicUpdateCheck starts a background goroutine for periodic update checks
func (m *Model) schedulePeriodicUpdateCheck() tea.Cmd {
	return func() tea.Msg {
		// Wait before first check to not interfere with app startup
		time.Sleep(30 * time.Second)

		// Start periodic checking
		ticker := time.NewTicker(1 * time.Hour)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					logger := cblog.With("component", "update")
					logger.Debug("Performing periodic update check")

					updateInfo, err := m.updateService.CheckForUpdates(appVersion)
					if err != nil {
						logger.Debug("Periodic update check failed", "err", err)
						continue
					}

					if updateInfo.Available {
						logger.Info("Update available during periodic check",
							"current", updateInfo.CurrentVersion,
							"latest", updateInfo.LatestVersion)
						// Send update info to UI
						// Note: In a real implementation, we'd need a way to send this back to the UI
						// For now, just log it
					}
				}
			}
		}()

		return nil
	}
}
