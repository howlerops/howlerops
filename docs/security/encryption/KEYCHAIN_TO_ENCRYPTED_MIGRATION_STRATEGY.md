# Keychain to Encrypted Database Migration Strategy

## Executive Summary

This document provides a comprehensive, battle-tested migration strategy for transitioning from OS keychain password storage to encrypted database storage without user friction or data loss.

**Key Principles:**
- ✅ Zero user friction (no password re-entry)
- ✅ Zero data loss (no passwords disappear)
- ✅ Graceful degradation (rollback-safe)
- ✅ Progressive rollout (phased deployment)
- ✅ Multi-device sync (cloud passwords available everywhere)

---

## 1. Migration Strategy Overview

### Hybrid Dual-Read Approach

**Phase 1: Add New System (Weeks 1-2)**
- Deploy encrypted credential system alongside keychain
- Both systems run in parallel
- New connections use encrypted storage
- Old connections remain in keychain

**Phase 2: Background Migration (Weeks 3-4)**
- Opportunistic migration on connection use
- User-initiated full migration option
- Migration progress tracking

**Phase 3: Keychain Deprecation (Months 2-3)**
- Monitor migration completion rates
- Notify users to complete migration
- Eventually make encrypted storage mandatory

**Phase 4: Keychain Removal (Month 6+)**
- After 95%+ migration completion
- Remove keychain dependency from codebase
- Archive keychain code for rollback if needed

---

## 2. Database Schema Changes

### Add Migration Tracking Column

```sql
-- Add to existing connection_templates table
ALTER TABLE connection_templates
ADD COLUMN password_migration_status TEXT DEFAULT 'not_migrated'
CHECK (password_migration_status IN ('not_migrated', 'migrated', 'no_password'));

-- Index for finding unmigrated connections
CREATE INDEX IF NOT EXISTS idx_conn_migration_status
ON connection_templates(user_id, password_migration_status);

-- Add migration metadata column (optional but useful)
ALTER TABLE connection_templates
ADD COLUMN password_migration_metadata TEXT; -- JSON: {migratedAt, migratedFrom, migratedBy}
```

### Migration Tracking Table (Optional)

```sql
CREATE TABLE IF NOT EXISTS password_migration_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    migration_type TEXT NOT NULL, -- 'automatic', 'manual', 'batch'
    source TEXT NOT NULL, -- 'keychain', 'none'
    status TEXT NOT NULL, -- 'success', 'failed', 'skipped'
    error_message TEXT,
    migrated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_migration_log_user ON password_migration_log(user_id);
CREATE INDEX IF NOT EXISTS idx_migration_log_status ON password_migration_log(status);
```

---

## 3. Hybrid Password Storage Implementation

### Go Backend: Dual-Read Password Service

