package treeview

import (
	"strings"
	"testing"

	"github.com/darksworm/argonaut/pkg/api"
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
