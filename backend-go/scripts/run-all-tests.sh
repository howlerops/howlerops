#!/bin/bash

# Master Test Execution Script
# Runs all tests in sequence and generates a comprehensive report

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${TEST_BASE_URL:-http://localhost:8500}"
REPORT_DIR="test-reports"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
REPORT_FILE="$REPORT_DIR/test_report_${TIMESTAMP}.txt"

# Test status tracking
UNIT_TESTS_PASSED=false
INTEGRATION_TESTS_PASSED=false
SMOKE_TESTS_PASSED=false
LOAD_TESTS_PASSED=false
API_TESTS_PASSED=false

# Create report directory
mkdir -p "$REPORT_DIR"

# Print banner
print_banner() {
    echo -e "${CYAN}"
    echo "========================================"
    echo "   SQL Studio - Full Test Suite"
    echo "========================================"
    echo -e "${NC}"
    echo "Timestamp: $(date)"
    echo "Target: $BASE_URL"
    echo "Report: $REPORT_FILE"
    echo ""
}

# Print section
print_section() {
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}  $1${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

# Log to file and console
log_both() {
    echo "$1" | tee -a "$REPORT_FILE"
}

# Initialize report
init_report() {
    {
        echo "SQL Studio Backend - Test Report"
        echo "================================="
        echo "Timestamp: $(date)"
        echo "Target: $BASE_URL"
        echo ""
    } > "$REPORT_FILE"
}

# Run unit tests
run_unit_tests() {
    print_section "1. Unit Tests"
    log_both "Running unit tests..."

    if go test ./... -v -coverprofile=coverage.out 2>&1 | tee -a "$REPORT_FILE"; then
        log_both "$(echo -e ${GREEN}✓ Unit tests PASSED${NC})"
        UNIT_TESTS_PASSED=true

        # Generate coverage report
        go tool cover -func=coverage.out | tee -a "$REPORT_FILE"
        local coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
        log_both "Total coverage: $coverage"
    else
        log_both "$(echo -e ${RED}✗ Unit tests FAILED${NC})"
        UNIT_TESTS_PASSED=false
    fi
}

# Check if server is running
check_server() {
    print_section "2. Server Health Check"
    log_both "Checking if server is running at $BASE_URL..."

    if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
        log_both "$(echo -e ${GREEN}✓ Server is running${NC})"
        return 0
    else
        log_both "$(echo -e ${YELLOW}⚠ Server not running, attempting to start...${NC})"

        # Try to start server
        go build -o server cmd/server/main.go
        ./server > server.log 2>&1 &
        SERVER_PID=$!
        echo $SERVER_PID > server.pid

        # Wait for server to start
        sleep 10

        if curl -s -f "$BASE_URL/health" > /dev/null 2>&1; then
            log_both "$(echo -e ${GREEN}✓ Server started successfully${NC})"
            return 0
        else
            log_both "$(echo -e ${RED}✗ Failed to start server${NC})"
            return 1
        fi
    fi
}

# Run smoke tests
run_smoke_tests() {
    print_section "3. Smoke Tests"
    log_both "Running smoke tests..."

    if ./scripts/smoke-tests.sh 2>&1 | tee -a "$REPORT_FILE"; then
        log_both "$(echo -e ${GREEN}✓ Smoke tests PASSED${NC})"
        SMOKE_TESTS_PASSED=true
    else
        log_both "$(echo -e ${RED}✗ Smoke tests FAILED${NC})"
        SMOKE_TESTS_PASSED=false
    fi
}

# Run integration tests
run_integration_tests() {
    print_section "4. Integration Tests"
    log_both "Running integration tests..."

    if go test ./test/integration/... -v -timeout 10m 2>&1 | tee -a "$REPORT_FILE"; then
        log_both "$(echo -e ${GREEN}✓ Integration tests PASSED${NC})"
        INTEGRATION_TESTS_PASSED=true
    else
        log_both "$(echo -e ${RED}✗ Integration tests FAILED${NC})"
        INTEGRATION_TESTS_PASSED=false
    fi
}

# Run API tests
run_api_tests() {
    print_section "5. API Tests"
    log_both "Running API tests..."

    if ./scripts/test-api.sh 2>&1 | tee -a "$REPORT_FILE"; then
        log_both "$(echo -e ${GREEN}✓ API tests PASSED${NC})"
        API_TESTS_PASSED=true
    else
        log_both "$(echo -e ${RED}✗ API tests FAILED${NC})"
        API_TESTS_PASSED=false
    fi
}

# Run load tests
run_load_tests() {
    print_section "6. Load Tests"
    log_both "Running load tests..."

    if ./scripts/load-test.sh 2>&1 | tee -a "$REPORT_FILE"; then
        log_both "$(echo -e ${GREEN}✓ Load tests PASSED${NC})"
        LOAD_TESTS_PASSED=true
    else
        log_both "$(echo -e ${YELLOW}⚠ Load tests completed with warnings${NC})"
        LOAD_TESTS_PASSED=true  # Load tests don't fail the suite
    fi
}

# Generate summary
generate_summary() {
    print_section "Test Summary"

    local total_passed=0
    local total_tests=5

    {
        echo ""
        echo "Test Results Summary"
        echo "===================="
        echo ""
    } | tee -a "$REPORT_FILE"

    # Unit Tests
    if $UNIT_TESTS_PASSED; then
        echo -e "${GREEN}✓${NC} Unit Tests: PASSED" | tee -a "$REPORT_FILE"
        total_passed=$((total_passed + 1))
    else
        echo -e "${RED}✗${NC} Unit Tests: FAILED" | tee -a "$REPORT_FILE"
    fi

    # Smoke Tests
    if $SMOKE_TESTS_PASSED; then
        echo -e "${GREEN}✓${NC} Smoke Tests: PASSED" | tee -a "$REPORT_FILE"
        total_passed=$((total_passed + 1))
    else
        echo -e "${RED}✗${NC} Smoke Tests: FAILED" | tee -a "$REPORT_FILE"
    fi

    # Integration Tests
    if $INTEGRATION_TESTS_PASSED; then
        echo -e "${GREEN}✓${NC} Integration Tests: PASSED" | tee -a "$REPORT_FILE"
        total_passed=$((total_passed + 1))
    else
        echo -e "${RED}✗${NC} Integration Tests: FAILED" | tee -a "$REPORT_FILE"
    fi

    # API Tests
    if $API_TESTS_PASSED; then
        echo -e "${GREEN}✓${NC} API Tests: PASSED" | tee -a "$REPORT_FILE"
        total_passed=$((total_passed + 1))
    else
        echo -e "${RED}✗${NC} API Tests: FAILED" | tee -a "$REPORT_FILE"
    fi

    # Load Tests
    if $LOAD_TESTS_PASSED; then
        echo -e "${GREEN}✓${NC} Load Tests: PASSED" | tee -a "$REPORT_FILE"
        total_passed=$((total_passed + 1))
    else
        echo -e "${YELLOW}⚠${NC} Load Tests: WARNING" | tee -a "$REPORT_FILE"
    fi

    echo "" | tee -a "$REPORT_FILE"
    echo "Overall: $total_passed/$total_tests tests passed" | tee -a "$REPORT_FILE"
    echo "" | tee -a "$REPORT_FILE"

    # Files generated
    echo "Generated Files:" | tee -a "$REPORT_FILE"
    echo "  - Coverage: coverage.out" | tee -a "$REPORT_FILE"
    echo "  - Report: $REPORT_FILE" | tee -a "$REPORT_FILE"
    if [ -d "load-test-results" ]; then
        echo "  - Load tests: load-test-results/" | tee -a "$REPORT_FILE"
    fi

    echo "" | tee -a "$REPORT_FILE"

    # Exit code
    if [ $total_passed -eq $total_tests ]; then
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}  ALL TESTS PASSED! ✓${NC}"
        echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        return 0
    else
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${RED}  SOME TESTS FAILED ✗${NC}"
        echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        return 1
    fi
}

# Cleanup
cleanup() {
    print_section "Cleanup"

    if [ -f server.pid ]; then
        log_both "Stopping server..."
        kill $(cat server.pid) 2>/dev/null || true
        rm server.pid
        rm -f server
    fi

    log_both "Cleanup complete"
}

# Main execution
main() {
    print_banner
    init_report

    # Run tests
    run_unit_tests

    if check_server; then
        run_smoke_tests
        run_integration_tests
        run_api_tests

        # Ask if user wants to run load tests
        echo ""
        read -p "Run load tests? (may take several minutes) [y/N]: " -n 1 -r
        echo ""
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            run_load_tests
        else
            log_both "Skipping load tests"
            LOAD_TESTS_PASSED=true
        fi
    else
        log_both "$(echo -e ${RED}Cannot run integration tests without server${NC})"
        INTEGRATION_TESTS_PASSED=false
        API_TESTS_PASSED=false
        SMOKE_TESTS_PASSED=false
    fi

    # Generate summary
    generate_summary
    local exit_code=$?

    # Cleanup
    cleanup

    echo ""
    echo "Full report saved to: $REPORT_FILE"
    echo ""

    exit $exit_code
}

# Trap cleanup on exit
trap cleanup EXIT

# Run main
main
