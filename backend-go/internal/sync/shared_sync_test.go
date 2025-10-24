package sync_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/internal/sync"
	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// Sync Protocol Tests for Organization-Aware Sync
// ====================================================================

func TestSyncPull_FiltersbyOrgAccess(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	syncService := sync.NewService(store, sync.Config{
		EnableSanitization: true,
	}, testLogger())

	ctx := context.Background()

	// Setup: User in org-1 and org-2, not in org-3
	userID := "sync-user"
	org1ID := "org-sync-1"
	org2ID := "org-sync-2"
	org3ID := "org-sync-3"

	setupOrgMembership(t, db, org1ID, userID, "member")
	setupOrgMembership(t, db, org2ID, userID, "admin")
	setupOrgMembership(t, db, org3ID, "other-user", "owner")

	// Create connections in different orgs
	conn1 := createSyncConnection("conn-org1", userID, "shared", &org1ID)
	require.NoError(t, store.SaveConnection(ctx, userID, conn1))

	conn2 := createSyncConnection("conn-org2", userID, "shared", &org2ID)
	require.NoError(t, store.SaveConnection(ctx, userID, conn2))

	conn3 := createSyncConnection("conn-org3", "other-user", "shared", &org3ID)
	require.NoError(t, store.SaveConnection(ctx, "other-user", conn3))

	// Personal connection
	connPersonal := createSyncConnection("conn-personal", userID, "personal", nil)
	require.NoError(t, store.SaveConnection(ctx, userID, connPersonal))

	// Test: Sync pull with org IDs
	orgIDs := []string{org1ID, org2ID}
	connections, err := store.ListAccessibleConnections(ctx, userID, orgIDs, time.Time{})

	// Verify: Returns personal + org1 + org2, NOT org3
	require.NoError(t, err)
	assert.Len(t, connections, 3, "Should see personal + 2 org connections")

	foundOrg3 := false
	for _, conn := range connections {
		if conn.OrganizationID != nil && *conn.OrganizationID == org3ID {
			foundOrg3 = true
		}
	}
	assert.False(t, foundOrg3, "Should not see org-3 connections")
}

func TestSyncPush_ValidatesOrgMembership(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	syncService := sync.NewService(store, sync.Config{
		EnableSanitization: true,
	}, testLogger())

	ctx := context.Background()

	// Setup: User NOT a member of org
	userID := "non-member"
	orgID := "restricted-org"

	setupOrgMembership(t, db, orgID, "org-owner", "owner")

	// Test: User tries to push shared connection to org they're not in
	conn := createSyncConnection("conn-unauthorized", userID, "shared", &orgID)

	changes := []sync.SyncChange{
		{
			ItemType:    sync.SyncItemTypeConnection,
			ItemID:      conn.ID,
			Action:      sync.SyncActionCreate,
			Data:        conn,
			SyncVersion: 1,
			DeviceID:    "device-1",
			Timestamp:   time.Now(),
		},
	}

	uploadReq := &sync.SyncUploadRequest{
		UserID:   userID,
		DeviceID: "device-1",
		Changes:  changes,
	}

	// Execute sync upload
	resp, err := syncService.Upload(ctx, uploadReq)

	// Verify: Should reject or have conflicts
	// Note: Implementation might vary - either error or in rejected list
	if err == nil {
		assert.Greater(t, len(resp.Rejected), 0, "Should reject unauthorized org connection")
	}
}

func TestConflictResolution_LastWriteWins(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	syncService := sync.NewService(store, sync.Config{
		ConflictStrategy:   sync.ConflictResolutionLastWriteWins,
		EnableSanitization: false,
	}, testLogger())

	ctx := context.Background()

	userID := "conflict-user"
	deviceID1 := "device-1"
	deviceID2 := "device-2"

	// Setup: Connection exists with version 1
	existingConn := createSyncConnection("conn-conflict", userID, "personal", nil)
	existingConn.SyncVersion = 1
	existingConn.UpdatedAt = time.Now().Add(-1 * time.Hour) // Older
	require.NoError(t, store.SaveConnection(ctx, userID, existingConn))

	// Test: Device 1 pushes update with newer timestamp
	newConn := createSyncConnection("conn-conflict", userID, "personal", nil)
	newConn.Name = "Updated from Device 1"
	newConn.SyncVersion = 2
	newConn.UpdatedAt = time.Now() // Newer

	changes := []sync.SyncChange{
		{
			ItemType:    sync.SyncItemTypeConnection,
			ItemID:      newConn.ID,
			Action:      sync.SyncActionUpdate,
			Data:        newConn,
			SyncVersion: 2,
			DeviceID:    deviceID1,
			Timestamp:   newConn.UpdatedAt,
		},
	}

	uploadReq := &sync.SyncUploadRequest{
		UserID:   userID,
		DeviceID: deviceID1,
		Changes:  changes,
	}

	resp, err := syncService.Upload(ctx, uploadReq)
	require.NoError(t, err)

	// Verify: Update applied (newer timestamp wins)
	assert.Len(t, resp.Conflicts, 0, "No conflicts with last-write-wins strategy")

	// Verify connection updated
	updated, err := store.GetConnection(ctx, userID, "conn-conflict")
	require.NoError(t, err)
	assert.Equal(t, "Updated from Device 1", updated.Name)
}

