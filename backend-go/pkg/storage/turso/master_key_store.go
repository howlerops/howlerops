package turso

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jbeck018/howlerops/backend-go/pkg/crypto"
	"github.com/sirupsen/logrus"
)

// MasterKeyStore handles storage operations for encrypted user master keys
type MasterKeyStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewMasterKeyStore creates a new master key store
func NewMasterKeyStore(db *sql.DB, logger *logrus.Logger) *MasterKeyStore {
	return &MasterKeyStore{
		db:     db,
		logger: logger,
	}
}

// StoreMasterKey stores an encrypted master key for a user
func (s *MasterKeyStore) StoreMasterKey(ctx context.Context, userID string, encryptedKey *crypto.EncryptedMasterKey) error {
	query := `
		INSERT INTO user_master_keys (
			user_id, encrypted_master_key, key_iv, key_auth_tag,
			pbkdf2_salt, pbkdf2_iterations, version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			encrypted_master_key = excluded.encrypted_master_key,
			key_iv = excluded.key_iv,
			key_auth_tag = excluded.key_auth_tag,
			pbkdf2_salt = excluded.pbkdf2_salt,
			pbkdf2_iterations = excluded.pbkdf2_iterations,
			version = version + 1,
			updated_at = excluded.updated_at
	`

	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, query,
		userID,
		encryptedKey.Ciphertext,
		encryptedKey.IV,
		encryptedKey.AuthTag,
		encryptedKey.Salt,
		encryptedKey.Iterations,
		1, // Initial version
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to store master key: %w", err)
	}

	s.logger.WithField("user_id", userID).Info("Master key stored successfully")
	return nil
}

// GetMasterKey retrieves an encrypted master key for a user
func (s *MasterKeyStore) GetMasterKey(ctx context.Context, userID string) (*crypto.EncryptedMasterKey, error) {
	query := `
		SELECT encrypted_master_key, key_iv, key_auth_tag, pbkdf2_salt, pbkdf2_iterations
		FROM user_master_keys
		WHERE user_id = ?
	`

	var encryptedKey crypto.EncryptedMasterKey
	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&encryptedKey.Ciphertext,
		&encryptedKey.IV,
		&encryptedKey.AuthTag,
		&encryptedKey.Salt,
		&encryptedKey.Iterations,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("master key not found for user")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve master key: %w", err)
	}

	return &encryptedKey, nil
}

// DeleteMasterKey removes a user's master key (when user is deleted)
func (s *MasterKeyStore) DeleteMasterKey(ctx context.Context, userID string) error {
	query := `DELETE FROM user_master_keys WHERE user_id = ?`

	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete master key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("master key not found")
	}

	s.logger.WithField("user_id", userID).Info("Master key deleted successfully")
	return nil
}

// CredentialStore handles storage operations for encrypted database credentials
type CredentialStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewCredentialStore creates a new credential store
func NewCredentialStore(db *sql.DB, logger *logrus.Logger) *CredentialStore {
	return &CredentialStore{
		db:     db,
		logger: logger,
	}
}

// StoreCredential stores an encrypted database password
func (s *CredentialStore) StoreCredential(ctx context.Context, userID, connectionID string, encryptedPassword *crypto.EncryptedPasswordData) error {
	query := `
		INSERT INTO encrypted_credentials (
			id, user_id, connection_id, encrypted_password,
			password_iv, password_auth_tag, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, connection_id) DO UPDATE SET
			encrypted_password = excluded.encrypted_password,
			password_iv = excluded.password_iv,
			password_auth_tag = excluded.password_auth_tag,
			updated_at = excluded.updated_at
	`

	now := time.Now().Unix()
	credID := uuid.New().String()

	_, err := s.db.ExecContext(ctx, query,
		credID,
		userID,
		connectionID,
		encryptedPassword.Ciphertext,
		encryptedPassword.IV,
		encryptedPassword.AuthTag,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to store credential: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"connection_id": connectionID,
	}).Info("Credential stored successfully")

	return nil
}

// GetCredential retrieves an encrypted password for a connection
func (s *CredentialStore) GetCredential(ctx context.Context, userID, connectionID string) (*crypto.EncryptedPasswordData, error) {
	query := `
		SELECT encrypted_password, password_iv, password_auth_tag
		FROM encrypted_credentials
		WHERE user_id = ? AND connection_id = ?
	`

	var encrypted crypto.EncryptedPasswordData
	err := s.db.QueryRowContext(ctx, query, userID, connectionID).Scan(
		&encrypted.Ciphertext,
		&encrypted.IV,
		&encrypted.AuthTag,
	)

	if err == sql.ErrNoRows {
		// Not all connections have passwords
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve credential: %w", err)
	}

	return &encrypted, nil
}

// DeleteCredential removes an encrypted password for a connection
func (s *CredentialStore) DeleteCredential(ctx context.Context, userID, connectionID string) error {
	query := `DELETE FROM encrypted_credentials WHERE user_id = ? AND connection_id = ?`

	_, err := s.db.ExecContext(ctx, query, userID, connectionID)
	if err != nil {
		return fmt.Errorf("failed to delete credential: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":       userID,
		"connection_id": connectionID,
	}).Info("Credential deleted successfully")

	return nil
}
