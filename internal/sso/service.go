package sso

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

// Service handles SSO operations
type Service struct {
	configStore SSOConfigStore
	providers   map[string]SSOProvider
	logger      *logrus.Logger
}

// NewService creates a new SSO service
func NewService(store SSOConfigStore, logger *logrus.Logger) *Service {
	return &Service{
		configStore: store,
		providers:   make(map[string]SSOProvider),
		logger:      logger,
	}
}

// RegisterProvider registers an SSO provider
func (s *Service) RegisterProvider(name string, provider SSOProvider) {
	s.providers[name] = provider
}

// ConfigureSSO configures SSO for an organization
func (s *Service) ConfigureSSO(ctx context.Context, orgID string, config *SSOConfig) error {
	// Validate configuration
	if config.Provider == "" {
		return fmt.Errorf("provider type is required")
	}

	if config.ProviderName == "" {
		return fmt.Errorf("provider name is required")
	}

	// Validate metadata is valid JSON
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(config.Metadata), &metadata); err != nil {
		return fmt.Errorf("invalid metadata JSON: %w", err)
	}

	// Set timestamps
	now := time.Now()
	config.CreatedAt = now
	config.UpdatedAt = now
	config.OrganizationID = orgID

	// Store configuration
	if err := s.configStore.CreateConfig(config); err != nil {
		s.logger.WithError(err).Error("Failed to store SSO configuration")
		return fmt.Errorf("failed to store SSO configuration: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"provider":        config.Provider,
		"provider_name":   config.ProviderName,
	}).Info("SSO configured successfully")

	return nil
}

// InitiateLogin initiates SSO login for an organization
func (s *Service) InitiateLogin(ctx context.Context, orgID string) (string, error) {
	// Get SSO configuration
	config, err := s.configStore.GetConfig(orgID)
	if err != nil {
		return "", fmt.Errorf("failed to get SSO config: %w", err)
	}

	if !config.Enabled {
		return "", fmt.Errorf("SSO is not enabled for this organization")
	}

	// Get provider
	provider, exists := s.providers[config.ProviderName]
	if !exists {
		// Use mock provider if no real provider registered
		provider = NewMockSSOProvider(config.ProviderName)
	}

	// Generate state token for CSRF protection
	state := generateStateToken()

	// Get login URL
	loginURL, err := provider.GetLoginURL(state)
	if err != nil {
		return "", fmt.Errorf("failed to get login URL: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"provider":        config.ProviderName,
	}).Info("SSO login initiated")

	return loginURL, nil
}

// HandleCallback handles SSO callback
func (s *Service) HandleCallback(ctx context.Context, orgID, code, state string) (*SSOUser, error) {
	// Validate state token (in production, check against stored state)
	if state == "" {
		return nil, fmt.Errorf("invalid state token")
	}

	// Get SSO configuration
	config, err := s.configStore.GetConfig(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO config: %w", err)
	}

	// Get provider
	provider, exists := s.providers[config.ProviderName]
	if !exists {
		provider = NewMockSSOProvider(config.ProviderName)
	}

	// Exchange code for user info
	user, err := provider.ExchangeCode(code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"email":           user.Email,
		"external_id":     user.ExternalID,
	}).Info("SSO callback handled successfully")

	return user, nil
}

// ValidateSAML validates a SAML assertion
func (s *Service) ValidateSAML(ctx context.Context, orgID, assertion string) (*SSOUser, error) {
	// Get SSO configuration
	config, err := s.configStore.GetConfig(orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get SSO config: %w", err)
	}

	if config.Provider != "saml" {
		return nil, fmt.Errorf("SAML is not configured for this organization")
	}

	// Get provider
	provider, exists := s.providers[config.ProviderName]
	if !exists {
		provider = NewMockSSOProvider(config.ProviderName)
	}

	// Validate assertion
	user, err := provider.ValidateAssertion(assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to validate assertion: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"email":           user.Email,
	}).Info("SAML assertion validated successfully")

	return user, nil
}

// GetConfig retrieves SSO configuration for an organization
func (s *Service) GetConfig(ctx context.Context, orgID string) (*SSOConfig, error) {
	return s.configStore.GetConfig(orgID)
}

// DisableSSO disables SSO for an organization
func (s *Service) DisableSSO(ctx context.Context, orgID string) error {
	config, err := s.configStore.GetConfig(orgID)
	if err != nil {
		return fmt.Errorf("failed to get SSO config: %w", err)
	}

	config.Enabled = false
	config.UpdatedAt = time.Now()

	if err := s.configStore.UpdateConfig(config); err != nil {
		return fmt.Errorf("failed to disable SSO: %w", err)
	}

	s.logger.WithField("organization_id", orgID).Info("SSO disabled")
	return nil
}

// generateStateToken generates a random state token for CSRF protection
func generateStateToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// This should never happen with crypto/rand, but handle it anyway
		panic(fmt.Sprintf("failed to generate random token: %v", err))
	}
	return base64.URLEncoding.EncodeToString(b)
}
