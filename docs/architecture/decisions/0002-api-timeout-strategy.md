# ADR-0002: API Timeout Strategy

## Status

Accepted

## Context

Argonaut provides a `request_timeout` config setting that lets users control how long API requests to ArgoCD are allowed to take. This is important for users with large deployments (>10k apps) where API responses can be slow.

### The Problem

The timeout infrastructure (`pkg/context/timeouts.go`) existed and was called in the right places, but it never actually took effect because of how Go contexts work:

1. **Call sites** in `api_integration.go` created contexts with hardcoded timeouts (e.g. `context.WithTimeout(ctx, 5*time.Second)`)
2. **Client methods** (`Client.Get`, etc.) then created child contexts with `WithAPITimeout(ctx)` using the configured value
3. Go contexts take the **minimum** deadline in the chain, so the hardcoded 5s always won over a configured 42s

Additionally, error messages referenced `DefaultTimeouts.API` (the configured value) even though the actual timeout was the hardcoded one, making debugging confusing.

### Why This Was Hard to Catch

- The code looked correct at each layer in isolation
- No compile-time or runtime error — the configured timeout was silently capped
- Error messages showed the configured value, masking the real timeout

## Decision

### Rule: Timeouts are set at call sites, not in the HTTP client layer

**Call sites** (functions in `api_integration.go`, `model_init.go`) are the only place that sets context timeouts. They know the operation semantics:

- **Standard API calls** → `appcontext.WithAPITimeout(context.Background())` — uses the configured `request_timeout`
- **Inherently slow operations** (diffs, rollbacks) → `appcontext.WithMinAPITimeout(context.Background(), minimum)` — uses `max(configured_timeout, minimum)` so slow operations get at least the minimum even if the user configured a shorter timeout
- **Batch loops** → timeout inside the loop (per-call), not a shared timeout outside — prevents later iterations from being starved

**Client methods** (`Client.Get`, `Post`, `Put`, `Delete`) do **not** set timeouts. They trust the caller's context. Adding a timeout here would undercut `WithMinAPITimeout` callers.

### The three timeout helpers

| Helper | When to use | Example |
|--------|------------|---------|
| `WithAPITimeout(ctx)` | Standard API calls | List apps, get resource tree, delete app |
| `WithMinAPITimeout(ctx, min)` | Slow operations that need a guaranteed floor | Diffs (45s), rollbacks (60s) |
| `WithSyncTimeout(ctx)` | Sync operations (if distinct timeout needed) | — |

### What NOT to do

- **Never use `context.WithTimeout(ctx, N*time.Second)` directly** for API calls — this bypasses the configured `request_timeout`
- **Never add `WithAPITimeout` inside `Client.Get/Post/Put/Delete`** — it undercuts `WithMinAPITimeout` callers
- **Never use a shared timeout across a batch loop** — use per-iteration timeouts instead

## Consequences

- Users can now actually control timeouts via `request_timeout` config
- Slow operations (diffs, rollbacks) have minimum floors so they work even with short configured timeouts
- Batch operations are fair — each iteration gets a full timeout budget
- Error messages show the actual timeout that was in effect
