-- Local Storage Schema
-- Migration: 001 - Initialize local storage tables

-- Connections (encrypted credentials)
CREATE TABLE IF NOT EXISTS connections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT,
    username TEXT,
    password_encrypted TEXT,
    ssl_config TEXT,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    team_id TEXT,
    is_shared BOOLEAN DEFAULT 0,
    metadata TEXT  -- JSON
);

-- Saved queries
CREATE TABLE IF NOT EXISTS saved_queries (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    query TEXT NOT NULL,
    description TEXT,
    connection_id TEXT,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    team_id TEXT,
    is_shared BOOLEAN DEFAULT 0,
    tags TEXT,  -- JSON array
    folder TEXT,
    FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE SET NULL
);

-- Query history
CREATE TABLE IF NOT EXISTS query_history (
    id TEXT PRIMARY KEY,
    query TEXT NOT NULL,
    connection_id TEXT,
    executed_by TEXT NOT NULL,
    executed_at INTEGER NOT NULL,
    duration_ms INTEGER,
    rows_returned INTEGER,
    success BOOLEAN NOT NULL,
    error TEXT,
    team_id TEXT,
    is_shared BOOLEAN DEFAULT 1,
    FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE SET NULL
);

-- Local credentials (AI keys, etc - NEVER synced)
CREATE TABLE IF NOT EXISTS local_credentials (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Settings
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    scope TEXT,  -- 'user', 'team'
    user_id TEXT,
    team_id TEXT
);

-- Schema cache
CREATE TABLE IF NOT EXISTS schema_cache (
    connection_id TEXT PRIMARY KEY,
    schema_data TEXT NOT NULL,  -- JSON
    cached_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    FOREIGN KEY (connection_id) REFERENCES connections(id) ON DELETE CASCADE
);

-- User metadata
CREATE TABLE IF NOT EXISTS user_metadata (
    id TEXT PRIMARY KEY,
    name TEXT,
    mode TEXT DEFAULT 'solo',  -- 'solo' or 'team'
    team_id TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

-- Teams (for team mode)
CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    settings TEXT  -- JSON
);

-- Team members
CREATE TABLE IF NOT EXISTS team_members (
    team_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,  -- 'admin', 'member', 'viewer'
    joined_at INTEGER NOT NULL,
    invited_by TEXT,
    PRIMARY KEY (team_id, user_id),
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_connections_created_by ON connections(created_by);
CREATE INDEX IF NOT EXISTS idx_connections_team_id ON connections(team_id);
CREATE INDEX IF NOT EXISTS idx_connections_type ON connections(type);

CREATE INDEX IF NOT EXISTS idx_saved_queries_created_by ON saved_queries(created_by);
CREATE INDEX IF NOT EXISTS idx_saved_queries_team_id ON saved_queries(team_id);
CREATE INDEX IF NOT EXISTS idx_saved_queries_connection_id ON saved_queries(connection_id);
CREATE INDEX IF NOT EXISTS idx_saved_queries_folder ON saved_queries(folder);

CREATE INDEX IF NOT EXISTS idx_query_history_executed_by ON query_history(executed_by);
CREATE INDEX IF NOT EXISTS idx_query_history_team_id ON query_history(team_id);
CREATE INDEX IF NOT EXISTS idx_query_history_connection_id ON query_history(connection_id);
CREATE INDEX IF NOT EXISTS idx_query_history_executed_at ON query_history(executed_at);
CREATE INDEX IF NOT EXISTS idx_query_history_success ON query_history(success);

CREATE INDEX IF NOT EXISTS idx_schema_cache_expires_at ON schema_cache(expires_at);

