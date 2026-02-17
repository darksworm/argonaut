package model

import (
	"time"

	apperrors "github.com/darksworm/argonaut/pkg/errors"
)

// NavigationState holds navigation-related state
type NavigationState struct {
	View           View  `json:"view"`
	SelectedIdx    int   `json:"selectedIdx"`
	LastGPressed   int64 `json:"lastGPressed"`
	LastEscPressed int64 `json:"lastEscPressed"`
	LastZPressed   int64 `json:"lastZPressed"`
}

// SelectionState holds selection-related state using map[string]bool for sets
type SelectionState struct {
	ScopeClusters        map[string]bool `json:"scopeClusters"`
	ScopeNamespaces      map[string]bool `json:"scopeNamespaces"`
	ScopeProjects        map[string]bool `json:"scopeProjects"`
	ScopeApplicationSets map[string]bool `json:"scopeApplicationSets"`
	SelectedApps         map[string]bool `json:"selectedApps"`
}

// NewSelectionState creates a new SelectionState with empty sets
func NewSelectionState() *SelectionState {
	return &SelectionState{
		ScopeClusters:        NewStringSet(),
		ScopeNamespaces:      NewStringSet(),
		ScopeProjects:        NewStringSet(),
		ScopeApplicationSets: NewStringSet(),
		SelectedApps:         NewStringSet(),
	}
}

// Helper methods for SelectionState

// AddCluster adds a cluster to the scope
func (s *SelectionState) AddCluster(cluster string) {
	s.ScopeClusters = AddToStringSet(s.ScopeClusters, cluster)
}

// HasCluster checks if a cluster is in scope
func (s *SelectionState) HasCluster(cluster string) bool {
	return HasInStringSet(s.ScopeClusters, cluster)
}

// AddNamespace adds a namespace to the scope
func (s *SelectionState) AddNamespace(namespace string) {
	s.ScopeNamespaces = AddToStringSet(s.ScopeNamespaces, namespace)
}

// HasNamespace checks if a namespace is in scope
func (s *SelectionState) HasNamespace(namespace string) bool {
	return HasInStringSet(s.ScopeNamespaces, namespace)
}

// AddProject adds a project to the scope
func (s *SelectionState) AddProject(project string) {
	s.ScopeProjects = AddToStringSet(s.ScopeProjects, project)
}

// HasProject checks if a project is in scope
func (s *SelectionState) HasProject(project string) bool {
	return HasInStringSet(s.ScopeProjects, project)
}

// AddApplicationSet adds an application set to the scope
func (s *SelectionState) AddApplicationSet(appset string) {
	s.ScopeApplicationSets = AddToStringSet(s.ScopeApplicationSets, appset)
}

// HasApplicationSet checks if an application set is in scope
func (s *SelectionState) HasApplicationSet(appset string) bool {
	return HasInStringSet(s.ScopeApplicationSets, appset)
}

// AddSelectedApp adds an app to the selected apps
func (s *SelectionState) AddSelectedApp(app string) {
	s.SelectedApps = AddToStringSet(s.SelectedApps, app)
}

// HasSelectedApp checks if an app is selected
func (s *SelectionState) HasSelectedApp(app string) bool {
	return HasInStringSet(s.SelectedApps, app)
}

// ToggleSelectedApp toggles an app's selection status
func (s *SelectionState) ToggleSelectedApp(app string) {
	if s.HasSelectedApp(app) {
		s.SelectedApps = RemoveFromStringSet(s.SelectedApps, app)
	} else {
		s.AddSelectedApp(app)
	}
}

// UIState holds UI-related state
type UIState struct {
	SearchQuery         string          `json:"searchQuery"`
	ActiveFilter        string          `json:"activeFilter"`
	Command             string          `json:"command"`
	IsVersionOutdated   bool            `json:"isVersionOutdated"`
	LatestVersion       *string         `json:"latestVersion,omitempty"`
	UpdateInfo          *UpdateInfo     `json:"updateInfo,omitempty"`
	CommandInputKey     int             `json:"commandInputKey"`
	TreeAppName         *string         `json:"treeAppName,omitempty"`
	ThemeSelectedIndex  int             `json:"themeSelectedIndex"`
	ThemeScrollOffset   int             `json:"themeScrollOffset"`
	ThemeOriginalName   string          `json:"themeOriginalName,omitempty"`
	CommandInvalid      bool            `json:"commandInvalid"`
	Sort                SortConfig      `json:"sort"`
	ShowWhatsNew        bool            `json:"showWhatsNew"`
	WhatsNewShownAt     *time.Time      `json:"whatsNewShownAt,omitempty"`
	RefreshFlashApps    map[string]bool `json:"-"` // Apps to highlight after refresh (transient)
	RefreshFlashTree    bool            `json:"-"` // Flash tree view after refresh (transient)
	SelectionCopied     bool            `json:"-"` // Show "Copied!" message briefly (transient)
}

