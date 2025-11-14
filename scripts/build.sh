#!/bin/bash

# HowlerOps Build Script
# This script builds the application for production

set -e

echo "🔨 Building HowlerOps Desktop Application"
echo "=========================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
BUILD_DIR="build"
DIST_DIR="dist"
VERSION=$(grep '"productVersion"' wails.json | cut -d'"' -f4 || echo "1.0.0")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GO_CACHE_DIR="${GOCACHE:-$(pwd)/.gocache}"
export GOCACHE="$GO_CACHE_DIR"

# Default values
PLATFORM="current"
CLEAN=false
SKIP_FRONTEND=false
DEBUG=false
PACKAGE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --clean)
            CLEAN=true
            shift
            ;;
        --skip-frontend)
            SKIP_FRONTEND=true
            shift
            ;;
        --debug)
            DEBUG=true
            shift
            ;;
        --package)
            PACKAGE=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --platform PLATFORM  Target platform (current, all, darwin, windows, linux)"
            echo "  --clean              Clean build directory before building"
            echo "  --skip-frontend      Skip frontend build"
            echo "  --debug              Build in debug mode"
            echo "  --package            Create distribution packages"
            echo "  -h, --help           Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                           # Build for current platform"
            echo "  $0 --platform all           # Build for all platforms"
            echo "  $0 --platform darwin        # Build for macOS"
            echo "  $0 --clean --package        # Clean build and create packages"
            exit 0
            ;;
        *)
            print_error "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check dependencies
check_dependencies() {
    print_status "Checking build dependencies..."

    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi

    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed. Please install Node.js 18 or later."
        exit 1
    fi

    if ! command -v wails &> /dev/null; then
        print_error "Wails CLI not found. Please install with: go install github.com/wailsapp/wails/v2/cmd/wails@latest"
        exit 1
    fi

    print_success "All dependencies are available"
}

# Clean build directory
clean_build() {
    if [[ "$CLEAN" == true ]]; then
        print_status "Cleaning build directory..."
        rm -rf "$BUILD_DIR"
        rm -rf "$DIST_DIR"
        rm -rf frontend/dist
        wails clean || true
        print_success "Build directory cleaned"
    fi
}

# Install dependencies
install_dependencies() {
    print_status "Installing Go dependencies..."
    go mod download
    go mod tidy

    if [[ "$SKIP_FRONTEND" != true ]]; then
        print_status "Installing frontend dependencies..."
        cd frontend
        npm ci --silent
        cd ..
    fi

    print_success "Dependencies installed"
}

# Build frontend
build_frontend() {
    if [[ "$SKIP_FRONTEND" != true ]]; then
        print_status "Building frontend..."
        cd frontend
        npm run build
        cd ..
        print_success "Frontend build completed"
    else
        print_warning "Skipping frontend build"
    fi
}

# Build for current platform
build_current() {
    print_status "Building for current platform..."

    local build_flags=""
    if [[ "$DEBUG" == true ]]; then
        build_flags="-debug"
    fi

    local ldflags="-w -s -X main.version=$VERSION -X main.buildDate=$BUILD_DATE -X main.commitHash=$COMMIT_HASH"

    wails build $build_flags -clean -ldflags "$ldflags"

    print_success "Build completed for current platform"
}

# Build for specific platform
build_platform() {
    local platform=$1
    print_status "Building for $platform..."

    local build_flags=""
    if [[ "$DEBUG" == true ]]; then
        build_flags="-debug"
    fi

    local ldflags="-w -s -X main.version=$VERSION -X main.buildDate=$BUILD_DATE -X main.commitHash=$COMMIT_HASH"

    case $platform in
        darwin)
            wails build $build_flags -clean -platform darwin/amd64 -ldflags "$ldflags" -o "$BUILD_DIR/darwin/amd64/"
            wails build $build_flags -clean -platform darwin/arm64 -ldflags "$ldflags" -o "$BUILD_DIR/darwin/arm64/"
            ;;
        windows)
            wails build $build_flags -clean -platform windows/amd64 -ldflags "$ldflags" -o "$BUILD_DIR/windows/amd64/"
            ;;
        linux)
            wails build $build_flags -clean -platform linux/amd64 -ldflags "$ldflags" -o "$BUILD_DIR/linux/amd64/"
            ;;
        all)
            build_platform darwin
            build_platform windows
            build_platform linux
            return
            ;;
        *)
            print_error "Unsupported platform: $platform"
            exit 1
            ;;
    esac

    print_success "Build completed for $platform"
}

