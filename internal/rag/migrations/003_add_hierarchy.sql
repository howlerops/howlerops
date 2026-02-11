-- SQLite Vector Store Migration: 003
-- Add hierarchical document structure for parent-child relationships

-- Add hierarchy columns to documents table
-- Note: SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN
-- These columns are now part of the base schema (001_init), but this migration
-- remains for backwards compatibility with existing databases.
-- The test setup should check if columns exist before running this migration.

-- Add index for parent-child lookups (critical for retrieval performance)
CREATE INDEX IF NOT EXISTS idx_documents_parent ON documents(parent_id);

-- Add index for level-based queries (to filter by document hierarchy level)
CREATE INDEX IF NOT EXISTS idx_documents_level ON documents(level);

-- Add composite index for common query pattern (connection + level + type)
CREATE INDEX IF NOT EXISTS idx_documents_conn_level_type
ON documents(connection_id, level, type);

-- Update migration tracking
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at INTEGER NOT NULL DEFAULT (unixepoch())
);

INSERT OR IGNORE INTO schema_migrations (version, description)
VALUES (3, 'Add hierarchical document structure');
