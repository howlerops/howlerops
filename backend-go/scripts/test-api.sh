#!/bin/bash

# API Testing Script for SQL Studio Backend
# Tests all endpoints with color-coded output

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${TEST_BASE_URL:-http://localhost:8500}"
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Test user credentials
TEST_EMAIL="apitest$(date +%s)@example.com"
TEST_USERNAME="apitest$(date +%s)"
TEST_PASSWORD="TestPassword123!"
TOKEN=""
REFRESH_TOKEN=""

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "========================================"
    echo "   SQL Studio API Test Suite"
    echo "========================================"
    echo -e "${NC}"
    echo "Base URL: $BASE_URL"
    echo "Test User: $TEST_USERNAME"
    echo ""
}

# Print section header
print_section() {
    echo -e "\n${BLUE}[TEST SECTION]${NC} $1"
    echo "----------------------------------------"
}

# Test result helper
test_result() {
    local test_name=$1
    local expected=$2
    local actual=$3
    local response=$4

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    if [ "$expected" -eq "$actual" ]; then
        echo -e "${GREEN}✓ PASS${NC} $test_name"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        return 0
    else
        echo -e "${RED}✗ FAIL${NC} $test_name"
        echo -e "  Expected: $expected, Got: $actual"
        if [ -n "$response" ]; then
            echo -e "  Response: $response"
        fi
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Test health endpoint
test_health() {
    print_section "Health Check Tests"

    # Test 1: Health endpoint should return 200
    response=$(curl -s -w "\n%{http_code}" "$BASE_URL/health")
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    test_result "Health endpoint returns 200" 200 "$status_code" "$body"

    # Validate JSON response
    if echo "$body" | jq -e '.status == "healthy"' > /dev/null 2>&1; then
        echo -e "${GREEN}✓ PASS${NC} Health status is 'healthy'"
        PASSED_TESTS=$((PASSED_TESTS + 1))
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC} Health status is not 'healthy'"
        echo -e "  Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        TOTAL_TESTS=$((TOTAL_TESTS + 1))
    fi
}

# Test auth signup
test_signup() {
    print_section "Auth Signup Tests"

    # Test 1: Valid signup
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}")

    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if test_result "Signup with valid credentials" 200 "$status_code" "$body" || \
       test_result "Signup with valid credentials" 201 "$status_code" "$body"; then
        # Extract token from response
        TOKEN=$(echo "$body" | jq -r '.token // empty')
        REFRESH_TOKEN=$(echo "$body" | jq -r '.refresh_token // empty')

        if [ -n "$TOKEN" ]; then
            echo -e "${GREEN}  Token received: ${TOKEN:0:20}...${NC}"
        fi
    fi

    # Test 2: Duplicate signup should fail
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"$TEST_EMAIL\",\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}")

    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$status_code" -eq 400 ] || [ "$status_code" -eq 409 ]; then
        echo -e "${GREEN}✓ PASS${NC} Duplicate signup correctly rejected"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC} Duplicate signup should fail with 400 or 409"
        echo -e "  Expected: 400 or 409, Got: $status_code"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    # Test 3: Invalid email format
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"invalid-email\",\"username\":\"user123\",\"password\":\"$TEST_PASSWORD\"}")

    status_code=$(echo "$response" | tail -n1)
    test_result "Invalid email format rejected" 400 "$status_code"

    # Test 4: Weak password
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/signup" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"test@example.com\",\"username\":\"user123\",\"password\":\"weak\"}")

    status_code=$(echo "$response" | tail -n1)
    test_result "Weak password rejected" 400 "$status_code"
}

# Test auth login
test_login() {
    print_section "Auth Login Tests"

    # Test 1: Valid login
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$TEST_USERNAME\",\"password\":\"$TEST_PASSWORD\"}")

    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if test_result "Login with valid credentials" 200 "$status_code" "$body"; then
        # Extract token from response
        TOKEN=$(echo "$body" | jq -r '.token // empty')
        REFRESH_TOKEN=$(echo "$body" | jq -r '.refresh_token // empty')

        if [ -n "$TOKEN" ]; then
            echo -e "${GREEN}  New token received${NC}"
        fi
    fi

    # Test 2: Invalid password
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$TEST_USERNAME\",\"password\":\"WrongPassword123!\"}")

    status_code=$(echo "$response" | tail -n1)
    test_result "Login with invalid password rejected" 401 "$status_code"

    # Test 3: Non-existent user
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"nonexistent\",\"password\":\"$TEST_PASSWORD\"}")

    status_code=$(echo "$response" | tail -n1)
    test_result "Login with non-existent user rejected" 401 "$status_code"

    # Test 4: Empty credentials
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{}")

    status_code=$(echo "$response" | tail -n1)
    test_result "Login with empty credentials rejected" 400 "$status_code"
}

# Test token refresh
test_refresh() {
    print_section "Token Refresh Tests"

    if [ -z "$REFRESH_TOKEN" ]; then
        echo -e "${YELLOW}⚠ SKIP${NC} No refresh token available"
        return
    fi

    # Test 1: Valid refresh token
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}")

    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    test_result "Refresh with valid token" 200 "$status_code" "$body"

    # Test 2: Invalid refresh token
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/refresh" \
        -H "Content-Type: application/json" \
        -d "{\"refresh_token\":\"invalid-token\"}")

    status_code=$(echo "$response" | tail -n1)
    test_result "Refresh with invalid token rejected" 401 "$status_code"
}

