# Database Migrations Guide

This guide explains how database migrations work in SQL Studio and how to create, test, and manage them.

## Table of Contents

- [Overview](#overview)
- [How Migrations Work](#how-migrations-work)
- [Creating New Migrations](#creating-new-migrations)
- [Testing Migrations](#testing-migrations)
- [Running Migrations](#running-migrations)
- [Rollback Process](#rollback-process)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

## Overview

SQL Studio uses a simple, transaction-safe migration system to manage database schema changes. Migrations are:

- **Versioned**: Each migration has a unique version number
- **Idempotent**: Safe to run multiple times
- **Transactional**: All-or-nothing execution
- **Tracked**: Applied migrations are recorded in `schema_migrations` table

## How Migrations Work

### Migration Lifecycle

1. **Initialization**: `schema_migrations` table is created if it doesn't exist
2. **Version Check**: System determines which migrations have been applied
3. **Execution**: Pending migrations are run in order within transactions
4. **Recording**: Successfully applied migrations are recorded with timestamps

### Migration Structure

Each migration consists of:

```go
Migration{
    Version:     3,                              // Unique sequential number
    Description: "Add organization support",    // Human-readable description
    SQL:         getAddOrganizationSQL(),       // SQL to execute
}
```

### Schema Migrations Table

The `schema_migrations` table tracks applied migrations:

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,      -- Migration version number
    description TEXT NOT NULL,        -- Description of the migration
    applied_at INTEGER NOT NULL,      -- Unix timestamp when applied
    checksum TEXT                     -- Optional: verify migration integrity
);
```

## Creating New Migrations

### Step 1: Create Migration SQL File

Create a new file in `pkg/storage/turso/migrations/`:

```sql
-- migrations/004_add_new_feature.sql
-- Migration: Add new feature
-- Version: 004
-- Date: 2025-10-24

-- Add new table
CREATE TABLE IF NOT EXISTS new_feature (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

-- Add index
CREATE INDEX IF NOT EXISTS idx_new_feature_name ON new_feature(name);

-- Modify existing table (if needed)
ALTER TABLE existing_table ADD COLUMN new_column TEXT;
```

### Step 2: Add Migration to migrate.go

Add the migration to the `Migrations` slice in `pkg/storage/turso/migrate.go`:

```go
var Migrations = []Migration{
    // ... existing migrations ...
    {
        Version:     4,
        Description: "Add new feature table",
        SQL:         getAddNewFeatureSQL(),
    },
}

// Add a function to return the SQL
func getAddNewFeatureSQL() string {
    return `
CREATE TABLE IF NOT EXISTS new_feature (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_new_feature_name ON new_feature(name);
`
}
```

### Step 3: Update Schema in client.go (for new installations)

Update the `getEmbeddedSchema()` function in `client.go` to include new tables for fresh installations:

```go
func getEmbeddedSchema() string {
    return `
-- ... existing schema ...

-- New feature (added in migration 004)
CREATE TABLE IF NOT EXISTS new_feature (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_new_feature_name ON new_feature(name);
`
}
```

## Testing Migrations

### Unit Tests

Create tests in `migrate_test.go`:

```go
func TestMigration004AppliesCorrectly(t *testing.T) {
    db, logger := setupTestDB(t)
    defer db.Close()

    // Initialize schema
    err := InitializeSchema(db, logger)
    require.NoError(t, err)

    // Run migrations
    err = RunMigrations(db, logger)
    require.NoError(t, err)

    // Verify table exists
    var tableName string
    err = db.QueryRow(`
        SELECT name FROM sqlite_master
        WHERE type='table' AND name='new_feature'
    `).Scan(&tableName)
    require.NoError(t, err)
    assert.Equal(t, "new_feature", tableName)

    // Verify columns
    columns := getTableColumns(t, db, "new_feature")
    assert.Contains(t, columns, "id")
    assert.Contains(t, columns, "name")
    assert.Contains(t, columns, "created_at")

    // Verify indexes
    indexes := getIndexes(t, db, "new_feature")
    assert.Contains(t, indexes, "idx_new_feature_name")
}
```

### Integration Tests

Test with actual database:

```bash
# Run all tests
go test ./pkg/storage/turso/... -v

# Run specific migration test
go test ./pkg/storage/turso/... -v -run TestMigration004

# Run with race detection
go test ./pkg/storage/turso/... -v -race
```

### Manual Testing

Use the verification script:

```bash
cd backend-go
go run scripts/verify_migrations.go
```

## Running Migrations

### Automatic (Recommended)

Migrations run automatically when the server starts:

```bash
# Migrations run during server startup
go run cmd/server/main.go
```

Server output:
```
INFO[0000] Initializing database schema...
INFO[0000] Running database migrations...
INFO[0001] Migration applied successfully               version=3
INFO[0001] All migrations applied successfully          count=1
```

### Programmatic

Run migrations in code:

```go
import "github.com/sql-studio/backend-go/pkg/storage/turso"

// Connect to database
db, err := turso.NewClient(&turso.Config{
    URL:       "libsql://your-db.turso.io",
    AuthToken: "your-token",
}, logger)
if err != nil {
    log.Fatal(err)
}

// Initialize schema (idempotent)
if err := turso.InitializeSchema(db, logger); err != nil {
    log.Fatal(err)
}

// Run migrations (idempotent)
if err := turso.RunMigrations(db, logger); err != nil {
    log.Fatal(err)
}
```

### Check Migration Status

```go
statuses, err := turso.GetMigrationStatus(db)
if err != nil {
    log.Fatal(err)
}

for _, status := range statuses {
    fmt.Printf("Migration %d: %s (%s)\n",
        status.Version,
        status.Description,
        status.Status)
    if status.AppliedAt != nil {
        fmt.Printf("  Applied: %s\n", status.AppliedAt)
    }
}
```

## Rollback Process

### Important: SQLite Limitations

SQLite does **not** support `DROP COLUMN`, making automatic rollbacks difficult. Rollbacks must be done manually.

### Manual Rollback Steps

For migration 003 (organization support):

```sql
-- Step 1: Create new tables without organization columns
CREATE TABLE connection_templates_new AS
SELECT
    id, user_id, name, type, host, port,
    database_name, username, ssl_config,
    created_at, updated_at, sync_version, deleted_at, metadata
FROM connection_templates;

CREATE TABLE saved_queries_sync_new AS
SELECT
    id, user_id, title, query, description, connection_id,
    folder, tags, created_at, updated_at, sync_version, deleted_at
FROM saved_queries_sync;

-- Step 2: Drop old tables
DROP TABLE connection_templates;
DROP TABLE saved_queries_sync;

-- Step 3: Rename new tables
ALTER TABLE connection_templates_new RENAME TO connection_templates;
ALTER TABLE saved_queries_sync_new RENAME TO saved_queries_sync;

-- Step 4: Recreate indexes (excluding organization indexes)
CREATE INDEX idx_connections_user_id ON connection_templates(user_id);
CREATE INDEX idx_connections_updated ON connection_templates(updated_at);
-- ... other indexes ...

CREATE INDEX idx_queries_user_id ON saved_queries_sync(user_id);
CREATE INDEX idx_queries_updated ON saved_queries_sync(updated_at);
-- ... other indexes ...

-- Step 5: Remove migration record
DELETE FROM schema_migrations WHERE version = 3;
```

### Rollback Helper

Check available rollback instructions:

```go
err := turso.RollbackMigration(3)
if err != nil {
    fmt.Println(err) // Prints rollback instructions
}
```

## Best Practices

### 1. Always Use Transactions

Migrations automatically run in transactions, ensuring atomicity:
- If any statement fails, the entire migration rolls back
- No partial schema changes

### 2. Make Migrations Idempotent

Use `IF NOT EXISTS` and `IF EXISTS`:

```sql
-- Good: Idempotent
CREATE TABLE IF NOT EXISTS new_table (...);
CREATE INDEX IF NOT EXISTS idx_name ON table(column);
ALTER TABLE table ADD COLUMN IF NOT EXISTS new_col TEXT;

-- Bad: Will fail on second run
CREATE TABLE new_table (...);
CREATE INDEX idx_name ON table(column);
```

### 3. Test Migrations Thoroughly

Before deploying:
1. Test on a copy of production data
2. Verify data integrity after migration
3. Check query performance with new indexes
4. Test rollback procedure (if needed)

### 4. Use Meaningful Version Numbers

- Sequential integers (1, 2, 3, ...)
- Gap tolerance (if migration 5 is removed, skip to 6)
- Never reuse version numbers

### 5. Keep Migrations Small and Focused

```go
// Good: Single responsibility
{
    Version: 4,
    Description: "Add audit logging table",
    SQL: "CREATE TABLE audit_logs (...)",
}

// Bad: Multiple unrelated changes
{
    Version: 4,
    Description: "Add audit logs and fix user table and update indexes",
    SQL: "CREATE TABLE audit_logs (...); ALTER TABLE users ...; ...",
}
```

### 6. Add Comments to SQL

```sql
-- Migration 003: Add organization support
-- This enables team collaboration features

-- Add organization reference for shared connections
ALTER TABLE connection_templates
ADD COLUMN organization_id TEXT
REFERENCES organizations(id) ON DELETE CASCADE;
```

### 7. Consider Data Migration

When adding NOT NULL columns to existing tables:

```sql
-- Step 1: Add column as nullable
ALTER TABLE users ADD COLUMN status TEXT;

-- Step 2: Populate existing rows
UPDATE users SET status = 'active' WHERE status IS NULL;

-- Step 3: Cannot add NOT NULL constraint in SQLite without recreating table
-- Document this limitation or recreate the table
```

### 8. Index Strategy

Add indexes for:
- Foreign keys
- Frequently queried columns
- Columns used in WHERE, JOIN, ORDER BY
- Composite indexes for common query patterns

```sql
-- Single column index
CREATE INDEX idx_users_email ON users(email);

-- Composite index for common query pattern
CREATE INDEX idx_connections_org_visibility
ON connection_templates(organization_id, visibility);
```

### 9. Version Control

- Commit migration files with descriptive messages
- Include migration version in commit message
- Tag releases with applied migration versions

```bash
git add pkg/storage/turso/migrations/004_*.sql
git commit -m "feat: Add migration 004 - audit logging table"
git tag v1.2.0-migration-004
```

### 10. Document Breaking Changes

If a migration requires application code changes:

```go
// Migration{
//     Version: 5,
//     Description: "Rename user.name to user.full_name",
//     SQL: "...",
//     // BREAKING: Update all queries using user.name to user.full_name
// }
```

## Troubleshooting

### Migration Fails

**Symptom**: Migration fails with SQL error

**Solution**:
1. Check logs for specific error message
2. Test SQL manually in SQLite
3. Verify foreign key references exist
4. Check for constraint violations

```bash
# Manual SQL testing
sqlite3 test.db
> .read migrations/003_add_organization_to_resources.sql
```

### Duplicate Migration Error

**Symptom**: "migration X already applied"

**Solution**: This is normal - migrations are idempotent. The system skips already-applied migrations.

### Foreign Key Constraint Failed

**Symptom**: "FOREIGN KEY constraint failed"

**Solution**:
1. Ensure referenced table exists
2. Verify referenced column has matching data type
3. Check that referenced rows exist

```sql
-- Enable foreign key debugging
PRAGMA foreign_keys = ON;
PRAGMA foreign_key_check;
```

### Migration Stuck

**Symptom**: Migration hangs indefinitely

**Solution**:
1. Check for table locks
2. Verify database connection
3. Review transaction isolation settings
4. Check for deadlocks (rare in SQLite)

### Schema Mismatch

**Symptom**: Fresh install schema differs from migrated schema

**Solution**:
1. Update `getEmbeddedSchema()` to match migrated state
2. Run migrations on fresh database
3. Compare schemas with `PRAGMA table_info(table_name)`

```bash
# Compare schemas
go run scripts/verify_migrations.go --check-schema
```

## Query Performance After Migration

### Analyze Index Usage

```sql
-- Check if indexes are being used
EXPLAIN QUERY PLAN
SELECT * FROM connection_templates
WHERE organization_id = 'org-123' AND visibility = 'shared';

-- Should see: "USING INDEX idx_connections_org_visibility"
```

### Benchmark Queries

Before and after migration:

```go
start := time.Now()
rows, _ := db.Query("SELECT * FROM connection_templates WHERE organization_id = ?", orgID)
duration := time.Since(start)
fmt.Printf("Query took: %v\n", duration)
```

### Optimize Indexes

If queries are slow after migration:

1. Check EXPLAIN QUERY PLAN
2. Add covering indexes
3. Consider denormalization
4. Review query patterns

## Additional Resources

- [SQLite ALTER TABLE documentation](https://www.sqlite.org/lang_altertable.html)
- [SQLite Foreign Key Support](https://www.sqlite.org/foreignkeys.html)
- [Turso Documentation](https://docs.turso.tech/)
- [Database Migration Best Practices](https://www.postgresql.org/docs/current/ddl-schemas.html)

## Support

For issues with migrations:

1. Check this guide first
2. Review test cases in `migrate_test.go`
3. Run verification script: `go run scripts/verify_migrations.go`
4. Open an issue with migration logs and error details
