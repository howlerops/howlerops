# Credential Sharing Architecture Design

## Executive Summary

This document outlines the solution to fix the critical credential sharing vulnerability where shared database connections cannot be decrypted by organization members because passwords remain encrypted with the original user's master key.

**Current State**: Broken - passwords encrypted with user A's key cannot be decrypted by user B
**Target State**: Working - organization members can decrypt shared connection passwords
**Approach**: Hybrid envelope encryption with per-member re-encryption

---

## Problem Analysis

### Current Architecture Issues

```
User A creates connection with password "db_pass123"
├─ User A's master key (encrypted with their login password)
│  └─ Encrypts connection password
├─ ShareConnection() called
│  └─ Only changes visibility flag
│  └─ Password STILL encrypted with User A's key
└─ User B tries to use connection
   └─ FAILS: User B doesn't have User A's master key
   └─ Cannot decrypt password
```

### Current Schema

```sql
-- User master keys: one per user
user_master_keys (
    user_id TEXT PRIMARY KEY,
    encrypted_master_key TEXT,  -- Encrypted with user's password-derived key
    key_iv, key_auth_tag, pbkdf2_salt
)

-- Credentials: one per user per connection
encrypted_credentials (
    user_id TEXT,
    connection_id TEXT,
    encrypted_password TEXT,  -- Problem: encrypted with user's master key
    password_iv, password_auth_tag,
    UNIQUE(user_id, connection_id)
)
```

**The Fundamental Flaw**: When `visibility='shared'`, there's only ONE encrypted_credentials record (for the creator), but MULTIPLE users need access.

---

## Solution: Hybrid Envelope Encryption with Per-Member Keys

### Design Principles

