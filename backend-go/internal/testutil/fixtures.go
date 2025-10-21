// Package testutil provides shared testing utilities and fixtures
package testutil

import (
\t"database/sql"
\t"testing"

\t_ "github.com/mattn/go-sqlite3"
\t"github.com/sql-studio/backend-go/internal/config"
\t"github.com/sql-studio/backend-go/pkg/database"
)

// NewTestConfig creates a test configuration with sensible defaults
func NewTestConfig() *config.Config {
\treturn &config.Config{
\t\tServer: config.ServerConfig{
\t\t\tHTTPPort:     8080,
\t\t\tGRPCPort:     9090,
\t\t\tTLSEnabled:   false,
\t\t\tReadTimeout:  30,
\t\t\tWriteTimeout: 30,
\t\t\tIdleTimeout:  60,
\t\t},
\t\tAuth: config.AuthConfig{
\t\t\tJWTSecret: "test-secret-key-for-testing-only",
\t\t\tEnabled:   true,
\t\t},
\t\tSecurity: config.SecurityConfig{
\t\t\tRateLimitRPS:   100,
\t\t\tRateLimitBurst: 200,
\t\t\tRequestTimeout: 30,
\t\t},
\t}
}

// NewTestDBConfig creates a test database configuration
func NewTestDBConfig() *database.Config {
\treturn &database.Config{
\t\tHost:     "localhost",
\t\tPort:     3306,
\t\tDatabase: "test",
\t\tUsername: "test",
\t\tPassword: "test",
\t\tSSLMode:  "disable",
\t}
}

// NewTestSQLiteDB creates an in-memory SQLite database for testing
func NewTestSQLiteDB(t *testing.T) *sql.DB {
\tt.Helper()

\tdb, err := sql.Open("sqlite3", ":memory:")
\tif err != nil {
\t\tt.Fatalf("failed to create in-memory SQLite database: %v", err)
\t}

\t// Ensure cleanup
\tt.Cleanup(func() {
\t\tif err := db.Close(); err != nil {
\t\t\tt.Logf("failed to close test database: %v", err)
\t\t}
\t})

\treturn db
}

// SetupTestSchema creates a test database schema for testing
func SetupTestSchema(t *testing.T, db *sql.DB) {
\tt.Helper()

\tschema := `
\t\tCREATE TABLE IF NOT EXISTS users (
\t\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,
\t\t\tusername TEXT NOT NULL UNIQUE,
\t\t\temail TEXT NOT NULL UNIQUE,
\t\t\tcreated_at DATETIME DEFAULT CURRENT_TIMESTAMP
\t\t);

\t\tCREATE TABLE IF NOT EXISTS posts (
\t\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,
\t\t\tuser_id INTEGER NOT NULL,
\t\t\ttitle TEXT NOT NULL,
\t\t\tcontent TEXT,
\t\t\tcreated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
\t\t\tFOREIGN KEY (user_id) REFERENCES users(id)
\t\t);

\t\tCREATE TABLE IF NOT EXISTS comments (
\t\t\tid INTEGER PRIMARY KEY AUTOINCREMENT,
\t\t\tpost_id INTEGER NOT NULL,
\t\t\tuser_id INTEGER NOT NULL,
\t\t\tcontent TEXT NOT NULL,
\t\t\tcreated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
\t\t\tFOREIGN KEY (post_id) REFERENCES posts(id),
\t\t\tFOREIGN KEY (user_id) REFERENCES users(id)
\t\t);
\t`

\t_, err := db.Exec(schema)
\tif err != nil {
\t\tt.Fatalf("failed to setup test schema: %v", err)
\t}
}

// SeedTestData inserts test data into the database
func SeedTestData(t *testing.T, db *sql.DB) {
\tt.Helper()

\tdata := `
\t\tINSERT INTO users (username, email) VALUES
\t\t\t('alice', 'alice@example.com'),
\t\t\t('bob', 'bob@example.com'),
\t\t\t('charlie', 'charlie@example.com');

\t\tINSERT INTO posts (user_id, title, content) VALUES
\t\t\t(1, 'First Post', 'This is Alice''s first post'),
\t\t\t(1, 'Second Post', 'Alice posts again'),
\t\t\t(2, 'Bob''s Thoughts', 'Some thoughts from Bob');

\t\tINSERT INTO comments (post_id, user_id, content) VALUES
\t\t\t(1, 2, 'Nice post!'),
\t\t\t(1, 3, 'I agree'),
\t\t\t(2, 3, 'Interesting perspective');
\t`

\t_, err := db.Exec(data)
\tif err != nil {
\t\tt.Fatalf("failed to seed test data: %v", err)
\t}
}
