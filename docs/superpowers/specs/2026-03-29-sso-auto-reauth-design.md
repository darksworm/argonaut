# SSO Auto Re-authentication Design

**Date:** 2026-03-29
**Scope:** SSO-only (username/password deferred)

## Problem

When an ArgoCD JWT expires or is revoked, argonaut drops to a dead-end `ModeAuthRequired` screen and the user must manually run `argocd login` and restart the app. The goal is to make re-authentication automatic and invisible — the app detects the expired session, triggers SSO login via the `argocd` CLI, and resumes without requiring user intervention beyond authenticating in the browser.

---

## Approach

Delegate the OIDC/SSO flow entirely to the `argocd` CLI via `tea.ExecProcess`. Argonaut suspends its TUI, hands the terminal to `argocd login <server> --sso`, waits for it to complete, then re-reads the updated token from the ArgoCD config and resumes.

This avoids reimplementing OIDC, PKCE, or keychain logic. The seam is a `JWTAuthProvider` interface so the CLI dependency can be swapped or mocked in tests.

---

## Architecture

### New package: `pkg/auth/`

Single file: `login.go`

```go
type LoginParams struct {
    ServerURL       string // bare hostname, no https:// prefix
    ContextName     string
    Insecure        bool
    GrpcWeb         bool
    GrpcWebRootPath string
    ConfigPath      string // empty = use argocd default
}

type JWTAuthProvider interface {
    LoginCmd(params LoginParams) *exec.Cmd
}

type ArgocdCLIAuthProvider struct{}

func (a ArgocdCLIAuthProvider) LoginCmd(params LoginParams) *exec.Cmd {
    // builds: argocd login <server> --sso [--insecure] [--grpc-web]
    //         [--grpc-web-root-path <p>] [--name <ctx>] [--config <path>]
}

func StripProtocol(url string) string {
    // "https://foo.com" → "foo.com", "http://foo.com:8080" → "foo.com:8080"
}
```

### Fix required: `pkg/model/types.go` + `pkg/config/cli_config.go`

`Server` struct is missing `GrpcWeb bool`. Add it and wire it up in `cli_config.go` where `GrpcWebRootPath` is already mapped. Without this, servers configured with `grpc-web: true` but no root path would have the flag silently dropped on re-login.

### Model field

`cmd/app/model.go` (or `model_init.go`): add `jwtAuthProvider auth.JWTAuthProvider`. Set to `ArgocdCLIAuthProvider{}` in `main.go`. Injected as a fake in tests.

---

## State & Messages

### New mode (`pkg/model/types.go`)

- `ModeReauthPending` — shown while `tea.ExecProcess` is in flight

No new modal state fields — no input forms.

### New messages (`pkg/model/messages.go`)

- `TriggerReauthMsg` — emitted when auth fails or token is absent; carries no payload (model already has everything needed)
- `ReauthCompleteMsg{Err error}` — `tea.ExecProcess` callback fires this after `argocd login` exits

### Reauth attempt counter

Model tracks `reauthAttempts int` (reset to 0 on successful load). Guards against infinite loops.

---

## Login Flow

### Startup (no token)

1. `validateAuthentication()` detects `Server.Token == ""` → emits `TriggerReauthMsg`
2. `TriggerReauthMsg` handler:
   - If already `ModeReauthPending`: drop (no-op)
   - Increment `reauthAttempts`; if > 2: transition to `ModeAuthRequired` with error message
   - Switch to `ModeReauthPending`
   - Return `tea.ExecProcess(m.jwtAuthProvider.LoginCmd(params), reauth callback)`
3. TUI suspends; `argocd login --sso` opens browser
4. `ReauthCompleteMsg{nil}`:
   - Re-read ArgoCD config from disk (existing `cli_config` logic)
   - Update `Server.Token`
   - Reset `reauthAttempts`
   - Call `validateAuthentication()` → continues to `ModeLoading` normally
5. `ReauthCompleteMsg{err}`:
   - Transition to `ModeAuthRequired` with descriptive error message

### Runtime expiry (mid-session)

1. SSE or API call returns HTTP 401 → existing code emits `AuthErrorMsg`
2. `AuthErrorMsg` handler: instead of → `ModeAuthRequired`, emit `TriggerReauthMsg`
3. Same `ExecProcess` flow above
4. On `ReauthCompleteMsg{nil}`: update token, increment `switchEpoch` to kill stale goroutines, restart watch stream (same machinery as context switching)

---

## `ModeReauthPending` View

Full-screen message:

```
Re-authenticating via SSO — check your browser.
```

Simple, no input required. Shown for however long `argocd login` takes.

---

## Error Handling

| Failure | Behaviour |
|---|---|
| `argocd` CLI not found | `ReauthCompleteMsg{err}` → `ModeAuthRequired` with: *"Re-auth failed: argocd CLI not found. Run `argocd login` manually."* |
| `argocd login` exits non-zero (cancelled, network error) | Same fallback with exit error shown |
| Re-login succeeds but 401 recurs (malformed write, etc.) | `validateAuthentication()` returns 401 → `TriggerReauthMsg` → attempt counter: max 2 attempts before falling back to `ModeAuthRequired` |
| `TriggerReauthMsg` while already in `ModeReauthPending` | No-op — drop the message |

