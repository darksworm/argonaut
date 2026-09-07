//go:build e2e && unix

package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
)

// The horizontal suite drives the real TUI against a real Argo CD and then
// asserts on the server's own state. The vertical suite proves argonaut sends
// what we think it sends; only this one proves Argo CD understood it.
//
//	make argocd-up && make argocd-git-daemon
//	./argocd/fixtures/seed-sync-fixtures.sh
//	make real-e2e
//
// Skipped unless ARGONAUT_REAL_ARGOCD=1, so it stays compiled — and therefore
// honest — without running in CI.

const (
	realArgoEnv = "ARGONAUT_REAL_ARGOCD"
	// Real syncs are not instant the way the mock is.
	realTimeout = 30 * time.Second
)

type realArgo struct {
	baseURL  string
	token    string
	insecure bool
	client   *http.Client
}

// connectRealArgo reads the same CLI config the app reads, so the test
// authenticates exactly the way a user does.
func connectRealArgo(t *testing.T) *realArgo {
	t.Helper()
	if os.Getenv(realArgoEnv) != "1" {
		t.Skipf("horizontal e2e: set %s=1 with a local Argo CD running (see make argocd-up)", realArgoEnv)
	}

	cfg, err := config.ReadCLIConfig()
	if err != nil {
		t.Fatalf("reading the argocd CLI config: %v (run 'make argocd-login')", err)
	}
	server, err := cfg.ToServerConfig()
	if err != nil {
		t.Fatalf("resolving the current argocd context: %v", err)
	}
	if server.Token == "" {
		t.Fatal("no auth token in the argocd CLI config — run 'make argocd-login'")
	}

	r, err := newRealArgo(server)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func newRealArgo(server *model.Server) (*realArgo, error) {
	target, err := url.Parse(server.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid local Argo CD URL: %w", err)
	}
	ip := net.ParseIP(target.Hostname())
	if (target.Scheme != "http" && target.Scheme != "https") || target.User != nil ||
		(target.Hostname() != "localhost" && (ip == nil || !ip.IsLoopback())) {
		return nil, fmt.Errorf("real Argo CD tests require a loopback HTTP(S) endpoint")
	}

	return &realArgo{
		baseURL:  server.BaseURL,
		insecure: server.Insecure,
		token:    server.Token,
		client: &http.Client{
			Timeout: 10 * time.Second,
			// A redirect must not escape the validated local endpoint.
			CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: server.Insecure}, //nolint:gosec // explicit CLI setting, loopback only
			},
		},
	}, nil
}

// app fetches an application straight from the Argo CD API. Assertions read
// this, never the screen, so a failure names the server state that was wrong.
func (r *realArgo) app(t *testing.T, name string) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, r.baseURL+"/api/v1/applications/"+name, nil)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.token)

	resp, err := r.client.Do(req)
	if err != nil {
		t.Fatalf("querying Argo CD for %q: %v", name, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Argo CD returned %d for %q: %s", resp.StatusCode, name, body)
	}

	var out map[string]any
	if err := json.Unmarshal(body, &out); err != nil {
		t.Fatalf("decoding the application: %v", err)
	}
	return out
}

// dig walks a decoded JSON object, returning nil at the first missing key
// rather than panicking, so a failure reports the path it wanted.
func dig(obj map[string]any, path ...string) any {
	var current any = obj
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = m[key]
		if !ok {
			return nil
		}
	}
	return current
}

func digString(obj map[string]any, path ...string) string {
	s, _ := dig(obj, path...).(string)
	return s
}

// mark waits for the next whole second before the caller drives the TUI.
// Argo CD records startedAt at one-second resolution: rounding down would
// allow an old operation from the current second to satisfy the wait.
// The new boundary is inclusive so an operation started immediately after
// mark returns is accepted. This assumes the local cluster clock is aligned
// with the test host; the fixtures must not be driven concurrently.
func mark() time.Time { return markWithClock(time.Now, time.Sleep) }

