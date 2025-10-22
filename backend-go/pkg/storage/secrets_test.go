package storage

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/crypto"
	_ "github.com/mattn/go-sqlite3"
)

func setupTestDB(t *testing.T) *sql.DB {
	// Create temporary database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS connection_secrets (
			connection_id TEXT NOT NULL,
			secret_type TEXT NOT NULL,
			ciphertext BLOB NOT NULL,
			nonce BLOB NOT NULL,
			salt BLOB,
			key_version INTEGER DEFAULT 1,
			updated_at INTEGER NOT NULL,
			updated_by TEXT NOT NULL,
			team_id TEXT,
			PRIMARY KEY (connection_id, secret_type)
		);

		CREATE TABLE IF NOT EXISTS migration_status (
			migration_name TEXT PRIMARY KEY,
			status TEXT NOT NULL,
			started_at INTEGER,
			completed_at INTEGER,
			error_message TEXT
		);
	`)
	if err != nil {
		t.Fatalf("Failed to create test tables: %v", err)
	}

	return db
}

func TestSecretStore_StoreSecret(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	secret := &crypto.Secret{
		ConnectionID: "test-connection-1",
		SecretType:   crypto.SecretTypeDBPassword,
		Ciphertext:   []byte("encrypted-password"),
		Nonce:        []byte("nonce-12345678"),
		Salt:         []byte("salt-12345678"),
		KeyVersion:   1,
		UpdatedAt:    time.Now(),
		UpdatedBy:    "test-user",
		TeamID:       "",
	}

	err := secretStore.StoreSecret(ctx, secret)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Verify secret was stored
	var count int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM connection_secrets 
		WHERE connection_id = ? AND secret_type = ?
	`, secret.ConnectionID, secret.SecretType).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query secret count: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 secret, got %d", count)
	}
}

func TestSecretStore_GetSecret(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	secret := &crypto.Secret{
		ConnectionID: "test-connection-1",
		SecretType:   crypto.SecretTypeDBPassword,
		Ciphertext:   []byte("encrypted-password"),
		Nonce:        []byte("nonce-12345678"),
		Salt:         []byte("salt-12345678"),
		KeyVersion:   1,
		UpdatedAt:    time.Now(),
		UpdatedBy:    "test-user",
		TeamID:       "",
	}

	// Store secret
	err := secretStore.StoreSecret(ctx, secret)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Retrieve secret
	retrievedSecret, err := secretStore.GetSecret(ctx, secret.ConnectionID, secret.SecretType)
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	// Verify secret data
	if retrievedSecret.ConnectionID != secret.ConnectionID {
		t.Errorf("Expected connection ID %s, got %s", secret.ConnectionID, retrievedSecret.ConnectionID)
	}

	if retrievedSecret.SecretType != secret.SecretType {
		t.Errorf("Expected secret type %s, got %s", secret.SecretType, retrievedSecret.SecretType)
	}

	if string(retrievedSecret.Ciphertext) != string(secret.Ciphertext) {
		t.Errorf("Expected ciphertext %s, got %s", string(secret.Ciphertext), string(retrievedSecret.Ciphertext))
	}

	if string(retrievedSecret.Nonce) != string(secret.Nonce) {
		t.Errorf("Expected nonce %s, got %s", string(secret.Nonce), string(retrievedSecret.Nonce))
	}
}

