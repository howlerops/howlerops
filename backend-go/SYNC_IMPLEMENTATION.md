# Howlerops Sync & Email Service Implementation

This document provides a comprehensive guide to the sync endpoints and email service implementation for Howlerops's Individual tier backend.

## Overview

The implementation includes:

1. **Email Service** - Resend API integration for transactional emails
2. **Sync Service** - Cloud sync with conflict resolution for Individual tier
3. **Extended Auth** - Email verification and password reset flows
4. **HTTP Endpoints** - RESTful API for sync operations
5. **Turso Storage** - SQLite-based cloud storage with Turso

## Architecture

```
┌─────────────────┐
│  Frontend App   │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────┐
│      HTTP/REST API Layer            │
│  ┌───────────┐    ┌──────────────┐ │
│  │   Auth    │    │     Sync     │ │
│  │ Endpoints │    │  Endpoints   │ │
│  └─────┬─────┘    └──────┬───────┘ │
└────────┼─────────────────┼─────────┘
         │                 │
         ▼                 ▼
┌─────────────────┐ ┌─────────────────┐
│  Auth Service   │ │  Sync Service   │
│  w/ Email       │ │ w/ Conflict Res │
└────────┬────────┘ └────────┬────────┘
         │                   │
         ▼                   ▼
┌─────────────────┐ ┌─────────────────┐
│  Email Service  │ │  Turso Store    │
│    (Resend)     │ │   (SQLite)      │
└─────────────────┘ └─────────────────┘
```

## File Structure

```
backend-go/
├── internal/
│   ├── auth/
│   │   ├── service.go           # Base auth service
│   │   ├── email_auth.go        # Extended auth with email
│   │   └── token_store.go       # Verification/reset tokens
│   ├── email/
│   │   ├── service.go           # Email service (Resend)
│   │   └── templates.go         # HTML email templates
│   ├── sync/
│   │   ├── types.go             # Sync data types
│   │   ├── service.go           # Sync business logic
│   │   ├── handlers.go          # HTTP handlers
│   │   ├── turso_store.go       # Turso database implementation
│   │   └── service_test.go      # Integration tests
│   └── config/
│       └── config.go            # Config with email & sync settings
└── configs/
    └── config.yaml              # Configuration file
```

## Email Service

### Features

- **Resend API Integration** - Production-ready email delivery
- **HTML Templates** - Beautiful, responsive email templates
- **Mock Implementation** - For testing without API calls

### Email Types

1. **Email Verification** - Verify email on signup
2. **Password Reset** - Secure password reset flow
3. **Welcome Email** - Onboarding email for new users

### Usage

```go
import "github.com/sql-studio/backend-go/internal/email"

// Create service
emailService, err := email.NewResendEmailService(
    apiKey,
    "noreply@sqlstudio.io",
    "Howlerops",
    logger,
)

// Send verification email
err = emailService.SendVerificationEmail(
    "user@example.com",
    "token123",
    "https://app.sqlstudio.io/verify?token=token123",
)

// Mock service for testing
mockService := email.NewMockEmailService(logger)
```

### Configuration

```yaml
email:
  provider: "resend"
  api_key: "${RESEND_API_KEY}"
  from_email: "noreply@sqlstudio.io"
  from_name: "Howlerops"
  base_url: "https://app.sqlstudio.io"
```

## Sync Service

### Features

- **Incremental Sync** - Only sync changes since last sync
- **Conflict Detection** - Automatic conflict detection using sync_version
- **Multiple Resolution Strategies**:
  - Last Write Wins (default)
  - Keep Both Versions
  - User Choice
- **Data Sanitization** - Prevents credential leakage
- **Rate Limiting** - 10 requests per minute per user

### Sync Data Types

1. **Connections** - Database connection configs (without credentials)
2. **Saved Queries** - User's saved SQL queries
3. **Query History** - Query execution history

### Conflict Resolution

Conflicts are detected when:
- `sync_version` on server > `sync_version` in upload
- `updated_at` timestamps differ

```go
// Conflict structure
type Conflict struct {
    ID            string
    ItemType      SyncItemType
    ItemID        string
    LocalVersion  *ConflictVersion
    RemoteVersion *ConflictVersion
    DetectedAt    time.Time
}
```

### Usage

```go
import "github.com/sql-studio/backend-go/internal/sync"

// Create service
config := sync.Config{
    MaxUploadSize:      10 * 1024 * 1024, // 10MB
    ConflictStrategy:   sync.ConflictResolutionLastWriteWins,
    RetentionDays:      30,
    MaxHistoryItems:    1000,
    EnableSanitization: true,
}

service := sync.NewService(store, config, logger)

// Upload changes
resp, err := service.Upload(ctx, &sync.SyncUploadRequest{
    UserID:   userID,
    DeviceID: deviceID,
    Changes:  changes,
})

// Download changes
resp, err := service.Download(ctx, &sync.SyncDownloadRequest{
    UserID:   userID,
    DeviceID: deviceID,
    Since:    lastSyncTime,
})
```

