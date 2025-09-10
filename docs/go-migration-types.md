# Go Migration: TypeScript to Go Type Mappings

This document maps TypeScript interfaces and types to their equivalent Go struct definitions for Phase 2 of the migration.

## Core Domain Types

### AppItem (TypeScript → Go)

```typescript
// TypeScript
export type AppItem = {
  name: string;
  sync: string;
  health: string;
  lastSyncAt?: string; // ISO
  project?: string;
  clusterId?: string;
  clusterLabel?: string;
  namespace?: string;
  appNamespace?: string;
};
```

```go
// Go
type App struct {
    Name         string `json:"name"`
    Sync         string `json:"sync"`
    Health       string `json:"health"`
    LastSyncAt   *time.Time `json:"lastSyncAt,omitempty"`
    Project      *string `json:"project,omitempty"`
    ClusterID    *string `json:"clusterId,omitempty"`
    ClusterLabel *string `json:"clusterLabel,omitempty"`
    Namespace    *string `json:"namespace,omitempty"`
    AppNamespace *string `json:"appNamespace,omitempty"`
}
```

### View and Mode (TypeScript → Go)

```typescript
// TypeScript
export type View = "clusters" | "namespaces" | "projects" | "apps";
export type Mode = "normal" | "loading" | "search" | "command" | "help" | 
                   "license" | "confirm-sync" | "rollback" | "external" | 
                   "resources" | "auth-required" | "rulerline" | "error" | "logs";
```

```go
// Go
type View string
const (
    ViewClusters   View = "clusters"
    ViewNamespaces View = "namespaces"
    ViewProjects   View = "projects"
    ViewApps       View = "apps"
)

type Mode string
const (
    ModeNormal      Mode = "normal"
    ModeLoading     Mode = "loading"
    ModeSearch      Mode = "search"
    ModeCommand     Mode = "command"
    ModeHelp        Mode = "help"
    ModeLicense     Mode = "license"
    ModeConfirmSync Mode = "confirm-sync"
    ModeRollback    Mode = "rollback"
    ModeExternal    Mode = "external"
    ModeResources   Mode = "resources"
    ModeAuthRequired Mode = "auth-required"
    ModeRulerLine   Mode = "rulerline"
    ModeError       Mode = "error"
    ModeLogs        Mode = "logs"
)
```

## State Structures

### NavigationState (TypeScript → Go)

```typescript
// TypeScript
export interface NavigationState {
  view: View;
  selectedIdx: number;
  lastGPressed: number;
  lastEscPressed: number;
}
```

```go
// Go
type NavigationState struct {
    View           View      `json:"view"`
    SelectedIdx    int       `json:"selectedIdx"`
    LastGPressed   int64     `json:"lastGPressed"`
    LastEscPressed int64     `json:"lastEscPressed"`
}
```

### SelectionState (TypeScript → Go)

```typescript
// TypeScript  
export interface SelectionState {
  scopeClusters: Set<string>;
  scopeNamespaces: Set<string>;
  scopeProjects: Set<string>;
  selectedApps: Set<string>;
}
```

```go
// Go
type SelectionState struct {
    ScopeClusters   map[string]bool `json:"scopeClusters"`
    ScopeNamespaces map[string]bool `json:"scopeNamespaces"`
    ScopeProjects   map[string]bool `json:"scopeProjects"`
    SelectedApps    map[string]bool `json:"selectedApps"`
}

// Helper methods for set operations
func (s *SelectionState) AddCluster(cluster string) {
    if s.ScopeClusters == nil {
        s.ScopeClusters = make(map[string]bool)
    }
    s.ScopeClusters[cluster] = true
}

func (s *SelectionState) HasCluster(cluster string) bool {
    return s.ScopeClusters != nil && s.ScopeClusters[cluster]
}
```

### UIState (TypeScript → Go)

```typescript
// TypeScript
export interface UIState {
  searchQuery: string;
  activeFilter: string;
  command: string;
  isVersionOutdated: boolean;
  latestVersion?: string;
  commandInputKey: number;
}
```

```go
// Go  
type UIState struct {
    SearchQuery       string  `json:"searchQuery"`
    ActiveFilter      string  `json:"activeFilter"`
    Command           string  `json:"command"`
    IsVersionOutdated bool    `json:"isVersionOutdated"`
    LatestVersion     *string `json:"latestVersion,omitempty"`
    CommandInputKey   int     `json:"commandInputKey"`
}
```

### ModalState (TypeScript → Go)

```typescript
// TypeScript
export interface ModalState {
  confirmTarget: string | null;
  confirmSyncPrune: boolean;
  confirmSyncWatch: boolean;
  rollbackAppName: string | null;
  syncViewApp: string | null;
}
```

```go
// Go
type ModalState struct {
    ConfirmTarget    *string `json:"confirmTarget,omitempty"`
    ConfirmSyncPrune bool    `json:"confirmSyncPrune"`
    ConfirmSyncWatch bool    `json:"confirmSyncWatch"`
    RollbackAppName  *string `json:"rollbackAppName,omitempty"`
    SyncViewApp      *string `json:"syncViewApp,omitempty"`
}
```

### Complete AppState (TypeScript → Go)

```typescript
// TypeScript
export interface AppState {
  mode: Mode;
  terminal: TerminalState;
  navigation: NavigationState;
  selections: SelectionState;
  ui: UIState;
  modals: ModalState;
  server: Server | null;
  apps: AppItem[];
  apiVersion: string;
  loadingAbortController: AbortController | null;
}
```

