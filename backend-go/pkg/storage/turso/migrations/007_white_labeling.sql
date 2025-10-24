-- Migration 007: White-labeling and Enterprise Multi-tenancy
-- Created: 2025-10-24

-- White-labeling configuration
CREATE TABLE IF NOT EXISTS white_label_config (
    organization_id TEXT PRIMARY KEY,
    custom_domain TEXT UNIQUE, -- app.customer.com
    logo_url TEXT,
    favicon_url TEXT,
    primary_color TEXT, -- #1E40AF
    secondary_color TEXT,
    accent_color TEXT,
    company_name TEXT,
    support_email TEXT,
    custom_css TEXT, -- Additional CSS overrides
    hide_branding BOOLEAN DEFAULT false, -- Hide "Powered by SQL Studio"
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Resource quotas per organization
CREATE TABLE IF NOT EXISTS organization_quotas (
    organization_id TEXT PRIMARY KEY,
    max_connections INTEGER DEFAULT 10,
    max_queries_per_day INTEGER DEFAULT 1000,
    max_storage_mb INTEGER DEFAULT 100,
    max_api_calls_per_hour INTEGER DEFAULT 1000,
    max_concurrent_queries INTEGER DEFAULT 5,
    max_team_members INTEGER DEFAULT 5,
    features_enabled TEXT DEFAULT 'basic', -- JSON array or CSV: 'basic,advanced,enterprise'
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Resource usage tracking (daily granularity)
CREATE TABLE IF NOT EXISTS organization_usage (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    usage_date INTEGER NOT NULL, -- Unix timestamp (day granularity)
    connections_count INTEGER DEFAULT 0,
    queries_count INTEGER DEFAULT 0,
    storage_used_mb REAL DEFAULT 0,
    api_calls_count INTEGER DEFAULT 0,
    concurrent_queries_peak INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(organization_id, usage_date)
);

-- Hourly usage tracking for rate limiting
CREATE TABLE IF NOT EXISTS organization_usage_hourly (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    usage_hour INTEGER NOT NULL, -- Unix timestamp (hour granularity)
    api_calls_count INTEGER DEFAULT 0,
    query_count INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(organization_id, usage_hour)
);

-- SLA monitoring metrics
CREATE TABLE IF NOT EXISTS sla_metrics (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    metric_date INTEGER NOT NULL, -- Unix timestamp (day granularity)
    uptime_percentage REAL DEFAULT 100.0, -- 99.9
    avg_response_time_ms REAL DEFAULT 0,
    error_rate REAL DEFAULT 0, -- 0.1 for 0.1%
    p95_response_time_ms REAL DEFAULT 0,
    p99_response_time_ms REAL DEFAULT 0,
    total_requests INTEGER DEFAULT 0,
    failed_requests INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(organization_id, metric_date)
);

-- Request log for SLA calculation (kept for 30 days)
CREATE TABLE IF NOT EXISTS request_log (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    response_time_ms INTEGER NOT NULL,
    status_code INTEGER NOT NULL,
    success BOOLEAN NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Custom domains verification
CREATE TABLE IF NOT EXISTS domain_verification (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    verification_token TEXT NOT NULL,
    verified BOOLEAN DEFAULT false,
    verified_at INTEGER,
    dns_record_type TEXT NOT NULL, -- 'TXT', 'CNAME'
    dns_record_name TEXT NOT NULL,
    dns_record_value TEXT NOT NULL,
    ssl_enabled BOOLEAN DEFAULT false,
    ssl_certificate_expires_at INTEGER,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(organization_id, domain)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_white_label_domain ON white_label_config(custom_domain);
CREATE INDEX IF NOT EXISTS idx_usage_org_date ON organization_usage(organization_id, usage_date);
CREATE INDEX IF NOT EXISTS idx_usage_hourly_org_hour ON organization_usage_hourly(organization_id, usage_hour);
CREATE INDEX IF NOT EXISTS idx_sla_org_date ON sla_metrics(organization_id, metric_date);
CREATE INDEX IF NOT EXISTS idx_request_log_org_created ON request_log(organization_id, created_at);
CREATE INDEX IF NOT EXISTS idx_domain_verification_domain ON domain_verification(domain);
CREATE INDEX IF NOT EXISTS idx_domain_verification_org ON domain_verification(organization_id);

-- Trigger to update updated_at timestamps
CREATE TRIGGER IF NOT EXISTS update_white_label_config_updated_at
    AFTER UPDATE ON white_label_config
    FOR EACH ROW
BEGIN
    UPDATE white_label_config SET updated_at = strftime('%s', 'now') WHERE organization_id = NEW.organization_id;
END;

CREATE TRIGGER IF NOT EXISTS update_organization_quotas_updated_at
    AFTER UPDATE ON organization_quotas
    FOR EACH ROW
BEGIN
    UPDATE organization_quotas SET updated_at = strftime('%s', 'now') WHERE organization_id = NEW.organization_id;
END;

CREATE TRIGGER IF NOT EXISTS update_domain_verification_updated_at
    AFTER UPDATE ON domain_verification
    FOR EACH ROW
BEGIN
    UPDATE domain_verification SET updated_at = strftime('%s', 'now') WHERE id = NEW.id;
END;

-- Default quotas for new organizations (via application logic)
-- Free tier: 5 connections, 100 queries/day, 50MB storage, 100 API calls/hour
-- Pro tier: 20 connections, 5000 queries/day, 500MB storage, 1000 API calls/hour
-- Enterprise tier: unlimited (high limits)
