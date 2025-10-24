package whitelabel

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store handles white-label configuration persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new white-label store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetByOrganization retrieves white-label config by organization ID
func (s *Store) GetByOrganization(ctx context.Context, orgID string) (*WhiteLabelConfig, error) {
	query := `
		SELECT organization_id, custom_domain, logo_url, favicon_url,
		       primary_color, secondary_color, accent_color, company_name,
		       support_email, custom_css, hide_branding, created_at, updated_at
		FROM white_label_config
		WHERE organization_id = ?
	`

	var config WhiteLabelConfig
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, orgID).Scan(
		&config.OrganizationID,
		&config.CustomDomain,
		&config.LogoURL,
		&config.FaviconURL,
		&config.PrimaryColor,
		&config.SecondaryColor,
		&config.AccentColor,
		&config.CompanyName,
		&config.SupportEmail,
		&config.CustomCSS,
		&config.HideBranding,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No config found
	}
	if err != nil {
		return nil, fmt.Errorf("query white-label config: %w", err)
	}

	config.CreatedAt = time.Unix(createdAt, 0)
	config.UpdatedAt = time.Unix(updatedAt, 0)

	return &config, nil
}

// GetByDomain retrieves white-label config by custom domain
func (s *Store) GetByDomain(ctx context.Context, domain string) (*WhiteLabelConfig, error) {
	query := `
		SELECT organization_id, custom_domain, logo_url, favicon_url,
		       primary_color, secondary_color, accent_color, company_name,
		       support_email, custom_css, hide_branding, created_at, updated_at
		FROM white_label_config
		WHERE custom_domain = ?
	`

	var config WhiteLabelConfig
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, domain).Scan(
		&config.OrganizationID,
		&config.CustomDomain,
		&config.LogoURL,
		&config.FaviconURL,
		&config.PrimaryColor,
		&config.SecondaryColor,
		&config.AccentColor,
		&config.CompanyName,
		&config.SupportEmail,
		&config.CustomCSS,
		&config.HideBranding,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query white-label config by domain: %w", err)
	}

	config.CreatedAt = time.Unix(createdAt, 0)
	config.UpdatedAt = time.Unix(updatedAt, 0)

	return &config, nil
}

// Create creates a new white-label configuration
func (s *Store) Create(ctx context.Context, config *WhiteLabelConfig) error {
	query := `
		INSERT INTO white_label_config (
			organization_id, custom_domain, logo_url, favicon_url,
			primary_color, secondary_color, accent_color, company_name,
			support_email, custom_css, hide_branding, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, query,
		config.OrganizationID,
		config.CustomDomain,
		config.LogoURL,
		config.FaviconURL,
		config.PrimaryColor,
		config.SecondaryColor,
		config.AccentColor,
		config.CompanyName,
		config.SupportEmail,
		config.CustomCSS,
		config.HideBranding,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("create white-label config: %w", err)
	}

	return nil
}

// Update updates an existing white-label configuration
func (s *Store) Update(ctx context.Context, config *WhiteLabelConfig) error {
	query := `
		UPDATE white_label_config
		SET custom_domain = ?, logo_url = ?, favicon_url = ?,
		    primary_color = ?, secondary_color = ?, accent_color = ?,
		    company_name = ?, support_email = ?, custom_css = ?,
		    hide_branding = ?, updated_at = ?
		WHERE organization_id = ?
	`

	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, query,
		config.CustomDomain,
		config.LogoURL,
		config.FaviconURL,
		config.PrimaryColor,
		config.SecondaryColor,
		config.AccentColor,
		config.CompanyName,
		config.SupportEmail,
		config.CustomCSS,
		config.HideBranding,
		now,
		config.OrganizationID,
	)

	if err != nil {
		return fmt.Errorf("update white-label config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("white-label config not found")
	}

	return nil
}

// Delete removes white-label configuration
func (s *Store) Delete(ctx context.Context, orgID string) error {
	query := `DELETE FROM white_label_config WHERE organization_id = ?`

	_, err := s.db.ExecContext(ctx, query, orgID)
	if err != nil {
		return fmt.Errorf("delete white-label config: %w", err)
	}

	return nil
}
