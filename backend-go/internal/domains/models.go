package domains

import "time"

// DomainVerification represents a domain verification record
type DomainVerification struct {
	ID                       string    `json:"id" db:"id"`
	OrganizationID           string    `json:"organization_id" db:"organization_id"`
	Domain                   string    `json:"domain" db:"domain"`
	VerificationToken        string    `json:"verification_token" db:"verification_token"`
	Verified                 bool      `json:"verified" db:"verified"`
	VerifiedAt               *time.Time `json:"verified_at,omitempty" db:"verified_at"`
	DNSRecordType            string    `json:"dns_record_type" db:"dns_record_type"`
	DNSRecordName            string    `json:"dns_record_name" db:"dns_record_name"`
	DNSRecordValue           string    `json:"dns_record_value" db:"dns_record_value"`
	SSLEnabled               bool      `json:"ssl_enabled" db:"ssl_enabled"`
	SSLCertificateExpiresAt  *time.Time `json:"ssl_certificate_expires_at,omitempty" db:"ssl_certificate_expires_at"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// AddDomainRequest represents a request to add a custom domain
type AddDomainRequest struct {
	Domain string `json:"domain" validate:"required"`
}

// VerifyDomainRequest represents a request to verify a domain
type VerifyDomainRequest struct {
	Domain string `json:"domain" validate:"required"`
}
