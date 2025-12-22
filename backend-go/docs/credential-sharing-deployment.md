# Credential Sharing System - Deployment Guide

## Overview

This document covers deployment, monitoring, and operational procedures for the Organization Envelope Key (OEK) credential sharing system. The OEK system enables secure sharing of database credentials within organizations using envelope encryption.

**Architecture Summary:**
- Each organization has one Organization Envelope Key (OEK)
- OEK is encrypted separately for each organization member using their master key
- Shared credentials are encrypted with the OEK (AES-256-GCM)
- Members decrypt their wrapped OEK copy, then use it to decrypt shared credentials

---

## Pre-Deployment Checklist

### Database Migrations

- [ ] Verify migration 008_encrypted_passwords.sql has been applied (prerequisite)
- [ ] Run migration 009_shared_credentials.sql
- [ ] Verify tables created:
  - `organization_envelope_keys`
  - `shared_credentials`
  - `credential_access_log`
- [ ] Verify `encryption_type` column added to `connection_templates`
- [ ] Verify all indexes created (see verification queries below)

### Environment Variables

No new environment variables required. The OEK system uses existing PBKDF2 settings from migration 008:
- PBKDF2 iterations: 600,000 (OWASP 2023 recommendation)
- Encryption: AES-256-GCM
- All cryptographic parameters stored per-record in the database

### Infrastructure Requirements

- [ ] Turso database access configured
- [ ] Backend deployment pipeline ready
- [ ] Monitoring/alerting infrastructure available
- [ ] Backup procedures verified

---

## Deployment Steps

### 1. Pre-Migration Backup

```bash
# Create a backup before migration
turso db shell <database-name> ".backup backup_pre_oek_$(date +%Y%m%d_%H%M%S).db"

# Alternatively, export critical tables
turso db shell <database-name> "SELECT * FROM connection_templates WHERE visibility = 'shared'" > shared_connections_backup.sql
```

### 2. Apply Database Migration

```bash
# Apply migration 009 (shared credentials and OEK tables)
turso db shell <database-name> < pkg/storage/turso/migrations/009_shared_credentials.sql

# Verify tables were created
turso db shell <database-name> ".tables"
```

**Expected output should include:**
- `organization_envelope_keys`
- `shared_credentials`
- `credential_access_log`

### 3. Verify Migration Success

```bash
# Check organization_envelope_keys table structure
turso db shell <database-name> ".schema organization_envelope_keys"

# Check shared_credentials table structure
turso db shell <database-name> ".schema shared_credentials"

# Check credential_access_log table structure
turso db shell <database-name> ".schema credential_access_log"

# Verify encryption_type column added to connection_templates
turso db shell <database-name> "PRAGMA table_info(connection_templates)" | grep encryption_type
```

### 4. Verify Indexes Created

```sql
-- List all indexes for OEK-related tables
SELECT name, tbl_name FROM sqlite_master
WHERE type = 'index'
AND tbl_name IN ('organization_envelope_keys', 'shared_credentials', 'credential_access_log')
ORDER BY tbl_name, name;
```

**Expected indexes:**
- `idx_oek_organization_id`
- `idx_oek_user_id`
- `idx_oek_org_user_version`
- `idx_oek_updated`
- `idx_shared_creds_connection_id`
- `idx_shared_creds_organization_id`
- `idx_shared_creds_created_by`
- `idx_shared_creds_org_connection`
- `idx_shared_creds_updated`
- `idx_cred_log_connection_id`
- `idx_cred_log_organization_id`
- `idx_cred_log_user_id`
- `idx_cred_log_timestamp`
- `idx_cred_log_action`
- `idx_cred_log_success`
- `idx_cred_log_org_user_time`
- `idx_connections_encryption_type`
- `idx_connections_org_encryption`

### 5. Run Migration Diagnostic (Optional)

If a migration diagnostic script exists:

```bash
cd backend-go
go run scripts/migrate_shared_credentials.go --dry-run
```

### 6. Deploy Backend

Standard deployment process - no special steps required beyond ensuring the new tables exist.

```bash
# Example deployment commands (adjust for your infrastructure)
docker build -t howlerops-backend:latest .
docker push howlerops-backend:latest

# Or for Kubernetes
kubectl apply -f k8s/backend-deployment.yaml
kubectl rollout status deployment/backend
```

