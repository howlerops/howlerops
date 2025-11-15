#!/usr/bin/env bash

#######################################################################################
# Homebrew Formula Testing Script for Howlerops
#
# This script validates the Homebrew formula locally before pushing to the tap.
# It performs comprehensive testing including syntax validation, installation,
# and functionality testing.
#
# Usage:
#   ./scripts/test-homebrew-formula.sh [FORMULA_PATH]
#
# Examples:
#   ./scripts/test-homebrew-formula.sh Formula/sql-studio.rb
#   ./scripts/test-homebrew-formula.sh /tmp/homebrew-tap/Formula/sql-studio.rb
#
# Requirements:
#   - Homebrew installed
#   - macOS (Intel or Apple Silicon)
#   - Formula file exists
#
#######################################################################################

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
FORMULA_PATH="${1:-Formula/sql-studio.rb}"
FORMULA_NAME="sql-studio"
TEST_FAILED=0

#######################################################################################
# Helper Functions
#######################################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    TEST_FAILED=1
}

print_section() {
    echo ""
    echo "========================================================================"
    echo "$1"
    echo "========================================================================"
}

check_homebrew() {
    if ! command -v brew &> /dev/null; then
        log_error "Homebrew is not installed"
        log_info "Install Homebrew from: https://brew.sh"
        exit 1
    fi

    log_success "Homebrew is installed"
    brew --version
}

check_formula_exists() {
    if [ ! -f "$FORMULA_PATH" ]; then
        log_error "Formula file not found: $FORMULA_PATH"
        exit 1
    fi

    log_success "Formula file found: $FORMULA_PATH"
}

#######################################################################################
# Test Functions
#######################################################################################

test_formula_syntax() {
    print_section "Test 1: Formula Syntax Validation"

    log_info "Running syntax check..."

    # Check Ruby syntax
    if ruby -c "$FORMULA_PATH" > /dev/null 2>&1; then
        log_success "Ruby syntax is valid"
    else
        log_error "Ruby syntax check failed"
        ruby -c "$FORMULA_PATH"
        return 1
    fi

    # Check Homebrew formula structure
    if brew info "$FORMULA_PATH" > /dev/null 2>&1; then
        log_success "Formula structure is valid"
    else
        log_error "Formula structure check failed"
        brew info "$FORMULA_PATH" || true
        return 1
    fi
}

test_formula_audit() {
    print_section "Test 2: Formula Audit"

    log_info "Running Homebrew audit..."

    # Run audit with different strictness levels
    local audit_output

    # Basic audit
    log_info "Running basic audit..."
    if audit_output=$(brew audit "$FORMULA_PATH" 2>&1); then
        log_success "Basic audit passed"
    else
        log_warning "Basic audit found issues:"
        echo "$audit_output"
    fi

    # Strict audit
    log_info "Running strict audit..."
    if audit_output=$(brew audit --strict "$FORMULA_PATH" 2>&1); then
        log_success "Strict audit passed"
    else
        log_warning "Strict audit found issues:"
        echo "$audit_output"
        # Strict audit warnings are non-fatal
    fi

    # Style audit
    log_info "Running style audit..."
    if audit_output=$(brew style "$FORMULA_PATH" 2>&1); then
        log_success "Style audit passed"
    else
        log_warning "Style audit found issues:"
        echo "$audit_output"
        # Style issues are non-fatal
    fi
}

test_formula_installation() {
    print_section "Test 3: Formula Installation"

    log_info "Testing formula installation..."

    # Uninstall if already installed
    if brew list "$FORMULA_NAME" &> /dev/null; then
        log_info "Uninstalling existing installation..."
        brew uninstall "$FORMULA_NAME" || true
    fi

    # Try installing from formula file
    log_info "Installing from formula file..."
    if brew install --build-from-source "$FORMULA_PATH"; then
        log_success "Installation successful"
    else
        log_error "Installation failed"
        return 1
    fi

    # Verify installation
    if brew list "$FORMULA_NAME" &> /dev/null; then
        log_success "Formula is installed"
    else
        log_error "Formula is not installed after installation"
        return 1
    fi
}

test_binary_functionality() {
    print_section "Test 4: Binary Functionality"

    log_info "Testing installed binary..."

    # Check if binary exists
    local binary_path
    binary_path=$(brew --prefix "$FORMULA_NAME")/bin/"$FORMULA_NAME"

    if [ -f "$binary_path" ]; then
        log_success "Binary exists: $binary_path"
    else
        log_error "Binary not found: $binary_path"
        return 1
    fi

    # Check if binary is executable
    if [ -x "$binary_path" ]; then
        log_success "Binary is executable"
    else
        log_error "Binary is not executable"
        return 1
    fi

    # Test version command
    log_info "Testing version command..."
    if "$binary_path" --version > /dev/null 2>&1; then
        log_success "Version command works"
        "$binary_path" --version
    else
        log_warning "Version command failed (may be expected if binary requires setup)"
    fi

    # Test help command
    log_info "Testing help command..."
    if "$binary_path" --help > /dev/null 2>&1; then
        log_success "Help command works"
    else
        log_warning "Help command failed (may be expected)"
    fi

    # Show binary info
    log_info "Binary information:"
    ls -lh "$binary_path"
    file "$binary_path"
}

