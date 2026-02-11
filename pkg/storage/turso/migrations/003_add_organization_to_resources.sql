-- Migration: Add organization support to connections and queries
-- Version: 003
-- Date: 2025-10-23
-- Description: Adds organization_id, visibility, and created_by columns to
--              connection_templates and saved_queries_sync tables to enable
--              team collaboration and resource sharing.

-- ============================================================================
-- CONNECTIONS: Add organization columns
-- ============================================================================

-- Add organization reference (nullable for personal connections)
ALTER TABLE connection_templates ADD COLUMN organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE;

-- Add visibility control (personal vs shared within organization)
ALTER TABLE connection_templates ADD COLUMN visibility TEXT NOT NULL DEFAULT 'personal' CHECK (visibility IN ('personal', 'shared'));

-- Add creator tracking (who created this connection)
ALTER TABLE connection_templates ADD COLUMN created_by TEXT NOT NULL DEFAULT '';

-- Add indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_connections_org_visibility ON connection_templates(organization_id, visibility);
CREATE INDEX IF NOT EXISTS idx_connections_created_by ON connection_templates(created_by);

-- ============================================================================
-- SAVED QUERIES: Add organization columns
-- ============================================================================

-- Add organization reference (nullable for personal queries)
ALTER TABLE saved_queries_sync ADD COLUMN organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE;

-- Add visibility control (personal vs shared within organization)
ALTER TABLE saved_queries_sync ADD COLUMN visibility TEXT NOT NULL DEFAULT 'personal' CHECK (visibility IN ('personal', 'shared'));

-- Add creator tracking (who created this query)
ALTER TABLE saved_queries_sync ADD COLUMN created_by TEXT NOT NULL DEFAULT '';

-- Add indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_queries_org_visibility ON saved_queries_sync(organization_id, visibility);
CREATE INDEX IF NOT EXISTS idx_queries_created_by ON saved_queries_sync(created_by);

-- ============================================================================
-- DATA MIGRATION: Update existing records
-- ============================================================================

-- Set created_by to user_id for all existing connections
UPDATE connection_templates SET created_by = user_id WHERE created_by = '';

-- Set created_by to user_id for all existing queries
UPDATE saved_queries_sync SET created_by = user_id WHERE created_by = '';

-- ============================================================================
-- VERIFICATION QUERIES (for testing)
-- ============================================================================

-- Verify connection_templates columns exist
-- SELECT
--   COUNT(DISTINCT organization_id) as org_count,
--   COUNT(CASE WHEN visibility = 'personal' THEN 1 END) as personal_count,
--   COUNT(CASE WHEN visibility = 'shared' THEN 1 END) as shared_count,
--   COUNT(CASE WHEN created_by != '' THEN 1 END) as has_creator_count
-- FROM connection_templates;

-- Verify saved_queries_sync columns exist
-- SELECT
--   COUNT(DISTINCT organization_id) as org_count,
--   COUNT(CASE WHEN visibility = 'personal' THEN 1 END) as personal_count,
--   COUNT(CASE WHEN visibility = 'shared' THEN 1 END) as shared_count,
--   COUNT(CASE WHEN created_by != '' THEN 1 END) as has_creator_count
-- FROM saved_queries_sync;
