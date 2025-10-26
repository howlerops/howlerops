package database

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test helper to create a sample TableStructure
func createSampleTableStructure(schema, table string) *TableStructure {
	return &TableStructure{
		Table: TableInfo{
			Schema:    schema,
			Name:      table,
			Type:      "table",
			Comment:   "Sample table",
			RowCount:  100,
			SizeBytes: 4096,
		},
		Columns: []ColumnInfo{
			{
				Name:            "id",
				DataType:        "integer",
				Nullable:        false,
				PrimaryKey:      true,
				OrdinalPosition: 1,
			},
			{
				Name:            "name",
				DataType:        "varchar",
				Nullable:        true,
				OrdinalPosition: 2,
			},
		},
		Indexes: []IndexInfo{
			{
				Name:   "idx_name",
				Unique: false,
			},
		},
		ForeignKeys: []ForeignKeyInfo{
			{
				Name:              "fk_user",
				Columns:           []string{"user_id"},
				ReferencedTable:   "users",
				ReferencedSchema:  "public",
				ReferencedColumns: []string{"id"},
			},
		},
		Triggers: []string{"trigger_update", "trigger_delete"},
		Statistics: map[string]string{
			"n_live_tup": "100",
			"n_dead_tup": "5",
		},
	}
}

func TestNewTableStructureCache(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		wantTTL time.Duration
	}{
		{
			name:    "custom TTL",
			ttl:     10 * time.Minute,
			wantTTL: 10 * time.Minute,
		},
		{
			name:    "zero TTL uses default",
			ttl:     0,
			wantTTL: 5 * time.Minute,
		},
		{
			name:    "negative TTL uses default",
			ttl:     -1 * time.Second,
			wantTTL: 5 * time.Minute,
		},
		{
			name:    "very short TTL",
			ttl:     100 * time.Millisecond,
			wantTTL: 100 * time.Millisecond,
		},
		{
			name:    "very long TTL",
			ttl:     24 * time.Hour,
			wantTTL: 24 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := newTableStructureCache(tt.ttl)
			require.NotNil(t, cache)
			assert.Equal(t, tt.wantTTL, cache.ttl)
			assert.NotNil(t, cache.entries)
			assert.Empty(t, cache.entries)
		})
	}
}

func TestTableStructureCache_GetSet(t *testing.T) {
	t.Run("get from empty cache returns false", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		retrieved, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("set and get returns structure", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		structure := createSampleTableStructure("public", "users")

		cache.set("public", "users", structure)

		retrieved, ok := cache.get("public", "users")
		require.True(t, ok)
		require.NotNil(t, retrieved)
		assert.Equal(t, "public", retrieved.Table.Schema)
		assert.Equal(t, "users", retrieved.Table.Name)
		assert.Len(t, retrieved.Columns, 2)
		assert.Len(t, retrieved.Indexes, 1)
		assert.Len(t, retrieved.ForeignKeys, 1)
		assert.Len(t, retrieved.Triggers, 2)
		assert.Len(t, retrieved.Statistics, 2)
	})

	t.Run("set overwrites existing entry", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		// Set initial structure
		structure1 := createSampleTableStructure("public", "users")
		structure1.Table.RowCount = 100
		cache.set("public", "users", structure1)

		// Overwrite with new structure
		structure2 := createSampleTableStructure("public", "users")
		structure2.Table.RowCount = 200
		cache.set("public", "users", structure2)

		// Verify new structure is returned
		retrieved, ok := cache.get("public", "users")
		require.True(t, ok)
		assert.Equal(t, int64(200), retrieved.Table.RowCount)
	})

	t.Run("set with nil structure does nothing", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		cache.set("public", "users", nil)

		retrieved, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("get from nil cache returns false", func(t *testing.T) {
		var cache *tableStructureCache

		retrieved, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("set on nil cache does nothing", func(t *testing.T) {
		var cache *tableStructureCache
		structure := createSampleTableStructure("public", "users")

		// Should not panic
		cache.set("public", "users", structure)
	})

	t.Run("multiple different tables", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		structure1 := createSampleTableStructure("public", "users")
		structure2 := createSampleTableStructure("public", "orders")
		structure3 := createSampleTableStructure("analytics", "events")

		cache.set("public", "users", structure1)
		cache.set("public", "orders", structure2)
		cache.set("analytics", "events", structure3)

		// All should be retrievable
		retrieved1, ok := cache.get("public", "users")
		assert.True(t, ok)
		assert.Equal(t, "users", retrieved1.Table.Name)

		retrieved2, ok := cache.get("public", "orders")
		assert.True(t, ok)
		assert.Equal(t, "orders", retrieved2.Table.Name)

		retrieved3, ok := cache.get("analytics", "events")
		assert.True(t, ok)
		assert.Equal(t, "events", retrieved3.Table.Name)
	})
}

func TestTableStructureCache_Expiration(t *testing.T) {
	t.Run("entry expires after TTL", func(t *testing.T) {
		cache := newTableStructureCache(50 * time.Millisecond)
		structure := createSampleTableStructure("public", "users")

		cache.set("public", "users", structure)

		// Should exist immediately
		retrieved, ok := cache.get("public", "users")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		// Should be expired and removed
		retrieved, ok = cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)

		// Verify entry was actually deleted from map
		cache.mu.RLock()
		_, exists := cache.entries[cacheKey("public", "users")]
		cache.mu.RUnlock()
		assert.False(t, exists)
	})

	t.Run("entry does not expire before TTL", func(t *testing.T) {
		cache := newTableStructureCache(200 * time.Millisecond)
		structure := createSampleTableStructure("public", "users")

		cache.set("public", "users", structure)

		// Wait for less than TTL
		time.Sleep(50 * time.Millisecond)

		// Should still exist
		retrieved, ok := cache.get("public", "users")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
	})

	t.Run("different entries expire independently", func(t *testing.T) {
		cache := newTableStructureCache(100 * time.Millisecond)

		// Set first entry
		structure1 := createSampleTableStructure("public", "users")
		cache.set("public", "users", structure1)

		// Wait a bit
		time.Sleep(60 * time.Millisecond)

		// Set second entry
		structure2 := createSampleTableStructure("public", "orders")
		cache.set("public", "orders", structure2)

		// Wait for first to expire
		time.Sleep(50 * time.Millisecond)

		// First should be expired
		retrieved1, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved1)

		// Second should still exist
		retrieved2, ok := cache.get("public", "orders")
		assert.True(t, ok)
		assert.NotNil(t, retrieved2)
	})
}

