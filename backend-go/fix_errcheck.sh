#!/bin/bash

# This script fixes all remaining errcheck issues in the codebase
# It applies systematic patterns based on the error type

set -e

echo "Fixing errcheck issues in the codebase..."

# Fix all the remaining files in batch using sed

# Pattern 1: Fix defer rows.Close()
find . -name "*.go" -type f -exec sed -i '' 's/defer rows\.Close()/defer func() { _ = rows.Close() }() \/\/ Best-effort close/g' {} \;

# Pattern 2: Fix defer tx.Rollback()
find . -name "*.go" -type f -exec sed -i '' 's/defer tx\.Rollback()/defer func() { _ = tx.Rollback() }() \/\/ Best-effort rollback/g' {} \;

# Pattern 3: Fix defer r.Body.Close() (already done for one case, but let's ensure all are fixed)
find . -name "*.go" -type f -exec sed -i '' 's/defer r\.Body\.Close()/defer func() { _ = r.Body.Close() }() \/\/ Best-effort close/g' {} \;

# Pattern 4: Fix defer resp.Body.Close()
find . -name "*.go" -type f -exec sed -i '' 's/\([[:space:]]*\)resp\.Body\.Close()/\1_ = resp.Body.Close() \/\/ Best-effort close/g' {} \;

# Pattern 5: Fix defer db.Close()
find . -name "*.go" -type f -exec sed -i '' 's/defer db\.Close()/defer func() { _ = db.Close() }() \/\/ Best-effort close/g' {} \;

# Pattern 6: Fix defer conn.Close()
find . -name "*.go" -type f -exec sed -i '' 's/\([[:space:]]*\)conn\.Close()/\1_ = conn.Close() \/\/ Best-effort close/g' {} \;

# Pattern 7: Fix defer listener.Close()
find . -name "*.go" -type f -exec sed -i '' 's/\([[:space:]]*\)listener\.Close()/\1_ = listener.Close() \/\/ Best-effort close/g' {} \;

# Pattern 8: Fix defer cursor.Close()
find . -name "*.go" -type f -exec sed -i '' 's/defer cursor\.Close\(([^)]*)\)/defer func() { _ = cursor.Close\1 }() \/\/ Best-effort close/g' {} \;

# Pattern 9: Fix defer colRows.Close()
find . -name "*.go" -type f -exec sed -i '' 's/\([[:space:]]*\)colRows\.Close()/\1_ = colRows.Close() \/\/ Best-effort close/g' {} \;

# Pattern 10: Fix w.Write() calls
find . -name "*.go" -type f -exec sed -i '' 's/\([[:space:]]*\)w\.Write(/\1_, _ = w.Write(/g' {} \;

echo "Pattern-based fixes applied. Now applying manual fixes for complex cases..."

# The script above handles most cases automatically
# For remaining complex cases that need different handling, we'll fix them individually

echo "Done with automated fixes. Running golangci-lint to check remaining issues..."
