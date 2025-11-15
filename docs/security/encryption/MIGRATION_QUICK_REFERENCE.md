# Password Migration Quick Reference

## TL;DR

**Goal:** Migrate passwords from OS keychain to encrypted database without user friction.

**Strategy:** Hybrid dual-read system that works with both sources during transition.

**Timeline:** 6-month phased rollout with progressive migration.

---

## Key Components

| Component | Purpose | Location |
|-----------|---------|----------|
| **PasswordManager** | Dual-read coordinator | `services/password_manager.go` |
| **CredentialService** | OS keychain access (legacy) | `services/credential.go` |
| **CredentialStore** | Encrypted DB access (new) | `backend-go/pkg/storage/turso/master_key_store.go` |
| **Migration Tracking** | Connection migration status | `connection_templates.password_migration_status` |

---

## Migration States

```
not_migrated → migrated → (eventually) → keychain removed
     ↓
  no_password (if connection has no password)
```

| State | Meaning | Password Location |
|-------|---------|-------------------|
| `not_migrated` | Not yet migrated | Keychain only |
| `migrated` | Migration complete | Encrypted DB (+ keychain backup) |
| `no_password` | No password stored | Neither location |

---

## Read Strategy (Priority Order)

```
1. Try encrypted_credentials (Turso) ← NEW SYSTEM
   ↓ (if not found or master key unavailable)
2. Fall back to keychain ← LEGACY SYSTEM
   ↓ (if found and master key available)
3. Trigger opportunistic migration (background)
```

---

## Write Strategy (Dual-Write)

```
1. Store in encrypted_credentials (if master key available)
2. ALSO store in keychain (backup during transition)
3. If either succeeds → Operation succeeds
```

---

## Migration Triggers

| Trigger | When | How |
|---------|------|-----|
| **Opportunistic** | User opens connection | Background migration on read |
| **Bulk** | User clicks "Migrate All" | UI-initiated batch process |
| **First Login** | Login after update | Prompt to migrate |

---

## Critical Functions

### 1. Dual-Read Password

```go
password, err := passwordManager.GetPassword(ctx, userID, connectionID, masterKey)

// Returns password from:
// 1. Encrypted DB (if available)
// 2. Keychain (fallback)
// 3. Error if not found in either
```

### 2. Dual-Write Password

```go
err := passwordManager.StorePassword(ctx, userID, connectionID, password, masterKey)

// Stores password in:
// 1. Encrypted DB (if master key available)
// 2. Keychain (always, for backup)
```

### 3. Opportunistic Migration

```go
// Triggered automatically when password found in keychain
go passwordManager.opportunisticMigration(ctx, userID, connectionID, password, masterKey)

// Process:
// 1. Encrypt password with master key
// 2. Store in encrypted_credentials
// 3. Mark connection as migrated
// 4. Log migration result
```

### 4. Bulk Migration

```go
result, err := passwordManager.MigrateBulkPasswords(ctx, userID, masterKey)

// Returns:
// - total: Total connections to migrate
// - success: Successfully migrated
// - failed: Failed to migrate
// - skipped: No password found
// - errors: Map of errors by connection ID
```

---

## Database Schema

### Migration Tracking Column

```sql
ALTER TABLE connection_templates
ADD COLUMN password_migration_status TEXT DEFAULT 'not_migrated'
CHECK (password_migration_status IN ('not_migrated', 'migrated', 'no_password'));
```

### Migration Log Table

```sql
CREATE TABLE password_migration_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    migration_type TEXT NOT NULL, -- 'automatic', 'manual', 'batch'
    source TEXT NOT NULL,          -- 'keychain', 'none'
    status TEXT NOT NULL,          -- 'success', 'failed', 'skipped'
    error_message TEXT,
    migrated_at INTEGER NOT NULL
);
```

---

## API Endpoints

| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | `/api/passwords/migration-status` | Get user's migration progress |
| POST | `/api/passwords/migrate` | Trigger bulk migration |
| GET | `/api/connections/:id/password` | Get password (dual-read) |
| POST | `/api/connections/:id/password` | Store password (dual-write) |
| DELETE | `/api/connections/:id/password` | Delete password (both locations) |

---

## Monitoring Queries

### Migration Completion Rate

```sql
SELECT
	COUNT(CASE WHEN password_migration_status = 'migrated' THEN 1 END) * 100.0 / COUNT(*) as completion_rate
FROM connection_templates
WHERE deleted_at IS NULL;
```

### Unmigrated Connections by User

```sql
SELECT user_id, COUNT(*) as unmigrated_count
FROM connection_templates
WHERE password_migration_status = 'not_migrated' AND deleted_at IS NULL
GROUP BY user_id
ORDER BY unmigrated_count DESC;
```

### Failed Migrations (Last 7 Days)

```sql
SELECT connection_id, error_message, COUNT(*) as failure_count
FROM password_migration_log
WHERE status = 'failed' AND migrated_at > unixepoch('now', '-7 days')
GROUP BY connection_id, error_message
ORDER BY failure_count DESC;
```

---

