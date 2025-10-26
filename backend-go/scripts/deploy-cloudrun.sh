#!/bin/bash
# =============================================================================
# Google Cloud Run Deployment Script
# =============================================================================
# This script automates the deployment of SQL Studio backend to GCP Cloud Run.
# It handles:
# - Environment validation
# - Secret management via Secret Manager
# - Service account creation
# - Cloud Build triggering
# - Post-deployment verification
#
# Prerequisites:
# - gcloud CLI installed and authenticated
# - Required environment variables set
# - GCP project with billing enabled
#
# Usage:
#   export GCP_PROJECT_ID="your-project-id"
#   export TURSO_URL="your-turso-url"
#   export TURSO_AUTH_TOKEN="your-token"
#   export RESEND_API_KEY="your-api-key"
#   export JWT_SECRET="your-jwt-secret"
#   ./scripts/deploy-cloudrun.sh
# =============================================================================

set -e  # Exit on error
set -u  # Exit on undefined variable
set -o pipefail  # Exit on pipe failure

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="howlerops-backend"
SERVICE_ACCOUNT_NAME="howlerops-backend"

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
        log_error "$1 is not installed. Please install it first."
        exit 1
    fi
}

# -----------------------------------------------------------------------------
# Validation
# -----------------------------------------------------------------------------

log_info "Starting GCP Cloud Run deployment for SQL Studio Backend"

# Check required commands
check_command "gcloud"
check_command "jq"

# Check required environment variables
required_vars=(
    "GCP_PROJECT_ID"
    "TURSO_URL"
    "TURSO_AUTH_TOKEN"
    "RESEND_API_KEY"
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

log_success "Environment validation passed"

# -----------------------------------------------------------------------------
# GCP Project Setup
# -----------------------------------------------------------------------------

log_info "Setting GCP project to $GCP_PROJECT_ID"
gcloud config set project "$GCP_PROJECT_ID"

# Get current project to verify
CURRENT_PROJECT=$(gcloud config get-value project)
if [ "$CURRENT_PROJECT" != "$GCP_PROJECT_ID" ]; then
    log_error "Failed to set project. Current project: $CURRENT_PROJECT"
    exit 1
fi

log_success "GCP project set successfully"

# -----------------------------------------------------------------------------
# Enable Required APIs
# -----------------------------------------------------------------------------

log_info "Enabling required GCP APIs..."

apis=(
    "cloudbuild.googleapis.com"
    "run.googleapis.com"
    "secretmanager.googleapis.com"
    "containerregistry.googleapis.com"
    "artifactregistry.googleapis.com"
)

for api in "${apis[@]}"; do
    log_info "Enabling $api..."
    gcloud services enable "$api" --quiet || log_warning "Failed to enable $api (may already be enabled)"
done

log_success "APIs enabled"

# -----------------------------------------------------------------------------
# Service Account Setup
# -----------------------------------------------------------------------------

log_info "Setting up service account..."

# Create service account if it doesn't exist
if ! gcloud iam service-accounts describe "${SERVICE_ACCOUNT_NAME}@${GCP_PROJECT_ID}.iam.gserviceaccount.com" &> /dev/null; then
    log_info "Creating service account: $SERVICE_ACCOUNT_NAME"
    gcloud iam service-accounts create "$SERVICE_ACCOUNT_NAME" \
        --display-name="SQL Studio Backend Service Account" \
        --description="Service account for SQL Studio backend running on Cloud Run"
    log_success "Service account created"
else
    log_info "Service account already exists"
fi

# Grant necessary roles to service account
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_NAME}@${GCP_PROJECT_ID}.iam.gserviceaccount.com"

log_info "Granting IAM roles to service account..."

roles=(
    "roles/secretmanager.secretAccessor"  # Access secrets
    "roles/logging.logWriter"             # Write logs
    "roles/cloudtrace.agent"              # Write traces
)

for role in "${roles[@]}"; do
    gcloud projects add-iam-policy-binding "$GCP_PROJECT_ID" \
        --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
        --role="$role" \
        --quiet > /dev/null || log_warning "Failed to grant $role"
done

log_success "Service account configured"

# -----------------------------------------------------------------------------
# Secret Manager Setup
# -----------------------------------------------------------------------------

