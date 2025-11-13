package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/sql-studio/backend-go/internal/quotas"
)

// OrgRateLimiter implements per-organization rate limiting
type OrgRateLimiter struct {
	limiters     sync.Map // map[orgID]*rate.Limiter
	quotaService *quotas.Service
	logger       *logrus.Logger
}

// NewOrgRateLimiter creates a new organization rate limiter
func NewOrgRateLimiter(quotaService *quotas.Service, logger *logrus.Logger) *OrgRateLimiter {
	return &OrgRateLimiter{
		quotaService: quotaService,
		logger:       logger,
	}
}

// Limit applies rate limiting per organization
func (r *OrgRateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		orgID := GetCurrentOrgID(req.Context())
		if orgID == "" {
			// No organization context, allow through (will be caught by auth middleware)
			next.ServeHTTP(w, req)
			return
		}

		// Check quota first (hard limit)
		err := r.quotaService.CheckQuota(req.Context(), orgID, quotas.ResourceAPI)
		if err != nil {
			if quotas.IsQuotaExceeded(err) {
				r.sendRateLimitResponse(w, 0, 0, time.Now().Add(time.Hour))
				r.logger.WithFields(logrus.Fields{
					"organization_id": orgID,
					"path":            req.URL.Path,
				}).Warn("API quota exceeded")
				return
			}
			// Other errors, log and allow through
			r.logger.WithError(err).Warn("Error checking quota")
		}

		// Apply token bucket rate limiting
		limiter := r.getLimiter(orgID)
		if !limiter.Allow() {
			// Calculate when quota resets
			resetTime := time.Now().Add(time.Hour)
			r.sendRateLimitResponse(w, limiter.Limit(), 0, resetTime)

			r.logger.WithFields(logrus.Fields{
				"organization_id": orgID,
				"path":            req.URL.Path,
			}).Debug("Rate limit exceeded")
			return
		}

		// Increment usage counter (async to not slow down request)
		go func() {
			if err := r.quotaService.IncrementUsage(req.Context(), orgID, quotas.ResourceAPI); err != nil {
				r.logger.WithError(err).Error("Failed to increment API usage")
			}
		}()

		// Calculate remaining tokens
		remaining := int(limiter.Tokens())
		resetTime := time.Now().Add(time.Hour)

		// Add rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(limiter.Limit()*3600)))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

		next.ServeHTTP(w, req)
	})
}

// getLimiter gets or creates a rate limiter for an organization
func (r *OrgRateLimiter) getLimiter(orgID string) *rate.Limiter {
	// Load existing limiter
	if limiterInterface, ok := r.limiters.Load(orgID); ok {
		return limiterInterface.(*rate.Limiter)
	}

	// Get organization quota to determine rate
	quota, err := r.quotaService.GetQuota(context.TODO(), orgID)
	if err != nil {
		r.logger.WithError(err).Error("Failed to get quota, using default")
		quota = &quotas.OrganizationQuota{
			MaxAPICallsPerHour: 1000,
		}
	}

	// Create new limiter
	// Rate: calls per second = max per hour / 3600
	// Burst: Allow small bursts (10% of hourly limit or 10, whichever is larger)
	callsPerSecond := float64(quota.MaxAPICallsPerHour) / 3600.0
	burst := quota.MaxAPICallsPerHour / 10
	if burst < 10 {
		burst = 10
	}

	limiter := rate.NewLimiter(rate.Limit(callsPerSecond), burst)

	// Store and return
	r.limiters.Store(orgID, limiter)

	r.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"rate_per_second": callsPerSecond,
		"burst":           burst,
	}).Debug("Created rate limiter")

	return limiter
}

// sendRateLimitResponse sends a rate limit exceeded response
func (r *OrgRateLimiter) sendRateLimitResponse(w http.ResponseWriter, limit rate.Limit, remaining int, resetTime time.Time) {
	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", int(limit*3600)))
	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
	w.Header().Set("Retry-After", fmt.Sprintf("%d", int(time.Until(resetTime).Seconds())))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)

	response := fmt.Sprintf(`{
		"error": "Rate limit exceeded",
		"message": "Too many requests. Please try again later.",
		"retry_after": %d,
		"reset_at": "%s"
	}`, int(time.Until(resetTime).Seconds()), resetTime.Format(time.RFC3339))

	if _, err := w.Write([]byte(response)); err != nil {
		r.logger.WithError(err).Error("Failed to write rate limit response")
	}
}

// CleanupStaleLimit ers removes limiters for inactive organizations
// Should be called periodically (e.g., every hour)
func (r *OrgRateLimiter) CleanupStaleLimiters() {
	r.limiters.Range(func(key, value interface{}) bool {
		// Could add logic to remove limiters that haven't been used recently
		// For now, we keep all limiters (they're lightweight)
		return true
	})
}

// StartCleanupScheduler starts a background goroutine to cleanup stale limiters
func (r *OrgRateLimiter) StartCleanupScheduler() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for range ticker.C {
			r.CleanupStaleLimiters()
		}
	}()
}
