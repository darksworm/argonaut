package model

import (
	"reflect"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestBuildAppIndex_Empty(t *testing.T) {
	idx := BuildAppIndex(nil)
	if idx.Total != 0 {
		t.Errorf("expected Total 0, got %d", idx.Total)
	}
	if len(idx.Clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(idx.Clusters))
	}
	if len(idx.NameToIndex) != 0 {
		t.Errorf("expected empty NameToIndex, got %d entries", len(idx.NameToIndex))
	}
}

func TestBuildAppIndex_SingleApp(t *testing.T) {
	apps := []App{
		{Name: "web", ClusterLabel: strPtr("prod"), Namespace: strPtr("default"), Project: strPtr("infra"), ApplicationSet: strPtr("web-set")},
	}
	idx := BuildAppIndex(apps)

	if idx.Total != 1 {
		t.Fatalf("expected Total 1, got %d", idx.Total)
	}
	if !reflect.DeepEqual(idx.Clusters, []string{"prod"}) {
		t.Errorf("Clusters = %v, want [prod]", idx.Clusters)
	}
	if !reflect.DeepEqual(idx.Namespaces, []string{"default"}) {
		t.Errorf("Namespaces = %v, want [default]", idx.Namespaces)
	}
	if !reflect.DeepEqual(idx.Projects, []string{"infra"}) {
		t.Errorf("Projects = %v, want [infra]", idx.Projects)
	}
	if !reflect.DeepEqual(idx.ApplicationSets, []string{"web-set"}) {
		t.Errorf("ApplicationSets = %v, want [web-set]", idx.ApplicationSets)
	}
	if i, ok := idx.NameToIndex["web"]; !ok || i != 0 {
		t.Errorf("NameToIndex[web] = %d, %v; want 0, true", i, ok)
	}
}

func TestBuildAppIndex_NilFields(t *testing.T) {
	apps := []App{
		{Name: "bare"},
	}
	idx := BuildAppIndex(apps)

	if idx.Total != 1 {
		t.Fatalf("expected Total 1, got %d", idx.Total)
	}
	if len(idx.Clusters) != 0 {
		t.Errorf("expected 0 clusters for nil ClusterLabel, got %v", idx.Clusters)
	}
	if len(idx.Namespaces) != 0 {
		t.Errorf("expected 0 namespaces for nil Namespace, got %v", idx.Namespaces)
	}
	if len(idx.Projects) != 0 {
		t.Errorf("expected 0 projects for nil Project, got %v", idx.Projects)
	}
	if len(idx.ApplicationSets) != 0 {
		t.Errorf("expected 0 appsets for nil ApplicationSet, got %v", idx.ApplicationSets)
	}
	if _, ok := idx.NameToIndex["bare"]; !ok {
		t.Error("expected NameToIndex to contain 'bare'")
	}
}

func TestBuildAppIndex_MultipleApps_SortedUnique(t *testing.T) {
	apps := []App{
		{Name: "c-app", ClusterLabel: strPtr("prod"), Namespace: strPtr("ns-b"), Project: strPtr("proj-2")},
		{Name: "a-app", ClusterLabel: strPtr("staging"), Namespace: strPtr("ns-a"), Project: strPtr("proj-1")},
		{Name: "b-app", ClusterLabel: strPtr("prod"), Namespace: strPtr("ns-b"), Project: strPtr("proj-1")},
	}
	idx := BuildAppIndex(apps)

	if idx.Total != 3 {
		t.Fatalf("expected Total 3, got %d", idx.Total)
	}
	// Clusters sorted
	if !reflect.DeepEqual(idx.Clusters, []string{"prod", "staging"}) {
		t.Errorf("Clusters = %v, want [prod staging]", idx.Clusters)
	}
	// Namespaces sorted (deduplicated)
	if !reflect.DeepEqual(idx.Namespaces, []string{"ns-a", "ns-b"}) {
		t.Errorf("Namespaces = %v, want [ns-a ns-b]", idx.Namespaces)
	}
	// Projects sorted (deduplicated)
	if !reflect.DeepEqual(idx.Projects, []string{"proj-1", "proj-2"}) {
		t.Errorf("Projects = %v, want [proj-1 proj-2]", idx.Projects)
	}
	// ByCluster
	if !reflect.DeepEqual(idx.ByCluster["prod"], []int{0, 2}) {
		t.Errorf("ByCluster[prod] = %v, want [0 2]", idx.ByCluster["prod"])
	}
	if !reflect.DeepEqual(idx.ByCluster["staging"], []int{1}) {
		t.Errorf("ByCluster[staging] = %v, want [1]", idx.ByCluster["staging"])
	}
	// NameToIndex
	for i, app := range apps {
		if idx.NameToIndex[app.Name] != i {
			t.Errorf("NameToIndex[%s] = %d, want %d", app.Name, idx.NameToIndex[app.Name], i)
		}
	}
}

func TestBuildAppIndex_EmptyStringFields(t *testing.T) {
	// Empty strings for optional fields should not be indexed
	empty := ""
	apps := []App{
		{Name: "test", ClusterLabel: &empty, Namespace: &empty, Project: &empty, ApplicationSet: &empty},
	}
	idx := BuildAppIndex(apps)

	if len(idx.Clusters) != 0 {
		t.Errorf("expected 0 clusters for empty string, got %v", idx.Clusters)
	}
	if len(idx.Namespaces) != 0 {
		t.Errorf("expected 0 namespaces for empty string, got %v", idx.Namespaces)
	}
}

func TestScopedNamespaces_NoScope(t *testing.T) {
	apps := []App{
		{Name: "a", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns-b")},
		{Name: "b", ClusterLabel: strPtr("c2"), Namespace: strPtr("ns-a")},
	}
	idx := BuildAppIndex(apps)

	result := idx.ScopedNamespaces(apps, nil)
	if !reflect.DeepEqual(result, []string{"ns-a", "ns-b"}) {
		t.Errorf("ScopedNamespaces(nil) = %v, want [ns-a ns-b]", result)
	}
}

func TestScopedNamespaces_WithScope(t *testing.T) {
	apps := []App{
		{Name: "a", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns-shared")},
		{Name: "b", ClusterLabel: strPtr("c2"), Namespace: strPtr("ns-only-c2")},
		{Name: "c", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns-only-c1")},
	}
	idx := BuildAppIndex(apps)

	scope := map[string]bool{"c1": true}
	result := idx.ScopedNamespaces(apps, scope)
	if !reflect.DeepEqual(result, []string{"ns-only-c1", "ns-shared"}) {
		t.Errorf("ScopedNamespaces(c1) = %v, want [ns-only-c1 ns-shared]", result)
	}
}

func TestScopedProjects_NoScope(t *testing.T) {
	apps := []App{
		{Name: "a", Project: strPtr("p2")},
		{Name: "b", Project: strPtr("p1")},
	}
	idx := BuildAppIndex(apps)

	result := idx.ScopedProjects(apps, nil, nil)
	if !reflect.DeepEqual(result, []string{"p1", "p2"}) {
		t.Errorf("ScopedProjects(nil, nil) = %v, want [p1 p2]", result)
	}
}

func TestScopedProjects_WithScope(t *testing.T) {
	apps := []App{
		{Name: "a", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns1"), Project: strPtr("p1")},
		{Name: "b", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns2"), Project: strPtr("p2")},
		{Name: "c", ClusterLabel: strPtr("c2"), Namespace: strPtr("ns1"), Project: strPtr("p3")},
	}
	idx := BuildAppIndex(apps)

	// Scope to cluster c1 only
	result := idx.ScopedProjects(apps, map[string]bool{"c1": true}, nil)
	if !reflect.DeepEqual(result, []string{"p1", "p2"}) {
		t.Errorf("ScopedProjects(c1, nil) = %v, want [p1 p2]", result)
	}

	// Scope to cluster c1 + namespace ns1
	result = idx.ScopedProjects(apps, map[string]bool{"c1": true}, map[string]bool{"ns1": true})
	if !reflect.DeepEqual(result, []string{"p1"}) {
		t.Errorf("ScopedProjects(c1, ns1) = %v, want [p1]", result)
	}
}

func TestScopedApps_NoScope(t *testing.T) {
	apps := []App{
		{Name: "a"},
		{Name: "b"},
	}
	idx := BuildAppIndex(apps)
	sel := NewSelectionState()

	result := idx.ScopedApps(apps, sel)
	if len(result) != 2 {
		t.Errorf("expected 2 apps with no scope, got %d", len(result))
	}
}

func TestScopedApps_WithScopes(t *testing.T) {
	apps := []App{
		{Name: "a", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns1"), Project: strPtr("p1")},
		{Name: "b", ClusterLabel: strPtr("c1"), Namespace: strPtr("ns2"), Project: strPtr("p1")},
		{Name: "c", ClusterLabel: strPtr("c2"), Namespace: strPtr("ns1"), Project: strPtr("p2")},
	}
	idx := BuildAppIndex(apps)

	sel := NewSelectionState()
	sel.ScopeClusters = map[string]bool{"c1": true}
	result := idx.ScopedApps(apps, sel)
	if len(result) != 2 || result[0].Name != "a" || result[1].Name != "b" {
		names := make([]string, len(result))
		for i, a := range result {
			names[i] = a.Name
		}
		t.Errorf("ScopedApps(c1) = %v, want [a b]", names)
	}

	// Add namespace scope
	sel.ScopeNamespaces = map[string]bool{"ns1": true}
	result = idx.ScopedApps(apps, sel)
	if len(result) != 1 || result[0].Name != "a" {
		t.Errorf("ScopedApps(c1+ns1) got %d apps, want 1 (a)", len(result))
	}
}

func TestScopedApps_ApplicationSetScope(t *testing.T) {
	apps := []App{
		{Name: "a", ApplicationSet: strPtr("set1")},
		{Name: "b", ApplicationSet: strPtr("set2")},
		{Name: "c"}, // no appset
	}
	idx := BuildAppIndex(apps)

	sel := NewSelectionState()
	sel.ScopeApplicationSets = map[string]bool{"set1": true}
	result := idx.ScopedApps(apps, sel)
	if len(result) != 1 || result[0].Name != "a" {
		t.Errorf("ScopedApps(set1) got %d apps, want 1 (a)", len(result))
	}
}

func TestScopedNamespaces_NilIndex(t *testing.T) {
	var idx *AppIndex
	result := idx.ScopedNamespaces(nil, nil)
	if result != nil {
		t.Errorf("expected nil from nil index, got %v", result)
	}
}

func TestScopedApps_NilIndex(t *testing.T) {
	var idx *AppIndex
	apps := []App{{Name: "a"}}
	sel := NewSelectionState()
	result := idx.ScopedApps(apps, sel)
	if len(result) != 1 {
		t.Errorf("expected passthrough from nil index, got %d apps", len(result))
	}
}
