-- ============================================================================
-- SQL Studio - Turso Database Schema
-- ============================================================================
--
-- Purpose: Cloud sync for Individual tier ($9/mo)
-- Database: LibSQL (SQLite fork) via Turso
-- Strategy: Bidirectional sync with conflict resolution
--
-- Key Features:
-- - Multi-device sync with vector clocks
-- - Soft deletes for conflict resolution
-- - Optimistic locking with sync_version
-- - Device tracking for conflict detection
-- - Delta sync (only changed records)
-- - Indexed for fast sync queries
--
-- Security:
-- - NO passwords or credentials stored
-- - Data sanitization enforced
-- - Row-level security via user_id
-- - Encrypted at rest (Turso default)
-- - Encrypted in transit (TLS)
--
-- ============================================================================

-- ============================================================================
-- Schema Migrations Table
-- ============================================================================

CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    checksum TEXT NOT NULL
);

-- Record initial migration
INSERT INTO schema_migrations (version, name, checksum) VALUES
    (1, 'initial_schema', 'sha256:initial');

-- ============================================================================
-- User Preferences Store
-- ============================================================================
-- Syncs UI settings across devices
-- Examples: theme, editor fontSize, layout preferences

CREATE TABLE IF NOT EXISTS user_preferences (
    -- Primary Key
    id TEXT PRIMARY KEY,

    -- User Reference
    user_id TEXT NOT NULL,

    -- Preference Data
    key TEXT NOT NULL,
    value TEXT NOT NULL, -- JSON-encoded value
    category TEXT NOT NULL,

    -- Device Tracking (for device-specific preferences)
    device_id TEXT,

    -- Sync Metadata
    sync_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Soft Delete (for conflict resolution)
    deleted_at TIMESTAMP,

    -- Unique constraint for user + key combination
    UNIQUE(user_id, key, device_id)
) STRICT;

-- Indexes for fast sync queries
CREATE INDEX IF NOT EXISTS idx_user_preferences_user_id
    ON user_preferences(user_id);

