//go:build unix

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	tea "charm.land/bubbletea/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/creack/pty"
	"golang.org/x/term"
)

// openK9s runs k9s in a PTY with a status bar at the bottom showing Argonaut context
func (m *Model) openK9s(kind, namespace, context, name string) tea.Cmd {
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
		resourceAlias := kind
		if alias, ok := k9sResourceMap[kind]; ok {
			resourceAlias = alias
		} else {
			resourceAlias = strings.ToLower(kind)
		}

		// Build args - include filter if name is provided
		var args []string
		if name != "" {
			args = []string{"-c", fmt.Sprintf("%s /%s", resourceAlias, name)}
		} else {
			args = []string{"-c", resourceAlias}
		}
		if namespace != "" {
			args = append(args, "-n", namespace)
		}

		// Allow context override via config
		if cfgCtx := m.config.GetK9sContext(); cfgCtx != "" {
			context = cfgCtx
		}
		if context != "" {
			args = append(args, "--context", context)
		}

		cblog.With("component", "k9s").Info("Launching k9s sandboxed", "args", args)

		// Get terminal size
		rows, cols := getTerminalSize()
		if rows < 3 {
			rows = 24
		}
		if cols < 10 {
			cols = 80
		}

		// Reserve 1 row for our status bar at the bottom
		k9sRows := rows - 1

		// Create PTY for k9s with reduced height
		cmd := exec.Command(k9sCmd, args...)
		cmd.Env = os.Environ()

		ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
			Rows: uint16(k9sRows),
			Cols: uint16(cols),
		})
		if err != nil {
			cblog.With("component", "k9s").Error("Failed to start k9s PTY", "err", err)
			return k9sDoneMsg{Err: err}
		}
		defer ptmx.Close()

		// Shared state for current terminal size
		var sizeMu sync.Mutex
		currentRows := rows
		currentCols := cols

		// Handle window resize
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGWINCH)
		defer signal.Stop(sigCh)
		defer close(sigCh) // Close channel to allow goroutine to exit

		go func() {
			for range sigCh {
				newRows, newCols := getTerminalSize()
				if newRows >= 3 && newCols >= 10 {
					sizeMu.Lock()
					currentRows = newRows
					currentCols = newCols
					sizeMu.Unlock()

					k9sRows := newRows - 1
					_ = pty.Setsize(ptmx, &pty.Winsize{
						Rows: uint16(k9sRows),
						Cols: uint16(newCols),
					})
				}
			}
		}()

		// Put stdin into raw mode so keystrokes are passed through immediately
		oldState, err := makeRaw(int(os.Stdin.Fd()))
		if err != nil {
			cblog.With("component", "k9s").Error("Failed to set raw mode", "err", err)
			return k9sDoneMsg{Err: err}
		}
		defer restoreTerminal(int(os.Stdin.Fd()), oldState)

		// Clear screen and draw initial status bar
		fmt.Print("\x1b[2J\x1b[H")
		drawStatusBarBottom(rows, cols, kind, namespace, context, name)

		// Set up input forwarding from stdin to PTY with cancellation
		// Note: On Unix terminals, blocking reads on stdin cannot be interrupted
		// without closing the file descriptor. We close ptmx to unblock writes,
		// but the goroutine may remain blocked on stdin read until the next keystroke.
		stdinDone := make(chan struct{})
		go func() {
			defer close(stdinDone)
			buf := make([]byte, 1024)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					_, err = ptmx.Write(buf[:n])
					if err != nil {
						// ptmx closed, exit goroutine
						return
					}
				}
			}
		}()

		// Process k9s output and inject status bar at frame boundaries
		processK9sOutputWithStatusBar(ptmx, &sizeMu, &currentRows, &currentCols, kind, namespace, context, name)

		// Wait for k9s to exit
		if err := cmd.Wait(); err != nil {
			cblog.With("component", "k9s").Debug("k9s exited", "err", err)
		}

		// Close ptmx to unblock the stdin forwarding goroutine's write
		ptmx.Close()

		// Wait briefly for stdin goroutine to exit (it will exit on next write attempt)
		// Don't block forever - stdin read may be blocked waiting for user input
		select {
		case <-stdinDone:
		case <-time.After(100 * time.Millisecond):
			// Goroutine is blocked on stdin read - this is expected behavior
			// It will exit when the user presses a key or the program terminates
		}

		return k9sDoneMsg{Err: nil}
	}
}