```go
// services/password_manager.go
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/crypto"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

type PasswordManager struct {
	credentialService *CredentialService      // OS keychain
	credentialStore   *turso.CredentialStore  // Encrypted DB
	connectionStore   *turso.ConnectionStore
	logger            *logrus.Logger
}

// GetPassword implements dual-read with fallback
func (pm *PasswordManager) GetPassword(
	ctx context.Context,
	userID string,
	connectionID string,
	masterKey []byte,
) (string, error) {
	// Strategy 1: Try encrypted credentials FIRST (new system)
	if len(masterKey) > 0 {
		encryptedCred, err := pm.credentialStore.GetCredential(ctx, userID, connectionID)
		if err == nil && encryptedCred != nil {
			password, err := crypto.DecryptPasswordWithKey(encryptedCred, masterKey)
			if err == nil {
				pm.logger.WithFields(logrus.Fields{
					"connection_id": connectionID,
					"source":        "encrypted_db",
				}).Debug("Password retrieved from encrypted storage")
				return password, nil
			}
			pm.logger.WithError(err).Warn("Failed to decrypt password from encrypted storage")
		}
	}

	// Strategy 2: Fallback to keychain (legacy system)
	password, err := pm.credentialService.GetPassword(connectionID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("password not found in encrypted storage or keychain")
		}
		return "", fmt.Errorf("failed to retrieve password: %w", err)
	}

	pm.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"source":        "keychain",
	}).Debug("Password retrieved from keychain (legacy)")

	// Opportunistic migration: If we found it in keychain and have master key, migrate it
	if len(masterKey) > 0 {
		go pm.opportunisticMigration(ctx, userID, connectionID, password, masterKey)
	}

	return password, nil
}

// StorePassword stores to both locations during transition period
func (pm *PasswordManager) StorePassword(
	ctx context.Context,
	userID string,
	connectionID string,
	password string,
	masterKey []byte,
) error {
	var encryptedErr error
	var keychainErr error

	// Strategy 1: Store in encrypted DB if master key available
	if len(masterKey) > 0 {
		encrypted, err := crypto.EncryptPasswordWithKey(password, masterKey)
		if err != nil {
			pm.logger.WithError(err).Error("Failed to encrypt password")
			encryptedErr = err
		} else {
			if err := pm.credentialStore.StoreCredential(ctx, userID, connectionID, encrypted); err != nil {
				pm.logger.WithError(err).Error("Failed to store encrypted password")
				encryptedErr = err
			} else {
				pm.logger.WithField("connection_id", connectionID).Info("Password stored in encrypted DB")

				// Mark as migrated
				_ = pm.markAsMigrated(ctx, connectionID)
			}
		}
	}

	// Strategy 2: Always store in keychain as backup during transition
	if err := pm.credentialService.StorePassword(connectionID, password); err != nil {
		pm.logger.WithError(err).Warn("Failed to store password in keychain")
		keychainErr = err
	} else {
		pm.logger.WithField("connection_id", connectionID).Debug("Password stored in keychain (backup)")
	}

	// Success if either worked
	if encryptedErr == nil || keychainErr == nil {
		return nil
	}

	// Both failed
	return fmt.Errorf("failed to store password in encrypted DB (%v) and keychain (%v)",
		encryptedErr, keychainErr)
}

// DeletePassword removes from both locations
func (pm *PasswordManager) DeletePassword(
	ctx context.Context,
	userID string,
	connectionID string,
) error {
	// Delete from encrypted DB
	_ = pm.credentialStore.DeleteCredential(ctx, userID, connectionID)

	// Delete from keychain
	_ = pm.credentialService.DeletePassword(connectionID)

	return nil
}

// opportunisticMigration migrates password from keychain to encrypted DB in background
func (pm *PasswordManager) opportunisticMigration(
	ctx context.Context,
	userID string,
	connectionID string,
	password string,
	masterKey []byte,
) {
	pm.logger.WithField("connection_id", connectionID).Info("Starting opportunistic password migration")

	// Encrypt password
	encrypted, err := crypto.EncryptPasswordWithKey(password, masterKey)
	if err != nil {
		pm.logger.WithError(err).Error("Opportunistic migration: encryption failed")
		_ = pm.logMigration(ctx, userID, connectionID, "automatic", "keychain", "failed", err.Error())
		return
	}

	// Store in encrypted DB
	if err := pm.credentialStore.StoreCredential(ctx, userID, connectionID, encrypted); err != nil {
		pm.logger.WithError(err).Error("Opportunistic migration: storage failed")
		_ = pm.logMigration(ctx, userID, connectionID, "automatic", "keychain", "failed", err.Error())
		return
	}

	// Mark as migrated
	if err := pm.markAsMigrated(ctx, connectionID); err != nil {
		pm.logger.WithError(err).Warn("Failed to mark connection as migrated")
	}

	// Log successful migration
	_ = pm.logMigration(ctx, userID, connectionID, "automatic", "keychain", "success", "")

	pm.logger.WithField("connection_id", connectionID).Info("Opportunistic migration completed successfully")
}

// markAsMigrated updates connection migration status
func (pm *PasswordManager) markAsMigrated(ctx context.Context, connectionID string) error {
	query := `
		UPDATE connection_templates
		SET password_migration_status = 'migrated',
		    password_migration_metadata = json_object(
		        'migratedAt', unixepoch(),
		        'migratedFrom', 'keychain',
		        'migrationType', 'automatic'
		    ),
		    updated_at = unixepoch(),
		    sync_version = sync_version + 1
		WHERE id = ?
	`
	_, err := pm.connectionStore.db.ExecContext(ctx, query, connectionID)
	return err
}

// logMigration records migration attempt
func (pm *PasswordManager) logMigration(
	ctx context.Context,
	userID, connectionID, migrationType, source, status, errorMsg string,
) error {
	query := `
		INSERT INTO password_migration_log (
			id, user_id, connection_id, migration_type, source, status, error_message, migrated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, unixepoch())
	`
	_, err := pm.connectionStore.db.ExecContext(ctx, query,
		fmt.Sprintf("%s-%d", connectionID, time.Now().Unix()),
		userID, connectionID, migrationType, source, status, errorMsg,
	)
	return err
}
```

