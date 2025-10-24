package logger

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// ContextKey type for context keys
type ContextKey string

const (
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// SessionIDKey is the context key for session ID
	SessionIDKey ContextKey = "session_id"
)

// StructuredLogger wraps logrus.Logger with context-aware structured logging
type StructuredLogger struct {
	*logrus.Logger
}

// NewStructuredLogger creates a new structured logger
func NewStructuredLogger(logger *logrus.Logger) *StructuredLogger {
	return &StructuredLogger{
		Logger: logger,
	}
}

// WithContext creates a logger with fields from context
func (l *StructuredLogger) WithContext(ctx context.Context) *logrus.Entry {
	fields := logrus.Fields{}

	// Extract common fields from context
	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		fields["trace_id"] = traceID
	}
	if userID := ctx.Value(UserIDKey); userID != nil {
		fields["user_id"] = userID
	}
	if requestID := ctx.Value(RequestIDKey); requestID != nil {
		fields["request_id"] = requestID
	}
	if sessionID := ctx.Value(SessionIDKey); sessionID != nil {
		fields["session_id"] = sessionID
	}

	return l.WithFields(fields)
}

// LogRequest logs HTTP request details
func (l *StructuredLogger) LogRequest(ctx context.Context, method, path string, duration time.Duration, status int) {
	l.WithContext(ctx).WithFields(logrus.Fields{
		"method":      method,
		"path":        path,
		"duration_ms": duration.Milliseconds(),
		"status":      status,
		"type":        "http_request",
	}).Info("HTTP request completed")
}

// LogDatabaseQuery logs database query details
func (l *StructuredLogger) LogDatabaseQuery(ctx context.Context, query string, duration time.Duration, rowCount int64, err error) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"query":       query,
		"duration_ms": duration.Milliseconds(),
		"row_count":   rowCount,
		"type":        "database_query",
	})

	if err != nil {
		entry.WithError(err).Error("Database query failed")
	} else {
		entry.Info("Database query completed")
	}
}

// LogAuthentication logs authentication events
func (l *StructuredLogger) LogAuthentication(ctx context.Context, email string, success bool, reason string) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"email":   email,
		"success": success,
		"type":    "authentication",
	})

	if reason != "" {
		entry = entry.WithField("reason", reason)
	}

	if success {
		entry.Info("Authentication successful")
	} else {
		entry.Warn("Authentication failed")
	}
}

// LogSync logs sync operation details
func (l *StructuredLogger) LogSync(ctx context.Context, operation string, dataSize int64, duration time.Duration, err error) {
	entry := l.WithContext(ctx).WithFields(logrus.Fields{
		"operation":   operation,
		"data_size":   dataSize,
		"duration_ms": duration.Milliseconds(),
		"type":        "sync_operation",
	})

	if err != nil {
		entry.WithError(err).Error("Sync operation failed")
	} else {
		entry.Info("Sync operation completed")
	}
}

// LogSecurityEvent logs security-related events
func (l *StructuredLogger) LogSecurityEvent(ctx context.Context, eventType string, severity string, details map[string]interface{}) {
	fields := logrus.Fields{
		"event_type": eventType,
		"severity":   severity,
		"type":       "security_event",
	}

	for k, v := range details {
		fields[k] = v
	}

	entry := l.WithContext(ctx).WithFields(fields)

	switch severity {
	case "critical":
		entry.Error("Critical security event")
	case "high":
		entry.Warn("High severity security event")
	case "medium":
		entry.Warn("Medium severity security event")
	default:
		entry.Info("Security event")
	}
}

// LogBusinessEvent logs business metrics and events
func (l *StructuredLogger) LogBusinessEvent(ctx context.Context, event string, details map[string]interface{}) {
	fields := logrus.Fields{
		"event": event,
		"type":  "business_event",
	}

	for k, v := range details {
		fields[k] = v
	}

	l.WithContext(ctx).WithFields(fields).Info("Business event")
}

// LogError logs errors with context
func (l *StructuredLogger) LogError(ctx context.Context, err error, message string, details map[string]interface{}) {
	fields := logrus.Fields{
		"type": "error",
	}

	for k, v := range details {
		fields[k] = v
	}

	l.WithContext(ctx).WithFields(fields).WithError(err).Error(message)
}

// LogPerformance logs performance metrics
func (l *StructuredLogger) LogPerformance(ctx context.Context, operation string, duration time.Duration, details map[string]interface{}) {
	fields := logrus.Fields{
		"operation":   operation,
		"duration_ms": duration.Milliseconds(),
		"type":        "performance",
	}

	for k, v := range details {
		fields[k] = v
	}

	entry := l.WithContext(ctx).WithFields(fields)

	// Log as warning if operation is slow
	if duration > 5*time.Second {
		entry.Warn("Slow operation detected")
	} else {
		entry.Debug("Performance metric")
	}
}

// SanitizeForLogging sanitizes sensitive data for logging
func SanitizeForLogging(data map[string]interface{}) map[string]interface{} {
	sanitized := make(map[string]interface{})

	sensitiveFields := map[string]bool{
		"password":     true,
		"token":        true,
		"api_key":      true,
		"secret":       true,
		"private_key":  true,
		"access_token": true,
		"refresh_token": true,
		"jwt":          true,
	}

	for k, v := range data {
		if sensitiveFields[k] {
			sanitized[k] = "REDACTED"
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}

// Example usage:
/*
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// Add trace ID to context
	traceID := generateTraceID()
	ctx = context.WithValue(ctx, logger.TraceIDKey, traceID)

	// Process request
	// ...

	// Log request
	logger.LogRequest(ctx, r.Method, r.URL.Path, time.Since(start), 200)
}
*/
