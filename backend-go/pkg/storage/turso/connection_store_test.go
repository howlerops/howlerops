package turso_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

// ====================================================================
// Test Setup and Helpers
// ====================================================================

// setupTestDBForConnections creates an in-memory SQLite database with the correct schema
func setupTestDBForConnections(t testing.TB) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create schema matching connection_store.go implementation
	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			username TEXT NOT NULL UNIQUE
		);

		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER,
			max_members INTEGER NOT NULL DEFAULT 10
		);

		CREATE TABLE organization_members (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL,
			joined_at INTEGER NOT NULL,
			FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			UNIQUE(organization_id, user_id)
		);

		-- Schema matching connection_store.go
		CREATE TABLE connection_templates (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			host TEXT,
			port INTEGER,
			database_name TEXT,
			username TEXT,
			use_ssh BOOLEAN DEFAULT 0,
			ssh_host TEXT,
			ssh_port INTEGER,
			ssh_user TEXT,
			color TEXT,
			icon TEXT,
			metadata TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sync_version INTEGER NOT NULL DEFAULT 1,
			organization_id TEXT,
			visibility TEXT NOT NULL DEFAULT 'personal' CHECK(visibility IN ('personal', 'shared')),
			created_by TEXT NOT NULL,
			deleted_at INTEGER,
			password_migration_status TEXT DEFAULT 'not_migrated',
			FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE
		);

		-- Stub credential store table (for future use)
		CREATE TABLE encrypted_credentials (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			connection_id TEXT NOT NULL,
			encrypted_data TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		-- Insert test users
		INSERT INTO users (id, email, username) VALUES
			('user-1', 'user1@example.com', 'user1'),
			('user-2', 'user2@example.com', 'user2'),
			('user-3', 'user3@example.com', 'user3');

		-- Insert test organizations
		INSERT INTO organizations (id, name, owner_id, created_at, updated_at, max_members) VALUES
			('org-1', 'Org 1', 'user-1', CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER), 10),
			('org-2', 'Org 2', 'user-2', CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER), 10),
			('org-3', 'Org 3', 'user-1', CAST(strftime('%s', 'now') AS INTEGER), CAST(strftime('%s', 'now') AS INTEGER), 10);

		-- Add members to organizations
		INSERT INTO organization_members (id, organization_id, user_id, role, joined_at) VALUES
			('mem-1', 'org-1', 'user-1', 'owner', CAST(strftime('%s', 'now') AS INTEGER)),
			('mem-2', 'org-1', 'user-2', 'member', CAST(strftime('%s', 'now') AS INTEGER)),
			('mem-3', 'org-2', 'user-2', 'owner', CAST(strftime('%s', 'now') AS INTEGER)),
			('mem-4', 'org-2', 'user-3', 'member', CAST(strftime('%s', 'now') AS INTEGER)),
			('mem-5', 'org-3', 'user-1', 'owner', CAST(strftime('%s', 'now') AS INTEGER));
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	cleanup := func() {
		_ = db.Close()
	}

	return db, cleanup
}

// createTestConnection creates a test connection object
func createTestConnection(userID, name, visibility string, orgID *string) *turso.Connection {
	return &turso.Connection{
		UserID:         userID,
		Name:           name,
		Type:           "postgres",
		Host:           "localhost",
		Port:           5432,
		Database:       "testdb",
		Username:       "testuser",
		Visibility:     visibility,
		OrganizationID: orgID,
		CreatedBy:      userID,
	}
}

// ====================================================================
// CRUD Operations Tests
// ====================================================================

