package storage

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/crypto"
	_ "github.com/mattn/go-sqlite3"
)

func TestMigrationManager_RunMigrations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)
	migrationManager := NewMigrationManager(db, secretStore, logger)

	// Create a mock secret manager
	keyStore := crypto.NewKeyStore()
	secretManager := crypto.NewSecretManager(secretStore, keyStore)

	// Unlock the key store with a test passphrase
	err := keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	ctx := context.Background()

	// Run migrations
	err = migrationManager.RunMigrations(ctx, secretManager)
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Verify migration status
	status, err := secretStore.GetMigrationStatus(ctx, "003_migrate_passwords_to_secrets")
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "completed" {
		t.Errorf("Expected migration status 'completed', got %s", status)
	}
}

func TestMigrationManager_GetMigrationHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)
	migrationManager := NewMigrationManager(db, secretStore, logger)

	ctx := context.Background()

	// Set some migration statuses
	migrations := []struct {
		name   string
		status string
	}{
		{"test-migration-1", "completed"},
		{"test-migration-2", "failed"},
		{"test-migration-3", "in_progress"},
	}

	for _, migration := range migrations {
		err := secretStore.SetMigrationStatus(ctx, migration.name, migration.status)
		if err != nil {
			t.Fatalf("Failed to set migration status: %v", err)
		}
	}

	// Get migration history
	history, err := migrationManager.GetMigrationHistory(ctx)
	if err != nil {
		t.Fatalf("Failed to get migration history: %v", err)
	}

	if len(history) != len(migrations) {
		t.Errorf("Expected %d migrations, got %d", len(migrations), len(history))
	}

	// Verify migration records
	migrationMap := make(map[string]string)
	for _, record := range history {
		migrationMap[record.Name] = record.Status
	}

	for _, expectedMigration := range migrations {
		status, exists := migrationMap[expectedMigration.name]
		if !exists {
			t.Errorf("Expected migration %s not found in history", expectedMigration.name)
		}
		if status != expectedMigration.status {
			t.Errorf("Expected migration %s status %s, got %s", expectedMigration.name, expectedMigration.status, status)
		}
	}
}

func TestMigrationManager_CleanupOldPasswords(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)
	migrationManager := NewMigrationManager(db, secretStore, logger)

	ctx := context.Background()

	// Test cleanup when migration is not completed
	err := migrationManager.CleanupOldPasswords(ctx)
	if err == nil {
		t.Error("Expected error when migration is not completed")
	}

	// Mark migration as completed
	err = secretStore.SetMigrationStatus(ctx, "003_migrate_passwords_to_secrets", "completed")
	if err != nil {
		t.Fatalf("Failed to set migration status: %v", err)
	}

	// Test cleanup when no passwords remain
	err = migrationManager.CleanupOldPasswords(ctx)
	if err != nil {
		t.Errorf("Failed to cleanup old passwords: %v", err)
	}
}

func TestMigrationManager_MigratePasswordsToSecrets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)
	migrationManager := NewMigrationManager(db, secretStore, logger)

	// Create connections table with password_encrypted data
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS connections (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			host TEXT,
			port INTEGER,
			database_name TEXT,
			username TEXT,
			password_encrypted TEXT,
			ssl_config TEXT,
			created_by TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL,
			team_id TEXT,
			is_shared BOOLEAN DEFAULT 0,
			metadata TEXT
		);

		INSERT INTO connections (id, name, type, password_encrypted, created_by, created_at, updated_at)
		VALUES 
			('conn-1', 'Test DB 1', 'postgresql', 'encrypted-password-1', 'user1', ?, ?),
			('conn-2', 'Test DB 2', 'mysql', 'encrypted-password-2', 'user2', ?, ?),
			('conn-3', 'Test DB 3', 'sqlite', NULL, 'user3', ?, ?);
	`, time.Now().Unix(), time.Now().Unix(), time.Now().Unix(), time.Now().Unix(), time.Now().Unix(), time.Now().Unix())
	if err != nil {
		t.Fatalf("Failed to create test connections: %v", err)
	}

	// Create a mock secret manager
	keyStore := crypto.NewKeyStore()
	secretManager := crypto.NewSecretManager(secretStore, keyStore)

	// Unlock the key store
	err = keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	ctx := context.Background()

	// Run password migration
	err = migrationManager.migratePasswordsToSecrets(ctx, secretManager)
	if err != nil {
		t.Fatalf("Failed to migrate passwords: %v", err)
	}

	// Verify passwords were migrated
	secrets, err := secretStore.ListSecrets(ctx, "conn-1")
	if err != nil {
		t.Fatalf("Failed to list secrets for conn-1: %v", err)
	}

	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret for conn-1, got %d", len(secrets))
	}

	if secrets[0] != crypto.SecretTypeDBPassword {
		t.Errorf("Expected secret type %s, got %s", crypto.SecretTypeDBPassword, secrets[0])
	}

	// Verify conn-2 has a secret
	secrets, err = secretStore.ListSecrets(ctx, "conn-2")
	if err != nil {
		t.Fatalf("Failed to list secrets for conn-2: %v", err)
	}

	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret for conn-2, got %d", len(secrets))
	}

	// Verify conn-3 has no secrets (no password)
	secrets, err = secretStore.ListSecrets(ctx, "conn-3")
	if err != nil {
		t.Fatalf("Failed to list secrets for conn-3: %v", err)
	}

	if len(secrets) != 0 {
		t.Errorf("Expected 0 secrets for conn-3, got %d", len(secrets))
	}

	// Verify migration status
	status, err := secretStore.GetMigrationStatus(ctx, "003_migrate_passwords_to_secrets")
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "completed" {
		t.Errorf("Expected migration status 'completed', got %s", status)
	}
}

func TestMigrationManager_MigratePasswordsToSecrets_AlreadyCompleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)
	migrationManager := NewMigrationManager(db, secretStore, logger)

	ctx := context.Background()

	// Mark migration as already completed
	err := secretStore.SetMigrationStatus(ctx, "003_migrate_passwords_to_secrets", "completed")
	if err != nil {
		t.Fatalf("Failed to set migration status: %v", err)
	}

	// Create a mock secret manager
	keyStore := crypto.NewKeyStore()
	secretManager := crypto.NewSecretManager(secretStore, keyStore)

	// Run migration - should skip
	err = migrationManager.migratePasswordsToSecrets(ctx, secretManager)
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Verify status is still completed
	status, err := secretStore.GetMigrationStatus(ctx, "003_migrate_passwords_to_secrets")
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "completed" {
		t.Errorf("Expected migration status 'completed', got %s", status)
	}
}

func TestMigrationManager_MigratePasswordsToSecrets_InProgress(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)
	migrationManager := NewMigrationManager(db, secretStore, logger)

	ctx := context.Background()

	// Mark migration as in progress
	err := secretStore.SetMigrationStatus(ctx, "003_migrate_passwords_to_secrets", "in_progress")
	if err != nil {
		t.Fatalf("Failed to set migration status: %v", err)
	}

	// Create a mock secret manager
	keyStore := crypto.NewKeyStore()
	secretManager := crypto.NewSecretManager(secretStore, keyStore)

	// Run migration - should skip
	err = migrationManager.migratePasswordsToSecrets(ctx, secretManager)
	if err != nil {
		t.Fatalf("Failed to run migration: %v", err)
	}

	// Verify status is still in progress
	status, err := secretStore.GetMigrationStatus(ctx, "003_migrate_passwords_to_secrets")
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "in_progress" {
		t.Errorf("Expected migration status 'in_progress', got %s", status)
	}
}