func markWithClock(now func() time.Time, sleep func(time.Duration)) time.Time {
	next := now().UTC().Truncate(time.Second).Add(time.Second)
	// This wait deliberately crosses the server timestamp's precision boundary.
	for current := now(); current.Before(next); current = now() {
		sleep(next.Sub(current))
	}
	return next
}

// waitForOperationAfter polls until an operation that started after `since`
// reaches a terminal phase, and returns the app at that point.
func (r *realArgo) waitForOperationAfter(t *testing.T, name string, since time.Time) map[string]any {
	t.Helper()
	deadline := time.Now().Add(realTimeout)
	var lastPhase, lastStart string
	for time.Now().Before(deadline) {
		app := r.app(t, name)
		started := digString(app, "status", "operationState", "startedAt")
		phase := digString(app, "status", "operationState", "phase")
		lastPhase, lastStart = phase, started

		startedAt, err := time.Parse(time.RFC3339, started)
		if err == nil && !startedAt.Before(since) {
			switch phase {
			case "Succeeded", "Failed", "Error":
				t.Logf("operation on %q started %s, phase %q, sync=%v",
					name, started, phase,
					dig(app, "status", "operationState", "operation", "sync"))
				return app
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("no operation on %q started after %s and finished within %s (latest started %q, phase %q)",
		name, since.Format(time.RFC3339), realTimeout, lastStart, lastPhase)
	return nil
}

// startAgainstRealArgo boots the TUI pointed at the live server. The driver
// itself is server-agnostic; only the 2s request timeout it writes by default
// is too tight for a real one.
func startAgainstRealArgo(t *testing.T, r *realArgo) *TUITestFramework {
	t.Helper()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)
	tf.requestTimeout = "15s"

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := writeArgoConfigWithTLS(cfgPath, r.baseURL, r.token, r.insecure); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}
	return tf
}

// openCommand retries the command bar. Against a real server the first
// keystrokes can land while the initial application load still owns the
// screen, and a swallowed ":" is not a failure worth failing a test over.
func openCommand(t *testing.T, tf *TUITestFramework) {
	t.Helper()
	var err error
	for attempt := 0; attempt < 5; attempt++ {
		if err = tf.OpenCommand(); err == nil {
			return
		}
		time.Sleep(time.Second)
	}
	t.Fatalf("command bar never opened: %v\n%s", err, tf.Screen())
}

// openSyncModalFor navigates to the named app and opens its sync confirmation.
func openSyncModalFor(t *testing.T, tf *TUITestFramework, app string) {
	t.Helper()
	// The TUI opens on the clusters view; apps are a level down.
	if !tf.WaitForPlain("NAME", realTimeout) {
		t.Fatalf("the TUI never connected:\n%s", tf.Screen())
	}
	openCommand(t, tf)
	_ = tf.Send("apps")
	_ = tf.Enter()

	// Filter down to the one app: a real cluster has more than fit on screen.
	if err := tf.OpenSearch(); err != nil {
		t.Fatalf("open search: %v\n%s", err, tf.Screen())
	}
	_ = tf.Send(app)
	_ = tf.Enter()
	if !tf.WaitForPlain(app, realTimeout) {
		t.Fatalf("app %q never appeared in the TUI:\n%s", app, tf.Screen())
	}
	openCommand(t, tf)
	_ = tf.Send("sync " + app)
	_ = tf.Enter()
	if !tf.WaitForScreen("Sync", 10*time.Second) {
		t.Fatalf("sync modal never opened:\n%s", tf.Screen())
	}
}

func TestRealArgoCD_DryRunSyncAppliesNothing(t *testing.T) {
	r := connectRealArgo(t)
	const app = "schema-error-demo"

	before := r.app(t, app)
	beforeSync := digString(before, "status", "sync", "status")
	since := mark()

	tf := startAgainstRealArgo(t, r)
	openSyncModalFor(t, tf, app)
	_ = tf.Send("d") // dry run
	_ = tf.Send("y") // confirm

	after := r.waitForOperationAfter(t, app, since)

	if dryRun, _ := dig(after, "status", "operationState", "operation", "sync", "dryRun").(bool); !dryRun {
		t.Errorf("expected operation.sync.dryRun=true, got %v", dig(after, "status", "operationState", "operation", "sync"))
	}
	if got := digString(after, "status", "sync", "status"); got != beforeSync {
		t.Errorf("a dry run changed the app's sync status: %q became %q", beforeSync, got)
	}

	// The pane label is the one deliverable only the screen can confirm;
	// everything else here is asserted against the server.
	if !tf.WaitForScreen("Sync (dry run)", realTimeout) {
		t.Errorf("expected the sync status pane to mark the dry run:\n%s", tf.Screen())
	}
}

func TestRealArgoCD_ForceSyncKeepsTheHookStrategy(t *testing.T) {
	r := connectRealArgo(t)
	const app = "prune-demo"
	since := mark()

	tf := startAgainstRealArgo(t, r)
	openSyncModalFor(t, tf, app)
	_ = tf.Send("f") // force
	_ = tf.Send("y") // confirm the sync
	if !tf.WaitForScreen("Force sync", 10*time.Second) {
		t.Fatalf("force confirmation never appeared:\n%s", tf.Screen())
	}
	_ = tf.Send("y") // confirm the force

	after := r.waitForOperationAfter(t, app, since)

	strategy, _ := dig(after, "status", "operationState", "operation", "sync", "syncStrategy").(map[string]any)
	if _, applyOnly := strategy["apply"]; applyOnly {
		t.Errorf("a forced sync used the apply strategy, which skips sync hooks: %v", strategy)
	}
	hook, ok := strategy["hook"].(map[string]any)
	if !ok {
		t.Fatalf("expected syncStrategy.hook, got %v", strategy)
	}
	if force, _ := hook["force"].(bool); !force {
		t.Errorf("expected hook.force=true, got %v", hook)
	}
}

func init() {
	// Surface the target so a failing run says which server it hit.
	if os.Getenv(realArgoEnv) == "1" {
		if cfg, err := config.ReadCLIConfig(); err == nil {
			if server, err := cfg.ToServerConfig(); err == nil {
				fmt.Printf("horizontal e2e target: %s\n", server.BaseURL)
			}
		}
	}
}

func TestRealArgoCD_ResourceSyncSendsOnlyThatResource(t *testing.T) {
	r := connectRealArgo(t)
	const app = "prune-demo"
	since := mark()

	tf := startAgainstRealArgo(t, r)
	if !tf.WaitForPlain("NAME", realTimeout) {
		t.Fatalf("the TUI never connected:\n%s", tf.Screen())
	}
	openCommand(t, tf)
	_ = tf.Send("resources " + app)
	_ = tf.Enter()
	if !tf.WaitForScreen("keep-me", realTimeout) {
		t.Fatalf("resource tree never loaded:\n%s", tf.Screen())
	}

	// Select the one ConfigMap under the application root, then sync it.
	_ = tf.Send("j")
	_ = tf.Send(" ")
	_ = tf.Send("s")
	if !tf.WaitForScreen("Sync", 10*time.Second) {
		t.Fatalf("resource sync modal never opened:\n%s", tf.Screen())
	}
	_ = tf.Send("y")

	after := r.waitForOperationAfter(t, app, since)

	resources, _ := dig(after, "status", "operationState", "operation", "sync", "resources").([]any)
	if len(resources) != 1 {
		t.Fatalf("expected the sync scoped to one resource, got %v",
			dig(after, "status", "operationState", "operation", "sync", "resources"))
	}
	if name, _ := resources[0].(map[string]any)["name"].(string); name != "keep-me" {
		t.Errorf("expected the selected resource in the request, got %v", resources[0])
	}
}
