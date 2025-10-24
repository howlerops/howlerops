#!/bin/bash

# Load Testing Script for SQL Studio Backend
# Tests performance under sustained load

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
BASE_URL="${TEST_BASE_URL:-http://localhost:8500}"
DURATION="${LOAD_TEST_DURATION:-60}"  # seconds
CONCURRENT="${LOAD_TEST_CONCURRENT:-10}"  # concurrent requests
RATE_LIMIT="${LOAD_TEST_RATE:-100}"  # requests per second

# Results directory
RESULTS_DIR="./load-test-results"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULT_FILE="$RESULTS_DIR/load_test_${TIMESTAMP}.txt"

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "========================================"
    echo "   SQL Studio Load Test"
    echo "========================================"
    echo -e "${NC}"
    echo "Target:       $BASE_URL"
    echo "Duration:     ${DURATION}s"
    echo "Concurrent:   $CONCURRENT"
    echo "Rate Limit:   ${RATE_LIMIT} req/s"
    echo ""
}

# Create results directory
mkdir -p "$RESULTS_DIR"

# Check for load testing tools
check_tools() {
    local missing_tools=()

    if ! command -v curl &> /dev/null; then
        missing_tools+=("curl")
    fi

    # Check for optional tools
    local has_hey=false
    local has_ab=false
    local has_wrk=false

    if command -v hey &> /dev/null; then
        has_hey=true
        echo -e "${GREEN}✓${NC} hey is installed"
    fi

    if command -v ab &> /dev/null; then
        has_ab=true
        echo -e "${GREEN}✓${NC} ab (Apache Bench) is installed"
    fi

    if command -v wrk &> /dev/null; then
        has_wrk=true
        echo -e "${GREEN}✓${NC} wrk is installed"
    fi

    if [ ${#missing_tools[@]} -gt 0 ]; then
        echo -e "${RED}Error: Missing required tools: ${missing_tools[*]}${NC}"
        exit 1
    fi

    if ! $has_hey && ! $has_ab && ! $has_wrk; then
        echo -e "${YELLOW}Warning: No load testing tool found${NC}"
        echo "Install one of: hey, ab (apache-bench), or wrk"
        echo ""
        echo "Installation:"
        echo "  macOS:   brew install hey"
        echo "  macOS:   brew install apache-bench (for ab)"
        echo "  macOS:   brew install wrk"
        echo ""
        echo "Falling back to basic curl-based load test..."
        echo ""
        return 1
    fi

    return 0
}

# Test with hey (modern HTTP load testing tool)
load_test_with_hey() {
    local endpoint=$1
    local name=$2

    echo -e "\n${BLUE}[TEST]${NC} $name"
    echo "Using: hey"

    local total_requests=$((RATE_LIMIT * DURATION))

    hey -n "$total_requests" -c "$CONCURRENT" -q "$RATE_LIMIT" \
        "$BASE_URL$endpoint" 2>&1 | tee -a "$RESULT_FILE"

    echo ""
}

# Test with Apache Bench
load_test_with_ab() {
    local endpoint=$1
    local name=$2

    echo -e "\n${BLUE}[TEST]${NC} $name"
    echo "Using: ab (Apache Bench)"

    local total_requests=$((RATE_LIMIT * DURATION))

    ab -n "$total_requests" -c "$CONCURRENT" -g "$RESULTS_DIR/gnuplot_${TIMESTAMP}.tsv" \
        "$BASE_URL$endpoint" 2>&1 | tee -a "$RESULT_FILE"

    echo ""
}

# Test with wrk
load_test_with_wrk() {
    local endpoint=$1
    local name=$2

    echo -e "\n${BLUE}[TEST]${NC} $name"
    echo "Using: wrk"

    wrk -t "$CONCURRENT" -c "$CONCURRENT" -d "${DURATION}s" \
        "$BASE_URL$endpoint" 2>&1 | tee -a "$RESULT_FILE"

    echo ""
}

# Basic curl-based load test (fallback)
load_test_with_curl() {
    local endpoint=$1
    local name=$2

    echo -e "\n${BLUE}[TEST]${NC} $name"
    echo "Using: curl (basic)"

    local total_requests=$((RATE_LIMIT * DURATION))
    local success=0
    local failed=0
    local total_time=0

    echo "Running $total_requests requests..."

    for i in $(seq 1 "$total_requests"); do
        local start=$(date +%s%N)

        if curl -s -f "$BASE_URL$endpoint" -o /dev/null --max-time 5 2>&1; then
            success=$((success + 1))
        else
            failed=$((failed + 1))
        fi

        local end=$(date +%s%N)
        local duration=$(( (end - start) / 1000000 ))  # Convert to milliseconds
        total_time=$((total_time + duration))

        # Progress indicator
        if [ $((i % 10)) -eq 0 ]; then
            echo -ne "\rProgress: $i/$total_requests"
        fi

        # Rate limiting
        sleep $(echo "scale=3; 1/$RATE_LIMIT" | bc)
    done

    echo ""
    echo ""
    echo "Results:"
    echo "  Total Requests:   $total_requests"
    echo "  Successful:       $success"
    echo "  Failed:           $failed"
    echo "  Success Rate:     $(echo "scale=2; $success * 100 / $total_requests" | bc)%"
    echo "  Avg Response Time: $(echo "scale=2; $total_time / $total_requests" | bc)ms"
    echo ""

    # Save results
    {
        echo "Test: $name"
        echo "Endpoint: $endpoint"
        echo "Total Requests: $total_requests"
        echo "Successful: $success"
        echo "Failed: $failed"
        echo "Success Rate: $(echo "scale=2; $success * 100 / $total_requests" | bc)%"
        echo "Avg Response Time: $(echo "scale=2; $total_time / $total_requests" | bc)ms"
        echo ""
    } >> "$RESULT_FILE"
}

# Test health endpoint under load
test_health_endpoint() {
    echo -e "${BLUE}[SCENARIO 1]${NC} Health Check Load Test"
    echo "Testing: GET /health"
    echo ""

    if command -v hey &> /dev/null; then
        load_test_with_hey "/health" "Health Check"
    elif command -v ab &> /dev/null; then
        load_test_with_ab "/health" "Health Check"
    elif command -v wrk &> /dev/null; then
        load_test_with_wrk "/health" "Health Check"
    else
        load_test_with_curl "/health" "Health Check"
    fi
}

# Test auth endpoint under load
test_auth_endpoint() {
    echo -e "${BLUE}[SCENARIO 2]${NC} Auth Login Load Test"
    echo "Testing: POST /api/auth/login"
    echo ""

    # For POST requests, we need to use curl with data
    local total_requests=$((RATE_LIMIT * DURATION / 2))  # Lower rate for POST
    local success=0
    local failed=0
    local total_time=0

    echo "Running $total_requests auth requests..."

    for i in $(seq 1 "$total_requests"); do
        local start=$(date +%s%N)

        local timestamp=$(date +%s)
        local data="{\"username\":\"loadtest${timestamp}\",\"password\":\"LoadTest123!\"}"

        if curl -s -X POST "$BASE_URL/api/auth/login" \
            -H "Content-Type: application/json" \
            -d "$data" \
            -o /dev/null \
            --max-time 5 2>&1; then
            success=$((success + 1))
        else
            failed=$((failed + 1))
        fi

        local end=$(date +%s%N)
        local duration=$(( (end - start) / 1000000 ))
        total_time=$((total_time + duration))

        if [ $((i % 10)) -eq 0 ]; then
            echo -ne "\rProgress: $i/$total_requests"
        fi

        sleep 0.02  # 50 requests per second
    done

    echo ""
    echo ""
    echo "Results:"
    echo "  Total Requests:   $total_requests"
    echo "  Successful:       $success"
    echo "  Failed:           $failed"
    echo "  Success Rate:     $(echo "scale=2; $success * 100 / $total_requests" | bc)%"
    echo "  Avg Response Time: $(echo "scale=2; $total_time / $total_requests" | bc)ms"
    echo ""

    # Save results
    {
        echo "Test: Auth Login Load Test"
        echo "Total Requests: $total_requests"
        echo "Successful: $success"
        echo "Failed: $failed"
        echo "Success Rate: $(echo "scale=2; $success * 100 / $total_requests" | bc)%"
        echo "Avg Response Time: $(echo "scale=2; $total_time / $total_requests" | bc)ms"
        echo ""
    } >> "$RESULT_FILE"
}

# Sustained load test
test_sustained_load() {
    echo -e "${BLUE}[SCENARIO 3]${NC} Sustained Load Test"
    echo "Testing: Mixed endpoints for ${DURATION}s"
    echo ""

    local success=0
    local failed=0
    local start_time=$(date +%s)
    local current_time=$start_time

    echo "Running sustained load test..."

    while [ $((current_time - start_time)) -lt $DURATION ]; do
        # Randomly test different endpoints
        local rand=$((RANDOM % 3))

        case $rand in
            0)
                curl -s -f "$BASE_URL/health" -o /dev/null --max-time 5 && success=$((success + 1)) || failed=$((failed + 1))
                ;;
            1)
                curl -s -X POST "$BASE_URL/api/auth/login" \
                    -H "Content-Type: application/json" \
                    -d '{"username":"test","password":"test"}' \
                    -o /dev/null --max-time 5 && success=$((success + 1)) || failed=$((failed + 1))
                ;;
            2)
                curl -s "$BASE_URL/api/sync/download?device_id=test" \
                    -o /dev/null --max-time 5 && success=$((success + 1)) || failed=$((failed + 1))
                ;;
        esac

        current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ $((success + failed)) -gt 0 ] && [ $(( (success + failed) % 50)) -eq 0 ]; then
            echo -ne "\rProgress: ${elapsed}s/${DURATION}s | Requests: $((success + failed))"
        fi

        sleep 0.01
    done

    echo ""
    echo ""
    echo "Results:"
    echo "  Duration:         ${DURATION}s"
    echo "  Total Requests:   $((success + failed))"
    echo "  Successful:       $success"
    echo "  Failed:           $failed"
    echo "  Success Rate:     $(echo "scale=2; $success * 100 / (success + failed)" | bc)%"
    echo "  Requests/sec:     $(echo "scale=2; (success + failed) / $DURATION" | bc)"
    echo ""

    # Save results
    {
        echo "Test: Sustained Load Test"
        echo "Duration: ${DURATION}s"
        echo "Total Requests: $((success + failed))"
        echo "Successful: $success"
        echo "Failed: $failed"
        echo "Success Rate: $(echo "scale=2; $success * 100 / (success + failed)" | bc)%"
        echo "Requests/sec: $(echo "scale=2; (success + failed) / $DURATION" | bc)"
        echo ""
    } >> "$RESULT_FILE"
}