func TestTableStructureCache_Invalidate(t *testing.T) {
	t.Run("invalidate existing entry", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		structure := createSampleTableStructure("public", "users")

		cache.set("public", "users", structure)

		// Verify it exists
		_, ok := cache.get("public", "users")
		assert.True(t, ok)

		// Invalidate
		cache.invalidate("public", "users")

		// Should no longer exist
		retrieved, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("invalidate non-existent entry does not panic", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		// Should not panic
		cache.invalidate("public", "nonexistent")
	})

	t.Run("invalidate on nil cache does not panic", func(t *testing.T) {
		var cache *tableStructureCache

		// Should not panic
		cache.invalidate("public", "users")
	})

	t.Run("invalidate only removes specified entry", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		structure1 := createSampleTableStructure("public", "users")
		structure2 := createSampleTableStructure("public", "orders")
		cache.set("public", "users", structure1)
		cache.set("public", "orders", structure2)

		// Invalidate one
		cache.invalidate("public", "users")

		// First should be gone
		_, ok := cache.get("public", "users")
		assert.False(t, ok)

		// Second should remain
		retrieved, ok := cache.get("public", "orders")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
	})
}

func TestTableStructureCache_Clear(t *testing.T) {
	t.Run("clear removes all entries", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		structure1 := createSampleTableStructure("public", "users")
		structure2 := createSampleTableStructure("public", "orders")
		structure3 := createSampleTableStructure("analytics", "events")

		cache.set("public", "users", structure1)
		cache.set("public", "orders", structure2)
		cache.set("analytics", "events", structure3)

		// Clear
		cache.clear()

		// All should be gone
		_, ok := cache.get("public", "users")
		assert.False(t, ok)
		_, ok = cache.get("public", "orders")
		assert.False(t, ok)
		_, ok = cache.get("analytics", "events")
		assert.False(t, ok)

		// Map should be empty
		cache.mu.RLock()
		assert.Empty(t, cache.entries)
		cache.mu.RUnlock()
	})

	t.Run("clear on empty cache", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		// Should not panic
		cache.clear()

		cache.mu.RLock()
		assert.Empty(t, cache.entries)
		cache.mu.RUnlock()
	})

	t.Run("clear on nil cache does not panic", func(t *testing.T) {
		var cache *tableStructureCache

		// Should not panic
		cache.clear()
	})

	t.Run("can add entries after clear", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)

		structure1 := createSampleTableStructure("public", "users")
		cache.set("public", "users", structure1)

		cache.clear()

		structure2 := createSampleTableStructure("public", "orders")
		cache.set("public", "orders", structure2)

		retrieved, ok := cache.get("public", "orders")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
	})
}

