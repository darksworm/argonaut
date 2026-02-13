package main

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
)

func buildTestApps() []model.App {
	cluster := "production"
	ns := "default"
	proj := "my-project"
	appset := "my-appset"
	return []model.App{
		{
			Name:           "app-1",
			ClusterLabel:   &cluster,
			Namespace:      &ns,
			Project:        &proj,
			ApplicationSet: &appset,
			Health:         "Healthy",
			Sync:           "Synced",
		},
	}
}

func TestDefaultViewWarning_MalformedConfig(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "foobar"

	m := NewModel(cfg)
	m.state.Mode = model.ModeLoading

	// Warning should already be set from parsing
	if m.state.Modals.DefaultViewWarning == nil {
		t.Fatal("expected DefaultViewWarning to be set for malformed config")
	}
	if !strings.Contains(*m.state.Modals.DefaultViewWarning, "foobar") {
		t.Errorf("warning should mention the invalid value, got: %s", *m.state.Modals.DefaultViewWarning)
	}

	// View should remain at default (clusters) since the parse failed
	if m.state.Navigation.View != model.ViewClusters {
		t.Errorf("expected default view clusters, got %s", m.state.Navigation.View)
	}

	// No pending scope to validate
	if m.pendingDefaultViewScope != nil {
		t.Error("expected no pending scope for malformed config")
	}
}

func TestDefaultViewWarning_ValidConfigNoPending(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "apps"

	m := NewModel(cfg)

	// No warning for valid config
	if m.state.Modals.DefaultViewWarning != nil {
		t.Errorf("unexpected warning for valid config: %s", *m.state.Modals.DefaultViewWarning)
	}

	// View should be set to apps
	if m.state.Navigation.View != model.ViewApps {
		t.Errorf("expected apps view, got %s", m.state.Navigation.View)
	}

	// No scope → no pending validation
	if m.pendingDefaultViewScope != nil {
		t.Error("expected no pending scope for view-only config")
	}
}

func TestDefaultViewWarning_ScopeEntityExists(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "cluster production"

	m := NewModel(cfg)

	// Should have pending scope
	if m.pendingDefaultViewScope == nil {
		t.Fatal("expected pending scope for scoped config")
	}
	if m.pendingDefaultViewScope.scopeType != "cluster" {
		t.Errorf("expected scope type 'cluster', got %s", m.pendingDefaultViewScope.scopeType)
	}
	if m.pendingDefaultViewScope.scopeValue != "production" {
		t.Errorf("expected scope value 'production', got %s", m.pendingDefaultViewScope.scopeValue)
	}

	// Simulate apps loaded — entity exists
	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	// No warning — entity found
	if m.state.Modals.DefaultViewWarning != nil {
		t.Errorf("unexpected warning when entity exists: %s", *m.state.Modals.DefaultViewWarning)
	}

	// Pending scope should be consumed
	if m.pendingDefaultViewScope != nil {
		t.Error("pending scope should be nil after validation")
	}

	// Navigation should remain as set (namespaces scoped to cluster)
	if m.state.Navigation.View != model.ViewNamespaces {
		t.Errorf("expected namespaces view, got %s", m.state.Navigation.View)
	}
}

func TestDefaultViewWarning_ScopeClusterNotFound(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "cluster nonexistent"

	m := NewModel(cfg)

	// Simulate apps loaded — entity does NOT exist
	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	// Warning should be set
	if m.state.Modals.DefaultViewWarning == nil {
		t.Fatal("expected warning when cluster not found")
	}
	if !strings.Contains(*m.state.Modals.DefaultViewWarning, "nonexistent") {
		t.Errorf("warning should mention the missing entity, got: %s", *m.state.Modals.DefaultViewWarning)
	}
	if !strings.Contains(*m.state.Modals.DefaultViewWarning, "Cluster") {
		t.Errorf("warning should mention the entity type, got: %s", *m.state.Modals.DefaultViewWarning)
	}

	// Should fall back to default view
	if m.state.Navigation.View != model.ViewClusters {
		t.Errorf("expected fallback to clusters view, got %s", m.state.Navigation.View)
	}

	// Scope selections should be cleared
	if len(m.state.Selections.ScopeClusters) != 0 {
		t.Errorf("expected scope clusters to be cleared, got %v", m.state.Selections.ScopeClusters)
	}
}

