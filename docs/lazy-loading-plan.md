# Performance Optimization for Large ArgoCD Deployments

## How ArgoCD Web UI Actually Works (from source analysis)

The ArgoCD web UI (`ui/src/app/applications/`) uses this pattern:

1. **Field selection on list AND watch** — requests only specific fields via `fields` query param, dramatically reducing response size. This is NOT in the swagger spec but IS supported (implemented via `pkg/apiclient/application/forwarder_overwrite.go` on the server).
2. **Load all apps** with field selection → get `resourceVersion` from response
3. **Start watch** from that `resourceVersion` (only gets changes, not full dump) with field selection
4. **Buffer watch events** at 500ms with RxJS `bufferTime(500)`
5. **All filtering is client-side** — cluster, namespace, project, health, sync all extracted from the loaded (minimal) app data

**The ArgoCD UI does NOT use separate cluster/project endpoints for navigation.** Everything comes from the app list with field selection.

### ArgoCD Web UI's Field Selection (for reference)
```
List fields:  metadata.resourceVersion, items.metadata.name, items.metadata.namespace,
              items.metadata.labels, items.spec, items.status.sync.status,
              items.status.health, items.status.resources, ...
Watch fields: result.type, result.application.metadata.name, result.application.spec, ...
```

Argonaut needs fewer fields than the ArgoCD web UI — notably we don't use `metadata.labels` (no label filtering) and don't need `status.resources` in the list call (only in watch for tree view updates).

## What Argonaut Should Do (matching ArgoCD's proven pattern)

### Current Problems
1. No field selection → full Application objects returned (huge response)
2. No `resourceVersion` → watch stream sends ALL apps as initial events (duplicate of list)
3. No event batching → each SSE event triggers a full render cycle
4. `getVisibleItems()` does O(n) scan per render

## Implementation Phases (4 phases)

---

### Phase 1: Field Selection on List and Watch (Biggest Impact) ✅ DONE

Add `fields` query parameter to both list and watch API calls, matching what the ArgoCD web UI does. This alone can reduce response size by 80%+.

**Defined field constants** in `pkg/api/applications.go`:
```go
var AppListFields = []string{
    "metadata.resourceVersion",
    "items.metadata.name",
    "items.metadata.namespace",
    "items.metadata.ownerReferences",
    "items.spec.destination",
    "items.spec.project",
    "items.status.sync.status",
    "items.status.health",
    "items.status.operationState.finishedAt",
    "items.status.operationState.startedAt",
}

var AppWatchFields = []string{
    "result.type",
    "result.application.metadata.name",
    "result.application.metadata.namespace",
    "result.application.metadata.ownerReferences",
    "result.application.spec.destination",
    "result.application.spec.project",
    "result.application.status.sync.status",
    "result.application.status.health",
    "result.application.status.operationState.finishedAt",
    "result.application.status.operationState.startedAt",
    "result.application.status.resources",
}
```

**Changes made:**
- `pkg/api/applications.go` — `ListApplicationsWithMeta()` adds `?fields=` param, returns `resourceVersion`; `WatchApplicationsWithOptions()` accepts `WatchOptions{ResourceVersion, Fields, Projects}`
- `pkg/services/argo.go` — new `ListApplicationsWithMeta` and `WatchApplicationsWithOptions` on interface + impl
- `pkg/model/messages.go` — `AppsLoadedMsg` carries `ResourceVersion`
- `cmd/app/api_integration.go` — threads `resourceVersion` from list to watch, passes `AppWatchFields`
- `cmd/app/model.go` — stores `lastResourceVersion` on Model

---

### Phase 2: Batched SSE Event Processing

Match ArgoCD UI's 500ms event batching.

**Current flow**: `consumeWatchEvent()` (`cmd/app/api_integration.go:131`) reads one event, returns one message, triggers one render cycle. With 1000+ apps and no `resourceVersion`, this means 1000+ sequential renders on connect.

After Phase 1 with `resourceVersion`, the initial burst is eliminated. But batching is still valuable for periods of high activity (e.g., cluster-wide sync).

**New message** in `pkg/model/messages.go`:
```go
type AppsBatchUpdateMsg struct {
    Updates []AppUpdatedMsg
    Deletes []string
}
```

**Replace `consumeWatchEvent()`** with `consumeWatchEvents()` in `cmd/app/api_integration.go`:
```go
func (m *Model) consumeWatchEvents() tea.Cmd {
    return func() tea.Msg {
        // Block on first event
        ev, ok := <-m.watchChan
        if !ok { return nil }

        batch := collectEvent(ev)
        if batch.immediate { return batch.msg }

        // Drain for up to 500ms
        timer := time.NewTimer(500 * time.Millisecond)
        defer timer.Stop()
        for {
            select {
            case ev, ok := <-m.watchChan:
                if !ok { break }
                if result := collectEvent(ev); result.immediate {
                    break
                }
                batch.add(result)
            case <-timer.C:
                break
            }
        }
        return model.AppsBatchUpdateMsg{Updates: batch.updates, Deletes: batch.deletes}
    }
}
```

