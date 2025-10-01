package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// Key handler functions for command mode input control

// handleCtrlCKey handles Ctrl+C in command mode - closes input without quitting app
func (m *Model) handleCtrlCKey(key string) (tea.Model, tea.Cmd, bool) {
	m.inputComponents.BlurInputs()
	m.inputComponents.ClearCommandInput()
	m.state.Mode = model.ModeNormal
	m.state.UI.Command = ""
	return m, nil, true
}

// handleEscapeKey handles Esc key in command mode - closes input
func (m *Model) handleEscapeKey(key string) (tea.Model, tea.Cmd, bool) {
	m.inputComponents.BlurInputs()
	m.inputComponents.ClearCommandInput()
	m.state.Mode = model.ModeNormal
	m.state.UI.Command = ""
	return m, nil, true
}

// handleTabKey handles Tab key in command mode - autocomplete
func (m *Model) handleTabKey(key string) (tea.Model, tea.Cmd, bool) {
	// Tab completion - accept the first autocomplete suggestion
	currentInput := m.inputComponents.GetCommandValue()
	// Build query with ':' prefix for the engine
	query := currentInput
	if !strings.HasPrefix(query, ":") {
		query = ":" + query
	}
	suggestions := m.autocompleteEngine.GetCommandAutocomplete(query, m.state)
	if len(suggestions) > 0 {
		// Apply the suggestion text to the input without the leading ':'
		applied := strings.TrimPrefix(suggestions[0], ":")
		m.inputComponents.SetCommandValue(applied)
		m.state.UI.Command = applied
		// Move the cursor to the end of the newly-applied text so the
		// user can continue typing immediately (e.g., ":ns <completed>")
		m.inputComponents.commandInput.CursorEnd()
	}
	return m, nil, true
}

// handleEnterKey handles Enter key in command mode - execute command
func (m *Model) handleEnterKey(key string) (tea.Model, tea.Cmd, bool) {
	// Execute simple navigation commands (clusters/namespaces/projects/apps) with aliases
	// but first, if there's an autocomplete suggestion that extends the input,
	// accept it implicitly so Enter completes rather than errors.
	typed := strings.TrimSpace(m.inputComponents.GetCommandValue())
	// Build query with ':' prefix
	q := typed
	if !strings.HasPrefix(q, ":") {
		q = ":" + q
	}
	sugg := m.autocompleteEngine.GetCommandAutocomplete(q, m.state)
	raw := typed
	if len(sugg) > 0 {
		applied := strings.TrimPrefix(sugg[0], ":")
		// Only accept if it continues what was typed (prefix match)
		if strings.HasPrefix(strings.ToLower(applied), strings.ToLower(typed)) {
			raw = applied
		}
	}
	if raw == "" {
		return m, nil, true
	}

	parts := strings.Fields(raw)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	// Pre-validate existence for arg-based commands before blurring input
	existsIn := func(list []string, name string) bool {
		for _, it := range list {
			if strings.EqualFold(it, name) {
				return true
			}
		}
		return false
	}
	canonical := m.autocompleteEngine.ResolveAlias(cmd)
	if arg != "" {
		switch canonical {
		case "cluster":
			all := m.autocompleteEngine.GetArgumentSuggestions("cluster", "", m.state)
			names := make([]string, 0, len(all))
			for _, s := range all {
				names = append(names, strings.TrimPrefix(s, ":cluster "))
			}
			if !existsIn(names, arg) {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown cluster: " + arg} }, true
			}
		case "namespace":
			all := m.autocompleteEngine.GetArgumentSuggestions("namespace", "", m.state)
			names := make([]string, 0, len(all))
			for _, s := range all {
				names = append(names, strings.TrimPrefix(s, ":namespace "))
			}
			if !existsIn(names, arg) {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown namespace: " + arg} }, true
			}
		case "project":
			all := m.autocompleteEngine.GetArgumentSuggestions("project", "", m.state)
			names := make([]string, 0, len(all))
			for _, s := range all {
				names = append(names, strings.TrimPrefix(s, ":project "))
			}
			if !existsIn(names, arg) {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown project: " + arg} }, true
			}
		case "app":
			ok := false
			for _, a := range m.state.Apps {
				if strings.EqualFold(a.Name, arg) {
					ok = true
					break
				}
			}
			if !ok {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown app: " + arg} }, true
			}
		}
	}

	m.inputComponents.BlurInputs()
	m.state.Mode = model.ModeNormal
	m.state.UI.Command = ""
	m.inputComponents.ClearCommandInput()

	// Reset navigation basics
	m.state.Navigation.SelectedIdx = 0
	m.state.UI.ActiveFilter = ""
	m.state.UI.SearchQuery = ""

	// Dispatch to command handler
	if handler, exists := m.commandRegistry.GetCommandHandler(canonical); exists {
		model, cmd := handler(canonical, arg)
		return model, cmd, true
	}

	// Unknown command
	return m, func() tea.Msg {
		return model.StatusChangeMsg{Status: "Unknown command: " + cmd}
	}, true
}

// parseCommand splits a command string into command and argument
func parseCommand(input string) (string, string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}