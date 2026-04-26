# ADR-0003: Async Message Gating (Epoch + Target)

## Status

Accepted

## Context

Argonaut is a Bubble Tea TUI: most API calls are dispatched as `tea.Cmd`s that run asynchronously and return a `tea.Msg` to the `Update` loop. Two events break the naive "the message I get back belongs to the operation I just started" assumption:

1. **Context switch.** When the user switches ArgoCD contexts (`switchEpoch++`), in-flight commands keyed to the old server can still complete and deliver messages to the new context, where they don't apply.
2. **Target switch.** When a modal is open and waiting on a load/execute, the user can dismiss it and open a new one (different resource) before the original request returns. The late completion can then clobber the unrelated newer modal.

Both have happened in practice. A single missing field on one message variant is enough to silently corrupt UI state — there's no compile-time check that says "this `tea.Msg` is stale, drop it."

## Decision

### Rule 1: Capture `m.switchEpoch` *before* every early return in a `tea.Cmd` producer.

```go
func (m *Model) loadX(target T) tea.Cmd {
    epoch := m.switchEpoch          // capture FIRST
    if m.state.Server == nil {
        return func() tea.Msg {
            return XErrorMsg{Target: target, Error: "...", SwitchEpoch: epoch}
        }
    }
    return func() tea.Msg {
        // ... uses epoch
    }
}
```

A nil-server (or any other) fast path that returns before `epoch := m.switchEpoch` will emit `SwitchEpoch: 0`. Once the user has ever switched contexts, the receiver drops it as stale and the UI hangs on Loading.

### Rule 2: Every message variant in a flow carries `SwitchEpoch` and (when the flow targets a specific resource) `Target`.

If `XLoadedMsg` carries `Target`, then `XErrorMsg` and `XExecutedMsg` and any tick/refresh messages in the same flow carry `Target` too. Asymmetry is the bug.

### Rule 3: Receivers gate by both, in this order, before mutating modal state:

```go
case XErrorMsg:
    if msg.SwitchEpoch != m.switchEpoch { return m, nil }
    st := m.state.Modals.X
    if st == nil || m.state.Mode != ModeX { return m, nil }
    if st.Target != msg.Target { return m, nil }
    // ... safe to mutate st
```

### Rule 4: Side effects that aren't tied to the modal run *outside* the gate.

Status messages and resource-tree refreshes belong to the operation, not the modal. They should fire even if the user has already moved on from the modal that started them. Only the modal-state mutation (close, set Error, set Loading) is gated by Target.

```go
case XExecutedMsg:
    if msg.SwitchEpoch != m.switchEpoch { return m, nil }
    m.statusService.Set("...")  // always runs
    if st := m.state.Modals.X; st != nil && st.Target == msg.Target {
        m.state.Mode = ModeNormal
        m.state.Modals.X = nil  // gated
    }
    return m, m.refreshTree(...) // always runs
```

## Consequences

- A late tick/error/completion from a previous modal session cannot clobber a newer modal.
- Context switches cleanly cut off all in-flight work without per-handler ad-hoc cleanup.
- Adding a new async flow is a checklist: every msg variant gets `SwitchEpoch` + `Target`; producer captures epoch first; receivers gate both. Reviewers can grep for the pattern.
- Mechanical cost: `Target` field on every msg variant, two extra checks per receiver. Cheap.

## References

- `cmd/app/api_integration.go` — producers (`loadResourceActions`, `executeResourceAction`, etc.)
- `cmd/app/model.go` — receivers
- `pkg/model/messages.go` — message definitions
