#!/bin/bash

# Comprehensive lint fix script for all 379 errors
# This script applies fixes systematically by category

set -e

echo "=== Applying Comprehensive Lint Fixes ==="
echo "Total errors to fix: 379"
echo ""

# Backup
echo "Creating backup..."
BACKUP_FILE="/tmp/backend-go-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
tar -czf "$BACKUP_FILE" . --exclude='.git' --exclude='vendor'
echo "Backup created: $BACKUP_FILE"
echo ""

#----------------------------------------
# 1. Fix dupword errors (2 errors)
#----------------------------------------
echo "Step 1/11: Fixing dupword errors..."
# TiDB TiDB -> TiDB
sed -i.bak 's/TiDB TiDB/TiDB/g' pkg/database/tidb.go
# user user -> user (in comments)
sed -i.bak 's/user user/user/g' pkg/storage/turso/example_integration.go

#----------------------------------------
# 2. Fix error wrapping (7 errors) - %v -> %w
#----------------------------------------
echo "Step 2/11: Fixing error wrapping issues..."

# Fix errorlint issues - replace %v with %w for error wrapping
find . -name "*.go" -not -path "./vendor/*" -exec sed -i.bak 's/fmt\.Errorf("\([^"]*\)%v", \([^,]*\), err)/fmt.Errorf("\1%w", \2, err)/g' {} \;
find . -name "*.go" -not -path "./vendor/*" -exec sed -i.bak 's/fmt\.Errorf("\([^"]*\): %v", err)/fmt.Errorf("\1: %w", err)/g' {} \;

#----------------------------------------
# 3. Fix nil context (SA1012) - 11 errors
#----------------------------------------
echo "Step 3/11: Fixing nil context issues..."

# Replace common nil context patterns with context.TODO()
find . -name "*.go" -not -path "./vendor/*" -type f -exec sed -i.bak '
s/\.Execute(nil,/\.Execute(context.TODO(),/g
s/\.Query(nil,/\.Query(context.TODO(),/g
s/\.QueryRow(nil,/\.QueryRow(context.TODO(),/g
s/store\.([A-Z][A-Za-z]*)(nil,/store.\1(context.TODO(),/g
s/service\.([A-Z][A-Za-z]*)(nil,/service.\1(context.TODO(),/g
' {} \;

#----------------------------------------
# 4. Fix unused code (22 errors)
#----------------------------------------
echo "Step 4/11: Fixing unused code..."

# Mark unused functions/types with underscore prefix or remove them
# This requires manual review, so we'll log them for now
echo "  Note: Unused code requires manual review - see lint output"

#----------------------------------------
# 5. Fix ineffassign (10 errors)
#----------------------------------------
echo "Step 5/11: Fixing ineffectual assignments..."

# These require AST-level changes - mark for manual fix
echo "  Note: Ineffectual assignments require manual review"

#----------------------------------------
# 6. Fix gosimple (10 errors)
#----------------------------------------
echo "Step 6/11: Fixing gosimple issues..."

# Fix S1021: merge variable declaration with assignment
# Fix S1009: omit nil check for len()
# Fix S1001: use copy() instead of loop
# Fix S1025: remove unnecessary fmt.Sprintf
find . -name "*.go" -not -path "./vendor/*" -exec sed -i.bak '
# Fix S1025: fmt.Sprintf with string argument
s/fmt\.Sprintf("%s", \([a-zA-Z_][a-zA-Z0-9_]*\))/\1/g
' {} \;

#----------------------------------------
# 7. Clean up backup files
#----------------------------------------
echo "Cleaning up backup files..."
find . -name "*.go.bak" -delete

#----------------------------------------
# 8. Run gofmt and goimports
#----------------------------------------
echo "Formatting code..."
find . -name "*.go" -not -path "./vendor/*" -exec gofmt -w {} \;

# Install goimports if not available
if ! command -v goimports &> /dev/null; then
    echo "Installing goimports..."
    go install golang.org/x/tools/cmd/goimports@latest
fi

find . -name "*.go" -not -path "./vendor/*" -exec goimports -w {} \;

echo ""
echo "=== Automated fixes complete ==="
echo ""
echo "IMPORTANT: The following error types require manual fixes:"
echo "  - errcheck (135 errors): Add error handling"
echo "  - govet shadow (71 errors): Rename shadowed variables"
echo "  - gosec (41 errors): Fix security issues"
echo "  - SA1029 (44 errors): Define custom context key types"
echo "  - nilerr (7 errors): Return err instead of nil"
echo "  - unused (remaining): Remove or use code"
echo ""
echo "Next steps:"
echo "  1. Review the changes: git diff"
echo "  2. Run: golangci-lint run ./... > /tmp/remaining_errors.txt"
echo "  3. Continue with manual fixes for remaining errors"
echo ""
