#!/usr/bin/env bash

#######################################################################################
# Homebrew Formula Update Script for SQL Studio
#
# This script automates updating the Homebrew formula when a new release is created.
# It fetches the latest release, calculates checksums, and updates the formula file.
#
# Usage:
#   ./scripts/update-homebrew-formula.sh [VERSION]
#
# Examples:
#   ./scripts/update-homebrew-formula.sh v2.0.0
#   ./scripts/update-homebrew-formula.sh latest
#
# Environment Variables:
#   GITHUB_TOKEN - GitHub personal access token for API access and pushing
#   HOMEBREW_TAP_REPO - Tap repository path (default: sql-studio/homebrew-tap)
#   DRY_RUN - If set to "true", only show what would be updated without making changes
#
# Requirements:
#   - curl, jq, shasum, git
#   - GitHub token with repo scope (for pushing to tap repository)
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
GITHUB_REPO="sql-studio/sql-studio"
HOMEBREW_TAP_REPO="${HOMEBREW_TAP_REPO:-sql-studio/homebrew-tap}"
FORMULA_NAME="sql-studio"
DRY_RUN="${DRY_RUN:-false}"

# Temporary directory for downloads
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

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
}

check_dependencies() {
    local missing_deps=()

    for cmd in curl jq shasum git; do
        if ! command -v "$cmd" &> /dev/null; then
            missing_deps+=("$cmd")
        fi
    done

    if [ ${#missing_deps[@]} -ne 0 ]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_info "Install missing dependencies:"
        for dep in "${missing_deps[@]}"; do
            case "$dep" in
                jq)
                    echo "  brew install jq"
                    ;;
                *)
                    echo "  brew install $dep"
                    ;;
            esac
        done
        exit 1
    fi
}

get_latest_release() {
    log_info "Fetching latest release information from GitHub..."

    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    local headers=""

    if [ -n "${GITHUB_TOKEN:-}" ]; then
        headers="-H 'Authorization: token ${GITHUB_TOKEN}'"
    fi

    # shellcheck disable=SC2086
    local response
    response=$(curl -s $headers "$api_url")

    if [ -z "$response" ] || echo "$response" | jq -e '.message == "Not Found"' > /dev/null 2>&1; then
        log_error "Failed to fetch release information. Repository may not exist or no releases found."
        exit 1
    fi

    echo "$response"
}

get_specific_release() {
    local version=$1
    log_info "Fetching release information for version $version..."

    # Remove 'v' prefix if present
    local tag="${version#v}"
    # Add 'v' prefix for tag lookup
    tag="v${tag}"

    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${tag}"
    local headers=""

    if [ -n "${GITHUB_TOKEN:-}" ]; then
        headers="-H 'Authorization: token ${GITHUB_TOKEN}'"
    fi

    # shellcheck disable=SC2086
    local response
    response=$(curl -s $headers "$api_url")

    if [ -z "$response" ] || echo "$response" | jq -e '.message == "Not Found"' > /dev/null 2>&1; then
        log_error "Release $tag not found."
        exit 1
    fi

    echo "$response"
}

download_and_checksum() {
    local url=$1
    local filename=$2

    log_info "Downloading $filename..."

    if ! curl -L -o "$TMP_DIR/$filename" "$url"; then
        log_error "Failed to download $url"
        exit 1
    fi

    log_info "Calculating SHA256 checksum for $filename..."
    local checksum
    checksum=$(shasum -a 256 "$TMP_DIR/$filename" | awk '{print $1}')

    echo "$checksum"
}

