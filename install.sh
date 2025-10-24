#!/bin/sh
# SQL Studio Universal Installer
# Inspired by rustup, deno, and bun installation best practices
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh
#
# Options:
#   -v, --verbose       Enable verbose output
#   --dry-run          Show what would be done without making changes
#   --force            Overwrite existing installation
#   --version VERSION  Install a specific version (default: latest)
#   --install-dir DIR  Custom installation directory

set -e

# ============================================================================
# Configuration
# ============================================================================

GITHUB_REPO="sql-studio/sql-studio"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${VERSION:-latest}"
VERBOSE="${VERBOSE:-0}"
DRY_RUN="${DRY_RUN:-0}"
FORCE="${FORCE:-0}"

# Colors (disable in non-interactive or unsupported terminals)
if [ -t 1 ] && command -v tput >/dev/null 2>&1 && tput colors >/dev/null 2>&1; then
    RED=$(tput setaf 1)
    GREEN=$(tput setaf 2)
    YELLOW=$(tput setaf 3)
    BLUE=$(tput setaf 4)
    MAGENTA=$(tput setaf 5)
    CYAN=$(tput setaf 6)
    BOLD=$(tput bold)
    RESET=$(tput sgr0)
else
    RED=""
    GREEN=""
    YELLOW=""
    BLUE=""
    MAGENTA=""
    CYAN=""
    BOLD=""
    RESET=""
fi

# ============================================================================
# Utility Functions
# ============================================================================

log() {
    echo "${BLUE}[INFO]${RESET} $*"
}

log_verbose() {
    if [ "$VERBOSE" -eq 1 ]; then
        echo "${CYAN}[VERBOSE]${RESET} $*"
    fi
}

log_success() {
    echo "${GREEN}[SUCCESS]${RESET} $*"
}

log_warn() {
    echo "${YELLOW}[WARN]${RESET} $*" >&2
}

log_error() {
    echo "${RED}[ERROR]${RESET} $*" >&2
}

fail() {
    log_error "$*"
    exit 1
}

dry_run() {
    if [ "$DRY_RUN" -eq 1 ]; then
        echo "${MAGENTA}[DRY-RUN]${RESET} $*"
        return 0
    fi
    return 1
}

command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# ============================================================================
# Platform Detection
# ============================================================================

detect_os() {
    local os
    os="$(uname -s)"

    case "$os" in
        Linux*)
            echo "linux"
            ;;
        Darwin*)
            echo "darwin"
            ;;
        MINGW* | MSYS* | CYGWIN*)
            echo "windows"
            ;;
        *)
            fail "Unsupported operating system: $os"
            ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"

    case "$arch" in
        x86_64 | amd64)
            echo "amd64"
            ;;
        aarch64 | arm64)
            echo "arm64"
            ;;
        armv7l | armv6l)
            echo "arm"
            ;;
        i386 | i686)
            fail "32-bit systems are not supported. Please use a 64-bit system."
            ;;
        *)
            fail "Unsupported architecture: $arch"
            ;;
    esac
}

get_platform() {
    local os arch
    os="$(detect_os)"
    arch="$(detect_arch)"
    echo "${os}-${arch}"
}

# ============================================================================
# Dependency Checks
# ============================================================================

check_dependencies() {
    log "Checking dependencies..."

    local missing_deps=""

    # Check for curl or wget
    if ! command_exists curl && ! command_exists wget; then
        missing_deps="${missing_deps}  - curl or wget\n"
    fi

    # Check for tar
    if ! command_exists tar; then
        missing_deps="${missing_deps}  - tar\n"
    fi

    # Check for shasum or sha256sum
    if ! command_exists shasum && ! command_exists sha256sum; then
        missing_deps="${missing_deps}  - shasum or sha256sum\n"
    fi

    if [ -n "$missing_deps" ]; then
        log_error "Missing required dependencies:"
        printf "%b" "$missing_deps"
        fail "Please install the missing dependencies and try again."
    fi

    log_verbose "All dependencies satisfied"
}

# ============================================================================
# Version Resolution
# ============================================================================

