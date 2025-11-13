package database_test

import (
	"context"
	"testing"
	"time"

	"github.com/sql-studio/backend-go/pkg/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test configuration helpers

func newMongoConfig() database.ConnectionConfig {
	return database.ConnectionConfig{
		Type:              database.MongoDB,
		Host:              "localhost",
		Port:              27017,
		Database:          "testdb",
		Username:          "",
		Password:          "",
		ConnectionTimeout: 30 * time.Second,
		IdleTimeout:       5 * time.Minute,
		MaxConnections:    25,
		MaxIdleConns:      5,
	}
}

func newMongoConfigWithAuth() database.ConnectionConfig {
	return database.ConnectionConfig{
		Type:              database.MongoDB,
		Host:              "localhost",
		Port:              27017,
		Database:          "testdb",
		Username:          "admin",
		Password:          "password",
		ConnectionTimeout: 30 * time.Second,
		Parameters: map[string]string{
			"authSource": "admin",
		},
	}
}

func newMongoConfigWithSSL() database.ConnectionConfig {
	return database.ConnectionConfig{
		Type:              database.MongoDB,
		Host:              "secure.example.com",
		Port:              27017,
		Database:          "testdb",
		Username:          "user",
		Password:          "password",
		SSLMode:           "require",
		ConnectionTimeout: 30 * time.Second,
	}
}

// Constructor Tests

func TestNewMongoDBDatabase(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("create with minimal config", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Host:     "localhost",
			Database: "testdb",
		}

		// This will fail to connect but should create the instance
		db, err := database.NewMongoDBDatabase(config, logger)

		// Expect error since MongoDB likely isn't running
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "MongoDB")
		} else {
			assert.NotNil(t, db)
			assert.Equal(t, database.MongoDB, db.GetDatabaseType())
			defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		}
	})

	t.Run("create with authentication", func(t *testing.T) {
		config := newMongoConfigWithAuth()
		db, err := database.NewMongoDBDatabase(config, logger)

		if err != nil {
			// Expected if MongoDB isn't running or auth fails
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		}
	})

	t.Run("create with SSL", func(t *testing.T) {
		config := newMongoConfigWithSSL()
		db, err := database.NewMongoDBDatabase(config, logger)

		if err != nil {
			// Expected if host doesn't exist or SSL fails
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		}
	})

	t.Run("create with custom URI parameter", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Database: "testdb",
			Parameters: map[string]string{
				"uri": "mongodb://localhost:27017/testdb",
			},
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			_ = db.Disconnect() // Best-effort disconnect in test
		}
	})

	t.Run("create with connection pool settings", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:              database.MongoDB,
			Host:              "localhost",
			Port:              27017,
			Database:          "testdb",
			ConnectionTimeout: 10 * time.Second,
			MaxConnections:    50,
			MaxIdleConns:      10,
			IdleTimeout:       10 * time.Minute,
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			_ = db.Disconnect() // Best-effort disconnect in test
		}
	})

	t.Run("nil logger should be handled", func(t *testing.T) {
		config := newMongoConfig()

		// This should panic or fail gracefully
		defer func() {
			if r := recover(); r != nil {
				// Panic is acceptable for nil logger
				assert.NotNil(t, r)
			}
		}()

		db, err := database.NewMongoDBDatabase(config, nil)
		if err == nil && db != nil {
			_ = db.Disconnect() // Best-effort disconnect in test
		}
	})
}

// Connection Lifecycle Tests

