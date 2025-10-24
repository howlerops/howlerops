#!/bin/bash
# =============================================================================
# Google Cloud Platform Deployment Script for SQL Studio Backend
# =============================================================================
# This script automates the deployment of SQL Studio backend to GCP Cloud Run
#
# Prerequisites:
#   - gcloud CLI installed and authenticated
#   - Docker installed (for local builds)
#   - GCP project with Cloud Run and Secret Manager APIs enabled
#   - Service account with necessary permissions created
#
# Usage:
#   ./scripts/deploy-gcp.sh [OPTIONS]
#
# Options:
#   --project PROJECT_ID      GCP project ID (required)
#   --region REGION           GCP region (default: us-central1)
#   --service SERVICE_NAME    Cloud Run service name (default: sql-studio-backend)
#   --min-instances N         Minimum instances (default: 0)
#   --max-instances N         Maximum instances (default: 10)
#   --memory SIZE             Memory limit (default: 512Mi)
#   --cpu N                   CPU allocation (default: 1)
#   --env ENV_NAME            Environment name (default: production)
#   --setup-secrets           Setup secrets in Secret Manager first
#   --use-cloudbuild          Use Cloud Build instead of local Docker
#   --dry-run                 Show what would be deployed without deploying
#   --help                    Show this help message
#
# Examples:
#   # First time setup (create secrets)
#   ./scripts/deploy-gcp.sh --project my-project --setup-secrets
#
#   # Deploy using Cloud Build (recommended for CI/CD)
#   ./scripts/deploy-gcp.sh --project my-project --use-cloudbuild
#
#   # Deploy with custom configuration
#   ./scripts/deploy-gcp.sh --project my-project --region us-west1 --max-instances 20
#
#   # Dry run to preview changes
#   ./scripts/deploy-gcp.sh --project my-project --dry-run
# =============================================================================

set -euo pipefail

# =============================================================================
# Configuration and Defaults
# =============================================================================

# Script directory and project root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
GCP_PROJECT="${GCP_PROJECT:-}"
GCP_REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="${SERVICE_NAME:-sql-studio-backend}"
MIN_INSTANCES="${MIN_INSTANCES:-0}"
MAX_INSTANCES="${MAX_INSTANCES:-10}"
MEMORY="${MEMORY:-512Mi}"
CPU="${CPU:-1}"
ENVIRONMENT="${ENVIRONMENT:-production}"
SETUP_SECRETS=false
USE_CLOUDBUILD=false
DRY_RUN=false

# Service account name
SERVICE_ACCOUNT="${SERVICE_NAME}@${GCP_PROJECT}.iam.gserviceaccount.com"

# Image name
IMAGE_NAME="gcr.io/${GCP_PROJECT}/${SERVICE_NAME}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# =============================================================================
# Helper Functions
# =============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

show_help() {
    grep '^#' "$0" | grep -v '#!/bin/bash' | sed 's/^# //g' | sed 's/^#//g'
}

check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check gcloud CLI
    if ! command -v gcloud &> /dev/null; then
        log_error "gcloud CLI not found. Install from: https://cloud.google.com/sdk/docs/install"
        exit 1
    fi

    # Check Docker (only if not using Cloud Build)
    if [[ "${USE_CLOUDBUILD}" == "false" ]] && ! command -v docker &> /dev/null; then
        log_error "Docker not found. Install from: https://docs.docker.com/get-docker/"
        exit 1
    fi

    # Check if project is set
    if [[ -z "${GCP_PROJECT}" ]]; then
        log_error "GCP project ID not set. Use --project flag or set GCP_PROJECT environment variable"
        exit 1
    fi

    # Verify gcloud authentication
    if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" &> /dev/null; then
        log_error "Not authenticated with gcloud. Run: gcloud auth login"
        exit 1
    fi

    # Set project
    gcloud config set project "${GCP_PROJECT}" --quiet

    log_success "Prerequisites check passed"
}

enable_apis() {
    log_info "Enabling required GCP APIs..."

    gcloud services enable \
        cloudbuild.googleapis.com \
        run.googleapis.com \
        secretmanager.googleapis.com \
        containerregistry.googleapis.com \
        --project="${GCP_PROJECT}" \
        --quiet

    log_success "APIs enabled"
}

setup_service_account() {
    log_info "Setting up service account: ${SERVICE_ACCOUNT}"

    # Check if service account exists
    if gcloud iam service-accounts describe "${SERVICE_ACCOUNT}" \
        --project="${GCP_PROJECT}" &> /dev/null; then
        log_warn "Service account already exists"
    else
        # Create service account
        gcloud iam service-accounts create "${SERVICE_NAME}" \
            --display-name="SQL Studio Backend Service Account" \
            --description="Service account for SQL Studio backend running on Cloud Run" \
            --project="${GCP_PROJECT}"

        log_success "Service account created"
    fi

    # Grant necessary permissions
    log_info "Granting IAM permissions..."

    gcloud projects add-iam-policy-binding "${GCP_PROJECT}" \
        --member="serviceAccount:${SERVICE_ACCOUNT}" \
        --role="roles/secretmanager.secretAccessor" \
        --condition=None \
        --quiet

    log_success "Service account configured"
}

