package monitoring

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	httpDuration *prometheus.HistogramVec
	httpRequests *prometheus.CounterVec
	httpInFlight prometheus.Gauge
	httpSize     *prometheus.HistogramVec
}

// HTTPMetrics represents metrics for a single HTTP request
type HTTPMetrics struct {
	Path       string
	Method     string
	StatusCode int
	Duration   time.Duration
	Size       int
}

// MonitoringMiddleware provides HTTP monitoring capabilities
type MonitoringMiddleware struct {
	metrics       *Metrics
	logger        *logrus.Logger
	slowThreshold time.Duration
	collector     *MetricsCollector
}

// MetricsCollector collects and stores metrics
type MetricsCollector struct {
	mu          sync.RWMutex
	requests    []RequestMetric
	maxRequests int
}

// RequestMetric represents a single request metric
type RequestMetric struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	Duration     int64     `json:"duration_ms"`
	Size         int       `json:"size_bytes"`
	UserID       string    `json:"user_id,omitempty"`
	IP           string    `json:"ip"`
	UserAgent    string    `json:"user_agent"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// responseWriter wraps http.ResponseWriter to capture status code and size
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(data)
	rw.size += size
	return size, err
}

// NewMonitoringMiddleware creates a new monitoring middleware
func NewMonitoringMiddleware(logger *logrus.Logger) *MonitoringMiddleware {
	// Create Prometheus metrics
	metrics := &Metrics{
		httpDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "Duration of HTTP requests in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		),
		httpRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		httpInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "Number of HTTP requests currently being processed",
			},
		),
		httpSize: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_response_size_bytes",
				Help:    "Size of HTTP responses in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000},
			},
			[]string{"method", "path"},
		),
	}

	// Register metrics
	prometheus.MustRegister(metrics.httpDuration)
	prometheus.MustRegister(metrics.httpRequests)
	prometheus.MustRegister(metrics.httpInFlight)
	prometheus.MustRegister(metrics.httpSize)

	return &MonitoringMiddleware{
		metrics:       metrics,
		logger:        logger,
		slowThreshold: 500 * time.Millisecond, // Default 500ms
		collector: &MetricsCollector{
			requests:    make([]RequestMetric, 0, 1000),
			maxRequests: 1000,
		},
	}
}

// Middleware returns the HTTP middleware handler
func (m *MonitoringMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip monitoring for metrics endpoint
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		path := m.normalizePath(r.URL.Path)

		// Track in-flight requests
		m.metrics.httpInFlight.Inc()
		defer m.metrics.httpInFlight.Dec()

		// Wrap response writer
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			size:           0,
		}

		// Serve the request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Extract user info from context
		userID := ""
		if uid := r.Context().Value("user_id"); uid != nil {
			userID = uid.(string)
		}

		// Record metrics
		statusStr := strconv.Itoa(wrapped.statusCode)
		m.metrics.httpDuration.WithLabelValues(r.Method, path, statusStr).Observe(duration.Seconds())
		m.metrics.httpRequests.WithLabelValues(r.Method, path, statusStr).Inc()
		m.metrics.httpSize.WithLabelValues(r.Method, path).Observe(float64(wrapped.size))

		// Create request metric
		metric := RequestMetric{
			Timestamp:  start,
			Method:     r.Method,
			Path:       path,
			StatusCode: wrapped.statusCode,
			Duration:   duration.Milliseconds(),
			Size:       wrapped.size,
			UserID:     userID,
			IP:         m.getClientIP(r),
			UserAgent:  r.UserAgent(),
		}

		// Store metric
		m.collector.Add(metric)

		// Log slow requests
		if duration > m.slowThreshold {
			m.logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        path,
				"status":      wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"size_bytes":  wrapped.size,
				"user_id":     userID,
				"ip":          metric.IP,
			}).Warn("Slow HTTP request detected")
		}

		// Log errors
		if wrapped.statusCode >= 500 {
			m.logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        path,
				"status":      wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"user_id":     userID,
			}).Error("HTTP request failed with server error")
		}
	})
}

// SetSlowThreshold sets the threshold for slow request detection
func (m *MonitoringMiddleware) SetSlowThreshold(threshold time.Duration) {
	m.slowThreshold = threshold
}

// normalizePath normalizes the request path for grouping
func (m *MonitoringMiddleware) normalizePath(path string) string {
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		path = "/"
	}

	// Replace IDs in common patterns
	// Example: /api/users/123 -> /api/users/:id
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Check if part looks like an ID (UUID, numeric, etc.)
		if m.looksLikeID(part) && i > 0 {
			parts[i] = ":id"
		}
	}

	return strings.Join(parts, "/")
}

// looksLikeID checks if a path segment looks like an ID
func (m *MonitoringMiddleware) looksLikeID(segment string) bool {
	// Check for UUID pattern
	if len(segment) == 36 && strings.Count(segment, "-") == 4 {
		return true
	}

	// Check if all digits
	if segment != "" {
		for _, r := range segment {
			if r < '0' || r > '9' {
				return false
			}
		}
		return true
	}

	return false
}

// getClientIP extracts the client IP from the request
func (m *MonitoringMiddleware) getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		return ip[:idx]
	}

	return ip
}

// MetricsCollector methods

// Add adds a new request metric
func (c *MetricsCollector) Add(metric RequestMetric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requests = append(c.requests, metric)

	// Keep only the last N requests
	if len(c.requests) > c.maxRequests {
		c.requests = c.requests[len(c.requests)-c.maxRequests:]
	}
}

// GetRecent returns recent request metrics
func (c *MetricsCollector) GetRecent(count int) []RequestMetric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if count > len(c.requests) {
		count = len(c.requests)
	}

	if count == 0 {
		return []RequestMetric{}
	}

	start := len(c.requests) - count
	result := make([]RequestMetric, count)
	copy(result, c.requests[start:])

	return result
}

// GetStats returns aggregated statistics
func (c *MetricsCollector) GetStats(duration time.Duration) map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := time.Now().Add(-duration)
	var totalRequests int
	var totalDuration int64
	var totalSize int
	statusCounts := make(map[int]int)
	methodCounts := make(map[string]int)
	var slowRequests int
	var errors int

	for _, req := range c.requests {
		if req.Timestamp.After(cutoff) {
			totalRequests++
			totalDuration += req.Duration
			totalSize += req.Size
			statusCounts[req.StatusCode]++
			methodCounts[req.Method]++

			if req.Duration > 500 { // 500ms threshold
				slowRequests++
			}

			if req.StatusCode >= 500 {
				errors++
			}
		}
	}

	avgDuration := float64(0)
	avgSize := float64(0)
	if totalRequests > 0 {
		avgDuration = float64(totalDuration) / float64(totalRequests)
		avgSize = float64(totalSize) / float64(totalRequests)
	}

	return map[string]interface{}{
		"total_requests":    totalRequests,
		"avg_duration_ms":   avgDuration,
		"avg_size_bytes":    avgSize,
		"slow_requests":     slowRequests,
		"errors":            errors,
		"status_breakdown":  statusCounts,
		"method_breakdown":  methodCounts,
		"requests_per_sec":  float64(totalRequests) / duration.Seconds(),
		"error_rate":        float64(errors) / float64(totalRequests) * 100,
		"slow_request_rate": float64(slowRequests) / float64(totalRequests) * 100,
	}
}

// HTTP Handlers

// MetricsHandler serves Prometheus metrics
func (m *MonitoringMiddleware) MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// RecentRequestsHandler returns recent request metrics
func (m *MonitoringMiddleware) RecentRequestsHandler(w http.ResponseWriter, r *http.Request) {
	count := 100
	if c := r.URL.Query().Get("count"); c != "" {
		if parsed, err := strconv.Atoi(c); err == nil && parsed > 0 {
			count = parsed
		}
	}

	recent := m.collector.GetRecent(count)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recent)
}

// StatsHandler returns aggregated statistics
func (m *MonitoringMiddleware) StatsHandler(w http.ResponseWriter, r *http.Request) {
	duration := 5 * time.Minute
	if d := r.URL.Query().Get("duration"); d != "" {
		if parsed, err := time.ParseDuration(d); err == nil {
			duration = parsed
		}
	}

	stats := m.collector.GetStats(duration)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// HealthHandler returns health status
func (m *MonitoringMiddleware) HealthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status": "healthy",
		"timestamp": time.Now().Unix(),
		"metrics": m.collector.GetStats(1 * time.Minute),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// RegisterRoutes registers monitoring HTTP routes
func (m *MonitoringMiddleware) RegisterRoutes(router *mux.Router) {
	router.Handle("/metrics", m.MetricsHandler())
	router.HandleFunc("/api/monitoring/recent", m.RecentRequestsHandler).Methods("GET")
	router.HandleFunc("/api/monitoring/stats", m.StatsHandler).Methods("GET")
	router.HandleFunc("/health", m.HealthHandler).Methods("GET")
}

// RequestLoggingMiddleware logs all HTTP requests
func RequestLoggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create wrapped response writer
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Process request
			next.ServeHTTP(wrapped, r)

			// Log request
			duration := time.Since(start)
			logger.WithFields(logrus.Fields{
				"method":      r.Method,
				"path":        r.URL.Path,
				"status":      wrapped.statusCode,
				"duration_ms": duration.Milliseconds(),
				"ip":          r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			}).Info("HTTP request processed")
		})
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithFields(logrus.Fields{
						"method": r.Method,
						"path":   r.URL.Path,
						"error":  err,
					}).Error("Panic recovered in HTTP handler")

					// Return 500 error
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Internal server error",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		})
	}
}