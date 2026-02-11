# IndexedDB to SQLite Migration Test Plan

## Overview

This document outlines the end-to-end test scenarios for migrating user data from WebKit IndexedDB (volatile, origin-bound storage) to SQLite (persistent storage at `~/.howlerops/local.db`).

### Architecture Summary

```
Frontend (React)                          Backend (Go)
+-------------------------+               +---------------------------+
| migrate-to-sqlite.ts    |   Wails v3   | storage_migration.go      |
| - useMigrateToSQLite()  |   Bindings   | - StorageMigrationService |
| - migrateIndexedDBToSQLite() <-------> | - GetMigrationStatus()    |
| - IndexedDB read        |              | - ImportConnections()     |
+-------------------------+               | - ImportQueries()         |
                                          | - ImportHistory()         |
                                          | - CompleteMigration()     |
                                          +---------------------------+
                                                       |
                                          +---------------------------+
                                          | sqlite_local.go           |
                                          | - settings table          |
                                          | - connections table       |
                                          | - saved_queries table     |
                                          | - query_history table     |
                                          +---------------------------+
```

### Migration Flow

1. App starts, `useMigrateToSQLite()` hook runs in `app.tsx`
2. Frontend calls `StorageMigrationStatus()` via Wails binding
3. Backend checks SQLite for:
   - `migration_from_indexeddb_complete` setting
   - Existing data counts (connections, queries, history)
4. Based on status, frontend either:
   - Skips migration (already done or has data)
   - Reads from IndexedDB and imports to SQLite
5. After import, `StorageCompleteMigration()` marks migration done

---

## Test Scenarios

### Scenario 1: Fresh Install (New User)

**Preconditions:**
- No IndexedDB database exists (or empty)
- SQLite database is empty (no tables or empty tables)
- `migration_from_indexeddb_complete` setting does not exist

**Steps:**
1. Start the application
2. Migration hook runs automatically
3. Frontend checks migration status
4. Frontend reads IndexedDB (returns empty arrays)
5. Frontend calls `StorageCompleteMigration()`

**Expected Behavior:**
- `getMigrationStatus()` returns:
  ```json
  {
    "sqlite_has_data": false,
    "migration_done": false,
    "connection_count": 0,
    "query_count": 0,
    "history_count": 0,
    "preferences_count": 0
  }
  ```
- IndexedDB read returns empty arrays for all stores
- Migration result:
  ```json
  {
    "success": true,
    "connectionsImported": 0,
    "queriesImported": 0,
    "historyImported": 0,
    "errors": [],
    "skipped": true,
    "reason": "No data to migrate"
  }
  ```
- `migration_from_indexeddb_complete` set to `"true"` in settings
- `migration_timestamp` set to current RFC3339 timestamp
- Console logs: `[StorageMigration] No data in IndexedDB to migrate`

**Verification Checklist:**
- [ ] App starts without errors
- [ ] Migration completes in < 1 second
- [ ] No error logs in console
- [ ] SQLite `settings` table contains migration_from_indexeddb_complete = "true"
- [ ] User can create new connections (stored in SQLite)

---

### Scenario 2: Upgrade from v2 (Existing User with Data)

**Preconditions:**
- IndexedDB has data:
  - 3 connections in `connections` store
  - 5 saved queries in `saved_queries` store
  - 100 history entries in `query_history` store
  - UI preferences in `ui_preferences` store
- SQLite database is empty
- `migration_from_indexeddb_complete` does not exist

**Steps:**
1. Start the application
2. Migration hook runs
3. Frontend reads all data from IndexedDB
4. Frontend sends data to backend via Wails bindings
5. Backend imports data with duplicate checking
6. Backend marks migration complete

**Expected Behavior:**
- `getMigrationStatus()` returns:
  ```json
  {
    "sqlite_has_data": false,
    "migration_done": false,
    "connection_count": 0,
    "query_count": 0,
    "history_count": 0,
    "preferences_count": 0
  }
  ```
- IndexedDB read returns populated arrays
- Console logs show progress:
  ```
  [StorageMigration] Starting migration from IndexedDB to SQLite...
  [StorageMigration] Found data to migrate: 3 connections, 5 queries, 100 history entries
  [StorageMigration] Imported 3 connections (0 skipped)
  [StorageMigration] Imported 5 queries (0 skipped)
  [StorageMigration] Imported 100 history entries
  [StorageMigration] Imported 4 preferences
  [StorageMigration] Migration completed successfully
  ```
