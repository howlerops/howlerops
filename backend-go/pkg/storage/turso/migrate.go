package turso

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
	AppliedAt   time.Time
}

// Migrations is the list of all migrations in order
// Version 1 and 2 are considered "already applied" (initial schema + Phase 3 tables)
var Migrations = []Migration{
	{
		Version:     1,
		Description: "Initial schema (users, sessions, auth tables)",
		SQL:         "", // Already applied via InitializeSchema
	},
	{
		Version:     2,
		Description: "Phase 3 organization tables",
		SQL:         "", // Already applied via InitializeSchema
	},
	{
		Version:     3,
		Description: "Add organization support to resources (connections and queries)",
		SQL:         getAddOrganizationToResourcesSQL(),
	},
}

// MigrationStatus represents the status of a migration
type MigrationStatus struct {
	Version     int
	Description string
	AppliedAt   *time.Time
	Status      string // "pending", "applied", "failed"
	Error       string
}

// RunMigrations executes all pending migrations in a transaction-safe manner
// Returns the number of migrations applied and any error encountered
func RunMigrations(db *sql.DB, logger *logrus.Logger) error {
	logger.Info("Starting migration runner")

	// Create migrations tracking table if it doesn't exist
	if err := createMigrationsTable(db); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get current migration version
	currentVersion, err := getCurrentVersion(db)
	if err != nil {
		return fmt.Errorf("failed to get current version: %w", err)
	}

	logger.WithField("current_version", currentVersion).Info("Current database version")

	// Find and run pending migrations
	pendingCount := 0
	for _, m := range Migrations {
		if m.Version <= currentVersion {
			logger.WithFields(logrus.Fields{
				"version":     m.Version,
				"description": m.Description,
			}).Debug("Migration already applied, skipping")
			continue
		}

		// Skip migrations with no SQL (placeholder migrations)
		if m.SQL == "" {
			logger.WithFields(logrus.Fields{
				"version":     m.Version,
				"description": m.Description,
			}).Info("Migration has no SQL (placeholder), marking as applied")

			if err := recordMigration(db, m.Version, m.Description); err != nil {
				return fmt.Errorf("failed to record placeholder migration %d: %w", m.Version, err)
			}
			continue
		}

		// Run the migration
		logger.WithFields(logrus.Fields{
			"version":     m.Version,
			"description": m.Description,
		}).Info("Applying migration")

		if err := applyMigration(db, m, logger); err != nil {
			return fmt.Errorf("migration %d failed: %w", m.Version, err)
		}

		pendingCount++
		logger.WithField("version", m.Version).Info("Migration applied successfully")
	}

	if pendingCount == 0 {
		logger.Info("No pending migrations found")
	} else {
		logger.WithField("count", pendingCount).Info("All migrations applied successfully")
	}

	return nil
}

