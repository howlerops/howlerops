# Password Migration Rollback Procedures

## Overview

This document provides detailed step-by-step procedures for rolling back the password migration at various stages of deployment. All procedures are designed to preserve data integrity and prevent password loss.

**Critical Principle:** Passwords are stored in BOTH keychain and encrypted DB during migration. Rollback is always safe because no data is deleted from keychain until Phase 4 (Month 6+).

---

## Rollback Levels

| Level | Scope | Data Loss Risk | Complexity | Downtime |
|-------|-------|----------------|------------|----------|
| **Level 1** | Feature flag disable | None | Low | None |
| **Level 2** | Code revert | None | Medium | < 5 minutes |
| **Level 3** | Database rollback | None | High | < 30 minutes |
| **Level 4** | Full restoration | None | Very High | < 1 hour |

---

## Level 1: Feature Flag Disable

**When to use:**
- Encrypted storage has bugs but keychain works fine
- Need to investigate issues without full rollback
- Want to pause migration without reverting code

**Impact:**
- ✅ No data loss
- ✅ No downtime
- ✅ Instant rollback
- ⚠️ Migration paused (can resume later)

### Procedure

#### Step 1: Disable Encrypted Storage Reads

```bash
# Set environment variable
export ENABLE_ENCRYPTED_STORAGE=false

# Or in .env file
echo "ENABLE_ENCRYPTED_STORAGE=false" >> .env

# Restart application
systemctl restart howlerops
# OR
kill -HUP $(pidof howlerops)
```

#### Step 2: Verify Rollback

```bash
# Check application logs
tail -f /var/log/howlerops.log | grep "ENABLE_ENCRYPTED_STORAGE"

# Should see:
# INFO: Encrypted storage disabled via feature flag
# INFO: Using keychain-only mode
```

#### Step 3: Monitor

```bash
# Verify connections still work
curl -X POST http://localhost:8080/api/connections/test-connection \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"connectionId": "test-conn-1"}'

# Should return success
```

#### Step 4: Re-enable (when ready)

```bash
# Remove flag
unset ENABLE_ENCRYPTED_STORAGE

# Restart application
systemctl restart howlerops
```

**Time to rollback:** < 1 minute
**Time to recovery:** Instant

---

## Level 2: Code Revert

**When to use:**
- Migration code has critical bugs
- Need to fully remove migration logic
- Feature flag not sufficient

**Impact:**
- ✅ No data loss (keychain still has passwords)
- ⚠️ Brief downtime (< 5 minutes)
- ⚠️ All users revert to keychain-only

### Procedure

#### Step 1: Identify Last Good Commit

```bash
# Find commit before migration was merged
git log --oneline --all | grep -B 5 "Add password migration"

# Example output:
# abc123 Fix: Connection timeout handling
# def456 Add password migration (dual-read system) ← PROBLEM COMMIT
# ghi789 Update database schema
```

#### Step 2: Create Rollback Branch

```bash
# Create rollback branch from last good commit
git checkout -b rollback/password-migration abc123

# Verify no migration code exists
grep -r "PasswordManager" services/
# Should return nothing
```

#### Step 3: Deploy Rollback

```bash
# Push rollback branch
git push origin rollback/password-migration

# Deploy to production
./deploy.sh rollback/password-migration

# OR trigger CI/CD pipeline
gh workflow run deploy.yml --ref rollback/password-migration
```

#### Step 4: Verify Deployment

```bash
# Check deployed version
curl http://localhost:8080/api/version

# Verify keychain still works
curl -X GET http://localhost:8080/api/connections/conn-1/password \
  -H "Authorization: Bearer $TOKEN"

# Should return password from keychain
```

#### Step 5: Communicate to Users

```
Subject: Temporary Password System Rollback

Hi,

We've temporarily reverted to our previous password storage system while we address a technical issue. Your database passwords are safe and all connections continue to work normally.

No action required on your part.

We'll notify you when the upgrade is available again.

Thanks,
HowlerOps Team
```

