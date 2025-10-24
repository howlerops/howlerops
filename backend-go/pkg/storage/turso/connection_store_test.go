package turso_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/sql-studio/backend-go/internal/sync"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// Test Setup and Helpers
// ====================================================================

func setupTestDBForConnections(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Create full schema including organizations and shared resources
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

		CREATE TABLE connection_templates (
			connection_id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			host TEXT NOT NULL,
			port INTEGER NOT NULL,
			database TEXT NOT NULL,
			username TEXT NOT NULL,
			ssl_mode TEXT NOT NULL DEFAULT 'disable',
			visibility TEXT NOT NULL DEFAULT 'personal' CHECK (visibility IN ('personal', 'shared')),
			organization_id TEXT,
			shared_by TEXT,
			shared_at INTEGER,
			last_used_at INTEGER DEFAULT 0,
			sync_version INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER DEFAULT 0,
			updated_at INTEGER DEFAULT 0,
			synced_at INTEGER DEFAULT 0,
			deleted_at INTEGER,
			FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE CASCADE,
			CHECK (type IN ('postgres', 'mysql', 'sqlite', 'mssql', 'oracle', 'mongodb')),
			CHECK (ssl_mode IN ('disable', 'require', 'verify-ca', 'verify-full')),
			CHECK (port > 0 AND port < 65536)
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
		db.Close()
	}

	return db, cleanup
}

func createTestConnection(userID, name, visibility string, orgID *string) *sync.ConnectionTemplate {
	now := time.Now()
	return &sync.ConnectionTemplate{
		ID:             "conn-" + name,
		UserID:         userID,
		Name:           name,
		Type:           "postgres",
		Host:           "localhost",
		Port:           5432,
		Database:       "testdb",
		Username:       "testuser",
		Visibility:     visibility,
		OrganizationID: orgID,
		CreatedAt:      now,
		UpdatedAt:      now,
		SyncVersion:    1,
	}
}

// ====================================================================
// Repository Layer Tests
// ====================================================================

func TestGetConnectionsByOrganization(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Setup: Create org with 2 members, 3 shared connections, 2 personal connections
	orgID := "org-1"

	// User 1 creates 2 connections: 1 shared, 1 personal
	conn1 := createTestConnection("user-1", "Shared DB 1", "shared", &orgID)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn1))

	conn2 := createTestConnection("user-1", "Personal DB 1", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn2))

	// User 2 creates 2 connections: 1 shared, 1 personal
	conn3 := createTestConnection("user-2", "Shared DB 2", "shared", &orgID)
	require.NoError(t, store.SaveConnection(ctx, "user-2", conn3))

	conn4 := createTestConnection("user-2", "Personal DB 2", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-2", conn4))

	// Test: Member 1 fetches org connections
	connections, err := store.GetConnectionsByOrganization(ctx, "org-1")

	// Verify: Returns only 2 shared connections, not personal ones
	require.NoError(t, err)
	assert.Len(t, connections, 2, "Should return only shared connections")

	for _, conn := range connections {
		assert.Equal(t, "shared", conn.Visibility, "All connections should be shared")
		assert.NotNil(t, conn.OrganizationID, "Shared connections should have org ID")
		assert.Equal(t, "org-1", *conn.OrganizationID, "Should be from the correct org")
	}
}

