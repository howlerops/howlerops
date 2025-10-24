-- Enterprise Security Features Migration
-- Phase 5: SSO, IP Whitelisting, 2FA, and API Keys

-- SSO configuration (framework ready)
CREATE TABLE IF NOT EXISTS sso_config (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    provider TEXT NOT NULL, -- 'saml', 'oauth2', 'oidc'
    provider_name TEXT NOT NULL, -- 'Okta', 'Auth0', 'Azure AD', etc.
    metadata TEXT NOT NULL, -- JSON config
    enabled BOOLEAN DEFAULT false,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    UNIQUE(organization_id) -- One SSO config per org
);

-- IP whitelist for organizations
CREATE TABLE IF NOT EXISTS ip_whitelist (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    ip_address TEXT NOT NULL,
    ip_range TEXT, -- CIDR notation: 192.168.1.0/24
    description TEXT,
    created_by TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Two-factor authentication
CREATE TABLE IF NOT EXISTS user_2fa (
    user_id TEXT PRIMARY KEY,
    enabled BOOLEAN DEFAULT false,
    secret TEXT NOT NULL, -- TOTP secret
    backup_codes TEXT, -- JSON array of backup codes
    created_at INTEGER NOT NULL,
    enabled_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- API keys for programmatic access
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    organization_id TEXT,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL, -- bcrypt hash of the key
    key_prefix TEXT NOT NULL, -- First 8 chars for identification: "sk_live_abc12345"
    permissions TEXT NOT NULL, -- JSON array of permissions
    last_used_at INTEGER,
    expires_at INTEGER,
    created_at INTEGER NOT NULL,
    revoked_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
);

-- Security events log
CREATE TABLE IF NOT EXISTS security_events (
    id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL, -- 'login_failed', '2fa_enabled', 'ip_blocked', etc.
    user_id TEXT,
    organization_id TEXT,
    ip_address TEXT,
    user_agent TEXT,
    details TEXT, -- JSON
    created_at INTEGER NOT NULL
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_ip_whitelist_org ON ip_whitelist(organization_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_user ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX IF NOT EXISTS idx_security_events_user ON security_events(user_id);
CREATE INDEX IF NOT EXISTS idx_security_events_type ON security_events(event_type);
CREATE INDEX IF NOT EXISTS idx_security_events_created ON security_events(created_at);
CREATE INDEX IF NOT EXISTS idx_sso_config_org ON sso_config(organization_id);