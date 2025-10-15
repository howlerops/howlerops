#!/bin/bash

# Install sqlite-vec extension for HowlerOps
# This is OPTIONAL - the pure Go implementation works fine
# This C extension provides faster vector search for large datasets

set -e

echo "🔧 Installing sqlite-vec extension..."
echo ""
echo "Note: This is optional. HowlerOps works fine without it!"
echo "The C extension provides ~2-3x faster vector search for large datasets."
echo ""

# Check if already installed
if [ -f ~/.howlerops/extensions/vec0.so ] || [ -f ~/.howlerops/extensions/vec0.dylib ]; then
    echo "✓ sqlite-vec extension already installed"
    exit 0
fi

# Detect OS
OS=$(uname -s)
ARCH=$(uname -m)

echo "Detected: $OS $ARCH"
echo ""

# Create extensions directory
mkdir -p ~/.howlerops/extensions

case "$OS" in
    Darwin)
        EXT="dylib"
        ;;
    Linux)
        EXT="so"
        ;;
    *)
        echo "❌ Unsupported OS: $OS"
        echo "Falling back to pure Go implementation (which works great!)"
        exit 0
        ;;
esac

# Check if we can download pre-built binary
echo "Checking for pre-built sqlite-vec extension..."
RELEASE_URL="https://github.com/asg017/sqlite-vec/releases/latest/download/sqlite-vec-${OS,,}-${ARCH}.tar.gz"

if command -v curl &> /dev/null; then
    if curl -fsSL "$RELEASE_URL" -o /tmp/sqlite-vec.tar.gz 2>/dev/null; then
        echo "✓ Downloaded pre-built extension"
        tar -xzf /tmp/sqlite-vec.tar.gz -C /tmp
        cp /tmp/vec0.$EXT ~/.howlerops/extensions/
        rm -rf /tmp/sqlite-vec.tar.gz /tmp/vec0.$EXT
        echo "✓ sqlite-vec extension installed successfully"
        echo "  Location: ~/.howlerops/extensions/vec0.$EXT"
        exit 0
    fi
fi

# Fall back to building from source
echo "Pre-built binary not available, attempting to build from source..."
echo ""

# Check for build dependencies
if ! command -v git &> /dev/null; then
    echo "❌ git not found. Install git or use pure Go implementation."
    exit 1
fi

if ! command -v make &> /dev/null; then
    echo "❌ make not found. Install build tools or use pure Go implementation."
    exit 1
fi

# Clone and build
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

echo "Cloning sqlite-vec repository..."
git clone --depth 1 https://github.com/asg017/sqlite-vec.git
cd sqlite-vec

echo "Building sqlite-vec extension..."
make loadable

# Copy to extensions directory
cp vec0.$EXT ~/.howlerops/extensions/

# Cleanup
cd -
rm -rf "$TMP_DIR"

echo ""
echo "✓ sqlite-vec extension installed successfully"
echo "  Location: ~/.howlerops/extensions/vec0.$EXT"
echo ""
echo "To use it, the extension will be automatically loaded by HowlerOps."
echo "You can verify it's working by checking the logs for 'sqlite-vec loaded'."