**Time to rollback:** 5-10 minutes
**Time to recovery:** When bug is fixed

---

## Level 3: Database Rollback

**When to use:**
- Database migration #008 caused issues
- Encrypted credentials table has corruption
- Need to remove migration tracking columns

**Impact:**
- ✅ No password data loss (keychain still has passwords)
- ⚠️ Lose migration tracking data (can rebuild)
- ⚠️ Lose encrypted credentials (can re-migrate)
- ⚠️ Moderate downtime (10-30 minutes)

### Procedure

#### Step 1: Backup Current Database

```bash
# Create full database backup
turso db backup howlerops-prod --output backup-$(date +%Y%m%d-%H%M%S).dump

# Verify backup
ls -lh backup-*.dump
# Should show recent backup file
```

#### Step 2: Prepare Rollback SQL

```sql
-- rollback_migration_008.sql

-- Step 1: Remove encrypted credentials
DROP TABLE IF EXISTS encrypted_credentials;

-- Step 2: Remove user master keys
DROP TABLE IF EXISTS user_master_keys;

-- Step 3: Remove migration log
DROP TABLE IF EXISTS password_migration_log;

-- Step 4: Remove migration tracking columns
ALTER TABLE connection_templates DROP COLUMN password_migration_status;
ALTER TABLE connection_templates DROP COLUMN password_migration_metadata;

-- Step 5: Remove migration record
DELETE FROM schema_migrations WHERE version = 8;

-- Verify
SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;
-- Should NOT show encrypted_credentials, user_master_keys, password_migration_log
```

#### Step 3: Execute Rollback

```bash
# Connect to Turso database
turso db shell howlerops-prod

# Execute rollback SQL
sqlite> .read rollback_migration_008.sql

# Verify tables removed
sqlite> SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;

# Exit
sqlite> .quit
```

#### Step 4: Revert Application Code

```bash
# Follow Level 2 procedure to revert code
git checkout -b rollback/migration-db abc123
git push origin rollback/migration-db
./deploy.sh rollback/migration-db
```

#### Step 5: Verify System

```bash
# Test connection creation
curl -X POST http://localhost:8080/api/connections \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "Test Connection",
    "type": "postgres",
    "host": "localhost",
    "port": 5432,
    "database": "testdb",
    "username": "user",
    "password": "pass123"
  }'

# Verify password stored in keychain
# Check Keychain Access.app (macOS) or Credential Manager (Windows)
```

#### Step 6: Monitor

```bash
# Check error logs
tail -f /var/log/howlerops.log | grep -i error

# Should see NO errors related to encrypted_credentials table
```

**Time to rollback:** 10-30 minutes
**Time to recovery:** When migration is fixed and re-deployed

---

## Level 4: Full Restoration (Emergency)

**When to use:**
- Complete system failure
- All password access methods failing
- Users cannot connect to databases
- Nuclear option - last resort only

**Impact:**
- ✅ No password data loss (restored from backup)
- ⚠️ Significant downtime (30-60 minutes)
- ⚠️ Lose all migration progress
- ⚠️ Require manual password re-entry for some users

### Procedure

#### Step 1: Assess Situation

```bash
# Check what's working
echo "Checking keychain..."
./scripts/test-keychain.sh

echo "Checking encrypted DB..."
./scripts/test-encrypted-db.sh

echo "Checking database connectivity..."
turso db shell howlerops-prod --execute "SELECT COUNT(*) FROM users;"
```

#### Step 2: Restore from Backup

```bash
# List available backups
turso db backup list howlerops-prod

# Choose most recent backup BEFORE migration
BACKUP_ID="backup-20250110-120000"

# Create new database from backup
turso db create howlerops-prod-restored --from-backup $BACKUP_ID

# Verify restored database
turso db shell howlerops-prod-restored --execute "
  SELECT name FROM sqlite_master WHERE type='table' ORDER BY name;
"

# Should NOT show migration tables
```

#### Step 3: Switch to Restored Database

