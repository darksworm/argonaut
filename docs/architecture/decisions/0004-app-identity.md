# ADR-0004: App Identity Is `(Name, AppNamespace)`, Never Name Alone

## Status

Accepted

## Context

ArgoCD supports multi-tenant deployments where the same `Application` name can exist in different ArgoCD namespaces (`spec.application.namespaces` in the ArgoCD ConfigMap). For example, two teams each have an Application called `my-app`, one in `argocd`, one in `team-a`.

The ArgoCD REST API treats these as distinct: every endpoint that operates on an Application accepts both `name` and `appNamespace` (sometimes called `appNs`). Sending only the name resolves to whichever app the server picks first — which may not be the one the user is looking at.

Argonaut's `m.state.Apps` is a flat slice. A first-name-match lookup —

```go
for i := range m.state.Apps {
    if m.state.Apps[i].Name == sel.AppName {
        appNamespace = m.state.Apps[i].AppNamespace
        break
    }
}
```

— silently picks the wrong app whenever names collide. The bug is invisible in single-tenant environments and is hard to reproduce in tests unless you remember to seed two apps with the same name.

## Decision

### Rule 1: Apps are identified by `(Name, AppNamespace)`. Never by Name alone.

Anywhere code constructs an API call, builds a target struct, looks up cluster info, or compares apps for equality, both fields are required. A function signature that takes only `appName string` is a bug waiting to happen — pass the whole `model.App` or both fields.

### Rule 2: The current view's app is `m.state.UI.TreeApp` (a `*TreeAppInfo`).

`TreeAppInfo` carries `Name`, `AppNamespace`, `DestNamespace`, and `Project` together. It is set atomically by `Model.setTreeApp` when entering the tree view, so callers in tree-scoped code never have to reconstruct it.

> Note: `NavigationState.TreeApp` also exists in the schema for legacy reasons. **Use `m.state.UI.TreeApp`** — that's the field everything else reads and writes.

### Rule 3: When the selection's app name matches `UI.TreeApp.Name`, prefer `UI.TreeApp.AppNamespace`.

In tree view the user is by definition looking at exactly one app. Use the scoped app's namespace before falling back to any name-only scan over `m.state.Apps`:

```go
var appNamespace *string
if treeApp := m.state.UI.TreeApp; treeApp != nil && treeApp.Name == sel.AppName {
    appNamespace = treeApp.AppNamespace
}
if appNamespace == nil {
    // last-resort fallback for app-of-apps child selections, etc.
    for i := range m.state.Apps {
        if m.state.Apps[i].Name == sel.AppName { ... }
    }
}
```

### Rule 4: Tests for any feature that issues an API call against an app must include a "two apps share a name" case.

`TestHandleResourceAction_DisambiguatesAppByNamespace`, `TestResCommand_CursorPos_PreservesNamespace`, and `TestHandleOpenK9s_UsesAppNamespaceForClusterLookup` are the templates. The setup is small (two apps with the same `Name`, different `AppNamespace`) and catches the entire bug class.

## Consequences

- Multi-tenant ArgoCD deployments work correctly. Single-tenant users see no behavior change.
- New features that touch apps must thread `AppNamespace` through the call chain. This is occasionally noisier but ensures correctness by construction.
- Any helper that takes `appName string` should be reviewed for upgrade to `(name, appNamespace *string)` or `model.App`.

## References

- `pkg/model/state.go` — `TreeAppInfo`, `NavigationState.TreeApp`, `UIState.TreeApp`
- `cmd/app/namespace_disambiguation_test.go` — canonical test pattern
- `cmd/app/input_handlers.go:handleResourceAction` — example of correct lookup
