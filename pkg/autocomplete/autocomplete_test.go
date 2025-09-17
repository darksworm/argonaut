package autocomplete

import (
	"reflect"
	"testing"

	"github.com/darksworm/argonaut/pkg/model"
)

func createTestState() *model.AppState {
	prodCluster := "prod"
	stagingCluster := "staging"
	analyticsCluster := "analytics"
	webNamespace := "web"
	apiNamespace := "api"
	dataNamespace := "data"
	metricsNamespace := "metrics"
	ecommerceProject := "ecommerce"
	platformProject := "platform"

	state := &model.AppState{
		Apps: []model.App{
			{Name: "frontend", ClusterLabel: &prodCluster, Namespace: &webNamespace, Project: &ecommerceProject},
			{Name: "backend", ClusterLabel: &prodCluster, Namespace: &apiNamespace, Project: &ecommerceProject},
			{Name: "database", ClusterLabel: &stagingCluster, Namespace: &dataNamespace, Project: &ecommerceProject},
			{Name: "cache", ClusterLabel: &prodCluster, Namespace: &webNamespace, Project: &platformProject},
			{Name: "analytics", ClusterLabel: &analyticsCluster, Namespace: &metricsNamespace, Project: &platformProject},
		},
		Selections: *model.NewSelectionState(),
	}
	return state
}

