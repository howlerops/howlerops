// Package testutil provides shared testing utilities and fixtures
package testutil

import (
	"database/sql"
	"testing"

	"github.com/jbeck018/howlerops/backend-go/internal/config"
	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	_ "github.com/mattn/go-sqlite3"
)

// NewTestConfig creates a test configuration with sensible defaults
func NewTestConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			HTTPPort:     8080,
			GRPCPort:     9090,
			TLSEnabled:   false,
			ReadTimeout:  30,
			WriteTimeout: 30,
			IdleTimeout:  60,
		},
		Auth: config.AuthConfig{
			JWTSecret: "test-secret-key-for-testing-only",
		},
		Security: config.SecurityConfig{
			RateLimitRPS:   100,
			RateLimitBurst: 200,
			RequestTimeout: 30,
		},
	}
}

// NewTestDBConfig creates a test database configuration
func NewTestDBConfig() *database.ConnectionConfig {
	return &database.ConnectionConfig{
		Host:     "localhost",
		Port:     3306,
		Database: "test",
		Username: "test",
		Password: "test",
		SSLMode:  "disable",
	}
}

// NewTestSQLiteDB creates an in-memory SQLite database for testing
func NewTestSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create in-memory SQLite database: %v", err)
	}

	// Ensure cleanup
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Logf("failed to close test database: %v", err)
		}
	})

	return db
}

// SetupTestSchema creates a test database schema for testing
func SetupTestSchema(t *testing.T, db *sql.DB) {
	t.Helper()

	schema := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);

		CREATE TABLE IF NOT EXISTS posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		);

		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (post_id) REFERENCES posts(id),
			FOREIGN KEY (user_id) REFERENCES users(id)
		);
	`

	_, err := db.Exec(schema)
	if err != nil {
		t.Fatalf("failed to setup test schema: %v", err)
	}
}

// SeedTestData inserts test data into the database
func SeedTestData(t *testing.T, db *sql.DB) {
	t.Helper()

	data := `
		INSERT INTO users (username, email) VALUES
			('alice', 'alice@example.com'),
			('bob', 'bob@example.com'),
			('charlie', 'charlie@example.com');

		INSERT INTO posts (user_id, title, content) VALUES
			(1, 'First Post', 'This is Alice''s first post'),
			(1, 'Second Post', 'Alice posts again'),
			(2, 'Bob''s Thoughts', 'Some thoughts from Bob');

		INSERT INTO comments (post_id, user_id, content) VALUES
			(1, 2, 'Nice post!'),
			(1, 3, 'I agree'),
			(2, 3, 'Interesting perspective');
	`

	_, err := db.Exec(data)
	if err != nil {
		t.Fatalf("failed to seed test data: %v", err)
	}
}
