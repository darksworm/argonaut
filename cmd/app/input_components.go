package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	cblog "github.com/charmbracelet/log"
	"github.com/darksworm/argonaut/pkg/config"
	"github.com/darksworm/argonaut/pkg/model"
	"github.com/darksworm/argonaut/pkg/theme"
	"github.com/darksworm/argonaut/pkg/tui/treeview"
)

// InputComponentState manages interactive input components
type InputComponentState struct {
	searchInput  textinput.Model
	commandInput textinput.Model
}

// NewInputComponents creates a new input component state
func NewInputComponents() *InputComponentState {
	// Create search input
	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."
	searchInput.CharLimit = 200
	searchInput.SetWidth(50)

	// Create command input
	commandInput := textinput.New()
	commandInput.Placeholder = "Enter command..."
	commandInput.CharLimit = 200
	commandInput.SetWidth(50)

	return &InputComponentState{
		searchInput:  searchInput,
		commandInput: commandInput,
	}
}

// UpdateSearchInput updates the search textinput component
func (ic *InputComponentState) UpdateSearchInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	ic.searchInput, cmd = ic.searchInput.Update(msg)
	return cmd
}

// UpdateCommandInput updates the command textinput component
func (ic *InputComponentState) UpdateCommandInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	ic.commandInput, cmd = ic.commandInput.Update(msg)
	return cmd
}

// FocusSearchInput focuses the search input
func (ic *InputComponentState) FocusSearchInput() {
	ic.searchInput.Focus()
}

// FocusCommandInput focuses the command input
func (ic *InputComponentState) FocusCommandInput() {
	ic.commandInput.Focus()
}

// BlurInputs removes focus from all inputs
func (ic *InputComponentState) BlurInputs() {
	ic.searchInput.Blur()
	ic.commandInput.Blur()
}

// GetSearchValue returns current search input value
func (ic *InputComponentState) GetSearchValue() string {
	return ic.searchInput.Value()
}

// GetCommandValue returns current command input value
func (ic *InputComponentState) GetCommandValue() string {
	return ic.commandInput.Value()
}

// SetSearchValue sets the search input value
func (ic *InputComponentState) SetSearchValue(value string) {
	ic.searchInput.SetValue(value)
}

// SetCommandValue sets the command input value
func (ic *InputComponentState) SetCommandValue(value string) {
	ic.commandInput.SetValue(value)
}

// ClearSearchInput clears the search input
func (ic *InputComponentState) ClearSearchInput() {
	ic.searchInput.SetValue("")
}

// ClearCommandInput clears the command input
func (ic *InputComponentState) ClearCommandInput() {
	ic.commandInput.SetValue("")
}

// Enhanced view functions that use bubbles textinput

// renderEnhancedSearchBar renders an interactive search bar using bubbles textinput
func (m *Model) renderEnhancedSearchBar() string {
	if m.state.Mode != model.ModeSearch {
		return ""
	}

	// Search bar with border (matches SearchBar Box with borderStyle="round" borderColor="yellow")
	searchBarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		PaddingLeft(1).
		PaddingRight(1)

	// Content matching SearchBar layout
	searchLabel := lipgloss.NewStyle().Bold(true).Foreground(cyanBright).Render("Search")

	// Compute widths to make input fill the full row (no trailing help text)
	totalWidth := m.state.Terminal.Cols
	// Make the OUTER width match the main bordered box outer width (cols-2)
	// Inner content width is then outer - borders(2) - padding(2) = cols-6
	styleWidth := maxInt(0, totalWidth-2)
	innerWidth := maxInt(0, styleWidth-4)

	// Allocate remaining width to the input field
	baseUsed := lipgloss.Width(searchLabel) + 1 /*space*/
	minInput := 5
	inputWidth := maxInt(minInput, innerWidth-baseUsed)
	if inputWidth != m.inputComponents.searchInput.Width() {
		m.inputComponents.searchInput.SetWidth(inputWidth)
	}

	// Render
	searchInputView := m.inputComponents.searchInput.View()
	content := fmt.Sprintf("%s %s", searchLabel, searchInputView)

	return searchBarStyle.Width(styleWidth).Render(content)
}