- Migration result:
  ```json
  {
    "success": true,
    "connectionsImported": 3,
    "queriesImported": 5,
    "historyImported": 100,
    "errors": [],
    "skipped": false
  }
  ```

**Data Mapping Verification:**

| IndexedDB Field | SQLite Field | Notes |
|-----------------|--------------|-------|
| `connection_id` | `id` | UUID preserved |
| `name` | `name` | Direct copy |
| `type` | `type` | postgres, mysql, etc. |
| `host` | `host` | Direct copy |
| `port` | `port` | Integer |
| `database` | `database_name` | Field renamed |
| `username` | `username` | Direct copy |
| `ssl_mode` | `ssl_config` | Converted to JSON |
| `environment_tags` | Stored in metadata | JSON array |
| `created_at` | `created_at` | RFC3339 -> Unix timestamp |
| `updated_at` | `updated_at` | RFC3339 -> Unix timestamp |

**Verification Checklist:**
- [ ] All 3 connections appear in SQLite `connections` table
- [ ] All 5 queries appear in SQLite `saved_queries` table
- [ ] All 100 history entries appear in SQLite `query_history` table
- [ ] Preferences stored with `pref_` prefix in settings
- [ ] Timestamps correctly converted
- [ ] Tags/arrays stored as JSON
- [ ] SSL config properly converted
- [ ] Connection IDs preserved exactly
- [ ] User can see migrated connections immediately
- [ ] Old queries accessible via saved queries feature

---

### Scenario 3: Already Migrated (Returning User)

**Preconditions:**
- Migration was completed previously
- `migration_from_indexeddb_complete` = `"true"` in SQLite settings
- SQLite has data (connections, queries, history)
- IndexedDB may or may not have data (irrelevant)

**Steps:**
1. Start the application
2. Migration hook runs
3. Frontend checks migration status
4. Migration is skipped

**Expected Behavior:**
- `getMigrationStatus()` returns:
  ```json
  {
    "sqlite_has_data": true,
    "migration_done": true,
    "connection_count": 3,
    "query_count": 5,
    "history_count": 100,
    "preferences_count": 0
  }
  ```
- Migration result:
  ```json
  {
    "success": true,
    "connectionsImported": 0,
    "queriesImported": 0,
    "historyImported": 0,
    "errors": [],
    "skipped": true,
    "reason": "Migration already completed"
  }
  ```
- Console logs: `[StorageMigration] Migration already completed, using SQLite`
- IndexedDB is NOT read (performance optimization)

**Verification Checklist:**
- [ ] Migration completes in < 100ms (no IndexedDB reads)
- [ ] No data is re-imported
- [ ] Existing SQLite data unchanged
- [ ] App uses SQLite data immediately

---

### Scenario 4: Partial Migration (Error Recovery)

**Preconditions:**
- Previous migration failed midway
- SQLite has partial data:
  - 2 of 3 connections imported
  - 0 queries (import failed)
  - 0 history
- `migration_from_indexeddb_complete` does NOT exist (migration not marked complete)
- IndexedDB still has all original data

**Current Behavior (Gap Identified):**

The current implementation checks `sqlite_has_data` and skips if true:
```typescript
// If SQLite already has data, skip migration but don't mark complete
if (status.sqlite_has_data) {
  result.skipped = true
  result.reason = 'SQLite already has data'
  result.success = true
  return result
}
```

**Issue:** This prevents recovery from partial migration. The code assumes if SQLite has ANY data, migration shouldn't run again.

**Recommended Behavior:**
The migration should:
1. Continue importing data even if SQLite has some data
2. Backend's duplicate checking prevents re-importing existing records
3. Only mark complete after all data types successfully processed

**Workaround (Current):**
- User can manually call `retryMigration()` which clears the flag
- Or delete SQLite database and restart app

**Verification Checklist:**
- [ ] ISSUE: Partial migration not automatically recovered
- [ ] Backend duplicate checking works (skips existing records)
- [ ] `retryMigration()` function available for manual recovery
- [ ] No data corruption from partial state

---

### Scenario 5: Concurrent Access (Multiple Windows)

**Preconditions:**
- User opens multiple browser windows/tabs
- Each window runs the migration hook independently
- SQLite database shared across all windows

