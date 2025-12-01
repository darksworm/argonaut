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
	cmdStr := m.config.GetDiffFormatter()
	cols := 0
	if m.state != nil {
		cols = m.state.Terminal.Cols
	}

	usingDelta := false
	if cmdStr == "" && inPath("delta") {
		usingDelta = true
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
		c.Env = append(os.Environ(), fmt.Sprintf("COLUMNS=%d", cols), "DELTA_PAGER=cat")
	} else {
		c.Env = append(os.Environ(), "DELTA_PAGER=cat")
	}
	c.Stdin = strings.NewReader(diffText)
	out, err := c.CombinedOutput()
	if err != nil {
		return diffText, err
	}

	// Post-process delta's output to clean up the header if we have a resource name
	if usingDelta && resourceName != "" {
		output := string(out)
		lines := strings.Split(output, "\n")

		// Look for the delta header line (contains the file paths)
		for i, line := range lines {
			// Delta's header typically contains "Δ" and the file paths
			if strings.Contains(line, "Δ") && strings.Contains(line, "/var/folders/") {
				// Extract ANSI codes to preserve colors
				// Delta typically uses color codes at the start and resets at the end
				ansiPrefix := ""
				if idx := strings.Index(line, "Δ"); idx > 0 {
					ansiPrefix = line[:idx]
				}

				// Check if line ends with ANSI reset code
				ansiSuffix := ""
				if strings.HasSuffix(line, "\x1b[0m") || strings.HasSuffix(line, "[0m") {
					// Line already has reset code, we'll add it back
					ansiSuffix = "\x1b[0m"
				}

				// Replace with clean header
				lines[i] = fmt.Sprintf("%sΔ %s: Live ⟶ Desired%s", ansiPrefix, resourceName, ansiSuffix)
				break
			}
		}

		return strings.Join(lines, "\n"), nil
	}

	return string(out), nil
}

func inPath(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
func shellEscape(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'" }

// k9sResourceMap maps Kubernetes resource kinds to k9s resource aliases
var k9sResourceMap = map[string]string{
	"Pod":                   "pod",
	"Deployment":            "deploy",
	"Service":               "svc",
	"Ingress":               "ing",
	"ConfigMap":             "cm",
	"Secret":                "secret",
	"ReplicaSet":            "rs",
	"StatefulSet":           "sts",
	"DaemonSet":             "ds",
	"Job":                   "job",
	"CronJob":               "cj",
	"PersistentVolumeClaim": "pvc",
	"PersistentVolume":      "pv",
	"ServiceAccount":        "sa",
	"Namespace":             "ns",
	"Node":                  "node",
	"Event":                 "event",
	"Endpoints":             "ep",
	"HorizontalPodAutoscaler": "hpa",
	"NetworkPolicy":         "netpol",
	"Role":                  "role",
	"RoleBinding":           "rolebinding",
	"ClusterRole":           "clusterrole",
	"ClusterRoleBinding":    "clusterrolebinding",
}

// k9sDoneMsg signals that k9s has exited
type k9sDoneMsg struct{ Err error }