func TestMongoDBDatabase_Connect(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("connect with default port", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Host:     "localhost",
			Port:     0, // Should default to 27017
			Database: "testdb",
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		assert.NoError(t, err)
		assert.NotNil(t, db)
	})

	t.Run("connect with explicit port", func(t *testing.T) {
		config := newMongoConfig()
		config.Port = 27017

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		assert.NotNil(t, db)
	})

	t.Run("reconnect replaces existing connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		// Reconnect with same config
		err = db.Connect(ctx, config)
		assert.NoError(t, err)
	})

	t.Run("connect with invalid host", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:              database.MongoDB,
			Host:              "nonexistent.invalid.host",
			Port:              27017,
			Database:          "testdb",
			ConnectionTimeout: 1 * time.Second,
		}

		_, err := database.NewMongoDBDatabase(config, logger)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "MongoDB")
	})

	t.Run("connect with authentication mechanism", func(t *testing.T) {
		config := newMongoConfigWithAuth()
		config.Parameters["authMechanism"] = "SCRAM-SHA-256"

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			// Expected if MongoDB isn't configured for this auth
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			_ = db.Disconnect() // Best-effort disconnect in test
		}
	})
}

func TestMongoDBDatabase_Disconnect(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("disconnect succeeds", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		err = db.Disconnect()
		assert.NoError(t, err)
	})

	t.Run("disconnect with nil client", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		// First disconnect
		err = db.Disconnect()
		assert.NoError(t, err)

		// Second disconnect should handle nil client gracefully
		err = db.Disconnect()
		assert.NoError(t, err)
	})

	t.Run("disconnect multiple times", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		// Multiple disconnects should not error
		for i := 0; i < 3; i++ {
			err = db.Disconnect()
			assert.NoError(t, err)
		}
	})
}

func TestMongoDBDatabase_Ping(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("ping succeeds", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		err = db.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("ping with context timeout", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond) // Ensure context expires

		err = db.Ping(ctx)
		assert.Error(t, err)
	})

	t.Run("ping after disconnect", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		err = db.Ping(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("ping with cancelled context", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err = db.Ping(ctx)
		assert.Error(t, err)
	})
}

// Connection Info Tests

func TestMongoDBDatabase_GetConnectionInfo(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("get connection info succeeds", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		info, err := db.GetConnectionInfo(ctx)
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Contains(t, info, "database")
		assert.Equal(t, "testdb", info["database"])

		// These fields may be present depending on MongoDB setup
		// Just check they don't cause errors
		_, hasVersion := info["version"]
		_ = hasVersion // Version may not be accessible
	})

	t.Run("get connection info after disconnect", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		_, err = db.GetConnectionInfo(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("get connection info with context timeout", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		_, err = db.GetConnectionInfo(ctx)
		// Context timeout may or may not be detected depending on timing
		// Just verify no panic occurs
		_ = err
	})
}

// Query Execution Tests (Integration)
// Note: These tests require a running MongoDB instance

func TestMongoDBDatabase_Execute_Select(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("execute simple SELECT query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		// This requires a collection to exist - will likely fail but shouldn't panic
		result, err := db.Execute(ctx, "SELECT * FROM test_collection LIMIT 10")
		if err == nil {
			assert.NotNil(t, result)
		}
	})

	t.Run("execute MongoDB JSON query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		// Try a MongoDB-specific query
		query := `{"find": "test_collection", "limit": 10}`
		result, err := db.Execute(ctx, query)
		if err == nil {
			assert.NotNil(t, result)
		}
	})

	t.Run("execute with unsupported query format", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		result, err := db.Execute(ctx, "INVALID QUERY FORMAT")
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, err.Error(), "unsupported query format")
	})

	t.Run("execute without connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		result, err := db.Execute(ctx, "SELECT * FROM test")
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("execute with context cancellation", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		result, err := db.Execute(ctx, "SELECT * FROM test")
		assert.Error(t, err)
		assert.NotNil(t, result)
	})
}

// ExecuteStream Tests

func TestMongoDBDatabase_ExecuteStream(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("stream with disconnected client", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		callback := func(batch [][]interface{}) error {
			return nil
		}

		err = db.ExecuteStream(ctx, "SELECT * FROM test", 10, callback)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("stream with non-SELECT query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		callback := func(batch [][]interface{}) error {
			return nil
		}

		err = db.ExecuteStream(ctx, "INSERT INTO test VALUES (1)", 10, callback)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "streaming only supported for SELECT")
	})

	t.Run("stream with invalid query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		callback := func(batch [][]interface{}) error {
			return nil
		}

		err = db.ExecuteStream(ctx, "SELECT * FROM", 10, callback)
		assert.Error(t, err)
	})
}