# Create distribution packages
create_packages() {
    if [[ "$PACKAGE" != true ]]; then
        return
    fi

    print_status "Creating distribution packages..."
    mkdir -p "$DIST_DIR"

    # Package macOS builds
    if [[ -d "$BUILD_DIR/darwin" ]]; then
        print_status "Packaging macOS builds..."

        if [[ -d "$BUILD_DIR/darwin/amd64" ]]; then
            cd "$BUILD_DIR/darwin/amd64"
            tar -czf "../../../$DIST_DIR/howlerops-$VERSION-darwin-amd64.tar.gz" ./*
            cd - > /dev/null
        fi

        if [[ -d "$BUILD_DIR/darwin/arm64" ]]; then
            cd "$BUILD_DIR/darwin/arm64"
            tar -czf "../../../$DIST_DIR/howlerops-$VERSION-darwin-arm64.tar.gz" ./*
            cd - > /dev/null
        fi
    fi

    # Package Windows builds
    if [[ -d "$BUILD_DIR/windows/amd64" ]]; then
        print_status "Packaging Windows builds..."
        cd "$BUILD_DIR/windows/amd64"
        tar -czf "../../../$DIST_DIR/howlerops-$VERSION-windows-amd64.tar.gz" ./*
        cd - > /dev/null
    fi

    # Package Linux builds
    if [[ -d "$BUILD_DIR/linux/amd64" ]]; then
        print_status "Packaging Linux builds..."
        cd "$BUILD_DIR/linux/amd64"
        tar -czf "../../../$DIST_DIR/howlerops-$VERSION-linux-amd64.tar.gz" ./*
        cd - > /dev/null
    fi

    print_success "Distribution packages created in $DIST_DIR/"
}

# Show build summary
show_summary() {
    print_success "Build Summary"
    echo "============="
    echo "Version: $VERSION"
    echo "Build Date: $BUILD_DATE"
    echo "Commit: $COMMIT_HASH"
    echo "Platform: $PLATFORM"
    echo "Debug Mode: $DEBUG"
    echo ""

    if [[ -d "$BUILD_DIR" ]]; then
        echo "Build outputs:"
        find "$BUILD_DIR" -name "howlerops*" -o -name "*.exe" -o -name "*.app" | while read -r file; do
            echo "  - $file"
        done
    fi

    if [[ -d "$DIST_DIR" ]]; then
        echo ""
        echo "Distribution packages:"
        find "$DIST_DIR" -name "*.zip" -o -name "*.tar.gz" | while read -r file; do
            echo "  - $file"
        done
    fi
}

# Main execution
main() {
    # Check if we're in the right directory
    if [[ ! -f "wails.json" ]]; then
        print_error "wails.json not found. Please run this script from the project root."
        exit 1
    fi

    print_status "Starting build process..."
    print_status "Version: $VERSION"
    print_status "Platform: $PLATFORM"
    print_status "Debug: $DEBUG"
    echo ""

    check_dependencies
    clean_build
    install_dependencies
    build_frontend

    case $PLATFORM in
        current)
            build_current
            ;;
        all|darwin|windows|linux)
            build_platform "$PLATFORM"
            ;;
        *)
            print_error "Invalid platform: $PLATFORM"
            exit 1
            ;;
    esac

    create_packages
    show_summary

    print_success "Build process completed successfully!"
}

# Handle interrupts gracefully
trap 'echo -e "\n${YELLOW}Build process interrupted.${NC}"; exit 1' INT

# Run main function
main "$@"
