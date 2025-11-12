package database_test

import (
	"context"
	"net"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// newTestLogger creates a logger for testing with reduced output.
// This is a shared helper used across all database test files.
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}

// isServiceAvailable checks if a service is available at the given address.
// Returns true if the service is reachable within the timeout.
func isServiceAvailable(address string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// shouldSkipMongoDBTests determines if MongoDB tests should be skipped.
// Tests are skipped in the following cases:
// 1. Running with -short flag (CI environment)
// 2. SKIP_MONGODB_TESTS environment variable is set
// 3. MongoDB is not available on localhost:27017
func shouldSkipMongoDBTests() (bool, string) {
	if os.Getenv("SKIP_MONGODB_TESTS") != "" {
		return true, "SKIP_MONGODB_TESTS environment variable is set"
	}

	// Quick check if MongoDB is available
	if !isServiceAvailable("localhost:27017", 2*time.Second) {
		return true, "MongoDB not available on localhost:27017 (connection timeout)"
	}

	return false, ""
}

// shouldSkipClickHouseTests determines if ClickHouse tests should be skipped.
// Tests are skipped in the following cases:
// 1. Running with -short flag (CI environment)
// 2. SKIP_CLICKHOUSE_TESTS environment variable is set
// 3. ClickHouse is not available on localhost:9000
func shouldSkipClickHouseTests() (bool, string) {
	if os.Getenv("SKIP_CLICKHOUSE_TESTS") != "" {
		return true, "SKIP_CLICKHOUSE_TESTS environment variable is set"
	}

	// Quick check if ClickHouse is available
	if !isServiceAvailable("localhost:9000", 2*time.Second) {
		return true, "ClickHouse not available on localhost:9000 (connection timeout)"
	}

	return false, ""
}

// requireMongoDB skips the test if MongoDB is not available.
// Should be called at the beginning of tests that require MongoDB.
func requireMongoDB(t interface{ Skip(args ...interface{}) }) {
	if skip, reason := shouldSkipMongoDBTests(); skip {
		t.Skip("Skipping MongoDB test: " + reason)
	}
}

// requireClickHouse skips the test if ClickHouse is not available.
// Should be called at the beginning of tests that require ClickHouse.
func requireClickHouse(t interface{ Skip(args ...interface{}) }) {
	if skip, reason := shouldSkipClickHouseTests(); skip {
		t.Skip("Skipping ClickHouse test: " + reason)
	}
}

// skipIfMongoDBUnavailable is a helper that skips tests when MongoDB isn't available.
// This is used for tests that attempt connection and should skip on failure.
func skipIfMongoDBUnavailable(t interface{ Skip(args ...interface{}) }, err error) {
	if err != nil {
		t.Skip("Skipping test: MongoDB not available: " + err.Error())
	}
}

// skipIfClickHouseUnavailable is a helper that skips tests when ClickHouse isn't available.
// This is used for tests that attempt connection and should skip on failure.
func skipIfClickHouseUnavailable(t interface{ Skip(args ...interface{}) }, err error) {
	if err != nil {
		t.Skip("Skipping test: ClickHouse not available: " + err.Error())
	}
}

// For backward compatibility with existing tests
func skipIfNoMongoDB(t interface {
	Helper()
	Skipf(format string, args ...interface{})
}, db interface {
	Ping(ctx context.Context) error
}) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.Ping(ctx); err != nil {
		t.Skipf("Skipping test: MongoDB not available: %v", err)
	}
}