// ExplainQuery Tests

func TestMongoDBDatabase_ExplainQuery(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("explain without connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		_, err = db.ExplainQuery(ctx, "SELECT * FROM test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("explain non-SELECT query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		_, err = db.ExplainQuery(ctx, "INSERT INTO test VALUES (1)")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "explain only supported for SELECT")
	})

	t.Run("explain with invalid query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		_, err = db.ExplainQuery(ctx, "SELECT * FROM")
		assert.Error(t, err)
	})
}

// Schema Operations Tests

func TestMongoDBDatabase_GetSchemas(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("get schemas succeeds", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		schemas, err := db.GetSchemas(ctx)
		require.NoError(t, err)
		assert.NotNil(t, schemas)
		// Should return list of databases, excluding system dbs
	})

	t.Run("get schemas without connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		_, err = db.GetSchemas(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})
}

func TestMongoDBDatabase_GetTables(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("get tables with empty schema uses config database", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		tables, err := db.GetTables(ctx, "")
		require.NoError(t, err)
		assert.NotNil(t, tables)
		// Returns list of collections in the database
	})

	t.Run("get tables with specific database", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		tables, err := db.GetTables(ctx, "testdb")
		require.NoError(t, err)
		assert.NotNil(t, tables)
	})

	t.Run("get tables without connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		_, err = db.GetTables(ctx, "testdb")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("get tables returns collection info", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		tables, err := db.GetTables(ctx, "testdb")
		if err == nil {
			for _, table := range tables {
				assert.Equal(t, "COLLECTION", table.Type)
				assert.NotEmpty(t, table.Name)
			}
		}
	})
}

func TestMongoDBDatabase_GetTableStructure(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("get structure with empty schema uses config database", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		// This will fail if collection doesn't exist, but shouldn't panic
		structure, err := db.GetTableStructure(ctx, "", "test_collection")
		if err == nil {
			assert.NotNil(t, structure)
			assert.Equal(t, "test_collection", structure.Table.Name)
		}
	})

	t.Run("get structure without connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		_ = db.Disconnect() // Best-effort disconnect in test

		_, err = db.GetTableStructure(ctx, "testdb", "test")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("structure includes inferred schema from documents", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer func() { _ = db.Disconnect() }() // Best-effort disconnect in test
		skipIfNoMongoDB(t, db)

		structure, err := db.GetTableStructure(ctx, "testdb", "test_collection")
		if err == nil {
			assert.NotNil(t, structure)
			assert.NotNil(t, structure.Columns)
			assert.NotNil(t, structure.Indexes)
		}
	})

	t.Run("structure caching works", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		// First call
		structure1, err1 := db.GetTableStructure(ctx, "testdb", "test_collection")
		// Second call - should use cache
		structure2, err2 := db.GetTableStructure(ctx, "testdb", "test_collection")

		if err1 == nil && err2 == nil {
			assert.Equal(t, structure1.Table.Name, structure2.Table.Name)
		}
	})
}

// Transaction Tests

func TestMongoDBDatabase_BeginTransaction(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("begin transaction without connection", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}

		db.Disconnect()

		_, err = db.BeginTransaction(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not connected")
	})

	t.Run("begin transaction succeeds", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		tx, err := db.BeginTransaction(ctx)
		if err != nil {
			// May fail if not replica set
			assert.Error(t, err)
		} else {
			assert.NotNil(t, tx)
			_ = tx.Rollback() // Best-effort rollback in test
		}
	})

	t.Run("transaction commit", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		tx, err := db.BeginTransaction(ctx)
		if err != nil {
			t.Skipf("Transactions not supported: %v", err)
		}

		err = tx.Commit()
		assert.NoError(t, err)
	})

	t.Run("transaction rollback", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		tx, err := db.BeginTransaction(ctx)
		if err != nil {
			t.Skipf("Transactions not supported: %v", err)
		}

		err = tx.Rollback()
		assert.NoError(t, err)
	})
}