get_latest_version() {
    local url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"

    log_verbose "Fetching latest version from GitHub API..."

    local version
    if command_exists curl; then
        version="$(curl -fsSL "$url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')"
    elif command_exists wget; then
        version="$(wget -qO- "$url" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')"
    else
        fail "Neither curl nor wget is available"
    fi

    echo "$version"
}

resolve_version() {
    if [ "$VERSION" = "latest" ]; then
        VERSION="$(get_latest_version)"
        if [ -z "$VERSION" ]; then
            if [ "$DRY_RUN" -eq 1 ]; then
                # In dry-run mode, use a placeholder version
                VERSION="v0.0.0-dryrun"
                log_warn "Could not fetch latest version (dry-run mode), using placeholder: ${VERSION}"
            else
                fail "Failed to determine latest version. Please:\n  - Check your internet connection\n  - Verify the repository has releases\n  - Or specify a version with --version"
            fi
        else
            log "Latest version: ${BOLD}${VERSION}${RESET}"
        fi
    else
        log "Installing version: ${BOLD}${VERSION}${RESET}"
    fi
}

# ============================================================================
# Download Functions
# ============================================================================

download_file() {
    local url="$1"
    local output="$2"

    log_verbose "Downloading $url to $output"

    if dry_run "Would download: $url"; then
        return 0
    fi

    if command_exists curl; then
        if [ "$VERBOSE" -eq 1 ]; then
            curl -fL --progress-bar "$url" -o "$output"
        else
            curl -fsSL "$url" -o "$output"
        fi
    elif command_exists wget; then
        if [ "$VERBOSE" -eq 1 ]; then
            wget --show-progress -q "$url" -O "$output"
        else
            wget -q "$url" -O "$output"
        fi
    else
        fail "Neither curl nor wget is available"
    fi
}

# ============================================================================
# Checksum Verification
# ============================================================================

verify_checksum() {
    local archive="$1"
    local checksums_file="$2"
    local binary_name="$3"

    log "Verifying checksum..."

    if dry_run "Would verify checksum for $archive"; then
        return 0
    fi

    # Extract expected checksum for our binary
    local expected_checksum
    expected_checksum="$(grep "${binary_name}.tar.gz" "$checksums_file" | awk '{print $1}')"

    if [ -z "$expected_checksum" ]; then
        log_warn "Checksum not found in checksums.txt, skipping verification"
        return 0
    fi

    log_verbose "Expected checksum: $expected_checksum"

    # Calculate actual checksum
    local actual_checksum
    if command_exists shasum; then
        actual_checksum="$(shasum -a 256 "$archive" | awk '{print $1}')"
    elif command_exists sha256sum; then
        actual_checksum="$(sha256sum "$archive" | awk '{print $1}')"
    else
        log_warn "Neither shasum nor sha256sum found, skipping checksum verification"
        return 0
    fi

    log_verbose "Actual checksum: $actual_checksum"

    if [ "$expected_checksum" != "$actual_checksum" ]; then
        fail "Checksum verification failed!\nExpected: $expected_checksum\nActual:   $actual_checksum"
    fi

    log_success "Checksum verified"
}

# ============================================================================
# Installation
# ============================================================================

