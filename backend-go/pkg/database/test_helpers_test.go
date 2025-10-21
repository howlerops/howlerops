package database_test

import (
	"github.com/sirupsen/logrus"
)

// newTestLogger creates a logger for testing with reduced output.
// This is a shared helper used across all database test files.
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}
