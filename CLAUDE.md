# Architecture

Before modifying API call sites, timeout handling, or the HTTP client layer, read the relevant ADRs in `docs/architecture/decisions/`:

- **ADR-0002**: API timeout strategy — timeouts must be set at call sites using `appcontext.WithAPITimeout()` or `WithMinAPITimeout()`, never hardcoded with `context.WithTimeout()` and never inside `Client.Get/Post/Put/Delete`. See the ADR for the full rationale.

