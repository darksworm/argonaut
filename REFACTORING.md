# Argonaut Refactoring Plan

**Started:** 2025-09-30
**Branch:** refactor/remove-dead-code

## Overview
This document tracks the systematic refactoring of the argonaut codebase to improve maintainability, reduce duplication, and enhance testability.

## Progress Tracking

### ✅ Completed
1. [x] **Modal Rendering Duplication** - Consolidate 6 duplicate modal functions
2. [x] **Layout Constants Consolidation** - Extract duplicate layout constant blocks

### 🚧 In Progress
- None

### 📋 Planned

#### High Priority
3. [ ] **Update() Method** - Break 777-line function into message handlers
4. [ ] **Command Handler Extraction** - Break 424-line handleEnhancedCommandModeKeys()
5. [ ] **Color Code Consolidation** - Centralize all color definitions

#### Medium Priority
6. [ ] **Context Timeout Pattern** - Create helper for 11 duplicate patterns
7. [ ] **Error Handling Unification** - Consolidate error handling (16 sites)
8. [ ] **Model Struct Organization** - Group 23 fields into logical components
9. [ ] **view.go File Split** - Break 1,258-line file into focused files
10. [ ] **handleKeyMsg() Refactoring** - Improve 249-line function
11. [ ] **startDiffSession() Refactoring** - Break 95-line function
12. [ ] **getVisibleItems() Refactoring** - Break 170-line function

#### Low Priority (Polish)
13. [ ] **Lipgloss Style Builders** - Reduce 147 inline style creations
14. [ ] **Timeout Constants** - Document timeout intent
15. [ ] **Dimension Constants** - Centralize UI measurements
16. [ ] **Naming Consistency** - Standardize function names

---

## Detailed Work Log

### 2025-09-30 - Modal Rendering Consolidation
**Status:** ✅ Completed
**Files affected:**
- `cmd/app/view_modals.go`

**Changes:**
- Created `renderSimpleLoadingModal(message, borderColor, minWidth)` helper function
- Refactored 6 modal functions to use the helper:
  - `renderDiffLoadingSpinner()`: 13 lines → 3 lines
  - `renderTreeLoadingSpinner()`: 13 lines → 3 lines
  - `renderRollbackLoadingModal()`: 20 lines → 9 lines
  - `renderSyncLoadingModal()`: 13 lines → 3 lines
  - `renderInitialLoadingModal()`: 13 lines → 3 lines
  - `renderNoServerModal()`: 13 lines → 3 lines

**Tests:**
- All existing golden tests pass unchanged
- `TestGolden_DiffLoadingSpinner` ✓
- `TestGolden_SyncLoadingModal` ✓
- `TestGolden_InitialLoadingModal` ✓

**Code reduction:**
- Before: ~85 lines of duplicated code
- After: ~18 lines (helper) + ~24 lines (6 wrapper functions) = 42 lines
- **Reduction: 51% (43 lines saved)**

**Commits:**
- `77ee0f6` refactor: consolidate duplicate modal rendering functions

---

### 2025-09-30 - Layout Constants Consolidation
**Status:** ✅ Completed
**Files affected:**
- `cmd/app/view_constants.go` (new)
- `cmd/app/view_modals.go`
- `cmd/app/view_layout.go`
- `cmd/app/input_handlers.go`
- `cmd/app/view.go`

**Changes:**
- Created new `view_constants.go` file with package-level layout constants:
  - `layoutBorderLines = 2`
  - `layoutTableHeaderLines = 0`
  - `layoutTagLine = 0`
  - `layoutStatusLines = 1`
  - `layoutMarginTopLines = 1`
- Removed 4 duplicate inline `const` blocks from view functions
- Replaced all references with package-level constants in 5 files

**Tests:**
- All existing tests pass unchanged
- Build successful: `go build ./cmd/app` ✓
- Test suite: `go test ./...` ✓

**Code reduction:**
- Before: 4 duplicate const blocks (20 lines total)
- After: 1 centralized const block (11 lines)
- **Reduction: 45% (9 lines saved)**

**Benefits:**
- Single source of truth for layout dimensions
- Easier to adjust layout metrics globally
- Eliminates risk of inconsistent values across views

**Commits:**
- `37ad1da` refactor: consolidate duplicate layout constants

---

## Code Metrics

### Before Refactoring
- Largest function: 777 lines (Update method)
- Largest file: 1,258 lines (view.go)
- Duplicate modal code: ~90 lines × 6 functions
- Magic numbers: ~50+ inline
- Model struct fields: 23 flat fields

### After Refactoring (Target)
- Largest function: ~150 lines
- Largest file: ~200 lines
- Duplicate modal code: ~20 lines × 1 function
- Magic numbers: All named constants
- Model struct fields: 4 grouped components

### Current Metrics
- Largest function: 777 lines (Update method)
- Largest file: 1,258 lines (view.go)
- Duplicate modal code: ~~90 lines × 6 functions~~ → **42 lines total (51% reduction)** ✅
- Layout constant duplication: ~~20 lines in 4 blocks~~ → **11 lines in 1 block (45% reduction)** ✅
- Magic numbers: ~50+ inline
- Model struct fields: 23 flat fields

---

## Testing Strategy

For each refactoring:
1. Write tests for original implementation (if not already tested)
2. Verify existing tests pass
3. Perform refactoring
4. Verify all tests still pass (no changes to test behavior)
5. Run `go build ./cmd/app` to ensure compilation
6. Run `go test ./...` to ensure all tests pass
7. Verify golden tests unchanged (unless intentional UI change)
8. Commit with descriptive message

---

## Notes

- All refactorings must maintain backward compatibility
- No changes to existing test expectations or golden files
- Each commit should be atomic and independently buildable
- Focus on internal refactoring, not external API changes