func TestConflictResolution_DetectsConflict(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	syncService := sync.NewService(store, sync.Config{
		ConflictStrategy:   sync.ConflictResolutionUserChoice,
		EnableSanitization: false,
	}, testLogger())

	ctx := context.Background()

	userID := "conflict-detect-user"

	// Setup: Connection with version 5
	existingConn := createSyncConnection("conn-detect", userID, "personal", nil)
	existingConn.SyncVersion = 5
	existingConn.Name = "Server Version"
	existingConn.UpdatedAt = time.Now().Add(-30 * time.Minute)
	require.NoError(t, store.SaveConnection(ctx, userID, existingConn))

	// Test: Client pushes older version
	oldConn := createSyncConnection("conn-detect", userID, "personal", nil)
	oldConn.SyncVersion = 3 // Older version
	oldConn.Name = "Client Version"
	oldConn.UpdatedAt = time.Now().Add(-1 * time.Hour)

	changes := []sync.SyncChange{
		{
			ItemType:    sync.SyncItemTypeConnection,
			ItemID:      oldConn.ID,
			Action:      sync.SyncActionUpdate,
			Data:        oldConn,
			SyncVersion: 3,
			DeviceID:    "device-old",
			Timestamp:   oldConn.UpdatedAt,
		},
	}

	uploadReq := &sync.SyncUploadRequest{
		UserID:   userID,
		DeviceID: "device-old",
		Changes:  changes,
	}

	resp, err := syncService.Upload(ctx, uploadReq)
	require.NoError(t, err)

	// Verify: Conflict detected
	assert.Greater(t, len(resp.Conflicts), 0, "Should detect version conflict")

	conflict := resp.Conflicts[0]
	assert.Equal(t, sync.SyncItemTypeConnection, conflict.ItemType)
	assert.Equal(t, "conn-detect", conflict.ItemID)
}

func TestSyncSharedQuery_OrgFiltering(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	ctx := context.Background()

	userID := "query-user"
	org1ID := "org-query-1"
	org2ID := "org-query-2"

	setupOrgMembership(t, db, org1ID, userID, "member")
	setupOrgMembership(t, db, org2ID, "other-user", "owner")

	// Personal query
	personalQuery := createSyncQuery("query-personal", userID, "personal", nil)
	require.NoError(t, store.SaveQuery(ctx, userID, personalQuery))

	// Shared in user's org
	sharedQuery1 := createSyncQuery("query-org1", "other-user", "shared", &org1ID)
	require.NoError(t, store.SaveQuery(ctx, "other-user", sharedQuery1))

	// Shared in org user is NOT in
	sharedQuery2 := createSyncQuery("query-org2", "other-user", "shared", &org2ID)
	require.NoError(t, store.SaveQuery(ctx, "other-user", sharedQuery2))

	// Test: Pull queries
	orgIDs := []string{org1ID}
	queries, err := store.ListAccessibleQueries(ctx, userID, orgIDs, time.Time{})

	// Verify: Personal + org1, NOT org2
	require.NoError(t, err)
	assert.Len(t, queries, 2, "Should see personal + org1 shared")

	foundOrg2 := false
	for _, q := range queries {
		if q.OrganizationID != nil && *q.OrganizationID == org2ID {
			foundOrg2 = true
		}
	}
	assert.False(t, foundOrg2, "Should not see org2 queries")
}

func TestSyncDelete_SharedResource(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	syncService := sync.NewService(store, sync.Config{}, testLogger())

	ctx := context.Background()

	userID := "delete-user"
	orgID := "delete-org"

	setupOrgMembership(t, db, orgID, userID, "admin")

	// Create shared connection
	conn := createSyncConnection("conn-to-delete", userID, "shared", &orgID)
	require.NoError(t, store.SaveConnection(ctx, userID, conn))

	// Test: Push delete operation
	changes := []sync.SyncChange{
		{
			ItemType:    sync.SyncItemTypeConnection,
			ItemID:      conn.ID,
			Action:      sync.SyncActionDelete,
			SyncVersion: 2,
			DeviceID:    "device-1",
			Timestamp:   time.Now(),
		},
	}

	uploadReq := &sync.SyncUploadRequest{
		UserID:   userID,
		DeviceID: "device-1",
		Changes:  changes,
	}

	resp, err := syncService.Upload(ctx, uploadReq)
	require.NoError(t, err)
	assert.Len(t, resp.Rejected, 0, "Delete should succeed")

	// Verify: Connection soft-deleted
	orgIDs := []string{orgID}
	connections, err := store.ListAccessibleConnections(ctx, userID, orgIDs, time.Time{})
	require.NoError(t, err)

	foundDeleted := false
	for _, c := range connections {
		if c.ID == conn.ID {
			foundDeleted = true
		}
	}
	assert.False(t, foundDeleted, "Deleted connection should not appear")
}

