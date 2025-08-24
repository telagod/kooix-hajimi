# Multi-stage build for Kooix Hajimi with proper cross-platform support
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    gcc \
    musl-dev \
    sqlite-dev

WORKDIR /app

# Copy go module files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with simplified CGO settings for better compatibility
# Use CGO_ENABLED=1 but with conservative settings to avoid cross-compilation issues
ENV CGO_ENABLED=1

# Build applications with SQLite optimizations
RUN go build \
    -tags "sqlite_omit_load_extension" \
    -ldflags "-s -w" \
    -o kooix-hajimi-server \
    cmd/server/main.go

RUN go build \
    -tags "sqlite_omit_load_extension" \
    -ldflags "-s -w" \
    -o kooix-hajimi-cli \
    cmd/cli/main.go

# Final stage - minimal runtime image
FROM debian:bullseye-slim

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata && \
    addgroup -g 1001 kooix && \
    adduser -D -s /bin/sh -u 1001 -G kooix kooix

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
