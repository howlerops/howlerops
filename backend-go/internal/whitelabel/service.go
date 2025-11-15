package whitelabel

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// Service handles white-labeling business logic
type Service struct {
	store  *Store
	logger *logrus.Logger
}

// NewService creates a new white-label service
func NewService(store *Store, logger *logrus.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

// GetConfigByOrganization retrieves white-label config for an organization
func (s *Service) GetConfigByOrganization(ctx context.Context, orgID string) (*WhiteLabelConfig, error) {
	config, err := s.store.GetByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get config: %w", err)
	}

	if config == nil {
		// Return default config
		return s.getDefaultConfig(orgID), nil
	}

	return config, nil
}

// GetConfigByDomain retrieves white-label config by custom domain
func (s *Service) GetConfigByDomain(ctx context.Context, domain string) (*WhiteLabelConfig, error) {
	// Normalize domain
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")

	config, err := s.store.GetByDomain(ctx, domain)
	if err != nil {
		return nil, fmt.Errorf("get config by domain: %w", err)
	}

	if config == nil {
		// Return default config
		return s.getDefaultConfig(""), nil
	}

	return config, nil
}

// UpdateConfig updates or creates white-label configuration
func (s *Service) UpdateConfig(ctx context.Context, orgID string, req *UpdateWhiteLabelRequest) (*WhiteLabelConfig, error) {
	// Get existing config or create new
	config, err := s.store.GetByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get existing config: %w", err)
	}

	isNew := config == nil
	if isNew {
		config = s.getDefaultConfig(orgID)
	}

	// Apply updates
	if req.CustomDomain != nil {
		if *req.CustomDomain != "" {
			if err := s.validateDomain(*req.CustomDomain); err != nil {
				return nil, fmt.Errorf("invalid custom domain: %w", err)
			}
		}
		config.CustomDomain = *req.CustomDomain
	}

	if req.LogoURL != nil {
		if *req.LogoURL != "" {
			if err := s.validateURL(*req.LogoURL); err != nil {
				return nil, fmt.Errorf("invalid logo URL: %w", err)
			}
		}
		config.LogoURL = *req.LogoURL
	}

	if req.FaviconURL != nil {
		if *req.FaviconURL != "" {
			if err := s.validateURL(*req.FaviconURL); err != nil {
				return nil, fmt.Errorf("invalid favicon URL: %w", err)
			}
		}
		config.FaviconURL = *req.FaviconURL
	}

	if req.PrimaryColor != nil {
		if *req.PrimaryColor != "" {
			if err := s.validateHexColor(*req.PrimaryColor); err != nil {
				return nil, fmt.Errorf("invalid primary color: %w", err)
			}
		}
		config.PrimaryColor = *req.PrimaryColor
	}

	if req.SecondaryColor != nil {
		if *req.SecondaryColor != "" {
			if err := s.validateHexColor(*req.SecondaryColor); err != nil {
				return nil, fmt.Errorf("invalid secondary color: %w", err)
			}
		}
		config.SecondaryColor = *req.SecondaryColor
	}

	if req.AccentColor != nil {
		if *req.AccentColor != "" {
			if err := s.validateHexColor(*req.AccentColor); err != nil {
				return nil, fmt.Errorf("invalid accent color: %w", err)
			}
		}
		config.AccentColor = *req.AccentColor
	}

	if req.CompanyName != nil {
		config.CompanyName = *req.CompanyName
	}

	if req.SupportEmail != nil {
		if *req.SupportEmail != "" {
			if err := s.validateEmail(*req.SupportEmail); err != nil {
				return nil, fmt.Errorf("invalid support email: %w", err)
			}
		}
		config.SupportEmail = *req.SupportEmail
	}

	if req.CustomCSS != nil {
		config.CustomCSS = *req.CustomCSS
	}

	if req.HideBranding != nil {
		config.HideBranding = *req.HideBranding
	}

	// Save to database
	if isNew {
		err = s.store.Create(ctx, config)
	} else {
		err = s.store.Update(ctx, config)
	}

	if err != nil {
		return nil, fmt.Errorf("save config: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"custom_domain":   config.CustomDomain,
		"is_new":          isNew,
	}).Info("White-label config updated")

	return config, nil
}

// GenerateBrandedCSS generates CSS with custom branding
func (s *Service) GenerateBrandedCSS(config *WhiteLabelConfig) string {
	primaryColor := config.PrimaryColor
	if primaryColor == "" {
		primaryColor = "#1E40AF" // Default blue
	}

	secondaryColor := config.SecondaryColor
	if secondaryColor == "" {
		secondaryColor = "#64748B" // Default gray
	}

	accentColor := config.AccentColor
	if accentColor == "" {
		accentColor = "#8B5CF6" // Default purple
	}

	css := fmt.Sprintf(`/* Auto-generated white-label CSS */
:root {
	--brand-primary: %s;
	--brand-secondary: %s;
	--brand-accent: %s;
}

/* Primary color applications */
.btn-primary,
.bg-primary {
	background-color: var(--brand-primary) !important;
}

.text-primary {
	color: var(--brand-primary) !important;
}

.border-primary {
	border-color: var(--brand-primary) !important;
}

/* Secondary color applications */
.btn-secondary,
.bg-secondary {
	background-color: var(--brand-secondary) !important;
}

.text-secondary {
	color: var(--brand-secondary) !important;
}

/* Accent color applications */
.btn-accent,
.bg-accent {
	background-color: var(--brand-accent) !important;
}

.text-accent {
	color: var(--brand-accent) !important;
}

/* Logo replacement */
.app-logo {
	background-image: url('%s');
	background-size: contain;
	background-repeat: no-repeat;
	background-position: center;
}

/* Hide default branding if configured */
.powered-by-sql-studio {
	display: %s !important;
}

/* Custom CSS overrides */
%s
`,
		primaryColor,
		secondaryColor,
		accentColor,
		config.LogoURL,
		func() string {
			if config.HideBranding {
				return "none"
			}
			return "block"
		}(),
		config.CustomCSS,
	)

	return css
}

// getDefaultConfig returns default white-label config
func (s *Service) getDefaultConfig(orgID string) *WhiteLabelConfig {
	return &WhiteLabelConfig{
		OrganizationID: orgID,
		PrimaryColor:   "#1E40AF",
		SecondaryColor: "#64748B",
		AccentColor:    "#8B5CF6",
		CompanyName:    "Howlerops",
		HideBranding:   false,
	}
}

// Validation functions

func (s *Service) validateHexColor(color string) error {
	matched, err := regexp.MatchString(`^#([A-Fa-f0-9]{6}|[A-Fa-f0-9]{3})$`, color)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("invalid hex color format (expected #RRGGBB or #RGB)")
	}
	return nil
}

func (s *Service) validateDomain(domain string) error {
	// Basic domain validation
	domainRegex := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("invalid domain format")
	}

	// Prevent common invalid domains
	invalidDomains := []string{"localhost", "example.com", "test.com"}
	for _, invalid := range invalidDomains {
		if strings.Contains(strings.ToLower(domain), invalid) {
			return fmt.Errorf("domain not allowed: %s", invalid)
		}
	}

	return nil
}

func (s *Service) validateURL(url string) error {
	// Basic URL validation
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}
	return nil
}

func (s *Service) validateEmail(email string) error {
	_, err := mail.ParseAddress(email)
	if err != nil {
		return fmt.Errorf("invalid email address")
	}
	return nil
}