// processK9sOutputWithStatusBar reads k9s output and injects status bar at frame boundaries
func processK9sOutputWithStatusBar(ptmx *os.File, sizeMu *sync.Mutex, rows, cols *int, kind, namespace, context, name string) {
	buf := make([]byte, 32*1024)

	for {
		n, err := ptmx.Read(buf)
		if n > 0 {
			data := buf[:n]

			// Get current terminal size
			sizeMu.Lock()
			r, c := *rows, *cols
			sizeMu.Unlock()

			// Look for frame boundaries and inject status bar
			output := injectStatusBarAtFrameBoundaries(data, r, c, kind, namespace, context, name)

			os.Stdout.Write(output)
		}
		if err != nil {
			break
		}
	}
}

// injectStatusBarAtFrameBoundaries finds frame boundary sequences and injects status bar after them
func injectStatusBarAtFrameBoundaries(data []byte, rows, cols int, kind, namespace, context, name string) []byte {
	// Build the status bar injection sequence
	statusBar := buildStatusBarSequence(rows, cols, kind, namespace, context, name)

	// Patterns that indicate frame boundaries:
	// ESC[2J - clear entire screen (clears our status bar too!)
	// ESC[H - cursor home
	// ESC[;H - cursor home (explicit form)
	// ESC[1;1H - cursor to row 1, col 1
	clearScreen := []byte("\x1b[2J")
	cursorHome := []byte("\x1b[H")
	cursorHomeExplicit := []byte("\x1b[;H")
	cursorHome11 := []byte("\x1b[1;1H")

	var result bytes.Buffer
	i := 0

	for i < len(data) {
		matched := false

		// Check for clear screen - inject status bar AFTER it
		if i+len(clearScreen) <= len(data) && bytes.Equal(data[i:i+len(clearScreen)], clearScreen) {
			result.Write(clearScreen)
			result.Write(statusBar)
			i += len(clearScreen)
			matched = true
		}

		// Check for cursor home patterns - inject status bar BEFORE them (end of previous frame)
		if !matched {
			for _, pattern := range [][]byte{cursorHome11, cursorHomeExplicit, cursorHome} {
				if i+len(pattern) <= len(data) && bytes.Equal(data[i:i+len(pattern)], pattern) {
					result.Write(statusBar)
					result.Write(pattern)
					i += len(pattern)
					matched = true
					break
				}
			}
		}

		if !matched {
			result.WriteByte(data[i])
			i++
		}
	}

	return result.Bytes()
}

// buildStatusBarSequence creates the ANSI sequence to draw the status bar on the last row
func buildStatusBarSequence(rows, cols int, kind, namespace, context, name string) []byte {
	var buf bytes.Buffer

	// Save cursor, move to last row, clear line
	buf.WriteString("\x1b7")                      // Save cursor
	buf.WriteString(fmt.Sprintf("\x1b[%d;1H", rows)) // Move to last row
	buf.WriteString("\x1b[2K")                    // Clear line

	// Build status bar content
	left := " Argonaut » k9s"
	if kind != "" {
		left += fmt.Sprintf(" (%s", kind)
		if namespace != "" {
			left += "/" + namespace
		}
		if name != "" {
			left += ": " + name
		}
		left += ")"
	}
	if context != "" {
		left += " [" + context + "]"
	}
	right := ":q to return "

	// Calculate padding
	padding := cols - len(left) - len(right)
	if padding < 1 {
		padding = 1
	}

	// Draw with colors: blue background (#0087af = 256-color 31), white text
	buf.WriteString("\x1b[48;5;31;97m")
	buf.WriteString(left)
	buf.WriteString(strings.Repeat(" ", padding))
	buf.WriteString(right)
	buf.WriteString("\x1b[0m")

	// Restore cursor
	buf.WriteString("\x1b8")

	return buf.Bytes()
}

// drawStatusBarBottom draws the status bar on the last row of the terminal (for initial draw)
func drawStatusBarBottom(rows, cols int, kind, namespace, context, name string) {
	os.Stdout.Write(buildStatusBarSequence(rows, cols, kind, namespace, context, name))
}

// getTerminalSize returns the current terminal rows and cols
func getTerminalSize() (int, int) {
	ws := struct {
		Row uint16
		Col uint16
		X   uint16
		Y   uint16
	}{}

	_, _, _ = syscall.Syscall(
		syscall.SYS_IOCTL,
		os.Stdout.Fd(),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(&ws)),
	)

	return int(ws.Row), int(ws.Col)
}

// makeRaw puts the terminal into raw mode and returns the previous state
func makeRaw(fd int) (*term.State, error) {
	return term.MakeRaw(fd)
}

// restoreTerminal restores the terminal to a previous state
func restoreTerminal(fd int, state *term.State) {
	if state != nil {
		_ = term.Restore(fd, state)
	}
}
