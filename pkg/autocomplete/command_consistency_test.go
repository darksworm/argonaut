package autocomplete

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// TestCommandHandlerConsistency ensures all commands handled in the input handler
// are also defined in the autocomplete engine
func TestCommandHandlerConsistency(t *testing.T) {
	// Parse the input handler file to extract commands from switch cases
	inputHandlerPath := filepath.Join("..", "..", "cmd", "app", "input_components.go")

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, inputHandlerPath, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("Failed to parse input handler: %v", err)
	}

	// Extract commands from switch statements
	handlerCommands := extractCommandsFromSwitches(node)

	// Get all commands from autocomplete engine
	engine := NewAutocompleteEngine()
	autocompleteCommands := make(map[string]bool)

	for _, cmdInfo := range engine.GetAllCommands() {
		autocompleteCommands[cmdInfo.Command] = true
		for _, alias := range cmdInfo.Aliases {
			autocompleteCommands[alias] = true
		}
	}

	// Find commands that are in handler but not in autocomplete
	var missing []string
	for cmd := range handlerCommands {
		if !autocompleteCommands[cmd] {
			// Skip compound cases and special patterns
			if !isSpecialCase(cmd) {
				missing = append(missing, cmd)
			}
		}
	}

	if len(missing) > 0 {
		t.Errorf("Commands found in handler but missing from autocomplete: %v", missing)
		t.Logf("Add these commands to the autocomplete engine in NewAutocompleteEngine()")
	}
}

// extractCommandsFromSwitches finds all string literals in switch cases
func extractCommandsFromSwitches(node *ast.File) map[string]bool {
	commands := make(map[string]bool)

	ast.Inspect(node, func(n ast.Node) bool {
		if switchStmt, ok := n.(*ast.SwitchStmt); ok {
			// Look for switch statements on 'canonical' variable
			if ident, ok := switchStmt.Tag.(*ast.Ident); ok && ident.Name == "canonical" {
				for _, stmt := range switchStmt.Body.List {
					if caseClause, ok := stmt.(*ast.CaseClause); ok {
						for _, expr := range caseClause.List {
							if basicLit, ok := expr.(*ast.BasicLit); ok && basicLit.Kind == token.STRING {
								cmd, _ := strconv.Unquote(basicLit.Value)
								commands[cmd] = true
							}
						}
					}
				}
			}
		}
		return true
	})

	return commands
}

// isSpecialCase checks if a command is a special case that doesn't need autocomplete
func isSpecialCase(cmd string) bool {
	// These are command variants that are handled in compound case statements
	// and don't need separate autocomplete entries
	specialCases := map[string]bool{
		"del":         true, // alias for delete
		"res":         true, // alias for resources
		"r":           true, // alias for resources
		"q!":          true, // alias for quit
		"wq":          true, // alias for quit
		"wq!":         true, // alias for quit
		"clusters":    true, // alias for cluster
		"cls":         true, // alias for cluster
		"namespaces":  true, // alias for namespace
		"ns":          true, // alias for namespace
		"projects":    true, // alias for project
		"proj":        true, // alias for project
		"apps":        true, // alias for app
		"update":      true, // alias for upgrade
	}

	return specialCases[cmd]
}