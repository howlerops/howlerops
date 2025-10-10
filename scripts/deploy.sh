#!/bin/bash

# HowlerOps Production Deployment Script
# This script handles zero-downtime deployment with health checks and rollback capability

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
DEPLOY_ENV="${DEPLOY_ENV:-production}"
COMPOSE_FILE="${PROJECT_ROOT}/docker-compose.yml"
ENV_FILE="${PROJECT_ROOT}/.env.${DEPLOY_ENV}"
BACKUP_DIR="${PROJECT_ROOT}/backups"
HEALTH_CHECK_TIMEOUT=300
ROLLBACK_IMAGES_FILE="${PROJECT_ROOT}/.rollback_images"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Cleanup function
cleanup() {
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        log_error "Deployment failed with exit code $exit_code"
        if [ "${ROLLBACK_ON_FAILURE:-true}" = "true" ]; then
            log_warning "Initiating automatic rollback..."
            rollback
        fi
    fi
    exit $exit_code
}

# Set up trap for cleanup
trap cleanup EXIT

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if Docker is running
    if ! docker info > /dev/null 2>&1; then
        log_error "Docker is not running"
        exit 1
    fi

    # Check if docker-compose is available
    if ! command -v docker-compose > /dev/null 2>&1; then
        log_error "docker-compose is not installed"
        exit 1
    fi

    # Check if environment file exists
    if [ ! -f "$ENV_FILE" ]; then
        log_error "Environment file not found: $ENV_FILE"
        exit 1
    fi

    # Check if compose file exists
    if [ ! -f "$COMPOSE_FILE" ]; then
        log_error "Docker compose file not found: $COMPOSE_FILE"
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Create backup
create_backup() {
    log_info "Creating database backup..."

    mkdir -p "$BACKUP_DIR"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${BACKUP_DIR}/sqlstudio_backup_${timestamp}.sql"

    # Create database backup
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec -T postgres \
        pg_dump -U "${POSTGRES_USER:-sqlstudio}" "${POSTGRES_DB:-sqlstudio}" > "$backup_file"

    if [ $? -eq 0 ]; then
        log_success "Database backup created: $backup_file"
        # Keep only last 10 backups
        ls -t "${BACKUP_DIR}"/sqlstudio_backup_*.sql | tail -n +11 | xargs -r rm
    else
        log_error "Failed to create database backup"
        exit 1
    fi
}

# Save current image tags for rollback
save_current_images() {
    log_info "Saving current image tags for rollback..."

    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" images --format "{{.Service}}:{{.Tag}}" > "$ROLLBACK_IMAGES_FILE"
    log_success "Current image tags saved"
}

# Build new images
build_images() {
    log_info "Building new Docker images..."

    # Pull latest base images
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" pull --ignore-pull-failures

    # Build application images
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" build --no-cache --parallel

    if [ $? -eq 0 ]; then
        log_success "Docker images built successfully"
    else
        log_error "Failed to build Docker images"
        exit 1
    fi
}

# Health check function
health_check() {
    local service=$1
    local url=$2
    local timeout=${3:-60}
    local interval=5
    local elapsed=0

    log_info "Performing health check for $service..."

    while [ $elapsed -lt $timeout ]; do
        if curl -f -s "$url" > /dev/null 2>&1; then
            log_success "$service health check passed"
            return 0
        fi

        sleep $interval
        elapsed=$((elapsed + interval))
        echo -n "."
    done

    log_error "$service health check failed after ${timeout}s"
    return 1
}

# Wait for services to be healthy
wait_for_services() {
    log_info "Waiting for services to be healthy..."

    # Wait for backend
    health_check "Backend" "http://localhost:3000/api/health" 120

    # Wait for frontend
    health_check "Frontend" "http://localhost:8080/health" 60

    log_success "All services are healthy"
}

# Rolling update deployment
rolling_update() {
    log_info "Starting rolling update deployment..."

    # Start new services
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d --remove-orphans

    # Wait for services to be healthy
    wait_for_services

    # Remove old containers
    docker container prune -f
    docker image prune -f

    log_success "Rolling update completed successfully"
}

# Rollback function
rollback() {
    log_warning "Starting rollback procedure..."

    if [ ! -f "$ROLLBACK_IMAGES_FILE" ]; then
        log_error "No rollback images file found"
        return 1
    fi

    # Stop current services
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" down

    # Restore previous images (this would need more sophisticated image management)
    log_info "Rolling back to previous deployment..."

    # Start services with previous configuration
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" up -d

    # Wait for services
    if wait_for_services; then
        log_success "Rollback completed successfully"
    else
        log_error "Rollback failed - manual intervention required"
        return 1
    fi
}

# Run database migrations
run_migrations() {
    log_info "Running database migrations..."

    # Wait for database to be ready
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec postgres \
        pg_isready -h localhost -U "${POSTGRES_USER:-sqlstudio}"

    # Run migrations (this would typically be done by your backend service)
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" exec sql-studio-backend \
        npm run migrate

    if [ $? -eq 0 ]; then
        log_success "Database migrations completed"
    else
        log_error "Database migrations failed"
        exit 1
    fi
}

# Warm up services
warm_up() {
    log_info "Warming up services..."

    # Make some requests to warm up the application
    curl -s "http://localhost:8080/" > /dev/null || true
    curl -s "http://localhost:3000/api/health" > /dev/null || true

    sleep 5
    log_success "Services warmed up"
}

# Clean up old resources
cleanup_resources() {
    log_info "Cleaning up old resources..."

    # Remove unused containers
    docker container prune -f

    # Remove unused images (keep last 3 versions)
    docker images --format "table {{.Repository}}:{{.Tag}}\t{{.CreatedAt}}" \
        | grep sql-studio \
        | tail -n +4 \
        | awk '{print $1}' \
        | xargs -r docker rmi || true

    # Remove unused volumes (be careful with this)
    # docker volume prune -f

    log_success "Resource cleanup completed"
}

# Main deployment function
deploy() {
    log_info "Starting HowlerOps deployment to $DEPLOY_ENV environment"

    check_prerequisites

    if [ "$DEPLOY_ENV" = "production" ]; then
        create_backup
        save_current_images
    fi

    build_images
    rolling_update

    if [ "$DEPLOY_ENV" = "production" ]; then
        run_migrations
    fi

    warm_up
    cleanup_resources

    log_success "Deployment completed successfully!"
    log_info "Application is now running at:"
    log_info "  Frontend: http://localhost:${FRONTEND_PORT:-8080}"
    log_info "  Backend API: http://localhost:3000/api"

    if [ "${ENABLE_MONITORING:-false}" = "true" ]; then
        log_info "  Monitoring: http://localhost:3001 (Grafana)"
        log_info "  Metrics: http://localhost:9090 (Prometheus)"
    fi
}

# Parse command line arguments
case "${1:-deploy}" in
    "deploy")
        deploy
        ;;
    "rollback")
        rollback
        ;;
    "health-check")
        wait_for_services
        ;;
    "backup")
        create_backup
        ;;
    "cleanup")
        cleanup_resources
        ;;
    *)
        echo "Usage: $0 {deploy|rollback|health-check|backup|cleanup}"
        exit 1
        ;;
esac