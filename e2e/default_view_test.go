//go:build e2e && unix

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultViewApps(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Set default_view to apps — app should start in apps view instead of clusters
	tf.extraConfig = `default_view = "apps"`

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// In apps view, we should see the app name "demo" with sync/health status
	if !tf.WaitForPlain("demo", 5*time.Second) {
		t.Fatal("expected app name 'demo' in apps view")
	}
	if !tf.WaitForPlain("Synced", 3*time.Second) {
		t.Fatal("expected 'Synced' status in apps view")
	}

	// Verify we're in apps view by checking the status line breadcrumb.
	// The status line shows "<apps>" in apps view and "<clusters>" in clusters view.
	if !tf.WaitForPlain("<apps>", 3*time.Second) {
		screen := tf.Screen()
		t.Fatalf("expected '<apps>' breadcrumb in status line, got:\n%s", screen)
	}
}

func TestDefaultViewWithScope(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Set default_view to scope to a cluster — should show namespaces view
	tf.extraConfig = `default_view = "cluster cluster-a"`

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// In namespaces view scoped to cluster-a, we should see "default"
	// namespace. Assert on the `<namespaces>` breadcrumb — the substring
	// "default" appears in the REST JSON payload regardless of view, so
	// `WaitForPlain("default", ...)` would pass even when `default_view`
	// is ignored entirely.
	if !tf.WaitForScreen("<namespaces>", 5*time.Second) {
		t.Log(tf.Screen())
		t.Fatal("expected `<namespaces>` breadcrumb when default_view scopes to a cluster")
	}
	if !strings.Contains(tf.Screen(), "default") {
		t.Log(tf.Screen())
		t.Fatal("expected `default` namespace row in the namespaces view")
	}
}

func TestDefaultViewMigrationPinsClustersForExistingUser(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// An existing user: their config predates the apps default and has no default_view
	tf.extraConfig = `last_seen_version = "2.17.0"`

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	// Their startup view must not change on upgrade
	if !tf.WaitForPlain("<clusters>", 5*time.Second) {
		t.Fatalf("expected '<clusters>' breadcrumb for migrated user, got:\n%s", tf.Screen())
	}

	// The pin was written back to their config with an explanation above it
	argonautConfig := filepath.Join(tf.workspace, ".config", "argonaut", "config.toml")
	data, err := os.ReadFile(argonautConfig)
	if err != nil {
		t.Fatalf("read argonaut config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, `default_view = 'clusters'`) {
		t.Errorf("expected pinned default_view in config, got:\n%s", content)
	}
	if !strings.Contains(content, "apps view by default") {
		t.Errorf("expected explanatory comment above the pin, got:\n%s", content)
	}
}

func TestDefaultViewMigrationSurvivesRealVersionedUpgrade(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// A real upgrade: released (non-dev) binary and an old last_seen_version.
	// Unlike dev builds, this also triggers the what's-new full config rewrite,
	// which the pin and its explanatory comment must survive.
	tf.binOverride = versionedBinPath
	tf.extraConfig = `last_seen_version = "2.17.0"`

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}

	if !tf.WaitForPlain("<clusters>", 5*time.Second) {
		t.Fatalf("expected '<clusters>' breadcrumb for upgraded user, got:\n%s", tf.Screen())
	}

	argonautConfig := filepath.Join(tf.workspace, ".config", "argonaut", "config.toml")
	data, err := os.ReadFile(argonautConfig)
	if err != nil {
		t.Fatalf("read argonaut config: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "last_seen_version = '"+releasedTestVersion+"'") {
		t.Errorf("expected the upgrade to record the new version, got:\n%s", content)
	}
	if !strings.Contains(content, `default_view = 'clusters'`) {
		t.Errorf("expected pinned default_view in config, got:\n%s", content)
	}
	if !strings.Contains(content, "apps view by default") {
		t.Errorf("expected the explanatory comment to survive the what's-new rewrite, got:\n%s", content)
	}
}

func TestFreshInstallDefaultsToAppsAcrossLaunches(t *testing.T) {
	t.Parallel()
	tf := NewTUITest(t)
	t.Cleanup(tf.Cleanup)

	srv, err := MockArgoServer()
	if err != nil {
		t.Fatalf("mock server: %v", err)
	}
	t.Cleanup(srv.Close)

	cfgPath, err := tf.SetupWorkspace()
	if err != nil {
		t.Fatalf("setup workspace: %v", err)
	}
	if err := WriteArgoConfig(cfgPath, srv.URL); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// A fresh install of a released binary: no Argonaut config file at all.
	tf.binOverride = versionedBinPath
	tf.skipArgonautConfig = true

	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("start app: %v", err)
	}
	if !tf.WaitForPlain("<apps>", 5*time.Second) {
		t.Fatalf("expected '<apps>' breadcrumb on fresh install, got:\n%s", tf.Screen())
	}

	// The first launch records the version but must not pin a default_view
	argonautConfig := filepath.Join(tf.workspace, ".config", "argonaut", "config.toml")
	data, err := os.ReadFile(argonautConfig)
	if err != nil {
		t.Fatalf("read argonaut config after first launch: %v", err)
	}
	if !strings.Contains(string(data), "last_seen_version = '"+releasedTestVersion+"'") {
		t.Errorf("expected first launch to record the version, got:\n%s", data)
	}
	if strings.Contains(string(data), "default_view") {
		t.Errorf("fresh install must not get a pinned default_view, got:\n%s", data)
	}

	// The second launch (config now exists) must not be mistaken for an
	// upgrade from the clusters-default era
	tf.StopApp()
	if err := tf.StartAppArgs([]string{"-argocd-config=" + cfgPath}); err != nil {
		t.Fatalf("restart app: %v", err)
	}
	if !tf.WaitForScreen("<apps>", 5*time.Second) {
		t.Fatalf("expected '<apps>' breadcrumb on second launch, got:\n%s", tf.Screen())
	}
	data, err = os.ReadFile(argonautConfig)
	if err != nil {
		t.Fatalf("read argonaut config after second launch: %v", err)
	}
	if strings.Contains(string(data), "default_view") {
		t.Errorf("second launch must not pin a default_view, got:\n%s", data)
	}
}