# Analyze results and identify bottlenecks
analyze_results() {
    echo -e "${BLUE}[ANALYSIS]${NC} Performance Analysis"
    echo ""

    if [ -f "$RESULT_FILE" ]; then
        echo "Full results saved to: $RESULT_FILE"
        echo ""

        # Extract key metrics
        local success_rates=$(grep "Success Rate:" "$RESULT_FILE" | awk '{print $3}' | sed 's/%//')

        if [ -n "$success_rates" ]; then
            local total=0
            local count=0

            while IFS= read -r rate; do
                total=$(echo "$total + $rate" | bc)
                count=$((count + 1))
            done <<< "$success_rates"

            local avg_success_rate=$(echo "scale=2; $total / $count" | bc)

            echo "Summary:"
            echo "  Average Success Rate: ${avg_success_rate}%"

            if [ "$(echo "$avg_success_rate < 95" | bc)" -eq 1 ]; then
                echo -e "  ${RED}⚠ Warning: Success rate below 95%${NC}"
                echo "  Possible bottlenecks:"
                echo "    - Database connection pool too small"
                echo "    - Rate limiting too aggressive"
                echo "    - Server resources insufficient"
            else
                echo -e "  ${GREEN}✓ Performance is good${NC}"
            fi
        fi
    fi

    echo ""
}

# Print summary
print_summary() {
    echo -e "${BLUE}========================================"
    echo "   Load Test Complete"
    echo "========================================${NC}"
    echo "Results saved to: $RESULT_FILE"
    echo ""
    echo "To view results:"
    echo "  cat $RESULT_FILE"
    echo ""
}

# Main execution
main() {
    print_banner
    check_tools

    # Initialize result file
    {
        echo "Load Test Results"
        echo "================="
        echo "Target: $BASE_URL"
        echo "Duration: ${DURATION}s"
        echo "Concurrent: $CONCURRENT"
        echo "Rate Limit: ${RATE_LIMIT} req/s"
        echo "Timestamp: $TIMESTAMP"
        echo ""
    } > "$RESULT_FILE"

    # Run load tests
    test_health_endpoint
    test_auth_endpoint
    test_sustained_load

    # Analyze results
    analyze_results
    print_summary
}

# Check for bc command (needed for calculations)
if ! command -v bc &> /dev/null; then
    echo -e "${YELLOW}Warning: bc is not installed. Some calculations may fail.${NC}"
    echo "Install bc: brew install bc (macOS) or apt-get install bc (Linux)"
    echo ""
fi

# Run main
main
