package crypto

import (
	"context"
	"fmt"
	"testing"
)

func TestTeamSecretManager_StoreTeamSecret(t *testing.T) {
	// Create mock secret store
	store := &mockSecretStore{}
	keyStore := NewKeyStore()
	secretManager := NewSecretManager(store, keyStore)

	// Unlock key store
	err := keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	// Create team secret manager
	teamID := "test-team-1"
	teamSecret := []byte("test-team-secret")
	teamSecretManager := NewTeamSecretManager(secretManager, keyStore, teamID, teamSecret)

	ctx := context.Background()
	connectionID := "test-connection-1"
	secretType := SecretTypeDBPassword
	plaintext := []byte("test-password")

	// Store team secret
	err = teamSecretManager.StoreTeamSecret(ctx, connectionID, secretType, plaintext)
	if err != nil {
		t.Fatalf("Failed to store team secret: %v", err)
	}

	// Verify secret was stored
	if len(store.secrets) != 1 {
		t.Errorf("Expected 1 secret stored, got %d", len(store.secrets))
	}

	storedSecret := store.secrets[0]
	if storedSecret.ConnectionID != connectionID {
		t.Errorf("Expected connection ID %s, got %s", connectionID, storedSecret.ConnectionID)
	}

	if storedSecret.SecretType != secretType {
		t.Errorf("Expected secret type %s, got %s", secretType, storedSecret.SecretType)
	}

	if storedSecret.TeamID != teamID {
		t.Errorf("Expected team ID %s, got %s", teamID, storedSecret.TeamID)
	}
}

func TestTeamSecretManager_GetTeamSecret(t *testing.T) {
	// Create mock secret store
	store := &mockSecretStore{}
	keyStore := NewKeyStore()
	secretManager := NewSecretManager(store, keyStore)

	// Unlock key store
	err := keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	// Create team secret manager
	teamID := "test-team-1"
	teamSecret := []byte("test-team-secret")
	teamSecretManager := NewTeamSecretManager(secretManager, keyStore, teamID, teamSecret)

	ctx := context.Background()
	connectionID := "test-connection-1"
	secretType := SecretTypeDBPassword
	plaintext := []byte("test-password")

	// Store team secret
	err = teamSecretManager.StoreTeamSecret(ctx, connectionID, secretType, plaintext)
	if err != nil {
		t.Fatalf("Failed to store team secret: %v", err)
	}

	// Retrieve team secret
	retrievedPlaintext, err := teamSecretManager.GetTeamSecret(ctx, connectionID, secretType)
	if err != nil {
		t.Fatalf("Failed to get team secret: %v", err)
	}

	// Verify plaintext matches
	if string(retrievedPlaintext) != string(plaintext) {
		t.Errorf("Expected plaintext %s, got %s", string(plaintext), string(retrievedPlaintext))
	}
}

func TestTeamSecretManager_RotateTeamSecret(t *testing.T) {
	// Create mock secret store
	store := &mockSecretStore{}
	keyStore := NewKeyStore()
	secretManager := NewSecretManager(store, keyStore)

	// Unlock key store
	err := keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	// Create team secret manager
	teamID := "test-team-1"
	teamSecret := []byte("test-team-secret")
	teamSecretManager := NewTeamSecretManager(secretManager, keyStore, teamID, teamSecret)

	ctx := context.Background()
	connectionID := "test-connection-1"
	secretType := SecretTypeDBPassword
	plaintext := []byte("test-password")

	// Store team secret
	err = teamSecretManager.StoreTeamSecret(ctx, connectionID, secretType, plaintext)
	if err != nil {
		t.Fatalf("Failed to store team secret: %v", err)
	}

	// Rotate team secret
	newTeamSecret := []byte("new-team-secret")
	err = teamSecretManager.RotateTeamSecret(ctx, newTeamSecret)
	if err != nil {
		t.Fatalf("Failed to rotate team secret: %v", err)
	}

	// Verify secret can still be retrieved
	retrievedPlaintext, err := teamSecretManager.GetTeamSecret(ctx, connectionID, secretType)
	if err != nil {
		t.Fatalf("Failed to get team secret after rotation: %v", err)
	}

	if string(retrievedPlaintext) != string(plaintext) {
		t.Errorf("Expected plaintext %s, got %s", string(plaintext), string(retrievedPlaintext))
	}

	// Verify team secret was updated
	if string(teamSecretManager.teamSecret) != string(newTeamSecret) {
		t.Errorf("Expected team secret %s, got %s", string(newTeamSecret), string(teamSecretManager.teamSecret))
	}
}

