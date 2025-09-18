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

// pagerDoneMsg signals that an external pager has closed
type pagerDoneMsg struct{ Err error }
type pauseRenderingMsg struct{}
type resumeRenderingMsg struct{}

// SetProgram stores the Bubble Tea program pointer for terminal hand-off
func (m *Model) SetProgram(p *tea.Program) { m.program = p }

// openTextPager releases the terminal and runs less -R with the given text
func (m *Model) openTextPager(title, text string) tea.Cmd {
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

		// Use less -R for everything (handles ANSI codes properly)
		c := exec.Command("less", "-R")
		c.Stdin = strings.NewReader(text)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		if err := c.Run(); err != nil {
			cblog.With("component", "pager").Error("Failed to run less", "err", err)
			return pagerDoneMsg{Err: err}
		}
		return pagerDoneMsg{Err: nil}
	}
}

// openInteractiveDiffViewer replaces the terminal with an interactive diff tool
// configured via ARGONAUT_DIFF_VIEWER. The command may include {left} and {right}
// placeholders for file paths.
func (m *Model) openInteractiveDiffViewer(leftFile, rightFile, cmdStr string) tea.Msg {
	if m.program != nil {
		m.program.Send(pauseRenderingMsg{})
		_ = m.program.ReleaseTerminal()
	}
	defer func() {
		fmt.Print("\x1b[2J\x1b[H")
		time.Sleep(150 * time.Millisecond)
		if m.program != nil {
			_ = m.program.RestoreTerminal()
			m.program.Send(resumeRenderingMsg{})
		}
	}()

	cmdStr = strings.ReplaceAll(cmdStr, "{left}", shellEscape(leftFile))
	cmdStr = strings.ReplaceAll(cmdStr, "{right}", shellEscape(rightFile))
	c := exec.Command("sh", "-lc", cmdStr)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		cblog.With("component", "diff").Error("interactive diff viewer failed", "err", err)
		return pagerDoneMsg{Err: err}
	}
	return pagerDoneMsg{Err: nil}
}

// runDiffFormatter runs a non-interactive diff formatter on diffText and returns its output.
// Priority: ARGONAUT_DIFF_FORMATTER if set; else delta (if present); else return input.
func (m *Model) runDiffFormatter(diffText string) (string, error) {
	return m.runDiffFormatterWithTitle(diffText, "")
}

// runDiffFormatterWithTitle runs a diff formatter with optional resource name for better headers
func (m *Model) runDiffFormatterWithTitle(diffText string, resourceName string) (string, error) {
	cmdStr := os.Getenv("ARGONAUT_DIFF_FORMATTER")
	cols := 0
	if m.state != nil {
		cols = m.state.Terminal.Cols
	}

	// Check if we're using delta
	usingDelta := false
	if cmdStr == "" && inPath("delta") {
		usingDelta = true
		if cols > 0 {
			cmdStr = fmt.Sprintf("delta --side-by-side --line-numbers --navigate --paging=never --width=%d", cols)
		} else {
			cmdStr = "delta --side-by-side --line-numbers --navigate --paging=never"
		}

		// Add custom file names if we have a resource name
		if resourceName != "" {
			cmdStr += fmt.Sprintf(" --file-renamed-label='%s Live ⟶   %s Desired'", resourceName, resourceName)
		}
	}

	if cmdStr == "" {
		return diffText, nil
	}

	// If using delta with a resource name, modify the diff headers
	inputDiff := diffText
	if usingDelta && resourceName != "" {
		// Replace the file paths in the diff with cleaner names
		lines := strings.Split(diffText, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "--- a/") {
				lines[i] = fmt.Sprintf("--- a/%s (Live)", resourceName)
			} else if strings.HasPrefix(line, "+++ b/") {
				lines[i] = fmt.Sprintf("+++ b/%s (Desired)", resourceName)
			}
		}
		inputDiff = strings.Join(lines, "\n")
	}

	c := exec.Command("sh", "-lc", cmdStr)
	if cols > 0 {
		c.Env = append(os.Environ(), fmt.Sprintf("COLUMNS=%d", cols), "DELTA_PAGER=cat")
	} else {
		c.Env = append(os.Environ(), "DELTA_PAGER=cat")
	}
	c.Stdin = strings.NewReader(inputDiff)
	out, err := c.CombinedOutput()
	if err != nil {
		return diffText, err
	}
	return string(out), nil
}

func inPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
func shellEscape(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }
