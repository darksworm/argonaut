package treeview

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
	model "github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
)

// TestRenderStatusPart verifies the status display logic:
// - Health only → "(Healthy)"
// - Sync only → "(Synced)"
// - Both different → "(Healthy, OutOfSync)"
// - Both same → just one "(Healthy)"
// - Neither → empty string
func TestRenderStatusPart(t *testing.T) {
	tests := []struct {
		name           string
		health         string
		sync           string
		wantHealth     bool   // expect health value in output
		wantSync       bool   // expect sync value in output
		wantBoth       bool   // expect comma-separated format
		wantEmpty      bool   // expect empty string
	}{
		{
			name:       "health only",
			health:     "Healthy",
			sync:       "",
			wantHealth: true,
		},
		{
			name:     "sync only",
			health:   "",
			sync:     "OutOfSync",
			wantSync: true,
		},
		{
			name:       "both different",
			health:     "Healthy",
			sync:       "OutOfSync",
			wantHealth: true,
			wantSync:   true,
			wantBoth:   true,
		},
		{
			name:       "both same (Healthy/Healthy)",
			health:     "Healthy",
			sync:       "Healthy",
			wantHealth: true,
			wantSync:   false, // should not duplicate
		},
		{
			name:       "both same case-insensitive",
			health:     "Synced",
			sync:       "synced",
			wantHealth: true,
			wantSync:   false, // should not duplicate due to EqualFold
		},
		{
			name:      "neither present",
			health:    "",
			sync:      "",
			wantEmpty: true,
		},
		{
			name:       "degraded health with OutOfSync",
			health:     "Degraded",
			sync:       "OutOfSync",
			wantHealth: true,
			wantSync:   true,
			wantBoth:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewTreeView(100, 20)
			v.ApplyTheme(theme.Default())

			node := &treeNode{
				health: tt.health,
				status: tt.sync,
			}

			result := v.renderStatusPart(node)

			// Strip ANSI codes for easier testing
			plain := stripANSI(result)

			if tt.wantEmpty {
				if result != "" {
					t.Errorf("expected empty string, got %q", plain)
				}
				return
			}

			if tt.wantHealth && !strings.Contains(plain, tt.health) {
				t.Errorf("expected health %q in output %q", tt.health, plain)
			}

			if tt.wantSync && !strings.Contains(plain, tt.sync) {
				t.Errorf("expected sync %q in output %q", tt.sync, plain)
			}

			if tt.wantBoth {
				if !strings.Contains(plain, ",") {
					t.Errorf("expected comma separator for both statuses, got %q", plain)
				}
			}

			// Verify parentheses format
			if !strings.HasPrefix(plain, "(") || !strings.HasSuffix(plain, ")") {
				t.Errorf("expected parentheses wrapping, got %q", plain)
			}
		})
	}
}

// TestDiscriminatorArrow verifies the expand/collapse arrow logic:
// - Expanded with children → no arrow
// - Collapsed with children → "▸" arrow
// - No children → no arrow
func TestDiscriminatorArrow(t *testing.T) {
	tests := []struct {
		name        string
		hasChildren bool
		expanded    bool
		wantArrow   bool
	}{
		{
			name:        "expanded with children - no arrow",
			hasChildren: true,
			expanded:    true,
			wantArrow:   false,
		},
		{
			name:        "collapsed with children - show arrow",
			hasChildren: true,
			expanded:    false,
			wantArrow:   true,
		},
		{
			name:        "leaf node (no children) - no arrow",
			hasChildren: false,
			expanded:    false,
			wantArrow:   false,
		},
		{
			name:        "expanded leaf (edge case) - no arrow",
			hasChildren: false,
			expanded:    true,
			wantArrow:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := NewTreeView(100, 20)
			v.ApplyTheme(theme.Default())

			// Create root node
			root := &treeNode{
				uid:  "root",
				kind: "Application",
				name: "test-app",
			}

			if tt.hasChildren {
				child := &treeNode{
					uid:    "child",
					kind:   "Deployment",
					name:   "web",
					parent: root,
				}
				root.children = []*treeNode{child}
			}

			v.nodesByUID = map[string]*treeNode{"root": root}
			v.roots = []*treeNode{root}
			v.expanded = map[string]bool{"root": tt.expanded}
			v.rebuildOrder()

			output := v.Render()
			plain := stripANSI(output)

			hasArrow := strings.Contains(plain, "▸")

			if tt.wantArrow && !hasArrow {
				t.Errorf("expected arrow in output:\n%s", plain)
			}
			if !tt.wantArrow && hasArrow {
				t.Errorf("expected no arrow in output:\n%s", plain)
			}
		})
	}
}

// TestCollapsedNodeShowsCount verifies that collapsed nodes show "(+N)" count
func TestCollapsedNodeShowsCount(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	// Create two roots - first expanded, second collapsed
	// This way we can test non-selected collapsed node
	root1 := &treeNode{uid: "root1", kind: "Application", name: "app-a", health: "Healthy"}
	root2 := &treeNode{uid: "root2", kind: "Application", name: "app-b"}
	child1 := &treeNode{uid: "c1", kind: "Deployment", name: "web", parent: root2}
	child2 := &treeNode{uid: "c2", kind: "Service", name: "svc", parent: root2}
	root2.children = []*treeNode{child1, child2}

	v.nodesByUID = map[string]*treeNode{
		"root1": root1, "root2": root2, "c1": child1, "c2": child2,
	}
	v.roots = []*treeNode{root1, root2}
	v.expanded = map[string]bool{"root1": true, "root2": false} // second collapsed
	v.rebuildOrder()

	// Selection is on root1 (index 0), root2 is not selected
	output := v.Render()
	plain := stripANSI(output)

	if !strings.Contains(plain, "(+2)") {
		t.Errorf("expected collapsed count (+2) in output:\n%s", plain)
	}
}

// TestTreeViewRenderingOrder verifies DFS order and proper tree structure
func TestTreeViewRenderingOrder(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	// Build a small tree:
	// Application [app]
	// ├── Deployment [web]
	// └── Service [svc]
	root := &treeNode{uid: "root", kind: "Application", name: "app", health: "Healthy"}
	deploy := &treeNode{uid: "d1", kind: "Deployment", name: "web", namespace: "ns", parent: root, health: "Healthy"}
	svc := &treeNode{uid: "s1", kind: "Service", name: "svc", namespace: "ns", parent: root, health: "Healthy"}
	root.children = []*treeNode{deploy, svc}

	v.nodesByUID = map[string]*treeNode{"root": root, "d1": deploy, "s1": svc}
	v.roots = []*treeNode{root}
	v.expanded = map[string]bool{"root": true}
	v.rebuildOrder()

	output := v.Render()
	plain := stripANSI(output)
	lines := strings.Split(plain, "\n")

	// Verify structure
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d:\n%s", len(lines), plain)
	}

	// First line should be Application
	if !strings.Contains(lines[0], "Application") {
		t.Errorf("first line should contain Application:\n%s", lines[0])
	}

	// Check tree connectors are present
	if !strings.Contains(plain, "├──") && !strings.Contains(plain, "└──") {
		t.Errorf("expected tree connectors in output:\n%s", plain)
	}
}

// TestSetResourceStatuses verifies that SetResourceStatuses correctly updates
// node sync status by matching on (group, kind, namespace, name)
func TestSetResourceStatuses(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	appName := "test-app"

	// Create nodes with specific group/kind/namespace/name
	deploy := &treeNode{
		uid:       appName + "::deploy1",
		group:     "apps",
		kind:      "Deployment",
		name:      "web",
		namespace: "default",
		status:    "", // Initially empty
		health:    "Healthy",
	}
	svc := &treeNode{
		uid:       appName + "::svc1",
		group:     "",
		kind:      "Service",
		name:      "web-svc",
		namespace: "default",
		status:    "",
		health:    "Healthy",
	}
	pod := &treeNode{
		uid:       appName + "::pod1",
		group:     "",
		kind:      "Pod",
		name:      "web-abc123",
		namespace: "default",
		status:    "",
		health:    "Running",
	}

	// Set up tree view state
	v.nodesByUID = map[string]*treeNode{
		deploy.uid: deploy,
		svc.uid:    svc,
		pod.uid:    pod,
	}
	v.nodesByApp = map[string][]string{
		appName: {deploy.uid, svc.uid, pod.uid},
	}

	// Create resource statuses (simulating Application.status.resources)
	resources := []api.ResourceStatus{
		{Group: "apps", Kind: "Deployment", Name: "web", Namespace: "default", Status: "OutOfSync"},
		{Group: "", Kind: "Service", Name: "web-svc", Namespace: "default", Status: "Synced"},
		// Pod not included - it's not a managed resource
	}

	// Apply resource statuses
	v.SetResourceStatuses(appName, resources)

	// Verify deployment got OutOfSync
	if deploy.status != "OutOfSync" {
		t.Errorf("expected Deployment status 'OutOfSync', got %q", deploy.status)
	}

	// Verify service got Synced
	if svc.status != "Synced" {
		t.Errorf("expected Service status 'Synced', got %q", svc.status)
	}

	// Verify pod status unchanged (not a managed resource)
	if pod.status != "" {
		t.Errorf("expected Pod status to remain empty, got %q", pod.status)
	}
}

// TestSetResourceStatuses_DifferentApp verifies that SetResourceStatuses only
// updates nodes for the specified app
func TestSetResourceStatuses_DifferentApp(t *testing.T) {
	v := NewTreeView(100, 20)

	// Create nodes for two different apps
	node1 := &treeNode{
		uid:       "app1::deploy1",
		group:     "apps",
		kind:      "Deployment",
		name:      "web",
		namespace: "default",
		status:    "",
	}
	node2 := &treeNode{
		uid:       "app2::deploy1",
		group:     "apps",
		kind:      "Deployment",
		name:      "web",
		namespace: "default",
		status:    "",
	}

	v.nodesByUID = map[string]*treeNode{
		node1.uid: node1,
		node2.uid: node2,
	}
	v.nodesByApp = map[string][]string{
		"app1": {node1.uid},
		"app2": {node2.uid},
	}

	// Update only app1
	resources := []api.ResourceStatus{
		{Group: "apps", Kind: "Deployment", Name: "web", Namespace: "default", Status: "OutOfSync"},
	}
	v.SetResourceStatuses("app1", resources)

	// Verify app1's node was updated
	if node1.status != "OutOfSync" {
		t.Errorf("expected app1 node status 'OutOfSync', got %q", node1.status)
	}

	// Verify app2's node was NOT updated
	if node2.status != "" {
		t.Errorf("expected app2 node status to remain empty, got %q", node2.status)
	}
}

// TestUpsertAppTree_DuplicateApplicationNodeFiltered verifies that when
// ArgoCD's resource tree includes the Application CR itself as a node,
// it is filtered out to avoid a doubled "Application" line.
func TestUpsertAppTree_DuplicateApplicationNodeFiltered(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())
	v.SetAppMeta("my-app", "Healthy", "Synced")

	tree := &api.ResourceTree{
		Nodes: []api.ResourceNode{
			{UID: "app-uid", Group: "argoproj.io", Version: "v1alpha1", Kind: "Application", Name: "my-app"},
			{UID: "deploy-uid", Group: "apps", Version: "v1", Kind: "Deployment", Name: "web", ParentRefs: []api.ResourceRef{{UID: "app-uid"}}},
		},
	}

	v.UpsertAppTree("my-app", tree)
	output := v.Render()
	plain := stripANSI(output)

	// Count occurrences of "Application" — should be exactly 1 (the synthetic root)
	count := strings.Count(plain, "Application")
	if count != 1 {
		t.Errorf("expected exactly 1 Application line, got %d:\n%s", count, plain)
	}

	// The Deployment child should still be present
	if !strings.Contains(plain, "Deployment") {
		t.Errorf("expected Deployment in output:\n%s", plain)
	}
}

// TestUpsertAppTree_DifferentApplicationNodePreserved verifies that Application
// nodes with a different name (e.g. app-of-apps children) are preserved.
func TestUpsertAppTree_DifferentApplicationNodePreserved(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())
	v.SetAppMeta("parent-app", "Healthy", "Synced")

	tree := &api.ResourceTree{
		Nodes: []api.ResourceNode{
			{UID: "child-app-uid", Group: "argoproj.io", Version: "v1alpha1", Kind: "Application", Name: "child-app"},
			{UID: "deploy-uid", Group: "apps", Version: "v1", Kind: "Deployment", Name: "web"},
		},
	}

	v.UpsertAppTree("parent-app", tree)
	output := v.Render()
	plain := stripANSI(output)

	// Should have 2 Application lines: the synthetic root "parent-app" + the child "child-app"
	count := strings.Count(plain, "Application")
	if count != 2 {
		t.Errorf("expected 2 Application lines (parent + child), got %d:\n%s", count, plain)
	}

	if !strings.Contains(plain, "parent-app") {
		t.Errorf("expected parent-app in output:\n%s", plain)
	}
	if !strings.Contains(plain, "child-app") {
		t.Errorf("expected child-app in output:\n%s", plain)
	}
}

// TestSetSortHealth verifies that SetSort with SortFieldHealth re-orders siblings
// so that Degraded nodes appear before Healthy ones when ascending.
func TestSetSortHealth(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	// Build tree: root → [healthy, degraded, progressing]
	root := &treeNode{uid: "root", kind: "Application", name: "app", health: "Healthy"}
	healthy := &treeNode{uid: "h1", kind: "Deployment", name: "healthy-dep", health: "Healthy", parent: root}
	degraded := &treeNode{uid: "d1", kind: "Deployment", name: "degraded-dep", health: "Degraded", parent: root}
	progressing := &treeNode{uid: "p1", kind: "Deployment", name: "progressing-dep", health: "Progressing", parent: root}
	root.children = []*treeNode{healthy, degraded, progressing}

	v.nodesByUID = map[string]*treeNode{"root": root, "h1": healthy, "d1": degraded, "p1": progressing}
	v.roots = []*treeNode{root}
	v.expanded = map[string]bool{"root": true}
	v.rebuildOrder()

	// Apply health asc sort: Degraded < Progressing < Healthy
	v.SetSort(model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortAsc})

	output := v.Render()
	plain := stripANSI(output)
	lines := strings.Split(plain, "\n")

	// Find lines for each resource
	findLine := func(name string) int {
		for i, l := range lines {
			if strings.Contains(l, name) {
				return i
			}
		}
		return -1
	}

	degradedLine := findLine("degraded-dep")
	progressingLine := findLine("progressing-dep")
	healthyLine := findLine("healthy-dep")

	if degradedLine < 0 || progressingLine < 0 || healthyLine < 0 {
		t.Fatalf("expected all three resources in output:\n%s", plain)
	}
	if degradedLine >= progressingLine {
		t.Errorf("expected Degraded (%d) before Progressing (%d):\n%s", degradedLine, progressingLine, plain)
	}
	if progressingLine >= healthyLine {
		t.Errorf("expected Progressing (%d) before Healthy (%d):\n%s", progressingLine, healthyLine, plain)
	}
}

// TestSetSortSync verifies that SetSort with SortFieldSync re-orders siblings
// so that OutOfSync nodes appear before Synced ones when ascending.
func TestSetSortSync(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	root := &treeNode{uid: "root", kind: "Application", name: "app"}
	synced := &treeNode{uid: "s1", kind: "Service", name: "synced-svc", status: "Synced", parent: root}
	outofsync := &treeNode{uid: "o1", kind: "Service", name: "outofsync-svc", status: "OutOfSync", parent: root}
	unknown := &treeNode{uid: "u1", kind: "Service", name: "unknown-svc", status: "Unknown", parent: root}
	root.children = []*treeNode{synced, outofsync, unknown}

	v.nodesByUID = map[string]*treeNode{"root": root, "s1": synced, "o1": outofsync, "u1": unknown}
	v.roots = []*treeNode{root}
	v.expanded = map[string]bool{"root": true}
	v.rebuildOrder()

	v.SetSort(model.SortConfig{Field: model.SortFieldSync, Direction: model.SortAsc})

	output := v.Render()
	plain := stripANSI(output)
	lines := strings.Split(plain, "\n")

	findLine := func(name string) int {
		for i, l := range lines {
			if strings.Contains(l, name) {
				return i
			}
		}
		return -1
	}

	outofsyncLine := findLine("outofsync-svc")
	unknownLine := findLine("unknown-svc")
	syncedLine := findLine("synced-svc")

	if outofsyncLine < 0 || unknownLine < 0 || syncedLine < 0 {
		t.Fatalf("expected all three resources in output:\n%s", plain)
	}
	if outofsyncLine >= unknownLine {
		t.Errorf("expected OutOfSync (%d) before Unknown (%d):\n%s", outofsyncLine, unknownLine, plain)
	}
	if unknownLine >= syncedLine {
		t.Errorf("expected Unknown (%d) before Synced (%d):\n%s", unknownLine, syncedLine, plain)
	}
}

// TestSetSortHealthDesc verifies that SetSort with desc direction reverses the order.
func TestSetSortHealthDesc(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	root := &treeNode{uid: "root", kind: "Application", name: "app"}
	healthy := &treeNode{uid: "h1", kind: "Deployment", name: "healthy-dep", health: "Healthy", parent: root}
	degraded := &treeNode{uid: "d1", kind: "Deployment", name: "degraded-dep", health: "Degraded", parent: root}
	root.children = []*treeNode{healthy, degraded}

	v.nodesByUID = map[string]*treeNode{"root": root, "h1": healthy, "d1": degraded}
	v.roots = []*treeNode{root}
	v.expanded = map[string]bool{"root": true}
	v.rebuildOrder()

	// Desc: Healthy first, Degraded last
	v.SetSort(model.SortConfig{Field: model.SortFieldHealth, Direction: model.SortDesc})

	output := v.Render()
	plain := stripANSI(output)
	lines := strings.Split(plain, "\n")

	findLine := func(name string) int {
		for i, l := range lines {
			if strings.Contains(l, name) {
				return i
			}
		}
		return -1
	}

	healthyLine := findLine("healthy-dep")
	degradedLine := findLine("degraded-dep")

	if healthyLine < 0 || degradedLine < 0 {
		t.Fatalf("expected both resources in output:\n%s", plain)
	}
	if healthyLine >= degradedLine {
		t.Errorf("expected Healthy (%d) before Degraded (%d) in desc order:\n%s", healthyLine, degradedLine, plain)
	}
}

// TestSetSortNameRestoresDefault verifies that setting sort to name restores (kind, name) order.
func TestSetSortNameRestoresDefault(t *testing.T) {
	v := NewTreeView(100, 20)
	v.ApplyTheme(theme.Default())

	root := &treeNode{uid: "root", kind: "Application", name: "app"}
	// Deliberately create nodes that, by health sort, would differ from name sort
	a := &treeNode{uid: "a1", kind: "Deployment", name: "aaa", health: "Healthy", parent: root}
	b := &treeNode{uid: "b1", kind: "Deployment", name: "bbb", health: "Degraded", parent: root}
	root.children = []*treeNode{b, a} // start reversed

	v.nodesByUID = map[string]*treeNode{"root": root, "a1": a, "b1": b}
	v.roots = []*treeNode{root}
	v.expanded = map[string]bool{"root": true}
	v.rebuildOrder()

	// Sort by name asc: should produce aaa before bbb regardless of health
	v.SetSort(model.SortConfig{Field: model.SortFieldName, Direction: model.SortAsc})

	output := v.Render()
	plain := stripANSI(output)
	lines := strings.Split(plain, "\n")

	findLine := func(name string) int {
		for i, l := range lines {
			if strings.Contains(l, name) {
				return i
			}
		}
		return -1
	}

	aaaLine := findLine("aaa")
	bbbLine := findLine("bbb")

	if aaaLine < 0 || bbbLine < 0 {
		t.Fatalf("expected both resources in output:\n%s", plain)
	}
	if aaaLine >= bbbLine {
		t.Errorf("expected aaa (%d) before bbb (%d) after name sort:\n%s", aaaLine, bbbLine, plain)
	}
}

// stripANSI removes ANSI escape codes from a string for easier testing
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
