-- Migration: 004 - Query Templates and Scheduled Queries
-- Phase 4: Enable users to save reusable query templates and schedule automated execution
-- =============================================================================

-- Query templates with parameter support
CREATE TABLE IF NOT EXISTS query_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    sql_template TEXT NOT NULL,
    parameters TEXT, -- JSON array: [{"name": "user_id", "type": "string", "default": ""}]
    tags TEXT, -- JSON array: ["reporting", "analytics"]
    category TEXT, -- 'reporting', 'analytics', 'maintenance', 'custom'
    organization_id TEXT,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    is_public BOOLEAN DEFAULT 0,
    usage_count INTEGER DEFAULT 0,
    deleted_at INTEGER, -- Soft delete for sync
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Scheduled query execution
CREATE TABLE IF NOT EXISTS query_schedules (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    frequency TEXT NOT NULL, -- cron expression: "0 9 * * *" (daily at 9am)
    parameters TEXT, -- JSON with param values: {"user_id": "123"}
    last_run_at INTEGER,
    next_run_at INTEGER,
    status TEXT DEFAULT 'active' CHECK (status IN ('active', 'paused', 'failed')),
    created_by TEXT NOT NULL,
    organization_id TEXT,
    notification_email TEXT,
    result_storage TEXT DEFAULT 'none', -- where to store results: 'none', 's3', 'database'
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER, -- Soft delete
    FOREIGN KEY (template_id) REFERENCES query_templates(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- Schedule execution history
CREATE TABLE IF NOT EXISTS schedule_executions (
    id TEXT PRIMARY KEY,
    schedule_id TEXT NOT NULL,
    executed_at INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('success', 'failed', 'timeout', 'cancelled')),
    duration_ms INTEGER,
    rows_returned INTEGER,
    error_message TEXT,
    result_preview TEXT, -- JSON: first 10 rows for quick viewing
    FOREIGN KEY (schedule_id) REFERENCES query_schedules(id) ON DELETE CASCADE
);

-- =============================================================================
-- INDEXES FOR QUERY TEMPLATES
-- =============================================================================

-- Query templates indexes
CREATE INDEX IF NOT EXISTS idx_templates_org ON query_templates(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_templates_creator ON query_templates(created_by) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_templates_category ON query_templates(category) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_templates_updated ON query_templates(updated_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_templates_public ON query_templates(is_public) WHERE deleted_at IS NULL AND is_public = 1;

-- Schedules indexes
CREATE INDEX IF NOT EXISTS idx_schedules_template ON query_schedules(template_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_schedules_org ON query_schedules(organization_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_schedules_creator ON query_schedules(created_by) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON query_schedules(next_run_at) WHERE status = 'active' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_schedules_status ON query_schedules(status) WHERE deleted_at IS NULL;

-- Execution history indexes
CREATE INDEX IF NOT EXISTS idx_schedule_exec_schedule ON schedule_executions(schedule_id);
CREATE INDEX IF NOT EXISTS idx_schedule_exec_time ON schedule_executions(executed_at);
CREATE INDEX IF NOT EXISTS idx_schedule_exec_status ON schedule_executions(status);

-- =============================================================================
-- NOTES
-- =============================================================================
-- 1. Templates support parameterized queries using {{param_name}} syntax
-- 2. Parameters are defined with type, default value, and optional validation
-- 3. Schedules use cron expressions for flexible timing (e.g., "0 9 * * MON-FRI")
-- 4. Organization-scoped templates can be shared across team members
-- 5. Public templates are available to all users in the organization
-- 6. Execution history is retained for audit and monitoring purposes
-- 7. Result storage can be extended to support S3/GCS for large result sets
-- 8. Soft deletes enable safe recovery and conflict resolution during sync
