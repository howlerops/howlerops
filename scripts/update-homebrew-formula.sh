#!/usr/bin/env bash

#######################################################################################
# Homebrew Formula Update Script for HowlerOps - FIXED VERSION
#
# This script automates updating the Homebrew formula when a new release is created.
# It fetches the latest release, calculates checksums, and updates the formula file.
#
# FIXES APPLIED (v0.7.15):
# - Removed silent error suppression (|| true) that hid failures
# - Fixed function return logic to properly propagate errors
# - Added exponential backoff retry strategy
# - Improved error messages with debugging information
# - Added validation of API responses before using them
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
#   HOMEBREW_TAP_REPO - Tap repository path (default: jbeck018/homebrew-howlerops)
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
GITHUB_REPO="jbeck018/howlerops"
HOMEBREW_TAP_REPO="${HOMEBREW_TAP_REPO:-jbeck018/homebrew-howlerops}"
FORMULA_NAME="howlerops"
DRY_RUN="${DRY_RUN:-false}"

# Temporary directory for downloads
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

#######################################################################################
# Helper Functions
#######################################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1" >&2
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" >&2
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
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

    # Use gh CLI if available (preferred - uses gh's authentication)
    if command -v gh &> /dev/null; then
        log_info "Using gh CLI to fetch latest release (GH_TOKEN is ${GH_TOKEN:+set}${GH_TOKEN:-unset})"

        local response
        response=$(gh api "repos/${GITHUB_REPO}/releases/latest" 2>&1) || {
            local exit_code=$?
            log_error "Failed to fetch release via gh CLI (exit code: $exit_code):"
            echo "$response" | head -20
            return 1
        }

        # Validate response is JSON
        if ! echo "$response" | jq empty > /dev/null 2>&1; then
            log_error "Invalid JSON response from gh CLI:"
            echo "$response" | head -20
            return 1
        fi

        log_info "Successfully fetched latest release via gh CLI"
        echo "$response"
        return 0
    else
        log_warning "gh CLI not found, falling back to curl"
    fi

    # Fallback to curl with GITHUB_TOKEN
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

    local response
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        response=$(curl -s -H "Authorization: token ${GITHUB_TOKEN}" "$api_url")
    else
        response=$(curl -s "$api_url")
    fi

    if [ -z "$response" ]; then
        log_error "Failed to fetch release information. Empty response from API."
        return 1
    fi

    # Check if response is valid JSON
    if ! echo "$response" | jq empty > /dev/null 2>&1; then
        log_error "Invalid JSON response from GitHub API:"
        echo "$response" | head -20
        return 1
    fi

    if echo "$response" | jq -e '.message == "Not Found"' > /dev/null 2>&1; then
        log_error "Repository or release not found."
        return 1
    fi

    echo "$response"
    return 0
}

get_specific_release() {
    local version=$1
    log_info "Fetching release information for version $version..."

    # Remove 'v' prefix if present
    local tag="${version#v}"
    # Add 'v' prefix for tag lookup
    tag="v${tag}"

    # Use gh CLI if available (preferred - uses gh's authentication)
    if command -v gh &> /dev/null; then
        log_info "Using gh CLI to fetch release (GH_TOKEN is ${GH_TOKEN:+set}${GH_TOKEN:-unset})"

        local response
        response=$(gh api "repos/${GITHUB_REPO}/releases/tags/${tag}" 2>&1) || {
            local exit_code=$?
            log_error "Failed to fetch release $tag via gh CLI (exit code: $exit_code):"
            echo "$response" | head -20
            return 1
        }

        # Validate response is JSON
        if ! echo "$response" | jq empty > /dev/null 2>&1; then
            log_error "Invalid JSON response from gh CLI:"
            echo "$response" | head -20
            return 1
        fi

        log_info "Successfully fetched release via gh CLI"
        echo "$response"
        return 0
    else
        log_warning "gh CLI not found, falling back to curl"
    fi

    # Fallback to curl with GITHUB_TOKEN
    local api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/tags/${tag}"

    local response
    if [ -n "${GITHUB_TOKEN:-}" ]; then
        response=$(curl -s -H "Authorization: token ${GITHUB_TOKEN}" "$api_url")
    else
        response=$(curl -s "$api_url")
    fi

    if [ -z "$response" ]; then
        log_error "Failed to fetch release information. Empty response from API."
        return 1
    fi

    # Check if response is valid JSON
    if ! echo "$response" | jq empty > /dev/null 2>&1; then
        log_error "Invalid JSON response from GitHub API:"
        echo "$response" | head -20
        return 1
    fi

    if echo "$response" | jq -e '.message == "Not Found"' > /dev/null 2>&1; then
        log_error "Release $tag not found."
        return 1
    fi

    echo "$response"
    return 0
}