### 7. Post-Deployment Verification

```bash
# Verify backend health
curl -s https://api.example.com/health | jq .

# Test a simple OEK operation (if test endpoint exists)
curl -s -H "Authorization: Bearer $TOKEN" https://api.example.com/api/organizations/$ORG_ID/oek-status
```

---

## Monitoring

### Key Metrics to Watch

#### 1. Credential Access Logs

Monitor the distribution of actions and their success rates:

```sql
-- Action distribution with success/failure rates
SELECT
    action,
    COUNT(*) as total,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failure_count,
    ROUND(100.0 * SUM(CASE WHEN success THEN 1 ELSE 0 END) / COUNT(*), 2) as success_rate
FROM credential_access_log
WHERE timestamp > unixepoch() - 86400  -- Last 24 hours
GROUP BY action
ORDER BY total DESC;
```

**Alert on:** Failure rate exceeding 5% for any action type.

#### 2. OEK Operations

Track share, decrypt, provision, and revoke actions:

```sql
-- OEK operations per hour
SELECT
    strftime('%Y-%m-%d %H:00', datetime(timestamp, 'unixepoch')) as hour,
    action,
    COUNT(*) as count
FROM credential_access_log
WHERE timestamp > unixepoch() - 86400
GROUP BY hour, action
ORDER BY hour DESC;
```

**Alert on:** Unusual patterns (e.g., decrypt attempts 10x normal rate).

#### 3. Failed Decrypt Attempts by User

Identify potential security issues:

```sql
-- Users with high failure rates in last hour
SELECT
    user_id,
    COUNT(*) as failures,
    MAX(datetime(timestamp, 'unixepoch')) as last_failure
FROM credential_access_log
WHERE action = 'decrypt'
    AND success = 0
    AND timestamp > unixepoch() - 3600
GROUP BY user_id
HAVING failures > 5
ORDER BY failures DESC;
```

**Alert on:** Any user exceeding 10 failed decrypts per hour.

### Grafana Dashboard Queries

#### Credential Operations Per Hour

```sql
SELECT
    strftime('%Y-%m-%d %H:00', datetime(timestamp, 'unixepoch')) as hour,
    action,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failure_count
FROM credential_access_log
WHERE timestamp > unixepoch() - 86400
GROUP BY hour, action
ORDER BY hour DESC;
```

#### Active Organizations Using OEK

```sql
SELECT
    COUNT(DISTINCT organization_id) as active_orgs,
    COUNT(DISTINCT user_id) as active_users,
    COUNT(*) as total_oek_records
FROM organization_envelope_keys;
```

#### Shared Credentials Overview

```sql
SELECT
    o.name as organization_name,
    COUNT(DISTINCT sc.connection_id) as shared_connections,
    COUNT(DISTINCT oek.user_id) as members_with_oek
FROM organizations o
LEFT JOIN shared_credentials sc ON sc.organization_id = o.id
LEFT JOIN organization_envelope_keys oek ON oek.organization_id = o.id
GROUP BY o.id, o.name
ORDER BY shared_connections DESC
LIMIT 20;
```

#### Error Distribution

```sql
SELECT
    error_message,
    COUNT(*) as occurrences,
    COUNT(DISTINCT user_id) as affected_users
FROM credential_access_log
WHERE success = 0
    AND timestamp > unixepoch() - 86400
GROUP BY error_message
ORDER BY occurrences DESC
LIMIT 10;
```

### Prometheus Metrics (If Instrumented)

```
# Latency
credential_decrypt_duration_seconds{action="decrypt"}
credential_decrypt_duration_seconds{action="share"}

# Failures
credential_decrypt_errors_total{action="decrypt"}
credential_decrypt_errors_total{action="share"}

# Operations
credential_operations_total{action="share"}
credential_operations_total{action="decrypt"}
credential_operations_total{action="revoke"}

# Key rotations
org_envelope_key_rotations_total{org_id="..."}
```

### Alert Conditions

| Condition | Threshold | Severity | Action |
|-----------|-----------|----------|--------|
| Decrypt error rate | > 5% | Warning | Investigate logs |
| Decrypt error rate | > 15% | Critical | Page on-call |
| Decrypt latency p99 | > 500ms | Warning | Check database performance |
| Failed attempts per user | > 10/min | Warning | Review user activity |
| OEK not found for active member | Any | Critical | Data inconsistency |

