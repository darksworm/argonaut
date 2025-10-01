package main

import (
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/model"
)

// Update and upgrade message handlers

// handleUpdateCheckCompleted processes update check completion results
func (m *Model) handleUpdateCheckCompleted(msg tea.Msg) (tea.Model, tea.Cmd) {
	updateMsg := msg.(model.UpdateCheckCompletedMsg)

	if updateMsg.Error != nil {
		cblog.With("component", "update").Error("Update check failed", "err", updateMsg.Error)
		return m, nil
	}

	if updateMsg.UpdateInfo != nil {
		// Check if this is a new update notification (different version or first time)
		isNewNotification := m.state.UI.UpdateInfo == nil ||
			!m.state.UI.UpdateInfo.Available ||
			m.state.UI.UpdateInfo.LatestVersion != updateMsg.UpdateInfo.LatestVersion

		m.state.UI.UpdateInfo = updateMsg.UpdateInfo
		m.state.UI.IsVersionOutdated = updateMsg.UpdateInfo.Available

		if updateMsg.UpdateInfo.Available {
			// Set notification timestamp for new notifications
			if isNewNotification && updateMsg.UpdateInfo.NotificationShownAt == nil {
				now := time.Now()
				updateMsg.UpdateInfo.NotificationShownAt = &now
				m.state.UI.UpdateInfo = updateMsg.UpdateInfo
			}

			m.state.UI.LatestVersion = &updateMsg.UpdateInfo.LatestVersion
			cblog.With("component", "update").Info("Update available",
				"current", updateMsg.UpdateInfo.CurrentVersion,
				"latest", updateMsg.UpdateInfo.LatestVersion,
				"install_method", updateMsg.UpdateInfo.InstallMethod)
		}
	}

	return m, nil
}

// handleUpgradeRequested processes upgrade requests
func (m *Model) handleUpgradeRequested(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, m.handleUpgradeRequest()
}

// handleUpgradeProgress processes upgrade progress updates
func (m *Model) handleUpgradeProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	progressMsg := msg.(model.UpgradeProgressMsg)
	m.statusService.Set(progressMsg.Message)
	return m, nil
}

// handleUpgradeCompleted processes upgrade completion/failure
func (m *Model) handleUpgradeCompleted(msg tea.Msg) (tea.Model, tea.Cmd) {
	completedMsg := msg.(model.UpgradeCompletedMsg)

	if completedMsg.Success {
		// Show upgrade success modal
		m.state.Mode = model.ModeUpgradeSuccess
		m.state.Modals.UpgradeLoading = false
	} else {
		// Show upgrade error modal with detailed instructions
		errorMsg := completedMsg.Error.Error()
		m.state.Modals.UpgradeError = &errorMsg
		m.state.Mode = model.ModeUpgradeError
		m.state.Modals.UpgradeLoading = false
	}

	return m, nil
}