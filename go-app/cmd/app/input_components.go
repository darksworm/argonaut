package main

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
    "github.com/a9s/go-app/pkg/model"
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
	searchInput.Width = 50

	// Create command input
	commandInput := textinput.New()
	commandInput.Placeholder = "Enter command..."
	commandInput.CharLimit = 200
	commandInput.Width = 50

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
func (m Model) renderEnhancedSearchBar() string {
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
	searchLabel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("Search")
	
    // Help text
    helpText := "Enter "
    if m.state.Navigation.View == model.ViewApps {
        helpText += "keeps filter"
    } else {
        helpText += "opens first result"
    }
    helpText += ", Esc cancels"

    // Compute widths to make input fill the full row
    totalWidth := m.state.Terminal.Cols
    styleWidth := maxInt(0, totalWidth-2) // account for main container left/right padding
    innerWidth := maxInt(0, styleWidth-4) // 2 borders + 2 padding (left/right)

    // Compute widths ensuring no wrap
    baseUsed := lipgloss.Width(searchLabel) + 1 /*space*/ + 2 /*two spaces before help*/
    // Provisional help width (plain, no ANSI), clipped to available space after input min
    minInput := 5
    maxInput := maxInt(minInput, innerWidth-baseUsed)
    // Start by allocating most space to input, then fit help in the rest
    inputWidth := maxInput
    helpPlain := "(" + helpText + ")"
    remaining := innerWidth - baseUsed - inputWidth
    if remaining < 0 { remaining = 0 }
    helpClipped := clipPlainToWidth(helpPlain, remaining)
    // If help was heavily clipped and we still have space, rebalance: reserve at least 8 cols for help
    if remaining == 0 && innerWidth-baseUsed-minInput > 8 {
        inputWidth = innerWidth - baseUsed - 8
        if inputWidth < minInput { inputWidth = minInput }
        remaining = innerWidth - baseUsed - inputWidth
        if remaining < 0 { remaining = 0 }
        helpClipped = clipPlainToWidth(helpPlain, remaining)
    }
    if inputWidth != m.inputComponents.searchInput.Width {
        m.inputComponents.searchInput.Width = inputWidth
    }

    // Render
    searchInputView := m.inputComponents.searchInput.View()
    helpRendered := statusStyle.Render(helpClipped)
    content := fmt.Sprintf("%s %s  %s", searchLabel, searchInputView, helpRendered)

    return searchBarStyle.Width(styleWidth).Render(content)
}

// renderEnhancedCommandBar renders an interactive command bar using bubbles textinput
func (m Model) renderEnhancedCommandBar() string {
    if m.state.Mode != model.ModeCommand {
        return ""
    }

    // Command bar with border (matches CommandBar Box with borderStyle="round" borderColor="yellow")
    commandBarStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(yellowBright).
        PaddingLeft(1).
        PaddingRight(1)

	// Content matching CommandBar layout
	cmdLabel := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14")).Render("CMD")
	colonPrefix := lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(":")
	
    helpText := "(Enter to run, Esc to cancel)"
    commandValue := m.inputComponents.GetCommandValue()
    if commandValue != "" {
        // TODO: Add command validation and hints
        helpText = "(Command: " + commandValue + ")"
    }

    // Compute widths for full-row input
    totalWidth := m.state.Terminal.Cols
    styleWidth := maxInt(0, totalWidth-2)
    innerWidth := maxInt(0, styleWidth-4)
    baseUsed := lipgloss.Width(cmdLabel) + 1 /*space*/ + lipgloss.Width(colonPrefix) + 2 /*two spaces*/
    minInput := 5
    maxInput := maxInt(minInput, innerWidth-baseUsed)
    inputWidth := maxInput
    // Compute remaining for help (plain text), clip to width
    remaining := innerWidth - baseUsed - inputWidth
    if remaining < 0 { remaining = 0 }
    helpClipped := clipPlainToWidth(helpText, remaining)
    // Rebalance if needed (reserve at least 8 cols for help when possible)
    if remaining == 0 && innerWidth-baseUsed-minInput > 8 {
        inputWidth = innerWidth - baseUsed - 8
        if inputWidth < minInput { inputWidth = minInput }
        remaining = innerWidth - baseUsed - inputWidth
        if remaining < 0 { remaining = 0 }
        helpClipped = clipPlainToWidth(helpText, remaining)
    }
    if inputWidth != m.inputComponents.commandInput.Width {
        m.inputComponents.commandInput.Width = inputWidth
    }

    // Render
    commandInputView := m.inputComponents.commandInput.View()
    helpRendered := statusStyle.Render(helpClipped)
    content := fmt.Sprintf("%s %s%s  %s", cmdLabel, colonPrefix, commandInputView, helpRendered)

    return commandBarStyle.Width(styleWidth).Render(content)
}

// Enhanced input handling for bubbles integration

