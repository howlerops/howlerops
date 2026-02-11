package database

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Database implementation for testing
type mockDatabase struct {
	schemas        []string
	tables         map[string][]TableInfo
	migrationTable string
	migrationRows  [][]interface{}
	executeError   error
}

func (m *mockDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	return m.schemas, nil
}

func (m *mockDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	if tables, ok := m.tables[schema]; ok {
		return tables, nil
	}
	return []TableInfo{}, nil
}

func (m *mockDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	if m.executeError != nil {
		return nil, m.executeError
	}

	// Simulate migration table queries
	if m.migrationTable != "" && m.migrationRows != nil {
		return &QueryResult{
			Columns:  []string{"version"},
			Rows:     m.migrationRows,
			RowCount: int64(len(m.migrationRows)),
		}, nil
	}

	return &QueryResult{
		Columns:  []string{},
		Rows:     [][]interface{}{},
		RowCount: 0,
	}, nil
}

func (m *mockDatabase) ExecuteWithOptions(ctx context.Context, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	// Delegate to Execute for test purposes
	return m.Execute(ctx, query, args...)
}

// Stub implementations for other Database interface methods
func (m *mockDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	return nil
}
func (m *mockDatabase) Disconnect() error              { return nil }
func (m *mockDatabase) Ping(ctx context.Context) error { return nil }
func (m *mockDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	return nil
}
func (m *mockDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	return "", nil
}
func (m *mockDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error) {
	return nil, nil
}
func (m *mockDatabase) GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	return nil, nil
}
func (m *mockDatabase) BeginTransaction(ctx context.Context) (Transaction, error) {
	return nil, nil
}
func (m *mockDatabase) UpdateRow(ctx context.Context, params UpdateRowParams) error {
	return nil
}
func (m *mockDatabase) InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}
func (m *mockDatabase) DeleteRow(ctx context.Context, params DeleteRowParams) error {
	return nil
}
func (m *mockDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	return m.schemas, nil
}
func (m *mockDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	return nil
}
func (m *mockDatabase) GetDatabaseType() DatabaseType {
	return PostgreSQL
}
func (m *mockDatabase) GetConnectionStats() PoolStats {
	return PoolStats{
		OpenConnections: 1,
		InUse:           0,
		Idle:            1,
	}
}
func (m *mockDatabase) QuoteIdentifier(identifier string) string {
	return fmt.Sprintf(`"%s"`, identifier)
}
func (m *mockDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{}
}

// Test helper to create a test logger
func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests
	return logger
}

func TestNewSchemaCache(t *testing.T) {
	tests := []struct {
		name   string
		logger *logrus.Logger
	}{
		{
			name:   "with valid logger",
			logger: newTestLogger(),
		},
		{
			name:   "with nil logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewSchemaCache(tt.logger)

			require.NotNil(t, cache)
			assert.NotNil(t, cache.cache)
			assert.Equal(t, 1*time.Hour, cache.defaultTTL)
			assert.Equal(t, 24*time.Hour, cache.maxCacheAge)
			assert.Equal(t, tt.logger, cache.logger)
		})
	}
}

