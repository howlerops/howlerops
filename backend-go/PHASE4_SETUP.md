# Phase 4 Setup Guide - Query Templates & Scheduling

## Prerequisites

- Go 1.24.0 or higher
- Turso database connection configured
- Existing backend running (Phases 1-3 complete)

## Installation Steps

### 1. Install Dependencies

The implementation requires the following new dependency:

```bash
cd /Users/jacob_1/projects/sql-studio/backend-go
go get github.com/robfig/cron/v3@latest
go get github.com/go-chi/chi/v5@latest  # If not already installed
go mod tidy
```

### 2. Run Database Migration

Apply the Phase 4 migration to create the necessary tables:

```bash
# Option 1: Using the migration tool
cd /Users/jacob_1/projects/sql-studio/backend-go
go run cmd/migrate/main.go up

# Option 2: Manually via sqlite3
sqlite3 your-database.db < pkg/storage/turso/migrations/004_query_templates.sql
```

Verify the tables were created:

```bash
sqlite3 your-database.db ".tables"
# Should see: query_templates, query_schedules, schedule_executions
```

### 3. Update Server Configuration

Add to your `main.go` or server initialization:

```go
import (
    "github.com/sql-studio/backend-go/internal/templates"
    "github.com/sql-studio/backend-go/internal/scheduler"
    "github.com/sql-studio/backend-go/pkg/storage/turso"
)

func main() {
    // ... existing setup ...

    // Initialize stores
    templateStore := turso.NewTemplateStore(db, logger)
    scheduleStore := turso.NewScheduleStore(db, logger)

    // Initialize templates service
    templateService := templates.NewService(
        templateStore,
        scheduleStore,
        orgRepo,
        logger,
    )

    // Initialize scheduler
    queryExecutor := NewYourQueryExecutor() // Implement QueryExecutor interface
    scheduler := scheduler.NewScheduler(
        scheduleStore,
        templateStore,
        templateService,
        queryExecutor,
        logger,
        scheduler.Config{
            Interval:      1 * time.Minute,
            MaxConcurrent: 10,
            Timeout:       5 * time.Minute,
        },
    )

    // Start scheduler
    if err := scheduler.Start(); err != nil {
        logger.WithError(err).Fatal("Failed to start scheduler")
    }
    defer scheduler.Stop()

    // Register HTTP handlers
    templateHandler := templates.NewHandler(templateService, logger)
    templateHandler.RegisterRoutes(router, authMiddleware)

    // ... start server ...
}
```

### 4. Implement Query Executor

The scheduler needs a query executor. Create your implementation:

```go
// internal/executor/query_executor.go
package executor

import (
    "context"
    "github.com/sql-studio/backend-go/internal/scheduler"
)

type QueryExecutor struct {
    // Your database connection manager
}

func (e *QueryExecutor) ExecuteQuery(ctx context.Context, sql string, connectionID string) (*scheduler.QueryResult, error) {
    // 1. Get database connection by connectionID
    // 2. Execute SQL query
    // 3. Fetch results
    // 4. Return QueryResult with rows, count, duration

    startTime := time.Now()

    // Your execution logic here
    rows, err := db.QueryContext(ctx, sql)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    // Process rows...

    duration := time.Since(startTime)

    return &scheduler.QueryResult{
        Rows:         results,
        RowsReturned: len(results),
        DurationMs:   duration.Milliseconds(),
    }, nil
}
```

### 5. Environment Variables

Add to your `.env` file (optional):

```bash
# Scheduler Configuration
SCHEDULER_INTERVAL=60s           # Check interval (default: 1 minute)
SCHEDULER_MAX_CONCURRENT=10      # Max concurrent executions (default: 10)
SCHEDULER_TIMEOUT=5m             # Query timeout (default: 5 minutes)

# Email Notifications (optional - for future)
EMAIL_SERVICE=sendgrid
SENDGRID_API_KEY=your-key
NOTIFICATION_FROM_EMAIL=noreply@yourdomain.com
```

### 6. Test the Installation

Run the test suite:

```bash
cd /Users/jacob_1/projects/sql-studio/backend-go

# Test parameter substitution
go test ./internal/templates -v -run TestSubstituteParameters

# Test all templates functionality
go test ./internal/templates -v

# Test scheduler (if you have integration tests)
go test ./internal/scheduler -v
```

### 7. Verify API Endpoints