setup_secrets() {
    log_info "Setting up secrets in Secret Manager..."

    # Function to create or update a secret
    create_or_update_secret() {
        local secret_name=$1
        local secret_description=$2

        echo ""
        echo -e "${YELLOW}Secret: ${secret_name}${NC}"
        echo "Description: ${secret_description}"
        read -p "Enter value (will be hidden): " -s secret_value
        echo ""

        if [[ -z "${secret_value}" ]]; then
            log_warn "Empty value provided, skipping ${secret_name}"
            return
        fi

        # Check if secret exists
        if gcloud secrets describe "${secret_name}" \
            --project="${GCP_PROJECT}" &> /dev/null 2>&1; then
            # Update existing secret
            echo -n "${secret_value}" | gcloud secrets versions add "${secret_name}" \
                --data-file=- \
                --project="${GCP_PROJECT}"
            log_success "Secret ${secret_name} updated"
        else
            # Create new secret
            echo -n "${secret_value}" | gcloud secrets create "${secret_name}" \
                --data-file=- \
                --replication-policy="automatic" \
                --project="${GCP_PROJECT}"
            log_success "Secret ${secret_name} created"
        fi
    }

    echo ""
    echo "=========================================="
    echo "Setting up required secrets"
    echo "=========================================="
    echo ""

    create_or_update_secret "turso-url" "Turso database URL (e.g., libsql://your-db.turso.io)"
    create_or_update_secret "turso-auth-token" "Turso authentication token"
    create_or_update_secret "jwt-secret" "JWT signing secret (min 32 characters)"
    create_or_update_secret "resend-api-key" "Resend email service API key"

    log_success "All secrets configured"
}

build_image_local() {
    log_info "Building Docker image locally..."

    cd "${PROJECT_ROOT}"

    # Get version info
    VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

    # Build image
    docker build \
        --build-arg VERSION="${VERSION}" \
        --build-arg BUILD_TIME="${BUILD_TIME}" \
        --build-arg GIT_COMMIT="${GIT_COMMIT}" \
        -t "${IMAGE_NAME}:${VERSION}" \
        -t "${IMAGE_NAME}:latest" \
        .

    log_success "Image built successfully"

    # Push to GCR
    log_info "Pushing image to Google Container Registry..."

    docker push "${IMAGE_NAME}:${VERSION}"
    docker push "${IMAGE_NAME}:latest"

    log_success "Image pushed to GCR"

    # Export for deployment
    export DEPLOY_IMAGE="${IMAGE_NAME}:${VERSION}"
}

build_image_cloudbuild() {
    log_info "Building image using Cloud Build..."

    cd "${PROJECT_ROOT}"

    # Get version info
    VERSION="${VERSION:-$(date +%Y%m%d-%H%M%S)}"
    GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

    # Submit build to Cloud Build
    gcloud builds submit \
        --config=cloudbuild.yaml \
        --substitutions="_VERSION=${VERSION},_GIT_COMMIT=${GIT_COMMIT}" \
        --project="${GCP_PROJECT}" \
        .

    log_success "Image built using Cloud Build"

    # Export for deployment
    export DEPLOY_IMAGE="${IMAGE_NAME}:${GIT_COMMIT}"
}

deploy_to_cloudrun() {
    log_info "Deploying to Cloud Run..."

    if [[ "${DRY_RUN}" == "true" ]]; then
        log_warn "DRY RUN MODE - Not actually deploying"
        echo ""
        echo "Would deploy with following configuration:"
        echo "  Project: ${GCP_PROJECT}"
        echo "  Region: ${GCP_REGION}"
        echo "  Service: ${SERVICE_NAME}"
        echo "  Image: ${DEPLOY_IMAGE}"
        echo "  Memory: ${MEMORY}"
        echo "  CPU: ${CPU}"
        echo "  Min Instances: ${MIN_INSTANCES}"
        echo "  Max Instances: ${MAX_INSTANCES}"
        echo "  Environment: ${ENVIRONMENT}"
        echo ""
        return
    fi

    cd "${PROJECT_ROOT}"

    # Deploy to Cloud Run
    gcloud run deploy "${SERVICE_NAME}" \
        --image="${DEPLOY_IMAGE}" \
        --platform=managed \
        --region="${GCP_REGION}" \
        --project="${GCP_PROJECT}" \
        --allow-unauthenticated \
        --port=8500 \
        --memory="${MEMORY}" \
        --cpu="${CPU}" \
        --min-instances="${MIN_INSTANCES}" \
        --max-instances="${MAX_INSTANCES}" \
        --concurrency=80 \
        --timeout=300 \
        --cpu-throttling \
        --set-env-vars="ENVIRONMENT=${ENVIRONMENT},LOG_FORMAT=json,LOG_LEVEL=info,SERVER_HTTP_PORT=8500,SERVER_GRPC_PORT=9500,METRICS_PORT=9100" \
        --set-secrets="TURSO_URL=turso-url:latest,TURSO_AUTH_TOKEN=turso-auth-token:latest,RESEND_API_KEY=resend-api-key:latest,JWT_SECRET=jwt-secret:latest" \
        --service-account="${SERVICE_ACCOUNT}" \
        --labels="app=sql-studio,component=backend,environment=${ENVIRONMENT},managed-by=deploy-script" \
        --execution-environment=gen2 \
        --quiet

    log_success "Deployed to Cloud Run"

    # Get service URL
    SERVICE_URL=$(gcloud run services describe "${SERVICE_NAME}" \
        --region="${GCP_REGION}" \
        --project="${GCP_PROJECT}" \
        --format='value(status.url)')

    log_success "Service URL: ${SERVICE_URL}"
}

