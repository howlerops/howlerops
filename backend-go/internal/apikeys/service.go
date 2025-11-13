package apikeys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// APIKey represents an API key
type APIKey struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	OrganizationID string     `json:"organization_id,omitempty"`
	Name           string     `json:"name"`
	KeyHash        string     `json:"-"` // Never expose the hash
	KeyPrefix      string     `json:"key_prefix"`
	Permissions    []string   `json:"permissions"`
	LastUsedAt     *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

// CreateAPIKeyInput represents the input for creating an API key
type CreateAPIKeyInput struct {
	Name           string   `json:"name" validate:"required,min=1,max=100"`
	OrganizationID string   `json:"organization_id,omitempty"`
	Permissions    []string `json:"permissions" validate:"required,min=1"`
	ExpiresInDays  int      `json:"expires_in_days" validate:"min=0,max=365"` // 0 = never expires
}

// APIKeyResponse represents the response when creating an API key
type APIKeyResponse struct {
	Key    string  `json:"key"`     // Full key (shown only once)
	Prefix string  `json:"prefix"`  // Key prefix for identification
	APIKey *APIKey `json:"api_key"` // Key metadata
}

// APIKeyStore defines the interface for API key storage
type APIKeyStore interface {
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKey(ctx context.Context, id string) (*APIKey, error)
	GetByPrefix(ctx context.Context, prefix string) ([]*APIKey, error)
	GetUserAPIKeys(ctx context.Context, userID string) ([]*APIKey, error)
	GetOrgAPIKeys(ctx context.Context, orgID string) ([]*APIKey, error)
	UpdateLastUsed(ctx context.Context, id string) error
	RevokeAPIKey(ctx context.Context, id string) error
	DeleteExpiredKeys(ctx context.Context) error
}

// SecurityEventLogger logs security events
type SecurityEventLogger interface {
	LogSecurityEvent(ctx context.Context, eventType, userID, orgID, ipAddress, userAgent string, details map[string]interface{}) error
}

// Service handles API key operations
type Service struct {
	store       APIKeyStore
	eventLogger SecurityEventLogger
	logger      *logrus.Logger
}

// NewService creates a new API key service
func NewService(store APIKeyStore, eventLogger SecurityEventLogger, logger *logrus.Logger) *Service {
	return &Service{
		store:       store,
		eventLogger: eventLogger,
		logger:      logger,
	}
}

// CreateAPIKey creates a new API key
func (s *Service) CreateAPIKey(ctx context.Context, userID string, input *CreateAPIKeyInput) (*APIKeyResponse, error) {
	// Validate input
	if input.Name == "" {
		return nil, fmt.Errorf("API key name is required")
	}

	if len(input.Permissions) == 0 {
		return nil, fmt.Errorf("at least one permission is required")
	}

	// Generate API key
	key := s.generateAPIKey()
	keyHash := s.hashAPIKey(key)
	keyPrefix := s.extractPrefix(key)

	// Calculate expiration
	var expiresAt *time.Time
	if input.ExpiresInDays > 0 {
		exp := time.Now().AddDate(0, 0, input.ExpiresInDays)
		expiresAt = &exp
	}

	// Create API key record
	apiKey := &APIKey{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		KeyHash:        keyHash,
		KeyPrefix:      keyPrefix,
		Permissions:    input.Permissions,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
	}

	// Store in database
	if err := s.store.CreateAPIKey(ctx, apiKey); err != nil {
		s.logger.WithError(err).Error("Failed to create API key")
		return nil, fmt.Errorf("failed to create API key")
	}

	// Log security event
	if s.eventLogger != nil {
		_ = s.eventLogger.LogSecurityEvent(
			ctx,
			"api_key_created",
			userID,
			input.OrganizationID,
			"",
			"",
			map[string]interface{}{
				"key_id":      apiKey.ID,
				"key_prefix":  keyPrefix,
				"permissions": input.Permissions,
				"expires_in":  input.ExpiresInDays,
			},
		)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":    userID,
		"key_id":     apiKey.ID,
		"key_prefix": keyPrefix,
	}).Info("API key created")

	// Return full key only once
	return &APIKeyResponse{
		Key:    key,
		Prefix: keyPrefix,
		APIKey: apiKey,
	}, nil
}

// ValidateAPIKey validates an API key and returns its metadata
func (s *Service) ValidateAPIKey(ctx context.Context, key string) (*APIKey, error) {
	// Validate key format
	if !strings.HasPrefix(key, "sk_") || len(key) < 40 {
		return nil, fmt.Errorf("invalid API key format")
	}

	// Extract prefix for lookup
	prefix := s.extractPrefix(key)

	// Lookup keys with matching prefix
	apiKeys, err := s.store.GetByPrefix(ctx, prefix)
	if err != nil {
		s.logger.WithError(err).Error("Failed to lookup API key")
		return nil, fmt.Errorf("invalid API key")
	}

	// Find matching key by comparing hashes
	for _, apiKey := range apiKeys {
		if s.verifyAPIKey(key, apiKey.KeyHash) {
			// Check if key is revoked
			if apiKey.RevokedAt != nil {
				s.logInvalidKeyUsage(ctx, "revoked", apiKey)
				return nil, fmt.Errorf("API key has been revoked")
			}

			// Check if key is expired
			if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
				s.logInvalidKeyUsage(ctx, "expired", apiKey)
				return nil, fmt.Errorf("API key has expired")
			}

			// Update last used timestamp
			if err := s.store.UpdateLastUsed(ctx, apiKey.ID); err != nil {
				s.logger.WithError(err).Warn("Failed to update last used timestamp")
			}

			return apiKey, nil
		}
	}

	// No matching key found
	s.logInvalidKeyUsage(ctx, "invalid", nil)
	return nil, fmt.Errorf("invalid API key")
}

