package middleware

import (
	"bufio"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sql-studio/backend-go/internal/metrics"
)

// responseWriter wraps http.ResponseWriter to capture status code and response size
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Hijack implements http.Hijacker interface for WebSocket support
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return h.Hijack()
}

// MetricsMiddleware collects HTTP metrics for Prometheus
func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status and size
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default status code
		}

		// Increment in-flight requests
		metrics.HTTPRequestsInFlight.Inc()
		defer metrics.HTTPRequestsInFlight.Dec()

		// Process request
		next.ServeHTTP(wrapped, r)

		// Calculate duration
		duration := time.Since(start)

		// Get route pattern (e.g., /api/users/:id instead of /api/users/123)
		endpoint := getRoutePattern(r)

		// Get request size (approximate from Content-Length header)
		requestSize := int64(0)
		if contentLength := r.Header.Get("Content-Length"); contentLength != "" {
			if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
				requestSize = size
			}
		}

		// Record metrics
		status := strconv.Itoa(wrapped.statusCode)
		metrics.RecordHTTPRequest(
			r.Method,
			endpoint,
			status,
			duration,
			requestSize,
			wrapped.bytesWritten,
		)
	})
}

// getRoutePattern extracts the route pattern from the request
// For Chi router, this gives us the route template (e.g., /api/users/:id)
func getRoutePattern(r *http.Request) string {
	// Try to get Chi route pattern
	rctx := chi.RouteContext(r.Context())
	if rctx != nil && rctx.RoutePattern() != "" {
		return rctx.RoutePattern()
	}

	// Fallback to path without query parameters
	path := r.URL.Path

	// Normalize common patterns
	path = normalizeEndpoint(path)

	return path
}

// normalizeEndpoint normalizes endpoint paths for better metric aggregation
// This prevents high cardinality issues from dynamic path segments
func normalizeEndpoint(path string) string {
	// Remove trailing slash
	path = strings.TrimSuffix(path, "/")

	// Handle common ID patterns in URLs
	// Replace UUIDs: /users/550e8400-e29b-41d4-a716-446655440000 -> /users/:id
	path = replaceUUIDs(path)

	// Replace numeric IDs: /users/123 -> /users/:id
	path = replaceNumericIDs(path)

	// If still empty after trimming, use root
	if path == "" {
		path = "/"
	}

	return path
}

// replaceUUIDs replaces UUID patterns in paths with :id
func replaceUUIDs(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Check if part looks like a UUID
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			// Validate UUID format (rough check)
			if isValidUUIDFormat(part) {
				parts[i] = ":id"
			}
		}
	}
	return strings.Join(parts, "/")
}

// replaceNumericIDs replaces numeric IDs in paths with :id
func replaceNumericIDs(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		// Skip the first part (empty string before first /)
		if i == 0 || part == "" {
			continue
		}

		// Check if part is purely numeric
		if isNumeric(part) {
			parts[i] = ":id"
		}
	}
	return strings.Join(parts, "/")
}

// isValidUUIDFormat checks if a string matches UUID format
func isValidUUIDFormat(s string) bool {
	if len(s) != 36 {
		return false
	}
	// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
				return false
			}
		}
	}
	return true
}

// isNumeric checks if a string contains only digits
func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// MetricsSummary returns a summary of current metrics (for debugging)
type MetricsSummary struct {
	RequestsInFlight int    `json:"requests_in_flight"`
	TotalRequests    int64  `json:"total_requests"`
	Uptime           string `json:"uptime"`
}

// GetMetricsSummary returns a summary of current metrics
// This is useful for health check endpoints and debugging
func GetMetricsSummary() *MetricsSummary {
	// Note: This is a simplified version. In production, you'd collect
	// actual values from the metrics collectors.
	return &MetricsSummary{
		RequestsInFlight: 0, // Would need to read from metrics.HTTPRequestsInFlight
		TotalRequests:    0, // Would need to read from metrics.HTTPRequestsTotal
		Uptime:           time.Since(startTime).String(),
	}
}

var startTime = time.Now()
