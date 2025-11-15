package main

import (
	"database/sql"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	_ "github.com/tursodatabase/libsql-client-go/libsql"

	"github.com/jbeck018/howlerops/backend-go/pkg/storage/turso"
)

var (
	dbPath      = flag.String("db", "", "Path to database file (default: creates temp database)")
	checkSchema = flag.Bool("check-schema", false, "Compare fresh install schema with migrated schema")
	verbose     = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Setup logger
	logger := logrus.New()
	if *verbose {
		logger.SetLevel(logrus.DebugLevel)
	} else {
		logger.SetLevel(logrus.InfoLevel)
	}
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║        Howlerops Migration Verification Tool            ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create or use specified database
	dbFile := *dbPath
	cleanup := false
	if dbFile == "" {
		// Create temporary database
		tmpDir, err := os.MkdirTemp("", "sqlstudio-verify-*")
		if err != nil {
			logger.Fatal("Failed to create temp directory: ", err)
		}
		dbFile = filepath.Join(tmpDir, "verify.db")
		cleanup = true
		defer func() {
			if cleanup {
				_ = os.RemoveAll(tmpDir) // Best-effort cleanup
				logger.Debug("Cleaned up temporary database")
			}
		}()
	}

	logger.WithField("path", dbFile).Info("Using database file")

	// Connect to database
	db, err := sql.Open("libsql", fmt.Sprintf("file:%s", dbFile))
	if err != nil {
		logger.Fatal("Failed to open database: ", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.WithError(err).Error("Failed to close database")
		}
	}()

	// Run verification tests
	passed := 0
	failed := 0

	tests := []struct {
		name string
		fn   func(*sql.DB, *logrus.Logger) error
	}{
		{"Schema Initialization", testSchemaInit},
		{"Migration Runner", testMigrationRunner},
		{"Migration Tracking Table", testMigrationTable},
		{"Connection Templates Schema", testConnectionTemplatesSchema},
		{"Saved Queries Schema", testSavedQueriesSchema},
		{"Indexes Created", testIndexes},
		{"Foreign Keys", testForeignKeys},
		{"Data Migration", testDataMigration},
		{"Idempotency", testIdempotency},
	}

	for i, test := range tests {
		fmt.Printf("\n[%d/%d] Testing: %s\n", i+1, len(tests), test.name)
		fmt.Println(strings.Repeat("─", 60))

		err := test.fn(db, logger)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
			failed++
		} else {
			fmt.Printf("✅ PASSED\n")
			passed++
		}
	}

	// Schema comparison test (optional)
	if *checkSchema {
		fmt.Printf("\n[%d/%d] Testing: %s\n", len(tests)+1, len(tests)+1, "Schema Comparison")
		fmt.Println(strings.Repeat("─", 60))
		err := testSchemaComparison(logger)
		if err != nil {
			fmt.Printf("❌ FAILED: %v\n", err)
			failed++
		} else {
			fmt.Printf("✅ PASSED\n")
			passed++
		}
	}

	// Summary
	fmt.Println()
	fmt.Println(strings.Repeat("═", 60))
	fmt.Printf("Test Results: %d passed, %d failed\n", passed, failed)
	fmt.Println(strings.Repeat("═", 60))

	if failed > 0 {
		os.Exit(1)
	}
}

func testSchemaInit(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Initializing schema...")
	if err := turso.InitializeSchema(db, logger); err != nil {
		return fmt.Errorf("schema initialization failed: %w", err)
	}

	// Verify core tables exist
	tables := []string{
		"users",
		"sessions",
		"connection_templates",
		"saved_queries_sync",
		"organizations",
		"organization_members",
	}

	for _, table := range tables {
		var name string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err == sql.ErrNoRows {
			return fmt.Errorf("table %s does not exist", table)
		}
		if err != nil {
			return fmt.Errorf("error checking table %s: %w", table, err)
		}
		logger.Debugf("  ✓ Table %s exists", table)
	}

	return nil
}

func testMigrationRunner(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Running migrations...")
	if err := turso.RunMigrations(db, logger); err != nil {
		return fmt.Errorf("migration runner failed: %w", err)
	}

	// Verify at least one migration was processed
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("no migrations were applied")
	}

	logger.Debugf("  ✓ %d migrations tracked", count)
	return nil
}