// ModalState holds modal-related state
type ModalState struct {
	ConfirmTarget    *string `json:"confirmTarget,omitempty"`
	ConfirmSyncPrune bool    `json:"confirmSyncPrune"`
	ConfirmSyncWatch bool    `json:"confirmSyncWatch"`
	// Which button is selected in confirm modal: 0 = Yes, 1 = Cancel
	ConfirmSyncSelected int `json:"confirmSyncSelected"`
	// When true, show a small syncing overlay instead of the confirm UI
	ConfirmSyncLoading bool `json:"confirmSyncLoading"`
	// When true, show initial loading modal overlay during app startup
	InitialLoading  bool    `json:"initialLoading"`
	RollbackAppName *string `json:"rollbackAppName,omitempty"`
	SyncViewApp     *string `json:"syncViewApp,omitempty"`
	// Upgrade confirmation modal state
	UpgradeSelected int     `json:"upgradeSelected"` // 0 = Continue, 1 = Cancel
	UpgradeLoading  bool    `json:"upgradeLoading"`
	UpgradeError    *string `json:"upgradeError,omitempty"` // Error message for upgrade failures
	// Delete confirmation modal state (for apps)
	DeleteAppName          *string `json:"deleteAppName,omitempty"`
	DeleteAppNamespace     *string `json:"deleteAppNamespace,omitempty"`
	DeleteConfirmationKey  string  `json:"deleteConfirmationKey"` // Track what user has typed
	DeleteLoading          bool    `json:"deleteLoading"`
	DeleteError            *string `json:"deleteError,omitempty"`
	DeleteCascade          bool    `json:"deleteCascade"`          // Default true
	DeletePropagationPolicy string `json:"deletePropagationPolicy"` // Default "foreground"
	// Resource delete confirmation modal state
	ResourceDeleteAppName           *string                `json:"resourceDeleteAppName,omitempty"`
	ResourceDeleteAppNamespace      *string                `json:"resourceDeleteAppNamespace,omitempty"`
	ResourceDeleteTargets           []ResourceDeleteTarget `json:"resourceDeleteTargets,omitempty"`
	ResourceDeleteConfirmationKey   string                 `json:"resourceDeleteConfirmationKey"` // Track what user has typed
	ResourceDeleteLoading           bool                   `json:"resourceDeleteLoading"`
	ResourceDeleteError             *string                `json:"resourceDeleteError,omitempty"`
	ResourceDeleteCascade           bool                   `json:"resourceDeleteCascade"`           // Default true
	ResourceDeletePropagationPolicy string                 `json:"resourceDeletePropagationPolicy"` // Default "foreground"
	ResourceDeleteForce             bool                   `json:"resourceDeleteForce"`             // Force deletion
	// Resource sync confirmation modal state
	ResourceSyncAppName         *string              `json:"resourceSyncAppName,omitempty"`
	ResourceSyncTargets         []ResourceSyncTarget `json:"resourceSyncTargets,omitempty"`
	ResourceSyncConfirmSelected int                  `json:"resourceSyncConfirmSelected"` // 0 = Sync, 1 = Cancel
	ResourceSyncLoading         bool                 `json:"resourceSyncLoading"`
	ResourceSyncError           *string              `json:"resourceSyncError,omitempty"`
	ResourceSyncPrune           bool                 `json:"resourceSyncPrune"` // Prune option
	ResourceSyncForce           bool                 `json:"resourceSyncForce"` // Force option
	// Changelog loading modal state
	ChangelogLoading bool `json:"changelogLoading"`
	// K9s error modal state
	K9sError *string `json:"k9sError,omitempty"`
	// Default view warning modal state
	DefaultViewWarning *string `json:"defaultViewWarning,omitempty"`
}

// AppState represents the complete application state for Bubbletea
type AppState struct {
	Mode       Mode            `json:"mode"`
	Terminal   TerminalState   `json:"terminal"`
	Navigation NavigationState `json:"navigation"`
	Selections SelectionState  `json:"selections"`
	UI         UIState         `json:"ui"`
	Modals     ModalState      `json:"modals"`
	Server     *Server         `json:"server,omitempty"`
	Apps       []App           `json:"apps"`
	Index      *AppIndex       `json:"-"` // Pre-computed index, rebuilt on mutation
	APIVersion   string          `json:"apiVersion"`
	ContextNames []string        `json:"contextNames,omitempty"`
	// Note: AbortController equivalent will use context.Context in Go services
	Diff     *DiffState     `json:"diff,omitempty"`
	Rollback *RollbackState `json:"rollback,omitempty"`
	// Store previous navigation state for restoration
	SavedNavigation *NavigationState `json:"savedNavigation,omitempty"`
	SavedSelections *SelectionState  `json:"savedSelections,omitempty"`
	// Store current error information for error screen display
	CurrentError *ApiError   `json:"currentError,omitempty"` // DEPRECATED: Use ErrorState
	ErrorState   *ErrorState `json:"errorState,omitempty"`
}

