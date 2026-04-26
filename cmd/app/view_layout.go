package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/darksworm/argonaut/pkg/model"
)

// moved: full-screen helpers remain in view.go

// overlaySpec describes a modal/spinner currently being shown above
// the base view. activeOverlay returns one of these (or nil) so the
// "what modal is up?" decision lives in exactly one place — both
// renderMainLayout (which does the actual composition) and row
// renderers via willDesaturateBase consult the same function.
type overlaySpec struct {
	modal       string           // primary modal content; centered on screen
	extraLayers []*lipgloss.Layer // any additional layers below the modal (e.g. a corner badge)
	desaturate  bool             // whether the base view should be dimmed under the modal
}

// activeOverlay returns the overlay currently shown above the base
// view, or nil if none. Branch order matches the original
// renderMainLayout decision tree so behaviour is preserved.
func (m *Model) activeOverlay() *overlaySpec {
	// Non-desaturating overlays (the modal carries its own opaque
	// content; we don't want to dim what's beneath it).
	if m.state.Mode == model.ModeTheme {
		return &overlaySpec{modal: m.renderThemeSelectionModal()}
	}
	if m.state.Mode == model.ModeK9sContextSelect {
		return &overlaySpec{modal: m.renderK9sContextSelectionModal()}
	}

	// Desaturating overlays.
	if m.state.Mode == model.ModeRollback && m.state.Rollback != nil && m.state.Rollback.Loading {
		return &overlaySpec{modal: m.renderRollbackLoadingModal(), desaturate: true}
	}
	if m.state.Navigation.View == model.ViewTree && m.treeLoading {
		return &overlaySpec{modal: m.renderTreeLoadingSpinner(), desaturate: true}
	}
	if m.state.Mode == model.ModeConfirmSync || m.state.Modals.ConfirmSyncLoading {
		modal := m.renderConfirmSyncModal()
		if m.state.Modals.ConfirmSyncLoading {
			modal = m.renderSyncLoadingModal()
		}
		return &overlaySpec{modal: modal, desaturate: true}
	}
	if m.state.Modals.ChangelogLoading {
		return &overlaySpec{modal: m.renderChangelogLoadingModal(), desaturate: true}
	}
	if m.state.Mode == model.ModeUpgrade || m.state.Mode == model.ModeUpgradeError || m.state.Mode == model.ModeUpgradeSuccess {
		var modal string
		switch {
		case m.state.Mode == model.ModeUpgradeError:
			modal = m.renderUpgradeErrorModal()
		case m.state.Mode == model.ModeUpgradeSuccess:
			modal = m.renderUpgradeSuccessModal()
		case m.state.Modals.UpgradeLoading:
			modal = m.renderUpgradeLoadingModal()
		default:
			modal = m.renderUpgradeConfirmModal()
		}
		return &overlaySpec{modal: modal, desaturate: true}
	}
	if m.state.Mode == model.ModeNoDiff {
		return &overlaySpec{modal: m.renderNoDiffModal(), desaturate: true}
	}
	if m.state.Mode == model.ModeK9sError {
		return &overlaySpec{modal: m.renderK9sErrorModal(), desaturate: true}
	}
	if m.state.Mode == model.ModeDefaultViewWarning {
		return &overlaySpec{modal: m.renderDefaultViewWarningModal(), desaturate: true}
	}
	if m.state.Mode == model.ModeConfirmAppDelete {
		modal := m.renderAppDeleteConfirmModal()
		if m.state.Modals.DeleteLoading {
			modal = m.renderAppDeleteLoadingModal()
		}
		return &overlaySpec{modal: modal, desaturate: true}
	}
	if m.state.Mode == model.ModeConfirmResourceDelete {
		modal := m.renderResourceDeleteConfirmModal()
		if m.state.Modals.ResourceDeleteLoading {
			modal = m.renderResourceDeleteLoadingModal()
		}
		return &overlaySpec{modal: modal, desaturate: true}
	}
	if m.state.Mode == model.ModeConfirmResourceSync {
		modal := m.renderResourceSyncConfirmModal()
		if m.state.Modals.ResourceSyncLoading {
			modal = m.renderResourceSyncLoadingModal()
		}
		return &overlaySpec{modal: modal, desaturate: true}
	}
	if m.state.Mode == model.ModeResourceAction {
		var modal string
		st := m.state.Modals.ResourceAction
		switch {
		case st == nil || st.Loading:
			modal = m.renderResourceActionLoadingModal()
		case st.Executing:
			modal = m.renderResourceActionExecutingModal()
		case len(st.Actions) == 0:
			modal = m.renderResourceActionInfoModal()
		default:
			modal = m.renderResourceActionModal()
		}
		return &overlaySpec{modal: modal, desaturate: true}
	}
	if m.state.Mode == model.ModeLoading && m.state.Navigation.View != model.ViewContexts {
		spec := &overlaySpec{modal: m.renderInitialLoadingModal(), desaturate: true}
		// Diff loading badge in the top-left corner, layered below the
		// loading modal but above the desaturated base.
		if m.state.Diff != nil && m.state.Diff.Loading {
			badge := m.renderSmallBadge(true, m.state.Terminal.Cols >= 72)
			spec.extraLayers = append(spec.extraLayers,
				lipgloss.NewLayer(badge).X(1).Y(1).Z(1))
		}
		return spec
	}
	if len(m.state.Apps) == 0 && m.state.Mode == model.ModeNormal && m.state.Navigation.View != model.ViewContexts {
		return &overlaySpec{modal: m.renderNoServerModal(), desaturate: true}
	}
	if m.state.Diff != nil && m.state.Diff.Loading {
		return &overlaySpec{modal: m.renderDiffLoadingSpinner(), desaturate: true}
	}
	return nil
}

