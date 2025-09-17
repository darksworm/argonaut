package main

import (
	"context"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
	"time"
)

// Init implements tea.Model.Init
func (m Model) Init() tea.Cmd {
	// Initialize with terminal size request and startup commands
	var cmds []tea.Cmd
	cmds = append(cmds, tea.EnterAltScreen, m.spinner.Tick)

	// Show initial loading modal immediately if server is configured
	if m.state.Server != nil {
		cmds = append(cmds, func() tea.Msg { return model.SetInitialLoadingMsg{Loading: true} })
	}

	cmds = append(cmds,
		func() tea.Msg { return model.StatusChangeMsg{Status: "Initializing..."} },
		// Validate authentication if server is configured
		m.validateAuthentication(),
	)

	_ = context.TODO() // keep import stable if unused on some builds
	_ = time.Second
	return tea.Batch(cmds...)
}
