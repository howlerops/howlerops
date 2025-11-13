package domains

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Store handles domain verification persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new domain verification store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create creates a new domain verification record
func (s *Store) Create(ctx context.Context, verification *DomainVerification) error {
	query := `
		INSERT INTO domain_verification (
			id, organization_id, domain, verification_token,
			verified, dns_record_type, dns_record_name, dns_record_value,
			ssl_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, query,
		verification.ID,
		verification.OrganizationID,
		verification.Domain,
		verification.VerificationToken,
		verification.Verified,
		verification.DNSRecordType,
		verification.DNSRecordName,
		verification.DNSRecordValue,
		verification.SSLEnabled,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("create domain verification: %w", err)
	}

	return nil
}

// GetByDomain retrieves domain verification by domain
func (s *Store) GetByDomain(ctx context.Context, orgID, domain string) (*DomainVerification, error) {
	query := `
		SELECT id, organization_id, domain, verification_token,
		       verified, verified_at, dns_record_type, dns_record_name,
		       dns_record_value, ssl_enabled, ssl_certificate_expires_at,
		       created_at, updated_at
		FROM domain_verification
		WHERE organization_id = ? AND domain = ?
	`

	var v DomainVerification
	var verifiedAt, sslExpiresAt sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, orgID, domain).Scan(
		&v.ID,
		&v.OrganizationID,
		&v.Domain,
		&v.VerificationToken,
		&v.Verified,
		&verifiedAt,
		&v.DNSRecordType,
		&v.DNSRecordName,
		&v.DNSRecordValue,
		&v.SSLEnabled,
		&sslExpiresAt,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query domain verification: %w", err)
	}

	if verifiedAt.Valid {
		t := time.Unix(verifiedAt.Int64, 0)
		v.VerifiedAt = &t
	}

	if sslExpiresAt.Valid {
		t := time.Unix(sslExpiresAt.Int64, 0)
		v.SSLCertificateExpiresAt = &t
	}

	v.CreatedAt = time.Unix(createdAt, 0)
	v.UpdatedAt = time.Unix(updatedAt, 0)

	return &v, nil
}

// ListByOrganization lists all domains for an organization
func (s *Store) ListByOrganization(ctx context.Context, orgID string) ([]*DomainVerification, error) {
	query := `
		SELECT id, organization_id, domain, verification_token,
		       verified, verified_at, dns_record_type, dns_record_name,
		       dns_record_value, ssl_enabled, ssl_certificate_expires_at,
		       created_at, updated_at
		FROM domain_verification
		WHERE organization_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("query domains: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	var domains []*DomainVerification
	for rows.Next() {
		var v DomainVerification
		var verifiedAt, sslExpiresAt sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(
			&v.ID,
			&v.OrganizationID,
			&v.Domain,
			&v.VerificationToken,
			&v.Verified,
			&verifiedAt,
			&v.DNSRecordType,
			&v.DNSRecordName,
			&v.DNSRecordValue,
			&v.SSLEnabled,
			&sslExpiresAt,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}

		if verifiedAt.Valid {
			t := time.Unix(verifiedAt.Int64, 0)
			v.VerifiedAt = &t
		}

		if sslExpiresAt.Valid {
			t := time.Unix(sslExpiresAt.Int64, 0)
			v.SSLCertificateExpiresAt = &t
		}

		v.CreatedAt = time.Unix(createdAt, 0)
		v.UpdatedAt = time.Unix(updatedAt, 0)

		domains = append(domains, &v)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domains: %w", err)
	}

	return domains, nil
}

// MarkVerified marks a domain as verified
func (s *Store) MarkVerified(ctx context.Context, verificationID string) error {
	query := `
		UPDATE domain_verification
		SET verified = true, verified_at = ?, updated_at = ?
		WHERE id = ?
	`

	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, query, now, now, verificationID)
	if err != nil {
		return fmt.Errorf("mark verified: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("domain verification not found")
	}

	return nil
}

// Delete removes a domain verification
func (s *Store) Delete(ctx context.Context, orgID, domain string) error {
	query := `
		DELETE FROM domain_verification
		WHERE organization_id = ? AND domain = ?
	`

	_, err := s.db.ExecContext(ctx, query, orgID, domain)
	if err != nil {
		return fmt.Errorf("delete domain verification: %w", err)
	}

	return nil
}
