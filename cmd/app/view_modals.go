package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
)

func (m *Model) renderHelpModal() string {

	// Layout toggle (match earlier TS threshold)
	isWide := m.state.Terminal.Cols >= 60

	// Small keycap style to make keys pop
	keycap := func(s string) string {
		return lipgloss.NewStyle().Foreground(whiteBright).Background(lipgloss.Color("238")).Padding(0, 1).Render(s)
	}
	mono := func(s string) string { return lipgloss.NewStyle().Foreground(cyanBright).Render(s) }
	bullet := func() string { return lipgloss.NewStyle().Foreground(dimColor).Render("•") }

	// GENERAL
	general := strings.Join([]string{
		mono(":"), " command ", bullet(), " ", mono("/"), " search ", bullet(), " ", mono("?"), " help",
	}, "")

	// NAV
	nav := strings.Join([]string{
		mono("j/k"), " up/down ", bullet(), " ", keycap("Space"), " select ", bullet(), " ", keycap("Enter"), " drill down ", bullet(), " ", keycap("Esc"), " clear/up",
	}, "")

	// VIEWS (two lines)
	views := strings.Join([]string{
		mono(":cls"), "|", mono(":clusters"), "|", mono(":cluster"), " ", bullet(), " ", mono(":ns"), "|", mono(":namespaces"), "|", mono(":namespace"),
		"\n",
		mono(":proj"), "|", mono(":projects"), "|", mono(":project"), " ", bullet(), " ", mono(":apps"),
	}, "")

	// ACTIONS (stacked for readability)
	actions := strings.Join([]string{
		mono(":diff"), " [app] ", bullet(), " ", mono(":sync"), " [app] ", bullet(), " ", mono(":rollback"), " [app]",
		"\n",
		mono(":resources"), " [app] ", bullet(), " ", mono(":up"), " go up level",
		"\n",
		// App view hotkeys grouped two per line
		keycap("s"), " sync modal (apps view) ", bullet(), " ", keycap("R"), " rollback modal (apps view)",
		"\n",
		keycap("r"), " resources (apps view) ", bullet(), " ", keycap("d"), " open diff (apps view)",
		"\n",
		keycap("Ctrl+D"), " delete app(s) (apps view)",
	}, "")

	// MISC (licenses removed)
	misc := strings.Join([]string{
		mono(":all"),
		"\n",
		mono(":logs"), " ", bullet(), " ", mono(":q"),
	}, "")

	var helpSections []string
	// Add a blank line between sections to mimic Ink's marginTop=1
	helpSections = append(helpSections, m.renderHelpSection("GENERAL", general, isWide))
	helpSections = append(helpSections, "")
	helpSections = append(helpSections, m.renderHelpSection("NAV", nav, isWide))
	helpSections = append(helpSections, "")
	helpSections = append(helpSections, m.renderHelpSection("VIEWS", views, isWide))
	helpSections = append(helpSections, "")
	helpSections = append(helpSections, m.renderHelpSection("ACTIONS", actions, isWide))
	helpSections = append(helpSections, "")
	helpSections = append(helpSections, m.renderHelpSection("MISC", misc, isWide))
	helpSections = append(helpSections, "")
	helpSections = append(helpSections, statusStyle.Render("Press ?, q or Esc to close"))

	body := "\n" + strings.Join(helpSections, "\n") + "\n"
	// No header: occupy full screen with the help box and status line
	return m.renderFullScreenViewWithOptions("", body, m.renderStatusLine(), FullScreenViewOptions{ContentBordered: true, BorderColor: magentaBright})
}

func (m *Model) renderDiffLoadingSpinner() string {
	spinnerContent := fmt.Sprintf("%s Loading diff...", m.spinner.View())
	spinnerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		Background(lipgloss.Color("0")).
		Foreground(whiteBright).
		Padding(1, 2).
		Bold(true).
		Align(lipgloss.Center)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(spinnerStyle.Render(spinnerContent))
}