func TestSyncSanitization_RejectsCredentials(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	syncService := sync.NewService(store, sync.Config{
		EnableSanitization: true,
	}, testLogger())

	ctx := context.Background()

	userID := "sanitize-user"

	// Test: Try to sync connection with password
	connWithPassword := map[string]interface{}{
		"id":         "conn-bad",
		"user_id":    userID,
		"name":       "DB with Password",
		"type":       "postgres",
		"host":       "localhost",
		"port":       5432,
		"password":   "secret123", // Should be rejected
		"visibility": "personal",
		"created_at": time.Now().Unix(),
		"updated_at": time.Now().Unix(),
	}

	changes := []sync.SyncChange{
		{
			ItemType:    sync.SyncItemTypeConnection,
			ItemID:      "conn-bad",
			Action:      sync.SyncActionCreate,
			Data:        connWithPassword,
			SyncVersion: 1,
			DeviceID:    "device-1",
			Timestamp:   time.Now(),
		},
	}

	uploadReq := &sync.SyncUploadRequest{
		UserID:   userID,
		DeviceID: "device-1",
		Changes:  changes,
	}

	// Execute
	_, err := syncService.Upload(ctx, uploadReq)

	// Verify: Rejected due to sanitization
	require.Error(t, err)
	assert.Contains(t, err.Error(), "password")
}

func TestSyncIncremental_SinceTimestamp(t *testing.T) {
	db, cleanup := setupSyncTestDB(t)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, testLogger())
	ctx := context.Background()

	userID := "incremental-user"

	// Create old connection
	oldConn := createSyncConnection("conn-old", userID, "personal", nil)
	oldConn.UpdatedAt = time.Now().Add(-2 * time.Hour)
	require.NoError(t, store.SaveConnection(ctx, userID, oldConn))

	// Mark sync point
	syncPoint := time.Now().Add(-1 * time.Hour)

	// Create new connection after sync point
	newConn := createSyncConnection("conn-new", userID, "personal", nil)
	newConn.UpdatedAt = time.Now()
	require.NoError(t, store.SaveConnection(ctx, userID, newConn))

	// Test: List connections since sync point
	connections, err := store.ListConnections(ctx, userID, syncPoint)

	// Verify: Only returns new connection
	require.NoError(t, err)
	assert.Len(t, connections, 1, "Should only return connections updated after sync point")
	assert.Equal(t, "conn-new", connections[0].ID)
}

// ====================================================================
// Helper Functions
// ====================================================================

func setupSyncTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			created_at INTEGER NOT NULL
		);

		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);

		CREATE TABLE organization_members (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL,
			joined_at INTEGER NOT NULL,
			UNIQUE(organization_id, user_id)
		);

		CREATE TABLE connection_templates (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			host TEXT,
			port INTEGER,
			database_name TEXT,
			username TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sync_version INTEGER DEFAULT 1,
			organization_id TEXT,
			visibility TEXT DEFAULT 'personal',
			created_by TEXT NOT NULL,
			deleted_at INTEGER
		);

		CREATE TABLE saved_queries_sync (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			query_text TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sync_version INTEGER DEFAULT 1,
			organization_id TEXT,
			visibility TEXT DEFAULT 'personal',
			created_by TEXT NOT NULL,
			deleted_at INTEGER
		);

		CREATE TABLE sync_conflicts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			item_type TEXT NOT NULL,
			item_id TEXT NOT NULL,
			local_data TEXT,
			remote_data TEXT,
			detected_at INTEGER NOT NULL,
			resolved_at INTEGER,
			resolution_strategy TEXT
		);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	return db, func() { db.Close() }
}

func setupOrgMembership(t *testing.T, db *sql.DB, orgID, userID, role string) {
	now := time.Now().Unix()

	// Create user
	_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, created_at) VALUES (?, ?, ?)",
		userID, userID+"@example.com", now)

	// Create org
	_, _ = db.Exec(`
		INSERT OR IGNORE INTO organizations (id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, orgID, "Org "+orgID, userID, now, now)

	// Add membership
	_, err := db.Exec(`
		INSERT OR IGNORE INTO organization_members (id, organization_id, user_id, role, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, "mem-"+orgID+"-"+userID, orgID, userID, role, now)
	require.NoError(t, err)
}

func createSyncConnection(id, userID, visibility string, orgID *string) *sync.ConnectionTemplate {
	now := time.Now()
	return &sync.ConnectionTemplate{
		ID:             id,
		UserID:         userID,
		Name:           "Connection " + id,
		Type:           "postgres",
		Host:           "localhost",
		Port:           5432,
		Database:       "testdb",
		Username:       "user",
		Visibility:     visibility,
		OrganizationID: orgID,
		CreatedAt:      now,
		UpdatedAt:      now,
		SyncVersion:    1,
	}
}

func createSyncQuery(id, userID, visibility string, orgID *string) *sync.SavedQuery {
	now := time.Now()
	return &sync.SavedQuery{
		ID:             id,
		UserID:         userID,
		Name:           "Query " + id,
		Query:          "SELECT * FROM users",
		Visibility:     visibility,
		OrganizationID: orgID,
		CreatedAt:      now,
		UpdatedAt:      now,
		SyncVersion:    1,
	}
}

func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}
