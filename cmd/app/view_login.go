package main

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// renderLoginModal renders the login modal for authentication
func (m *Model) renderLoginModal() string {
	serverURL := m.state.Modals.LoginServerURL
	if serverURL == "" {
		serverURL = "—"
	}

	// Modal width: centered and reasonably sized
	half := m.state.Terminal.Cols / 2
	modalWidth := min(max(50, half), m.state.Terminal.Cols-6)
	innerWidth := max(0, modalWidth-4) // border(2)+padding(2)

	// Styles
	dim := lipgloss.NewStyle().Foreground(dimColor)
	bold := lipgloss.NewStyle().Foreground(whiteBright).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(outOfSyncColor)

	// Active/inactive button styles
	inactiveFG := ensureContrastingForeground(inactiveBG, whiteBright)
	active := lipgloss.NewStyle().Background(magentaBright).Foreground(textOnAccent).Bold(true).Padding(0, 2)
	inactive := lipgloss.NewStyle().Background(inactiveBG).Foreground(inactiveFG).Padding(0, 2)

	// Center helper
	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)
	left := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Left)

	// Title
	title := center.Render(bold.Render("Login to ArgoCD"))

	// Server URL
	serverLine := left.Render(dim.Render("Server: ") + bold.Render(serverURL))

	// Username input
	usernameLabel := labelStyle.Render("Username:")
	// Set input width based on modal
	inputWidth := max(20, innerWidth-15)
	m.inputComponents.usernameInput.SetWidth(inputWidth)
	usernameInput := m.inputComponents.RenderUsernameInput()
	usernameLine := left.Render(usernameLabel + " " + usernameInput)

	// Password input
	passwordLabel := labelStyle.Render("Password:")
	m.inputComponents.passwordInput.SetWidth(inputWidth)
	passwordInput := m.inputComponents.RenderPasswordInput()
	passwordLine := left.Render(passwordLabel + " " + passwordInput)

	// Save credentials checkbox
	checkboxStyle := lipgloss.NewStyle().Foreground(cyanBright)
	checkbox := "[ ]"
	if m.state.Modals.LoginSaveCredentials {
		checkbox = "[x]"
	}
	// Highlight checkbox if focused
	checkboxText := "Save credentials to keychain"
	if m.state.Modals.LoginFieldFocus == 2 {
		checkbox = checkboxStyle.Bold(true).Render(checkbox)
		checkboxText = bold.Render(checkboxText)
	} else {
		checkbox = dim.Render(checkbox)
		checkboxText = dim.Render(checkboxText)
	}
	saveLine := left.Render(checkbox + " " + checkboxText)

	// Buttons
	loginBtn := inactive.Render("Login")
	cancelBtn := inactive.Render("Cancel")
	if m.state.Modals.LoginFieldFocus == 3 {
		loginBtn = active.Render("Login")
	}
	if m.state.Modals.LoginFieldFocus == 4 {
		cancelBtn = active.Render("Cancel")
	}
	buttons := lipgloss.JoinHorizontal(lipgloss.Center, loginBtn, strings.Repeat(" ", 4), cancelBtn)
	buttonsLine := center.Render(buttons)

	// Error message (if any)
	var errorLine string
	if m.state.Modals.LoginError != nil && *m.state.Modals.LoginError != "" {
		errorLine = center.Render(errorStyle.Render(*m.state.Modals.LoginError))
	}

	// Loading indicator
	var loadingLine string
	if m.state.Modals.LoginLoading {
		loadingLine = center.Render(dim.Render("Logging in..."))
	}

	// Build modal content
	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")
	lines = append(lines, serverLine)
	lines = append(lines, "")
	lines = append(lines, usernameLine)
	lines = append(lines, passwordLine)
	lines = append(lines, "")
	lines = append(lines, saveLine)
	lines = append(lines, "")
	lines = append(lines, buttonsLine)
	if errorLine != "" {
		lines = append(lines, "")
		lines = append(lines, errorLine)
	}
	if loadingLine != "" {
		lines = append(lines, "")
		lines = append(lines, loadingLine)
	}

	body := strings.Join(lines, "\n")

	// Modal wrapper with border
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2).
		Width(modalWidth)

	// Add outer whitespace
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(body))
}

// renderLoginLoadingView renders a loading view during auto-login
func (m *Model) renderLoginLoadingView() string {
	serverURL := m.state.Modals.LoginServerURL
	if serverURL == "" {
		serverURL = "ArgoCD"
	}

	// Modal width: centered and reasonably sized
	half := m.state.Terminal.Cols / 2
	modalWidth := min(max(40, half), m.state.Terminal.Cols-6)
	innerWidth := max(0, modalWidth-4)

	// Styles
	bold := lipgloss.NewStyle().Foreground(whiteBright).Bold(true)
	dim := lipgloss.NewStyle().Foreground(dimColor)

	// Center helper
	center := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center)

	// Content
	title := center.Render(bold.Render("Authenticating..."))
	serverLine := center.Render(dim.Render("Connecting to " + serverURL))
	spinnerLine := center.Render(m.spinner.View())

	body := strings.Join([]string{title, "", serverLine, "", spinnerLine}, "\n")

	// Modal wrapper with border
	wrapper := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyanBright).
		Padding(1, 2).
		Width(modalWidth)

	// Add outer whitespace
	outer := lipgloss.NewStyle().Padding(1, 1)
	return outer.Render(wrapper.Render(body))
}

// renderLoginView renders the full login view (modal centered on screen)
func (m *Model) renderLoginView() string {
	var modal string
	if m.state.Mode == model.ModeLoginLoading {
		modal = m.renderLoginLoadingView()
	} else {
		modal = m.renderLoginModal()
	}

	// Center the modal on screen
	modalHeight := strings.Count(modal, "\n") + 1
	modalWidth := lipgloss.Width(modal)

	// Calculate padding to center
	topPadding := max(0, (m.state.Terminal.Rows-modalHeight)/2)
	leftPadding := max(0, (m.state.Terminal.Cols-modalWidth)/2)

	// Build centered view
	var lines []string
	for i := 0; i < topPadding; i++ {
		lines = append(lines, "")
	}

	// Split modal into lines and add left padding
	modalLines := strings.Split(modal, "\n")
	for _, line := range modalLines {
		lines = append(lines, strings.Repeat(" ", leftPadding)+line)
	}

	return strings.Join(lines, "\n")
}