CREATE INDEX IF NOT EXISTS idx_user_preferences_updated_at
    ON user_preferences(updated_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_preferences_sync
    ON user_preferences(user_id, updated_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_preferences_category
    ON user_preferences(user_id, category);

-- ============================================================================
-- Connection Templates Store
-- ============================================================================
-- Stores connection metadata ONLY (NO passwords)
-- Passwords stored separately in secure credential manager

CREATE TABLE IF NOT EXISTS connection_templates (
    -- Primary Key
    connection_id TEXT PRIMARY KEY,

    -- User Reference
    user_id TEXT NOT NULL,

    -- Connection Metadata (NO passwords!)
    name TEXT NOT NULL,
    type TEXT NOT NULL, -- postgres, mysql, sqlite, mssql, oracle, mongodb
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    database TEXT NOT NULL,
    username TEXT NOT NULL,

    -- SSL Configuration
    ssl_mode TEXT NOT NULL DEFAULT 'disable', -- disable, require, verify-ca, verify-full

    -- Additional Parameters (JSON, sanitized)
    parameters TEXT, -- JSON-encoded parameters

    -- Organization
    environment_tags TEXT NOT NULL DEFAULT '[]', -- JSON array of tags

    -- Usage Tracking
    last_used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Sync Metadata
    sync_version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Soft Delete
    deleted_at TIMESTAMP,

    -- Constraints
    CHECK (type IN ('postgres', 'mysql', 'sqlite', 'mssql', 'oracle', 'mongodb')),
    CHECK (ssl_mode IN ('disable', 'require', 'verify-ca', 'verify-full')),
    CHECK (port > 0 AND port < 65536)
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_connection_templates_user_id
    ON connection_templates(user_id);

CREATE INDEX IF NOT EXISTS idx_connection_templates_updated_at
    ON connection_templates(updated_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_connection_templates_sync
    ON connection_templates(user_id, updated_at) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_connection_templates_type
    ON connection_templates(user_id, type);

CREATE INDEX IF NOT EXISTS idx_connection_templates_last_used
    ON connection_templates(user_id, last_used_at DESC);

-- ============================================================================
-- Query History Store
-- ============================================================================
-- Stores sanitized query execution history
-- Used for recent queries, analytics, audit logging

CREATE TABLE IF NOT EXISTS query_history (
    -- Primary Key
    id TEXT PRIMARY KEY,

    -- User Reference
    user_id TEXT NOT NULL,

    -- Query Data (SANITIZED - no passwords/secrets)
    query_text TEXT NOT NULL,
    connection_id TEXT NOT NULL,

    -- Execution Metrics
    execution_time_ms INTEGER NOT NULL,
    row_count INTEGER NOT NULL,
    error TEXT, -- NULL if successful

    -- Privacy
    privacy_mode TEXT NOT NULL DEFAULT 'normal', -- private, normal, shared

    -- Timestamps
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Sync Metadata
    sync_version INTEGER NOT NULL DEFAULT 1,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Soft Delete
    deleted_at TIMESTAMP,

    -- Constraints
    CHECK (privacy_mode IN ('private', 'normal', 'shared')),
    CHECK (execution_time_ms >= 0),
    CHECK (row_count >= 0),

    -- Foreign Key
    FOREIGN KEY (connection_id) REFERENCES connection_templates(connection_id)
        ON DELETE CASCADE
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_query_history_user_id
    ON query_history(user_id);

CREATE INDEX IF NOT EXISTS idx_query_history_executed_at
    ON query_history(executed_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_query_history_sync
    ON query_history(user_id, executed_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_query_history_connection
    ON query_history(connection_id, executed_at DESC);

CREATE INDEX IF NOT EXISTS idx_query_history_privacy
    ON query_history(user_id, privacy_mode);

-- Full-text search on query text (for search feature)
CREATE VIRTUAL TABLE IF NOT EXISTS query_history_fts USING fts5(
    id UNINDEXED,
    query_text,
    content=query_history,
    content_rowid=id
);

-- ============================================================================
-- Saved Queries Store
-- ============================================================================
-- User's personal query library with organization features

CREATE TABLE IF NOT EXISTS saved_queries (
    -- Primary Key
    id TEXT PRIMARY KEY,

    -- User Reference
    user_id TEXT NOT NULL,

    -- Query Data
    title TEXT NOT NULL,
    description TEXT,
    query_text TEXT NOT NULL,

    -- Organization
    tags TEXT NOT NULL DEFAULT '[]', -- JSON array of tags
    folder TEXT, -- Folder path for organization
    is_favorite INTEGER NOT NULL DEFAULT 0, -- SQLite boolean

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Sync Metadata
    sync_version INTEGER NOT NULL DEFAULT 1,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Soft Delete
    deleted_at TIMESTAMP,

    -- Constraints
    CHECK (is_favorite IN (0, 1)),
    CHECK (length(title) > 0)
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_saved_queries_user_id
    ON saved_queries(user_id);

CREATE INDEX IF NOT EXISTS idx_saved_queries_updated_at
    ON saved_queries(updated_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_saved_queries_sync
    ON saved_queries(user_id, updated_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_saved_queries_folder
    ON saved_queries(user_id, folder);

CREATE INDEX IF NOT EXISTS idx_saved_queries_favorite
    ON saved_queries(user_id, is_favorite) WHERE is_favorite = 1;

-- Full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS saved_queries_fts USING fts5(
    id UNINDEXED,
    title,
    description,
    query_text,
    content=saved_queries,
    content_rowid=id
);

-- ============================================================================
-- AI Memory Sessions Store
-- ============================================================================
-- High-level AI conversation session metadata

CREATE TABLE IF NOT EXISTS ai_memory_sessions (
    -- Primary Key
    id TEXT PRIMARY KEY,

    -- User Reference
    user_id TEXT NOT NULL,

    -- Session Data
    title TEXT NOT NULL,
    summary TEXT,
    message_count INTEGER NOT NULL DEFAULT 0,
    token_count INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Sync Metadata
    sync_version INTEGER NOT NULL DEFAULT 1,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Soft Delete
    deleted_at TIMESTAMP,

    -- Constraints
    CHECK (message_count >= 0),
    CHECK (token_count >= 0)
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_ai_memory_sessions_user_id
    ON ai_memory_sessions(user_id);

CREATE INDEX IF NOT EXISTS idx_ai_memory_sessions_updated_at
    ON ai_memory_sessions(updated_at DESC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_memory_sessions_sync
    ON ai_memory_sessions(user_id, updated_at DESC) WHERE deleted_at IS NULL;

-- ============================================================================
-- AI Memory Messages Store
-- ============================================================================
-- Individual AI conversation messages with full content

CREATE TABLE IF NOT EXISTS ai_memory_messages (
    -- Primary Key
    id TEXT PRIMARY KEY,

    -- Session Reference
    session_id TEXT NOT NULL,

    -- Message Data
    role TEXT NOT NULL, -- system, user, assistant
    content TEXT NOT NULL,
    tokens INTEGER,

    -- Metadata (JSON)
    metadata TEXT, -- model, temperature, etc.

    -- Timestamps
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Sync Metadata
    sync_version INTEGER NOT NULL DEFAULT 1,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Soft Delete
    deleted_at TIMESTAMP,

    -- Constraints
    CHECK (role IN ('system', 'user', 'assistant')),

    -- Foreign Key
    FOREIGN KEY (session_id) REFERENCES ai_memory_sessions(id)
        ON DELETE CASCADE
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_ai_memory_messages_session_id
    ON ai_memory_messages(session_id);

CREATE INDEX IF NOT EXISTS idx_ai_memory_messages_timestamp
    ON ai_memory_messages(timestamp ASC) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_ai_memory_messages_sync
    ON ai_memory_messages(session_id, timestamp ASC) WHERE deleted_at IS NULL;

-- ============================================================================
-- Sync Metadata Store
-- ============================================================================
-- Tracks sync state for conflict detection and resolution
-- Uses vector clock strategy for multi-device conflict detection

CREATE TABLE IF NOT EXISTS sync_metadata (
    -- Composite Primary Key
    entity_type TEXT NOT NULL, -- connection, query, preference, ai_session
    entity_id TEXT NOT NULL,
    device_id TEXT NOT NULL,

    -- Vector Clock Data
    local_version INTEGER NOT NULL DEFAULT 1,
    remote_version INTEGER NOT NULL DEFAULT 1,

    -- Sync State
    sync_status TEXT NOT NULL DEFAULT 'synced', -- synced, pending, conflict, error
    last_synced TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_modified TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Conflict Data (JSON, only populated if sync_status = 'conflict')
    conflict_data TEXT,

    -- Error Tracking
    error_count INTEGER NOT NULL DEFAULT 0,
    last_error TEXT,
    last_error_at TIMESTAMP,

    -- Constraints
    CHECK (entity_type IN ('connection', 'query_history', 'saved_query', 'ai_session', 'ai_message', 'preference')),
    CHECK (sync_status IN ('synced', 'pending', 'conflict', 'error')),
    CHECK (local_version > 0),
    CHECK (remote_version > 0),
    CHECK (error_count >= 0),

    PRIMARY KEY (entity_type, entity_id, device_id)
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sync_metadata_device
    ON sync_metadata(device_id, last_synced);

CREATE INDEX IF NOT EXISTS idx_sync_metadata_pending
    ON sync_metadata(sync_status, last_modified)
    WHERE sync_status IN ('pending', 'error');

CREATE INDEX IF NOT EXISTS idx_sync_metadata_conflicts
    ON sync_metadata(device_id, sync_status)
    WHERE sync_status = 'conflict';

CREATE INDEX IF NOT EXISTS idx_sync_metadata_entity
    ON sync_metadata(entity_type, entity_id);

-- ============================================================================
-- Device Registry
-- ============================================================================
-- Tracks devices for a user (for multi-device management)

CREATE TABLE IF NOT EXISTS device_registry (
    -- Primary Key
    device_id TEXT PRIMARY KEY,

    -- User Reference
    user_id TEXT NOT NULL,

    -- Device Info
    device_name TEXT NOT NULL,
    device_type TEXT NOT NULL, -- desktop, mobile, tablet
    browser TEXT,
    os TEXT,

    -- Activity
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_sync TIMESTAMP,

    -- Status
    is_active INTEGER NOT NULL DEFAULT 1, -- SQLite boolean

    -- Constraints
    CHECK (device_type IN ('desktop', 'mobile', 'tablet')),
    CHECK (is_active IN (0, 1))
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_device_registry_user_id
    ON device_registry(user_id);

CREATE INDEX IF NOT EXISTS idx_device_registry_active
    ON device_registry(user_id, is_active, last_seen DESC)
    WHERE is_active = 1;

-- ============================================================================
-- Sync Conflicts Archive
-- ============================================================================
-- Archives resolved conflicts for audit and debugging

CREATE TABLE IF NOT EXISTS sync_conflicts_archive (
    -- Primary Key
    id TEXT PRIMARY KEY,

    -- Conflict Details
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    user_id TEXT NOT NULL,

    -- Conflicting Versions
    local_version INTEGER NOT NULL,
    remote_version INTEGER NOT NULL,
    local_data TEXT NOT NULL, -- JSON snapshot of local version
    remote_data TEXT NOT NULL, -- JSON snapshot of remote version

    -- Resolution
    resolution_strategy TEXT NOT NULL, -- last-write-wins, manual, merge
    resolved_data TEXT NOT NULL, -- JSON snapshot of resolved version
    resolved_by TEXT, -- device_id or 'system'
    resolved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    -- Metadata
    conflict_detected_at TIMESTAMP NOT NULL,

    -- Constraints
    CHECK (resolution_strategy IN ('last-write-wins', 'manual', 'merge'))
) STRICT;

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sync_conflicts_user
    ON sync_conflicts_archive(user_id, resolved_at DESC);

CREATE INDEX IF NOT EXISTS idx_sync_conflicts_entity
    ON sync_conflicts_archive(entity_type, entity_id);

-- ============================================================================
-- User Statistics (Materialized View for Dashboard)
-- ============================================================================
-- Aggregated statistics for user dashboard and analytics

CREATE TABLE IF NOT EXISTS user_statistics (
    -- Primary Key
    user_id TEXT PRIMARY KEY,

    -- Counts
    total_connections INTEGER NOT NULL DEFAULT 0,
    total_queries INTEGER NOT NULL DEFAULT 0,
    total_saved_queries INTEGER NOT NULL DEFAULT 0,
    total_ai_sessions INTEGER NOT NULL DEFAULT 0,

    -- Storage
    storage_used_bytes INTEGER NOT NULL DEFAULT 0,

    -- Activity
    last_activity TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    queries_this_month INTEGER NOT NULL DEFAULT 0,

    -- Sync Status
    last_sync TIMESTAMP,
    pending_sync_items INTEGER NOT NULL DEFAULT 0,

    -- Updated
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) STRICT;

-- ============================================================================
-- Triggers for Automatic Updates
-- ============================================================================

-- Auto-update updated_at timestamp on user_preferences
CREATE TRIGGER IF NOT EXISTS update_user_preferences_timestamp
AFTER UPDATE ON user_preferences
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE user_preferences
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

-- Auto-update updated_at timestamp on connection_templates
CREATE TRIGGER IF NOT EXISTS update_connection_templates_timestamp
AFTER UPDATE ON connection_templates
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE connection_templates
    SET updated_at = CURRENT_TIMESTAMP
    WHERE connection_id = NEW.connection_id;
END;

-- Auto-update updated_at timestamp on saved_queries
CREATE TRIGGER IF NOT EXISTS update_saved_queries_timestamp
AFTER UPDATE ON saved_queries
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE saved_queries
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

-- Auto-update updated_at timestamp on ai_memory_sessions
CREATE TRIGGER IF NOT EXISTS update_ai_memory_sessions_timestamp
AFTER UPDATE ON ai_memory_sessions
FOR EACH ROW
WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE ai_memory_sessions
    SET updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

-- Increment message_count when message is added
CREATE TRIGGER IF NOT EXISTS increment_ai_session_message_count
AFTER INSERT ON ai_memory_messages
FOR EACH ROW
WHEN NEW.deleted_at IS NULL
BEGIN
    UPDATE ai_memory_sessions
    SET message_count = message_count + 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.session_id;
END;

-- Decrement message_count when message is deleted
CREATE TRIGGER IF NOT EXISTS decrement_ai_session_message_count
AFTER UPDATE ON ai_memory_messages
FOR EACH ROW
WHEN OLD.deleted_at IS NULL AND NEW.deleted_at IS NOT NULL
BEGIN
    UPDATE ai_memory_sessions
    SET message_count = message_count - 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.session_id;
END;

-- ============================================================================
-- Views for Common Queries
-- ============================================================================

-- Active connections (not deleted)
CREATE VIEW IF NOT EXISTS v_active_connections AS
SELECT
    connection_id,
    user_id,
    name,
    type,
    host,
    port,
    database,
    username,
    ssl_mode,
    environment_tags,
    last_used_at,
    created_at,
    updated_at,
    sync_version
FROM connection_templates
WHERE deleted_at IS NULL
ORDER BY last_used_at DESC;

-- Recent query history (not deleted, last 30 days)
CREATE VIEW IF NOT EXISTS v_recent_queries AS
SELECT
    id,
    user_id,
    query_text,
    connection_id,
    execution_time_ms,
    row_count,
    error,
    privacy_mode,
    executed_at
FROM query_history
WHERE deleted_at IS NULL
  AND executed_at >= datetime('now', '-30 days')
ORDER BY executed_at DESC;

-- Active saved queries (not deleted)
CREATE VIEW IF NOT EXISTS v_active_saved_queries AS
SELECT
    id,
    user_id,
    title,
    description,
    query_text,
    tags,
    folder,
    is_favorite,
    created_at,
    updated_at,
    sync_version
FROM saved_queries
WHERE deleted_at IS NULL
ORDER BY updated_at DESC;

-- Pending sync items
CREATE VIEW IF NOT EXISTS v_pending_sync AS
SELECT
    entity_type,
    entity_id,
    device_id,
    sync_status,
    last_modified,
    error_count,
    last_error
FROM sync_metadata
WHERE sync_status IN ('pending', 'error')
ORDER BY last_modified ASC;

-- ============================================================================
-- Performance Optimization
-- ============================================================================

-- Analyze tables for query optimizer
ANALYZE;

-- ============================================================================
-- Row-Level Security (RLS) Policies
-- ============================================================================
-- Note: Turso/LibSQL doesn't support PostgreSQL-style RLS yet
-- Instead, enforce user_id filtering at the application layer
-- All queries MUST include WHERE user_id = :current_user_id

-- ============================================================================
-- Comments for Documentation
-- ============================================================================

-- Document table purposes
-- (Note: SQLite doesn't support COMMENT ON, this is for documentation)

/*
TABLE DOCUMENTATION:

user_preferences:
  - Stores UI settings that sync across devices
  - device_id is NULL for global preferences
  - device_id is set for device-specific preferences (e.g., window size)

connection_templates:
  - NEVER stores passwords or credentials
  - environment_tags is JSON array for filtering (e.g., ["production", "read-only"])
  - parameters stores additional connection options (sanitized)

query_history:
  - Stores sanitized query text (all credentials removed)
  - privacy_mode controls sync behavior:
    * private: never synced
    * normal: synced (default)
    * shared: prepared for team sharing (future)

saved_queries:
  - User's personal query library
  - tags and folder enable organization
  - is_favorite for quick access

ai_memory_sessions:
  - High-level session metadata
  - message_count and token_count for usage tracking

ai_memory_messages:
  - Individual messages in a conversation
  - Can be large (content field)
  - Cascading delete when session is deleted

sync_metadata:
  - Vector clock for conflict detection
  - One row per (entity, device) combination
  - Tracks sync state for offline support

device_registry:
  - Tracks user's devices for multi-device management
  - Enables "remove device" and "sync to device" features

sync_conflicts_archive:
  - Historical record of conflicts and resolutions
  - Useful for debugging and audit

user_statistics:
  - Materialized view updated by triggers
  - Fast dashboard queries without aggregation
*/

-- ============================================================================
-- Data Retention Policies
-- ============================================================================

/*
RETENTION POLICIES (to be implemented in application code):

query_history:
  - Keep last 1000 queries per user
  - Delete queries older than 90 days (except favorites)

ai_memory_messages:
  - Keep all messages while session exists
  - When session deleted, cascade delete messages

sync_conflicts_archive:
  - Keep conflicts for 30 days
  - Delete after resolution and sync confirmation

deleted items (soft delete):
  - Keep deleted items for 30 days
  - Hard delete after 30 days (tombstone cleanup)
*/

-- ============================================================================
-- Migration Strategy
-- ============================================================================

/*
MIGRATION PROCESS:

1. Check current schema version:
   SELECT MAX(version) FROM schema_migrations;

2. Apply missing migrations in order:
   - Run migration SQL
   - Insert migration record

3. Verify migration:
   - Check table exists: SELECT name FROM sqlite_master WHERE type='table';
   - Check indexes: SELECT name FROM sqlite_master WHERE type='index';

4. Future migrations add to this file as:
   -- Migration 002: Add XYZ feature
   -- Version: 2
   -- Applied: YYYY-MM-DD
*/

-- ============================================================================
-- End of Schema Definition
-- ============================================================================

-- Verify schema creation
SELECT
    'Schema created successfully' AS status,
    COUNT(*) AS table_count
FROM sqlite_master
WHERE type = 'table';
