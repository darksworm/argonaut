//go:build e2e && unix

package main

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "strings"
    "sync"
    "syscall"
    "testing"
    "time"
    "unsafe"

    "github.com/creack/pty"
)

const ringSize = 1 << 20 // 1 MiB scrollback

// ANSI cleaner (CSI + OSC + CR)
var ansiRe = regexp.MustCompile(`(?:\x1b\[[0-9;?]*[ -/]*[@-~])|(?:\x1b\][^\x07]*\x07)|\r`)

// TUITestFramework is a minimal driver for Argonaut e2e tests
type TUITestFramework struct {
    t   *testing.T
    pty *os.File
    tty *os.File
    cmd *exec.Cmd

    workspace string

    mu   sync.Mutex
    buf  []byte
    head int
    full bool
}

func NewTUITest(t *testing.T) *TUITestFramework {
    t.Helper()
    return &TUITestFramework{t: t, buf: make([]byte, ringSize)}
}

// SetupWorkspace creates an isolated HOME and returns ARGOCD_CONFIG path to write
func (tf *TUITestFramework) SetupWorkspace() (string, error) {
    tf.t.Helper()
    dir := tf.t.TempDir()
    tf.workspace = dir
    // Ensure ~/.config/argocd exists
    cfgDir := filepath.Join(dir, ".config", "argocd")
    if err := os.MkdirAll(cfgDir, 0o755); err != nil { return "", err }
    return filepath.Join(cfgDir, "config"), nil
}

// StartApp runs the compiled binary under a PTY
func (tf *TUITestFramework) StartApp(extraEnv ...string) error {
    tf.t.Helper()
    tf.cmd = exec.Command(binPath)
    p, t, err := pty.Open()
    if err != nil { return err }
    tf.pty, tf.tty = p, t

    tf.cmd.Stdout, tf.cmd.Stdin, tf.cmd.Stderr = t, t, t
    env := append(os.Environ(),
        "TERM=xterm-256color",
        "LC_ALL=C",
        "LANG=C",
        "HOME="+tf.workspace,
    )
    env = append(env, extraEnv...)
    tf.cmd.Env = env

    // Set window size
    ws := struct{ Row, Col, X, Y uint16 }{40, 120, 0, 0}
    syscall.Syscall(syscall.SYS_IOCTL, p.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))

    if err := tf.cmd.Start(); err != nil { _ = p.Close(); _ = t.Close(); return err }
    go tf.readLoop()
    return nil
}

// StartAppArgs starts the app with explicit CLI args and optional env
func (tf *TUITestFramework) StartAppArgs(args []string, extraEnv ...string) error {
    tf.t.Helper()
    tf.cmd = exec.Command(binPath, args...)
    p, t, err := pty.Open()
    if err != nil { return err }
    tf.pty, tf.tty = p, t
    tf.cmd.Stdout, tf.cmd.Stdin, tf.cmd.Stderr = t, t, t
    env := append(os.Environ(),
        "TERM=xterm-256color",
        "LC_ALL=C",
        "LANG=C",
        "HOME="+tf.workspace,
    )
    env = append(env, extraEnv...)
    tf.cmd.Env = env
    ws := struct{ Row, Col, X, Y uint16 }{40, 120, 0, 0}
    syscall.Syscall(syscall.SYS_IOCTL, p.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))
    if err := tf.cmd.Start(); err != nil { _ = p.Close(); _ = t.Close(); return err }
    go tf.readLoop()
    return nil
}

func (tf *TUITestFramework) readLoop() {
    buf := make([]byte, 8192)
    for {
        n, err := tf.pty.Read(buf)
        if n > 0 {
            tf.mu.Lock()
            for i := 0; i < n; i++ {
                tf.buf[tf.head] = buf[i]
                tf.head = (tf.head + 1) % ringSize
                if tf.head == 0 { tf.full = true }
            }
            tf.mu.Unlock()
        }
        if err != nil { return }
    }
}

func (tf *TUITestFramework) Send(keys string) error { _, err := tf.pty.Write([]byte(keys)); return err }
func (tf *TUITestFramework) CtrlC() error { return tf.Send("\x03") }
func (tf *TUITestFramework) Enter() error { return tf.Send("\r") }

func (tf *TUITestFramework) Snapshot() string {
    tf.mu.Lock(); defer tf.mu.Unlock()
    if !tf.full { return string(tf.buf[:tf.head]) }
    out := make([]byte, ringSize)
    copy(out, tf.buf[tf.head:])
    copy(out[ringSize-tf.head:], tf.buf[:tf.head])
    return string(out)
}
func (tf *TUITestFramework) SnapshotPlain() string { return ansiRe.ReplaceAllString(tf.Snapshot(), "") }

func (tf *TUITestFramework) WaitForPlain(substr string, timeout time.Duration) bool {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if strings.Contains(tf.SnapshotPlain(), substr) { return true }
        time.Sleep(25 * time.Millisecond)
    }
    return false
}

func (tf *TUITestFramework) Cleanup() {
    if tf.pty != nil { _ = tf.pty.Close(); tf.pty = nil }
    if tf.tty != nil { _ = tf.tty.Close(); tf.tty = nil }
    if tf.cmd != nil && tf.cmd.Process != nil { _ = tf.cmd.Process.Kill(); _, _ = tf.cmd.Process.Wait() }
}

// MockArgoServer spins an httptest server that serves minimal Argo endpoints
func MockArgoServer() (*httptest.Server, error) {
    mux := http.NewServeMux()
    mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
    mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
        // return one simple app
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"destination":{"namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
    })
    mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
    mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte(`{"nodes":[]}`))
    })
    // apps watch stream: send a single apps-loaded style event not required; ListApplications already populates
    mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
        fl, _ := w.(http.Flusher)
        w.Header().Set("Content-Type", "application/json")
        lines := []string{
            `{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo"},"spec":{"destination":{"namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`,
        }
        for _, ln := range lines { _, _ = w.Write([]byte(ln+"\n")); if fl != nil { fl.Flush() } }
    })
    srv := httptest.NewServer(mux)
    return srv, nil
}

// WriteArgoConfig writes an argocd CLI config pointing to our test server
func WriteArgoConfig(path, baseURL string) error {
    var y bytes.Buffer
    y.WriteString("contexts:\n")
    y.WriteString("  - name: default\n    server: "+baseURL+"\n    user: default-user\n")
    y.WriteString("servers:\n")
    y.WriteString("  - server: "+baseURL+"\n    insecure: true\n")
    y.WriteString("users:\n")
    y.WriteString("  - name: default-user\n    auth-token: test-token\n")
    y.WriteString("current-context: default\n")
    return os.WriteFile(path, y.Bytes(), 0o644)
}
