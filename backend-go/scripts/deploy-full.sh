#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}🚀 SQL Studio Backend - Full Production Deployment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "This script will:"
echo "  1. Check production readiness"
echo "  2. Setup GCP secrets"
echo "  3. Deploy to Cloud Run"
echo "  4. Verify the deployment"
echo ""

# Confirm deployment
read -p "Continue with deployment? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Deployment cancelled."
    exit 0
fi
echo ""

# Change to script directory to ensure relative paths work
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 1: Production Readiness Check
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 1/4: Production Readiness Check${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if ! ./scripts/prod-readiness-check.sh; then
    echo ""
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ Production readiness check failed${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Please fix the issues above before deploying."
    echo ""
    echo "Common issues:"
    echo "  - Missing environment variables"
    echo "  - Tests failing"
    echo "  - Build errors"
    echo "  - GCP authentication issues"
    echo ""
    echo "See PRODUCTION_CHECKLIST.md for detailed requirements."
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Readiness check passed${NC}"
echo ""

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 2: Setup GCP Secrets
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 2/4: Setting up GCP Secrets${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if ! ./scripts/setup-secrets.sh; then
    echo ""
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ Failed to setup GCP secrets${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Please check:"
    echo "  - GCP authentication is valid"
    echo "  - Required APIs are enabled"
    echo "  - IAM permissions are correct"
    echo ""
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Secrets configured${NC}"
echo ""

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 3: Deploy to Cloud Run
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 3/4: Deploying to Cloud Run${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Check if deploy-cloudrun.sh exists
if [ -f "./scripts/deploy-cloudrun.sh" ]; then
    echo "Using existing deploy-cloudrun.sh script..."
    if ! ./scripts/deploy-cloudrun.sh; then
        echo ""
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${RED}❌ Deployment failed${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "Check the logs above for errors."
        echo ""
        exit 1
    fi
else
    echo "deploy-cloudrun.sh not found, using inline deployment..."

    # Set project
    echo "Setting GCP project..."
    gcloud config set project $GCP_PROJECT_ID

    # Deploy using gcloud run deploy (simple method)
    echo "Deploying to Cloud Run..."

    if ! gcloud run deploy sql-studio-backend \
        --source . \
        --region=us-central1 \
        --platform=managed \
        --allow-unauthenticated \
        --min-instances=0 \
        --max-instances=10 \
        --memory=512Mi \
        --timeout=300 \
        --set-env-vars="ENVIRONMENT=production,LOG_LEVEL=info,LOG_FORMAT=json" \
        --set-secrets="TURSO_URL=turso-url:latest,TURSO_AUTH_TOKEN=turso-auth-token:latest,JWT_SECRET=jwt-secret:latest,RESEND_API_KEY=resend-api-key:latest" \
        --quiet; then

        echo ""
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${RED}❌ Deployment failed${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo ""
        echo "Common issues:"
        echo "  - Build errors in Dockerfile"
        echo "  - Missing dependencies in go.mod"
        echo "  - Incorrect Cloud Build configuration"
        echo ""
        echo "Check build logs:"
        echo "  gcloud builds list --limit=5"
        echo "  gcloud builds log BUILD_ID"
        echo ""
        exit 1
    fi
fi

echo ""
echo -e "${GREEN}✅ Deployment successful${NC}"
echo ""

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Step 4: Verify Deployment
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 4/4: Verifying Deployment${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Get service URL
echo "Retrieving service URL..."
SERVICE_URL=$(gcloud run services describe sql-studio-backend \
    --region=us-central1 \
    --format='value(status.url)' 2>/dev/null)

if [ -z "$SERVICE_URL" ]; then
    echo -e "${RED}❌ Could not retrieve service URL${NC}"
    echo "Service might not be deployed yet. Check:"
    echo "  gcloud run services list"
    exit 1
fi

echo "Service URL: $SERVICE_URL"
echo ""

# Wait a few seconds for the service to be fully ready
echo "Waiting for service to be ready..."
sleep 5

# Run verification
if ! ./scripts/verify-deployment.sh "$SERVICE_URL"; then
    echo ""
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}⚠️  Some verification tests failed${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "The service is deployed, but some tests failed."
    echo "Check the logs for details:"
    echo "  gcloud run services logs tail sql-studio-backend"
    echo ""
    echo -e "${YELLOW}Do you want to continue anyway? (y/N) ${NC}"
    read -p "" -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Deployment verification failed."
        exit 1
    fi
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Success!
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🎉 Deployment Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "📋 Deployment Summary"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  🌐 Service URL:      $SERVICE_URL"
echo "  📦 Project ID:       $GCP_PROJECT_ID"
echo "  🌍 Region:           us-central1"
echo "  📅 Deployed:         $(date)"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🎯 Next Steps"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  1. Update frontend configuration:"
echo "     - Set API_URL to: $SERVICE_URL"
echo ""
echo "  2. Test the API manually:"
echo "     curl $SERVICE_URL/health"
echo ""
echo "  3. Configure custom domain (optional):"
echo "     gcloud run domain-mappings create --service=sql-studio-backend --domain=api.yourdomain.com"
echo ""
echo "  4. Set up monitoring alerts:"
echo "     - Error rate > 1%"
echo "     - Latency p95 > 1s"
echo "     - Request count approaching 2M/month"
echo ""
echo "  5. Monitor the service:"
echo "     gcloud run services logs tail sql-studio-backend"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🔗 Useful Links"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  - Cloud Run Console:  https://console.cloud.google.com/run?project=$GCP_PROJECT_ID"
echo "  - Logs:               https://console.cloud.google.com/logs?project=$GCP_PROJECT_ID"
echo "  - Monitoring:         https://console.cloud.google.com/monitoring?project=$GCP_PROJECT_ID"
echo "  - Error Reporting:    https://console.cloud.google.com/errors?project=$GCP_PROJECT_ID"
echo "  - Billing:            https://console.cloud.google.com/billing?project=$GCP_PROJECT_ID"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "💡 Useful Commands"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "  View logs:        make prod-logs"
echo "  Check status:     make prod-status"
echo "  Check costs:      make check-costs"
echo "  Verify again:     make verify-prod SERVICE_URL=$SERVICE_URL"
echo ""
echo "  Rollback:         gcloud run services update-traffic sql-studio-backend \\"
echo "                      --to-revisions=PREVIOUS_REVISION=100"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "${GREEN}✨ Happy shipping! ✨${NC}"
echo ""
