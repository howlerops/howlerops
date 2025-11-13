package middleware

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// IPWhitelistEntry represents an IP whitelist entry
type IPWhitelistEntry struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	IPAddress      string    `json:"ip_address"`
	IPRange        string    `json:"ip_range"` // CIDR notation
	Description    string    `json:"description"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// IPWhitelistStore defines the interface for IP whitelist storage
type IPWhitelistStore interface {
	GetWhitelist(ctx context.Context, organizationID string) ([]*IPWhitelistEntry, error)
	AddIPToWhitelist(ctx context.Context, entry *IPWhitelistEntry) error
	RemoveIPFromWhitelist(ctx context.Context, id string) error
	IsIPWhitelisted(ctx context.Context, organizationID, ip string) (bool, error)
}

// SecurityEventLogger logs security events
type SecurityEventLogger interface {
	LogSecurityEvent(ctx context.Context, eventType, userID, orgID, ipAddress, userAgent string, details map[string]interface{}) error
}

// IPWhitelistMiddleware provides IP whitelisting functionality
type IPWhitelistMiddleware struct {
	store       IPWhitelistStore
	eventLogger SecurityEventLogger
	logger      *logrus.Logger
	enabled     bool
}

// NewIPWhitelistMiddleware creates a new IP whitelist middleware
func NewIPWhitelistMiddleware(store IPWhitelistStore, eventLogger SecurityEventLogger, logger *logrus.Logger) *IPWhitelistMiddleware {
	return &IPWhitelistMiddleware{
		store:       store,
		eventLogger: eventLogger,
		logger:      logger,
		enabled:     true,
	}
}

// CheckIP middleware checks if the client IP is whitelisted
func (m *IPWhitelistMiddleware) CheckIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Extract client IP
		clientIP := getClientIP(r)

		// Get organization from context
		orgID := getOrgIDFromContext(r.Context())
		if orgID == "" {
			// No organization context, skip IP check
			next.ServeHTTP(w, r)
			return
		}

		// Check if organization has IP whitelist
		whitelist, err := m.store.GetWhitelist(r.Context(), orgID)
		if err != nil {
			m.logger.WithError(err).Error("Failed to get IP whitelist")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// If no whitelist configured, allow all IPs
		if len(whitelist) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		// Check if IP is whitelisted
		if !isIPWhitelisted(clientIP, whitelist) {
			// Log security event
			if m.eventLogger != nil {
				if err := m.eventLogger.LogSecurityEvent(
					r.Context(),
					"ip_blocked",
					getUserIDFromContext(r.Context()),
					orgID,
					clientIP,
					r.Header.Get("User-Agent"),
					map[string]interface{}{
						"path":   r.URL.Path,
						"method": r.Method,
					},
				); err != nil {
					m.logger.WithError(err).Warn("Failed to log IP blocked security event")
				}
			}

			m.logger.WithFields(logrus.Fields{
				"ip":              clientIP,
				"organization_id": orgID,
			}).Warn("Access denied: IP not whitelisted")

			http.Error(w, "Access denied: IP address not authorized", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isIPWhitelisted checks if an IP is in the whitelist
func isIPWhitelisted(clientIP string, whitelist []*IPWhitelistEntry) bool {
	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, entry := range whitelist {
		// Check exact IP match
		if entry.IPAddress == clientIP {
			return true
		}

		// Check CIDR range if specified
		if entry.IPRange != "" {
			_, ipnet, err := net.ParseCIDR(entry.IPRange)
			if err == nil && ipnet.Contains(ip) {
				return true
			}
		}
	}

	return false
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (proxy/load balancer)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// Return the first IP (original client)
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Check CF-Connecting-IP for Cloudflare
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// getOrgIDFromContext extracts organization ID from context
func getOrgIDFromContext(ctx context.Context) string {
	if orgID, ok := ctx.Value("organization_id").(string); ok {
		return orgID
	}
	return ""
}

// getUserIDFromContext extracts user ID from context
func getUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// ValidateCIDR validates a CIDR notation string
func ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	return err
}

// ValidateIP validates an IP address
func ValidateIP(ip string) error {
	if net.ParseIP(ip) == nil {
		return net.InvalidAddrError(ip)
	}
	return nil
}

// IPWhitelistService handles IP whitelist management
type IPWhitelistService struct {
	store       IPWhitelistStore
	eventLogger SecurityEventLogger
	logger      *logrus.Logger
}

// NewIPWhitelistService creates a new IP whitelist service
func NewIPWhitelistService(store IPWhitelistStore, eventLogger SecurityEventLogger, logger *logrus.Logger) *IPWhitelistService {
	return &IPWhitelistService{
		store:       store,
		eventLogger: eventLogger,
		logger:      logger,
	}
}

// AddIP adds an IP or CIDR range to the whitelist
func (s *IPWhitelistService) AddIP(ctx context.Context, entry *IPWhitelistEntry) error {
	// Validate IP address if specified
	if entry.IPAddress != "" {
		if err := ValidateIP(entry.IPAddress); err != nil {
			return err
		}
	}

	// Validate CIDR range if specified
	if entry.IPRange != "" {
		if err := ValidateCIDR(entry.IPRange); err != nil {
			return err
		}
	}

	// At least one must be specified
	if entry.IPAddress == "" && entry.IPRange == "" {
		return net.InvalidAddrError("either IP address or CIDR range must be specified")
	}

	entry.CreatedAt = time.Now()

	if err := s.store.AddIPToWhitelist(ctx, entry); err != nil {
		return err
	}

	// Log security event
	if s.eventLogger != nil {
		s.eventLogger.LogSecurityEvent(
			ctx,
			"ip_whitelist_added",
			entry.CreatedBy,
			entry.OrganizationID,
			"",
			"",
			map[string]interface{}{
				"ip_address": entry.IPAddress,
				"ip_range":   entry.IPRange,
			},
		)
	}

	return nil
}

// RemoveIP removes an IP from the whitelist
func (s *IPWhitelistService) RemoveIP(ctx context.Context, id, userID, orgID string) error {
	if err := s.store.RemoveIPFromWhitelist(ctx, id); err != nil {
		return err
	}

	// Log security event
	if s.eventLogger != nil {
		s.eventLogger.LogSecurityEvent(
			ctx,
			"ip_whitelist_removed",
			userID,
			orgID,
			"",
			"",
			map[string]interface{}{
				"entry_id": id,
			},
		)
	}

	return nil
}

// GetWhitelist retrieves the IP whitelist for an organization
func (s *IPWhitelistService) GetWhitelist(ctx context.Context, orgID string) ([]*IPWhitelistEntry, error) {
	return s.store.GetWhitelist(ctx, orgID)
}
