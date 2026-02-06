#!/bin/bash
set -euo pipefail

# =============================================================================
# Sync OAuth Secrets to Google Secret Manager
# =============================================================================
# This script creates/updates OAuth credentials in Google Cloud Secret Manager.
#
# Two modes:
# 1. Interactive mode (default): Prompts for secret values
# 2. Environment mode: Reads from environment variables
#
# Prerequisites:
# - gcloud CLI installed and authenticated
# - Environment variables set (if using env mode):
#   GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET, GH_CLIENT_ID, GH_CLIENT_SECRET
# =============================================================================

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODE="${1:-interactive}"  # interactive or env

# Get GCP project ID
GCP_PROJECT_ID=$(gcloud config get-value project 2>/dev/null)

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}🔐 OAuth Secrets Sync to GCP${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check prerequisites
echo -e "${BLUE}Checking prerequisites...${NC}"

if ! command -v gcloud &> /dev/null; then
    echo -e "${RED}❌ gcloud CLI is not installed${NC}"
    echo "Install: https://cloud.google.com/sdk/docs/install"
    exit 1
fi

# Check gcloud authentication
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" &> /dev/null; then
    echo -e "${RED}❌ gcloud is not authenticated${NC}"
    echo "Run: gcloud auth login"
    exit 1
fi

if [ -z "$GCP_PROJECT_ID" ]; then
    echo -e "${RED}❌ No GCP project selected${NC}"
    echo "Run: gcloud config set project YOUR_PROJECT_ID"
    exit 1
fi

echo -e "${GREEN}✅ Prerequisites check passed${NC}"
echo -e "   GCP Project: ${YELLOW}${GCP_PROJECT_ID}${NC}"
echo -e "   Mode: ${YELLOW}${MODE}${NC}"
echo ""

# Function to create/update a secret
create_or_update_secret() {
    local secret_name=$1
    local secret_value=$2
    local description=$3

    if [ -z "$secret_value" ]; then
        echo -e "${YELLOW}⚠️  Skipping ${secret_name} (no value provided)${NC}"
        return 0
    fi

    echo -n "Processing ${secret_name}... "

    # Check if secret exists
    if gcloud secrets describe "$secret_name" --project="$GCP_PROJECT_ID" &> /dev/null; then
        # Update existing secret
        echo -n "$secret_value" | gcloud secrets versions add "$secret_name" \
            --project="$GCP_PROJECT_ID" \
            --data-file=- &> /dev/null
        echo -e "${GREEN}✅ Updated${NC}"
    else
        # Create new secret
        echo -n "$secret_value" | gcloud secrets create "$secret_name" \
            --project="$GCP_PROJECT_ID" \
            --replication-policy="automatic" \
            --data-file=- \
            --labels="app=howlerops,component=oauth,managed-by=script" &> /dev/null
        echo -e "${GREEN}✅ Created${NC}"
    fi
}

# Get secret values based on mode
if [ "$MODE" = "env" ]; then
    echo -e "${BLUE}Reading from environment variables...${NC}"
    echo ""

    GOOGLE_CLIENT_ID_VALUE="${GOOGLE_CLIENT_ID:-}"
    GOOGLE_CLIENT_SECRET_VALUE="${GOOGLE_CLIENT_SECRET:-}"
    GH_CLIENT_ID_VALUE="${GH_CLIENT_ID:-}"
    GH_CLIENT_SECRET_VALUE="${GH_CLIENT_SECRET:-}"

    if [ -z "$GOOGLE_CLIENT_ID_VALUE" ] || [ -z "$GOOGLE_CLIENT_SECRET_VALUE" ] || \
       [ -z "$GH_CLIENT_ID_VALUE" ] || [ -z "$GH_CLIENT_SECRET_VALUE" ]; then
        echo -e "${RED}❌ Missing required environment variables${NC}"
        echo ""
        echo "Please set all of the following:"
        echo "  export GOOGLE_CLIENT_ID='your-client-id.apps.googleusercontent.com'"
        echo "  export GOOGLE_CLIENT_SECRET='GOCSPX-...'"
        echo "  export GH_CLIENT_ID='Ov23li...'"
        echo "  export GH_CLIENT_SECRET='your-secret'"
        echo ""
        echo "Or run in interactive mode: $0 interactive"
        exit 1
    fi
else
    echo -e "${BLUE}Interactive mode: Please enter OAuth credentials${NC}"
    echo -e "${YELLOW}(You can find these in your OAuth app settings)${NC}"
    echo ""

    # Google OAuth
    echo -e "${BLUE}Google OAuth Credentials:${NC}"
    echo -n "  Client ID: "
    read GOOGLE_CLIENT_ID_VALUE
    echo -n "  Client Secret: "
    read -s GOOGLE_CLIENT_SECRET_VALUE
    echo ""
    echo ""

    # GitHub OAuth
    echo -e "${BLUE}GitHub OAuth Credentials:${NC}"
    echo -n "  Client ID: "
    read GH_CLIENT_ID_VALUE
    echo -n "  Client Secret: "
    read -s GH_CLIENT_SECRET_VALUE
    echo ""
    echo ""
fi

# Create/update all secrets
echo -e "${BLUE}Creating/updating secrets in GCP...${NC}"
echo ""

create_or_update_secret "google-oauth-client-id" "$GOOGLE_CLIENT_ID_VALUE" "Google OAuth Client ID"
create_or_update_secret "google-oauth-client-secret" "$GOOGLE_CLIENT_SECRET_VALUE" "Google OAuth Client Secret"
create_or_update_secret "github-oauth-client-id" "$GH_CLIENT_ID_VALUE" "GitHub OAuth Client ID"
create_or_update_secret "github-oauth-client-secret" "$GH_CLIENT_SECRET_VALUE" "GitHub OAuth Client Secret"

# Summary
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ OAuth Secrets Sync Complete${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "Secrets synced to GCP project: ${YELLOW}${GCP_PROJECT_ID}${NC}"
echo ""
echo "Next steps:"
echo ""
echo "  1. ${BLUE}Verify secrets were created:${NC}"
echo "     ${YELLOW}gcloud secrets list --project=${GCP_PROJECT_ID} | grep oauth${NC}"
echo ""
echo "  2. ${BLUE}Grant Cloud Run service account access:${NC}"
echo "     ${YELLOW}gcloud projects add-iam-policy-binding ${GCP_PROJECT_ID} \\${NC}"
echo "     ${YELLOW}  --member='serviceAccount:howlerops-backend@${GCP_PROJECT_ID}.iam.gserviceaccount.com' \\${NC}"
echo "     ${YELLOW}  --role='roles/secretmanager.secretAccessor'${NC}"
echo ""
echo "  3. ${BLUE}Update cloudbuild.yaml to include OAuth secrets:${NC}"
echo "     Add to --set-secrets:"
echo "     ${YELLOW}GOOGLE_CLIENT_ID=google-oauth-client-id:latest,${NC}"
echo "     ${YELLOW}GOOGLE_CLIENT_SECRET=google-oauth-client-secret:latest,${NC}"
echo "     ${YELLOW}GH_CLIENT_ID=github-oauth-client-id:latest,${NC}"
echo "     ${YELLOW}GH_CLIENT_SECRET=github-oauth-client-secret:latest${NC}"
echo ""
echo "  4. ${BLUE}Deploy to Cloud Run:${NC}"
echo "     ${YELLOW}./scripts/deploy-cloudrun.sh${NC}"
echo ""
