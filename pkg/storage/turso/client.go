package turso

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

// Config holds Turso database configuration
type Config struct {
	URL       string // libsql://[db-name].turso.io OR file:./path/to/local.db
	AuthToken string // Required for libsql://, optional for file:
	MaxConns  int
}

// DB is the database interface
type DB interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Begin() (*sql.Tx, error)
	Ping() error
	Close() error
}

// NewClient creates a new Turso database connection
// Supports both local SQLite (file:) and Turso cloud (libsql:)
func NewClient(config *Config, logger *logrus.Logger) (*sql.DB, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	isLocal := strings.HasPrefix(config.URL, "file:")
	isLibSQL := strings.HasPrefix(config.URL, "libsql:")

	// Validate configuration based on connection type
	if isLibSQL && config.AuthToken == "" {
		return nil, fmt.Errorf("auth token is required for libsql:// connections")
	}

	// Build DSN
	var dsn string
	if isLocal {
		// Local SQLite file - no auth token needed
		dsn = config.URL
		logger.WithField("path", config.URL).Info("Using local SQLite database")
	} else {
		// Turso cloud - requires auth token
		dsn = fmt.Sprintf("%s?authToken=%s", config.URL, config.AuthToken)
		logger.WithField("url", config.URL).Info("Using Turso cloud database")
	}

	logger.WithFields(logrus.Fields{
		"url":      config.URL,
		"is_local": isLocal,
	}).Debug("Connecting to database")

	// Open connection
	db, err := sql.Open("libsql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	// Use different settings for local vs cloud
	maxConns := config.MaxConns
	if maxConns <= 0 {
		if isLocal {
			maxConns = 25 // Higher for local SQLite
		} else {
			maxConns = 10 // Conservative for cloud
		}
	}

	db.SetMaxOpenConns(maxConns)
	if isLocal {
		db.SetMaxIdleConns(10)
		db.SetConnMaxLifetime(0) // No limit for local
		db.SetConnMaxIdleTime(0) // No limit for local
	} else {
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)
		db.SetConnMaxIdleTime(1 * time.Minute)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close() // Best-effort close on error
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if isLocal {
		logger.Info("Successfully connected to local SQLite database")
	} else {
		logger.Info("Successfully connected to Turso cloud database")
	}
	return db, nil
}

// InitializeSchema creates all tables and indexes if they don't exist
func InitializeSchema(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Initializing Turso schema")

	// Check if tables exist
	var tableExists bool
	err := db.QueryRow(`
		SELECT COUNT(*) > 0
		FROM sqlite_master
		WHERE type='table' AND name='users'
	`).Scan(&tableExists)

	if err != nil {
		return fmt.Errorf("failed to check if tables exist: %w", err)
	}

	if tableExists {
		logger.Debug("Turso tables already exist, skipping schema initialization")
		return nil
	}

	// Read and execute schema
	// In production, you might embed the schema file or load it from a location
	// For now, we'll execute the schema statements directly
	schema := getEmbeddedSchema()
	statements := parseSchemaStatements(schema)

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Best-effort rollback

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		logger.WithFields(logrus.Fields{
			"statement": i + 1,
			"total":     len(statements),
		}).Debug("Executing schema statement")

		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to execute schema statement %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema: %w", err)
	}

	logger.Info("Turso schema initialized successfully")
	return nil
}

// parseSchemaStatements splits SQL schema into individual statements
func parseSchemaStatements(schema string) []string {
	var statements []string
	var currentStmt strings.Builder

	lines := strings.Split(schema, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines and comments when not in a statement
		if currentStmt.Len() == 0 && (trimmed == "" || strings.HasPrefix(trimmed, "--")) {
			continue
		}

		// Add line to current statement
		if currentStmt.Len() > 0 {
			currentStmt.WriteString("\n")
		}
		currentStmt.WriteString(line)

		// Check if statement is complete (ends with semicolon)
		if strings.HasSuffix(trimmed, ";") {
			statements = append(statements, currentStmt.String())
			currentStmt.Reset()
		}
	}

	// Add any remaining statement
	if currentStmt.Len() > 0 {
		statements = append(statements, currentStmt.String())
	}

	return statements
}

