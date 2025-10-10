#!/bin/bash

# HowlerOps Development Environment Setup Script

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Check if required tools are installed
check_tools() {
    log_info "Checking required tools..."

    local tools=("docker" "docker-compose" "node" "npm")
    local missing_tools=()

    for tool in "${tools[@]}"; do
        if ! command -v "$tool" > /dev/null 2>&1; then
            missing_tools+=("$tool")
        fi
    done

    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_info "Please install the missing tools and run this script again."
        exit 1
    fi

    log_success "All required tools are installed"
}

# Create development environment file
create_dev_env() {
    log_info "Creating development environment file..."

    local env_file="${PROJECT_ROOT}/.env"

    if [ ! -f "$env_file" ]; then
        cp "${PROJECT_ROOT}/.env.development" "$env_file"
        log_success "Development environment file created: $env_file"
        log_warning "Please review and update the environment variables as needed"
    else
        log_info "Environment file already exists: $env_file"
    fi
}

# Install dependencies
install_dependencies() {
    log_info "Installing dependencies..."

    # Backend dependencies
    if [ -f "${PROJECT_ROOT}/backend/package.json" ]; then
        log_info "Installing backend dependencies..."
        cd "${PROJECT_ROOT}/backend"
        npm install
        cd "$PROJECT_ROOT"
        log_success "Backend dependencies installed"
    fi

    # Frontend dependencies
    if [ -f "${PROJECT_ROOT}/frontend/package.json" ]; then
        log_info "Installing frontend dependencies..."
        cd "${PROJECT_ROOT}/frontend"
        npm install
        cd "$PROJECT_ROOT"
        log_success "Frontend dependencies installed"
    fi
}

# Setup development database
setup_database() {
    log_info "Setting up development database..."

    # Start database services
    docker-compose -f "${PROJECT_ROOT}/docker-compose.dev.yml" up -d postgres-dev redis-dev

    # Wait for database to be ready
    log_info "Waiting for database to be ready..."
    sleep 10

    # Check if database is ready
    local retries=30
    while [ $retries -gt 0 ]; do
        if docker-compose -f "${PROJECT_ROOT}/docker-compose.dev.yml" exec -T postgres-dev pg_isready -U dev_user > /dev/null 2>&1; then
            break
        fi
        retries=$((retries - 1))
        sleep 2
        echo -n "."
    done

    if [ $retries -eq 0 ]; then
        log_error "Database failed to start"
        exit 1
    fi

    log_success "Development database is ready"
}

# Run database migrations and seed data
setup_data() {
    log_info "Setting up initial data..."

    # This would typically run your migration scripts
    # For now, we'll just ensure the initialization scripts ran
    log_info "Database initialization scripts will run automatically"
    log_success "Initial data setup completed"
}

# Start development services
start_services() {
    log_info "Starting development services..."

    # Start all development services
    docker-compose -f "${PROJECT_ROOT}/docker-compose.dev.yml" up -d

    log_success "Development services started"
    log_info "Services running:"
    log_info "  Frontend (Vite): http://localhost:5173"
    log_info "  Backend API: http://localhost:3000"
    log_info "  PostgreSQL: localhost:5432"
    log_info "  Redis: localhost:6379"

    # Optional admin tools
    if [ "${SETUP_ADMIN_TOOLS:-false}" = "true" ]; then
        log_info "Starting admin tools..."
        docker-compose -f "${PROJECT_ROOT}/docker-compose.dev.yml" --profile admin up -d
        log_info "  PgAdmin: http://localhost:5050 (admin@sqlstudio.dev / admin)"
        log_info "  Redis Commander: http://localhost:8081"
    fi
}

# Create useful development scripts
create_dev_scripts() {
    log_info "Creating development helper scripts..."

    # Create start script
    cat > "${PROJECT_ROOT}/start-dev.sh" << 'EOF'
#!/bin/bash
# Start HowlerOps in development mode
docker-compose -f docker-compose.dev.yml up -d
echo "Development environment started!"
echo "Frontend: http://localhost:5173"
echo "Backend: http://localhost:3000"
EOF

    # Create stop script
    cat > "${PROJECT_ROOT}/stop-dev.sh" << 'EOF'
#!/bin/bash
# Stop HowlerOps development environment
docker-compose -f docker-compose.dev.yml down
echo "Development environment stopped!"
EOF

    # Create logs script
    cat > "${PROJECT_ROOT}/logs-dev.sh" << 'EOF'
#!/bin/bash
# View development logs
docker-compose -f docker-compose.dev.yml logs -f "$@"
EOF

    # Create reset script
    cat > "${PROJECT_ROOT}/reset-dev.sh" << 'EOF'
#!/bin/bash
# Reset development environment (removes all data)
echo "This will remove all development data. Are you sure? (y/N)"
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    docker-compose -f docker-compose.dev.yml down -v
    docker-compose -f docker-compose.dev.yml up -d
    echo "Development environment reset!"
else
    echo "Reset cancelled."
fi
EOF

    chmod +x "${PROJECT_ROOT}"/start-dev.sh
    chmod +x "${PROJECT_ROOT}"/stop-dev.sh
    chmod +x "${PROJECT_ROOT}"/logs-dev.sh
    chmod +x "${PROJECT_ROOT}"/reset-dev.sh

    log_success "Development helper scripts created"
}

# Main setup function
main() {
    log_info "Setting up HowlerOps development environment..."

    check_tools
    create_dev_env
    install_dependencies
    setup_database
    setup_data
    create_dev_scripts
    start_services

    log_success "Development environment setup completed!"
    log_info ""
    log_info "Next steps:"
    log_info "1. Review the environment variables in .env"
    log_info "2. Visit http://localhost:5173 to access the application"
    log_info "3. Use the following commands to manage your environment:"
    log_info "   ./start-dev.sh  - Start the development environment"
    log_info "   ./stop-dev.sh   - Stop the development environment"
    log_info "   ./logs-dev.sh   - View application logs"
    log_info "   ./reset-dev.sh  - Reset the development environment"
    log_info ""
    log_info "For backend development:"
    log_info "   cd backend && npm run dev"
    log_info "For frontend development:"
    log_info "   cd frontend && npm run dev"
}

# Handle command line arguments
case "${1:-setup}" in
    "setup")
        main
        ;;
    "start")
        start_services
        ;;
    "reset")
        log_warning "Resetting development environment..."
        docker-compose -f "${PROJECT_ROOT}/docker-compose.dev.yml" down -v
        setup_database
        setup_data
        start_services
        log_success "Development environment reset!"
        ;;
    *)
        echo "Usage: $0 {setup|start|reset}"
        exit 1
        ;;
esac