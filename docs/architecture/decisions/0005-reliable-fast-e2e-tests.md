# ADR-0005: Reliable Fast E2E Tests

## Status

Accepted

## Context

Argonaut's e2e suite drives the real built binary through a PTY against a stack of `httptest` mock servers. Over time the suite accumulated ~70+ tests with a wall-clock around 16s and intermittent CI flakes. A focused investigation showed the same anti-patterns repeating across files — almost none of the slowness was inherent to the things being tested, and almost none of the flakes were genuine race conditions in production code. They were test-side time arithmetic that worked on a fast dev box but broke on a 2-core CI runner.

Wall-clock is now ~3.4s locally, ~10s on CI, and the suite runs 15/15 stable on the worst-case CI configuration we could reproduce. This ADR codifies the rules that got it there so the next round of tests doesn't regress to the same patterns.

## Decision

### Rule 1: Poll for state, never sleep on time.

A fixed `time.Sleep` is either too long (slowing the suite) or too short (flaking on slow runners). Replace every gate-style sleep with a state poll.

```go
// Wrong — pads against the worst-case render time.
_ = tf.Send("sort name desc")
_ = tf.Enter()
time.Sleep(500 * time.Millisecond)
if !strings.Contains(tf.SnapshotPlain(), "▼") { t.Fatal(...) }

// Right — exits as soon as the new state is observed.
_ = tf.Send("sort name desc")
_ = tf.Enter()
if !tf.WaitForPlain("▼", 2*time.Second) { t.Fatal(...) }
```

The exceptions are negative assertions ("verify X does *not* happen") — those still need a bounded sleep, but that sleep should be small (50–100ms) and ideally combined with a no-op key that round-trips through the app as a barrier confirming earlier input was processed.

### Rule 2: A `WaitFor*` that times out is a test failure. Always check the return.

```go
// Wrong — if the wait times out, follow-up assertions run on stale state
// and may "pass" by coincidence.
waitUntil(t, func() bool { return f(...) }, 2*time.Second)
if positionOf(...) < ... { t.Fatal(...) }

// Right — surface the timeout as the actual failure cause.
if !waitUntil(t, func() bool { return f(...) }, 2*time.Second) {
    t.Fatalf("expected ordering not reached:\n%s", tf.Screen())
}
```

### Rule 3: Pick the right snapshot helper for what you're asserting.

The framework exposes three:

- `Snapshot()` — raw cumulative byte ring (with ANSI). Anything that ever appeared is here.
- `SnapshotPlain()` — `Snapshot()` with ANSI escapes stripped.
- `Screen()` — the terminal-emulator's view of what the user sees right now.

Cumulative snapshots are forgiving but lossy:
- They match a substring that ever appeared, even if it's been overwritten — good for "this happened at some point" (e.g., "Mock k9s was launched"), wrong for "this is the current state".
- ANSI cursor-positioning sequences can interleave between adjacent rendered characters, so `SnapshotPlain()` may not contain `"jk"` even when the user's terminal clearly shows `Search > jk`. Use `Screen()` (or `WaitForScreen`) for assertions on rendered prose.

### Rule 4: Don't assert on transition history under CPU load.

Bubbletea's diff renderer only writes a string to the terminal if the model held that state on a render frame. Two messages that arrive within one batch-drain window collapse into a single update, and the intermediate state never reaches the PTY. Tests that say "first OutOfSync should appear, then Synced" are timing-sensitive and break on slow CI.

Make the assertion depend on *final state* (or any single state), and engineer the mock so that state can only be produced by the path under test:

```go
// Wrong — relies on the renderer producing two distinct frames.
// REST sends OutOfSync; SSE sends OutOfSync then Synced.
if !tf.WaitForPlain("OutOfSync", ...) { t.Fatal(...) }
if !tf.WaitForPlain("Synced", ...)    { t.Fatal(...) }

// Right — the only way "OutOfSync" can appear is if streaming was applied.
// REST sends Synced; SSE sends one OutOfSync event.
if !tf.WaitForPlain("OutOfSync", 5*time.Second) {
    t.Fatal("streaming update did not reach the apps view")
}
```

### Rule 5: PTY preserves keystroke order. Coalesce sequential `Send`s.

A `Send("j") + time.Sleep(100ms) + Send("K")` triple is identical to `Send("jK")` from the app's point of view — the bytes hit `tty` in order and the receiver dequeues them sequentially. The only time a barrier between keystrokes is needed is when the *render* of the first key must complete before the second is processed (cursor must move to row N *before* Enter activates that row). In that case, use a **render-level** barrier (e.g., `WaitForScreen` on a marker that proves the cursor settled), not a fixed sleep.

### Rule 6: Production timing knobs must be overridable from tests.

