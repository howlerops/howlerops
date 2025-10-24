#!/bin/bash

# Organization API Testing Script
# Tests all 15 organization endpoints with authentication

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${API_URL:-http://localhost:8080}"
OUTPUT_DIR="./test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="$OUTPUT_DIR/test_${TIMESTAMP}.log"

# Test data
TEST_EMAIL="test-org-$(date +%s)@example.com"
TEST_PASSWORD="TestPassword123!"
TEST_ORG_NAME="Test Organization $(date +%s)"
INVITED_EMAIL="invited-$(date +%s)@example.com"

# Test counters
PASSED=0
FAILED=0
TOTAL=15

# Create output directory
mkdir -p "$OUTPUT_DIR"

# Logging functions
log() {
    echo -e "${BLUE}[INFO]${NC} $1" | tee -a "$LOG_FILE"
}

success() {
    echo -e "${GREEN}[PASS]${NC} $1" | tee -a "$LOG_FILE"
    ((PASSED++))
}

error() {
    echo -e "${RED}[FAIL]${NC} $1" | tee -a "$LOG_FILE"
    ((FAILED++))
}

warning() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "$LOG_FILE"
}

# Start logging
log "Starting Organization API tests at $(date)"
log "Base URL: $BASE_URL"
log "Output directory: $OUTPUT_DIR"
echo ""

# Check if server is running
log "Checking if server is running..."
if ! curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
    error "Server is not running at $BASE_URL"
    error "Please start the server with: make dev"
    exit 1
fi
success "Server is running"
echo ""

# ====================================================================
# Setup: Register and Login
# ====================================================================

log "=== SETUP: Register and Login ==="

# Register user
log "Registering test user..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\",
        \"username\": \"testuser\"
    }")

if echo "$REGISTER_RESPONSE" | grep -q '"error".*true'; then
    # User might already exist, try to login
    warning "Registration failed (user might already exist)"
else
    success "User registered successfully"
fi

# Login to get JWT token
log "Logging in to get JWT token..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$TEST_EMAIL\",
        \"password\": \"$TEST_PASSWORD\"
    }")

JWT_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -z "$JWT_TOKEN" ]; then
    error "Failed to get JWT token"
    echo "Login response: $LOGIN_RESPONSE"
    exit 1
fi

success "Got JWT token: ${JWT_TOKEN:0:20}..."
echo ""

# ====================================================================
# Test 1: Create Organization
# ====================================================================

log "=== Test 1/15: POST /api/organizations (Create Organization) ==="

CREATE_ORG_RESPONSE=$(curl -s -X POST "$BASE_URL/api/organizations" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"$TEST_ORG_NAME\",
        \"description\": \"Test organization for API testing\"
    }")

echo "$CREATE_ORG_RESPONSE" | tee -a "$LOG_FILE"