func TestTeamSecretManager_ReencryptSecretsForNewMember(t *testing.T) {
	// Create mock secret store
	store := &mockSecretStore{}
	keyStore := NewKeyStore()
	secretManager := NewSecretManager(store, keyStore)

	// Unlock key store
	err := keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	// Create team secret manager
	teamID := "test-team-1"
	teamSecret := []byte("test-team-secret")
	teamSecretManager := NewTeamSecretManager(secretManager, keyStore, teamID, teamSecret)

	ctx := context.Background()
	connectionID := "test-connection-1"
	secretType := SecretTypeDBPassword
	plaintext := []byte("test-password")

	// Store team secret
	err = teamSecretManager.StoreTeamSecret(ctx, connectionID, secretType, plaintext)
	if err != nil {
		t.Fatalf("Failed to store team secret: %v", err)
	}

	// Generate new user key
	newUserKey, err := GenerateRandomBytes(KeySize)
	if err != nil {
		t.Fatalf("Failed to generate new user key: %v", err)
	}

	// Re-encrypt secrets for new member
	err = teamSecretManager.ReencryptSecretsForNewMember(ctx, newUserKey)
	if err != nil {
		t.Fatalf("Failed to re-encrypt secrets for new member: %v", err)
	}

	// Verify secret can still be retrieved with original key
	retrievedPlaintext, err := teamSecretManager.GetTeamSecret(ctx, connectionID, secretType)
	if err != nil {
		t.Fatalf("Failed to get team secret after re-encryption: %v", err)
	}

	if string(retrievedPlaintext) != string(plaintext) {
		t.Errorf("Expected plaintext %s, got %s", string(plaintext), string(retrievedPlaintext))
	}
}

func TestTeamSecretManager_GetTeamSecretInfo(t *testing.T) {
	// Create mock secret store
	store := &mockSecretStore{}
	keyStore := NewKeyStore()
	secretManager := NewSecretManager(store, keyStore)

	// Unlock key store
	err := keyStore.Unlock("test-passphrase", []byte("test-salt"))
	if err != nil {
		t.Fatalf("Failed to unlock key store: %v", err)
	}

	// Create team secret manager
	teamID := "test-team-1"
	teamSecret := []byte("test-team-secret")
	teamSecretManager := NewTeamSecretManager(secretManager, keyStore, teamID, teamSecret)

	ctx := context.Background()

	// Store multiple secrets
	secrets := []struct {
		connectionID string
		secretType   SecretType
		plaintext    []byte
	}{
		{"conn-1", SecretTypeDBPassword, []byte("password-1")},
		{"conn-2", SecretTypeSSHPassword, []byte("ssh-password-2")},
		{"conn-3", SecretTypeSSHPrivateKey, []byte("private-key-3")},
	}

	for _, secret := range secrets {
		err := teamSecretManager.StoreTeamSecret(ctx, secret.connectionID, secret.secretType, secret.plaintext)
		if err != nil {
			t.Fatalf("Failed to store secret: %v", err)
		}
	}

	// Get team secret info
	info, err := teamSecretManager.GetTeamSecretInfo(ctx)
	if err != nil {
		t.Fatalf("Failed to get team secret info: %v", err)
	}

	// Verify info
	if info.TeamID != teamID {
		t.Errorf("Expected team ID %s, got %s", teamID, info.TeamID)
	}

	if info.TotalSecrets != len(secrets) {
		t.Errorf("Expected %d total secrets, got %d", len(secrets), info.TotalSecrets)
	}

	// Verify secret counts
	expectedCounts := map[SecretType]int{
		SecretTypeDBPassword:    1,
		SecretTypeSSHPassword:   1,
		SecretTypeSSHPrivateKey: 1,
	}

	for secretType, expectedCount := range expectedCounts {
		actualCount, exists := info.SecretCounts[secretType]
		if !exists {
			t.Errorf("Expected secret type %s not found in counts", secretType)
		}
		if actualCount != expectedCount {
			t.Errorf("Expected %d secrets of type %s, got %d", expectedCount, secretType, actualCount)
		}
	}
}