// willDesaturateBase reports whether the base view will be rendered
// behind a desaturating overlay. Row renderers consult this so the
// cursor's bg highlight is suppressed while a modal is up — otherwise
// the bg-styled segment survives the per-segment desaturation pass
// and the cursor row leaks through. Single source of truth: derived
// directly from activeOverlay so it can never drift from the actual
// composition.
func (m *Model) willDesaturateBase() bool {
	ov := m.activeOverlay()
	return ov != nil && ov.desaturate
}

// renderTreePanel renders the resource tree view inside a bordered container with scrolling
func (m *Model) renderTreePanel(availableRows int) string {
	contentWidth := max(0, m.contentInnerWidth())
	treeContent := "(no data)"
	if m.treeView != nil {
		treeContent = m.treeView.Render()
	}

	// Split content into lines for scrolling
	lines := strings.Split(treeContent, "\n")
	totalLines := len(lines)

	// Calculate viewport
	viewportHeight := availableRows
	cursorIdx := 0
	if m.treeView != nil {
		// Account for blank separator lines inserted between app roots
		if s, ok := interface{}(m.treeView).(interface{ SelectedLineIndex() int }); ok {
			cursorIdx = s.SelectedLineIndex()
		} else {
			cursorIdx = m.treeView.SelectedIndex()
		}
	}
	// Use treeNav for scroll offset
	scrollOffset := m.treeNav.ScrollOffset()

	// Clamp cursor to valid range
	if cursorIdx >= totalLines {
		cursorIdx = max(0, totalLines-1)
	}

	// Ensure scroll offset keeps cursor in view
	if cursorIdx < scrollOffset {
		scrollOffset = cursorIdx
	} else if cursorIdx >= scrollOffset+viewportHeight {
		scrollOffset = cursorIdx - viewportHeight + 1
	}

	// Clamp scroll offset
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > max(0, totalLines-viewportHeight) {
		scrollOffset = max(0, totalLines-viewportHeight)
	}

	// Update the tree navigator with the adjusted scroll and item count
	m.treeNav.SetItemCount(totalLines)
	m.treeNav.SetViewportHeight(viewportHeight)
	// Note: We don't call SetCursor here because tree view manages its own cursor
	// The scroll offset adjustment is handled by ensuring cursor is visible above

	// Extract visible lines
	visibleLines := []string{}
	for i := scrollOffset; i < min(scrollOffset+viewportHeight, totalLines); i++ {
		line := lines[i]
		visibleLines = append(visibleLines, line)
	}

	// Join visible lines
	visibleContent := strings.Join(visibleLines, "\n")
	visibleContent = normalizeLinesToWidth(visibleContent, contentWidth)

	// Add scroll indicator if needed
	if totalLines > viewportHeight {
		scrollInfo := fmt.Sprintf(" [Line %d/%d, View %d-%d] ",
			cursorIdx+1,
			totalLines,
			scrollOffset+1,
			min(scrollOffset+viewportHeight, totalLines))
		// We'll add this to the border title or status line
		_ = scrollInfo
	}

	adjustedWidth := max(0, m.state.Terminal.Cols-2)
	return contentBorderStyle.Width(adjustedWidth).Height(availableRows + 1).AlignVertical(lipgloss.Top).Render(visibleContent)
}

