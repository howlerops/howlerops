# Credential Sharing Implementation Checklist

## Phase 1: Database Schema (Week 1, Days 1-2)

### Migration Script: 009_shared_credentials.sql

```sql
-- File: pkg/storage/turso/migrations/009_shared_credentials.sql

-- [ ] Create organization_envelope_keys table
CREATE TABLE IF NOT EXISTS organization_envelope_keys (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    encrypted_oek TEXT NOT NULL,
    oek_iv TEXT NOT NULL,
    oek_auth_tag TEXT NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(organization_id, user_id, key_version)
);

-- [ ] Create indexes
CREATE INDEX idx_org_keys_org_id ON organization_envelope_keys(organization_id);
CREATE INDEX idx_org_keys_user_id ON organization_envelope_keys(user_id);
CREATE INDEX idx_org_keys_version ON organization_envelope_keys(organization_id, key_version);

-- [ ] Create shared_credentials table
CREATE TABLE IF NOT EXISTS shared_credentials (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    encrypted_password TEXT NOT NULL,
    password_iv TEXT NOT NULL,
    password_auth_tag TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,
    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    UNIQUE(connection_id, organization_id)
);

-- [ ] Create indexes
CREATE INDEX idx_shared_creds_connection ON shared_credentials(connection_id);
CREATE INDEX idx_shared_creds_org ON shared_credentials(organization_id);
CREATE INDEX idx_shared_creds_created_by ON shared_credentials(created_by);

-- [ ] Create credential_access_log table
CREATE TABLE IF NOT EXISTS credential_access_log (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    ip_address TEXT,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- [ ] Create indexes
CREATE INDEX idx_cred_access_conn ON credential_access_log(connection_id);
CREATE INDEX idx_cred_access_user ON credential_access_log(user_id);
CREATE INDEX idx_cred_access_time ON credential_access_log(timestamp);

-- [ ] Add encryption_type to connection_templates
ALTER TABLE connection_templates ADD COLUMN encryption_type TEXT DEFAULT 'personal';
```

### Testing

- [ ] Test migration on local SQLite
- [ ] Test migration on Turso staging
- [ ] Verify indexes created correctly
- [ ] Verify foreign key constraints work
- [ ] Test rollback (if needed)

---

## Phase 2: Crypto Implementation (Week 1, Days 3-5)

### File: pkg/crypto/org_envelope.go

```go
// [ ] Create file pkg/crypto/org_envelope.go

package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
)

// [ ] Implement GenerateOrgEnvelopeKey
func GenerateOrgEnvelopeKey() ([]byte, error) {
    key := make([]byte, 32) // AES-256
    _, err := rand.Read(key)
    if err != nil {
        return nil, fmt.Errorf("failed to generate key: %w", err)
    }
    return key, nil
}

// [ ] Implement EncryptWithOEK
type OEKEncryptedData struct {
    Ciphertext string
    IV         string
    AuthTag    string
}

func EncryptWithOEK(plaintext []byte, oek []byte) (*OEKEncryptedData, error) {
    block, err := aes.NewCipher(oek)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }

    ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

    return &OEKEncryptedData{
        Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
        IV:         base64.StdEncoding.EncodeToString(nonce),
        AuthTag:    "", // GCM includes auth tag in ciphertext
    }, nil
}

// [ ] Implement DecryptWithOEK
func DecryptWithOEK(encrypted *OEKEncryptedData, oek []byte) ([]byte, error) {
    block, err := aes.NewCipher(oek)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
    if err != nil {
        return nil, err
    }

    nonce, err := base64.StdEncoding.DecodeString(encrypted.IV)
    if err != nil {
        return nil, err
    }

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, fmt.Errorf("decryption failed: %w", err)
    }

    return plaintext, nil
}

// [ ] Implement EncryptOEKWithMasterKey
func EncryptOEKWithMasterKey(oek []byte, masterKey []byte) (*OEKEncryptedData, error) {
    return EncryptWithOEK(oek, masterKey)
}

// [ ] Implement DecryptOEKWithMasterKey
func DecryptOEKWithMasterKey(encrypted *OEKEncryptedData, masterKey []byte) ([]byte, error) {
    return DecryptWithOEK(encrypted, masterKey)
}
```

### File: pkg/crypto/org_envelope_test.go

