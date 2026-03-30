package oidc

import (
	"fmt"
	"os/exec"
	"runtime"
)

// OpenBrowser opens url in the system default browser.
// On failure it returns the error but does not panic — the caller should print
// the URL as a fallback.
func OpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: // Linux and other Unix
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("opening browser: %w", err)
	}
	return nil
}