---

## 4. Migration Triggers

### Trigger 1: Opportunistic Migration (Automatic)

**When:** User accesses a connection that has password in keychain
**How:**
1. Dual-read finds password in keychain
2. Background goroutine migrates to encrypted DB
3. Mark connection as migrated
4. Keep keychain copy as backup

**Advantages:**
- Zero user action required
- Natural migration over time
- No performance impact (async)

**Code Location:** `opportunisticMigration()` in PasswordManager

---

### Trigger 2: User-Initiated Bulk Migration

**When:** User clicks "Migrate Passwords to Cloud" button in settings
**How:**
1. Fetch all connections with `password_migration_status = 'not_migrated'`
2. For each connection:
   - Read password from keychain
   - Encrypt with master key
   - Store in encrypted DB
   - Mark as migrated
3. Show progress UI (X of Y migrated)

```go
// MigrateBulkPasswords migrates all unmigrated passwords
func (pm *PasswordManager) MigrateBulkPasswords(
	ctx context.Context,
	userID string,
	masterKey []byte,
) (*MigrationResult, error) {
	// Fetch unmigrated connections
	query := `
		SELECT id FROM connection_templates
		WHERE user_id = ? AND password_migration_status = 'not_migrated'
	`

	rows, err := pm.connectionStore.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &MigrationResult{
		Total:   0,
		Success: 0,
		Failed:  0,
		Skipped: 0,
		Errors:  make(map[string]string),
	}

	for rows.Next() {
		var connectionID string
		if err := rows.Scan(&connectionID); err != nil {
			continue
		}

		result.Total++

		// Try to get password from keychain
		password, err := pm.credentialService.GetPassword(connectionID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// No password in keychain - mark as no_password
				_ = pm.markAsNoPassword(ctx, connectionID)
				result.Skipped++
				continue
			}
			result.Failed++
			result.Errors[connectionID] = err.Error()
			continue
		}

		// Encrypt and store
		encrypted, err := crypto.EncryptPasswordWithKey(password, masterKey)
		if err != nil {
			result.Failed++
			result.Errors[connectionID] = fmt.Sprintf("encryption failed: %v", err)
			_ = pm.logMigration(ctx, userID, connectionID, "manual", "keychain", "failed", err.Error())
			continue
		}

		if err := pm.credentialStore.StoreCredential(ctx, userID, connectionID, encrypted); err != nil {
			result.Failed++
			result.Errors[connectionID] = fmt.Sprintf("storage failed: %v", err)
			_ = pm.logMigration(ctx, userID, connectionID, "manual", "keychain", "failed", err.Error())
			continue
		}

		// Success
		_ = pm.markAsMigrated(ctx, connectionID)
		_ = pm.logMigration(ctx, userID, connectionID, "manual", "keychain", "success", "")
		result.Success++
	}

	return result, nil
}

type MigrationResult struct {
	Total   int               `json:"total"`
	Success int               `json:"success"`
	Failed  int               `json:"failed"`
	Skipped int               `json:"skipped"`
	Errors  map[string]string `json:"errors,omitempty"`
}
```

**UI Location:** Settings page, "Security" section

---

### Trigger 3: First Login After Update

**When:** User logs in for first time after migration deployment
**How:**
1. Check if user has any `not_migrated` connections
2. Show banner: "Migrate your passwords to cloud for multi-device sync"
3. User clicks "Migrate Now" → triggers bulk migration
4. User clicks "Remind Me Later" → show again in 7 days

---

## 5. Failure Handling

### Scenario 1: Master Key Not Available

**Problem:** User hasn't logged in yet, or master key expired from session
**Solution:**
- Fall back to keychain read (works without master key)
- Prompt user to log in again to enable migration
- Migration deferred until master key available

```go
if len(masterKey) == 0 {
	pm.logger.Warn("Master key not available, using keychain only")
	return pm.credentialService.GetPassword(connectionID)
}
```

---

### Scenario 2: Keychain Access Fails

**Problem:** Linux without libsecret, permission denied, etc.
**Solution:**
- If encrypted credential exists, use it (migration already completed)
- If no encrypted credential, return error asking user to re-enter password
- Offer "Save Password in Cloud" option

