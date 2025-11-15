package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewEditableMetadata tests the newEditableMetadata constructor function
func TestNewEditableMetadata(t *testing.T) {
	t.Run("creates metadata with valid column list", func(t *testing.T) {
		columns := []string{"id", "name", "email"}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled, "should be disabled by default")
		assert.Equal(t, 3, len(metadata.Columns), "should have 3 columns")
		assert.Equal(t, 0, len(metadata.PrimaryKeys), "should have empty primary keys")

		// Verify each column is initialized correctly
		for i, col := range metadata.Columns {
			assert.Equal(t, columns[i], col.Name, "column name should match input")
			assert.Equal(t, columns[i], col.ResultName, "result name should match column name")
			assert.False(t, col.Editable, "should be non-editable by default")
			assert.False(t, col.PrimaryKey, "should not be primary key by default")
		}
	})

	t.Run("creates metadata with empty column list", func(t *testing.T) {
		columns := []string{}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled)
		assert.Equal(t, 0, len(metadata.Columns), "should have no columns")
		assert.Equal(t, 0, len(metadata.PrimaryKeys))
		assert.NotNil(t, metadata.Columns, "columns slice should not be nil")
		assert.NotNil(t, metadata.PrimaryKeys, "primary keys slice should not be nil")
	})

	t.Run("creates metadata with nil column list", func(t *testing.T) {
		var columns []string = nil
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled)
		assert.Equal(t, 0, len(metadata.Columns))
		assert.Equal(t, 0, len(metadata.PrimaryKeys))
		assert.NotNil(t, metadata.Columns, "columns slice should be initialized, not nil")
		assert.NotNil(t, metadata.PrimaryKeys, "primary keys slice should be initialized, not nil")
	})

	t.Run("creates metadata with single column", func(t *testing.T) {
		columns := []string{"id"}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled)
		assert.Equal(t, 1, len(metadata.Columns))
		assert.Equal(t, "id", metadata.Columns[0].Name)
		assert.Equal(t, "id", metadata.Columns[0].ResultName)
		assert.False(t, metadata.Columns[0].Editable)
		assert.False(t, metadata.Columns[0].PrimaryKey)
	})

	t.Run("creates metadata with many columns", func(t *testing.T) {
		columns := []string{
			"id", "first_name", "last_name", "email", "phone",
			"address", "city", "state", "zip", "country",
			"created_at", "updated_at", "deleted_at",
		}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.False(t, metadata.Enabled)
		assert.Equal(t, len(columns), len(metadata.Columns))

		// Verify all columns are initialized
		for i, col := range metadata.Columns {
			assert.Equal(t, columns[i], col.Name)
			assert.Equal(t, columns[i], col.ResultName)
			assert.False(t, col.Editable)
			assert.False(t, col.PrimaryKey)
		}
	})

	t.Run("creates metadata with duplicate column names", func(t *testing.T) {
		columns := []string{"id", "name", "id", "name"}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.Equal(t, 4, len(metadata.Columns), "should preserve duplicates")

		// Verify duplicates are both present
		assert.Equal(t, "id", metadata.Columns[0].Name)
		assert.Equal(t, "name", metadata.Columns[1].Name)
		assert.Equal(t, "id", metadata.Columns[2].Name)
		assert.Equal(t, "name", metadata.Columns[3].Name)
	})

	t.Run("creates metadata with special column names", func(t *testing.T) {
		columns := []string{
			"user.id",          // qualified name
			"COUNT(*)",         // aggregate function
			"CONCAT(a, b)",     // expression
			"a AS alias",       // aliased column
			"table.*",          // wildcard
			"",                 // empty string
			"column-with-dash", // special chars
			"column_with_underscore",
			"CamelCaseColumn",
			"ALL_CAPS",
		}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.Equal(t, len(columns), len(metadata.Columns))

		// Verify all special names are preserved as-is
		for i, col := range metadata.Columns {
			assert.Equal(t, columns[i], col.Name, "should preserve special column names")
			assert.Equal(t, columns[i], col.ResultName)
		}
	})

	t.Run("creates metadata with unicode column names", func(t *testing.T) {
		columns := []string{"用户ID", "名前", "电子邮件", "téléphone"}
		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.Equal(t, 4, len(metadata.Columns))

		for i, col := range metadata.Columns {
			assert.Equal(t, columns[i], col.Name, "should preserve unicode column names")
		}
	})

	t.Run("metadata structure has correct default values", func(t *testing.T) {
		columns := []string{"id", "name"}
		metadata := newEditableMetadata(columns)

		// Verify all top-level fields
		assert.False(t, metadata.Enabled, "Enabled should default to false")
		assert.Empty(t, metadata.Reason, "Reason should be empty")
		assert.Empty(t, metadata.Schema, "Schema should be empty")
		assert.Empty(t, metadata.Table, "Table should be empty")
		assert.NotNil(t, metadata.PrimaryKeys, "PrimaryKeys should be initialized")
		assert.Empty(t, metadata.PrimaryKeys, "PrimaryKeys should be empty slice")
		assert.NotNil(t, metadata.Columns, "Columns should be initialized")
		assert.False(t, metadata.Pending, "Pending should be false")
		assert.Empty(t, metadata.JobID, "JobID should be empty")
		assert.Nil(t, metadata.Capabilities, "Capabilities should be nil")
	})

	t.Run("column structure has correct default values", func(t *testing.T) {
		columns := []string{"test_column"}
		metadata := newEditableMetadata(columns)

		col := metadata.Columns[0]
		assert.Equal(t, "test_column", col.Name)
		assert.Equal(t, "test_column", col.ResultName)
		assert.Empty(t, col.DataType, "DataType should be empty")
		assert.False(t, col.Editable, "Editable should be false")
		assert.False(t, col.PrimaryKey, "PrimaryKey should be false")
		assert.Nil(t, col.ForeignKey, "ForeignKey should be nil")
		assert.False(t, col.HasDefault, "HasDefault should be false")
		assert.Nil(t, col.DefaultVal, "DefaultVal should be nil")
		assert.Empty(t, col.DefaultExp, "DefaultExp should be empty")
		assert.False(t, col.AutoNumber, "AutoNumber should be false")
		assert.False(t, col.TimeZone, "TimeZone should be false")
		assert.Nil(t, col.Precision, "Precision should be nil")
	})

	t.Run("metadata slices are independent", func(t *testing.T) {
		columns := []string{"id", "name"}
		metadata1 := newEditableMetadata(columns)
		metadata2 := newEditableMetadata(columns)

		// Modify metadata1
		metadata1.Enabled = true
		metadata1.Schema = "public"
		metadata1.Table = "users"
		metadata1.PrimaryKeys = append(metadata1.PrimaryKeys, "id")
		metadata1.Columns[0].Editable = true

		// Verify metadata2 is unchanged
		assert.False(t, metadata2.Enabled, "metadata2 should be independent")
		assert.Empty(t, metadata2.Schema)
		assert.Empty(t, metadata2.Table)
		assert.Empty(t, metadata2.PrimaryKeys)
		assert.False(t, metadata2.Columns[0].Editable)
	})

	t.Run("columns slice has correct capacity", func(t *testing.T) {
		columns := []string{"id", "name", "email"}
		metadata := newEditableMetadata(columns)

		assert.Equal(t, 3, len(metadata.Columns))
		assert.GreaterOrEqual(t, cap(metadata.Columns), 3, "capacity should be at least length")
	})

	t.Run("primary keys slice is initialized with zero capacity", func(t *testing.T) {
		columns := []string{"id", "name"}
		metadata := newEditableMetadata(columns)

		assert.Equal(t, 0, len(metadata.PrimaryKeys))
		assert.Equal(t, 0, cap(metadata.PrimaryKeys), "should have zero capacity initially")
	})

	t.Run("can modify returned metadata", func(t *testing.T) {
		columns := []string{"id", "name"}
		metadata := newEditableMetadata(columns)

		// Verify we can modify the returned metadata
		metadata.Enabled = true
		metadata.Schema = "test_schema"
		metadata.Table = "test_table"
		metadata.Reason = "test reason"
		metadata.PrimaryKeys = append(metadata.PrimaryKeys, "id")
		metadata.Columns[0].Editable = true
		metadata.Columns[0].PrimaryKey = true
		metadata.Columns[0].DataType = "INTEGER"
		metadata.Capabilities = &MutationCapabilities{
			CanInsert: true,
			CanUpdate: true,
			CanDelete: true,
		}

		// Verify all modifications
		assert.True(t, metadata.Enabled)
		assert.Equal(t, "test_schema", metadata.Schema)
		assert.Equal(t, "test_table", metadata.Table)
		assert.Equal(t, "test reason", metadata.Reason)
		assert.Equal(t, []string{"id"}, metadata.PrimaryKeys)
		assert.True(t, metadata.Columns[0].Editable)
		assert.True(t, metadata.Columns[0].PrimaryKey)
		assert.Equal(t, "INTEGER", metadata.Columns[0].DataType)
		assert.NotNil(t, metadata.Capabilities)
		assert.True(t, metadata.Capabilities.CanInsert)
	})

	t.Run("large column list performance", func(t *testing.T) {
		// Create a large column list to verify performance
		columns := make([]string, 1000)
		for i := 0; i < 1000; i++ {
			columns[i] = "column_" + string(rune('0'+i%10))
		}

		metadata := newEditableMetadata(columns)

		assert.NotNil(t, metadata)
		assert.Equal(t, 1000, len(metadata.Columns))
		assert.False(t, metadata.Enabled)

		// Spot check a few columns
		assert.Equal(t, "column_0", metadata.Columns[0].Name)
		assert.Equal(t, "column_9", metadata.Columns[999].Name)
	})
}