**Current Behavior (Gap Identified):**

The current implementation has NO explicit locking mechanism:
```go
// LocalSQLiteStorage has a mutex but...
type LocalSQLiteStorage struct {
    mu          sync.RWMutex  // Only protects individual operations
    db          *sql.DB
    ...
}
```

**Race Condition Risks:**

1. **Double Migration:**
   - Window A reads status: `migration_done: false`
   - Window B reads status: `migration_done: false`
   - Both windows start importing
   - Duplicate data potentially created

2. **Inconsistent State:**
   - Window A starts importing connections
   - Window B checks status during import
   - Window B sees partial data, behavior undefined

**Mitigating Factors:**
1. Backend's duplicate checking by ID prevents true duplicates:
   ```go
   existing, err := s.storageManager.GetConnection(ctx, conn.ID)
   if err == nil && existing != nil {
       result.Skipped++
       continue
   }
   ```
2. SQLite's built-in locking provides some protection
3. Wails v3 is typically single-window (desktop app)

**Recommended Improvements:**
1. Add a migration lock (e.g., `migration_in_progress` setting with timestamp)
2. Use SQLite transaction for entire migration
3. Implement "leader election" for multi-window scenarios

**Verification Checklist:**
- [ ] ISSUE: No explicit concurrent access protection
- [ ] Backend duplicate checking provides partial protection
- [ ] SQLite integrity maintained (no corrupted records)
- [ ] Race condition impact limited by desktop app nature

---

## Edge Cases

### E1: IndexedDB Access Denied

**Scenario:** Browser blocks IndexedDB access (private browsing, permissions)

**Expected Behavior:**
- `getIndexedDBClient()` throws `StorageError` with code `NOT_SUPPORTED`
- Migration gracefully fails with informative error
- App continues to function with SQLite-only storage

**Current Behavior:** Handled - errors caught and logged:
```typescript
catch (error) {
  console.warn('[StorageMigration] Failed to read connections from IndexedDB:', error)
  return []
}
```

### E2: SQLite Database Locked

**Scenario:** Another process has exclusive lock on SQLite file

**Expected Behavior:**
- Backend returns error from Wails binding
- Frontend logs error, migration marked unsuccessful
- User can retry later

**Verification:**
- [ ] Error message is user-friendly
- [ ] App doesn't crash
- [ ] Retry mechanism works

### E3: Large Data Migration

**Scenario:** User has massive amounts of data
- 100+ connections
- 1000+ saved queries
- 50,000+ history entries

**Concerns:**
- Memory usage during JSON serialization
- Migration timeout
- UI responsiveness

**Current Mitigations:**
- Async/await pattern keeps UI responsive
- JSON batch processing
- No explicit timeout handling (potential issue)

**Verification:**
- [ ] Migration completes (may take several seconds)
- [ ] No browser memory issues
- [ ] Progress visible in console logs
- [ ] UI remains responsive

### E4: Malformed IndexedDB Data

**Scenario:** IndexedDB contains corrupted or malformed records

**Expected Behavior:**
- Individual records with parse errors logged
- Valid records still imported
- Errors collected in result

**Current Behavior:** Handled per-record:
```go
if err := s.storageManager.SaveConnection(ctx, storageConn); err != nil {
    result.Errors = append(result.Errors, "Connection "+conn.Name+": "+err.Error())
    continue  // Continue with next record
}
```

### E5: Field Aliasing

**Scenario:** IndexedDB schema evolved, old records use different field names

**Example:**
- Old: `{ query: "SELECT *" }`
- New: `{ query_text: "SELECT *" }`

**Current Handling:** Backend supports aliases:
```go
// Handle title/name alias
title := q.Title
if title == "" {
    title = q.Name
}

// Handle query/query_text alias
queryText := q.QueryText
if queryText == "" {
    queryText = q.Query
}
```

---

## Manual QA Checklist

### Pre-Migration Setup

- [ ] Create test IndexedDB data using browser DevTools
- [ ] Verify IndexedDB data visible in Application > IndexedDB
- [ ] Delete SQLite database: `rm ~/.howlerops/local.db`
- [ ] Clear any cached migration state

### Migration Execution

- [ ] Start application
- [ ] Open DevTools console to monitor logs
- [ ] Verify migration starts automatically
- [ ] Check for `[StorageMigration]` log messages
- [ ] Confirm no errors in console

