# Argonaut Refactoring Plan

**Started:** 2025-09-30
**Branch:** refactor/remove-dead-code

## Overview
This document tracks the systematic refactoring of the argonaut codebase to improve maintainability, reduce duplication, and enhance testability.

## Progress Tracking

### ✅ Completed
1. [x] **Modal Rendering Duplication** - Consolidate 6 duplicate modal functions
2. [x] **Layout Constants Consolidation** - Extract duplicate layout constant blocks
3. [x] **Color Code Consolidation** - Centralize all color definitions
4. [x] **Context Timeout Pattern** - Create helper for 11 duplicate patterns
5. [x] **Timeout Constants** - Document timeout intent with semantic constants

### 🚧 In Progress
- None

### 📋 Planned

#### High Priority
6. [ ] **Update() Method** - Break 777-line function into message handlers
7. [ ] **Command Handler Extraction** - Break 424-line handleEnhancedCommandModeKeys()

#### Medium Priority
8. [ ] **Error Handling Unification** - Consolidate error handling (16 sites)
9. [ ] **Model Struct Organization** - Group 23 fields into logical components
10. [ ] **view.go File Split** - Break 1,258-line file into focused files
11. [ ] **handleKeyMsg() Refactoring** - Improve 249-line function
12. [ ] **startDiffSession() Refactoring** - Break 95-line function
13. [ ] **getVisibleItems() Refactoring** - Break 170-line function

#### Low Priority (Polish)
14. [ ] **Lipgloss Style Builders** - Reduce 147 inline style creations
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

### 2025-09-30 - Color Code Consolidation
**Status:** ✅ Completed
**Files affected:**
- `cmd/app/view_constants.go` (modified)
- `cmd/app/view.go` (modified)
- `cmd/app/main.go` (modified)
- `cmd/app/input_components.go` (modified)
- `cmd/app/view_modals.go` (modified)
- `cmd/app/model_init.go` (modified)
- `cmd/app/model_tables.go` (modified)
- `cmd/app/view_banner.go` (modified)
- `cmd/app/view_status.go` (modified)

**Changes:**
- Moved all color constants from `view.go` to `view_constants.go`
- Added new color constants for all previously inline colors:
  - Core UI colors: `magentaBright`, `yellowBright`, `dimColor`, `cyanBright`, `whiteBright`, `blueBright`
  - Status colors: `syncedColor`, `outOfSyncColor`, `progressColor`, `unknownColor`
  - Modal colors: `black`, `white`, `redColor`
  - Gray variants: `grayDesaturated`, `grayInactiveButton`, `grayButtonDisabled`
  - UI-specific: `grayBorder`, `grayPrompt`, `pinkSpinner`, `yellowTable`, `blueTable`, `grayBadgeBg`, `blackBadgeFg`, `grayServerLabel`
- Replaced all inline `lipgloss.Color()` calls with named constants across 9 files
- Updated `main.go` help color definitions to reference centralized constants

**Tests:**
- All existing tests pass unchanged
- Build successful: `go build ./cmd/app` ✓
- Test suite: `go test ./...` ✓

**Code reduction:**
- Before: 47 inline `lipgloss.Color()` calls scattered across 9 files
- After: 22 centralized color constants in 1 file
- **Benefits:**
  - Single source of truth for all application colors
  - Easier to adjust color scheme globally
  - Improved code readability with semantic color names
  - Eliminates magic number color codes

**Commits:**
- `829ead3` refactor: consolidate all color definitions into centralized constants

---

### 2025-09-30 - Context Timeout Pattern Consolidation
**Status:** ✅ Completed
**Files affected:**
- `cmd/app/context_helpers.go` (new)
- `cmd/app/api_integration.go` (modified)
- `cmd/app/model_init.go` (modified)

**Changes:**
- Created `context_helpers.go` with `contextWithTimeout()` helper function
- Replaced 11 duplicate context.WithTimeout patterns across 2 files
- Simplified all timeout context creation to use centralized helper

**Pattern replaced:**
```go
// Before:
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// After:
ctx, cancel := contextWithTimeout(10 * time.Second)
defer cancel()
```

**Occurrences replaced:**
- `api_integration.go`: 10 instances (5s, 10s, 30s, 45s, 60s timeouts)
- `model_init.go`: 1 instance (10s timeout)

**Tests:**
- All existing tests pass unchanged
- Build successful: `go build ./cmd/app` ✓
- Test suite: `go test ./...` ✓

**Benefits:**
- Eliminates boilerplate code duplication
- Consistent timeout context creation pattern
- Easier to modify timeout behavior globally if needed
- Cleaner, more readable code

**Commits:**
- `0c84e36` refactor: consolidate duplicate context timeout patterns

---

### 2025-09-30 - Timeout Constants Documentation
**Status:** ✅ Completed
**Files affected:**
- `cmd/app/context_helpers.go` (modified)
- `cmd/app/api_integration.go` (modified)
- `cmd/app/model_init.go` (modified)

**Changes:**
- Added semantic timeout constants to document intent of each timeout duration
- Created 5 named constants with descriptive comments:
  - `timeoutQuick` (5s): Quick operations like sync triggers, app list loading
  - `timeoutStandard` (10s): Standard operations like API version, resource tree, auth validation
  - `timeoutMedium` (30s): Medium operations like rollback session loading
  - `timeoutLong` (45s): Long operations like diff sessions, rollback diffs
  - `timeoutExtended` (60s): Extended operations like rollback execution
- Replaced all magic number timeout durations with named constants
- Removed unused `time` import from `api_integration.go`

**Pattern replaced:**
```go
// Before:
ctx, cancel := contextWithTimeout(10 * time.Second)

// After:
ctx, cancel := contextWithTimeout(timeoutStandard)
```

**Tests:**
- All existing tests pass unchanged
- Build successful: `go build ./cmd/app` ✓
- Test suite: `go test ./...` ✓

**Benefits:**
- Self-documenting code: timeout values now have semantic meaning
- Easier to understand operation expectations
- Simpler to adjust timeout categories globally
- Improved code maintainability

**Commits:**
- (pending)

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
- Color code duplication: ~~47 inline lipgloss.Color() calls~~ → **22 centralized constants** ✅
- Context timeout pattern: ~~11 duplicate 2-line patterns~~ → **1 helper function** ✅
- Timeout magic numbers: ~~11 inline duration values~~ → **5 semantic constants** ✅
- Magic numbers: ~40+ inline (colors and timeouts eliminated, others remain)
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