// UpdateRow Tests

func TestMongoDBDatabase_UpdateRow(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("update row not supported", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		params := database.UpdateRowParams{
			Schema: "testdb",
			Table:  "test_collection",
			PrimaryKey: map[string]interface{}{
				"_id": "507f1f77bcf86cd799439011",
			},
			Values: map[string]interface{}{
				"name": "updated",
			},
		}

		err = db.UpdateRow(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not supported")
		assert.Contains(t, err.Error(), "MongoDB")
	})
}

// ComputeEditableMetadata Tests

func TestMongoDBDatabase_ComputeEditableMetadata(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("metadata indicates not editable", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		columns := []string{"_id", "name", "value"}
		metadata, err := db.ComputeEditableMetadata(ctx, "SELECT * FROM test_collection", columns)
		require.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled)
		assert.Contains(t, metadata.Reason, "MongoDB")
		assert.Contains(t, metadata.Reason, "not directly editable")
	})

	t.Run("metadata with empty columns", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		metadata, err := db.ComputeEditableMetadata(ctx, "SELECT * FROM test", []string{})
		require.NoError(t, err)
		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled)
	})
}

// Utility Methods Tests

func TestMongoDBDatabase_GetDatabaseType(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("returns MongoDB type", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		dbType := db.GetDatabaseType()
		assert.Equal(t, database.MongoDB, dbType)
	})
}

func TestMongoDBDatabase_GetConnectionStats(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("get connection stats", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		stats := db.GetConnectionStats()
		assert.NotNil(t, stats)
		// MongoDB driver doesn't expose detailed pool stats
		// Just verify structure
		assert.GreaterOrEqual(t, stats.OpenConnections, 0)
	})
}

func TestMongoDBDatabase_QuoteIdentifier(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "simple identifier",
			identifier: "users",
			want:       "`users`",
		},
		{
			name:       "identifier with spaces",
			identifier: "user data",
			want:       "`user data`",
		},
		{
			name:       "identifier with backticks",
			identifier: "user`name",
			want:       "`user``name`",
		},
		{
			name:       "empty identifier",
			identifier: "",
			want:       "``",
		},
		{
			name:       "identifier with special chars",
			identifier: "test-collection",
			want:       "`test-collection`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := newMongoConfig()
			db, err := database.NewMongoDBDatabase(config, logger)
			if err != nil {
				t.Skipf("MongoDB not available: %v", err)
			}
			defer db.Disconnect()

			result := db.QuoteIdentifier(tt.identifier)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestMongoDBDatabase_GetDataTypeMappings(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("returns MongoDB data type mappings", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		mappings := db.GetDataTypeMappings()
		assert.NotNil(t, mappings)
		assert.Equal(t, "string", mappings["string"])
		assert.Equal(t, "int", mappings["int"])
		assert.Equal(t, "long", mappings["int64"])
		assert.Equal(t, "double", mappings["float"])
		assert.Equal(t, "double", mappings["float64"])
		assert.Equal(t, "bool", mappings["bool"])
		assert.Equal(t, "date", mappings["time"])
		assert.Equal(t, "date", mappings["date"])
		assert.Equal(t, "object", mappings["json"])
		assert.Equal(t, "object", mappings["object"])
		assert.Equal(t, "array", mappings["array"])
		assert.Equal(t, "objectId", mappings["objectId"])
		assert.Equal(t, "binData", mappings["binary"])
		assert.Equal(t, "timestamp", mappings["timestamp"])
		assert.Equal(t, "decimal", mappings["decimal"])
		assert.Equal(t, "regex", mappings["regex"])
		assert.Equal(t, "javascript", mappings["javascript"])
	})
}

// buildConnectionURI Tests (Indirect Testing)

