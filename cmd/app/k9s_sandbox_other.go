//go:build !unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
)

// openK9s on non-Unix systems falls back to running k9s without the status bar sandbox
func (m *Model) openK9s(params K9sResourceParams) tea.Cmd {
	// Store current mode to restore later
	if m.state != nil {
		m.previousMode = m.state.Mode
	}
	
	return func() tea.Msg {
		if m.program != nil {
			m.program.Send(pauseRenderingMsg{})
			_ = m.program.ReleaseTerminal()
		}
		defer func() {
			// Clear screen and restore terminal to Bubble Tea
			fmt.Print("\x1b[2J\x1b[H")
			time.Sleep(150 * time.Millisecond)
			if m.program != nil {
				_ = m.program.RestoreTerminal()
				m.program.Send(resumeRenderingMsg{})
			}
		}()

		// Check if k9s is available
		k9sCmd := m.config.GetK9sCommand()
		if !inPath(k9sCmd) {
			cblog.With("component", "k9s").Error("k9s not found in PATH")
			return k9sDoneMsg{Err: fmt.Errorf("k9s not found in PATH")}
		}

		// Map the kind to k9s resource alias
		resourceAlias := params.Kind
		if alias, ok := k9sResourceMap[params.Kind]; ok {
			resourceAlias = alias
		} else {
			resourceAlias = strings.ToLower(params.Kind)
		}

		// Build args - include filter if name is provided
		var args []string
		if params.Name != "" {
			args = []string{"-c", fmt.Sprintf("%s /%s", resourceAlias, params.Name)}
		} else {
			args = []string{"-c", resourceAlias}
		}
		if params.Namespace != "" {
			args = append(args, "-n", params.Namespace)
		}

		// Allow context override via config
		context := params.Context
		if cfgCtx := m.config.GetK9sContext(); cfgCtx != "" {
			context = cfgCtx
		}
		if context != "" {
			args = append(args, "--context", context)
		}

		cblog.With("component", "k9s").Info("Launching k9s", "args", args)

		c := exec.Command(k9sCmd, args...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			cblog.With("component", "k9s").Error("k9s exited with error", "err", err)
			return k9sDoneMsg{Err: err}
		}
		return k9sDoneMsg{Err: nil}
	}
}
