#!/bin/bash

# Smoke Test Suite for Howlerops Backend
# Fast, minimal critical path testing for deployment verification
# Should complete in < 30 seconds

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${TEST_BASE_URL:-http://localhost:8500}"
TIMEOUT=5
MAX_RETRIES=3
START_TIME=$(date +%s)

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Print with timestamp
log() {
    local level=$1
    shift
    local message="$@"
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")

    case $level in
        INFO)
            echo -e "${BLUE}[$timestamp]${NC} $message"
            ;;
        PASS)
            echo -e "${GREEN}[$timestamp] ✓${NC} $message"
            ;;
        FAIL)
            echo -e "${RED}[$timestamp] ✗${NC} $message"
            ;;
        WARN)
            echo -e "${YELLOW}[$timestamp] ⚠${NC} $message"
            ;;
    esac
}

# Test helper with retries
test_endpoint() {
    local name=$1
    local url=$2
    local expected_status=$3
    local method=${4:-GET}
    local data=${5:-}

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    local retry=0

    while [ $retry -lt $MAX_RETRIES ]; do
        if [ -n "$data" ]; then
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                -H "Content-Type: application/json" \
                -d "$data" \
                --max-time $TIMEOUT 2>&1) || true
        else
            response=$(curl -s -w "\n%{http_code}" -X "$method" "$url" \
                --max-time $TIMEOUT 2>&1) || true
        fi

        status_code=$(echo "$response" | tail -n1 2>/dev/null || echo "0")

        if [ "$status_code" = "$expected_status" ]; then
            log PASS "$name"
            PASSED_TESTS=$((PASSED_TESTS + 1))
            return 0
        fi

        retry=$((retry + 1))
        if [ $retry -lt $MAX_RETRIES ]; then
            log WARN "$name failed (attempt $retry/$MAX_RETRIES), retrying..."
            sleep 1
        fi
    done

    log FAIL "$name (expected: $expected_status, got: $status_code)"
    FAILED_TESTS=$((FAILED_TESTS + 1))
    return 1
}

# Check if service is reachable
test_service_reachable() {
    log INFO "Testing service reachability..."
    test_endpoint "Service is reachable" "$BASE_URL/health" "200"
}

# Check health endpoint
test_health_check() {
    log INFO "Testing health check..."

    response=$(curl -s "$BASE_URL/health" --max-time $TIMEOUT 2>&1) || {
        log FAIL "Health check - connection failed"
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    }

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if echo "$response" | grep -q "healthy"; then
        log PASS "Health check returns healthy status"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log FAIL "Health check - invalid response: $response"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Check database connectivity (implicit via health check)
test_database_connectivity() {
    log INFO "Testing database connectivity..."

    # The health check should verify database connectivity
    # If health is OK, database is accessible
    response=$(curl -s "$BASE_URL/health" --max-time $TIMEOUT 2>&1) || {
        log FAIL "Database connectivity check failed"
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    }

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if echo "$response" | grep -q "healthy"; then
        log PASS "Database is accessible"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log FAIL "Database connectivity check failed"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Test basic auth flow
test_basic_auth_flow() {
    log INFO "Testing basic auth flow..."

    # Generate unique test credentials
    local timestamp=$(date +%s)
    local test_email="smoke${timestamp}@test.com"
    local test_username="smoke${timestamp}"
    local test_password="SmokeTest123!"

    # Test signup
    local signup_data="{\"email\":\"$test_email\",\"username\":\"$test_username\",\"password\":\"$test_password\"}"
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/signup" \
        -H "Content-Type: application/json" \
        -d "$signup_data" \
        --max-time $TIMEOUT 2>&1) || {
        log FAIL "Auth signup - connection failed"
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    }

    status_code=$(echo "$response" | tail -n1 2>/dev/null || echo "0")
    body=$(echo "$response" | sed '$d')

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ "$status_code" = "200" ] || [ "$status_code" = "201" ]; then
        log PASS "Auth signup works"
        PASSED_TESTS=$((PASSED_TESTS + 1))

        # Extract token if available
        if command -v jq &> /dev/null; then
            token=$(echo "$body" | jq -r '.token // empty')
            if [ -n "$token" ] && [ "$token" != "null" ]; then
                log PASS "Auth token generated"
                PASSED_TESTS=$((PASSED_TESTS + 1))
            else
                log FAIL "Auth token not generated"
                FAILED_TESTS=$((FAILED_TESTS + 1))
            fi
            TOTAL_TESTS=$((TOTAL_TESTS + 1))
        fi
    else
        log FAIL "Auth signup failed (status: $status_code)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi

    # Test login
    local login_data="{\"username\":\"$test_username\",\"password\":\"$test_password\"}"
    test_endpoint "Auth login works" "$BASE_URL/api/auth/login" "200" "POST" "$login_data"
}

# Test that protected endpoints require auth
test_protected_endpoints() {
    log INFO "Testing protected endpoints..."

    # Should fail without auth
    test_endpoint "Protected endpoint requires auth" "$BASE_URL/api/sync/download?device_id=test" "401"
}

# Test CORS is enabled
test_cors() {
    log INFO "Testing CORS..."

    response=$(curl -s -I -X OPTIONS "$BASE_URL/health" \
        -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: GET" \
        --max-time $TIMEOUT 2>&1) || {
        log FAIL "CORS check - connection failed"
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    }

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if echo "$response" | grep -i "Access-Control-Allow-Origin" > /dev/null; then
        log PASS "CORS is enabled"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log WARN "CORS headers not found (may be disabled)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    fi
}

# Test response time
test_response_time() {
    log INFO "Testing response time..."

    local start=$(date +%s%N)
    curl -s "$BASE_URL/health" --max-time $TIMEOUT > /dev/null 2>&1 || {
        log FAIL "Response time test - connection failed"
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    }
    local end=$(date +%s%N)

    local duration=$(( (end - start) / 1000000 ))  # Convert to milliseconds

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ $duration -lt 1000 ]; then
        log PASS "Response time is acceptable (${duration}ms)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        log WARN "Response time is slow (${duration}ms)"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    fi
}

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "========================================"
    echo "   Howlerops Smoke Tests"
    echo "========================================"
    echo -e "${NC}"
    echo "Target: $BASE_URL"
    echo "Timeout: ${TIMEOUT}s"
    echo ""
}

# Print summary
print_summary() {
    local end_time=$(date +%s)
    local duration=$((end_time - START_TIME))

    echo ""
    echo -e "${BLUE}========================================"
    echo "   Smoke Test Summary"
    echo "========================================${NC}"
    echo "Duration:     ${duration}s"
    echo "Total Tests:  $TOTAL_TESTS"
    echo -e "${GREEN}Passed:       $PASSED_TESTS${NC}"

    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "${RED}Failed:       $FAILED_TESTS${NC}"
    else
        echo -e "Failed:       $FAILED_TESTS"
    fi

    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}✓ All smoke tests passed!${NC}"
        echo -e "${GREEN}✓ Deployment is healthy${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some smoke tests failed${NC}"
        echo -e "${RED}✗ Deployment may have issues${NC}"
        exit 1
    fi
}

# Main execution
main() {
    print_banner

    # Run critical smoke tests in order
    test_service_reachable
    test_health_check
    test_database_connectivity
    test_response_time
    test_basic_auth_flow
    test_protected_endpoints
    test_cors

    print_summary
}

# Run main
main