// renderTreeLoadingSpinner displays a centered loading spinner for resources/tree operations
func (m *Model) renderTreeLoadingSpinner() string {
	spinnerContent := fmt.Sprintf("%s Loading resources...", m.spinner.View())
	spinnerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Background(lipgloss.Color("0")).
		Foreground(whiteBright).
		Padding(1, 2).
		Bold(true).
		Align(lipgloss.Center)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(spinnerStyle.Render(spinnerContent))
}

// renderRollbackLoadingModal displays a centered modal while rollback is loading/executing
func (m *Model) renderRollbackLoadingModal() string {
	msg := "Loading rollback…"
	if m.state.Rollback != nil {
		if m.state.Rollback.Mode == "confirm" {
			msg = "Executing rollback…"
		} else if m.state.Modals.RollbackAppName != nil {
			msg = "Loading rollback for " + *m.state.Modals.RollbackAppName + "…"
		}
	}
	content := m.spinner.View() + " " + statusStyle.Render(msg)
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outOfSyncColor).
		Padding(1, 2)
	minW := 28
	w := max(minW, lipgloss.Width(content)+4)
	wrapper = wrapper.Width(w)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(content))
}

func (m *Model) renderSyncLoadingModal() string {
	msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Syncing…"))
	content := msg
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2)
	minW := 24
	w := max(minW, lipgloss.Width(content)+4)
	wrapper = wrapper.Width(w)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(content))
}

func (m *Model) renderInitialLoadingModal() string {
	msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Loading..."))
	content := msg
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(magentaBright).
		Padding(1, 2)
	minW := 32
	w := max(minW, lipgloss.Width(content)+4)
	wrapper = wrapper.Width(w)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(content))
}

func (m *Model) renderNoServerModal() string {
	msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Connecting to Argo CD..."))
	content := msg
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(magentaBright).
		Padding(1, 2)
	minW := 40
	w := max(minW, lipgloss.Width(content)+4)
	wrapper = wrapper.Width(w)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(content))
}

func (m *Model) renderRollbackModal() string {
	header := m.renderBanner()
	headerLines := countLines(header)
	const BORDER_LINES = 2
	const STATUS_LINES = 1
	const MARGIN_TOP_LINES = 1 // blank line between header and box
	overhead := BORDER_LINES + headerLines + STATUS_LINES + MARGIN_TOP_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)

	containerWidth := max(0, m.state.Terminal.Cols-2)
	// Expand modal height to fully occupy available space (align with other views)
	// Use +2 here and adjust overall container height below to avoid clipping the status line.
	contentHeight := max(3, availableRows+2)
	innerWidth := max(0, containerWidth-4)
	innerHeight := max(0, contentHeight-2)

	if m.state.Rollback == nil || m.state.Modals.RollbackAppName == nil {
		var content string
		if m.state.Modals.RollbackAppName == nil {
			content = "No app selected for rollback"
		} else {
			content = fmt.Sprintf("Loading deployment history for %s...\n\n%s", *m.state.Modals.RollbackAppName, m.spinner.View())
		}
		return m.renderSimpleModal("Rollback", content)
	}

	rollback := m.state.Rollback
	var modalContent string
	if rollback.Loading {
		if rollback.Mode == "confirm" {
			modalContent = fmt.Sprintf("%s Executing rollback for %s...", m.spinner.View(), rollback.AppName)
		} else {
			modalContent = fmt.Sprintf("%s Loading deployment history for %s...", m.spinner.View(), *m.state.Modals.RollbackAppName)
		}
	} else if rollback.Error != "" {
		errorStyle := lipgloss.NewStyle().Foreground(outOfSyncColor)
		modalContent = errorStyle.Render(fmt.Sprintf("Error loading rollback history:\n%s", rollback.Error))
	} else if rollback.Mode == "confirm" {
		modalContent = m.renderRollbackConfirmation(rollback, innerHeight, innerWidth)
	} else {
		modalContent = m.renderRollbackHistory(rollback)
	}

	if rollback.Mode != "confirm" {
		instructionStyle := lipgloss.NewStyle().Foreground(cyanBright)
		instructions := "j/k: Navigate • Enter: Select • Esc: Cancel"
		modalContent += "\n\n" + instructionStyle.Render(instructions)
	}

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Width(containerWidth).
		Height(contentHeight).
		AlignVertical(lipgloss.Top).
		PaddingLeft(1).
		PaddingRight(1)

	modalContent = normalizeLinesToWidth(modalContent, innerWidth)
	modalContent = clipAnsiToLines(modalContent, innerHeight)
	styledContent := modalStyle.Render(modalContent)

	var sections []string
	sections = append(sections, header)
	// Add one blank line margin above the modal box to match other views
	sections = append(sections, "")
	sections = append(sections, styledContent)
	// Add status line to ensure full-height composition like other views
	status := m.renderStatusLine()
	sections = append(sections, status)

	content := strings.Join(sections, "\n")
	// Use full terminal height here to accommodate the taller rollback modal while
	// keeping the status line visible.
	totalHeight := m.state.Terminal.Rows
	content = clipAnsiToLines(content, totalHeight)
	return mainContainerStyle.Height(totalHeight).Render(content)
}

