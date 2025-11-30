//go:build !unix

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
)

// openK9s on non-Unix systems falls back to running k9s without the status bar sandbox
func (m *Model) openK9s(kind, namespace, context string) tea.Cmd {
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
		k9sCmd := os.Getenv("ARGONAUT_K9S_COMMAND")
		if k9sCmd == "" {
			k9sCmd = "k9s"
		}
		if !inPath(k9sCmd) {
			cblog.With("component", "k9s").Error("k9s not found in PATH")
			return k9sDoneMsg{Err: fmt.Errorf("k9s not found in PATH")}
		}

		// Map the kind to k9s resource alias
		resourceAlias := kind
		if alias, ok := k9sResourceMap[kind]; ok {
			resourceAlias = alias
		} else {
			resourceAlias = strings.ToLower(kind)
		}

		// Build args
		args := []string{"-c", resourceAlias}
		if namespace != "" {
			args = append(args, "-n", namespace)
		}

		// Allow context override via environment variable
		if envCtx := os.Getenv("ARGONAUT_K9S_CONTEXT"); envCtx != "" {
			context = envCtx
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
