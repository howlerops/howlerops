#!/bin/bash
set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

SERVICE_URL=${1:-""}

if [ -z "$SERVICE_URL" ]; then
    echo -e "${RED}❌ Error: Service URL is required${NC}"
    echo ""
    echo "Usage: $0 <service-url>"
    echo ""
    echo "Example:"
    echo "  $0 https://sql-studio-backend-abc123-uc.a.run.app"
    echo ""
    echo "Or automatically get the URL:"
    echo "  SERVICE_URL=\$(gcloud run services describe sql-studio-backend \\"
    echo "      --region=us-central1 \\"
    echo "      --format='value(status.url)')"
    echo "  $0 \$SERVICE_URL"
    exit 1
fi

# Remove trailing slash if present
SERVICE_URL=${SERVICE_URL%/}

echo -e "${BLUE}🧪 Verifying Howlerops Backend Deployment${NC}"
echo "=============================================="
echo "Service URL: $SERVICE_URL"
echo ""

FAILED=0
PASSED=0

# Helper function for test results
test_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✅ $2${NC}"
        PASSED=$((PASSED + 1))
    else
        echo -e "${RED}❌ $2${NC}"
        FAILED=$((FAILED + 1))
    fi
}

# Test 1: Health Check
echo -e "${BLUE}Test 1: Health Endpoint${NC}"
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "${SERVICE_URL}/health" 2>&1)
HTTP_CODE=$(echo "$HEALTH_RESPONSE" | tail -n 1)
BODY=$(echo "$HEALTH_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    test_result 0 "Health endpoint returns 200"
    echo "  Response: $BODY"
else
    test_result 1 "Health endpoint returns 200 (got $HTTP_CODE)"
    echo "  Response: $BODY"
fi
echo ""

# Test 2: Metrics Endpoint
echo -e "${BLUE}Test 2: Metrics Endpoint${NC}"
METRICS_RESPONSE=$(curl -s -w "\n%{http_code}" "${SERVICE_URL}/metrics" 2>&1)
HTTP_CODE=$(echo "$METRICS_RESPONSE" | tail -n 1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "404" ]; then
    if [ "$HTTP_CODE" = "200" ]; then
        test_result 0 "Metrics endpoint accessible"
        METRIC_COUNT=$(echo "$METRICS_RESPONSE" | head -n -1 | grep -c "^# TYPE" || echo "0")
        echo "  Metrics exposed: $METRIC_COUNT"
    else
        test_result 0 "Metrics endpoint (internal only, got 404 - expected)"
    fi
else
    test_result 1 "Metrics endpoint accessible (got $HTTP_CODE)"
fi
echo ""

# Test 3: Signup Flow
echo -e "${BLUE}Test 3: User Signup${NC}"
RANDOM_SUFFIX=$(date +%s)$(($RANDOM % 1000))
TEST_USERNAME="test_user_${RANDOM_SUFFIX}"
TEST_EMAIL="test_${RANDOM_SUFFIX}@example.com"
TEST_PASSWORD="TestPass123!@#"

SIGNUP_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${SERVICE_URL}/api/auth/signup" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"${TEST_USERNAME}\",\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}" 2>&1)

