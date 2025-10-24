-- Migration 006: Data Management & Compliance Features
-- Enhanced audit logging, data retention, GDPR compliance, backups, and PII detection

-- Enhanced audit logs with field-level tracking
CREATE TABLE IF NOT EXISTS audit_logs_detailed (
    id TEXT PRIMARY KEY,
    audit_log_id TEXT NOT NULL, -- References parent audit_logs entry
    table_name TEXT NOT NULL,
    record_id TEXT NOT NULL,
    field_name TEXT NOT NULL,
    old_value TEXT,
    new_value TEXT,
    field_type TEXT, -- 'pii', 'sensitive', 'normal'
    created_at INTEGER NOT NULL,
    FOREIGN KEY (audit_log_id) REFERENCES audit_logs(id) ON DELETE CASCADE
);

-- Data retention policies per organization
CREATE TABLE IF NOT EXISTS data_retention_policies (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    resource_type TEXT NOT NULL, -- 'query_history', 'audit_logs', 'connections', 'templates'
    retention_days INTEGER NOT NULL,
    auto_archive BOOLEAN DEFAULT true,
    archive_location TEXT, -- 's3://bucket/path' or 'local'
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE(organization_id, resource_type)
);

-- Data export requests (GDPR)
CREATE TABLE IF NOT EXISTS data_export_requests (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    organization_id TEXT,
    request_type TEXT NOT NULL, -- 'export', 'delete'
    status TEXT NOT NULL, -- 'pending', 'processing', 'completed', 'failed'
    export_url TEXT, -- S3 URL or file path
    requested_at INTEGER NOT NULL,
    completed_at INTEGER,
    error_message TEXT,
    metadata TEXT, -- JSON with additional info
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE SET NULL
);

-- Database backups
CREATE TABLE IF NOT EXISTS database_backups (
    id TEXT PRIMARY KEY,
    backup_type TEXT NOT NULL, -- 'full', 'incremental'
    status TEXT NOT NULL, -- 'in_progress', 'completed', 'failed'
    file_path TEXT NOT NULL,
    file_size INTEGER,
    tables_included TEXT, -- JSON array of table names
    started_at INTEGER NOT NULL,
    completed_at INTEGER,
    error_message TEXT,
    created_by TEXT,
    FOREIGN KEY (created_by) REFERENCES users(id)
);

-- PII field catalog
CREATE TABLE IF NOT EXISTS pii_fields (
    id TEXT PRIMARY KEY,
    table_name TEXT NOT NULL,
    field_name TEXT NOT NULL,
    pii_type TEXT NOT NULL, -- 'email', 'phone', 'ssn', 'credit_card', 'address', 'name'
    detection_method TEXT NOT NULL, -- 'manual', 'pattern', 'ml'
    confidence_score REAL, -- 0.0 to 1.0 for automated detection
    verified BOOLEAN DEFAULT false,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    UNIQUE(table_name, field_name)
);

-- Data archival log
CREATE TABLE IF NOT EXISTS data_archive_log (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    records_archived INTEGER NOT NULL,
    archive_location TEXT NOT NULL,
    archive_date INTEGER NOT NULL,
    cutoff_date INTEGER NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_audit_detailed_audit_log ON audit_logs_detailed(audit_log_id);
CREATE INDEX IF NOT EXISTS idx_audit_detailed_table ON audit_logs_detailed(table_name, record_id);
CREATE INDEX IF NOT EXISTS idx_audit_detailed_field ON audit_logs_detailed(field_name);
CREATE INDEX IF NOT EXISTS idx_audit_detailed_type ON audit_logs_detailed(field_type);

CREATE INDEX IF NOT EXISTS idx_retention_org ON data_retention_policies(organization_id);
CREATE INDEX IF NOT EXISTS idx_retention_resource ON data_retention_policies(resource_type);

CREATE INDEX IF NOT EXISTS idx_export_user ON data_export_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_export_org ON data_export_requests(organization_id);
CREATE INDEX IF NOT EXISTS idx_export_status ON data_export_requests(status);
CREATE INDEX IF NOT EXISTS idx_export_type ON data_export_requests(request_type);
CREATE INDEX IF NOT EXISTS idx_export_requested_at ON data_export_requests(requested_at);

CREATE INDEX IF NOT EXISTS idx_backup_status ON database_backups(status);
CREATE INDEX IF NOT EXISTS idx_backup_type ON database_backups(backup_type);
CREATE INDEX IF NOT EXISTS idx_backup_started ON database_backups(started_at);

CREATE INDEX IF NOT EXISTS idx_pii_table ON pii_fields(table_name);
CREATE INDEX IF NOT EXISTS idx_pii_type ON pii_fields(pii_type);
CREATE INDEX IF NOT EXISTS idx_pii_verified ON pii_fields(verified);

CREATE INDEX IF NOT EXISTS idx_archive_org ON data_archive_log(organization_id);
CREATE INDEX IF NOT EXISTS idx_archive_date ON data_archive_log(archive_date);