func TestGetSharedConnections(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Setup: User in 2 orgs, each org has shared connections
	org1ID := "org-1"
	org2ID := "org-2"

	// Personal connections for user-2
	conn1 := createTestConnection("user-2", "Personal DB 1", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-2", conn1))

	conn2 := createTestConnection("user-2", "Personal DB 2", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-2", conn2))

	// Shared connections in org-1 (user-2 is member)
	conn3 := createTestConnection("user-1", "Org1 Shared DB 1", "shared", &org1ID)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn3))

	conn4 := createTestConnection("user-1", "Org1 Shared DB 2", "shared", &org1ID)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn4))

	// Shared connections in org-2 (user-2 is owner)
	conn5 := createTestConnection("user-2", "Org2 Shared DB 1", "shared", &org2ID)
	require.NoError(t, store.SaveConnection(ctx, "user-2", conn5))

	// Shared connections in org-3 (user-2 is NOT a member)
	org3ID := "org-3"
	conn6 := createTestConnection("user-1", "Org3 Shared DB", "shared", &org3ID)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn6))

	// Test: Fetch all accessible connections for user-2
	orgIDs := []string{"org-1", "org-2"} // User-2's organizations
	connections, err := store.ListAccessibleConnections(ctx, "user-2", orgIDs, time.Time{})

	// Verify: Returns personal (2) + shared from org-1 (2) + shared from org-2 (1) = 5 total
	require.NoError(t, err)
	assert.Len(t, connections, 5, "Should return personal + shared from both orgs")

	personalCount := 0
	sharedOrg1Count := 0
	sharedOrg2Count := 0

	for _, conn := range connections {
		if conn.Visibility == "personal" {
			personalCount++
			assert.Equal(t, "user-2", conn.UserID, "Personal connections should belong to user")
		} else if conn.Visibility == "shared" && conn.OrganizationID != nil {
			if *conn.OrganizationID == "org-1" {
				sharedOrg1Count++
			} else if *conn.OrganizationID == "org-2" {
				sharedOrg2Count++
			} else if *conn.OrganizationID == "org-3" {
				t.Error("Should not include connections from org-3")
			}
		}
	}

	assert.Equal(t, 2, personalCount, "Should have 2 personal connections")
	assert.Equal(t, 2, sharedOrg1Count, "Should have 2 shared connections from org-1")
	assert.Equal(t, 1, sharedOrg2Count, "Should have 1 shared connection from org-2")
}

func TestUpdateConnectionVisibility_Authorized(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Setup: User owns a personal connection
	conn := createTestConnection("user-1", "My DB", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn))

	// Test: Change visibility to shared
	orgID := "org-1"
	conn.Visibility = "shared"
	conn.OrganizationID = &orgID
	err := store.UpdateConnectionVisibility(ctx, "user-1", conn.ID, "shared", &orgID)

	// Verify: Visibility updated, organization_id set
	require.NoError(t, err)

	updated, err := store.GetConnection(ctx, "user-1", conn.ID)
	require.NoError(t, err)
	assert.Equal(t, "shared", updated.Visibility, "Visibility should be updated to shared")
	assert.NotNil(t, updated.OrganizationID, "Organization ID should be set")
	assert.Equal(t, "org-1", *updated.OrganizationID, "Should be linked to correct org")
}

func TestUpdateConnectionVisibility_Unauthorized(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Setup: User 1 owns a connection
	conn := createTestConnection("user-1", "User1 DB", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn))

	// Test: User 2 tries to share User 1's connection
	orgID := "org-1"
	err := store.UpdateConnectionVisibility(ctx, "user-2", conn.ID, "shared", &orgID)

	// Verify: Returns error, no changes made
	assert.Error(t, err, "Should fail when non-owner tries to update visibility")
	assert.Contains(t, err.Error(), "not authorized", "Error should mention authorization")

	// Verify connection unchanged
	original, err := store.GetConnection(ctx, "user-1", conn.ID)
	require.NoError(t, err)
	assert.Equal(t, "personal", original.Visibility, "Visibility should remain personal")
	assert.Nil(t, original.OrganizationID, "Organization ID should remain nil")
}

func TestFilterByOrganization(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Setup: 10 connections across 3 orgs
	org1ID := "org-1"
	org2ID := "org-2"
	org3ID := "org-3"

	// 3 connections in org-1
	for i := 1; i <= 3; i++ {
		conn := createTestConnection("user-1", "Org1-Conn-"+string(rune(i+'0')), "shared", &org1ID)
		require.NoError(t, store.SaveConnection(ctx, "user-1", conn))
	}

	// 4 connections in org-2
	for i := 1; i <= 4; i++ {
		conn := createTestConnection("user-2", "Org2-Conn-"+string(rune(i+'0')), "shared", &org2ID)
		require.NoError(t, store.SaveConnection(ctx, "user-2", conn))
	}

	// 3 connections in org-3
	for i := 1; i <= 3; i++ {
		conn := createTestConnection("user-1", "Org3-Conn-"+string(rune(i+'0')), "shared", &org3ID)
		require.NoError(t, store.SaveConnection(ctx, "user-1", conn))
	}

	// Test: Filter by specific org ID (org-2)
	connections, err := store.GetConnectionsByOrganization(ctx, "org-2")

	// Verify: Only returns connections for that org
	require.NoError(t, err)
	assert.Len(t, connections, 4, "Should return exactly 4 connections from org-2")

	for _, conn := range connections {
		assert.NotNil(t, conn.OrganizationID, "Connections should have org ID")
		assert.Equal(t, "org-2", *conn.OrganizationID, "All connections should be from org-2")
	}
}

