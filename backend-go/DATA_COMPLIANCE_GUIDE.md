# Data Management & Compliance Guide

## Overview

SQL Studio includes comprehensive data management and compliance features designed for enterprise deployments. This guide covers all compliance capabilities including GDPR, data retention, backup/restore, audit logging, and PII detection.

## Table of Contents

1. [GDPR Compliance](#gdpr-compliance)
2. [Data Retention Policies](#data-retention-policies)
3. [Backup & Restore](#backup--restore)
4. [Enhanced Audit Logging](#enhanced-audit-logging)
5. [PII Detection & Handling](#pii-detection--handling)
6. [API Reference](#api-reference)
7. [Configuration](#configuration)
8. [Best Practices](#best-practices)

---

## GDPR Compliance

SQL Studio provides full GDPR compliance features including the right to access, right to erasure, and data portability.

### Right to Access (Article 15)

Users can request a complete export of all their data:

```bash
# Request data export
curl -X POST http://localhost:8080/api/gdpr/export?user_id=<USER_ID>

# Check export status
curl http://localhost:8080/api/gdpr/export/<REQUEST_ID>
```

**Export includes:**
- User profile data
- All database connections
- Saved queries and history
- Query templates
- Scheduled queries
- Organization memberships
- Audit logs

The export is generated as a JSON file with all user data in a structured format.

### Right to Erasure (Article 17)

Users can request complete deletion of their data:

```bash
# Request data deletion
curl -X POST http://localhost:8080/api/gdpr/delete?user_id=<USER_ID>

# Monitor deletion status
curl http://localhost:8080/api/gdpr/requests?user_id=<USER_ID>
```

**Deletion process:**
1. All connections are deleted
2. All queries are deleted
3. Query history is deleted
4. Templates are deleted
5. Scheduled queries are deleted
6. Audit logs are anonymized (kept for compliance but PII removed)
7. User account is deleted

**Important:** Audit logs are anonymized rather than deleted to maintain compliance audit trails. The user_id is replaced with `[DELETED]` but the action records remain.

### Request Management

Track all GDPR requests:

```bash
# List all GDPR requests for a user
curl http://localhost:8080/api/gdpr/requests?user_id=<USER_ID>
```

Response includes:
- Request ID
- Request type (export/delete)
- Status (pending/processing/completed/failed)
- Timestamps
- Export URL (for completed exports)

---

## Data Retention Policies

Automate data lifecycle management with organization-level retention policies.

### Creating Retention Policies

```bash
POST /api/organizations/{org_id}/retention-policy
Content-Type: application/json

{
  "resource_type": "query_history",
  "retention_days": 90,
  "auto_archive": true,
  "archive_location": "local"
}
```

**Supported resource types:**
- `query_history` - Query execution history
- `audit_logs` - Audit log entries
- `connections` - Database connections
- `templates` - Query templates

### Retention Policy Configuration

| Field | Type | Description |
|-------|------|-------------|
| `resource_type` | string | Type of data to manage |
| `retention_days` | int | Days to retain data (1-3650) |
| `auto_archive` | boolean | Archive before deletion |
| `archive_location` | string | Archive storage location |

### Archive Storage

**Local archival:**
- Archives stored in compressed JSON format (gzip)
- Location: `{backup_path}/archives/`
- Format: `{org_id}_{resource_type}_{timestamp}.json.gz`

**Archive structure:**
```json
{
  "resource_type": "query_history",
  "records": [...],
  "metadata": {
    "archived_at": 1698765432,
    "record_count": 1500,
    "archive_version": "1.0"
  }
}
```

### Retention Policy Enforcement

Policies are automatically enforced daily at 2 AM local time.

**Manual enforcement:**
```go
// In application code
err := retentionService.ApplyRetentionPolicies(ctx)
```

### Viewing Retention Statistics

```bash
GET /api/organizations/{org_id}/retention-stats/{resource_type}
```

Response:
```json
{
  "resource_type": "query_history",
  "total_records": 5000,
  "oldest_record": "2023-01-15T10:30:00Z",
  "records_to_archive": 1200,
  "estimated_size_bytes": 524288
}
```

---

## Backup & Restore

Automated database backups with scheduled execution and verification.

### Creating Backups

```bash
POST /api/admin/backups
Content-Type: application/json

{
  "backup_type": "full",
  "compress": true,
  "max_backups": 30
}
```

**Backup types:**
- `full` - Complete database backup
- `incremental` - Incremental backup (future enhancement)

### Backup Process

SQL Studio uses SQLite's `VACUUM INTO` command for consistent snapshots:

1. Creates a consistent point-in-time snapshot
2. Compresses the backup (optional)
3. Stores metadata in `database_backups` table
4. Verifies backup integrity
5. Cleans up old backups (if max_backups set)

### Listing Backups

```bash
GET /api/admin/backups?limit=50
```

Response:
```json
[
  {
    "id": "backup-123",
    "backup_type": "full",
    "status": "completed",
    "file_path": "/backups/backup_full_20231115_030000.db",
    "file_size": 52428800,
    "started_at": "2023-11-15T03:00:00Z",
    "completed_at": "2023-11-15T03:05:32Z"
  }
]
```

### Backup Verification

```go
// Verify backup integrity
err := backupService.VerifyBackup(ctx, backupID)
```

Verification checks:
- File exists
- File size matches metadata
- Database can be opened
- Basic integrity query succeeds

### Restore Process

```bash
POST /api/admin/backups/{backup_id}/restore
Content-Type: application/json

{
  "dry_run": false
}
```

**Important:** Restore requires application restart. The process:
1. Verifies backup exists and is complete
2. Stops all connections
3. Replaces current database file
4. Restarts application

### Automated Backups

Backups run automatically at 3 AM local time daily.

**Configuration:**
```go
backupService.StartScheduler(ctx, &backup.BackupOptions{
    BackupType: "full",
    Compress:   true,
    MaxBackups: 30,
})
```

### Backup Statistics

```bash
GET /api/admin/backups/stats
```

Response:
```json
{
  "total_backups": 30,
  "total_size_bytes": 1572864000,
  "latest_backup": "2023-11-15T03:00:00Z",
  "oldest_backup": "2023-10-16T03:00:00Z",
  "successful_backups": 30,
  "failed_backups": 0
}
```

---

## Enhanced Audit Logging

Field-level change tracking for comprehensive audit trails.

### Field-Level Change Tracking

Standard audit logs record high-level actions. Detailed audit logs track individual field changes:

```go
// Detect changes between old and new records
changes := auditLogger.DetectChanges("users", userID, oldUser, newUser)

// Log detailed changes
err := auditLogger.LogUpdate(ctx, auditLogID, changes)
```

### Viewing Change History

```bash
# Get all changes for a record
GET /api/audit/detailed/users/user-123

# Get changes for specific field
GET /api/audit/field/users/user-123/email
```

Response:
```json
{
  "table_name": "users",
  "record_id": "user-123",
  "fields": {
    "email": [
      {
        "field_name": "email",
        "old_value": "old@example.com",
        "new_value": "new@example.com",
        "field_type": "pii",
        "changed_at": "2023-11-15T14:30:00Z",
        "changed_by": "admin-user",
        "audit_id": "audit-789"
      }
    ]
  }
}
```

### Field Classification

Fields are automatically classified:
- `pii` - Contains personally identifiable information
- `sensitive` - Sensitive data (passwords, tokens, keys)
- `normal` - Regular data

### PII Access Tracking

Track access to PII fields:

```go
// Log PII access
auditLogger.LogPIIAccess(ctx, &audit.PIIAccessLog{
    UserID:     userID,
    TableName:  "users",
    FieldName:  "email",
    RecordID:   recordID,
    AccessType: "read",
    IPAddress:  clientIP,
})
```

---

## PII Detection & Handling

Automated detection and masking of personally identifiable information.

### Automatic PII Detection

The PII detector uses pattern matching and field name analysis to identify PII:

**Supported PII types:**
- Email addresses
- Phone numbers
- Social Security Numbers (SSN)
- Credit card numbers (with Luhn validation)
- Addresses
- Names

### Scanning Data for PII

```bash
POST /api/pii/scan
Content-Type: application/json

{
  "data": [
    {
      "id": 1,
      "name": "John Doe",
      "email": "john@example.com",
      "phone": "15551234567"
    }
  ]
}
```

Response:
```json
{
  "total_fields": 4,
  "pii_fields_found": 3,
  "matches": [
    {
      "field": "name",
      "type": "name",
      "value": "John Doe",
      "confidence_score": 0.80,
      "masked": false,
      "masked_value": "J*** D***"
    },
    {
      "field": "email",
      "type": "email",
      "value": "john@example.com",
      "confidence_score": 0.95,
      "masked": false,
      "masked_value": "jo***@example.com"
    }
  ],
  "scanned_at": "2023-11-15T14:30:00Z"
}
```

### Registering PII Fields

Manually register fields as containing PII:

```bash
POST /api/pii/fields
Content-Type: application/json

{
  "table_name": "customers",
  "field_name": "tax_id",
  "pii_type": "ssn"
}
```

### Listing PII Fields

```bash
GET /api/pii/fields
```

Response:
```json
[
  {
    "id": "pii-123",
    "table_name": "customers",
    "field_name": "tax_id",
    "pii_type": "ssn",
    "detection_method": "manual",
    "confidence_score": 1.0,
    "verified": true,
    "created_at": "2023-11-15T10:00:00Z"
  }
]
```

### PII Masking

Values are automatically masked based on type:

| PII Type | Masking Strategy | Example |
|----------|------------------|---------|
| Email | Keep first 2 chars + domain | `jo***@example.com` |
| Phone | Keep last 4 digits | `***-***-4567` |
| SSN | Keep last 4 digits | `***-**-6789` |
| Credit Card | Keep last 4 digits | `****-****-****-0366` |
| Address | Keep last 10 chars | `*** 90210 CA` |
| Name | Keep first char of each part | `J*** D***` |

---

## API Reference

### Data Retention Endpoints

```
POST   /api/organizations/{id}/retention-policy           Create policy
GET    /api/organizations/{id}/retention-policy           List policies
PUT    /api/organizations/{id}/retention-policy/{type}    Update policy
DELETE /api/organizations/{id}/retention-policy/{type}    Delete policy
GET    /api/organizations/{id}/retention-stats/{type}     Get statistics
```

### GDPR Endpoints

```
POST   /api/gdpr/export                     Request data export
GET    /api/gdpr/export/{request_id}        Get export status
POST   /api/gdpr/delete                     Request data deletion
GET    /api/gdpr/requests                   List user requests
```

### Backup Endpoints (Admin Only)

```
POST   /api/admin/backups                   Create backup
GET    /api/admin/backups                   List backups
GET    /api/admin/backups/{id}              Get backup details
POST   /api/admin/backups/{id}/restore      Restore backup
DELETE /api/admin/backups/{id}              Delete backup
GET    /api/admin/backups/stats             Get backup statistics
```

### Audit Log Endpoints

```
GET    /api/audit/detailed/{table}/{id}           Get record change history
GET    /api/audit/field/{table}/{id}/{field}      Get field change history
```

### PII Detection Endpoints

```
POST   /api/pii/scan                        Scan data for PII
GET    /api/pii/fields                      List registered PII fields
POST   /api/pii/fields                      Register PII field
POST   /api/pii/fields/{id}/verify          Verify PII field
```

---

## Configuration

### Environment Variables

```bash
# Backup configuration
BACKUP_PATH=/var/backups/sqlstudio
BACKUP_SCHEDULE="0 3 * * *"  # 3 AM daily
MAX_BACKUPS=30

# Retention configuration
RETENTION_SCHEDULE="0 2 * * *"  # 2 AM daily
ARCHIVE_PATH=/var/archives/sqlstudio

# GDPR configuration
GDPR_EXPORT_PATH=/var/exports/sqlstudio
GDPR_RETENTION_DAYS=90  # Keep export files for 90 days

# Audit configuration
DETAILED_AUDIT_ENABLED=true
PII_DETECTION_ENABLED=true
```

### Application Configuration

```go
// Initialize services
retentionService := retention.NewService(
    retentionStore,
    archiver,
    logger,
)

gdprService := gdpr.NewService(
    gdprStore,
    exportPath,
    logger,
)

backupService := backup.NewService(
    db,
    backupStore,
    backupPath,
    logger,
)

// Start schedulers
go retentionService.StartScheduler(ctx)
go backupService.StartScheduler(ctx, backupOptions)
```

---

## Best Practices

### Data Retention

1. **Set appropriate retention periods**
   - Audit logs: 7 years (compliance requirement)
   - Query history: 90-180 days
   - Connections: Review annually
   - Templates: No automatic deletion

2. **Always enable auto-archive**
   - Maintains compliance while reducing database size
   - Enables data recovery if needed
   - Archive to separate storage for disaster recovery

3. **Monitor retention statistics**
   - Review monthly to optimize policies
   - Adjust retention periods based on usage
   - Alert on large pending archives

### Backup Strategy

1. **Follow the 3-2-1 rule**
   - 3 copies of data
   - 2 different media types
   - 1 offsite copy

2. **Test restores regularly**
   - Monthly restore tests
   - Document restore procedures
   - Measure Recovery Time Objective (RTO)

3. **Verify backup integrity**
   - Automated verification after each backup
   - Alert on failed verifications
   - Manual spot checks

4. **Backup retention**
   - Keep 30 daily backups
   - Keep 12 monthly backups
   - Keep yearly backups indefinitely

### GDPR Compliance

1. **Data minimization**
   - Only collect necessary data
   - Implement retention policies
   - Regular data audits

2. **Right to access**
   - Process requests within 30 days
   - Provide data in machine-readable format
   - Include all systems and backups

3. **Right to erasure**
   - Verify identity before deletion
   - Document deletion process
   - Anonymize audit logs (don't delete)

4. **Consent management**
   - Track consent for data processing
   - Enable easy consent withdrawal
   - Document legal basis for processing

### PII Handling

1. **Minimize PII exposure**
   - Use PII detection in development/staging
   - Mask PII in logs and error messages
   - Encrypt PII at rest and in transit

2. **Access controls**
   - Role-based access to PII
   - Audit all PII access
   - Implement need-to-know principle

3. **Register known PII fields**
   - Document all PII fields
   - Verify automated detection
   - Update as schema changes

### Audit Logging

1. **Log all critical actions**
   - Data access and modifications
   - Permission changes
   - Configuration changes
   - Failed authentication attempts

2. **Include sufficient context**
   - User ID and IP address
   - Timestamp with timezone
   - Action details
   - Before/after values

3. **Protect audit logs**
   - Immutable storage
   - Regular backups
   - Access restricted to auditors

---

## Compliance Certifications

SQL Studio's compliance features support:

- **GDPR** (General Data Protection Regulation)
- **SOC 2** (Service Organization Control 2)
- **HIPAA** (Health Insurance Portability and Accountability Act)
- **ISO 27001** (Information Security Management)
- **PCI DSS** (Payment Card Industry Data Security Standard)

### Audit Readiness

When preparing for compliance audits:

1. **Generate compliance reports**
   - Retention policy summary
   - Backup verification logs
   - GDPR request history
   - PII access logs

2. **Document procedures**
   - Data lifecycle management
   - Incident response
   - Backup and recovery
   - Access control policies

3. **Demonstrate controls**
   - Automated retention enforcement
   - Backup verification
   - PII detection and masking
   - Comprehensive audit trails

---

## Troubleshooting

### Retention Policy Not Running

Check scheduler status:
```bash
# View retention logs
tail -f /var/log/sqlstudio/retention.log

# Manually trigger retention
curl -X POST http://localhost:8080/api/admin/retention/apply
```

### Backup Failures

Common issues:
- Insufficient disk space
- Database locks
- Permission errors

Check backup logs:
```bash
tail -f /var/log/sqlstudio/backup.log
```

### GDPR Export Timeout

For large exports:
- Increase timeout settings
- Monitor background job progress
- Check export file generation

### PII Detection False Positives

Adjust confidence thresholds or manually verify:
```bash
POST /api/pii/fields/{id}/verify
```

---

## Support

For compliance-related questions or issues:

- Documentation: https://docs.sqlstudio.com/compliance
- Support: support@sqlstudio.com
- Security: security@sqlstudio.com

**Security Vulnerabilities:** Please report to security@sqlstudio.com (PGP key available)

---

*Last updated: 2024-10-24*
*Version: 1.0.0*
