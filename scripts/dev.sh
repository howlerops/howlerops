#!/bin/bash

# HowlerOps Development Script
# This script sets up and runs the development environment

set -e

echo "🚀 Starting HowlerOps Desktop Development Environment"
echo "=================================================="

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

# Check if required tools are installed
check_dependencies() {
    print_status "Checking dependencies..."

    # Check Go
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed. Please install Go 1.21 or later."
        exit 1
    fi

    # Check Node.js
    if ! command -v node &> /dev/null; then
        print_error "Node.js is not installed. Please install Node.js 18 or later."
        exit 1
    fi

    # Check Wails
    if ! command -v wails &> /dev/null; then
        print_warning "Wails CLI not found. Installing..."
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
        if ! command -v wails &> /dev/null; then
            print_error "Failed to install Wails CLI. Please install manually."
            exit 1
        fi
    fi

    print_success "All dependencies are available"
}

# Install Go dependencies
install_go_deps() {
    print_status "Installing Go dependencies..."
    go mod download
    go mod tidy
    print_success "Go dependencies installed"
}

# Install frontend dependencies
install_frontend_deps() {
    print_status "Installing frontend dependencies..."
    cd frontend
    npm install
    cd ..
    print_success "Frontend dependencies installed"
}

# Generate protobuf files
generate_proto() {
    print_status "Generating protobuf files..."
    cd frontend
    if npm run proto:build; then
        print_success "Protobuf files generated"
    else
        print_warning "Failed to generate protobuf files (this may be OK for first run)"
    fi
    cd ..
}

# Run development server
run_dev_server() {
    print_status "Starting Wails development server..."
    print_status "The application will open automatically when ready."
    print_status "Use Ctrl+C to stop the development server."
    echo ""

    wails dev
}

# Main execution
main() {
    # Parse command line arguments
    SKIP_DEPS=false
    SKIP_PROTO=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-deps)
                SKIP_DEPS=true
                shift
                ;;
            --skip-proto)
                SKIP_PROTO=true
                shift
                ;;
            -h|--help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --skip-deps   Skip dependency installation"
                echo "  --skip-proto  Skip protobuf generation"
                echo "  -h, --help    Show this help message"
                exit 0
                ;;
            *)
                print_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Check if we're in the right directory
    if [[ ! -f "wails.json" ]]; then
        print_error "wails.json not found. Please run this script from the project root."
        exit 1
    fi

    # Execute steps
    check_dependencies

    if [[ "$SKIP_DEPS" != true ]]; then
        install_go_deps
        install_frontend_deps
    fi

    if [[ "$SKIP_PROTO" != true ]]; then
        generate_proto
    fi

    print_success "Development environment setup complete!"
    echo ""

    run_dev_server
}

# Handle interrupts gracefully
trap 'echo -e "\n${YELLOW}Development server stopped.${NC}"; exit 0' INT

# Run main function
main "$@"