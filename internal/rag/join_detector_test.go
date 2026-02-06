package rag

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

func TestNewJoinDetector(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)
	require.NotNil(t, jd)
	assert.Equal(t, logger, jd.logger)
}

func TestJoinDetector_DetectTables(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	t.Run("returns empty for nil context", func(t *testing.T) {
		tables := jd.DetectTables("SELECT * FROM users", nil)
		assert.Empty(t, tables)
	})

	t.Run("returns empty for empty context", func(t *testing.T) {
		ctx := &QueryContext{}
		tables := jd.DetectTables("SELECT * FROM users", ctx)
		assert.Empty(t, tables)
	})

	t.Run("detects table from query", func(t *testing.T) {
		ctx := &QueryContext{
			RelevantSchemas: []SchemaContext{
				{TableName: "users"},
				{TableName: "orders"},
			},
		}
		tables := jd.DetectTables("SELECT * FROM users", ctx)
		assert.Contains(t, tables, "users")
		assert.NotContains(t, tables, "orders")
	})

	t.Run("detects multiple tables", func(t *testing.T) {
		ctx := &QueryContext{
			RelevantSchemas: []SchemaContext{
				{TableName: "users"},
				{TableName: "orders"},
				{TableName: "products"},
			},
		}
		tables := jd.DetectTables("SELECT * FROM users JOIN orders", ctx)
		assert.Contains(t, tables, "users")
		assert.Contains(t, tables, "orders")
		assert.NotContains(t, tables, "products")
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		ctx := &QueryContext{
			RelevantSchemas: []SchemaContext{
				{TableName: "Users"},
			},
		}
		tables := jd.DetectTables("SELECT * FROM users", ctx)
		assert.Contains(t, tables, "Users")
	})

	t.Run("detects tables with FK columns", func(t *testing.T) {
		ctx := &QueryContext{
			RelevantSchemas: []SchemaContext{
				{
					TableName: "orders",
					Columns: []ColumnInfo{
						{Name: "user_id", IsForeignKey: true},
					},
				},
			},
		}
		tables := jd.DetectTables("SELECT * WHERE user_id = 1", ctx)
		assert.Contains(t, tables, "orders")
	})
}

func TestJoinDetector_FindJoinPath(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	t.Run("returns empty for no tables", func(t *testing.T) {
		path := jd.FindJoinPath([]string{}, []SchemaContext{})
		assert.Empty(t, path.Tables)
		assert.Empty(t, path.Joins)
	})

	t.Run("returns single table without joins", func(t *testing.T) {
		path := jd.FindJoinPath([]string{"users"}, []SchemaContext{})
		assert.Equal(t, []string{"users"}, path.Tables)
		assert.Empty(t, path.Joins)
		assert.Equal(t, 0, path.Cost)
	})

	t.Run("finds join path with FK relationships", func(t *testing.T) {
		schemas := []SchemaContext{
			{
				TableName: "users",
				Relationships: []RelationshipInfo{
					{
						Type:          "one-to-many",
						TargetTable:   "orders",
						LocalColumn:   "id",
						ForeignColumn: "user_id",
					},
				},
			},
			{
				TableName: "orders",
			},
		}
		tables := []string{"users", "orders"}

		path := jd.FindJoinPath(tables, schemas)
		assert.Len(t, path.Tables, 2)
		assert.Contains(t, path.Tables, "users")
		assert.Contains(t, path.Tables, "orders")
	})

	t.Run("finds shortest path with multiple hops", func(t *testing.T) {
		schemas := []SchemaContext{
			{
				TableName: "users",
				Relationships: []RelationshipInfo{
					{
						Type:          "one-to-many",
						TargetTable:   "orders",
						LocalColumn:   "id",
						ForeignColumn: "user_id",
					},
				},
			},
			{
				TableName: "orders",
				Relationships: []RelationshipInfo{
					{
						Type:          "one-to-many",
						TargetTable:   "order_items",
						LocalColumn:   "id",
						ForeignColumn: "order_id",
					},
				},
			},
			{
				TableName: "order_items",
			},
		}
		tables := []string{"users", "order_items"}

		path := jd.FindJoinPath(tables, schemas)
		// Should find path: users -> orders -> order_items
		assert.LessOrEqual(t, len(path.Joins), 2)
	})

	t.Run("handles unreachable tables", func(t *testing.T) {
		schemas := []SchemaContext{
			{TableName: "users"},
			{TableName: "products"}, // No relationship to users
		}
		tables := []string{"users", "products"}

		path := jd.FindJoinPath(tables, schemas)
		// Should include both tables even if no path found
		assert.Contains(t, path.Tables, "users")
		assert.Contains(t, path.Tables, "products")
	})
}

