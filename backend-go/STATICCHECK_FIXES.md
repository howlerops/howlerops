# Staticcheck Fixes Summary

## Issues Fixed (8 total)

### 1. ST1005: Error strings should not be capitalized

**File: cmd/server/main.go:174**
- Before: `"Turso URL must be configured in production"`
- After: `"turso URL must be configured in production"`

**File: pkg/database/postgres.go:326**
- Before: `"Result set is missing primary key columns: %s"`
- After: `"result set is missing primary key columns: %s"`

**File: pkg/database/postgres.go:389**
- Before: `"No editable columns found in result set"`
- After: `"no editable columns found in result set"`

### 2. SA1012: Do not pass a nil Context

**File: internal/middleware/org_rate_limit.go:97**
- Before: `r.quotaService.GetQuota(nil, orgID)`
- After: `r.quotaService.GetQuota(context.TODO(), orgID)`

**File: internal/organization/service_test.go:1311**
- Before: `service.CreateOrganization(nil, "user-1", input)`
- After: `service.CreateOrganization(context.TODO(), "user-1", input)`

### 3. SA1029: Should not use built-in type string as key for value

**File: internal/organization/handlers_test.go:150-152**

Added custom context key type:
```go
type contextKey string

const (
    contextKeyUserID   contextKey = "user_id"
    contextKeyUsername contextKey = "username"
    contextKeyRole     contextKey = "role"
)
```

Changed:
- Before: `context.WithValue(req.Context(), "user_id", "test-user-123")`
- After: `context.WithValue(req.Context(), contextKeyUserID, "test-user-123")`

And similarly for "username" and "role" keys.

## Verification

```bash
golangci-lint run --no-config --enable=staticcheck 2>&1 | grep -E "(SA|ST)[0-9]+"
# Output: (empty - no staticcheck errors)
```

All 8 staticcheck issues have been successfully resolved.