// GetMigrationStatus returns the status of all migrations
func GetMigrationStatus(db *sql.DB) ([]MigrationStatus, error) {
	// Create migrations table if needed
	if err := createMigrationsTable(db); err != nil {
		return nil, fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	appliedVersions := make(map[int]time.Time)
	rows, err := db.Query("SELECT version, applied_at FROM schema_migrations ORDER BY version")
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var version int
		var appliedAt int64
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, fmt.Errorf("failed to scan migration: %w", err)
		}
		appliedVersions[version] = time.Unix(appliedAt, 0)
	}

	// Build status list
	statuses := make([]MigrationStatus, 0, len(Migrations))
	for _, m := range Migrations {
		status := MigrationStatus{
			Version:     m.Version,
			Description: m.Description,
		}

		if appliedAt, exists := appliedVersions[m.Version]; exists {
			status.Status = "applied"
			status.AppliedAt = &appliedAt
		} else {
			status.Status = "pending"
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// RollbackMigration rolls back a specific migration (manual process)
// Note: SQLite doesn't support DROP COLUMN, so rollbacks must be done manually
// This function provides guidance on manual rollback steps
func RollbackMigration(version int) error {
	switch version {
	case 3:
		return fmt.Errorf("rollback for migration 3: SQLite doesn't support DROP COLUMN. Manual steps:\n" +
			"1. Create new tables without organization columns\n" +
			"2. Copy data from old tables\n" +
			"3. Drop old tables\n" +
			"4. Rename new tables\n" +
			"5. Recreate indexes\n" +
			"6. DELETE FROM schema_migrations WHERE version = 3")
	default:
		return fmt.Errorf("no rollback defined for migration %d", version)
	}
}

// createMigrationsTable creates the schema_migrations table
func createMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at INTEGER NOT NULL,
			checksum TEXT
		)
	`)
	return err
}

// getCurrentVersion returns the highest applied migration version
func getCurrentVersion(db *sql.DB) (int, error) {
	var version int
	err := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	return version, nil
}

// applyMigration executes a migration in a transaction
func applyMigration(db *sql.DB, m Migration, logger *logrus.Logger) error {
	// Check for special migration handlers
	if m.SQL == "MIGRATION_003_SPECIAL" {
		// Migration 003 has custom logic for idempotent column additions
		if err := applyMigration003(db, logger); err != nil {
			return err
		}

		// Record migration
		_, err := db.Exec(
			"INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
			m.Version,
			m.Description,
			time.Now().Unix(),
		)
		return err
	}

	// Standard migration execution
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Execute migration SQL
	logger.WithField("version", m.Version).Debug("Executing migration SQL")
	if _, err := tx.Exec(m.SQL); err != nil {
		return fmt.Errorf("failed to execute migration SQL: %w", err)
	}

	// Record migration
	logger.WithField("version", m.Version).Debug("Recording migration in schema_migrations")
	_, err = tx.Exec(
		"INSERT INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
		m.Version,
		m.Description,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration: %w", err)
	}

	return nil
}

// recordMigration records a placeholder migration (no SQL to execute)
func recordMigration(db *sql.DB, version int, description string) error {
	_, err := db.Exec(
		"INSERT OR IGNORE INTO schema_migrations (version, description, applied_at) VALUES (?, ?, ?)",
		version,
		description,
		time.Now().Unix(),
	)
	return err
}

// getAddOrganizationToResourcesSQL returns the SQL for migration 003
// Note: SQLite doesn't support IF NOT EXISTS for ALTER TABLE ADD COLUMN,
// so we check for column existence programmatically instead
func getAddOrganizationToResourcesSQL() string {
	return "MIGRATION_003_SPECIAL" // Special marker for custom logic
}

// applyMigration003 applies migration 003 with idempotent column additions
func applyMigration003(db *sql.DB, logger *logrus.Logger) error {
	// Helper function to check if column exists
	columnExists := func(table, column string) (bool, error) {
		rows, err := db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			return false, err
		}
		defer rows.Close()

		for rows.Next() {
			var cid int
			var name, colType string
			var notNull, pk int
			var dfltValue interface{}
			if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
				return false, err
			}
			if name == column {
				return true, nil
			}
		}
		return false, nil
	}

	// Start transaction
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Add columns to connection_templates if they don't exist
	if exists, _ := columnExists("connection_templates", "organization_id"); !exists {
		logger.Debug("Adding organization_id to connection_templates")
		if _, err := tx.Exec("ALTER TABLE connection_templates ADD COLUMN organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE"); err != nil {
			return fmt.Errorf("failed to add organization_id: %w", err)
		}
	}

	if exists, _ := columnExists("connection_templates", "visibility"); !exists {
		logger.Debug("Adding visibility to connection_templates")
		if _, err := tx.Exec("ALTER TABLE connection_templates ADD COLUMN visibility TEXT NOT NULL DEFAULT 'personal'"); err != nil {
			return fmt.Errorf("failed to add visibility: %w", err)
		}
	}

	if exists, _ := columnExists("connection_templates", "created_by"); !exists {
		logger.Debug("Adding created_by to connection_templates")
		if _, err := tx.Exec("ALTER TABLE connection_templates ADD COLUMN created_by TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("failed to add created_by: %w", err)
		}
	}

	// Add columns to saved_queries_sync if they don't exist
	if exists, _ := columnExists("saved_queries_sync", "organization_id"); !exists {
		logger.Debug("Adding organization_id to saved_queries_sync")
		if _, err := tx.Exec("ALTER TABLE saved_queries_sync ADD COLUMN organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE"); err != nil {
			return fmt.Errorf("failed to add organization_id: %w", err)
		}
	}

	if exists, _ := columnExists("saved_queries_sync", "visibility"); !exists {
		logger.Debug("Adding visibility to saved_queries_sync")
		if _, err := tx.Exec("ALTER TABLE saved_queries_sync ADD COLUMN visibility TEXT NOT NULL DEFAULT 'personal'"); err != nil {
			return fmt.Errorf("failed to add visibility: %w", err)
		}
	}

	if exists, _ := columnExists("saved_queries_sync", "created_by"); !exists {
		logger.Debug("Adding created_by to saved_queries_sync")
		if _, err := tx.Exec("ALTER TABLE saved_queries_sync ADD COLUMN created_by TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("failed to add created_by: %w", err)
		}
	}

	// Create indexes (IF NOT EXISTS is supported for indexes)
	logger.Debug("Creating indexes")
	_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_connections_org_visibility ON connection_templates(organization_id, visibility)")
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_connections_created_by ON connection_templates(created_by)")
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_queries_org_visibility ON saved_queries_sync(organization_id, visibility)")
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	_, err = tx.Exec("CREATE INDEX IF NOT EXISTS idx_queries_created_by ON saved_queries_sync(created_by)")
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Update existing records (only if created_by is empty)
	logger.Debug("Updating existing records")
	_, err = tx.Exec("UPDATE connection_templates SET created_by = user_id WHERE created_by = ''")
	if err != nil {
		return fmt.Errorf("failed to update connection_templates: %w", err)
	}

	_, err = tx.Exec("UPDATE saved_queries_sync SET created_by = user_id WHERE created_by = ''")
	if err != nil {
		return fmt.Errorf("failed to update saved_queries_sync: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