// TestNewEditableMetadata_Integration tests how newEditableMetadata integrates with the broader system
func TestNewEditableMetadata_Integration(t *testing.T) {
	t.Run("used as base for enabled metadata", func(t *testing.T) {
		columns := []string{"id", "name", "email"}
		metadata := newEditableMetadata(columns)

		// Simulate what computeEditableMetadata would do for a valid editable query
		metadata.Enabled = true
		metadata.Schema = "public"
		metadata.Table = "users"
		metadata.PrimaryKeys = []string{"id"}
		metadata.Columns[0].Editable = true
		metadata.Columns[0].PrimaryKey = true
		metadata.Columns[0].DataType = "INTEGER"
		metadata.Columns[1].Editable = true
		metadata.Columns[1].DataType = "TEXT"
		metadata.Columns[2].Editable = true
		metadata.Columns[2].DataType = "TEXT"
		metadata.Capabilities = &MutationCapabilities{
			CanInsert: true,
			CanUpdate: true,
			CanDelete: true,
		}

		assert.True(t, metadata.Enabled)
		assert.Equal(t, "public", metadata.Schema)
		assert.Equal(t, "users", metadata.Table)
		assert.Contains(t, metadata.PrimaryKeys, "id")
		assert.NotNil(t, metadata.Capabilities)
	})

	t.Run("used as base for disabled metadata with reason", func(t *testing.T) {
		columns := []string{"count"}
		metadata := newEditableMetadata(columns)

		// Simulate what computeEditableMetadata would do for a non-editable query
		metadata.Reason = "Query contains GROUP BY aggregation"
		metadata.Capabilities = &MutationCapabilities{
			CanInsert: false,
			CanUpdate: false,
			CanDelete: false,
			Reason:    metadata.Reason,
		}

		assert.False(t, metadata.Enabled)
		assert.Equal(t, "Query contains GROUP BY aggregation", metadata.Reason)
		assert.NotNil(t, metadata.Capabilities)
		assert.False(t, metadata.Capabilities.CanInsert)
	})

	t.Run("used for empty query", func(t *testing.T) {
		columns := []string{}
		metadata := newEditableMetadata(columns)

		metadata.Reason = "Empty query"

		assert.False(t, metadata.Enabled)
		assert.Equal(t, "Empty query", metadata.Reason)
		assert.Empty(t, metadata.Columns)
	})

	t.Run("preserves column order", func(t *testing.T) {
		// Important for result set mapping
		columns := []string{"email", "name", "id", "created_at"}
		metadata := newEditableMetadata(columns)

		for i, col := range metadata.Columns {
			assert.Equal(t, columns[i], col.Name, "column order must be preserved")
		}
	})

	t.Run("supports database-specific behavior", func(t *testing.T) {
		columns := []string{"id", "data"}

		// MongoDB - not editable
		mongoMetadata := newEditableMetadata(columns)
		mongoMetadata.Reason = "MongoDB collections are not directly editable via SQL interface"
		mongoMetadata.Capabilities = &MutationCapabilities{
			CanInsert: false,
			CanUpdate: false,
			CanDelete: false,
		}
		assert.False(t, mongoMetadata.Enabled)

		// Elasticsearch - not editable
		esMetadata := newEditableMetadata(columns)
		esMetadata.Reason = "Elasticsearch indices are not directly editable"
		esMetadata.Capabilities = &MutationCapabilities{
			CanInsert: false,
			CanUpdate: false,
			CanDelete: false,
		}
		assert.False(t, esMetadata.Enabled)

		// ClickHouse - immutable
		chMetadata := newEditableMetadata(columns)
		chMetadata.Reason = "ClickHouse tables are immutable and not directly editable"
		assert.False(t, chMetadata.Enabled)

		// SQLite/Postgres/MySQL - potentially editable
		sqlMetadata := newEditableMetadata(columns)
		sqlMetadata.Enabled = true
		sqlMetadata.Schema = "main"
		sqlMetadata.Table = "test"
		assert.True(t, sqlMetadata.Enabled)
	})
}

// TestEditableMetadata_MemoryAllocation verifies memory efficiency
func TestEditableMetadata_MemoryAllocation(t *testing.T) {
	t.Run("slices are not over-allocated", func(t *testing.T) {
		columns := []string{"id", "name"}
		metadata := newEditableMetadata(columns)

		// Columns should have exact capacity
		assert.Equal(t, len(columns), cap(metadata.Columns))

		// PrimaryKeys should start empty
		assert.Equal(t, 0, len(metadata.PrimaryKeys))
		assert.Equal(t, 0, cap(metadata.PrimaryKeys))
	})

	t.Run("no shared backing arrays", func(t *testing.T) {
		columns := []string{"a", "b", "c"}
		metadata1 := newEditableMetadata(columns)
		metadata2 := newEditableMetadata(columns)

		// Modify one, ensure the other is unaffected
		metadata1.Columns[0].Name = "modified"

		assert.NotEqual(t, metadata1.Columns[0].Name, metadata2.Columns[0].Name)
		assert.Equal(t, "a", metadata2.Columns[0].Name)
	})
}
