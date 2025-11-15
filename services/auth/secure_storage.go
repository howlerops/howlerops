package auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	serviceName = "howlerops" // Shared service name for keyring operations
)

// SecureStorage manages secure token storage using OS keyring
type SecureStorage struct{}

// NewSecureStorage creates a new SecureStorage instance
func NewSecureStorage() *SecureStorage {
	return &SecureStorage{}
}

// StoredToken represents a token stored in the keyring
type StoredToken struct {
	AccessToken string    `json:"access_token"`
	Provider    string    `json:"provider"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
}

// StoreToken stores an access token in the OS keyring
func (ss *SecureStorage) StoreToken(provider string, token *StoredToken) error {
	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if token == nil {
		return fmt.Errorf("token cannot be nil")
	}

	// Serialize token to JSON
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Store in keyring with key format: "{provider}_token"
	key := fmt.Sprintf("%s_token", provider)
	if err := keyring.Set(serviceName, key, string(data)); err != nil {
		return fmt.Errorf("failed to store token in keyring: %w", err)
	}

	return nil
}

// RetrieveToken retrieves an access token from the OS keyring
func (ss *SecureStorage) RetrieveToken(provider string) (*StoredToken, error) {
	if provider == "" {
		return nil, fmt.Errorf("provider cannot be empty")
	}

	// Retrieve from keyring
	key := fmt.Sprintf("%s_token", provider)
	data, err := keyring.Get(serviceName, key)
	if err != nil {
		if err == keyring.ErrNotFound {
			return nil, nil // No token stored, not an error
		}
		return nil, fmt.Errorf("failed to retrieve token from keyring: %w", err)
	}

	// Deserialize JSON
	var token StoredToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	// Check if token is expired
	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		// Token expired, delete it
		_ = ss.DeleteToken(provider)
		return nil, nil
	}

	return &token, nil
}

// DeleteToken removes a token from the OS keyring
func (ss *SecureStorage) DeleteToken(provider string) error {
	if provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}

	key := fmt.Sprintf("%s_token", provider)
	if err := keyring.Delete(serviceName, key); err != nil {
		if err == keyring.ErrNotFound {
			return nil // Already deleted, not an error
		}
		return fmt.Errorf("failed to delete token from keyring: %w", err)
	}

	return nil
}

// CheckTokenExists checks if a token exists for a provider without retrieving it
func (ss *SecureStorage) CheckTokenExists(provider string) (bool, error) {
	token, err := ss.RetrieveToken(provider)
	if err != nil {
		return false, err
	}
	return token != nil, nil
}
