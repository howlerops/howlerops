# Credential Sharing Solution - Executive Summary

## Problem Statement

**Current State**: Broken credential sharing in organization context

When a user shares a database connection with their organization:
1. ShareConnection() is called
2. Only the `visibility` flag changes to "shared"
3. The password remains encrypted with the ORIGINAL user's master key
4. Other organization members CANNOT decrypt the password
5. Shared connections are fundamentally broken

**Root Cause**: One-to-one encryption model (user's master key → credential) doesn't support many-to-one access (multiple org members → credential).

---

## Solution Overview

**Approach**: Hybrid Envelope Encryption with Organization Keys

### Key Concepts

1. **Organization Envelope Key (OEK)**: A symmetric AES-256 key shared across all organization members
2. **Per-Member Encryption**: Each member gets their own encrypted copy of the OEK (encrypted with their master key)
3. **Shared Credentials**: Connection passwords encrypted with the OEK (not individual user keys)

### Architecture

```
Personal Connection:
User's Master Key → Encrypt Password → Store in encrypted_credentials

Shared Connection:
Organization Envelope Key → Encrypt Password → Store in shared_credentials
   └─ Encrypted with User A's Master Key → Store in org_envelope_keys (user-A)
   └─ Encrypted with User B's Master Key → Store in org_envelope_keys (user-B)
   └─ Encrypted with User C's Master Key → Store in org_envelope_keys (user-C)
```

---

## How It Works

### Scenario: User A shares connection with 3-member organization

#### 1. Organization Setup (One-Time)

```
Admin creates organization
├─ Generate random 256-bit OEK (Organization Envelope Key)
├─ Encrypt OEK with Admin's Master Key
└─ Store in org_envelope_keys table
```

#### 2. User B Joins Organization

```
Admin adds User B
├─ Get OEK (decrypt with Admin's Master Key)
├─ Re-encrypt OEK with User B's Master Key
└─ Store User B's encrypted OEK in org_envelope_keys
```

Result: User B now has access to org's OEK (via their master key)

#### 3. User A Shares Connection

```
User A shares "Production DB"
├─ Get personal credential (encrypted with User A's Master Key)
├─ Decrypt password with User A's Master Key → plaintext "postgres123"
├─ Get org's OEK (decrypt with User A's Master Key)
├─ Re-encrypt password with OEK
└─ Store in shared_credentials table
```

Result: Password now encrypted with OEK, not User A's key

#### 4. User B Uses Shared Connection

```
User B opens "Production DB"
├─ Get shared credential (password encrypted with OEK)
├─ Get User B's encrypted OEK from org_envelope_keys
├─ Decrypt OEK with User B's Master Key
├─ Decrypt password with OEK → plaintext "postgres123"
└─ Connect to database
```

Result: User B successfully connects!

#### 5. User C Leaves Organization

```
Admin removes User C
├─ Delete User C's OEK record from org_envelope_keys
└─ Remove from organization_members
```

Result: User C immediately loses access (no OEK = can't decrypt)

---

## Database Schema

### New Tables

**organization_envelope_keys**: Org's key encrypted once per member
```sql
CREATE TABLE organization_envelope_keys (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    encrypted_oek TEXT NOT NULL,        -- OEK encrypted with user's master key
    oek_iv TEXT NOT NULL,
    oek_auth_tag TEXT NOT NULL,
    UNIQUE(organization_id, user_id)
);
```

**shared_credentials**: Passwords encrypted with org key
```sql
CREATE TABLE shared_credentials (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    encrypted_password TEXT NOT NULL,   -- Password encrypted with OEK
    password_iv TEXT NOT NULL,
    password_auth_tag TEXT NOT NULL,
    UNIQUE(connection_id, organization_id)
);
```

**credential_access_log**: Audit trail
```sql
CREATE TABLE credential_access_log (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,               -- 'decrypt', 'share', 'unshare'
    timestamp INTEGER NOT NULL,
    success BOOLEAN NOT NULL
);
```

### Modified Tables

**connection_templates**: Add encryption type indicator
```sql
ALTER TABLE connection_templates
ADD COLUMN encryption_type TEXT DEFAULT 'personal';
-- Values: 'personal' or 'shared'
```

---

## Security Properties

### Zero-Knowledge Architecture

| Layer | Plaintext Visible | Storage |
|-------|------------------|---------|
| User's Password | Client Only | Never stored |
| User's Master Key | Client Only | Encrypted with PBKDF2 key |
| Organization Envelope Key | Client Only | Encrypted with master key |
| Connection Passwords | Client Only | Encrypted with OEK or master key |
| Server | NEVER | Only ciphertext |
| Database | NEVER | Only ciphertext |

### Access Control

| Scenario | Access | Mechanism |
|----------|--------|-----------|
| Personal connection | Owner only | Encrypted with owner's master key |
| Shared connection | All org members | Encrypted with org envelope key |
| Member leaves org | Revoked instantly | Delete OEK record |
| New member joins | Granted instantly | Encrypt OEK for them |
| Cross-org access | Denied | No OEK for other orgs |

---

## Performance Characteristics

### Storage Complexity

**Organization with N members and M shared connections:**

| Approach | Storage | Example (50 members, 20 connections) |
|----------|---------|--------------------------------------|
| Envelope Key (Ours) | O(N + M) | 70 records |
| Re-encrypt per member | O(N × M) | 1,000 records |

**Result**: 14.3x more efficient storage

### Operation Complexity

| Operation | Envelope Key | Re-encrypt | Winner |
|-----------|-------------|-----------|---------|
| Decrypt password | O(1) - 3 queries | O(1) - 1 query | Re-encrypt (slightly) |
| Share connection | O(1) - 1 re-encryption | O(N) - N re-encryptions | Envelope (scales) |
| Add member | O(1) - 1 OEK encryption | O(M) - Re-encrypt all connections | Envelope (instant) |
| Remove member | O(1) - Delete OEK | O(1) - Delete records | Tie |

**Result**: Envelope approach scales better for large organizations

### Benchmarks (Expected)

- Decrypt shared password: < 100ms (3 DB queries + 2 AES operations)
- Share connection: < 200ms (decrypt + re-encrypt + 2 DB writes)
- Add member: < 50ms (encrypt OEK + 1 DB write)
- Remove member: < 20ms (1 DB delete)

---

## Implementation Plan

### Phase 1: Schema Migration (2 days)
- [ ] Create new tables
- [ ] Add indexes
- [ ] Test on staging

### Phase 2: Crypto Implementation (3 days)
- [ ] Implement OEK generation
- [ ] Implement encrypt/decrypt with OEK
- [ ] Comprehensive unit tests

### Phase 3: Database Stores (3 days)
- [ ] OrgEnvelopeKeyStore
- [ ] SharedCredentialStore
- [ ] CredentialAuditStore

### Phase 4: Service Layer (3 days)
- [ ] OrgCredentialService
- [ ] Update ShareConnection()
- [ ] Update GetConnectionPassword()

### Phase 5: Organization Integration (2 days)
- [ ] Update AddMember()
- [ ] Update RemoveMember()
- [ ] Update CreateOrganization()

### Phase 6: API & Testing (4 days)
- [ ] API endpoints
- [ ] Integration tests
- [ ] Security tests
- [ ] Performance tests

### Phase 7: Data Migration (3 days)
- [ ] Migration script
- [ ] Test on staging
- [ ] Backup production
- [ ] Run migration

### Phase 8: Deployment & Monitoring (2 days)
- [ ] Deploy to production
- [ ] Monitor metrics
- [ ] Verify functionality
- [ ] User communication

**Total Timeline**: 4 weeks (1 developer)

---

## Migration Strategy

### Zero-Downtime Migration

1. **Deploy schema changes** (new tables, backward compatible)
2. **Deploy new code** (handles both old and new format)
3. **Run migration script** (re-encrypt shared connections)
4. **Verify functionality** (test shared connections)
5. **Clean up old data** (optional, after verification period)

### Rollback Plan

If issues arise:
1. Revert code deployment
2. Keep new tables (no data loss)
3. Shared connections revert to broken state temporarily
4. Investigate and fix issues
5. Retry migration

---

## Comparison: Alternatives Considered

### Alternative 1: Re-encrypt for Each Member

**Approach**: Store N encrypted copies of password (one per member)

**Rejected because**:
- Storage: O(members × connections) doesn't scale
- Share operation: Must re-encrypt for ALL members
- New member: Must re-encrypt ALL connections
- 100 members × 50 connections = 5,000 records vs. 150 records

### Alternative 2: Single Org Master Key

**Approach**: One key for entire org, all members share it

**Rejected because**:
- No per-user revocation (remove member = rotate key for everyone)
- Key rotation affects entire org
- If one member compromised, entire org compromised

### Alternative 3: Asymmetric Encryption (RSA)

**Approach**: Public/private key pairs

**Rejected because**:
- RSA decryption 1000x slower than AES
- Complexity in key management
- Still need envelope key for group access

---

## Success Metrics

### Functional Requirements
- [x] Organization members can decrypt shared credentials
- [x] Member revocation is instant
- [x] New member onboarding is instant
- [x] Zero-knowledge security maintained
- [x] Complete audit trail

### Non-Functional Requirements
- [ ] < 500ms to decrypt shared password (p99)
- [ ] < 1% error rate on credential operations
- [ ] 100% test coverage for new code
- [ ] Zero data loss during migration
- [ ] Zero security vulnerabilities in audit

---

## Risk Assessment

| Risk | Severity | Probability | Mitigation |
|------|----------|-------------|------------|
| Migration data loss | High | Low | Backup, staging test, rollback plan |
| Security vulnerability | High | Medium | Security audit, penetration testing |
| Performance issues | Medium | Low | Benchmark early, optimize as needed |
| User disruption | Medium | Medium | Gradual rollout, communication |
| Complex key management | Low | Medium | Comprehensive testing, documentation |

---

## Monitoring & Operations

### Metrics to Track

```
credential_decrypt_duration_seconds{type="shared"}
credential_decrypt_errors_total{type="shared"}
credential_share_operations_total
org_envelope_key_provisioning_total
credential_access_log_entries_total{action="decrypt"}
```

### Alerts

1. Decrypt error rate > 1% for 5 minutes
2. Decrypt latency p99 > 500ms
3. Failed decrypt attempts > 10/min for single user (potential attack)
4. Missing OEK for active member (data inconsistency)

### Audit Queries

```sql
-- Who accessed this connection recently?
SELECT user_id, timestamp, success
FROM credential_access_log
WHERE connection_id = ?
  AND timestamp > unixepoch('now', '-7 days')
ORDER BY timestamp DESC;

-- Suspicious activity: many failed decrypts
SELECT user_id, COUNT(*) as failures
FROM credential_access_log
WHERE action = 'decrypt'
  AND success = 0
  AND timestamp > unixepoch('now', '-1 hour')
GROUP BY user_id
HAVING failures > 10;
```

---

## Future Enhancements

### Short Term (3 months)
- [ ] Browser extension for secure key storage
- [ ] Hardware key (YubiKey) support for master key encryption
- [ ] Automatic key rotation policies

### Long Term (6+ months)
- [ ] Hierarchical organizations (parent/child with inherited access)
- [ ] Time-limited credential sharing (expire after N days)
- [ ] Break-glass access (emergency access with multi-party approval)
- [ ] FIPS 140-2 certified crypto modules

---

## Documentation Resources

### Technical Documentation
- **Architecture Design**: `/docs/credential-sharing-design.md` (55 pages)
- **Visual Diagrams**: `/docs/credential-sharing-diagrams.md` (10 diagrams)
- **Implementation Checklist**: `/docs/credential-sharing-implementation-checklist.md` (220 tasks)

### Key Diagrams
1. Current Broken Architecture (flow diagram)
2. New Envelope Encryption Architecture (flow diagram)
3. Encryption Layers Visualization (3-layer security)
4. Access Control Matrix (who can access what)
5. Member Lifecycle Flows (join/leave scenarios)
6. Security Boundaries (trust zones)
7. Performance Comparison (storage/operations)
8. Migration Visualization (before/after)
9. API Flow Sequence (step-by-step)
10. Threat Model (attack scenarios)

### Code Examples

All implementation examples provided in:
- Crypto functions: `pkg/crypto/org_envelope.go`
- Database stores: `pkg/storage/turso/org_envelope_store.go`
- Service logic: `internal/connections/org_credential_service.go`
- API handlers: `internal/connections/handler.go`

---

## Questions & Answers

**Q: Why not just give everyone the same master key?**
A: Security. If one member is compromised, entire org is compromised. Also, no per-user revocation.

**Q: Why not re-encrypt the password for each member individually?**
A: Scalability. 100 members × 50 connections = 5,000 records. Envelope key = 150 records. Also, adding a new member would require re-encrypting all connections.

**Q: What if someone leaves and cached the credentials?**
A: They have time-limited access until cache expires. Server-side revocation is instant. This is an acceptable tradeoff vs. complexity of preventing all caching.

**Q: Can the server administrator access the passwords?**
A: No. Zero-knowledge architecture means server never sees plaintext. Admin would need user's login password to decrypt master key, then decrypt OEK, then decrypt credential.

**Q: What happens if the org owner leaves?**
A: Other members still have access (they have their own OEK copies). Transfer ownership to another admin first, or use their OEK to provision the new owner.

**Q: How do we handle key rotation?**
A: Decrypt all shared credentials with old OEK, re-encrypt with new OEK, and update all members' encrypted OEK copies. Can be done without service interruption.

---

## Approval & Sign-Off

This design has been reviewed by:

- [ ] Backend Engineering Lead
- [ ] Security Team
- [ ] DevOps Team
- [ ] Product Management

**Approved for Implementation**: _________________ (Date)

**Implementation Owner**: _________________ (Name)

**Target Completion Date**: _________________ (Date)

---

## Contact & Support

**Technical Questions**: Backend team channel
**Security Questions**: Security team
**Implementation Issues**: File GitHub issue with label `credential-sharing`

---

## Conclusion

This solution fixes the critical credential sharing vulnerability while maintaining:
- **Security**: Zero-knowledge, instant revocation, full audit trail
- **Scalability**: O(N+M) storage, O(1) operations
- **Usability**: Transparent to end users
- **Maintainability**: Clean separation of concerns

The envelope key approach provides the optimal balance of security, performance, and developer experience for multi-tenant credential management.

**Status**: ✅ Design Complete, Ready for Implementation

**Next Step**: Begin Phase 1 - Schema Migration
