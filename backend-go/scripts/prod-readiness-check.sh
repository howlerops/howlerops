#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Production Readiness Check for SQL Studio Backend${NC}"
echo "====================================================="
echo ""

FAILED=0
WARNINGS=0

# Helper function for check results
check_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
    else
        echo -e "${RED}❌ $2${NC}"
        FAILED=1
    fi
}

warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    WARNINGS=$((WARNINGS + 1))
}

# 1. Environment Variables Check
echo -e "${BLUE}1. Environment Variables${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

required_vars=("TURSO_URL" "TURSO_AUTH_TOKEN" "JWT_SECRET" "RESEND_API_KEY" "GCP_PROJECT_ID")

for var in "${required_vars[@]}"; do
    if [ -n "${!var}" ]; then
        check_result 0 "$var is set"
    else
        check_result 1 "$var is set"
        echo -e "  ${YELLOW}Set with: export $var=your-value${NC}"
    fi
done

# Check optional but recommended vars
if [ -n "$RESEND_FROM_EMAIL" ]; then
    check_result 0 "RESEND_FROM_EMAIL is set"
else
    warning "RESEND_FROM_EMAIL not set (will use default)"
fi

if [ -n "$ALLOWED_ORIGINS" ]; then
    check_result 0 "ALLOWED_ORIGINS is set"
else
    warning "ALLOWED_ORIGINS not set (will allow all origins)"
fi

echo ""

