# Multi-stage build for HowlerOps (Node.js Backend + React Frontend)

# Stage 1: Build React frontend
FROM node:20-alpine AS frontend-builder

# Set working directory
WORKDIR /build

# Copy package files
COPY frontend/package*.json ./

# Install dependencies with cache optimization
RUN npm ci --only=production --silent

# Copy frontend source
COPY frontend/ ./

# Build frontend with optimizations
ENV NODE_ENV=production
ENV VITE_BUILD_ANALYZER=false
RUN npm run build

# Stage 2: Build Node.js backend
FROM node:20-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache python3 make g++ sqlite-dev

# Set working directory
WORKDIR /build

# Copy package files
COPY backend/package*.json ./

# Install all dependencies (including dev for build)
RUN npm install --silent

# Copy backend source
COPY backend/ ./

# Build TypeScript to JavaScript
RUN npm run build

# Install only production dependencies
RUN npm install --only=production --silent && npm cache clean --force

# Stage 3: Final runtime image with PM2
FROM node:20-alpine AS runtime

# Install runtime dependencies and PM2
RUN apk add --no-cache \
    ca-certificates \
    sqlite-libs \
    tzdata \
    wget \
    && npm install -g pm2@latest \
    && addgroup -g 1000 sqlstudio \
    && adduser -D -u 1000 -G sqlstudio sqlstudio

# Create necessary directories
RUN mkdir -p /app/data /app/config /app/logs /app/static /app/uploads \
    && chown -R sqlstudio:sqlstudio /app

# Set working directory
WORKDIR /app

# Copy backend from builder (dist folder and node_modules)
COPY --from=backend-builder --chown=sqlstudio:sqlstudio /build/dist ./dist
COPY --from=backend-builder --chown=sqlstudio:sqlstudio /build/node_modules ./node_modules
COPY --from=backend-builder --chown=sqlstudio:sqlstudio /build/package*.json ./

# Copy frontend build from builder
COPY --from=frontend-builder --chown=sqlstudio:sqlstudio /build/dist ./static

# Copy PM2 ecosystem file
COPY --chown=sqlstudio:sqlstudio ecosystem.config.js ./

# Set user
USER sqlstudio

# Expose port
EXPOSE 3000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/api/health || exit 1

# Run with PM2
CMD ["pm2-runtime", "start", "ecosystem.config.js", "--env", "production"]