```go
// [ ] Create comprehensive tests

func TestOrgEnvelopeKeyGeneration(t *testing.T) {}
func TestOEKEncryptDecrypt(t *testing.T) {}
func TestOEKWithMasterKey(t *testing.T) {}
func TestInvalidOEKDecryption(t *testing.T) {}
func TestOEKTampering(t *testing.T) {}
```

---

## Phase 3: Database Stores (Week 2, Days 1-3)

### File: pkg/storage/turso/org_envelope_store.go

```go
// [ ] Create file pkg/storage/turso/org_envelope_store.go

package turso

import (
    "context"
    "database/sql"
    "github.com/jbeck018/howlerops/backend-go/pkg/crypto"
)

type OrgEnvelopeKeyStore struct {
    db     *sql.DB
    logger *logrus.Logger
}

// [ ] Implement NewOrgEnvelopeKeyStore
func NewOrgEnvelopeKeyStore(db *sql.DB, logger *logrus.Logger) *OrgEnvelopeKeyStore

// [ ] Implement StoreOEKForUser
func (s *OrgEnvelopeKeyStore) StoreOEKForUser(
    ctx context.Context,
    orgID, userID string,
    encryptedOEK *crypto.OEKEncryptedData,
) error

// [ ] Implement GetOEKForUser
func (s *OrgEnvelopeKeyStore) GetOEKForUser(
    ctx context.Context,
    orgID, userID string,
) (*crypto.OEKEncryptedData, error)

// [ ] Implement GetAnyOEKForOrg (for copying to new member)
func (s *OrgEnvelopeKeyStore) GetAnyOEKForOrg(
    ctx context.Context,
    orgID string,
) (userID string, encrypted *crypto.OEKEncryptedData, error)

// [ ] Implement DeleteOEKForUser (revocation)
func (s *OrgEnvelopeKeyStore) DeleteOEKForUser(
    ctx context.Context,
    orgID, userID string,
) error

// [ ] Implement UpdateOEKForUser (key rotation)
func (s *OrgEnvelopeKeyStore) UpdateOEKForUser(
    ctx context.Context,
    orgID, userID string,
    encryptedOEK *crypto.OEKEncryptedData,
    version int,
) error
```

### File: pkg/storage/turso/shared_credential_store.go

```go
// [ ] Create file pkg/storage/turso/shared_credential_store.go

package turso

type SharedCredentialStore struct {
    db     *sql.DB
    logger *logrus.Logger
}

// [ ] Implement NewSharedCredentialStore
func NewSharedCredentialStore(db *sql.DB, logger *logrus.Logger) *SharedCredentialStore

// [ ] Implement StoreSharedCredential
func (s *SharedCredentialStore) StoreSharedCredential(
    ctx context.Context,
    connID, orgID string,
    encrypted *crypto.OEKEncryptedData,
    createdBy string,
) error

// [ ] Implement GetSharedCredential
func (s *SharedCredentialStore) GetSharedCredential(
    ctx context.Context,
    connID, orgID string,
) (*crypto.OEKEncryptedData, error)

// [ ] Implement DeleteSharedCredential
func (s *SharedCredentialStore) DeleteSharedCredential(
    ctx context.Context,
    connID, orgID string,
) error

// [ ] Implement GetAllForOrganization (for key rotation)
func (s *SharedCredentialStore) GetAllForOrganization(
    ctx context.Context,
    orgID string,
) ([]*SharedCredential, error)

// [ ] Implement UpdateSharedCredential (for key rotation)
func (s *SharedCredentialStore) UpdateSharedCredential(
    ctx context.Context,
    connID, orgID string,
    encrypted *crypto.OEKEncryptedData,
) error
```

### File: pkg/storage/turso/credential_audit_store.go

```go
// [ ] Create file pkg/storage/turso/credential_audit_store.go

type CredentialAuditStore struct {
    db     *sql.DB
    logger *logrus.Logger
}

// [ ] Implement LogCredentialAccess
func (s *CredentialAuditStore) LogCredentialAccess(
    ctx context.Context,
    connID, orgID, userID, action string,
    success bool,
    errMsg *string,
) error

// [ ] Implement GetAccessLogs
func (s *CredentialAuditStore) GetAccessLogs(
    ctx context.Context,
    filters map[string]interface{},
    limit, offset int,
) ([]*CredentialAccessLog, error)
```

### Testing