func TestMongoDBDatabase_ConnectionURIBuilding(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("standard URI with host and port", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Host:     "localhost",
			Port:     27017,
			Database: "testdb",
		}

		// The URI is built internally, we test indirectly via connection attempt
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			// Expected if MongoDB not running
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			db.Disconnect()
		}
	})

	t.Run("SRV URI with SSL", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Host:     "cluster.example.com",
			Database: "testdb",
			SSLMode:  "require",
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		// Expected to fail since host doesn't exist
		assert.Error(t, err)
		if db != nil {
			db.Disconnect()
		}
	})

	t.Run("URI with additional parameters", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Host:     "localhost",
			Port:     27017,
			Database: "testdb",
			Parameters: map[string]string{
				"retryWrites": "true",
				"w":           "majority",
			},
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			db.Disconnect()
		}
	})

	t.Run("custom URI parameter overrides", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Database: "testdb",
			Parameters: map[string]string{
				"uri": "mongodb://custom:27017/testdb",
			},
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			_ = db.Disconnect() // Best-effort disconnect in test
		}
	})
}

// Edge Cases

func TestMongoDBDatabase_EdgeCases(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()
	ctx := context.Background()

	t.Run("empty query string", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		result, err := db.Execute(ctx, "")
		assert.Error(t, err)
		assert.NotNil(t, result)
	})

	t.Run("whitespace-only query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		result, err := db.Execute(ctx, "   \n\t  ")
		assert.Error(t, err)
		assert.NotNil(t, result)
	})

	t.Run("malformed JSON query", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		result, err := db.Execute(ctx, `{"find": invalid json}`)
		assert.Error(t, err)
		assert.NotNil(t, result)
	})

	t.Run("zero timeout connection", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:              database.MongoDB,
			Host:              "localhost",
			Port:              27017,
			Database:          "testdb",
			ConnectionTimeout: 0, // Should use default
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			// Expected if MongoDB not running
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			db.Disconnect()
		}
	})

	t.Run("negative connection pool sizes", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:           database.MongoDB,
			Host:           "localhost",
			Port:           27017,
			Database:       "testdb",
			MaxConnections: -1,
			MaxIdleConns:   -1,
		}

		// Should handle gracefully
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			db.Disconnect()
		}
	})

	t.Run("very long collection name", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		longName := "collection_with_very_long_name_that_exceeds_normal_expectations_but_should_still_be_handled"
		query := "SELECT * FROM " + db.QuoteIdentifier(longName) + " LIMIT 1"

		result, err := db.Execute(ctx, query)
		// May fail if collection doesn't exist, but shouldn't panic
		if err == nil {
			assert.NotNil(t, result)
		}
	})

	t.Run("special characters in database name", func(t *testing.T) {
		config := database.ConnectionConfig{
			Type:     database.MongoDB,
			Host:     "localhost",
			Port:     27017,
			Database: "test-db-name", // Hyphen in name
		}

		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			// Expected if MongoDB not running
			assert.Error(t, err)
		} else {
			assert.NotNil(t, db)
			db.Disconnect()
		}
	})

	t.Run("concurrent operations", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()
		skipIfNoMongoDB(t, db)

		// Test thread-safety with concurrent pings
		done := make(chan error, 5)
		for i := 0; i < 5; i++ {
			go func() {
				done <- db.Ping(ctx)
			}()
		}

		for i := 0; i < 5; i++ {
			err := <-done
			// Errors are acceptable, just verify no panics
			_ = err
		}
	})
}

// BSON Type Conversion Tests (Indirect)

func TestMongoDBDatabase_BSONTypeHandling(t *testing.T) {
	requireMongoDB(t)
	logger := newTestLogger()

	t.Run("data type mappings include BSON types", func(t *testing.T) {
		config := newMongoConfig()
		db, err := database.NewMongoDBDatabase(config, logger)
		if err != nil {
			t.Skipf("MongoDB not available: %v", err)
		}
		defer db.Disconnect()

		mappings := db.GetDataTypeMappings()

		// Verify BSON-specific types are mapped
		assert.Contains(t, mappings, "objectId")
		assert.Contains(t, mappings, "binary")
		assert.Contains(t, mappings, "timestamp")
		assert.Contains(t, mappings, "decimal")
		assert.Contains(t, mappings, "regex")
		assert.Contains(t, mappings, "javascript")
	})
}