### Post-Migration Verification

- [ ] Check SQLite database with: `sqlite3 ~/.howlerops/local.db`
- [ ] Verify tables exist: `.tables`
- [ ] Count connections: `SELECT COUNT(*) FROM connections;`
- [ ] Check settings: `SELECT * FROM settings WHERE key LIKE 'migration%';`
- [ ] Verify data integrity matches original IndexedDB

### Functional Testing

- [ ] List connections - shows migrated data
- [ ] Open a migrated connection - works correctly
- [ ] View saved queries - all present
- [ ] Execute a saved query - works
- [ ] Check query history - entries present
- [ ] Verify UI preferences (theme, etc.)

### Negative Testing

- [ ] Restart app - migration skips (already done)
- [ ] Multiple restarts - no duplicate data
- [ ] Delete SQLite, restart - migration runs again
- [ ] Corrupt IndexedDB entry - other entries still import

---

## Identified Gaps and Recommendations

### Gap 1: Partial Migration Recovery

**Issue:** If migration fails midway, the `sqlite_has_data` check prevents automatic retry.

**Recommendation:**
```typescript
// Change from:
if (status.sqlite_has_data) {
  result.skipped = true
  ...
}

// To:
// Allow migration to continue - backend handles duplicates
// Only skip if migration_done flag is set
```

### Gap 2: Concurrent Access Protection

**Issue:** No explicit locking for multi-window scenarios.

**Recommendation:**
Add migration lock mechanism:
```sql
INSERT INTO settings (key, value) VALUES ('migration_lock', timestamp)
-- Check lock age, acquire if stale (> 5 minutes)
```

### Gap 3: Migration Progress Feedback

**Issue:** No UI feedback during migration - only console logs.

**Recommendation:**
- Add migration state to React context
- Show subtle progress indicator
- Allow viewing migration history/status

### Gap 4: Rollback Capability

**Issue:** No way to rollback a failed migration.

**Recommendation:**
- Take SQLite snapshot before migration
- Implement `rollbackMigration()` function
- Store original IndexedDB export as backup

### Gap 5: Data Validation

**Issue:** Limited validation of imported data.

**Recommendation:**
- Validate required fields before import
- Check referential integrity (connection_id in queries)
- Report validation failures separately from import errors

---

## Test Data Fixtures

### Sample IndexedDB Connection

```json
{
  "connection_id": "conn-123-uuid",
  "user_id": "local-user",
  "name": "Production DB",
  "type": "postgres",
  "host": "db.example.com",
  "port": 5432,
  "database": "myapp",
  "username": "admin",
  "ssl_mode": "require",
  "environment_tags": ["production", "us-east"],
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-06-20T14:45:00Z",
  "last_used_at": "2024-12-01T09:00:00Z",
  "synced": false,
  "sync_version": 0
}
```

### Sample IndexedDB Saved Query

```json
{
  "id": "query-456-uuid",
  "user_id": "local-user",
  "title": "Active Users Report",
  "description": "Monthly active users by region",
  "query_text": "SELECT region, COUNT(*) FROM users WHERE active = true GROUP BY region",
  "tags": ["analytics", "monthly"],
  "folder": "Reports/Users",
  "is_favorite": true,
  "created_at": "2024-03-10T08:00:00Z",
  "updated_at": "2024-11-15T16:30:00Z",
  "synced": false,
  "sync_version": 0
}
```

### Sample IndexedDB Query History

```json
{
  "id": "hist-789-uuid",
  "user_id": "local-user",
  "query_text": "SELECT * FROM orders LIMIT 100",
  "connection_id": "conn-123-uuid",
  "execution_time_ms": 245,
  "row_count": 100,
  "privacy_mode": "normal",
  "executed_at": "2024-12-20T11:30:00Z",
  "synced": false,
  "sync_version": 0
}
```

---

## Conclusion

The migration implementation is solid for the primary use cases (fresh install, upgrade, already migrated). The main gaps are around error recovery and concurrent access, which are lower-priority given the desktop app context. The backend's duplicate checking provides good protection against data corruption.

Priority improvements:
1. **P1:** Fix partial migration recovery (simple code change)
2. **P2:** Add UI progress feedback
3. **P3:** Implement migration lock for concurrent access
4. **P4:** Add rollback capability
