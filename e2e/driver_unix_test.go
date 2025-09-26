//go:build e2e && unix

package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
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

// ensureBinary builds the app test binary if it doesn't exist yet.
func ensureBinary(t *testing.T) error {
	t.Helper()
	// Resolve absolute binPath under e2e dir
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	p := binPath
	if !filepath.IsAbs(p) {
		p = filepath.Join(cwd, p)
	}
	// Already exists?
	if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
		binPath = p
		return nil
	}
	// Build it
	cmd := exec.Command("go", "build", "-o", p, "./cmd/app")
	cmd.Dir = ".."
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("build failed: %v\n%s", err, string(out))
	}
	binPath = p
	return nil
}

// SetupWorkspace creates an isolated HOME and returns ARGOCD_CONFIG path to write
func (tf *TUITestFramework) SetupWorkspace() (string, error) {
	tf.t.Helper()
	dir := tf.t.TempDir()
	tf.workspace = dir
	// Ensure ~/.config/argocd exists
	cfgDir := filepath.Join(dir, ".config", "argocd")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(cfgDir, "config"), nil
}

// StartApp runs the compiled binary under a PTY
func (tf *TUITestFramework) StartApp(extraEnv ...string) error {
	tf.t.Helper()
	if err := ensureBinary(tf.t); err != nil {
		return err
	}
	tf.cmd = exec.Command(binPath)
	p, t, err := pty.Open()
	if err != nil {
		return err
	}
	tf.pty, tf.tty = p, t

	tf.cmd.Stdout, tf.cmd.Stdin, tf.cmd.Stderr = t, t, t
	// Run the app in the isolated workspace so per-test files (e.g., logs) don't clash
	if tf.workspace != "" {
		tf.cmd.Dir = tf.workspace
	}
	env := append(os.Environ(),
		"TERM=xterm-256color",
		"LC_ALL=C",
		"LANG=C",
		"HOME="+tf.workspace,
		"ARGONAUT_E2E=1",
	)
	env = append(env, extraEnv...)
	tf.cmd.Env = env

	// Set window size
	ws := struct{ Row, Col, X, Y uint16 }{40, 120, 0, 0}
	syscall.Syscall(syscall.SYS_IOCTL, p.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))

	if err := tf.cmd.Start(); err != nil {
		_ = p.Close()
		_ = t.Close()
		return err
	}
	go tf.readLoop()
	return nil
}

// StartAppArgs starts the app with explicit CLI args and optional env
func (tf *TUITestFramework) StartAppArgs(args []string, extraEnv ...string) error {
	tf.t.Helper()
	if err := ensureBinary(tf.t); err != nil {
		return err
	}
	tf.cmd = exec.Command(binPath, args...)
	p, t, err := pty.Open()
	if err != nil {
		return err
	}
	tf.pty, tf.tty = p, t
	tf.cmd.Stdout, tf.cmd.Stdin, tf.cmd.Stderr = t, t, t
	if tf.workspace != "" {
		tf.cmd.Dir = tf.workspace
	}
	env := append(os.Environ(),
		"TERM=xterm-256color",
		"LC_ALL=C",
		"LANG=C",
		"HOME="+tf.workspace,
		"ARGONAUT_E2E=1",
	)
	env = append(env, extraEnv...)
	tf.cmd.Env = env
	ws := struct{ Row, Col, X, Y uint16 }{40, 120, 0, 0}
	syscall.Syscall(syscall.SYS_IOCTL, p.Fd(), uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(&ws)))
	if err := tf.cmd.Start(); err != nil {
		_ = p.Close()
		_ = t.Close()
		return err
	}
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
				if tf.head == 0 {
					tf.full = true
				}
			}
			tf.mu.Unlock()
		}
		if err != nil {
			return
		}
	}
}

func (tf *TUITestFramework) Send(keys string) error { _, err := tf.pty.Write([]byte(keys)); return err }
func (tf *TUITestFramework) CtrlC() error           { return tf.Send("\x03") }
func (tf *TUITestFramework) Enter() error           { return tf.Send("\r") }