func TestSecretStore_DeleteSecret(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	secret := &crypto.Secret{
		ConnectionID: "test-connection-1",
		SecretType:   crypto.SecretTypeDBPassword,
		Ciphertext:   []byte("encrypted-password"),
		Nonce:        []byte("nonce-12345678"),
		Salt:         []byte("salt-12345678"),
		KeyVersion:   1,
		UpdatedAt:    time.Now(),
		UpdatedBy:    "test-user",
		TeamID:       "",
	}

	// Store secret
	err := secretStore.StoreSecret(ctx, secret)
	if err != nil {
		t.Fatalf("Failed to store secret: %v", err)
	}

	// Delete secret
	err = secretStore.DeleteSecret(ctx, secret.ConnectionID, secret.SecretType)
	if err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	// Verify secret was deleted
	var count int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM connection_secrets 
		WHERE connection_id = ? AND secret_type = ?
	`, secret.ConnectionID, secret.SecretType).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query secret count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 secrets after deletion, got %d", count)
	}
}

func TestSecretStore_DeleteConnectionSecrets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	connectionID := "test-connection-1"

	// Store multiple secrets for the same connection
	secrets := []*crypto.Secret{
		{
			ConnectionID: connectionID,
			SecretType:   crypto.SecretTypeDBPassword,
			Ciphertext:   []byte("encrypted-password"),
			Nonce:        []byte("nonce-1"),
			Salt:         []byte("salt-1"),
			KeyVersion:   1,
			UpdatedAt:    time.Now(),
			UpdatedBy:    "test-user",
			TeamID:       "",
		},
		{
			ConnectionID: connectionID,
			SecretType:   crypto.SecretTypeSSHPassword,
			Ciphertext:   []byte("encrypted-ssh-password"),
			Nonce:        []byte("nonce-2"),
			Salt:         []byte("salt-2"),
			KeyVersion:   1,
			UpdatedAt:    time.Now(),
			UpdatedBy:    "test-user",
			TeamID:       "",
		},
	}

	for _, secret := range secrets {
		err := secretStore.StoreSecret(ctx, secret)
		if err != nil {
			t.Fatalf("Failed to store secret: %v", err)
		}
	}

	// Delete all secrets for the connection
	err := secretStore.DeleteConnectionSecrets(ctx, connectionID)
	if err != nil {
		t.Fatalf("Failed to delete connection secrets: %v", err)
	}

	// Verify all secrets were deleted
	var count int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM connection_secrets 
		WHERE connection_id = ?
	`, connectionID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query secret count: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 secrets after deletion, got %d", count)
	}
}

func TestSecretStore_ListSecrets(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	connectionID := "test-connection-1"

	// Store multiple secrets
	secretTypes := []crypto.SecretType{
		crypto.SecretTypeDBPassword,
		crypto.SecretTypeSSHPassword,
		crypto.SecretTypeSSHPrivateKey,
	}

	for i, secretType := range secretTypes {
		secret := &crypto.Secret{
			ConnectionID: connectionID,
			SecretType:   secretType,
			Ciphertext:   []byte("encrypted-data"),
			Nonce:        []byte("nonce"),
			Salt:         []byte("salt"),
			KeyVersion:   1,
			UpdatedAt:    time.Now(),
			UpdatedBy:    "test-user",
			TeamID:       "",
		}

		err := secretStore.StoreSecret(ctx, secret)
		if err != nil {
			t.Fatalf("Failed to store secret %d: %v", i, err)
		}
	}

	// List secrets
	secretTypesList, err := secretStore.ListSecrets(ctx, connectionID)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(secretTypesList) != len(secretTypes) {
		t.Errorf("Expected %d secrets, got %d", len(secretTypes), len(secretTypesList))
	}

	// Verify all secret types are present
	secretTypeMap := make(map[crypto.SecretType]bool)
	for _, secretType := range secretTypesList {
		secretTypeMap[secretType] = true
	}

	for _, expectedType := range secretTypes {
		if !secretTypeMap[expectedType] {
			t.Errorf("Expected secret type %s not found in list", expectedType)
		}
	}
}