determine_install_dir() {
    if [ -n "$INSTALL_DIR" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    # Prefer ~/.local/bin (no sudo required)
    local local_bin="$HOME/.local/bin"
    if [ -d "$local_bin" ] || mkdir -p "$local_bin" 2>/dev/null; then
        echo "$local_bin"
        return
    fi

    # Fall back to /usr/local/bin (requires sudo)
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    # Check if user can sudo
    if command_exists sudo && sudo -n true 2>/dev/null; then
        echo "/usr/local/bin"
        return
    fi

    fail "Cannot determine installation directory. Please specify with --install-dir"
}

install_binary() {
    local archive="$1"
    local install_dir="$2"

    log "Installing to ${BOLD}${install_dir}${RESET}..."

    if dry_run "Would install to $install_dir"; then
        return 0
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$install_dir" ]; then
        log_verbose "Creating installation directory: $install_dir"
        if ! mkdir -p "$install_dir" 2>/dev/null; then
            log_verbose "Trying with sudo..."
            sudo mkdir -p "$install_dir" || fail "Failed to create installation directory"
        fi
    fi

    # Extract archive to temp location
    local temp_extract
    temp_extract="$(mktemp -d)"
    log_verbose "Extracting to temporary directory: $temp_extract"

    tar -xzf "$archive" -C "$temp_extract" || fail "Failed to extract archive"

    # Find the binary in the extracted files
    local binary
    binary="$(find "$temp_extract" -type f -name "sql-studio*" ! -name "*.tar.gz" | head -n 1)"

    if [ -z "$binary" ] || [ ! -f "$binary" ]; then
        rm -rf "$temp_extract"
        fail "Binary not found in archive"
    fi

    local target="$install_dir/sql-studio"

    # Check if already installed
    if [ -f "$target" ] && [ "$FORCE" -eq 0 ]; then
        rm -rf "$temp_extract"
        fail "SQL Studio is already installed at $target. Use --force to overwrite."
    fi

    # Install binary
    log_verbose "Copying binary to $target"
    if ! cp "$binary" "$target" 2>/dev/null; then
        log_verbose "Trying with sudo..."
        sudo cp "$binary" "$target" || fail "Failed to install binary"
        sudo chmod +x "$target" || fail "Failed to make binary executable"
    else
        chmod +x "$target" || fail "Failed to make binary executable"
    fi

    # Clean up
    rm -rf "$temp_extract"

    log_success "Binary installed successfully"
}

# ============================================================================
# PATH Configuration
# ============================================================================

detect_shell() {
    # Try to detect current shell
    if [ -n "$BASH_VERSION" ]; then
        echo "bash"
    elif [ -n "$ZSH_VERSION" ]; then
        echo "zsh"
    elif [ -n "$FISH_VERSION" ]; then
        echo "fish"
    else
        # Fall back to parsing $SHELL
        case "$SHELL" in
            */bash)
                echo "bash"
                ;;
            */zsh)
                echo "zsh"
                ;;
            */fish)
                echo "fish"
                ;;
            *)
                echo "sh"
                ;;
        esac
    fi
}

get_shell_profile() {
    local shell="$1"

    case "$shell" in
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                echo "$HOME/.bashrc"
            elif [ -f "$HOME/.bash_profile" ]; then
                echo "$HOME/.bash_profile"
            else
                echo "$HOME/.profile"
            fi
            ;;
        zsh)
            echo "$HOME/.zshrc"
            ;;
        fish)
            echo "$HOME/.config/fish/config.fish"
            ;;
        *)
            echo "$HOME/.profile"
            ;;
    esac
}

check_path() {
    local install_dir="$1"

    # Check if install_dir is in PATH
    case ":$PATH:" in
        *":$install_dir:"*)
            log_verbose "Installation directory is already in PATH"
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

update_path() {
    local install_dir="$1"

    if check_path "$install_dir"; then
        return 0
    fi

    log_warn "Installation directory is not in your PATH"

    local shell
    shell="$(detect_shell)"
    local profile
    profile="$(get_shell_profile "$shell")"

    log ""
    log "To add SQL Studio to your PATH, run:"
    log ""

    if [ "$shell" = "fish" ]; then
        log "  ${CYAN}set -Ua fish_user_paths $install_dir${RESET}"
    else
        log "  ${CYAN}echo 'export PATH=\"$install_dir:\$PATH\"' >> $profile${RESET}"
        log "  ${CYAN}source $profile${RESET}"
    fi

    log ""
    log "Or manually add ${CYAN}$install_dir${RESET} to your PATH in ${CYAN}$profile${RESET}"
    log ""
}

# ============================================================================
# Cleanup
# ============================================================================

cleanup() {
    local temp_dir="$1"

    if [ -n "$temp_dir" ] && [ -d "$temp_dir" ]; then
        log_verbose "Cleaning up temporary files..."
        rm -rf "$temp_dir"
    fi
}

# ============================================================================
# Main Installation Flow
# ============================================================================

