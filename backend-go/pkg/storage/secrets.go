package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/jbeck018/howlerops/backend-go/pkg/crypto"
)

// SecretStore implements the crypto.SecretStore interface for SQLite storage
type SecretStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewSecretStore creates a new SecretStore instance
func NewSecretStore(db *sql.DB, logger *logrus.Logger) *SecretStore {
	return &SecretStore{
		db:     db,
		logger: logger,
	}
}

// StoreSecret encrypts and stores a secret in the database
func (s *SecretStore) StoreSecret(ctx context.Context, ownerID string, secretType crypto.SecretType, plaintext []byte, sessionKey []byte) (*crypto.EncryptedSecret, error) {
	if len(sessionKey) != crypto.KeySize {
		return nil, fmt.Errorf("invalid session key length: expected %d bytes, got %d", crypto.KeySize, len(sessionKey))
	}

	ciphertext, nonce, err := crypto.EncryptSecret(plaintext, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Combine nonce and ciphertext for storage
	combinedCiphertext := append(nonce, ciphertext...)

	now := time.Now()
	secret := &crypto.EncryptedSecret{
		ID:         fmt.Sprintf("%s-%s", ownerID, secretType),
		OwnerID:    ownerID,
		Type:       secretType,
		Ciphertext: combinedCiphertext,
		Salt:       []byte("dummy_salt_for_now"),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO connection_secrets (id, owner_id, secret_type, ciphertext, salt, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			ciphertext = excluded.ciphertext,
			salt = excluded.salt,
			updated_at = excluded.updated_at
	`, secret.ID, secret.OwnerID, secret.Type, secret.Ciphertext, secret.Salt, secret.CreatedAt.Unix(), secret.UpdatedAt.Unix())

	if err != nil {
		return nil, fmt.Errorf("failed to store encrypted secret: %w", err)
	}

	return secret, nil
}

// GetSecret retrieves and decrypts a secret from the database
func (s *SecretStore) GetSecret(ctx context.Context, ownerID string, secretType crypto.SecretType) ([]byte, error) {
	var ciphertext []byte
	var salt []byte
	var id string

	row := s.db.QueryRowContext(ctx, `
		SELECT id, ciphertext, salt FROM connection_secrets WHERE owner_id = ? AND secret_type = ?
	`, ownerID, secretType)

	err := row.Scan(&id, &ciphertext, &salt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("secret not found for owner %s and type %s", ownerID, secretType)
		}
		return nil, fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// TODO: Retrieve session key from KeyStore based on current session
	return nil, fmt.Errorf("session key not available for decryption")
}

// DeleteSecret removes a secret from the database
func (s *SecretStore) DeleteSecret(ctx context.Context, ownerID string, secretType crypto.SecretType) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM connection_secrets WHERE owner_id = ? AND secret_type = ?
	`, ownerID, secretType)
	if err != nil {
		return fmt.Errorf("failed to delete secret: %w", err)
	}
	return nil
}

// ListSecrets lists all secrets for a given owner
func (s *SecretStore) ListSecrets(ctx context.Context, ownerID string) ([]*crypto.EncryptedSecret, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, owner_id, secret_type, ciphertext, salt, created_at, updated_at
		FROM connection_secrets WHERE owner_id = ?
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't fail the query - data already retrieved
			_ = err
		}
	}()

	var secrets []*crypto.EncryptedSecret
	for rows.Next() {
		secret := &crypto.EncryptedSecret{}
		var createdAt, updatedAt int64
		err := rows.Scan(&secret.ID, &secret.OwnerID, &secret.Type, &secret.Ciphertext, &secret.Salt, &createdAt, &updatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan secret row: %w", err)
		}
		secret.CreatedAt = time.Unix(createdAt, 0)
		secret.UpdatedAt = time.Unix(updatedAt, 0)
		secrets = append(secrets, secret)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating secret rows: %w", err)
	}

	return secrets, nil
}