func TestCacheKey(t *testing.T) {
	tests := []struct {
		name   string
		schema string
		table  string
		want   string
	}{
		{
			name:   "normal case",
			schema: "public",
			table:  "users",
			want:   "public.users",
		},
		{
			name:   "uppercase normalized to lowercase",
			schema: "PUBLIC",
			table:  "USERS",
			want:   "public.users",
		},
		{
			name:   "mixed case normalized",
			schema: "MySchema",
			table:  "MyTable",
			want:   "myschema.mytable",
		},
		{
			name:   "with whitespace trimmed",
			schema: "  public  ",
			table:  "  users  ",
			want:   "public.users",
		},
		{
			name:   "empty schema",
			schema: "",
			table:  "users",
			want:   ".users",
		},
		{
			name:   "empty table",
			schema: "public",
			table:  "",
			want:   "public.",
		},
		{
			name:   "both empty",
			schema: "",
			table:  "",
			want:   ".",
		},
		{
			name:   "special characters preserved",
			schema: "my_schema",
			table:  "my-table",
			want:   "my_schema.my-table",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cacheKey(tt.schema, tt.table)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCloneTableStructure(t *testing.T) {
	t.Run("nil structure returns nil", func(t *testing.T) {
		clone := cloneTableStructure(nil)
		assert.Nil(t, clone)
	})

	t.Run("clone creates independent copy", func(t *testing.T) {
		original := createSampleTableStructure("public", "users")

		clone := cloneTableStructure(original)
		require.NotNil(t, clone)

		// Verify deep copy - modify clone and check original unchanged
		clone.Table.Name = "modified"
		clone.Columns[0].Name = "modified_id"
		clone.Indexes[0].Name = "modified_idx"
		clone.ForeignKeys[0].Name = "modified_fk"
		clone.Triggers[0] = "modified_trigger"
		clone.Statistics["new_key"] = "new_value"

		// Original should be unchanged
		assert.Equal(t, "users", original.Table.Name)
		assert.Equal(t, "id", original.Columns[0].Name)
		assert.Equal(t, "idx_name", original.Indexes[0].Name)
		assert.Equal(t, "fk_user", original.ForeignKeys[0].Name)
		assert.Equal(t, "trigger_update", original.Triggers[0])
		assert.NotContains(t, original.Statistics, "new_key")
	})

	t.Run("empty slices cloned correctly", func(t *testing.T) {
		original := &TableStructure{
			Table:       TableInfo{Schema: "public", Name: "empty"},
			Columns:     []ColumnInfo{},
			Indexes:     []IndexInfo{},
			ForeignKeys: []ForeignKeyInfo{},
			Triggers:    []string{},
			Statistics:  map[string]string{},
		}

		clone := cloneTableStructure(original)
		require.NotNil(t, clone)

		assert.Nil(t, clone.Columns)
		assert.Nil(t, clone.Indexes)
		assert.Nil(t, clone.ForeignKeys)
		assert.Nil(t, clone.Triggers)
		assert.NotNil(t, clone.Statistics)
		assert.Empty(t, clone.Statistics)
	})

	t.Run("nil slices cloned correctly", func(t *testing.T) {
		original := &TableStructure{
			Table:       TableInfo{Schema: "public", Name: "nil_slices"},
			Columns:     nil,
			Indexes:     nil,
			ForeignKeys: nil,
			Triggers:    nil,
			Statistics:  map[string]string{"key": "value"},
		}

		clone := cloneTableStructure(original)
		require.NotNil(t, clone)

		assert.Nil(t, clone.Columns)
		assert.Nil(t, clone.Indexes)
		assert.Nil(t, clone.ForeignKeys)
		assert.Nil(t, clone.Triggers)
		assert.Len(t, clone.Statistics, 1)
	})
}

func TestTableStructureCache_Concurrency(t *testing.T) {
	t.Run("concurrent gets and sets", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		var wg sync.WaitGroup
		numGoroutines := 100
		numOperations := 50

		// Concurrent sets
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					structure := createSampleTableStructure("public", "table")
					structure.Table.RowCount = int64(id*numOperations + j)
					cache.set("public", "table", structure)
				}
			}(i)
		}

		// Concurrent gets
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					cache.get("public", "table")
				}
			}()
		}

		wg.Wait()

		// Verify cache is still functional
		retrieved, ok := cache.get("public", "table")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
	})

	t.Run("concurrent operations on different tables", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		var wg sync.WaitGroup
		numTables := 50

		for i := 0; i < numTables; i++ {
			wg.Add(1)
			go func(tableNum int) {
				defer wg.Done()
				// Use unique table name for each goroutine
				tableName := fmt.Sprintf("table_%d", tableNum)
				structure := createSampleTableStructure("public", tableName)
				structure.Table.RowCount = int64(tableNum)

				// Set
				cache.set("public", tableName, structure)

				// Get
				retrieved, ok := cache.get("public", tableName)
				assert.True(t, ok)
				assert.NotNil(t, retrieved)

				// Invalidate half of them
				if tableNum%2 == 0 {
					cache.invalidate("public", tableName)
				}
			}(i)
		}

		wg.Wait()

		// Verify that odd-numbered tables still exist and even ones are gone
		for i := 0; i < numTables; i++ {
			tableName := fmt.Sprintf("table_%d", i)
			retrieved, ok := cache.get("public", tableName)
			if i%2 == 0 {
				// Even tables should be invalidated
				assert.False(t, ok, "table %s should be invalidated", tableName)
				assert.Nil(t, retrieved)
			} else {
				// Odd tables should still exist
				assert.True(t, ok, "table %s should exist", tableName)
				assert.NotNil(t, retrieved)
			}
		}
	})

	t.Run("concurrent invalidate and get", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		structure := createSampleTableStructure("public", "users")
		cache.set("public", "users", structure)

		var wg sync.WaitGroup
		numGoroutines := 50

		// Concurrent invalidations
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.invalidate("public", "users")
			}()
		}

		// Concurrent gets
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cache.get("public", "users")
			}()
		}

		wg.Wait()

		// Should not panic, and entry should be gone
		retrieved, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})

	t.Run("concurrent clear and set", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		var wg sync.WaitGroup
		numGoroutines := 50

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				if id%5 == 0 {
					cache.clear()
				} else {
					structure := createSampleTableStructure("public", "table")
					cache.set("public", "table", structure)
				}
			}(i)
		}

		wg.Wait()

		// Should not panic
	})

	t.Run("concurrent expiration cleanup", func(t *testing.T) {
		cache := newTableStructureCache(50 * time.Millisecond)
		structure := createSampleTableStructure("public", "users")
		cache.set("public", "users", structure)

		var wg sync.WaitGroup
		numGoroutines := 50

		// Wait for expiration
		time.Sleep(60 * time.Millisecond)

		// Concurrent gets that will trigger expiration cleanup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				retrieved, ok := cache.get("public", "users")
				assert.False(t, ok)
				assert.Nil(t, retrieved)
			}()
		}

		wg.Wait()

		// Entry should be cleaned up
		cache.mu.RLock()
		_, exists := cache.entries[cacheKey("public", "users")]
		cache.mu.RUnlock()
		assert.False(t, exists)
	})
}

