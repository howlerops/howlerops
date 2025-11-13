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

	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// Shared Resources Repository Tests - Connections
// ====================================================================

func TestGetConnectionsByOrganization_ReturnsOnlyShared(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	// Setup: Create organization with mixed connections
	orgID := "org-test"
	createTestOrg(t, db, orgID, "user-1")

	// User 1: 2 shared + 1 personal
	sharedConn1 := createConnection("user-1", "Shared DB 1", "shared", &orgID)
	require.NoError(t, store.Create(ctx, sharedConn1))

	sharedConn2 := createConnection("user-1", "Shared DB 2", "shared", &orgID)
	require.NoError(t, store.Create(ctx, sharedConn2))

	personalConn := createConnection("user-1", "Personal DB", "personal", nil)
	require.NoError(t, store.Create(ctx, personalConn))

	// Test: Fetch connections by organization
	connections, err := store.GetConnectionsByOrganization(ctx, orgID)

	// Verify: Returns only 2 shared connections
	require.NoError(t, err)
	assert.Len(t, connections, 2, "Should return only shared connections")

	for _, conn := range connections {
		assert.Equal(t, "shared", conn.Visibility)
		assert.NotNil(t, conn.OrganizationID)
		assert.Equal(t, orgID, *conn.OrganizationID)
	}
}

func TestGetConnectionsByOrganization_EmptyWhenNoShared(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	orgID := "org-empty"
	createTestOrg(t, db, orgID, "user-1")

	// Create only personal connections
	conn := createConnection("user-1", "Personal Only", "personal", nil)
	require.NoError(t, store.Create(ctx, conn))

	// Test
	connections, err := store.GetConnectionsByOrganization(ctx, orgID)

	// Verify: Empty result
	require.NoError(t, err)
	assert.Empty(t, connections)
}

func TestGetSharedConnections_MultipleOrganizations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	// Setup: User in 3 organizations
	org1ID := "org-1"
	org2ID := "org-2"
	org3ID := "org-3"

	createTestOrg(t, db, org1ID, "user-admin")
	createTestOrg(t, db, org2ID, "user-admin")
	createTestOrg(t, db, org3ID, "user-other")

	addOrgMember(t, db, org1ID, "user-test", "member")
	addOrgMember(t, db, org2ID, "user-test", "member")
	// user-test NOT a member of org-3

	// Personal connections
	personal1 := createConnection("user-test", "My DB 1", "personal", nil)
	require.NoError(t, store.Create(ctx, personal1))

	personal2 := createConnection("user-test", "My DB 2", "personal", nil)
	require.NoError(t, store.Create(ctx, personal2))

	// Shared in org-1
	shared1 := createConnection("user-admin", "Org1 DB 1", "shared", &org1ID)
	require.NoError(t, store.Create(ctx, shared1))

	shared2 := createConnection("user-admin", "Org1 DB 2", "shared", &org1ID)
	require.NoError(t, store.Create(ctx, shared2))

	// Shared in org-2
	shared3 := createConnection("user-admin", "Org2 DB", "shared", &org2ID)
	require.NoError(t, store.Create(ctx, shared3))

	// Shared in org-3 (user-test should NOT see this)
	shared4 := createConnection("user-other", "Org3 DB", "shared", &org3ID)
	require.NoError(t, store.Create(ctx, shared4))

	// Test: Fetch all accessible connections for user-test
	connections, err := store.GetSharedConnections(ctx, "user-test")

	// Verify: 2 personal + 2 from org-1 + 1 from org-2 = 5 total
	require.NoError(t, err)
	assert.Len(t, connections, 5, "Should see personal + shared from member orgs")

	// Count by type
	personalCount := 0
	org1Count := 0
	org2Count := 0
	org3Count := 0

	for _, conn := range connections {
		if conn.Visibility == "personal" {
			personalCount++
		} else if conn.OrganizationID != nil {
			switch *conn.OrganizationID {
			case org1ID:
				org1Count++
			case org2ID:
				org2Count++
			case org3ID:
				org3Count++
			}
		}
	}

	assert.Equal(t, 2, personalCount, "Should have 2 personal connections")
	assert.Equal(t, 2, org1Count, "Should have 2 from org-1")
	assert.Equal(t, 1, org2Count, "Should have 1 from org-2")
	assert.Equal(t, 0, org3Count, "Should NOT have any from org-3")
}