```bash
# Update connection string in production
export TURSO_DATABASE_URL="libsql://howlerops-prod-restored-$ORG.turso.io"

# Update in environment config
cat > .env.production <<EOF
TURSO_DATABASE_URL=libsql://howlerops-prod-restored-$ORG.turso.io
TURSO_AUTH_TOKEN=$TURSO_AUTH_TOKEN
EOF
```

#### Step 4: Deploy Rollback Code

```bash
# Deploy last known good version
git checkout tags/v1.0.0-pre-migration
./deploy.sh production
```

#### Step 5: Verify All Systems

```bash
# Test user login
curl -X POST http://localhost:8080/api/auth/login \
  -d '{"email": "test@example.com", "password": "testpass"}'

# Test connection list
curl -X GET http://localhost:8080/api/connections \
  -H "Authorization: Bearer $TOKEN"

# Test password retrieval
curl -X GET http://localhost:8080/api/connections/conn-1/password \
  -H "Authorization: Bearer $TOKEN"
```

#### Step 6: Restore Keychain Passwords (If Needed)

If keychain was cleared and encrypted DB is gone, passwords need manual re-entry:

```bash
# Generate password reset script for support team
cat > restore-passwords.sql <<EOF
-- Find connections with no password
SELECT
	u.email,
	ct.id as connection_id,
	ct.name as connection_name,
	ct.type as db_type,
	ct.host
FROM connection_templates ct
JOIN users u ON ct.user_id = u.id
WHERE ct.deleted_at IS NULL
ORDER BY u.email, ct.name;
EOF

turso db shell howlerops-prod-restored < restore-passwords.sql > connections-needing-passwords.csv

# Send to support team
echo "Support team: Use connections-needing-passwords.csv to help users restore passwords"
```

#### Step 7: User Communication

```
Subject: Emergency Password System Restoration

Hi,

We've completed an emergency restoration of HowlerOps to ensure system stability. Your data is safe, but you may need to re-enter some database connection passwords.

What happened:
- We encountered a critical issue with the password migration
- We've restored from a backup to ensure data integrity
- All your connections are intact

Action required:
1. Log in to HowlerOps
2. Open each database connection
3. If prompted, re-enter your database password

We apologize for the inconvenience and have implemented additional safeguards to prevent this in the future.

If you need assistance, please contact support@howlerops.com

Thanks for your patience,
HowlerOps Team
```

**Time to rollback:** 30-60 minutes
**Time to recovery:** Immediate (with password re-entry)

---

## Post-Rollback Procedures

### For All Rollback Levels

#### 1. Incident Report

```markdown
## Incident Report: Password Migration Rollback

**Date:** 2025-01-15
**Rollback Level:** [1/2/3/4]
**Duration:** [X hours]
**Users Affected:** [X users]

### Timeline
- 09:00 AM: Migration deployed to production
- 10:30 AM: Issues detected (describe)
- 10:45 AM: Decision to rollback
- 11:00 AM: Rollback initiated
- 11:15 AM: Rollback complete
- 11:30 AM: Verification complete

### Root Cause
[Detailed analysis of what went wrong]

### Impact
- [ ] Data loss: None
- [ ] Downtime: X minutes
- [ ] User complaints: X tickets
- [ ] Financial impact: $X

### Resolution
[What was done to fix]

### Prevention
- [ ] Action 1: Add test for [specific scenario]
- [ ] Action 2: Improve monitoring for [specific metric]
- [ ] Action 3: Update documentation
```

#### 2. User Communication

```bash
# Send status update email
./scripts/send-email.sh \
  --template rollback-complete \
  --subject "System Restored: All Services Normal" \
  --to all-users

# Post to status page
curl -X POST https://status.howlerops.com/api/incidents \
  -d '{
    "name": "Password Migration Rollback",
    "status": "resolved",
    "message": "System has been restored to full functionality"
  }'
```

#### 3. Code Review

