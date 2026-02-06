-- Turso Database Schema for Howlerops Individual Tier
-- Migration: 001 - Initialize Turso storage tables

-- =============================================================================
-- AUTH TABLES
-- =============================================================================

-- Users (auth) - passwords are hashed with bcrypt
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT DEFAULT 'user',
    active BOOLEAN DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    last_login INTEGER,
    metadata TEXT  -- JSON
);

-- Sessions
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    refresh_token TEXT UNIQUE NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    last_access INTEGER NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    active BOOLEAN DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Login attempts (brute force protection)
CREATE TABLE IF NOT EXISTS login_attempts (
    id TEXT PRIMARY KEY,
    ip TEXT NOT NULL,
    username TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    success BOOLEAN NOT NULL
);

-- Email verification tokens
CREATE TABLE IF NOT EXISTS email_verification_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used BOOLEAN DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Password reset tokens
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    token TEXT UNIQUE NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used BOOLEAN DEFAULT 0,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- License keys (Individual tier)
CREATE TABLE IF NOT EXISTS license_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    key TEXT UNIQUE NOT NULL,
    tier TEXT NOT NULL,  -- 'individual', 'team'
    status TEXT NOT NULL,  -- 'active', 'expired', 'cancelled'
    created_at INTEGER NOT NULL,
    expires_at INTEGER,
    stripe_subscription_id TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- =============================================================================
-- INDEXES FOR AUTH TABLES
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(active);

CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_time ON login_attempts(ip, timestamp);
CREATE INDEX IF NOT EXISTS idx_login_attempts_username_time ON login_attempts(username, timestamp);

CREATE INDEX IF NOT EXISTS idx_email_verification_user ON email_verification_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_email_verification_token ON email_verification_tokens(token);

CREATE INDEX IF NOT EXISTS idx_password_reset_user ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_token ON password_reset_tokens(token);

CREATE INDEX IF NOT EXISTS idx_license_keys_user_id ON license_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_license_keys_status ON license_keys(status);

-- =============================================================================
-- APP DATA SYNC TABLES (SANITIZED - NO PASSWORDS!)
-- =============================================================================

-- Connection templates (NO passwords stored in Turso!)
-- Passwords stay local only. This is for connection metadata sync.
CREATE TABLE IF NOT EXISTS connection_templates (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,  -- 'postgres', 'mysql', 'sqlite', etc
    host TEXT,
    port INTEGER,
    database_name TEXT,
    username TEXT,
    ssl_config TEXT,  -- JSON
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sync_version INTEGER DEFAULT 1,
    deleted_at INTEGER,  -- Soft delete for sync
    metadata TEXT,  -- JSON (environments, etc)
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Saved queries
CREATE TABLE IF NOT EXISTS saved_queries_sync (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    query TEXT NOT NULL,
    description TEXT,
    connection_id TEXT,
    folder TEXT,
    tags TEXT,  -- JSON array
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sync_version INTEGER DEFAULT 1,
    deleted_at INTEGER,  -- Soft delete for sync
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Query history (sanitized queries only - NO data literals!)
CREATE TABLE IF NOT EXISTS query_history_sync (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    query_sanitized TEXT NOT NULL,  -- Sanitized SQL with placeholders
    connection_id TEXT,
    executed_at INTEGER NOT NULL,
    duration_ms INTEGER,
    rows_returned INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    sync_version INTEGER DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Sync metadata (track last sync per user/device)
CREATE TABLE IF NOT EXISTS sync_metadata (
    user_id TEXT PRIMARY KEY,
    last_sync_at INTEGER NOT NULL,
    device_id TEXT,
    client_version TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- =============================================================================
-- INDEXES FOR APP DATA TABLES
-- =============================================================================

CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connection_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_connections_updated ON connection_templates(updated_at);
CREATE INDEX IF NOT EXISTS idx_connections_deleted ON connection_templates(deleted_at);
CREATE INDEX IF NOT EXISTS idx_connections_type ON connection_templates(type);

CREATE INDEX IF NOT EXISTS idx_queries_user_id ON saved_queries_sync(user_id);
CREATE INDEX IF NOT EXISTS idx_queries_updated ON saved_queries_sync(updated_at);
CREATE INDEX IF NOT EXISTS idx_queries_deleted ON saved_queries_sync(deleted_at);
CREATE INDEX IF NOT EXISTS idx_queries_connection ON saved_queries_sync(connection_id);
CREATE INDEX IF NOT EXISTS idx_queries_folder ON saved_queries_sync(folder);

CREATE INDEX IF NOT EXISTS idx_history_user_id ON query_history_sync(user_id);
CREATE INDEX IF NOT EXISTS idx_history_executed ON query_history_sync(executed_at);
CREATE INDEX IF NOT EXISTS idx_history_connection ON query_history_sync(connection_id);
CREATE INDEX IF NOT EXISTS idx_history_success ON query_history_sync(success);

-- =============================================================================
-- NOTES
-- =============================================================================
-- 1. All timestamps are stored as INTEGER (Unix epoch seconds)
-- 2. JSON fields are stored as TEXT and must be marshaled/unmarshaled
-- 3. Passwords are NEVER stored in connection_templates - only locally
-- 4. Query history is sanitized to remove data literals before sync
-- 5. Soft deletes (deleted_at) allow conflict resolution during sync
-- 6. sync_version enables optimistic locking for conflict detection