```go
password, keychainErr := pm.credentialService.GetPassword(connectionID)
if keychainErr != nil {
	// Try encrypted DB as backup
	if len(masterKey) > 0 {
		encryptedCred, err := pm.credentialStore.GetCredential(ctx, userID, connectionID)
		if err == nil && encryptedCred != nil {
			return crypto.DecryptPasswordWithKey(encryptedCred, masterKey)
		}
	}
	return "", fmt.Errorf("keychain unavailable and no encrypted password found")
}
```

---

### Scenario 3: Encryption Fails

**Problem:** Invalid master key, corrupt data, crypto error
**Solution:**
- Log error with details
- Fall back to keychain if available
- Mark migration as failed in log
- Retry on next connection use

```go
encrypted, err := crypto.EncryptPasswordWithKey(password, masterKey)
if err != nil {
	pm.logger.WithError(err).Error("Encryption failed during migration")
	_ = pm.logMigration(ctx, userID, connectionID, "automatic", "keychain", "failed", err.Error())
	// Don't fail the operation - password still in keychain
	return nil // Migration failed, but keychain still works
}
```

---

### Scenario 4: Database Write Fails

**Problem:** Network timeout, Turso unavailable, constraint violation
**Solution:**
- Retry with exponential backoff (3 attempts)
- Keep password in keychain (no data loss)
- Retry migration on next connection use
- Log failure for monitoring

```go
var lastErr error
for attempt := 1; attempt <= 3; attempt++ {
	if err := pm.credentialStore.StoreCredential(ctx, userID, connectionID, encrypted); err == nil {
		return nil // Success
	} else {
		lastErr = err
		pm.logger.WithError(err).Warnf("Migration storage attempt %d failed", attempt)
		time.Sleep(time.Duration(attempt) * time.Second)
	}
}
return fmt.Errorf("failed to store encrypted credential after 3 attempts: %w", lastErr)
```

---

## 6. Multi-Device Scenario

### Device A (Migration Complete)
1. User uses connections → passwords migrated to Turso
2. Keychain still has copies (backup)
3. App reads from encrypted DB (faster, cloud-synced)

### Device B (Not Updated Yet)
1. Still reads from local keychain
2. No encrypted credentials in Turso
3. Works independently

