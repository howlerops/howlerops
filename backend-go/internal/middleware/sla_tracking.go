package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/sla"
)

// SLATracking middleware tracks requests for SLA monitoring
type SLATracking struct {
	monitor *sla.Monitor
	logger  *logrus.Logger
}

// NewSLATracking creates a new SLA tracking middleware
func NewSLATracking(monitor *sla.Monitor, logger *logrus.Logger) *SLATracking {
	return &SLATracking{
		monitor: monitor,
		logger:  logger,
	}
}

// Track wraps the handler to track request metrics
func (s *SLATracking) Track(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get organization ID from context
		orgID := GetCurrentOrgID(r.Context())
		if orgID == "" {
			// No organization context, skip tracking
			next.ServeHTTP(w, r)
			return
		}

		// Record start time
		startTime := time.Now()

		// Create response writer wrapper to capture status code
		wrapper := &slaResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // Default to 200
		}

		// Call next handler
		next.ServeHTTP(wrapper, r)

		// Calculate duration
		duration := time.Since(startTime)

		// Record request (async to not slow down response)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err := s.monitor.RecordRequest(
				ctx,
				orgID,
				r.URL.Path,
				r.Method,
				duration,
				wrapper.statusCode,
			); err != nil {
				s.logger.WithError(err).WithFields(logrus.Fields{
					"organization_id": orgID,
					"path":            r.URL.Path,
					"method":          r.Method,
				}).Error("Failed to record request for SLA")
			}
		}()
	})
}

// slaResponseWriter wraps http.ResponseWriter to capture status code
type slaResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *slaResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Write implements http.ResponseWriter
func (rw *slaResponseWriter) Write(b []byte) (int, error) {
	return rw.ResponseWriter.Write(b)
}