- [ ] Unit tests for OrgEnvelopeKeyStore
- [ ] Unit tests for SharedCredentialStore
- [ ] Unit tests for CredentialAuditStore
- [ ] Integration tests with real database

---

## Phase 4: Service Layer (Week 2, Days 4-5)

### File: internal/connections/org_credential_service.go

```go
// [ ] Create file internal/connections/org_credential_service.go

package connections

type OrgCredentialService struct {
    oekStore           *turso.OrgEnvelopeKeyStore
    sharedCredStore    *turso.SharedCredentialStore
    credStore          *turso.CredentialStore
    masterKeyStore     *turso.MasterKeyStore
    auditStore         *turso.CredentialAuditStore
    logger             *logrus.Logger
}

// [ ] Implement NewOrgCredentialService
func NewOrgCredentialService(...) *OrgCredentialService

// [ ] Implement ShareConnectionWithOrg
func (s *OrgCredentialService) ShareConnectionWithOrg(
    ctx context.Context,
    connID, userID, orgID string,
) error {
    // 1. Get personal credential
    // 2. Decrypt with user's master key
    // 3. Get org envelope key
    // 4. Re-encrypt with OEK
    // 5. Store in shared_credentials
    // 6. Update connection metadata
    // 7. Audit log
}

// [ ] Implement GetSharedConnectionPassword
func (s *OrgCredentialService) GetSharedConnectionPassword(
    ctx context.Context,
    connID, userID, orgID string,
) (string, error) {
    // 1. Verify membership
    // 2. Get shared credential
    // 3. Get user's OEK
    // 4. Decrypt OEK with master key
    // 5. Decrypt password with OEK
    // 6. Audit log
}

// [ ] Implement UnshareConnection
func (s *OrgCredentialService) UnshareConnection(
    ctx context.Context,
    connID, userID string,
) error

// [ ] Implement ProvisionOEKForNewMember
func (s *OrgCredentialService) ProvisionOEKForNewMember(
    ctx context.Context,
    orgID, newUserID, adminUserID string,
) error {
    // 1. Get OEK from existing member
    // 2. Decrypt with admin's master key
    // 3. Re-encrypt with new user's master key
    // 4. Store OEK for new user
}

// [ ] Implement RevokeOEKForMember
func (s *OrgCredentialService) RevokeOEKForMember(
    ctx context.Context,
    orgID, userID string,
) error
```

### Update: internal/connections/service.go

```go
// [ ] Modify ShareConnection to use OrgCredentialService
func (s *Service) ShareConnection(ctx context.Context, connID, userID, orgID string) error {
    // Add call to orgCredService.ShareConnectionWithOrg
}

// [ ] Modify UnshareConnection to use OrgCredentialService
func (s *Service) UnshareConnection(ctx context.Context, connID, userID string) error {
    // Add call to orgCredService.UnshareConnection
}
```

---

## Phase 5: Organization Integration (Week 3, Days 1-2)

### Update: internal/organization/service.go

```go
// [ ] Modify AddMember to provision OEK
func (s *Service) AddMember(ctx context.Context, orgID, userID string, role OrganizationRole) error {
    // After adding member:
    err := s.credService.ProvisionOEKForNewMember(ctx, orgID, userID, adminUserID)
    if err != nil {
        return fmt.Errorf("failed to provision credentials: %w", err)
    }
}

// [ ] Modify RemoveMember to revoke OEK
func (s *Service) RemoveMember(ctx context.Context, orgID, userID string) error {
    // Before removing member:
    err := s.credService.RevokeOEKForMember(ctx, orgID, userID)
    if err != nil {
        return fmt.Errorf("failed to revoke credentials: %w", err)
    }
}

// [ ] Add CreateOrganization OEK initialization
func (s *Service) CreateOrganization(ctx context.Context, org *Organization, ownerUserID string) error {
    // After creating org:
    // Initialize OEK for owner
    oek, err := crypto.GenerateOrgEnvelopeKey()
    // ... encrypt with owner's master key
    // ... store in org_envelope_keys
}
```

---

## Phase 6: API Endpoints (Week 3, Days 3-4)

### File: internal/connections/handler.go