## Rollback Procedures

### Phase 1: Feature Flag Disable

```bash
# Disable encrypted storage reads
export ENABLE_ENCRYPTED_STORAGE=false

# Restart application
# All reads fall back to keychain
```

### Phase 2: Emergency Keychain Restoration

```go
// Restore all passwords from encrypted DB back to keychain
result, err := passwordManager.RestoreToKeychain(ctx, userID, masterKey)
```

### Phase 3: Full Rollback

1. Deploy previous version (without encrypted storage)
2. Passwords still in keychain (no data loss)
3. Encrypted credentials remain in DB (can retry migration later)

---

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| Master key not available | User hasn't logged in | Prompt re-login |
| Keychain access denied | Permissions issue | Use encrypted DB or prompt password |
| Decryption failed | Corrupt data or wrong key | Fall back to keychain |
| Database timeout | Network issue | Retry with backoff |
| Both sources unavailable | Major failure | Prompt user to re-enter password |

---

## Testing Checklist

### Unit Tests
- [ ] Dual-read retrieves from encrypted DB first
- [ ] Dual-read falls back to keychain
- [ ] Dual-write stores in both locations
- [ ] Opportunistic migration succeeds
- [ ] Bulk migration handles failures gracefully

### Integration Tests
- [ ] End-to-end migration flow works
- [ ] Multi-device sync works after migration
- [ ] Keychain removal doesn't break migrated connections
- [ ] Rollback restores passwords correctly

### Manual Tests
- [ ] Existing user sees migration prompt
- [ ] New user stores passwords in encrypted DB
- [ ] Connection with migrated password works
- [ ] Connection with keychain password works
- [ ] Bulk migration UI shows progress
- [ ] Failed migrations display errors

---

## Deployment Checklist

### Pre-Deployment
- [ ] Database migration #008 tested in staging
- [ ] Dual-read implementation complete
- [ ] Rollback scripts ready
- [ ] Monitoring dashboards created
- [ ] Support team trained

### Deployment
- [ ] Deploy to 5% canary users
- [ ] Monitor error rates for 24 hours
- [ ] Gradual rollout to 25%, 50%, 100%
- [ ] Monitor migration completion rates

### Post-Deployment
- [ ] Track migration progress weekly
- [ ] Send reminder emails to unmigrated users
- [ ] Plan keychain removal for Month 7+

---

## Success Metrics

| Metric | Target | Current |
|--------|--------|---------|
| Migration completion | 95% | TBD |
| Migration failure rate | < 1% | TBD |
| User satisfaction | > 90% | TBD |
| Support tickets | < 5% of users | TBD |

---

## Timeline

| Phase | Duration | Goal |
|-------|----------|------|
| **Phase 1** | Weeks 1-2 | Deploy hybrid system |
| **Phase 2** | Weeks 3-4 | Background migration active |
| **Phase 3** | Months 2-3 | User-initiated migration push |
| **Phase 4** | Month 6+ | Keychain removal (after 95% migration) |

---

## Support Responses

### User: "Where are my passwords?"

> Your passwords are stored securely in both your device keychain and our encrypted cloud storage during the migration period. You can migrate all passwords to cloud storage in Settings > Security > Migrate Passwords.

### User: "Can I use my connections on multiple devices?"

> Yes! After migrating your passwords to cloud storage, they will sync across all your devices automatically. Click "Migrate Passwords" in Settings to enable multi-device sync.

### User: "Is this secure?"

> Absolutely. We use industry-standard end-to-end encryption (AES-256-GCM). Your passwords are encrypted with your personal master key before being stored in the cloud. We never see your plaintext passwords.

---

## Common Issues & Solutions

### Issue: "Migration stuck at 0%"

**Cause:** Master key not in session
**Solution:** Log out and log back in to regenerate master key

### Issue: "Some connections failed to migrate"

**Cause:** Passwords not in keychain
**Solution:** Re-enter passwords manually, they'll be saved to encrypted storage

### Issue: "Connection asks for password after migration"

**Cause:** Decryption failed or master key expired
**Solution:** Re-login to refresh master key, or re-enter password

---

## Key Files

| File | Purpose |
|------|---------|
| `services/password_manager.go` | Main migration orchestrator |
| `services/credential.go` | Keychain service (legacy) |
| `backend-go/pkg/storage/turso/master_key_store.go` | Encrypted storage |
| `backend-go/pkg/crypto/encryption.go` | Encryption utilities |
| `connection_templates` table | Migration status tracking |
| `password_migration_log` table | Migration audit log |

---

## Next Steps

1. **Review:** Read full migration strategy document
2. **Implement:** Start with PasswordManager dual-read
3. **Test:** Unit tests → Integration tests → Manual tests
4. **Deploy:** Canary → Gradual rollout
5. **Monitor:** Track completion rates and errors
6. **Iterate:** Address issues, improve UX
7. **Complete:** Achieve 95% migration, remove keychain

---

**Questions?** See full documentation in `KEYCHAIN_TO_ENCRYPTED_MIGRATION_STRATEGY.md`