// contentInnerWidth computes inner content width inside the bordered box
func (m *Model) contentInnerWidth() int {
	return max(0, m.state.Terminal.Cols-6)
}

// Main layout
func (m *Model) renderMainLayout() string {
	const (
		BORDER_LINES       = 2
		TABLE_HEADER_LINES = 0
		TAG_LINE           = 0
		STATUS_LINES       = 1
	)
	header := m.renderBanner()
	searchBar := ""
	if m.state.Mode == model.ModeSearch {
		searchBar = m.renderEnhancedSearchBar()
	}
	commandBar := ""
	if m.state.Mode == model.ModeCommand {
		commandBar = m.renderEnhancedCommandBar()
	}
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
	if searchBar != "" {
		sections = append(sections, searchBar)
	}
	if commandBar != "" {
		sections = append(sections, commandBar)
	}

	// Set desaturate mode on tree view if a modal with desaturation will be shown
	// This makes the tree view only highlight selected items (not cursor) with scoped highlights
	if m.treeView != nil && m.state.Navigation.View == model.ViewTree {
		m.treeView.SetDesaturateMode(m.willDesaturateBase())
	}

	if m.state.Navigation.View == model.ViewTree {
		sections = append(sections, m.renderTreePanel(listRows))
	} else {
		sections = append(sections, m.renderListView(listRows))
	}
	sections = append(sections, m.renderStatusLine())

	content := strings.Join(sections, "\n")
	baseView := mainContainerStyle.Render(content)

	ov := m.activeOverlay()
	if ov == nil {
		return baseView
	}

	base := baseView
	if ov.desaturate {
		base = desaturateANSI(baseView)
	}
	layers := []*lipgloss.Layer{lipgloss.NewLayer(base)}
	layers = append(layers, ov.extraLayers...)

	modalX := (m.state.Terminal.Cols - lipgloss.Width(ov.modal)) / 2
	modalY := (m.state.Terminal.Rows - lipgloss.Height(ov.modal)) / 2
	// Modal sits above any extra layers (badges, etc.) the spec carries.
	modalZ := 1
	if len(ov.extraLayers) > 0 {
		modalZ = 2
	}
	layers = append(layers, lipgloss.NewLayer(ov.modal).X(modalX).Y(modalY).Z(modalZ))

	return m.composeOverlay(layers...)
}

// composeOverlay composites the given layers onto a full-screen canvas and
// returns the rendered string. Layers are drawn in the order provided; use
// .Z() on individual layers to control their stacking order.
func (m *Model) composeOverlay(layers ...*lipgloss.Layer) string {
	canvas := lipgloss.NewCanvas(m.state.Terminal.Cols, m.state.Terminal.Rows)
	canvas.Compose(lipgloss.NewCompositor(layers...))
	return canvas.Render()
}