func TestUpdateConnectionVisibility_PersonalToShared(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	orgID := "org-share"
	createTestOrg(t, db, orgID, "user-1")

	// Create personal connection
	conn := createConnection("user-1", "Will Share", "personal", nil)
	conn.CreatedBy = "user-1"
	require.NoError(t, store.Create(ctx, conn))

	// Test: Change to shared
	err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-1", "shared")
	require.NoError(t, err)

	// Update org ID separately
	conn.OrganizationID = &orgID
	conn.Visibility = "shared"
	err = store.Update(ctx, conn)
	require.NoError(t, err)

	// Verify: Now appears in organization connections
	orgConns, err := store.GetConnectionsByOrganization(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, orgConns, 1)
	assert.Equal(t, "shared", orgConns[0].Visibility)
}

func TestUpdateConnectionVisibility_SharedToPersonal(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	orgID := "org-unshare"
	createTestOrg(t, db, orgID, "user-1")

	// Create shared connection
	conn := createConnection("user-1", "Will Unshare", "shared", &orgID)
	conn.CreatedBy = "user-1"
	require.NoError(t, store.Create(ctx, conn))

	// Test: Change to personal
	err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-1", "personal")
	require.NoError(t, err)

	// Update org ID to nil
	conn.OrganizationID = nil
	conn.Visibility = "personal"
	err = store.Update(ctx, conn)
	require.NoError(t, err)

	// Verify: No longer in organization connections
	orgConns, err := store.GetConnectionsByOrganization(ctx, orgID)
	require.NoError(t, err)
	assert.Empty(t, orgConns)

	// But still accessible to owner
	retrieved, err := store.GetByID(ctx, conn.ID)
	require.NoError(t, err)
	assert.Equal(t, "personal", retrieved.Visibility)
	assert.Nil(t, retrieved.OrganizationID)
}

func TestUpdateConnectionVisibility_OnlyCreatorCanChange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	// User 1 creates connection
	conn := createConnection("user-1", "User1 DB", "personal", nil)
	conn.CreatedBy = "user-1"
	require.NoError(t, store.Create(ctx, conn))

	// Test: User 2 tries to change visibility
	err := store.UpdateConnectionVisibility(ctx, conn.ID, "user-2", "shared")

	// Verify: Error returned
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only the creator")
}

// ====================================================================
// Shared Resources Repository Tests - Queries
// ====================================================================

func TestGetQueriesByOrganization_ReturnsOnlyShared(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewQueryStore(db, testLogger())
	ctx := context.Background()

	orgID := "org-queries"
	createTestOrg(t, db, orgID, "user-1")

	// Shared queries
	shared1 := createQuery("user-1", "Shared Query 1", "shared", &orgID)
	require.NoError(t, store.Create(ctx, shared1))

	shared2 := createQuery("user-1", "Shared Query 2", "shared", &orgID)
	require.NoError(t, store.Create(ctx, shared2))

	// Personal query
	personal := createQuery("user-1", "Personal Query", "personal", nil)
	require.NoError(t, store.Create(ctx, personal))

	// Test
	queries, err := store.GetQueriesByOrganization(ctx, orgID)

	// Verify
	require.NoError(t, err)
	assert.Len(t, queries, 2)

	for _, q := range queries {
		assert.Equal(t, "shared", q.Visibility)
		assert.NotNil(t, q.OrganizationID)
		assert.Equal(t, orgID, *q.OrganizationID)
	}
}