// handleEnhancedSearchModeKeys handles input when in search mode with bubbles textinput
func (m Model) handleEnhancedSearchModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c":
        // Treat Ctrl+C as closing the input (do not quit app)
        m.inputComponents.BlurInputs()
        m.inputComponents.ClearSearchInput()
        m.state.Mode = model.ModeNormal
        m.state.UI.SearchQuery = ""
        return m, nil
    case "up", "k":
        // Navigate results while search is active
        return m.handleNavigationUp()
    case "down", "j":
        // Navigate results while search is active
        return m.handleNavigationDown()
    case "esc":
        m.inputComponents.BlurInputs()
        m.inputComponents.ClearSearchInput()
        m.state.Mode = model.ModeNormal
        m.state.UI.SearchQuery = ""
        return m, nil
    case "enter":
        // Apply search filter and exit search mode or drill down for non-app views
        searchValue := m.inputComponents.GetSearchValue()
        if m.state.Navigation.View == model.ViewApps {
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
        newModel.inputComponents.BlurInputs()
        newModel.state.Mode = model.ModeNormal
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
func (m Model) handleEnhancedCommandModeKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
    switch msg.String() {
    case "ctrl+c":
        // Treat Ctrl+C as closing the input (do not quit app)
        m.inputComponents.BlurInputs()
        m.inputComponents.ClearCommandInput()
        m.state.Mode = model.ModeNormal
        m.state.UI.Command = ""
        return m, nil
    case "esc":
        m.inputComponents.BlurInputs()
        m.inputComponents.ClearCommandInput()
        m.state.Mode = model.ModeNormal
        m.state.UI.Command = ""
		return m, nil
    case "enter":
        // Execute simple navigation commands (clusters/namespaces/projects/apps) with aliases
        raw := strings.TrimSpace(m.inputComponents.GetCommandValue())
        m.inputComponents.BlurInputs()
        m.state.Mode = model.ModeNormal
        m.state.UI.Command = ""
        m.inputComponents.ClearCommandInput()

        if raw == "" {
            return m, nil
        }

        parts := strings.Fields(raw)
        cmd := strings.ToLower(parts[0])
        arg := ""
        if len(parts) > 1 {
            arg = parts[1]
        }

        // Reset navigation basics
        m.state.Navigation.SelectedIdx = 0
        m.state.UI.ActiveFilter = ""
        m.state.UI.SearchQuery = ""

        switch cmd {
        case "cluster", "clusters", "cls":
            // Switch to clusters view
            m.state.Navigation.View = model.ViewClusters
            m.state.Selections.SelectedApps = model.NewStringSet()
            if arg != "" {
                // Set cluster scope and advance to namespaces
                m.state.Selections.ScopeClusters = model.StringSetFromSlice([]string{arg})
                m.state.Navigation.View = model.ViewNamespaces
            } else {
                m.state.Selections.ScopeClusters = model.NewStringSet()
            }
            return m, nil
        case "namespace", "namespaces", "ns":
            m.state.Navigation.View = model.ViewNamespaces
            m.state.Selections.SelectedApps = model.NewStringSet()
            if arg != "" {
                m.state.Selections.ScopeNamespaces = model.StringSetFromSlice([]string{arg})
                m.state.Navigation.View = model.ViewProjects
            } else {
                m.state.Selections.ScopeNamespaces = model.NewStringSet()
            }
            return m, nil
        case "project", "projects", "proj":
            m.state.Navigation.View = model.ViewProjects
            m.state.Selections.SelectedApps = model.NewStringSet()
            if arg != "" {
                m.state.Selections.ScopeProjects = model.StringSetFromSlice([]string{arg})
                m.state.Navigation.View = model.ViewApps
            } else {
                m.state.Selections.ScopeProjects = model.NewStringSet()
            }
            return m, nil
        case "app", "apps":
            m.state.Navigation.View = model.ViewApps
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
        default:
            // Unknown: set status for feedback
            return m, func() tea.Msg { return model.StatusChangeMsg{Status: "Unknown command: " + raw} }
        }
	default:
		// Let bubbles textinput handle the key
		cmd := m.inputComponents.UpdateCommandInput(msg)
		// Sync the command with the input value
		m.state.UI.Command = m.inputComponents.GetCommandValue()
		return m, cmd
	}
}

// Enhanced mode entry handlers that activate bubbles inputs

// handleEnhancedEnterSearchMode switches to search mode and activates textinput
func (m Model) handleEnhancedEnterSearchMode() (Model, tea.Cmd) {
	m.state.Mode = model.ModeSearch
	m.state.UI.SearchQuery = ""
	m.inputComponents.ClearSearchInput()
	m.inputComponents.FocusSearchInput()
	return m, nil
}

// handleEnhancedEnterCommandMode switches to command mode and activates textinput
func (m Model) handleEnhancedEnterCommandMode() (Model, tea.Cmd) {
    m.state.Mode = model.ModeCommand
    m.state.UI.Command = ""
    m.inputComponents.ClearCommandInput()
    m.inputComponents.FocusCommandInput()
    return m, nil
}

// local helpers
func maxInt(a, b int) int {
    if a > b { return a }
    return b
}

// clipPlainToWidth trims a plain (non-ANSI) string to the given display width
func clipPlainToWidth(s string, width int) string {
    if width <= 0 { return "" }
    w := 0
    out := make([]rune, 0, len(s))
    for _, r := range s {
        rw := 1 // assume width 1 for TUI plain text
        if w+rw > width { break }
        out = append(out, r)
        w += rw
    }
    return string(out)
}