// getEmbeddedSchema returns the embedded schema SQL
// In production, you might use go:embed or read from a file
func getEmbeddedSchema() string {
	return `
-- Users (auth)
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
    metadata TEXT
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);

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

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_token ON sessions(refresh_token);
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(active);

-- Login attempts
CREATE TABLE IF NOT EXISTS login_attempts (
    id TEXT PRIMARY KEY,
    ip TEXT NOT NULL,
    username TEXT NOT NULL,
    timestamp INTEGER NOT NULL,
    success BOOLEAN NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_login_attempts_ip_time ON login_attempts(ip, timestamp);
CREATE INDEX IF NOT EXISTS idx_login_attempts_username_time ON login_attempts(username, timestamp);

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

CREATE INDEX IF NOT EXISTS idx_email_verification_user ON email_verification_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_email_verification_token ON email_verification_tokens(token);

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

CREATE INDEX IF NOT EXISTS idx_password_reset_user ON password_reset_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_password_reset_token ON password_reset_tokens(token);

-- License keys
CREATE TABLE IF NOT EXISTS license_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    key TEXT UNIQUE NOT NULL,
    tier TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    expires_at INTEGER,
    stripe_subscription_id TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_license_keys_user_id ON license_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_license_keys_status ON license_keys(status);

-- Connection templates
CREATE TABLE IF NOT EXISTS connection_templates (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT,
    username TEXT,
    ssl_config TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sync_version INTEGER DEFAULT 1,
    deleted_at INTEGER,
    metadata TEXT,
    organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    visibility TEXT NOT NULL DEFAULT 'personal' CHECK (visibility IN ('personal', 'shared')),
    created_by TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connection_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_connections_updated ON connection_templates(updated_at);
CREATE INDEX IF NOT EXISTS idx_connections_deleted ON connection_templates(deleted_at);
CREATE INDEX IF NOT EXISTS idx_connections_type ON connection_templates(type);
CREATE INDEX IF NOT EXISTS idx_connections_org_visibility ON connection_templates(organization_id, visibility);
CREATE INDEX IF NOT EXISTS idx_connections_created_by ON connection_templates(created_by);

-- Saved queries
CREATE TABLE IF NOT EXISTS saved_queries_sync (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    title TEXT NOT NULL,
    query TEXT NOT NULL,
    description TEXT,
    connection_id TEXT,
    folder TEXT,
    tags TEXT,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sync_version INTEGER DEFAULT 1,
    deleted_at INTEGER,
    organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    visibility TEXT NOT NULL DEFAULT 'personal' CHECK (visibility IN ('personal', 'shared')),
    created_by TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_queries_user_id ON saved_queries_sync(user_id);
CREATE INDEX IF NOT EXISTS idx_queries_updated ON saved_queries_sync(updated_at);
CREATE INDEX IF NOT EXISTS idx_queries_deleted ON saved_queries_sync(deleted_at);
CREATE INDEX IF NOT EXISTS idx_queries_connection ON saved_queries_sync(connection_id);
CREATE INDEX IF NOT EXISTS idx_queries_folder ON saved_queries_sync(folder);
CREATE INDEX IF NOT EXISTS idx_queries_org_visibility ON saved_queries_sync(organization_id, visibility);
CREATE INDEX IF NOT EXISTS idx_queries_created_by ON saved_queries_sync(created_by);

-- Query history
CREATE TABLE IF NOT EXISTS query_history_sync (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    query_sanitized TEXT NOT NULL,
    connection_id TEXT,
    executed_at INTEGER NOT NULL,
    duration_ms INTEGER,
    rows_returned INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    sync_version INTEGER DEFAULT 1,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_history_user_id ON query_history_sync(user_id);
CREATE INDEX IF NOT EXISTS idx_history_executed ON query_history_sync(executed_at);
CREATE INDEX IF NOT EXISTS idx_history_connection ON query_history_sync(connection_id);
CREATE INDEX IF NOT EXISTS idx_history_success ON query_history_sync(success);

-- Sync metadata
CREATE TABLE IF NOT EXISTS sync_metadata (
    user_id TEXT PRIMARY KEY,
    last_sync_at INTEGER NOT NULL,
    device_id TEXT,
    client_version TEXT,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- ====================================================================
-- PHASE 3: Team Collaboration Tables
-- ====================================================================

-- Organizations (teams/workspaces)
CREATE TABLE IF NOT EXISTS organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    owner_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    deleted_at INTEGER,
    max_members INTEGER DEFAULT 10,
    settings TEXT,
    FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT name_not_empty CHECK (LENGTH(name) > 0)
);

CREATE INDEX IF NOT EXISTS idx_organizations_owner ON organizations(owner_id, deleted_at);
CREATE INDEX IF NOT EXISTS idx_organizations_name ON organizations(name) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_organizations_created ON organizations(created_at);

-- Organization members with roles
CREATE TABLE IF NOT EXISTS organization_members (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL,
    invited_by TEXT,
    joined_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT valid_role CHECK (role IN ('owner', 'admin', 'member')),
    UNIQUE(organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_org_members_org ON organization_members(organization_id);
CREATE INDEX IF NOT EXISTS idx_org_members_user ON organization_members(user_id);
CREATE INDEX IF NOT EXISTS idx_org_members_role ON organization_members(role);

-- Organization invitations
CREATE TABLE IF NOT EXISTS organization_invitations (
    id TEXT PRIMARY KEY,
    organization_id TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    invited_by TEXT NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at INTEGER NOT NULL,
    accepted_at INTEGER,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT valid_invite_role CHECK (role IN ('admin', 'member')),
    UNIQUE(organization_id, email)
);

CREATE INDEX IF NOT EXISTS idx_invitations_org ON organization_invitations(organization_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email ON organization_invitations(email);
CREATE INDEX IF NOT EXISTS idx_invitations_token ON organization_invitations(token);
CREATE INDEX IF NOT EXISTS idx_invitations_expires ON organization_invitations(expires_at);

-- Audit logs for compliance and security
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    organization_id TEXT,
    user_id TEXT NOT NULL,
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    ip_address TEXT,
    user_agent TEXT,
    details TEXT,
    created_at INTEGER NOT NULL,
    FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_org ON audit_logs(organization_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user ON audit_logs(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
`
}