func TestTableStructureCache_DeepCopyVerification(t *testing.T) {
	t.Run("modifications to retrieved structure do not affect cache", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		original := createSampleTableStructure("public", "users")

		cache.set("public", "users", original)

		// Get and modify
		retrieved1, ok := cache.get("public", "users")
		require.True(t, ok)
		retrieved1.Table.Name = "modified"
		retrieved1.Columns[0].Name = "modified_id"
		retrieved1.Statistics["new_key"] = "new_value"

		// Get again and verify unmodified
		retrieved2, ok := cache.get("public", "users")
		require.True(t, ok)
		assert.Equal(t, "users", retrieved2.Table.Name)
		assert.Equal(t, "id", retrieved2.Columns[0].Name)
		assert.NotContains(t, retrieved2.Statistics, "new_key")
	})

	t.Run("modifications to set structure do not affect cache", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		structure := createSampleTableStructure("public", "users")

		cache.set("public", "users", structure)

		// Modify original after set
		structure.Table.Name = "modified"
		structure.Columns[0].Name = "modified_id"

		// Get and verify unmodified
		retrieved, ok := cache.get("public", "users")
		require.True(t, ok)
		assert.Equal(t, "users", retrieved.Table.Name)
		assert.Equal(t, "id", retrieved.Columns[0].Name)
	})
}

func TestTableStructureCache_CaseInsensitivity(t *testing.T) {
	t.Run("case-insensitive key matching", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		structure := createSampleTableStructure("public", "users")

		cache.set("PUBLIC", "USERS", structure)

		// Should find with different case
		retrieved, ok := cache.get("public", "users")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)

		retrieved, ok = cache.get("Public", "Users")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)

		retrieved, ok = cache.get("PUBLIC", "USERS")
		assert.True(t, ok)
		assert.NotNil(t, retrieved)
	})

	t.Run("invalidate with different case", func(t *testing.T) {
		cache := newTableStructureCache(1 * time.Minute)
		structure := createSampleTableStructure("public", "users")

		cache.set("public", "users", structure)

		// Invalidate with different case
		cache.invalidate("PUBLIC", "USERS")

		// Should be gone
		retrieved, ok := cache.get("public", "users")
		assert.False(t, ok)
		assert.Nil(t, retrieved)
	})
}
