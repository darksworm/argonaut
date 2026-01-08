// Package clipboard provides clipboard operations for TUI applications.
package clipboard

import (
	"encoding/base64"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
)

var (
	// customCopyCmd is the configured clipboard copy command (set from config)
	customCopyCmd string
	customCopyMu  sync.RWMutex
)

// SetCopyCommand configures a custom clipboard copy command.
// The command receives text via stdin. Examples: "pbcopy", "xclip -selection clipboard"
// Pass an empty string to use auto-detection.
func SetCopyCommand(cmd string) {
	customCopyMu.Lock()
	defer customCopyMu.Unlock()
	customCopyCmd = cmd
}

// GetCopyCommand returns the current custom copy command, or empty string if using auto-detect.
func GetCopyCommand() string {
	customCopyMu.RLock()
	defer customCopyMu.RUnlock()
	return customCopyCmd
}

// OSC 52 escape sequence format:
// ESC ] 52 ; <clipboard> ; <base64-data> BEL
// Where:
// - ESC = \x1b
// - BEL = \x07
// - clipboard = "c" for system clipboard

// CopyMsg is sent after a clipboard copy operation completes.
type CopyMsg struct {
	Success bool
	Text    string
	Method  string // "osc52" or "native"
}

// CopyCmd returns a tea.Cmd that copies text to clipboard.
// It tries OSC 52 first, then falls back to native clipboard (pbcopy on macOS).
func CopyCmd(text string) tea.Cmd {
	if text == "" {
		return func() tea.Msg {
			return CopyMsg{Success: false}
		}
	}

	return func() tea.Msg {
		// Try native clipboard first (more reliable)
		if err := copyNative(text); err == nil {
			cblog.Info("Copied to clipboard via native method", "len", len(text))
			return CopyMsg{Success: true, Text: text, Method: "native"}
		}

		// Fall back to OSC 52
		cblog.Info("Native clipboard failed, trying OSC 52")
		return CopyMsg{Success: true, Text: text, Method: "osc52"}
	}
}

// CopyWithOSC52 returns just the OSC 52 sequence command (for terminals that support it).
func CopyWithOSC52(text string) tea.Cmd {
	if text == "" {
		return nil
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	sequence := "\x1b]52;c;" + encoded + "\x07"

	return tea.Printf("%s", sequence)
}

// copyNative uses the system clipboard directly, or a custom command if configured.
func copyNative(text string) error {
	// Check for custom copy command (from config)
	if customCmd := GetCopyCommand(); customCmd != "" {
		// Parse the command (supports arguments like "xclip -selection clipboard")
		parts := strings.Fields(customCmd)
		if len(parts) == 0 {
			return exec.ErrNotFound
		}
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}

	// Auto-detect clipboard command based on OS
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return exec.ErrNotFound
		}
	default:
		return exec.ErrNotFound
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// Copy is a synchronous version for simple use cases.
func Copy(text string) CopyMsg {
	if text == "" {
		return CopyMsg{Success: false}
	}
	if err := copyNative(text); err == nil {
		return CopyMsg{Success: true, Text: text, Method: "native"}
	}
	return CopyMsg{Success: false}
}
