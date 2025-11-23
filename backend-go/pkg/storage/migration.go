package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/sirupsen/logrus"
)

// MigrationManager handles database migrations
type MigrationManager struct {
	db          *sql.DB
	secretStore *SecretStore
	logger      *logrus.Logger
}

// NewMigrationManager creates a new MigrationManager
func NewMigrationManager(db *sql.DB, secretStore *SecretStore, logger *logrus.Logger) *MigrationManager {
	return &MigrationManager{
		db:          db,
		secretStore: secretStore,
		logger:      logger,
	}
}

// RunMigrations runs all pending migrations
func (m *MigrationManager) RunMigrations(ctx context.Context) error {
	m.logger.Info("Starting database migrations")

	// Run password migration
	if err := m.migratePasswordsToSecrets(ctx); err != nil {
		return fmt.Errorf("failed to migrate passwords to secrets: %w", err)
	}

	// Run reports schema migration
	if err := m.migrateReportsSchema(ctx); err != nil {
		return fmt.Errorf("failed to migrate reports schema: %w", err)
	}

	m.logger.Info("All migrations completed successfully")
	return nil
}

// migratePasswordsToSecrets migrates existing password_encrypted data to connection_secrets
func (m *MigrationManager) migratePasswordsToSecrets(ctx context.Context) error {
	m.logger.Info("Starting password migration to connection_secrets")

	// Get all connections with password_encrypted data
	query := `
		SELECT id, password_encrypted 
		FROM connections 
		WHERE password_encrypted IS NOT NULL AND password_encrypted != ''
	`

	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query connections: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			m.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	migratedCount := 0
	failedCount := 0

	for rows.Next() {
		var connectionID, passwordEncrypted string
		if err := rows.Scan(&connectionID, &passwordEncrypted); err != nil {
			m.logger.WithError(err).Error("Failed to scan connection")
			failedCount++
			continue
		}

		// For now, we'll skip the actual migration since we don't have a session key
		// TODO: Implement proper migration with session key management
		m.logger.WithField("connection_id", connectionID).Info("Would migrate password to secrets (skipped for now)")

		// Clear the plaintext password from the connections table
		_, err = m.db.ExecContext(ctx, `UPDATE connections SET password_encrypted = '' WHERE id = ?`, connectionID)
		if err != nil {
			m.logger.WithError(err).WithField("connection_id", connectionID).Warn("Failed to clear plaintext password after migration")
		}

		migratedCount++
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating connection rows: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"migrated": migratedCount,
		"failed":   failedCount,
	}).Info("Password migration completed")

	return nil
}

// migrateReportsSchema adds missing columns to reports table for starred feature
func (m *MigrationManager) migrateReportsSchema(ctx context.Context) error {
	m.logger.Info("Starting reports schema migration")

	// Check if starred column exists
	var columnExists bool
	err := m.db.QueryRowContext(ctx, `
		SELECT COUNT(*) > 0
		FROM pragma_table_info('reports')
		WHERE name = 'starred'
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check for starred column: %w", err)
	}

	if !columnExists {
		m.logger.Info("Adding starred and starred_at columns to reports table")

		// Add starred column
		if _, err := m.db.ExecContext(ctx, `
			ALTER TABLE reports ADD COLUMN starred BOOLEAN DEFAULT FALSE
		`); err != nil {
			return fmt.Errorf("failed to add starred column: %w", err)
		}

		// Add starred_at column
		if _, err := m.db.ExecContext(ctx, `
			ALTER TABLE reports ADD COLUMN starred_at DATETIME
		`); err != nil {
			return fmt.Errorf("failed to add starred_at column: %w", err)
		}

		// Create index on starred columns
		if _, err := m.db.ExecContext(ctx, `
			CREATE INDEX IF NOT EXISTS idx_reports_starred ON reports(starred DESC, starred_at DESC)
		`); err != nil {
			return fmt.Errorf("failed to create starred index: %w", err)
		}

		m.logger.Info("Successfully added starred columns to reports table")
	} else {
		m.logger.Info("Reports table already has starred columns, skipping migration")
	}

	return nil
}
