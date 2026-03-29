package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/api"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// TestResCommand_CursorPos_PreservesNamespace verifies that when the user runs :r
// with the cursor on a same-named app in a second namespace, the correct app's
// namespace is preserved in TreeApp — not the first-match-by-name from the list.
func TestResCommand_CursorPos_PreservesNamespace(t *testing.T) {
	m := buildSyncTestModel(100, 30)

	nsArgocd := "argocd"
	nsTeamA := "team-a"
	// The "wrong" app is first in the slice; the cursor sits on the second one.
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd},
		{Name: "my-app", AppNamespace: &nsTeamA},
	}
	m.state.Navigation.View = model.ViewApps
	m.state.Navigation.SelectedIdx = 1 // cursor on the team-a app

	// Enter command mode with "r" typed, then press Enter to execute.
	m.state.Mode = model.ModeCommand
	m.inputComponents.SetCommandValue("r")
	m.state.UI.Command = "r"

	newModel, _ := m.handleEnhancedCommandModeKeys(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = newModel.(*Model)

	if m.state.Navigation.View != model.ViewTree {
		t.Fatalf("expected ViewTree after :r command, got %s", m.state.Navigation.View)
	}
	if m.state.UI.TreeApp == nil {
		t.Fatal("expected TreeApp to be set after :r command")
	}
	if m.state.UI.TreeApp.AppNamespace == nil || *m.state.UI.TreeApp.AppNamespace != nsTeamA {
		t.Errorf("expected TreeApp.AppNamespace %q (the cursor app), got %v",
			nsTeamA, m.state.UI.TreeApp.AppNamespace)
	}
}

// TestHandleOpenK9s_UsesAppNamespaceForClusterLookup verifies that when two apps share
// a name but live in different namespaces (and thus belong to different clusters),
// handleOpenK9s picks the cluster that belongs to the app currently shown in the tree
// view (identified by both Name and AppNamespace), not the first name-match in the list.
//
// We arrange things so that only the correct app's cluster has a matching kubeconfig
// context.  Before the fix the wrong cluster is queried → no context → mode becomes
// ModeK9sContextSelect.  After the fix the right cluster is queried → context found →
// mode stays ModeNormal and k9s is launched directly.
func TestHandleOpenK9s_UsesAppNamespaceForClusterLookup(t *testing.T) {
	// Write a kubeconfig that only has a context matching "cluster-b"
	dir := t.TempDir()
	kubeconfigPath := filepath.Join(dir, "kubeconfig")
	content := `
apiVersion: v1
kind: Config
contexts:
  - name: cluster-b
    context:
      cluster: cluster-b
      user: user-b
clusters:
  - name: cluster-b
    cluster:
      server: https://cluster-b.example.com:6443
users:
  - name: user-b
    user:
      token: test
current-context: cluster-b
`
	if err := os.WriteFile(kubeconfigPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	t.Setenv("KUBECONFIG", kubeconfigPath)

	m := buildSyncTestModel(100, 30)

	nsArgocd := "argocd"
	nsTeamA := "team-a"
	clusterA := "cluster-a" // no context in kubeconfig → findK9sContext fails
	clusterB := "cluster-b" // has a matching context → findK9sContext succeeds
	m.state.Apps = []model.App{
		{Name: "my-app", AppNamespace: &nsArgocd, ClusterID: &clusterA},
		{Name: "my-app", AppNamespace: &nsTeamA, ClusterID: &clusterB},
	}

	// TreeApp points to the team-a app (cluster-b).
	m.state.UI.TreeApp = &model.TreeAppInfo{
		Name:         "my-app",
		AppNamespace: &nsTeamA,
	}
	m.state.Navigation.View = model.ViewTree

	// Populate treeView with a non-Application resource so handleOpenK9s takes
	// the cluster-lookup path rather than the openK9sForApplicationCR path.
	m.treeView = treeview.NewTreeView(0, 0)
	ns := "default"
	healthy := "Healthy"
	tree := api.ResourceTree{Nodes: []api.ResourceNode{
		{UID: "d1", Kind: "Deployment", Name: "my-deploy", Namespace: &ns, Health: &api.ResourceHealth{Status: &healthy}},
	}}
	m.treeView.SetAppMeta("my-app", "Healthy", "Synced")
	m.treeView.UpsertAppTree("my-app", &tree)
	// Move selection to the Deployment node (index 1; index 0 is the synthetic Application root).
	m.treeView.SetSelectedIndex(1)

	newModel, _ := m.handleOpenK9s()
	m = newModel.(*Model)

	// With the correct app (cluster-b), findK9sContext succeeds → k9s is launched
	// directly without showing the context picker.
	if m.state.Mode == model.ModeK9sContextSelect {
		t.Errorf("expected context to be found (mode should not be ModeK9sContextSelect); " +
			"the wrong cluster (cluster-a) was likely used instead of cluster-b")
	}
}