log_info "Setting up secrets in Secret Manager..."

create_or_update_secret() {
    local secret_name="$1"
    local secret_value="$2"

    if gcloud secrets describe "$secret_name" &> /dev/null; then
        log_info "Updating existing secret: $secret_name"
        echo -n "$secret_value" | gcloud secrets versions add "$secret_name" --data-file=-
    else
        log_info "Creating new secret: $secret_name"
        echo -n "$secret_value" | gcloud secrets create "$secret_name" \
            --replication-policy="automatic" \
            --data-file=-
    fi

    # Grant service account access to the secret
    gcloud secrets add-iam-policy-binding "$secret_name" \
        --member="serviceAccount:$SERVICE_ACCOUNT_EMAIL" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet > /dev/null
}

# Create/update all secrets
create_or_update_secret "turso-url" "$TURSO_URL"
create_or_update_secret "turso-auth-token" "$TURSO_AUTH_TOKEN"
create_or_update_secret "resend-api-key" "$RESEND_API_KEY"
create_or_update_secret "jwt-secret" "$JWT_SECRET"

log_success "Secrets configured in Secret Manager"

# -----------------------------------------------------------------------------
# Build and Deploy
# -----------------------------------------------------------------------------

log_info "Building and deploying application..."

# Change to backend-go directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR/.."

# Submit build to Cloud Build
log_info "Submitting build to Cloud Build..."

# Get commit SHA (use git if available, otherwise use timestamp)
if command -v git &> /dev/null && git rev-parse HEAD &> /dev/null; then
    COMMIT_SHA=$(git rev-parse --short=8 HEAD)
    log_info "Using git commit SHA: $COMMIT_SHA"
else
    COMMIT_SHA=$(date +%Y%m%d-%H%M%S)
    log_warning "Git not available, using timestamp as version: $COMMIT_SHA"
fi

gcloud builds submit \
    --config cloudbuild.yaml \
    --substitutions=COMMIT_SHA="$COMMIT_SHA" \
    --timeout=30m

log_success "Build and deployment completed"

# -----------------------------------------------------------------------------
# Post-Deployment Verification
# -----------------------------------------------------------------------------

log_info "Verifying deployment..."

# Get service URL
SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" \
    --region="$REGION" \
    --format='value(status.url)')

if [ -z "$SERVICE_URL" ]; then
    log_error "Failed to get service URL"
    exit 1
fi

log_success "Service deployed successfully"
log_success "Service URL: $SERVICE_URL"

# Wait a bit for the service to be fully ready
log_info "Waiting for service to be ready..."
sleep 10

# Test health endpoint
log_info "Testing health endpoint..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/health" || echo "000")

if [ "$HTTP_STATUS" = "200" ]; then
    log_success "Health check passed (HTTP $HTTP_STATUS)"
else
    log_warning "Health check returned HTTP $HTTP_STATUS (service may still be starting)"
fi

# -----------------------------------------------------------------------------
# Display Information
# -----------------------------------------------------------------------------

echo ""
echo "========================================================================="
echo "Deployment Summary"
echo "========================================================================="
echo "Project ID:      $GCP_PROJECT_ID"
echo "Region:          $REGION"
echo "Service Name:    $SERVICE_NAME"
echo "Service URL:     $SERVICE_URL"
echo "Service Account: $SERVICE_ACCOUNT_EMAIL"
echo ""
echo "Next Steps:"
echo "1. Test your API: curl $SERVICE_URL/health"
echo "2. View logs: gcloud run services logs read $SERVICE_NAME --region=$REGION"
echo "3. View service: gcloud run services describe $SERVICE_NAME --region=$REGION"
echo "4. Update domain: gcloud run services update-traffic $SERVICE_NAME --to-latest --region=$REGION"
echo ""
echo "Monitoring:"
echo "- Cloud Console: https://console.cloud.google.com/run/detail/$REGION/$SERVICE_NAME"
echo "- Logs: https://console.cloud.google.com/logs/query;query=resource.type%3D%22cloud_run_revision%22"
echo "- Metrics: https://console.cloud.google.com/monitoring"
echo "========================================================================="
echo ""

log_success "Deployment completed successfully!"