## HTTP Endpoints

### Auth Endpoints

```
POST /api/auth/verify-email
  Body: { "token": "verification_token" }
  Response: { "success": true, "message": "Email verified" }

POST /api/auth/resend-verification
  Headers: Authorization: Bearer <token>
  Response: { "success": true, "message": "Email sent" }

POST /api/auth/request-password-reset
  Body: { "email": "user@example.com" }
  Response: { "success": true, "message": "Reset link sent" }

POST /api/auth/reset-password
  Body: { "token": "reset_token", "new_password": "..." }
  Response: { "success": true, "message": "Password reset" }
```

### Sync Endpoints

```
POST /api/sync/upload
  Headers: Authorization: Bearer <token>
  Body: {
    "device_id": "device123",
    "last_sync_at": "2024-01-01T00:00:00Z",
    "changes": [
      {
        "item_type": "connection",
        "item_id": "conn123",
        "action": "create",
        "data": { ... },
        "updated_at": "2024-01-01T01:00:00Z",
        "sync_version": 1
      }
    ]
  }
  Response: {
    "success": true,
    "synced_at": "2024-01-01T02:00:00Z",
    "conflicts": [],
    "rejected": []
  }

GET /api/sync/download?since=2024-01-01T00:00:00Z&device_id=device123
  Headers: Authorization: Bearer <token>
  Response: {
    "connections": [...],
    "saved_queries": [...],
    "query_history": [...],
    "sync_timestamp": "2024-01-01T02:00:00Z",
    "has_more": false
  }

GET /api/sync/conflicts
  Headers: Authorization: Bearer <token>
  Response: {
    "conflicts": [...],
    "count": 2
  }

POST /api/sync/conflicts/{id}/resolve
  Headers: Authorization: Bearer <token>
  Body: {
    "strategy": "last_write_wins",
    "chosen_version": "local"  // optional, for user_choice
  }
  Response: {
    "success": true,
    "resolved_at": "2024-01-01T02:00:00Z"
  }
```

## Database Schema (Turso/SQLite)

### Connections Table

```sql
CREATE TABLE connections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT NOT NULL,
    username TEXT,
    use_ssh BOOLEAN DEFAULT 0,
    ssh_host TEXT,
    ssh_port INTEGER,
    ssh_user TEXT,
    color TEXT,
    icon TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER DEFAULT 1,
    deleted_at TIMESTAMP,
    UNIQUE(user_id, id)
);
```

### Saved Queries Table

```sql
CREATE TABLE saved_queries (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    query TEXT NOT NULL,
    connection_id TEXT,
    tags TEXT,
    favorite BOOLEAN DEFAULT 0,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sync_version INTEGER DEFAULT 1,
    deleted_at TIMESTAMP,
    UNIQUE(user_id, id)
);
```

### Query History Table

```sql
CREATE TABLE query_history (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    query TEXT NOT NULL,
    connection_id TEXT,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    duration_ms INTEGER,
    rows_affected INTEGER,
    success BOOLEAN DEFAULT 1,
    error TEXT,
    metadata TEXT,
    sync_version INTEGER DEFAULT 1,
    UNIQUE(user_id, id)
);
```

### Conflicts Table

```sql
CREATE TABLE conflicts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    item_type TEXT NOT NULL,
    item_id TEXT NOT NULL,
    local_version TEXT,
    remote_version TEXT,
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolution TEXT,
    UNIQUE(user_id, id)
);
```

## Configuration

### Environment Variables

```bash
# Email Service
export RESEND_API_KEY="re_xxx..."

# Turso Database
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="eyJhbGc..."

# JWT Secret
export SQL_STUDIO_AUTH_JWT_SECRET="your-super-secret-key-min-32-chars"
```

### Configuration File (config.yaml)

```yaml
email:
  provider: "resend"
  api_key: "${RESEND_API_KEY}"
  from_email: "noreply@sqlstudio.io"
  from_name: "Howlerops"
  base_url: "https://app.sqlstudio.io"

sync:
  enabled: true
  max_upload_size: 10485760  # 10MB
  conflict_strategy: "last_write_wins"
  retention_days: 30
  max_history_items: 1000
  enable_sanitization: true
  rate_limit_rpm: 10

turso:
  url: "${TURSO_URL}"
  auth_token: "${TURSO_AUTH_TOKEN}"
  max_connections: 25
```

## Security Considerations

### 1. Credential Sanitization

The sync service validates that **no credentials** are included in uploads:

```go
// Automatically rejected
{
  "item_type": "connection",
  "data": {
    "password": "secret123"  // ❌ Rejected
  }
}
```

### 2. Rate Limiting

- **Upload**: 10 requests per minute per user
- **Download**: 20 requests per minute per user
- **Conflicts**: 10 requests per minute per user

### 3. Token Expiration

- **Email Verification**: 24 hours
- **Password Reset**: 1 hour (for security)

### 4. Data Isolation

All queries are scoped to `user_id` to prevent cross-user data access.

## Testing