run_smoke_tests() {
    log_info "Running smoke tests..."

    if [[ "${DRY_RUN}" == "true" ]]; then
        log_warn "Skipping smoke tests in dry-run mode"
        return
    fi

    # Get service URL
    SERVICE_URL=$(gcloud run services describe "${SERVICE_NAME}" \
        --region="${GCP_REGION}" \
        --project="${GCP_PROJECT}" \
        --format='value(status.url)')

    # Wait for service to be ready
    log_info "Waiting for service to be ready..."
    sleep 10

    # Test health endpoint
    log_info "Testing health endpoint: ${SERVICE_URL}/health"
    HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "${SERVICE_URL}/health")

    if [[ "${HTTP_CODE}" == "200" ]]; then
        log_success "Health check passed (HTTP ${HTTP_CODE})"
    else
        log_error "Health check failed (HTTP ${HTTP_CODE})"
        exit 1
    fi

    log_success "All smoke tests passed"
}

show_deployment_info() {
    log_info "Deployment information:"
    echo ""
    echo "=========================================="
    echo "Deployment Summary"
    echo "=========================================="
    echo "Project:      ${GCP_PROJECT}"
    echo "Region:       ${GCP_REGION}"
    echo "Service:      ${SERVICE_NAME}"
    echo "Environment:  ${ENVIRONMENT}"
    echo ""

    if [[ "${DRY_RUN}" == "false" ]]; then
        SERVICE_URL=$(gcloud run services describe "${SERVICE_NAME}" \
            --region="${GCP_REGION}" \
            --project="${GCP_PROJECT}" \
            --format='value(status.url)' 2>/dev/null || echo "N/A")

        echo "Service URL:  ${SERVICE_URL}"
        echo ""
        echo "Useful commands:"
        echo "  View logs:    gcloud run services logs read ${SERVICE_NAME} --region=${GCP_REGION} --project=${GCP_PROJECT}"
        echo "  View metrics: gcloud run services describe ${SERVICE_NAME} --region=${GCP_REGION} --project=${GCP_PROJECT}"
        echo "  Update config: gcloud run services update ${SERVICE_NAME} --region=${GCP_REGION} --project=${GCP_PROJECT}"
        echo "  Delete:       gcloud run services delete ${SERVICE_NAME} --region=${GCP_REGION} --project=${GCP_PROJECT}"
    fi

    echo "=========================================="
    echo ""
}

# =============================================================================
# Parse Command Line Arguments
# =============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --project)
            GCP_PROJECT="$2"
            shift 2
            ;;
        --region)
            GCP_REGION="$2"
            shift 2
            ;;
        --service)
            SERVICE_NAME="$2"
            shift 2
            ;;
        --min-instances)
            MIN_INSTANCES="$2"
            shift 2
            ;;
        --max-instances)
            MAX_INSTANCES="$2"
            shift 2
            ;;
        --memory)
            MEMORY="$2"
            shift 2
            ;;
        --cpu)
            CPU="$2"
            shift 2
            ;;
        --env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --setup-secrets)
            SETUP_SECRETS=true
            shift
            ;;
        --use-cloudbuild)
            USE_CLOUDBUILD=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            log_error "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# =============================================================================
# Main Execution
# =============================================================================

log_info "Starting GCP Cloud Run deployment..."
echo ""

# Check prerequisites
check_prerequisites

# Enable required APIs
enable_apis

# Setup service account
setup_service_account

# Setup secrets if requested
if [[ "${SETUP_SECRETS}" == "true" ]]; then
    setup_secrets
    log_info "Secrets setup complete. Run the script again without --setup-secrets to deploy."
    exit 0
fi

# Build image
if [[ "${USE_CLOUDBUILD}" == "true" ]]; then
    build_image_cloudbuild
else
    build_image_local
fi

# Deploy to Cloud Run
deploy_to_cloudrun

# Run smoke tests
run_smoke_tests

# Show deployment info
show_deployment_info

log_success "Deployment completed successfully!"
