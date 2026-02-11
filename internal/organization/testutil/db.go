package testutil

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// TestDB wraps a database connection for testing
type TestDB struct {
	DB     *sql.DB
	Logger *logrus.Logger
}

// SetupTestDB creates an in-memory SQLite database for testing
func SetupTestDB(t *testing.T) *TestDB {
	t.Helper()

	// Create in-memory database
	db, err := sql.Open("sqlite3", ":memory:?cache=shared")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	// Create test database wrapper
	testDB := &TestDB{
		DB:     db,
		Logger: logger,
	}

	// Create tables
	if err := testDB.CreateTables(); err != nil {
		t.Fatalf("Failed to create tables: %v", err)
	}

	return testDB
}

// CreateTables creates all required tables for organization tests
func (tdb *TestDB) CreateTables() error {
	schema := `
	-- Users table (simplified for testing)
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		username TEXT UNIQUE NOT NULL,
		display_name TEXT,
		password TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- Organizations table
	CREATE TABLE IF NOT EXISTS organizations (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		owner_id TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		deleted_at TIMESTAMP,
		max_members INTEGER DEFAULT 10,
		settings TEXT,
		FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Organization members table
	CREATE TABLE IF NOT EXISTS organization_members (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL,
		user_id TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('owner', 'admin', 'member')),
		invited_by TEXT,
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE SET NULL,
		UNIQUE(organization_id, user_id)
	);

	-- Organization invitations table
	CREATE TABLE IF NOT EXISTS organization_invitations (
		id TEXT PRIMARY KEY,
		organization_id TEXT NOT NULL,
		email TEXT NOT NULL,
		role TEXT NOT NULL CHECK(role IN ('admin', 'member')),
		invited_by TEXT NOT NULL,
		token TEXT UNIQUE NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		accepted_at TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
		FOREIGN KEY (invited_by) REFERENCES users(id) ON DELETE CASCADE,
		UNIQUE(organization_id, email, accepted_at)
	);

	-- Audit logs table
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
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- Create indexes for better query performance
	CREATE INDEX IF NOT EXISTS idx_org_members_org_id ON organization_members(organization_id);
	CREATE INDEX IF NOT EXISTS idx_org_members_user_id ON organization_members(user_id);
	CREATE INDEX IF NOT EXISTS idx_invitations_org_id ON organization_invitations(organization_id);
	CREATE INDEX IF NOT EXISTS idx_invitations_email ON organization_invitations(email);
	CREATE INDEX IF NOT EXISTS idx_invitations_token ON organization_invitations(token);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_org_id ON audit_logs(organization_id);
	CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id ON audit_logs(user_id);
	`

	_, err := tdb.DB.Exec(schema)
	return err
}

// Close closes the database connection
func (tdb *TestDB) Close() error {
	return tdb.DB.Close()
}

// CleanupTables truncates all tables for a fresh test state
func (tdb *TestDB) CleanupTables(ctx context.Context) error {
	tables := []string{
		"audit_logs",
		"organization_invitations",
		"organization_members",
		"organizations",
		"users",
	}

	for _, table := range tables {
		// #nosec G202 - table names from hardcoded test list, safe for test cleanup
		if _, err := tdb.DB.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return err
		}
	}

	return nil
}

// InsertTestUser inserts a test user into the database
func (tdb *TestDB) InsertTestUser(ctx context.Context, id, email, username, password string) error {
	_, err := tdb.DB.ExecContext(ctx,
		`INSERT INTO users (id, email, username, password, display_name) VALUES (?, ?, ?, ?, ?)`,
		id, email, username, password, username,
	)
	return err
}

// GetUserByID retrieves a user by ID
func (tdb *TestDB) GetUserByID(ctx context.Context, id string) (map[string]interface{}, error) {
	row := tdb.DB.QueryRowContext(ctx, `SELECT id, email, username, display_name FROM users WHERE id = ?`, id)

	var userID, email, username string
	var displayName sql.NullString

	if err := row.Scan(&userID, &email, &username, &displayName); err != nil {
		return nil, err
	}

	user := map[string]interface{}{
		"id":       userID,
		"email":    email,
		"username": username,
	}

	if displayName.Valid {
		user["display_name"] = displayName.String
	}

	return user, nil
}

// CountOrganizations returns the number of organizations
func (tdb *TestDB) CountOrganizations(ctx context.Context) (int, error) {
	var count int
	err := tdb.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM organizations WHERE deleted_at IS NULL").Scan(&count)
	return count, err
}

// CountMembers returns the number of members in an organization
func (tdb *TestDB) CountMembers(ctx context.Context, orgID string) (int, error) {
	var count int
	err := tdb.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM organization_members WHERE organization_id = ?", orgID).Scan(&count)
	return count, err
}

// CountInvitations returns the number of pending invitations for an organization
func (tdb *TestDB) CountInvitations(ctx context.Context, orgID string) (int, error) {
	var count int
	err := tdb.DB.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM organization_invitations WHERE organization_id = ? AND accepted_at IS NULL",
		orgID,
	).Scan(&count)
	return count, err
}

// CountAuditLogs returns the number of audit log entries
func (tdb *TestDB) CountAuditLogs(ctx context.Context, orgID string) (int, error) {
	var count int
	err := tdb.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_logs WHERE organization_id = ?", orgID).Scan(&count)
	return count, err
}

// GetOrganizationByID retrieves an organization by ID
func (tdb *TestDB) GetOrganizationByID(ctx context.Context, id string) (map[string]interface{}, error) {
	row := tdb.DB.QueryRowContext(ctx,
		`SELECT id, name, description, owner_id, max_members FROM organizations WHERE id = ? AND deleted_at IS NULL`,
		id,
	)

	var orgID, name, description, ownerID string
	var maxMembers int

	if err := row.Scan(&orgID, &name, &description, &ownerID, &maxMembers); err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":          orgID,
		"name":        name,
		"description": description,
		"owner_id":    ownerID,
		"max_members": maxMembers,
	}, nil
}

// GetMemberRole retrieves a member's role
func (tdb *TestDB) GetMemberRole(ctx context.Context, orgID, userID string) (string, error) {
	var role string
	err := tdb.DB.QueryRowContext(ctx,
		`SELECT role FROM organization_members WHERE organization_id = ? AND user_id = ?`,
		orgID, userID,
	).Scan(&role)
	return role, err
}

// MemberExists checks if a member exists in an organization
func (tdb *TestDB) MemberExists(ctx context.Context, orgID, userID string) (bool, error) {
	var count int
	err := tdb.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organization_members WHERE organization_id = ? AND user_id = ?`,
		orgID, userID,
	).Scan(&count)
	return count > 0, err
}

// InvitationExists checks if an invitation exists
func (tdb *TestDB) InvitationExists(ctx context.Context, token string) (bool, error) {
	var count int
	err := tdb.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organization_invitations WHERE token = ?`,
		token,
	).Scan(&count)
	return count > 0, err
}