Reconnect delays, debounces, retry windows, and batch drains exist for real-user UX (anti-thrash, anti-spam). Tests don't want them. Wire them through env vars with sane defaults:

```go
var reconnectDelay = func() time.Duration {
    if v := os.Getenv("ARGONAUT_PF_RECONNECT_DELAY"); v != "" {
        if d, err := time.ParseDuration(v); err == nil && d >= 0 {
            return d
        }
    }
    return 2 * time.Second
}()
```

The framework sets the test-friendly values once in `StartAppArgs`; individual tests override per-case via `extraEnv`. Always reject negative parsed values — `time.ParseDuration("-5s")` succeeds and would produce tight loops.

Currently overridable: `ARGONAUT_RETRY_MAX_ATTEMPTS`, `ARGONAUT_RETRY_INITIAL_DELAY`, `ARGONAUT_PF_RECONNECT_DELAY`, `ARGONAUT_WATCH_SCOPE_DEBOUNCE`, `ARGONAUT_WATCH_BATCH_DRAIN`.

### Rule 7: Cleanup is LIFO. Register process-kill *after* `t.TempDir()`.

`t.TempDir()` registers a `RemoveAll` cleanup automatically. If the test body does `t.Cleanup(tf.Cleanup)` *before* calling `tf.SetupWorkspace()` (which calls `t.TempDir()` internally), the RemoveAll runs first while the app subprocess is still alive — producing intermittent `unlinkat: directory not empty` errors for tests that mutate config (themes, etc.).

Fix once at the framework level: have `SetupWorkspace` register the process-kill cleanup itself, so it runs after the auto-registered RemoveAll registers but before it runs. Tests don't need to know.

### Rule 8: Mock-server handlers must be bounded.

```go
// Wrong — deadlocks t.Cleanup. httptest.Server.Close blocks on
// in-flight handlers but does NOT cancel their request contexts.
mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
    sendInitialEvent(w)
    <-r.Context().Done()
})

// Right — bounded select. Exits early if the client disconnects,
// hard-caps so cleanup never deadlocks.
mux.HandleFunc("/api/v1/stream/applications", func(w http.ResponseWriter, r *http.Request) {
    sendInitialEvent(w)
    select {
    case <-r.Context().Done():
    case <-time.After(500 * time.Millisecond):
    }
})
```

### Rule 9: Disable real-network behaviour when `ARGONAUT_E2E=1`.

Anything that talks to the public internet during normal startup (the GitHub-API update check, telemetry, etc.) is dead weight in tests: it adds latency, paints unrelated UI ("New version available"), and contends for CPU. The framework already exports `ARGONAUT_E2E=1` to every subprocess; production code should branch on it for these specific external calls and short-circuit.

### Rule 10: Match `parallel` to the runner's core count.

Running `go test -parallel N` with N greater than the available cores oversubscribes the CPU. The PTY subprocess + mock server + `go test` framework all contend, and timing assertions that work on a 16-core dev box collapse on a 2-core CI runner.

- **Local dev:** `parallel = nproc` (or just below).
- **CI:** `parallel = runner cores` (1× oversubscription). On GitHub Actions standard runners that's 2.

To reproduce CI failures locally: `taskset -c 0,1 go test -tags e2e ./e2e -count=1 -parallel 4`. This is how every flake in the investigation was caught.

### Rule 11: Build cache is enough.

`testmain_test.go` rebuilds the test binary on every test run via `go build`. That's fine — Go's incremental build cache makes it ~120ms once warm. Don't add manual caching layers; they only matter when source actually changed.

## Consequences

- New tests that follow the rules: <200ms each, deterministic across local and CI.
- Adding a new async flow with timing knobs is a checklist: extract the constant to a `var`, read it from `ARGONAUT_<NAME>` (with `>= 0` guard), set the test-friendly value in `StartAppArgs`.
- Mechanical cost: a handful of env-var indirections in `pkg/retry`, `pkg/portforward`, and `cmd/app/api_integration.go`. Production behaviour is unchanged when the env var is unset.
- Reviewers can grep for `time.Sleep(` in `e2e/*_test.go` — every remaining call should have an explicit comment justifying why a poll won't work (almost always: a bounded negative assertion).

## References

- `e2e/driver_unix_test.go` — `WaitForPlain`, `WaitForScreen`, `waitUntil`, `mergeEnv`, the env defaults set in `StartAppArgs`.
- `pkg/retry/retry.go`, `pkg/portforward/kubectl.go`, `cmd/app/api_integration.go`, `cmd/app/upgrade.go` — overridable timing knobs and the `ARGONAUT_E2E=1` short-circuits.
- `.github/workflows/test.yml` — `make test PARALLEL=2` matches the 2-core runner.
- PR #240 — the cleanup that produced the rules above; commits there are the worked examples for each rule.
