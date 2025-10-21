# Bug Fixes - Day 2

**Date**: 2025-10-19
**Context**: Bug fixes discovered during SQLite driver testing

---

## Summary

Fixed 2 critical bugs in SQLite driver that were preventing index introspection from working correctly:

1. **PRAGMA index_list origin field type mismatch**
2. **Shared memory database connection pooling**

**Impact**: SQLite driver can now properly retrieve index metadata including columns, which is essential for schema introspection.

---

## Bug #1: PRAGMA index_list Origin Field Type Mismatch

### Problem

**File**: `pkg/database/sqlite.go:634`

The code tried to scan PRAGMA `index_list`'s "origin" column (a string) directly into a boolean field:

```go
err := rows.Scan(
    &seq,
    &idx.Name,
    &idx.Unique,
    &idx.Primary, // ❌ WRONG: boolean field, but origin is a string
    &partial,
)
```

**PRAGMA index_list schema**:
- `origin`: string - "c" (CREATE INDEX), "u" (UNIQUE), or "pk" (PRIMARY KEY)
- `Primary`: boolean field in `IndexInfo` struct

**Error**: Type mismatch causes scan error, preventing index metadata retrieval.

### Solution

**File**: `pkg/database/sqlite.go:629-643`

1. Scan origin into a string variable
2. Set `idx.Primary` based on string value

```go
var origin string // PRAGMA index_list origin: "c"=CREATE INDEX, "u"=UNIQUE, "pk"=PRIMARY KEY

err := rows.Scan(
    &seq,
    &idx.Name,
    &idx.Unique,
    &origin, // ✅ CORRECT: scan into string
    &partial,
)
if err != nil {
    return nil, err
}

// Set Primary flag based on origin
idx.Primary = (origin == "pk") // ✅ CORRECT: convert to boolean
```

---

## Bug #2: Shared Memory Database Connection Pooling

### Problem

**Files**:
- `pkg/database/pool.go:212-227` (DSN builder)
- `pkg/database/sqlite_test.go:17-29` (test helper)

**Issue 1 - Incorrect DSN Format for Shared Memory**:

When using `:memory:` with `cache=shared`, the DSN was built as:
```
:memory:?cache=shared
```

But SQLite requires the format:
```
file::memory:?cache=shared
```

**Result**: Each connection got a separate in-memory database instead of sharing one, causing PRAGMA index_info queries to return no rows (querying a different empty database).

**Issue 2 - Test Cross-Contamination**:

All tests used the same shared `:memory:` database, causing tests to interfere with each other when run together.

### Solution

**Part 1 - Fix DSN Format**

**File**: `pkg/database/pool.go:215-221`

```go
// For :memory: databases with cache=shared, use file::memory: format
// This ensures the in-memory database is properly shared across connections
if dsn == ":memory:" {
    if cacheMode, ok := p.config.Parameters["cache"]; ok && cacheMode == "shared" {
        dsn = "file::memory:" // ✅ Use correct format
    }
}
```

**Part 2 - Handle Existing Parameters in DSN**

**File**: `pkg/database/pool.go:223-228`

```go
if len(p.config.Parameters) > 0 {
    // Check if DSN already has parameters (contains '?')
    separator := "?"
    if strings.Contains(dsn, "?") {
        separator = "&" // ✅ Use & for additional parameters
    }
    dsn += separator
    // ... append parameters
}
```

**Part 3 - Unique Database Names Per Test**

**File**: `pkg/database/sqlite_test.go:17-33`

```go
func newMemoryConfig() database.ConnectionConfig {
    // Use a unique database name for each test to avoid cross-test contamination
    // when using shared cache mode
    dbName := fmt.Sprintf("file:testdb_%d?mode=memory", time.Now().UnixNano())

    return database.ConnectionConfig{
        Type:     database.SQLite,
        Database: dbName, // ✅ Unique per test
        // ...
        Parameters: map[string]string{
            "cache": "shared",
        },
    }
}
```

**Part 4 - Update Test Assertion**

**File**: `pkg/database/sqlite_test.go:190-191`

Changed from:
```go
assert.Equal(t, ":memory:", info["database"]) // ❌ Expected exact match
```

To:
```go
// Check that it's an in-memory database (unique name format)
assert.Contains(t, info["database"], "mode=memory") // ✅ Check for memory mode
```

---

## Additional Fixes

### PRAGMA Quoting Issues

**Files**: `pkg/database/sqlite.go:575, 616, 646, 673`

Changed PRAGMA statements from using `QuoteIdentifier()` (double quotes) to single quotes:

```go
// Before
query := fmt.Sprintf("PRAGMA index_info(%s)", s.QuoteIdentifier(idx.Name))

// After
colQuery := fmt.Sprintf("PRAGMA index_info('%s')", idx.Name)
```

**Reason**: While SQLite PRAGMA commands accept various quoting styles, single quotes are more standard and consistent.

---

## Test Results

### Before Fixes
- **Failing**: `TestSQLiteDatabase_GetTableStructure/get_structure_with_indexes`
- **Reason**: Index columns array was empty `[]`
- **Many other tests failing** due to shared database contamination

### After Fixes
- **All SQLite tests passing**: 90/90 ✅
- **Coverage improved**: 18.3% → 20.0%
- **Full test suite**: PASS

---

## Impact

### What Works Now
1. ✅ Index metadata retrieval including column names
2. ✅ Shared memory databases work correctly with connection pooling
3. ✅ Tests run independently without cross-contamination
4. ✅ PRAGMA commands execute correctly

### Schema Introspection
The SQLite driver can now properly:
- List all indexes on a table
- Identify unique vs non-unique indexes
- Identify primary key indexes (origin="pk")
- List all columns in each index
- Retrieve foreign key information

---

## Files Modified

1. `pkg/database/sqlite.go`
   - Line 575: PRAGMA table_info quoting
   - Line 616: PRAGMA index_list quoting
   - Line 629-643: Fixed origin field scanning
   - Line 646: PRAGMA index_info quoting
   - Line 673: PRAGMA foreign_key_list quoting

2. `pkg/database/pool.go`
   - Line 7: Added `strings` import
   - Line 215-241: Fixed buildSQLiteDSN for shared memory and parameter handling

3. `pkg/database/sqlite_test.go`
   - Line 6: Added `fmt` import
   - Line 17-33: Updated newMemoryConfig to use unique database names
   - Line 190-191: Updated assertion for memory database info

---

## Lessons Learned

1. **SQLite Shared Memory**: Requires `file::memory:?cache=shared` format, not `:memory:?cache=shared`
2. **PRAGMA Type Mismatch**: Always check PRAGMA result column types match Go struct fields
3. **Test Isolation**: Shared databases require unique names to prevent test interference
4. **DSN Parameter Building**: Check for existing parameters before adding `?` vs `&`

---

*Fixes validated with full test suite: All tests passing ✅*
