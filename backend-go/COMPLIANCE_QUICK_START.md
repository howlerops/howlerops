# Compliance Features - Quick Start Guide

## 5-Minute Setup

### 1. Run Migration

```bash
cd /Users/jacob_1/projects/sql-studio/backend-go
go run cmd/migrate/main.go up
```

### 2. Initialize Services

```go
package main

import (
    "database/sql"
    "github.com/sql-studio/backend-go/internal/audit"
    "github.com/sql-studio/backend-go/internal/backup"
    "github.com/sql-studio/backend-go/internal/gdpr"
    "github.com/sql-studio/backend-go/internal/pii"
    "github.com/sql-studio/backend-go/internal/retention"
    "github.com/sirupsen/logrus"
)

func InitCompliance(db *sql.DB, logger *logrus.Logger) {
    // Create stores
    auditStore := audit.NewStore(db)
    retentionStore := retention.NewStore(db)
    gdprStore := gdpr.NewStore(db)
    backupStore := backup.NewStore(db)
    piiStore := pii.NewStore(db)

    // Create services
    auditLogger := audit.NewDetailedAuditLogger(auditStore, logger)

    archiver := retention.NewLocalArchiver("/var/archives/sqlstudio", logger)
    retentionService := retention.NewService(retentionStore, archiver, logger)

    gdprService := gdpr.NewService(gdprStore, "/var/exports/sqlstudio", logger)

    backupService := backup.NewService(db, backupStore, "/var/backups/sqlstudio", logger)

    piiDetector := pii.NewDetector(piiStore, logger)

    // Start schedulers
    ctx := context.Background()
    go retentionService.StartScheduler(ctx)
    go backupService.StartScheduler(ctx, &backup.BackupOptions{
        BackupType: "full",
        Compress:   true,
        MaxBackups: 30,
    })
}
```

### 3. Register API Routes

```go
import "github.com/sql-studio/backend-go/internal/handlers"

complianceHandler := handlers.NewComplianceHandler(
    retentionService,
    gdprService,
    backupService,
    auditLogger,
    piiDetector,
    logger,
)

complianceHandler.RegisterRoutes(router)
```

---

## Common Operations

### Create Retention Policy

```bash
curl -X POST http://localhost:8080/api/organizations/org-123/retention-policy \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "query_history",
    "retention_days": 90,
    "auto_archive": true,
    "archive_location": "local"
  }'
```

### Request GDPR Data Export

```bash
curl -X POST http://localhost:8080/api/gdpr/export?user_id=user-123
```

### Create Database Backup

```bash
curl -X POST http://localhost:8080/api/admin/backups \
  -H "Content-Type: application/json" \
  -d '{
    "backup_type": "full",
    "compress": true,
    "max_backups": 30
  }'
```

### Scan Data for PII

```bash
curl -X POST http://localhost:8080/api/pii/scan \
  -H "Content-Type: application/json" \
  -d '{
    "data": [
      {"name": "John Doe", "email": "john@example.com"}
    ]
  }'
```

### View Change History

```bash
curl http://localhost:8080/api/audit/detailed/users/user-123
```

---

## Usage Examples

### Log Field Changes

```go
// Detect changes
changes := auditLogger.DetectChanges("users", userID, oldUser, newUser)

// Log to audit trail
err := auditLogger.LogUpdate(ctx, auditLogID, changes)
```

### Apply Retention Policy

```go
// Manual enforcement (normally runs automatically)
err := retentionService.ApplyRetentionPolicies(ctx)
```

### Verify Backup

```go
err := backupService.VerifyBackup(ctx, backupID)
if err != nil {
    log.Error("Backup verification failed:", err)
}
```

### Detect PII in Query Results

```go
results := []map[string]interface{}{
    {"id": 1, "email": "user@example.com"},
}

scanResult, err := piiDetector.ScanQueryResults(ctx, results)
fmt.Printf("Found %d PII fields\n", scanResult.PIIFieldsFound)
```

---

## Scheduled Operations

### Retention Policy Enforcement
- **Schedule:** Daily at 2:00 AM local time
- **Action:** Archive and delete old data per policy
- **Log Location:** `/var/log/sqlstudio/retention.log`

### Automated Backups
- **Schedule:** Daily at 3:00 AM local time
- **Action:** Full database backup with verification
- **Log Location:** `/var/log/sqlstudio/backup.log`

---

## Monitoring

### Check Backup Status

```bash
curl http://localhost:8080/api/admin/backups/stats
```

### View GDPR Requests

```bash
curl http://localhost:8080/api/gdpr/requests?user_id=user-123
```

### List Retention Policies

```bash
curl http://localhost:8080/api/organizations/org-123/retention-policy
```

### View PII Fields

```bash
curl http://localhost:8080/api/pii/fields
```

---

## Troubleshooting

### Backup Failed

```bash
# Check backup logs
tail -f /var/log/sqlstudio/backup.log

# Verify disk space
df -h /var/backups/sqlstudio

# List recent backups
curl http://localhost:8080/api/admin/backups?limit=10
```

### Retention Not Running

```bash
# Check scheduler logs
tail -f /var/log/sqlstudio/retention.log

# Manually trigger
curl -X POST http://localhost:8080/api/admin/retention/apply
```

### GDPR Export Timeout

```bash
# Check export status
curl http://localhost:8080/api/gdpr/export/{request_id}

# Check export path permissions
ls -la /var/exports/sqlstudio
```

---

## Configuration

### Environment Variables

```bash
# Required directories
export BACKUP_PATH=/var/backups/sqlstudio
export ARCHIVE_PATH=/var/archives/sqlstudio
export GDPR_EXPORT_PATH=/var/exports/sqlstudio

# Create directories
mkdir -p $BACKUP_PATH $ARCHIVE_PATH $GDPR_EXPORT_PATH

# Set permissions
chmod 750 $BACKUP_PATH $ARCHIVE_PATH $GDPR_EXPORT_PATH
```

---

## Testing

### Run PII Detection Tests

```bash
cd /Users/jacob_1/projects/sql-studio/backend-go
go test -v ./internal/pii/...
```

### Test Backup Creation

```bash
curl -X POST http://localhost:8080/api/admin/backups \
  -H "Content-Type: application/json" \
  -d '{"backup_type": "full"}'
```

### Test Retention Policy

```bash
# Create test policy
curl -X POST http://localhost:8080/api/organizations/test-org/retention-policy \
  -H "Content-Type: application/json" \
  -d '{
    "resource_type": "query_history",
    "retention_days": 1,
    "auto_archive": false
  }'

# Wait 24 hours or manually trigger
# Verify old data is deleted
```

---

## Resources

- **Full Documentation:** `/backend-go/DATA_COMPLIANCE_GUIDE.md`
- **Implementation Summary:** `/PHASE_5_IMPLEMENTATION_SUMMARY.md`
- **API Reference:** See documentation sections
- **Support:** support@sqlstudio.com

---

## Checklist

### Initial Setup
- [ ] Run database migration
- [ ] Create required directories
- [ ] Set environment variables
- [ ] Initialize services
- [ ] Register API routes
- [ ] Start schedulers

### Testing
- [ ] Create test retention policy
- [ ] Trigger manual backup
- [ ] Request GDPR export
- [ ] Scan sample data for PII
- [ ] Verify audit logging
- [ ] Check scheduled tasks

### Production
- [ ] Configure backup retention
- [ ] Set up monitoring alerts
- [ ] Document recovery procedures
- [ ] Test restore process
- [ ] Review compliance logs
- [ ] Schedule regular audits

---

*Quick Start Guide v1.0*
*Last Updated: 2024-10-24*