// renderEnhancedCommandBar renders an interactive command bar using bubbles textinput
func (m *Model) renderEnhancedCommandBar() string {
	if m.state.Mode != model.ModeCommand {
		return ""
	}

	// Command bar with border (matches CommandBar Box with borderStyle="round" borderColor="yellow")
	commandBarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		PaddingLeft(1).
		PaddingRight(1)

	// Compute widths for full-row input (no label, fill full width)
	totalWidth := m.state.Terminal.Cols
	// Match OUTER width of main content border (cols-2); inner width = cols-6
	styleWidth := maxInt(0, totalWidth-2)
	innerWidth := maxInt(0, styleWidth-4)
	minInput := 5
	inputWidth := maxInt(minInput, innerWidth)
	if inputWidth != m.inputComponents.commandInput.Width() {
		m.inputComponents.commandInput.SetWidth(inputWidth)
	}

	// Render with autocomplete suggestions
	commandInputView := m.renderCommandInputWithAutocomplete(inputWidth)
	return commandBarStyle.Width(styleWidth).Render(commandInputView)
}

// validateCommand checks if a command is valid without executing it
func (m *Model) validateCommand(input string) bool {
	if input == "" {
		return true // Empty is neutral, not invalid
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return true
	}

	cmd := strings.ToLower(parts[0])
	canonical := m.autocompleteEngine.ResolveAlias(cmd)

	// Check if command exists
	if m.autocompleteEngine.GetCommandInfo(canonical) == nil {
		return false
	}

	// If command takes an argument and one is provided, validate it
	if len(parts) >= 2 {
		arg := parts[1]
		switch canonical {
		case "cluster":
			all := m.autocompleteEngine.GetArgumentSuggestions("cluster", "", m.state)
			names := make([]string, 0, len(all))
			for _, s := range all {
				names = append(names, strings.TrimPrefix(s, ":cluster "))
			}
			for _, name := range names {
				if strings.EqualFold(name, arg) {
					return true
				}
			}
			return false
		case "namespace":
			all := m.autocompleteEngine.GetArgumentSuggestions("namespace", "", m.state)
			names := make([]string, 0, len(all))
			for _, s := range all {
				names = append(names, strings.TrimPrefix(s, ":namespace "))
			}
			for _, name := range names {
				if strings.EqualFold(name, arg) {
					return true
				}
			}
			return false
		case "project":
			all := m.autocompleteEngine.GetArgumentSuggestions("project", "", m.state)
			names := make([]string, 0, len(all))
			for _, s := range all {
				names = append(names, strings.TrimPrefix(s, ":project "))
			}
			for _, name := range names {
				if strings.EqualFold(name, arg) {
					return true
				}
			}
			return false
		case "app", "delete", "sync", "diff", "rollback", "resources":
			for _, a := range m.state.Apps {
				if strings.EqualFold(a.Name, arg) {
					return true
				}
			}
			return false
		case "theme":
			themeNames := theme.GetAvailableThemes()
			for _, themeName := range themeNames {
				if strings.EqualFold(themeName, arg) {
					return true
				}
			}
			return false
		}
	}

	return true
}

