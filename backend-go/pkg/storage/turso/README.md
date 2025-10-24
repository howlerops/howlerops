# Turso Storage Implementation for SQL Studio Individual Tier

Complete Turso cloud database storage layer for SQL Studio's Individual tier backend.

## Overview

This implementation provides a production-ready Turso storage layer that:

- Implements all auth interfaces (UserStore, SessionStore, LoginAttemptStore)
- Stores user accounts, sessions, and authentication data securely
- Syncs app data (connections, queries, history) across devices
- **NEVER stores passwords in the cloud** - only connection metadata
- Uses prepared statements to prevent SQL injection
- Follows the exact patterns from LocalSQLiteStorage

## Architecture

```
backend-go/pkg/storage/turso/
├── schema.sql              # Complete database schema
├── client.go               # Turso client factory & initialization
├── user_store.go           # UserStore implementation
├── session_store.go        # SessionStore implementation
├── login_attempt_store.go  # LoginAttemptStore implementation
└── app_data_store.go       # App data sync (connections, queries, history)
```

## Key Features

### Security

- **No passwords in cloud**: Connection passwords stay local only
- **Bcrypt password hashing**: User passwords hashed with bcrypt (cost 12)
- **Prepared statements**: All queries use parameterized statements
- **Brute force protection**: Login attempt tracking with lockout
- **Session management**: Token-based auth with refresh tokens

### Sync

- **Soft deletes**: Uses `deleted_at` for conflict resolution
- **Version tracking**: `sync_version` column for optimistic locking
- **Sanitized history**: Query history strips data literals
- **Device tracking**: Sync metadata tracks last sync per device

### Performance

- **Indexes**: Comprehensive indexes on all foreign keys and filters
- **Connection pooling**: Configurable connection pool (default 10)
- **Unix timestamps**: INTEGER timestamps for efficiency
- **JSON encoding**: Structured metadata stored as JSON TEXT

## Database Schema

### Auth Tables

- **users**: User accounts with bcrypt password hashes
- **sessions**: JWT tokens and refresh tokens
- **login_attempts**: Brute force protection tracking
- **email_verification_tokens**: Email verification flow
- **password_reset_tokens**: Password reset flow
- **license_keys**: Individual tier license management

### App Data Tables (Sanitized)

- **connection_templates**: Connection metadata (NO passwords!)
- **saved_queries_sync**: Saved queries across devices
- **query_history_sync**: Sanitized query history
- **sync_metadata**: Last sync tracking per user/device

## Usage

### 1. Initialize Turso Client

```go
import (
    "github.com/sql-studio/backend-go/pkg/storage/turso"
    "github.com/sirupsen/logrus"
)

// Create Turso config
config := &turso.Config{
    URL:       "libsql://your-db.turso.io",
    AuthToken: "your-auth-token",
    MaxConns:  10,
}

// Initialize client
logger := logrus.New()
db, err := turso.NewClient(config, logger)
if err != nil {
    log.Fatal(err)
}
defer db.Close()

// Initialize schema (creates tables if needed)
if err := turso.InitializeSchema(db, logger); err != nil {
    log.Fatal(err)
}
```

### 2. Create Store Instances

```go
// Create auth stores
userStore := turso.NewTursoUserStore(db, logger)
sessionStore := turso.NewTursoSessionStore(db, logger)
loginAttemptStore := turso.NewTursoLoginAttemptStore(db, logger)

// Create app data store
appDataStore := turso.NewTursoAppDataStore(db, logger)
```

### 3. Use with Auth Service

```go
import (
    "github.com/sql-studio/backend-go/internal/auth"
    "github.com/sql-studio/backend-go/internal/middleware"
)

// Create auth middleware
jwtSecret := "your-jwt-secret"
authMiddleware := middleware.NewAuthMiddleware(jwtSecret, logger)

// Create auth service
authService := auth.NewService(
    userStore,
    sessionStore,
    loginAttemptStore,
    authMiddleware,
    auth.Config{
        BcryptCost:        12,
        JWTExpiration:     24 * time.Hour,
        RefreshExpiration: 7 * 24 * time.Hour,
        MaxLoginAttempts:  5,
        LockoutDuration:   15 * time.Minute,
    },
    logger,
)

// Use auth service
loginResp, err := authService.Login(ctx, &auth.LoginRequest{
    Username:   "user@example.com",
    Password:   "password",
    RememberMe: true,
    IPAddress:  "192.168.1.1",
    UserAgent:  "SQL Studio/1.0",
})
```

