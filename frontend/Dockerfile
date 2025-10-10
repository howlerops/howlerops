# Multi-stage build for React frontend with Nginx

# Stage 1: Build React application
FROM node:20-alpine AS builder

# Set working directory
WORKDIR /build

# Copy package files
COPY package*.json ./

# Install all dependencies (including dev for build)
RUN npm ci --silent

# Copy source code
COPY . .

# Build application with production optimizations
ENV NODE_ENV=production
ENV GENERATE_SOURCEMAP=false
ENV VITE_BUILD_ANALYZER=false
RUN npm run build

# Stage 2: Production server with Nginx
FROM nginx:1.25-alpine AS production

# Install security updates and add non-root user
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    ca-certificates \
    tzdata \
    wget \
    && addgroup -g 1001 -S nginx-user \
    && adduser -S -D -H -u 1001 -h /var/cache/nginx -s /sbin/nologin -G nginx-user -g nginx-user nginx-user

# Remove default nginx website
RUN rm -rf /usr/share/nginx/html/*

# Copy built application
COPY --from=builder /build/dist /usr/share/nginx/html

# Copy nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf
COPY nginx-default.conf /etc/nginx/conf.d/default.conf

# Create nginx directories and set permissions
RUN mkdir -p /var/cache/nginx/client_temp \
    /var/cache/nginx/proxy_temp \
    /var/cache/nginx/fastcgi_temp \
    /var/cache/nginx/uwsgi_temp \
    /var/cache/nginx/scgi_temp \
    /var/log/nginx \
    && chown -R nginx-user:nginx-user /var/cache/nginx \
    && chown -R nginx-user:nginx-user /var/log/nginx \
    && chown -R nginx-user:nginx-user /usr/share/nginx/html \
    && chmod -R 755 /usr/share/nginx/html

# Create pid directory
RUN mkdir -p /run/nginx && chown -R nginx-user:nginx-user /run/nginx

# Switch to non-root user
USER nginx-user

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]