-- Migration 008: Add encrypted password storage
-- Date: 2024-11-15
-- Purpose: Enable secure storage of database passwords in Turso using master key encryption
-- Security: Zero-knowledge architecture - server never sees plaintext passwords

-- =============================================================================
-- ENCRYPTED PASSWORD STORAGE TABLES
-- =============================================================================

-- User Master Keys
-- Stores encrypted master keys (encrypted with user's password-derived key)
CREATE TABLE IF NOT EXISTS user_master_keys (
    user_id TEXT PRIMARY KEY,
    encrypted_master_key TEXT NOT NULL,  -- Base64-encoded ciphertext
    key_iv TEXT NOT NULL,                -- Base64-encoded IV/nonce for master key encryption
    key_auth_tag TEXT NOT NULL,          -- Base64-encoded GCM auth tag
    pbkdf2_salt TEXT NOT NULL,           -- Base64-encoded PBKDF2 salt
    pbkdf2_iterations INTEGER NOT NULL DEFAULT 600000,  -- OWASP 2023 recommendation
    version INTEGER NOT NULL DEFAULT 1,  -- For future key rotation
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Encrypted Credentials
-- Stores database passwords encrypted with user's master key
CREATE TABLE IF NOT EXISTS encrypted_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    connection_id TEXT NOT NULL,         -- References connection_templates.id
    encrypted_password TEXT NOT NULL,    -- Base64-encoded ciphertext
    password_iv TEXT NOT NULL,           -- Base64-encoded IV/nonce
    password_auth_tag TEXT NOT NULL,     -- Base64-encoded GCM auth tag
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (connection_id) REFERENCES connection_templates(id) ON DELETE CASCADE,
    UNIQUE(user_id, connection_id)       -- One credential per user per connection
);

-- =============================================================================
-- INDEXES FOR ENCRYPTED PASSWORD TABLES
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_encrypted_creds_user_id ON encrypted_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_creds_connection_id ON encrypted_credentials(connection_id);
CREATE INDEX IF NOT EXISTS idx_encrypted_creds_updated ON encrypted_credentials(updated_at);

-- =============================================================================
-- MIGRATION NOTES
-- =============================================================================
-- 1. Master keys are encrypted with PBKDF2-derived key from user's login password
-- 2. Database passwords are encrypted with the master key (AES-256-GCM)
-- 3. Server never sees plaintext passwords (zero-knowledge architecture)
-- 4. PBKDF2 uses 600,000 iterations (OWASP 2023 recommendation)
-- 5. All encrypted data uses unique IVs and includes GCM auth tags
-- 6. Master key can be rotated by updating user_master_keys with new version
-- 7. Password change requires decrypting old master key and re-encrypting with new password-derived key