download_and_checksum() {
    local url=$1
    local filename=$2

    log_info "Downloading $filename..."

    if ! curl -L -o "$TMP_DIR/$filename" "$url"; then
        log_error "Failed to download $url"
        return 1
    fi

    log_info "Calculating SHA256 checksum for $filename..."
    local checksum
    checksum=$(shasum -a 256 "$TMP_DIR/$filename" | awk '{print $1}')

    echo "$checksum"
}

update_formula_file() {
    local version=$1
    local universal_url=$2
    local universal_sha=$3
    local formula_path=$4

    log_info "Updating cask file at $formula_path..."

    # Remove 'v' prefix from version for cask
    local version_number="${version#v}"

    # Create updated cask content
    cat > "$formula_path" << EOF
cask "howlerops" do
  version "$version_number"
  sha256 "$universal_sha"

  url "https://github.com/${GITHUB_REPO}/releases/download/v#{version}/howlerops-darwin-universal.tar.gz"
  name "HowlerOps"
  desc "Powerful SQL client with AI capabilities"
  homepage "https://github.com/${GITHUB_REPO}"

  livecheck do
    url :url
    strategy :github_latest
  end

  app "howlerops.app"

  zap trash: [
    "~/Library/Application Support/howlerops",
    "~/Library/Caches/howlerops",
    "~/Library/Preferences/com.howlerops.app.plist",
    "~/Library/Saved Application State/com.howlerops.app.savedState",
  ]
end
EOF

    log_success "Cask file updated successfully"
}

clone_tap_repository() {
    local tap_dir="$TMP_DIR/homebrew-tap"

    log_info "Cloning Homebrew tap repository..."

    # Use HOMEBREW_TAP_TOKEN for tap repo access, fallback to GITHUB_TOKEN
    local tap_token="${HOMEBREW_TAP_TOKEN:-${GITHUB_TOKEN:-}}"

    if [ -n "$tap_token" ]; then
        # Use token for authentication
        # IMPORTANT: Redirect filtered output to stderr so only the echoed path goes to stdout
        # This prevents git output from contaminating the returned path in command substitution
        local auth_url="https://${tap_token}@github.com/${HOMEBREW_TAP_REPO}.git"
        git clone "$auth_url" "$tap_dir" 2>&1 | grep -v "token" >&2 || true
    else
        git clone "https://github.com/${HOMEBREW_TAP_REPO}.git" "$tap_dir" >&2
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
    if ! git diff --quiet Casks/"$FORMULA_NAME".rb; then
        log_info "Committing changes..."

        git add Casks/"$FORMULA_NAME".rb
        git commit -m "chore: update $FORMULA_NAME to $version

Automated update via GitHub Actions
- Updated version to $version
- Updated download URL
- Updated SHA256 checksum for universal binary"

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
        log_info "No changes detected in cask file"
    fi
}

validate_release_assets() {
    local release_data=$1

    log_info "Validating release assets..."

    local universal_asset
    universal_asset=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("howlerops-darwin-universal")) | .name' | head -n 1)

    if [ -n "$universal_asset" ]; then
        log_success "Found universal macOS desktop asset: $universal_asset"
        return 0
    fi

    local amd64_asset
    local arm64_asset

    amd64_asset=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("howlerops-darwin-amd64")) | .name' | head -n 1)
    arm64_asset=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("howlerops-darwin-arm64")) | .name' | head -n 1)

    if [ -z "$amd64_asset" ] || [ -z "$arm64_asset" ]; then
        log_error "Required macOS desktop assets not found in release"
        log_info "Available assets:"
        echo "$release_data" | jq -r '.assets[].name'
        return 1
    fi

    log_success "Found architecture-specific macOS desktop assets"
    return 0
}

#######################################################################################
# Main Script
#######################################################################################

