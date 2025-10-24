#!/bin/bash
# =============================================================================
# Fly.io Deployment Script
# =============================================================================
# This script automates the deployment of SQL Studio backend to Fly.io.
# It handles:
# - Environment validation
# - Fly.io CLI installation check
# - App creation (if needed)
# - Secret management
# - Deployment with zero-downtime
# - Post-deployment verification
#
# Prerequisites:
# - flyctl installed (https://fly.io/docs/hands-on/install-flyctl/)
# - Fly.io account with payment method configured
# - Required environment variables set
#
# Usage:
#   export TURSO_URL="your-turso-url"
#   export TURSO_AUTH_TOKEN="your-token"
#   export RESEND_API_KEY="your-api-key"
#   export RESEND_FROM_EMAIL="noreply@yourdomain.com"
#   export JWT_SECRET="your-jwt-secret"
#   ./scripts/deploy-fly.sh
# =============================================================================

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
APP_NAME="${FLY_APP_NAME:-sql-studio-backend}"
REGION="${FLY_REGION:-sjc}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# -----------------------------------------------------------------------------
# Helper Functions
# -----------------------------------------------------------------------------

log_info() {
    echo -e "${BLUE}INFO:${NC} $1"
}

log_success() {
    echo -e "${GREEN}SUCCESS:${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}WARNING:${NC} $1"
}

log_error() {
    echo -e "${RED}ERROR:${NC} $1"
}

check_command() {
    if ! command -v "$1" &> /dev/null; then
        log_error "$1 is not installed."
        if [ "$1" = "flyctl" ]; then
            log_info "Install flyctl from: https://fly.io/docs/hands-on/install-flyctl/"
            log_info "  macOS: brew install flyctl"
            log_info "  Linux: curl -L https://fly.io/install.sh | sh"
            log_info "  Windows: iwr https://fly.io/install.ps1 -useb | iex"
        fi
        exit 1
    fi
}

# -----------------------------------------------------------------------------
# Validation
# -----------------------------------------------------------------------------

log_info "Starting Fly.io deployment for SQL Studio Backend"

# Check required commands
check_command "flyctl"
check_command "jq"

# Check if authenticated with Fly.io
if ! flyctl auth whoami &> /dev/null; then
    log_error "Not authenticated with Fly.io"
    log_info "Please run: flyctl auth login"
    exit 1
fi

# Check required environment variables
required_vars=(
    "TURSO_URL"
    "TURSO_AUTH_TOKEN"
    "RESEND_API_KEY"
    "RESEND_FROM_EMAIL"
    "JWT_SECRET"
)

for var in "${required_vars[@]}"; do
    if [ -z "${!var:-}" ]; then
        log_error "$var is not set. Please set all required environment variables."
        log_info "Required variables: ${required_vars[*]}"
        exit 1
    fi
done

