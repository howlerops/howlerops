package crypto

import (
	"context"
	"fmt"
	"time"
)

// SecretType represents the type of secret being stored
type SecretType string

const (
	SecretTypeDBPassword    SecretType = "db_password"
	SecretTypeSSHPassword   SecretType = "ssh_password"
	SecretTypeSSHPrivateKey SecretType = "ssh_private_key"
	SecretTypeAPIKey        SecretType = "api_key"
)

// EncryptedSecret represents a secret stored in an encrypted format
type EncryptedSecret struct {
	ID         string
	OwnerID    string // Connection ID, Team ID, User ID, etc.
	Type       SecretType
	Ciphertext []byte
	Salt       []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SecretStore defines the interface for storing and retrieving encrypted secrets
type SecretStore interface {
	// StoreSecret encrypts and stores a secret
	StoreSecret(ctx context.Context, ownerID string, secretType SecretType, plaintext []byte, sessionKey []byte) (*EncryptedSecret, error)

	// GetSecret retrieves and decrypts a secret
	GetSecret(ctx context.Context, ownerID string, secretType SecretType) ([]byte, error)

	// DeleteSecret removes a secret
	DeleteSecret(ctx context.Context, ownerID string, secretType SecretType) error

	// ListSecrets returns all secrets for an owner
	ListSecrets(ctx context.Context, ownerID string) ([]*EncryptedSecret, error)
}

// SecretManager provides high-level secret management operations
type SecretManager struct {
	store    SecretStore
	keyStore *KeyStore
}

// NewSecretManager creates a new secret manager
func NewSecretManager(store SecretStore, keyStore *KeyStore) *SecretManager {
	return &SecretManager{
		store:    store,
		keyStore: keyStore,
	}
}

// StoreSecret encrypts and stores a secret using a session key
func (sm *SecretManager) StoreSecret(ctx context.Context, connectionID string, secretType SecretType, plaintext []byte) error {
	// Get user key from key store
	userKey, err := sm.keyStore.GetUserKey()
	if err != nil {
		return fmt.Errorf("failed to get user key: %w", err)
	}

	// Store in database using the SecretStore interface
	_, err = sm.store.StoreSecret(ctx, connectionID, secretType, plaintext, userKey)
	if err != nil {
		return fmt.Errorf("failed to store secret: %w", err)
	}

	return nil
}

// GetSecret retrieves and decrypts a secret
func (sm *SecretManager) GetSecret(ctx context.Context, connectionID string, secretType SecretType) ([]byte, error) {
	// Get encrypted secret from store
	plaintext, err := sm.store.GetSecret(ctx, connectionID, secretType)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret from store: %w", err)
	}

	return plaintext, nil
}

// StoreTeamSecret encrypts and stores a secret using a team key
func (sm *SecretManager) StoreTeamSecret(ctx context.Context, connectionID string, secretType SecretType, plaintext []byte, teamID string) error {
	// For now, use the same session key approach
	// TODO: Implement team-specific key derivation
	return sm.StoreSecret(ctx, connectionID, secretType, plaintext)
}

// GetTeamSecret retrieves and decrypts a team secret
func (sm *SecretManager) GetTeamSecret(ctx context.Context, connectionID string, secretType SecretType, teamID string) ([]byte, error) {
	// For now, use the same approach as regular secrets
	// TODO: Implement team-specific key derivation
	return sm.GetSecret(ctx, connectionID, secretType)
}

// DeleteSecret removes a secret
func (sm *SecretManager) DeleteSecret(ctx context.Context, connectionID string, secretType SecretType) error {
	return sm.store.DeleteSecret(ctx, connectionID, secretType)
}

// ListSecrets returns all secrets for a connection
func (sm *SecretManager) ListSecrets(ctx context.Context, connectionID string) ([]*EncryptedSecret, error) {
	return sm.store.ListSecrets(ctx, connectionID)
}

// ReencryptAllSecrets re-encrypts all secrets with a new key
// This is used for key rotation or passphrase changes
func (sm *SecretManager) ReencryptAllSecrets(ctx context.Context, oldKey []byte, newKey []byte) error {
	// TODO: Implement key rotation
	// This would involve:
	// 1. Getting all secrets from the store
	// 2. Decrypting with old key
	// 3. Re-encrypting with new key
	// 4. Updating in store
	return fmt.Errorf("key rotation not implemented yet")
}
