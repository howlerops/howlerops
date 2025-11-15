package auth

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/zalando/go-keyring"
)

const (
	// Key prefix for WebAuthn credentials
	webauthnCredentialKeyPrefix = "webauthn_credential_"
)

// CredentialStore handles secure storage of WebAuthn credentials
type CredentialStore struct {
	mu sync.RWMutex
	// In-memory cache for performance (optional, but recommended)
	cache map[string][]webauthn.Credential
}

// NewCredentialStore creates a new credential store
func NewCredentialStore() *CredentialStore {
	return &CredentialStore{
		cache: make(map[string][]webauthn.Credential),
	}
}

// StoreCredential stores a WebAuthn credential securely
func (cs *CredentialStore) StoreCredential(userID string, credential *webauthn.Credential) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Get existing credentials
	credentials, err := cs.getCredentialsUnsafe(userID)
	if err != nil && err.Error() != "no credentials found" {
		return err
	}

	// Add new credential (or replace if it already exists with same ID)
	found := false
	for i, cred := range credentials {
		if string(cred.ID) == string(credential.ID) {
			credentials[i] = *credential
			found = true
			break
		}
	}
	if !found {
		credentials = append(credentials, *credential)
	}

	// Serialize credentials
	data, err := json.Marshal(credentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	// Store in keyring
	key := webauthnCredentialKeyPrefix + userID
	if err := keyring.Set(serviceName, key, string(data)); err != nil {
		return fmt.Errorf("failed to store credential in keyring: %w", err)
	}

	// Update cache
	cs.cache[userID] = credentials

	return nil
}

// UpdateCredential updates an existing credential (e.g., sign count)
func (cs *CredentialStore) UpdateCredential(userID string, credential *webauthn.Credential) error {
	return cs.StoreCredential(userID, credential)
}

// GetCredentials retrieves all credentials for a user
func (cs *CredentialStore) GetCredentials(userID string) ([]webauthn.Credential, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	return cs.getCredentialsUnsafe(userID)
}

// getCredentialsUnsafe retrieves credentials without locking (internal use)
func (cs *CredentialStore) getCredentialsUnsafe(userID string) ([]webauthn.Credential, error) {
	// Check cache first
	if credentials, ok := cs.cache[userID]; ok {
		return credentials, nil
	}

	// Retrieve from keyring
	key := webauthnCredentialKeyPrefix + userID
	data, err := keyring.Get(serviceName, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, fmt.Errorf("no credentials found")
		}
		return nil, fmt.Errorf("failed to retrieve credential from keyring: %w", err)
	}

	// Deserialize credentials
	var credentials []webauthn.Credential
	if err := json.Unmarshal([]byte(data), &credentials); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	// Update cache
	cs.cache[userID] = credentials

	return credentials, nil
}

// GetCredential retrieves a specific credential by ID
func (cs *CredentialStore) GetCredential(userID string, credentialID []byte) (*webauthn.Credential, error) {
	credentials, err := cs.GetCredentials(userID)
	if err != nil {
		return nil, err
	}

	for _, cred := range credentials {
		if string(cred.ID) == string(credentialID) {
			return &cred, nil
		}
	}

	return nil, fmt.Errorf("credential not found")
}

// DeleteCredential removes a specific credential
func (cs *CredentialStore) DeleteCredential(userID string, credentialID []byte) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// Get existing credentials
	credentials, err := cs.getCredentialsUnsafe(userID)
	if err != nil {
		return err
	}

	// Remove the credential
	newCredentials := make([]webauthn.Credential, 0, len(credentials))
	for _, cred := range credentials {
		if string(cred.ID) != string(credentialID) {
			newCredentials = append(newCredentials, cred)
		}
	}

	// If no credentials left, delete the key
	if len(newCredentials) == 0 {
		key := webauthnCredentialKeyPrefix + userID
		if err := keyring.Delete(serviceName, key); err != nil {
			return fmt.Errorf("failed to delete credential from keyring: %w", err)
		}
		delete(cs.cache, userID)
		return nil
	}

	// Otherwise, update the stored credentials
	data, err := json.Marshal(newCredentials)
	if err != nil {
		return fmt.Errorf("failed to marshal credentials: %w", err)
	}

	key := webauthnCredentialKeyPrefix + userID
	if err := keyring.Set(serviceName, key, string(data)); err != nil {
		return fmt.Errorf("failed to update credentials in keyring: %w", err)
	}

	// Update cache
	cs.cache[userID] = newCredentials

	return nil
}

// DeleteAllCredentials removes all credentials for a user
func (cs *CredentialStore) DeleteAllCredentials(userID string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	key := webauthnCredentialKeyPrefix + userID
	if err := keyring.Delete(serviceName, key); err != nil {
		if err == keyring.ErrNotFound {
			// Already deleted or never existed
			return nil
		}
		return fmt.Errorf("failed to delete credentials from keyring: %w", err)
	}

	// Clear cache
	delete(cs.cache, userID)

	return nil
}

// HasCredentials checks if a user has any credentials
func (cs *CredentialStore) HasCredentials(userID string) bool {
	credentials, err := cs.GetCredentials(userID)
	return err == nil && len(credentials) > 0
}
