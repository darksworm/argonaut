package model

import tea "github.com/charmbracelet/bubbletea/v2"

// Navigation Messages - correspond to TypeScript navigation actions

// SetViewMsg sets the current view
type SetViewMsg struct {
	View View
}

// SetSelectedIdxMsg sets the selected index
type SetSelectedIdxMsg struct {
	SelectedIdx int
}

// ResetNavigationMsg resets navigation state
type ResetNavigationMsg struct {
	View *View
}

// UpdateLastGPressedMsg updates the last G press timestamp
type UpdateLastGPressedMsg struct {
	Timestamp int64
}

// UpdateLastEscPressedMsg updates the last Esc press timestamp
type UpdateLastEscPressedMsg struct {
	Timestamp int64
}

// Selection Messages - correspond to TypeScript selection actions

// SetScopeClustersMsg sets the cluster scope
type SetScopeClustersMsg struct {
	Clusters map[string]bool
}

// SetScopeNamespacesMsg sets the namespace scope
type SetScopeNamespacesMsg struct {
	Namespaces map[string]bool
}

// SetScopeProjectsMsg sets the project scope
type SetScopeProjectsMsg struct {
	Projects map[string]bool
}

// SetSelectedAppsMsg sets the selected apps
type SetSelectedAppsMsg struct {
	Apps map[string]bool
}

// ClearAllSelectionsMsg clears all selections
type ClearAllSelectionsMsg struct{}

// ClearLowerLevelSelectionsMsg clears lower level selections based on view
type ClearLowerLevelSelectionsMsg struct {
	View View
}

// UI Messages - correspond to TypeScript UI actions

// SetSearchQueryMsg sets the search query
type SetSearchQueryMsg struct {
	Query string
}

// SetActiveFilterMsg sets the active filter
type SetActiveFilterMsg struct {
	Filter string
}

// SetCommandMsg sets the command
type SetCommandMsg struct {
	Command string
}

// BumpCommandInputKeyMsg bumps the command input key
type BumpCommandInputKeyMsg struct{}

// SetVersionOutdatedMsg sets the version outdated flag
type SetVersionOutdatedMsg struct {
	IsOutdated bool
}

// SetLatestVersionMsg sets the latest version
type SetLatestVersionMsg struct {
	Version *string
}

// ClearFiltersMsg clears all filters
type ClearFiltersMsg struct{}

// Modal Messages - correspond to TypeScript modal actions

// SetConfirmTargetMsg sets the confirm target
type SetConfirmTargetMsg struct {
	Target *string
}

// SetConfirmSyncPruneMsg sets the confirm sync prune flag
type SetConfirmSyncPruneMsg struct {
	SyncPrune bool
}

// SetConfirmSyncWatchMsg sets the confirm sync watch flag
type SetConfirmSyncWatchMsg struct {
	SyncWatch bool
}

// SetRollbackAppNameMsg sets the rollback app name
type SetRollbackAppNameMsg struct {
	AppName *string
}

// SetSyncViewAppMsg sets the sync view app
type SetSyncViewAppMsg struct {
	AppName *string
}

// ClearModalsMsg clears all modal state
type ClearModalsMsg struct{}

// Server/Data Messages - correspond to TypeScript data actions

// SetAppsMsg sets the applications list
type SetAppsMsg struct {
	Apps []App
}

// SetServerMsg sets the server configuration
type SetServerMsg struct {
	Server *Server
}

// SetModeMsg sets the application mode
type SetModeMsg struct {
	Mode Mode
}

// SetTerminalSizeMsg sets the terminal size
type SetTerminalSizeMsg struct {
	Rows int
	Cols int
}

// SetAPIVersionMsg sets the API version
type SetAPIVersionMsg struct {
	Version string
}

// API Event Messages - correspond to ArgoApiService events

// AppsLoadedMsg is sent when apps are loaded
type AppsLoadedMsg struct {
	Apps []App
}

// AppUpdatedMsg is sent when an app is updated
type AppUpdatedMsg struct {
	App App
}

// AppDeletedMsg is sent when an app is deleted
type AppDeletedMsg struct {
	AppName string
}

// AuthErrorMsg is sent when authentication is required
type AuthErrorMsg struct {
	Error error
}

// ApiErrorMsg is sent when there's an API error
type ApiErrorMsg struct {
	Message    string
	StatusCode int    `json:"statusCode,omitempty"` // HTTP status code if available
	ErrorCode  int    `json:"errorCode,omitempty"`  // API error code if available
	Details    string `json:"details,omitempty"`    // Additional error details
}

// StatusChangeMsg is sent when status changes
type StatusChangeMsg struct {
	Status string
}

// Navigation Event Messages - correspond to navigation service results

// NavigationUpdateMsg is sent when navigation should be updated
type NavigationUpdateMsg struct {
	NewView                         *View
	ScopeClusters                   map[string]bool
	ScopeNamespaces                 map[string]bool
	ScopeProjects                   map[string]bool
	SelectedApps                    map[string]bool
	ShouldResetNavigation           bool
	ShouldClearLowerLevelSelections bool
}

// SelectionUpdateMsg is sent when selections should be updated
type SelectionUpdateMsg struct {
	SelectedApps map[string]bool
}

// Terminal/System Messages

// WindowSizeMsg is sent when the terminal window is resized
type WindowSizeMsg tea.WindowSizeMsg

// KeyMsg wraps Bubbletea's KeyMsg
type KeyMsg tea.KeyMsg

// QuitMsg is sent to quit the application
type QuitMsg struct{}

// TickMsg is sent on timer ticks
type TickMsg struct{}

// Command Messages - for handling async operations

// LoadAppsCmd represents a command to load applications
type LoadAppsCmd struct {
	Server *Server
}

// SyncAppCmd represents a command to sync an application
type SyncAppCmd struct {
	Server  *Server
	AppName string
	Prune   bool
}

// WatchAppsCmd represents a command to start watching applications
type WatchAppsCmd struct {
	Server *Server
}

// Generic result messages for async operations

// ResultMsg wraps the result of an operation
type ResultMsg struct {
	Success bool
	Error   error
	Data    interface{}
}

// LoadingMsg indicates a loading state change
type LoadingMsg struct {
	IsLoading bool
	Message   string
}

// Sync completion messages

// SyncCompletedMsg indicates a single app sync has completed
type SyncCompletedMsg struct {
	AppName string
	Success bool
}

// MultiSyncCompletedMsg indicates multiple app sync has completed
type MultiSyncCompletedMsg struct {
	AppCount int
	Success  bool
}