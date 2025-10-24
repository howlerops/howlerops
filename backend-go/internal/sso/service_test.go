package sso

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSSOConfigStore is a mock implementation of SSOConfigStore
type MockSSOConfigStore struct {
	mock.Mock
}

func (m *MockSSOConfigStore) CreateConfig(config *SSOConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockSSOConfigStore) GetConfig(organizationID string) (*SSOConfig, error) {
	args := m.Called(organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SSOConfig), args.Error(1)
}

func (m *MockSSOConfigStore) UpdateConfig(config *SSOConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockSSOConfigStore) DeleteConfig(organizationID string) error {
	args := m.Called(organizationID)
	return args.Error(0)
}

func TestSSOService_ConfigureSSO(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockSSOConfigStore)
	service := NewService(mockStore, logger)

	t.Run("successful configuration", func(t *testing.T) {
		config := &SSOConfig{
			Provider:     "saml",
			ProviderName: "Okta",
			Metadata:     `{"idp_metadata_url": "https://example.com/metadata"}`,
			CreatedBy:    "user123",
		}

		mockStore.On("CreateConfig", mock.AnythingOfType("*sso.SSOConfig")).Return(nil).Once()

		err := service.ConfigureSSO(context.Background(), "org123", config)
		assert.NoError(t, err)
		assert.Equal(t, "org123", config.OrganizationID)
		mockStore.AssertExpectations(t)
	})

	t.Run("missing provider", func(t *testing.T) {
		config := &SSOConfig{
			ProviderName: "Okta",
			Metadata:     `{}`,
		}

		err := service.ConfigureSSO(context.Background(), "org123", config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "provider type is required")
	})

	t.Run("invalid JSON metadata", func(t *testing.T) {
		config := &SSOConfig{
			Provider:     "saml",
			ProviderName: "Okta",
			Metadata:     `{invalid json}`,
		}

		err := service.ConfigureSSO(context.Background(), "org123", config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid metadata JSON")
	})
}

func TestSSOService_InitiateLogin(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockSSOConfigStore)
	service := NewService(mockStore, logger)

	t.Run("successful login initiation", func(t *testing.T) {
		config := &SSOConfig{
			OrganizationID: "org123",
			Provider:       "oauth2",
			ProviderName:   "MockProvider",
			Enabled:        true,
		}

		mockStore.On("GetConfig", "org123").Return(config, nil).Once()

		loginURL, err := service.InitiateLogin(context.Background(), "org123")
		assert.NoError(t, err)
		assert.Contains(t, loginURL, "https://mock-sso.example.com")
		mockStore.AssertExpectations(t)
	})

	t.Run("SSO not enabled", func(t *testing.T) {
		config := &SSOConfig{
			OrganizationID: "org123",
			Provider:       "oauth2",
			ProviderName:   "MockProvider",
			Enabled:        false,
		}

		mockStore.On("GetConfig", "org123").Return(config, nil).Once()

		_, err := service.InitiateLogin(context.Background(), "org123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SSO is not enabled")
		mockStore.AssertExpectations(t)
	})
}

func TestSSOService_HandleCallback(t *testing.T) {
	logger := logrus.New()
	mockStore := new(MockSSOConfigStore)
	service := NewService(mockStore, logger)

	t.Run("successful callback", func(t *testing.T) {
		config := &SSOConfig{
			OrganizationID: "org123",
			Provider:       "oauth2",
			ProviderName:   "MockProvider",
			Enabled:        true,
		}

		mockStore.On("GetConfig", "org123").Return(config, nil).Once()

		user, err := service.HandleCallback(context.Background(), "org123", "test_code", "test_state")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user@example.com", user.Email)
		mockStore.AssertExpectations(t)
	})

	t.Run("invalid state", func(t *testing.T) {
		_, err := service.HandleCallback(context.Background(), "org123", "test_code", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid state token")
	})
}

func TestMockSSOProvider(t *testing.T) {
	provider := NewMockSSOProvider("TestProvider")

	t.Run("GetLoginURL", func(t *testing.T) {
		url, err := provider.GetLoginURL("test_state")
		assert.NoError(t, err)
		assert.Contains(t, url, "TestProvider")
		assert.Contains(t, url, "test_state")
	})

	t.Run("ExchangeCode - success", func(t *testing.T) {
		user, err := provider.ExchangeCode("valid_code")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "user@example.com", user.Email)
	})

	t.Run("ExchangeCode - admin user", func(t *testing.T) {
		user, err := provider.ExchangeCode("admin")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "admin@example.com", user.Email)
		assert.Equal(t, "admin", user.Attributes["role"])
	})

	t.Run("ExchangeCode - error", func(t *testing.T) {
		_, err := provider.ExchangeCode("error")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
	})

	t.Run("ValidateAssertion - success", func(t *testing.T) {
		user, err := provider.ValidateAssertion("valid_assertion")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "saml.user@example.com", user.Email)
	})

	t.Run("ValidateAssertion - empty", func(t *testing.T) {
		_, err := provider.ValidateAssertion("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty assertion")
	})

	t.Run("ValidateAssertion - invalid", func(t *testing.T) {
		_, err := provider.ValidateAssertion("invalid")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid assertion signature")
	})
}

func TestSSOProviderConfig_Serialization(t *testing.T) {
	config := &SSOProviderConfig{
		ClientID:     "client123",
		ClientSecret: "secret456",
		AuthURL:      "https://auth.example.com",
		TokenURL:     "https://token.example.com",
		RedirectURL:  "https://callback.example.com",
		Scopes:       []string{"openid", "profile", "email"},
	}

	// Test JSON serialization
	data, err := json.Marshal(config)
	assert.NoError(t, err)

	// Test JSON deserialization
	var decoded SSOProviderConfig
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, config.ClientID, decoded.ClientID)
	assert.Equal(t, config.Scopes, decoded.Scopes)
}