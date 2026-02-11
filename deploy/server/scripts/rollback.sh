#!/bin/bash
# ============================================================================
# Quick Rollback Script for Howlerops Backend
# ============================================================================
# This script provides fast rollback to the previous stable version
#
# Usage:
#   ./scripts/rollback.sh [OPTIONS]
#
# Options:
#   --project PROJECT_ID   GCP project ID (required)
#   --region REGION        GCP region (default: us-central1)
#   --service SERVICE      Cloud Run service name (default: sql-studio-backend)
#   --revision REVISION    Specific revision to rollback to (optional)
#   --percentage PERCENT   Traffic percentage for gradual rollback (default: 100)
#   --dry-run              Show what would be done without executing
#   --help                 Show this help message
# ============================================================================

set -euo pipefail

# Configuration
GCP_PROJECT="${GCP_PROJECT:-}"
GCP_REGION="${GCP_REGION:-us-central1}"
SERVICE_NAME="${SERVICE_NAME:-sql-studio-backend}"
TARGET_REVISION=""
TRAFFIC_PERCENTAGE=100
DRY_RUN=false

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

show_help() {
    grep '^#' "$0" | grep -v '#!/bin/bash' | head -n 20 | sed 's/^# //g'
}

# Parse arguments
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
        --revision)
            TARGET_REVISION="$2"
            shift 2
            ;;
        --percentage)
            TRAFFIC_PERCENTAGE="$2"
            shift 2
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
            echo "Use --help for usage"
            exit 1
            ;;
    esac
done

# Validate required parameters
if [[ -z "$GCP_PROJECT" ]]; then
    log_error "Project ID is required (--project or GCP_PROJECT env var)"
    exit 1
fi

echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Howlerops Backend - Emergency Rollback${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

log_info "Service: $SERVICE_NAME"
log_info "Region: $GCP_REGION"
log_info "Project: $GCP_PROJECT"
echo ""

# Get current revision info
log_info "Fetching current deployment status..."
CURRENT_REVISION=$(gcloud run services describe "$SERVICE_NAME" \
    --region="$GCP_REGION" \
    --project="$GCP_PROJECT" \
    --format="value(status.latestReadyRevisionName)" 2>/dev/null || echo "")

if [[ -z "$CURRENT_REVISION" ]]; then
    log_error "Could not fetch current revision. Is the service deployed?"
    exit 1
fi

log_info "Current revision: $CURRENT_REVISION"

# Get traffic distribution
CURRENT_TRAFFIC=$(gcloud run services describe "$SERVICE_NAME" \
    --region="$GCP_REGION" \
    --project="$GCP_PROJECT" \
    --format="table(status.traffic[].revisionName,status.traffic[].percent)" 2>/dev/null || echo "")

echo ""
echo "Current traffic distribution:"
echo "$CURRENT_TRAFFIC"
echo ""

# Determine target revision
if [[ -z "$TARGET_REVISION" ]]; then
    log_info "No specific revision specified, finding previous stable version..."

    # Get list of revisions (excluding current)
    REVISIONS=$(gcloud run revisions list \
        --service="$SERVICE_NAME" \
        --region="$GCP_REGION" \
        --project="$GCP_PROJECT" \
        --format="value(metadata.name)" \
        --filter="metadata.name!=$CURRENT_REVISION" \
        --limit=5 2>/dev/null)

    if [[ -z "$REVISIONS" ]]; then
        log_error "No previous revisions found. Cannot rollback."
        exit 1
    fi

    # Use the most recent previous revision
    TARGET_REVISION=$(echo "$REVISIONS" | head -n 1)
    log_info "Selected previous revision: $TARGET_REVISION"
else
    log_info "Using specified revision: $TARGET_REVISION"
fi

# Check if target revision exists and is ready
log_info "Verifying target revision..."
REVISION_STATUS=$(gcloud run revisions describe "$TARGET_REVISION" \
    --region="$GCP_REGION" \
    --project="$GCP_PROJECT" \
    --format="value(status.conditions[0].status)" 2>/dev/null || echo "")

if [[ "$REVISION_STATUS" != "True" ]]; then
    log_error "Target revision $TARGET_REVISION is not ready (status: $REVISION_STATUS)"
    exit 1
fi

log_success "Target revision is ready"

# Confirm rollback
echo ""
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}Rollback Plan:${NC}"
echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo "From: $CURRENT_REVISION"
echo "To:   $TARGET_REVISION"
echo "Traffic: ${TRAFFIC_PERCENTAGE}%"
echo ""

