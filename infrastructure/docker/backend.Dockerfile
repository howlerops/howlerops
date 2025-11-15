# =============================================================================
# Howlerops Backend - Production Dockerfile
# =============================================================================
# Optimized multi-stage production build with security best practices
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder - Compile the Go application
# -----------------------------------------------------------------------------
FROM golang:1.24-alpine AS builder

LABEL stage=builder \
      maintainer="Howlerops Team"

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    sqlite-dev \
    ca-certificates \
    upx

WORKDIR /app

# Copy dependency files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build arguments
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT

# Build with production optimizations
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

# Verify binary
RUN test -f sql-studio-backend && \
    chmod +x sql-studio-backend && \
    ls -lh sql-studio-backend

# -----------------------------------------------------------------------------
# Stage 2: Runtime - Minimal production image
# -----------------------------------------------------------------------------
FROM alpine:3.20 AS runtime

# Metadata
LABEL maintainer="Howlerops Team" \
      org.opencontainers.image.title="Howlerops Backend" \
      org.opencontainers.image.description="Production Howlerops backend service" \
      org.opencontainers.image.vendor="Howlerops" \
      org.opencontainers.image.licenses="MIT"

# Install runtime dependencies
RUN apk --no-cache add \
    ca-certificates \
    tzdata \
    sqlite-libs \
    curl \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary
COPY --from=builder --chown=appuser:appgroup /app/sql-studio-backend .
COPY --from=builder --chown=appuser:appgroup /app/configs ./configs 2>/dev/null || true

# Create necessary directories
RUN mkdir -p /app/logs /app/data /tmp && \
    chown -R appuser:appgroup /app /tmp && \
    chmod -R 755 /app && \
    chmod 1777 /tmp

# Switch to non-root user
USER appuser

# Expose ports
EXPOSE 8500 9500 9100

# Environment variables
ENV SERVER_HTTP_PORT=8500 \
    SERVER_GRPC_PORT=9500 \
    METRICS_PORT=9100 \
    ENVIRONMENT=production \
    LOG_LEVEL=info \
    LOG_FORMAT=json \
    LOG_OUTPUT=stdout \
    TZ=UTC

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
    CMD curl -f http://localhost:${SERVER_HTTP_PORT:-8500}/health || exit 1

# Run application
ENTRYPOINT ["./sql-studio-backend"]
CMD []
