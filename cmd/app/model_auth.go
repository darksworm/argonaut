package main

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/auth"
	"github.com/darksworm/argonaut/pkg/model"
)

// handleAutoLoginAttempt attempts to authenticate using stored credentials
func (m *Model) handleAutoLoginAttempt(msg model.AutoLoginAttemptMsg) tea.Cmd {
	return func() tea.Msg {
		log := cblog.With("component", "auth")
		serverURL := msg.ServerURL
		insecure := msg.Insecure

		if serverURL == "" {
			log.Debug("No server URL for auto-login")
			return model.AutoLoginResultMsg{
				Success: false,
				Error:   nil, // No error, just no server configured
			}
		}

		log.Debug("Attempting auto-login", "serverURL", serverURL)

		authManager := auth.NewAuthManager(auth.AuthManagerConfig{
			ServerURL: serverURL,
			Insecure:  insecure,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := authManager.TryAutoLogin(ctx)
		if err != nil {
			log.Debug("Auto-login failed", "err", err)
			return model.AutoLoginResultMsg{
				Success: false,
				Error:   err,
			}
		}

		log.Info("Auto-login successful", "username", result.Username)
		return model.AutoLoginResultMsg{
			Success:  true,
			Token:    result.Token,
			Username: result.Username,
		}
	}
}

// handleLoginSubmit handles the login form submission
func (m *Model) handleLoginSubmit(msg model.LoginSubmitMsg) tea.Cmd {
	return func() tea.Msg {
		log := cblog.With("component", "auth")
		serverURL := m.state.Modals.LoginServerURL

		if serverURL == "" {
			return model.LoginErrorMsg{Error: "No server configured"}
		}

		log.Debug("Attempting login", "serverURL", serverURL, "username", msg.Username)

		// Get insecure setting from the server state if available
		insecure := false
		if m.state.Server != nil {
			insecure = m.state.Server.Insecure
		}

		authManager := auth.NewAuthManager(auth.AuthManagerConfig{
			ServerURL: serverURL,
			Insecure:  insecure,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		result, err := authManager.LoginWithCredentials(ctx, msg.Username, msg.Password, msg.SaveCredentials)
		if err != nil {
			log.Debug("Login failed", "err", err)
			return model.LoginErrorMsg{Error: err.Error()}
		}

		log.Info("Login successful", "username", msg.Username)
		return model.LoginSuccessMsg{
			Token:    result.Token,
			Username: result.Username,
		}
	}
}

// handleLoginSuccess processes successful login
func (m *Model) handleLoginSuccess(msg model.LoginSuccessMsg) (tea.Model, tea.Cmd) {
	log := cblog.With("component", "auth")
	log.Info("Processing login success")

	// Update server with new token
	if m.state.Server == nil {
		m.state.Server = &model.Server{
			BaseURL: m.state.Modals.LoginServerURL,
		}
	}
	m.state.Server.Token = msg.Token

	// Clear login state
	m.state.Modals.LoginLoading = false
	m.state.Modals.LoginError = nil
	m.inputComponents.ClearLoginInputs()

	// Transition to loading mode to fetch apps
	m.state.Mode = model.ModeLoading

	// Start loading apps
	return m, m.validateAuthentication()
}

// handleLoginError processes login error
func (m *Model) handleLoginError(msg model.LoginErrorMsg) (tea.Model, tea.Cmd) {
	log := cblog.With("component", "auth")
	log.Debug("Login error", "error", msg.Error)

	m.state.Modals.LoginLoading = false
	m.state.Modals.LoginError = &msg.Error

	// Re-focus username input
	m.state.Modals.LoginFieldFocus = 0
	m.inputComponents.FocusUsernameInput()

	return m, nil
}

// handleAutoLoginResult processes the result of auto-login attempt
func (m *Model) handleAutoLoginResult(msg model.AutoLoginResultMsg) (tea.Model, tea.Cmd) {
	log := cblog.With("component", "auth")

	if msg.Success {
		log.Info("Auto-login succeeded, updating server token")

		// Update server with new token
		if m.state.Server == nil {
			m.state.Server = &model.Server{
				BaseURL: m.state.Modals.LoginServerURL,
			}
		}
		m.state.Server.Token = msg.Token

		// Transition to loading mode
		m.state.Mode = model.ModeLoading
		return m, m.validateAuthentication()
	}

	// Auto-login failed, show login modal
	log.Debug("Auto-login failed, showing login modal")

	// Pre-fill username if we have stored credentials
	if m.state.Modals.LoginServerURL != "" {
		authManager := auth.NewAuthManager(auth.AuthManagerConfig{
			ServerURL: m.state.Modals.LoginServerURL,
		})
		if storedUsername := authManager.GetStoredUsername(); storedUsername != "" {
			m.inputComponents.SetUsernameValue(storedUsername)
			m.state.Modals.LoginUsername = storedUsername
		}
	}

	// Show login modal
	m.state.Mode = model.ModeLogin
	m.state.Modals.LoginFieldFocus = 0
	m.state.Modals.LoginSaveCredentials = true // Default to save
	m.inputComponents.FocusUsernameInput()

	return m, nil
}

// initLoginModal initializes the login modal with server URL
func (m *Model) initLoginModal(serverURL string, insecure bool) {
	m.state.Modals.LoginServerURL = serverURL
	m.state.Modals.LoginFieldFocus = 0
	m.state.Modals.LoginSaveCredentials = true
	m.state.Modals.LoginError = nil
	m.state.Modals.LoginLoading = false
	m.inputComponents.ClearLoginInputs()

	// Store insecure setting
	if m.state.Server == nil {
		m.state.Server = &model.Server{
			BaseURL:  serverURL,
			Insecure: insecure,
		}
	}
}

// triggerAutoLogin creates a command to attempt auto-login
func (m *Model) triggerAutoLogin() tea.Cmd {
	serverURL := m.state.Modals.LoginServerURL
	insecure := false
	if m.state.Server != nil {
		insecure = m.state.Server.Insecure
	}

	return func() tea.Msg {
		return model.AutoLoginAttemptMsg{
			ServerURL: serverURL,
			Insecure:  insecure,
		}
	}
}