# 2. Security Checks
echo -e "${BLUE}2. Security Configuration${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check JWT_SECRET strength
if [ -n "$JWT_SECRET" ]; then
    JWT_LENGTH=${#JWT_SECRET}
    if [ $JWT_LENGTH -ge 32 ]; then
        check_result 0 "JWT_SECRET is strong enough ($JWT_LENGTH chars)"
    else
        check_result 1 "JWT_SECRET is strong enough ($JWT_LENGTH chars, need 32+)"
        echo -e "  ${YELLOW}Generate a strong secret: openssl rand -base64 32${NC}"
    fi

    # Check if JWT_SECRET looks random (not a simple string)
    if echo "$JWT_SECRET" | grep -qE '[A-Za-z0-9+/]{32,}'; then
        check_result 0 "JWT_SECRET appears cryptographically random"
    else
        warning "JWT_SECRET might not be cryptographically random"
        echo -e "  ${YELLOW}Use: openssl rand -base64 32${NC}"
    fi
fi

# Check TURSO_URL format
if [ -n "$TURSO_URL" ]; then
    if [[ $TURSO_URL =~ ^libsql:// ]]; then
        check_result 0 "TURSO_URL has correct format (libsql://)"
    else
        warning "TURSO_URL should start with 'libsql://' for production"
        echo "  Current: $TURSO_URL"
    fi
fi

# Check RESEND_API_KEY format
if [ -n "$RESEND_API_KEY" ]; then
    if [[ $RESEND_API_KEY =~ ^re_ ]]; then
        check_result 0 "RESEND_API_KEY has correct format (re_...)"
    else
        warning "RESEND_API_KEY should start with 're_'"
        echo "  Current: ${RESEND_API_KEY:0:10}..."
    fi
fi

echo ""

# 3. Code Quality Checks
echo -e "${BLUE}3. Code Quality${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check if go.mod exists
if [ -f "go.mod" ]; then
    check_result 0 "go.mod exists"
else
    check_result 1 "go.mod exists"
fi

# Check Go version
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    check_result 0 "Go is installed ($GO_VERSION)"
else
    check_result 1 "Go is installed"
fi

# Run go fmt check
echo -n "Checking code formatting... "
UNFORMATTED=$(gofmt -l . 2>/dev/null | grep -v vendor || echo "")
if [ -z "$UNFORMATTED" ]; then
    echo -e "${GREEN}✅ Code is formatted${NC}"
else
    echo -e "${YELLOW}⚠️  Some files need formatting${NC}"
    echo "$UNFORMATTED" | sed 's/^/  /'
    echo -e "  ${YELLOW}Run: gofmt -w .${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

# Run go vet
echo -n "Running go vet... "
if go vet ./... &>/dev/null; then
    echo -e "${GREEN}✅ go vet passed${NC}"
else
    echo -e "${RED}❌ go vet found issues${NC}"
    go vet ./... 2>&1 | head -n 10 | sed 's/^/  /'
    FAILED=1
fi

echo ""

# 4. Tests
echo -e "${BLUE}4. Tests${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Run tests
echo -n "Running unit tests... "
if go test ./... -short -timeout=30s &>/dev/null; then
    echo -e "${GREEN}✅ Unit tests pass${NC}"
else
    echo -e "${RED}❌ Unit tests fail${NC}"
    echo "Run: go test ./... -short -v"
    FAILED=1
fi

# Run race detector (optional but recommended)
echo -n "Checking for race conditions... "
if go test -race ./... -short -timeout=30s &>/dev/null; then
    echo -e "${GREEN}✅ No race conditions detected${NC}"
else
    echo -e "${YELLOW}⚠️  Potential race conditions detected${NC}"
    echo -e "  ${YELLOW}Run: go test -race ./... -v${NC}"
    WARNINGS=$((WARNINGS + 1))
fi

echo ""

# 5. Build Check
echo -e "${BLUE}5. Build${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo -n "Building application... "
if go build -o /tmp/sql-studio-backend-test cmd/server/main.go 2>/dev/null; then
    echo -e "${GREEN}✅ Build succeeds${NC}"
    rm -f /tmp/sql-studio-backend-test
else
    echo -e "${RED}❌ Build fails${NC}"
    echo "Run: go build -v cmd/server/main.go"
    FAILED=1
fi

# Check binary size (if build succeeded)
if [ -f "/tmp/sql-studio-backend-test" ]; then
    SIZE=$(ls -lh /tmp/sql-studio-backend-test | awk '{print $5}')
    echo "Binary size: $SIZE"
    rm -f /tmp/sql-studio-backend-test
fi

echo ""

# 6. Dependencies
echo -e "${BLUE}6. Dependencies${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check for go.sum
if [ -f "go.sum" ]; then
    check_result 0 "go.sum exists"
else
    check_result 1 "go.sum exists"
    echo -e "  ${YELLOW}Run: go mod tidy${NC}"
fi

# Check for outdated dependencies
echo -n "Checking for dependency updates... "
if command -v go &> /dev/null; then
    UPDATES=$(go list -u -m -json all 2>/dev/null | jq -r 'select(.Update) | .Path' | wc -l | tr -d ' ')
    if [ "$UPDATES" -gt 0 ]; then
        echo -e "${YELLOW}⚠️  $UPDATES dependencies have updates available${NC}"
        echo -e "  ${YELLOW}Run: go list -u -m all${NC}"
        WARNINGS=$((WARNINGS + 1))
    else
        echo -e "${GREEN}✅ Dependencies are up to date${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Cannot check (jq not installed)${NC}"
fi

echo ""

# 7. Docker/Container Checks
echo -e "${BLUE}7. Container Configuration${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check Dockerfile exists
if [ -f "Dockerfile" ]; then
    check_result 0 "Dockerfile exists"

    # Check for multi-stage build
    if grep -q "FROM.*AS" Dockerfile; then
        check_result 0 "Dockerfile uses multi-stage build"
    else
        warning "Dockerfile should use multi-stage build for smaller images"
    fi

    # Check for non-root user
    if grep -q "USER" Dockerfile; then
        check_result 0 "Dockerfile runs as non-root user"
    else
        warning "Dockerfile should run as non-root user"
    fi
else
    check_result 1 "Dockerfile exists"
fi

# Check cloudbuild.yaml
if [ -f "cloudbuild.yaml" ]; then
    check_result 0 "cloudbuild.yaml exists"
else
    warning "cloudbuild.yaml not found (needed for GCP Cloud Build)"
fi

# Check cloudrun.yaml
if [ -f "cloudrun.yaml" ]; then
    check_result 0 "cloudrun.yaml exists"
else
    warning "cloudrun.yaml not found (optional)"
fi

echo ""

# 8. GCP Tools
echo -e "${BLUE}8. GCP Tools & Authentication${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check gcloud
if command -v gcloud &> /dev/null; then
    GCLOUD_VERSION=$(gcloud version --format="value(core)" 2>/dev/null)
    check_result 0 "gcloud CLI installed ($GCLOUD_VERSION)"
else
    check_result 1 "gcloud CLI installed"
    echo -e "  ${YELLOW}Install: https://cloud.google.com/sdk/install${NC}"
fi

# Check gcloud authentication
if command -v gcloud &> /dev/null; then
    if gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | grep -q "@"; then
        ACCOUNT=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -n 1)
        check_result 0 "Authenticated with gcloud ($ACCOUNT)"
    else
        check_result 1 "Authenticated with gcloud"
        echo -e "  ${YELLOW}Run: gcloud auth login${NC}"
    fi

    # Check project is set
    CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
    if [ -n "$CURRENT_PROJECT" ] && [ "$CURRENT_PROJECT" != "(unset)" ]; then
        if [ "$CURRENT_PROJECT" = "$GCP_PROJECT_ID" ]; then
            check_result 0 "GCP project set correctly ($CURRENT_PROJECT)"
        else
            warning "GCP project mismatch: gcloud=$CURRENT_PROJECT, env=$GCP_PROJECT_ID"
        fi
    else
        warning "GCP project not set in gcloud config"
        echo -e "  ${YELLOW}Run: gcloud config set project $GCP_PROJECT_ID${NC}"
    fi
fi

echo ""

# 9. Configuration Files
echo -e "${BLUE}9. Configuration Files${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check for config.yaml
if [ -f "configs/config.yaml" ]; then
    check_result 0 "configs/config.yaml exists"
else
    warning "configs/config.yaml not found"
fi

# Check for .env files (should NOT be in git)
if [ -f ".env" ]; then
    warning ".env file exists (should not be committed to git)"
    if git ls-files --error-unmatch .env &>/dev/null; then
        echo -e "${RED}  ❌ .env is tracked by git! Remove it immediately!${NC}"
        echo -e "  ${YELLOW}Run: git rm --cached .env && echo '.env' >> .gitignore${NC}"
        FAILED=1
    fi
fi

# Check .gitignore
if [ -f ".gitignore" ]; then
    check_result 0 ".gitignore exists"
    if grep -q "\.env" .gitignore; then
        check_result 0 ".gitignore includes .env"
    else
        warning ".gitignore should include .env"
    fi
else
    warning ".gitignore not found"
fi

echo ""

# 10. Database Connectivity
echo -e "${BLUE}10. Database Connectivity${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ -n "$TURSO_URL" ] && [ -n "$TURSO_AUTH_TOKEN" ]; then
    # Try to connect to Turso (this is a basic check)
    echo -n "Testing Turso connection... "
    # We can't easily test this without running the app, so we'll skip for now
    echo -e "${YELLOW}⚠️  Manual verification needed${NC}"
    echo -e "  ${YELLOW}Test by running the app locally: make dev${NC}"
else
    warning "Cannot test database connection (TURSO_URL or TURSO_AUTH_TOKEN not set)"
fi

echo ""

# Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

if [ $FAILED -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ All Checks Passed!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "🚀 Ready for production deployment!"
    echo ""
    echo "Next steps:"
    echo "  1. Review PRODUCTION_CHECKLIST.md"
    echo "  2. Run: ./scripts/deploy-full.sh"
    echo "  3. Or step-by-step:"
    echo "     - ./scripts/setup-secrets.sh"
    echo "     - ./scripts/deploy-cloudrun.sh"
    echo "     - ./scripts/verify-deployment.sh SERVICE_URL"
    echo ""
    exit 0
elif [ $FAILED -eq 0 ] && [ $WARNINGS -gt 0 ]; then
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}⚠️  Passed with $WARNINGS warning(s)${NC}"
    echo -e "${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "You can proceed with deployment, but consider addressing the warnings."
    echo ""
    echo "Next steps:"
    echo "  1. Review warnings above"
    echo "  2. (Optional) Fix warnings"
    echo "  3. Deploy: ./scripts/deploy-full.sh"
    echo ""
    exit 0
else
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ $FAILED Critical Issue(s) Found${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "Please fix the issues above before deploying to production."
    echo ""
    echo "Common fixes:"
    echo "  - Set environment variables: source .env.production"
    echo "  - Generate JWT secret: export JWT_SECRET=\$(openssl rand -base64 32)"
    echo "  - Install gcloud: https://cloud.google.com/sdk/install"
    echo "  - Authenticate: gcloud auth login"
    echo "  - Set project: gcloud config set project $GCP_PROJECT_ID"
    echo "  - Fix code issues: go vet ./..."
    echo "  - Fix tests: go test ./... -v"
    echo ""
    exit 1
fi
