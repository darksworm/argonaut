package autocomplete

import (
	"sort"
	"strings"

	"github.com/darksworm/argonaut/pkg/model"
	th "github.com/darksworm/argonaut/pkg/theme"
)

// CommandAlias represents a command with its aliases and metadata
type CommandAlias struct {
	Command     string   // Primary command name
	Aliases     []string // All possible aliases for this command
	Description string   // Help text for the command
	TakesArg    bool     // Whether command accepts an argument
	ArgType     string   // Type of argument (e.g., "app", "cluster")
}

// AliasMap maps all command variants to their canonical command
type AliasMap map[string]string

// AutocompleteEngine handles command completion and aliases
type AutocompleteEngine struct {
	commands []CommandAlias
	aliasMap AliasMap
}

// NewAutocompleteEngine creates a new autocomplete engine with command definitions
func NewAutocompleteEngine() *AutocompleteEngine {
	commands := []CommandAlias{
		{
			Command:     "cluster",
			Aliases:     []string{"cluster", "clusters", "cls"},
			Description: "Navigate to clusters view",
			TakesArg:    true,
			ArgType:     "cluster",
		},
		{
			Command:     "namespace",
			Aliases:     []string{"namespace", "namespaces", "ns"},
			Description: "Navigate to namespaces view",
			TakesArg:    true,
			ArgType:     "namespace",
		},
		{
			Command:     "project",
			Aliases:     []string{"project", "projects", "proj"},
			Description: "Navigate to projects view",
			TakesArg:    true,
			ArgType:     "project",
		},
		{
			Command:     "app",
			Aliases:     []string{"app", "apps", "application", "applications"},
			Description: "Navigate to applications view",
			TakesArg:    true,
			ArgType:     "app",
		},
		{
			Command:     "sync",
			Aliases:     []string{"sync", "s"},
			Description: "Sync selected applications",
			TakesArg:    true,
			ArgType:     "app",
		},
		{
			Command:     "diff",
			Aliases:     []string{"diff", "d"},
			Description: "Show diff for application",
			TakesArg:    true,
			ArgType:     "app",
		},
		{
			Command:     "rollback",
			Aliases:     []string{"rollback", "rb", "revert"},
			Description: "Rollback application to previous revision",
			TakesArg:    true,
			ArgType:     "app",
		},
		{
			Command:     "resources",
			Aliases:     []string{"resources", "res", "r"},
			Description: "Show resources for application",
			TakesArg:    true,
			ArgType:     "app",
		},
		{
			Command:     "logs",
			Aliases:     []string{"logs", "log", "l"},
			Description: "Show application logs",
			TakesArg:    false,
			ArgType:     "",
		},
		{
			Command:     "all",
			Aliases:     []string{"all", "clear", "reset"},
			Description: "Clear all filters and selections",
			TakesArg:    false,
			ArgType:     "",
		},
		{
			Command:     "up",
			Aliases:     []string{"up", "back", ".."},
			Description: "Go up one level in navigation",
			TakesArg:    false,
			ArgType:     "",
		},
		{
			Command:     "theme",
			Aliases:     []string{"theme"},
			Description: "Switch UI theme (built-in names)",
			TakesArg:    true,
			ArgType:     "theme",
		},
		{
			Command:     "quit",
			Aliases:     []string{"quit", "q", "q!", "exit"},
			Description: "Exit the application",
			TakesArg:    false,
			ArgType:     "",
		},
	}

	// Build alias map
	aliasMap := make(AliasMap)
	for _, cmd := range commands {
		for _, alias := range cmd.Aliases {
			aliasMap[alias] = cmd.Command
		}
	}

	return &AutocompleteEngine{
		commands: commands,
		aliasMap: aliasMap,
	}
}

// ResolveAlias converts any command alias to its canonical form
func (e *AutocompleteEngine) ResolveAlias(input string) string {
	if canonical, exists := e.aliasMap[strings.ToLower(input)]; exists {
		return canonical
	}
	return input
}

// GetCommandInfo returns command information for a given command or alias
func (e *AutocompleteEngine) GetCommandInfo(input string) *CommandAlias {
	canonical := e.ResolveAlias(input)
	for _, cmd := range e.commands {
		if cmd.Command == canonical {
			return &cmd
		}
	}
	return nil
}

// GetCommandAutocomplete returns autocomplete suggestions for command input
func (e *AutocompleteEngine) GetCommandAutocomplete(input string, state *model.AppState) []string {
	// Check for trailing space BEFORE trimming
	hasTrailingSpace := strings.HasSuffix(input, " ")

	// Trim only leading space, keep trailing space info
	input = strings.TrimSpace(input)

	// Must start with ":"
	if !strings.HasPrefix(input, ":") {
		return nil
	}

	// Remove the ":"
	command := input[1:]

	// Split into command and argument parts
	parts := strings.Fields(command)

	if len(parts) == 0 {
		// Just ":" - return all command suggestions
		return e.getAllCommandSuggestions("")
	}

	if len(parts) == 1 {
		if hasTrailingSpace {
			// Command is complete, suggest arguments (e.g., ":cluster ")
			return e.getArgumentSuggestions(parts[0], "", state)
		} else {
			// Command completion (no space yet, e.g., ":cl")
			return e.getAllCommandSuggestions(parts[0])
		}
	}

	if len(parts) == 2 {
		// Argument completion (e.g., ":cluster pr")
		return e.getArgumentSuggestions(parts[0], parts[1], state)
	}

	return nil
}