### 4. Sync App Data

```go
// Save connection template (NO password!)
conn := &turso.ConnectionTemplate{
    UserID:       userID,
    Name:         "Production DB",
    Type:         "postgres",
    Host:         "db.example.com",
    Port:         5432,
    DatabaseName: "myapp",
    Username:     "dbuser",
    SSLConfig:    `{"mode": "require"}`,
    Metadata: map[string]string{
        "environment": "production",
    },
}
if err := appDataStore.SaveConnectionTemplate(ctx, conn); err != nil {
    log.Fatal(err)
}

// Get connections for user
templates, err := appDataStore.GetConnectionTemplates(ctx, userID, false)
if err != nil {
    log.Fatal(err)
}

// Save query
query := &turso.SavedQuerySync{
    UserID:       userID,
    Title:        "Active Users",
    Query:        "SELECT * FROM users WHERE active = true",
    Description:  "Get all active users",
    ConnectionID: conn.ID,
    Folder:       "Analytics",
    Tags:         []string{"users", "reporting"},
}
if err := appDataStore.SaveQuerySync(ctx, query); err != nil {
    log.Fatal(err)
}

// Save sanitized query history
history := &turso.QueryHistorySync{
    UserID:         userID,
    QuerySanitized: "SELECT * FROM users WHERE active = ?", // No literal values!
    ConnectionID:   conn.ID,
    ExecutedAt:     time.Now(),
    DurationMS:     123,
    RowsReturned:   42,
    Success:        true,
}
if err := appDataStore.SaveQueryHistory(ctx, history); err != nil {
    log.Fatal(err)
}

// Update sync metadata
syncMeta := &turso.SyncMetadata{
    UserID:        userID,
    LastSyncAt:    time.Now(),
    DeviceID:      "device-123",
    ClientVersion: "1.0.0",
}
if err := appDataStore.UpdateSyncMetadata(ctx, syncMeta); err != nil {
    log.Fatal(err)
}
```

## Environment Variables

```bash
# Turso Configuration
TURSO_URL="libsql://your-db.turso.io"
TURSO_AUTH_TOKEN="your-auth-token"
TURSO_MAX_CONNS=10

# JWT Configuration
JWT_SECRET="your-secret-key"
JWT_EXPIRATION="24h"
REFRESH_EXPIRATION="168h"

# Auth Configuration
BCRYPT_COST=12
MAX_LOGIN_ATTEMPTS=5
LOCKOUT_DURATION="15m"
```

## Migration from In-Memory Stores

Replace this in `internal/services/stores.go`:

```go
// OLD (in-memory)
userStore := NewInMemoryUserStore()
sessionStore := NewInMemorySessionStore()
loginAttemptStore := NewInMemoryLoginAttemptStore()
```

With this:

```go
// NEW (Turso)
import "github.com/sql-studio/backend-go/pkg/storage/turso"

config := &turso.Config{
    URL:       os.Getenv("TURSO_URL"),
    AuthToken: os.Getenv("TURSO_AUTH_TOKEN"),
    MaxConns:  10,
}

db, err := turso.NewClient(config, logger)
if err != nil {
    log.Fatal(err)
}

if err := turso.InitializeSchema(db, logger); err != nil {
    log.Fatal(err)
}

userStore := turso.NewTursoUserStore(db, logger)
sessionStore := turso.NewTursoSessionStore(db, logger)
loginAttemptStore := turso.NewTursoLoginAttemptStore(db, logger)
```

## Testing

### Unit Tests

```go
// Create test database
config := &turso.Config{
    URL:       "libsql://test-db.turso.io",
    AuthToken: "test-token",
    MaxConns:  5,
}

db, err := turso.NewClient(config, logger)
if err != nil {
    t.Fatal(err)
}
defer db.Close()

// Initialize schema
if err := turso.InitializeSchema(db, logger); err != nil {
    t.Fatal(err)
}

// Test user store
userStore := turso.NewTursoUserStore(db, logger)

user := &auth.User{
    ID:       uuid.New().String(),
    Username: "testuser",
    Email:    "test@example.com",
    Password: "hashed-password",
    Role:     "user",
    Active:   true,
}

if err := userStore.CreateUser(ctx, user); err != nil {
    t.Fatal(err)
}

retrieved, err := userStore.GetUser(ctx, user.ID)
if err != nil {
    t.Fatal(err)
}

if retrieved.Username != user.Username {
    t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
}
```