func testMigrationTable(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Checking migration tracking...")

	// Check table exists
	var tableName string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='schema_migrations'").Scan(&tableName)
	if err != nil {
		return fmt.Errorf("schema_migrations table does not exist: %w", err)
	}

	// Check structure
	columns := make(map[string]bool)
	rows, err := db.Query("PRAGMA table_info(schema_migrations)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		columns[name] = true
	}

	required := []string{"version", "description", "applied_at"}
	for _, col := range required {
		if !columns[col] {
			return fmt.Errorf("missing required column: %s", col)
		}
		logger.Debugf("  ✓ Column %s exists", col)
	}

	return nil
}

func testConnectionTemplatesSchema(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Verifying connection_templates schema...")

	columns := make(map[string]bool)
	rows, err := db.Query("PRAGMA table_info(connection_templates)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		columns[name] = true
	}

	// Check for organization columns
	required := []string{"organization_id", "visibility", "created_by"}
	for _, col := range required {
		if !columns[col] {
			return fmt.Errorf("missing required column: %s", col)
		}
		logger.Debugf("  ✓ Column %s exists", col)
	}

	return nil
}

func testSavedQueriesSchema(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Verifying saved_queries_sync schema...")

	columns := make(map[string]bool)
	rows, err := db.Query("PRAGMA table_info(saved_queries_sync)")
	if err != nil {
		return fmt.Errorf("failed to get table info: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan column info: %w", err)
		}
		columns[name] = true
	}

	// Check for organization columns
	required := []string{"organization_id", "visibility", "created_by"}
	for _, col := range required {
		if !columns[col] {
			return fmt.Errorf("missing required column: %s", col)
		}
		logger.Debugf("  ✓ Column %s exists", col)
	}

	return nil
}

func testIndexes(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Verifying indexes...")

	tests := []struct {
		table string
		index string
	}{
		{"connection_templates", "idx_connections_org_visibility"},
		{"connection_templates", "idx_connections_created_by"},
		{"saved_queries_sync", "idx_queries_org_visibility"},
		{"saved_queries_sync", "idx_queries_created_by"},
	}

	for _, test := range tests {
		var name string
		err := db.QueryRow(
			"SELECT name FROM sqlite_master WHERE type='index' AND tbl_name=? AND name=?",
			test.table, test.index,
		).Scan(&name)

		if err == sql.ErrNoRows {
			return fmt.Errorf("index %s on %s does not exist", test.index, test.table)
		}
		if err != nil {
			return fmt.Errorf("error checking index %s: %w", test.index, err)
		}
		logger.Debugf("  ✓ Index %s exists on %s", test.index, test.table)
	}

	return nil
}

