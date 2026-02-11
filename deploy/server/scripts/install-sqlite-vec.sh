#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
LIBS_DIR="$PROJECT_ROOT/libs"

mkdir -p "$LIBS_DIR"

VERSION="0.1.6"
PLATFORM=""
ARCH=""

# Detect platform and architecture
case "$(uname -s)" in
    Darwin*)
        PLATFORM="macos"
        case "$(uname -m)" in
            arm64) ARCH="aarch64" ;;
            x86_64) ARCH="x86_64" ;;
            *) echo "Unsupported architecture"; exit 1 ;;
        esac
        ;;
    Linux*)
        PLATFORM="linux"
        case "$(uname -m)" in
            x86_64) ARCH="x86_64" ;;
            aarch64) ARCH="aarch64" ;;
            *) echo "Unsupported architecture"; exit 1 ;;
        esac
        ;;
    MINGW*|MSYS*|CYGWIN*)
        PLATFORM="windows"
        ARCH="x86_64"
        ;;
    *)
        echo "Unsupported platform"
        exit 1
        ;;
esac

# Download URL
FILENAME="sqlite-vec-${VERSION}-loadable-${PLATFORM}-${ARCH}.tar.gz"
URL="https://github.com/asg017/sqlite-vec/releases/download/v${VERSION}/${FILENAME}"

echo "Downloading sqlite-vec for ${PLATFORM}-${ARCH}..."
echo "URL: $URL"

cd "$LIBS_DIR"
curl -L -o "$FILENAME" "$URL"
tar -xzf "$FILENAME"
rm "$FILENAME"

echo "sqlite-vec installed successfully to $LIBS_DIR"
ls -la "$LIBS_DIR"