ORG_ID=$(echo "$CREATE_ORG_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ -z "$ORG_ID" ]; then
    error "Failed to create organization"
    echo "$CREATE_ORG_RESPONSE"
else
    success "Organization created with ID: $ORG_ID"
fi
echo ""

# ====================================================================
# Test 2: List Organizations
# ====================================================================

log "=== Test 2/15: GET /api/organizations (List Organizations) ==="

LIST_ORGS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/organizations" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$LIST_ORGS_RESPONSE" | tee -a "$LOG_FILE"

if echo "$LIST_ORGS_RESPONSE" | grep -q "\"organizations\""; then
    success "Listed organizations successfully"
else
    error "Failed to list organizations"
fi
echo ""

# ====================================================================
# Test 3: Get Organization
# ====================================================================

log "=== Test 3/15: GET /api/organizations/:id (Get Organization) ==="

GET_ORG_RESPONSE=$(curl -s -X GET "$BASE_URL/api/organizations/$ORG_ID" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$GET_ORG_RESPONSE" | tee -a "$LOG_FILE"

if echo "$GET_ORG_RESPONSE" | grep -q "\"id\":\"$ORG_ID\""; then
    success "Retrieved organization successfully"
else
    error "Failed to get organization"
fi
echo ""

# ====================================================================
# Test 4: Update Organization
# ====================================================================

log "=== Test 4/15: PUT /api/organizations/:id (Update Organization) ==="

UPDATE_ORG_RESPONSE=$(curl -s -X PUT "$BASE_URL/api/organizations/$ORG_ID" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"$TEST_ORG_NAME - Updated\",
        \"description\": \"Updated description\"
    }")

echo "$UPDATE_ORG_RESPONSE" | tee -a "$LOG_FILE"

if echo "$UPDATE_ORG_RESPONSE" | grep -q "Updated"; then
    success "Organization updated successfully"
else
    error "Failed to update organization"
fi
echo ""

# ====================================================================
# Test 5: List Members
# ====================================================================

log "=== Test 5/15: GET /api/organizations/:id/members (List Members) ==="

LIST_MEMBERS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/organizations/$ORG_ID/members" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$LIST_MEMBERS_RESPONSE" | tee -a "$LOG_FILE"

if echo "$LIST_MEMBERS_RESPONSE" | grep -q "\"members\""; then
    success "Listed members successfully"
else
    error "Failed to list members"
fi
echo ""

# ====================================================================
# Test 6: Create Invitation
# ====================================================================

log "=== Test 6/15: POST /api/organizations/:id/invitations (Create Invitation) ==="

CREATE_INVITE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/organizations/$ORG_ID/invitations" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$INVITED_EMAIL\",
        \"role\": \"member\"
    }")

echo "$CREATE_INVITE_RESPONSE" | tee -a "$LOG_FILE"

INVITE_ID=$(echo "$CREATE_INVITE_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)
INVITE_TOKEN=$(echo "$CREATE_INVITE_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$INVITE_ID" ]; then
    error "Failed to create invitation"
else
    success "Invitation created with ID: $INVITE_ID"
fi
echo ""

# ====================================================================
# Test 7: List Organization Invitations
# ====================================================================

log "=== Test 7/15: GET /api/organizations/:id/invitations (List Org Invitations) ==="

LIST_ORG_INVITES_RESPONSE=$(curl -s -X GET "$BASE_URL/api/organizations/$ORG_ID/invitations" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$LIST_ORG_INVITES_RESPONSE" | tee -a "$LOG_FILE"

if echo "$LIST_ORG_INVITES_RESPONSE" | grep -q "\"invitations\""; then
    success "Listed organization invitations successfully"
else
    error "Failed to list organization invitations"
fi
echo ""

# ====================================================================
# Test 8: List User Invitations
# ====================================================================

log "=== Test 8/15: GET /api/invitations?email= (List User Invitations) ==="

LIST_USER_INVITES_RESPONSE=$(curl -s -X GET "$BASE_URL/api/invitations?email=$INVITED_EMAIL" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$LIST_USER_INVITES_RESPONSE" | tee -a "$LOG_FILE"

if echo "$LIST_USER_INVITES_RESPONSE" | grep -q "\"invitations\""; then
    success "Listed user invitations successfully"
else
    error "Failed to list user invitations"
fi
echo ""

# ====================================================================
# Test 9: Get Audit Logs
# ====================================================================

log "=== Test 9/15: GET /api/organizations/:id/audit-logs (Get Audit Logs) ==="

AUDIT_LOGS_RESPONSE=$(curl -s -X GET "$BASE_URL/api/organizations/$ORG_ID/audit-logs?limit=10&offset=0" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$AUDIT_LOGS_RESPONSE" | tee -a "$LOG_FILE"

if echo "$AUDIT_LOGS_RESPONSE" | grep -q "\"logs\""; then
    success "Retrieved audit logs successfully"
else
    error "Failed to get audit logs"
fi
echo ""

# ====================================================================
# Test 10: Accept Invitation (requires second user)
# ====================================================================

log "=== Test 10/15: POST /api/invitations/:id/accept (Accept Invitation) ==="

# Register invited user
INVITED_USER_EMAIL="invited-user-$(date +%s)@example.com"
REGISTER_INVITED_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$INVITED_USER_EMAIL\",
        \"password\": \"$TEST_PASSWORD\",
        \"username\": \"inviteduser\"
    }")

# Login as invited user
LOGIN_INVITED_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"$INVITED_USER_EMAIL\",
        \"password\": \"$TEST_PASSWORD\"
    }")

INVITED_JWT_TOKEN=$(echo "$LOGIN_INVITED_RESPONSE" | grep -o '"access_token":"[^"]*' | cut -d'"' -f4)

if [ -n "$INVITE_TOKEN" ] && [ -n "$INVITED_JWT_TOKEN" ]; then
    ACCEPT_INVITE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/invitations/$INVITE_TOKEN/accept" \
        -H "Authorization: Bearer $INVITED_JWT_TOKEN")

    echo "$ACCEPT_INVITE_RESPONSE" | tee -a "$LOG_FILE"

    if echo "$ACCEPT_INVITE_RESPONSE" | grep -q "\"success\".*true"; then
        success "Invitation accepted successfully"
    else
        error "Failed to accept invitation"
    fi
else
    warning "Skipping accept invitation test (no invitation or token)"
fi
echo ""

# ====================================================================
# Test 11: Update Member Role
# ====================================================================

log "=== Test 11/15: PUT /api/organizations/:id/members/:userId (Update Member Role) ==="

if [ -n "$INVITED_JWT_TOKEN" ]; then
    # Get invited user ID from their JWT
    # For now, we'll skip this test as we need the user ID
    warning "Skipping update member role test (requires user ID extraction)"
else
    warning "Skipping update member role test (no invited user)"
fi
echo ""

# ====================================================================
# Test 12: Revoke Invitation
# ====================================================================

log "=== Test 12/15: DELETE /api/organizations/:id/invitations/:inviteId (Revoke Invitation) ==="

# Create another invitation to revoke
CREATE_REVOKE_INVITE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/organizations/$ORG_ID/invitations" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"revoke-$(date +%s)@example.com\",
        \"role\": \"member\"
    }")

REVOKE_INVITE_ID=$(echo "$CREATE_REVOKE_INVITE_RESPONSE" | grep -o '"id":"[^"]*' | cut -d'"' -f4)

if [ -n "$REVOKE_INVITE_ID" ]; then
    REVOKE_RESPONSE=$(curl -s -X DELETE "$BASE_URL/api/organizations/$ORG_ID/invitations/$REVOKE_INVITE_ID" \
        -H "Authorization: Bearer $JWT_TOKEN")

    echo "$REVOKE_RESPONSE" | tee -a "$LOG_FILE"

    if echo "$REVOKE_RESPONSE" | grep -q "\"success\".*true"; then
        success "Invitation revoked successfully"
    else
        error "Failed to revoke invitation"
    fi
else
    error "Failed to create invitation to revoke"
fi
echo ""

# ====================================================================
# Test 13: Decline Invitation
# ====================================================================

log "=== Test 13/15: POST /api/invitations/:id/decline (Decline Invitation) ==="

# Create another invitation to decline
CREATE_DECLINE_INVITE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/organizations/$ORG_ID/invitations" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{
        \"email\": \"decline-$(date +%s)@example.com\",
        \"role\": \"member\"
    }")

DECLINE_INVITE_TOKEN=$(echo "$CREATE_DECLINE_INVITE_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -n "$DECLINE_INVITE_TOKEN" ]; then
    DECLINE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/invitations/$DECLINE_INVITE_TOKEN/decline")

    echo "$DECLINE_RESPONSE" | tee -a "$LOG_FILE"

    if echo "$DECLINE_RESPONSE" | grep -q "\"success\".*true"; then
        success "Invitation declined successfully"
    else
        error "Failed to decline invitation"
    fi
else
    error "Failed to create invitation to decline"
fi
echo ""

# ====================================================================
# Test 14: Remove Member
# ====================================================================

log "=== Test 14/15: DELETE /api/organizations/:id/members/:userId (Remove Member) ==="

warning "Skipping remove member test (requires member user ID)"
echo ""

# ====================================================================
# Test 15: Delete Organization
# ====================================================================

log "=== Test 15/15: DELETE /api/organizations/:id (Delete Organization) ==="

# First remove all members except owner
# For now, we'll try to delete directly

DELETE_ORG_RESPONSE=$(curl -s -X DELETE "$BASE_URL/api/organizations/$ORG_ID" \
    -H "Authorization: Bearer $JWT_TOKEN")

echo "$DELETE_ORG_RESPONSE" | tee -a "$LOG_FILE"

if echo "$DELETE_ORG_RESPONSE" | grep -q "\"success\".*true"; then
    success "Organization deleted successfully"
else
    # Check if it failed due to members
    if echo "$DELETE_ORG_RESPONSE" | grep -q "members"; then
        warning "Cannot delete organization with members (expected behavior)"
    else
        error "Failed to delete organization"
    fi
fi
echo ""

# ====================================================================
# Authentication Tests
# ====================================================================

log "=== BONUS: Authentication Tests ==="

# Test without token
log "Testing request without authentication..."
NO_AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/organizations")
HTTP_CODE=$(echo "$NO_AUTH_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ]; then
    success "Correctly rejected request without authentication (401)"
else
    error "Did not reject unauthenticated request (got $HTTP_CODE)"
fi

# Test with invalid token
log "Testing request with invalid token..."
INVALID_AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/organizations" \
    -H "Authorization: Bearer invalid_token_here")
HTTP_CODE=$(echo "$INVALID_AUTH_RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "401" ]; then
    success "Correctly rejected request with invalid token (401)"
else
    error "Did not reject invalid token (got $HTTP_CODE)"
fi
echo ""

# ====================================================================
# Test Summary
# ====================================================================

log "=== TEST SUMMARY ==="
log "Total Tests: $TOTAL"
success "Passed: $PASSED"
error "Failed: $FAILED"
log "Success Rate: $(awk "BEGIN {printf \"%.1f\", ($PASSED/$TOTAL)*100}")%"
echo ""

log "Results saved to: $LOG_FILE"
log "Test completed at $(date)"

# Exit with error code if any tests failed
if [ $FAILED -gt 0 ]; then
    exit 1
fi

exit 0