func TestConnectionVisibility_PersonalToShared(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Create personal connection
	conn := createTestConnection("user-1", "My DB", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn))

	// Share with organization
	orgID := "org-1"
	err := store.UpdateConnectionVisibility(ctx, "user-1", conn.ID, "shared", &orgID)
	require.NoError(t, err)

	// Verify it's now accessible to other org members
	orgConnections, err := store.GetConnectionsByOrganization(ctx, "org-1")
	require.NoError(t, err)
	assert.Len(t, orgConnections, 1, "Connection should now be shared")
	assert.Equal(t, conn.ID, orgConnections[0].ID)
}

func TestConnectionVisibility_SharedToPersonal(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Create shared connection
	orgID := "org-1"
	conn := createTestConnection("user-1", "Shared DB", "shared", &orgID)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn))

	// Unshare (make personal)
	err := store.UpdateConnectionVisibility(ctx, "user-1", conn.ID, "personal", nil)
	require.NoError(t, err)

	// Verify it's no longer accessible to org
	orgConnections, err := store.GetConnectionsByOrganization(ctx, "org-1")
	require.NoError(t, err)
	assert.Len(t, orgConnections, 0, "Connection should no longer be shared")

	// Verify owner can still access it
	personal, err := store.GetConnection(ctx, "user-1", conn.ID)
	require.NoError(t, err)
	assert.Equal(t, "personal", personal.Visibility)
	assert.Nil(t, personal.OrganizationID)
}

func TestConnectionStore_Pagination(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Create 50 connections
	for i := 1; i <= 50; i++ {
		conn := createTestConnection("user-1", "DB-"+string(rune(i+'0')), "personal", nil)
		conn.UpdatedAt = time.Now().Add(time.Duration(i) * time.Second)
		require.NoError(t, store.SaveConnection(ctx, "user-1", conn))
	}

	// Get first page (20 items)
	page1, err := store.ListConnectionsPaginated(ctx, "user-1", 20, 0)
	require.NoError(t, err)
	assert.Len(t, page1, 20)

	// Get second page
	page2, err := store.ListConnectionsPaginated(ctx, "user-1", 20, 20)
	require.NoError(t, err)
	assert.Len(t, page2, 20)

	// Verify no overlap
	page1IDs := make(map[string]bool)
	for _, conn := range page1 {
		page1IDs[conn.ID] = true
	}
	for _, conn := range page2 {
		assert.False(t, page1IDs[conn.ID], "Pages should not overlap")
	}
}

func TestConnectionStore_SoftDelete(t *testing.T) {
	db, cleanup := setupTestDBForConnections(t)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Create connection
	conn := createTestConnection("user-1", "My DB", "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, "user-1", conn))

	// Soft delete
	err := store.DeleteConnection(ctx, "user-1", conn.ID)
	require.NoError(t, err)

	// Verify not returned in normal queries
	connections, err := store.ListConnections(ctx, "user-1", time.Time{})
	require.NoError(t, err)
	assert.Len(t, connections, 0, "Deleted connection should not appear")

	// But can still be retrieved directly (for conflict resolution)
	deleted, err := store.GetConnectionIncludingDeleted(ctx, "user-1", conn.ID)
	require.NoError(t, err)
	assert.NotNil(t, deleted, "Deleted connection should still exist")
	assert.True(t, deleted.DeletedAt != nil, "Should have deletion timestamp")
}

// ====================================================================
// Benchmark Tests
// ====================================================================

func BenchmarkListAccessibleConnections(b *testing.B) {
	db, cleanup := setupTestDBForConnections(b)
	defer cleanup()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	store := turso.NewConnectionStore(db, logger)
	ctx := context.Background()

	// Setup: 1000 connections across 10 orgs
	for i := 0; i < 1000; i++ {
		orgID := "org-" + string(rune((i%10)+'0'))
		conn := createTestConnection("user-1", "DB-"+string(rune(i+'0')), "shared", &orgID)
		store.SaveConnection(ctx, "user-1", conn)
	}

	orgIDs := []string{"org-1", "org-2", "org-3"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.ListAccessibleConnections(ctx, "user-1", orgIDs, time.Time{})
	}
}