```go
// Go
type AppState struct {
    Mode        Mode            `json:"mode"`
    Terminal    TerminalState   `json:"terminal"`
    Navigation  NavigationState `json:"navigation"`
    Selections  SelectionState  `json:"selections"`
    UI          UIState         `json:"ui"`
    Modals      ModalState      `json:"modals"`
    Server      *Server         `json:"server,omitempty"`
    Apps        []App           `json:"apps"`
    APIVersion  string          `json:"apiVersion"`
    // Note: AbortController doesn't have a Go equivalent, will use context.Context
}
```

## Action Types → Go Messages

All TypeScript action types will become Go message types in the Bubbletea pattern:

### Navigation Messages

```typescript
// TypeScript Actions
{ type: "SET_VIEW"; payload: View }
{ type: "SET_SELECTED_IDX"; payload: number }
{ type: "RESET_NAVIGATION"; payload?: { view?: View } }
```

```go
// Go Messages
type SetViewMsg struct {
    View View
}

type SetSelectedIdxMsg struct {
    SelectedIdx int
}

type ResetNavigationMsg struct {
    View *View
}
```

### Selection Messages

```typescript
// TypeScript Actions
{ type: "SET_SCOPE_CLUSTERS"; payload: Set<string> }
{ type: "SET_SELECTED_APPS"; payload: Set<string> }
{ type: "CLEAR_ALL_SELECTIONS" }
```

```go
// Go Messages
type SetScopeClustersMsg struct {
    Clusters map[string]bool
}

type SetSelectedAppsMsg struct {
    Apps map[string]bool
}

type ClearAllSelectionsMsg struct{}
```

### UI Messages

```typescript
// TypeScript Actions
{ type: "SET_SEARCH_QUERY"; payload: string }
{ type: "SET_COMMAND"; payload: string }
{ type: "CLEAR_FILTERS" }
```

```go
// Go Messages
type SetSearchQueryMsg struct {
    Query string
}

type SetCommandMsg struct {
    Command string
}

type ClearFiltersMsg struct{}
```

### Server/Data Messages

```typescript
// TypeScript Actions
{ type: "SET_APPS"; payload: AppItem[] }
{ type: "SET_SERVER"; payload: Server | null }
{ type: "SET_MODE"; payload: Mode }
```

```go
// Go Messages
type SetAppsMsg struct {
    Apps []App
}

type SetServerMsg struct {
    Server *Server
}

type SetModeMsg struct {
    Mode Mode
}
```

## Service Interfaces → Go Interfaces

### ArgoApiService (TypeScript → Go)

```typescript
// TypeScript
export class ArgoApiService {
  async listApplications(server: Server): Promise<Result<AppItem[], ApiError>>
  async watchApplications(server: Server, handler: ArgoApiEventHandler): Promise<() => void>
  async syncApplication(server: Server, appName: string): Promise<Result<void, ApiError>>
}
```

```go
// Go
type ArgoApiService interface {
    ListApplications(ctx context.Context, server *Server) ([]App, error)
    WatchApplications(ctx context.Context, server *Server) (<-chan ArgoApiEvent, error)
    SyncApplication(ctx context.Context, server *Server, appName string) error
}

type ArgoApiEvent struct {
    Type string      `json:"type"`
    Apps []App       `json:"apps,omitempty"`
    App  *App        `json:"app,omitempty"`  
    AppName string   `json:"appName,omitempty"`
    Error error      `json:"error,omitempty"`
    Status string    `json:"status,omitempty"`
}
```

### NavigationService (TypeScript → Go)

```typescript
// TypeScript
export class NavigationService {
  static drillDown(currentView: View, selectedItem: any): NavigationUpdate | null
  static toggleSelection(currentView: View, selectedItem: any): SelectionUpdate | null
  static validateBounds(selectedIdx: number, itemCount: number): number
}
```

```go
// Go
type NavigationService interface {
    DrillDown(currentView View, selectedItem interface{}) *NavigationUpdate
    ToggleSelection(currentView View, selectedItem interface{}, currentSelections map[string]bool) *SelectionUpdate
    ValidateBounds(selectedIdx, itemCount int) int
}

type NavigationUpdate struct {
    NewView                      *View            `json:"newView,omitempty"`
    ScopeClusters               map[string]bool  `json:"scopeClusters,omitempty"`
    ScopeNamespaces             map[string]bool  `json:"scopeNamespaces,omitempty"`
    ScopeProjects               map[string]bool  `json:"scopeProjects,omitempty"`
    ShouldResetNavigation       bool             `json:"shouldResetNavigation"`
    ShouldClearLowerLevelSelections bool         `json:"shouldClearLowerLevelSelections"`
}
```

## Key Differences: TypeScript vs Go

1. **Optional fields**: `string?` becomes `*string` in Go
2. **Sets**: `Set<string>` becomes `map[string]bool` in Go
3. **Promises**: `Promise<T>` becomes channels or direct returns with error
4. **Error handling**: `Result<T, Error>` becomes `(T, error)` tuple
5. **Event handlers**: Callbacks become channels
6. **AbortController**: Replaced with `context.Context`
7. **JSON tags**: Added for proper serialization
8. **Methods**: Static methods become interface methods or package functions

## Migration Strategy

1. **Phase 2**: Implement these Go structs and interfaces
2. **Phase 3**: Create Bubbletea models using these types
3. **Phase 4**: Replace TypeScript message dispatch with Go message passing

This type mapping ensures that the Go implementation maintains the same data contracts and behavior as the TypeScript version.