func (m *Model) renderSimpleModal(title, content string) string {
	header := m.renderBanner()
	headerLines := countLines(header)
	const BORDER_LINES = 2
	const STATUS_LINES = 1
	overhead := BORDER_LINES + headerLines + STATUS_LINES
	availableRows := max(0, m.state.Terminal.Rows-overhead)

	containerWidth := max(0, m.state.Terminal.Cols-2)
	contentWidth := max(0, containerWidth-4)
	contentHeight := max(3, availableRows)

	titleStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
	modalContent := titleStyle.Render(title) + "\n\n" + content

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Width(contentWidth).
		Height(contentHeight).
		AlignVertical(lipgloss.Top).
		PaddingLeft(1).
		PaddingRight(1)

	styledContent := modalStyle.Render(modalContent)
	var sections []string
	sections = append(sections, header)
	sections = append(sections, styledContent)
	// Add status line for consistent height
	sections = append(sections, m.renderStatusLine())

	content = strings.Join(sections, "\n")
	totalHeight := m.state.Terminal.Rows - 1
	return mainContainerStyle.Height(totalHeight).Render(content)
}

// renderUpgradeConfirmModal renders the upgrade confirmation modal
func (m *Model) renderUpgradeConfirmModal() string {
	if m.state.UI.UpdateInfo == nil {
		return ""
	}

	updateInfo := m.state.UI.UpdateInfo

	// Modal styling with reduced padding for smaller terminals
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2).
		Width(68).
		AlignHorizontal(lipgloss.Center)

	// Title with icon
	title := lipgloss.NewStyle().
		Foreground(cyanBright).
		Bold(true).
		Render("🚀 Upgrade Available")

	// Version info with styling (clean up version strings)
	cleanCurrent := strings.TrimPrefix(updateInfo.CurrentVersion, "v")
	cleanLatest := strings.TrimPrefix(updateInfo.LatestVersion, "v")

	currentVersion := lipgloss.NewStyle().
		Foreground(dimColor).
		Render(cleanCurrent)

	latestVersion := lipgloss.NewStyle().
		Foreground(cyanBright).
		Bold(true).
		Render(cleanLatest)

	arrow := lipgloss.NewStyle().
		Foreground(yellowBright).
		Render("→")

	versionInfo := fmt.Sprintf("Current: %s %s Latest: %s",
		currentVersion, arrow, latestVersion)

	// Package manager notice
	notice := lipgloss.NewStyle().
		Foreground(dimColor).
		Render("If you installed argonaut using a package manager\nplease use it to upgrade instead of this in-app upgrade.")

	// Fixed button styling with consistent dimensions
	baseButtonStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(12).
		AlignHorizontal(lipgloss.Center)

	var upgradeButton, cancelButton string
	if m.state.Modals.UpgradeSelected == 0 {
		// Upgrade button selected
		upgradeButton = baseButtonStyle.Copy().
			Background(cyanBright).
			Foreground(black).
			Bold(true).
			Render("Upgrade")
		cancelButton = baseButtonStyle.Copy().
			Background(lipgloss.Color("236")).
			Foreground(dimColor).
			Render("Cancel")
	} else {
		// Cancel button selected
		upgradeButton = baseButtonStyle.Copy().
			Background(lipgloss.Color("236")).
			Foreground(dimColor).
			Render("Upgrade")
		cancelButton = baseButtonStyle.Copy().
			Background(redColor).
			Foreground(white).
			Bold(true).
			Render("Cancel")
	}

	// Build modal content with better spacing
	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n\n")
	content.WriteString(versionInfo)
	content.WriteString("\n")
	content.WriteString(notice)
	content.WriteString("\n\n")

	// Join buttons horizontally with proper spacing
	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Top, upgradeButton, "    ", cancelButton)
	// Center the buttons within the modal content area
	content.WriteString(lipgloss.NewStyle().
		AlignHorizontal(lipgloss.Center).
		Render(buttonsRow))

	return modalStyle.Render(content.String())
}

