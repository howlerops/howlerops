#!/bin/bash
# ============================================================================
# Production Verification Script for SQL Studio Backend
# ============================================================================
# This script performs comprehensive checks to verify production deployment
# readiness and health. It validates environment variables, tests connectivity,
# runs smoke tests, and generates a deployment report.
#
# Usage:
#   ./scripts/verify-production.sh [OPTIONS]
#
# Options:
#   --env-file FILE       Path to environment file (default: .env.production)
#   --service-url URL     Production service URL for smoke tests
#   --skip-smoke-tests    Skip smoke tests (only check configuration)
#   --output-report FILE  Save report to file (default: stdout)
#   --strict              Exit with error on any warning (not just failures)
#   --help                Show this help message
# ============================================================================

set -euo pipefail

# ============================================================================
# Configuration
# ============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default values
ENV_FILE="${ENV_FILE:-.env.production}"
SERVICE_URL="${SERVICE_URL:-}"
SKIP_SMOKE_TESTS=false
OUTPUT_REPORT=""
STRICT_MODE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Counters
PASSED=0
FAILED=0
WARNINGS=0
SKIPPED=0

# Report content
REPORT=""

# ============================================================================
# Helper Functions
# ============================================================================

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
    REPORT="${REPORT}\n[INFO] $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
    REPORT="${REPORT}\n[PASS] $1"
    PASSED=$((PASSED + 1))
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
    REPORT="${REPORT}\n[FAIL] $1"
    FAILED=$((FAILED + 1))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    REPORT="${REPORT}\n[WARN] $1"
    WARNINGS=$((WARNINGS + 1))
}

log_skip() {
    echo -e "${CYAN}[SKIP]${NC} $1"
    REPORT="${REPORT}\n[SKIP] $1"
    SKIPPED=$((SKIPPED + 1))
}

log_section() {
    echo ""
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${MAGENTA}$1${NC}"
    echo -e "${BOLD}${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    REPORT="${REPORT}\n\n========== $1 ==========\n"
}

show_help() {
    grep '^#' "$0" | grep -v '#!/bin/bash' | head -n 20 | sed 's/^# //g'
}

check_command() {
    if command -v "$1" &> /dev/null; then
        return 0
    else
        return 1
    fi
}

load_env_file() {
    if [[ -f "$ENV_FILE" ]]; then
        log_info "Loading environment from: $ENV_FILE"
        set -a
        source "$ENV_FILE"
        set +a
        log_success "Environment file loaded"
    else
        log_warn "Environment file not found: $ENV_FILE"
    fi
}

# ============================================================================
# Environment Variable Checks
# ============================================================================

check_environment_variables() {
    log_section "1. ENVIRONMENT VARIABLES"

    # Required variables
    local required_vars=(
        "TURSO_URL"
        "TURSO_AUTH_TOKEN"
        "JWT_SECRET"
        "ENVIRONMENT"
    )

    for var in "${required_vars[@]}"; do
        if [[ -n "${!var:-}" ]]; then
            # Don't print sensitive values
            if [[ "$var" == *"TOKEN"* ]] || [[ "$var" == *"SECRET"* ]] || [[ "$var" == *"KEY"* ]]; then
                log_success "$var is set (length: ${#!var})"
            else
                log_success "$var is set: ${!var}"
            fi
        else
            log_error "$var is not set"
        fi
    done

    # Check JWT_SECRET strength
    if [[ -n "${JWT_SECRET:-}" ]]; then
        local jwt_length=${#JWT_SECRET}
        if [[ $jwt_length -ge 64 ]]; then
            log_success "JWT_SECRET is strong ($jwt_length characters)"
        elif [[ $jwt_length -ge 32 ]]; then
            log_warn "JWT_SECRET is acceptable but could be stronger ($jwt_length characters, recommend 64+)"
        else
            log_error "JWT_SECRET is too weak ($jwt_length characters, minimum 32 required)"
        fi

        # Check if it looks random
        if echo "$JWT_SECRET" | grep -qE '^[A-Za-z0-9+/=]{32,}$'; then
            log_success "JWT_SECRET appears to be properly encoded"
        else
            log_warn "JWT_SECRET may not be cryptographically random"
        fi
    fi

    # Optional but recommended
    local optional_vars=(
        "RESEND_API_KEY"
        "RESEND_FROM_EMAIL"
        "CORS_ORIGINS"
        "RATE_LIMIT_ENABLED"
        "LOG_LEVEL"
        "LOG_FORMAT"
    )

    for var in "${optional_vars[@]}"; do
        if [[ -n "${!var:-}" ]]; then
            if [[ "$var" == *"KEY"* ]]; then
                log_success "$var is set (optional)"
            else
                log_success "$var is set: ${!var} (optional)"
            fi
        else
            log_warn "$var is not set (optional but recommended)"
        fi
    done
}

# ============================================================================
# GCP Authentication Check
# ============================================================================

check_gcp_auth() {
    log_section "2. GCP AUTHENTICATION"

    if check_command gcloud; then
        log_success "gcloud CLI is installed"

        # Check authentication
        if gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | grep -q "@"; then
            local account=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null | head -n 1)
            log_success "Authenticated as: $account"
        else
            log_error "Not authenticated with gcloud"
            log_info "Run: gcloud auth login"
        fi

        # Check project
        local current_project=$(gcloud config get-value project 2>/dev/null || echo "")
        if [[ -n "$current_project" ]] && [[ "$current_project" != "(unset)" ]]; then
            log_success "GCP project set: $current_project"

            # Check if project exists and is accessible
            if gcloud projects describe "$current_project" &>/dev/null; then
                log_success "GCP project is accessible"
            else
                log_error "Cannot access GCP project: $current_project"
            fi
        else
            log_error "GCP project not set"
            log_info "Run: gcloud config set project PROJECT_ID"
        fi

        # Check required APIs
        if [[ -n "$current_project" ]]; then
            local required_apis=(
                "run.googleapis.com"
                "secretmanager.googleapis.com"
                "containerregistry.googleapis.com"
            )

            for api in "${required_apis[@]}"; do
                if gcloud services list --enabled --filter="name:$api" --format="value(name)" 2>/dev/null | grep -q "$api"; then
                    log_success "API enabled: $api"
                else
                    log_error "API not enabled: $api"
                    log_info "Enable with: gcloud services enable $api"
                fi
            done
        fi
    else
        log_error "gcloud CLI not installed"
        log_info "Install from: https://cloud.google.com/sdk/docs/install"
    fi
}