// RevokeAPIKey revokes an API key
func (s *Service) RevokeAPIKey(ctx context.Context, keyID, userID string) error {
	// Get API key to verify ownership
	apiKey, err := s.store.GetAPIKey(ctx, keyID)
	if err != nil {
		return fmt.Errorf("API key not found")
	}

	// Verify ownership
	if apiKey.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Check if already revoked
	if apiKey.RevokedAt != nil {
		return fmt.Errorf("API key already revoked")
	}

	// Revoke the key
	if err := s.store.RevokeAPIKey(ctx, keyID); err != nil {
		return fmt.Errorf("failed to revoke API key")
	}

	// Log security event
	if s.eventLogger != nil {
		_ = s.eventLogger.LogSecurityEvent(
			ctx,
			"api_key_revoked",
			userID,
			apiKey.OrganizationID,
			"",
			"",
			map[string]interface{}{
				"key_id":     keyID,
				"key_prefix": apiKey.KeyPrefix,
			},
		)
	}

	s.logger.WithFields(logrus.Fields{
		"key_id":     keyID,
		"user_id":    userID,
		"key_prefix": apiKey.KeyPrefix,
	}).Info("API key revoked")

	return nil
}

// ListUserAPIKeys lists all API keys for a user
func (s *Service) ListUserAPIKeys(ctx context.Context, userID string) ([]*APIKey, error) {
	keys, err := s.store.GetUserAPIKeys(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys")
	}

	// Filter out sensitive information
	for _, key := range keys {
		key.KeyHash = "" // Never expose the hash
	}

	return keys, nil
}

// ListOrgAPIKeys lists all API keys for an organization
func (s *Service) ListOrgAPIKeys(ctx context.Context, orgID string) ([]*APIKey, error) {
	keys, err := s.store.GetOrgAPIKeys(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization API keys")
	}

	// Filter out sensitive information
	for _, key := range keys {
		key.KeyHash = "" // Never expose the hash
	}

	return keys, nil
}

// GetAPIKey retrieves API key metadata
func (s *Service) GetAPIKey(ctx context.Context, keyID, userID string) (*APIKey, error) {
	apiKey, err := s.store.GetAPIKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("API key not found")
	}

	// Verify ownership
	if apiKey.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}

	// Remove sensitive information
	apiKey.KeyHash = ""

	return apiKey, nil
}

// CleanupExpiredKeys removes expired API keys
func (s *Service) CleanupExpiredKeys(ctx context.Context) error {
	if err := s.store.DeleteExpiredKeys(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to cleanup expired API keys")
		return err
	}

	s.logger.Info("Expired API keys cleaned up")
	return nil
}

// generateAPIKey generates a new API key
func (s *Service) generateAPIKey() string {
	// Format: sk_live_xxxxxxxxxxxxxxxxxxxxx (32 random chars)
	// or sk_test_xxxxxxxxxxxxxxxxxxxxx for test environment
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		// This should never fail, but if it does, use a fallback
		panic(fmt.Sprintf("crypto/rand.Read failed: %v", err))
	}
	randomPart := hex.EncodeToString(randomBytes)[:32]

	// Determine environment prefix
	prefix := "sk_live_"
	// In test/dev environment, use sk_test_
	// This can be configured based on environment

	return prefix + randomPart
}

// extractPrefix extracts the prefix from an API key for identification
func (s *Service) extractPrefix(key string) string {
	// Extract first 15 characters as prefix
	// Example: "sk_live_abc1234" from "sk_live_abc1234567890..."
	if len(key) < 15 {
		return key
	}
	return key[:15]
}

// hashAPIKey creates a bcrypt hash of the API key
func (s *Service) hashAPIKey(key string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
	if err != nil {
		// This should never happen with valid input
		s.logger.WithError(err).Error("Failed to hash API key")
		return ""
	}
	return string(hash)
}

// verifyAPIKey verifies an API key against a hash
func (s *Service) verifyAPIKey(key, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(key))
	return err == nil
}

// logInvalidKeyUsage logs invalid API key usage attempts
func (s *Service) logInvalidKeyUsage(ctx context.Context, reason string, apiKey *APIKey) {
	if s.eventLogger == nil {
		return
	}

	details := map[string]interface{}{
		"reason": reason,
	}

	if apiKey != nil {
		details["key_id"] = apiKey.ID
		details["key_prefix"] = apiKey.KeyPrefix
		details["user_id"] = apiKey.UserID
	}

	_ = s.eventLogger.LogSecurityEvent(
		ctx,
		"api_key_invalid_usage",
		"",
		"",
		"",
		"",
		details,
	)
}

// ValidatePermission checks if an API key has a specific permission
func (s *Service) ValidatePermission(apiKey *APIKey, permission string) bool {
	for _, p := range apiKey.Permissions {
		if p == permission || p == "*" { // Wildcard permission
			return true
		}
		// Check for pattern matching (e.g., "connections:*" matches "connections:read")
		if strings.HasSuffix(p, ":*") {
			prefix := strings.TrimSuffix(p, "*")
			if strings.HasPrefix(permission, prefix) {
				return true
			}
		}
	}
	return false
}
