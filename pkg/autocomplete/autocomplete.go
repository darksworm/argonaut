package autocomplete

import (
	"sort"
	"strings"

	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
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
			Command:     "delete",
			Aliases:     []string{"delete", "del", "rm"},
			Description: "Delete application",
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
			Aliases:     []string{"theme", "themes"},
			Description: "Switch UI theme (built-in presets or 'custom')",
			TakesArg:    true,
			ArgType:     "theme",
		},
		{
			Command:     "quit",
			Aliases:     []string{"quit", "q", "q!", "wq", "wq!", "exit"},
			Description: "Exit the application",
			TakesArg:    false,
			ArgType:     "",
		},
		{
			Command:     "help",
			Aliases:     []string{"help", "h", "?"},
			Description: "Show help modal",
			TakesArg:    false,
			ArgType:     "",
		},
		{
			Command:     "upgrade",
			Aliases:     []string{"upgrade", "update"},
			Description: "Upgrade to the latest version",
			TakesArg:    false,
			ArgType:     "",
		},
		{
			Command:     "sort",
			Aliases:     []string{"sort"},
			Description: "Sort apps by field and direction (e.g., :sort name asc)",
			TakesArg:    true,
			ArgType:     "sort",
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
		if hasTrailingSpace {
			// First arg is complete, suggest second argument if applicable (e.g., ":sort name ")
			return e.getSecondArgumentSuggestions(parts[0], parts[1], "", true, state)
		}
		// Check if first arg exactly matches a valid option for commands with second args
		// This allows showing "asc" suggestion right after typing "name" without needing a space
		cmdInfo := e.GetCommandInfo(parts[0])
		if cmdInfo != nil && cmdInfo.Command == "sort" {
			firstArgSuggestions := e.getSortSuggestions("")
			for _, s := range firstArgSuggestions {
				if strings.EqualFold(s, parts[1]) {
					// First arg is complete, suggest second argument with a leading space
					return e.getSecondArgumentSuggestions(parts[0], parts[1], "", false, state)
				}
			}
		}
		// Argument completion (e.g., ":cluster pr")
		return e.getArgumentSuggestions(parts[0], parts[1], state)
	}

	if len(parts) == 3 {
		// Second argument completion (e.g., ":sort name as")
		return e.getSecondArgumentSuggestions(parts[0], parts[1], parts[2], hasTrailingSpace, state)
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
	case "sort":
		suggestions = e.getSortSuggestions(argPrefix)
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

// getThemeSuggestions returns available theme suggestions
func (e *AutocompleteEngine) getThemeSuggestions(prefix string) []string {
	var suggestions []string

	// Get all built-in theme names
	themeNames := theme.GetAvailableThemes()

	for _, themeName := range themeNames {
		if strings.HasPrefix(strings.ToLower(themeName), strings.ToLower(prefix)) {
			suggestions = append(suggestions, themeName)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// getSortSuggestions returns available sort option suggestions
func (e *AutocompleteEngine) getSortSuggestions(prefix string) []string {
	// Sort suggestions are just field names - direction is a second argument
	options := []string{
		"name", "sync", "health",
	}

	var suggestions []string
	prefix = strings.ToLower(prefix)

	for _, opt := range options {
		if strings.HasPrefix(strings.ToLower(opt), prefix) {
			suggestions = append(suggestions, opt)
		}
	}

	sort.Strings(suggestions)
	return suggestions
}

// getSecondArgumentSuggestions returns suggestions for a second argument (e.g., sort direction)
// The hasTrailingSpace parameter indicates if the original input had a trailing space after the current token
func (e *AutocompleteEngine) getSecondArgumentSuggestions(command, firstArg, prefix string, hasTrailingSpace bool, state *model.AppState) []string {
	cmdInfo := e.GetCommandInfo(command)
	if cmdInfo == nil {
		return nil
	}

	// Currently only sort command has a second argument
	if cmdInfo.Command != "sort" {
		return nil
	}

	// Suggest direction options
	options := []string{"asc", "desc"}
	var suggestions []string
	prefix = strings.ToLower(prefix)

	for _, opt := range options {
		if strings.HasPrefix(opt, prefix) {
			suggestions = append(suggestions, opt)
		}
	}

	// Build suggestions that match the input format exactly
	// When hasTrailingSpace is true and prefix is empty, input is "sort name "
	// so suggestion should be "sort name asc" (matching the space)
	prefixedSuggestions := make([]string, len(suggestions))
	for i, suggestion := range suggestions {
		if hasTrailingSpace && prefix == "" {
			// Input: "sort name " -> Suggestion: "sort name asc"
			prefixedSuggestions[i] = ":" + command + " " + firstArg + " " + suggestion
		} else if prefix != "" {
			// Input: "sort name a" or "sort name as" -> Suggestion: "sort name asc"
			prefixedSuggestions[i] = ":" + command + " " + firstArg + " " + suggestion
		} else {
			// Input: "sort name" (no trailing space) -> Suggestion: "sort name asc" (add space before direction)
			prefixedSuggestions[i] = ":" + command + " " + firstArg + " " + suggestion
		}
	}

	return prefixedSuggestions
}

// GetAllCommands returns all available commands for help/reference
func (e *AutocompleteEngine) GetAllCommands() []CommandAlias {
	return e.commands
}
