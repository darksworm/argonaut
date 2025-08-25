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

### Phase 1: Foundation & State Management ✅ COMPLETED
- [x] Create AppStateContext with useReducer for centralized state management
- [x] Extract state interfaces and action types
- [x] Create initial state structure

### Phase 2: Component Extraction ✅ COMPLETED
- [x] Extract modal components (ConfirmSyncModal, RollbackModal, HelpModal)
- [x] Create view components (LoadingView, ListView, MainLayout)
- [x] Create input components (SearchBar, CommandBar)
- [x] Extract data processing logic (useVisibleItems hook)

### Phase 3: Input System Refactoring ✅ COMPLETED
- [x] Implement command pattern for input handling
- [x] Extract keyboard handling logic
- [x] Create command registry system
- [x] Replace massive useInput with organized command system

### Phase 4: Business Logic Separation ✅ COMPLETED
- [x] Create orchestrator services (AppOrchestrator)
- [x] Move API coordination logic out of components
- [x] Extract navigation logic into specialized hooks
- [x] Create lifecycle management hooks
- [x] Extract input system into dedicated hook
- [x] Separate live data management logic

### Phase 5: Integration & Testing ✅ CURRENT
- [x] Complete architecture foundation
- [ ] Integrate all components into new App.tsx
- [ ] Test all refactored components
- [ ] Ensure functionality preservation
- [ ] Clean up unused code
- [ ] Document new architecture

## Architecture Summary

We've successfully broken down the 1400-line monolithic App.tsx into:

**✅ Completed Components:**
- **State Management**: `AppStateContext` - Centralized state with useReducer
- **Modal Components**: `ConfirmSyncModal`, `RollbackModal`, `HelpModal`
- **View Components**: `LoadingView`, `ListView`, `MainLayout`, `SearchBar`, `CommandBar`  
- **Command System**: Complete command pattern with registry and prioritized handlers
- **Business Logic**: `AppOrchestrator` service for complex workflows
- **Specialized Hooks**: `useAppLifecycle`, `useInputSystem`, `useNavigationLogic`, `useLiveData`, `useVisibleItems`

**📋 Next Steps:**
- **Integration**: Wire everything together in a clean new App.tsx
- **Testing**: Verify all functionality works as expected
- **Cleanup**: Remove old patterns and ensure consistency

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