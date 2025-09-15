package main

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss/v2"
)

func (m Model) renderHelpModal() string {
    var sections []string
    header := m.renderBanner()
    sections = append(sections, header)

    isWide := m.state.Terminal.Cols >= 60
    var helpSections []string

    generalContent := ": command • / search • ? help"
    helpSections = append(helpSections, m.renderHelpSection("GENERAL", generalContent, isWide))

    navContent := "j/k up/down • Space select • Enter drill down • Esc clear/up"
    helpSections = append(helpSections, m.renderHelpSection("NAV", navContent, isWide))

    viewsContent := ":cls|:clusters|:cluster • :ns|:namespaces|:namespace\n:proj|:projects|:project • :apps"
    helpSections = append(helpSections, m.renderHelpSection("VIEWS", viewsContent, isWide))

    actionsContent := ":diff [app] • :sync [app] • :rollback [app]\n:resources [app] • :up go up level\ns sync modal • R rollback modal (apps view)"
    helpSections = append(helpSections, m.renderHelpSection("ACTIONS", actionsContent, isWide))

    miscContent := ":all • :help • :logs • :q"
    helpSections = append(helpSections, m.renderHelpSection("MISC", miscContent, isWide))

    helpSections = append(helpSections, "")
    helpSections = append(helpSections, statusStyle.Render("Press ?, q or Esc to close"))

    helpContent := strings.Join(helpSections, "\n")
    body := "\n" + helpContent + "\n"
    return m.renderFullScreenViewWithOptions(header, body, m.renderStatusLine(), FullScreenViewOptions{
        ContentBordered: true,
        BorderColor:     magentaBright,
    })
}

func (m Model) renderDiffLoadingSpinner() string {
    spinnerContent := fmt.Sprintf("%s Loading diff...", m.spinner.View())
    spinnerStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(yellowBright).
        Background(lipgloss.Color("0")).
        Foreground(whiteBright).
        Padding(1, 2).
        Bold(true).
        Align(lipgloss.Center)
    outer := lipgloss.NewStyle().Padding(1, 1)
    return outer.Render(spinnerStyle.Render(spinnerContent))
}

func (m Model) renderSyncLoadingModal() string {
    msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Syncing…"))
    content := msg
    wrapper := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
        Padding(1, 2)
    minW := 24
    w := max(minW, lipgloss.Width(content)+4)
    wrapper = wrapper.Width(w)
    outer := lipgloss.NewStyle().Padding(1, 1)
    return outer.Render(wrapper.Render(content))
}

func (m Model) renderInitialLoadingModal() string {
    msg := fmt.Sprintf("%s %s", m.spinner.View(), statusStyle.Render("Loading..."))
    content := msg
    wrapper := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(magentaBright).
        Padding(1, 2)
    minW := 32
    w := max(minW, lipgloss.Width(content)+4)
    wrapper = wrapper.Width(w)
    outer := lipgloss.NewStyle().Padding(1, 1)
    return outer.Render(wrapper.Render(content))
}

func (m Model) renderRollbackModal() string {
    header := m.renderBanner()
    headerLines := countLines(header)
    const BORDER_LINES = 2
    const STATUS_LINES = 1
    overhead := BORDER_LINES + headerLines + STATUS_LINES
    availableRows := max(0, m.state.Terminal.Rows-overhead)

    containerWidth := max(0, m.state.Terminal.Cols-2)
    contentHeight := max(3, availableRows)
    innerWidth := max(0, containerWidth-4)
    innerHeight := max(0, contentHeight-2)

    if m.state.Rollback == nil || m.state.Modals.RollbackAppName == nil {
        var content string
        if m.state.Modals.RollbackAppName == nil {
            content = "No app selected for rollback"
        } else {
            content = fmt.Sprintf("Loading deployment history for %s...\n\n%s", *m.state.Modals.RollbackAppName, m.spinner.View())
        }
        return m.renderSimpleModal("Rollback", content)
    }

    rollback := m.state.Rollback
    var modalContent string
    if rollback.Loading {
        if rollback.Mode == "confirm" {
            modalContent = fmt.Sprintf("%s Executing rollback for %s...", m.spinner.View(), rollback.AppName)
        } else {
            modalContent = fmt.Sprintf("%s Loading deployment history for %s...", m.spinner.View(), *m.state.Modals.RollbackAppName)
        }
    } else if rollback.Error != "" {
        errorStyle := lipgloss.NewStyle().Foreground(outOfSyncColor)
        modalContent = errorStyle.Render(fmt.Sprintf("Error loading rollback history:\n%s", rollback.Error))
    } else if rollback.Mode == "confirm" {
        modalContent = m.renderRollbackConfirmation(rollback, innerHeight, innerWidth)
    } else {
        modalContent = m.renderRollbackHistory(rollback)
    }

    if rollback.Mode != "confirm" {
        instructionStyle := lipgloss.NewStyle().Foreground(cyanBright)
        instructions := "j/k: Navigate • Enter: Select • Esc: Cancel"
        modalContent += "\n\n" + instructionStyle.Render(instructions)
    }

    modalStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
        Width(containerWidth).
        Height(contentHeight).
        AlignVertical(lipgloss.Top).
        PaddingLeft(1).
        PaddingRight(1)

    modalContent = normalizeLinesToWidth(modalContent, innerWidth)
    modalContent = clipAnsiToLines(modalContent, innerHeight)
    styledContent := modalStyle.Render(modalContent)

    var sections []string
    sections = append(sections, header)
    sections = append(sections, styledContent)

    content := strings.Join(sections, "\n")
    totalHeight := m.state.Terminal.Rows - 1
    content = clipAnsiToLines(content, totalHeight)
    return mainContainerStyle.Height(totalHeight).Render(content)
}

func (m Model) renderSimpleModal(title, content string) string {
    header := m.renderBanner()
    headerLines := countLines(header)
    const BORDER_LINES = 2
    const STATUS_LINES = 1
    overhead := BORDER_LINES + headerLines + STATUS_LINES
    availableRows := max(0, m.state.Terminal.Rows-overhead)

    containerWidth := max(0, m.state.Terminal.Cols-2)
    contentWidth := max(0, containerWidth-4)
    contentHeight := max(3, availableRows)

    titleStyle := lipgloss.NewStyle().Foreground(cyanBright).Bold(true)
    modalContent := titleStyle.Render(title) + "\n\n" + content

    modalStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
        Width(contentWidth).
        Height(contentHeight).
        AlignVertical(lipgloss.Top).
        PaddingLeft(1).
        PaddingRight(1)

    styledContent := modalStyle.Render(modalContent)
    var sections []string
    sections = append(sections, header)
    sections = append(sections, styledContent)

    content = strings.Join(sections, "\n")
    totalHeight := m.state.Terminal.Rows - 1
    return mainContainerStyle.Height(totalHeight).Render(content)
}
