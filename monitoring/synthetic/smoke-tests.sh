#!/bin/bash

# SQL Studio Smoke Tests
# Post-deployment validation script
# Tests critical user flows to ensure the system is functioning

set -e  # Exit on any error

# Configuration
BASE_URL="${BASE_URL:-https://sqlstudio.example.com}"
API_URL="${API_URL:-https://api.sqlstudio.example.com}"
TIMEOUT=10

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Function to print test result
print_result() {
    local test_name=$1
    local result=$2

    TESTS_RUN=$((TESTS_RUN + 1))

    if [ "$result" = "PASS" ]; then
        echo -e "${GREEN}✓${NC} $test_name"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${RED}✗${NC} $test_name"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
}

# Function to make HTTP request and check status code
check_http() {
    local url=$1
    local expected_status=${2:-200}
    local method=${3:-GET}

    local status_code=$(curl -s -o /dev/null -w "%{http_code}" \
        -X "$method" \
        --max-time $TIMEOUT \
        "$url")

    if [ "$status_code" = "$expected_status" ]; then
        return 0
    else
        echo "    Expected $expected_status, got $status_code" >&2
        return 1
    fi
}

echo "========================================="
echo "SQL Studio Smoke Tests"
echo "========================================="
echo "Base URL: $BASE_URL"
echo "API URL: $API_URL"
echo ""

# ============================================================================
# Test 1: Homepage loads
# ============================================================================
echo "Testing homepage..."
if check_http "$BASE_URL"; then
    print_result "Homepage loads (200 OK)" "PASS"
else
    print_result "Homepage loads (200 OK)" "FAIL"
fi

# ============================================================================
# Test 2: API health endpoint
# ============================================================================
echo "Testing API health..."
if check_http "$API_URL/health"; then
    print_result "API health endpoint responds" "PASS"
else
    print_result "API health endpoint responds" "FAIL"
fi

# ============================================================================
# Test 3: API readiness endpoint
# ============================================================================
echo "Testing API readiness..."
if check_http "$API_URL/health/ready"; then
    print_result "API readiness check passes" "PASS"
else
    print_result "API readiness check passes" "FAIL"
fi

# ============================================================================
# Test 4: API liveness endpoint
# ============================================================================
echo "Testing API liveness..."
if check_http "$API_URL/health/live"; then
    print_result "API liveness check passes" "PASS"
else
    print_result "API liveness check passes" "FAIL"
fi

# ============================================================================
# Test 5: Metrics endpoint (should be accessible)
# ============================================================================
echo "Testing metrics endpoint..."
if check_http "$API_URL/metrics"; then
    print_result "Metrics endpoint accessible" "PASS"
else
    print_result "Metrics endpoint accessible" "FAIL"
fi

# ============================================================================
# Test 6: API responds with JSON
# ============================================================================
echo "Testing API JSON response..."
response=$(curl -s --max-time $TIMEOUT "$API_URL/health")
if echo "$response" | jq . >/dev/null 2>&1; then
    print_result "API returns valid JSON" "PASS"
else
    print_result "API returns valid JSON" "FAIL"
fi

# ============================================================================
# Test 7: Health check shows healthy status
# ============================================================================
echo "Testing health status..."
health_status=$(curl -s --max-time $TIMEOUT "$API_URL/health" | jq -r '.status')
if [ "$health_status" = "healthy" ] || [ "$health_status" = "degraded" ]; then
    print_result "System reports healthy/degraded status" "PASS"
else
    print_result "System reports healthy/degraded status" "FAIL"
    echo "    Got status: $health_status" >&2
fi

# ============================================================================
# Test 8: SSL certificate is valid
# ============================================================================
echo "Testing SSL certificate..."
if openssl s_client -connect "$(echo $BASE_URL | sed 's|https://||'):443" \
    -servername "$(echo $BASE_URL | sed 's|https://||')" \
    </dev/null 2>/dev/null | openssl x509 -noout -checkend 604800 >/dev/null; then
    print_result "SSL certificate is valid (>7 days)" "PASS"
else
    print_result "SSL certificate is valid (>7 days)" "FAIL"
fi

# ============================================================================
# Test 9: CORS headers present
# ============================================================================
echo "Testing CORS headers..."
cors_header=$(curl -s -I -H "Origin: https://example.com" "$API_URL/health" | grep -i "access-control-allow-origin")
if [ -n "$cors_header" ]; then
    print_result "CORS headers present" "PASS"
else
    print_result "CORS headers present" "FAIL"
fi

# ============================================================================
# Test 10: API response time is acceptable
# ============================================================================
echo "Testing API response time..."
response_time=$(curl -o /dev/null -s -w '%{time_total}' "$API_URL/health")
if (( $(echo "$response_time < 1.0" | bc -l) )); then
    print_result "API response time <1s ($response_time)" "PASS"
else
    print_result "API response time <1s ($response_time)" "FAIL"
fi

# ============================================================================
# Test 11: Database health check (if accessible)
# ============================================================================
echo "Testing database health..."
db_status=$(curl -s --max-time $TIMEOUT "$API_URL/health" | jq -r '.checks.database.status' 2>/dev/null)
if [ "$db_status" = "healthy" ]; then
    print_result "Database connection healthy" "PASS"
elif [ "$db_status" = "null" ] || [ -z "$db_status" ]; then
    print_result "Database health check (not available)" "SKIP"
    TESTS_RUN=$((TESTS_RUN - 1))  # Don't count skipped tests
else
    print_result "Database connection healthy" "FAIL"
    echo "    Got status: $db_status" >&2
fi

# ============================================================================
# Test 12: Critical API endpoints respond
# ============================================================================
echo "Testing critical endpoints..."

# These should return 401 (unauthorized) not 500 (server error)
auth_status=$(curl -s -o /dev/null -w "%{http_code}" --max-time $TIMEOUT "$API_URL/api/user/profile")
if [ "$auth_status" = "401" ] || [ "$auth_status" = "200" ]; then
    print_result "Auth-protected endpoint responds correctly" "PASS"
else
    print_result "Auth-protected endpoint responds correctly" "FAIL"
    echo "    Expected 401 or 200, got $auth_status" >&2
fi

# ============================================================================
# Test 13: Static assets load
# ============================================================================
echo "Testing static assets..."
if check_http "$BASE_URL/favicon.ico"; then
    print_result "Static assets accessible" "PASS"
else
    print_result "Static assets accessible" "FAIL"
fi

# ============================================================================
# Summary
# ============================================================================
echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo "Tests Run:    $TESTS_RUN"
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
else
    echo "Tests Failed: $TESTS_FAILED"
fi
echo "========================================="

# Exit with error if any tests failed
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "${RED}SMOKE TESTS FAILED${NC}"
    exit 1
else
    echo -e "${GREEN}ALL SMOKE TESTS PASSED${NC}"
    exit 0
fi