```go
// [ ] Add endpoint GET /api/connections/:id/password
func (h *Handler) GetConnectionPassword(w http.ResponseWriter, r *http.Request) {
    // 1. Extract connection ID from URL
    // 2. Get user ID from context
    // 3. Check if personal or shared
    // 4. Call appropriate service method
    // 5. Return encrypted payload (client decrypts)
}

// [ ] Update POST /api/connections/:id/share
func (h *Handler) ShareConnection(w http.ResponseWriter, r *http.Request) {
    // Use new OrgCredentialService.ShareConnectionWithOrg
}

// [ ] Update DELETE /api/connections/:id/share
func (h *Handler) UnshareConnection(w http.ResponseWriter, r *http.Request) {
    // Use new OrgCredentialService.UnshareConnection
}

// [ ] Add endpoint GET /api/connections/:id/audit-log
func (h *Handler) GetCredentialAuditLog(w http.ResponseWriter, r *http.Request) {
    // Return audit logs for connection (admin only)
}
```

### API Documentation

- [ ] Document new endpoints in OpenAPI spec
- [ ] Add request/response examples
- [ ] Document error codes
- [ ] Add authentication requirements

---

## Phase 7: Client-Side Integration (Week 3, Day 5)

### Frontend Changes (if applicable)

```javascript
// [ ] Update connection password decryption flow
async function getConnectionPassword(connectionId) {
    // 1. Fetch encrypted bundle from API
    const response = await fetch(`/api/connections/${connectionId}/password`);
    const { encrypted_password, encryption_type, org_id } = await response.json();

    // 2. Check encryption type
    if (encryption_type === 'personal') {
        // Decrypt with user's master key
        return decryptWithMasterKey(encrypted_password);
    } else if (encryption_type === 'shared') {
        // Decrypt with org envelope key
        const oek = await getOrgEnvelopeKey(org_id);
        return decryptWithOEK(encrypted_password, oek);
    }
}

// [ ] Add OEK management
async function getOrgEnvelopeKey(orgId) {
    // Fetch encrypted OEK
    // Decrypt with master key
    // Cache in session
}
```

---

## Phase 8: Data Migration (Week 4, Days 1-2)

### Migration Script: cmd/migrate-shared-credentials/main.go

```go
// [ ] Create migration command

package main

func main() {
    // [ ] Find all existing shared connections
    sharedConns := findAllSharedConnections()

    for _, conn := range sharedConns {
        if conn.EncryptionType == "shared" {
            continue // Already migrated
        }

        // [ ] Get creator's personal credential
        personalCred := getPersonalCredential(conn.CreatedBy, conn.ID)

        // [ ] Decrypt with creator's master key
        // REQUIRES: Creator to login and provide master key
        // OR: One-time migration where we temporarily have master keys

        // [ ] Get/create org envelope key
        oek := getOrCreateOEK(conn.OrganizationID)

        // [ ] Re-encrypt with OEK
        sharedCred := encryptWithOEK(password, oek)

        // [ ] Store in shared_credentials
        storeSharedCredential(sharedCred)

        // [ ] Update connection.encryption_type
        updateConnection(conn.ID, "encryption_type", "shared")
    }
}
```

### Migration Plan

- [ ] Announce migration window to users
- [ ] Backup database before migration
- [ ] Run migration script on staging
- [ ] Verify all shared connections work
- [ ] Run migration script on production
- [ ] Monitor for errors
- [ ] Rollback plan ready

---

## Phase 9: Testing (Week 4, Days 3-4)

### Unit Tests

- [ ] Crypto functions (100% coverage)
- [ ] Database stores (100% coverage)
- [ ] Service methods (100% coverage)

### Integration Tests

```go
// [ ] Test: End-to-end sharing flow
func TestE2ESharing(t *testing.T) {
    // Create org, add members, share connection, verify access
}

// [ ] Test: Member revocation
func TestMemberRevocation(t *testing.T) {
    // Remove member, verify no access
}

// [ ] Test: Cross-org isolation
func TestCrossOrgIsolation(t *testing.T) {
    // User in org A cannot access org B's credentials
}

// [ ] Test: Audit logging
func TestAuditLogging(t *testing.T) {
    // Verify all access is logged
}

// [ ] Test: Key rotation
func TestKeyRotation(t *testing.T) {
    // Rotate OEK, verify all credentials still accessible
}
```

### Security Tests

- [ ] Penetration testing
- [ ] SQL injection attempts
- [ ] Authorization bypass attempts
- [ ] Audit log completeness

### Performance Tests