update_formula_file() {
    local version=$1
    local amd64_url=$2
    local amd64_sha=$3
    local arm64_url=$4
    local arm64_sha=$5
    local formula_path=$6

    log_info "Updating formula file at $formula_path..."

    # Remove 'v' prefix from version for formula
    local version_number="${version#v}"

    # Create updated formula content
    cat > "$formula_path" << EOF
# typed: false
# frozen_string_literal: true

# SQL Studio Homebrew Formula
# This formula allows users to install SQL Studio via Homebrew
# Usage: brew install sql-studio/tap/sql-studio

class SqlStudio < Formula
  desc "Modern SQL database client with cloud sync capabilities"
  homepage "https://github.com/sql-studio/sql-studio"
  version "$version_number"
  license "MIT"

  # Intel (x86_64) binary
  on_intel do
    url "$amd64_url"
    sha256 "$amd64_sha"
  end

  # Apple Silicon (ARM64) binary
  on_arm do
    url "$arm64_url"
    sha256 "$arm64_sha"
  end

  # Installation steps
  def install
    # Install the binary to Homebrew's bin directory
    bin.install "sql-studio"

    # Generate and install shell completions if available
    if File.exist?("completions")
      bash_completion.install "completions/sql-studio.bash" if File.exist?("completions/sql-studio.bash")
      fish_completion.install "completions/sql-studio.fish" if File.exist?("completions/sql-studio.fish")
      zsh_completion.install "completions/_sql-studio" if File.exist?("completions/_sql-studio")
    end

    # Install man pages if available
    man1.install Dir["man/*.1"] if Dir.exist?("man")
  end

  # Post-installation message
  def caveats
    <<~EOS
      SQL Studio has been installed successfully!

      To get started, run:
        sql-studio

      For help and documentation, visit:
        https://github.com/sql-studio/sql-studio

      To check the version:
        sql-studio --version

      Note: SQL Studio stores its configuration in:
        ~/.config/sql-studio/
    EOS
  end

  # Test block to verify installation
  test do
    # Verify the binary exists and is executable
    assert_predicate bin/"sql-studio", :exist?
    assert_predicate bin/"sql-studio", :executable?

    # Test version command
    version_output = shell_output("#{bin}/sql-studio --version")
    assert_match version.to_s, version_output

    # Verify the binary runs without errors for help command
    help_output = shell_output("#{bin}/sql-studio --help")
    assert_match "SQL Studio", help_output
  end
end
EOF

    log_success "Formula file updated successfully"
}

clone_tap_repository() {
    local tap_dir="$TMP_DIR/homebrew-tap"

    log_info "Cloning Homebrew tap repository..."

    if [ -n "${GITHUB_TOKEN:-}" ]; then
        # Use token for authentication
        local auth_url="https://${GITHUB_TOKEN}@github.com/${HOMEBREW_TAP_REPO}.git"
        git clone "$auth_url" "$tap_dir" 2>&1 | grep -v "token" || true
    else
        git clone "https://github.com/${HOMEBREW_TAP_REPO}.git" "$tap_dir"
    fi

    echo "$tap_dir"
}

commit_and_push_formula() {
    local tap_dir=$1
    local version=$2

    cd "$tap_dir"

    # Configure git if needed
    if [ -z "$(git config user.email)" ]; then
        git config user.email "github-actions[bot]@users.noreply.github.com"
        git config user.name "github-actions[bot]"
    fi

    # Check if there are changes
    if ! git diff --quiet Formula/"$FORMULA_NAME".rb; then
        log_info "Committing changes..."

        git add Formula/"$FORMULA_NAME".rb
        git commit -m "Update $FORMULA_NAME to $version

Automated update via GitHub Actions
- Updated version to $version
- Updated download URLs
- Updated SHA256 checksums for both architectures"

        if [ "$DRY_RUN" = "true" ]; then
            log_warning "DRY RUN: Would push changes to repository"
            log_info "Commit details:"
            git show --stat
        else
            log_info "Pushing changes to repository..."
            git push origin main
            log_success "Changes pushed successfully"
        fi
    else
        log_info "No changes detected in formula file"
    fi
}

validate_release_assets() {
    local release_data=$1

    log_info "Validating release assets..."

    # Check for required assets
    local amd64_asset
    local arm64_asset

    amd64_asset=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("darwin-amd64")) | .browser_download_url' | head -n 1)
    arm64_asset=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("darwin-arm64")) | .browser_download_url' | head -n 1)

    if [ -z "$amd64_asset" ]; then
        log_error "AMD64 (Intel) asset not found in release"
        log_info "Available assets:"
        echo "$release_data" | jq -r '.assets[].name'
        exit 1
    fi

    if [ -z "$arm64_asset" ]; then
        log_error "ARM64 (Apple Silicon) asset not found in release"
        log_info "Available assets:"
        echo "$release_data" | jq -r '.assets[].name'
        exit 1
    fi

    log_success "All required assets found"
}

