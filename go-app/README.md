# ArgoCD Apps - Go Implementation

This is the Go port of the TypeScript React+Ink ArgoCD application, built using Bubbletea and Lipgloss.

## Phase 2 Status: ✅ COMPLETE

### Architecture Overview

The Go implementation follows the same "screaming architecture" principles as the TypeScript version:

```
go-app/
├── cmd/app/              # Main application entry point
│   ├── main.go          # Application bootstrap
│   ├── model.go         # Bubbletea Model with MVU pattern
│   └── view.go          # Bubbletea View rendering
├── pkg/
│   ├── model/           # Core domain types and state
│   │   ├── types.go     # Domain types (App, View, Mode, etc.)
│   │   ├── state.go     # Application state structures
│   │   └── messages.go  # Bubbletea message types
│   └── services/        # Business logic services
│       ├── argo.go      # ArgoCD API service interface
│       ├── navigation.go # Navigation logic service
│       └── status.go    # Status/logging service
└── go.mod               # Go module dependencies
```

### Key Features Implemented

- ✅ **MVU Pattern**: Full Model-View-Update architecture with Bubbletea
- ✅ **Service Abstraction**: Clean interfaces matching TypeScript services
- ✅ **Type Safety**: Complete Go type system mapping from TypeScript
- ✅ **Message Handling**: All TypeScript actions converted to Go messages
- ✅ **Navigation Logic**: Drill-down and selection behaviors
- ✅ **Keyboard Handling**: Vi-style navigation (hjkl, enter, space, esc)
- ✅ **Terminal UI**: Styled rendering with Lipgloss

### Services Architecture

#### ArgoApiService
- Interface-based design for ArgoCD API interactions
- Event-driven architecture with channels
- Mock implementation with placeholder for real API calls

#### NavigationService  
- Pure functions for navigation logic
- Drill-down hierarchy: Clusters → Namespaces → Projects → Apps
- Selection toggling with set operations

#### StatusService
- Configurable logging with different levels (info, warn, error, debug)
- Pluggable handlers for different output destinations
- Current status tracking

### Message System

All TypeScript Redux actions are mapped to Go Bubbletea messages:

- **Navigation**: `SetViewMsg`, `SetSelectedIdxMsg`, `ResetNavigationMsg`
- **Selection**: `SetSelectedAppsMsg`, `ClearAllSelectionsMsg`
- **UI State**: `SetSearchQueryMsg`, `SetCommandMsg`, `ClearFiltersMsg`
- **Data**: `SetAppsMsg`, `SetServerMsg`, `AppsLoadedMsg`
- **System**: `WindowSizeMsg`, `KeyMsg`, `QuitMsg`

### Build & Run

```bash
# Build the application
go build -o bin/a9s ./cmd/app

# Run the application
./bin/a9s
```

### Dependencies

- **bubbletea**: TUI framework for Go
- **lipgloss**: Style definitions and rendering

### Controls

- `↑/k`: Move up
- `↓/j`: Move down
- `Enter`: Drill down / Select
- `Space`: Toggle selection (apps view only)
- `Esc`: Back / Clear filters
- `/`: Search mode (placeholder)
- `:`: Command mode (placeholder)
- `r`: Refresh (placeholder)
- `q/Ctrl+C`: Quit

### Type Mappings

Complete mapping from TypeScript to Go types:

- `Set<string>` → `map[string]bool`
- `string?` → `*string`
- `Promise<T>` → channels or `(T, error)` tuple
- `Result<T, Error>` → `(T, error)` tuple
- Redux actions → Bubbletea messages

### Mock Data

The application includes sample data for demonstration:
- 3 mock applications with different sync/health states  
- Server configuration pointing to example ArgoCD instance
- Placeholder clusters, namespaces, and projects

### Next Steps (Phase 3)

The foundation is now ready for:

1. **Real API Integration**: Replace mock ArgoApiService with actual ArgoCD client
2. **Advanced UI Features**: Search, filtering, command mode implementation  
3. **Application Operations**: Sync, rollback, resource viewing
4. **Configuration**: CLI args, config files, authentication
5. **Error Handling**: Robust error states and recovery
6. **Testing**: Unit tests for services and integration tests

This Go implementation maintains full feature parity with the TypeScript version's architecture while leveraging Go's type system and Bubbletea's MVU pattern.