**Handler** in `cmd/app/model.go`:
- Apply all upserts/deletes in one pass
- Single index rebuild (Phase 3)
- Continue with `m.consumeWatchEvents()`

**Update call sites**: Replace `m.consumeWatchEvent()` with `m.consumeWatchEvents()`:
- `AppsLoadedMsg` handler (line 335)
- Remove individual `AppUpdatedMsg`/`AppDeletedMsg` handlers (replaced by batch)

**Files changed:**
- `pkg/model/messages.go` — add `AppsBatchUpdateMsg`
- `cmd/app/api_integration.go` — new `consumeWatchEvents()`
- `cmd/app/model.go` — batch handler, update call sites

---

### Phase 3: Client-Side Indexing

Pre-compute indices when apps change so `getVisibleItems()` uses O(1) lookups.

**New file**: `pkg/model/index.go`
```go
type AppIndex struct {
    Clusters        []string             // sorted unique
    Namespaces      []string             // sorted unique
    Projects        []string             // sorted unique
    ApplicationSets []string             // sorted unique

    ByCluster        map[string][]int    // clusterLabel → app indices
    ByNamespace      map[string][]int    // namespace → app indices
    ByProject        map[string][]int    // project → app indices
    ByApplicationSet map[string][]int    // appset → app indices
    NameToIndex      map[string]int      // app name → index
}

func BuildAppIndex(apps []App) *AppIndex
```

**Add `Index *AppIndex` to `AppState`** in `pkg/model/state.go:171`.

**Rebuild index** after every mutation of `m.state.Apps`:
- `AppsLoadedMsg` handler
- `AppsBatchUpdateMsg` handler
- `AppDeleteSuccessMsg` handler

**Rewrite `getVisibleItems()`** (`cmd/app/view.go:452`) to use index:
- `ViewClusters`: return `m.state.Index.Clusters` directly
- `ViewNamespaces` with cluster scope: look up `Index.ByCluster[cluster]` → extract unique namespaces
- `ViewProjects` with scope: intersect indices → extract unique projects
- `ViewApps`: gather app indices from scope intersections, sort copy
- Text search remains linear over the scope-filtered subset

**Files changed:**
- `pkg/model/index.go` — NEW
- `pkg/model/state.go` — add `Index` field
- `cmd/app/view.go` — rewrite `getVisibleItems()`
- `cmd/app/model.go` — add `BuildAppIndex()` calls

---

### Phase 4: Scoped Streaming with Project Filters

When user drills down to specific projects, restart SSE with `projects` query param (supported by ArgoCD API on both list and stream).

**Add options** to `pkg/api/applications.go`:
```go
type WatchOptions struct {
    Projects        []string
    ResourceVersion string
    Fields          []string
}
```

**Scoped watch trigger** — when `ScopeProjects` changes:
1. `m.argoService.Cleanup()` to close current stream
2. Start new watch with `WatchOptions{Projects: selectedProjects}`
3. Debounce 500ms to avoid thrashing during rapid navigation

**Files changed:**
- `pkg/api/applications.go` — `WatchOptions` struct, update URL building
- `pkg/services/argo.go` — update interface to accept options
- `cmd/app/api_integration.go` — `restartWatchWithScope()`
- `cmd/app/model.go` — trigger reconnect on scope change

---

## Files Summary

| File | Phase | Change |
|------|-------|--------|
| `pkg/api/applications.go` | 1,4 | Add `fields`+`resourceVersion` params, `WatchOptions` |
| `pkg/services/argo.go` | 1,4 | Thread fields/resourceVersion/options through interface |
| `cmd/app/api_integration.go` | 1,2,4 | ResourceVersion threading, batch consumer, scoped watch |
| `cmd/app/model.go` | 2,3 | Batch handler, index rebuild calls |
| `pkg/model/messages.go` | 2 | Add `AppsBatchUpdateMsg` |
| `pkg/model/index.go` | 3 | NEW — `AppIndex` + `BuildAppIndex()` |
| `pkg/model/state.go` | 3 | Add `Index` field |
| `cmd/app/view.go` | 3 | Rewrite `getVisibleItems()` with index |

## Testing Strategy

**Unit tests:**
- `pkg/model/index_test.go` — `BuildAppIndex()` with edge cases (nil fields, duplicates, empty)
- Test field selection URL building (fields param correctly formatted)
- Test batch consumer (collects events, handles non-batchable, respects timeout)

**E2E tests** (existing framework in `/e2e`):
- Verify field selection doesn't break mock server responses
- Test navigation with indexed data
- Test with large mock dataset (500+ apps)

**Manual verification:**
```bash
go test ./...
go test -tags e2e -v
go build -o argonaut ./cmd/app && ./argonaut
```

## Implementation Order

Phase 1 (field selection) → Phase 2 (batching) → Phase 3 (indexing) → Phase 4 (scoped streaming)

Phase 1 is the biggest win: dramatically reduces API response size AND eliminates the SSE initial burst via `resourceVersion`. Phase 2+3 optimize client-side processing. Phase 4 reduces ongoing network traffic.