test_formula_test_block() {
    print_section "Test 5: Formula Test Block"

    log_info "Running formula test block..."

    if brew test "$FORMULA_NAME"; then
        log_success "Formula test block passed"
    else
        log_error "Formula test block failed"
        return 1
    fi
}

test_formula_cleanup() {
    print_section "Test 6: Formula Cleanup"

    log_info "Testing formula uninstallation..."

    if brew uninstall "$FORMULA_NAME"; then
        log_success "Uninstallation successful"
    else
        log_error "Uninstallation failed"
        return 1
    fi

    # Verify cleanup
    if ! brew list "$FORMULA_NAME" &> /dev/null; then
        log_success "Formula is completely removed"
    else
        log_error "Formula is still installed after uninstallation"
        return 1
    fi
}

display_formula_info() {
    print_section "Formula Information"

    log_info "Formula details:"
    echo ""

    # Extract key information from formula
    if [ -f "$FORMULA_PATH" ]; then
        echo "Name: $(basename "$FORMULA_PATH" .rb)"
        echo "Path: $FORMULA_PATH"
        echo ""

        # Try to get description, homepage, version
        grep -E "(desc|homepage|version|license)" "$FORMULA_PATH" | head -20 || true
        echo ""

        # Show URL blocks
        echo "Download URLs:"
        grep -A 2 "on_intel\|on_arm" "$FORMULA_PATH" || true
        echo ""
    fi
}

test_checksums() {
    print_section "Test 7: Checksum Validation"

    log_info "Validating SHA256 checksums..."

    # Extract URLs and checksums from formula
    local intel_url arm_url intel_sha arm_sha

    intel_url=$(grep -A 1 "on_intel" "$FORMULA_PATH" | grep "url" | sed 's/.*"\(.*\)".*/\1/')
    arm_url=$(grep -A 1 "on_arm" "$FORMULA_PATH" | grep "url" | sed 's/.*"\(.*\)".*/\1/')

    intel_sha=$(grep -A 2 "on_intel" "$FORMULA_PATH" | grep "sha256" | sed 's/.*"\(.*\)".*/\1/')
    arm_sha=$(grep -A 2 "on_arm" "$FORMULA_PATH" | grep "sha256" | sed 's/.*"\(.*\)".*/\1/')

    log_info "Intel URL: $intel_url"
    log_info "ARM URL: $arm_url"

    # Check if checksums are placeholders
    if [[ "$intel_sha" == *"PLACEHOLDER"* ]] || [[ "$arm_sha" == *"PLACEHOLDER"* ]]; then
        log_warning "Formula contains placeholder checksums"
        log_info "Update checksums before publishing to tap"
        return 0
    fi

    log_success "Checksums are not placeholders"

    # Validate checksum format (64 hex characters)
    if [[ "$intel_sha" =~ ^[a-f0-9]{64}$ ]]; then
        log_success "Intel SHA256 format is valid"
    else
        log_error "Intel SHA256 format is invalid: $intel_sha"
    fi

    if [[ "$arm_sha" =~ ^[a-f0-9]{64}$ ]]; then
        log_success "ARM SHA256 format is valid"
    else
        log_error "ARM SHA256 format is invalid: $arm_sha"
    fi
}

#######################################################################################
# Main Script
#######################################################################################

main() {
    log_info "Starting Homebrew formula testing for Howlerops"
    log_info "Formula: $FORMULA_PATH"
    echo ""

    # Prerequisites
    check_homebrew
    check_formula_exists

    # Display formula info
    display_formula_info

    # Run tests
    test_formula_syntax || true
    test_checksums || true
    test_formula_audit || true

    # Only run installation tests if not in CI or explicitly requested
    if [ "${SKIP_INSTALL_TESTS:-false}" != "true" ]; then
        test_formula_installation || true
        test_binary_functionality || true
        test_formula_test_block || true
        test_formula_cleanup || true
    else
        log_warning "Skipping installation tests (SKIP_INSTALL_TESTS=true)"
    fi

    # Summary
    print_section "Test Summary"

    if [ $TEST_FAILED -eq 0 ]; then
        log_success "All tests passed!"
        echo ""
        log_info "Formula is ready for publishing"
        exit 0
    else
        log_error "Some tests failed"
        echo ""
        log_info "Please fix the issues before publishing"
        exit 1
    fi
}

# Show usage if help requested
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
    cat << EOF
Homebrew Formula Testing Script for Howlerops

Usage:
  $0 [FORMULA_PATH]

Arguments:
  FORMULA_PATH    Path to the formula file to test
                  Default: Formula/sql-studio.rb

Environment Variables:
  SKIP_INSTALL_TESTS    Set to 'true' to skip installation tests
                        Useful for CI or quick syntax checks

Examples:
  # Test formula in Formula directory
  $0

  # Test specific formula file
  $0 /tmp/homebrew-tap/Formula/sql-studio.rb

  # Skip installation tests
  SKIP_INSTALL_TESTS=true $0

  # Run only syntax and audit checks
  SKIP_INSTALL_TESTS=true $0 Formula/sql-studio.rb

Requirements:
  - Homebrew installed (https://brew.sh)
  - macOS (Intel or Apple Silicon)
  - Ruby (pre-installed on macOS)

For more information, see HOMEBREW.md
EOF
    exit 0
fi

# Run main function
main "$@"
