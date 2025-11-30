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

	// Also ensure ~/.config/argonaut exists for isolated Argonaut config
	argonautCfgDir := filepath.Join(dir, ".config", "argonaut")
	if err := os.MkdirAll(argonautCfgDir, 0o755); err != nil {
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
		// Force isolated Argonaut config - clear any inherited config paths
		"ARGONAUT_CONFIG="+filepath.Join(tf.workspace, ".config", "argonaut", "config.toml"),
		"XDG_CONFIG_HOME=", // Clear XDG_CONFIG_HOME to ensure HOME-based path is used
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
		// Force isolated Argonaut config - clear any inherited config paths
		"ARGONAUT_CONFIG="+filepath.Join(tf.workspace, ".config", "argonaut", "config.toml"),
		"XDG_CONFIG_HOME=", // Clear XDG_CONFIG_HOME to ensure HOME-based path is used
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
	// Command bar shows "│ > " with box drawing character - unique to command input
	// Note: "> " alone matches the ASCII art logo, so we need the box char prefix
	if !tf.WaitForPlain("│ > ", 2*time.Second) {
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
	// Save snapshot if test failed
	if tf.t.Failed() {
		tf.saveFailureSnapshot()
	}

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

// saveFailureSnapshot saves both raw and plain snapshots to files for debugging
func (tf *TUITestFramework) saveFailureSnapshot() {
	// Create snapshots directory if it doesn't exist
	snapshotDir := "test-snapshots"
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		tf.t.Logf("Failed to create snapshot dir: %v", err)
		return
	}

	// Sanitize test name for filename
	testName := strings.ReplaceAll(tf.t.Name(), "/", "_")
	testName = strings.ReplaceAll(testName, " ", "_")

	// Save raw snapshot (with ANSI codes) for replay
	rawPath := filepath.Join(snapshotDir, testName+".raw")
	if err := os.WriteFile(rawPath, []byte(tf.Snapshot()), 0o644); err != nil {
		tf.t.Logf("Failed to save raw snapshot: %v", err)
	} else {
		tf.t.Logf("Saved raw snapshot to %s", rawPath)
	}

	// Save plain snapshot (ANSI stripped) for easy reading
	plainPath := filepath.Join(snapshotDir, testName+".txt")
	if err := os.WriteFile(plainPath, []byte(tf.SnapshotPlain()), 0o644); err != nil {
		tf.t.Logf("Failed to save plain snapshot: %v", err)
	} else {
		tf.t.Logf("Saved plain snapshot to %s", plainPath)
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
	// Handle delete operations
	mux.HandleFunc("/api/v1/applications/demo", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			// Add small delay to ensure loading modal is visible
			time.Sleep(200 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			// Return proper AppDeleteResponse with Success field
			_, _ = w.Write([]byte(`{"Success": true}`))
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
	y.WriteString("  - server: " + baseURL + "\n    insecure: false\n") // Note: insecure: false for TLS validation
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

// ---- K9s testing helpers ----

// createMockK9s creates a mock k9s shell script in the workspace and returns its path and args file path.
// The mock script records its arguments to a file and exits with the given exit code.
func createMockK9s(t *testing.T, workspace string, exitCode int) (scriptPath, argsFile string) {
	t.Helper()

	// Create bin directory in workspace
	binDir := filepath.Join(workspace, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	scriptPath = filepath.Join(binDir, "mock-k9s")
	argsFile = filepath.Join(workspace, "k9s_args.txt")

	// Create the mock script - output escape sequences that Argonaut's PTY handler recognizes
	// The script must output a clear screen sequence for the status bar injection to work
	script := fmt.Sprintf(`#!/bin/sh
# Mock k9s - records args and exits
printf "%%s" "$*" > %q
# Output clear screen sequence (triggers Argonaut's status bar injection)
printf '\033[2J\033[H'
printf 'Mock k9s\n'
# Brief delay then exit
sleep 0.2
exit %d
`, argsFile, exitCode)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create mock k9s: %v", err)
	}

	return scriptPath, argsFile
}

// createInteractiveMockK9s creates a mock k9s that reads stdin and records it to a file.
// This is used to verify that keyboard input is correctly forwarded to k9s.
func createInteractiveMockK9s(t *testing.T, workspace string) (scriptPath, argsFile, inputFile string) {
	t.Helper()

	binDir := filepath.Join(workspace, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	scriptPath = filepath.Join(binDir, "mock-k9s-interactive")
	argsFile = filepath.Join(workspace, "k9s_args.txt")
	inputFile = filepath.Join(workspace, "k9s_input.txt")

	// Create a mock script that:
	// 1. Records args
	// 2. Outputs clear screen (triggers status bar)
	// 3. Reads stdin with timeout and records to input file
	// 4. Exits
	//
	// We use 'read' with timeout since head -c in background doesn't work
	// for PTY stdin. The -t flag provides timeout, -n provides char count.
	script := fmt.Sprintf(`#!/bin/bash
# Record args
printf "%%s" "$*" > %q
# Clear screen and show message
printf '\033[2J\033[H'
printf 'Mock k9s - type keys\n'
# Read 5 chars with 2 second timeout
read -t 2 -n 5 INPUT 2>/dev/null || true
printf "%%s" "$INPUT" > %q
exit 0
`, argsFile, inputFile)

	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to create interactive mock k9s: %v", err)
	}

	return scriptPath, argsFile, inputFile
}

// readMockK9sInput reads the keyboard input that was captured by the interactive mock k9s.
func readMockK9sInput(t *testing.T, inputFile string) string {
	t.Helper()
	data, err := os.ReadFile(inputFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ""
		}
		t.Fatalf("failed to read k9s input file: %v", err)
	}
	return string(data)
}

// readMockK9sArgs reads the arguments that were passed to the mock k9s script.
func readMockK9sArgs(t *testing.T, argsFile string) string {
	t.Helper()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "" // k9s was not invoked
		}
		t.Fatalf("failed to read k9s args file: %v", err)
	}
	return strings.TrimSpace(string(data))
}

// setupSingleContextKubeconfig creates a kubeconfig file with a single context.
func setupSingleContextKubeconfig(t *testing.T, workspace, contextName string) string {
	t.Helper()

	kubeDir := filepath.Join(workspace, ".kube")
	if err := os.MkdirAll(kubeDir, 0o755); err != nil {
		t.Fatalf("failed to create .kube dir: %v", err)
	}

	kubeconfigPath := filepath.Join(kubeDir, "config")
	content := fmt.Sprintf(`apiVersion: v1
kind: Config
current-context: %s
contexts:
  - name: %s
    context:
      cluster: %s
      user: %s-user
clusters:
  - name: %s
    cluster:
      server: https://kubernetes.local:6443
users:
  - name: %s-user
    user:
      token: test-token
`, contextName, contextName, contextName, contextName, contextName, contextName)

	if err := os.WriteFile(kubeconfigPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	return kubeconfigPath
}

// setupMultipleContextsKubeconfig creates a kubeconfig file with multiple contexts.
// Set currentContext to empty string to NOT set a current context (triggers context picker).
func setupMultipleContextsKubeconfig(t *testing.T, workspace string, contexts []string) string {
	return setupMultipleContextsKubeconfigWithCurrent(t, workspace, contexts, contexts[0])
}

// setupMultipleContextsKubeconfigNoCurrent creates a kubeconfig with multiple contexts
// but NO current-context set. This forces the context picker to appear.
func setupMultipleContextsKubeconfigNoCurrent(t *testing.T, workspace string, contexts []string) string {
	return setupMultipleContextsKubeconfigWithCurrent(t, workspace, contexts, "")
}

// setupMultipleContextsKubeconfigWithCurrent creates a kubeconfig with multiple contexts
// and optionally sets the current context.
func setupMultipleContextsKubeconfigWithCurrent(t *testing.T, workspace string, contexts []string, currentContext string) string {
	t.Helper()

	kubeDir := filepath.Join(workspace, ".kube")
	if err := os.MkdirAll(kubeDir, 0o755); err != nil {
		t.Fatalf("failed to create .kube dir: %v", err)
	}

	kubeconfigPath := filepath.Join(kubeDir, "config")

	var sb strings.Builder
	sb.WriteString("apiVersion: v1\n")
	sb.WriteString("kind: Config\n")
	if currentContext != "" {
		sb.WriteString(fmt.Sprintf("current-context: %s\n", currentContext))
	}
	sb.WriteString("contexts:\n")
	for _, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("  - name: %s\n", ctx))
		sb.WriteString(fmt.Sprintf("    context:\n"))
		sb.WriteString(fmt.Sprintf("      cluster: %s\n", ctx))
		sb.WriteString(fmt.Sprintf("      user: %s-user\n", ctx))
	}
	sb.WriteString("clusters:\n")
	for _, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("  - name: %s\n", ctx))
		sb.WriteString(fmt.Sprintf("    cluster:\n"))
		sb.WriteString(fmt.Sprintf("      server: https://%s.local:6443\n", ctx))
	}
	sb.WriteString("users:\n")
	for _, ctx := range contexts {
		sb.WriteString(fmt.Sprintf("  - name: %s-user\n", ctx))
		sb.WriteString(fmt.Sprintf("    user:\n"))
		sb.WriteString(fmt.Sprintf("      token: test-token-%s\n", ctx))
	}

	if err := os.WriteFile(kubeconfigPath, []byte(sb.String()), 0o644); err != nil {
		t.Fatalf("failed to write kubeconfig: %v", err)
	}

	return kubeconfigPath
}

