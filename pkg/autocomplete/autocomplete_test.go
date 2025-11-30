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

func TestSortCommandAutocomplete(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test sort command exists
	info := engine.GetCommandInfo("sort")
	if info == nil {
		t.Fatal("sort command should be registered")
	}
	if !info.TakesArg {
		t.Error("sort command should take an argument")
	}

	// Test sort field suggestions with trailing space
	suggestions := engine.GetCommandAutocomplete(":sort ", state)
	expectedFields := []string{":sort health", ":sort name", ":sort sync"}
	if !reflect.DeepEqual(suggestions, expectedFields) {
		t.Errorf("Expected %v, got %v", expectedFields, suggestions)
	}

	// Test partial field completion
	suggestions = engine.GetCommandAutocomplete(":sort n", state)
	expected := []string{":sort name"}
	if !reflect.DeepEqual(suggestions, expected) {
		t.Errorf("Expected %v, got %v", expected, suggestions)
	}

	// Test direction suggestions after field with trailing space
	suggestions = engine.GetCommandAutocomplete(":sort name ", state)
	expectedDirs := []string{":sort name asc", ":sort name desc"}
	if !reflect.DeepEqual(suggestions, expectedDirs) {
		t.Errorf("Expected %v, got %v", expectedDirs, suggestions)
	}

	// Test partial direction completion
	suggestions = engine.GetCommandAutocomplete(":sort name d", state)
	expected = []string{":sort name desc"}
	if !reflect.DeepEqual(suggestions, expected) {
		t.Errorf("Expected %v, got %v", expected, suggestions)
	}
}

// TestSortCommandRequiresDirection tests that ":sort name" (without direction)
// shows direction suggestions to guide the user to complete the command.
// Direction is required - the command is not valid without it.
func TestSortCommandRequiresDirection(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// When user types ":sort name" (complete field, no trailing space),
	// autocomplete should suggest directions to help complete the command
	suggestions := engine.GetCommandAutocomplete(":sort name", state)

	// Should suggest direction options
	if len(suggestions) != 2 {
		t.Errorf("Expected 2 direction suggestions, got %d: %v", len(suggestions), suggestions)
	}

	// Verify the suggestions are the expected directions
	expected := map[string]bool{":sort name asc": true, ":sort name desc": true}
	for _, s := range suggestions {
		if !expected[s] {
			t.Errorf("Unexpected suggestion: %s", s)
		}
	}
}

func TestThemeCommandAutocomplete(t *testing.T) {
	engine := NewAutocompleteEngine()
	state := createTestState()

	// Test theme command exists
	info := engine.GetCommandInfo("theme")
	if info == nil {
		t.Fatal("theme command should be registered")
	}
	if !info.TakesArg {
		t.Error("theme command should take an argument")
	}
	if info.ArgType != "theme" {
		t.Errorf("Expected ArgType 'theme', got %s", info.ArgType)
	}

	// Test theme argument suggestions
	suggestions := engine.GetArgumentSuggestions("theme", "", state)
	if len(suggestions) == 0 {
		t.Error("Should return theme suggestions")
	}

	// Verify expected themes are present (suggestions will have ":theme " prefix)
	expectedThemes := []string{":theme oxocarbon", ":theme dracula", ":theme nord", ":theme gruvbox", ":theme tokyo-night", ":theme monokai"}
	suggestionMap := make(map[string]bool)
	for _, suggestion := range suggestions {
		suggestionMap[suggestion] = true
	}

	for _, expected := range expectedThemes {
		if !suggestionMap[expected] {
			t.Errorf("Expected theme %q not found in suggestions: %v", expected, suggestions)
		}
	}

	// Test prefix matching
	prefixSuggestions := engine.GetArgumentSuggestions("theme", "d", state)
	found := false
	for _, suggestion := range prefixSuggestions {
		if suggestion == ":theme dracula" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Should find ':theme dracula' when searching with prefix 'd'")
	}

	// Test full command autocomplete
	fullSuggestions := engine.GetCommandAutocomplete(":theme ", state)
	if len(fullSuggestions) == 0 {
		t.Error("Should return suggestions for ':theme ' command")
	}
}

func TestAppSetCommandAutocomplete(t *testing.T) {
	engine := NewAutocompleteEngine()

	// Create state with apps that have ApplicationSet references
	appset1 := "nerdy-demo"
	appset2 := "production-apps"

	state := &model.AppState{
		Apps: []model.App{
			{Name: "app-from-appset-1", ApplicationSet: &appset1},
			{Name: "app-from-appset-2", ApplicationSet: &appset1},
			{Name: "prod-app-1", ApplicationSet: &appset2},
			{Name: "standalone-app", ApplicationSet: nil}, // No ApplicationSet
		},
		Selections: *model.NewSelectionState(),
	}

	// Test appset command exists
	info := engine.GetCommandInfo("appset")
	if info == nil {
		t.Fatal("appset command should be registered")
	}
	if !info.TakesArg {
		t.Error("appset command should take an argument")
	}
	if info.ArgType != "appset" {
		t.Errorf("Expected ArgType 'appset', got %s", info.ArgType)
	}

	// Test appset argument suggestions
	suggestions := engine.GetArgumentSuggestions("appset", "", state)
	if len(suggestions) != 2 {
		t.Errorf("Expected 2 unique ApplicationSet suggestions, got %d: %v", len(suggestions), suggestions)
	}

	// Verify expected appsets are present
	suggestionMap := make(map[string]bool)
	for _, s := range suggestions {
		suggestionMap[s] = true
	}
	if !suggestionMap[":appset nerdy-demo"] {
		t.Error("Should suggest ':appset nerdy-demo'")
	}
	if !suggestionMap[":appset production-apps"] {
		t.Error("Should suggest ':appset production-apps'")
	}

	// Test prefix matching
	prefixSuggestions := engine.GetArgumentSuggestions("appset", "n", state)
	if len(prefixSuggestions) != 1 {
		t.Errorf("Expected 1 suggestion for prefix 'n', got %d: %v", len(prefixSuggestions), prefixSuggestions)
	}
	if len(prefixSuggestions) > 0 && prefixSuggestions[0] != ":appset nerdy-demo" {
		t.Errorf("Expected ':appset nerdy-demo', got %s", prefixSuggestions[0])
	}

	// Test alias resolution
	if engine.ResolveAlias("appsets") != "appset" {
		t.Error("'appsets' should resolve to 'appset'")
	}
	if engine.ResolveAlias("applicationsets") != "appset" {
		t.Error("'applicationsets' should resolve to 'appset'")
	}
	if engine.ResolveAlias("as") != "appset" {
		t.Error("'as' should resolve to 'appset'")
	}
}

func TestAppSetCommandAutocomplete_NoAppSets(t *testing.T) {
	engine := NewAutocompleteEngine()

	// Create state with apps that have NO ApplicationSet references
	state := &model.AppState{
		Apps: []model.App{
			{Name: "standalone-app-1"},
			{Name: "standalone-app-2"},
		},
		Selections: *model.NewSelectionState(),
	}

	suggestions := engine.GetArgumentSuggestions("appset", "", state)
	if len(suggestions) != 0 {
		t.Errorf("Expected 0 suggestions when no apps have ApplicationSet, got %d: %v", len(suggestions), suggestions)
	}
}
