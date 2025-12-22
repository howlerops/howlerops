-- Migration 009: Add Organization Envelope Key (OEK) system for shared credentials
-- Date: 2024-12-22
-- Purpose: Enable secure sharing of database credentials within organizations
-- Security: Organization-wide encryption with per-member key wrapping (envelope encryption)
--
-- Architecture:
-- 1. Each organization has an Organization Envelope Key (OEK)
-- 2. OEK is encrypted separately for each organization member using their master key
-- 3. Shared credentials are encrypted with the OEK
-- 4. Members decrypt their wrapped OEK copy, then use it to decrypt shared credentials
-- 5. Complete audit trail for all credential access operations

-- =============================================================================
-- ORGANIZATION ENVELOPE KEYS
-- =============================================================================

-- Organization Envelope Keys (OEK)
-- Stores the OEK encrypted separately for each member of the organization
-- Each member can decrypt their copy using their own master key
CREATE TABLE IF NOT EXISTS organization_envelope_keys (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,      -- References organizations.id
    user_id TEXT NOT NULL,              -- References users.id (member who can decrypt this OEK copy)
    encrypted_oek TEXT NOT NULL,        -- Base64-encoded ciphertext of the OEK (encrypted with user's master key)
    oek_iv TEXT NOT NULL,               -- Base64-encoded IV/nonce for OEK encryption
    oek_auth_tag TEXT NOT NULL,         -- Base64-encoded GCM auth tag for OEK encryption
    key_version INTEGER NOT NULL DEFAULT 1,  -- For OEK rotation support
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(organization_id, user_id, key_version)  -- One OEK copy per member per version
);

-- =============================================================================
-- SHARED CREDENTIALS
-- =============================================================================

-- Shared Credentials
-- Database passwords shared within an organization, encrypted with the OEK
CREATE TABLE IF NOT EXISTS shared_credentials (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,        -- References connection_templates.id
    organization_id TEXT NOT NULL,      -- References organizations.id
    encrypted_password TEXT NOT NULL,   -- Base64-encoded ciphertext (encrypted with OEK)
    password_iv TEXT NOT NULL,          -- Base64-encoded IV/nonce for password encryption
    password_auth_tag TEXT NOT NULL,    -- Base64-encoded GCM auth tag for password encryption
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,           -- References users.id (who created/shared this credential)
    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE(connection_id, organization_id)  -- One shared credential per connection per organization
);

-- =============================================================================
-- CREDENTIAL ACCESS AUDIT LOG
-- =============================================================================

-- Credential Access Log
-- Complete audit trail for all operations on shared credentials
CREATE TABLE IF NOT EXISTS credential_access_log (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,        -- References connection_templates.id
    organization_id TEXT NOT NULL,      -- References organizations.id
    user_id TEXT NOT NULL,              -- References users.id (who performed the action)
    action TEXT NOT NULL,               -- Action type: 'decrypt', 'share', 'unshare', 'rotate'
    timestamp INTEGER NOT NULL,         -- Unix timestamp of the action
    ip_address TEXT,                    -- IP address of the request
    success BOOLEAN NOT NULL,           -- Whether the action succeeded
    error_message TEXT,                 -- Error details if success = false
    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- =============================================================================
-- ALTER EXISTING TABLES
-- =============================================================================

-- Add encryption type to connection_templates
-- 'personal' = encrypted with user's master key (existing behavior)
-- 'shared' = encrypted with organization's OEK (new capability)
ALTER TABLE connection_templates ADD COLUMN encryption_type TEXT DEFAULT 'personal'
    CHECK (encryption_type IN ('personal', 'shared'));

-- =============================================================================
-- INDEXES FOR PERFORMANCE
-- =============================================================================

-- Organization Envelope Keys indexes
CREATE INDEX IF NOT EXISTS idx_oek_organization_id ON organization_envelope_keys(organization_id);
CREATE INDEX IF NOT EXISTS idx_oek_user_id ON organization_envelope_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_oek_org_user_version ON organization_envelope_keys(organization_id, user_id, key_version);
CREATE INDEX IF NOT EXISTS idx_oek_updated ON organization_envelope_keys(updated_at);

-- Shared Credentials indexes
CREATE INDEX IF NOT EXISTS idx_shared_creds_connection_id ON shared_credentials(connection_id);
CREATE INDEX IF NOT EXISTS idx_shared_creds_organization_id ON shared_credentials(organization_id);
CREATE INDEX IF NOT EXISTS idx_shared_creds_created_by ON shared_credentials(created_by);
CREATE INDEX IF NOT EXISTS idx_shared_creds_org_connection ON shared_credentials(organization_id, connection_id);
CREATE INDEX IF NOT EXISTS idx_shared_creds_updated ON shared_credentials(updated_at);

-- Credential Access Log indexes
CREATE INDEX IF NOT EXISTS idx_cred_log_connection_id ON credential_access_log(connection_id);
CREATE INDEX IF NOT EXISTS idx_cred_log_organization_id ON credential_access_log(organization_id);
CREATE INDEX IF NOT EXISTS idx_cred_log_user_id ON credential_access_log(user_id);
CREATE INDEX IF NOT EXISTS idx_cred_log_timestamp ON credential_access_log(timestamp);
CREATE INDEX IF NOT EXISTS idx_cred_log_action ON credential_access_log(action);
CREATE INDEX IF NOT EXISTS idx_cred_log_success ON credential_access_log(success);
CREATE INDEX IF NOT EXISTS idx_cred_log_org_user_time ON credential_access_log(organization_id, user_id, timestamp);

-- Connection templates encryption type index
CREATE INDEX IF NOT EXISTS idx_connections_encryption_type ON connection_templates(encryption_type);
CREATE INDEX IF NOT EXISTS idx_connections_org_encryption ON connection_templates(organization_id, encryption_type);

-- =============================================================================
-- MIGRATION NOTES
-- =============================================================================
-- 1. Organization Envelope Key (OEK) Architecture:
--    - Each organization has ONE OEK that encrypts all shared credentials
--    - The OEK is wrapped (encrypted) separately for each organization member
--    - Members use their master key to unwrap their OEK copy
--    - Members use the unwrapped OEK to decrypt shared credentials
--
-- 2. Security Model:
--    - OEK is encrypted with each member's master key (AES-256-GCM)
--    - Shared passwords are encrypted with the OEK (AES-256-GCM)
--    - Server never sees plaintext OEK or passwords (zero-knowledge)
--    - All encrypted data uses unique IVs and includes GCM auth tags
--
-- 3. Key Rotation:
--    - OEK can be rotated by incrementing key_version
--    - New members receive only the current key_version
--    - Old versions can be maintained for decrypting historical data
--    - Credential rotation requires re-encrypting with new OEK
--
-- 4. Member Management:
--    - Adding member: Create new organization_envelope_keys row with OEK wrapped by their master key
--    - Removing member: Delete their organization_envelope_keys row(s)
--    - Member should also be removed from organization_members table
--
-- 5. Credential Lifecycle:
--    - Share: Create shared_credentials row, encrypt password with OEK
--    - Unshare: Delete shared_credentials row
--    - Rotate: Update encrypted_password with new ciphertext
--    - Access: Member decrypts their OEK copy, then uses OEK to decrypt password
--
-- 6. Audit Requirements:
--    - Log ALL credential operations (decrypt, share, unshare, rotate)
--    - Include IP address for security monitoring
--    - Track success/failure for anomaly detection
--    - Retain logs for compliance (consider TTL/archival strategy)
--
-- 7. Migration Path:
--    - Existing personal credentials remain unchanged (encryption_type = 'personal')
--    - New shared credentials use encryption_type = 'shared'
--    - Connections can be migrated from personal to shared via re-encryption
--
-- 8. Query Patterns:
--    - Find user's accessible OEKs: WHERE user_id = ? AND key_version = (SELECT MAX(key_version) ...)
--    - Find org's shared credentials: WHERE organization_id = ? AND encryption_type = 'shared'
--    - Audit user activity: WHERE user_id = ? ORDER BY timestamp DESC
--    - Detect anomalies: WHERE success = 0 GROUP BY user_id, ip_address
