package main

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss/v2"
    "github.com/darksworm/argonaut/pkg/model"
)

// renderStatusLine - 1:1 mapping from MainLayout status Box
func (m Model) renderStatusLine() string {
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

    rightText := "Ready"
    if position != "" {
        rightText = fmt.Sprintf("Ready • %s", position)
    }
    if m.state.UI.IsVersionOutdated {
        rightText += " • Update available!"
    }

    // Layout matching MainLayout justifyContent="space-between"
    leftStyled := statusStyle.Render(leftText)
    rightStyled := statusStyle.Render(rightText)

    // Available width inside main container (accounts for its padding)
    available := max(0, m.state.Terminal.Cols-2)
    // Use lipgloss.Width for accurate spacing
    gap := max(0, available-lipgloss.Width(leftText)-lipgloss.Width(rightText))
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

func (m Model) getSyncIcon(sync string) string {
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

func (m Model) getHealthIcon(health string) string {
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

func (m Model) getColorForStatus(status string) lipgloss.Style {
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
