###########
# Builder #
###########
# Build a static Linux binary with Go
FROM golang:1.23-alpine AS builder
WORKDIR /src

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    cd cmd/app && \
    CGO_ENABLED=0 GOOS=linux GOARCH=$(go env GOARCH) go build -ldflags "-s -w" -o /out/argonaut

#############
# Runtime   #
#############
# Use Alpine (musl) to match the compiled binary's dynamic linker
FROM alpine:3.20

ARG TARGETARCH

# Add required runtimes and tools for the compiled binary
# - ca-certificates: HTTPS requests
# - curl, git: fetch Argo CD CLI and provide git diff fallback
RUN apk add --no-cache ca-certificates curl git \
  && curl -sSL -o /usr/local/bin/argocd https://github.com/argoproj/argo-cd/releases/latest/download/argocd-linux-${TARGETARCH} \
  && chmod +x /usr/local/bin/argocd \
  && adduser -D -u 10001 appuser

# Copy the compiled binary from the builder stage
COPY --from=builder /out/argonaut /usr/local/bin/argonaut

# Run as non-root for safety
USER 10001:10001

# Ensure colors render nicely by default when attached to a TTY
ENV TERM=xterm-256color

# Default entrypoint (pass CLI flags at runtime)
ENTRYPOINT ["/usr/local/bin/argonaut"]
