package main

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
)

// performContextSwitch re-reads the ArgoCD config and resolves the named context.
// Returns a ContextSwitchResultMsg with the new server config or an error.
func (m *Model) performContextSwitch(contextName string) tea.Cmd {
	configPath := m.argoConfigPath
	currentCtx := m.currentContextName
	return func() tea.Msg {
		// Same-context no-op
		if contextName == currentCtx {
			return model.StatusChangeMsg{Status: "Already on context: " + contextName}
		}

		// Re-read config from disk (tokens may have changed)
		cfg, err := config.ReadCLIConfigFromPath(configPath)
		if err != nil {
			return model.ContextSwitchResultMsg{Error: err}
		}

		// Check for unsupported modes
		if isPF, pfErr := cfg.IsContextPortForward(contextName); pfErr == nil && isPF {
			return model.ContextSwitchResultMsg{
				Error: fmt.Errorf("context %q uses port-forward mode, which is not supported for live switching", contextName),
			}
		}
		if isCore, coreErr := cfg.IsContextCore(contextName); coreErr == nil && isCore {
			return model.ContextSwitchResultMsg{
				Error: fmt.Errorf("context %q uses core mode, which is not supported for live switching", contextName),
			}
		}

		// Resolve context to Server
		server, err := cfg.ToServerConfigForContext(contextName)
		if err != nil {
			return model.ContextSwitchResultMsg{Error: err}
		}

		return model.ContextSwitchResultMsg{
			Server:       server,
			ContextName:  contextName,
			ContextNames: cfg.GetContextNames(),
		}
	}
}

// handleContextSwitchResult processes the result of a context switch.
// It tears down the old model, creates a fresh one via NewModel(), and
// transfers only the narrow set of fields that must survive across contexts.
func (m *Model) handleContextSwitchResult(msg model.ContextSwitchResultMsg) (tea.Model, tea.Cmd) {
	if msg.Error != nil {
		m.statusService.Error("Context switch failed: " + msg.Error.Error())
		return m, nil
	}

	cblog.With("component", "context-switch").Info("Switching context",
		"from", m.currentContextName,
		"to", msg.ContextName,
		"server", msg.Server.BaseURL)

	// 1. Cleanup old goroutines — ORDER MATTERS:
	//    a. Stop forwarding goroutine FIRST
	m.cleanupAppWatcher()
	//    b. Stop tree watchers
	_ = m.cleanupTreeWatchers()
	//    c. Cancel HTTP SSE stream SECOND (after forwarder is stopped)
	if m.watchCleanup != nil {
		m.watchCleanup()
	}

	// 2. Create fresh model with same config (re-applies preferences)
	newM := NewModel(m.config)

	// 3. Re-run Init() side effects that BubbleTea won't call on model swap
	newM.applyThemeToModel()

	// 4. Transfer ONLY what must survive (narrowly defined, test-locked)
	newM.program = m.program                   // BubbleTea program pointer
	newM.state.Terminal = m.state.Terminal      // Current terminal size
	newM.ready = true                          // Already have terminal size
	newM.argoConfigPath = m.argoConfigPath     // For future switches
	newM.currentContextName = msg.ContextName  // New context name
	newM.state.Server = msg.Server             // New server config
	newM.state.ContextNames = msg.ContextNames // From result (no 2nd config read)
	newM.switchEpoch = m.switchEpoch + 1       // Increment epoch

	// 5. Start fresh load cycle
	return newM, tea.Batch(
		newM.spinner.Tick,
		func() tea.Msg { return model.SetInitialLoadingMsg{Loading: true} },
		newM.validateAuthentication(),
	)
}
