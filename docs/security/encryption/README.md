# Password Migration Documentation

This directory contains comprehensive documentation for migrating from OS keychain password storage to encrypted database storage.

---

## 📚 Document Index

### Quick Start

| Document | Purpose | Audience | Read Time |
|----------|---------|----------|-----------|
| **[Executive Summary](MIGRATION_EXECUTIVE_SUMMARY.md)** | High-level overview, business case, recommendation | Leadership, Product | 10 min |
| **[Quick Reference](MIGRATION_QUICK_REFERENCE.md)** | Cheat sheet for common operations | Developers | 5 min |

### Implementation

| Document | Purpose | Audience | Read Time |
|----------|---------|----------|-----------|
| **[Migration Strategy](KEYCHAIN_TO_ENCRYPTED_MIGRATION_STRATEGY.md)** | Comprehensive migration plan with all details | Engineering Lead, Architects | 45 min |
| **[Pseudocode Guide](MIGRATION_PSEUDOCODE.md)** | Detailed implementation pseudocode | Backend Developers | 30 min |
| **[Rollback Procedures](MIGRATION_ROLLBACK_PROCEDURES.md)** | Step-by-step rollback for each scenario | DevOps, On-Call Engineers | 20 min |

### Background & Design

| Document | Purpose | Audience | Read Time |
|----------|---------|----------|-----------|
| **[Password Storage Design](password-storage-design.md)** | Zero-knowledge encryption architecture | Security Engineers | 30 min |
| **[Implementation Status](implementation-status.md)** | Current implementation progress | Project Managers | 10 min |

---

## 🚀 Getting Started

### For Executives
1. Read: [Executive Summary](MIGRATION_EXECUTIVE_SUMMARY.md)
2. Review: Business metrics and timeline
3. Decide: Approve or request changes

### For Developers
1. Read: [Quick Reference](MIGRATION_QUICK_REFERENCE.md)
2. Study: [Pseudocode Guide](MIGRATION_PSEUDOCODE.md)
3. Implement: Follow pseudocode patterns
4. Test: Reference [Migration Strategy](KEYCHAIN_TO_ENCRYPTED_MIGRATION_STRATEGY.md) testing section

### For DevOps
1. Read: [Rollback Procedures](MIGRATION_ROLLBACK_PROCEDURES.md)
2. Bookmark: Emergency contacts section
3. Practice: Run rollback simulation in staging
4. Monitor: Set up alerts from Quick Reference

---

## 🎯 Key Concepts

### Hybrid Dual-Read System

The migration uses a **hybrid approach** where both storage systems (keychain and encrypted DB) work in parallel:

```
Password Read:
1. Try encrypted_credentials (Turso) ← NEW
2. Fall back to keychain ← LEGACY
3. Migrate opportunistically (background)

Password Write:
1. Store in encrypted_credentials (if master key available)
2. ALSO store in keychain (backup)
3. Success if either works
```

**Why this works:**
- ✅ Zero data loss (both locations have passwords)
- ✅ Zero downtime (read/write always succeeds)
- ✅ Zero user friction (migration is automatic)
- ✅ Rollback-safe (can revert at any time)

---

## 📊 Migration Phases

| Phase | Timeline | Goal | User Impact |
|-------|----------|------|-------------|
| **1. Deploy Hybrid** | Weeks 1-2 | Add encrypted storage | None (transparent) |
| **2. Background Migration** | Weeks 3-4 | 30-40% passive migration | None (automatic) |
| **3. Active Push** | Months 2-3 | 80% completion | Optional migration UI |
| **4. Keychain Removal** | Month 6+ | 95%+ migrated, remove keychain | None (already migrated) |

---

## 🔒 Security Architecture

### Zero-Knowledge Encryption

```
User Password (login)
    ↓ PBKDF2 (600k iterations)
User-Derived Key
    ↓ Encrypt
Master Key (random 256-bit)
    ↓ Store in Turso (encrypted)

Master Key
    ↓ AES-256-GCM
Database Password (encrypted)
    ↓ Store in Turso
```

**Server never sees:**
- User login password
- User-derived key
- Master key (plaintext)
- Database passwords (plaintext)

**Encryption standards:**
- PBKDF2-SHA256 with 600,000 iterations (OWASP 2023)
- AES-256-GCM authenticated encryption
- Unique IV and salt per operation

---

## 🛠️ Technical Components

### Core Services

```go
// services/password_manager.go
type PasswordManager struct {
    credentialService *CredentialService      // OS keychain (legacy)
    credentialStore   *turso.CredentialStore  // Encrypted DB (new)
    connectionStore   *turso.ConnectionStore
    logger            *logrus.Logger
}

// Dual-read with fallback
func (pm *PasswordManager) GetPassword(
    ctx context.Context,
    userID, connectionID string,
    masterKey []byte,
) (string, error)

// Dual-write with opportunistic migration
func (pm *PasswordManager) StorePassword(
    ctx context.Context,
    userID, connectionID, password string,
    masterKey []byte,
) error
```

### Database Tables

```sql
-- User encrypted master keys
CREATE TABLE user_master_keys (
    user_id TEXT PRIMARY KEY,
    encrypted_master_key TEXT NOT NULL,
    key_iv TEXT NOT NULL,
    key_auth_tag TEXT NOT NULL,
    pbkdf2_salt TEXT NOT NULL,
    pbkdf2_iterations INTEGER NOT NULL DEFAULT 600000,
    version INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Encrypted database passwords
CREATE TABLE encrypted_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    encrypted_password TEXT NOT NULL,
    password_iv TEXT NOT NULL,
    password_auth_tag TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(user_id, connection_id)
);

-- Migration tracking
ALTER TABLE connection_templates
ADD COLUMN password_migration_status TEXT DEFAULT 'not_migrated'
CHECK (password_migration_status IN ('not_migrated', 'migrated', 'no_password'));

-- Migration audit log
CREATE TABLE password_migration_log (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,
    migration_type TEXT NOT NULL,
    source TEXT NOT NULL,
    status TEXT NOT NULL,
    error_message TEXT,
    migrated_at INTEGER NOT NULL
);
```

---

## 📋 Quick Commands

### Migration Monitoring

```sql
-- Migration completion rate
SELECT
    COUNT(CASE WHEN password_migration_status = 'migrated' THEN 1 END) * 100.0 / COUNT(*) as completion_rate
FROM connection_templates
WHERE deleted_at IS NULL;

-- Unmigrated connections by user
SELECT user_id, COUNT(*) as unmigrated_count
FROM connection_templates
WHERE password_migration_status = 'not_migrated' AND deleted_at IS NULL
GROUP BY user_id
ORDER BY unmigrated_count DESC;
```

### Emergency Rollback

```bash
# Level 1: Feature flag disable (< 1 minute)
export ENABLE_ENCRYPTED_STORAGE=false
systemctl restart howlerops

# Level 2: Code revert (5 minutes)
git checkout rollback/password-migration
./deploy.sh production
```

---

## 📞 Support

### Common User Questions

**Q: Do I need to do anything?**
> No action required. Your passwords will migrate automatically as you use connections.

**Q: Are my passwords safe?**
> Yes. We use industry-standard end-to-end encryption (AES-256-GCM).

**Q: Can I use my connections on multiple devices now?**
> Yes! After migration, your passwords sync across all your devices automatically.

---

## 📚 Related Documentation

- [Deployment Guide](../../deployment/migration-deployment.md)
- [Security Best Practices](../README.md)
- [Keychain Integration](../keychain/README.md)

---

**Questions?** Contact the Security Team or see the full documentation above.