func TestTeamSecretManager_LockedKeyStore(t *testing.T) {
	// Create mock secret store
	store := &mockSecretStore{}
	keyStore := NewKeyStore() // Start locked
	secretManager := NewSecretManager(store, keyStore)

	// Create team secret manager
	teamID := "test-team-1"
	teamSecret := []byte("test-team-secret")
	teamSecretManager := NewTeamSecretManager(secretManager, keyStore, teamID, teamSecret)

	ctx := context.Background()
	connectionID := "test-connection-1"
	secretType := SecretTypeDBPassword
	plaintext := []byte("test-password")

	// Try to store team secret with locked key store
	err := teamSecretManager.StoreTeamSecret(ctx, connectionID, secretType, plaintext)
	if err == nil {
		t.Error("Expected error when key store is locked")
	}

	// Try to get team secret with locked key store
	_, err = teamSecretManager.GetTeamSecret(ctx, connectionID, secretType)
	if err == nil {
		t.Error("Expected error when key store is locked")
	}
}

// Mock secret store for testing
type mockSecretStore struct {
	secrets []*Secret
}

func (m *mockSecretStore) StoreSecret(ctx context.Context, secret *Secret) error {
	// Find existing secret and update, or add new one
	for i, existingSecret := range m.secrets {
		if existingSecret.ConnectionID == secret.ConnectionID && existingSecret.SecretType == secret.SecretType {
			m.secrets[i] = secret
			return nil
		}
	}
	m.secrets = append(m.secrets, secret)
	return nil
}

func (m *mockSecretStore) GetSecret(ctx context.Context, connectionID string, secretType SecretType) (*Secret, error) {
	for _, secret := range m.secrets {
		if secret.ConnectionID == connectionID && secret.SecretType == secretType {
			return secret, nil
		}
	}
	return nil, fmt.Errorf("secret not found")
}

func (m *mockSecretStore) DeleteSecret(ctx context.Context, connectionID string, secretType SecretType) error {
	for i, secret := range m.secrets {
		if secret.ConnectionID == connectionID && secret.SecretType == secretType {
			m.secrets = append(m.secrets[:i], m.secrets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("secret not found")
}

func (m *mockSecretStore) DeleteConnectionSecrets(ctx context.Context, connectionID string) error {
	var filtered []*Secret
	for _, secret := range m.secrets {
		if secret.ConnectionID != connectionID {
			filtered = append(filtered, secret)
		}
	}
	m.secrets = filtered
	return nil
}

func (m *mockSecretStore) ListSecrets(ctx context.Context, connectionID string) ([]SecretType, error) {
	var secretTypes []SecretType
	for _, secret := range m.secrets {
		if secret.ConnectionID == connectionID {
			secretTypes = append(secretTypes, secret.SecretType)
		}
	}
	return secretTypes, nil
}

func (m *mockSecretStore) GetSecretsByTeam(ctx context.Context, teamID string) ([]*Secret, error) {
	var teamSecrets []*Secret
	for _, secret := range m.secrets {
		if secret.TeamID == teamID {
			teamSecrets = append(teamSecrets, secret)
		}
	}
	return teamSecrets, nil
}