---

## Rollback Plan

### If Issues Occur

The OEK implementation is designed for safe rollback:

1. **Immediate Response**: Feature flag to disable OEK sharing (fall back to legacy direct encryption)
2. **Data Safety**: OEK data is additive - original credential store (`encrypted_credentials`) remains unchanged
3. **Gradual Recovery**: Users can continue using personal connections while issues are investigated

### Rollback Levels

#### Level 1: Feature Flag Disable (Fastest)

Disable OEK sharing while keeping data intact:

```bash
# Set environment variable or feature flag
export DISABLE_OEK_SHARING=true

# Restart backend to pick up configuration
kubectl rollout restart deployment/backend
```

Users will:
- Continue to access their personal connections normally
- Be unable to share new connections
- Be unable to decrypt previously shared connections (temporary)

#### Level 2: Backend Version Rollback

Revert to the previous backend version:

```bash
# Kubernetes rollback
kubectl rollout undo deployment/backend

# Verify rollback
kubectl rollout status deployment/backend

# Or explicit version
kubectl set image deployment/backend backend=howlerops-backend:v1.2.3
```

#### Level 3: Full Data Rollback (Last Resort)

Only if data corruption is detected:

```sql
-- Keep tables but clear OEK data if corrupted
-- WARNING: This removes all shared credential access

-- Backup first
SELECT * FROM organization_envelope_keys;
SELECT * FROM shared_credentials;
SELECT * FROM credential_access_log;

-- Reset shared connections to personal
UPDATE connection_templates
SET encryption_type = 'personal', visibility = 'personal'
WHERE encryption_type = 'shared';

-- Clear OEK data
DELETE FROM shared_credentials;
DELETE FROM organization_envelope_keys;

-- Note: credential_access_log should be preserved for audit
```

### Post-Rollback Recovery

After fixing issues:

1. Re-deploy fixed backend
2. Users re-share connections (re-encrypts with OEK)
3. Monitor closely for 24-48 hours

---

## Security Considerations

### Key Points

1. **OEK Generation**: One OEK per organization, generated on first member join
2. **OEK Distribution**: OEK encrypted per-user with their master key
3. **Zero-Knowledge**: Master keys never stored on server in plaintext
4. **Audit Trail**: All operations logged to `credential_access_log`
5. **Immediate Revocation**: Deleting OEK record revokes access instantly

### Security Monitoring

#### Watch For:

1. **Multiple failed decrypt attempts from same user**
   ```sql
   SELECT user_id, ip_address, COUNT(*) as attempts
   FROM credential_access_log
   WHERE action = 'decrypt' AND success = 0
       AND timestamp > unixepoch() - 3600
   GROUP BY user_id, ip_address
   HAVING attempts > 5;
   ```

2. **Unusual access patterns (time of day, frequency)**
   ```sql
   SELECT
       user_id,
       strftime('%H', datetime(timestamp, 'unixepoch')) as hour,
       COUNT(*) as access_count
   FROM credential_access_log
   WHERE timestamp > unixepoch() - 86400
   GROUP BY user_id, hour
   HAVING access_count > 50;
   ```

3. **Access from new IP addresses**
   ```sql
   SELECT DISTINCT user_id, ip_address,
       MIN(datetime(timestamp, 'unixepoch')) as first_seen
   FROM credential_access_log
   WHERE timestamp > unixepoch() - 86400
   GROUP BY user_id, ip_address
   HAVING first_seen > datetime('now', '-1 day');
   ```

4. **OEK revocation failures**
   ```sql
   SELECT * FROM credential_access_log
   WHERE action = 'revoke' AND success = 0
   ORDER BY timestamp DESC
   LIMIT 20;
   ```

### Security Incident Response

If suspicious activity is detected:

1. **Immediate**: Lock affected user account
2. **Investigate**: Review audit logs for scope of access
3. **Contain**: Revoke OEK for affected users
4. **Remediate**: Force password reset, rotate OEK if needed
5. **Report**: Document incident per security policy

---

## Troubleshooting

### Common Issues

#### 1. "User does not have access to organization's OEK"

**Symptoms:** User cannot decrypt shared credentials, error logged in audit.

