#!/bin/bash

# Wails Installation Script for HowlerOps
# This script installs Wails and its dependencies

set -e

echo "🔧 Installing Wails and Dependencies for HowlerOps"
echo "================================================="

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

# Detect operating system
detect_os() {
    case "$(uname -s)" in
        Darwin*)
            OS="macos"
            ;;
        Linux*)
            OS="linux"
            ;;
        CYGWIN*|MINGW32*|MSYS*|MINGW*)
            OS="windows"
            ;;
        *)
            print_error "Unsupported operating system: $(uname -s)"
            exit 1
            ;;
    esac
    print_status "Detected OS: $OS"
}

# Check if Go is installed
check_go() {
    print_status "Checking Go installation..."

    if command -v go &> /dev/null; then
        GO_VERSION=$(go version | cut -d' ' -f3 | cut -c3-)
        print_success "Go is installed: $GO_VERSION"

        # Check if Go version is 1.21 or later
        if ! go version | grep -E "go1\.(2[1-9]|[3-9][0-9])" &> /dev/null; then
            print_warning "Go version 1.21 or later is recommended. Current: $GO_VERSION"
        fi
    else
        print_error "Go is not installed. Please install Go 1.21 or later from https://golang.org/dl/"
        print_status "Installation instructions:"
        case $OS in
            macos)
                echo "  - Download and install from https://golang.org/dl/"
                echo "  - Or use Homebrew: brew install go"
                ;;
            linux)
                echo "  - Download and install from https://golang.org/dl/"
                echo "  - Or use package manager: sudo apt install golang-go (Ubuntu/Debian)"
                echo "  - Or use package manager: sudo yum install golang (RHEL/CentOS)"
                ;;
            windows)
                echo "  - Download and install from https://golang.org/dl/"
                echo "  - Or use Chocolatey: choco install golang"
                echo "  - Or use Scoop: scoop install go"
                ;;
        esac
        exit 1
    fi
}

# Check if Node.js is installed
check_node() {
    print_status "Checking Node.js installation..."

    if command -v node &> /dev/null; then
        NODE_VERSION=$(node --version)
        print_success "Node.js is installed: $NODE_VERSION"

        # Check if Node.js version is 18 or later
        NODE_MAJOR=$(echo $NODE_VERSION | cut -d'.' -f1 | cut -c2-)
        if [ "$NODE_MAJOR" -lt 18 ]; then
            print_warning "Node.js version 18 or later is recommended. Current: $NODE_VERSION"
        fi
    else
        print_error "Node.js is not installed. Please install Node.js 18 or later from https://nodejs.org/"
        print_status "Installation instructions:"
        case $OS in
            macos)
                echo "  - Download and install from https://nodejs.org/"
                echo "  - Or use Homebrew: brew install node"
                echo "  - Or use n: curl -L https://bit.ly/n-install | bash"
                ;;
            linux)
                echo "  - Download and install from https://nodejs.org/"
                echo "  - Or use package manager: sudo apt install nodejs npm (Ubuntu/Debian)"
                echo "  - Or use NodeSource: curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -"
                echo "  - Or use n: curl -L https://bit.ly/n-install | bash"
                ;;
            windows)
                echo "  - Download and install from https://nodejs.org/"
                echo "  - Or use Chocolatey: choco install nodejs"
                echo "  - Or use Scoop: scoop install nodejs"
                ;;
        esac
        exit 1
    fi
}