func testForeignKeys(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Verifying foreign keys...")

	// Enable foreign keys
	_, err := db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Check foreign key integrity
	rows, err := db.Query("PRAGMA foreign_key_check")
	if err != nil {
		return fmt.Errorf("failed to check foreign keys: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	violations := 0
	for rows.Next() {
		violations++
		var table, rowid, parent, fkid string
		if err := rows.Scan(&table, &rowid, &parent, &fkid); err != nil {
			logger.WithError(err).Warn("Failed to scan FK violation row")
			continue
		}
		logger.Errorf("  ✗ FK violation in %s, row %s -> %s.%s", table, rowid, parent, fkid)
	}

	if violations > 0 {
		return fmt.Errorf("found %d foreign key violations", violations)
	}

	logger.Debug("  ✓ No foreign key violations")
	return nil
}

func testDataMigration(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Testing data migration...")

	// Insert test user
	userID := fmt.Sprintf("test-user-%d", time.Now().Unix())
	_, err := db.Exec(`
		INSERT INTO users (id, username, email, password_hash, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, userID, "testuser", fmt.Sprintf("test-%d@example.com", time.Now().Unix()),
		"hash", time.Now().Unix(), time.Now().Unix())
	if err != nil {
		return fmt.Errorf("failed to insert test user: %w", err)
	}

	// Insert test connection
	connID := fmt.Sprintf("test-conn-%d", time.Now().Unix())
	_, err = db.Exec(`
		INSERT INTO connection_templates (id, user_id, name, type, created_at, updated_at, created_by, visibility)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, connID, userID, "Test Connection", "postgresql",
		time.Now().Unix(), time.Now().Unix(), userID, "personal")
	if err != nil {
		return fmt.Errorf("failed to insert test connection: %w", err)
	}

	// Verify data
	var createdBy, visibility string
	err = db.QueryRow("SELECT created_by, visibility FROM connection_templates WHERE id = ?", connID).
		Scan(&createdBy, &visibility)
	if err != nil {
		return fmt.Errorf("failed to query connection: %w", err)
	}

	if createdBy != userID {
		return fmt.Errorf("created_by mismatch: got %s, expected %s", createdBy, userID)
	}

	if visibility != "personal" {
		return fmt.Errorf("visibility mismatch: got %s, expected personal", visibility)
	}

	logger.Debug("  ✓ Data migration working correctly")
	return nil
}

func testIdempotency(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Testing idempotency (running migrations again)...")

	// Get migration count
	var count1 int
	err := db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count1)
	if err != nil {
		return fmt.Errorf("failed to count migrations: %w", err)
	}

	// Run migrations again
	if err := turso.RunMigrations(db, logger); err != nil {
		return fmt.Errorf("second migration run failed: %w", err)
	}

	// Count should be the same
	var count2 int
	err = db.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count2)
	if err != nil {
		return fmt.Errorf("failed to count migrations: %w", err)
	}

	if count1 != count2 {
		return fmt.Errorf("migration count changed: %d -> %d (should be idempotent)", count1, count2)
	}

	logger.Debugf("  ✓ Idempotency verified (%d migrations)", count1)
	return nil
}

func testSchemaComparison(logger *logrus.Logger) error {
	logger.Info("Comparing fresh install vs migrated schema...")

	// Create two temporary databases
	tmpDir, err := os.MkdirTemp("", "sqlstudio-compare-*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }() // Best-effort cleanup

	freshDB := filepath.Join(tmpDir, "fresh.db")
	migratedDB := filepath.Join(tmpDir, "migrated.db")

	// Setup fresh database
	dbFresh, err := sql.Open("libsql", fmt.Sprintf("file:%s", freshDB))
	if err != nil {
		return fmt.Errorf("failed to open fresh db: %w", err)
	}
	defer func() { _ = dbFresh.Close() }() // Best-effort close

	if err := turso.InitializeSchema(dbFresh, logger); err != nil {
		return fmt.Errorf("failed to init fresh schema: %w", err)
	}

	// Setup migrated database (old schema + migrations)
	dbMigrated, err := sql.Open("libsql", fmt.Sprintf("file:%s", migratedDB))
	if err != nil {
		return fmt.Errorf("failed to open migrated db: %w", err)
	}
	defer func() { _ = dbMigrated.Close() }() // Best-effort close

	if err := turso.InitializeSchema(dbMigrated, logger); err != nil {
		return fmt.Errorf("failed to init migrated schema: %w", err)
	}

	if err := turso.RunMigrations(dbMigrated, logger); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Compare schemas
	tables := []string{"connection_templates", "saved_queries_sync"}
	for _, table := range tables {
		freshCols := getColumns(dbFresh, table)
		migratedCols := getColumns(dbMigrated, table)

		if len(freshCols) != len(migratedCols) {
			return fmt.Errorf("column count mismatch for %s: fresh=%d, migrated=%d",
				table, len(freshCols), len(migratedCols))
		}

		for col := range freshCols {
			if !migratedCols[col] {
				return fmt.Errorf("column %s.%s exists in fresh but not in migrated", table, col)
			}
		}

		for col := range migratedCols {
			if !freshCols[col] {
				return fmt.Errorf("column %s.%s exists in migrated but not in fresh", table, col)
			}
		}

		logger.Debugf("  ✓ Table %s schema matches", table)
	}

	return nil
}

func getColumns(db *sql.DB, table string) map[string]bool {
	columns := make(map[string]bool)
	rows, _ := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	defer func() { _ = rows.Close() }() // Best-effort close

	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue interface{}
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			// Skip rows that fail to scan
			continue
		}
		columns[name] = true
	}

	return columns
}