// renderCommandInputWithAutocomplete renders the command input with dim autocomplete suggestions
func (m *Model) renderCommandInputWithAutocomplete(maxWidth int) string {
	currentInput := m.inputComponents.GetCommandValue()

	// Build query for the autocomplete engine. The engine expects a leading ':',
	// but our command mode does not include ':' in the text input; ':' is only
	// used to enter the mode. So prepend it for querying suggestions.
	query := currentInput
	if !strings.HasPrefix(query, ":") {
		query = ":" + query
	}

	// Get autocomplete suggestions
	suggestions := m.autocompleteEngine.GetCommandAutocomplete(query, m.state)
	var firstPlain string
	if len(suggestions) > 0 {
		firstPlain = strings.TrimPrefix(suggestions[0], ":")
	}

	// Style the current input, colorizing the argument validity for known commands
	inputText := currentInput
	parts := strings.Fields(currentInput)
	if len(parts) >= 1 {
		cmdWord := strings.ToLower(parts[0])
		canonical := m.autocompleteEngine.ResolveAlias(cmdWord)
		if info := m.autocompleteEngine.GetCommandInfo(canonical); info != nil && info.TakesArg && len(parts) >= 2 {
			arg := parts[1]
			all := m.autocompleteEngine.GetArgumentSuggestions(canonical, "", m.state)
			valid := false
			for _, s := range all {
				cand := strings.TrimPrefix(s, ":"+canonical+" ")
				if strings.EqualFold(cand, arg) {
					valid = true
					break
				}
			}
			argStyle := lipgloss.NewStyle().Foreground(outOfSyncColor) // red
			if valid {
				argStyle = lipgloss.NewStyle().Foreground(cyanBright)
			} // blue
			if strings.Contains(currentInput, " ") {
				rest := ""
				if len(parts) > 2 {
					rest = " " + strings.Join(parts[2:], " ")
				}
				inputText = parts[0] + " " + argStyle.Render(arg) + rest
			}
		}
	}

	// Optional dim suggestion suffix
	dimSuggestion := ""
	if firstPlain != "" && len(firstPlain) > len(currentInput) && strings.HasPrefix(strings.ToLower(firstPlain), strings.ToLower(currentInput)) {
		suggestionSuffix := firstPlain[len(currentInput):]
		dimSuggestion = lipgloss.NewStyle().Foreground(dimColor).Render(suggestionSuffix)
	}

	// Determine prompt symbol and style based on command validity
	var prompt string
	if m.state.UI.CommandInvalid {
		// Red warning triangle for invalid commands
		promptStyle := lipgloss.NewStyle().Foreground(outOfSyncColor) // red
		prompt = promptStyle.Render("⚠ ")
	} else {
		// Normal prompt
		promptStyle := lipgloss.NewStyle().Foreground(dimColor) // light gray
		prompt = promptStyle.Render("> ")
	}

	// Add invalid command message if needed
	invalidMessage := ""
	if m.state.UI.CommandInvalid && currentInput != "" {
		availableWidth := maxWidth - lipgloss.Width(prompt) - lipgloss.Width(inputText) - lipgloss.Width(dimSuggestion)
		if availableWidth > 20 { // Only show if there's enough space
			messageStyle := lipgloss.NewStyle().Foreground(dimColor) // dim gray
			invalidMessage = messageStyle.Render(" (unknown command, see :help)")
		}
	}

	// Combine all parts
	content := prompt + inputText + dimSuggestion + invalidMessage
	if w := lipgloss.Width(content); w < maxWidth {
		content += strings.Repeat(" ", maxWidth-w)
	}
	return content
}

// Enhanced input handling for bubbles integration

// handleEnhancedSearchModeKeys handles input when in search mode with bubbles textinput
func (m *Model) handleEnhancedSearchModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		// Treat Ctrl+C as closing the input (do not quit app)
		m.inputComponents.BlurInputs()
		m.inputComponents.ClearSearchInput()
		if m.state.Diff != nil {
			m.state.Mode = model.ModeDiff
		} else {
			m.state.Mode = model.ModeNormal
			m.state.UI.SearchQuery = ""
		}
		return m, nil
	case "up", "k":
		// Navigate results while search is active
		return m.handleNavigationUp()
	case "down", "j":
		// Navigate results while search is active
		return m.handleNavigationDown()
	case "esc":
		// Exit search; if coming from diff mode, return to diff; else normal
		m.inputComponents.BlurInputs()
		m.inputComponents.ClearSearchInput()
		if m.state.Diff != nil {
			m.state.Mode = model.ModeDiff
		} else {
			m.state.Mode = model.ModeNormal
			m.state.UI.SearchQuery = ""
		}
		return m, nil
	case "enter":
		// Apply search filter and exit search mode or drill down for non-app views
		searchValue := m.inputComponents.GetSearchValue()
		if m.state.Mode == model.ModeDiff {
			// Apply filter to diff view
			if m.state.Diff != nil {
				m.state.Diff.SearchQuery = searchValue
				m.state.Diff.Offset = 0
			}
			m.inputComponents.BlurInputs()
			m.state.Mode = model.ModeDiff
			return m, nil
		} else if m.state.Navigation.View == model.ViewApps {
			// Keep filter applied in apps view
			m.inputComponents.BlurInputs()
			m.state.Mode = model.ModeNormal
			m.state.UI.SearchQuery = searchValue
			m.state.UI.ActiveFilter = searchValue
			m.state.Navigation.SelectedIdx = 0
			return m, nil
		}
		// For other views, drill down using current filtered results
		// Do NOT exit search mode until after drill-down so filtering remains active
		m.state.UI.SearchQuery = searchValue
		// Perform drill-down based on current selection under active search filter
		newModel, cmd := m.handleDrillDown()
		if modelPtr, ok := newModel.(*Model); ok {
			modelPtr.inputComponents.BlurInputs()
			modelPtr.state.Mode = model.ModeNormal
			return modelPtr, cmd
		}
		return newModel, cmd
	default:
		// Let bubbles textinput handle the key
		cmd := m.inputComponents.UpdateSearchInput(msg)
		// Sync the search query with the input value
		m.state.UI.SearchQuery = m.inputComponents.GetSearchValue()
		// Clamp selection within new filtered results
		m.state.Navigation.SelectedIdx = m.navigationService.ValidateBounds(
			m.state.Navigation.SelectedIdx,
			len(m.getVisibleItems()),
		)
		return m, cmd
	}
}

