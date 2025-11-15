-- ============================================================================
-- Howlerops - Shared Resources Feature Migration
-- ============================================================================
-- Migration Version: 2
-- Purpose: Add support for shared connections and queries within organizations
-- ============================================================================

-- Migration record
INSERT INTO schema_migrations (version, name, checksum) VALUES
    (2, 'shared_resources_support', 'sha256:shared_resources_v2');

-- ============================================================================
-- Add shared resource columns to connection_templates
-- ============================================================================

-- Add visibility and organization_id columns
ALTER TABLE connection_templates ADD COLUMN visibility TEXT NOT NULL DEFAULT 'personal'
    CHECK (visibility IN ('personal', 'shared'));

ALTER TABLE connection_templates ADD COLUMN organization_id TEXT
    REFERENCES organizations(id) ON DELETE CASCADE;

-- Add shared_by and shared_at for audit trail
ALTER TABLE connection_templates ADD COLUMN shared_by TEXT;
ALTER TABLE connection_templates ADD COLUMN shared_at INTEGER;

-- Add indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_connection_templates_org
    ON connection_templates(organization_id) WHERE organization_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_connection_templates_visibility
    ON connection_templates(user_id, visibility) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_connection_templates_shared
    ON connection_templates(organization_id, visibility)
    WHERE visibility = 'shared' AND deleted_at IS NULL;

-- ============================================================================
-- Add shared resource columns to saved_queries
-- ============================================================================

ALTER TABLE saved_queries ADD COLUMN visibility TEXT NOT NULL DEFAULT 'personal'
    CHECK (visibility IN ('personal', 'shared'));

ALTER TABLE saved_queries ADD COLUMN organization_id TEXT
    REFERENCES organizations(id) ON DELETE CASCADE;

ALTER TABLE saved_queries ADD COLUMN shared_by TEXT;
ALTER TABLE saved_queries ADD COLUMN shared_at INTEGER;

-- Add indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_saved_queries_org
    ON saved_queries(organization_id) WHERE organization_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_saved_queries_visibility
    ON saved_queries(user_id, visibility) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_saved_queries_shared
    ON saved_queries(organization_id, visibility)
    WHERE visibility = 'shared' AND deleted_at IS NULL;

-- ============================================================================
-- Update Views
-- ============================================================================

-- Drop and recreate v_active_connections with new fields
DROP VIEW IF EXISTS v_active_connections;
CREATE VIEW v_active_connections AS
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
    visibility,
    organization_id,
    shared_by,
    shared_at,
    last_used_at,
    created_at,
    updated_at,
    sync_version
FROM connection_templates
WHERE deleted_at IS NULL
ORDER BY last_used_at DESC;

-- Drop and recreate v_active_saved_queries with new fields
DROP VIEW IF EXISTS v_active_saved_queries;
CREATE VIEW v_active_saved_queries AS
SELECT
    id,
    user_id,
    title,
    description,
    query_text,
    tags,
    folder,
    is_favorite,
    visibility,
    organization_id,
    shared_by,
    shared_at,
    created_at,
    updated_at,
    sync_version
FROM saved_queries
WHERE deleted_at IS NULL
ORDER BY updated_at DESC;

-- ============================================================================
-- New Views for Shared Resources
-- ============================================================================