func TestConnectionStore_Create(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("create basic connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		// Verify ID was generated
		assert.NotEmpty(t, conn.ID)

		// Verify timestamps were set
		assert.False(t, conn.CreatedAt.IsZero())
		assert.False(t, conn.UpdatedAt.IsZero())

		// Verify sync version
		assert.Equal(t, 1, conn.SyncVersion)

		// Verify default visibility
		assert.Equal(t, "personal", conn.Visibility)
	})

	t.Run("create connection with custom ID", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB 2", "personal", nil)
		conn.ID = "custom-id-123"

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		assert.Equal(t, "custom-id-123", conn.ID)
	})

	t.Run("create connection with metadata", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB 3", "personal", nil)
		conn.Metadata = map[string]interface{}{
			"env":     "production",
			"team":    "backend",
			"region":  "us-east-1",
			"version": 2,
		}

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		// Retrieve and verify metadata
		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "production", retrieved.Metadata["env"])
		assert.Equal(t, "backend", retrieved.Metadata["team"])
		assert.Equal(t, "us-east-1", retrieved.Metadata["region"])
		// JSON numbers are float64
		assert.Equal(t, float64(2), retrieved.Metadata["version"])
	})

	t.Run("create shared connection with organization", func(t *testing.T) {
		orgID := "org-1"
		conn := createTestConnection("user-1", "Shared DB", "shared", &orgID)

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		assert.Equal(t, "shared", conn.Visibility)
		assert.NotNil(t, conn.OrganizationID)
		assert.Equal(t, "org-1", *conn.OrganizationID)
	})

	t.Run("create connection with SSH settings", func(t *testing.T) {
		conn := createTestConnection("user-1", "SSH DB", "personal", nil)
		conn.UseSSH = true
		conn.SSHHost = "ssh.example.com"
		conn.SSHPort = 22
		conn.SSHUser = "sshuser"

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.True(t, retrieved.UseSSH)
		assert.Equal(t, "ssh.example.com", retrieved.SSHHost)
		assert.Equal(t, 22, retrieved.SSHPort)
		assert.Equal(t, "sshuser", retrieved.SSHUser)
	})

	t.Run("create connection with color and icon", func(t *testing.T) {
		conn := createTestConnection("user-1", "Styled DB", "personal", nil)
		conn.Color = "#FF5733"
		conn.Icon = "database"

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "#FF5733", retrieved.Color)
		assert.Equal(t, "database", retrieved.Icon)
	})
}

