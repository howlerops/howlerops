package connections_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/sql-studio/backend-go/pkg/storage/turso"
)

// ====================================================================
// Performance Benchmark Tests
// ====================================================================

func BenchmarkGetSharedConnections_100Orgs_1000Connections(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 100, 1000)
	defer cleanup()

	store := turso.NewConnectionStore(db, benchLogger())
	ctx := context.Background()

	userID := "bench-user"

	// User is member of 10 orgs
	orgIDs := make([]string, 10)
	for i := 0; i < 10; i++ {
		orgIDs[i] = fmt.Sprintf("org-%d", i)
		addBenchMember(b, db, orgIDs[i], userID, "member")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetSharedConnections(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetSharedConnections_10Orgs_10000Connections(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 10, 10000)
	defer cleanup()

	store := turso.NewConnectionStore(db, benchLogger())
	ctx := context.Background()

	userID := "bench-user"

	// User in 5 orgs
	for i := 0; i < 5; i++ {
		orgID := fmt.Sprintf("org-%d", i)
		addBenchMember(b, db, orgID, userID, "member")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetSharedConnections(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetConnectionsByOrganization_LargeOrg(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 1, 5000)
	defer cleanup()

	store := turso.NewConnectionStore(db, benchLogger())
	ctx := context.Background()

	orgID := "org-0"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetConnectionsByOrganization(ctx, orgID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateConnectionVisibility(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 1, 100)
	defer cleanup()

	store := turso.NewConnectionStore(db, benchLogger())
	ctx := context.Background()

	userID := "bench-user-update"
	setupBenchOrg(b, db, "org-update", userID)

	// Create connections to update
	connections := make([]*turso.Connection, 100)
	for i := 0; i < 100; i++ {
		conn := &turso.Connection{
			ID:         fmt.Sprintf("conn-update-%d", i),
			UserID:     userID,
			Name:       fmt.Sprintf("DB %d", i),
			Type:       "postgres",
			Host:       "localhost",
			Port:       5432,
			Database:   "testdb",
			Username:   "user",
			CreatedBy:  userID,
			Visibility: "personal",
		}
		require.NoError(b, store.Create(ctx, conn))
		connections[i] = conn
	}

	b.ResetTimer()
	b.ReportAllocs()

	idx := 0
	for i := 0; i < b.N; i++ {
		conn := connections[idx%100]
		visibility := "shared"
		if i%2 == 0 {
			visibility = "personal"
		}

		_ = store.UpdateConnectionVisibility(ctx, conn.ID, userID, visibility)
		idx++
	}
}

func BenchmarkGetSharedQueries_MultiOrg(b *testing.B) {
	db, cleanup := setupBenchmarkDBQueries(b, 50, 2000)
	defer cleanup()

	store := turso.NewQueryStore(db, benchLogger())
	ctx := context.Background()

	userID := "bench-query-user"

	// User in 20 orgs
	for i := 0; i < 20; i++ {
		orgID := fmt.Sprintf("org-query-%d", i)
		addBenchMember(b, db, orgID, userID, "member")
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetSharedQueries(ctx, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentGetSharedConnections(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 20, 1000)
	defer cleanup()

	store := turso.NewConnectionStore(db, benchLogger())
	ctx := context.Background()

	// Setup 10 users, each in different orgs
	users := make([]string, 10)
	for i := 0; i < 10; i++ {
		users[i] = fmt.Sprintf("user-%d", i)
		for j := 0; j < 5; j++ {
			orgID := fmt.Sprintf("org-%d", (i*5+j)%20)
			addBenchMember(b, db, orgID, users[i], "member")
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		userIdx := 0
		for pb.Next() {
			userID := users[userIdx%10]
			_, err := store.GetSharedConnections(ctx, userID)
			if err != nil {
				b.Fatal(err)
			}
			userIdx++
		}
	})
}

func BenchmarkFilterByOrganization_WithIndex(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 100, 5000)
	defer cleanup()

	// Create index on organization_id for optimization
	_, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_conn_org ON connection_templates(organization_id, visibility)")
	require.NoError(b, err)

	store := turso.NewConnectionStore(db, benchLogger())
	ctx := context.Background()

	orgID := "org-50" // Middle org

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.GetConnectionsByOrganization(ctx, orgID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkIncrementalSync_1000Changes(b *testing.B) {
	db, cleanup := setupBenchmarkDB(b, 10, 1000)
	defer cleanup()

	store := turso.NewSyncStoreAdapter(db, benchLogger())
	ctx := context.Background()

	userID := "sync-user"

	// Mark sync point
	syncPoint := time.Now().Add(-1 * time.Hour)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := store.ListConnections(ctx, userID, syncPoint)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ====================================================================
// Benchmark Helper Functions
// ====================================================================

func setupBenchmarkDB(b *testing.B, numOrgs, numConnections int) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(b, err)

	// Schema
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

		CREATE INDEX idx_conn_user ON connection_templates(user_id);
		CREATE INDEX idx_conn_visibility ON connection_templates(visibility);
	`

	_, err = db.Exec(schema)
	require.NoError(b, err)

	// Populate data
	now := time.Now().Unix()

	// Create orgs
	for i := 0; i < numOrgs; i++ {
		orgID := fmt.Sprintf("org-%d", i)
		ownerID := fmt.Sprintf("owner-%d", i)

		_, _ = db.Exec("INSERT INTO users (id, email, created_at) VALUES (?, ?, ?)",
			ownerID, ownerID+"@example.com", now)

		_, err = db.Exec(`
			INSERT INTO organizations (id, name, owner_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, orgID, "Org "+orgID, ownerID, now, now)
		require.NoError(b, err)
	}

	// Create connections distributed across orgs
	for i := 0; i < numConnections; i++ {
		orgID := fmt.Sprintf("org-%d", i%numOrgs)
		userID := fmt.Sprintf("user-%d", i%100)

		// Create user if not exists
		_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, created_at) VALUES (?, ?, ?)",
			userID, userID+"@example.com", now)

		visibility := "shared"
		var orgIDPtr *string
		if i%5 == 0 {
			visibility = "personal"
			orgIDPtr = nil
		} else {
			orgIDPtr = &orgID
		}

		_, err = db.Exec(`
			INSERT INTO connection_templates
			(id, user_id, name, type, host, port, database_name, username, created_at, updated_at, visibility, organization_id, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("conn-%d", i), userID, fmt.Sprintf("DB %d", i), "postgres", "localhost", 5432, "db", "user", now, now, visibility, orgIDPtr, userID)
		require.NoError(b, err)
	}

	return db, func() { db.Close() }
}

func setupBenchmarkDBQueries(b *testing.B, numOrgs, numQueries int) (*sql.DB, func()) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(b, err)

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

		CREATE INDEX idx_query_user ON saved_queries_sync(user_id);
		CREATE INDEX idx_query_visibility ON saved_queries_sync(visibility);
	`

	_, err = db.Exec(schema)
	require.NoError(b, err)

	now := time.Now().Unix()

	// Create orgs
	for i := 0; i < numOrgs; i++ {
		orgID := fmt.Sprintf("org-query-%d", i)
		ownerID := fmt.Sprintf("owner-query-%d", i)

		_, _ = db.Exec("INSERT INTO users (id, email, created_at) VALUES (?, ?, ?)",
			ownerID, ownerID+"@example.com", now)

		_, err = db.Exec(`
			INSERT INTO organizations (id, name, owner_id, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?)
		`, orgID, "Org "+orgID, ownerID, now, now)
		require.NoError(b, err)
	}

	// Create queries
	for i := 0; i < numQueries; i++ {
		orgID := fmt.Sprintf("org-query-%d", i%numOrgs)
		userID := fmt.Sprintf("user-query-%d", i%100)

		_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, created_at) VALUES (?, ?, ?)",
			userID, userID+"@example.com", now)

		visibility := "shared"
		var orgIDPtr *string
		if i%5 == 0 {
			visibility = "personal"
			orgIDPtr = nil
		} else {
			orgIDPtr = &orgID
		}

		_, err = db.Exec(`
			INSERT INTO saved_queries_sync
			(id, user_id, name, query_text, created_at, updated_at, visibility, organization_id, created_by)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("query-%d", i), userID, fmt.Sprintf("Query %d", i), "SELECT * FROM users", now, now, visibility, orgIDPtr, userID)
		require.NoError(b, err)
	}

	return db, func() { db.Close() }
}

func setupBenchOrg(b *testing.B, db *sql.DB, orgID, ownerID string) {
	now := time.Now().Unix()

	_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, created_at) VALUES (?, ?, ?)",
		ownerID, ownerID+"@example.com", now)

	_, err := db.Exec(`
		INSERT INTO organizations (id, name, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, orgID, "Org "+orgID, ownerID, now, now)
	require.NoError(b, err)
}

func addBenchMember(b *testing.B, db *sql.DB, orgID, userID, role string) {
	now := time.Now().Unix()

	_, _ = db.Exec("INSERT OR IGNORE INTO users (id, email, created_at) VALUES (?, ?, ?)",
		userID, userID+"@example.com", now)

	_, _ = db.Exec(`
		INSERT OR IGNORE INTO organization_members (id, organization_id, user_id, role, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, "mem-"+orgID+"-"+userID, orgID, userID, role, now)
}

func benchLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.FatalLevel) // Minimal logging for benchmarks
	return logger
}