# Validate JWT secret length (minimum 32 characters for security)
if [ ${#JWT_SECRET} -lt 32 ]; then
    log_error "JWT_SECRET must be at least 32 characters long"
    exit 1
fi

# Validate email format
if [[ ! "$RESEND_FROM_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
    log_error "RESEND_FROM_EMAIL is not a valid email address"
    exit 1
fi

log_success "Environment validation passed"

# -----------------------------------------------------------------------------
# Change to Project Directory
# -----------------------------------------------------------------------------

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

log_info "Working directory: $(pwd)"

# -----------------------------------------------------------------------------
# App Creation/Verification
# -----------------------------------------------------------------------------

log_info "Checking if Fly.io app exists..."

if flyctl apps list | grep -q "^$APP_NAME"; then
    log_info "App '$APP_NAME' already exists"
else
    log_info "Creating new Fly.io app: $APP_NAME"

    # Create app with fly.toml configuration
    # The --now flag deploys immediately, we'll skip it and deploy separately
    flyctl apps create "$APP_NAME" --org personal || {
        log_warning "App creation may have been cancelled or failed"
        log_info "Attempting to continue with existing app..."
    }

    log_success "App created"
fi

# -----------------------------------------------------------------------------
# Allocate IPv4 Address (Required for Public Access)
# -----------------------------------------------------------------------------

log_info "Checking IP addresses..."

# Check if app has IPv4 allocated
if ! flyctl ips list --app "$APP_NAME" 2>/dev/null | grep -q "v4"; then
    log_info "Allocating IPv4 address..."
    flyctl ips allocate-v4 --app "$APP_NAME" || log_warning "IPv4 allocation failed (may already exist)"
fi

# IPv6 is automatically allocated
log_success "IP addresses configured"

# -----------------------------------------------------------------------------
# Secret Management
# -----------------------------------------------------------------------------

log_info "Setting secrets in Fly.io..."

# Set all secrets in one command for atomic update
flyctl secrets set \
    --app "$APP_NAME" \
    TURSO_URL="$TURSO_URL" \
    TURSO_AUTH_TOKEN="$TURSO_AUTH_TOKEN" \
    RESEND_API_KEY="$RESEND_API_KEY" \
    RESEND_FROM_EMAIL="$RESEND_FROM_EMAIL" \
    JWT_SECRET="$JWT_SECRET" \
    --stage  # Stage changes without deploying

log_success "Secrets configured"

# -----------------------------------------------------------------------------
# Deploy Application
# -----------------------------------------------------------------------------

log_info "Deploying application to Fly.io..."

# Deploy with rolling strategy for zero-downtime
# --ha=false: Don't create multiple instances (save costs)
# --strategy rolling: Zero-downtime deployment
flyctl deploy \
    --app "$APP_NAME" \
    --ha=false \
    --strategy rolling

log_success "Deployment completed"

# -----------------------------------------------------------------------------
# Post-Deployment Verification
# -----------------------------------------------------------------------------

log_info "Verifying deployment..."

# Get app hostname
APP_HOSTNAME=$(flyctl info --app "$APP_NAME" --json | jq -r '.Hostname')

if [ -z "$APP_HOSTNAME" ] || [ "$APP_HOSTNAME" = "null" ]; then
    log_error "Failed to get app hostname"
    exit 1
fi

APP_URL="https://$APP_HOSTNAME"

log_success "App deployed successfully"
log_success "App URL: $APP_URL"

# Wait for app to be ready
log_info "Waiting for app to be ready..."
sleep 15

# Test health endpoint
log_info "Testing health endpoint..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$APP_URL/health" || echo "000")

if [ "$HTTP_STATUS" = "200" ]; then
    log_success "Health check passed (HTTP $HTTP_STATUS)"
else
    log_warning "Health check returned HTTP $HTTP_STATUS"
    log_info "The service may still be starting. Check logs with: flyctl logs --app $APP_NAME"
fi

# -----------------------------------------------------------------------------
# Display Information
# -----------------------------------------------------------------------------

echo ""
echo "========================================================================="
echo "Deployment Summary"
echo "========================================================================="
echo "App Name:     $APP_NAME"
echo "Region:       $REGION"
echo "App URL:      $APP_URL"
echo "Dashboard:    https://fly.io/apps/$APP_NAME"
echo ""
echo "Next Steps:"
echo "1. Test your API: curl $APP_URL/health"
echo "2. View logs: flyctl logs --app $APP_NAME"
echo "3. View status: flyctl status --app $APP_NAME"
echo "4. SSH to machine: flyctl ssh console --app $APP_NAME"
echo "5. Monitor metrics: flyctl dashboard metrics --app $APP_NAME"
echo ""
echo "Scaling:"
echo "- Scale instances: flyctl scale count 2 --app $APP_NAME"
echo "- Scale VM: flyctl scale vm shared-cpu-1x --memory 1024 --app $APP_NAME"
echo "- Auto-scale to zero: Already configured in fly.toml"
echo ""
echo "Cost Optimization:"
echo "- Current config: ~\$2-5/month (scale-to-zero enabled)"
echo "- View usage: https://fly.io/dashboard/personal/usage"
echo ""
echo "Monitoring:"
echo "- Dashboard: https://fly.io/apps/$APP_NAME/monitoring"
echo "- Metrics: https://fly.io/apps/$APP_NAME/metrics"
echo "- Logs: flyctl logs --app $APP_NAME --follow"
echo "========================================================================="
echo ""

log_success "Deployment completed successfully!"

# -----------------------------------------------------------------------------
# Optional: Show App Info
# -----------------------------------------------------------------------------

if command -v jq &> /dev/null; then
    log_info "Current app status:"
    flyctl status --app "$APP_NAME"
fi
