package services

import "github.com/a9s/go-app/pkg/model"

// NavigationService interface defines operations for navigation logic
type NavigationService interface {
	// DrillDown handles drill-down navigation from one view to the next
	DrillDown(currentView model.View, selectedItem interface{}, visibleItems []interface{}, selectedIdx int) *NavigationUpdate

	// ToggleSelection handles selection toggling (only works in apps view)
	ToggleSelection(currentView model.View, selectedItem interface{}, visibleItems []interface{}, selectedIdx int, currentSelectedApps map[string]bool) *SelectionUpdate

	// ValidateBounds ensures selectedIdx stays within valid bounds
	ValidateBounds(selectedIdx, itemCount int) int

	// ClearLowerLevelSelections clears selections based on current view
	ClearLowerLevelSelections(view model.View) map[string]interface{}

	// ResetNavigation resets navigation state to defaults
	ResetNavigation(view *model.View) map[string]interface{}

	// ClearAllSelections clears all selections
	ClearAllSelections() map[string]interface{}

	// ClearFilters clears all filters and search
	ClearFilters() map[string]interface{}

	// CanDrillDown determines if drill down is possible from current view
	CanDrillDown(view model.View) bool

	// CanToggleSelection determines if selection toggle is possible from current view
	CanToggleSelection(view model.View) bool

	// GetNextView gets the next view in the drill down hierarchy
	GetNextView(currentView model.View) *model.View

	// GetPreviousView gets the previous view in the drill down hierarchy
	GetPreviousView(currentView model.View) *model.View
}

// NavigationUpdate represents the result of a navigation operation
type NavigationUpdate struct {
	NewView                         *model.View     `json:"newView,omitempty"`
	ScopeClusters                   map[string]bool `json:"scopeClusters,omitempty"`
	ScopeNamespaces                 map[string]bool `json:"scopeNamespaces,omitempty"`
	ScopeProjects                   map[string]bool `json:"scopeProjects,omitempty"`
	SelectedApps                    map[string]bool `json:"selectedApps,omitempty"`
	ShouldResetNavigation           bool            `json:"shouldResetNavigation"`
	ShouldClearLowerLevelSelections bool            `json:"shouldClearLowerLevelSelections"`
}

// SelectionUpdate represents the result of a selection operation
type SelectionUpdate struct {
	SelectedApps map[string]bool `json:"selectedApps"`
}

// NavigationServiceImpl provides a concrete implementation of NavigationService
type NavigationServiceImpl struct{}

// NewNavigationService creates a new NavigationService implementation
func NewNavigationService() NavigationService {
	return &NavigationServiceImpl{}
}

// DrillDown implements NavigationService.DrillDown
func (s *NavigationServiceImpl) DrillDown(currentView model.View, selectedItem interface{}, visibleItems []interface{}, selectedIdx int) *NavigationUpdate {
	if selectedIdx >= len(visibleItems) || selectedIdx < 0 {
		return nil
	}

	item := visibleItems[selectedIdx]
	if item == nil {
		return nil
	}

	val := stringValue(item)
	next := model.AddToStringSet(model.NewStringSet(), val)

	result := &NavigationUpdate{
		ShouldResetNavigation:           true,
		ShouldClearLowerLevelSelections: true,
	}

	switch currentView {
	case model.ViewClusters:
		newView := model.ViewNamespaces
		result.NewView = &newView
		result.ScopeClusters = next
	case model.ViewNamespaces:
		newView := model.ViewProjects
		result.NewView = &newView
		result.ScopeNamespaces = next
	case model.ViewProjects:
		newView := model.ViewApps
		result.NewView = &newView
		result.ScopeProjects = next
	default:
		return nil // Can't drill down from apps view
	}

	return result
}

// ToggleSelection implements NavigationService.ToggleSelection
func (s *NavigationServiceImpl) ToggleSelection(currentView model.View, selectedItem interface{}, visibleItems []interface{}, selectedIdx int, currentSelectedApps map[string]bool) *SelectionUpdate {
	// Only allow toggle selection in apps view
	if currentView != model.ViewApps {
		return nil
	}

	if selectedIdx >= len(visibleItems) || selectedIdx < 0 {
		return nil
	}

	item := visibleItems[selectedIdx]
	if item == nil {
		return nil
	}

	appName := appNameFromItem(item)
	next := make(map[string]bool)
	for k, v := range currentSelectedApps {
		next[k] = v
	}

	if model.HasInStringSet(next, appName) {
		next = model.RemoveFromStringSet(next, appName)
	} else {
		next = model.AddToStringSet(next, appName)
	}

	return &SelectionUpdate{
		SelectedApps: next,
	}
}