Start the server and test the endpoints:

```bash
# 1. Create a test template
curl -X POST http://localhost:8080/api/templates \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Template",
    "sql_template": "SELECT * FROM users WHERE id = {{user_id}}",
    "parameters": [
      {"name": "user_id", "type": "number", "required": true}
    ],
    "category": "custom"
  }'

# 2. Execute the template
curl -X POST http://localhost:8080/api/templates/TEMPLATE_ID/execute \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "parameters": {"user_id": 123}
  }'

# 3. Create a schedule
curl -X POST http://localhost:8080/api/schedules \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "template_id": "TEMPLATE_ID",
    "name": "Test Schedule",
    "frequency": "*/5 * * * *",
    "parameters": {"user_id": 123}
  }'
```

## Files Created

This implementation added the following files:

### Migration
- `/Users/jacob_1/projects/sql-studio/backend-go/pkg/storage/turso/migrations/004_query_templates.sql`

### Storage Layer
- `/Users/jacob_1/projects/sql-studio/backend-go/pkg/storage/turso/template_store.go`
- `/Users/jacob_1/projects/sql-studio/backend-go/pkg/storage/turso/schedule_store.go`

### Service Layer
- `/Users/jacob_1/projects/sql-studio/backend-go/internal/templates/service.go`
- `/Users/jacob_1/projects/sql-studio/backend-go/internal/templates/parameter_substitution.go`
- `/Users/jacob_1/projects/sql-studio/backend-go/internal/templates/handler.go`

### Scheduler
- `/Users/jacob_1/projects/sql-studio/backend-go/internal/scheduler/scheduler.go`

### Tests
- `/Users/jacob_1/projects/sql-studio/backend-go/internal/templates/parameter_substitution_test.go`

### Documentation
- `/Users/jacob_1/projects/sql-studio/backend-go/QUERY_TEMPLATES_IMPLEMENTATION.md`
- `/Users/jacob_1/projects/sql-studio/backend-go/PHASE4_SETUP.md` (this file)

## Troubleshooting

### Migration Fails

If the migration fails:

```bash
# Check current schema
sqlite3 your-database.db ".schema"

# Drop tables if needed (CAUTION: data loss)
sqlite3 your-database.db "DROP TABLE IF EXISTS schedule_executions"
sqlite3 your-database.db "DROP TABLE IF EXISTS query_schedules"
sqlite3 your-database.db "DROP TABLE IF EXISTS query_templates"

# Re-run migration
go run cmd/migrate/main.go up
```

### Scheduler Not Starting

Check logs for errors:
- Verify database connection
- Check if tables exist
- Ensure cron library is installed

### Parameter Substitution Errors

Common issues:
- Undefined parameters: Ensure all `{{param}}` references are defined
- Type mismatches: Check parameter types match definitions
- SQL injection detected: Review parameter values for dangerous patterns

### Permission Errors

Ensure:
- User has valid authentication token
- User is member of organization (for org-scoped resources)
- User has required permissions (connections:create, connections:view, etc.)

## Next Steps

After setup:

1. **Create Default Templates**
   - Add common query templates for your users
   - Set up example schedules

2. **Configure Monitoring**
   - Set up logging for scheduler events
   - Monitor execution history
   - Track failure rates

3. **Email Notifications**
   - Implement email service integration
   - Configure SMTP or SendGrid
   - Test notification flow

4. **Performance Tuning**
   - Adjust scheduler interval based on load
   - Configure concurrent execution limits
   - Set appropriate query timeouts

5. **Documentation**
   - Share API documentation with frontend team
   - Create user guides for templates
   - Document cron expression syntax for users

## Support

For issues or questions:
- Review `/Users/jacob_1/projects/sql-studio/backend-go/QUERY_TEMPLATES_IMPLEMENTATION.md`
- Check test files for usage examples
- Review handler.go for API endpoint implementations

## Success Checklist

- [ ] Dependencies installed (`go mod tidy` runs without errors)
- [ ] Migration applied successfully
- [ ] Server starts without errors
- [ ] Scheduler logs show it's running
- [ ] API endpoints respond correctly
- [ ] Template creation works
- [ ] Parameter substitution works
- [ ] Schedule creation works
- [ ] Test suite passes

---

**Phase 4 Setup Complete!** 🎉

Your Howlerops backend now supports query templates and scheduled execution.
