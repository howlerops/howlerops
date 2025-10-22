-- Migration: 003 - Migrate passwords to connection_secrets
-- This migration moves existing password_encrypted data to the new connection_secrets table

-- First, we need to create a temporary table to store the migration data
-- since we can't directly migrate encrypted data without the encryption key

-- Create a temporary table to track migration status
CREATE TABLE IF NOT EXISTS migration_status (
    migration_name TEXT PRIMARY KEY,
    status TEXT NOT NULL,  -- 'pending', 'in_progress', 'completed', 'failed'
    started_at INTEGER,
    completed_at INTEGER,
    error_message TEXT
);

-- Insert migration record
INSERT OR IGNORE INTO migration_status (migration_name, status) 
VALUES ('003_migrate_passwords_to_secrets', 'pending');

-- Note: The actual migration of password_encrypted to connection_secrets
-- will be handled by the Go application at runtime, not in SQL
-- This is because we need to:
-- 1. Decrypt the existing password_encrypted data
-- 2. Re-encrypt it with the new AES-256-GCM encryption
-- 3. Store it in the connection_secrets table
-- 4. Verify the migration was successful
-- 5. Only then remove the password_encrypted column

-- The Go application will:
-- 1. Check migration_status for this migration
-- 2. If status is 'pending', start the migration
-- 3. Set status to 'in_progress'
-- 4. Migrate each connection's password_encrypted to connection_secrets
-- 5. Set status to 'completed' on success or 'failed' on error
-- 6. Only after successful migration, remove password_encrypted column

-- For now, we'll leave the password_encrypted column in place
-- It will be removed in a future migration after the Go migration is complete