// ValidateBounds implements NavigationService.ValidateBounds
func (s *NavigationServiceImpl) ValidateBounds(selectedIdx, itemCount int) int {
	if itemCount == 0 {
		return 0
	}
	if selectedIdx < 0 {
		return 0
	}
	if selectedIdx >= itemCount {
		return itemCount - 1
	}
	return selectedIdx
}

// ClearLowerLevelSelections implements NavigationService.ClearLowerLevelSelections
func (s *NavigationServiceImpl) ClearLowerLevelSelections(view model.View) map[string]interface{} {
	emptySet := model.NewStringSet()
	result := make(map[string]interface{})

	switch view {
	case model.ViewClusters:
		result["scopeNamespaces"] = emptySet
		result["scopeProjects"] = emptySet
		result["selectedApps"] = emptySet
	case model.ViewNamespaces:
		result["scopeProjects"] = emptySet
		result["selectedApps"] = emptySet
	case model.ViewProjects:
		result["selectedApps"] = emptySet
	}

	return result
}

// ResetNavigation implements NavigationService.ResetNavigation
func (s *NavigationServiceImpl) ResetNavigation(view *model.View) map[string]interface{} {
	result := map[string]interface{}{
		"selectedIdx":    0,
		"activeFilter":   "",
		"searchQuery":    "",
	}
	if view != nil {
		result["view"] = *view
	}
	return result
}

// ClearAllSelections implements NavigationService.ClearAllSelections
func (s *NavigationServiceImpl) ClearAllSelections() map[string]interface{} {
	return map[string]interface{}{
		"scopeClusters":   model.NewStringSet(),
		"scopeNamespaces": model.NewStringSet(),
		"scopeProjects":   model.NewStringSet(),
		"selectedApps":    model.NewStringSet(),
	}
}

// ClearFilters implements NavigationService.ClearFilters
func (s *NavigationServiceImpl) ClearFilters() map[string]interface{} {
	return map[string]interface{}{
		"activeFilter": "",
		"searchQuery":  "",
	}
}

// CanDrillDown implements NavigationService.CanDrillDown
func (s *NavigationServiceImpl) CanDrillDown(view model.View) bool {
	return view != model.ViewApps
}

// CanToggleSelection implements NavigationService.CanToggleSelection
func (s *NavigationServiceImpl) CanToggleSelection(view model.View) bool {
	return view == model.ViewApps
}

// GetNextView implements NavigationService.GetNextView
func (s *NavigationServiceImpl) GetNextView(currentView model.View) *model.View {
	switch currentView {
	case model.ViewClusters:
		view := model.ViewNamespaces
		return &view
	case model.ViewNamespaces:
		view := model.ViewProjects
		return &view
	case model.ViewProjects:
		view := model.ViewApps
		return &view
	default:
		return nil
	}
}

// GetPreviousView implements NavigationService.GetPreviousView
func (s *NavigationServiceImpl) GetPreviousView(currentView model.View) *model.View {
	switch currentView {
	case model.ViewApps:
		view := model.ViewProjects
		return &view
	case model.ViewProjects:
		view := model.ViewNamespaces
		return &view
	case model.ViewNamespaces:
		view := model.ViewClusters
		return &view
	default:
		return nil
	}
}

// Helper functions

// stringValue extracts string representation from an interface{}
func stringValue(item interface{}) string {
	if item == nil {
		return ""
	}
	if str, ok := item.(string); ok {
		return str
	}
	// Handle App struct
	if app, ok := item.(model.App); ok {
		return app.Name
	}
	// Handle App pointer
	if app, ok := item.(*model.App); ok && app != nil {
		return app.Name
	}
	return ""
}

// appNameFromItem extracts app name from an item (assuming it's an App)
func appNameFromItem(item interface{}) string {
	if app, ok := item.(model.App); ok {
		return app.Name
	}
	if app, ok := item.(*model.App); ok && app != nil {
		return app.Name
	}
	return ""
}