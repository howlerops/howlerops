#!/bin/bash
# Test script for install.sh
# This script validates the installer in various scenarios

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_SCRIPT="${SCRIPT_DIR}/../install.sh"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

log_test() {
    echo -e "${BLUE}[TEST]${NC} $*"
}

log_pass() {
    echo -e "${GREEN}[PASS]${NC} $*"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

log_fail() {
    echo -e "${RED}[FAIL]${NC} $*"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

run_test() {
    local test_name="$1"
    shift
    local test_command="$*"

    TESTS_RUN=$((TESTS_RUN + 1))
    log_test "$test_name"

    if eval "$test_command" >/dev/null 2>&1; then
        log_pass "$test_name"
        return 0
    else
        log_fail "$test_name"
        return 1
    fi
}

echo ""
echo "=========================================="
echo "  SQL Studio Installer Test Suite"
echo "=========================================="
echo ""

# Test 1: Script exists and is executable
log_test "Script exists and is executable"
if [ -x "$INSTALL_SCRIPT" ]; then
    log_pass "Script exists and is executable"
else
    log_fail "Script is not executable or missing"
    exit 1
fi

# Test 2: Help flag works
run_test "Help flag works" "$INSTALL_SCRIPT --help"

# Test 3: Script is valid shell script
run_test "Script is valid shell script" "sh -n $INSTALL_SCRIPT"

# Test 4: Dry run mode works
log_test "Dry run mode works"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --dry-run 2>&1); then
    if echo "$output" | grep -q "DRY-RUN"; then
        log_pass "Dry run mode works"
    else
        log_fail "Dry run mode doesn't show DRY-RUN messages"
    fi
else
    log_fail "Dry run mode failed"
fi

# Test 5: Verbose mode works
log_test "Verbose mode works"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --dry-run --verbose 2>&1); then
    if echo "$output" | grep -q "VERBOSE"; then
        log_pass "Verbose mode works"
    else
        log_fail "Verbose mode doesn't show VERBOSE messages"
    fi
else
    log_fail "Verbose mode failed"
fi

# Test 6: Platform detection works
log_test "Platform detection works"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --dry-run 2>&1); then
    if echo "$output" | grep -q "Detected platform:"; then
        log_pass "Platform detection works"
    else
        log_fail "Platform detection failed"
    fi
else
    log_fail "Platform detection failed"
fi

# Test 7: Dependency checking works
log_test "Dependency checking works"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --dry-run --verbose 2>&1); then
    if echo "$output" | grep -q "Checking dependencies"; then
        log_pass "Dependency checking works"
    else
        log_fail "Dependency checking failed"
    fi
else
    log_fail "Dependency checking failed"
fi

# Test 8: Install directory determination
log_test "Install directory determination works"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --dry-run 2>&1); then
    if echo "$output" | grep -q "Installation directory:"; then
        log_pass "Install directory determination works"
    else
        log_fail "Install directory determination failed"
    fi
else
    log_fail "Install directory determination failed"
fi

# Test 9: Custom install directory
log_test "Custom install directory works"
TESTS_RUN=$((TESTS_RUN + 1))
custom_dir="/tmp/sql-studio-test-install"
if output=$("$INSTALL_SCRIPT" --dry-run --install-dir "$custom_dir" 2>&1); then
    if echo "$output" | grep -q "$custom_dir"; then
        log_pass "Custom install directory works"
    else
        log_fail "Custom install directory not respected"
    fi
else
    log_fail "Custom install directory failed"
fi

# Test 10: Version flag works
log_test "Version flag works"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --dry-run --version v1.0.0 2>&1); then
    if echo "$output" | grep -q "v1.0.0"; then
        log_pass "Version flag works"
    else
        log_fail "Version flag doesn't work"
    fi
else
    log_fail "Version flag failed"
fi

# Test 11: Error handling for unknown flags
log_test "Error handling for unknown flags"
TESTS_RUN=$((TESTS_RUN + 1))
if output=$("$INSTALL_SCRIPT" --unknown-flag 2>&1); then
    log_fail "Should have failed with unknown flag"
else
    if echo "$output" | grep -q "Unknown option"; then
        log_pass "Error handling for unknown flags works"
    else
        log_fail "Error message for unknown flag is incorrect"
    fi
fi

# Test 12: POSIX compliance (sh compatibility)
log_test "POSIX compliance (sh compatibility)"
TESTS_RUN=$((TESTS_RUN + 1))
if sh -n "$INSTALL_SCRIPT" >/dev/null 2>&1; then
    log_pass "Script is POSIX compliant"
else
    log_fail "Script is not POSIX compliant"
fi

# Test 13: shellcheck (if available)
if command -v shellcheck >/dev/null 2>&1; then
    log_test "Shellcheck analysis"
    TESTS_RUN=$((TESTS_RUN + 1))
    if shellcheck -x "$INSTALL_SCRIPT" >/dev/null 2>&1; then
        log_pass "Shellcheck passed"
    else
        log_fail "Shellcheck found issues (run 'shellcheck $INSTALL_SCRIPT' for details)"
    fi
else
    echo -e "${YELLOW}[SKIP]${NC} Shellcheck not available, skipping static analysis"
fi

# Summary
echo ""
echo "=========================================="
echo "  Test Summary"
echo "=========================================="
echo "Tests run:    $TESTS_RUN"
echo -e "Tests passed: ${GREEN}$TESTS_PASSED${NC}"
if [ $TESTS_FAILED -gt 0 ]; then
    echo -e "Tests failed: ${RED}$TESTS_FAILED${NC}"
else
    echo -e "Tests failed: $TESTS_FAILED"
fi
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
