# Architecture

Before modifying API call sites, timeout handling, or the HTTP client layer, read the relevant ADRs in `docs/architecture/decisions/`:

- **ADR-0002**: API timeout strategy — timeouts must be set at call sites using `appcontext.WithAPITimeout()` or `WithMinAPITimeout()`, never hardcoded with `context.WithTimeout()` and never inside `Client.Get/Post/Put/Delete`. See the ADR for the full rationale.

# E2E Test Instructions

## Running E2E Tests

### Run all E2E tests:
```bash
go test -tags e2e -v
```

### Run a specific E2E test:
```bash
go test -tags e2e -run TestSyncLastAppShowsCorrectConfirmation -v
```

### Run E2E tests in parallel with custom parallelism:
```bash
go test -tags e2e -v -parallel 4
```

## Test Structure

E2E tests are located in the `/e2e` directory and require the `e2e && unix` build tags. They:

1. Build a test binary from `cmd/app`
2. Start mock ArgoCD servers
3. Use TUI testing framework to simulate user interactions
4. Verify expected behavior through snapshots and API call recording

## Key Test Files

- `sync_test.go` - Tests for sync functionality including the bug fix for last app selection
- `commands_test.go` - Command execution tests
- `auth_*_test.go` - Authentication scenarios
- `driver_unix_test.go` - Test framework and mock server implementations