1. **Zero-Knowledge**: Server never sees plaintext passwords
2. **Scalability**: Support organizations with 100+ members
3. **Security**: Maintain isolation between personal and shared credentials
4. **Auditability**: Track all access to shared credentials
5. **Revocability**: Instantly revoke access when member leaves

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    ORGANIZATION LEVEL                        │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Organization Envelope Key (OEK)                             │
│  ├─ Generated once per organization                          │
│  ├─ Never stored in plaintext                                │
│  └─ Re-encrypted for each member using their master key      │
│                                                               │
│  organization_envelope_keys table                            │
│  ├─ org_id: "org-123"                                        │
│  ├─ user_id: "user-A"                                        │
│  ├─ encrypted_oek: "..." (OEK encrypted with user-A's MK)   │
│  ├─ user_id: "user-B"                                        │
│  └─ encrypted_oek: "..." (OEK encrypted with user-B's MK)   │
│                                                               │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                    CONNECTION LEVEL                          │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  Shared Connection Password                                  │
│  ├─ Encrypted with organization's OEK                        │
│  └─ Stored in shared_credentials table                       │
│                                                               │
│  shared_credentials table                                    │
│  ├─ connection_id: "conn-xyz"                                │
│  ├─ organization_id: "org-123"                               │
│  └─ encrypted_password: "..." (encrypted with OEK)           │
│                                                               │
└─────────────────────────────────────────────────────────────┘

Decryption Flow for User B:
1. Get User B's master key (decrypt with their password)
2. Get organization's OEK encrypted for User B
3. Decrypt OEK using User B's master key
4. Decrypt connection password using OEK
```

---

## Database Schema Changes

### New Tables

```sql
-- =====================================================================
-- Organization Envelope Keys
-- One organization key, encrypted once per member
-- =====================================================================
CREATE TABLE IF NOT EXISTS organization_envelope_keys (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,

    -- Organization's envelope key encrypted with user's master key
    encrypted_oek TEXT NOT NULL,      -- Base64-encoded ciphertext
    oek_iv TEXT NOT NULL,             -- Base64-encoded IV
    oek_auth_tag TEXT NOT NULL,       -- Base64-encoded GCM auth tag

    -- Key rotation support
    key_version INTEGER NOT NULL DEFAULT 1,

    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,

    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(organization_id, user_id, key_version)
);

CREATE INDEX idx_org_keys_org_id ON organization_envelope_keys(organization_id);
CREATE INDEX idx_org_keys_user_id ON organization_envelope_keys(user_id);
CREATE INDEX idx_org_keys_version ON organization_envelope_keys(organization_id, key_version);

-- =====================================================================
-- Shared Connection Credentials
-- Passwords for shared connections, encrypted with org envelope key
-- =====================================================================
CREATE TABLE IF NOT EXISTS shared_credentials (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,

    -- Password encrypted with organization envelope key
    encrypted_password TEXT NOT NULL,
    password_iv TEXT NOT NULL,
    password_auth_tag TEXT NOT NULL,

    -- Metadata
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,          -- User who shared the connection

    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(connection_id, organization_id)
);

CREATE INDEX idx_shared_creds_connection ON shared_credentials(connection_id);
CREATE INDEX idx_shared_creds_org ON shared_credentials(organization_id);
CREATE INDEX idx_shared_creds_created_by ON shared_credentials(created_by);

-- =====================================================================
-- Credential Access Audit Log
-- Track all access to shared credentials
-- =====================================================================
CREATE TABLE IF NOT EXISTS credential_access_log (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,              -- 'decrypt', 'share', 'unshare', 'rotate'
    timestamp INTEGER NOT NULL,
    ip_address TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,

    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX idx_cred_access_conn ON credential_access_log(connection_id);
CREATE INDEX idx_cred_access_user ON credential_access_log(user_id);
CREATE INDEX idx_cred_access_time ON credential_access_log(timestamp);
```

### Modified Tables

```sql
-- Keep existing tables but clarify their usage:

-- encrypted_credentials: NOW ONLY FOR PERSONAL CONNECTIONS
-- One record per user per personal connection
-- No changes needed

-- connection_templates: Add field to distinguish personal vs shared
ALTER TABLE connection_templates
ADD COLUMN encryption_type TEXT DEFAULT 'personal';
-- 'personal' = encrypted_credentials
-- 'shared' = shared_credentials
```

---

## Implementation Flow

### 1. Organization Creation / User Joins

```go
// When organization is created or user joins
func (s *Service) AddMemberToOrganization(ctx context.Context, orgID, userID string) error {
    // 1. Get or create organization envelope key (OEK)
    oek, err := s.getOrCreateOrgEnvelopeKey(ctx, orgID)
    if err != nil {
        return err
    }

    // 2. Get user's master key
    userMasterKey, err := s.getMasterKeyForUser(ctx, userID)
    if err != nil {
        return err
    }

    // 3. Encrypt OEK with user's master key
    encryptedOEK, err := crypto.EncryptAESGCM(oek, userMasterKey)
    if err != nil {
        return err
    }

    // 4. Store encrypted OEK for this user
    err = s.storeOrgKeyForUser(ctx, orgID, userID, encryptedOEK)
    if err != nil {
        return err
    }

    // 5. Add member to organization
    return s.repo.AddMember(ctx, &OrganizationMember{
        OrganizationID: orgID,
        UserID:         userID,
        Role:           RoleMember,
    })
}

func (s *Service) getOrCreateOrgEnvelopeKey(ctx context.Context, orgID string) ([]byte, error) {
    // Try to get existing OEK from any member
    members, err := s.repo.GetMembers(ctx, orgID)
    if err != nil {
        return nil, err
    }

    if len(members) > 0 {
        // Get OEK from first member (decrypt it temporarily)
        firstMemberID := members[0].UserID
        encryptedOEK, err := s.getOrgKeyForUser(ctx, orgID, firstMemberID)
        if err == nil {
            // OEK exists, decrypt it temporarily to re-encrypt for new member
            masterKey, err := s.getMasterKeyForUser(ctx, firstMemberID)
            if err != nil {
                return nil, err
            }
            return crypto.DecryptAESGCM(encryptedOEK, masterKey)
        }
    }

    // First member - generate new OEK
    return crypto.GenerateAESKey() // 256-bit random key
}
```

### 2. Share Connection

```go
func (s *Service) ShareConnection(ctx context.Context, connID, userID, orgID string) error {
    // 1. Verify permissions (existing code)
    // ...

    // 2. Get connection and verify ownership
    conn, err := s.store.GetByID(ctx, connID)
    if err != nil {
        return err
    }

    // 3. Get personal encrypted password for original user
    personalCred, err := s.credStore.GetCredential(ctx, userID, connID)
    if err != nil {
        return err
    }

    // 4. Decrypt password using user's master key
    userMasterKey, err := s.getMasterKeyForUser(ctx, userID)
    if err != nil {
        return err
    }

    plaintextPassword, err := crypto.DecryptPasswordWithMasterKey(
        personalCred,
        userMasterKey,
    )
    if err != nil {
        return err
    }

    // 5. Get organization envelope key for this user
    encryptedOEK, err := s.getOrgKeyForUser(ctx, orgID, userID)
    if err != nil {
        return err
    }

    oek, err := crypto.DecryptAESGCM(encryptedOEK, userMasterKey)
    if err != nil {
        return err
    }

    // 6. Re-encrypt password with org envelope key
    sharedCred, err := crypto.EncryptPasswordWithOrgKey(plaintextPassword, oek)
    if err != nil {
        return err
    }

    // 7. Store in shared_credentials table
    err = s.sharedCredStore.StoreSharedCredential(ctx, &SharedCredential{
        ConnectionID:     connID,
        OrganizationID:   orgID,
        EncryptedPassword: sharedCred.Ciphertext,
        PasswordIV:       sharedCred.IV,
        PasswordAuthTag:  sharedCred.AuthTag,
        CreatedBy:        userID,
    })
    if err != nil {
        return err
    }

    // 8. Update connection metadata
    conn.OrganizationID = &orgID
    conn.Visibility = "shared"
    conn.EncryptionType = "shared"

    err = s.store.Update(ctx, conn)
    if err != nil {
        return err
    }

    // 9. Audit log
    s.logCredentialAccess(ctx, connID, orgID, userID, "share", true, nil)

    return nil
}
```

### 3. Decrypt Shared Password (Any Member)

```go
func (s *Service) GetSharedConnectionPassword(ctx context.Context, connID, userID, orgID string) (string, error) {
    // 1. Verify membership and permissions
    member, err := s.repo.GetMember(ctx, orgID, userID)
    if err != nil {
        return "", fmt.Errorf("user not member of organization")
    }

    // 2. Get shared credential
    sharedCred, err := s.sharedCredStore.GetSharedCredential(ctx, connID, orgID)
    if err != nil {
        s.logCredentialAccess(ctx, connID, orgID, userID, "decrypt", false, err)
        return "", err
    }

    // 3. Get user's master key
    userMasterKey, err := s.getMasterKeyForUser(ctx, userID)
    if err != nil {
        s.logCredentialAccess(ctx, connID, orgID, userID, "decrypt", false, err)
        return "", err
    }

    // 4. Get org envelope key encrypted for this user
    encryptedOEK, err := s.getOrgKeyForUser(ctx, orgID, userID)
    if err != nil {
        s.logCredentialAccess(ctx, connID, orgID, userID, "decrypt", false, err)
        return "", err
    }

    // 5. Decrypt OEK with user's master key
    oek, err := crypto.DecryptAESGCM(encryptedOEK, userMasterKey)
    if err != nil {
        s.logCredentialAccess(ctx, connID, orgID, userID, "decrypt", false, err)
        return "", err
    }

    // 6. Decrypt password with OEK
    password, err := crypto.DecryptPasswordWithOrgKey(sharedCred, oek)
    if err != nil {
        s.logCredentialAccess(ctx, connID, orgID, userID, "decrypt", false, err)
        return "", err
    }

    // 7. Audit log
    s.logCredentialAccess(ctx, connID, orgID, userID, "decrypt", true, nil)

    return password, nil
}
```

### 4. Member Leaves Organization

```go
func (s *Service) RemoveMemberFromOrganization(ctx context.Context, orgID, userID string) error {
    // 1. Remove user's encrypted copy of org envelope key
    err := s.deleteOrgKeyForUser(ctx, orgID, userID)
    if err != nil {
        return err
    }

    // 2. Remove from organization members
    err = s.repo.RemoveMember(ctx, orgID, userID)
    if err != nil {
        return err
    }

    // 3. Audit log
    s.createAuditLog(ctx, &AuditLog{
        OrganizationID: &orgID,
        UserID:         userID,
        Action:         "member_removed",
        ResourceType:   "organization",
        ResourceID:     &orgID,
    })

    // Note: User immediately loses access because their encrypted OEK is deleted
    // No need to re-encrypt anything for other members

    return nil
}
```

### 5. Unshare Connection

```go
func (s *Service) UnshareConnection(ctx context.Context, connID, userID string) error {
    // Get connection
    conn, err := s.store.GetByID(ctx, connID)
    if err != nil {
        return err
    }

    // Verify ownership
    if conn.CreatedBy != userID {
        return fmt.Errorf("only creator can unshare")
    }

    // Delete shared credential
    err = s.sharedCredStore.DeleteSharedCredential(ctx, connID, *conn.OrganizationID)
    if err != nil {
        return err
    }

    // Update connection to personal
    conn.OrganizationID = nil
    conn.Visibility = "personal"
    conn.EncryptionType = "personal"

    err = s.store.Update(ctx, conn)
    if err != nil {
        return err
    }

    // Personal encrypted_credentials record still exists, so user can still use it

    return nil
}
```

### 6. Key Rotation (Advanced)

```go
func (s *Service) RotateOrganizationEnvelopeKey(ctx context.Context, orgID, adminUserID string) error {
    // 1. Verify admin permissions
    member, err := s.repo.GetMember(ctx, orgID, adminUserID)
    if err != nil || member.Role != RoleOwner {
        return fmt.Errorf("insufficient permissions")
    }

    // 2. Generate new OEK
    newOEK, err := crypto.GenerateAESKey()
    if err != nil {
        return err
    }

    // 3. Get old OEK
    encryptedOldOEK, err := s.getOrgKeyForUser(ctx, orgID, adminUserID)
    if err != nil {
        return err
    }

    adminMasterKey, err := s.getMasterKeyForUser(ctx, adminUserID)
    if err != nil {
        return err
    }

    oldOEK, err := crypto.DecryptAESGCM(encryptedOldOEK, adminMasterKey)
    if err != nil {
        return err
    }

    // 4. Get all shared credentials for this org
    sharedCreds, err := s.sharedCredStore.GetAllForOrganization(ctx, orgID)
    if err != nil {
        return err
    }

    // 5. Re-encrypt each credential with new OEK
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    for _, cred := range sharedCreds {
        // Decrypt with old OEK
        password, err := crypto.DecryptPasswordWithOrgKey(cred, oldOEK)
        if err != nil {
            return err
        }

        // Re-encrypt with new OEK
        newCred, err := crypto.EncryptPasswordWithOrgKey(password, newOEK)
        if err != nil {
            return err
        }

        // Update in database
        err = s.sharedCredStore.UpdateCredential(ctx, cred.ConnectionID, orgID, newCred)
        if err != nil {
            return err
        }
    }

    // 6. Re-encrypt OEK for all members
    members, err := s.repo.GetMembers(ctx, orgID)
    if err != nil {
        return err
    }

    for _, member := range members {
        memberMasterKey, err := s.getMasterKeyForUser(ctx, member.UserID)
        if err != nil {
            return err
        }

        encryptedNewOEK, err := crypto.EncryptAESGCM(newOEK, memberMasterKey)
        if err != nil {
            return err
        }

        err = s.updateOrgKeyForUser(ctx, orgID, member.UserID, encryptedNewOEK, 2) // version 2
        if err != nil {
            return err
        }
    }

    if err := tx.Commit(); err != nil {
        return err
    }

    return nil
}
```

---

## Security Properties

### Zero-Knowledge Guarantees

1. **Server Never Sees Plaintext**:
   - User passwords: Only client knows them
   - Master keys: Encrypted with user passwords, never in plaintext on server
   - OEK: Never stored in plaintext, always encrypted per-user
   - Connection passwords: Always encrypted (personal MK or org OEK)

2. **Encryption Layers**:
   ```
   Connection Password (plaintext) - CLIENT ONLY
   └─ Layer 1: Encrypted with OEK → stored in shared_credentials
      └─ Layer 2: OEK encrypted with User's Master Key → stored in org_envelope_keys
         └─ Layer 3: Master Key encrypted with User's Password → stored in user_master_keys
   ```

### Access Control

1. **Personal Connections**: Only creator can decrypt (has master key)
2. **Shared Connections**: Any org member can decrypt (has OEK via their master key)
3. **Revocation**: Instant (delete user's OEK record)
4. **Audit**: All decryptions logged with timestamp, IP, success/failure

### Attack Resistance

| Attack | Mitigation |
|--------|-----------|
| Database breach | All data encrypted, keys not stored in plaintext |
| Member exfiltration | Per-user keys revocable, audit logs track access |
| SQL injection | Parameterized queries throughout |
| Replay attacks | GCM auth tags prevent tampering |
| Key compromise | Rotation capability without service interruption |

---

## Performance Considerations

### Scalability

**Current Design**: O(1) operations for all credential access
- Get shared password: 3 database queries (fixed)
  1. Get shared_credentials
  2. Get user's org_envelope_key
  3. Get user's master_key (cached in session)

**Storage Overhead**:
- Organization with N members and M shared connections:
  - org_envelope_keys: N records (one OEK per member)
  - shared_credentials: M records (one per shared connection)
  - Total: O(N + M)

**Example**:
- 50 members, 20 shared connections
- Storage: 50 + 20 = 70 records
- vs. Re-encrypt approach: 50 × 20 = 1,000 records

### Caching Strategy

```go
type SessionCache struct {
    UserMasterKey []byte        // Cached after login
    OrgKeys       map[string][]byte // orgID -> decrypted OEK
}

// Cache OEK after first use
func (s *Service) getCachedOEK(ctx context.Context, orgID, userID string) ([]byte, error) {
    // Check session cache
    if oek, ok := s.sessionCache.OrgKeys[orgID]; ok {
        return oek, nil
    }

    // Decrypt and cache
    oek, err := s.decryptOEKForUser(ctx, orgID, userID)
    if err != nil {
        return nil, err
    }

    s.sessionCache.OrgKeys[orgID] = oek
    return oek, nil
}
```

---

## Migration Plan

### Phase 1: Schema Migration (Zero Downtime)

```sql
-- Add new tables (doesn't affect existing data)
CREATE TABLE organization_envelope_keys ...
CREATE TABLE shared_credentials ...
CREATE TABLE credential_access_log ...

-- Add encryption_type column with default
ALTER TABLE connection_templates
ADD COLUMN encryption_type TEXT DEFAULT 'personal';
```

### Phase 2: Code Deployment

1. Deploy new crypto functions (backward compatible)
2. Deploy new stores (SharedCredentialStore, OrgKeyStore)
3. Deploy new service methods (GetSharedConnectionPassword)
4. Update ShareConnection() to use new flow

### Phase 3: Data Migration

```go
func MigrateExistingSharedConnections(ctx context.Context) error {
    // Find all shared connections
    sharedConns := getAllSharedConnections(ctx)

    for _, conn := range sharedConns {
        // Skip if already migrated
        if conn.EncryptionType == "shared" {
            continue
        }

        // Get creator's personal credential
        personalCred := getPersonalCredential(ctx, conn.CreatedBy, conn.ID)

        // Decrypt with creator's master key
        password := decryptPersonal(personalCred, conn.CreatedBy)

        // Re-encrypt with org envelope key
        sharedCred := encryptShared(password, conn.OrganizationID)

        // Store in shared_credentials
        storeSharedCredential(ctx, sharedCred)

        // Update connection
        conn.EncryptionType = "shared"
        updateConnection(ctx, conn)
    }
}
```

### Phase 4: Cleanup (After Full Migration)

```sql
-- Optionally remove old personal credentials for shared connections
DELETE FROM encrypted_credentials
WHERE connection_id IN (
    SELECT id FROM connection_templates WHERE encryption_type = 'shared'
);
```

---

## Alternative Approaches Considered

### Alternative 1: Re-Encrypt for Each Member

**Approach**: Store one encrypted password per member per shared connection

```sql
shared_credentials_per_member (
    connection_id TEXT,
    user_id TEXT,
    encrypted_password TEXT,  -- Encrypted with user's master key
    PRIMARY KEY (connection_id, user_id)
)
```

**Pros**:
- Simpler crypto model (no OEK)
- Revocation is delete

**Cons**:
- Storage: O(members × connections) - doesn't scale
- Share operation: Must re-encrypt for ALL members
- New member joins: Must re-encrypt ALL shared connections
- 100 members, 50 connections = 5,000 records vs. 150 records

**Verdict**: Rejected due to poor scalability

### Alternative 2: Single Organization Master Key

**Approach**: One master key for entire org, all members share it

**Cons**:
- No per-user revocation
- Key rotation requires re-encrypting all members
- If one member's device compromised, entire org compromised

**Verdict**: Rejected due to security concerns

### Alternative 3: Asymmetric Encryption (RSA)

**Approach**: Public/private key pairs for each user

**Pros**:
- Elegant key distribution

**Cons**:
- Performance: RSA decryption is 1000x slower than AES
- Complexity: Key management, rotation, backup/recovery
- Still need envelope key for group access

**Verdict**: Rejected due to complexity and performance

---

## Testing Strategy

### Unit Tests

```go
// Test OEK encryption/decryption
func TestOrgEnvelopeKeyEncryption(t *testing.T) {
    masterKey := crypto.GenerateAESKey()
    oek := crypto.GenerateAESKey()

    encrypted, err := crypto.EncryptAESGCM(oek, masterKey)
    assert.NoError(t, err)

    decrypted, err := crypto.DecryptAESGCM(encrypted, masterKey)
    assert.NoError(t, err)
    assert.Equal(t, oek, decrypted)
}

// Test share flow
func TestShareConnection(t *testing.T) {
    // User A creates connection
    // User A shares with org
    // User B decrypts successfully
}

// Test revocation
func TestMemberRevocation(t *testing.T) {
    // User B in org
    // User B can decrypt shared connection
    // Admin removes User B
    // User B cannot decrypt anymore
}
```

### Integration Tests

```go
func TestEndToEndSharing(t *testing.T) {
    // 1. User A creates org
    // 2. User A creates connection with password
    // 3. User B joins org
    // 4. User A shares connection
    // 5. User B successfully connects to database
    // 6. User C joins org
    // 7. User C successfully connects to database
    // 8. User A unshares connection
    // 9. User B/C cannot access anymore
}
```

### Security Tests

```go
func TestCrossOrgIsolation(t *testing.T) {
    // User in Org A cannot decrypt Org B's credentials
}

func TestAuditLogging(t *testing.T) {
    // All credential access logged
    // Failed attempts logged
}

func TestZeroKnowledge(t *testing.T) {
    // Server code never logs plaintext passwords
    // Database contains no plaintext passwords
}
```

---

## Rollout Plan

### Week 1: Development

- [ ] Implement crypto functions (EncryptAESGCM, DecryptAESGCM)
- [ ] Create new stores (OrgKeyStore, SharedCredentialStore)
- [ ] Write unit tests (100% coverage)

### Week 2: Integration

- [ ] Implement ShareConnection() new flow
- [ ] Implement GetSharedConnectionPassword()
- [ ] Implement AddMember() OEK distribution
- [ ] Implement RemoveMember() revocation
- [ ] Integration tests

### Week 3: Migration & Testing

- [ ] Create migration scripts
- [ ] Test on staging with production data copy
- [ ] Security audit
- [ ] Performance testing (1000 members, 100 connections)

### Week 4: Deployment

- [ ] Deploy to production (off-hours)
- [ ] Run migration
- [ ] Monitor audit logs
- [ ] Verify shared connections work

---

## Monitoring & Observability

### Metrics to Track

```go
// Prometheus metrics
credential_decrypt_duration_seconds      // Latency
credential_decrypt_errors_total         // Failures
credential_share_operations_total       // Share events
org_envelope_key_rotations_total        // Key rotations
```

### Alert Conditions

1. Credential decrypt error rate > 1%
2. Decrypt latency > 500ms (p99)
3. Failed decrypt attempts > 10/min for single user (potential attack)
4. OEK not found for active member (data inconsistency)

### Audit Log Queries

```sql
-- Who accessed this connection recently?
SELECT user_id, timestamp, ip_address, success
FROM credential_access_log
WHERE connection_id = 'conn-xyz'
  AND timestamp > unixepoch('now', '-7 days')
ORDER BY timestamp DESC;

-- Suspicious activity: many failed decrypts
SELECT user_id, COUNT(*) as failures
FROM credential_access_log
WHERE action = 'decrypt' AND success = 0
  AND timestamp > unixepoch('now', '-1 hour')
GROUP BY user_id
HAVING failures > 10;
```

---

## Future Enhancements

### Short Term (3 months)

1. **Browser Extension Support**: Store master key in browser extension, never in web app
2. **Hardware Key Support**: Use YubiKey for master key encryption
3. **Key Escrow**: Optional recovery mechanism for lost passwords

### Long Term (6+ months)

1. **Hierarchical Organizations**: Parent/child orgs with inherited access
2. **Time-Limited Sharing**: Credentials expire after N days
3. **Break-Glass Access**: Emergency access with multi-party approval
4. **FIPS 140-2 Compliance**: Certified crypto modules

---

## Conclusion

This design solves the credential sharing problem with:

✅ **Security**: Zero-knowledge, per-user revocation, full audit trail
✅ **Scalability**: O(N + M) storage, O(1) operations
✅ **Usability**: Transparent to users, instant revocation
✅ **Maintainability**: Clean separation of personal vs. shared credentials

The envelope key approach provides the best balance of security, performance, and maintainability for multi-tenant credential sharing.

---

## Appendices

### Appendix A: Crypto Implementation

See `/Users/jacob_1/projects/howlerops/backend-go/pkg/crypto/org_envelope.go` (to be created)

### Appendix B: Migration Scripts

See `/Users/jacob_1/projects/howlerops/backend-go/pkg/storage/turso/migrations/009_shared_credentials.sql` (to be created)

### Appendix C: API Changes

```go
// New endpoints
POST /api/connections/{id}/share       // Share connection with org
POST /api/connections/{id}/unshare     // Unshare connection
GET  /api/connections/{id}/password    // Get decrypted password (personal or shared)

// New internal methods
GetSharedConnectionPassword(connID, userID, orgID) string
ShareConnectionWithOrg(connID, userID, orgID) error
UnshareConnection(connID, userID) error
RotateOrgEnvelopeKey(orgID, adminUserID) error
```

### Appendix D: Database Schema Summary

```
New Tables:
- organization_envelope_keys (N records per org, one per member)
- shared_credentials (M records per org, one per shared connection)
- credential_access_log (audit trail)

Modified Tables:
- connection_templates (add encryption_type field)

Unchanged Tables:
- user_master_keys (still used for personal credentials)
- encrypted_credentials (still used for personal credentials)
```

### Appendix E: Zero-Knowledge Verification

```
Client-side operations (plaintext visible):
- User enters password
- Derive PBKDF2 key from password
- Decrypt master key with PBKDF2 key
- Use master key to decrypt credentials

Server-side operations (no plaintext):
- Store encrypted_master_key
- Store encrypted_oek
- Store encrypted_password
- Return encrypted data to client
- Log metadata (not plaintext)

Verification: Grep codebase for "TODO: ZERO_KNOWLEDGE_VIOLATION"
```