**Causes:**
- User joined organization but OEK was not provisioned
- OEK record was inadvertently deleted
- Database sync issue in distributed setup

**Diagnosis:**
```sql
-- Check if OEK exists for user
SELECT * FROM organization_envelope_keys
WHERE user_id = 'user-123' AND organization_id = 'org-456';

-- Check organization membership
SELECT * FROM organization_members
WHERE user_id = 'user-123' AND organization_id = 'org-456';
```

**Resolution:**
- If member exists but OEK missing: Existing member must provision access
- User may need to "re-join" the organization
- Admin can trigger OEK provisioning via API

#### 2. "Failed to decrypt OEK"

**Symptoms:** Decryption fails with authentication error.

**Causes:**
- Wrong master key (user using different device/browser)
- Corrupted encrypted data
- Key version mismatch

**Diagnosis:**
```sql
-- Check OEK key version
SELECT key_version, updated_at
FROM organization_envelope_keys
WHERE user_id = 'user-123' AND organization_id = 'org-456';

-- Check user's master key
SELECT version, updated_at
FROM user_master_keys
WHERE user_id = 'user-123';
```

**Resolution:**
- User should re-authenticate (logout/login)
- If master key was rotated, OEK needs re-encryption
- Check for data corruption (auth_tag mismatch indicates tampering)

#### 3. "Shared credential not found"

**Symptoms:** Connection exists but shared credential record missing.

**Causes:**
- Connection marked as shared but credential not migrated
- Race condition during share operation
- Partial share failure

**Diagnosis:**
```sql
-- Check connection status
SELECT id, encryption_type, visibility, organization_id
FROM connection_templates
WHERE id = 'conn-789';

-- Check if shared credential exists
SELECT * FROM shared_credentials
WHERE connection_id = 'conn-789';
```

**Resolution:**
- Owner needs to re-share the connection
- This triggers re-encryption with OEK

#### 4. Concurrent Modification Errors

**Symptoms:** "database is locked" or unique constraint violations.

**Causes:**
- Multiple users provisioning OEK simultaneously
- Race condition in share operations

**Diagnosis:**
```sql
-- Check for duplicate records
SELECT organization_id, user_id, key_version, COUNT(*)
FROM organization_envelope_keys
GROUP BY organization_id, user_id, key_version
HAVING COUNT(*) > 1;
```

**Resolution:**
- Built-in retry logic should handle most cases
- If persistent, check for serialization issues
- Consider adding explicit locking for critical operations

#### 5. Audit Log Growing Too Large

**Symptoms:** Slow queries on credential_access_log, disk usage increasing.

**Diagnosis:**
```sql
SELECT COUNT(*) as total_records,
       MIN(datetime(timestamp, 'unixepoch')) as oldest,
       MAX(datetime(timestamp, 'unixepoch')) as newest
FROM credential_access_log;
```

**Resolution:**
- Implement log rotation/archival
- Archive records older than retention period (e.g., 90 days)

```sql
-- Archive old logs (example: older than 90 days)
INSERT INTO credential_access_log_archive
SELECT * FROM credential_access_log
WHERE timestamp < unixepoch() - (90 * 86400);

DELETE FROM credential_access_log
WHERE timestamp < unixepoch() - (90 * 86400);
```

---

## Support Runbook

### Investigating User Access Issues

```sql
-- Step 1: Check user's OEK status
SELECT
    oek.id,
    oek.organization_id,
    oek.key_version,
    datetime(oek.created_at, 'unixepoch') as created,
    datetime(oek.updated_at, 'unixepoch') as updated
FROM organization_envelope_keys oek
WHERE oek.user_id = 'USER_ID_HERE';

-- Step 2: Check organization membership
SELECT
    om.organization_id,
    om.role,
    datetime(om.created_at, 'unixepoch') as joined
FROM organization_members om
WHERE om.user_id = 'USER_ID_HERE';

-- Step 3: Check recent access attempts
SELECT
    datetime(timestamp, 'unixepoch') as time,
    action,
    success,
    error_message,
    ip_address
FROM credential_access_log
WHERE user_id = 'USER_ID_HERE'
ORDER BY timestamp DESC
LIMIT 20;

-- Step 4: Check specific connection
SELECT
    ct.id,
    ct.name,
    ct.encryption_type,
    ct.visibility,
    ct.organization_id
FROM connection_templates ct
WHERE ct.id = 'CONNECTION_ID_HERE';

-- Step 5: Check shared credential exists
SELECT
    sc.id,
    sc.connection_id,
    sc.organization_id,
    sc.created_by,
    datetime(sc.created_at, 'unixepoch') as created
FROM shared_credentials sc
WHERE sc.connection_id = 'CONNECTION_ID_HERE';
```

