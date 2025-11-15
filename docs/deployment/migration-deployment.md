# Migration Deployment - How It Works Automatically

## ✅ GREAT NEWS: Everything Is Already Set Up! 🎉

You **don't need to do anything** for migrations to run. The system is already fully automated.

---

## How Migrations Work Automatically

### 1. **On Every App Startup** ⚡
**File**: `backend-go/cmd/server/main.go`

```go
// Lines ~160-165
if schemaErr := turso.InitializeSchema(tursoClient, logger); schemaErr != nil {
    logger.WithError(schemaErr).Fatal("Failed to initialize Turso schema")
}

if migErr := turso.RunMigrations(tursoClient, logger); migErr != nil {
    logger.WithError(migErr).Fatal("Failed to run Turso migrations")
}
```

**What happens:**
1. App starts up
2. Connects to Turso database
3. Runs `InitializeSchema()` - Creates base tables if they don't exist
4. Runs `RunMigrations()` - Applies any pending migrations
5. **Migration #008 will be applied automatically** on next startup

---

### 2. **On Every Deployment** 🚀
**File**: `.github/workflows/deploy-cloud-run.yml`

**Current workflow:**
1. Builds Docker image (lines 176-194)
2. Deploys to Cloud Run (lines 235-290)
3. **Cloud Run starts container**
4. **App startup runs migrations** (see above)
5. Smoke tests verify health (lines 320-356)

**What this means:**
- Every push to `main` triggers deployment
- Every deployment restarts the app
- Every app restart runs pending migrations
- **Zero manual intervention needed**

---

### 3. **Migration Tracking** 📊

**Table**: `schema_migrations` (auto-created)
```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at INTEGER NOT NULL,
    checksum TEXT
);
```

**How it works:**
- Each migration has a version number (1, 2, 3... 8)
- When migration runs, it checks `schema_migrations` table
- Only runs migrations with version > current max version
- Records each migration in the table after success
- **Idempotent**: Running migrations multiple times is safe

---

## Migration #008 Status

### ✅ What's Complete

1. **Migration SQL defined** in `migrate.go`:
   ```go
   {
       Version:     8,
       Description: "Add encrypted password storage (zero-knowledge)",
       SQL:         getEncryptedPasswordStorageSQL(),
   }
   ```

2. **Migration function created**:
   - Creates `user_master_keys` table
   - Creates `encrypted_credentials` table
   - Adds indexes for performance

3. **Will run automatically** on next deployment

### 🔍 How to Verify

**Check migration status** (after deployment):
```bash
# Connect to Turso
turso db shell <your-db-name>

# Check migrations table
SELECT * FROM schema_migrations ORDER BY version;

# Should see:
# version | description                               | applied_at
# --------+------------------------------------------+-----------
# 1       | Initial schema                            | ...
# 2       | Phase 3 organization tables               | ...
# 3       | Add organization support                  | ...
# 8       | Add encrypted password storage           | <timestamp>

# Verify new tables exist
SELECT name FROM sqlite_master WHERE type='table'
  AND name IN ('user_master_keys', 'encrypted_credentials');
```

---

## Deployment Flow Diagram

```
┌─────────────────────────────────────────────────────────────┐
│  1. Push to main branch                                     │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  2. GitHub Actions: deploy-cloud-run.yml                    │
│     - Run tests                                             │
│     - Build Docker image                                    │
│     - Push to GCR                                           │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  3. Deploy to Cloud Run                                     │
│     - Pull Docker image                                     │
│     - Start new container                                   │
│     - Wait for health check                                 │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  4. Container Startup (cmd/server/main.go)                  │
│     a. Connect to Turso                                     │
│     b. Run InitializeSchema()                               │
│     c. Run RunMigrations()  ← MIGRATIONS RUN HERE!         │
│        - Check schema_migrations table                      │
│        - Find pending migrations (version > current)        │
│        - Execute migration #008 SQL                         │
│        - Record in schema_migrations                        │
│     d. Start HTTP server                                    │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│  5. Health Check (GitHub Actions)                           │
│     - curl /health endpoint                                 │
│     - Verify successful response                            │
│     - Deployment complete ✅                                │
└─────────────────────────────────────────────────────────────┘
```