func (tf *TUITestFramework) Snapshot() string {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	if !tf.full {
		return string(tf.buf[:tf.head])
	}
	out := make([]byte, ringSize)
	copy(out, tf.buf[tf.head:])
	copy(out[ringSize-tf.head:], tf.buf[:tf.head])
	return string(out)
}
func (tf *TUITestFramework) SnapshotPlain() string { return ansiRe.ReplaceAllString(tf.Snapshot(), "") }

func (tf *TUITestFramework) WaitForPlain(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(tf.SnapshotPlain(), substr) {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// OpenCommand enters command mode and waits for the command bar to be ready.
func (tf *TUITestFramework) OpenCommand() error {
	if err := tf.Send(":"); err != nil {
		return err
	}
	// Command bar shows a light-gray prompt "> " when ready
	if !tf.WaitForPlain("> ", 2*time.Second) {
		return fmt.Errorf("command bar not ready")
	}
	return nil
}

// OpenSearch enters search mode and waits for the search bar to be ready.
func (tf *TUITestFramework) OpenSearch() error {
	if err := tf.Send("/"); err != nil {
		return err
	}
	// Search bar shows label "Search" when ready
	if !tf.WaitForPlain("Search", 2*time.Second) {
		return fmt.Errorf("search bar not ready")
	}
	return nil
}

func (tf *TUITestFramework) Cleanup() {
	if tf.pty != nil {
		_ = tf.pty.Close()
		tf.pty = nil
	}
	if tf.tty != nil {
		_ = tf.tty.Close()
		tf.tty = nil
	}
	if tf.cmd != nil && tf.cmd.Process != nil {
		_ = tf.cmd.Process.Kill()
		_, _ = tf.cmd.Process.Wait()
	}
}

// MockArgoServer spins an httptest server that serves minimal Argo endpoints
func MockArgoServer() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		// return one simple app with cluster and project metadata
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
            {"kind":"Deployment","name":"demo","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced"},
            {"kind":"ReplicaSet","name":"demo-rs","namespace":"default","version":"v1","group":"apps","uid":"rs-1","status":"Synced","parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"demo","namespace":"default","group":"apps","version":"v1"}]}
        ]}`))
	})
	// apps watch stream: send a single apps-loaded style event not required; ListApplications already populates
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")
		lines := []string{
			`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`,
		}
		for _, ln := range lines {
			_, _ = w.Write([]byte(ln + "\n"))
			if fl != nil {
				fl.Flush()
			}
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// MockArgoServerStreaming creates a server that sends multiple streaming updates
func MockArgoServerStreaming() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		// Initial app with OutOfSync status
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"demo","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced"},
			{"kind":"ReplicaSet","name":"demo-rs","namespace":"default","version":"v1","group":"apps","uid":"rs-1","status":"Synced","parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"demo","namespace":"default","group":"apps","version":"v1"}]}
		]}`))
	})
	// Streaming endpoint that sends multiple updates
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")

		// Send initial state in SSE format
		lines := []string{
			`data: {"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}}}`,
		}

		for _, ln := range lines {
			_, _ = w.Write([]byte(ln + "\n"))
			if fl != nil {
				fl.Flush()
			}
		}

		// Wait for UI to have time to render initial state before sending update
		time.Sleep(1500 * time.Millisecond)

		// Send sync status update in SSE format
		updateLines := []string{
			`data: {"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`,
		}

		for _, ln := range updateLines {
			_, _ = w.Write([]byte(ln + "\n"))
			if fl != nil {
				fl.Flush()
			}
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// WriteArgoConfig writes an argocd CLI config pointing to our test server
func WriteArgoConfig(path, baseURL string) error {
	return WriteArgoConfigWithToken(path, baseURL, "test-token")
}

// MockArgoServerAuth requires Authorization: Bearer <validToken> or returns 401
func MockArgoServerAuth(validToken string) (*httptest.Server, error) {
	mux := http.NewServeMux()
	// Enforce auth on userinfo and applications to drive auth-required view deterministically
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+validToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+validToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo"}}}}\n`))
		if fl != nil {
			fl.Flush()
		}
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// MockArgoServerExpiredToken returns 401 with a structured JSON body like Argo CD when token is expired
func MockArgoServerExpiredToken() (*httptest.Server, error) {
	mux := http.NewServeMux()
	body := `{"error":"invalid session: token has invalid claims: token is expired","code":16,"message":"invalid session: token has invalid claims: token is expired"}`
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(body))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	srv := httptest.NewServer(mux)
	return srv, nil
}

// WriteArgoConfigWithToken writes a CLI config using a specific token
func WriteArgoConfigWithToken(path, baseURL, token string) error {
	var y bytes.Buffer
	y.WriteString("contexts:\n")
	y.WriteString("  - name: default\n    server: " + baseURL + "\n    user: default-user\n")
	y.WriteString("servers:\n")
	y.WriteString("  - server: " + baseURL + "\n    insecure: true\n")
	y.WriteString("users:\n")
	y.WriteString("  - name: default-user\n    auth-token: " + token + "\n")
	y.WriteString("current-context: default\n")
	return os.WriteFile(path, y.Bytes(), 0o644)
}

// MockArgoServerForbidden returns 403 Forbidden for applications (simulating RBAC/forbidden)
func MockArgoServerForbidden() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden","message":"forbidden"}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden","message":"forbidden"}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"forbidden","message":"forbidden"}`))
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// MockArgoServerStreamUnauthorized returns 200 for list but 401 for stream (expired token mid-flow)
func MockArgoServerStreamUnauthorized() (*httptest.Server, error) {
	mux := http.NewServeMux()
	// Require valid token for userinfo and applications
	requireAuth := func(w http.ResponseWriter, r *http.Request) bool {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+"valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return false
		}
		return true
	}
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		// Simulate expired token on stream
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid session: token is expired","code":16,"message":"invalid session: token is expired"}`))
	})
	srv := httptest.NewServer(mux)
	return srv, nil
}

