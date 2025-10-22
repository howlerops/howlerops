package crypto

import (
	"context"
	"fmt"
)

// TeamSecretManager provides team-specific secret management
type TeamSecretManager struct {
	secretStore SecretStore
	keyStore    *KeyStore
}

// NewTeamSecretManager creates a new TeamSecretManager
func NewTeamSecretManager(store SecretStore, ks *KeyStore) *TeamSecretManager {
	return &TeamSecretManager{
		secretStore: store,
		keyStore:    ks,
	}
}

// StoreTeamSecret encrypts a secret using a team-specific key and stores it
func (tsm *TeamSecretManager) StoreTeamSecret(ctx context.Context, teamID string, secretType SecretType, plaintext []byte, teamMasterKey []byte) (*EncryptedSecret, error) {
	if len(teamMasterKey) != KeySize {
		return nil, fmt.Errorf("invalid team master key length: expected %d bytes, got %d", KeySize, len(teamMasterKey))
	}

	// Derive a unique key for this specific secret using the teamMasterKey and secretType as context
	derivationSalt := []byte(fmt.Sprintf("%s-%s", teamID, secretType))
	
	// Use DeriveKey to get the encryption key
	secretEncryptionKey, err := DeriveKey(string(teamMasterKey), derivationSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Encrypt the plaintext with the derived secretEncryptionKey
	ciphertext, nonce, err := EncryptSecret(plaintext, secretEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt team secret: %w", err)
	}

	// Combine ciphertext and nonce for storage (as per our current implementation)
	combinedCiphertext := append(nonce, ciphertext...)

	// Store the encrypted secret in the underlying SecretStore
	return tsm.secretStore.StoreSecret(ctx, teamID, secretType, combinedCiphertext, teamMasterKey)
}

// GetTeamSecret retrieves and decrypts a team secret
func (tsm *TeamSecretManager) GetTeamSecret(ctx context.Context, teamID string, secretType SecretType, teamMasterKey []byte) ([]byte, error) {
	if len(teamMasterKey) != KeySize {
		return nil, fmt.Errorf("invalid team master key length: expected %d bytes, got %d", KeySize, len(teamMasterKey))
	}

	// Retrieve the encrypted secret from the underlying SecretStore
	encryptedSecretBytes, err := tsm.secretStore.GetSecret(ctx, teamID, secretType)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve encrypted team secret: %w", err)
	}

	// Re-derive the salt as it's deterministic
	derivationSalt := []byte(fmt.Sprintf("%s-%s", teamID, secretType))
	secretEncryptionKey, err := DeriveKey(string(teamMasterKey), derivationSalt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive encryption key: %w", err)
	}

	// Split nonce and ciphertext
	if len(encryptedSecretBytes) < NonceSize {
		return nil, fmt.Errorf("invalid encrypted data length")
	}
	nonce := encryptedSecretBytes[:NonceSize]
	ciphertext := encryptedSecretBytes[NonceSize:]

	plaintext, err := DecryptSecret(ciphertext, nonce, secretEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt team secret: %w", err)
	}

	return plaintext, nil
}