### Integration Tests

```bash
# Set up test Turso database
turso db create sql-studio-test

# Get auth token
turso db tokens create sql-studio-test

# Run tests
TURSO_URL="libsql://sql-studio-test.turso.io" \
TURSO_AUTH_TOKEN="your-test-token" \
go test ./pkg/storage/turso/... -v
```

## Performance Considerations

### Connection Pool

- **MaxOpenConns**: 10 (cloud optimized)
- **MaxIdleConns**: 5
- **ConnMaxLifetime**: 5 minutes
- **ConnMaxIdleTime**: 1 minute

### Indexes

All foreign keys and filter columns are indexed:

```sql
-- User lookups
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Session lookups
CREATE INDEX idx_sessions_token ON sessions(token);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);

-- Query filtering
CREATE INDEX idx_queries_user_id ON saved_queries_sync(user_id);
CREATE INDEX idx_queries_updated ON saved_queries_sync(updated_at);
CREATE INDEX idx_queries_folder ON saved_queries_sync(folder);
```

### Query Optimization

- Use prepared statements (prevents SQL injection AND improves performance)
- Limit result sets with LIMIT/OFFSET
- Filter early with WHERE clauses
- Use EXISTS for existence checks instead of COUNT

## Security Best Practices

### 1. Password Handling

```go
// NEVER store passwords in Turso
conn := &ConnectionTemplate{
    // ... other fields ...
    // NO PasswordEncrypted field!
}

// Passwords stay in local SQLite only
localConn := &storage.Connection{
    PasswordEncrypted: encryptedPassword, // Local only
}
```

### 2. Query Sanitization

```go
// BAD - Contains literal values
history := &QueryHistorySync{
    QuerySanitized: "SELECT * FROM users WHERE email = 'user@example.com'",
}

// GOOD - Replace literals with placeholders
history := &QueryHistorySync{
    QuerySanitized: "SELECT * FROM users WHERE email = ?",
}
```

### 3. Token Security

```go
// Store tokens securely
session := &auth.Session{
    Token:        secureRandomToken(),  // JWT
    RefreshToken: secureRandomToken(),  // Crypto random
    ExpiresAt:    time.Now().Add(24 * time.Hour),
}
```

## Troubleshooting

### Connection Issues

```go
// Test connection
if err := db.Ping(); err != nil {
    log.Fatal("Failed to ping Turso:", err)
}

// Check URL format
// Correct: libsql://db-name.turso.io
// Wrong: https://db-name.turso.io
```

### Schema Issues

```sql
-- Check if tables exist
SELECT name FROM sqlite_master WHERE type='table';

-- Check indexes
SELECT name FROM sqlite_master WHERE type='index';

-- Drop and recreate (dev only!)
DROP TABLE IF EXISTS users;
-- Then run InitializeSchema again
```

### Performance Issues

```go
// Enable query logging
logger.SetLevel(logrus.DebugLevel)

// Monitor slow queries
startTime := time.Now()
rows, err := db.QueryContext(ctx, query, args...)
duration := time.Since(startTime)
if duration > 100*time.Millisecond {
    logger.Warnf("Slow query: %v (%s)", duration, query)
}
```

## Production Deployment

### 1. Create Turso Database

```bash
# Create production database
turso db create sql-studio-prod --location sjc

# Create auth token
turso db tokens create sql-studio-prod

# Get database URL
turso db show sql-studio-prod --url
```

### 2. Set Environment Variables

```bash
export TURSO_URL="libsql://sql-studio-prod.turso.io"
export TURSO_AUTH_TOKEN="your-production-token"
export JWT_SECRET="your-secure-random-secret"
```

### 3. Run Migrations

```go
// Initialize schema on first deploy
if err := turso.InitializeSchema(db, logger); err != nil {
    log.Fatal(err)
}
```

### 4. Monitoring

```go
// Add health check endpoint
func healthCheck(w http.ResponseWriter, r *http.Request) {
    if err := db.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        return
    }
    w.WriteHeader(http.StatusOK)
}

// Monitor session cleanup
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        if err := sessionStore.CleanupExpiredSessions(ctx); err != nil {
            logger.Error("Failed to cleanup sessions:", err)
        }
    }
}()
```

## License

MIT License - See LICENSE file for details

## Support

For issues, questions, or contributions, please open an issue on GitHub.