-- View for all accessible connections (personal + shared from user's orgs)
CREATE VIEW IF NOT EXISTS v_user_accessible_connections AS
SELECT DISTINCT
    c.connection_id,
    c.user_id AS owner_id,
    c.name,
    c.type,
    c.host,
    c.port,
    c.database,
    c.username,
    c.ssl_mode,
    c.environment_tags,
    c.visibility,
    c.organization_id,
    c.shared_by,
    c.shared_at,
    c.last_used_at,
    c.created_at,
    c.updated_at,
    c.sync_version,
    CASE
        WHEN c.visibility = 'personal' THEN 'owner'
        ELSE 'viewer'
    END AS access_level
FROM connection_templates c
LEFT JOIN organization_members om ON c.organization_id = om.organization_id
WHERE c.deleted_at IS NULL
    AND (
        -- Personal connections owned by user
        (c.visibility = 'personal' AND c.user_id = om.user_id)
        OR
        -- Shared connections in user's organizations
        (c.visibility = 'shared' AND om.user_id IS NOT NULL)
    );

-- View for all accessible saved queries (personal + shared from user's orgs)
CREATE VIEW IF NOT EXISTS v_user_accessible_queries AS
SELECT DISTINCT
    q.id,
    q.user_id AS owner_id,
    q.title,
    q.description,
    q.query_text,
    q.tags,
    q.folder,
    q.is_favorite,
    q.visibility,
    q.organization_id,
    q.shared_by,
    q.shared_at,
    q.created_at,
    q.updated_at,
    q.sync_version,
    CASE
        WHEN q.visibility = 'personal' THEN 'owner'
        ELSE 'viewer'
    END AS access_level
FROM saved_queries q
LEFT JOIN organization_members om ON q.organization_id = om.organization_id
WHERE q.deleted_at IS NULL
    AND (
        -- Personal queries owned by user
        (q.visibility = 'personal' AND q.user_id = om.user_id)
        OR
        -- Shared queries in user's organizations
        (q.visibility = 'shared' AND om.user_id IS NOT NULL)
    );

-- ============================================================================
-- Audit Triggers
-- ============================================================================

-- Trigger to set shared_at timestamp when making a connection shared
CREATE TRIGGER IF NOT EXISTS set_connection_shared_timestamp
AFTER UPDATE OF visibility ON connection_templates
FOR EACH ROW
WHEN NEW.visibility = 'shared' AND OLD.visibility = 'personal'
BEGIN
    UPDATE connection_templates
    SET shared_at = CURRENT_TIMESTAMP
    WHERE connection_id = NEW.connection_id;
END;

-- Trigger to set shared_at timestamp when making a query shared
CREATE TRIGGER IF NOT EXISTS set_query_shared_timestamp
AFTER UPDATE OF visibility ON saved_queries
FOR EACH ROW
WHEN NEW.visibility = 'shared' AND OLD.visibility = 'personal'
BEGIN
    UPDATE saved_queries
    SET shared_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;

-- ============================================================================
-- Data Validation Constraints
-- ============================================================================

-- Ensure shared resources have an organization_id
CREATE TRIGGER IF NOT EXISTS validate_connection_shared_org
BEFORE UPDATE OF visibility ON connection_templates
FOR EACH ROW
WHEN NEW.visibility = 'shared' AND NEW.organization_id IS NULL
BEGIN
    SELECT RAISE(ABORT, 'Shared connections must have an organization_id');
END;

CREATE TRIGGER IF NOT EXISTS validate_query_shared_org
BEFORE UPDATE OF visibility ON saved_queries
FOR EACH ROW
WHEN NEW.visibility = 'shared' AND NEW.organization_id IS NULL
BEGIN
    SELECT RAISE(ABORT, 'Shared queries must have an organization_id');
END;

-- Ensure personal resources don't have organization_id
CREATE TRIGGER IF NOT EXISTS validate_connection_personal_no_org
BEFORE UPDATE OF visibility ON connection_templates
FOR EACH ROW
WHEN NEW.visibility = 'personal' AND NEW.organization_id IS NOT NULL
BEGIN
    UPDATE connection_templates
    SET organization_id = NULL, shared_by = NULL, shared_at = NULL
    WHERE connection_id = NEW.connection_id;
END;

CREATE TRIGGER IF NOT EXISTS validate_query_personal_no_org
BEFORE UPDATE OF visibility ON saved_queries
FOR EACH ROW
WHEN NEW.visibility = 'personal' AND NEW.organization_id IS NOT NULL
BEGIN
    UPDATE saved_queries
    SET organization_id = NULL, shared_by = NULL, shared_at = NULL
    WHERE id = NEW.id;
END;

-- ============================================================================
-- Statistics Update
-- ============================================================================

-- Add shared resource counts to user_statistics
ALTER TABLE user_statistics ADD COLUMN total_shared_connections INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_statistics ADD COLUMN total_shared_queries INTEGER NOT NULL DEFAULT 0;

-- ============================================================================
-- Comments for Documentation
-- ============================================================================

/*
SHARED RESOURCES FEATURE DOCUMENTATION:

Visibility Modes:
- personal: Only visible to the owner
- shared: Visible to all members of the organization

Access Control:
- Only the owner can change visibility from personal to shared
- Only the owner can unshare (change from shared to personal)
- Organization members can view and use shared resources
- Permission checks are enforced at the service layer

Sync Behavior:
- Personal resources sync only for the owner
- Shared resources sync for all organization members
- Conflicts are resolved using last-write-wins strategy
- Metadata tracks who shared the resource and when

Views:
- v_user_accessible_connections: All connections a user can access
- v_user_accessible_queries: All queries a user can access
- Both views include personal + shared from user's orgs

Audit Trail:
- shared_by: User ID who made the resource shared
- shared_at: Timestamp when the resource was shared
- Audit logs track all sharing/unsharing actions

Migration Notes:
- All existing resources default to 'personal' visibility
- No existing data is modified during migration
- Triggers ensure data integrity for visibility changes
*/

-- ============================================================================
-- Verify Migration
-- ============================================================================

SELECT
    'Migration 2 completed successfully' AS status,
    COUNT(*) AS total_connections,
    SUM(CASE WHEN visibility = 'personal' THEN 1 ELSE 0 END) AS personal_connections,
    SUM(CASE WHEN visibility = 'shared' THEN 1 ELSE 0 END) AS shared_connections
FROM connection_templates;
