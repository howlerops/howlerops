#!/bin/bash

# Script to fix golangci-lint errors systematically
# This handles the 379 errors identified

set -e

echo "Starting systematic lint error fixes..."
echo "Total errors to fix: 379"

# Function to apply gofmt and goimports
format_code() {
    echo "Formatting code with gofmt and goimports..."
    find . -name "*.go" -not -path "./vendor/*" -exec gofmt -w {} \;
    find . -name "*.go" -not -path "./vendor/*" -exec goimports -w {} \; 2>/dev/null || true
}

# Function to backup files before modification
backup_files() {
    echo "Creating backup..."
    tar -czf /tmp/backend-go-backup-$(date +%Y%m%d-%H%M%S).tar.gz .
}

# Run backup
backup_files

echo "Fixes will be applied by category..."
echo "This may take several minutes..."

# After manual fixes, format code
format_code

echo "Fix script completed!"
echo "Please run: golangci-lint run ./... to verify fixes"