main() {
    echo ""
    echo "${BOLD}${BLUE}╔═══════════════════════════════════════╗${RESET}"
    echo "${BOLD}${BLUE}║    SQL Studio Universal Installer    ║${RESET}"
    echo "${BOLD}${BLUE}╚═══════════════════════════════════════╝${RESET}"
    echo ""

    # Parse command line arguments
    while [ $# -gt 0 ]; do
        case "$1" in
            -v|--verbose)
                VERBOSE=1
                shift
                ;;
            --dry-run)
                DRY_RUN=1
                log_warn "Running in dry-run mode - no changes will be made"
                shift
                ;;
            --force)
                FORCE=1
                shift
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --install-dir)
                INSTALL_DIR="$2"
                shift 2
                ;;
            -h|--help)
                cat <<EOF
SQL Studio Universal Installer

Usage:
  curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

Options:
  -v, --verbose       Enable verbose output
  --dry-run          Show what would be done without making changes
  --force            Overwrite existing installation
  --version VERSION  Install a specific version (default: latest)
  --install-dir DIR  Custom installation directory
  -h, --help         Show this help message

Environment Variables:
  INSTALL_DIR        Installation directory
  VERSION            Version to install

Examples:
  # Install latest version
  curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh

  # Install specific version
  curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --version v2.0.0

  # Install to custom directory
  curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --install-dir /usr/local/bin

  # Dry run
  curl -fsSL https://raw.githubusercontent.com/sql-studio/sql-studio/main/install.sh | sh -s -- --dry-run

For more information, visit: https://docs.sqlstudio.io
EOF
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                echo "Run with --help for usage information"
                exit 1
                ;;
        esac
    done

    # Detect platform
    local platform
    platform="$(get_platform)"
    log "Detected platform: ${BOLD}${platform}${RESET}"

    # Check dependencies
    check_dependencies

    # Resolve version
    resolve_version

    # Determine installation directory
    local install_dir
    install_dir="$(determine_install_dir)"
    log "Installation directory: ${BOLD}${install_dir}${RESET}"

    # Construct binary name and URLs
    local binary_name="sql-studio-${platform}"
    local base_url="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"
    local archive_url="${base_url}/${binary_name}.tar.gz"
    local checksums_url="${base_url}/checksums.txt"

    log_verbose "Archive URL: $archive_url"
    log_verbose "Checksums URL: $checksums_url"

    # Create temporary directory
    local temp_dir
    temp_dir="$(mktemp -d)"
    trap 'cleanup "$temp_dir"' EXIT

    local archive="${temp_dir}/${binary_name}.tar.gz"
    local checksums="${temp_dir}/checksums.txt"

    # Download archive
    log "Downloading SQL Studio ${VERSION}..."
    if ! download_file "$archive_url" "$archive"; then
        fail "Failed to download SQL Studio. Please check:\n  - Version ${VERSION} exists\n  - Platform ${platform} is supported\n  - Network connection is working"
    fi

    # Download checksums
    log_verbose "Downloading checksums..."
    if download_file "$checksums_url" "$checksums" 2>/dev/null; then
        verify_checksum "$archive" "$checksums" "$binary_name"
    else
        log_warn "Could not download checksums.txt, skipping verification"
    fi

    # Install binary
    install_binary "$archive" "$install_dir"

    # Check PATH and provide instructions if needed
    update_path "$install_dir"

    # Get installed version
    local installed_version
    if [ "$DRY_RUN" -eq 0 ] && [ -x "$install_dir/sql-studio" ]; then
        installed_version="$("$install_dir/sql-studio" --version 2>/dev/null || echo "$VERSION")"
    else
        installed_version="$VERSION"
    fi

    # Success message
    echo ""
    echo "${GREEN}${BOLD}✓ SQL Studio has been installed successfully!${RESET}"
    echo ""
    echo "  Location: ${CYAN}$install_dir/sql-studio${RESET}"
    echo "  Version:  ${CYAN}$installed_version${RESET}"
    echo ""
    echo "${BOLD}Get started:${RESET}"
    echo "  ${CYAN}sql-studio --help${RESET}"
    echo "  ${CYAN}sql-studio version${RESET}"
    echo ""
    echo "${BOLD}Documentation:${RESET} ${CYAN}https://docs.sqlstudio.io${RESET}"
    echo ""
}

# ============================================================================
# Entry Point
# ============================================================================

main "$@"
