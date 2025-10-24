package sso

import (
	"fmt"
	"time"
)

// MockSSOProvider implements a mock SSO provider for testing
type MockSSOProvider struct {
	ProviderName string
}

// NewMockSSOProvider creates a new mock SSO provider
func NewMockSSOProvider(providerName string) *MockSSOProvider {
	return &MockSSOProvider{
		ProviderName: providerName,
	}
}

// GetLoginURL returns a mock login URL
func (m *MockSSOProvider) GetLoginURL(state string) (string, error) {
	return fmt.Sprintf("https://mock-sso.example.com/%s/login?state=%s", m.ProviderName, state), nil
}

// ExchangeCode exchanges a mock code for user information
func (m *MockSSOProvider) ExchangeCode(code string) (*SSOUser, error) {
	// Simulate different responses based on code
	switch code {
	case "error":
		return nil, fmt.Errorf("authentication failed")
	case "admin":
		return &SSOUser{
			Email:      "admin@example.com",
			FirstName:  "Admin",
			LastName:   "User",
			ExternalID: "mock-admin-123",
			Attributes: map[string]string{
				"role":       "admin",
				"department": "IT",
			},
		}, nil
	default:
		// Default user
		return &SSOUser{
			Email:      "user@example.com",
			FirstName:  "John",
			LastName:   "Doe",
			ExternalID: fmt.Sprintf("mock-id-%s-%d", code, time.Now().Unix()),
			Attributes: map[string]string{
				"role":       "user",
				"department": "Engineering",
			},
		}, nil
	}
}

// ValidateAssertion validates a mock SAML assertion
func (m *MockSSOProvider) ValidateAssertion(assertion string) (*SSOUser, error) {
	// Simulate validation
	if assertion == "" {
		return nil, fmt.Errorf("empty assertion")
	}

	if assertion == "invalid" {
		return nil, fmt.Errorf("invalid assertion signature")
	}

	// Return mock user based on assertion
	return &SSOUser{
		Email:      "saml.user@example.com",
		FirstName:  "SAML",
		LastName:   "User",
		ExternalID: fmt.Sprintf("saml-%s", assertion[:8]),
		Attributes: map[string]string{
			"auth_method": "saml",
			"provider":    m.ProviderName,
		},
	}, nil
}

// GetProviderName returns the provider name
func (m *MockSSOProvider) GetProviderName() string {
	return m.ProviderName
}

// MockOAuth2Provider implements OAuth2-specific mock provider
type MockOAuth2Provider struct {
	*MockSSOProvider
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// NewMockOAuth2Provider creates a new mock OAuth2 provider
func NewMockOAuth2Provider(name, clientID, clientSecret, redirectURL string) *MockOAuth2Provider {
	return &MockOAuth2Provider{
		MockSSOProvider: NewMockSSOProvider(name),
		ClientID:        clientID,
		ClientSecret:    clientSecret,
		RedirectURL:     redirectURL,
	}
}

// GetLoginURL returns OAuth2 authorization URL
func (m *MockOAuth2Provider) GetLoginURL(state string) (string, error) {
	return fmt.Sprintf(
		"https://mock-oauth2.example.com/authorize?client_id=%s&redirect_uri=%s&state=%s&response_type=code",
		m.ClientID, m.RedirectURL, state,
	), nil
}

// MockSAMLProvider implements SAML-specific mock provider
type MockSAMLProvider struct {
	*MockSSOProvider
	EntityID     string
	AssertionURL string
}

// NewMockSAMLProvider creates a new mock SAML provider
func NewMockSAMLProvider(name, entityID, assertionURL string) *MockSAMLProvider {
	return &MockSAMLProvider{
		MockSSOProvider: NewMockSSOProvider(name),
		EntityID:        entityID,
		AssertionURL:    assertionURL,
	}
}

// GetLoginURL returns SAML SSO URL
func (m *MockSAMLProvider) GetLoginURL(state string) (string, error) {
	return fmt.Sprintf(
		"https://mock-saml.example.com/sso?entity_id=%s&acs_url=%s&relay_state=%s",
		m.EntityID, m.AssertionURL, state,
	), nil
}