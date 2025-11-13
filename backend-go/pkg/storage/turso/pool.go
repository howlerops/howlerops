package turso

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type ConnectionPool struct {
	db              *sql.DB
	config          *PoolConfig
	logger          *logrus.Logger
	stats           *PoolStats
	healthCheckStop chan struct{}
	mu              sync.RWMutex
	healthStatus    bool
}

type PoolConfig struct {
	DSN                string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    time.Duration
	ConnMaxIdleTime    time.Duration
	HealthCheckPeriod  time.Duration
	SlowQueryThreshold time.Duration
}

type PoolStats struct {
	OpenConnections   int32   `json:"open_connections"`
	InUse             int32   `json:"in_use"`
	Idle              int32   `json:"idle"`
	WaitCount         int64   `json:"wait_count"`
	WaitDuration      int64   `json:"wait_duration_ms"`
	MaxIdleClosed     int64   `json:"max_idle_closed"`
	MaxLifetimeClosed int64   `json:"max_lifetime_closed"`
	TotalRequests     uint64  `json:"total_requests"`
	TotalErrors       uint64  `json:"total_errors"`
	SlowQueries       uint64  `json:"slow_queries"`
	HealthStatus      bool    `json:"health_status"`
	Efficiency        float64 `json:"efficiency_percent"`
}

// DefaultPoolConfig returns optimized default configuration
func DefaultPoolConfig(dsn string) *PoolConfig {
	return &PoolConfig{
		DSN:                dsn,
		MaxOpenConns:       25,
		MaxIdleConns:       10,
		ConnMaxLifetime:    5 * time.Minute,
		ConnMaxIdleTime:    2 * time.Minute,
		HealthCheckPeriod:  30 * time.Second,
		SlowQueryThreshold: 100 * time.Millisecond,
	}
}

// NewOptimizedPool creates a new optimized connection pool
func NewOptimizedPool(config *PoolConfig, logger *logrus.Logger) (*ConnectionPool, error) {
	if config == nil {
		return nil, fmt.Errorf("pool config is required")
	}

	db, err := sql.Open("libsql", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Apply optimized pool settings
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)
	db.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		if closeErr := db.Close(); closeErr != nil {
			logger.Errorf("Failed to close database after ping failure: %v", closeErr)
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	pool := &ConnectionPool{
		db:              db,
		config:          config,
		logger:          logger,
		stats:           &PoolStats{},
		healthCheckStop: make(chan struct{}),
		healthStatus:    true,
	}

	// Start health monitoring
	go pool.monitorHealth()

	logger.WithFields(logrus.Fields{
		"max_open_conns": config.MaxOpenConns,
		"max_idle_conns": config.MaxIdleConns,
		"max_lifetime":   config.ConnMaxLifetime,
		"max_idle_time":  config.ConnMaxIdleTime,
	}).Info("Connection pool initialized")

	return pool, nil
}

// GetDB returns the underlying database connection
func (p *ConnectionPool) GetDB() *sql.DB {
	return p.db
}

// Execute executes a query with performance tracking
func (p *ConnectionPool) Execute(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	atomic.AddUint64(&p.stats.TotalRequests, 1)

	result, err := p.db.ExecContext(ctx, query, args...)

	duration := time.Since(start)
	if duration > p.config.SlowQueryThreshold {
		atomic.AddUint64(&p.stats.SlowQueries, 1)
		p.logger.WithFields(logrus.Fields{
			"duration_ms": duration.Milliseconds(),
			"query":       truncateQuery(query, 100),
		}).Warn("Slow query detected")
	}

	if err != nil {
		atomic.AddUint64(&p.stats.TotalErrors, 1)
		return nil, err
	}

	return result, nil
}

// Query executes a query and returns rows with performance tracking
func (p *ConnectionPool) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	atomic.AddUint64(&p.stats.TotalRequests, 1)

	rows, err := p.db.QueryContext(ctx, query, args...)

	duration := time.Since(start)
	if duration > p.config.SlowQueryThreshold {
		atomic.AddUint64(&p.stats.SlowQueries, 1)
		p.logger.WithFields(logrus.Fields{
			"duration_ms": duration.Milliseconds(),
			"query":       truncateQuery(query, 100),
		}).Warn("Slow query detected")
	}

	if err != nil {
		atomic.AddUint64(&p.stats.TotalErrors, 1)
		return nil, err
	}

	return rows, nil
}

