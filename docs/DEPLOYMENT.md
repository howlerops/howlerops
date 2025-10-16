# HowlerOps Deployment Guide

This guide provides comprehensive instructions for deploying HowlerOps in different environments.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Configuration](#environment-configuration)
- [Development Deployment](#development-deployment)
- [Production Deployment](#production-deployment)
- [Monitoring and Logging](#monitoring-and-logging)
- [Backup and Recovery](#backup-and-recovery)
- [Troubleshooting](#troubleshooting)
- [Security Considerations](#security-considerations)

## Prerequisites

### System Requirements

- **Docker Engine**: 20.10.0 or higher
- **Docker Compose**: 2.0.0 or higher
- **Node.js**: 20.x LTS (for local development)
- **Memory**: Minimum 4GB RAM, Recommended 8GB+ for production
- **Storage**: Minimum 20GB free space
- **OS**: Linux (Ubuntu 20.04+), macOS, or Windows with WSL2

### Required Ports

- **8080**: Frontend (Nginx)
- **3000**: Backend API
- **5432**: PostgreSQL database
- **6379**: Redis cache
- **9090**: Prometheus metrics (optional)
- **3001**: Grafana dashboard (optional)

## Environment Configuration

### 1. Environment Files

Copy the appropriate environment template:

```bash
# For development
cp .env.development .env

# For production
cp .env.production .env
```

### 2. Required Environment Variables

Update the following critical variables in your `.env` file:

```bash
# Database (REQUIRED)
POSTGRES_PASSWORD=your-strong-database-password
POSTGRES_USER=sqlstudio
POSTGRES_DB=sqlstudio

# Security (REQUIRED)
JWT_SECRET=your-very-secure-jwt-secret-min-32-chars
SESSION_SECRET=your-secure-session-secret

# Redis (REQUIRED)
REDIS_PASSWORD=your-redis-password

# Application
CORS_ORIGIN=https://yourdomain.com  # Update for production
```

## Development Deployment

### Quick Start

1. **Automated Setup** (Recommended):
   ```bash
   chmod +x scripts/dev-setup.sh
   ./scripts/dev-setup.sh
   ```

2. **Manual Setup**:
   ```bash
   # Install dependencies
   cd backend && npm install && cd ..
   cd frontend && npm install && cd ..

   # Start development environment
   docker-compose -f docker-compose.dev.yml up -d

   # Access the application
   # Frontend: http://localhost:5173
   # Backend: http://localhost:3000
   ```

### Development Commands

```bash
# Start development environment
./start-dev.sh

# Stop development environment
./stop-dev.sh

# View logs
./logs-dev.sh

# Reset environment (removes all data)
./reset-dev.sh
```

### Development Database Access

- **PgAdmin**: http://localhost:5050
  - Email: `admin@sqlstudio.dev`
  - Password: `admin`

- **Redis Commander**: http://localhost:8081

## Production Deployment

### 1. Pre-deployment Checklist

- [ ] Update all passwords and secrets in `.env.production`
- [ ] Configure SSL certificates
- [ ] Set up domain DNS records
- [ ] Configure firewall rules
- [ ] Set up monitoring alerts
- [ ] Create backup strategy
- [ ] Test rollback procedures

### 2. Deployment Methods

#### Option A: Automated Deployment Script

```bash
# Set environment
export DEPLOY_ENV=production

# Run deployment
chmod +x scripts/deploy.sh
./scripts/deploy.sh deploy
```

#### Option B: Manual Deployment

```bash
# Build and start services
docker-compose -f docker-compose.yml up -d --build

# Check service health
docker-compose ps
docker-compose logs -f
```

### 3. Zero-Downtime Deployment

The deployment script includes zero-downtime deployment features:

- **Health checks**: Services must pass health checks before traffic is routed
- **Rolling updates**: New containers are started before old ones are stopped
- **Automatic rollback**: Failed deployments automatically rollback
- **Database backups**: Automatic backup before deployment

```bash
# Deploy with monitoring
./scripts/deploy.sh deploy

# Manual rollback if needed
./scripts/deploy.sh rollback
```

### 4. Load Balancer Configuration

For production environments, place a load balancer (nginx, HAProxy, or cloud LB) in front of the application:

```nginx
upstream sql_studio_backend {
    server app1.example.com:3000;
    server app2.example.com:3000;
}

upstream sql_studio_frontend {
    server app1.example.com:8080;
    server app2.example.com:8080;
}

server {
    listen 443 ssl;
    server_name sqlstudio.example.com;

    ssl_certificate /path/to/ssl/cert.pem;
    ssl_certificate_key /path/to/ssl/key.pem;

    location / {
        proxy_pass http://sql_studio_frontend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/ {
        proxy_pass http://sql_studio_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Monitoring and Logging

### 1. Enable Monitoring Stack

```bash
# Start with monitoring enabled
docker-compose --profile monitoring up -d

# Access monitoring tools
# Grafana: http://localhost:3001 (admin/admin)
# Prometheus: http://localhost:9090
```

### 2. Log Management

Logs are stored in Docker volumes and can be accessed via:

```bash
# View application logs
docker-compose logs -f sql-studio-backend
docker-compose logs -f sql-studio-frontend

# Access log files directly
docker exec -it sql-studio-backend tail -f /app/logs/combined.log
```

### 3. Monitoring Endpoints

- **Application Health**: `GET /api/health`
- **Metrics**: `GET /api/metrics` (Prometheus format)
- **Database Health**: Included in application health check

## Backup and Recovery

### 1. Database Backups

**Automated Backups** (via deployment script):
```bash
./scripts/deploy.sh backup
```

**Manual Backup**:
```bash
# Create backup
docker-compose exec postgres pg_dump -U sqlstudio sqlstudio > backup_$(date +%Y%m%d_%H%M%S).sql

# Restore backup
docker-compose exec -T postgres psql -U sqlstudio sqlstudio < backup_file.sql
```

### 2. File Uploads Backup

```bash
# Backup uploaded files
docker run --rm -v sql-studio_sql-studio-uploads:/data -v $(pwd):/backup alpine tar czf /backup/uploads_backup.tar.gz /data

# Restore uploaded files
docker run --rm -v sql-studio_sql-studio-uploads:/data -v $(pwd):/backup alpine tar xzf /backup/uploads_backup.tar.gz -C /
```

### 3. Complete System Backup

```bash
# Stop services
docker-compose down

# Backup all volumes
docker run --rm -v sql-studio_postgres-data:/postgres -v sql-studio_redis-data:/redis -v sql-studio_sql-studio-uploads:/uploads -v $(pwd):/backup alpine tar czf /backup/complete_backup.tar.gz /postgres /redis /uploads

# Restore (after recreating volumes)
docker run --rm -v sql-studio_postgres-data:/postgres -v sql-studio_redis-data:/redis -v sql-studio_sql-studio-uploads:/uploads -v $(pwd):/backup alpine tar xzf /backup/complete_backup.tar.gz -C /
```

## Troubleshooting

### Common Issues

#### 1. Services Won't Start

```bash
# Check service status
docker-compose ps

# Check logs for errors
docker-compose logs

# Check disk space
df -h

# Check memory usage
free -m
```

#### 2. Database Connection Issues

```bash
# Check database container
docker-compose exec postgres pg_isready -U sqlstudio

# Check database logs
docker-compose logs postgres

# Verify environment variables
docker-compose config
```

#### 3. Frontend Build Issues

```bash
# Rebuild frontend
docker-compose build --no-cache sql-studio-frontend

# Check frontend logs
docker-compose logs sql-studio-frontend
```

#### 4. Performance Issues

```bash
# Check resource usage
docker stats

# Check database performance
docker-compose exec postgres psql -U sqlstudio -c "SELECT * FROM pg_stat_activity;"

# Enable slow query logging
# Add to postgres environment: POSTGRES_INITDB_ARGS="-c log_statement=all"
```

### Health Check Endpoints

- **Frontend**: `GET http://localhost:8080/health`
- **Backend**: `GET http://localhost:3000/api/health`
- **Database**: Via backend health check

## Security Considerations

### 1. Environment Security

- [ ] Change all default passwords
- [ ] Use strong, unique secrets for JWT and sessions
- [ ] Enable SSL/TLS in production
- [ ] Restrict database access to application only
- [ ] Use non-root users in containers
- [ ] Enable firewall rules

### 2. Database Security

```bash
# Create read-only user for monitoring
docker-compose exec postgres psql -U sqlstudio -c "
CREATE USER monitoring WITH PASSWORD 'monitoring_password';
GRANT CONNECT ON DATABASE sqlstudio TO monitoring;
GRANT USAGE ON SCHEMA public TO monitoring;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO monitoring;
"
```

### 3. Container Security

- Regular security updates via base image updates
- Vulnerability scanning enabled in CI/CD
- Non-privileged container execution
- Network isolation between services

### 4. Backup Security

- Encrypt backups at rest
- Secure backup storage location
- Regular backup restoration tests
- Access logging for backup operations

## Scaling Considerations

### Horizontal Scaling

1. **Backend Scaling**:
   ```bash
   # Scale backend instances
   docker-compose up -d --scale sql-studio-backend=3
   ```

2. **Database Scaling**:
   - Use read replicas for read-heavy workloads
   - Consider PostgreSQL clustering solutions
   - Implement connection pooling

3. **Frontend Scaling**:
   - Use CDN for static assets
   - Multiple frontend instances behind load balancer
   - Enable caching headers

### Performance Optimization

1. **Database Optimization**:
   - Regular VACUUM and ANALYZE
   - Appropriate indexing
   - Connection pooling
   - Query optimization

2. **Application Optimization**:
   - Enable Redis caching
   - Optimize API response sizes
   - Implement pagination
   - Use compression

3. **Infrastructure Optimization**:
   - Use SSD storage
   - Adequate RAM allocation
   - Network optimization
   - Monitoring and alerting

## Support

For additional support:

1. Check the [GitHub Issues](https://github.com/your-org/sql-studio/issues)
2. Review application logs
3. Check the monitoring dashboards
4. Consult the API documentation

Remember to always test deployments in a staging environment before deploying to production.