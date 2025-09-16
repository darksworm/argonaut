package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/noborus/ov/oviewer"
)

// pagerDoneMsg signals that an external pager has closed
type pagerDoneMsg struct{ Err error }
type pauseRenderingMsg struct{}
type resumeRenderingMsg struct{}

// SetProgram stores the Bubble Tea program pointer for terminal hand-off
func (m *Model) SetProgram(p *tea.Program) { m.program = p }

// openTextPager releases the terminal and runs an oviewer pager with the given text
func (m Model) openTextPager(title, text string) tea.Cmd {
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

		// Prepare pager root
		r := strings.NewReader(text)
		root, err := oviewer.NewRoot(r)
		if err != nil {
			cblog.With("component", "pager").Error("Failed to create oviewer root", "err", err)
			return pagerDoneMsg{Err: err}
		}

		cfg := oviewer.NewConfig()
		cfg.IsWriteOnExit = false
		cfg.IsWriteOriginal = false
		cfg.ViewMode = ""

		if err := configureVimKeyBindings(&cfg); err != nil {
			cblog.With("component", "pager").Error("Failed to configure vim keybindings", "err", err)
			return pagerDoneMsg{Err: err}
		}

		root.SetConfig(cfg)

		// Enable ANSI color support by disabling plain mode
		// This preserves ANSI escape sequences for syntax highlighting
		root.Doc.PlainMode = false
		// Capture any error from Run()
		if err := root.Run(); err != nil {
			cblog.With("component", "pager").Error("Failed to run oviewer", "err", err)
			return pagerDoneMsg{Err: err}
		}
		return pagerDoneMsg{Err: nil}
	}
}

// openInteractiveDiffViewer replaces the terminal with an interactive diff tool
// configured via ARGONAUT_DIFF_VIEWER. The command may include {left} and {right}
// placeholders for file paths.
func (m Model) openInteractiveDiffViewer(leftFile, rightFile, cmdStr string) tea.Msg {
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
func (m Model) runDiffFormatter(diffText string) (string, error) {
	cmdStr := os.Getenv("ARGONAUT_DIFF_FORMATTER")
	cols := 0
	if m.state != nil {
		cols = m.state.Terminal.Cols
	}
	if cmdStr == "" && inPath("delta") {
		if cols > 0 {
			cmdStr = fmt.Sprintf("delta --side-by-side --line-numbers --navigate --paging=never --width=%d", cols)
		} else {
			cmdStr = "delta --side-by-side --line-numbers --navigate --paging=never"
		}
	}
	if cmdStr == "" {
		return diffText, nil
	}
	c := exec.Command("sh", "-lc", cmdStr)
	if cols > 0 {
		c.Env = append(os.Environ(), fmt.Sprintf("COLUMNS=%d", cols))
	} else {
		c.Env = os.Environ()
	}
	c.Stdin = strings.NewReader(diffText)
	out, err := c.CombinedOutput()
	if err != nil {
		return diffText, err
	}
	return string(out), nil
}

func inPath(name string) bool     { _, err := exec.LookPath(name); return err == nil }
func shellEscape(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

// configureVimKeyBindings adds vim-like key bindings to the oviewer config
func configureVimKeyBindings(cfg *oviewer.Config) error {
	cfg.DefaultKeyBind = "disable"
	cfg.Keybind = make(map[string][]string)
	set := func(action string, keys ...string) {
		uniq := make(map[string]struct{})
		out := make([]string, 0, len(keys))
		for _, k := range keys {
			if _, ok := uniq[k]; ok {
				continue
			}
			uniq[k] = struct{}{}
			out = append(out, k)
		}
		cfg.Keybind[action] = out
	}
	set("left", "h")
	set("right", "l")
	set("up", "k")
	set("down", "j")
	set("top", "g")
	set("bottom", "G")
	set("search", "/")
	set("exit", "q")
	set("page_down", " ")
	set("page_up", "b")
	return nil
}