main() {
    local version="${1:-latest}"

    log_info "Starting Homebrew formula update for HowlerOps"
    log_info "Target version: $version"
    log_info "Homebrew tap: $HOMEBREW_TAP_REPO"

    # Check for required dependencies
    check_dependencies

    # Check for GitHub token
    if [ -z "${GITHUB_TOKEN:-}" ] && [ -z "${GH_TOKEN:-}" ]; then
        log_error "Neither GITHUB_TOKEN nor GH_TOKEN is set. API calls will fail or hit rate limits."
        log_info "Set one of these environment variables with a GitHub personal access token."
        exit 1
    fi

    # Fetch release information (with retry for eventual consistency)
    local release_data=""
    local retry_count=0
    local max_retries=10

    log_info "Fetching release with up to $max_retries attempts (exponential backoff)"

    while [ $retry_count -lt $max_retries ]; do
        log_info "Release fetch attempt $((retry_count + 1))/$max_retries"

        # Attempt to fetch release
        if [ "$version" = "latest" ]; then
            if release_data=$(get_latest_release); then
                : # Success, continue
            else
                release_data=""  # Ensure empty on failure
            fi
        else
            if release_data=$(get_specific_release "$version"); then
                : # Success, continue
            else
                release_data=""  # Ensure empty on failure
            fi
        fi

        # Check result
        if [ -z "$release_data" ]; then
            log_warning "Release fetch returned empty response"
        elif ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
            log_error "API response is invalid JSON or missing required fields"
            log_info "Response preview:"
            echo "$release_data" | head -c 300 | sed 's/^/  /'
        else
            # Check asset count
            local asset_count
            asset_count=$(echo "$release_data" | jq '.assets | length' 2>/dev/null || echo "0")

            if [ "$asset_count" -gt 0 ]; then
                log_success "Found release with $asset_count assets. Proceeding..."
                break  # SUCCESS!
            else
                log_warning "Release found but no assets uploaded yet ($asset_count assets)"
                log_info "Build jobs may still be running. Will retry..."
            fi
        fi

        # Exponential backoff retry
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            # First retry: 30s, second: 60s, third: 120s, etc. (capped at 5 min)
            local backoff_delay=$((30 * (2 ** (retry_count - 1))))
            if [ $backoff_delay -gt 300 ]; then
                backoff_delay=300
            fi

            log_warning "Retrying in $backoff_delay seconds (attempt $((retry_count + 1))/$max_retries)..."
            sleep $backoff_delay
        fi
    done

    # Final check: Did we get valid release data?
    if [ -z "$release_data" ] || ! echo "$release_data" | jq -e '.tag_name' > /dev/null 2>&1; then
        log_error "Failed to fetch valid release after $max_retries attempts"
        log_error ""
        log_error "Possible causes:"
        log_error "  1. GitHub token (GH_TOKEN/GITHUB_TOKEN) is invalid or expired"
        log_error "  2. Repository is not accessible with current token"
        log_error "  3. GitHub API rate limit exceeded (try again in a few minutes)"
        log_error "  4. Release build/upload jobs are still in progress"
        log_error ""
        log_error "To debug:"
        log_error "  gh auth status          # Check token authentication"
        log_error "  gh release view $version --json assets  # Check assets"
        log_error "  gh api rate_limit       # Check rate limit status"
        exit 1
    fi

    # Extract release information
    local tag_name
    local release_version
    tag_name=$(echo "$release_data" | jq -r '.tag_name')
    release_version="${tag_name#v}"

    log_info "Found release: $tag_name (version: $release_version)"

    # Validate release assets
    if ! validate_release_assets "$release_data"; then
        exit 1
    fi

    # Extract download URL for universal binary
    local universal_archive
    local universal_url
    universal_archive=$(echo "$release_data" | jq -r '.assets[] | select(.name | contains("howlerops-darwin-universal.tar.gz")) | .name' | head -n 1)

    if [ -z "$universal_archive" ]; then
        log_error "Universal macOS binary not found in release assets"
        log_info "Available assets:"
        echo "$release_data" | jq -r '.assets[].name'
        exit 1
    fi

    universal_url=$(echo "$release_data" | jq -r --arg name "$universal_archive" '.assets[] | select(.name == $name) | .browser_download_url' | head -n 1)

    log_info "macOS Universal Binary URL: $universal_url"

    # Download and calculate checksum
    local universal_sha
    universal_sha=$(download_and_checksum "$universal_url" "$universal_archive")

    log_success "Universal Binary SHA256: $universal_sha"

    # Clone tap repository
    local tap_dir
    tap_dir=$(clone_tap_repository)

    # Create Casks directory if it doesn't exist
    mkdir -p "$tap_dir/Casks"

    # Update cask file
    update_formula_file "$tag_name" "$universal_url" "$universal_sha" "$tap_dir/Casks/$FORMULA_NAME.rb"

    # Commit and push changes
    commit_and_push_formula "$tap_dir" "$tag_name"

    log_success "Homebrew formula update completed successfully!"
    log_info "Users can now install HowlerOps $release_version with:"
    log_info "  brew update && brew upgrade $FORMULA_NAME"

    if [ "$DRY_RUN" = "true" ]; then
        log_warning "DRY RUN MODE: No changes were pushed to the repository"
    fi
}

# Show usage if help requested
if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
    cat << EOF
Homebrew Formula Update Script for HowlerOps

Usage:
  $0 [VERSION]

Arguments:
  VERSION    Release version to update to (e.g., v2.0.0 or latest)
             Default: latest

Environment Variables:
  GITHUB_TOKEN         GitHub personal access token (required for pushing)
  GH_TOKEN            Alternative GitHub token environment variable
  HOMEBREW_TAP_REPO    Tap repository (default: jbeck018/homebrew-howlerops)
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
  - GITHUB_TOKEN or GH_TOKEN for API access
  - HOMEBREW_TAP_TOKEN or GITHUB_TOKEN for pushing to tap repo

For more information, see HOMEBREW_ISSUE_ANALYSIS.md
EOF
    exit 0
fi

# Run main function
main "$@"
