# Enterprise Features Integration Example

This guide shows how to integrate all Phase 5 enterprise features into your SQL Studio backend.

## Complete Integration Example

### 1. Main Server Setup

```go
package main

import (
    "context"
    "database/sql"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/gorilla/mux"
    "github.com/sirupsen/logrus"
    _ "github.com/libsql/libsql-client-go/libsql"

    "backend-go/internal/domains"
    "backend-go/internal/handlers"
    "backend-go/internal/middleware"
    "backend-go/internal/quotas"
    "backend-go/internal/sla"
    "backend-go/internal/whitelabel"
)

func main() {
    // Initialize logger
    logger := logrus.New()
    logger.SetLevel(logrus.InfoLevel)
    logger.SetFormatter(&logrus.JSONFormatter{})

    // Database connection
    dbURL := os.Getenv("DATABASE_URL")
    db, err := sql.Open("libsql", dbURL)
    if err != nil {
        logger.WithError(err).Fatal("Failed to connect to database")
    }
    defer db.Close()

    // Test connection
    if err := db.Ping(); err != nil {
        logger.WithError(err).Fatal("Failed to ping database")
    }

    // Initialize stores
    whiteLabelStore := whitelabel.NewStore(db)
    domainStore := domains.NewStore(db)
    quotaStore := quotas.NewStore(db)
    slaStore := sla.NewStore(db)

    // Initialize services
    whiteLabelService := whitelabel.NewService(whiteLabelStore, logger)
    domainService := domains.NewService(domainStore, logger)
    quotaService := quotas.NewService(quotaStore, logger)
    slaMonitor := sla.NewMonitor(slaStore, logger)

    // Initialize middleware
    tenantIsolation := middleware.NewTenantIsolationMiddleware(db, logger)
    orgRateLimiter := middleware.NewOrgRateLimiter(quotaService, logger)
    slaTracking := middleware.NewSLATracking(slaMonitor, logger)

    // Initialize handlers
    enterpriseHandlers := handlers.NewEnterpriseHandlers(
        whiteLabelService,
        domainService,
        quotaService,
        slaMonitor,
        logger,
    )

    // Setup router
    router := mux.NewRouter()

    // Apply middleware in order
    router.Use(loggingMiddleware(logger))           // 1. Logging
    router.Use(authMiddleware())                     // 2. Authentication
    router.Use(tenantIsolation.EnforceTenantIsolation) // 3. Tenant isolation
    router.Use(orgRateLimiter.Limit)                 // 4. Rate limiting
    router.Use(slaTracking.Track)                    // 5. SLA tracking

    // Register routes
    enterpriseHandlers.RegisterRoutes(router)

    // Example: protected endpoint
    router.HandleFunc("/api/connections", listConnectionsHandler(db)).Methods("GET")

    // Health check (no auth)
    router.HandleFunc("/health", healthCheckHandler).Methods("GET")

    // Start background jobs
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // SLA monitoring scheduler
    slaMonitor.StartScheduler(ctx)
    slaMonitor.StartCleanupScheduler(ctx, 30) // 30-day retention

    // Rate limiter cleanup
    orgRateLimiter.StartCleanupScheduler()

    // Start server
    srv := &http.Server{
        Addr:         ":8080",
        Handler:      router,
        ReadTimeout:  15 * time.Second,
        WriteTimeout: 15 * time.Second,
        IdleTimeout:  60 * time.Second,
    }

    // Graceful shutdown
    go func() {
        logger.Info("Server starting on :8080")
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            logger.WithError(err).Fatal("Server failed")
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")

    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer shutdownCancel()

    if err := srv.Shutdown(shutdownCtx); err != nil {
        logger.WithError(err).Fatal("Server forced to shutdown")
    }

    logger.Info("Server exited")
}

// Example: List connections with tenant isolation
func listConnectionsHandler(db *sql.DB) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // Get organization context (set by middleware)
        orgID := middleware.GetCurrentOrgID(r.Context())

        // Build query with organization filter
        whereClause, args := middleware.BuildOrgFilterQuery(r.Context(), "organization_id")
        query := "SELECT id, name, type FROM connections WHERE " + whereClause

        rows, err := db.QueryContext(r.Context(), query, args...)
        if err != nil {
            http.Error(w, "Database error", http.StatusInternalServerError)
            return
        }
        defer rows.Close()

        // Process results...
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{"connections": []}`))
    }
}