```bash
# Create post-mortem branch
git checkout -b post-mortem/password-migration

# Document lessons learned
cat > docs/post-mortem/password-migration-rollback.md <<EOF
# Password Migration Rollback Post-Mortem

## What Went Wrong
[Analysis]

## What Went Right
[What worked well]

## Action Items
- [ ] Fix: [specific bug]
- [ ] Test: [specific scenario]
- [ ] Monitor: [specific metric]
EOF

# Commit and share
git add docs/post-mortem/
git commit -m "Post-mortem: Password migration rollback"
git push origin post-mortem/password-migration
```

#### 4. Re-Deployment Plan

```markdown
## Re-Deployment Checklist

### Before Re-Attempt
- [ ] All bugs from rollback fixed
- [ ] Additional tests added
- [ ] Monitoring dashboards created
- [ ] Rollback procedures tested
- [ ] Team trained on rollback procedures

### Canary Deployment
- [ ] Deploy to 1% of users
- [ ] Monitor for 48 hours
- [ ] Check error rates
- [ ] Check migration success rates
- [ ] Get user feedback

### Gradual Rollout
- [ ] 5% → Monitor 24h
- [ ] 25% → Monitor 24h
- [ ] 50% → Monitor 24h
- [ ] 100% → Monitor 72h
```

---

## Testing Rollback Procedures

### Rollback Simulation (Staging)

```bash
#!/bin/bash
# test-rollback.sh

set -e

echo "=== Password Migration Rollback Test ==="

# Step 1: Deploy migration
echo "Deploying migration to staging..."
./deploy.sh staging migration-feature

# Step 2: Create test data
echo "Creating test connections..."
./scripts/create-test-connections.sh

# Step 3: Verify migration works
echo "Verifying migration..."
./scripts/verify-migration.sh

# Step 4: Simulate rollback
echo "Simulating rollback..."
export ENABLE_ENCRYPTED_STORAGE=false
systemctl restart howlerops-staging

# Step 5: Verify rollback
echo "Verifying rollback..."
./scripts/verify-keychain-only.sh

# Step 6: Re-enable
echo "Re-enabling migration..."
unset ENABLE_ENCRYPTED_STORAGE
systemctl restart howlerops-staging

# Step 7: Final verification
echo "Final verification..."
./scripts/verify-migration.sh

echo "=== Rollback test complete ==="
```

---

## Emergency Contacts

### Rollback Decision Tree

```
Issue detected
    ↓
Is it affecting users?
    ├─ No → Monitor, investigate
    └─ Yes → How severe?
        ├─ Minor → Level 1 (Feature flag)
        ├─ Moderate → Level 2 (Code revert)
        ├─ Severe → Level 3 (Database rollback)
        └─ Critical → Level 4 (Full restoration)
```

### On-Call Contacts

```
Primary: DevOps Lead - [phone]
Secondary: Backend Lead - [phone]
Escalation: CTO - [phone]

Turso Support: support@turso.tech
Database DBA: [contact]
```

---

## Rollback Metrics

### Success Criteria

After rollback, verify:
- [ ] User login works
- [ ] Connection list loads
- [ ] Passwords retrieved from keychain
- [ ] New connections can be created
- [ ] Existing connections work
- [ ] Error rate < 0.1%
- [ ] No user data loss
- [ ] Response time < 500ms

### Monitoring

```bash
# Check error rate
curl http://localhost:9090/api/v1/query?query=rate(http_errors_total[5m])

# Check response time
curl http://localhost:9090/api/v1/query?query=http_request_duration_seconds{quantile="0.95"}

# Check user activity
curl http://localhost:9090/api/v1/query?query=active_users
```

---

## Summary

| Level | Time | Risk | When to Use |
|-------|------|------|-------------|
| **1: Feature Flag** | < 1 min | None | Minor issues, investigation needed |
| **2: Code Revert** | 5-10 min | Low | Code bugs, need clean slate |
| **3: Database Rollback** | 10-30 min | Medium | Database corruption, schema issues |
| **4: Full Restoration** | 30-60 min | High | Complete failure, emergency only |

**Key Principle:** No data loss at any rollback level because keychain passwords are preserved during migration.

---

**Remember:** The best rollback is the one you never have to execute. Test thoroughly before deployment!