---

## How `argocd login` Command Is Reconstructed

`argocd login` does **not** read or merge existing server settings — it overwrites them. All flags must be re-specified explicitly:

```
argocd login <StripProtocol(Server.BaseURL)>
  --sso
  --name <currentContextName>
  [--insecure]                  if Server.Insecure
  [--grpc-web]                  if Server.GrpcWeb   ← requires fix above
  [--grpc-web-root-path <p>]    if Server.GrpcWebRootPath != ""
  [--config <argoConfigPath>]   if non-default config path
```

All inputs come from existing model fields.

---

## ArgoCD Auth Signal

HTTP 401 is the single reliable signal for all auth failures (expired token, revoked token, SSO expiry, no session). The existing detection in `pkg/api/client.go` and `cmd/app/api_integration.go` is already correct — no changes needed to error detection.

---

## Testing Strategy (TDD)

### `pkg/auth/` — unit tests (write before implementation)

- `TestStripProtocol` — `https://foo.com` → `foo.com`, `http://foo.com:8080` → `foo.com:8080`, bare hostname unchanged
- `TestBuildLoginCmd_SSO` — minimal params produce correct args
- `TestBuildLoginCmd_Insecure` — `--insecure` present when `Insecure: true`
- `TestBuildLoginCmd_GrpcWeb` — `--grpc-web` present when `GrpcWeb: true`
- `TestBuildLoginCmd_GrpcWebRootPath` — `--grpc-web-root-path` present when set
- `TestBuildLoginCmd_CustomConfig` — `--config` present when `ConfigPath` non-empty
- `TestBuildLoginCmd_ContextName` — `--name` always present

### `cmd/app/` — model unit tests (write before implementation)

- `TestTriggerReauthOnNoToken` — model with empty `Server.Token` → `validateAuthentication()` → model in `ModeReauthPending`
- `TestTriggerReauthOnAuthError` — `AuthErrorMsg` → model emits `TriggerReauthMsg` (not `ModeAuthRequired`)
- `TestReauthCompleteSuccess` — `ReauthCompleteMsg{nil}` with mocked config re-read → calls `validateAuthentication()`
- `TestReauthCompleteFailure` — `ReauthCompleteMsg{err}` → `ModeAuthRequired` with error message
- `TestReauthAlreadyPending` — `TriggerReauthMsg` while `ModeReauthPending` → no-op
- `TestReauthInfiniteLoopGuard` — attempt counter hits limit → `ModeAuthRequired` instead of another `TriggerReauthMsg`
- `TestReauthViewMessage` — `ModeReauthPending` renders the expected "Re-authenticating via SSO" string

### E2E tests (`e2e/`)

- **`TestSSOReauthOnStartup`** — mock server, no token in ArgoCD config; mock `JWTAuthProvider` writes a valid token to temp config; assert: view shows reauth message during `ModeReauthPending`, then app transitions → `ModeLoading` → `ModeApps`

- **`TestSSOReauthOnExpiredSSEStream`** — mock server serves apps via SSE normally, then sends a 401 auth error event mid-stream; mock `JWTAuthProvider` writes a fresh token; assert: view shows reauth message during `ModeReauthPending`, watch restarts, apps re-loaded correctly

---

## Files Changed

| File | Change |
|---|---|
| `pkg/auth/login.go` | New — `JWTAuthProvider`, `ArgocdCLIAuthProvider`, `LoginParams`, `StripProtocol` |
| `pkg/auth/login_test.go` | New — unit tests |
| `pkg/model/types.go` | Add `GrpcWeb bool` to `Server` struct; add `ModeReauthPending` |
| `pkg/model/messages.go` | Add `TriggerReauthMsg`, `ReauthCompleteMsg` |
| `pkg/config/cli_config.go` | Wire `GrpcWeb` into `Server` when reading config |
| `cmd/app/model.go` (or new `cmd/app/reauth.go`) | `TriggerReauthMsg` + `ReauthCompleteMsg` handlers; `reauthAttempts` counter |
| `cmd/app/model_init.go` | `validateAuthentication()` emits `TriggerReauthMsg` on empty token |
| `cmd/app/api_integration.go` | `AuthErrorMsg` handler routes to `TriggerReauthMsg` instead of `ModeAuthRequired` |
| `cmd/app/view.go` | `ModeReauthPending` view rendering |
| `cmd/app/main.go` | Inject `ArgocdCLIAuthProvider{}` into model |
| `e2e/sso_reauth_test.go` | New — E2E tests |

---

## Out of Scope

- Username/password login (deferred)
- Credential/token storage in OS keychain (deferred)
- Headless/SSH environments (deferred)
- ArgoCD Core mode re-auth
