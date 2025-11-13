#!/bin/bash

# Safer errcheck fix script - only fixes patterns we're 100% sure about

set -e

echo "Starting errcheck fixes..."

# 1. Fix defer rows.Close() - very specific pattern
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" -exec \
  sed -i.bak 's/^\(\s*\)defer rows\.Close()$/\1defer func() { _ = rows.Close() }() \/\/ Best-effort close/g' {} \;

# 2. Fix defer tx.Rollback() - very specific pattern
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" -exec \
  sed -i.bak 's/^\(\s*\)defer tx\.Rollback()$/\1defer func() { _ = tx.Rollback() }() \/\/ Best-effort rollback/g' {} \;

# 3. Fix defer r.Body.Close() in HTTP handlers
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" -exec \
  sed -i.bak 's/^\(\s*\)defer r\.Body\.Close()$/\1defer func() { _ = r.Body.Close() }() \/\/ Best-effort close/g' {} \;

# 4. Fix defer cursor.Close(ctx)
find . -name "*.go" -type f -not -path "./vendor/*" -not -path "./.git/*" -exec \
  sed -i.bak 's/^\(\s*\)defer cursor\.Close(ctx)$/\1defer func() { _ = cursor.Close(ctx) }() \/\/ Best-effort close/g' {} \;

# Clean up backup files
find . -name "*.go.bak" -type f -delete

echo "Basic defer fixes applied."
echo "Verifying changes don't break syntax..."

# Quick syntax check
if ! go build ./... > /dev/null 2>&1; then
  echo "ERROR: Changes broke the build! Reverting..."
  git checkout -- .
  exit 1
fi

echo "Syntax check passed! Checking remaining errcheck issues..."
golangci-lint run --enable-only=errcheck ./... 2>&1 | head -50
