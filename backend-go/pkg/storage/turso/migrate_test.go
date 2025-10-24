package turso

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) (*sql.DB, *logrus.Logger) {
	// Create temporary directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := sql.Open("libsql", fmt.Sprintf("file:%s", dbPath))
	require.NoError(t, err, "Failed to open test database")

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetOutput(os.Stdout)

	return db, logger
}

// TestMigrationTableCreation tests that the migrations table is created correctly
func TestMigrationTableCreation(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	// Create migrations table
	err := createMigrationsTable(db)
	require.NoError(t, err, "Failed to create migrations table")

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
	require.NoError(t, err, "Migrations table should exist")
	assert.Equal(t, "schema_migrations", tableName)

	// Verify table structure
	rows, err := db.Query("PRAGMA table_info(schema_migrations)")
	require.NoError(t, err)
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		require.NoError(t, err)
		columns[name] = colType
	}

	assert.Equal(t, "INTEGER", columns["version"])
	assert.Equal(t, "TEXT", columns["description"])
	assert.Equal(t, "INTEGER", columns["applied_at"])
}

// TestGetCurrentVersion tests getting the current migration version
func TestGetCurrentVersion(t *testing.T) {
	db, _ := setupTestDB(t)
	defer db.Close()

	// Create migrations table
	err := createMigrationsTable(db)
	require.NoError(t, err)

	// Should return 0 when no migrations applied
	version, err := getCurrentVersion(db)
	require.NoError(t, err)
	assert.Equal(t, 0, version)

	// Insert a migration
	_, err = db.Exec("INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
		1, "Test migration", time.Now().Unix())
	require.NoError(t, err)

	version, err = getCurrentVersion(db)
	require.NoError(t, err)
	assert.Equal(t, 1, version)

	// Insert higher version
	_, err = db.Exec("INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
		3, "Another migration", time.Now().Unix())
	require.NoError(t, err)

	version, err = getCurrentVersion(db)
	require.NoError(t, err)
	assert.Equal(t, 3, version, "Should return highest version")
}

// TestRunMigrationsIdempotency tests that running migrations twice is safe
func TestRunMigrationsIdempotency(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema first
	err := InitializeSchema(db, logger)
	require.NoError(t, err)

	// Run migrations first time
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Get migration count
	var count1 int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count1)
	require.NoError(t, err)

	// Run migrations again
	err = RunMigrations(db, logger)
	require.NoError(t, err, "Running migrations twice should be safe")

	// Count should be the same
	var count2 int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count2)
	require.NoError(t, err)

	assert.Equal(t, count1, count2, "Migration count should not change on second run")
}

// TestMigration003AppliesCorrectly tests that migration 003 applies the correct schema changes
func TestMigration003AppliesCorrectly(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema
	err := InitializeSchema(db, logger)
	require.NoError(t, err)

	// Run migrations
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Verify connection_templates has new columns
	t.Run("ConnectionTemplatesColumns", func(t *testing.T) {
		columns := getTableColumns(t, db, "connection_templates")

		assert.Contains(t, columns, "organization_id", "Should have organization_id column")
		assert.Contains(t, columns, "visibility", "Should have visibility column")
		assert.Contains(t, columns, "created_by", "Should have created_by column")
	})

	// Verify saved_queries_sync has new columns
	t.Run("SavedQueriesColumns", func(t *testing.T) {
		columns := getTableColumns(t, db, "saved_queries_sync")

		assert.Contains(t, columns, "organization_id", "Should have organization_id column")
		assert.Contains(t, columns, "visibility", "Should have visibility column")
		assert.Contains(t, columns, "created_by", "Should have created_by column")
	})

	// Verify indexes were created
	t.Run("Indexes", func(t *testing.T) {
		indexes := getIndexes(t, db, "connection_templates")
		assert.Contains(t, indexes, "idx_connections_org_visibility")
		assert.Contains(t, indexes, "idx_connections_created_by")

		indexes = getIndexes(t, db, "saved_queries_sync")
		assert.Contains(t, indexes, "idx_queries_org_visibility")
		assert.Contains(t, indexes, "idx_queries_created_by")
	})
}