// ---- Sync capturing mock server ----

type SyncCall struct {
	Name string
	Body string
}

type SyncRecorder struct {
	mu    sync.Mutex
	Calls []SyncCall
}

func (sr *SyncRecorder) add(call SyncCall) {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	sr.Calls = append(sr.Calls, call)
}

func (sr *SyncRecorder) len() int {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	return len(sr.Calls)
}

// MockArgoServerSync returns an auth-checking server and a recorder for /sync calls
func MockArgoServerSync(validToken string) (*httptest.Server, *SyncRecorder, error) {
	rec := &SyncRecorder{}
	mux := http.NewServeMux()
	requireAuth := func(w http.ResponseWriter, r *http.Request) bool {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+validToken {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
			return false
		}
		return true
	}
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// Two apps for multi-select scenario
		_, _ = w.Write([]byte(`{"items":[
            {"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}},
            {"metadata":{"name":"demo2","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"OutOfSync"},"health":{"status":"Healthy"}}}
        ]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"nodes":[]}`))
	})
	mux.HandleFunc("/api/v1/applications/demo2/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		_, _ = w.Write([]byte(`{"nodes":[]}`))
	})

	// Diffs for :diff
	mux.HandleFunc("/api/v1/applications/demo/managed-resources", func(w http.ResponseWriter, r *http.Request) {
		if !requireAuth(w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		// One resource with different image tag
		live := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"demo","namespace":"default"},"data":{"IMAGE":"nginx:1.25"}}`
		desired := `{"apiVersion":"v1","kind":"ConfigMap","metadata":{"name":"demo","namespace":"default"},"data":{"IMAGE":"nginx:1.26"}}`
		_, _ = w.Write([]byte(`{"items":[{"kind":"ConfigMap","namespace":"default","name":"demo","liveState":` + jsonEscape(live) + `,"targetState":` + jsonEscape(desired) + `}]}`))
	})

	// Prefix handler for sync
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if !requireAuth(w, r) {
			return
		}
		// Expect path like /api/v1/applications/<name>/sync
		p := r.URL.Path
		if !strings.HasSuffix(p, "/sync") {
			http.NotFound(w, r)
			return
		}
		segs := strings.Split(p, "/")
		if len(segs) < 6 {
			http.NotFound(w, r)
			return
		}
		name := segs[4]
		body, _ := io.ReadAll(r.Body)
		rec.add(SyncCall{Name: name, Body: string(body)})
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})

	srv := httptest.NewServer(mux)
	return srv, rec, nil
}

// jsonEscape returns a JSON string literal for a raw string (quoted)
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// MockArgoServerHTTPS creates an HTTPS test server using provided certificate files
func MockArgoServerHTTPS(certFile, keyFile string) (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		// return one simple app with cluster and project metadata
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
            {"kind":"Deployment","name":"demo","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced"},
            {"kind":"ReplicaSet","name":"demo-rs","namespace":"default","version":"v1","group":"apps","uid":"rs-1","status":"Synced","parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"demo","namespace":"default","group":"apps","version":"v1"}]}
        ]}`))
	})
	// apps watch stream
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")
		lines := []string{
			`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`,
		}
		for _, ln := range lines {
			_, _ = w.Write([]byte(ln + "\n"))
			if fl != nil {
				fl.Flush()
			}
		}
	})

	srv := httptest.NewUnstartedServer(mux)

	// Load the certificate and key
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %v", err)
	}

	// Configure TLS
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	srv.StartTLS()
	return srv, nil
}