func TestGetSharedQueries_MultipleOrganizations(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewQueryStore(db, testLogger())
	ctx := context.Background()

	org1ID := "org-q1"
	org2ID := "org-q2"
	org3ID := "org-q3"

	createTestOrg(t, db, org1ID, "admin")
	createTestOrg(t, db, org2ID, "admin")
	createTestOrg(t, db, org3ID, "other")

	addOrgMember(t, db, org1ID, "user-test", "member")
	addOrgMember(t, db, org2ID, "user-test", "admin")

	// Personal
	p1 := createQuery("user-test", "Personal 1", "personal", nil)
	require.NoError(t, store.Create(ctx, p1))

	// Shared in user's orgs
	s1 := createQuery("admin", "Org1 Query", "shared", &org1ID)
	require.NoError(t, store.Create(ctx, s1))

	s2 := createQuery("admin", "Org2 Query", "shared", &org2ID)
	require.NoError(t, store.Create(ctx, s2))

	// Shared in org user is NOT member of
	s3 := createQuery("other", "Org3 Query", "shared", &org3ID)
	require.NoError(t, store.Create(ctx, s3))

	// Test
	queries, err := store.GetSharedQueries(ctx, "user-test")

	// Verify: 1 personal + 1 from org1 + 1 from org2 = 3
	require.NoError(t, err)
	assert.Len(t, queries, 3)

	org3Found := false
	for _, q := range queries {
		if q.OrganizationID != nil && *q.OrganizationID == org3ID {
			org3Found = true
		}
	}
	assert.False(t, org3Found, "Should not see queries from non-member org")
}

func TestUpdateQueryVisibility_OnlyCreatorCanChange(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewQueryStore(db, testLogger())
	ctx := context.Background()

	// User 1 creates query
	query := createQuery("user-1", "User1 Query", "personal", nil)
	query.CreatedBy = "user-1"
	require.NoError(t, store.Create(ctx, query))

	// Test: User 2 tries to change
	err := store.UpdateQueryVisibility(ctx, query.ID, "user-2", "shared")

	// Verify: Fails
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only the creator")
}

// ====================================================================
// Cross-Organization Isolation Tests
// ====================================================================

func TestCrossOrganizationIsolation_Connections(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	// Create 2 separate organizations
	org1ID := "org-isolated-1"
	org2ID := "org-isolated-2"

	createTestOrg(t, db, org1ID, "user-1")
	createTestOrg(t, db, org2ID, "user-2")

	// Org 1 shared connections
	conn1 := createConnection("user-1", "Org1 Secret DB", "shared", &org1ID)
	require.NoError(t, store.Create(ctx, conn1))

	// Org 2 shared connections
	conn2 := createConnection("user-2", "Org2 Secret DB", "shared", &org2ID)
	require.NoError(t, store.Create(ctx, conn2))

	// Test: Org 1 members should NOT see Org 2 connections
	org1Conns, err := store.GetConnectionsByOrganization(ctx, org1ID)
	require.NoError(t, err)
	assert.Len(t, org1Conns, 1)
	assert.Equal(t, "Org1 Secret DB", org1Conns[0].Name)

	// Test: Org 2 members should NOT see Org 1 connections
	org2Conns, err := store.GetConnectionsByOrganization(ctx, org2ID)
	require.NoError(t, err)
	assert.Len(t, org2Conns, 1)
	assert.Equal(t, "Org2 Secret DB", org2Conns[0].Name)
}

func TestCrossOrganizationIsolation_Queries(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewQueryStore(db, testLogger())
	ctx := context.Background()

	org1ID := "org-q-isolated-1"
	org2ID := "org-q-isolated-2"

	createTestOrg(t, db, org1ID, "user-1")
	createTestOrg(t, db, org2ID, "user-2")

	// Org-specific queries
	q1 := createQuery("user-1", "Org1 Secret Query", "shared", &org1ID)
	require.NoError(t, store.Create(ctx, q1))

	q2 := createQuery("user-2", "Org2 Secret Query", "shared", &org2ID)
	require.NoError(t, store.Create(ctx, q2))

	// Test isolation
	org1Queries, err := store.GetQueriesByOrganization(ctx, org1ID)
	require.NoError(t, err)
	assert.Len(t, org1Queries, 1)
	assert.Equal(t, "Org1 Secret Query", org1Queries[0].Name)

	org2Queries, err := store.GetQueriesByOrganization(ctx, org2ID)
	require.NoError(t, err)
	assert.Len(t, org2Queries, 1)
	assert.Equal(t, "Org2 Secret Query", org2Queries[0].Name)
}

// ====================================================================
// Soft Delete Tests for Shared Resources
// ====================================================================

