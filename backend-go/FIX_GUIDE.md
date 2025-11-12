# Comprehensive Guide to Fix All 379 Golangci-Lint Errors

## Summary

Total errors: **379**

Error breakdown:
- 135 errcheck (unchecked error returns)
- 74 staticcheck (various SA issues)
- 71 govet (variable shadowing)
- 41 gosec (security vulnerabilities)
- 22 unused (unused code)
- 10 ineffassign (ineffectual assignments)
- 10 gosimple (code simplifications)
- 7 nilerr (returning nil when err != nil)
- 7 errorlint (error wrapping)
- 2 dupword (duplicate words)

## Strategy

### Phase 1: Automated Fixes (Est. ~80-100 errors)

Run the provided automation tools to fix mechanical issues.

### Phase 2: Category-by-Category Manual Fixes

Work through remaining errors systematically by type.

---

## Phase 1: Automated Fixes

###Step 1: Run the automated fixer

```bash
# Build the fixer tool
cd tools/lint-fixer
go build -o ../../lint-fixer .
cd ../..

# Run it
./lint-fixer .
```

### Step 2: Run the shell script for additional fixes

```bash
chmod +x apply_lint_fixes.sh
./apply_lint_fixes.sh
```

These will automatically fix:
- ✅ All 2 dupword errors
- ✅ Most errorlint errors (7)
- ✅ Many nil context errors (SA1012)
- ✅ Some gosimple issues

---

## Phase 2: Manual Fixes by Category

### 1. SA1029: Context Key Types (44 errors)

**Problem**: Using built-in types as context keys causes collisions.

**Files affected**:
- `internal/middleware/auth.go` (3 errors)
- `internal/middleware/auth_test.go` (29 errors)
- `internal/middleware/http_auth.go` (6 errors)
- `internal/organization/testutil/auth.go` (6 errors)

**Solution**: Define custom types for context keys.

```go
// Before
ctx = context.WithValue(ctx, "user_id", userID)

// After - add at package level
type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
	contextKeyOrgID  contextKey = "organization_id"
	contextKeyEmail  contextKey = "email"
)

// Then use
ctx = context.WithValue(ctx, contextKeyUserID, userID)
```

**Action**:
1. Open `internal/middleware/auth.go`
2. Add custom context key type at top of file
3. Replace all string context keys with the custom type
4. Repeat for other affected files

---

### 2. Errcheck: Unchecked Errors (135 errors)

**Problem**: Error returns not being checked.

**Categories**:

#### A. JSON Encoding (50+ errors)
**Pattern**: `json.NewEncoder(w).Encode(data)`

```go
// Before
json.NewEncoder(w).Encode(data)

// After
if err := json.NewEncoder(w).Encode(data); err != nil {
	http.Error(w, "Internal server error", http.StatusInternalServerError)
	return
}
```

**Files**: All handlers in:
- `internal/handlers/compliance.go` (19 errors)
- `internal/security/handler.go` (16 errors)
- `internal/templates/handler.go` (multiple errors)
- `internal/sync/handlers.go`
- `internal/monitoring/middleware.go`

#### B. Database Operations
**Pattern**: `rows.Scan(...)`, `json.Unmarshal(...)`

```go
// Before
rows.Scan(&id, &name)

// After
if err := rows.Scan(&id, &name); err != nil {
	continue // or return err
}
```

#### C. Cleanup Operations
**Pattern**: `tx.Rollback()`, `file.Close()`

```go
// Before
tx.Rollback()

// After
_ = tx.Rollback() // Explicitly ignore if in error path
// OR
if err := tx.Rollback(); err != nil {
	log.WithError(err).Error("Failed to rollback transaction")
}
```

**Action**: Use find-and-replace with regex in your editor for each pattern.

---

### 3. Govet Shadow (71 errors)

**Problem**: Variables being shadowed in inner scopes.

```go
// Before
err := doSomething()
if err != nil {
	err := doRecovery() // Shadows outer err!
	...
}

// After
err := doSomething()
if err != nil {
	recoveryErr := doRecovery() // Different name
	...
}
```

**Action**: For each shadowing error, rename the inner variable with a descriptive name:
- `err` → `parseErr`, `validateErr`, `fetchErr`, `saveErr`
- `ctx` → `cancelCtx`, `timeoutCtx`

**Files with most shadows**:
- `pkg/storage/turso/migrate.go` (9 shadows)
- `internal/whitelabel/service.go` (7 shadows)
- `pkg/storage/turso/organization_store.go` (8 shadows)

---

### 4. Gosec: Security Issues (41 errors)

#### A. G201: SQL Injection (30+ errors)

**Problem**: String formatting in SQL queries.

```go
// Before
query := fmt.Sprintf("SELECT * FROM %s WHERE id = %d", table, id)

// After - use query builder or parameterized queries
query := "SELECT * FROM users WHERE id = ?"
// OR use a SQL builder library
```

**Files**:
- `internal/analytics/dashboard.go` (10 SQL injection issues)
- `pkg/database/mysql.go` (4 issues)
- `pkg/database/postgres.go` (3 issues)
- `pkg/database/sqlite.go` (2 issues)
- Others

**Action**: Refactor to use parameterized queries throughout.

#### B. G115: Integer Overflow (5 errors)

**Problem**: Converting int to int32 without bounds checking.

```go
// Before
count32 := int32(count)

// After
if count > math.MaxInt32 {
	return fmt.Errorf("count exceeds int32 max")
}
count32 := int32(count)
```

