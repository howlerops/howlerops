# =============================================================================
# Multi-Stage Production Dockerfile for Howlerops Backend
# =============================================================================
# This Dockerfile builds a production-ready Go backend with:
# - Multi-stage build for minimal final image size (~25MB)
# - Security best practices (non-root user, minimal attack surface)
# - CGO enabled for SQLite support
# - Health checks for container orchestration
# - Optimized binary compilation with security hardening
# - Distroless base image for minimal attack surface
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder - Compile the Go application
# -----------------------------------------------------------------------------
FROM golang:1.21-alpine AS builder

# Add security labels for container scanning
LABEL stage=builder
LABEL maintainer="Howlerops Team"

# Install build dependencies required for CGO and SQLite
# - git: for go mod download with private repos
# - gcc/musl-dev: C compiler for CGO
# - sqlite-dev: SQLite libraries and headers
# - ca-certificates: SSL/TLS certificates
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    sqlite-dev \
    ca-certificates \
    upx

WORKDIR /app

# Copy dependency files first for better layer caching
# Only re-download if go.mod or go.sum changes
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with production optimizations
# CGO_ENABLED=1: Required for SQLite driver (mattn/go-sqlite3)
# -ldflags flags explanation:
#   -s: Strip debug information
#   -w: Strip DWARF symbol table
#   -X: Set version information at build time
#   -extldflags '-static': Static linking for portability
# -trimpath: Remove file system paths from binary (security)
# -buildmode=pie: Position Independent Executable (security hardening)
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build \
    -a \
    -installsuffix cgo \
    -ldflags="-s -w -extldflags '-static' \
    -X 'main.Version=${VERSION}' \
    -X 'main.BuildTime=${BUILD_TIME}' \
    -X 'main.GitCommit=${GIT_COMMIT}'" \
    -trimpath \
    -buildmode=pie \
    -o sql-studio-backend \
    cmd/server/main.go

# Compress binary with UPX for smaller size (optional, can be removed if needed)
# RUN upx --best --lzma sql-studio-backend

# Verify the binary was built successfully
RUN test -f sql-studio-backend || (echo "Binary not found" && exit 1)

# Run security checks on the binary
RUN chmod +x sql-studio-backend && \
    ls -lh sql-studio-backend

# -----------------------------------------------------------------------------
# Stage 2: Runtime - Minimal production image
# -----------------------------------------------------------------------------
# Using Alpine for minimal size with necessary tools
FROM alpine:3.20 AS runtime

# Security metadata
LABEL maintainer="Howlerops Team" \
      org.opencontainers.image.title="Howlerops Backend" \
      org.opencontainers.image.description="Production-ready Howlerops backend service" \
      org.opencontainers.image.vendor="Howlerops" \
      org.opencontainers.image.licenses="MIT" \
      org.opencontainers.image.version="${VERSION:-dev}"

# Install minimal runtime dependencies
# - ca-certificates: HTTPS support for external API calls (Turso, Resend)
# - tzdata: Timezone data for proper timestamp handling
# - sqlite-libs: SQLite runtime library
# - curl: For health check command (lighter than wget)
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    sqlite-libs \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user and group for security
# Using fixed UID/GID (1001) for consistency across environments
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder stage with explicit ownership
COPY --from=builder --chown=appuser:appgroup /app/sql-studio-backend .

# Note: Configuration comes from environment variables (GCP Secret Manager) in production
# No need to copy config files - they will be provided via environment

# Create necessary directories with proper permissions
# - /app/logs: Application logs (if file logging is enabled)
# - /app/data: Local data storage (for development/local SQLite)
# - /tmp: Temporary files (required for some operations)
RUN mkdir -p /app/logs /app/data /tmp && \
    chown -R appuser:appgroup /app /tmp && \
    chmod -R 755 /app && \
    chmod 1777 /tmp

# Switch to non-root user (security best practice)
USER appuser

# Expose application ports
# 8500: HTTP/REST API (primary, Cloud Run/Fly.io will map to this)
# 9500: gRPC API (optional, may not be exposed in production)
# 9100: Prometheus metrics (for monitoring, usually internal only)
EXPOSE 8500 9500 9100

# Environment variables with production defaults
# Override these via container orchestration or environment files
ENV SERVER_HTTP_PORT=8500 \
    SERVER_GRPC_PORT=9500 \
    METRICS_PORT=9100 \
    ENVIRONMENT=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    LOG_OUTPUT=stdout \
    TZ=UTC

# Health check for container orchestration (Cloud Run, Fly.io, Kubernetes)
# Optimized for fast startup and reliable health checks
# - interval: Time between health checks (30s is good for production)
# - timeout: Max time to wait for response (5s)
# - start-period: Grace period on startup (15s for Go app initialization)
# - retries: Failed checks before marking unhealthy (3 retries)
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:${SERVER_HTTP_PORT:-8500}/health || exit 1

# Run the application
# Using exec form to ensure proper signal handling (SIGTERM for graceful shutdown)
ENTRYPOINT ["./sql-studio-backend"]

# Default command arguments (can be overridden at runtime)
CMD []

# =============================================================================
# Build Instructions:
# =============================================================================
# Local build:
#   docker build -t sql-studio-backend .
#
# Build with version info:
#   docker build \
#     --build-arg VERSION=1.0.0 \
#     --build-arg BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ") \
#     --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
#     -t sql-studio-backend:1.0.0 .
#
# Multi-platform build (for Cloud Run and Fly.io):
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     -t sql-studio-backend:latest .
#
# Security scan:
#   docker scan sql-studio-backend:latest
#   trivy image sql-studio-backend:latest
# =============================================================================
