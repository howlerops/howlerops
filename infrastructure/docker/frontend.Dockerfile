# =============================================================================
# SQL Studio Frontend - Production Dockerfile
# =============================================================================
# Multi-stage build with nginx serving optimized static assets
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Builder - Build React application
# -----------------------------------------------------------------------------
FROM node:20-alpine AS builder

LABEL stage=builder

WORKDIR /build

# Copy package files
COPY package*.json ./

# Install dependencies with cache optimization
RUN npm ci --silent --no-audit --no-fund

# Copy source code
COPY . .

# Build production bundle
ENV NODE_ENV=production
ENV GENERATE_SOURCEMAP=false
ENV VITE_BUILD_ANALYZER=false

RUN npm run build && \
    ls -lah dist/

# -----------------------------------------------------------------------------
# Stage 2: Production - nginx serving static files
# -----------------------------------------------------------------------------
FROM nginx:1.25-alpine AS production

# Metadata
LABEL maintainer="SQL Studio Team" \
      org.opencontainers.image.title="SQL Studio Frontend" \
      org.opencontainers.image.description="Production SQL Studio frontend" \
      org.opencontainers.image.vendor="SQL Studio"

# Install runtime dependencies and create non-root user
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && rm -rf /var/cache/apk/*

# Remove default nginx content
RUN rm -rf /usr/share/nginx/html/*

# Copy built application
COPY --from=builder /build/dist /usr/share/nginx/html

# Copy nginx configurations
COPY --chown=nginx:nginx infrastructure/docker/nginx/nginx.conf /etc/nginx/nginx.conf
COPY --chown=nginx:nginx infrastructure/docker/nginx/default.conf /etc/nginx/conf.d/default.conf

# Create health check endpoint
RUN echo "healthy" > /usr/share/nginx/html/health

# Create necessary directories with proper permissions
RUN mkdir -p /var/cache/nginx/client_temp \
    /var/cache/nginx/proxy_temp \
    /var/cache/nginx/fastcgi_temp \
    /var/cache/nginx/uwsgi_temp \
    /var/cache/nginx/scgi_temp \
    /var/log/nginx \
    /run/nginx \
    && chown -R nginx:nginx /var/cache/nginx \
    /var/log/nginx \
    /run/nginx \
    /usr/share/nginx/html \
    && chmod -R 755 /usr/share/nginx/html

# Switch to non-root user
USER nginx

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
