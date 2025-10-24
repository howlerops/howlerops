package middleware

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// InvitationRateLimiter provides rate limiting for organization invitations
type InvitationRateLimiter struct {
	// Per-user limiters: max 20 invitations per hour per user
	userLimiters map[string]*invitationLimiter
	// Per-org limiters: max 5 invitations per hour per organization
	orgLimiters map[string]*invitationLimiter
	mu          sync.RWMutex

	// Configuration
	userLimit int // invitations per hour per user
	orgLimit  int // invitations per hour per organization
}

// invitationLimiter tracks rate limiting for a specific entity
type invitationLimiter struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// NewInvitationRateLimiter creates a new invitation rate limiter
// userLimit: max invitations per hour per user (default: 20)
// orgLimit: max invitations per hour per organization (default: 5)
func NewInvitationRateLimiter(userLimit, orgLimit int) *InvitationRateLimiter {
	if userLimit <= 0 {
		userLimit = 20
	}
	if orgLimit <= 0 {
		orgLimit = 5
	}

	rl := &InvitationRateLimiter{
		userLimiters: make(map[string]*invitationLimiter),
		orgLimiters:  make(map[string]*invitationLimiter),
		userLimit:    userLimit,
		orgLimit:     orgLimit,
	}

	// Start cleanup goroutine to prevent memory leaks
	go rl.cleanupExpiredLimiters()

	return rl
}

// CheckUserLimit checks if a user is within their invitation rate limit
// Returns true if the request is allowed, false if rate limit is exceeded
func (r *InvitationRateLimiter) CheckUserLimit(userID string) bool {
	return r.checkLimit(userID, r.userLimiters, r.userLimit)
}

// CheckOrgLimit checks if an organization is within its invitation rate limit
// Returns true if the request is allowed, false if rate limit is exceeded
func (r *InvitationRateLimiter) CheckOrgLimit(orgID string) bool {
	return r.checkLimit(orgID, r.orgLimiters, r.orgLimit)
}

// CheckBothLimits checks both user and organization limits
// Returns true if both limits allow the request
func (r *InvitationRateLimiter) CheckBothLimits(userID, orgID string) (allowed bool, reason string) {
	if !r.CheckUserLimit(userID) {
		return false, "user rate limit exceeded"
	}
	if !r.CheckOrgLimit(orgID) {
		return false, "organization rate limit exceeded"
	}
	return true, ""
}

// checkLimit is the internal method that performs rate limit checks
func (r *InvitationRateLimiter) checkLimit(key string, limiters map[string]*invitationLimiter, maxPerHour int) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := limiters[key]
	if !exists {
		// Create new limiter with rate of maxPerHour per hour
		// Convert to per-second rate: maxPerHour/3600
		ratePerSecond := float64(maxPerHour) / 3600.0
		limiter = &invitationLimiter{
			limiter:    rate.NewLimiter(rate.Limit(ratePerSecond), maxPerHour),
			lastAccess: time.Now(),
		}
		limiters[key] = limiter
	}

	// Update last access time
	limiter.lastAccess = time.Now()

	// Check if request is allowed
	return limiter.limiter.Allow()
}

// GetUserRetryAfter returns the time until a user can send another invitation
func (r *InvitationRateLimiter) GetUserRetryAfter(userID string) time.Duration {
	return r.getRetryAfter(userID, r.userLimiters, r.userLimit)
}

// GetOrgRetryAfter returns the time until an organization can send another invitation
func (r *InvitationRateLimiter) GetOrgRetryAfter(orgID string) time.Duration {
	return r.getRetryAfter(orgID, r.orgLimiters, r.orgLimit)
}

// getRetryAfter calculates when the next request will be allowed
func (r *InvitationRateLimiter) getRetryAfter(key string, limiters map[string]*invitationLimiter, maxPerHour int) time.Duration {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limiter, exists := limiters[key]
	if !exists {
		return 0 // No limiter exists, request would be allowed
	}

	// Calculate when the next token will be available
	// This is an approximation based on the rate
	reservation := limiter.limiter.Reserve()
	delay := reservation.Delay()
	reservation.Cancel()

	return delay
}

// GetUserRemainingInvitations returns how many invitations a user can send immediately
func (r *InvitationRateLimiter) GetUserRemainingInvitations(userID string) int {
	return r.getRemainingInvitations(userID, r.userLimiters, r.userLimit)
}

// GetOrgRemainingInvitations returns how many invitations an org can send immediately
func (r *InvitationRateLimiter) GetOrgRemainingInvitations(orgID string) int {
	return r.getRemainingInvitations(orgID, r.orgLimiters, r.orgLimit)
}

// getRemainingInvitations calculates available tokens
func (r *InvitationRateLimiter) getRemainingInvitations(key string, limiters map[string]*invitationLimiter, maxPerHour int) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	limiter, exists := limiters[key]
	if !exists {
		return maxPerHour // No limiter exists, full quota available
	}

	// Get current number of tokens (burst capacity)
	tokens := int(limiter.limiter.Tokens())
	if tokens < 0 {
		return 0
	}
	return tokens
}

// ResetUserLimit resets the rate limit for a specific user
func (r *InvitationRateLimiter) ResetUserLimit(userID string) {
	r.resetLimit(userID, r.userLimiters)
}

// ResetOrgLimit resets the rate limit for a specific organization
func (r *InvitationRateLimiter) ResetOrgLimit(orgID string) {
	r.resetLimit(orgID, r.orgLimiters)
}

// resetLimit removes a limiter, effectively resetting it
func (r *InvitationRateLimiter) resetLimit(key string, limiters map[string]*invitationLimiter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(limiters, key)
}

// cleanupExpiredLimiters removes inactive limiters to prevent memory leaks
func (r *InvitationRateLimiter) cleanupExpiredLimiters() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		expirationThreshold := 2 * time.Hour

		// Cleanup user limiters
		for key, limiter := range r.userLimiters {
			if now.Sub(limiter.lastAccess) > expirationThreshold {
				delete(r.userLimiters, key)
			}
		}

		// Cleanup org limiters
		for key, limiter := range r.orgLimiters {
			if now.Sub(limiter.lastAccess) > expirationThreshold {
				delete(r.orgLimiters, key)
			}
		}
		r.mu.Unlock()
	}
}

// GetStats returns statistics about current rate limiting state
func (r *InvitationRateLimiter) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return map[string]interface{}{
		"user_limiters_count": len(r.userLimiters),
		"org_limiters_count":  len(r.orgLimiters),
		"user_limit":          r.userLimit,
		"org_limit":           r.orgLimit,
	}
}

// FormatRetryAfter formats a duration for HTTP Retry-After header (in seconds)
func FormatRetryAfter(d time.Duration) string {
	seconds := int(d.Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return fmt.Sprintf("%d", seconds)
}
