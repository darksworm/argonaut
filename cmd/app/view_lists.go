package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// renderListView - custom list/table rendering with fixed inner width
func (m *Model) renderListView(availableRows int) string {
	visibleItems := m.getVisibleItems()

	contentWidth := max(0, m.contentInnerWidth())
	// Leave room for the table header row inside the bordered area
	tableHeight := max(3, availableRows-1)

	// Prepare data and update the appropriate table directly
	var tableView string

	// Handle empty state - let it flow through normal rendering but with empty tableView
	if len(visibleItems) == 0 {
		tableView = "" // Empty table view will render clean empty space
	} else {

		switch m.state.Navigation.View {
		case model.ViewApps:
			// Custom-render apps list to restore full-row selection highlight and per-cell colors
			// Determine viewport to keep selection visible
			total := len(visibleItems)
			visibleRows := max(0, tableHeight-1) // leave 1 line for header
			if visibleRows <= 0 {
				visibleRows = 0
			}
			cursor := m.state.Navigation.SelectedIdx
			if cursor < 0 {
				cursor = 0
			}
			if cursor >= total {
				cursor = max(0, total-1)
			}
			start := cursor - visibleRows/2
			if start < 0 {
				start = 0
			}
			if start > max(0, total-visibleRows) {
				start = max(0, total-visibleRows)
			}
			end := min(total, start+visibleRows)

			// Build header + rows
			var b strings.Builder
			b.WriteString(m.renderListHeader())
			b.WriteString("\n")
			for i := start; i < end; i++ {
				app := visibleItems[i].(model.App)
				isCursor := (i == cursor)
				b.WriteString(m.renderAppRow(app, isCursor))
				if i < end-1 {
					b.WriteString("\n")
				}
			}
			// Pad remaining lines to maintain fixed height inside border
			for pad := end - start; pad < visibleRows; pad++ {
				b.WriteString("\n")
			}
			tableView = b.String()

		case model.ViewClusters, model.ViewNamespaces, model.ViewProjects:
			// Custom-render single-column lists with full-row highlight
			total := len(visibleItems)
			visibleRows := max(0, tableHeight-1)
			cursor := m.state.Navigation.SelectedIdx
			if cursor < 0 {
				cursor = 0
			}
			if cursor >= total {
				cursor = max(0, total-1)
			}
			start := cursor - visibleRows/2
			if start < 0 {
				start = 0
			}
			if start > max(0, total-visibleRows) {
				start = max(0, total-visibleRows)
			}
			end := min(total, start+visibleRows)

			var b strings.Builder
			b.WriteString(m.renderListHeader())
			b.WriteString("\n")
			for i := start; i < end; i++ {
				label := fmt.Sprintf("%v", visibleItems[i])
				isCursor := (i == cursor)
				b.WriteString(m.renderSimpleRow(label, isCursor))
				if i < end-1 {
					b.WriteString("\n")
				}
			}
			for pad := end - start; pad < visibleRows; pad++ {
				b.WriteString("\n")
			}
			tableView = b.String()
		}
	}

	// Render the table/content ensuring each line fits the content width
	var content strings.Builder
	content.WriteString(normalizeLinesToWidth(tableView, contentWidth))

	// Apply border style with proper width. For empty content, set fixed height to fill space
	if tableView == "" {
		// Empty state: use fixed height to fill available space like other views
		// Adjust width to properly fill horizontal space
		adjustedWidth := max(0, m.state.Terminal.Cols-2) // Expand width to fill space
		return contentBorderStyle.Width(adjustedWidth).Height(availableRows + 1).AlignVertical(lipgloss.Center).Render(content.String())
	}

	// Non-empty content: let height auto-size to content to avoid tmux line-wrapping issues
	return contentBorderStyle.Render(content.String())
}

// renderListHeader - matches ListView header row with responsive widths
func (m *Model) renderListHeader() string {
	if m.state.Navigation.View == model.ViewApps {
		// Fixed-width columns with full text headers
		contentWidth := m.contentInnerWidth()
		syncWidth := 12
		healthWidth := 15
		nameWidth := max(10, contentWidth-syncWidth-healthWidth-2)

		nameHeader := headerStyle.Render("NAME")
		syncHeader := headerStyle.Render("SYNC")
		healthHeader := headerStyle.Render("HEALTH")

		nameCell := padRight(clipAnsiToWidth(nameHeader, nameWidth), nameWidth)
		syncCell := padLeft(clipAnsiToWidth(syncHeader, syncWidth), syncWidth)
		healthCell := padLeft(clipAnsiToWidth(healthHeader, healthWidth), healthWidth)

		header := fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)
		// Guarantee exact width to prevent underline overflow
		if lipgloss.Width(header) < contentWidth {
			header = padRight(header, contentWidth)
		} else if lipgloss.Width(header) > contentWidth {
			header = clipAnsiToWidth(header, contentWidth)
		}
		return header
	}

	// Simple header for other views padded to full content width
	contentWidth := m.contentInnerWidth()
	hdr := headerStyle.Render("NAME")
	if lipgloss.Width(hdr) < contentWidth {
		hdr = padRight(hdr, contentWidth)
	} else if lipgloss.Width(hdr) > contentWidth {
		hdr = clipAnsiToWidth(hdr, contentWidth)
	}
	return hdr
}

