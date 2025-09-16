package main

import (
    "fmt"
    "strings"

    "github.com/charmbracelet/lipgloss/v2"
    "github.com/darksworm/argonaut/pkg/model"
)

// moved: full-screen helpers remain in view.go

// renderTreePanel renders the resource tree view inside a bordered container with scrolling
func (m Model) renderTreePanel(availableRows int) string {
    contentWidth := max(0, m.contentInnerWidth())
    treeContent := "(no data)"
    if m.treeView != nil {
        treeContent = m.treeView.View()
    }

    // Split content into lines for scrolling
    lines := strings.Split(treeContent, "\n")
    totalLines := len(lines)

    // Calculate viewport
    viewportHeight := availableRows
    scrollOffset := m.treeScrollOffset

    // Clamp scroll offset
    if scrollOffset < 0 {
        scrollOffset = 0
    }
    if scrollOffset > max(0, totalLines-viewportHeight) {
        scrollOffset = max(0, totalLines-viewportHeight)
    }

    // Extract visible lines
    visibleLines := []string{}
    for i := scrollOffset; i < min(scrollOffset+viewportHeight, totalLines); i++ {
        visibleLines = append(visibleLines, lines[i])
    }

    // Join visible lines
    visibleContent := strings.Join(visibleLines, "\n")
    visibleContent = normalizeLinesToWidth(visibleContent, contentWidth)

    // Add scroll indicator if needed
    if totalLines > viewportHeight {
        scrollInfo := fmt.Sprintf(" [%d-%d/%d] ",
            scrollOffset+1,
            min(scrollOffset+viewportHeight, totalLines),
            totalLines)
        // We'll add this to the border title or status line
        _ = scrollInfo
    }

    adjustedWidth := max(0, m.state.Terminal.Cols-2)
    return contentBorderStyle.Width(adjustedWidth).Height(availableRows + 1).AlignVertical(lipgloss.Top).Render(visibleContent)
}

// contentInnerWidth computes inner content width inside the bordered box
func (m Model) contentInnerWidth() int {
    return max(0, m.state.Terminal.Cols-6)
}

// Main layout
func (m Model) renderMainLayout() string {
    const (
        BORDER_LINES       = 2
        TABLE_HEADER_LINES = 0
        TAG_LINE           = 0
        STATUS_LINES       = 1
    )
    header := m.renderBanner()
    searchBar := ""
    if m.state.Mode == model.ModeSearch { searchBar = m.renderEnhancedSearchBar() }
    commandBar := ""
    if m.state.Mode == model.ModeCommand { commandBar = m.renderEnhancedCommandBar() }
    headerLines := countLines(header)
    searchLines := countLines(searchBar)
    commandLines := countLines(commandBar)
    overhead := BORDER_LINES + headerLines + searchLines + commandLines + TABLE_HEADER_LINES + TAG_LINE + STATUS_LINES
    availableRows := max(0, m.state.Terminal.Rows-overhead)
    listRows := max(0, availableRows)

    var sections []string
    sections = append(sections, header)
    // Add a subtle vertical gap only in wide layout. The narrow banner
    // already includes spacing, so avoid doubling it.
    if m.state.Terminal.Cols > 100 {
        sections = append(sections, "")
    }
    if searchBar != "" { sections = append(sections, searchBar) }
    if commandBar != "" { sections = append(sections, commandBar) }

    if m.state.Navigation.View == model.ViewTree {
        sections = append(sections, m.renderTreePanel(listRows))
    } else {
        sections = append(sections, m.renderListView(listRows))
    }
    sections = append(sections, m.renderStatusLine())

    content := strings.Join(sections, "\n")
    baseView := mainContainerStyle.Render(content)

    // Overlays
    // Rollback loading overlay (history load or executing rollback)
    if m.state.Mode == model.ModeRollback && m.state.Rollback != nil && m.state.Rollback.Loading {
        modal := m.renderRollbackLoadingModal()
        grayBase := desaturateANSI(baseView)
        baseLayer := lipgloss.NewLayer(grayBase)
        modalX := (m.state.Terminal.Cols - lipgloss.Width(modal)) / 2
        modalY := (m.state.Terminal.Rows - lipgloss.Height(modal)) / 2
        modalLayer := lipgloss.NewLayer(modal).X(modalX).Y(modalY).Z(1)
        canvas := lipgloss.NewCanvas(baseLayer, modalLayer)
        return canvas.Render()
    }
    // Tree loading overlay when entering resources view
    if m.state.Navigation.View == model.ViewTree && m.treeLoading {
        spinner := m.renderTreeLoadingSpinner()
        grayBase := desaturateANSI(baseView)
        baseLayer := lipgloss.NewLayer(grayBase)
        spinnerLayer := lipgloss.NewLayer(spinner).
            X((m.state.Terminal.Cols - lipgloss.Width(spinner)) / 2).
            Y((m.state.Terminal.Rows - lipgloss.Height(spinner)) / 2).
            Z(1)
        canvas := lipgloss.NewCanvas(baseLayer, spinnerLayer)
        return canvas.Render()
    }
    // Confirm Sync modal (confirmation or loading state)
    if m.state.Mode == model.ModeConfirmSync || m.state.Modals.ConfirmSyncLoading {
        modal := ""
        if m.state.Modals.ConfirmSyncLoading {
            modal = m.renderSyncLoadingModal()
        } else {
            modal = m.renderConfirmSyncModal()
        }
        grayBase := desaturateANSI(baseView)
        baseLayer := lipgloss.NewLayer(grayBase)
        modalX := (m.state.Terminal.Cols - lipgloss.Width(modal)) / 2
        modalY := (m.state.Terminal.Rows - lipgloss.Height(modal)) / 2
        modalLayer := lipgloss.NewLayer(modal).X(modalX).Y(modalY).Z(1)
        canvas := lipgloss.NewCanvas(baseLayer, modalLayer)
        return canvas.Render()
    }
    if m.state.Mode == model.ModeLoading {
        modal := m.renderInitialLoadingModal()
        grayBase := desaturateANSI(baseView)
        baseLayer := lipgloss.NewLayer(grayBase)
        modalX := (m.state.Terminal.Cols - lipgloss.Width(modal)) / 2
        modalY := (m.state.Terminal.Rows - lipgloss.Height(modal)) / 2
        if m.state.Diff != nil && m.state.Diff.Loading {
            badge := m.renderSmallBadge(true)
            badgeLayer := lipgloss.NewLayer(badge).X(1).Y(1).Z(1)
            modalLayer := lipgloss.NewLayer(modal).X(modalX).Y(modalY).Z(2)
            canvas := lipgloss.NewCanvas(baseLayer, badgeLayer, modalLayer)
            return canvas.Render()
        }
        modalLayer := lipgloss.NewLayer(modal).X(modalX).Y(modalY).Z(1)
        canvas := lipgloss.NewCanvas(baseLayer, modalLayer)
        return canvas.Render()
    }
    if m.state.Diff != nil && m.state.Diff.Loading {
        spinner := m.renderDiffLoadingSpinner()
        grayBase := desaturateANSI(baseView)
        baseLayer := lipgloss.NewLayer(grayBase)
        spinnerLayer := lipgloss.NewLayer(spinner).
            X((m.state.Terminal.Cols - lipgloss.Width(spinner)) / 2).
            Y((m.state.Terminal.Rows - lipgloss.Height(spinner)) / 2).
            Z(1)
        canvas := lipgloss.NewCanvas(baseLayer, spinnerLayer)
        return canvas.Render()
    }
    return baseView
}