func TestSoftDelete_SharedConnection_NotVisibleInOrgList(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := turso.NewConnectionStore(db, testLogger())
	ctx := context.Background()

	orgID := "org-delete"
	createTestOrg(t, db, orgID, "user-1")

	conn := createConnection("user-1", "Will Delete", "shared", &orgID)
	require.NoError(t, store.Create(ctx, conn))

	// Verify it exists
	orgConns, err := store.GetConnectionsByOrganization(ctx, orgID)
	require.NoError(t, err)
	assert.Len(t, orgConns, 1)

	// Soft delete
	err = store.Delete(ctx, conn.ID)
	require.NoError(t, err)

	// Verify not in org list
	orgConns, err = store.GetConnectionsByOrganization(ctx, orgID)
	require.NoError(t, err)
	assert.Empty(t, orgConns)
}

// ====================================================================
// Helper Functions
// ====================================================================

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	schema := `
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT UNIQUE NOT NULL,
			username TEXT UNIQUE NOT NULL,
			created_at INTEGER NOT NULL
		);

		CREATE TABLE organizations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			deleted_at INTEGER,
			max_members INTEGER DEFAULT 10
		);

		CREATE TABLE organization_members (
			id TEXT PRIMARY KEY,
			organization_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			role TEXT NOT NULL,
			joined_at INTEGER NOT NULL,
			FOREIGN KEY (organization_id) REFERENCES organizations(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
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
			use_ssh BOOLEAN DEFAULT 0,
			ssh_host TEXT,
			ssh_port INTEGER,
			ssh_user TEXT,
			color TEXT,
			icon TEXT,
			metadata TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sync_version INTEGER DEFAULT 1,
			organization_id TEXT,
			visibility TEXT DEFAULT 'personal' CHECK(visibility IN ('personal', 'shared')),
			created_by TEXT NOT NULL,
			deleted_at INTEGER,
			FOREIGN KEY (organization_id) REFERENCES organizations(id)
		);

		CREATE TABLE saved_queries_sync (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			query_text TEXT NOT NULL,
			connection_id TEXT,
			tags TEXT,
			favorite BOOLEAN DEFAULT 0,
			metadata TEXT,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			sync_version INTEGER DEFAULT 1,
			organization_id TEXT,
			visibility TEXT DEFAULT 'personal' CHECK(visibility IN ('personal', 'shared')),
			created_by TEXT NOT NULL,
			deleted_at INTEGER,
			FOREIGN KEY (organization_id) REFERENCES organizations(id)
		);
	`

	_, err = db.Exec(schema)
	require.NoError(t, err)

	cleanup := func() {
		_ = db.Close() // Best-effort close in test
	}

	return db, cleanup
}

func testLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}

func createTestOrg(t *testing.T, db *sql.DB, orgID, ownerID string) {
	now := time.Now().Unix()

	// Create owner user if not exists
	_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, username, created_at) VALUES (?, ?, ?, ?)",
		ownerID, ownerID+"@example.com", ownerID, now)

	// Create organization
	_, err := db.Exec(`
		INSERT INTO organizations (id, name, owner_id, created_at, updated_at, max_members)
		VALUES (?, ?, ?, ?, ?, ?)
	`, orgID, "Test Org "+orgID, ownerID, now, now, 10)
	require.NoError(t, err)

	// Add owner as member
	_, err = db.Exec(`
		INSERT INTO organization_members (id, organization_id, user_id, role, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, "mem-"+orgID+"-"+ownerID, orgID, ownerID, "owner", now)
	require.NoError(t, err)
}

func addOrgMember(t *testing.T, db *sql.DB, orgID, userID, role string) {
	now := time.Now().Unix()

	// Create user if not exists
	_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, username, created_at) VALUES (?, ?, ?, ?)",
		userID, userID+"@example.com", userID, now)

	_, err := db.Exec(`
		INSERT INTO organization_members (id, organization_id, user_id, role, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, "mem-"+orgID+"-"+userID, orgID, userID, role, now)
	require.NoError(t, err)
}

func createConnection(userID, name, visibility string, orgID *string) *turso.Connection {
	now := time.Now()
	return &turso.Connection{
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
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
		SyncVersion:    1,
	}
}

func createQuery(userID, name, visibility string, orgID *string) *turso.SavedQuery {
	now := time.Now()
	return &turso.SavedQuery{
		ID:             "query-" + name,
		UserID:         userID,
		Name:           name,
		Description:    "Test query",
		Query:          "SELECT * FROM users",
		Visibility:     visibility,
		OrganizationID: orgID,
		CreatedBy:      userID,
		CreatedAt:      now,
		UpdatedAt:      now,
		SyncVersion:    1,
	}
}