func TestResolveAlias(t *testing.T) {
	engine := NewAutocompleteEngine()

	tests := []struct {
		input    string
		expected string
	}{
		{"cluster", "cluster"},
		{"clusters", "cluster"},
		{"cls", "cluster"},
		{"namespace", "namespace"},
		{"namespaces", "namespace"},
		{"ns", "namespace"},
		{"app", "app"},
		{"apps", "app"},
		{"applications", "app"},
		{"sync", "sync"},
		{"s", "sync"},
		{"diff", "diff"},
		{"d", "diff"},
		{"logs", "logs"},
		{"log", "logs"},
		{"l", "logs"},
		{"unknown", "unknown"}, // Should return input if not found
	}

	for _, test := range tests {
		result := engine.ResolveAlias(test.input)
		if result != test.expected {
			t.Errorf("ResolveAlias(%s) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestGetCommandInfo(t *testing.T) {
	engine := NewAutocompleteEngine()

	// Test valid command
	info := engine.GetCommandInfo("cluster")
	if info == nil {
		t.Fatal("GetCommandInfo(cluster) should not return nil")
	}
	if info.Command != "cluster" {
		t.Errorf("Expected command 'cluster', got %s", info.Command)
	}
	if !info.TakesArg {
		t.Error("cluster command should take an argument")
	}

	// Test alias resolution
	info = engine.GetCommandInfo("cls")
	if info == nil || info.Command != "cluster" {
		t.Error("GetCommandInfo should resolve alias 'cls' to 'cluster'")
	}

	// Test invalid command
	info = engine.GetCommandInfo("invalid")
	if info != nil {
		t.Error("GetCommandInfo(invalid) should return nil")
	}
}

func TestGetCommandAutocomplete_EmptyInput(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test just ":"
	suggestions := engine.GetCommandAutocomplete(":", state)
	if len(suggestions) == 0 {
		t.Error("Should return command suggestions for ':'")
	}

	// Should contain all primary commands
	expected := []string{":all", ":app", ":cluster", ":diff", ":logs", ":namespace", ":project", ":resources", ":rollback", ":sync", ":up"}

	// Check that all expected commands are present (subset check since we have aliases)
	for _, exp := range expected {
		found := false
		for _, sug := range suggestions {
			if sug == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected suggestion '%s' not found in %v", exp, suggestions)
		}
	}
}

func TestGetCommandAutocomplete_CommandCompletion(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test partial command
	suggestions := engine.GetCommandAutocomplete(":cl", state)
	expected := []string{":clear", ":cluster", ":clusters", ":cls"}

	for _, exp := range expected {
		found := false
		for _, sug := range suggestions {
			if sug == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected suggestion '%s' not found in %v", exp, suggestions)
		}
	}
}

func TestGetCommandAutocomplete_ArgumentCompletion(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test GetArgumentSuggestions directly first
	directSuggestions := engine.GetArgumentSuggestions("cluster", "", state)
	t.Logf("Direct argument suggestions for 'cluster': %v", directSuggestions)

	// Test cluster argument completion
	suggestions := engine.GetCommandAutocomplete(":cluster ", state)
	t.Logf("GetCommandAutocomplete result for ':cluster ': %v", suggestions)

	// Should suggest all clusters
	expectedClusters := []string{":cluster analytics", ":cluster prod", ":cluster staging"}
	if !reflect.DeepEqual(suggestions, expectedClusters) {
		t.Errorf("Expected %v, got %v", expectedClusters, suggestions)
	}

	// Test partial cluster completion
	suggestions = engine.GetCommandAutocomplete(":cluster p", state)
	expected := []string{":cluster prod"}
	if !reflect.DeepEqual(suggestions, expected) {
		t.Errorf("Expected %v, got %v", expected, suggestions)
	}
}

func TestGetCommandAutocomplete_FilteredSuggestions(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Set cluster scope to "prod"
	state.Selections.ScopeClusters = model.StringSetFromSlice([]string{"prod"})

	// Test namespace completion with cluster filter
	suggestions := engine.GetCommandAutocomplete(":namespace ", state)

	// Should only suggest namespaces from "prod" cluster
	expected := []string{":namespace api", ":namespace web"}
	if !reflect.DeepEqual(suggestions, expected) {
		t.Errorf("Expected %v, got %v", expected, suggestions)
	}

	// Add namespace scope
	state.Selections.ScopeNamespaces = model.StringSetFromSlice([]string{"web"})

	// Test project completion with cluster and namespace filters
	suggestions = engine.GetCommandAutocomplete(":project ", state)
	expected = []string{":project ecommerce", ":project platform"}
	if !reflect.DeepEqual(suggestions, expected) {
		t.Errorf("Expected %v, got %v", expected, suggestions)
	}

	// Test app completion with all filters
	state.Selections.ScopeProjects = model.StringSetFromSlice([]string{"ecommerce"})
	suggestions = engine.GetCommandAutocomplete(":app ", state)
	expected = []string{":app frontend"}
	if !reflect.DeepEqual(suggestions, expected) {
		t.Errorf("Expected %v, got %v", expected, suggestions)
	}
}

func TestGetCommandAutocomplete_NoArgumentCommands(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test commands that don't take arguments
	suggestions := engine.GetCommandAutocomplete(":logs ", state)
	if suggestions != nil {
		t.Error("logs command should not provide argument suggestions")
	}

	suggestions = engine.GetCommandAutocomplete(":all ", state)
	if suggestions != nil {
		t.Error("all command should not provide argument suggestions")
	}
}

func TestGetCommandAutocomplete_InvalidInput(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test input not starting with ":"
	suggestions := engine.GetCommandAutocomplete("cluster", state)
	if suggestions != nil {
		t.Error("Should return nil for input not starting with ':'")
	}

	// Test empty input
	suggestions = engine.GetCommandAutocomplete("", state)
	if suggestions != nil {
		t.Error("Should return nil for empty input")
	}
}

func TestGetCommandAutocomplete_CaseInsensitive(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test case insensitive command matching
	suggestions1 := engine.GetCommandAutocomplete(":CL", state)
	suggestions2 := engine.GetCommandAutocomplete(":cl", state)

	if !reflect.DeepEqual(suggestions1, suggestions2) {
		t.Error("Command matching should be case insensitive")
	}

	// Test case insensitive argument matching
	suggestions1 = engine.GetCommandAutocomplete(":cluster P", state)
	suggestions2 = engine.GetCommandAutocomplete(":cluster p", state)

	if !reflect.DeepEqual(suggestions1, suggestions2) {
		t.Error("Argument matching should be case insensitive")
	}
}