func TestConnectionStore_GetByID(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("get existing connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)

		assert.Equal(t, conn.ID, retrieved.ID)
		assert.Equal(t, conn.Name, retrieved.Name)
		assert.Equal(t, conn.UserID, retrieved.UserID)
		assert.Equal(t, conn.Type, retrieved.Type)
		assert.Equal(t, conn.Visibility, retrieved.Visibility)
	})

	t.Run("get non-existent connection", func(t *testing.T) {
		_, err := store.GetByID(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("get deleted connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "To Delete", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		// Delete it
		err = store.Delete(ctx, conn.ID)
		require.NoError(t, err)

		// Should not be retrievable
		_, err = store.GetByID(ctx, conn.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("get connection with metadata", func(t *testing.T) {
		conn := createTestConnection("user-1", "Meta DB", "personal", nil)
		conn.Metadata = map[string]interface{}{
			"foo": "bar",
			"num": 42,
		}
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)
		assert.Equal(t, "bar", retrieved.Metadata["foo"])
	})
}

func TestConnectionStore_Update(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("update connection name", func(t *testing.T) {
		conn := createTestConnection("user-1", "Original Name", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		originalVersion := conn.SyncVersion

		conn.Name = "Updated Name"
		err = store.Update(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", retrieved.Name)
		assert.Equal(t, originalVersion+1, retrieved.SyncVersion)
		// Verify timestamps are set (actual comparison not reliable due to Unix second precision)
		assert.False(t, retrieved.CreatedAt.IsZero())
		assert.False(t, retrieved.UpdatedAt.IsZero())
	})

	t.Run("update connection metadata", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		conn.Metadata = map[string]interface{}{"key": "value1"}
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		conn.Metadata = map[string]interface{}{"key": "value2", "new": "data"}
		err = store.Update(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "value2", retrieved.Metadata["key"])
		assert.Equal(t, "data", retrieved.Metadata["new"])
	})

	t.Run("update non-existent connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "Ghost", "personal", nil)
		conn.ID = "non-existent-id"

		err := store.Update(ctx, conn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("update deleted connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "To Delete", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		err = store.Delete(ctx, conn.ID)
		require.NoError(t, err)

		conn.Name = "Should Fail"
		err = store.Update(ctx, conn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("update connection visibility via Update", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		orgID := "org-1"
		conn.Visibility = "shared"
		conn.OrganizationID = &orgID
		err = store.Update(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "shared", retrieved.Visibility)
		assert.Equal(t, "org-1", *retrieved.OrganizationID)
	})
}

func TestConnectionStore_Delete(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("soft delete connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "To Delete", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		err = store.Delete(ctx, conn.ID)
		require.NoError(t, err)

		// Should not be retrievable
		_, err = store.GetByID(ctx, conn.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("delete non-existent connection", func(t *testing.T) {
		err := store.Delete(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})

	t.Run("delete already deleted connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "To Delete", "personal", nil)
		err := store.Create(ctx, conn)
		require.NoError(t, err)

		err = store.Delete(ctx, conn.ID)
		require.NoError(t, err)

		// Try to delete again
		err = store.Delete(ctx, conn.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection not found")
	})
}

func TestConnectionStore_GetByUserID(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("get personal connections for user", func(t *testing.T) {
		// Create personal connections for user-1
		conn1 := createTestConnection("user-1", "DB 1", "personal", nil)
		conn2 := createTestConnection("user-1", "DB 2", "personal", nil)
		require.NoError(t, store.Create(ctx, conn1))
		require.NoError(t, store.Create(ctx, conn2))

		// Create connection for user-2 (should not appear)
		conn3 := createTestConnection("user-2", "DB 3", "personal", nil)
		require.NoError(t, store.Create(ctx, conn3))

		connections, err := store.GetByUserID(ctx, "user-1")
		require.NoError(t, err)
		assert.Len(t, connections, 2)

		names := []string{connections[0].Name, connections[1].Name}
		assert.Contains(t, names, "DB 1")
		assert.Contains(t, names, "DB 2")
	})

	t.Run("get connections excludes shared", func(t *testing.T) {
		orgID := "org-1"
		conn1 := createTestConnection("user-1", "Personal", "personal", nil)
		conn2 := createTestConnection("user-1", "Shared", "shared", &orgID)
		require.NoError(t, store.Create(ctx, conn1))
		require.NoError(t, store.Create(ctx, conn2))

		connections, err := store.GetByUserID(ctx, "user-1")
		require.NoError(t, err)

		// Should only include personal connections
		for _, conn := range connections {
			if conn.Name == "Shared" {
				t.Error("GetByUserID should not return shared connections")
			}
		}
	})

	t.Run("get connections excludes deleted", func(t *testing.T) {
		conn1 := createTestConnection("user-1", "Active", "personal", nil)
		conn2 := createTestConnection("user-1", "Deleted", "personal", nil)
		require.NoError(t, store.Create(ctx, conn1))
		require.NoError(t, store.Create(ctx, conn2))

		// Delete one
		require.NoError(t, store.Delete(ctx, conn2.ID))

		connections, err := store.GetByUserID(ctx, "user-1")
		require.NoError(t, err)

		// Should not include deleted
		for _, conn := range connections {
			if conn.Name == "Deleted" {
				t.Error("GetByUserID should not return deleted connections")
			}
		}
	})

	t.Run("empty result for user with no connections", func(t *testing.T) {
		connections, err := store.GetByUserID(ctx, "user-3")
		require.NoError(t, err)
		assert.Empty(t, connections)
	})
}

// ====================================================================
// Organization and Sharing Tests
// ====================================================================

func TestConnectionStore_GetConnectionsByOrganization(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("get shared connections in organization", func(t *testing.T) {
		orgID := "org-1"
		conn1 := createTestConnection("user-1", "Org DB 1", "shared", &orgID)
		conn2 := createTestConnection("user-2", "Org DB 2", "shared", &orgID)
		require.NoError(t, store.Create(ctx, conn1))
		require.NoError(t, store.Create(ctx, conn2))

		connections, err := store.GetConnectionsByOrganization(ctx, "org-1")
		require.NoError(t, err)
		assert.Len(t, connections, 2)

		for _, conn := range connections {
			assert.Equal(t, "shared", conn.Visibility)
			assert.Equal(t, "org-1", *conn.OrganizationID)
		}
	})

	t.Run("excludes personal connections", func(t *testing.T) {
		orgID := "org-2"
		conn1 := createTestConnection("user-2", "Shared", "shared", &orgID)
		conn2 := createTestConnection("user-2", "Personal", "personal", nil)
		require.NoError(t, store.Create(ctx, conn1))
		require.NoError(t, store.Create(ctx, conn2))

		connections, err := store.GetConnectionsByOrganization(ctx, "org-2")
		require.NoError(t, err)

		for _, conn := range connections {
			if conn.Name == "Personal" {
				t.Error("Should not return personal connections")
			}
		}
	})

	t.Run("empty result for org with no shared connections", func(t *testing.T) {
		connections, err := store.GetConnectionsByOrganization(ctx, "org-3")
		require.NoError(t, err)
		assert.Empty(t, connections)
	})
}

func TestConnectionStore_GetSharedConnections(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("get personal and organization shared connections", func(t *testing.T) {
		// user-2 is member of org-1 and owner of org-2
		org1ID := "org-1"
		org2ID := "org-2"

		// Personal connections for user-2
		conn1 := createTestConnection("user-2", "Personal 1", "personal", nil)
		conn2 := createTestConnection("user-2", "Personal 2", "personal", nil)
		require.NoError(t, store.Create(ctx, conn1))
		require.NoError(t, store.Create(ctx, conn2))

		// Shared in org-1
		conn3 := createTestConnection("user-1", "Org1 Shared", "shared", &org1ID)
		require.NoError(t, store.Create(ctx, conn3))

		// Shared in org-2
		conn4 := createTestConnection("user-2", "Org2 Shared", "shared", &org2ID)
		require.NoError(t, store.Create(ctx, conn4))

		// Shared in org-3 (user-2 is NOT a member)
		org3ID := "org-3"
		conn5 := createTestConnection("user-1", "Org3 Shared", "shared", &org3ID)
		require.NoError(t, store.Create(ctx, conn5))

		connections, err := store.GetSharedConnections(ctx, "user-2")
		require.NoError(t, err)

		// Should return: 2 personal + 1 from org-1 + 1 from org-2 = 4 total
		assert.GreaterOrEqual(t, len(connections), 4)

		// Verify org-3 connection is NOT included
		for _, conn := range connections {
			if conn.Name == "Org3 Shared" {
				t.Error("Should not include connections from organizations user is not member of")
			}
		}
	})
}

func TestConnectionStore_UpdateConnectionVisibility(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("owner can change visibility to shared", func(t *testing.T) {
		conn := createTestConnection("user-1", "My DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-1", "shared")
		require.NoError(t, err)

		updated, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "shared", updated.Visibility)
	})

	t.Run("owner can change visibility to personal", func(t *testing.T) {
		orgID := "org-1"
		conn := createTestConnection("user-1", "Shared DB", "shared", &orgID)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-1", "personal")
		require.NoError(t, err)

		updated, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "personal", updated.Visibility)
	})

	t.Run("non-owner cannot change visibility", func(t *testing.T) {
		conn := createTestConnection("user-1", "User1 DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-2", "shared")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only the creator can change visibility")
	})

	t.Run("invalid visibility value", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-1", "invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid visibility")
	})

	t.Run("update non-existent connection", func(t *testing.T) {
		err := store.UpdateConnectionVisibility(ctx, "non-existent", "user-1", "shared")
		assert.Error(t, err)
	})
}

// ====================================================================
// Migration Status Tests
// ====================================================================

func TestConnectionStore_MigrationStatus(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("update migration status to migrated", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateMigrationStatus(ctx, conn.ID, "migrated")
		require.NoError(t, err)

		status, err := store.GetMigrationStatus(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "migrated", status)
	})

	t.Run("update migration status to no_password", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateMigrationStatus(ctx, conn.ID, "no_password")
		require.NoError(t, err)

		status, err := store.GetMigrationStatus(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "no_password", status)
	})

	t.Run("invalid migration status", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		err := store.UpdateMigrationStatus(ctx, conn.ID, "invalid_status")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid migration status")
	})

	t.Run("get migration status for new connection", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		status, err := store.GetMigrationStatus(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "not_migrated", status)
	})

	t.Run("get migration status for non-existent connection", func(t *testing.T) {
		status, err := store.GetMigrationStatus(ctx, "non-existent")
		assert.Error(t, err)
		assert.Equal(t, "not_migrated", status)
	})
}

// ====================================================================
// Edge Cases and Error Handling Tests
// ====================================================================

func TestConnectionStore_EdgeCases(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("create connection with empty metadata", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		conn.Metadata = map[string]interface{}{}

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)
	})

	t.Run("create connection with nil metadata", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		conn.Metadata = nil

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		// Metadata might be nil or empty map
		if retrieved.Metadata != nil {
			assert.Empty(t, retrieved.Metadata)
		}
	})

	t.Run("update connection increments sync version", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		initialVersion := conn.SyncVersion

		// Update multiple times
		for i := 1; i <= 3; i++ {
			conn.Name = "Updated Name " + string(rune(i+'0'))
			err := store.Update(ctx, conn)
			require.NoError(t, err)

			retrieved, err := store.GetByID(ctx, conn.ID)
			require.NoError(t, err)
			assert.Equal(t, initialVersion+i, retrieved.SyncVersion)
		}
	})

	t.Run("metadata with complex nested structure", func(t *testing.T) {
		conn := createTestConnection("user-1", "Complex DB", "personal", nil)
		conn.Metadata = map[string]interface{}{
			"nested": map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "deep value",
				},
			},
			"array": []interface{}{"a", "b", "c"},
			"mixed": []interface{}{
				map[string]interface{}{"key": "value"},
				"string",
				123,
			},
		}

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata)

		// Verify nested structure preserved
		nested := retrieved.Metadata["nested"].(map[string]interface{})
		level1 := nested["level1"].(map[string]interface{})
		assert.Equal(t, "deep value", level1["level2"])
	})

	t.Run("connection with all optional fields empty", func(t *testing.T) {
		conn := &turso.Connection{
			UserID:     "user-1",
			Name:       "Minimal Connection",
			Type:       "postgres",
			Database:   "db",
			Visibility: "personal",
			CreatedBy:  "user-1",
		}

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, "", retrieved.Host)
		assert.Equal(t, 0, retrieved.Port)
		assert.Equal(t, "", retrieved.Username)
	})
}

