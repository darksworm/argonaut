# App.tsx Refactoring Progress

## Overview
Refactoring the monolithic 1400-line App.tsx component into a clean, maintainable architecture.

## Problems Identified
1. **State Management**: 20+ useState calls for UI state, navigation, selections, terminal dimensions
2. **Input Handling**: Massive useInput handler with mode-specific logic across 9 different modes
3. **Command Processing**: Large runCommand function handling 15+ commands
4. **Business Logic**: Mixed with presentation logic throughout
5. **View Logic**: Complex conditional rendering for different modes and views
6. **Side Effects**: Multiple useEffect hooks managing everything from auth to data fetching

## Refactoring Strategy

### Phase 1: Foundation & State Management ✅ CURRENT
- [ ] Create AppStateContext with useReducer for centralized state management
- [ ] Extract state interfaces and action types
- [ ] Create initial state structure

### Phase 2: Component Extraction
- [ ] Extract modal components (ConfirmSyncModal, RollbackModal, HelpModal)
- [ ] Create view components (ClusterView, NamespaceView, ProjectView, AppListView)
- [ ] Create layout components (Header, SearchBar, CommandBar, StatusBar)

### Phase 3: Input System Refactoring
- [ ] Implement command pattern for input handling
- [ ] Extract keyboard handling logic
- [ ] Create command registry system
- [ ] Replace massive useInput with organized command system

### Phase 4: Business Logic Separation
- [ ] Create orchestrator services
- [ ] Move API coordination logic out of components
- [ ] Extract navigation logic into specialized hooks
- [ ] Implement proper error boundaries

### Phase 5: Testing & Cleanup
- [ ] Test all refactored components
- [ ] Ensure functionality preservation
- [ ] Clean up unused code
- [ ] Document new architecture

## Current State Analysis

### State Variables Found (20+):
- Layout: `termRows`, `termCols`
- Mode/View: `mode`, `view`
- Authentication: `server`
- Data: `apps`, `apiVersion`
- UI State: `searchQuery`, `activeFilter`, `command`, `selectedIdx`, `status`, `isVersionOutdated`
- Selections: `scopeClusters`, `scopeNamespaces`, `scopeProjects`, `selectedApps`, `confirmTarget`
- Modal States: `rollbackAppName`, `syncViewApp`, `confirmSyncPrune`, `confirmSyncWatch`
- Navigation: `lastGPressed`
- Cleanup: `loadingAbortControllerRef`

### Input Modes Identified (9):
1. `loading` - Boot/auth process
2. `auth-required` - Authentication needed
3. `normal` - Main navigation mode
4. `search` - Search input mode
5. `command` - Command input mode
6. `confirm-sync` - Sync confirmation dialog
7. `help` - Help overlay
8. `rollback` - Rollback flow
9. `resources` - Resource stream view
10. `external` - External tool mode
11. `rulerline` - Ruler line mode

### Commands Identified (15+):
- Navigation: `cluster`, `namespace`, `project`, `app`
- Actions: `sync`, `diff`, `rollback`, `resources`
- Utilities: `help`, `logs`, `license`, `clear`, `all`, `login`, `q/quit/exit`

## Next Steps
Starting with Phase 1 - creating the centralized state management system.