// renderUpgradeLoadingModal renders the upgrade loading modal
func (m *Model) renderUpgradeLoadingModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2).
		Width(50).
		AlignHorizontal(lipgloss.Center)

	title := lipgloss.NewStyle().
		Foreground(cyanBright).
		Bold(true).
		Render("Upgrading...")

	spinner := m.spinner.View()

	content := fmt.Sprintf("%s\n\n%s Downloading and installing update...\n\nPlease wait...",
		title, spinner)

	return modalStyle.Render(content)
}

// renderUpgradeErrorModal renders the upgrade error modal with manual installation instructions
func (m *Model) renderUpgradeErrorModal() string {
	if m.state.Modals.UpgradeError == nil {
		return ""
	}

	errorMsg := *m.state.Modals.UpgradeError

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(redColor).
		Padding(1, 2).
		Width(80).
		AlignHorizontal(lipgloss.Center)

	title := lipgloss.NewStyle().
		Foreground(redColor).
		Bold(true).
		Render("Upgrade Failed")

	// Format the error message nicely
	content := fmt.Sprintf("%s\n\n%s\n\nPress Enter or Esc to close", title, errorMsg)

	return modalStyle.Render(content)
}

// renderUpgradeSuccessModal renders the upgrade success modal
func (m *Model) renderUpgradeSuccessModal() string {
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(syncedColor).
		Padding(1, 2).
		Width(60).
		AlignHorizontal(lipgloss.Center)

	// Title with icon
	title := lipgloss.NewStyle().
		Foreground(syncedColor).
		Bold(true).
		Render("🎉 Upgrade Complete!")

	// Success checkmark
	checkmark := lipgloss.NewStyle().
		Foreground(syncedColor).
		Bold(true).
		Render("✓")

	// Success message
	successMsg := lipgloss.NewStyle().
		Foreground(whiteBright).
		Render("Successfully upgraded to the latest version")

	// Restart instruction with emphasis
	restartLabel := lipgloss.NewStyle().
		Foreground(yellowBright).
		Bold(true).
		Render("Next step:")

	restartMsg := lipgloss.NewStyle().
		Foreground(whiteBright).
		Render("Restart argonaut to use the new version")

	// Action instruction with styling
	actionMsg := lipgloss.NewStyle().
		Foreground(cyanBright).
		Bold(true).
		Render("Press Enter or Esc to exit")

	// Build content with better spacing and structure
	var content strings.Builder
	content.WriteString(title)
	content.WriteString("\n\n")
	content.WriteString(fmt.Sprintf("%s %s", checkmark, successMsg))
	content.WriteString("\n\n")
	content.WriteString(restartLabel)
	content.WriteString("\n")
	content.WriteString(restartMsg)
	content.WriteString("\n\n")
	content.WriteString(actionMsg)

	return modalStyle.Render(content.String())
}