func TestSchemaCache_CacheSchema(t *testing.T) {
	logger := newTestLogger()
	cache := NewSchemaCache(logger)
	ctx := context.Background()

	mockDB := &mockDatabase{
		schemas: []string{"public", "test"},
		tables: map[string][]TableInfo{
			"public": {
				{Name: "users", Schema: "public", Type: "table"},
				{Name: "posts", Schema: "public", Type: "table"},
			},
			"test": {
				{Name: "temp", Schema: "test", Type: "table"},
			},
		},
		migrationTable: "schema_migrations",
		migrationRows: [][]interface{}{
			{"001_initial"},
			{"002_add_users"},
		},
	}

	schemas := []string{"public", "test"}
	tables := map[string][]TableInfo{
		"public": mockDB.tables["public"],
		"test":   mockDB.tables["test"],
	}

	err := cache.CacheSchema(ctx, "conn1", mockDB, schemas, tables)
	require.NoError(t, err)

	// Verify cache entry exists
	cached, err := cache.GetCachedSchema(ctx, "conn1", mockDB)
	require.NoError(t, err)
	require.NotNil(t, cached)

	assert.Equal(t, "conn1", cached.ConnectionID)
	assert.Equal(t, schemas, cached.Schemas)
	assert.Equal(t, tables, cached.Tables)
	assert.NotEmpty(t, cached.Hash)
	assert.NotEmpty(t, cached.MigrationHash)
	assert.False(t, cached.CachedAt.IsZero())
	assert.False(t, cached.ExpiresAt.IsZero())
	assert.True(t, cached.ExpiresAt.After(cached.CachedAt))
}

func TestSchemaCache_GetCachedSchema(t *testing.T) {
	tests := []struct {
		name         string
		setupCache   func(*SchemaCache)
		connectionID string
		expectCached bool
		expectNil    bool
	}{
		{
			name: "cache miss - no entry",
			setupCache: func(sc *SchemaCache) {
				// Don't add anything
			},
			connectionID: "conn1",
			expectCached: false,
			expectNil:    true,
		},
		{
			name: "cache hit - valid entry",
			setupCache: func(sc *SchemaCache) {
				sc.cache["conn1"] = &CachedSchema{
					ConnectionID:  "conn1",
					Schemas:       []string{"public"},
					Tables:        map[string][]TableInfo{},
					Hash:          "hash123",
					CachedAt:      time.Now(),
					ExpiresAt:     time.Now().Add(1 * time.Hour),
					LastCheckedAt: time.Now(),
				}
			},
			connectionID: "conn1",
			expectCached: true,
			expectNil:    false,
		},
		{
			name: "cache expired",
			setupCache: func(sc *SchemaCache) {
				sc.cache["conn1"] = &CachedSchema{
					ConnectionID:  "conn1",
					Schemas:       []string{"public"},
					Tables:        map[string][]TableInfo{},
					Hash:          "hash123",
					CachedAt:      time.Now().Add(-2 * time.Hour),
					ExpiresAt:     time.Now().Add(-1 * time.Hour), // Expired
					LastCheckedAt: time.Now().Add(-2 * time.Hour),
				}
			},
			connectionID: "conn1",
			expectCached: false,
			expectNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewSchemaCache(newTestLogger())
			tt.setupCache(cache)

			mockDB := &mockDatabase{
				schemas: []string{"public"},
			}

			cached, err := cache.GetCachedSchema(context.Background(), tt.connectionID, mockDB)
			require.NoError(t, err)

			if tt.expectNil {
				assert.Nil(t, cached)
			} else {
				assert.NotNil(t, cached)
				assert.Equal(t, tt.connectionID, cached.ConnectionID)
			}
		})
	}
}

func TestSchemaCache_GetCachedSchema_FreshCache(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	// Add a fresh cache entry (< 5 minutes old)
	cache.cache["conn1"] = &CachedSchema{
		ConnectionID:  "conn1",
		Schemas:       []string{"public"},
		Tables:        map[string][]TableInfo{},
		Hash:          "hash123",
		CachedAt:      time.Now(),
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		LastCheckedAt: time.Now(),
	}

	mockDB := &mockDatabase{}

	cached, err := cache.GetCachedSchema(ctx, "conn1", mockDB)
	require.NoError(t, err)
	require.NotNil(t, cached)
	assert.Equal(t, "conn1", cached.ConnectionID)
}