**Files**:
- `internal/ai/grpc.go` (5 errors)
- `pkg/database/mongodb.go` (1 error)
- `pkg/storage/turso/pool.go` (3 errors)

#### C. G204: Subprocess Injection (1 error)

**File**: `internal/ai/handlers.go:216`

```go
// Before
cmd := exec.Command("sh", "-c", userInput)

// After
// Validate and sanitize userInput first
// OR use exec.Command with separate args
cmd := exec.Command(program, arg1, arg2)
```

#### D. Other Security Issues
- G402: TLS MinVersion too low (`internal/server/grpc.go`)
- G401: Weak crypto (`internal/rag/embedding_service.go` - MD5 usage)
- G501: Blocklisted import crypto/md5
- G106: SSH InsecureIgnoreHostKey
- G306: File permissions too open

---

### 5. Nilerr: Returning nil when err != nil (7 errors)

**Problem**: Functions return nil even though they have an error.

```go
// Before
if err != nil {
	log.Error(err)
	return nil // Wrong!
}

// After
if err != nil {
	log.Error(err)
	return err // Correct
}
```

**Files**:
- `internal/ai/ollama_detector.go` (2 errors)
- `internal/auth/email_auth.go` (1 error)
- `internal/auth/service.go` (1 error)
- `internal/auth/two_factor.go` (2 errors)
- `internal/sync/metadata.go` (1 error)
- `pkg/database/postgres.go` (1 error)

---

### 6. Unused Code (22 errors)

**Action**: Either remove or use the code, or prefix with `_` if needed for interface compliance.

**Major items**:
- `internal/ai/handlers.go`: `min` function (unused)
- `internal/ai/ollama.go`: `ollamaPullResponse` type
- `internal/analyzer/parser.go`: `selectPattern` var
- `pkg/crypto/team_secrets_test.go`: Entire `mockSecretStore` type and methods (7 items)
- `internal/rag/embedding_service.go`: 3 unused fields
- `pkg/database/manager.go`: `resolveConnectionID` function
- `pkg/database/multiquery/executor.go`: `executeParallel` function
- `internal/sync/handlers.go`: `validateSyncUploadRequest` function
- `internal/organization/service.go`: `checkPermission` function
- Test helpers: multiple unused functions in test files

---

### 7. Ineffassign (10 errors)

**Problem**: Assignments that are immediately overwritten.

```go
// Before
err := nil
err = doSomething() // First assignment is pointless

// After
err := doSomething()
```

**Files**:
- `internal/analytics/dashboard.go` (2)
- `pkg/database/elasticsearch.go` (5)
- `pkg/database/tidb_test.go` (2)
- `pkg/storage/manager_test.go` (1)

---

### 8. Gosimple (10 errors)

**Issues**:
- S1021: Merge variable declaration with assignment
- S1009: Omit nil check for len()
- S1001: Use copy() instead of loop
- S1025: Unnecessary fmt.Sprintf

```go
// S1021 - Before
var x int
x = 5

// After
x := 5

// S1009 - Before
if arr != nil && len(arr) > 0 {}

// After
if len(arr) > 0 {} // len(nil) is 0

// S1001 - Before
for i := 0; i < len(src); i++ {
	dst[i] = src[i]
}

// After
copy(dst, src)
```

---

### 9. Staticcheck (remaining SA issues)

- **SA1000**: Fix regexp escape sequences (`internal/analyzer/query_analyzer.go`)
- **SA9003**: Remove empty branches (`internal/templates/handler.go`, `pkg/database/postgres_test.go`)
- **SA4010**: Unused append result (`internal/pii/detector.go`)
- **SA1019**: Deprecated API usage (`internal/testutil/server.go`, `internal/tracing/tracer.go`)

---

## Verification

After all fixes:

```bash
# Check compilation
go build ./...

# Run tests
go test ./...

# Verify lint issues
golangci-lint run ./...

# Should show 0 errors
```

---

## Files Requiring Most Work

1. **internal/middleware/auth_test.go** (37 errors) - Mostly SA1029 context keys
2. **internal/handlers/compliance.go** (19 errors) - Mostly errcheck on JSON encoding
3. **internal/security/handler.go** (16 errors) - Mostly errcheck
4. **internal/analytics/dashboard.go** (16 errors) - SQL injection + errcheck
5. **pkg/database/elasticsearch.go** (13 errors) - Mixed issues
6. **internal/templates/handler.go** (12 errors) - errcheck + other

---

## Time Estimates

- Phase 1 (automated): 5 minutes
- SA1029 context keys: 30 minutes
- Errcheck JSON encoding: 45 minutes
- Errcheck other: 1 hour
- Govet shadows: 1.5 hours
- Gosec security: 2-3 hours
- Other categories: 1 hour

**Total estimated time: 6-8 hours of focused work**

---

## Tips

1. **Use IDE refactoring tools**: VSCode Go extension can help with renaming
2. **Work file-by-file**: Complete one file fully before moving to next
3. **Test frequently**: Run `go test ./path/to/package` after each file
4. **Use regex find/replace**: For repetitive patterns like JSON encoding
5. **Git commit frequently**: Commit after each category of fixes
6. **Run lint incrementally**: Check progress with `golangci-lint run ./path`

---

## Need Help?

For complex refactoring (especially SQL injection fixes), consider:
1. Using a SQL builder library (squirrel, goqu)
2. Creating helper functions for common patterns
3. Reviewing security best practices for your database driver

Good luck! 🚀
