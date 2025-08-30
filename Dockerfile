###########
# Builder #
###########
# Build a static Linux binary with Bun in an Alpine (musl) environment
FROM oven/bun:alpine AS builder
WORKDIR /app

# Install dependencies first for better layer caching
COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile

# Copy the rest of the sources
COPY . .

# Compile to a single binary
# Output directly to /out to avoid copying node_modules into the final image
RUN mkdir -p /out \
 && bun build --compile --minify --sourcemap src/main.tsx --outfile /out/argonaut

#############
# Runtime   #
#############
# Use Alpine (musl) to match the compiled binary's dynamic linker
FROM alpine:3.20

# Add required runtimes for the compiled binary
# - ca-certificates: HTTPS requests
# - libstdc++, libgcc: C++ runtime and unwinder needed by the bun-compiled binary
RUN apk add --no-cache ca-certificates libstdc++ libgcc \
  && adduser -D -u 10001 appuser

# Copy the compiled binary from the builder stage
COPY --from=builder /out/argonaut /usr/local/bin/argonaut

# Run as non-root for safety
USER 10001:10001

# Ensure colors render nicely by default when attached to a TTY
ENV TERM=xterm-256color

# Default entrypoint (pass CLI flags at runtime)
ENTRYPOINT ["/usr/local/bin/argonaut"]