// handleEnhancedCommandModeKeys handles input when in command mode with bubbles textinput
func (m *Model) handleEnhancedCommandModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		// Treat Ctrl+C as closing the input (do not quit app)
		m.inputComponents.BlurInputs()
		m.inputComponents.ClearCommandInput()
		m.state.Mode = model.ModeNormal
		m.state.UI.Command = ""
		m.state.UI.CommandInvalid = false
		return m, nil
	case "esc":
		m.inputComponents.BlurInputs()
		m.inputComponents.ClearCommandInput()
		m.state.Mode = model.ModeNormal
		m.state.UI.Command = ""
		m.state.UI.CommandInvalid = false
		return m, nil
	case "tab":
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
		return m, nil
	case "enter":
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
			return m, nil
		}

		// Validate command before proceeding
		if !m.validateCommand(raw) {
			// Mark as invalid and stay in command mode
			m.state.UI.CommandInvalid = true
			return m, nil
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
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown cluster: " + arg} }
				}
			case "namespace":
				all := m.autocompleteEngine.GetArgumentSuggestions("namespace", "", m.state)
				names := make([]string, 0, len(all))
				for _, s := range all {
					names = append(names, strings.TrimPrefix(s, ":namespace "))
				}
				if !existsIn(names, arg) {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown namespace: " + arg} }
				}
			case "project":
				all := m.autocompleteEngine.GetArgumentSuggestions("project", "", m.state)
				names := make([]string, 0, len(all))
				for _, s := range all {
					names = append(names, strings.TrimPrefix(s, ":project "))
				}
				if !existsIn(names, arg) {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown project: " + arg} }
				}
			case "app", "delete":
				ok := false
				for _, a := range m.state.Apps {
					if strings.EqualFold(a.Name, arg) {
						ok = true
						break
					}
				}
				if !ok {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown app: " + arg} }
				}
			}
		}

		m.inputComponents.BlurInputs()
		m.state.Mode = model.ModeNormal
		m.state.UI.Command = ""
		m.state.UI.CommandInvalid = false
		m.inputComponents.ClearCommandInput()

		// Clear UI state for all commands
		m.state.UI.ActiveFilter = ""
		m.state.UI.SearchQuery = ""

		// IMPORTANT: When adding new commands here, also add them to pkg/autocomplete/autocomplete.go
		// to ensure they appear in autocomplete and validation works correctly.
		switch canonical {
		case "logs":
			// Open logs using the configured log file (via ARGONAUT_LOG_FILE) with a sensible fallback.
			// Reuse the view helper so behavior matches the Logs view.
			body := m.readLogContent()
			return m, m.openTextPager("Logs", body)
		case "sync":
			mdl, cmd := m.handleSyncModal()
			return mdl, cmd
		case "delete", "del":
			target := arg
			if target == "" {
				// Check if we're in apps view
				if m.state.Navigation.View != model.ViewApps {
					return m, func() tea.Msg {
						return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for deletion"}
					}
				}

				// Use the same logic as handleAppDelete() - check for multi-selection first
				if len(m.state.Selections.SelectedApps) == 0 {
					// No apps selected, use current cursor position
					items := m.getVisibleItemsForCurrentView()
					if len(items) > 0 && m.state.Navigation.SelectedIdx < len(items) {
						if app, ok := items[m.state.Navigation.SelectedIdx].(model.App); ok {
							target = app.Name
						}
					}
					if target == "" {
						return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for deletion"} }
					}

					// Find the app object to get namespace
					var targetApp *model.App
					for _, app := range m.state.Apps {
						if strings.EqualFold(app.Name, target) {
							targetApp = &app
							break
						}
					}
					if targetApp == nil {
						return m, func() tea.Msg { return model.StatusChangeMsg{Status: "App not found: " + target} }
					}

					// Single app deletion
					cblog.With("component", "app-delete").Debug(":delete command invoked", "app", target)
					m.state.Mode = model.ModeConfirmAppDelete
					m.state.Modals.DeleteAppName = &target
					m.state.Modals.DeleteAppNamespace = targetApp.Namespace
					m.state.Modals.DeleteConfirmationKey = ""
					m.state.Modals.DeleteError = nil
					m.state.Modals.DeleteLoading = false
					m.state.Modals.DeleteCascade = true // Default to cascade
					m.state.Modals.DeletePropagationPolicy = "foreground"
					return m, nil
				} else {
					// Multiple apps selected - use multi-delete logic
					cblog.With("component", "app-delete").Debug(":delete command invoked for multi-selection", "count", len(m.state.Selections.SelectedApps))
					multiTarget := "__MULTI__"
					m.state.Mode = model.ModeConfirmAppDelete
					m.state.Modals.DeleteAppName = &multiTarget
					m.state.Modals.DeleteAppNamespace = nil // Not applicable for multi-delete
					m.state.Modals.DeleteConfirmationKey = ""
					m.state.Modals.DeleteError = nil
					m.state.Modals.DeleteLoading = false
					m.state.Modals.DeleteCascade = true // Default to cascade
					m.state.Modals.DeletePropagationPolicy = "foreground"
					return m, nil
				}
			} else {
				// Specific app name provided as argument
				// Find the app object to get namespace
				var targetApp *model.App
				for _, app := range m.state.Apps {
					if strings.EqualFold(app.Name, target) {
						targetApp = &app
						break
					}
				}
				if targetApp == nil {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "App not found: " + target} }
				}

				// Single app deletion
				cblog.With("component", "app-delete").Debug(":delete command invoked", "app", target)
				m.state.Mode = model.ModeConfirmAppDelete
				m.state.Modals.DeleteAppName = &target
				m.state.Modals.DeleteAppNamespace = targetApp.Namespace
				m.state.Modals.DeleteConfirmationKey = ""
				m.state.Modals.DeleteError = nil
				m.state.Modals.DeleteLoading = false
				m.state.Modals.DeleteCascade = true // Default to cascade
				m.state.Modals.DeletePropagationPolicy = "foreground"
				return m, nil
			}
		case "rollback":
			target := arg
			if target == "" {
				// Only try to get current selection if we're in the apps view
				if m.state.Navigation.View == model.ViewApps {
					items := m.getVisibleItemsForCurrentView()
					if len(items) > 0 && m.state.Navigation.SelectedIdx < len(items) {
						if app, ok := items[m.state.Navigation.SelectedIdx].(model.App); ok {
							target = app.Name
						}
					}
				} else {
					return m, func() tea.Msg {
						return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for rollback"}
					}
				}
			}
			if target == "" {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for rollback"} }
			}

			// Use the same rollback logic as the R key
			cblog.With("component", "rollback").Debug(":rollback command invoked", "app", target)
			m.state.Modals.RollbackAppName = &target
			m.state.Mode = model.ModeRollback

			// Initialize rollback state with loading
			m.state.Rollback = &model.RollbackState{
				AppName: target,
				Loading: true,
				Mode:    "list",
			}

			// Start loading rollback history using the same function as R key
			return m, m.startRollbackSession(target)
		case "resources", "res", "r":
			target := arg

			// If no explicit target provided, check for multiple selections first (like 'r' key does)
			if target == "" {
				sel := m.state.Selections.SelectedApps
				names := make([]string, 0, len(sel))
				for name, ok := range sel {
					if ok {
						names = append(names, name)
					}
				}

				if len(names) > 1 {
					// Clean up any existing tree watchers before starting new ones
					m.cleanupTreeWatchers()
					// Multiple apps selected - open multi tree view with live updates
					m.treeView = treeview.NewTreeView(0, 0)
					m.treeView.ApplyTheme(currentPalette)
					m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)
					m.treeScrollOffset = 0 // Reset scroll position
					m.state.SaveNavigationState()
					m.state.Navigation.View = model.ViewTree
					m.state.UI.TreeAppName = nil
					m.treeLoading = true
					var cmds []tea.Cmd
					for _, n := range names {
						var appObj *model.App
						for i := range m.state.Apps {
							if m.state.Apps[i].Name == n {
								appObj = &m.state.Apps[i]
								break
							}
						}
						if appObj == nil {
							tmp := model.App{Name: n}
							appObj = &tmp
						}
						cmds = append(cmds, m.startLoadingResourceTree(*appObj))
						cmds = append(cmds, m.startWatchingResourceTree(*appObj))
					}
					cmds = append(cmds, m.consumeTreeEvent())
					return m, tea.Batch(cmds...)
				} else if len(names) == 1 {
					// Single app selected via checkbox
					target = names[0]
				} else {
					// No apps selected via checkbox, try cursor position
					if m.state.Navigation.View == model.ViewApps {
						items := m.getVisibleItemsForCurrentView()
						if len(items) > 0 && m.state.Navigation.SelectedIdx < len(items) {
							if app, ok := items[m.state.Navigation.SelectedIdx].(model.App); ok {
								target = app.Name
							}
						}
					} else {
						return m, func() tea.Msg {
							return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for resources"}
						}
					}
				}
			}

			if target == "" {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for resources"} }
			}
			// Single app: open tree view with watch (reset tree view)
			m.treeView = treeview.NewTreeView(0, 0)
			m.treeView.ApplyTheme(currentPalette)
			m.treeView.SetSize(m.state.Terminal.Cols, m.state.Terminal.Rows)
			m.treeScrollOffset = 0 // Reset scroll position
			m.state.SaveNavigationState()
			var selectedApp *model.App
			for i := range m.state.Apps {
				if m.state.Apps[i].Name == target {
					selectedApp = &m.state.Apps[i]
					break
				}
			}
			if selectedApp == nil {
				selectedApp = &model.App{Name: target}
			}
			// Clean up any existing tree watchers before starting new one
			m.cleanupTreeWatchers()
			m.state.Navigation.View = model.ViewTree
			m.state.UI.TreeAppName = &target
			m.treeLoading = true
			return m, tea.Batch(m.startLoadingResourceTree(*selectedApp), m.startWatchingResourceTree(*selectedApp), m.consumeTreeEvent())
		case "all":
			m.state.Selections = *model.NewSelectionState()
			m.state.UI.SearchQuery = ""
			m.state.UI.ActiveFilter = ""
			return m, func() tea.Msg { return model.StatusChangeMsg{Status: "All filtering cleared."} }
		case "up":
			return m.handleEscape()
		case "diff":
			// :diff [app]
			target := arg
			if target == "" {
				// Only try to get current selection if we're in the apps view
				if m.state.Navigation.View == model.ViewApps {
					items := m.getVisibleItemsForCurrentView()
					if len(items) > 0 && m.state.Navigation.SelectedIdx < len(items) {
						if app, ok := items[m.state.Navigation.SelectedIdx].(model.App); ok {
							target = app.Name
						}
					}
				} else {
					return m, func() tea.Msg {
						return model.StatusChangeMsg{Status: "Navigate to apps view first to select an app for diff"}
					}
				}
			}
			if target == "" {
				return m, func() tea.Msg { return model.StatusChangeMsg{Status: "No app selected for diff"} }
			}
			// Initialize diff state with loading
			if m.state.Diff == nil {
				m.state.Diff = &model.DiffState{}
			}
			m.state.Diff.Loading = true
			return m, m.startDiffSession(target)
		case "cluster", "clusters", "cls":
			// Exit deep views and clear lower-level scopes
			m.state.UI.TreeAppName = nil
			m.treeLoading = false
			m.state.Selections.SelectedApps = model.NewStringSet()
			m.state.Navigation.SelectedIdx = 0 // Reset navigation for view change
			m = m.safeChangeView(model.ViewClusters)
			if arg != "" {
				// Validate cluster exists
				all := m.autocompleteEngine.GetArgumentSuggestions("cluster", "", m.state)
				names := make([]string, 0, len(all))
				for _, s := range all {
					names = append(names, strings.TrimPrefix(s, ":cluster "))
				}
				matched := false
				for _, n := range names {
					if strings.EqualFold(n, arg) {
						arg = n
						matched = true
						break
					}
				}
				if !matched {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown cluster: " + arg} }
				}
				m.state.Selections.ScopeClusters = model.StringSetFromSlice([]string{arg})
				m.state.Selections.ScopeNamespaces = model.NewStringSet()
				m.state.Selections.ScopeProjects = model.NewStringSet()
				m = m.safeChangeView(model.ViewNamespaces)
			} else {
				m.state.Selections.ScopeClusters = model.NewStringSet()
				m.state.Selections.ScopeNamespaces = model.NewStringSet()
				m.state.Selections.ScopeProjects = model.NewStringSet()
			}
			return m, nil
		case "namespace", "namespaces", "ns":
			m.state.UI.TreeAppName = nil
			m.treeLoading = false
			m.state.Navigation.SelectedIdx = 0 // Reset navigation for view change
			m = m.safeChangeView(model.ViewNamespaces)
			m.state.Selections.SelectedApps = model.NewStringSet()
			if arg != "" {
				all := m.autocompleteEngine.GetArgumentSuggestions("namespace", "", m.state)
				names := make([]string, 0, len(all))
				for _, s := range all {
					names = append(names, strings.TrimPrefix(s, ":namespace "))
				}
				matched := false
				for _, n := range names {
					if strings.EqualFold(n, arg) {
						arg = n
						matched = true
						break
					}
				}
				if !matched {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown namespace: " + arg} }
				}
				m.state.Selections.ScopeNamespaces = model.StringSetFromSlice([]string{arg})
				m.state.Selections.ScopeProjects = model.NewStringSet()
				m = m.safeChangeView(model.ViewProjects)
			} else {
				m.state.Selections.ScopeNamespaces = model.NewStringSet()
				m.state.Selections.ScopeProjects = model.NewStringSet()
			}
			return m, nil
		case "project", "projects", "proj":
			m.state.UI.TreeAppName = nil
			m.treeLoading = false
			m.state.Navigation.SelectedIdx = 0 // Reset navigation for view change
			m = m.safeChangeView(model.ViewProjects)
			m.state.Selections.SelectedApps = model.NewStringSet()
			if arg != "" {
				all := m.autocompleteEngine.GetArgumentSuggestions("project", "", m.state)
				names := make([]string, 0, len(all))
				for _, s := range all {
					names = append(names, strings.TrimPrefix(s, ":project "))
				}
				matched := false
				for _, n := range names {
					if strings.EqualFold(n, arg) {
						arg = n
						matched = true
						break
					}
				}
				if !matched {
					return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown project: " + arg} }
				}
				m.state.Selections.ScopeProjects = model.StringSetFromSlice([]string{arg})
				m = m.safeChangeView(model.ViewApps)
			} else {
				m.state.Selections.ScopeProjects = model.NewStringSet()
			}
			return m, nil
		case "app", "apps":
			m.state.Navigation.SelectedIdx = 0 // Reset navigation for view change
			m = m.safeChangeView(model.ViewApps)
			if arg != "" {
				// Select the app and move cursor to it if found
				m.state.Selections.SelectedApps = model.StringSetFromSlice([]string{arg})
				idx := -1
				for i, a := range m.state.Apps {
					if a.Name == arg {
						idx = i
						break
					}
				}
				if idx >= 0 {
					m.state.Navigation.SelectedIdx = idx
				}
			} else {
				m.state.Selections.SelectedApps = model.NewStringSet()
			}
			return m, nil
		case "help":
			// Show help modal
			m.state.Mode = model.ModeHelp
			return m, nil
		case "theme":
			return m.handleThemeCommand(arg)
		case "quit", "q", "q!", "wq", "wq!", "exit":
			// Exit the application
			return m, func() tea.Msg { return model.QuitMsg{} }
		case "upgrade", "update":
			// Trigger upgrade process
			return m, func() tea.Msg { return model.UpgradeRequestedMsg{} }
		default:
			// Unknown: set status for feedback
			return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown command: " + raw} }
		}
	default:
		// Let bubbles textinput handle the key
		cmd := m.inputComponents.UpdateCommandInput(msg)
		// Sync the command with the input value
		m.state.UI.Command = m.inputComponents.GetCommandValue()
		// Clear invalid flag when user types (any change resets the warning)
		m.state.UI.CommandInvalid = false
		return m, cmd
	}
}