func TestSecretStore_GetSecretsByTeam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	teamID := "test-team-1"

	// Store secrets for different teams
	secrets := []*crypto.Secret{
		{
			ConnectionID: "connection-1",
			SecretType:   crypto.SecretTypeDBPassword,
			Ciphertext:   []byte("encrypted-password-1"),
			Nonce:        []byte("nonce-1"),
			Salt:         []byte("salt-1"),
			KeyVersion:   1,
			UpdatedAt:    time.Now(),
			UpdatedBy:    "test-user",
			TeamID:       teamID,
		},
		{
			ConnectionID: "connection-2",
			SecretType:   crypto.SecretTypeSSHPassword,
			Ciphertext:   []byte("encrypted-ssh-password-2"),
			Nonce:        []byte("nonce-2"),
			Salt:         []byte("salt-2"),
			KeyVersion:   1,
			UpdatedAt:    time.Now(),
			UpdatedBy:    "test-user",
			TeamID:       teamID,
		},
		{
			ConnectionID: "connection-3",
			SecretType:   crypto.SecretTypeDBPassword,
			Ciphertext:   []byte("encrypted-password-3"),
			Nonce:        []byte("nonce-3"),
			Salt:         []byte("salt-3"),
			KeyVersion:   1,
			UpdatedAt:    time.Now(),
			UpdatedBy:    "test-user",
			TeamID:       "other-team",
		},
	}

	for _, secret := range secrets {
		err := secretStore.StoreSecret(ctx, secret)
		if err != nil {
			t.Fatalf("Failed to store secret: %v", err)
		}
	}

	// Get secrets for specific team
	teamSecrets, err := secretStore.GetSecretsByTeam(ctx, teamID)
	if err != nil {
		t.Fatalf("Failed to get team secrets: %v", err)
	}

	if len(teamSecrets) != 2 {
		t.Errorf("Expected 2 team secrets, got %d", len(teamSecrets))
	}

	// Verify all secrets belong to the correct team
	for _, secret := range teamSecrets {
		if secret.TeamID != teamID {
			t.Errorf("Expected team ID %s, got %s", teamID, secret.TeamID)
		}
	}
}

func TestSecretStore_MigrationStatus(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	migrationName := "test-migration"

	// Test getting status for non-existent migration
	status, err := secretStore.GetMigrationStatus(ctx, migrationName)
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "not_found" {
		t.Errorf("Expected status 'not_found', got %s", status)
	}

	// Test setting migration status
	err = secretStore.SetMigrationStatus(ctx, migrationName, "in_progress")
	if err != nil {
		t.Fatalf("Failed to set migration status: %v", err)
	}

	// Test getting the status
	status, err = secretStore.GetMigrationStatus(ctx, migrationName)
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "in_progress" {
		t.Errorf("Expected status 'in_progress', got %s", status)
	}

	// Test updating status
	err = secretStore.SetMigrationStatus(ctx, migrationName, "completed")
	if err != nil {
		t.Fatalf("Failed to update migration status: %v", err)
	}

	status, err = secretStore.GetMigrationStatus(ctx, migrationName)
	if err != nil {
		t.Fatalf("Failed to get migration status: %v", err)
	}

	if status != "completed" {
		t.Errorf("Expected status 'completed', got %s", status)
	}
}

func TestSecretStore_ConcurrentAccess(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()
	connectionID := "test-connection-1"

	// Test concurrent writes
	numGoroutines := 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(i int) {
			secret := &crypto.Secret{
				ConnectionID: connectionID,
				SecretType:   crypto.SecretType(crypto.SecretTypeDBPassword + crypto.SecretType(i)),
				Ciphertext:   []byte("encrypted-data"),
				Nonce:        []byte("nonce"),
				Salt:         []byte("salt"),
				KeyVersion:   1,
				UpdatedAt:    time.Now(),
				UpdatedBy:    "test-user",
				TeamID:       "",
			}

			err := secretStore.StoreSecret(ctx, secret)
			done <- err
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		if err != nil {
			t.Errorf("Goroutine %d failed: %v", i, err)
		}
	}

	// Verify all secrets were stored
	var count int
	err := db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM connection_secrets 
		WHERE connection_id = ?
	`, connectionID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query secret count: %v", err)
	}

	if count != numGoroutines {
		t.Errorf("Expected %d secrets, got %d", numGoroutines, count)
	}
}

func TestSecretStore_ErrorHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	secretStore := NewSecretStore(db, logger)

	ctx := context.Background()

	// Test getting non-existent secret
	_, err := secretStore.GetSecret(ctx, "non-existent", crypto.SecretTypeDBPassword)
	if err == nil {
		t.Error("Expected error for non-existent secret")
	}

	// Test deleting non-existent secret
	err = secretStore.DeleteSecret(ctx, "non-existent", crypto.SecretTypeDBPassword)
	if err == nil {
		t.Error("Expected error for deleting non-existent secret")
	}

	// Test listing secrets for non-existent connection
	secretTypes, err := secretStore.ListSecrets(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Failed to list secrets for non-existent connection: %v", err)
	}

	if len(secretTypes) != 0 {
		t.Errorf("Expected 0 secrets for non-existent connection, got %d", len(secretTypes))
	}
}
