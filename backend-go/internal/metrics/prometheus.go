package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for SQL Studio backend
// These metrics are automatically registered with the default Prometheus registry

var (
	// ========================================================================
	// HTTP Request Metrics
	// ========================================================================

	// HTTPRequestDuration tracks the duration of HTTP requests
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestsTotal tracks the total number of HTTP requests
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestsInFlight tracks the number of HTTP requests currently being processed
	HTTPRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "http",
			Name:      "requests_in_flight",
			Help:      "Number of HTTP requests currently being processed",
		},
	)

	// HTTPRequestSize tracks the size of HTTP request bodies
	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "Size of HTTP request bodies in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8), // 100B to 100MB
		},
		[]string{"method", "endpoint"},
	)

	// HTTPResponseSize tracks the size of HTTP response bodies
	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "Size of HTTP response bodies in bytes",
			Buckets:   prometheus.ExponentialBuckets(100, 10, 8), // 100B to 100MB
		},
		[]string{"method", "endpoint", "status"},
	)

	// ========================================================================
	// Database Query Metrics
	// ========================================================================

	// DatabaseQueryDuration tracks the duration of database queries
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "query_duration_seconds",
			Help:      "Duration of database queries in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"operation", "status"},
	)

	// DatabaseQueriesTotal tracks the total number of database queries
	DatabaseQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "queries_total",
			Help:      "Total number of database queries executed",
		},
		[]string{"operation", "status"},
	)

	// DatabaseConnectionsActive tracks the number of active database connections
	DatabaseConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "connections_active",
			Help:      "Number of active database connections",
		},
		[]string{"pool"},
	)

	// DatabaseConnectionsMax tracks the maximum number of database connections
	DatabaseConnectionsMax = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "connections_max",
			Help:      "Maximum number of database connections",
		},
		[]string{"pool"},
	)

	// DatabaseConnectionsOpened tracks the total number of opened connections
	DatabaseConnectionsOpened = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "connections_opened_total",
			Help:      "Total number of database connections opened",
		},
		[]string{"pool"},
	)

	// DatabaseConnectionsClosed tracks the total number of closed connections
	DatabaseConnectionsClosed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "connections_closed_total",
			Help:      "Total number of database connections closed",
		},
		[]string{"pool"},
	)

	// DatabaseRowsReturned tracks the number of rows returned by queries
	DatabaseRowsReturned = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "database",
			Name:      "rows_returned",
			Help:      "Number of rows returned by database queries",
			Buckets:   []float64{1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"operation"},
	)

	// ========================================================================
	// Authentication Metrics
	// ========================================================================

	// AuthAttemptsTotal tracks authentication attempts
	AuthAttemptsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "auth",
			Name:      "attempts_total",
			Help:      "Total number of authentication attempts",
		},
		[]string{"method", "status"}, // method: email, oauth; status: success, failed
	)

	// AuthLockoutsTotal tracks account lockouts
	AuthLockoutsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "auth",
			Name:      "lockouts_total",
			Help:      "Total number of account lockouts",
		},
	)

	// AuthSessionsActive tracks active sessions
	AuthSessionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "auth",
			Name:      "sessions_active",
			Help:      "Number of active user sessions",
		},
	)

	// AuthTokensIssued tracks JWT tokens issued
	AuthTokensIssued = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "auth",
			Name:      "tokens_issued_total",
			Help:      "Total number of JWT tokens issued",
		},
		[]string{"type"}, // type: access, refresh
	)

	// ========================================================================
	// Sync Service Metrics
	// ========================================================================

	// SyncOperationsTotal tracks sync operations
	SyncOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "sync",
			Name:      "operations_total",
			Help:      "Total number of sync operations",
		},
		[]string{"operation", "status"}, // operation: push, pull, merge; status: success, failed
	)

	// SyncDuration tracks sync operation duration
	SyncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "sync",
			Name:      "duration_seconds",
			Help:      "Duration of sync operations in seconds",
			Buckets:   []float64{.1, .25, .5, 1, 2.5, 5, 10, 30, 60},
		},
		[]string{"operation"},
	)

	// SyncConflictsTotal tracks sync conflicts
	SyncConflictsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "sync",
			Name:      "conflicts_total",
			Help:      "Total number of sync conflicts detected",
		},
		[]string{"resolution"}, // resolution: last_write_wins, keep_both, user_choice
	)

	// SyncDataSize tracks the size of synced data
	SyncDataSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "sync",
			Name:      "data_size_bytes",
			Help:      "Size of synced data in bytes",
			Buckets:   prometheus.ExponentialBuckets(1024, 10, 7), // 1KB to 10MB
		},
		[]string{"operation"},
	)

	// SyncLagSeconds tracks sync lag
	SyncLagSeconds = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "sync",
			Name:      "lag_seconds",
			Help:      "Sync lag in seconds",
		},
		[]string{"user_id"},
	)

	// ========================================================================
	// Business Metrics
	// ========================================================================

	// UserRegistrationsTotal tracks user registrations
	UserRegistrationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "business",
			Name:      "user_registrations_total",
			Help:      "Total number of user registrations",
		},
	)

	// OrganizationsCreatedTotal tracks organization creation
	OrganizationsCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "business",
			Name:      "organizations_created_total",
			Help:      "Total number of organizations created",
		},
	)

	// ActiveUsers tracks currently active users
	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "business",
			Name:      "active_users",
			Help:      "Number of currently active users",
		},
	)

	// FeatureUsageTotal tracks feature usage
	FeatureUsageTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "business",
			Name:      "feature_usage_total",
			Help:      "Total number of times features are used",
		},
		[]string{"feature"},
	)

	// ========================================================================
	// Cache Metrics
	// ========================================================================

	// CacheHitsTotal tracks cache hits
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "cache",
			Name:      "hits_total",
			Help:      "Total number of cache hits",
		},
		[]string{"cache_name"},
	)

	// CacheMissesTotal tracks cache misses
	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "cache",
			Name:      "misses_total",
			Help:      "Total number of cache misses",
		},
		[]string{"cache_name"},
	)

	// CacheSize tracks cache size
	CacheSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "sql_studio",
			Subsystem: "cache",
			Name:      "size_bytes",
			Help:      "Current cache size in bytes",
		},
		[]string{"cache_name"},
	)

	// ========================================================================
	// Background Job Metrics
	// ========================================================================

	// BackgroundJobDuration tracks background job duration
	BackgroundJobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "sql_studio",
			Subsystem: "jobs",
			Name:      "duration_seconds",
			Help:      "Duration of background jobs in seconds",
			Buckets:   []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600},
		},
		[]string{"job_name"},
	)

	// BackgroundJobsTotal tracks background job executions
	BackgroundJobsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "sql_studio",
			Subsystem: "jobs",
			Name:      "executions_total",
			Help:      "Total number of background job executions",
		},
		[]string{"job_name", "status"},
	)
)