- [ ] Benchmark: Decrypt 100 shared credentials
- [ ] Benchmark: Add member to org with 100 shared connections
- [ ] Benchmark: Share connection in org with 100 members
- [ ] Load test: 1000 concurrent credential decryptions

---

## Phase 10: Deployment (Week 4, Day 5)

### Pre-Deployment Checklist

- [ ] All tests passing
- [ ] Security audit completed
- [ ] Performance benchmarks acceptable
- [ ] Database migration tested on staging
- [ ] Rollback plan documented
- [ ] Monitoring/alerts configured
- [ ] Documentation updated

### Deployment Steps

1. [ ] Deploy schema migration (zero downtime)
2. [ ] Deploy new code (backward compatible)
3. [ ] Run data migration script
4. [ ] Verify all shared connections work
5. [ ] Monitor error rates
6. [ ] Check audit logs for issues
7. [ ] Announce completion to users

### Post-Deployment

- [ ] Monitor Prometheus metrics
- [ ] Check error logs
- [ ] Verify audit logs
- [ ] Collect user feedback
- [ ] Document lessons learned

---

## Rollback Plan

### If Migration Fails

1. [ ] Stop migration script
2. [ ] Revert code deployment
3. [ ] Keep new tables (no data loss)
4. [ ] Shared connections revert to "broken" state
5. [ ] Investigate issues
6. [ ] Fix and retry

### If Production Issues

1. [ ] Feature flag to disable sharing
2. [ ] Revert to previous code version
3. [ ] Database rollback (if needed)
4. [ ] Restore from backup (last resort)

---

## Monitoring & Alerts

### Metrics to Track

- [ ] `credential_decrypt_duration_seconds` (latency)
- [ ] `credential_decrypt_errors_total` (failures)
- [ ] `credential_share_operations_total` (shares)
- [ ] `org_envelope_key_provisioning_total` (new members)

### Alerts

- [ ] Decrypt error rate > 1%
- [ ] Decrypt latency p99 > 500ms
- [ ] Failed decrypt attempts > 10/min per user
- [ ] Missing OEK for active member

---

## Documentation

### For Developers

- [ ] Architecture overview
- [ ] API documentation
- [ ] Database schema documentation
- [ ] Testing guide
- [ ] Deployment guide

### For Users

- [ ] How to share connections
- [ ] How to access shared connections
- [ ] Security best practices
- [ ] Troubleshooting guide

---

## Success Criteria

- [ ] All existing shared connections migrated successfully
- [ ] Organization members can decrypt shared credentials
- [ ] Member revocation is instant
- [ ] Zero-knowledge security maintained
- [ ] Performance acceptable (< 500ms decrypt time)
- [ ] Complete audit trail
- [ ] No security vulnerabilities found in audit
- [ ] 100% test coverage for new code

---

## Timeline Summary

| Week | Phase | Tasks |
|------|-------|-------|
| 1 | Setup | Schema, crypto, stores |
| 2 | Core Logic | Services, integration |
| 3 | API & Migration | Endpoints, migration script |
| 4 | Testing & Deploy | Testing, deployment, monitoring |

**Total Time**: 4 weeks (1 developer full-time)

---

## Risk Mitigation

| Risk | Mitigation |
|------|-----------|
| Migration data loss | Backup before migration, test on staging |
| Performance issues | Benchmark early, optimize if needed |
| Security vulnerabilities | Security audit, penetration testing |
| User disruption | Gradual rollout, rollback plan ready |
| Key management complexity | Comprehensive testing, documentation |

---

## Next Steps

1. **Get approval** for architecture design
2. **Allocate resources** (1 developer, 4 weeks)
3. **Start Phase 1** (schema migration)
4. **Daily standups** to track progress
5. **Weekly demos** to stakeholders

---

## Questions to Resolve Before Starting

1. [ ] Do we need hardware key (YubiKey) support now or later?
2. [ ] Should key rotation be manual or automatic?
3. [ ] What's the retention policy for audit logs?
4. [ ] Do we need real-time alerts for suspicious activity?
5. [ ] Should we support hierarchical organizations now?
6. [ ] What's the rollback deadline if migration fails?

---

## Resources

- Architecture Design: `/docs/credential-sharing-design.md`
- Visual Diagrams: `/docs/credential-sharing-diagrams.md`
- Security Audit Report: (to be created)
- Performance Benchmarks: (to be created)
- API Documentation: (to be updated)