# ============================================================================
# Turso Database Check
# ============================================================================

check_turso_database() {
    log_section "3. TURSO DATABASE"

    if [[ -n "${TURSO_URL:-}" ]]; then
        # Check URL format
        if [[ "$TURSO_URL" =~ ^libsql:// ]]; then
            log_success "TURSO_URL has correct format"

            # Extract database name from URL
            local db_name=$(echo "$TURSO_URL" | sed -n 's|libsql://\([^-]*\).*|\1|p')
            if [[ -n "$db_name" ]]; then
                log_info "Database name: $db_name"
            fi
        else
            log_error "TURSO_URL should start with 'libsql://'"
        fi
    else
        log_error "TURSO_URL not set"
    fi

    if [[ -n "${TURSO_AUTH_TOKEN:-}" ]]; then
        log_success "TURSO_AUTH_TOKEN is set (length: ${#TURSO_AUTH_TOKEN})"

        # Basic token format check
        if [[ ${#TURSO_AUTH_TOKEN} -ge 20 ]]; then
            log_success "TURSO_AUTH_TOKEN appears valid"
        else
            log_warn "TURSO_AUTH_TOKEN seems short"
        fi
    else
        log_error "TURSO_AUTH_TOKEN not set"
    fi

    # Check Turso CLI if available
    if check_command turso; then
        log_info "Turso CLI is installed"

        # Check if logged in
        if turso auth status &>/dev/null; then
            log_success "Turso CLI authenticated"
        else
            log_warn "Turso CLI not authenticated"
            log_info "Run: turso auth login"
        fi
    else
        log_warn "Turso CLI not installed (optional)"
        log_info "Install from: https://docs.turso.tech/cli/installation"
    fi
}

# ============================================================================
# Email Service Check
# ============================================================================

check_email_service() {
    log_section "4. EMAIL SERVICE"

    if [[ -n "${RESEND_API_KEY:-}" ]]; then
        log_success "RESEND_API_KEY is set"

        # Check key format
        if [[ "$RESEND_API_KEY" =~ ^re_ ]]; then
            log_success "RESEND_API_KEY has correct format"
        else
            log_warn "RESEND_API_KEY should start with 're_'"
        fi
    else
        log_warn "RESEND_API_KEY not set (email features will be disabled)"
    fi

    if [[ -n "${RESEND_FROM_EMAIL:-}" ]]; then
        log_success "RESEND_FROM_EMAIL is set: $RESEND_FROM_EMAIL"

        # Check email format
        if [[ "$RESEND_FROM_EMAIL" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
            log_success "RESEND_FROM_EMAIL has valid format"
        else
            log_error "RESEND_FROM_EMAIL has invalid format"
        fi
    else
        log_warn "RESEND_FROM_EMAIL not set"
    fi
}

# ============================================================================
# Security Configuration Check
# ============================================================================

check_security_config() {
    log_section "5. SECURITY CONFIGURATION"

    # CORS
    if [[ "${CORS_ENABLED:-}" == "true" ]]; then
        log_success "CORS is enabled"

        if [[ -n "${CORS_ORIGINS:-}" ]]; then
            log_success "CORS_ORIGINS configured: $CORS_ORIGINS"

            # Check for wildcards in production
            if [[ "$CORS_ORIGINS" == "*" ]]; then
                log_error "CORS_ORIGINS should not use wildcard (*) in production"
            fi
        else
            log_warn "CORS_ORIGINS not set (will block all cross-origin requests)"
        fi
    else
        log_warn "CORS is disabled"
    fi

    # Rate Limiting
    if [[ "${RATE_LIMIT_ENABLED:-}" == "true" ]]; then
        log_success "Rate limiting is enabled"

        if [[ -n "${RATE_LIMIT_RPS:-}" ]]; then
            log_success "Rate limit: ${RATE_LIMIT_RPS} requests/second"
        fi
    else
        log_warn "Rate limiting is disabled (not recommended for production)"
    fi

    # HTTPS/TLS
    if [[ "${TLS_ENABLED:-}" == "true" ]]; then
        log_success "TLS is enabled"
    else
        log_warn "TLS is not enabled (Cloud Run provides TLS, so this may be ok)"
    fi

    # Bcrypt cost
    if [[ -n "${BCRYPT_COST:-}" ]]; then
        if [[ $BCRYPT_COST -ge 12 ]]; then
            log_success "Bcrypt cost is secure: $BCRYPT_COST"
        else
            log_warn "Bcrypt cost is low for production: $BCRYPT_COST (recommend 12+)"
        fi
    else
        log_warn "BCRYPT_COST not set (will use default)"
    fi
}

# ============================================================================
# Smoke Tests
# ============================================================================

run_smoke_tests() {
    log_section "6. SMOKE TESTS"

    if [[ -z "$SERVICE_URL" ]]; then
        log_skip "No service URL provided, skipping smoke tests"
        log_info "Provide --service-url to run smoke tests"
        return
    fi

    if [[ "$SKIP_SMOKE_TESTS" == "true" ]]; then
        log_skip "Smoke tests skipped by user request"
        return
    fi

    # Remove trailing slash
    SERVICE_URL="${SERVICE_URL%/}"

    log_info "Testing service at: $SERVICE_URL"

    # Test 1: Health check
    log_info "Testing health endpoint..."
    if curl -f -s -m 10 "$SERVICE_URL/health" > /dev/null; then
        log_success "Health endpoint is responsive"
    else
        log_error "Health endpoint is not responding"
    fi

    # Test 2: Check response time
    log_info "Checking response time..."
    local response_time=$(curl -s -o /dev/null -w "%{time_total}" "$SERVICE_URL/health")
    local response_ms=$(echo "$response_time * 1000" | bc 2>/dev/null || echo "N/A")

    if [[ "$response_ms" != "N/A" ]]; then
        if (( $(echo "$response_ms < 1000" | bc -l) )); then
            log_success "Response time is good: ${response_ms}ms"
        else
            log_warn "Response time is slow: ${response_ms}ms"
        fi
    fi

    # Test 3: API endpoint
    log_info "Testing API endpoint..."
    local api_response=$(curl -s -w "\n%{http_code}" "$SERVICE_URL/api/v1/health" 2>/dev/null || echo "000")
    local http_code=$(echo "$api_response" | tail -n 1)

    if [[ "$http_code" == "200" ]]; then
        log_success "API endpoint returns 200 OK"
    elif [[ "$http_code" == "404" ]]; then
        log_warn "API endpoint returns 404 (endpoint may not exist)"
    else
        log_error "API endpoint returns unexpected status: $http_code"
    fi

    # Test 4: Authentication endpoint
    log_info "Testing authentication endpoint..."
    local auth_response=$(curl -s -X POST "$SERVICE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"email":"test@example.com","password":"test"}' \
        -w "\n%{http_code}" 2>/dev/null || echo "000")
    local auth_code=$(echo "$auth_response" | tail -n 1)

    if [[ "$auth_code" == "401" ]] || [[ "$auth_code" == "400" ]]; then
        log_success "Auth endpoint properly rejects invalid credentials"
    elif [[ "$auth_code" == "200" ]]; then
        log_error "Auth endpoint accepts invalid credentials (security issue!)"
    else
        log_warn "Auth endpoint returns unexpected status: $auth_code"
    fi

    # Test 5: CORS headers
    log_info "Testing CORS configuration..."
    local cors_response=$(curl -s -I -X OPTIONS "$SERVICE_URL/api/auth/login" \
        -H "Origin: https://example.com" \
        -H "Access-Control-Request-Method: POST" 2>/dev/null || echo "")

    if echo "$cors_response" | grep -qi "Access-Control-Allow-Origin"; then
        log_success "CORS headers are present"
    else
        log_warn "CORS headers not detected"
    fi
}

# ============================================================================
# Generate Deployment Report
# ============================================================================

generate_report() {
    log_section "DEPLOYMENT VERIFICATION REPORT"

    echo ""
    echo -e "${BOLD}Summary:${NC}"
    echo -e "${GREEN}Passed:${NC} $PASSED"
    echo -e "${RED}Failed:${NC} $FAILED"
    echo -e "${YELLOW}Warnings:${NC} $WARNINGS"
    echo -e "${CYAN}Skipped:${NC} $SKIPPED"
    echo ""

    local total=$((PASSED + FAILED + WARNINGS + SKIPPED))
    local success_rate=0
    if [[ $total -gt 0 ]]; then
        success_rate=$((PASSED * 100 / total))
    fi

    echo -e "${BOLD}Success Rate:${NC} ${success_rate}%"
    echo ""

    # Determine overall status
    if [[ $FAILED -eq 0 ]]; then
        if [[ $WARNINGS -eq 0 ]]; then
            echo -e "${GREEN}${BOLD}✅ READY FOR PRODUCTION${NC}"
            echo "All checks passed successfully!"
        else
            if [[ "$STRICT_MODE" == "true" ]]; then
                echo -e "${YELLOW}${BOLD}⚠️ READY WITH WARNINGS (STRICT MODE)${NC}"
                echo "Some warnings detected. Fix them for strict compliance."
            else
                echo -e "${YELLOW}${BOLD}⚠️ READY WITH WARNINGS${NC}"
                echo "Deployment can proceed, but review warnings."
            fi
        fi
    else
        echo -e "${RED}${BOLD}❌ NOT READY FOR PRODUCTION${NC}"
        echo "Critical issues must be resolved before deployment."
    fi

    # Generate timestamp
    local timestamp=$(date -u +"%Y-%m-%d %H:%M:%S UTC")
    REPORT="${REPORT}\n\nGenerated: $timestamp"

    # Save report if requested
    if [[ -n "$OUTPUT_REPORT" ]]; then
        echo -e "$REPORT" > "$OUTPUT_REPORT"
        echo ""
        echo -e "${BLUE}Report saved to:${NC} $OUTPUT_REPORT"
    fi

    # Recommendations
    echo ""
    echo -e "${BOLD}Recommendations:${NC}"

    if [[ $FAILED -gt 0 ]]; then
        echo "1. Fix all failed checks before deploying"
        echo "2. Run this script again after fixes"
    fi

    if [[ $WARNINGS -gt 0 ]]; then
        echo "1. Review and address warnings for better security"
        echo "2. Consider enabling recommended optional features"
    fi

    if [[ -z "$SERVICE_URL" ]]; then
        echo "1. Deploy the service and run with --service-url for complete verification"
    fi

    echo ""
    echo -e "${BOLD}Next Steps:${NC}"
    echo "1. Review the deployment checklist: DEPLOYMENT_CHECKLIST.md"
    echo "2. Run production readiness check: ./scripts/prod-readiness-check.sh"
    echo "3. Deploy using: ./scripts/deploy-gcp.sh --project PROJECT_ID"
    echo "4. Verify deployment: ./scripts/verify-deployment.sh SERVICE_URL"
}

# ============================================================================
# Parse Command Line Arguments
# ============================================================================

while [[ $# -gt 0 ]]; do
    case $1 in
        --env-file)
            ENV_FILE="$2"
            shift 2
            ;;
        --service-url)
            SERVICE_URL="$2"
            shift 2
            ;;
        --skip-smoke-tests)
            SKIP_SMOKE_TESTS=true
            shift
            ;;
        --output-report)
            OUTPUT_REPORT="$2"
            shift 2
            ;;
        --strict)
            STRICT_MODE=true
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

# ============================================================================
# Main Execution
# ============================================================================

echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}${BLUE}SQL Studio Backend - Production Verification${NC}"
echo -e "${BOLD}${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

# Load environment file
load_env_file

# Run all checks
check_environment_variables
check_gcp_auth
check_turso_database
check_email_service
check_security_config
run_smoke_tests

# Generate report
generate_report

# Exit code
if [[ $FAILED -gt 0 ]]; then
    exit 1
elif [[ $WARNINGS -gt 0 ]] && [[ "$STRICT_MODE" == "true" ]]; then
    exit 1
else
    exit 0
fi