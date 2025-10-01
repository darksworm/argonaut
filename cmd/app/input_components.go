package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/v2/textinput"
	tea "github.com/charmbracelet/bubbletea/v2"
	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// InputComponentState manages interactive input components
type InputComponentState struct {
	searchInput  textinput.Model
	commandInput textinput.Model
}

// NewInputComponents creates a new input component state
func NewInputComponents() *InputComponentState {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search..."
	searchInput.CharLimit = 200
	searchInput.SetWidth(50)

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

	// Search bar with rounded border
	searchBarStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(yellowBright).
		PaddingLeft(1).
		PaddingRight(1)

	// Search label styling
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

	// Command bar with rounded border
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

// renderCommandInputWithAutocomplete renders the command input with dim autocomplete suggestions
func (m *Model) renderCommandInputWithAutocomplete(maxWidth int) string {
	currentInput := m.inputComponents.GetCommandValue()

	// The autocomplete engine expects a leading ':', but command mode doesn't include it
	// in the text input (it's only used to enter the mode). So prepend it for the query.
	query := currentInput
	if !strings.HasPrefix(query, ":") {
		query = ":" + query
	}

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
		dimSuggestion = lipgloss.NewStyle().Foreground(grayBorder).Render(suggestionSuffix)
	}

	promptStyle := lipgloss.NewStyle().Foreground(grayPrompt)
	prompt := promptStyle.Render("> ")
	content := prompt + inputText + dimSuggestion
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
		return m.handleNavigationUp()
	case "down", "j":
		return m.handleNavigationDown()
	case "esc":
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
	key := msg.String()

	// Use command registry for type-safe, extensible command handling
	if handler, exists := m.commandRegistry.GetKeyHandler(key); exists {
		if model, cmd, handled := handler(key); handled {
			return model, cmd
		}
	}

	// Default: let bubbles textinput handle the key
	cmd := m.inputComponents.UpdateCommandInput(msg)
	// Sync the command with the input value
	m.state.UI.Command = m.inputComponents.GetCommandValue()
	return m, cmd
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
	m.inputComponents.ClearCommandInput()
	m.inputComponents.FocusCommandInput()
	return m, nil
}

// local helpers
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
