# Turso Storage Quick Start Guide

Get Howlerops backend running with Turso in 5 minutes.

## Step 1: Create Turso Database (2 minutes)

```bash
# Install Turso CLI (if not installed)
curl -sSfL https://get.tur.so/install.sh | bash

# Login to Turso
turso auth login

# Create database
turso db create sql-studio-dev

# Get database URL
turso db show sql-studio-dev --url
# Output: libsql://sql-studio-dev-[your-org].turso.io

# Create auth token
turso db tokens create sql-studio-dev
# Output: eyJhbGc... (save this token!)
```

## Step 2: Set Environment Variables (1 minute)

Create `.env` file in `backend-go/`:

```bash
# Turso
TURSO_URL=libsql://sql-studio-dev-[your-org].turso.io
TURSO_AUTH_TOKEN=eyJhbGc...your-token-here
TURSO_MAX_CONNS=10

# JWT
JWT_SECRET=your-random-secret-min-32-characters-long
JWT_EXPIRATION=24h
REFRESH_EXPIRATION=168h

# Auth
BCRYPT_COST=12
MAX_LOGIN_ATTEMPTS=5
LOCKOUT_DURATION=15m

# Server
PORT=8080
LOG_LEVEL=debug
```

## Step 3: Update Code (2 minutes)

### Option A: Create New File

Create `backend-go/internal/services/turso_stores.go`:

```go
package services

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/auth"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

type TursoStores struct {
	UserStore         auth.UserStore
	SessionStore      auth.SessionStore
	LoginAttemptStore auth.LoginAttemptStore
	AppDataStore      *turso.TursoAppDataStore
}

func NewTursoStores(logger *logrus.Logger) *TursoStores {
	// Load config
	config := &turso.Config{
		URL:       os.Getenv("TURSO_URL"),
		AuthToken: os.Getenv("TURSO_AUTH_TOKEN"),
		MaxConns:  10,
	}

	// Connect to Turso
	db, err := turso.NewClient(config, logger)
	if err != nil {
		log.Fatal("Failed to connect to Turso:", err)
	}

	// Initialize schema
	if err := turso.InitializeSchema(db, logger); err != nil {
		log.Fatal("Failed to initialize schema:", err)
	}

	logger.Info("Turso storage initialized successfully")

	// Create stores
	stores := &TursoStores{
		UserStore:         turso.NewTursoUserStore(db, logger),
		SessionStore:      turso.NewTursoSessionStore(db, logger),
		LoginAttemptStore: turso.NewTursoLoginAttemptStore(db, logger),
		AppDataStore:      turso.NewTursoAppDataStore(db, logger),
	}

	// Start cleanup tasks
	go startCleanupTasks(context.Background(), stores, logger)

	return stores
}

func startCleanupTasks(ctx context.Context, stores *TursoStores, logger *logrus.Logger) {
	sessionTicker := time.NewTicker(1 * time.Hour)
	attemptTicker := time.NewTicker(6 * time.Hour)

	for {
		select {
		case <-sessionTicker.C:
			if err := stores.SessionStore.CleanupExpiredSessions(ctx); err != nil {
				logger.WithError(err).Error("Failed to cleanup sessions")
			}
		case <-attemptTicker.C:
			before := time.Now().Add(-24 * time.Hour)
			if err := stores.LoginAttemptStore.CleanupOldAttempts(ctx, before); err != nil {
				logger.WithError(err).Error("Failed to cleanup login attempts")
			}
		case <-ctx.Done():
			return
		}
	}
}
```

### Option B: Update Existing File

In `backend-go/internal/services/stores.go`, replace:

```go
// OLD
func NewStores() (*Stores, error) {
    return &Stores{
        UserStore:         NewInMemoryUserStore(),
        SessionStore:      NewInMemorySessionStore(),
        LoginAttemptStore: NewInMemoryLoginAttemptStore(),
    }, nil
}
```

With:

```go
// NEW
import "github.com/sql-studio/backend-go/pkg/storage/turso"

func NewStores(logger *logrus.Logger) (*Stores, error) {
    tursoStores := NewTursoStores(logger)
    return &Stores{
        UserStore:         tursoStores.UserStore,
        SessionStore:      tursoStores.SessionStore,
        LoginAttemptStore: tursoStores.LoginAttemptStore,
    }, nil
}
```

## Step 4: Test It!

### Test 1: Check Connection

```bash
cd backend-go
go run cmd/server/main.go
```

You should see:
```
INFO[0000] Connecting to Turso database
INFO[0000] Successfully connected to Turso database
INFO[0000] Turso schema initialized successfully
INFO[0000] Turso storage initialized successfully
INFO[0000] Howlerops backend started
```

### Test 2: Create Test User