// getAllCommandSuggestions returns command name suggestions
func (e *AutocompleteEngine) getAllCommandSuggestions(prefix string) []string {
	var suggestions []string
	prefix = strings.ToLower(prefix)

	// Collect unique command names that match prefix
	seen := make(map[string]bool)

	for _, cmd := range e.commands {
		for _, alias := range cmd.Aliases {
			if strings.HasPrefix(strings.ToLower(alias), prefix) {
				if !seen[alias] {
					suggestions = append(suggestions, ":"+alias)
					seen[alias] = true
				}
			}
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// GetArgumentSuggestions returns argument suggestions for a command (public for debugging)
func (e *AutocompleteEngine) GetArgumentSuggestions(command, argPrefix string, state *model.AppState) []string {
	return e.getArgumentSuggestions(command, argPrefix, state)
}

// getArgumentSuggestions returns argument suggestions for a command
func (e *AutocompleteEngine) getArgumentSuggestions(command, argPrefix string, state *model.AppState) []string {
	cmdInfo := e.GetCommandInfo(command)
	if cmdInfo == nil || !cmdInfo.TakesArg {
		return nil
	}

	argPrefix = strings.ToLower(argPrefix)
	var suggestions []string

	switch cmdInfo.ArgType {
	case "cluster":
		suggestions = e.getClusterSuggestions(argPrefix, state)
	case "namespace":
		suggestions = e.getNamespaceSuggestions(argPrefix, state)
	case "project":
		suggestions = e.getProjectSuggestions(argPrefix, state)
	case "app":
		suggestions = e.getAppSuggestions(argPrefix, state)
	case "theme":
		suggestions = e.getThemeSuggestions(argPrefix)
	}

	// Add command prefix to suggestions
	prefixedSuggestions := make([]string, len(suggestions))
	for i, suggestion := range suggestions {
		prefixedSuggestions[i] = ":" + command + " " + suggestion
	}

	return prefixedSuggestions
}

// getClusterSuggestions returns cluster name suggestions
func (e *AutocompleteEngine) getClusterSuggestions(prefix string, state *model.AppState) []string {
	var suggestions []string
	seen := make(map[string]bool)

	for _, app := range state.Apps {
		if app.ClusterLabel == nil {
			continue
		}
		cluster := strings.ToLower(*app.ClusterLabel)
		if strings.HasPrefix(cluster, prefix) && !seen[cluster] {
			suggestions = append(suggestions, *app.ClusterLabel)
			seen[cluster] = true
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// getNamespaceSuggestions returns namespace suggestions (filtered by selected clusters)
func (e *AutocompleteEngine) getNamespaceSuggestions(prefix string, state *model.AppState) []string {
	var suggestions []string
	seen := make(map[string]bool)

	for _, app := range state.Apps {
		if app.ClusterLabel == nil || app.Namespace == nil {
			continue
		}

		// Apply cluster filtering if clusters are selected
		if len(state.Selections.ScopeClusters) > 0 {
			if !model.HasInStringSet(state.Selections.ScopeClusters, *app.ClusterLabel) {
				continue
			}
		}

		namespace := strings.ToLower(*app.Namespace)
		if strings.HasPrefix(namespace, prefix) && !seen[namespace] {
			suggestions = append(suggestions, *app.Namespace)
			seen[namespace] = true
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// getProjectSuggestions returns project suggestions (filtered by selected clusters/namespaces)
func (e *AutocompleteEngine) getProjectSuggestions(prefix string, state *model.AppState) []string {
	var suggestions []string
	seen := make(map[string]bool)

	for _, app := range state.Apps {
		if app.ClusterLabel == nil || app.Namespace == nil || app.Project == nil {
			continue
		}

		// Apply cluster filtering
		if len(state.Selections.ScopeClusters) > 0 {
			if !model.HasInStringSet(state.Selections.ScopeClusters, *app.ClusterLabel) {
				continue
			}
		}

		// Apply namespace filtering
		if len(state.Selections.ScopeNamespaces) > 0 {
			if !model.HasInStringSet(state.Selections.ScopeNamespaces, *app.Namespace) {
				continue
			}
		}

		project := strings.ToLower(*app.Project)
		if strings.HasPrefix(project, prefix) && !seen[project] {
			suggestions = append(suggestions, *app.Project)
			seen[project] = true
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// getAppSuggestions returns app name suggestions (filtered by current selections)
func (e *AutocompleteEngine) getAppSuggestions(prefix string, state *model.AppState) []string {
	var suggestions []string

	for _, app := range state.Apps {
		// Apply all current filters
		if len(state.Selections.ScopeClusters) > 0 {
			if app.ClusterLabel == nil || !model.HasInStringSet(state.Selections.ScopeClusters, *app.ClusterLabel) {
				continue
			}
		}

		if len(state.Selections.ScopeNamespaces) > 0 {
			if app.Namespace == nil || !model.HasInStringSet(state.Selections.ScopeNamespaces, *app.Namespace) {
				continue
			}
		}

		if len(state.Selections.ScopeProjects) > 0 {
			if app.Project == nil || !model.HasInStringSet(state.Selections.ScopeProjects, *app.Project) {
				continue
			}
		}

		if strings.HasPrefix(strings.ToLower(app.Name), prefix) {
			suggestions = append(suggestions, app.Name)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// getThemeSuggestions returns available theme names
func (e *AutocompleteEngine) getThemeSuggestions(prefix string) []string {
	var suggestions []string
	names := th.Names()
	prefix = strings.ToLower(prefix)
	for _, n := range names {
		if strings.HasPrefix(strings.ToLower(n), prefix) {
			suggestions = append(suggestions, n)
		}
	}
	sort.Strings(suggestions)
	return suggestions
}

// GetAllCommands returns all available commands for help/reference
func (e *AutocompleteEngine) GetAllCommands() []CommandAlias {
	return e.commands
}
