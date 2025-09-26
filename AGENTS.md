# Agent Guidelines for Argonaut

## Build/Test Commands
- **Run locally**: `go run ./cmd/app`
- **Build**: `go build -o ./bin/argonaut ./cmd/app`
- **Unit tests**: `make unit` or `go test ./... -v`
- **E2E tests**: `make e2e` (unix only)
- **All tests**: `make test` or `go test -tags e2e ./... -v`
- **Single test**: `go test -run TestName ./pkg/package`
- **Golden snapshots**: `make goldens` or `UPDATE_GOLDEN=1 go test ./cmd/app -run TestGolden_ -v`
- **Format**: `gofmt -s -w .`
- **Lint**: `go vet ./...`

## Code Style Guidelines
- **Packages**: Short, lowercase names
- **Naming**: Exported=PascalCase, unexported=camelCase
- **Files**: lowercase, tests=`*_test.go`
- **Imports**: stdlib → third-party → internal (grouped)
- **Functions**: Clear, small; avoid stutter; return `error` not booleans
- **Errors**: Use structured errors with context; wrap with `fmt.Errorf("...%w", err)`
- **Types**: Prefer interfaces; use context for timeouts/cancellation
- **Comments**: None unless complex logic; prefer self-documenting code

