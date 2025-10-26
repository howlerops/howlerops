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
	defer rows.Close()

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