### Run Tests

```bash
# Unit tests
go test ./internal/sync/... -v

# Integration tests
go test ./internal/sync/... -tags=integration -v

# Email service tests (mock)
go test ./internal/email/... -v
```

### Example Test

```go
func TestSyncUpload(t *testing.T) {
    store := NewMockStore()
    service := NewService(store, config, logger)

    req := &SyncUploadRequest{
        UserID:   "user123",
        DeviceID: "device123",
        Changes:  []SyncChange{...},
    }

    resp, err := service.Upload(ctx, req)
    assert.NoError(t, err)
    assert.True(t, resp.Success)
}
```

## Integration Guide

### 1. Initialize Services

```go
// Load config
cfg, err := config.Load()

// Create email service
emailService, err := email.NewResendEmailService(
    cfg.Email.APIKey,
    cfg.Email.FromEmail,
    cfg.Email.FromName,
    logger,
)

// Create Turso store
syncStore, err := sync.NewTursoStore(
    cfg.Turso.URL,
    cfg.Turso.AuthToken,
    logger,
)

// Create sync service
syncConfig := sync.Config{
    MaxUploadSize:      cfg.Sync.MaxUploadSize,
    ConflictStrategy:   ConflictResolutionStrategy(cfg.Sync.ConflictStrategy),
    RetentionDays:      cfg.Sync.RetentionDays,
    MaxHistoryItems:    cfg.Sync.MaxHistoryItems,
    EnableSanitization: cfg.Sync.EnableSanitization,
}
syncService := sync.NewService(syncStore, syncConfig, logger)

// Create extended auth service
extendedAuth := auth.NewExtendedService(
    baseAuthService,
    emailService,
    tokenStore,
    cfg.Email.BaseURL,
)
```

### 2. Register HTTP Routes

```go
// Sync routes
syncHandler := sync.NewHandler(syncService, logger)
syncRouter := router.PathPrefix("/api/sync").Subrouter()
syncRouter.Use(authMiddleware.Authenticate) // Require auth
syncHandler.RegisterRoutes(syncRouter)

// Auth routes
authHandler := sync.NewAuthHandler(extendedAuth, logger)
authRouter := router.PathPrefix("/api/auth").Subrouter()
authHandler.RegisterAuthRoutes(authRouter)
```

### 3. Frontend Integration

```typescript
// Upload changes
const syncUpload = async (changes: SyncChange[]) => {
  const response = await fetch('/api/sync/upload', {
    method: 'POST',
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      device_id: deviceId,
      last_sync_at: lastSyncTime,
      changes,
    }),
  });

  const data = await response.json();

  if (data.conflicts.length > 0) {
    // Handle conflicts
    await handleConflicts(data.conflicts);
  }

  return data;
};

// Download changes
const syncDownload = async () => {
  const since = lastSyncTime.toISOString();
  const response = await fetch(
    `/api/sync/download?since=${since}&device_id=${deviceId}`,
    {
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    }
  );

  return response.json();
};
```

## Performance Considerations

### 1. Batch Size

- Maximum upload size: 10MB per request
- Maximum history items: 1000 per download
- Use pagination for large datasets

### 2. Indexes

All tables include indexes on:
- `user_id` for fast user-scoped queries
- `updated_at` for incremental sync
- `executed_at` for history queries

### 3. Connection Pooling

Turso connection pool configured with:
- Max connections: 25
- Idle timeout: 5 minutes
- Connection lifetime: 1 hour

## Monitoring

### Metrics to Track

1. **Sync Metrics**:
   - Upload requests per minute
   - Download requests per minute
   - Average upload size
   - Conflict rate

2. **Email Metrics**:
   - Emails sent per hour
   - Delivery success rate
   - Email type distribution

3. **Error Metrics**:
   - Upload failures
   - Download failures
   - Email send failures

### Logging

All services use structured logging with:
- User ID
- Device ID
- Operation type
- Duration
- Error details

## Troubleshooting

### Common Issues

1. **"Token expired"**
   - Verification tokens expire after 24 hours
   - Reset tokens expire after 1 hour
   - Request a new token

2. **"Conflict detected"**
   - Review conflicting versions
   - Choose resolution strategy
   - Use "keep both" if unsure

3. **"Upload rejected"**
   - Check for credentials in data
   - Verify upload size < 10MB
   - Check rate limits

4. **"Email not sent"**
   - Verify Resend API key
   - Check email configuration
   - Review logs for errors

## Next Steps

1. **Production Deployment**:
   - Set up Turso database
   - Configure Resend API key
   - Enable rate limiting
   - Set up monitoring

2. **Frontend Integration**:
   - Implement sync client
   - Add conflict resolution UI
   - Handle offline mode
   - Show sync status

3. **Testing**:
   - Load testing with 1000+ users
   - Conflict resolution scenarios
   - Email delivery testing
   - Security penetration testing

## Support

For questions or issues:
- Check logs in `backend-go/logs/`
- Review test cases in `*_test.go`
- Email: support@sqlstudio.io
