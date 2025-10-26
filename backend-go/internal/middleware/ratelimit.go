package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimitMiddleware provides rate limiting for gRPC requests
type RateLimitMiddleware struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      int
	burst    int
}

// NewRateLimitMiddleware creates a new rate limit middleware
func NewRateLimitMiddleware(rps, burst int) *RateLimitMiddleware {
	return &RateLimitMiddleware{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}
}

// UnaryInterceptor provides rate limiting for unary gRPC calls
func (r *RateLimitMiddleware) UnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Get client IP
	clientIP := r.getClientIP(ctx)

	// Check rate limit
	if !r.checkRateLimit(clientIP) {
		return nil, status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}

	return handler(ctx, req)
}

// StreamInterceptor provides rate limiting for streaming gRPC calls
func (r *RateLimitMiddleware) StreamInterceptor(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	// Get client IP
	clientIP := r.getClientIP(stream.Context())

	// Check rate limit
	if !r.checkRateLimit(clientIP) {
		return status.Errorf(codes.ResourceExhausted, "rate limit exceeded")
	}

	return handler(srv, stream)
}

// getClientIP extracts the client IP from the context
func (r *RateLimitMiddleware) getClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}

	// Extract IP from address
	addr := p.Addr.String()
	if host, _, err := splitHostPort(addr); err == nil {
		return host
	}

	return addr
}

// checkRateLimit checks if the request is within rate limits
func (r *RateLimitMiddleware) checkRateLimit(clientIP string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	limiter, exists := r.limiters[clientIP]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(r.rps), r.burst)
		r.limiters[clientIP] = limiter
	}

	return limiter.Allow()
}

// CleanupExpiredLimiters removes expired rate limiters to prevent memory leaks
func (r *RateLimitMiddleware) CleanupExpiredLimiters() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		r.mu.Lock()
		for ip, limiter := range r.limiters {
			// Remove limiters that have full tokens (indicating they haven't been used recently)
			if limiter.Tokens() == float64(r.burst) {
				delete(r.limiters, ip)
			}
		}
		r.mu.Unlock()
	}
}

// PerUserRateLimiter provides per-user rate limiting
type PerUserRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rps      int
	burst    int
}

// NewPerUserRateLimiter creates a new per-user rate limiter
func NewPerUserRateLimiter(rps, burst int) *PerUserRateLimiter {
	return &PerUserRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rps:      rps,
		burst:    burst,
	}
}

// CheckLimit checks if the user is within rate limits
func (p *PerUserRateLimiter) CheckLimit(userID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	limiter, exists := p.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(p.rps), p.burst)
		p.limiters[userID] = limiter
	}

	return limiter.Allow()
}

// PerMethodRateLimiter provides per-method rate limiting
type PerMethodRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   map[string]MethodRateConfig
}

// MethodRateConfig holds rate limiting configuration for a specific method
type MethodRateConfig struct {
	RPS   int
	Burst int
}

// NewPerMethodRateLimiter creates a new per-method rate limiter
func NewPerMethodRateLimiter(config map[string]MethodRateConfig) *PerMethodRateLimiter {
	return &PerMethodRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
	}
}

// CheckLimit checks if the method call is within rate limits
func (p *PerMethodRateLimiter) CheckLimit(method, clientIP string) bool {
	key := fmt.Sprintf("%s:%s", method, clientIP)

	p.mu.Lock()
	defer p.mu.Unlock()

	limiter, exists := p.limiters[key]
	if !exists {
		config, configExists := p.config[method]
		if !configExists {
			// Use default config if method-specific config doesn't exist
			config = MethodRateConfig{RPS: 100, Burst: 200}
		}

		limiter = rate.NewLimiter(rate.Limit(config.RPS), config.Burst)
		p.limiters[key] = limiter
	}

	return limiter.Allow()
}

// AdaptiveRateLimiter adjusts rate limits based on system load
type AdaptiveRateLimiter struct {
	baseLimiter *RateLimitMiddleware
	mu          sync.RWMutex
	loadFactor  float64
	minRPS      int
	maxRPS      int
}

// NewAdaptiveRateLimiter creates a new adaptive rate limiter
func NewAdaptiveRateLimiter(baseRPS, burst, minRPS, maxRPS int) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		baseLimiter: NewRateLimitMiddleware(baseRPS, burst),
		loadFactor:  1.0,
		minRPS:      minRPS,
		maxRPS:      maxRPS,
	}
}

// UpdateLoadFactor updates the system load factor
func (a *AdaptiveRateLimiter) UpdateLoadFactor(loadFactor float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.loadFactor = loadFactor

	// Adjust rate limits based on load
	newRPS := int(float64(a.baseLimiter.rps) / loadFactor)
	if newRPS < a.minRPS {
		newRPS = a.minRPS
	}
	if newRPS > a.maxRPS {
		newRPS = a.maxRPS
	}

	a.baseLimiter.rps = newRPS
}

// CheckLimit checks if the request is within adaptive rate limits
func (a *AdaptiveRateLimiter) CheckLimit(clientIP string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.baseLimiter.checkRateLimit(clientIP)
}

// CircuitBreakerRateLimiter combines rate limiting with circuit breaker pattern
type CircuitBreakerRateLimiter struct {
	rateLimiter      *RateLimitMiddleware
	failureCount     int
	lastFailureTime  time.Time
	state            CircuitState
	mu               sync.RWMutex
	failureThreshold int
	recoveryTimeout  time.Duration
}

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreakerRateLimiter creates a new circuit breaker rate limiter
func NewCircuitBreakerRateLimiter(rps, burst, failureThreshold int, recoveryTimeout time.Duration) *CircuitBreakerRateLimiter {
	return &CircuitBreakerRateLimiter{
		rateLimiter:      NewRateLimitMiddleware(rps, burst),
		state:            CircuitClosed,
		failureThreshold: failureThreshold,
		recoveryTimeout:  recoveryTimeout,
	}
}

// CheckLimit checks if the request is allowed by both rate limiter and circuit breaker
func (c *CircuitBreakerRateLimiter) CheckLimit(clientIP string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check circuit breaker state
	switch c.state {
	case CircuitOpen:
		// Check if recovery timeout has passed
		if time.Since(c.lastFailureTime) > c.recoveryTimeout {
			c.state = CircuitHalfOpen
		} else {
			return false
		}
	case CircuitHalfOpen:
		// Allow limited requests in half-open state
		// This is a simplified implementation
		break
	case CircuitClosed:
		// Normal operation
		break
	}

	// Check rate limit
	return c.rateLimiter.checkRateLimit(clientIP)
}

// RecordFailure records a failure for circuit breaker logic
func (c *CircuitBreakerRateLimiter) RecordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount++
	c.lastFailureTime = time.Now()

	if c.failureCount >= c.failureThreshold {
		c.state = CircuitOpen
	}
}

// RecordSuccess records a success for circuit breaker logic
func (c *CircuitBreakerRateLimiter) RecordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.failureCount = 0
	c.state = CircuitClosed
}

// splitHostPort is a helper function to split host and port
func splitHostPort(hostport string) (host, port string, err error) {
	// Simple implementation - in production, use net.SplitHostPort
	lastColon := -1
	for i := len(hostport) - 1; i >= 0; i-- {
		if hostport[i] == ':' {
			lastColon = i
			break
		}
	}

	if lastColon == -1 {
		return hostport, "", nil
	}

	return hostport[:lastColon], hostport[lastColon+1:], nil
}