// WriteArgoConfigHTTPS writes an ArgoCD CLI config pointing to an HTTPS server without insecure flag
func WriteArgoConfigHTTPS(path, baseURL string) error {
	return WriteArgoConfigHTTPSWithToken(path, baseURL, "test-token")
}

// WriteArgoConfigHTTPSWithToken writes a CLI config using a specific token for HTTPS server
func WriteArgoConfigHTTPSWithToken(path, baseURL, token string) error {
	var y bytes.Buffer
	y.WriteString("contexts:\n")
	y.WriteString("  - name: default\n    server: " + baseURL + "\n    user: default-user\n")
	y.WriteString("servers:\n")
	y.WriteString("  - server: " + baseURL + "\n    insecure: false\n")  // Note: insecure: false for TLS validation
	y.WriteString("users:\n")
	y.WriteString("  - name: default-user\n    auth-token: " + token + "\n")
	y.WriteString("current-context: default\n")
	return os.WriteFile(path, y.Bytes(), 0o644)
}

// MockArgoServerHTTPSWithClientAuth creates an HTTPS test server that requires client certificates
func MockArgoServerHTTPSWithClientAuth(certFile, keyFile, caFile string) (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200); _, _ = w.Write([]byte(`{}`)) })
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		// return one simple app with cluster and project metadata
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(`{"version":"e2e"}`)) })
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"nodes":[
            {"kind":"Deployment","name":"demo","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced"},
            {"kind":"ReplicaSet","name":"demo-rs","namespace":"default","version":"v1","group":"apps","uid":"rs-1","status":"Synced","parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"demo","namespace":"default","group":"apps","version":"v1"}]}
        ]}`))
	})
	// apps watch stream
	mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
		fl, _ := w.(http.Flusher)
		w.Header().Set("Content-Type", "application/json")
		lines := []string{
			`{"result":{"type":"MODIFIED","application":{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}}}`,
		}
		for _, ln := range lines {
			_, _ = w.Write([]byte(ln + "\n"))
			if fl != nil {
				fl.Flush()
			}
		}
	})

	srv := httptest.NewUnstartedServer(mux)

	// Load the server certificate and key
	serverCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %v", err)
	}

	// Load the CA certificate for client verification
	caCertPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA file: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// Configure TLS with client certificate requirement
	srv.TLS = &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    caCertPool,
	}

	srv.StartTLS()
	return srv, nil
}