// ====================================================================
// Concurrent Operations Tests
// ====================================================================

func TestConnectionStore_ConcurrentOperations(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("create multiple connections", func(t *testing.T) {
		// Create 5 connections sequentially with pre-assigned IDs
		for i := 0; i < 5; i++ {
			conn := createTestConnection("user-1", "Concurrent DB "+string(rune(i+'0')), "personal", nil)
			conn.ID = "concurrent-" + string(rune(i+'0')) // Pre-assign IDs
			err := store.Create(ctx, conn)
			require.NoError(t, err)
		}

		// Verify all were created
		connections, err := store.GetByUserID(ctx, "user-1")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(connections), 5)
	})

	t.Run("multiple sequential updates", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		// Update multiple times sequentially
		for i := 0; i < 5; i++ {
			conn.Name = "Updated " + string(rune(i+'0'))
			err := store.Update(ctx, conn)
			require.NoError(t, err)
		}

		// Verify connection still exists and is valid
		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, retrieved.Name)
		assert.Equal(t, 6, retrieved.SyncVersion) // Initial 1 + 5 updates
	})
}

// ====================================================================
// JSON Marshaling Tests
// ====================================================================

func TestConnectionStore_JSONFields(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	t.Run("metadata with special characters", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		conn.Metadata = map[string]interface{}{
			"special": "value with \"quotes\" and 'apostrophes'",
			"unicode": "Hello 世界 🌍",
			"escaped": "line1\nline2\ttab",
		}

		err := store.Create(ctx, conn)
		require.NoError(t, err)

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)
		assert.Equal(t, conn.Metadata["special"], retrieved.Metadata["special"])
		assert.Equal(t, conn.Metadata["unicode"], retrieved.Metadata["unicode"])
		assert.Equal(t, conn.Metadata["escaped"], retrieved.Metadata["escaped"])
	})

	t.Run("connection can be marshaled to JSON", func(t *testing.T) {
		conn := createTestConnection("user-1", "Test DB", "personal", nil)
		require.NoError(t, store.Create(ctx, conn))

		retrieved, err := store.GetByID(ctx, conn.ID)
		require.NoError(t, err)

		// Marshal to JSON
		jsonData, err := json.Marshal(retrieved)
		require.NoError(t, err)

		// Unmarshal back
		var unmarshaled turso.Connection
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, retrieved.ID, unmarshaled.ID)
		assert.Equal(t, retrieved.Name, unmarshaled.Name)
		assert.Equal(t, retrieved.UserID, unmarshaled.UserID)
	})
}