# Install platform-specific dependencies
install_platform_deps() {
    print_status "Installing platform-specific dependencies for $OS..."

    case $OS in
        macos)
            # Check for Xcode command line tools
            if ! xcode-select -p &> /dev/null; then
                print_status "Installing Xcode command line tools..."
                xcode-select --install
                print_success "Xcode command line tools installation started"
                print_warning "Please complete the Xcode installation and run this script again"
                exit 1
            else
                print_success "Xcode command line tools are installed"
            fi
            ;;
        linux)
            # Check for common build tools
            if command -v apt-get &> /dev/null; then
                print_status "Installing build dependencies (Debian/Ubuntu)..."
                sudo apt-get update
                sudo apt-get install -y build-essential pkg-config libgtk-3-dev libwebkit2gtk-4.0-dev
            elif command -v yum &> /dev/null; then
                print_status "Installing build dependencies (RHEL/CentOS)..."
                sudo yum groupinstall -y "Development Tools"
                sudo yum install -y pkg-config gtk3-devel webkit2gtk3-devel
            elif command -v dnf &> /dev/null; then
                print_status "Installing build dependencies (Fedora)..."
                sudo dnf groupinstall -y "Development Tools"
                sudo dnf install -y pkg-config gtk3-devel webkit2gtk3-devel
            elif command -v pacman &> /dev/null; then
                print_status "Installing build dependencies (Arch Linux)..."
                sudo pacman -S --needed base-devel pkg-config gtk3 webkit2gtk
            else
                print_warning "Unknown Linux distribution. Please install build tools manually:"
                echo "  - gcc/g++ compiler"
                echo "  - pkg-config"
                echo "  - GTK3 development headers"
                echo "  - WebKit2GTK development headers"
            fi
            ;;
        windows)
            print_status "Windows detected. Please ensure you have:"
            echo "  - Microsoft C++ Build Tools or Visual Studio with C++ workload"
            echo "  - Windows SDK"
            print_warning "WebView2 will be automatically handled by Wails"
            ;;
    esac
}

# Install Wails CLI
install_wails() {
    print_status "Installing Wails CLI..."

    if command -v wails &> /dev/null; then
        WAILS_VERSION=$(wails version | grep "Wails:" | cut -d' ' -f2)
        print_success "Wails is already installed: $WAILS_VERSION"

        print_status "Updating Wails to latest version..."
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
    else
        print_status "Installing Wails CLI..."
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
    fi

    # Verify installation
    if command -v wails &> /dev/null; then
        WAILS_VERSION=$(wails version | grep "Wails:" | cut -d' ' -f2)
        print_success "Wails CLI installed successfully: $WAILS_VERSION"
    else
        print_error "Failed to install Wails CLI"
        print_status "Please ensure your GOPATH/bin is in your PATH"
        print_status "Add this to your shell profile (.bashrc, .zshrc, etc.):"
        echo "  export PATH=\$PATH:\$(go env GOPATH)/bin"
        exit 1
    fi
}

# Run Wails doctor
run_wails_doctor() {
    print_status "Running Wails doctor to check installation..."

    if command -v wails &> /dev/null; then
        echo ""
        wails doctor
        echo ""
        print_success "Wails doctor completed"
    else
        print_error "Wails CLI not found"
        exit 1
    fi
}

# Show next steps
show_next_steps() {
    print_success "Installation completed successfully!"
    echo ""
    echo "🎉 Next Steps:"
    echo "=============="
    echo ""
    echo "1. Navigate to the HowlerOps project directory:"
    echo "   cd sql-studio"
    echo ""
    echo "2. Install project dependencies:"
    echo "   ./scripts/dev.sh --skip-deps  # (if you just want to check setup)"
    echo "   # or"
    echo "   make deps"
    echo ""
    echo "3. Start development server:"
    echo "   ./scripts/dev.sh"
    echo "   # or"
    echo "   make dev"
    echo ""
    echo "4. Build for production:"
    echo "   ./scripts/build.sh"
    echo "   # or"
    echo "   make build"
    echo ""
    echo "📚 Documentation:"
    echo "- Wails: https://wails.io/docs/"
    echo "- HowlerOps: ./README-WAILS.md"
    echo ""
    print_success "Happy coding! 🚀"
}

# Main execution
main() {
    echo "This script will install Wails and its dependencies for HowlerOps development."
    echo ""

    detect_os
    check_go
    check_node
    install_platform_deps
    install_wails
    run_wails_doctor
    show_next_steps
}

# Handle interrupts gracefully
trap 'echo -e "\n${YELLOW}Installation interrupted.${NC}"; exit 1' INT

# Run main function
main "$@"