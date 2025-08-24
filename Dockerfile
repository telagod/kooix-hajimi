# Modern multi-platform build using tonistiigi/xx for Go SQLite3 cross-compilation
FROM --platform=$BUILDPLATFORM tonistiigi/xx:1.4.0 AS xx

# Build stage with proper cross-compilation support
FROM --platform=$BUILDPLATFORM golang:1.21-bullseye AS builder

# Copy cross-compilation helpers
COPY --from=xx / /

WORKDIR /app

# Install build dependencies and prepare for cross-compilation
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        git \
        ca-certificates \
        clang \
        lld && \
    rm -rf /var/lib/apt/lists/*

# Declare build arguments
ARG TARGETPLATFORM

# Install target-platform specific dependencies
RUN xx-apt-get update && \
    xx-apt-get install -y --no-install-recommends \
        gcc \
        libc6-dev \
        libsqlite3-dev

# Copy go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Pre-install go-sqlite3 to reduce build time
RUN CGO_ENABLED=1 go install github.com/mattn/go-sqlite3

# Copy source code
COPY . .

# Enable CGO for SQLite3 and build with cross-compilation support
ENV CGO_ENABLED=1

# Build applications using xx-go wrapper
RUN xx-go build \
    -tags "sqlite_omit_load_extension" \
    -ldflags "-s -w" \
    -o kooix-hajimi-server \
    cmd/server/main.go && \
    xx-verify kooix-hajimi-server

RUN xx-go build \
    -tags "sqlite_omit_load_extension" \
    -ldflags "-s -w" \
    -o kooix-hajimi-cli \
    cmd/cli/main.go && \
    xx-verify kooix-hajimi-cli

# Final stage - minimal runtime image
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates tzdata wget && \
    groupadd -g 1001 kooix && \
    useradd -u 1001 -g kooix -s /bin/sh -m kooix && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /app/kooix-hajimi-server ./
COPY --from=builder /app/kooix-hajimi-cli ./

# Copy application resources
COPY --from=builder /app/configs ./configs/
COPY --from=builder /app/web ./web/

# Create data directory and set permissions
RUN mkdir -p /app/data && \
    chown -R kooix:kooix /app

# Switch to non-root user
USER kooix

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/api/status || exit 1

# Expose port
EXPOSE 8080

# Default command
CMD ["./kooix-hajimi-server"]