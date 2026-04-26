# Architecture Decision Records (ADRs)

This directory contains Architecture Decision Records for Argonaut, documenting important architectural and technical decisions made during development.

## Format

ADRs follow a consistent format:

- **Status**: Proposed | Accepted | Rejected | Deprecated | Superseded
- **Context**: The situation and requirements that led to this decision
- **Decision**: What was decided and why
- **Consequences**: Positive and negative outcomes of this decision

## Naming Convention

ADRs are numbered sequentially with descriptive names:
- `0001-custom-sse-reader-implementation.md`
- `0002-example-next-decision.md`

## Index

| ADR | Title | Status |
|-----|-------|--------|
| [0001](./0001-custom-sse-reader-implementation.md) | Custom SSE Reader Implementation | Accepted |
| [0002](./0002-api-timeout-strategy.md) | API Timeout Strategy | Accepted |
| [0003](./0003-async-message-gating.md) | Async Message Gating (Epoch + Target) | Accepted |
| [0004](./0004-app-identity.md) | App Identity Is `(Name, AppNamespace)` | Accepted |

## Guidelines

- ADRs are immutable once accepted - create new ADRs to modify decisions
- Link related ADRs when one supersedes another
- Include relevant code examples and references
- Document both technical and business rationale