// Health check handler
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    w.Write([]byte(`{"status": "healthy"}`))
}

// Logging middleware
func loggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            next.ServeHTTP(w, r)

            logger.WithFields(logrus.Fields{
                "method":   r.Method,
                "path":     r.URL.Path,
                "duration": time.Since(start),
            }).Info("Request processed")
        })
    }
}

// Auth middleware (simplified example)
func authMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Skip auth for health check
            if r.URL.Path == "/health" {
                next.ServeHTTP(w, r)
                return
            }

            // Extract JWT token from Authorization header
            authHeader := r.Header.Get("Authorization")
            if authHeader == "" {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            // Validate token and extract user ID
            // (Implementation depends on your auth system)
            userID := extractUserIDFromToken(authHeader)
            if userID == "" {
                http.Error(w, "Invalid token", http.StatusUnauthorized)
                return
            }

            // Add user ID to context
            ctx := context.WithValue(r.Context(), "user_id", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

func extractUserIDFromToken(authHeader string) string {
    // Implement JWT validation here
    // Return user ID from token claims
    return "user-123" // Placeholder
}
```

---

## 2. Database Initialization

```go
package main

import (
    "database/sql"
    "io/fs"
    "log"
    "os"
    "path/filepath"
    "sort"
)

func runMigrations(db *sql.DB, migrationsDir string) error {
    // Get all migration files
    files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
    if err != nil {
        return err
    }

    // Sort by filename (001_, 002_, etc.)
    sort.Strings(files)

    // Create migrations table
    _, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at INTEGER NOT NULL
        )
    `)
    if err != nil {
        return err
    }

    // Run each migration
    for _, file := range files {
        // Extract version from filename (e.g., "007_white_labeling.sql" -> 7)
        var version int
        _, err := fmt.Sscanf(filepath.Base(file), "%d_", &version)
        if err != nil {
            continue
        }

        // Check if already applied
        var count int
        err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations WHERE version = ?", version).Scan(&count)
        if err != nil {
            return err
        }

        if count > 0 {
            log.Printf("Migration %d already applied, skipping", version)
            continue
        }

        // Read and execute migration
        content, err := os.ReadFile(file)
        if err != nil {
            return err
        }

        _, err = db.Exec(string(content))
        if err != nil {
            return fmt.Errorf("migration %d failed: %w", version, err)
        }

        // Record migration
        _, err = db.Exec(
            "INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)",
            version,
            time.Now().Unix(),
        )
        if err != nil {
            return err
        }

        log.Printf("Migration %d applied successfully", version)
    }

    return nil
}
```

---

## 3. Usage Examples

### Check and Increment Quota

```go
func executeQueryWithQuota(ctx context.Context, quotaSvc *quotas.Service, orgID string) error {
    // Check quota before executing query
    if err := quotaSvc.CheckQuota(ctx, orgID, quotas.ResourceQuery); err != nil {
        if quotas.IsQuotaExceeded(err) {
            return fmt.Errorf("query quota exceeded, please upgrade plan")
        }
        return err
    }

    // Execute query...
    // ...

    // Increment usage
    if err := quotaSvc.IncrementUsage(ctx, orgID, quotas.ResourceQuery); err != nil {
        log.Printf("Failed to increment usage: %v", err)
        // Don't fail the request, just log
    }

    return nil
}
```

### Create Connection with Tenant Isolation

```go
func createConnection(ctx context.Context, db *sql.DB, conn *Connection) error {
    // Get organization from context
    orgID := middleware.GetCurrentOrgID(ctx)
    if orgID == "" {
        return fmt.Errorf("no organization context")
    }

    // Verify user has access to this organization
    if err := middleware.VerifyOrgAccess(ctx, orgID); err != nil {
        return err
    }

    // Set organization ID
    conn.OrganizationID = orgID

    // Insert with organization
    query := `
        INSERT INTO connections (id, organization_id, name, type, config)
        VALUES (?, ?, ?, ?, ?)
    `

    _, err := db.ExecContext(ctx, query,
        conn.ID,
        conn.OrganizationID,
        conn.Name,
        conn.Type,
        conn.Config,
    )

    return err
}
```

### List Resources with Multi-Org Support

```go
func listUserConnections(ctx context.Context, db *sql.DB) ([]*Connection, error) {
    // Build organization filter
    whereClause, args := middleware.BuildOrgFilterQuery(ctx, "organization_id")

    query := `
        SELECT id, organization_id, name, type, config, created_at
        FROM connections
        WHERE ` + whereClause + `
        ORDER BY created_at DESC
    `

    rows, err := db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var connections []*Connection
    for rows.Next() {
        var conn Connection
        err := rows.Scan(
            &conn.ID,
            &conn.OrganizationID,
            &conn.Name,
            &conn.Type,
            &conn.Config,
            &conn.CreatedAt,
        )
        if err != nil {
            return nil, err
        }
        connections = append(connections, &conn)
    }

    return connections, rows.Err()
}
```

---

## 4. Testing Integration

### Integration Test Example

```go
package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "backend-go/internal/middleware"
)

func TestTenantIsolation(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer db.Close()

    // Create test organizations
    org1 := createTestOrg(t, db, "org-1", "Organization 1")
    org2 := createTestOrg(t, db, "org-2", "Organization 2")

    // Create test user with access to org1 only
    user := createTestUser(t, db, "user-1")
    addUserToOrg(t, db, user.ID, org1.ID)

    // Create connections in both orgs
    conn1 := createTestConnection(t, db, org1.ID, "Conn 1")
    conn2 := createTestConnection(t, db, org2.ID, "Conn 2")

    // Create context with user
    ctx := context.WithValue(context.Background(), "user_id", user.ID)
    ctx = context.WithValue(ctx, middleware.UserOrganizationsKey, []string{org1.ID})

    // List connections - should only see org1
    connections, err := listUserConnections(ctx, db)
    assert.NoError(t, err)
    assert.Len(t, connections, 1)
    assert.Equal(t, conn1.ID, connections[0].ID)

    // Try to access org2 connection - should fail
    err = middleware.VerifyOrgAccess(ctx, org2.ID)
    assert.Error(t, err)
}
```

---

## 5. Environment Configuration

```bash
# .env.production

# Database
DATABASE_URL=libsql://your-database.turso.io

# Server
PORT=8080
ENVIRONMENT=production

# SLA Monitoring
SLA_LOG_RETENTION_DAYS=30
SLA_CALCULATION_HOUR=1
SLA_CLEANUP_HOUR=2

# Rate Limiting
DEFAULT_API_CALLS_PER_HOUR=1000

# Domain Verification
DNS_VERIFICATION_TIMEOUT=60s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

---

## 6. Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/pkg/storage/turso/migrations ./migrations

EXPOSE 8080
CMD ["./server"]
```

```yaml
# docker-compose.yml
version: '3.8'

services:
  backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - LOG_LEVEL=info
    restart: unless-stopped
```

---

## 7. Monitoring & Alerts

### Prometheus Metrics Example

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    QuotaChecks = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "quota_checks_total",
            Help: "Total number of quota checks",
        },
        []string{"organization_id", "resource_type", "result"},
    )

    QuotaUsage = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "quota_usage_current",
            Help: "Current quota usage",
        },
        []string{"organization_id", "resource_type"},
    )

    SLAUptime = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "sla_uptime_percentage",
            Help: "SLA uptime percentage",
        },
        []string{"organization_id"},
    )
)
```

---

## Complete Example Repository Structure

```
backend-go/
├── cmd/
│   └── server/
│       └── main.go                      # Main server with full integration
├── internal/
│   ├── middleware/
│   │   ├── tenant_isolation.go
│   │   ├── org_rate_limit.go
│   │   └── sla_tracking.go
│   ├── handlers/
│   │   └── enterprise_handlers.go
│   ├── whitelabel/
│   ├── domains/
│   ├── quotas/
│   └── sla/
├── pkg/
│   └── storage/
│       └── turso/
│           └── migrations/
│               └── 007_white_labeling.sql
├── .env.example
├── docker-compose.yml
├── Dockerfile
└── ENTERPRISE_INTEGRATION_EXAMPLE.md
```

This integration example provides a complete, working reference for implementing all Phase 5 enterprise features in your SQL Studio backend.
