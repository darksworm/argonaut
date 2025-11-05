package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// renderStatusLine - 1:1 mapping from MainLayout status Box
func (m *Model) renderStatusLine() string {
	visibleItems := m.getVisibleItems()

	// Left side: view and filter info (matches MainLayout left Box)
	leftText := fmt.Sprintf("<%s>", m.state.Navigation.View)
	if m.state.UI.ActiveFilter != "" && m.state.Navigation.View == model.ViewApps {
		leftText = fmt.Sprintf("<%s:%s>", m.state.Navigation.View, m.state.UI.ActiveFilter)
	}

	// Right side: status and position (matches MainLayout right Box)
	// For tree view, use treeView counts; otherwise use list counts.
	position := ""
	if m.state.Navigation.View == model.ViewTree && m.treeView != nil {
		total := m.treeView.VisibleCount()
		if total > 0 {
			position = fmt.Sprintf("%d/%d", m.treeView.SelectedIndex()+1, total)
		}
	} else {
		// Keep old behavior for list views: show 0/0 when empty
		if len(visibleItems) > 0 {
			position = fmt.Sprintf("%d/%d", m.state.Navigation.SelectedIdx+1, len(visibleItems))
		} else {
			position = "0/0"
		}
	}

	// Build the right side text with upgrade notification
	var rightText string

	// Get upgrade notification text based on screen width
	if m.state.UI.IsVersionOutdated && m.shouldShowUpgradeNotification() {
		available := max(0, m.state.Terminal.Cols-2)

		// Progressive text shortening based on available space
		upgradeBG := lipgloss.Color("240")
		upgradeFG := ensureContrastingForeground(upgradeBG, whiteBright)
		upgradeCmd := lipgloss.NewStyle().
			Background(upgradeBG).
			Foreground(upgradeFG).
			Render(":upgrade")

		// Calculate base width (left text + ready + position + spacing)
		baseWidth := lipgloss.Width(leftText) + len("Ready") + 10 // rough estimate for spacing
		if position != "" {
			baseWidth += len(position) + 3 // " • " + position
		}

		remainingSpace := available - baseWidth

		if remainingSpace >= len("New version available, run ")+10 { // +10 for styled command
			rightText = fmt.Sprintf("New version available, run %s • ", upgradeCmd)
		} else if remainingSpace >= len("please ")+10 {
			rightText = fmt.Sprintf("please %s • ", upgradeCmd)
		} else if remainingSpace >= len("pls ")+10 {
			rightText = fmt.Sprintf("pls %s • ", upgradeCmd)
		} else if remainingSpace >= 10 {
			rightText = fmt.Sprintf("%s • ", upgradeCmd)
		}
		// If even that doesn't fit, rightText stays empty (removes notification entirely)
	}

	// Add current status or Ready and position
	statusText := m.statusService.GetCurrentStatus()
	if statusText == "" {
		statusText = "Ready"
	}
	if position != "" {
		statusText += fmt.Sprintf(" • %s", position)
	}

	// Combine the full right side text
	fullRightText := rightText + statusText

	// Layout matching MainLayout justifyContent="space-between"
	leftStyled := statusStyle.Render(leftText)
	rightStyled := statusStyle.Render(fullRightText)

	// Available width inside main container (accounts for its padding)
	available := max(0, m.state.Terminal.Cols-2)
	// Use lipgloss.Width for accurate spacing
	gap := max(0, available-lipgloss.Width(leftText)-lipgloss.Width(fullRightText))
	line := lipgloss.JoinHorizontal(
		lipgloss.Center,
		leftStyled,
		strings.Repeat(" ", gap),
		rightStyled,
	)
	// Ensure the status line exactly fits the available width
	w := lipgloss.Width(line)
	if w < available {
		line = padRight(line, available)
	} else if w > available {
		line = clipAnsiToWidth(line, available)
	}
	return line
}

// Helper functions matching TypeScript utilities

func (m *Model) getSyncIcon(sync string) string {
	switch sync {
	case "Synced":
		return checkIcon
	case "OutOfSync":
		return deltaIcon
	case "Unknown":
		return questIcon
	default:
		return warnIcon
	}
}

func (m *Model) getHealthIcon(health string) string {
	switch health {
	case "Healthy":
		return checkIcon
	case "Missing":
		return questIcon
	case "Degraded":
		return warnIcon
	case "Progressing":
		return dotIcon
	default:
		return questIcon
	}
}

func (m *Model) getColorForStatus(status string) lipgloss.Style {
	switch status {
	case "Synced", "Healthy":
		return lipgloss.NewStyle().Foreground(syncedColor)
	case "OutOfSync", "Degraded":
		return lipgloss.NewStyle().Foreground(outOfSyncColor)
	case "Progressing":
		return lipgloss.NewStyle().Foreground(progressColor)
	default:
		return lipgloss.NewStyle().Foreground(unknownColor)
	}
}

const (
	// upgradeNotificationTimeout defines how long the upgrade notification stays visible
	upgradeNotificationTimeout = 30 * time.Second
)

// shouldShowUpgradeNotification checks if the upgrade notification should be shown
// Returns false if the notification has been shown for more than the timeout duration
func (m *Model) shouldShowUpgradeNotification() bool {
	if m.state.UI.UpdateInfo == nil || m.state.UI.UpdateInfo.NotificationShownAt == nil {
		return true // Show if we haven't started timing yet
	}

	elapsed := time.Since(*m.state.UI.UpdateInfo.NotificationShownAt)
	return elapsed < upgradeNotificationTimeout
}