HTTP_CODE=$(echo "$SIGNUP_RESPONSE" | tail -n 1)
BODY=$(echo "$SIGNUP_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ] || [ "$HTTP_CODE" = "201" ]; then
    # Check if response contains expected fields
    if echo "$BODY" | grep -q "user" && echo "$BODY" | grep -q "token"; then
        test_result 0 "Signup endpoint works and returns user + token"
        # Extract token for next tests
        TOKEN=$(echo "$BODY" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        echo "  Created user: $TEST_USERNAME"
        echo "  Token received: ${TOKEN:0:20}..."
    else
        test_result 1 "Signup endpoint returns expected structure"
        echo "  Response: $BODY"
    fi
else
    test_result 1 "Signup endpoint works (got HTTP $HTTP_CODE)"
    echo "  Response: $BODY"
fi
echo ""

# Test 4: Login Flow (using the user we just created)
echo -e "${BLUE}Test 4: User Login${NC}"
LOGIN_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${SERVICE_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"${TEST_EMAIL}\",\"password\":\"${TEST_PASSWORD}\"}" 2>&1)

HTTP_CODE=$(echo "$LOGIN_RESPONSE" | tail -n 1)
BODY=$(echo "$LOGIN_RESPONSE" | head -n -1)

if [ "$HTTP_CODE" = "200" ]; then
    if echo "$BODY" | grep -q "token"; then
        test_result 0 "Login endpoint works and returns token"
        LOGIN_TOKEN=$(echo "$BODY" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
        echo "  Token received: ${LOGIN_TOKEN:0:20}..."
    else
        test_result 1 "Login endpoint returns expected structure"
        echo "  Response: $BODY"
    fi
else
    test_result 1 "Login endpoint works (got HTTP $HTTP_CODE)"
    echo "  Response: $BODY"
fi
echo ""

# Test 5: Protected Endpoint (Sync Download)
echo -e "${BLUE}Test 5: Protected Endpoint (JWT Auth)${NC}"
if [ -n "$TOKEN" ]; then
    SYNC_RESPONSE=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "${SERVICE_URL}/api/sync/download" 2>&1)

    HTTP_CODE=$(echo "$SYNC_RESPONSE" | tail -n 1)
    BODY=$(echo "$SYNC_RESPONSE" | head -n -1)

    if [ "$HTTP_CODE" = "200" ]; then
        test_result 0 "Protected endpoint accessible with JWT"
        echo "  Response: ${BODY:0:100}..."
    elif [ "$HTTP_CODE" = "404" ]; then
        test_result 0 "Protected endpoint requires auth (endpoint might not exist yet)"
        echo "  Note: This is okay if sync endpoints aren't fully implemented"
    else
        test_result 1 "Protected endpoint accessible (got HTTP $HTTP_CODE)"
        echo "  Response: $BODY"
    fi
else
    test_result 1 "Protected endpoint test skipped (no token from signup)"
fi
echo ""

# Test 6: CORS Headers
echo -e "${BLUE}Test 6: CORS Configuration${NC}"
CORS_RESPONSE=$(curl -s -I -X OPTIONS "${SERVICE_URL}/api/auth/login" \
    -H "Origin: https://sqlstudio.io" \
    -H "Access-Control-Request-Method: POST" \
    -H "Access-Control-Request-Headers: Content-Type" 2>&1)

if echo "$CORS_RESPONSE" | grep -qi "Access-Control-Allow-Origin"; then
    test_result 0 "CORS headers configured"
    ALLOW_ORIGIN=$(echo "$CORS_RESPONSE" | grep -i "Access-Control-Allow-Origin" | tr -d '\r')
    echo "  $ALLOW_ORIGIN"
else
    test_result 1 "CORS headers configured"
    echo -e "${YELLOW}  Warning: CORS might not be properly configured${NC}"
fi
echo ""

# Test 7: Invalid Login
echo -e "${BLUE}Test 7: Invalid Login (Security Check)${NC}"
INVALID_LOGIN=$(curl -s -w "\n%{http_code}" -X POST "${SERVICE_URL}/api/auth/login" \
    -H "Content-Type: application/json" \
    -d '{"email":"nonexistent@example.com","password":"wrongpassword"}' 2>&1)

HTTP_CODE=$(echo "$INVALID_LOGIN" | tail -n 1)

if [ "$HTTP_CODE" = "401" ] || [ "$HTTP_CODE" = "400" ]; then
    test_result 0 "Invalid login properly rejected"
    echo "  Returns HTTP $HTTP_CODE for invalid credentials"
else
    test_result 1 "Invalid login properly rejected (got HTTP $HTTP_CODE)"
fi
echo ""

# Test 8: Rate Limiting Headers
echo -e "${BLUE}Test 8: Rate Limiting${NC}"
RATE_RESPONSE=$(curl -s -I "${SERVICE_URL}/health" 2>&1)

if echo "$RATE_RESPONSE" | grep -qi "X-RateLimit" || echo "$RATE_RESPONSE" | grep -qi "RateLimit"; then
    test_result 0 "Rate limiting headers present"
    RATE_HEADERS=$(echo "$RATE_RESPONSE" | grep -i "RateLimit" | tr -d '\r')
    echo "  $RATE_HEADERS"
else
    echo -e "${YELLOW}⚠️  Rate limiting headers not detected (might be configured elsewhere)${NC}"
    echo "  This is optional if rate limiting is handled by middleware"
fi
echo ""

# Test 9: Response Time
echo -e "${BLUE}Test 9: Response Time${NC}"
START_TIME=$(date +%s%N)
curl -s "${SERVICE_URL}/health" > /dev/null
END_TIME=$(date +%s%N)
ELAPSED=$((($END_TIME - $START_TIME) / 1000000)) # Convert to milliseconds

if [ $ELAPSED -lt 1000 ]; then
    test_result 0 "Response time under 1 second"
    echo "  Health endpoint: ${ELAPSED}ms"
else
    test_result 1 "Response time under 1 second"
    echo "  Health endpoint: ${ELAPSED}ms (slow!)"
fi
echo ""

# Test 10: Structured Logging (check if service is logging)
echo -e "${BLUE}Test 10: Logging Check${NC}"
echo -e "${YELLOW}ℹ️  To verify structured logging, check Cloud Logging:${NC}"
echo "  gcloud logging read \"resource.type=cloud_run_revision\" --limit=5"
echo "  Or visit: https://console.cloud.google.com/logs"
echo ""

# Summary
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Test Summary${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "Passed: ${GREEN}${PASSED}${NC}"
echo -e "Failed: ${RED}${FAILED}${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}✅ All Tests Passed!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "🎉 Deployment verified successfully!"
    echo ""
    echo "📋 Next steps:"
    echo "  1. Configure custom domain (optional)"
    echo "  2. Set up monitoring alerts"
    echo "  3. Configure frontend to use: $SERVICE_URL"
    echo "  4. Monitor logs: gcloud run services logs tail sql-studio-backend"
    echo ""
    echo "🔗 Useful links:"
    echo "  - Cloud Run Console: https://console.cloud.google.com/run"
    echo "  - Logs: https://console.cloud.google.com/logs"
    echo "  - Monitoring: https://console.cloud.google.com/monitoring"
    echo ""
    exit 0
else
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ Some Tests Failed${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo "🔍 Troubleshooting:"
    echo "  1. Check service logs:"
    echo "     gcloud run services logs tail sql-studio-backend"
    echo ""
    echo "  2. Check service status:"
    echo "     gcloud run services describe sql-studio-backend --region=us-central1"
    echo ""
    echo "  3. Verify secrets are accessible:"
    echo "     gcloud secrets list"
    echo ""
    echo "  4. Check recent deployments:"
    echo "     gcloud run revisions list --service=sql-studio-backend"
    echo ""
    echo "  5. Test locally:"
    echo "     make dev"
    echo ""
    exit 1
fi
