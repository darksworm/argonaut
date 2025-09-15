.PHONY: test unit e2e goldens

# Default parallelism for tests
PARALLEL ?= 8

# Run all tests, including e2e (unix-only), with parallelism and no cache.
test:
	go test -tags e2e ./... -v -count=1 -parallel $(PARALLEL)

# Run only unit tests (no e2e).
unit:
	go test ./... -v -count=1 -parallel $(PARALLEL)

# Run only e2e tests.
e2e:
	go test -tags e2e ./e2e -v -count=1 -parallel $(PARALLEL)

# Regenerate golden snapshots for app tests.
goldens:
	UPDATE_GOLDEN=1 go test ./cmd/app -run TestGolden_ -v

