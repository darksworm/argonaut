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

    // Layout toggle (match earlier TS threshold)
    isWide := m.state.Terminal.Cols >= 60

    // Small keycap style to make keys pop
    keycap := func(s string) string {
        return lipgloss.NewStyle().Foreground(whiteBright).Background(lipgloss.Color("238")).Padding(0, 1).Render(s)
    }
    mono := func(s string) string { return lipgloss.NewStyle().Foreground(cyanBright).Render(s) }
    bullet := func() string { return lipgloss.NewStyle().Foreground(dimColor).Render("•") }

    // GENERAL
    general := strings.Join([]string{
        mono(":"), " command ", bullet(), " ", mono("/"), " search ", bullet(), " ", mono("?"), " help",
    }, "")

    // NAV
    nav := strings.Join([]string{
        mono("j/k"), " up/down ", bullet(), " ", keycap("Space"), " select ", bullet(), " ", keycap("Enter"), " drill down ", bullet(), " ", keycap("Esc"), " clear/up",
    }, "")

    // VIEWS (two lines)
    views := strings.Join([]string{
        mono(":cls"), "|", mono(":clusters"), "|", mono(":cluster"), " ", bullet(), " ", mono(":ns"), "|", mono(":namespaces"), "|", mono(":namespace"),
        "\n",
        mono(":proj"), "|", mono(":projects"), "|", mono(":project"), " ", bullet(), " ", mono(":apps"),
    }, "")

    // ACTIONS (stacked for readability)
    actions := strings.Join([]string{
        mono(":diff"), " [app] ", bullet(), " ", mono(":sync"), " [app] ", bullet(), " ", mono(":rollback"), " [app]",
        "\n",
        mono(":resources"), " [app] ", bullet(), " ", mono(":up"), " go up level",
        "\n",
        mono("s"), " sync modal (apps view)",
        "\n",
        mono("R"), " rollback modal (apps view)",
        "\n",
        mono("r"), " resources (apps view)",
        "\n",
        mono("d"), " open diff (apps view)",
    }, "")

    // MISC
    misc := strings.Join([]string{
        mono(":all"), " ", bullet(), " ", mono(":licenses"),
        "\n",
        mono(":logs"), " ", bullet(), " ", mono(":q"),
    }, "")

    var helpSections []string
    // Add a blank line between sections to mimic Ink's marginTop=1
    helpSections = append(helpSections, m.renderHelpSection("GENERAL", general, isWide))
    helpSections = append(helpSections, "")
    helpSections = append(helpSections, m.renderHelpSection("NAV", nav, isWide))
    helpSections = append(helpSections, "")
    helpSections = append(helpSections, m.renderHelpSection("VIEWS", views, isWide))
    helpSections = append(helpSections, "")
    helpSections = append(helpSections, m.renderHelpSection("ACTIONS", actions, isWide))
    helpSections = append(helpSections, "")
    helpSections = append(helpSections, m.renderHelpSection("MISC", misc, isWide))
    helpSections = append(helpSections, "")
    helpSections = append(helpSections, statusStyle.Render("Press ?, q or Esc to close"))

    body := "\n" + strings.Join(helpSections, "\n") + "\n"
    return m.renderFullScreenViewWithOptions(header, body, m.renderStatusLine(), FullScreenViewOptions{ ContentBordered: true, BorderColor: magentaBright })
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

// renderTreeLoadingSpinner displays a centered loading spinner for resources/tree operations
func (m Model) renderTreeLoadingSpinner() string {
    spinnerContent := fmt.Sprintf("%s Loading resources...", m.spinner.View())
    spinnerStyle := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(cyanBright).
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
    // Add status line to ensure full-height composition like other views
    status := m.renderStatusLine()
    sections = append(sections, status)

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
    // Add status line for consistent height
    sections = append(sections, m.renderStatusLine())

    content = strings.Join(sections, "\n")
    totalHeight := m.state.Terminal.Rows - 1
    return mainContainerStyle.Height(totalHeight).Render(content)
}