// Enhanced mode entry handlers that activate bubbles inputs

// handleEnhancedEnterSearchMode switches to search mode and activates textinput
func (m *Model) handleEnhancedEnterSearchMode() (tea.Model, tea.Cmd) {
	m.state.Mode = model.ModeSearch
	m.state.UI.SearchQuery = ""
	m.inputComponents.ClearSearchInput()
	m.inputComponents.FocusSearchInput()
	return m, nil
}

// handleEnhancedEnterCommandMode switches to command mode and activates textinput
func (m *Model) handleEnhancedEnterCommandMode() (tea.Model, tea.Cmd) {
	m.state.Mode = model.ModeCommand
	m.state.UI.Command = ""
	m.state.UI.CommandInvalid = false
	m.inputComponents.ClearCommandInput()
	m.inputComponents.FocusCommandInput()
	return m, nil
}

// handleThemeCommand handles the :theme command for switching UI themes
func (m *Model) handleThemeCommand(arg string) (*Model, tea.Cmd) {
	// Load current config for both selection and application paths
	argonautConfig, err := config.LoadArgonautConfig()
	if err != nil {
		cblog.Warn("Could not load config, using defaults", "err", err)
		argonautConfig = config.GetDefaultConfig()
	}

	// Build theme options
	m.themeOptions = buildThemeOptions()

	if arg == "" {
		// Clear command input state first
		m.inputComponents.BlurInputs()
		m.inputComponents.ClearCommandInput()
		m.state.UI.Command = ""

		// Switch to theme selection mode
		m.state.Mode = model.ModeTheme

		// Set initial selection to current theme
		currentTheme := argonautConfig.Appearance.Theme
		if currentTheme == "" {
			currentTheme = config.DefaultThemeName
		}

		// Store original theme so we can restore it if cancelled
		m.state.UI.ThemeOriginalName = currentTheme

		selectedIndex := 0
		for i, option := range m.themeOptions {
			if option.Name == currentTheme {
				selectedIndex = i
				break
			}
		}
		if len(m.themeOptions) > 0 && selectedIndex >= len(m.themeOptions) {
			selectedIndex = len(m.themeOptions) - 1
		}
		m.state.UI.ThemeSelectedIndex = selectedIndex

		// Initialize scroll offset to show the selected theme
		m.adjustThemeScrollOffset()

		return m, nil
	}

	// Validate theme name
	if _, ok := theme.Get(arg); !ok {
		return m, func() tea.Msg {
			return model.StatusChangeMsg{Status: "Unknown theme: " + arg + ". Type ':theme' to see available themes."}
		}
	}

	// Update theme in config
	argonautConfig.Appearance.Theme = arg

	// Save the updated config
	if err := config.SaveArgonautConfig(argonautConfig); err != nil {
		cblog.Error("Failed to save config", "err", err)
		return m, func() tea.Msg {
			return model.StatusChangeMsg{Status: "Failed to save theme configuration: " + err.Error()}
		}
	}

	// Apply the new theme immediately
	palette := theme.FromConfig(argonautConfig)
	applyTheme(palette)
	m.applyThemeToModel()

	return m, nil
}

// local helpers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