// renderAppRow - matches ListView app row rendering
func (m *Model) renderAppRow(app model.App, isCursor bool) string {
	// Selection checking (matches ListView isChecked logic)
	isSelected := m.state.Selections.HasSelectedApp(app.Name)
	active := isCursor || isSelected

	// Get project name (for future use)
	_ = "default"
	if app.Project != nil {
		_ = *app.Project
	}

	// Prepare texts and widths using same responsive logic as header
	syncIcon := m.getSyncIcon(app.Sync)
	healthIcon := m.getHealthIcon(app.Health)

	contentWidth := m.contentInnerWidth() // Match header/content inner width
	nameWidth, syncWidth, healthWidth := calculateColumnWidths(contentWidth)

	// Generate text based on available width (either full text or icons only)
	// Colored status strings with icons (as before)
	syncText := fmt.Sprintf("%s %s", syncIcon, app.Sync)
	healthText := fmt.Sprintf("%s %s", healthIcon, app.Health)

	// Truncate app name with ellipsis if it's too long
	truncatedName := truncateWithEllipsis(app.Name, nameWidth)

	var nameCell, syncCell, healthCell string
	// Build cells with clipping to assigned widths to prevent wrapping
	nameCell = padRight(truncateWithEllipsis(truncatedName, nameWidth), nameWidth)

	if isCursor || isSelected {
		// Active row: avoid inner color styles so background highlight spans the whole row
		if lipgloss.Width(syncText) > syncWidth {
			syncText = clipAnsiToWidth(syncText, syncWidth)
		}
		if lipgloss.Width(healthText) > healthWidth {
			healthText = clipAnsiToWidth(healthText, healthWidth)
		}
		syncCell = padLeft(syncText, syncWidth)
		healthCell = padLeft(healthText, healthWidth)
	} else {
		// Inactive row: apply color styles to sync/health then clip if needed
		syncStyled := m.getColorForStatus(app.Sync).Render(syncText)
		healthStyled := m.getColorForStatus(app.Health).Render(healthText)
		if lipgloss.Width(syncStyled) > syncWidth {
			syncStyled = clipAnsiToWidth(syncStyled, syncWidth)
		}
		if lipgloss.Width(healthStyled) > healthWidth {
			healthStyled = clipAnsiToWidth(healthStyled, healthWidth)
		}
		syncCell = padLeft(syncStyled, syncWidth)
		healthCell = padLeft(healthStyled, healthWidth)
	}

	row := fmt.Sprintf("%s %s %s", nameCell, syncCell, healthCell)

	// Ensure row is exactly the content width to avoid wrapping
	fullRowWidth := nameWidth + syncWidth + healthWidth + 2 // +2 for separators
	if lipgloss.Width(row) < fullRowWidth {
		row = padRight(row, fullRowWidth)
	} else if lipgloss.Width(row) > fullRowWidth {
		row = clipAnsiToWidth(row, fullRowWidth)
	}

	// Apply selection highlight (matches ListView backgroundColor)
	if active {
		row = selectedStyle.Render(row)
		// After styling, clip again defensively (some terminals render bold differently)
		if lipgloss.Width(row) > fullRowWidth {
			row = clipAnsiToWidth(row, fullRowWidth)
		}
	}

	return row
}

// renderSimpleRow - matches ListView non-app row rendering
func (m *Model) renderSimpleRow(label string, isCursor bool) string {
	// Check if selected based on view (matches ListView isChecked logic)
	isSelected := false
	switch m.state.Navigation.View {
	case model.ViewClusters:
		isSelected = m.state.Selections.HasCluster(label)
	case model.ViewNamespaces:
		isSelected = m.state.Selections.HasNamespace(label)
	case model.ViewProjects:
		isSelected = m.state.Selections.HasProject(label)
	}

	active := isCursor || isSelected

	// Calculate available width for simple rows (full content width minus padding)
	contentWidth := m.contentInnerWidth()

	// Truncate and pad label to full width
	truncatedLabel := truncateWithEllipsis(label, contentWidth)
	row := padRight(truncatedLabel, contentWidth)

	// Apply selection highlight if active
	if active {
		return selectedStyle.Render(row)
	}
	return row
}