// ApiError holds structured error information for display - DEPRECATED: Use ErrorState
type ApiError struct {
	Message    string `json:"message"`
	StatusCode int    `json:"statusCode,omitempty"`
	ErrorCode  int    `json:"errorCode,omitempty"`
	Details    string `json:"details,omitempty"`
	Timestamp  int64  `json:"timestamp"`
}

// ErrorState holds comprehensive error state information
type ErrorState struct {
	Current          *apperrors.ArgonautError  `json:"current"`
	History          []apperrors.ArgonautError `json:"history"`
	RetryCount       int                       `json:"retryCount"`
	LastRetryAt      *time.Time                `json:"lastRetryAt,omitempty"`
	AutoHideAt       *time.Time                `json:"autoHideAt,omitempty"`
	RecoveryAttempts int                       `json:"recoveryAttempts"`
}

// DiffState holds state for the diff pager view
type DiffState struct {
	Title       string   `json:"title"`
	Content     []string `json:"content"`
	Filtered    []int    `json:"filtered"`
	Offset      int      `json:"offset"`
	SearchQuery string   `json:"searchQuery"`
	Loading     bool     `json:"loading"`
}

// SaveNavigationState saves current navigation and selection state
func (s *AppState) SaveNavigationState() {
	s.SavedNavigation = &NavigationState{
		View:           s.Navigation.View,
		SelectedIdx:    s.Navigation.SelectedIdx,
		LastGPressed:   s.Navigation.LastGPressed,
		LastEscPressed: s.Navigation.LastEscPressed,
		LastZPressed:   s.Navigation.LastZPressed,
	}
	s.SavedSelections = &SelectionState{
		ScopeClusters:        copyStringSet(s.Selections.ScopeClusters),
		ScopeNamespaces:      copyStringSet(s.Selections.ScopeNamespaces),
		ScopeProjects:        copyStringSet(s.Selections.ScopeProjects),
		ScopeApplicationSets: copyStringSet(s.Selections.ScopeApplicationSets),
		SelectedApps:         copyStringSet(s.Selections.SelectedApps),
	}
}

// RestoreNavigationState restores previously saved navigation state
func (s *AppState) RestoreNavigationState() {
	if s.SavedNavigation != nil {
		s.Navigation.View = s.SavedNavigation.View
		s.Navigation.SelectedIdx = s.SavedNavigation.SelectedIdx
		s.Navigation.LastGPressed = s.SavedNavigation.LastGPressed
		s.Navigation.LastEscPressed = s.SavedNavigation.LastEscPressed
		s.Navigation.LastZPressed = s.SavedNavigation.LastZPressed
		// Clear the saved state after restoration
		s.SavedNavigation = nil
	}
}

// ClearSelectionsAfterDetailView clears only app selections when returning from detail views
// Preserves scope filters (clusters, namespaces, projects) to maintain the filtered view
func (s *AppState) ClearSelectionsAfterDetailView() {
	// Only clear selected apps, preserve scope filters
	s.Selections.SelectedApps = NewStringSet()
	// Clear saved selections as well
	s.SavedSelections = nil
}

// Helper function to copy a string set
func copyStringSet(original map[string]bool) map[string]bool {
	c := make(map[string]bool)
	for k, v := range original {
		c[k] = v
	}
	return c
}

// NewAppState creates a new AppState with default values
func NewAppState() *AppState {
	return &AppState{
		Mode: ModeNormal,
		Terminal: TerminalState{
			Rows: 24,
			Cols: 80,
		},
		Navigation: NavigationState{
			View:           ViewClusters,
			SelectedIdx:    0,
			LastGPressed:   0,
			LastEscPressed: 0,
			LastZPressed:   0,
		},
		Selections: *NewSelectionState(),
		UI: UIState{
			SearchQuery:       "",
			ActiveFilter:      "",
			Command:           "",
			IsVersionOutdated: false,
			LatestVersion:     nil,
			CommandInputKey:   0,
			Sort:              DefaultSortConfig(),
		},
		Modals: ModalState{
			ConfirmTarget:       nil,
			ConfirmSyncPrune:    false,
			ConfirmSyncWatch:    true,
			ConfirmSyncSelected: 0,
			ConfirmSyncLoading:  false,
			InitialLoading:      false,
			RollbackAppName:     nil,
			SyncViewApp:         nil,
		},
		Server:          nil,
		Apps:            []App{},
		APIVersion:      "",
		SavedNavigation: nil,
		SavedSelections: nil,
	}
}