func TestDefaultViewWarning_ScopeNamespaceNotFound(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "ns nonexistent-ns"

	m := NewModel(cfg)

	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	if m.state.Modals.DefaultViewWarning == nil {
		t.Fatal("expected warning when namespace not found")
	}
	if !strings.Contains(*m.state.Modals.DefaultViewWarning, "Namespace") {
		t.Errorf("warning should mention 'Namespace', got: %s", *m.state.Modals.DefaultViewWarning)
	}
	if m.state.Navigation.View != model.ViewClusters {
		t.Errorf("expected fallback to clusters view, got %s", m.state.Navigation.View)
	}
}

func TestDefaultViewWarning_ScopeProjectNotFound(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "project nonexistent-proj"

	m := NewModel(cfg)

	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	if m.state.Modals.DefaultViewWarning == nil {
		t.Fatal("expected warning when project not found")
	}
	if !strings.Contains(*m.state.Modals.DefaultViewWarning, "Project") {
		t.Errorf("warning should mention 'Project', got: %s", *m.state.Modals.DefaultViewWarning)
	}
	if m.state.Navigation.View != model.ViewClusters {
		t.Fatalf("expected fallback to ViewClusters, got: %v", m.state.Navigation.View)
	}
}

func TestDefaultViewWarning_ScopeAppsetNotFound(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "appset nonexistent-set"

	m := NewModel(cfg)

	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	if m.state.Modals.DefaultViewWarning == nil {
		t.Fatal("expected warning when appset not found")
	}
	if !strings.Contains(*m.state.Modals.DefaultViewWarning, "ApplicationSet") {
		t.Errorf("warning should mention 'ApplicationSet', got: %s", *m.state.Modals.DefaultViewWarning)
	}
	if m.state.Navigation.View != model.ViewClusters {
		t.Fatalf("expected fallback to ViewClusters, got: %v", m.state.Navigation.View)
	}
}

func TestDefaultViewWarning_NilIndexPreservesPendingScope(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "cluster production"

	m := NewModel(cfg)

	// Pending scope should be set
	if m.pendingDefaultViewScope == nil {
		t.Fatal("expected pending scope")
	}

	// Call validate WITHOUT building the index — scope must be preserved
	m.validateDefaultViewScope()

	if m.pendingDefaultViewScope == nil {
		t.Fatal("pendingDefaultViewScope was consumed despite nil index")
	}

	// Now build index and validate — scope should be consumed successfully
	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	if m.pendingDefaultViewScope != nil {
		t.Error("pendingDefaultViewScope should be nil after successful validation")
	}
	if m.state.Modals.DefaultViewWarning != nil {
		t.Errorf("unexpected warning when entity exists: %s", *m.state.Modals.DefaultViewWarning)
	}
}

func TestDefaultViewWarning_NoValidationWithoutPendingScope(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = "apps"

	m := NewModel(cfg)

	// No pending scope
	m.state.Apps = buildTestApps()
	m.state.Index = model.BuildAppIndex(m.state.Apps)
	m.validateDefaultViewScope()

	// Should stay clean
	if m.state.Modals.DefaultViewWarning != nil {
		t.Errorf("unexpected warning when no scope is pending: %s", *m.state.Modals.DefaultViewWarning)
	}
	if m.state.Navigation.View != model.ViewApps {
		t.Errorf("expected apps view to be preserved, got %s", m.state.Navigation.View)
	}
}

func TestDefaultViewWarning_EmptyConfig(t *testing.T) {
	cfg := config.GetDefaultConfig()
	cfg.DefaultView = ""

	m := NewModel(cfg)

	if m.state.Modals.DefaultViewWarning != nil {
		t.Errorf("unexpected warning for empty config: %s", *m.state.Modals.DefaultViewWarning)
	}
	if m.pendingDefaultViewScope != nil {
		t.Error("unexpected pending scope for empty config")
	}
	// Default view should remain clusters
	if m.state.Navigation.View != model.ViewClusters {
		t.Errorf("expected clusters view for empty config, got %s", m.state.Navigation.View)
	}
}
