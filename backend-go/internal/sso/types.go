package sso

import (
	"time"
)

// SSOProvider defines the interface for SSO providers
type SSOProvider interface {
	// GetLoginURL returns the URL to redirect user for SSO login
	GetLoginURL(state string) (string, error)

	// ExchangeCode exchanges authorization code for user info
	ExchangeCode(code string) (*SSOUser, error)

	// ValidateAssertion validates SAML assertion or JWT token
	ValidateAssertion(assertion string) (*SSOUser, error)

	// GetProviderName returns the provider name
	GetProviderName() string
}

// SSOUser represents user information from SSO provider
type SSOUser struct {
	Email      string            `json:"email"`
	FirstName  string            `json:"first_name"`
	LastName   string            `json:"last_name"`
	ExternalID string            `json:"external_id"`
	Attributes map[string]string `json:"attributes"`
}

// SSOConfig represents SSO configuration for an organization
type SSOConfig struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	Provider       string    `json:"provider"`      // 'saml', 'oauth2', 'oidc'
	ProviderName   string    `json:"provider_name"` // 'Okta', 'Auth0', 'Azure AD'
	Metadata       string    `json:"metadata"`      // JSON configuration
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	CreatedBy      string    `json:"created_by"`
}

// SSOSession represents an SSO session
type SSOSession struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	OrganizationID string    `json:"organization_id"`
	Provider       string    `json:"provider"`
	ExternalID     string    `json:"external_id"`
	SessionData    string    `json:"session_data"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// SSOConfigStore defines the interface for SSO configuration storage
type SSOConfigStore interface {
	CreateConfig(config *SSOConfig) error
	GetConfig(organizationID string) (*SSOConfig, error)
	UpdateConfig(config *SSOConfig) error
	DeleteConfig(organizationID string) error
}

// SSOProviderConfig contains provider-specific configuration
type SSOProviderConfig struct {
	// OAuth2/OIDC
	ClientID     string   `json:"client_id,omitempty"`
	ClientSecret string   `json:"client_secret,omitempty"`
	AuthURL      string   `json:"auth_url,omitempty"`
	TokenURL     string   `json:"token_url,omitempty"`
	RedirectURL  string   `json:"redirect_url,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`

	// SAML
	IDPMetadataURL string `json:"idp_metadata_url,omitempty"`
	IDPMetadata    string `json:"idp_metadata,omitempty"`
	SPEntityID     string `json:"sp_entity_id,omitempty"`
	SPAssertionURL string `json:"sp_assertion_url,omitempty"`
	Certificate    string `json:"certificate,omitempty"`
	PrivateKey     string `json:"private_key,omitempty"`
}
