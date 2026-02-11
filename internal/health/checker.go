package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// Status represents the health status
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// HealthCheck represents the overall health check response
type HealthCheck struct {
	Status    Status                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    time.Duration          `json:"uptime_seconds"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of an individual health check
type CheckResult struct {
	Status       Status        `json:"status"`
	Message      string        `json:"message,omitempty"`
	ResponseTime time.Duration `json:"response_time_ms"`
	Timestamp    time.Time     `json:"timestamp"`
	Details      interface{}   `json:"details,omitempty"`
}

// Checker interface for health checks
type Checker interface {
	Check(ctx context.Context) CheckResult
	Name() string
}

// HealthChecker performs health checks
type HealthChecker struct {
	version   string
	startTime time.Time
	checkers  map[string]Checker
	mu        sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		version:   version,
		startTime: time.Now(),
		checkers:  make(map[string]Checker),
	}
}

// RegisterChecker registers a health check
func (h *HealthChecker) RegisterChecker(checker Checker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[checker.Name()] = checker
}

// Check performs all registered health checks
func (h *HealthChecker) Check(ctx context.Context) *HealthCheck {
	h.mu.RLock()
	checkers := make(map[string]Checker, len(h.checkers))
	for name, checker := range h.checkers {
		checkers[name] = checker
	}
	h.mu.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Run all checks in parallel with timeout
	for name, checker := range checkers {
		wg.Add(1)
		go func(name string, checker Checker) {
			defer wg.Done()

			// Create timeout context for individual check
			checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			result := checker.Check(checkCtx)

			mu.Lock()
			results[name] = result
			mu.Unlock()
		}(name, checker)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := h.determineOverallStatus(results)

	return &HealthCheck{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime),
		Version:   h.version,
		Checks:    results,
	}
}

// Live returns a simple liveness check (for Kubernetes liveness probe)
func (h *HealthChecker) Live() bool {
	return true // If we can respond, we're alive
}

// Ready returns a readiness check (for Kubernetes readiness probe)
func (h *HealthChecker) Ready(ctx context.Context) bool {
	check := h.Check(ctx)
	// Ready if healthy or degraded (not unhealthy)
	return check.Status != StatusUnhealthy
}

// determineOverallStatus determines overall health status from individual checks
func (h *HealthChecker) determineOverallStatus(results map[string]CheckResult) Status {
	if len(results) == 0 {
		return StatusHealthy
	}

	hasUnhealthy := false
	hasDegraded := false

	for _, result := range results {
		switch result.Status {
		case StatusUnhealthy:
			hasUnhealthy = true
		case StatusDegraded:
			hasDegraded = true
		}
	}

	if hasUnhealthy {
		return StatusUnhealthy
	}
	if hasDegraded {
		return StatusDegraded
	}
	return StatusHealthy
}

// ============================================================================
// Built-in Checkers
// ============================================================================

// DatabaseChecker checks database connectivity
type DatabaseChecker struct {
	name string
	db   *sql.DB
}

// NewDatabaseChecker creates a new database health checker
func NewDatabaseChecker(name string, db *sql.DB) *DatabaseChecker {
	return &DatabaseChecker{
		name: name,
		db:   db,
	}
}

func (c *DatabaseChecker) Name() string {
	return c.name
}

func (c *DatabaseChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()

	// Try to ping database
	err := c.db.PingContext(ctx)
	responseTime := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:       StatusUnhealthy,
			Message:      fmt.Sprintf("Database ping failed: %v", err),
			ResponseTime: responseTime,
			Timestamp:    time.Now(),
		}
	}

	// Get database stats
	stats := c.db.Stats()
	details := map[string]interface{}{
		"open_connections": stats.OpenConnections,
		"in_use":           stats.InUse,
		"idle":             stats.Idle,
		"max_open":         stats.MaxOpenConnections,
	}

	// Check if connection pool is nearly exhausted
	if stats.MaxOpenConnections > 0 {
		utilization := float64(stats.OpenConnections) / float64(stats.MaxOpenConnections)
		if utilization > 0.9 {
			return CheckResult{
				Status:       StatusDegraded,
				Message:      fmt.Sprintf("Connection pool utilization high: %.1f%%", utilization*100),
				ResponseTime: responseTime,
				Timestamp:    time.Now(),
				Details:      details,
			}
		}
	}

	return CheckResult{
		Status:       StatusHealthy,
		Message:      "Database connection healthy",
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
		Details:      details,
	}
}

// RedisChecker checks Redis connectivity
type RedisChecker struct {
	name string
	ping func(ctx context.Context) error
}

// NewRedisChecker creates a new Redis health checker
func NewRedisChecker(name string, pingFunc func(ctx context.Context) error) *RedisChecker {
	return &RedisChecker{
		name: name,
		ping: pingFunc,
	}
}

func (c *RedisChecker) Name() string {
	return c.name
}

func (c *RedisChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()

	err := c.ping(ctx)
	responseTime := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:       StatusUnhealthy,
			Message:      fmt.Sprintf("Redis ping failed: %v", err),
			ResponseTime: responseTime,
			Timestamp:    time.Now(),
		}
	}

	return CheckResult{
		Status:       StatusHealthy,
		Message:      "Redis connection healthy",
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
	}
}

// HTTPChecker checks external HTTP dependencies
type HTTPChecker struct {
	name string
	url  string
}

// NewHTTPChecker creates a new HTTP health checker
func NewHTTPChecker(name string, url string) *HTTPChecker {
	return &HTTPChecker{
		name: name,
		url:  url,
	}
}

func (c *HTTPChecker) Name() string {
	return c.name
}

func (c *HTTPChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", c.url, nil)
	if err != nil {
		return CheckResult{
			Status:       StatusUnhealthy,
			Message:      fmt.Sprintf("Failed to create request: %v", err),
			ResponseTime: time.Since(start),
			Timestamp:    time.Now(),
		}
	}

	// Make request
	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:       StatusUnhealthy,
			Message:      fmt.Sprintf("HTTP request failed: %v", err),
			ResponseTime: responseTime,
			Timestamp:    time.Now(),
		}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Log error but don't fail the check - response already received
			_ = err
		}
	}()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return CheckResult{
			Status:       StatusHealthy,
			Message:      fmt.Sprintf("HTTP endpoint healthy (status: %d)", resp.StatusCode),
			ResponseTime: responseTime,
			Timestamp:    time.Now(),
			Details: map[string]interface{}{
				"status_code": resp.StatusCode,
			},
		}
	}

	return CheckResult{
		Status:       StatusUnhealthy,
		Message:      fmt.Sprintf("HTTP endpoint returned error status: %d", resp.StatusCode),
		ResponseTime: responseTime,
		Timestamp:    time.Now(),
		Details: map[string]interface{}{
			"status_code": resp.StatusCode,
		},
	}
}

// CustomChecker allows custom health check logic
type CustomChecker struct {
	name      string
	checkFunc func(ctx context.Context) CheckResult
}

// NewCustomChecker creates a new custom health checker
func NewCustomChecker(name string, checkFunc func(ctx context.Context) CheckResult) *CustomChecker {
	return &CustomChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

func (c *CustomChecker) Name() string {
	return c.name
}

func (c *CustomChecker) Check(ctx context.Context) CheckResult {
	return c.checkFunc(ctx)
}
