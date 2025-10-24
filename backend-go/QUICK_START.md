# Quick Start Guide - SQL Studio Sync System

## 5-Minute Setup

### Prerequisites

```bash
# Install Go 1.24+
go version

# Install dependencies
cd backend-go
go mod download
```

### 1. Configure Environment

Create `.env` file:

```bash
# Email Service (Resend)
export RESEND_API_KEY="re_your_api_key_here"

# Database (Turso)
export TURSO_URL="libsql://your-db.turso.io"
export TURSO_AUTH_TOKEN="your_auth_token_here"

# JWT Secret (min 32 chars)
export SQL_STUDIO_AUTH_JWT_SECRET="your-super-secret-jwt-key-min-32-chars"
```

### 2. Initialize Database

```bash
# Option A: Use Turso CLI
turso db create sql-studio-sync
turso db show sql-studio-sync

# Option B: Use local SQLite for development
# Database will be auto-created when server starts
```

### 3. Run Server

```bash
# Load environment
source .env

# Run server
go run cmd/server/main.go

# Server will start on http://localhost:8500
```

### 4. Test Endpoints

```bash
# Health check
curl http://localhost:8500/health

# Register user
curl -X POST http://localhost:8500/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "SecurePass123!"
  }'

# Login to get token
curl -X POST http://localhost:8500/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "SecurePass123!"
  }'

# Upload sync data (use token from login)
curl -X POST http://localhost:8500/api/sync/upload \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device-123",
    "changes": [
      {
        "item_type": "connection",
        "item_id": "conn-1",
        "action": "create",
        "data": {
          "id": "conn-1",
          "name": "My Database",
          "type": "postgres",
          "database": "mydb",
          "created_at": "2024-01-01T00:00:00Z",
          "updated_at": "2024-01-01T00:00:00Z",
          "sync_version": 1
        },
        "updated_at": "2024-01-01T00:00:00Z",
        "sync_version": 1
      }
    ]
  }'
```

## Development Mode

### Using Mock Services

For local development without external dependencies:

```go
// Use mock email service
emailService := email.NewMockEmailService(logger)

// Use in-memory stores
userStore := services.NewInMemoryUserStore()
sessionStore := services.NewInMemorySessionStore()
tokenStore := auth.NewInMemoryTokenStore()
```

### Run Tests

```bash
# Unit tests
go test ./internal/sync/... -v

# Integration tests
go test ./internal/... -v

# With coverage
go test ./internal/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Common Tasks

### Add New User

```go
user := &auth.User{
    Username: "newuser",
    Email:    "newuser@example.com",
    Role:     "user",
}

err := authService.CreateUser(ctx, user)
```

### Sync Connection

```go
conn := &sync.ConnectionTemplate{
    ID:       uuid.New().String(),
    Name:     "Production DB",
    Type:     "postgres",
    Host:     "db.example.com",
    Port:     5432,
    Database: "prod",
    Username: "dbuser",
}

err := syncStore.SaveConnection(ctx, userID, conn)
```

### Resolve Conflict

```go
req := &sync.ConflictResolutionRequest{
    ConflictID: conflictID,
    Strategy:   sync.ConflictResolutionLastWriteWins,
}

resp, err := syncService.ResolveConflict(ctx, userID, req)
```

## Project Structure

```
backend-go/
├── cmd/
│   └── server/
│       └── main.go              # Entry point
├── internal/
│   ├── auth/                    # Authentication
│   │   ├── service.go
│   │   ├── email_auth.go
│   │   └── token_store.go
│   ├── email/                   # Email service
│   │   ├── service.go
│   │   └── templates.go
│   ├── sync/                    # Sync service
│   │   ├── types.go
│   │   ├── service.go
│   │   ├── handlers.go
│   │   ├── turso_store.go
│   │   └── service_test.go
│   ├── config/                  # Configuration
│   │   └── config.go
│   ├── middleware/              # HTTP middleware
│   │   ├── auth.go
│   │   └── ratelimit.go
│   └── server/                  # HTTP/gRPC servers
│       ├── http.go
│       └── grpc.go
├── configs/
│   └── config.yaml              # Config file
├── docs/
│   ├── SYNC_IMPLEMENTATION.md   # Implementation guide
│   ├── API_DOCUMENTATION.md     # API reference
│   ├── ARCHITECTURE.md          # Architecture docs
│   └── QUICK_START.md           # This file
└── go.mod
```

## Troubleshooting

### "Token expired"

Tokens expire after 24 hours. Get a new one:

```bash
curl -X POST http://localhost:8500/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "testuser", "password": "SecurePass123!"}'
```

### "Connection refused"

Check if server is running:

```bash
curl http://localhost:8500/health
```

### "Invalid credentials"

Check environment variables:

```bash
echo $TURSO_URL
echo $RESEND_API_KEY
```

### "Conflict detected"

View conflicts:

```bash
curl -X GET http://localhost:8500/api/sync/conflicts \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Next Steps

1. Read [SYNC_IMPLEMENTATION.md](/Users/jacob_1/projects/sql-studio/backend-go/SYNC_IMPLEMENTATION.md) for details
2. Check [API_DOCUMENTATION.md](/Users/jacob_1/projects/sql-studio/backend-go/API_DOCUMENTATION.md) for API reference
3. Review [ARCHITECTURE.md](/Users/jacob_1/projects/sql-studio/backend-go/ARCHITECTURE.md) for system design
4. Explore test files for usage examples

## Support

- Issues: https://github.com/sql-studio/backend-go/issues
- Email: support@sqlstudio.io
- Docs: https://docs.sqlstudio.io
