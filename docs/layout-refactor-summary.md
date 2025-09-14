# Layout System Refactoring Summary

## Problem Solved
Fixed the inconsistent layout issue where the loading screen showed a small centered box instead of the full-height bordered container used in the main application views.

## Root Cause
The issue was caused by inconsistent layout implementation across different views. Each view manually implemented its own version of the "header + bordered content + status" pattern, leading to:

1. **Different sizing logic** - Some views calculated available space differently
2. **Inconsistent border application** - Loading view didn't use the standard `contentBorderStyle`
3. **Manual duplication** - Each view repeated the same layout structure
4. **Maintenance difficulty** - Adding new views required remembering complex layout patterns

## Solution: Centralized Layout Helpers

### Two Layout Functions Created:

#### 1. `renderFullScreenView(header, content, status, bordered)`
- **Purpose**: Standard full-terminal layout for most views
- **Used by**: Loading, Auth, Error, Connection Error views
- **Features**:
  - Consistent height calculation
  - Automatic border sizing
  - Header/content/status structure
  - Customizable border colors

#### 2. `renderModalContent(content)`
- **Purpose**: Simple modal content styling
- **Used by**: Help modal
- **Features**:
  - Standard padding
  - Consistent border style

### Views Migrated:
- ✅ **Loading View** - Now uses full-height bordered container
- ✅ **Auth Required View** - Uses red border for error styling
- ✅ **Error View** - Uses red border for error styling
- ✅ **Connection Error View** - Uses red border for error styling
- ✅ **Help Modal** - Uses standard modal content helper

### Benefits:
1. **Consistent Layout** - All views now use the same sizing logic
2. **DRY Principle** - No more duplicated layout code
3. **Easy Maintenance** - New views just call the helper functions
4. **Customizable** - Support for different border colors when needed
5. **Type Safety** - Proper color type handling with Go's `color.Color` interface

### Code Reduction:
- **Before**: ~150 lines of duplicated layout logic across 5 views
- **After**: ~50 lines total with 2 reusable helper functions
- **Savings**: ~100 lines of code eliminated, much cleaner and maintainable

## Technical Details

### Layout Helper API:
```go
// Standard full-screen layout
func (m Model) renderFullScreenView(header, content, status string, bordered bool) string

// Custom options (for special border colors)
func (m Model) renderFullScreenViewWithOptions(header, content, status string, opts FullScreenViewOptions) string

// Modal content only
func (m Model) renderModalContent(content string) string
```

### Usage Examples:
```go
// Loading view (simple)
return m.renderFullScreenView(header, content, status, true)

// Auth view (custom red border)
return m.renderFullScreenViewWithOptions(header, content, status, FullScreenViewOptions{
    ContentBordered: true,
    BorderColor:     outOfSyncColor, // red
})
```

This refactoring ensures that the layout inconsistency bug won't happen again when adding new views, as developers can simply use the centralized layout helpers instead of manually implementing layout logic.