// Helper functions for common metric operations

// RecordHTTPRequest records metrics for an HTTP request
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration, requestSize, responseSize int64) {
	HTTPRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration.Seconds())
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	HTTPResponseSize.WithLabelValues(method, endpoint, status).Observe(float64(responseSize))
}

// RecordDatabaseQuery records metrics for a database query
func RecordDatabaseQuery(operation, status string, duration time.Duration, rowCount int64) {
	DatabaseQueryDuration.WithLabelValues(operation, status).Observe(duration.Seconds())
	DatabaseQueriesTotal.WithLabelValues(operation, status).Inc()
	if rowCount >= 0 {
		DatabaseRowsReturned.WithLabelValues(operation).Observe(float64(rowCount))
	}
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(method, status string) {
	AuthAttemptsTotal.WithLabelValues(method, status).Inc()
}

// RecordSyncOperation records a sync operation
func RecordSyncOperation(operation, status string, duration time.Duration, dataSize int64) {
	SyncOperationsTotal.WithLabelValues(operation, status).Inc()
	SyncDuration.WithLabelValues(operation).Observe(duration.Seconds())
	if dataSize > 0 {
		SyncDataSize.WithLabelValues(operation).Observe(float64(dataSize))
	}
}

// UpdateConnectionPoolMetrics updates database connection pool metrics
func UpdateConnectionPoolMetrics(pool string, active, max int) {
	DatabaseConnectionsActive.WithLabelValues(pool).Set(float64(active))
	DatabaseConnectionsMax.WithLabelValues(pool).Set(float64(max))
}

// RecordCacheAccess records cache hit or miss
func RecordCacheAccess(cacheName string, hit bool) {
	if hit {
		CacheHitsTotal.WithLabelValues(cacheName).Inc()
	} else {
		CacheMissesTotal.WithLabelValues(cacheName).Inc()
	}
}

// RecordBackgroundJob records background job execution
func RecordBackgroundJob(jobName, status string, duration time.Duration) {
	BackgroundJobDuration.WithLabelValues(jobName).Observe(duration.Seconds())
	BackgroundJobsTotal.WithLabelValues(jobName, status).Inc()
}