### Diagnosing Organization-Wide Issues

```sql
-- Check organization's OEK distribution
SELECT
    u.email,
    oek.key_version,
    datetime(oek.created_at, 'unixepoch') as oek_created
FROM organization_envelope_keys oek
JOIN users u ON u.id = oek.user_id
WHERE oek.organization_id = 'ORG_ID_HERE'
ORDER BY oek.created_at DESC;

-- Check organization's shared connections
SELECT
    ct.name,
    sc.created_by,
    datetime(sc.created_at, 'unixepoch') as shared_at
FROM shared_credentials sc
JOIN connection_templates ct ON ct.id = sc.connection_id
WHERE sc.organization_id = 'ORG_ID_HERE'
ORDER BY sc.created_at DESC;

-- Check for OEK version inconsistencies
SELECT
    organization_id,
    key_version,
    COUNT(*) as member_count
FROM organization_envelope_keys
WHERE organization_id = 'ORG_ID_HERE'
GROUP BY organization_id, key_version;
```

---

## User Migration

### Existing Shared Connections

Users with existing shared connections (pre-OEK) will need to:

1. Navigate to connection settings
2. Click "Re-share" to encrypt with OEK
3. This happens automatically on next connection use (if auto-migration enabled)

### Manual Migration Script

If batch migration is needed:

```sql
-- Identify connections needing migration
SELECT
    ct.id,
    ct.name,
    ct.created_by,
    ct.organization_id
FROM connection_templates ct
WHERE ct.visibility = 'shared'
    AND ct.encryption_type = 'personal'
    AND ct.organization_id IS NOT NULL;
```

Migration requires the connection creator to be logged in (to access their master key for re-encryption).

---

## Performance Tuning

### Index Optimization

If queries are slow, verify indexes exist:

```sql
-- Check index usage
EXPLAIN QUERY PLAN
SELECT * FROM credential_access_log
WHERE user_id = 'user-123'
ORDER BY timestamp DESC
LIMIT 20;
```

### Query Optimization

For large audit log tables:

```sql
-- Add covering index for common query pattern
CREATE INDEX IF NOT EXISTS idx_cred_log_user_time_action
ON credential_access_log(user_id, timestamp DESC, action, success);
```

### Connection Pool Settings

No special settings required for OEK operations. Standard connection pooling applies.

---

## Appendix: Quick Reference

### Table Summary

| Table | Purpose | Records per Org |
|-------|---------|-----------------|
| `organization_envelope_keys` | OEK encrypted per member | N (members) |
| `shared_credentials` | Passwords encrypted with OEK | M (connections) |
| `credential_access_log` | Audit trail | Unbounded |

### Action Types in Audit Log

| Action | Description |
|--------|-------------|
| `decrypt` | Member decrypted a shared credential |
| `share` | Owner shared a connection with organization |
| `unshare` | Owner removed sharing from connection |
| `rotate` | OEK was rotated (re-encrypted) |
| `provision` | OEK provisioned for new member |
| `revoke` | OEK revoked for departing member |

### Key File Locations

- Migration: `pkg/storage/turso/migrations/009_shared_credentials.sql`
- Crypto: `pkg/crypto/org_envelope.go`
- OEK Store: `pkg/storage/turso/org_envelope_store.go`
- Shared Credentials Store: `pkg/storage/turso/shared_credential_store.go`
- Audit Store: `pkg/storage/turso/credential_audit_store.go`
- Service: `internal/connections/org_credential_service.go`

---

## Related Documentation

- [Credential Sharing Design](/Users/jacob_1/projects/howlerops/backend-go/docs/credential-sharing-design.md)
- [Implementation Checklist](/Users/jacob_1/projects/howlerops/backend-go/docs/credential-sharing-implementation-checklist.md)
- [Architecture Diagrams](/Users/jacob_1/projects/howlerops/backend-go/docs/credential-sharing-diagrams.md)