// renderAppDeleteConfirmModal renders the application delete confirmation modal
func (m *Model) renderAppDeleteConfirmModal() string {
	if m.state.Modals.DeleteAppName == nil {
		return ""
	}

	appName := *m.state.Modals.DeleteAppName
	isMulti := appName == "__MULTI__"

	// Modal width: compact and centered (like sync modal)
	half := m.state.Terminal.Cols / 2
	modalWidth := min(max(36, half), m.state.Terminal.Cols-6)
	innerWidth := max(0, modalWidth-4) // border(2)+padding(2)

	// Message: make all title text bright and readable
	var titleLine string
	{
		// Build parts with consistent bright styling
		deletePart := lipgloss.NewStyle().Foreground(whiteBright).Render("Delete ")
		var subject string
		if isMulti {
			subject = fmt.Sprintf("%d application(s)", len(m.state.Selections.SelectedApps))
		} else {
			subject = appName
		}
		subjectStyled := lipgloss.NewStyle().Foreground(whiteBright).Bold(true).Render(subject)
		qmark := lipgloss.NewStyle().Foreground(whiteBright).Render("?")
		titleLine = deletePart + subjectStyled + qmark
	}

	// Delete button shows confirmation requirement and state
	active := lipgloss.NewStyle().Background(outOfSyncColor).Foreground(whiteBright).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(whiteBright).Padding(0, 2)

	var deleteBtn string
	if m.state.Modals.DeleteConfirmationKey == "y" || m.state.Modals.DeleteConfirmationKey == "Y" {
		deleteBtn = active.Render("Delete")
	} else {
		deleteBtn = inactive.Render("Delete (y)")
	}

	// Simple rounded border with red accent for danger
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outOfSyncColor).
		Padding(1, 2).
		Width(modalWidth)

	// Center helpers
	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)

	title := center.Render(titleLine)

	buttons := center.Render(deleteBtn)

	// Options line for cascade toggle (like prune/watch in sync modal)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	on := lipgloss.NewStyle().Foreground(yellowBright).Bold(true)
	var optsLine strings.Builder
	optsLine.WriteString(dim.Render("c: Cascade "))
	if m.state.Modals.DeleteCascade {
		optsLine.WriteString(on.Render("On"))
		optsLine.WriteString(dim.Render(" (all resources deleted)"))
	} else {
		optsLine.WriteString(dim.Render("Off"))
		optsLine.WriteString(dim.Render(" (resources orphaned)"))
	}
	aux := center.Render(optsLine.String())

	// Lines are already centered to innerWidth
	body := strings.Join([]string{title, "", buttons, "", aux}, "\n")

	// Error display if any
	if m.state.Modals.DeleteError != nil {
		errorMsg := center.Render(lipgloss.NewStyle().
			Foreground(outOfSyncColor).
			Render("Error: " + *m.state.Modals.DeleteError))
		body += "\n\n" + errorMsg
	}

	// Add outer whitespace so the modal doesn't sit directly on top of content
	outer := lipgloss.NewStyle().Padding(1, 1) // 1 blank line top/bottom, 1 space left/right
	return outer.Render(wrapper.Render(body))
}

// renderAppDeleteLoadingModal renders the loading state during application deletion
func (m *Model) renderAppDeleteLoadingModal() string {
	msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Deleting application..."))
	content := msg
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(outOfSyncColor).
		Padding(1, 2)
	minW := 32
	w := max(minW, lipgloss.Width(content)+4)
	wrapper = wrapper.Width(w)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(content))
}

// renderNoDiffModal renders a simple modal for when there are no differences
func (m *Model) renderNoDiffModal() string {
	msg := "✓ " + statusStyle.Render("No differences found")
	content := msg
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(syncedColor).
		Padding(1, 2)
	minW := 28
	w := max(minW, lipgloss.Width(content)+4)
	wrapper = wrapper.Width(w)
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(content))
}
