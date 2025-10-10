package logger

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger configuration
type Config struct {
	Level      string `json:"level"`
	Format     string `json:"format"`
	Output     string `json:"output"`
	File       string `json:"file"`
	MaxSize    int    `json:"max_size"`
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"`
	Compress   bool   `json:"compress"`
}

// NewLogger creates a new logger instance
func NewLogger(config Config) (*logrus.Logger, error) {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.Level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level: %w", err)
	}
	logger.SetLevel(level)

	// Set log format
	switch strings.ToLower(config.Format) {
	case "json":
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	case "text":
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		return nil, fmt.Errorf("invalid log format: %s", config.Format)
	}

	// Set log output
	switch strings.ToLower(config.Output) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "stderr":
		logger.SetOutput(os.Stderr)
	case "file":
		if config.File == "" {
			return nil, fmt.Errorf("log file path is required when output is 'file'")
		}

		// Use lumberjack for log rotation
		lumberjackLogger := &lumberjack.Logger{
			Filename:   config.File,
			MaxSize:    config.MaxSize,    // megabytes
			MaxBackups: config.MaxBackups, // number of backups
			MaxAge:     config.MaxAge,     // days
			Compress:   config.Compress,   // compress rotated files
		}

		logger.SetOutput(lumberjackLogger)
	case "both":
		if config.File == "" {
			return nil, fmt.Errorf("log file path is required when output is 'both'")
		}

		// Use lumberjack for log rotation
		lumberjackLogger := &lumberjack.Logger{
			Filename:   config.File,
			MaxSize:    config.MaxSize,
			MaxBackups: config.MaxBackups,
			MaxAge:     config.MaxAge,
			Compress:   config.Compress,
		}

		// Write to both stdout and file
		multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
		logger.SetOutput(multiWriter)
	default:
		return nil, fmt.Errorf("invalid log output: %s", config.Output)
	}

	return logger, nil
}

// WithFields creates a logger with predefined fields
func WithFields(logger *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return logger.WithFields(logrus.Fields(fields))
}

// WithComponent creates a logger with component field
func WithComponent(logger *logrus.Logger, component string) *logrus.Entry {
	return logger.WithField("component", component)
}

// WithRequest creates a logger with request context
func WithRequest(logger *logrus.Logger, requestID string) *logrus.Entry {
	return logger.WithField("request_id", requestID)
}

// WithConnection creates a logger with connection context
func WithConnection(logger *logrus.Logger, connectionID string) *logrus.Entry {
	return logger.WithField("connection_id", connectionID)
}

// WithQuery creates a logger with query context
func WithQuery(logger *logrus.Logger, queryID string) *logrus.Entry {
	return logger.WithField("query_id", queryID)
}

// WithUser creates a logger with user context
func WithUser(logger *logrus.Logger, userID string) *logrus.Entry {
	return logger.WithField("user_id", userID)
}

// WithError creates a logger with error field
func WithError(logger *logrus.Logger, err error) *logrus.Entry {
	return logger.WithError(err)
}

// GetDefaultConfig returns default logger configuration
func GetDefaultConfig() Config {
	return Config{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
}

// GetDevelopmentConfig returns development logger configuration
func GetDevelopmentConfig() Config {
	return Config{
		Level:      "debug",
		Format:     "text",
		Output:     "stdout",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     28,
		Compress:   true,
	}
}

// GetProductionConfig returns production logger configuration
func GetProductionConfig() Config {
	return Config{
		Level:      "info",
		Format:     "json",
		Output:     "file",
		File:       "/var/log/sql-studio/app.log",
		MaxSize:    100,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
}

// ContextualLogger wraps logrus.Logger with additional context
type ContextualLogger struct {
	*logrus.Logger
	fields logrus.Fields
}

// NewContextualLogger creates a new contextual logger
func NewContextualLogger(logger *logrus.Logger) *ContextualLogger {
	return &ContextualLogger{
		Logger: logger,
		fields: make(logrus.Fields),
	}
}

// WithField adds a field to the contextual logger
func (l *ContextualLogger) WithField(key string, value interface{}) *ContextualLogger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &ContextualLogger{
		Logger: l.Logger,
		fields: newFields,
	}
}

// WithFields adds multiple fields to the contextual logger
func (l *ContextualLogger) WithFields(fields map[string]interface{}) *ContextualLogger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &ContextualLogger{
		Logger: l.Logger,
		fields: newFields,
	}
}

// Entry returns a logrus entry with all contextual fields
func (l *ContextualLogger) Entry() *logrus.Entry {
	return l.Logger.WithFields(l.fields)
}

// Debug logs a debug message with context
func (l *ContextualLogger) Debug(args ...interface{}) {
	l.Entry().Debug(args...)
}

// Debugf logs a formatted debug message with context
func (l *ContextualLogger) Debugf(format string, args ...interface{}) {
	l.Entry().Debugf(format, args...)
}

// Info logs an info message with context
func (l *ContextualLogger) Info(args ...interface{}) {
	l.Entry().Info(args...)
}

// Infof logs a formatted info message with context
func (l *ContextualLogger) Infof(format string, args ...interface{}) {
	l.Entry().Infof(format, args...)
}

// Warn logs a warning message with context
func (l *ContextualLogger) Warn(args ...interface{}) {
	l.Entry().Warn(args...)
}

// Warnf logs a formatted warning message with context
func (l *ContextualLogger) Warnf(format string, args ...interface{}) {
	l.Entry().Warnf(format, args...)
}

// Error logs an error message with context
func (l *ContextualLogger) Error(args ...interface{}) {
	l.Entry().Error(args...)
}

// Errorf logs a formatted error message with context
func (l *ContextualLogger) Errorf(format string, args ...interface{}) {
	l.Entry().Errorf(format, args...)
}

// Fatal logs a fatal message with context
func (l *ContextualLogger) Fatal(args ...interface{}) {
	l.Entry().Fatal(args...)
}

// Fatalf logs a formatted fatal message with context
func (l *ContextualLogger) Fatalf(format string, args ...interface{}) {
	l.Entry().Fatalf(format, args...)
}

// Panic logs a panic message with context
func (l *ContextualLogger) Panic(args ...interface{}) {
	l.Entry().Panic(args...)
}

// Panicf logs a formatted panic message with context
func (l *ContextualLogger) Panicf(format string, args ...interface{}) {
	l.Entry().Panicf(format, args...)
}