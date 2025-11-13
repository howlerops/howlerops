#!/bin/bash

# Safe script to fix errcheck issues using gofmt with precise replacements

set -e

echo "Fixing errcheck issues..."

# Use gofmt -r for safe, syntax-aware replacements
# This only works for simple patterns - we'll do the rest manually

# Fix defer rows.Close()
gofmt -w -r 'defer rows.Close() -> defer func() { _ = rows.Close() }()' ./...

# Fix defer tx.Rollback()
gofmt -w -r 'defer tx.Rollback() -> defer func() { _ = tx.Rollback() }()' ./...

echo "Basic defer fixes applied. Checking status..."
golangci-lint run --enable-only=errcheck ./... 2>&1 | head -20