// TestMigration003DataMigration tests that existing data is migrated correctly
func TestMigration003DataMigration(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema
	err := InitializeSchema(db, logger)
	require.NoError(t, err)

	// Insert test user
	userID := "test-user-123"
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "testuser", "test@example.com", "hash", time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Insert test connection before migration
	connectionID := "test-conn-123"
	_, err = db.Exec(`
		INSERT INTO connection_templates (id, user_id, name, type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, connectionID, userID, "Test Connection", "postgresql", time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Insert test query before migration
	queryID := "test-query-123"
	_, err = db.Exec(`
		INSERT INTO saved_queries_sync (id, user_id, title, query, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, queryID, userID, "Test Query", "SELECT 1", time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Run migrations
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Verify connection created_by was set
	var connCreatedBy, connVisibility string
	err = db.QueryRow("SELECT created_by, visibility FROM connection_templates WHERE id = ?", connectionID).
		Scan(&connCreatedBy, &connVisibility)
	require.NoError(t, err)
	assert.Equal(t, userID, connCreatedBy, "created_by should be set to user_id")
	assert.Equal(t, "personal", connVisibility, "visibility should default to personal")

	// Verify query created_by was set
	var queryCreatedBy, queryVisibility string
	err = db.QueryRow("SELECT created_by, visibility FROM saved_queries_sync WHERE id = ?", queryID).
		Scan(&queryCreatedBy, &queryVisibility)
	require.NoError(t, err)
	assert.Equal(t, userID, queryCreatedBy, "created_by should be set to user_id")
	assert.Equal(t, "personal", queryVisibility, "visibility should default to personal")
}

// TestGetMigrationStatus tests retrieving migration status
func TestGetMigrationStatus(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema
	err := InitializeSchema(db, logger)
	require.NoError(t, err)

	// Get status before migrations
	statuses, err := GetMigrationStatus(db)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(statuses), len(Migrations), "Should have status for all migrations")

	for _, status := range statuses {
		assert.Equal(t, "pending", status.Status, "All migrations should be pending initially")
	}

	// Run migrations
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Get status after migrations
	statuses, err = GetMigrationStatus(db)
	require.NoError(t, err)

	appliedCount := 0
	for _, status := range statuses {
		if status.Status == "applied" {
			appliedCount++
			assert.NotNil(t, status.AppliedAt, "Applied migration should have timestamp")
		}
	}

	assert.Greater(t, appliedCount, 0, "At least one migration should be applied")
}

// TestVisibilityConstraint tests the visibility CHECK constraint
func TestVisibilityConstraint(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema and run migrations
	err := InitializeSchema(db, logger)
	require.NoError(t, err)
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Insert test user
	userID := "test-user-456"
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "testuser2", "test2@example.com", "hash", time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Test valid visibility values
	t.Run("ValidVisibility", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO connection_templates (id, user_id, name, type, visibility, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "conn-personal", userID, "Personal", "postgresql", "personal", userID, time.Now().Unix(), time.Now().Unix())
		assert.NoError(t, err, "Should accept 'personal' visibility")

		_, err = db.Exec(`
			INSERT INTO connection_templates (id, user_id, name, type, visibility, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "conn-shared", userID, "Shared", "postgresql", "shared", userID, time.Now().Unix(), time.Now().Unix())
		assert.NoError(t, err, "Should accept 'shared' visibility")
	})

	// Test invalid visibility value
	t.Run("InvalidVisibility", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO connection_templates (id, user_id, name, type, visibility, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, "conn-invalid", userID, "Invalid", "postgresql", "public", userID, time.Now().Unix(), time.Now().Unix())
		assert.Error(t, err, "Should reject invalid visibility value")
	})
}

// TestOrganizationForeignKey tests the organization_id foreign key constraint
func TestOrganizationForeignKey(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema and run migrations
	err := InitializeSchema(db, logger)
	require.NoError(t, err)
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Insert test user
	userID := "test-user-789"
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "testuser3", "test3@example.com", "hash", time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Insert test organization
	orgID := "test-org-123"
	_, err = db.Exec(`
		INSERT INTO organizations (id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, orgID, "Test Org", userID, time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Test connection with valid organization
	t.Run("ValidOrganization", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO connection_templates (id, user_id, name, type, organization_id, visibility, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "conn-org", userID, "Org Connection", "postgresql", orgID, "shared", userID, time.Now().Unix(), time.Now().Unix())
		assert.NoError(t, err, "Should accept valid organization_id")
	})

	// Test connection with invalid organization
	t.Run("InvalidOrganization", func(t *testing.T) {
		_, err := db.Exec(`
			INSERT INTO connection_templates (id, user_id, name, type, organization_id, visibility, created_by, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, "conn-invalid-org", userID, "Invalid Org", "postgresql", "non-existent-org", "shared", userID, time.Now().Unix(), time.Now().Unix())
		assert.Error(t, err, "Should reject invalid organization_id")
	})
}

// TestCascadeDelete tests that deleting an organization cascades to connections and queries
func TestCascadeDelete(t *testing.T) {
	db, logger := setupTestDB(t)
	defer db.Close()

	// Initialize schema and run migrations
	err := InitializeSchema(db, logger)
	require.NoError(t, err)
	err = RunMigrations(db, logger)
	require.NoError(t, err)

	// Insert test user
	userID := "test-user-cascade"
	_, err = db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "cascadeuser", "cascade@example.com", "hash", time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Insert test organization
	orgID := "test-org-cascade"
	_, err = db.Exec(`
		INSERT INTO organizations (id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, orgID, "Cascade Org", userID, time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Insert connection with organization
	_, err = db.Exec(`
		INSERT INTO connection_templates (id, user_id, name, type, organization_id, visibility, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "conn-cascade", userID, "Cascade Connection", "postgresql", orgID, "shared", userID, time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Insert query with organization
	_, err = db.Exec(`
		INSERT INTO saved_queries_sync (id, user_id, title, query, organization_id, visibility, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "query-cascade", userID, "Cascade Query", "SELECT 1", orgID, "shared", userID, time.Now().Unix(), time.Now().Unix())
	require.NoError(t, err)

	// Verify resources exist
	var connCount, queryCount int
	err = db.QueryRow("SELECT COUNT(*) FROM connection_templates WHERE organization_id = ?", orgID).Scan(&connCount)
	require.NoError(t, err)
	assert.Equal(t, 1, connCount)

	err = db.QueryRow("SELECT COUNT(*) FROM saved_queries_sync WHERE organization_id = ?", orgID).Scan(&queryCount)
	require.NoError(t, err)
	assert.Equal(t, 1, queryCount)

	// Delete organization
	_, err = db.Exec("DELETE FROM organizations WHERE id = ?", orgID)
	require.NoError(t, err)

	// Verify resources were cascade deleted
	err = db.QueryRow("SELECT COUNT(*) FROM connection_templates WHERE organization_id = ?", orgID).Scan(&connCount)
	require.NoError(t, err)
	assert.Equal(t, 0, connCount, "Connection should be cascade deleted")

	err = db.QueryRow("SELECT COUNT(*) FROM saved_queries_sync WHERE organization_id = ?", orgID).Scan(&queryCount)
	require.NoError(t, err)
	assert.Equal(t, 0, queryCount, "Query should be cascade deleted")
}

// Helper functions

func getTableColumns(t *testing.T, db *sql.DB, tableName string) []string {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	require.NoError(t, err)
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk)
		require.NoError(t, err)
		columns = append(columns, name)
	}
	return columns
}

func getIndexes(t *testing.T, db *sql.DB, tableName string) []string {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name=?", tableName)
	require.NoError(t, err)
	defer rows.Close()

	var indexes []string
	for rows.Next() {
		var name string
		err := rows.Scan(&name)
		require.NoError(t, err)
		indexes = append(indexes, name)
	}
	return indexes
}