// MockArgoServerWithResources creates a mock server with a richer resource tree for k9s tests.
// The resource tree includes Pod, Deployment, Service, ReplicaSet nodes.
func MockArgoServerWithResources() (*httptest.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/session/userinfo", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"metadata":{"name":"demo","namespace":"argocd"},"spec":{"project":"demo","destination":{"name":"cluster-a","namespace":"default"}},"status":{"sync":{"status":"Synced"},"health":{"status":"Healthy"}}}]}`))
	})
	mux.HandleFunc("/api/version", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"version":"e2e"}`))
	})
	mux.HandleFunc("/api/v1/applications/demo/resource-tree", func(w http.ResponseWriter, r *http.Request) {
		// Rich resource tree with multiple resource types
		_, _ = w.Write([]byte(`{"nodes":[
			{"kind":"Deployment","name":"demo-deploy","namespace":"default","version":"v1","group":"apps","uid":"dep-1","status":"Synced"},
			{"kind":"ReplicaSet","name":"demo-rs","namespace":"default","version":"v1","group":"apps","uid":"rs-1","status":"Synced","parentRefs":[{"uid":"dep-1","kind":"Deployment","name":"demo-deploy","namespace":"default","group":"apps","version":"v1"}]},
			{"kind":"Pod","name":"demo-pod-1","namespace":"default","version":"v1","group":"","uid":"pod-1","status":"Synced","parentRefs":[{"uid":"rs-1","kind":"ReplicaSet","name":"demo-rs","namespace":"default","group":"apps","version":"v1"}]},
			{"kind":"Service","name":"demo-svc","namespace":"default","version":"v1","group":"","uid":"svc-1","status":"Synced"}
		]}`))
	})
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