// GetStats returns current pool statistics
func (p *ConnectionPool) GetStats() *PoolStats {
	dbStats := p.db.Stats()

	stats := &PoolStats{
		// #nosec G115 - connection counts are reasonable (<1000), well within int32 range
		OpenConnections:   int32(dbStats.OpenConnections),
		// #nosec G115 - connection counts are reasonable (<1000), well within int32 range
		InUse:             int32(dbStats.InUse),
		// #nosec G115 - connection counts are reasonable (<1000), well within int32 range
		Idle:              int32(dbStats.Idle),
		WaitCount:         dbStats.WaitCount,
		WaitDuration:      dbStats.WaitDuration.Milliseconds(),
		MaxIdleClosed:     dbStats.MaxIdleClosed,
		MaxLifetimeClosed: dbStats.MaxLifetimeClosed,
		TotalRequests:     atomic.LoadUint64(&p.stats.TotalRequests),
		TotalErrors:       atomic.LoadUint64(&p.stats.TotalErrors),
		SlowQueries:       atomic.LoadUint64(&p.stats.SlowQueries),
		HealthStatus:      p.getHealthStatus(),
	}

	// Calculate efficiency
	if stats.OpenConnections > 0 {
		stats.Efficiency = float64(stats.InUse) / float64(stats.OpenConnections) * 100
	}

	return stats
}

// monitorHealth performs periodic health checks
func (p *ConnectionPool) monitorHealth() {
	ticker := time.NewTicker(p.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.performHealthCheck()
		case <-p.healthCheckStop:
			return
		}
	}
}

// performHealthCheck checks pool health
func (p *ConnectionPool) performHealthCheck() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := p.db.PingContext(ctx)

	p.mu.Lock()
	p.healthStatus = err == nil
	p.mu.Unlock()

	if err != nil {
		p.logger.WithError(err).Error("Connection pool health check failed")

		// Try to recover
		go p.attemptRecovery()
	}

	// Log pool stats periodically
	stats := p.GetStats()
	if stats.WaitCount > 100 || stats.Efficiency < 50 {
		p.logger.WithFields(logrus.Fields{
			"efficiency":    stats.Efficiency,
			"wait_count":    stats.WaitCount,
			"wait_duration": stats.WaitDuration,
			"in_use":        stats.InUse,
			"idle":          stats.Idle,
		}).Warn("Pool performance degradation detected")
	}
}

// attemptRecovery attempts to recover unhealthy connections
func (p *ConnectionPool) attemptRecovery() {
	p.logger.Info("Attempting connection pool recovery")

	// Reset idle connections
	p.db.SetMaxIdleConns(0)
	time.Sleep(100 * time.Millisecond)
	p.db.SetMaxIdleConns(p.config.MaxIdleConns)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := p.db.PingContext(ctx); err == nil {
		p.mu.Lock()
		p.healthStatus = true
		p.mu.Unlock()
		p.logger.Info("Connection pool recovered successfully")
	}
}

// getHealthStatus returns current health status
func (p *ConnectionPool) getHealthStatus() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.healthStatus
}

// OptimizeForWorkload adjusts pool settings based on workload
func (p *ConnectionPool) OptimizeForWorkload(workloadType string) {
	switch workloadType {
	case "high_throughput":
		p.db.SetMaxOpenConns(50)
		p.db.SetMaxIdleConns(25)
		p.db.SetConnMaxLifetime(10 * time.Minute)
	case "low_latency":
		p.db.SetMaxOpenConns(30)
		p.db.SetMaxIdleConns(20)
		p.db.SetConnMaxIdleTime(1 * time.Minute)
	case "batch_processing":
		p.db.SetMaxOpenConns(10)
		p.db.SetMaxIdleConns(5)
		p.db.SetConnMaxLifetime(30 * time.Minute)
	default:
		// Keep current settings
		return
	}

	p.logger.WithField("workload_type", workloadType).Info("Pool optimized for workload")
}

// WarmUp pre-establishes connections
func (p *ConnectionPool) WarmUp(ctx context.Context, connections int) error {
	if connections > p.config.MaxOpenConns {
		connections = p.config.MaxOpenConns
	}

	var wg sync.WaitGroup
	errors := make(chan error, connections)

	for i := 0; i < connections; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, err := p.db.Conn(ctx)
			if err != nil {
				errors <- err
				return
			}
			defer func() { _ = conn.Close() }() // Best-effort close

			// Execute a simple query to warm the connection
			if err := conn.PingContext(ctx); err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		if err != nil {
			return fmt.Errorf("failed to warm up connections: %w", err)
		}
	}

	p.logger.WithField("connections", connections).Info("Connection pool warmed up")
	return nil
}

// Close gracefully closes the connection pool
func (p *ConnectionPool) Close() error {
	// Stop health monitoring
	close(p.healthCheckStop)

	// Log final stats
	stats := p.GetStats()
	p.logger.WithFields(logrus.Fields{
		"total_requests": stats.TotalRequests,
		"total_errors":   stats.TotalErrors,
		"slow_queries":   stats.SlowQueries,
		"efficiency":     stats.Efficiency,
	}).Info("Closing connection pool")

	return p.db.Close()
}

// truncateQuery truncates long queries for logging
func truncateQuery(query string, maxLen int) string {
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen] + "..."
}
