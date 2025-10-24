package whitelabel

import "time"

// WhiteLabelConfig represents the white-labeling configuration for an organization
type WhiteLabelConfig struct {
	OrganizationID string    `json:"organization_id" db:"organization_id"`
	CustomDomain   string    `json:"custom_domain,omitempty" db:"custom_domain"`
	LogoURL        string    `json:"logo_url,omitempty" db:"logo_url"`
	FaviconURL     string    `json:"favicon_url,omitempty" db:"favicon_url"`
	PrimaryColor   string    `json:"primary_color,omitempty" db:"primary_color"`
	SecondaryColor string    `json:"secondary_color,omitempty" db:"secondary_color"`
	AccentColor    string    `json:"accent_color,omitempty" db:"accent_color"`
	CompanyName    string    `json:"company_name,omitempty" db:"company_name"`
	SupportEmail   string    `json:"support_email,omitempty" db:"support_email"`
	CustomCSS      string    `json:"custom_css,omitempty" db:"custom_css"`
	HideBranding   bool      `json:"hide_branding" db:"hide_branding"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// UpdateWhiteLabelRequest represents a request to update white-label config
type UpdateWhiteLabelRequest struct {
	CustomDomain   *string `json:"custom_domain,omitempty"`
	LogoURL        *string `json:"logo_url,omitempty"`
	FaviconURL     *string `json:"favicon_url,omitempty"`
	PrimaryColor   *string `json:"primary_color,omitempty"`
	SecondaryColor *string `json:"secondary_color,omitempty"`
	AccentColor    *string `json:"accent_color,omitempty"`
	CompanyName    *string `json:"company_name,omitempty"`
	SupportEmail   *string `json:"support_email,omitempty"`
	CustomCSS      *string `json:"custom_css,omitempty"`
	HideBranding   *bool   `json:"hide_branding,omitempty"`
}

// BrandedCSSResponse contains the generated CSS for white-labeling
type BrandedCSSResponse struct {
	CSS            string           `json:"css"`
	Config         WhiteLabelConfig `json:"config"`
	DefaultBranding bool            `json:"default_branding"`
}