#######################################################################################
# Main Script
#######################################################################################

main() {
    local version="${1:-latest}"

    log_info "Starting Homebrew formula update for SQL Studio"
    log_info "Target version: $version"
    log_info "Homebrew tap: $HOMEBREW_TAP_REPO"

    # Check for required dependencies
    check_dependencies

    # Check for GitHub token
    if [ -z "${GITHUB_TOKEN:-}" ]; then
        log_warning "GITHUB_TOKEN not set. API rate limits may apply and push operations will fail."
        log_info "Set GITHUB_TOKEN environment variable with a GitHub personal access token."
    fi

    # Fetch release information
    local release_data
    if [ "$version" = "latest" ]; then
        release_data=$(get_latest_release)
    else
        release_data=$(get_specific_release "$version")
    fi

    # Extract release information
    local tag_name
    local release_version
    tag_name=$(echo "$release_data" | jq -r '.tag_name')
    release_version="${tag_name#v}"

    log_info "Found release: $tag_name (version: $release_version)"

    # Validate release assets
    validate_release_assets "$release_data"

    # Extract download URLs
    local amd64_url
    local arm64_url
    amd64_url=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("darwin-amd64")) | .browser_download_url' | head -n 1)
    arm64_url=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("darwin-arm64")) | .browser_download_url' | head -n 1)

    log_info "AMD64 URL: $amd64_url"
    log_info "ARM64 URL: $arm64_url"

    # Download and calculate checksums
    local amd64_sha
    local arm64_sha
    amd64_sha=$(download_and_checksum "$amd64_url" "sql-studio-darwin-amd64.tar.gz")
    arm64_sha=$(download_and_checksum "$arm64_url" "sql-studio-darwin-arm64.tar.gz")

    log_success "AMD64 SHA256: $amd64_sha"
    log_success "ARM64 SHA256: $arm64_sha"

    # Clone tap repository
    local tap_dir
    tap_dir=$(clone_tap_repository)

    # Create Formula directory if it doesn't exist
    mkdir -p "$tap_dir/Formula"

    # Update formula file
    update_formula_file "$tag_name" "$amd64_url" "$amd64_sha" "$arm64_url" "$arm64_sha" "$tap_dir/Formula/$FORMULA_NAME.rb"

    # Commit and push changes
    commit_and_push_formula "$tap_dir" "$tag_name"

    log_success "Homebrew formula update completed successfully!"
    log_info "Users can now install SQL Studio $release_version with:"
    log_info "  brew update && brew upgrade $FORMULA_NAME"

    if [ "$DRY_RUN" = "true" ]; then
        log_warning "DRY RUN MODE: No changes were pushed to the repository"
    fi
}

# Show usage if help requested
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
    cat << EOF
Homebrew Formula Update Script for SQL Studio

Usage:
  $0 [VERSION]

Arguments:
  VERSION    Release version to update to (e.g., v2.0.0 or latest)
             Default: latest

Environment Variables:
  GITHUB_TOKEN         GitHub personal access token (required for pushing)
  HOMEBREW_TAP_REPO    Tap repository (default: sql-studio/homebrew-tap)
  DRY_RUN             Set to 'true' to preview changes without pushing

Examples:
  # Update to latest release
  $0

  # Update to specific version
  $0 v2.0.0

  # Dry run to preview changes
  DRY_RUN=true $0 v2.0.0

  # Use custom tap repository
  HOMEBREW_TAP_REPO=myorg/homebrew-tap $0

Requirements:
  - curl, jq, shasum, git
  - GITHUB_TOKEN for pushing changes

For more information, see HOMEBREW.md
EOF
    exit 0
fi

# Run main function
main "$@"
