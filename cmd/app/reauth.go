package main

import (
	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/auth"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
)

// handleTriggerReauthMsg starts the SSO re-authentication flow.
// It is called from Update() when a TriggerReauthMsg is received.
func (m *Model) handleTriggerReauthMsg() (tea.Model, tea.Cmd) {
	// If already waiting for argocd login, ignore duplicate signals.
	if m.state.Mode == model.ModeReauthPending {
		return m, nil
	}

	// Guard against infinite reauth loops.
	m.reauthAttempts++
	if m.reauthAttempts > 2 {
		m.reauthAttempts = 0
		cblog.With("component", "reauth").Warn("Reauth attempt limit reached, falling back to manual login")
		m.state.Mode = model.ModeAuthRequired
		return m, nil
	}

	m.state.Mode = model.ModeReauthPending
	m.state.Modals.InitialLoading = false

	params := auth.LoginParams{
		ServerURL:       auth.StripProtocol(m.state.Server.BaseURL),
		ContextName:     m.currentContextName,
		Insecure:        m.state.Server.Insecure,
		GrpcWeb:         m.state.Server.GrpcWeb,
		GrpcWebRootPath: m.state.Server.GrpcWebRootPath,
		ConfigPath:      m.argoConfigPath,
	}

	epoch := m.switchEpoch
	loginCmd := m.jwtAuthProvider.LoginCmd(params)
	return m, tea.ExecProcess(loginCmd, func(err error) tea.Msg {
		return model.ReauthCompleteMsg{Err: err, SwitchEpoch: epoch}
	})
}

// handleReauthCompleteMsg processes the result of argocd login --sso.
// It is called from Update() when a ReauthCompleteMsg is received.
func (m *Model) handleReauthCompleteMsg(msg model.ReauthCompleteMsg) (tea.Model, tea.Cmd) {
	// Discard results from a previous epoch (e.g. a context switch during reauth).
	if msg.SwitchEpoch != m.switchEpoch {
		cblog.With("component", "reauth").Debug("ReauthCompleteMsg: stale epoch, discarding",
			"msg_epoch", msg.SwitchEpoch, "current_epoch", m.switchEpoch)
		return m, nil
	}

	if msg.Err != nil {
		cblog.With("component", "reauth").Error("argocd login failed", "err", msg.Err)
		m.state.Mode = model.ModeAuthRequired
		m.err = msg.Err
		return m, nil
	}

	// Re-read the updated token from disk.
	cliCfg, err := config.ReadCLIConfigFromPath(m.argoConfigPath)
	if err != nil {
		cblog.With("component", "reauth").Error("Failed to re-read ArgoCD config after reauth", "err", err)
		m.state.Mode = model.ModeAuthRequired
		m.err = err
		return m, nil
	}
	server, err := cliCfg.ToServerConfig()
	if err != nil {
		cblog.With("component", "reauth").Error("Failed to parse server config after reauth", "err", err)
		m.state.Mode = model.ModeAuthRequired
		m.err = err
		return m, nil
	}

	cblog.With("component", "reauth").Info("Reauth succeeded, resuming")
	m.state.Server = server
	m.reauthAttempts = 0
	// Increment epoch to invalidate any goroutines from before the reauth
	// (stale watch consumers, inflight API calls).
	m.switchEpoch++

	return m, m.validateAuthentication()
}