func TestSchemaCache_InvalidateCache(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())

	// Add some cache entries
	cache.cache["conn1"] = &CachedSchema{ConnectionID: "conn1"}
	cache.cache["conn2"] = &CachedSchema{ConnectionID: "conn2"}

	assert.Len(t, cache.cache, 2)

	// Invalidate one connection
	cache.InvalidateCache("conn1")

	assert.Len(t, cache.cache, 1)
	_, exists := cache.cache["conn1"]
	assert.False(t, exists)
	_, exists = cache.cache["conn2"]
	assert.True(t, exists)
}

func TestSchemaCache_InvalidateAll(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())

	// Add some cache entries
	cache.cache["conn1"] = &CachedSchema{ConnectionID: "conn1"}
	cache.cache["conn2"] = &CachedSchema{ConnectionID: "conn2"}
	cache.cache["conn3"] = &CachedSchema{ConnectionID: "conn3"}

	assert.Len(t, cache.cache, 3)

	// Invalidate all
	cache.InvalidateAll()

	assert.Len(t, cache.cache, 0)
}

func TestSchemaCache_GetCacheStats(t *testing.T) {
	tests := []struct {
		name         string
		setupCache   func(*SchemaCache)
		expectedKeys []string
		checkValues  func(*testing.T, map[string]interface{})
	}{
		{
			name: "empty cache",
			setupCache: func(sc *SchemaCache) {
				// Don't add anything
			},
			expectedKeys: []string{"total_cached", "connections", "oldest_cache", "newest_cache", "total_tables"},
			checkValues: func(t *testing.T, stats map[string]interface{}) {
				assert.Equal(t, 0, stats["total_cached"])
				assert.Equal(t, []string{}, stats["connections"])
				assert.Equal(t, "", stats["oldest_cache"])
				assert.Equal(t, "", stats["newest_cache"])
				assert.Equal(t, 0, stats["total_tables"])
			},
		},
		{
			name: "single cache entry",
			setupCache: func(sc *SchemaCache) {
				sc.cache["conn1"] = &CachedSchema{
					ConnectionID: "conn1",
					CachedAt:     time.Now(),
					Tables: map[string][]TableInfo{
						"public": {
							{Name: "users"},
							{Name: "posts"},
						},
					},
				}
			},
			expectedKeys: []string{"total_cached", "connections", "oldest_cache", "newest_cache", "total_tables"},
			checkValues: func(t *testing.T, stats map[string]interface{}) {
				assert.Equal(t, 1, stats["total_cached"])
				connections := stats["connections"].([]string)
				assert.Len(t, connections, 1)
				assert.Contains(t, connections, "conn1")
				assert.NotEqual(t, "", stats["oldest_cache"])
				assert.NotEqual(t, "", stats["newest_cache"])
				assert.Equal(t, 2, stats["total_tables"])
			},
		},
		{
			name: "multiple cache entries",
			setupCache: func(sc *SchemaCache) {
				now := time.Now()
				sc.cache["conn1"] = &CachedSchema{
					ConnectionID: "conn1",
					CachedAt:     now.Add(-1 * time.Hour),
					Tables: map[string][]TableInfo{
						"public": {{Name: "users"}},
					},
				}
				sc.cache["conn2"] = &CachedSchema{
					ConnectionID: "conn2",
					CachedAt:     now,
					Tables: map[string][]TableInfo{
						"test": {{Name: "temp"}},
					},
				}
			},
			expectedKeys: []string{"total_cached", "connections", "oldest_cache", "newest_cache", "total_tables"},
			checkValues: func(t *testing.T, stats map[string]interface{}) {
				assert.Equal(t, 2, stats["total_cached"])
				connections := stats["connections"].([]string)
				assert.Len(t, connections, 2)
				assert.Equal(t, 2, stats["total_tables"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewSchemaCache(newTestLogger())
			tt.setupCache(cache)

			stats := cache.GetCacheStats()

			for _, key := range tt.expectedKeys {
				assert.Contains(t, stats, key)
			}

			if tt.checkValues != nil {
				tt.checkValues(t, stats)
			}
		})
	}
}

func TestSchemaCache_Concurrency(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	mockDB := &mockDatabase{
		schemas: []string{"public"},
		tables: map[string][]TableInfo{
			"public": {{Name: "users", Schema: "public"}},
		},
	}

	// Number of concurrent operations
	numGoroutines := 100
	var wg sync.WaitGroup

	// Test concurrent cache operations
	t.Run("concurrent cache and retrieve", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				connID := fmt.Sprintf("conn%d", id)
				schemas := []string{"public"}
				tables := map[string][]TableInfo{
					"public": {{Name: fmt.Sprintf("table%d", id), Schema: "public"}},
				}

				// Cache schema
				err := cache.CacheSchema(ctx, connID, mockDB, schemas, tables)
				assert.NoError(t, err)

				// Retrieve schema
				cached, err := cache.GetCachedSchema(ctx, connID, mockDB)
				assert.NoError(t, err)
				assert.NotNil(t, cached)
			}(i)
		}
		wg.Wait()

		// Verify all entries were cached
		stats := cache.GetCacheStats()
		assert.Equal(t, numGoroutines, stats["total_cached"])
	})

	// Test concurrent invalidation
	t.Run("concurrent invalidation", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				connID := fmt.Sprintf("conn%d", id)
				cache.InvalidateCache(connID)
			}(i)
		}
		wg.Wait()

		// Verify all entries were invalidated
		stats := cache.GetCacheStats()
		assert.Equal(t, 0, stats["total_cached"])
	})

	// Test concurrent read/write
	t.Run("concurrent read and write", func(t *testing.T) {
		// Add initial cache entry
		err := cache.CacheSchema(ctx, "shared", mockDB, []string{"public"}, mockDB.tables)
		require.NoError(t, err)

		for i := 0; i < numGoroutines; i++ {
			wg.Add(2)

			// Reader goroutine
			go func() {
				defer wg.Done()
				cached, err := cache.GetCachedSchema(ctx, "shared", mockDB)
				assert.NoError(t, err)
				if cached != nil {
					assert.Equal(t, "shared", cached.ConnectionID)
				}
			}()

			// Writer goroutine
			go func() {
				defer wg.Done()
				err := cache.CacheSchema(ctx, "shared", mockDB, []string{"public"}, mockDB.tables)
				assert.NoError(t, err)
			}()
		}
		wg.Wait()

		// Verify cache is still consistent
		cached, err := cache.GetCachedSchema(ctx, "shared", mockDB)
		assert.NoError(t, err)
		assert.NotNil(t, cached)
		assert.Equal(t, "shared", cached.ConnectionID)
	})
}

