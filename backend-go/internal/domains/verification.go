package domains

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Service handles domain verification logic
type Service struct {
	store  *Store
	logger *logrus.Logger
}

// NewService creates a new domain verification service
func NewService(store *Store, logger *logrus.Logger) *Service {
	return &Service{
		store:  store,
		logger: logger,
	}
}

// InitiateVerification starts the domain verification process
func (s *Service) InitiateVerification(ctx context.Context, orgID, domain string) (*DomainVerification, error) {
	// Normalize domain
	domain = strings.ToLower(strings.TrimSpace(domain))
	domain = strings.TrimPrefix(domain, "http://")
	domain = strings.TrimPrefix(domain, "https://")
	domain = strings.TrimSuffix(domain, "/")

	// Check if domain already exists
	existing, err := s.store.GetByDomain(ctx, orgID, domain)
	if err != nil {
		return nil, fmt.Errorf("check existing domain: %w", err)
	}

	if existing != nil {
		if existing.Verified {
			return existing, nil // Already verified
		}
		// Delete old verification and create new one
		if err := s.store.Delete(ctx, orgID, domain); err != nil {
			s.logger.WithError(err).Warn("Failed to delete old verification")
		}
	}

	// Generate verification token
	token, err := s.generateVerificationToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	// Create verification record
	verification := &DomainVerification{
		ID:                uuid.New().String(),
		OrganizationID:    orgID,
		Domain:            domain,
		VerificationToken: token,
		Verified:          false,
		DNSRecordType:     "TXT",
		DNSRecordName:     fmt.Sprintf("_sql-studio-verification.%s", domain),
		DNSRecordValue:    token,
		SSLEnabled:        false,
	}

	if err := s.store.Create(ctx, verification); err != nil {
		return nil, fmt.Errorf("create verification: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"domain":          domain,
		"record_name":     verification.DNSRecordName,
	}).Info("Domain verification initiated")

	return verification, nil
}

// VerifyDomain checks DNS records and marks domain as verified
func (s *Service) VerifyDomain(ctx context.Context, orgID, domain string) (*DomainVerification, error) {
	// Normalize domain
	domain = strings.ToLower(strings.TrimSpace(domain))

	// Get verification record
	verification, err := s.store.GetByDomain(ctx, orgID, domain)
	if err != nil {
		return nil, fmt.Errorf("get verification: %w", err)
	}

	if verification == nil {
		return nil, fmt.Errorf("domain verification not found")
	}

	if verification.Verified {
		return verification, nil // Already verified
	}

	// Check DNS TXT record
	txtRecords, err := net.LookupTXT(verification.DNSRecordName)
	if err != nil {
		s.logger.WithError(err).WithFields(logrus.Fields{
			"domain":      domain,
			"record_name": verification.DNSRecordName,
		}).Warn("DNS lookup failed")
		return nil, fmt.Errorf("DNS lookup failed: %w (make sure you've added the TXT record)", err)
	}

	// Verify token
	tokenFound := false
	for _, record := range txtRecords {
		s.logger.WithFields(logrus.Fields{
			"domain":   domain,
			"record":   record,
			"expected": verification.VerificationToken,
		}).Debug("Checking TXT record")

		if strings.TrimSpace(record) == verification.VerificationToken {
			tokenFound = true
			break
		}
	}

	if !tokenFound {
		return nil, fmt.Errorf("verification token not found in DNS records (found %d TXT records)", len(txtRecords))
	}

	// Mark as verified
	if err := s.store.MarkVerified(ctx, verification.ID); err != nil {
		return nil, fmt.Errorf("mark verified: %w", err)
	}

	// Reload verification
	verification, err = s.store.GetByDomain(ctx, orgID, domain)
	if err != nil {
		return nil, fmt.Errorf("reload verification: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"domain":          domain,
	}).Info("Domain verified successfully")

	return verification, nil
}

// ListDomains lists all domains for an organization
func (s *Service) ListDomains(ctx context.Context, orgID string) ([]*DomainVerification, error) {
	domains, err := s.store.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	return domains, nil
}

// RemoveDomain removes a domain verification
func (s *Service) RemoveDomain(ctx context.Context, orgID, domain string) error {
	// Normalize domain
	domain = strings.ToLower(strings.TrimSpace(domain))

	if err := s.store.Delete(ctx, orgID, domain); err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"domain":          domain,
	}).Info("Domain removed")

	return nil
}

// generateVerificationToken generates a random verification token
func (s *Service) generateVerificationToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GetVerificationInstructions returns human-readable instructions for domain verification
func (s *Service) GetVerificationInstructions(verification *DomainVerification) string {
	return fmt.Sprintf(`To verify ownership of %s, please add the following DNS record:

Type: %s
Name: %s
Value: %s

Instructions:
1. Log in to your DNS provider (e.g., Cloudflare, GoDaddy, Route53)
2. Add a new TXT record with the information above
3. Wait for DNS propagation (typically 5-60 minutes)
4. Click "Verify Domain" to complete the process

Note: DNS changes can take up to 48 hours to fully propagate, but usually complete within an hour.`,
		verification.Domain,
		verification.DNSRecordType,
		verification.DNSRecordName,
		verification.VerificationToken,
	)
}
