#!/bin/bash

# RBAC Security Penetration Testing Script
# Tests permission bypass attempts and security vulnerabilities
# Author: Security Audit Team
# Date: 2025-10-23

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
API_BASE_URL="${API_BASE_URL:-http://localhost:8080}"
TEST_RESULTS_FILE="security-test-results.log"

# Test user credentials (these should be set up in advance)
OWNER_TOKEN="${OWNER_TOKEN:-}"
ADMIN_TOKEN="${ADMIN_TOKEN:-}"
MEMBER_TOKEN="${MEMBER_TOKEN:-}"
NON_MEMBER_TOKEN="${NON_MEMBER_TOKEN:-}"

# Test organization ID
TEST_ORG_ID="${TEST_ORG_ID:-}"

# Initialize results file
echo "RBAC Security Penetration Test Results - $(date)" > "$TEST_RESULTS_FILE"
echo "==========================================" >> "$TEST_RESULTS_FILE"

# Helper functions
log_test() {
    echo -e "${YELLOW}[TEST]${NC} $1"
    echo "[TEST] $1" >> "$TEST_RESULTS_FILE"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $1"
    echo "[PASS] $1" >> "$TEST_RESULTS_FILE"
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $1"
    echo "[FAIL] $1" >> "$TEST_RESULTS_FILE"
    FAILED_TESTS=$((FAILED_TESTS + 1))
}

# Test counter
TOTAL_TESTS=0
FAILED_TESTS=0

