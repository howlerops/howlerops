#!/bin/bash

# Database Verification Script for Organization Tables
# Checks if all required tables, indexes, and constraints exist

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
TURSO_URL="${TURSO_URL:-}"
TURSO_AUTH_TOKEN="${TURSO_AUTH_TOKEN:-}"

log() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[OK]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check if environment variables are set
if [ -z "$TURSO_URL" ] || [ -z "$TURSO_AUTH_TOKEN" ]; then
    warning "TURSO_URL or TURSO_AUTH_TOKEN not set in environment"
    log "Attempting to load from .env file..."

    if [ -f ".env" ]; then
        export $(cat .env | grep -v '^#' | xargs)
    elif [ -f "backend-go/.env" ]; then
        export $(cat backend-go/.env | grep -v '^#' | xargs)
    else
        error "No .env file found and environment variables not set"
        error "Please set TURSO_URL and TURSO_AUTH_TOKEN or create a .env file"
        exit 1
    fi
fi

log "Checking database schema..."
log "Turso URL: ${TURSO_URL:0:30}..."
echo ""

# Function to execute SQL query via Turso API
execute_sql() {
    local query="$1"
    curl -s -X POST "$TURSO_URL" \
        -H "Authorization: Bearer $TURSO_AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"statements\": [\"$query\"]}"
}

# Function to check if table exists
check_table() {
    local table_name="$1"
    log "Checking table: $table_name"

    local result=$(execute_sql "SELECT name FROM sqlite_master WHERE type='table' AND name='$table_name';")

    if echo "$result" | grep -q "\"name\":\"$table_name\""; then
        success "Table $table_name exists"
        return 0
    else
        error "Table $table_name does not exist"
        return 1
    fi
}

# Function to show table schema
show_table_schema() {
    local table_name="$1"
    log "Schema for $table_name:"

    local result=$(execute_sql "PRAGMA table_info($table_name);")
    echo "$result" | jq -r '.results[0].rows[] | @json' 2>/dev/null || echo "$result"
    echo ""
}

# Function to count rows in table
count_rows() {
    local table_name="$1"
    log "Row count for $table_name:"

    local result=$(execute_sql "SELECT COUNT(*) as count FROM $table_name;")
    local count=$(echo "$result" | grep -o '"count":[0-9]*' | cut -d':' -f2)

    if [ -n "$count" ]; then
        success "$table_name has $count rows"
    else
        warning "Could not get row count for $table_name"
    fi
    echo ""
}

# Function to show sample data
show_sample_data() {
    local table_name="$1"
    local limit="${2:-5}"
    log "Sample data from $table_name (limit $limit):"

    local result=$(execute_sql "SELECT * FROM $table_name LIMIT $limit;")
    echo "$result" | jq '.' 2>/dev/null || echo "$result"
    echo ""
}

# Function to check indexes
check_indexes() {
    local table_name="$1"
    log "Indexes for $table_name:"

    local result=$(execute_sql "SELECT name, sql FROM sqlite_master WHERE type='index' AND tbl_name='$table_name';")
    echo "$result" | jq -r '.results[0].rows[] | @json' 2>/dev/null || echo "$result"
    echo ""
}

# ====================================================================
# Main Checks
# ====================================================================

log "=== ORGANIZATION DATABASE VERIFICATION ==="
echo ""

# Check all required tables
log "=== Checking Required Tables ==="
TABLES=(
    "organizations"
    "organization_members"
    "organization_invitations"
    "audit_logs"
)

ALL_TABLES_EXIST=true
for table in "${TABLES[@]}"; do
    if ! check_table "$table"; then
        ALL_TABLES_EXIST=false
    fi
done
echo ""

if [ "$ALL_TABLES_EXIST" = false ]; then
    error "Some required tables are missing!"
    error "Please run database migrations or initialize the schema"
    exit 1
fi

success "All required tables exist!"
echo ""

# Show schemas
log "=== Table Schemas ==="
for table in "${TABLES[@]}"; do
    show_table_schema "$table"
done

# Show row counts
log "=== Row Counts ==="
for table in "${TABLES[@]}"; do
    count_rows "$table"
done

# Show indexes
log "=== Indexes ==="
for table in "${TABLES[@]}"; do
    check_indexes "$table"
done

# Show sample data (if exists)
log "=== Sample Data ==="
log "Organizations:"
show_sample_data "organizations" 3

log "Organization Members:"
show_sample_data "organization_members" 5

log "Organization Invitations:"
show_sample_data "organization_invitations" 5

log "Audit Logs:"
show_sample_data "audit_logs" 10

# ====================================================================
# Verify Constraints
# ====================================================================

log "=== Verifying Constraints ==="

# Check if foreign keys are enabled
log "Checking foreign key support..."
FK_RESULT=$(execute_sql "PRAGMA foreign_keys;")
if echo "$FK_RESULT" | grep -q '"foreign_keys":1'; then
    success "Foreign keys are enabled"
else
    warning "Foreign keys might not be enabled"
fi
echo ""

# Check unique constraints
log "Checking unique constraints..."

# organizations table should have unique name
ORG_UNIQUE=$(execute_sql "SELECT sql FROM sqlite_master WHERE type='table' AND name='organizations';")
if echo "$ORG_UNIQUE" | grep -q "UNIQUE"; then
    success "Organizations table has unique constraints"
else
    warning "Organizations table might not have unique constraints"
fi

# organization_members should have unique (organization_id, user_id)
MEMBER_UNIQUE=$(execute_sql "SELECT sql FROM sqlite_master WHERE type='table' AND name='organization_members';")
if echo "$MEMBER_UNIQUE" | grep -q "UNIQUE"; then
    success "Organization members table has unique constraints"
else
    warning "Organization members table might not have unique constraints"
fi

# organization_invitations should have unique (organization_id, email)
INVITE_UNIQUE=$(execute_sql "SELECT sql FROM sqlite_master WHERE type='table' AND name='organization_invitations';")
if echo "$INVITE_UNIQUE" | grep -q "UNIQUE"; then
    success "Organization invitations table has unique constraints"
else
    warning "Organization invitations table might not have unique constraints"
fi
echo ""

# ====================================================================
# Summary
# ====================================================================

log "=== VERIFICATION SUMMARY ==="
success "Database schema verification completed"
success "All required tables exist and are accessible"
log "Check the output above for detailed information"
echo ""

log "To manually query the database, use:"
log "  turso db shell <database-name>"
log "Or use the Turso CLI with your credentials"
echo ""

exit 0