func TestSchemaCache_DetectSchemaChange(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	t.Run("no change detected", func(t *testing.T) {
		mockDB := &mockDatabase{
			schemas: []string{"public"},
			tables: map[string][]TableInfo{
				"public": {{Name: "users", Schema: "public"}},
			},
			migrationTable: "schema_migrations",
			migrationRows: [][]interface{}{
				{"001_initial"},
			},
		}

		// Cache initial schema
		err := cache.CacheSchema(ctx, "conn1", mockDB, []string{"public"}, mockDB.tables)
		require.NoError(t, err)

		cached := cache.cache["conn1"]
		require.NotNil(t, cached)

		// Set last checked to old time to trigger change detection
		cached.LastCheckedAt = time.Now().Add(-10 * time.Minute)

		// Get cached schema (should detect no change)
		result, err := cache.GetCachedSchema(ctx, "conn1", mockDB)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("table change detected", func(t *testing.T) {
		mockDB := &mockDatabase{
			schemas: []string{"public"},
			tables: map[string][]TableInfo{
				"public": {{Name: "users", Schema: "public"}},
			},
		}

		// Cache initial schema
		err := cache.CacheSchema(ctx, "conn2", mockDB, []string{"public"}, mockDB.tables)
		require.NoError(t, err)

		cached := cache.cache["conn2"]
		require.NotNil(t, cached)

		// Set last checked to old time to trigger change detection
		cached.LastCheckedAt = time.Now().Add(-10 * time.Minute)

		// Modify database schema
		mockDB.tables["public"] = append(mockDB.tables["public"], TableInfo{Name: "posts", Schema: "public"})

		// Get cached schema (should detect change and invalidate)
		result, err := cache.GetCachedSchema(ctx, "conn2", mockDB)
		require.NoError(t, err)
		assert.Nil(t, result) // Cache should be invalidated
	})

	t.Run("migration change detected", func(t *testing.T) {
		mockDB := &mockDatabase{
			schemas: []string{"public"},
			tables: map[string][]TableInfo{
				"public": {{Name: "users", Schema: "public"}},
			},
			migrationTable: "schema_migrations",
			migrationRows: [][]interface{}{
				{"001_initial"},
			},
		}

		// Cache initial schema
		err := cache.CacheSchema(ctx, "conn3", mockDB, []string{"public"}, mockDB.tables)
		require.NoError(t, err)

		cached := cache.cache["conn3"]
		require.NotNil(t, cached)

		// Set last checked to old time to trigger change detection
		cached.LastCheckedAt = time.Now().Add(-10 * time.Minute)

		// Add new migration
		mockDB.migrationRows = append(mockDB.migrationRows, []interface{}{"002_add_posts"})

		// Get cached schema (should detect change and invalidate)
		result, err := cache.GetCachedSchema(ctx, "conn3", mockDB)
		require.NoError(t, err)
		assert.Nil(t, result) // Cache should be invalidated
	})
}

func TestSchemaCache_HashingFunctions(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())

	t.Run("hashStringList deterministic", func(t *testing.T) {
		list1 := []string{"table1", "table2", "table3"}
		list2 := []string{"table1", "table2", "table3"}

		hash1 := cache.hashStringList(list1)
		hash2 := cache.hashStringList(list2)

		assert.Equal(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
	})

	t.Run("hashStringList different for different lists", func(t *testing.T) {
		list1 := []string{"table1", "table2"}
		list2 := []string{"table1", "table3"}

		hash1 := cache.hashStringList(list1)
		hash2 := cache.hashStringList(list2)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("combineHashes deterministic", func(t *testing.T) {
		hash1 := cache.combineHashes("hash1", "hash2", "hash3")
		hash2 := cache.combineHashes("hash1", "hash2", "hash3")

		assert.Equal(t, hash1, hash2)
		assert.NotEmpty(t, hash1)
	})

	t.Run("combineHashes different for different inputs", func(t *testing.T) {
		hash1 := cache.combineHashes("hash1", "hash2")
		hash2 := cache.combineHashes("hash1", "hash3")

		assert.NotEqual(t, hash1, hash2)
	})
}

func TestSchemaCache_ExtractTableList(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())

	tests := []struct {
		name     string
		tables   map[string][]TableInfo
		expected []string
	}{
		{
			name:     "empty tables",
			tables:   map[string][]TableInfo{},
			expected: nil,
		},
		{
			name: "single schema",
			tables: map[string][]TableInfo{
				"public": {
					{Name: "users", Schema: "public"},
					{Name: "posts", Schema: "public"},
				},
			},
			expected: []string{"public.posts", "public.users"},
		},
		{
			name: "multiple schemas",
			tables: map[string][]TableInfo{
				"public": {
					{Name: "users", Schema: "public"},
				},
				"test": {
					{Name: "temp", Schema: "test"},
				},
			},
			expected: []string{"public.users", "test.temp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.extractTableList(tt.tables)
			if tt.expected == nil {
				assert.Empty(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestSchemaCache_CountTables(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())

	tests := []struct {
		name     string
		tables   map[string][]TableInfo
		expected int
	}{
		{
			name:     "empty tables",
			tables:   map[string][]TableInfo{},
			expected: 0,
		},
		{
			name: "single schema",
			tables: map[string][]TableInfo{
				"public": {
					{Name: "users"},
					{Name: "posts"},
					{Name: "comments"},
				},
			},
			expected: 3,
		},
		{
			name: "multiple schemas",
			tables: map[string][]TableInfo{
				"public": {
					{Name: "users"},
					{Name: "posts"},
				},
				"test": {
					{Name: "temp"},
				},
			},
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.countTables(tt.tables)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaCache_GenerateFingerprint(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	t.Run("with migration table", func(t *testing.T) {
		mockDB := &mockDatabase{
			migrationTable: "schema_migrations",
			migrationRows: [][]interface{}{
				{"001_initial"},
				{"002_add_users"},
			},
		}

		schemas := []string{"public"}
		tables := map[string][]TableInfo{
			"public": {{Name: "users", Schema: "public"}},
		}

		fingerprint, err := cache.generateFingerprint(ctx, mockDB, schemas, tables)
		require.NoError(t, err)
		assert.NotNil(t, fingerprint)
		assert.NotEmpty(t, fingerprint.Hash)
		assert.NotEmpty(t, fingerprint.MigrationState)
		assert.Equal(t, []string{"public.users"}, fingerprint.TableList)
	})

	t.Run("without migration table", func(t *testing.T) {
		mockDB := &mockDatabase{
			executeError: fmt.Errorf("table does not exist"),
		}

		schemas := []string{"public"}
		tables := map[string][]TableInfo{
			"public": {{Name: "users", Schema: "public"}},
		}

		fingerprint, err := cache.generateFingerprint(ctx, mockDB, schemas, tables)
		require.NoError(t, err)
		assert.NotNil(t, fingerprint)
		assert.NotEmpty(t, fingerprint.Hash)
		assert.Empty(t, fingerprint.MigrationState) // No migration table
		assert.Equal(t, []string{"public.users"}, fingerprint.TableList)
	})
}

func TestSchemaCache_GenerateLightweightFingerprint(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	mockDB := &mockDatabase{
		schemas: []string{"public"},
		tables: map[string][]TableInfo{
			"public": {
				{Name: "users", Schema: "public"},
				{Name: "posts", Schema: "public"},
			},
		},
		migrationTable: "schema_migrations",
		migrationRows: [][]interface{}{
			{"001_initial"},
		},
	}

	fingerprint, err := cache.generateLightweightFingerprint(ctx, mockDB)
	require.NoError(t, err)
	assert.NotNil(t, fingerprint)
	assert.NotEmpty(t, fingerprint.Hash)
	assert.NotEmpty(t, fingerprint.MigrationState)
	assert.Contains(t, fingerprint.TableList, "public.users")
	assert.Contains(t, fingerprint.TableList, "public.posts")
}

func TestSchemaCache_GetMigrationStateHash(t *testing.T) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	tests := []struct {
		name           string
		mockDB         *mockDatabase
		expectNonEmpty bool
	}{
		{
			name: "with schema_migrations table",
			mockDB: &mockDatabase{
				migrationTable: "schema_migrations",
				migrationRows: [][]interface{}{
					{"001_initial"},
					{"002_add_users"},
				},
			},
			expectNonEmpty: true,
		},
		{
			name: "no migration table",
			mockDB: &mockDatabase{
				executeError: fmt.Errorf("table does not exist"),
			},
			expectNonEmpty: false,
		},
		{
			name: "empty migration table",
			mockDB: &mockDatabase{
				migrationTable: "schema_migrations",
				migrationRows:  [][]interface{}{},
			},
			expectNonEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := cache.getMigrationStateHash(ctx, tt.mockDB)
			require.NoError(t, err)

			if tt.expectNonEmpty {
				assert.NotEmpty(t, hash)
			} else {
				assert.Empty(t, hash)
			}
		})
	}
}

func TestSchemaCache_EdgeCases(t *testing.T) {
	t.Run("cache with zero TTL", func(t *testing.T) {
		cache := NewSchemaCache(newTestLogger())
		cache.defaultTTL = 0 // Edge case: zero TTL

		ctx := context.Background()
		mockDB := &mockDatabase{
			schemas: []string{"public"},
			tables: map[string][]TableInfo{
				"public": {{Name: "users", Schema: "public"}},
			},
		}

		err := cache.CacheSchema(ctx, "conn1", mockDB, []string{"public"}, mockDB.tables)
		require.NoError(t, err)

		// Cache should expire immediately
		cached := cache.cache["conn1"]
		assert.True(t, time.Now().After(cached.ExpiresAt) || time.Now().Equal(cached.ExpiresAt))
	})

	t.Run("cache with empty schemas", func(t *testing.T) {
		cache := NewSchemaCache(newTestLogger())
		ctx := context.Background()
		mockDB := &mockDatabase{}

		err := cache.CacheSchema(ctx, "conn1", mockDB, []string{}, map[string][]TableInfo{})
		require.NoError(t, err)

		cached, err := cache.GetCachedSchema(ctx, "conn1", mockDB)
		require.NoError(t, err)
		assert.NotNil(t, cached)
		assert.Empty(t, cached.Schemas)
	})

	t.Run("invalidate non-existent connection", func(t *testing.T) {
		cache := NewSchemaCache(newTestLogger())

		// Should not panic
		assert.NotPanics(t, func() {
			cache.InvalidateCache("non-existent")
		})
	})

	t.Run("get cached schema with fresh entry skips change detection", func(t *testing.T) {
		cache := NewSchemaCache(newTestLogger())
		ctx := context.Background()

		// Add a fresh cache entry (< 5 minutes old)
		// This should skip change detection entirely
		cache.cache["conn1"] = &CachedSchema{
			ConnectionID:  "conn1",
			Schemas:       []string{"public"},
			Tables:        map[string][]TableInfo{},
			Hash:          "abcdef0123456789",
			MigrationHash: "fedcba9876543210",
			CachedAt:      time.Now(),
			ExpiresAt:     time.Now().Add(1 * time.Hour),
			LastCheckedAt: time.Now(), // Fresh timestamp
		}

		// Mock database that would fail if accessed
		// But it shouldn't be accessed for fresh cache
		mockDB := &mockDatabase{
			executeError: fmt.Errorf("should not be called"),
		}

		// Should return cached data without calling database
		cached, err := cache.GetCachedSchema(ctx, "conn1", mockDB)
		require.NoError(t, err)
		assert.NotNil(t, cached)
		assert.Equal(t, "conn1", cached.ConnectionID)
	})
}

func BenchmarkSchemaCache_CacheAndRetrieve(b *testing.B) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	mockDB := &mockDatabase{
		schemas: []string{"public"},
		tables: map[string][]TableInfo{
			"public": {
				{Name: "users", Schema: "public"},
				{Name: "posts", Schema: "public"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		connID := fmt.Sprintf("conn%d", i)
		_ = cache.CacheSchema(ctx, connID, mockDB, mockDB.schemas, mockDB.tables)
		_, _ = cache.GetCachedSchema(ctx, connID, mockDB)
	}
}

func BenchmarkSchemaCache_ConcurrentAccess(b *testing.B) {
	cache := NewSchemaCache(newTestLogger())
	ctx := context.Background()

	mockDB := &mockDatabase{
		schemas: []string{"public"},
		tables: map[string][]TableInfo{
			"public": {{Name: "users", Schema: "public"}},
		},
	}

	// Pre-populate cache
	_ = cache.CacheSchema(ctx, "shared", mockDB, mockDB.schemas, mockDB.tables)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = cache.GetCachedSchema(ctx, "shared", mockDB)
		}
	})
}