# Function to make API request and check response
test_api_call() {
    local method="$1"
    local endpoint="$2"
    local token="$3"
    local expected_code="$4"
    local body="$5"
    local test_description="$6"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    log_test "$test_description"

    # Build curl command
    if [ -n "$body" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Authorization: Bearer $token" \
            -H "Content-Type: application/json" \
            -d "$body" \
            "$API_BASE_URL$endpoint" 2>&1)
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" \
            -H "Authorization: Bearer $token" \
            "$API_BASE_URL$endpoint" 2>&1)
    fi

    # Extract status code (last line)
    status_code=$(echo "$response" | tail -n1)
    response_body=$(echo "$response" | head -n -1)

    if [ "$status_code" = "$expected_code" ]; then
        log_pass "Expected $expected_code, got $status_code"
        return 0
    else
        log_fail "Expected $expected_code, got $status_code. Response: $response_body"
        return 1
    fi
}

# =====================================================================
# TEST SUITE 1: Member Permission Bypass Attempts
# =====================================================================
echo ""
echo "=== TEST SUITE 1: Member Permission Bypass Attempts ==="

# Test 1.1: Member tries to invite (should fail with 403)
test_api_call "POST" "/api/organizations/$TEST_ORG_ID/invitations" "$MEMBER_TOKEN" "403" \
    '{"email":"test@example.com","role":"member"}' \
    "Member attempts to invite new member (should be denied)"

# Test 1.2: Member tries to update organization (should fail with 403)
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID" "$MEMBER_TOKEN" "403" \
    '{"name":"Hacked Org Name"}' \
    "Member attempts to update organization (should be denied)"

# Test 1.3: Member tries to delete organization (should fail with 403)
test_api_call "DELETE" "/api/organizations/$TEST_ORG_ID" "$MEMBER_TOKEN" "403" "" \
    "Member attempts to delete organization (should be denied)"

# Test 1.4: Member tries to remove another member (should fail with 403)
test_api_call "DELETE" "/api/organizations/$TEST_ORG_ID/members/some-user-id" "$MEMBER_TOKEN" "403" "" \
    "Member attempts to remove another member (should be denied)"

# Test 1.5: Member tries to change roles (should fail with 403)
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID/members/some-user-id" "$MEMBER_TOKEN" "403" \
    '{"role":"admin"}' \
    "Member attempts to promote user to admin (should be denied)"

# Test 1.6: Member tries to view audit logs (should fail with 403)
test_api_call "GET" "/api/organizations/$TEST_ORG_ID/audit-logs" "$MEMBER_TOKEN" "403" "" \
    "Member attempts to view audit logs (should be denied)"

# =====================================================================
# TEST SUITE 2: Admin Permission Bypass Attempts
# =====================================================================
echo ""
echo "=== TEST SUITE 2: Admin Permission Bypass Attempts ==="

# Test 2.1: Admin tries to delete organization (should fail with 403)
test_api_call "DELETE" "/api/organizations/$TEST_ORG_ID" "$ADMIN_TOKEN" "403" "" \
    "Admin attempts to delete organization (should be denied)"

# Test 2.2: Admin tries to promote to owner (should fail)
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID/members/some-user-id" "$ADMIN_TOKEN" "400" \
    '{"role":"owner"}' \
    "Admin attempts to promote user to owner (should be denied)"

# Test 2.3: Admin tries to remove owner (should fail)
OWNER_USER_ID="${OWNER_USER_ID:-owner-id}"
test_api_call "DELETE" "/api/organizations/$TEST_ORG_ID/members/$OWNER_USER_ID" "$ADMIN_TOKEN" "400" "" \
    "Admin attempts to remove owner (should be denied)"

# Test 2.4: Admin tries to change owner's role (should fail)
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID/members/$OWNER_USER_ID" "$ADMIN_TOKEN" "400" \
    '{"role":"member"}' \
    "Admin attempts to demote owner (should be denied)"

# =====================================================================
# TEST SUITE 3: Non-Member Access Attempts
# =====================================================================
echo ""
echo "=== TEST SUITE 3: Non-Member Access Attempts ==="

# Test 3.1: Non-member tries to access organization (should fail with 403)
test_api_call "GET" "/api/organizations/$TEST_ORG_ID" "$NON_MEMBER_TOKEN" "403" "" \
    "Non-member attempts to view organization (should be denied)"

# Test 3.2: Non-member tries to view members (should fail with 403)
test_api_call "GET" "/api/organizations/$TEST_ORG_ID/members" "$NON_MEMBER_TOKEN" "403" "" \
    "Non-member attempts to view members (should be denied)"

# Test 3.3: Non-member tries to invite (should fail with 403)
test_api_call "POST" "/api/organizations/$TEST_ORG_ID/invitations" "$NON_MEMBER_TOKEN" "403" \
    '{"email":"hacker@example.com","role":"owner"}' \
    "Non-member attempts to create invitation (should be denied)"

# =====================================================================
# TEST SUITE 4: Token and Invitation Security
# =====================================================================
echo ""
echo "=== TEST SUITE 4: Token and Invitation Security ==="

# Test 4.1: Expired token acceptance (should fail)
EXPIRED_TOKEN="expired-invitation-token"
test_api_call "POST" "/api/invitations/$EXPIRED_TOKEN/accept" "$NON_MEMBER_TOKEN" "400" "" \
    "Attempt to accept expired invitation (should fail)"

# Test 4.2: Invalid token format (should fail)
test_api_call "POST" "/api/invitations/invalid-token-123/accept" "$NON_MEMBER_TOKEN" "404" "" \
    "Attempt to accept invalid token (should fail)"

# Test 4.3: Token reuse after acceptance (should fail)
USED_TOKEN="already-used-token"
test_api_call "POST" "/api/invitations/$USED_TOKEN/accept" "$NON_MEMBER_TOKEN" "400" "" \
    "Attempt to reuse accepted invitation (should fail)"

# =====================================================================
# TEST SUITE 5: Input Validation and Injection Attempts
# =====================================================================
echo ""
echo "=== TEST SUITE 5: Input Validation and Injection Attempts ==="

# Test 5.1: SQL injection in organization name
test_api_call "POST" "/api/organizations" "$OWNER_TOKEN" "400" \
    '{"name":"Test\" OR 1=1 --","description":"SQL injection test"}' \
    "SQL injection in organization name (should be rejected)"

# Test 5.2: XSS attempt in organization description
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID" "$OWNER_TOKEN" "200" \
    '{"description":"<script>alert(\"XSS\")</script>"}' \
    "XSS in organization description (should be sanitized)"

# Test 5.3: Invalid email format
test_api_call "POST" "/api/organizations/$TEST_ORG_ID/invitations" "$OWNER_TOKEN" "400" \
    '{"email":"not-an-email","role":"member"}' \
    "Invalid email format (should be rejected)"

# Test 5.4: Invalid role value
test_api_call "POST" "/api/organizations/$TEST_ORG_ID/invitations" "$OWNER_TOKEN" "400" \
    '{"email":"test@example.com","role":"superadmin"}' \
    "Invalid role value (should be rejected)"

# Test 5.5: Overly long organization name
LONG_NAME=$(printf 'A%.0s' {1..100})
test_api_call "POST" "/api/organizations" "$OWNER_TOKEN" "400" \
    "{\"name\":\"$LONG_NAME\",\"description\":\"Test\"}" \
    "Overly long organization name (should be rejected)"

# =====================================================================
# TEST SUITE 6: Rate Limiting
# =====================================================================
echo ""
echo "=== TEST SUITE 6: Rate Limiting ==="

# Test 6.1: Invitation rate limit bypass attempts
log_test "Testing invitation rate limits (sending 25 requests)"
RATE_LIMIT_EXCEEDED=0
for i in {1..25}; do
    response=$(curl -s -w "\n%{http_code}" -X POST \
        -H "Authorization: Bearer $OWNER_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"email\":\"test$i@example.com\",\"role\":\"member\"}" \
        "$API_BASE_URL/api/organizations/$TEST_ORG_ID/invitations" 2>&1)

    status_code=$(echo "$response" | tail -n1)
    if [ "$status_code" = "429" ]; then
        RATE_LIMIT_EXCEEDED=1
        break
    fi
done

if [ $RATE_LIMIT_EXCEEDED -eq 1 ]; then
    log_pass "Rate limit enforced (429 returned)"
else
    log_fail "Rate limit not enforced (no 429 received after 25 requests)"
fi

# =====================================================================
# TEST SUITE 7: Authentication Bypass Attempts
# =====================================================================
echo ""
echo "=== TEST SUITE 7: Authentication Bypass Attempts ==="

# Test 7.1: No token provided (should fail with 401)
response=$(curl -s -w "\n%{http_code}" -X GET \
    "$API_BASE_URL/api/organizations/$TEST_ORG_ID" 2>&1)
status_code=$(echo "$response" | tail -n1)
if [ "$status_code" = "401" ]; then
    log_pass "No token request properly rejected with 401"
else
    log_fail "No token request not rejected (got $status_code)"
fi

# Test 7.2: Invalid token format (should fail with 401)
response=$(curl -s -w "\n%{http_code}" -X GET \
    -H "Authorization: Bearer invalid.token.here" \
    "$API_BASE_URL/api/organizations/$TEST_ORG_ID" 2>&1)
status_code=$(echo "$response" | tail -n1)
if [ "$status_code" = "401" ]; then
    log_pass "Invalid token format properly rejected with 401"
else
    log_fail "Invalid token format not rejected (got $status_code)"
fi

# Test 7.3: Malformed Authorization header (should fail with 401)
response=$(curl -s -w "\n%{http_code}" -X GET \
    -H "Authorization: NotBearer $MEMBER_TOKEN" \
    "$API_BASE_URL/api/organizations/$TEST_ORG_ID" 2>&1)
status_code=$(echo "$response" | tail -n1)
if [ "$status_code" = "401" ]; then
    log_pass "Malformed auth header properly rejected with 401"
else
    log_fail "Malformed auth header not rejected (got $status_code)"
fi

# =====================================================================
# TEST SUITE 8: Privilege Escalation Attempts
# =====================================================================
echo ""
echo "=== TEST SUITE 8: Privilege Escalation Attempts ==="

# Test 8.1: Member tries to self-promote to admin
MEMBER_USER_ID="${MEMBER_USER_ID:-member-id}"
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID/members/$MEMBER_USER_ID" "$MEMBER_TOKEN" "403" \
    '{"role":"admin"}' \
    "Member attempts self-promotion to admin (should be denied)"

# Test 8.2: Admin tries to self-promote to owner
ADMIN_USER_ID="${ADMIN_USER_ID:-admin-id}"
test_api_call "PUT" "/api/organizations/$TEST_ORG_ID/members/$ADMIN_USER_ID" "$ADMIN_TOKEN" "400" \
    '{"role":"owner"}' \
    "Admin attempts self-promotion to owner (should be denied)"

# =====================================================================
# TEST SUITE 9: Data Leakage Tests
# =====================================================================
echo ""
echo "=== TEST SUITE 9: Data Leakage Tests ==="

# Test 9.1: Check error messages don't reveal sensitive data
log_test "Checking error messages for data leakage"
response=$(curl -s -X GET \
    -H "Authorization: Bearer $NON_MEMBER_TOKEN" \
    "$API_BASE_URL/api/organizations/non-existent-org-id" 2>&1)

if echo "$response" | grep -q "password\|secret\|key\|token"; then
    log_fail "Error message contains sensitive keywords"
else
    log_pass "Error messages don't contain sensitive data"
fi

# =====================================================================
# TEST SUITE 10: CORS and CSRF Tests
# =====================================================================
echo ""
echo "=== TEST SUITE 10: CORS and CSRF Tests ==="

# Test 10.1: CORS headers check
log_test "Checking CORS headers"
response=$(curl -s -I -X OPTIONS \
    -H "Origin: https://evil.com" \
    -H "Access-Control-Request-Method: POST" \
    "$API_BASE_URL/api/organizations" 2>&1)

if echo "$response" | grep -q "Access-Control-Allow-Origin: \*"; then
    log_fail "CORS allows all origins (security risk)"
elif echo "$response" | grep -q "Access-Control-Allow-Origin"; then
    log_pass "CORS headers present and restricted"
else
    log_pass "CORS not enabled (API-only access)"
fi

# =====================================================================
# RESULTS SUMMARY
# =====================================================================
echo ""
echo "==========================================="
echo "SECURITY PENETRATION TEST SUMMARY"
echo "==========================================="
echo "Total Tests Run: $TOTAL_TESTS"
echo "Tests Passed: $((TOTAL_TESTS - FAILED_TESTS))"
echo "Tests Failed: $FAILED_TESTS"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    echo -e "${GREEN}ALL SECURITY TESTS PASSED!${NC}"
    echo "The RBAC implementation appears to be secure."
else
    echo -e "${RED}SECURITY ISSUES DETECTED!${NC}"
    echo "Please review the failed tests above and fix the vulnerabilities."
fi

echo ""
echo "Detailed results saved to: $TEST_RESULTS_FILE"
echo ""

# Exit with error if any tests failed
exit $FAILED_TESTS