func TestJoinDetector_InferJoinType(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	tests := []struct {
		relType  string
		expected string
	}{
		{"one-to-many", "LEFT"},
		{"1:n", "LEFT"},
		{"many-to-one", "INNER"},
		{"n:1", "INNER"},
		{"many-to-many", "INNER"},
		{"n:n", "INNER"},
		{"one-to-one", "INNER"},
		{"1:1", "INNER"},
		{"unknown", "INNER"},
	}

	for _, tc := range tests {
		t.Run(tc.relType, func(t *testing.T) {
			result := jd.inferJoinType(tc.relType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinDetector_InferReverseJoinType(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	tests := []struct {
		relType  string
		expected string
	}{
		{"one-to-many", "INNER"},
		{"1:n", "INNER"},
		{"many-to-one", "LEFT"},
		{"n:1", "LEFT"},
		{"unknown", "INNER"},
	}

	for _, tc := range tests {
		t.Run(tc.relType, func(t *testing.T) {
			result := jd.inferReverseJoinType(tc.relType)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJoinDetector_InferTargetTable(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	schemas := []SchemaContext{
		{TableName: "users"},
		{TableName: "orders"},
		{TableName: "categories"},
		{TableName: "statuses"},
	}

	t.Run("infers plural form (user_id -> users)", func(t *testing.T) {
		result := jd.inferTargetTable("user_id", schemas)
		assert.Equal(t, "users", result)
	})

	t.Run("infers plural form (order_id -> orders)", func(t *testing.T) {
		result := jd.inferTargetTable("order_id", schemas)
		assert.Equal(t, "orders", result)
	})

	t.Run("infers -ies plural (category_id -> categories)", func(t *testing.T) {
		result := jd.inferTargetTable("category_id", schemas)
		assert.Equal(t, "categories", result)
	})

	t.Run("infers -es plural (status_id -> statuses)", func(t *testing.T) {
		result := jd.inferTargetTable("status_id", schemas)
		assert.Equal(t, "statuses", result)
	})

	t.Run("returns empty for unknown column", func(t *testing.T) {
		result := jd.inferTargetTable("foo_id", schemas)
		assert.Empty(t, result)
	})

	t.Run("returns empty for non-id column", func(t *testing.T) {
		result := jd.inferTargetTable("username", schemas)
		assert.Empty(t, result)
	})
}

func TestJoinDetector_GenerateJoinConditions(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	t.Run("returns joins from path", func(t *testing.T) {
		path := JoinPath{
			Tables: []string{"users", "orders"},
			Joins: []JoinCondition{
				{
					LeftTable:   "users",
					RightTable:  "orders",
					LeftColumn:  "id",
					RightColumn: "user_id",
					JoinType:    "LEFT",
				},
			},
		}

		conditions := jd.GenerateJoinConditions(path)
		require.Len(t, conditions, 1)
		assert.Equal(t, "users", conditions[0].LeftTable)
		assert.Equal(t, "orders", conditions[0].RightTable)
		assert.Equal(t, "id", conditions[0].LeftColumn)
		assert.Equal(t, "user_id", conditions[0].RightColumn)
		assert.Equal(t, "LEFT", conditions[0].JoinType)
	})

	t.Run("returns empty for empty path", func(t *testing.T) {
		path := JoinPath{}
		conditions := jd.GenerateJoinConditions(path)
		assert.Empty(t, conditions)
	})
}

func TestJoinDetector_BuildRelationshipGraph(t *testing.T) {
	logger := newTestLogger()
	jd := NewJoinDetector(logger)

	t.Run("builds graph from relationships", func(t *testing.T) {
		schemas := []SchemaContext{
			{
				TableName: "users",
				Relationships: []RelationshipInfo{
					{
						Type:          "one-to-many",
						TargetTable:   "orders",
						LocalColumn:   "id",
						ForeignColumn: "user_id",
					},
				},
			},
			{
				TableName: "orders",
			},
		}

		graph := jd.buildRelationshipGraph(schemas)
		assert.Contains(t, graph, "users")
		assert.Contains(t, graph, "orders")
		assert.Len(t, graph["users"], 1)
		assert.Equal(t, "orders", graph["users"][0].targetTable)
	})

	t.Run("adds bidirectional edges", func(t *testing.T) {
		schemas := []SchemaContext{
			{
				TableName: "users",
				Relationships: []RelationshipInfo{
					{
						Type:          "one-to-many",
						TargetTable:   "orders",
						LocalColumn:   "id",
						ForeignColumn: "user_id",
					},
				},
			},
		}

		graph := jd.buildRelationshipGraph(schemas)
		// Should have edge from orders back to users
		assert.Contains(t, graph, "orders")
		found := false
		for _, edge := range graph["orders"] {
			if edge.targetTable == "users" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected reverse edge from orders to users")
	})

	t.Run("handles empty schemas", func(t *testing.T) {
		graph := jd.buildRelationshipGraph([]SchemaContext{})
		assert.Empty(t, graph)
	})

	t.Run("handles FK columns for inference", func(t *testing.T) {
		schemas := []SchemaContext{
			{
				TableName: "orders",
				Columns: []ColumnInfo{
					{Name: "user_id", IsForeignKey: true},
				},
			},
			{
				TableName: "users",
			},
		}

		graph := jd.buildRelationshipGraph(schemas)
		// Should infer edge from orders to users via user_id
		assert.Contains(t, graph, "orders")
		found := false
		for _, edge := range graph["orders"] {
			if edge.targetTable == "users" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected inferred edge from orders to users")
	})
}

func TestMin(t *testing.T) {
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(2, 1))
	assert.Equal(t, 5, min(5, 5))
	assert.Equal(t, -1, min(-1, 0))
}
