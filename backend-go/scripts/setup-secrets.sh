#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔐 Setting up GCP Secrets for SQL Studio Backend${NC}"
echo "=================================================="

# Check required variables
echo -e "${BLUE}Checking required environment variables...${NC}"
required_vars=("GCP_PROJECT_ID" "TURSO_URL" "TURSO_AUTH_TOKEN" "RESEND_API_KEY" "JWT_SECRET")
missing_vars=()

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        missing_vars+=("$var")
    fi
done

if [ ${#missing_vars[@]} -gt 0 ]; then
    echo -e "${RED}❌ Error: The following environment variables are not set:${NC}"
    for var in "${missing_vars[@]}"; do
        echo -e "${RED}  - $var${NC}"
    done
    echo ""
    echo -e "${YELLOW}Please set all required environment variables:${NC}"
    echo "  export GCP_PROJECT_ID=your-project-id"
    echo "  export TURSO_URL=libsql://your-db.turso.io"
    echo "  export TURSO_AUTH_TOKEN=your-turso-token"
    echo "  export RESEND_API_KEY=re_your-resend-key"
    echo "  export JWT_SECRET=\$(openssl rand -base64 32)"
    echo ""
    echo -e "${YELLOW}Tip: You can copy .env.production.example to .env.production and source it:${NC}"
    echo "  cp .env.production.example .env.production"
    echo "  # Edit .env.production with your values"
    echo "  source .env.production"
    exit 1
fi

echo -e "${GREEN}✅ All required variables are set${NC}"

# Validate variable formats
echo ""
echo -e "${BLUE}Validating variable formats...${NC}"

# Validate TURSO_URL
if [[ ! $TURSO_URL =~ ^libsql:// ]]; then
    echo -e "${YELLOW}⚠️  Warning: TURSO_URL should start with 'libsql://' for production${NC}"
    echo "  Current value: $TURSO_URL"
fi

# Validate JWT_SECRET length
if [ ${#JWT_SECRET} -lt 32 ]; then
    echo -e "${RED}❌ Error: JWT_SECRET is too short (${#JWT_SECRET} characters)${NC}"
    echo "  Minimum length: 32 characters"
    echo "  Generate a strong secret: openssl rand -base64 32"
    exit 1
fi

# Validate RESEND_API_KEY format
if [[ ! $RESEND_API_KEY =~ ^re_ ]]; then
    echo -e "${YELLOW}⚠️  Warning: RESEND_API_KEY should start with 're_'${NC}"
    echo "  Current value: $RESEND_API_KEY"
fi

echo -e "${GREEN}✅ Variable formats look good${NC}"

# Set GCP project
echo ""
echo -e "${BLUE}Setting GCP project to: $GCP_PROJECT_ID${NC}"
gcloud config set project $GCP_PROJECT_ID

# Check if user is authenticated
if ! gcloud auth list --filter=status:ACTIVE --format="value(account)" | grep -q "@"; then
    echo -e "${RED}❌ Error: Not authenticated with gcloud${NC}"
    echo "Please run: gcloud auth login"
    exit 1
fi

# Enable required APIs
echo ""
echo -e "${BLUE}Enabling required GCP APIs...${NC}"
apis=(
    "cloudbuild.googleapis.com"
    "run.googleapis.com"
    "secretmanager.googleapis.com"
    "logging.googleapis.com"
    "monitoring.googleapis.com"
)

for api in "${apis[@]}"; do
    echo -n "  Enabling $api... "
    if gcloud services enable "$api" --quiet 2>/dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        echo -e "${RED}Failed to enable $api. Please check your permissions.${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✅ All APIs enabled${NC}"

# Create or update secrets
echo ""
echo -e "${BLUE}Creating/updating secrets in Secret Manager...${NC}"

create_or_update_secret() {
    local secret_name=$1
    local secret_value=$2

    if gcloud secrets describe "$secret_name" &>/dev/null; then
        echo -n "  Updating $secret_name... "
        if echo -n "$secret_value" | gcloud secrets versions add "$secret_name" --data-file=- --quiet 2>/dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
            echo -e "${RED}Failed to update $secret_name${NC}"
            return 1
        fi
    else
        echo -n "  Creating $secret_name... "
        if echo -n "$secret_value" | gcloud secrets create "$secret_name" --data-file=- --replication-policy="automatic" --quiet 2>/dev/null; then
            echo -e "${GREEN}✓${NC}"
        else
            echo -e "${RED}✗${NC}"
            echo -e "${RED}Failed to create $secret_name${NC}"
            return 1
        fi
    fi
}

# Create secrets
create_or_update_secret "turso-url" "$TURSO_URL"
create_or_update_secret "turso-auth-token" "$TURSO_AUTH_TOKEN"
create_or_update_secret "resend-api-key" "$RESEND_API_KEY"
create_or_update_secret "jwt-secret" "$JWT_SECRET"

echo -e "${GREEN}✅ All secrets created/updated${NC}"

# Get Cloud Run service account
echo ""
echo -e "${BLUE}Granting Cloud Run service account access to secrets...${NC}"

PROJECT_NUMBER=$(gcloud projects describe "$GCP_PROJECT_ID" --format="value(projectNumber)")
SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

echo "  Service Account: $SERVICE_ACCOUNT"

# Grant access to each secret
for secret in turso-url turso-auth-token resend-api-key jwt-secret; do
    echo -n "  Granting access to $secret... "
    if gcloud secrets add-iam-policy-binding "$secret" \
        --member="serviceAccount:${SERVICE_ACCOUNT}" \
        --role="roles/secretmanager.secretAccessor" \
        --quiet &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        echo -e "${YELLOW}⚠️  Warning: Could not grant access to $secret${NC}"
        echo "  This might be okay if it was already granted"
    fi
done

echo -e "${GREEN}✅ IAM permissions configured${NC}"

# Verify secrets
echo ""
echo -e "${BLUE}Verifying secrets are accessible...${NC}"

for secret in turso-url turso-auth-token resend-api-key jwt-secret; do
    echo -n "  Verifying $secret... "
    if gcloud secrets versions access latest --secret="$secret" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗${NC}"
        echo -e "${RED}Failed to access $secret${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✅ All secrets verified and accessible${NC}"

# Summary
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}✅ GCP Secrets Setup Complete!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "📋 Summary:"
echo "  Project ID:      $GCP_PROJECT_ID"
echo "  Service Account: $SERVICE_ACCOUNT"
echo "  Secrets created: turso-url, turso-auth-token, resend-api-key, jwt-secret"
echo ""
echo "🎯 Next steps:"
echo "  1. Deploy the service: ./scripts/deploy-cloudrun.sh"
echo "  2. Or run full deployment: ./scripts/deploy-full.sh"
echo ""
echo "💡 Tips:"
echo "  - View secrets: gcloud secrets list"
echo "  - View secret value: gcloud secrets versions access latest --secret=SECRET_NAME"
echo "  - Update secret: ./scripts/setup-secrets.sh (run this script again)"
echo ""
