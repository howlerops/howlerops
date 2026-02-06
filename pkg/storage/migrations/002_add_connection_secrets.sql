-- Migration: 002 - Add connection_secrets table
-- This migration adds the connection_secrets table for storing encrypted credentials

-- Connection secrets table for encrypted credentials
CREATE TABLE IF NOT EXISTS connection_secrets (
    connection_id TEXT NOT NULL,
    secret_type TEXT NOT NULL,      -- 'db_password', 'ssh_password', 'ssh_private_key'
    ciphertext BLOB NOT NULL,        -- AES-256-GCM encrypted
    nonce BLOB NOT NULL,             -- GCM nonce (96 bits)
    salt BLOB,                       -- Argon2id salt (for key derivation)
    key_version INTEGER DEFAULT 1,   -- Support key rotation
    updated_at INTEGER NOT NULL,
    updated_by TEXT NOT NULL,
    team_id TEXT,                    -- NULL for local-only, set for team-shared
    PRIMARY KEY (connection_id, secret_type),
    FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
);

-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_connection_secrets_connection_id ON connection_secrets(connection_id);
CREATE INDEX IF NOT EXISTS idx_connection_secrets_team_id ON connection_secrets(team_id);
CREATE INDEX IF NOT EXISTS idx_connection_secrets_updated_at ON connection_secrets(updated_at);

-- Add ssh_config and vpc_config columns to connections table
-- These will store non-sensitive SSH and VPC configuration as JSON
ALTER TABLE connections ADD COLUMN ssh_config TEXT;  -- JSON for SSH tunnel config
ALTER TABLE connections ADD COLUMN vpc_config TEXT;   -- JSON for VPC config

-- Add indexes for the new columns
CREATE INDEX IF NOT EXISTS idx_connections_ssh_config ON connections(ssh_config);
CREATE INDEX IF NOT EXISTS idx_connections_vpc_config ON connections(vpc_config);