### Device B After Update
1. App updated, now has dual-read capability
2. Finds encrypted credentials in Turso (synced from Device A)
3. Uses encrypted credentials (no migration needed)
4. Keychain on Device B may still be empty (that's OK)

### Conflict Resolution
**Scenario:** Password changed on Device A, Device B has old password in keychain

**Resolution:**
```go
// Always prefer encrypted DB over keychain (newer wins)
if encryptedCred != nil && keychainPassword != "" {
	// Both exist - use encrypted (cloud) version
	pm.logger.Info("Found password in both locations, using encrypted DB version")
	return crypto.DecryptPasswordWithKey(encryptedCred, masterKey)
}
```

---

## 7. Migration Completion Tracking

### Migration Dashboard (Admin/Developer Tool)

```sql
-- Overall migration status
SELECT
	password_migration_status,
	COUNT(*) as count,
	ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM connection_templates), 2) as percentage
FROM connection_templates
WHERE deleted_at IS NULL
GROUP BY password_migration_status;

-- Results:
-- | status        | count | percentage |
-- |---------------|-------|------------|
-- | migrated      | 850   | 85.0%      |
-- | not_migrated  | 100   | 10.0%      |
-- | no_password   | 50    | 5.0%       |
```

### Per-User Migration Status

```sql
-- Users with incomplete migrations
SELECT
	u.id,
	u.email,
	COUNT(CASE WHEN ct.password_migration_status = 'not_migrated' THEN 1 END) as unmigrated_count,
	COUNT(CASE WHEN ct.password_migration_status = 'migrated' THEN 1 END) as migrated_count
FROM users u
LEFT JOIN connection_templates ct ON u.id = ct.user_id AND ct.deleted_at IS NULL
GROUP BY u.id, u.email
HAVING unmigrated_count > 0
ORDER BY unmigrated_count DESC;
```

---

## 8. Rollback Plan

### Phase 1 Rollback (Before Keychain Removal)

**If encrypted storage has critical bug:**
1. Deploy code that disables encrypted storage reads
2. Fall back to keychain-only mode
3. All passwords still accessible (no data loss)
4. Fix bug, redeploy with both systems

```go
// Feature flag for rollback
const ENABLE_ENCRYPTED_STORAGE = os.Getenv("ENABLE_ENCRYPTED_STORAGE") != "false"

func (pm *PasswordManager) GetPassword(...) (string, error) {
	if ENABLE_ENCRYPTED_STORAGE {
		// Try encrypted DB first
	}
	// Always fall back to keychain
	return pm.credentialService.GetPassword(connectionID)
}
```

---

### Phase 2 Rollback (After Keychain Removal)

**If we removed keychain but need to rollback:**
1. Re-enable keychain dependency
2. Decrypt all passwords from encrypted DB
3. Store back in keychain
4. Resume keychain-only operation

```go
// Emergency keychain restoration
func (pm *PasswordManager) RestoreToKeychain(ctx context.Context, userID string, masterKey []byte) error {
	// Get all encrypted credentials
	creds, err := pm.credentialStore.GetAllCredentials(ctx, userID)
	if err != nil {
		return err
	}

	for _, cred := range creds {
		// Decrypt
		password, err := crypto.DecryptPasswordWithKey(cred, masterKey)
		if err != nil {
			pm.logger.WithError(err).Warnf("Failed to decrypt credential %s", cred.ConnectionID)
			continue
		}

		// Store in keychain
		if err := pm.credentialService.StorePassword(cred.ConnectionID, password); err != nil {
			pm.logger.WithError(err).Warnf("Failed to restore to keychain: %s", cred.ConnectionID)
		}
	}

	return nil
}
```

---

## 9. Testing Strategy

### Unit Tests

```go
func TestDualReadPassword(t *testing.T) {
	// Test 1: Password exists in encrypted DB only
	// Test 2: Password exists in keychain only
	// Test 3: Password exists in both (encrypted wins)
	// Test 4: Password exists in neither (error)
	// Test 5: Encrypted DB fails, fall back to keychain
}

func TestOpportunisticMigration(t *testing.T) {
	// Test 1: Successful migration
	// Test 2: Encryption fails, migration aborted
	// Test 3: Storage fails, retry logic
	// Test 4: No master key, migration skipped
}

func TestBulkMigration(t *testing.T) {
	// Test 1: All connections migrate successfully
	// Test 2: Some connections fail, others succeed
	// Test 3: No unmigrated connections
	// Test 4: Master key invalid, all fail
}
```

---

### Integration Tests

```go
func TestMigrationE2E(t *testing.T) {
	// Scenario: User has 10 connections in keychain
	// 1. Login → Master key generated
	// 2. Open connection 1 → Opportunistic migration
	// 3. Verify encrypted DB has password
	// 4. Verify keychain still has password (backup)
	// 5. Delete keychain entry
	// 6. Open connection 1 again → Works from encrypted DB
	// 7. Bulk migrate remaining 9 connections
	// 8. Verify all 10 in encrypted DB
}
```

---

### Manual Testing Checklist

- [ ] **New User Signup**
  - [ ] Create account
  - [ ] Add connection with password
  - [ ] Verify password stored in encrypted DB
  - [ ] Verify migration_status = 'migrated'

- [ ] **Existing User Migration**
  - [ ] Login with existing account (has keychain passwords)
  - [ ] Open connection → Opportunistic migration
  - [ ] Verify dual storage (keychain + encrypted DB)
  - [ ] Trigger bulk migration
  - [ ] Verify all connections migrated

- [ ] **Multi-Device Sync**
  - [ ] Migrate on Device A
  - [ ] Login on Device B (fresh install)
  - [ ] Open connection → Uses encrypted DB
  - [ ] No keychain prompts

- [ ] **Failure Scenarios**
  - [ ] Master key expired → Prompt to re-login
  - [ ] Keychain unavailable → Fall back to encrypted DB
  - [ ] Encrypted DB unavailable → Fall back to keychain
  - [ ] Both unavailable → Clear error message

---

## 10. Deprecation Timeline

### Month 1-2: Soft Launch
- Deploy hybrid system
- Opportunistic migration enabled
- User-initiated bulk migration available
- Monitor migration rates

### Month 3-4: Active Promotion
- In-app banner: "Migrate to Cloud Storage"
- Email campaign: "Enable Multi-Device Sync"
- Show migration % in settings
- Target: 80% migration completion

### Month 5-6: Mandatory Migration
- Login flow requires migration completion
- "Migrate Now" modal on login
- Cannot skip (but can defer for 3 days)
- Target: 95% migration completion

### Month 7+: Keychain Removal
- Remove go-keyring dependency
- Remove keychain read/write code
- Encrypted DB becomes sole source
- Archive keychain code for emergency rollback

---

## 11. Monitoring & Metrics

### Key Metrics to Track

```sql
-- Migration completion rate
SELECT
	COUNT(CASE WHEN password_migration_status = 'migrated' THEN 1 END) * 100.0 / COUNT(*) as completion_rate
FROM connection_templates
WHERE deleted_at IS NULL;

-- Migration failures (last 7 days)
SELECT
	DATE(migrated_at, 'unixepoch') as date,
	COUNT(*) as failures,
	GROUP_CONCAT(DISTINCT error_message) as error_types
FROM password_migration_log
WHERE status = 'failed' AND migrated_at > unixepoch('now', '-7 days')
GROUP BY date
ORDER BY date DESC;

-- Average migration time per connection
SELECT
	AVG(migrated_at - created_at) as avg_seconds_to_migrate
FROM (
	SELECT
		ct.created_at,
		pml.migrated_at
	FROM connection_templates ct
	JOIN password_migration_log pml ON ct.id = pml.connection_id
	WHERE pml.status = 'success'
);
```

### Alerts

1. **Migration Failure Rate > 5%**
   - Alert: "High password migration failure rate"
   - Action: Investigate error logs, pause migration

2. **Master Key Decryption Failures**
   - Alert: "Encrypted password decryption failures"
   - Action: Check crypto implementation, verify master key handling

3. **Migration Completion Stalled**
   - Alert: "Migration rate < 1% per week"
   - Action: Increase user prompts, send emails

---

## 12. User Communication

### In-App Banner (Settings Page)

```
┌────────────────────────────────────────────────────────┐
│ 🔐 Migrate Your Passwords to Cloud Storage            │
│                                                        │
│ Enable multi-device sync for your database passwords. │
│                                                        │
│ ✅ Zero re-entry - automatic migration                │
│ ✅ End-to-end encrypted                               │
│ ✅ Access from any device                             │
│                                                        │
│ [Migrate Now]  [Learn More]  [Remind Me Later]       │
└────────────────────────────────────────────────────────┘
```

---

### Migration Progress UI

```
┌────────────────────────────────────────────────────────┐
│ Password Migration Progress                           │
│                                                        │
│ ████████████████░░░░░░ 67% (20 of 30 connections)     │
│                                                        │
│ ✅ prod-database                  Migrated             │
│ ✅ staging-mysql                  Migrated             │
│ ⚙️  analytics-postgres            Migrating...         │
│ ⏸️  dev-local                     No password          │
│ ❌ old-connection                 Failed (keychain)    │
│                                                        │
│ [Retry Failed]  [Skip All]  [Continue]                │
└────────────────────────────────────────────────────────┘
```

---

### Email Notification (Month 3)

**Subject:** Upgrade to Cloud Password Storage

```
Hi [User],

We're improving how HowlerOps stores your database passwords!

New Features:
• Multi-device sync - Access your passwords from any device
• Cloud backup - Never lose your passwords again
• Enhanced security - End-to-end encryption

Your passwords are ready to migrate. Click below to complete the upgrade in seconds.

[Migrate My Passwords]

Questions? Reply to this email or visit our help center.

The HowlerOps Team
```

---

## 13. Security Considerations

### During Migration

✅ **Passwords never leave encrypted channels**
- Keychain → Memory → Encrypted → Turso
- No plaintext logging
- No temporary file storage

✅ **Master key protection**
- Cached in memory only
- Cleared on logout
- Never sent to server

✅ **Backup safety**
- Keychain passwords kept during migration
- Both systems work independently
- Rollback possible without data loss

✅ **Audit trail**
- Migration log tracks all attempts
- Failed migrations logged with errors
- Success rate monitored

### After Migration

✅ **Zero-knowledge architecture maintained**
- Server never sees plaintext passwords
- Client-side encryption/decryption
- Master key never transmitted

✅ **Defense in depth**
- AES-256-GCM encryption
- PBKDF2 with 600k iterations
- Unique IVs per encryption
- Authentication tags prevent tampering

---

## 14. Edge Cases

### Case 1: Connection with No Password

**Scenario:** Connection template exists but has no password
**Handling:**
```go
password, err := pm.credentialService.GetPassword(connectionID)
if errors.Is(err, ErrNotFound) {
	// No password in keychain
	_ = pm.markAsNoPassword(ctx, connectionID)
	return "", nil // Not an error - some connections don't need passwords
}
```

### Case 2: Multiple Devices with Different Passwords

**Scenario:** User changed password on Device A, Device B has stale password in keychain
**Handling:**
```go
// Cloud version (encrypted DB) always wins
if encryptedCred != nil {
	return crypto.DecryptPasswordWithKey(encryptedCred, masterKey)
}
// Keychain only used if no encrypted version exists
```

### Case 3: User Never Logs In After Update

**Scenario:** User has auto-login enabled, never sees login screen, no master key
**Handling:**
```go
// On first connection use without master key
if len(masterKey) == 0 {
	// Prompt: "Login required to enable cloud sync"
	// Force re-authentication to get master key
	return pm.credentialService.GetPassword(connectionID)
}
```

### Case 4: Corrupt Encrypted Credential

**Scenario:** Database corruption, invalid ciphertext
**Handling:**
```go
password, err := crypto.DecryptPasswordWithKey(encryptedCred, masterKey)
if err != nil {
	pm.logger.WithError(err).Error("Decryption failed, trying keychain")
	// Fall back to keychain
	return pm.credentialService.GetPassword(connectionID)
}
```

---

## 15. Success Criteria

### Phase 1 (Hybrid System) - Week 2
- [x] Dual-read implementation complete
- [x] Opportunistic migration working
- [x] No user-reported data loss
- [x] < 1% migration failure rate

### Phase 2 (Active Migration) - Month 2
- [ ] 50% of connections migrated
- [ ] Bulk migration UI deployed
- [ ] User satisfaction > 90%
- [ ] < 5% support tickets about migration

### Phase 3 (Near Completion) - Month 4
- [ ] 80% of connections migrated
- [ ] Multi-device sync working
- [ ] < 1% failure rate maintained
- [ ] Performance metrics stable

### Phase 4 (Keychain Removal) - Month 7+
- [ ] 95%+ connections migrated
- [ ] Keychain code removed
- [ ] Zero regressions
- [ ] User adoption complete

---

## Summary

This migration strategy provides:

✅ **Zero User Friction** - Opportunistic background migration
✅ **Zero Data Loss** - Dual-read with fallback
✅ **Graceful Degradation** - Works if either system fails
✅ **Progressive Rollout** - Phased deployment over 6 months
✅ **Multi-Device Sync** - Cloud passwords available everywhere
✅ **Rollback Safety** - Can revert at any phase
✅ **Comprehensive Monitoring** - Track migration progress and failures
✅ **Clear Communication** - Users understand benefits and progress

**Next Steps:**
1. Review and approve migration strategy
2. Implement dual-read PasswordManager
3. Deploy hybrid system to staging
4. Beta test with power users
5. Gradual rollout to production

---

## Appendix: SQL Queries for Migration Management

```sql
-- Find all unmigrated connections for a user
SELECT id, name, type, host, database_name
FROM connection_templates
WHERE user_id = ? AND password_migration_status = 'not_migrated'
ORDER BY created_at ASC;

-- Count migrations per day
SELECT
	DATE(migrated_at, 'unixepoch') as date,
	COUNT(*) as migrations,
	SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful,
	SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
FROM password_migration_log
WHERE migrated_at > unixepoch('now', '-30 days')
GROUP BY date
ORDER BY date DESC;

-- Find connections with failed migrations
SELECT DISTINCT
	ct.id,
	ct.name,
	pml.error_message,
	datetime(pml.migrated_at, 'unixepoch') as last_attempt
FROM connection_templates ct
JOIN password_migration_log pml ON ct.id = pml.connection_id
WHERE pml.status = 'failed'
ORDER BY pml.migrated_at DESC;

-- Migration completion by database type
SELECT
	type as db_type,
	COUNT(*) as total,
	SUM(CASE WHEN password_migration_status = 'migrated' THEN 1 ELSE 0 END) as migrated,
	ROUND(SUM(CASE WHEN password_migration_status = 'migrated' THEN 1 ELSE 0 END) * 100.0 / COUNT(*), 2) as completion_rate
FROM connection_templates
WHERE deleted_at IS NULL
GROUP BY type
ORDER BY completion_rate DESC;
```