if [[ "$DRY_RUN" == "true" ]]; then
    log_warn "DRY RUN MODE - No changes will be made"
    echo ""
    echo "Would execute:"
    echo "gcloud run services update-traffic $SERVICE_NAME \\"
    echo "  --region=$GCP_REGION \\"
    echo "  --project=$GCP_PROJECT \\"
    echo "  --to-revisions=${TARGET_REVISION}=${TRAFFIC_PERCENTAGE}"
    exit 0
fi

read -p "Proceed with rollback? (yes/no): " confirmation
if [[ "$confirmation" != "yes" ]]; then
    log_warn "Rollback cancelled"
    exit 0
fi

# Perform rollback
log_info "Initiating rollback..."
if [[ $TRAFFIC_PERCENTAGE -eq 100 ]]; then
    # Full rollback
    gcloud run services update-traffic "$SERVICE_NAME" \
        --region="$GCP_REGION" \
        --project="$GCP_PROJECT" \
        --to-revisions="${TARGET_REVISION}=100" \
        --quiet
else
    # Gradual rollback
    REMAINING=$((100 - TRAFFIC_PERCENTAGE))
    gcloud run services update-traffic "$SERVICE_NAME" \
        --region="$GCP_REGION" \
        --project="$GCP_PROJECT" \
        --to-revisions="${TARGET_REVISION}=${TRAFFIC_PERCENTAGE},${CURRENT_REVISION}=${REMAINING}" \
        --quiet
fi

log_success "Traffic updated"

# Wait for changes to propagate
log_info "Waiting for changes to propagate..."
sleep 10

# Verify rollback
log_info "Verifying rollback..."
NEW_TRAFFIC=$(gcloud run services describe "$SERVICE_NAME" \
    --region="$GCP_REGION" \
    --project="$GCP_PROJECT" \
    --format="table(status.traffic[].revisionName,status.traffic[].percent)" 2>/dev/null)

echo ""
echo "New traffic distribution:"
echo "$NEW_TRAFFIC"
echo ""

# Test health endpoint
SERVICE_URL=$(gcloud run services describe "$SERVICE_NAME" \
    --region="$GCP_REGION" \
    --project="$GCP_PROJECT" \
    --format="value(status.url)" 2>/dev/null)

if [[ -n "$SERVICE_URL" ]]; then
    log_info "Testing health endpoint..."
    if curl -f -s -m 10 "$SERVICE_URL/health" > /dev/null 2>&1; then
        log_success "Health check passed"
    else
        log_error "Health check failed - manual intervention may be required"
    fi
fi

# Summary
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}Rollback Complete${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Service: $SERVICE_NAME"
echo "Active Revision: $TARGET_REVISION"
echo "Service URL: $SERVICE_URL"
echo ""
echo "Next steps:"
echo "1. Monitor service metrics and logs"
echo "2. Investigate root cause of the issue"
echo "3. Create incident report"
echo "4. Fix issues before next deployment"
echo ""
echo "Commands for monitoring:"
echo "  # View logs"
echo "  gcloud logging read \"resource.type=cloud_run_revision\" --limit=50 --project=$GCP_PROJECT"
echo ""
echo "  # Check metrics"
echo "  gcloud monitoring dashboards list --project=$GCP_PROJECT"
echo ""
echo "  # View service details"
echo "  gcloud run services describe $SERVICE_NAME --region=$GCP_REGION --project=$GCP_PROJECT"