---

## What Happens on Next Deployment

1. **You push code to main** (or merge PR)
2. **GitHub Actions automatically**:
   - Builds new Docker image
   - Deploys to Cloud Run
   - Cloud Run starts new container
3. **App startup automatically**:
   - Connects to Turso
   - Runs pending migrations
   - **Migration #008 executes** (creates tables)
   - Records version 8 in `schema_migrations`
   - Starts serving requests
4. **Done!** Tables are ready, app is running

---

## Safety Features

### ✅ Transaction Safety
Every migration runs in a transaction:
- All-or-nothing execution
- Rollback on error
- No partial migrations

### ✅ Idempotent Operations
Migration SQL uses `IF NOT EXISTS`:
```sql
CREATE TABLE IF NOT EXISTS user_master_keys ...
CREATE INDEX IF NOT EXISTS idx_encrypted_creds_user_id ...
```
- Safe to run multiple times
- Won't fail if tables exist
- Won't duplicate data

### ✅ Version Tracking
```go
if m.Version <= currentVersion {
    logger.Debug("Migration already applied, skipping")
    continue
}
```
- Only runs migrations once
- Skips already-applied versions
- Logs all migration activity

---

## Manual Migration Execution (Optional)

**If you want to run migrations manually** (not needed for normal deployment):

```bash
# Using the migrate CLI tool
cd backend-go
go run cmd/migrate/main.go --db-url="<turso-url>" --auth-token="<token>"

# Or using the verification script
cd backend-go
go run scripts/verify_migrations.go
```

---

## Troubleshooting

### Migration Failed During Deployment?

**Check Cloud Run logs:**
```bash
gcloud run services logs read howlerops-backend --region=us-central1
```

**Look for:**
```
INFO Starting migration runner
INFO Current database version: 3
INFO Applying migration: version=8
ERROR Migration 8 failed: <error message>
```

**Common issues:**
1. **Foreign key constraint**: Ensure `users` and `connection_templates` tables exist
2. **Syntax error**: Check SQL in `getEncryptedPasswordStorageSQL()`
3. **Connection timeout**: Turso might be temporarily unavailable

**Fix:**
- Migrations are safe to re-run
- Next deployment will retry failed migration
- Version isn't recorded until successful

---

## Summary

### ✅ Migrations Are Fully Automated

| Component | Status | Details |
|-----------|--------|---------|
| **Migration SQL** | ✅ Complete | Embedded in `migrate.go` |
| **Migration tracking** | ✅ Exists | `schema_migrations` table |
| **App startup integration** | ✅ Exists | `cmd/server/main.go` runs migrations |
| **CI/CD integration** | ✅ Exists | Cloud Run deployment triggers startup |
| **Error handling** | ✅ Exists | Transactions + logging |
| **Idempotency** | ✅ Exists | `IF NOT EXISTS` + version tracking |

### 🎯 Next Deployment Will:

1. Build and deploy new code
2. Start app container
3. **Automatically run migration #008**
4. Create `user_master_keys` table
5. Create `encrypted_credentials` table
6. Add indexes
7. Record migration as complete
8. Start serving requests

**You don't need to do ANYTHING** - it's fully automatic! 🚀

---

## Background Processes

The background bash processes you saw are just monitoring CI/CD runs and tests. They don't affect migrations and will complete on their own. You can safely ignore them or I can kill them to clean up.

---

## Files Modified

- `backend-go/pkg/storage/turso/migrate.go` - Added migration #008
- `backend-go/pkg/crypto/encryption.go` - Added PBKDF2 functions
- `frontend/src/lib/crypto/encryption.ts` - Added encryption utilities

**All migration infrastructure was already in place** - I just added the new migration to the existing system.