```bash
# In a new terminal
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "testpassword123"
  }'
```

Expected response:
```json
{
  "id": "uuid-here",
  "message": "User created successfully"
}
```

### Test 3: Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "password": "testpassword123"
  }'
```

Expected response:
```json
{
  "user": {
    "id": "uuid-here",
    "username": "testuser",
    "email": "test@example.com",
    "role": "user",
    "active": true
  },
  "token": "eyJhbGc...",
  "refresh_token": "eyJhbGc...",
  "expires_at": "2025-10-24T..."
}
```

### Test 4: Verify in Turso

```bash
# Query the database
turso db shell sql-studio-dev

# Check users
SELECT id, username, email, role, active FROM users;

# Check sessions
SELECT id, user_id, expires_at, ip_address FROM sessions;

# Exit
.quit
```

## Troubleshooting

### Error: "Failed to connect to Turso"

```bash
# Check URL format
echo $TURSO_URL
# Should be: libsql://db-name.turso.io

# Test connection manually
turso db shell sql-studio-dev
```

### Error: "TURSO_URL is required"

```bash
# Make sure .env is loaded
export $(cat .env | xargs)

# Or use godotenv
go get github.com/joho/godotenv
```

### Error: "Failed to initialize schema"

```bash
# Check database permissions
turso db inspect sql-studio-dev

# Verify auth token is valid
turso auth token
```

### Error: "table users already exists"

This is OK! The schema initialization checks for existing tables.

## Next Steps

1. **Add HTTP Endpoints**: See `example_integration.go` for handler examples
2. **Test Auth Flow**: Register → Login → Logout → Refresh Token
3. **Add Sync Endpoints**: Connection templates, queries, history
4. **Set Up Frontend**: Connect React app to new API endpoints
5. **Production Deploy**: Create production database, update env vars

## Production Checklist

Before deploying to production:

- [ ] Create production database: `turso db create sql-studio-prod --location sjc`
- [ ] Generate production token (never commit to git!)
- [ ] Use strong JWT secret (32+ random characters)
- [ ] Set LOG_LEVEL=info
- [ ] Enable TLS for backend API
- [ ] Set up health check endpoint
- [ ] Configure monitoring/alerting
- [ ] Test auth flow end-to-end
- [ ] Test error cases (invalid credentials, expired tokens)
- [ ] Load test with expected traffic

## Help & Resources

- **Turso Docs**: https://docs.turso.tech
- **Implementation Details**: See `README.md` in this directory
- **Integration Examples**: See `example_integration.go`
- **Schema Reference**: See `schema.sql`

## Common Commands

```bash
# List databases
turso db list

# Show database info
turso db show sql-studio-dev

# Open database shell
turso db shell sql-studio-dev

# Create new auth token
turso db tokens create sql-studio-dev

# Destroy database (careful!)
turso db destroy sql-studio-dev
```

## Quick Reference

### Store Interfaces

```go
// Auth stores
userStore := turso.NewTursoUserStore(db, logger)
sessionStore := turso.NewTursoSessionStore(db, logger)
loginAttemptStore := turso.NewTursoLoginAttemptStore(db, logger)

// App data store
appDataStore := turso.NewTursoAppDataStore(db, logger)
```

### User Operations

```go
// Create user
user := &auth.User{...}
userStore.CreateUser(ctx, user)

// Get user
user, err := userStore.GetUser(ctx, userID)
user, err := userStore.GetUserByUsername(ctx, username)
user, err := userStore.GetUserByEmail(ctx, email)

// Update user
userStore.UpdateUser(ctx, user)

// Delete user
userStore.DeleteUser(ctx, userID)
```

### Session Operations

```go
// Create session
session := &auth.Session{...}
sessionStore.CreateSession(ctx, session)

// Get session
session, err := sessionStore.GetSession(ctx, token)

// Cleanup expired
sessionStore.CleanupExpiredSessions(ctx)
```

### App Data Operations

```go
// Save connection (no password!)
conn := &turso.ConnectionTemplate{...}
appDataStore.SaveConnectionTemplate(ctx, conn)

// Get connections
conns, err := appDataStore.GetConnectionTemplates(ctx, userID, false)

// Save query
query := &turso.SavedQuerySync{...}
appDataStore.SaveQuerySync(ctx, query)

// Save history (sanitized!)
history := &turso.QueryHistorySync{
    QuerySanitized: "SELECT * FROM users WHERE id = ?", // No literals!
    ...
}
appDataStore.SaveQueryHistory(ctx, history)
```

## That's It!

You now have a fully functional Turso-backed authentication and sync system.

For more examples and detailed documentation, see:
- `README.md` - Complete documentation
- `example_integration.go` - Full integration examples
- `schema.sql` - Database schema reference

Happy coding! 🚀