# Test protected endpoints
test_protected_endpoints() {
    print_section "Protected Endpoint Tests"

    if [ -z "$TOKEN" ]; then
        echo -e "${YELLOW}⚠ SKIP${NC} No token available"
        return
    fi

    # Test 1: Access without token
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/auth/profile")
    status_code=$(echo "$response" | tail -n1)
    test_result "Protected endpoint without token rejected" 401 "$status_code"

    # Test 2: Access with valid token
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/auth/profile" \
        -H "Authorization: Bearer $TOKEN")
    status_code=$(echo "$response" | tail -n1)
    test_result "Protected endpoint with valid token" 200 "$status_code"

    # Test 3: Access with invalid token
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/auth/profile" \
        -H "Authorization: Bearer invalid-token")
    status_code=$(echo "$response" | tail -n1)
    test_result "Protected endpoint with invalid token rejected" 401 "$status_code"
}

# Test sync endpoints
test_sync() {
    print_section "Sync Endpoint Tests"

    if [ -z "$TOKEN" ]; then
        echo -e "${YELLOW}⚠ SKIP${NC} No token available"
        return
    fi

    # Test 1: Upload without auth
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/sync/upload" \
        -H "Content-Type: application/json" \
        -d "{}")
    status_code=$(echo "$response" | tail -n1)
    test_result "Sync upload without auth rejected" 401 "$status_code"

    # Test 2: Upload with auth but empty data
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/sync/upload" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "{\"device_id\":\"test-device\",\"changes\":[]}")
    status_code=$(echo "$response" | tail -n1)
    test_result "Sync upload with empty changes rejected" 400 "$status_code"

    # Test 3: Download without auth
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/sync/download?device_id=test")
    status_code=$(echo "$response" | tail -n1)
    test_result "Sync download without auth rejected" 401 "$status_code"

    # Test 4: Download with auth
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/sync/download?device_id=test-device" \
        -H "Authorization: Bearer $TOKEN")
    status_code=$(echo "$response" | tail -n1)
    test_result "Sync download with auth" 200 "$status_code"

    # Test 5: List conflicts
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/sync/conflicts" \
        -H "Authorization: Bearer $TOKEN")
    status_code=$(echo "$response" | tail -n1)
    test_result "Sync list conflicts" 200 "$status_code"
}

# Test logout
test_logout() {
    print_section "Logout Tests"

    if [ -z "$TOKEN" ]; then
        echo -e "${YELLOW}⚠ SKIP${NC} No token available"
        return
    fi

    # Test 1: Logout with valid token
    response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/api/auth/logout" \
        -H "Authorization: Bearer $TOKEN")
    status_code=$(echo "$response" | tail -n1)
    test_result "Logout with valid token" 200 "$status_code"

    # Test 2: Access after logout should fail
    response=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/auth/profile" \
        -H "Authorization: Bearer $TOKEN")
    status_code=$(echo "$response" | tail -n1)
    test_result "Protected endpoint after logout rejected" 401 "$status_code"
}

# Test CORS
test_cors() {
    print_section "CORS Tests"

    # Test 1: OPTIONS request
    response=$(curl -s -w "\n%{http_code}" -X OPTIONS "$BASE_URL/health" \
        -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: GET")
    status_code=$(echo "$response" | tail -n1)
    test_result "CORS preflight request" 200 "$status_code"

    # Test 2: Check CORS headers
    headers=$(curl -s -I -X OPTIONS "$BASE_URL/health" \
        -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: GET")

    if echo "$headers" | grep -i "Access-Control-Allow-Origin" > /dev/null; then
        echo -e "${GREEN}✓ PASS${NC} CORS headers present"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo -e "${RED}✗ FAIL${NC} CORS headers missing"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
}

# Print summary
print_summary() {
    echo ""
    echo -e "${BLUE}========================================"
    echo "   Test Summary"
    echo "========================================${NC}"
    echo "Total Tests:  $TOTAL_TESTS"
    echo -e "${GREEN}Passed:       $PASSED_TESTS${NC}"

    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "${RED}Failed:       $FAILED_TESTS${NC}"
    else
        echo -e "Failed:       $FAILED_TESTS"
    fi

    echo ""

    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "${GREEN}✓ All tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}✗ Some tests failed${NC}"
        exit 1
    fi
}

# Check if server is reachable
check_server() {
    echo -n "Checking if server is reachable... "
    if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
        echo -e "${GREEN}OK${NC}"
    else
        echo -e "${RED}FAILED${NC}"
        echo "Error: Cannot reach server at $BASE_URL"
        echo "Please ensure the server is running and the URL is correct."
        exit 1
    fi
}

# Main execution
main() {
    print_banner
    check_server

    test_health
    test_signup
    test_login
    test_refresh
    test_protected_endpoints
    test_sync
    test_logout
    test_cors

    print_summary
}

# Check if jq is installed
if ! command -v jq &> /dev/null; then
    echo -e "${YELLOW}Warning: jq is not installed. Some tests may be skipped.${NC}"
    echo "Install jq: brew install jq (macOS) or apt-get install jq (Linux)"
    echo ""
fi

# Run main
main
