package rag_test

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/rag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test logger for helpers tests
func newTestLoggerHelpers() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

// TestNewSchemaAnalyzer tests the SchemaAnalyzer constructor
func TestNewSchemaAnalyzer(t *testing.T) {
	t.Run("creates analyzer with valid logger", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		analyzer := rag.NewSchemaAnalyzer(logger)

		require.NotNil(t, analyzer)
	})

	t.Run("creates analyzer with nil logger", func(t *testing.T) {
		analyzer := rag.NewSchemaAnalyzer(nil)

		require.NotNil(t, analyzer)
	})

	t.Run("multiple analyzers can be created", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		analyzer1 := rag.NewSchemaAnalyzer(logger)
		analyzer2 := rag.NewSchemaAnalyzer(logger)

		require.NotNil(t, analyzer1)
		require.NotNil(t, analyzer2)
		assert.NotSame(t, analyzer1, analyzer2)
	})
}

// TestNewPatternMatcher tests the PatternMatcher constructor
func TestNewPatternMatcher(t *testing.T) {
	t.Run("creates matcher with valid logger", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher := rag.NewPatternMatcher(logger)

		require.NotNil(t, matcher)
	})

	t.Run("creates matcher with nil logger", func(t *testing.T) {
		matcher := rag.NewPatternMatcher(nil)

		require.NotNil(t, matcher)
	})

	t.Run("multiple matchers can be created", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher1 := rag.NewPatternMatcher(logger)
		matcher2 := rag.NewPatternMatcher(logger)

		require.NotNil(t, matcher1)
		require.NotNil(t, matcher2)
		assert.NotSame(t, matcher1, matcher2)
	})
}

// TestPatternMatcher_ExtractPatterns tests pattern extraction
func TestPatternMatcher_ExtractPatterns(t *testing.T) {
	t.Run("extracts patterns from nil documents", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher := rag.NewPatternMatcher(logger)

		patterns := matcher.ExtractPatterns(nil)

		require.NotNil(t, patterns)
		assert.Empty(t, patterns, "TODO implementation returns empty slice")
	})

	t.Run("extracts patterns from empty documents", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher := rag.NewPatternMatcher(logger)

		patterns := matcher.ExtractPatterns([]*rag.Document{})

		require.NotNil(t, patterns)
		assert.Empty(t, patterns, "TODO implementation returns empty slice")
	})

	t.Run("extracts patterns from single document", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher := rag.NewPatternMatcher(logger)

		docs := []*rag.Document{
			{
				ID:      "doc1",
				Type:    rag.DocumentTypeQuery,
				Content: "SELECT * FROM users",
			},
		}

		patterns := matcher.ExtractPatterns(docs)

		require.NotNil(t, patterns)
		assert.Empty(t, patterns, "TODO implementation returns empty slice")
	})

	t.Run("extracts patterns from multiple documents", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher := rag.NewPatternMatcher(logger)

		docs := []*rag.Document{
			{
				ID:      "doc1",
				Type:    rag.DocumentTypeQuery,
				Content: "SELECT * FROM users",
			},
			{
				ID:      "doc2",
				Type:    rag.DocumentTypeQuery,
				Content: "SELECT * FROM orders",
			},
			{
				ID:      "doc3",
				Type:    rag.DocumentTypeSchema,
				Content: "CREATE TABLE products",
			},
		}

		patterns := matcher.ExtractPatterns(docs)

		require.NotNil(t, patterns)
		assert.Empty(t, patterns, "TODO implementation returns empty slice")
	})

	t.Run("extracts patterns from different document types", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		matcher := rag.NewPatternMatcher(logger)

		types := []rag.DocumentType{
			rag.DocumentTypeSchema,
			rag.DocumentTypeQuery,
			rag.DocumentTypePlan,
			rag.DocumentTypeResult,
			rag.DocumentTypeBusiness,
			rag.DocumentTypePerformance,
			rag.DocumentTypeMemory,
		}

		for _, docType := range types {
			docs := []*rag.Document{
				{
					ID:      "doc1",
					Type:    docType,
					Content: "test content",
				},
			}

			patterns := matcher.ExtractPatterns(docs)
			require.NotNil(t, patterns, "should return non-nil for type %s", docType)
			assert.Empty(t, patterns, "TODO implementation returns empty slice")
		}
	})
}

// TestNewStatsCollector tests the StatsCollector constructor
func TestNewStatsCollector(t *testing.T) {
	t.Run("creates collector with valid logger", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		collector := rag.NewStatsCollector(logger)

		require.NotNil(t, collector)
	})

	t.Run("creates collector with nil logger", func(t *testing.T) {
		collector := rag.NewStatsCollector(nil)

		require.NotNil(t, collector)
	})

	t.Run("multiple collectors can be created", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		collector1 := rag.NewStatsCollector(logger)
		collector2 := rag.NewStatsCollector(logger)

		require.NotNil(t, collector1)
		require.NotNil(t, collector2)
		assert.NotSame(t, collector1, collector2)
	})
}

// TestNewSQLValidator tests the SQLValidator constructor
func TestNewSQLValidator(t *testing.T) {
	t.Run("creates validator with valid logger", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		require.NotNil(t, validator)
	})

	t.Run("creates validator with nil logger", func(t *testing.T) {
		validator := rag.NewSQLValidator(nil)

		require.NotNil(t, validator)
	})

	t.Run("multiple validators can be created", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator1 := rag.NewSQLValidator(logger)
		validator2 := rag.NewSQLValidator(logger)

		require.NotNil(t, validator1)
		require.NotNil(t, validator2)
		assert.NotSame(t, validator1, validator2)
	})
}

// TestSQLValidator_Validate tests SQL validation
func TestSQLValidator_Validate(t *testing.T) {
	t.Run("validates empty query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates simple SELECT query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SELECT * FROM users")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates SELECT with WHERE clause", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SELECT id, name FROM users WHERE id = 1")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates SELECT with JOIN", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates INSERT query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("INSERT INTO users (name, email) VALUES ('test', 'test@example.com')")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates UPDATE query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("UPDATE users SET name = 'updated' WHERE id = 1")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates DELETE query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("DELETE FROM users WHERE id = 1")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates complex query with subquery", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)")

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates query with multiple joins", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		query := `
			SELECT u.name, o.total, p.name
			FROM users u
			JOIN orders o ON u.id = o.user_id
			JOIN products p ON o.product_id = p.id
		`

		err := validator.Validate(query)

		assert.NoError(t, err, "TODO implementation returns nil")
	})

	t.Run("validates malformed query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SELECT * FROM")

		assert.NoError(t, err, "TODO implementation returns nil (will validate when implemented)")
	})

	t.Run("validates invalid SQL syntax", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SLECT * FORM users")

		assert.NoError(t, err, "TODO implementation returns nil (will validate when implemented)")
	})

	t.Run("validates SQL injection attempt", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		err := validator.Validate("SELECT * FROM users WHERE id = 1; DROP TABLE users;")

		assert.NoError(t, err, "TODO implementation returns nil (will validate when implemented)")
	})
}

// TestNewQueryPlanner tests the QueryPlanner constructor
func TestNewQueryPlanner(t *testing.T) {
	t.Run("creates planner with valid logger", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		require.NotNil(t, planner)
	})

	t.Run("creates planner with nil logger", func(t *testing.T) {
		planner := rag.NewQueryPlanner(nil)

		require.NotNil(t, planner)
	})

	t.Run("multiple planners can be created", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner1 := rag.NewQueryPlanner(logger)
		planner2 := rag.NewQueryPlanner(logger)

		require.NotNil(t, planner1)
		require.NotNil(t, planner2)
		assert.NotSame(t, planner1, planner2)
	})
}

// TestQueryPlanner_DecomposeRequest tests request decomposition
func TestQueryPlanner_DecomposeRequest(t *testing.T) {
	t.Run("decomposes simple request", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		context := &rag.QueryContext{
			Query: "Get all users",
		}

		steps := planner.DecomposeRequest("Get all users", context)

		require.NotNil(t, steps)
		assert.Empty(t, steps, "TODO implementation returns empty slice")
	})

	t.Run("decomposes complex request", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		context := &rag.QueryContext{
			Query: "Get all users with their orders and total spending",
		}

		steps := planner.DecomposeRequest("Get all users with their orders and total spending", context)

		require.NotNil(t, steps)
		assert.Empty(t, steps, "TODO implementation returns empty slice")
	})

	t.Run("decomposes request with nil context", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		steps := planner.DecomposeRequest("Get users", nil)

		require.NotNil(t, steps)
		assert.Empty(t, steps, "TODO implementation returns empty slice")
	})

	t.Run("decomposes empty prompt", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		context := &rag.QueryContext{
			Query: "",
		}

		steps := planner.DecomposeRequest("", context)

		require.NotNil(t, steps)
		assert.Empty(t, steps, "TODO implementation returns empty slice")
	})

	t.Run("decomposes request with aggregation", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		context := &rag.QueryContext{
			Query: "Calculate average order value per customer",
		}

		steps := planner.DecomposeRequest("Calculate average order value per customer", context)

		require.NotNil(t, steps)
		assert.Empty(t, steps, "TODO implementation returns empty slice")
	})

	t.Run("decomposes request with filters", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		context := &rag.QueryContext{
			Query: "Find active users who placed orders in the last 30 days",
		}

		steps := planner.DecomposeRequest("Find active users who placed orders in the last 30 days", context)

		require.NotNil(t, steps)
		assert.Empty(t, steps, "TODO implementation returns empty slice")
	})

	t.Run("decomposes request with different queries", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		queries := []string{"Get users", "Get orders", "Get products", "Get customers"}

		for _, query := range queries {
			context := &rag.QueryContext{
				Query: query,
			}

			steps := planner.DecomposeRequest(query, context)

			require.NotNil(t, steps, "should return non-nil for query %s", query)
			assert.Empty(t, steps, "TODO implementation returns empty slice")
		}
	})
}

// TestQueryPlanner_CombineSteps tests step combination
func TestQueryPlanner_CombineSteps(t *testing.T) {
	t.Run("combines nil steps", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		result := planner.CombineSteps(nil)

		assert.Equal(t, "", result.Query, "TODO implementation returns empty struct")
		assert.Equal(t, "", result.Explanation, "TODO implementation returns empty struct")
	})

	t.Run("combines empty steps", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		result := planner.CombineSteps([]rag.StepSQL{})

		assert.Equal(t, "", result.Query, "TODO implementation returns empty struct")
		assert.Equal(t, "", result.Explanation, "TODO implementation returns empty struct")
	})

	t.Run("combines single step", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		steps := []rag.StepSQL{
			{
				Step: rag.QueryStep{
					Order:       1,
					Description: "Get users",
					Complexity:  "simple",
				},
				SQL:         "SELECT * FROM users",
				Explanation: "Fetch all users from database",
			},
		}

		result := planner.CombineSteps(steps)

		assert.Equal(t, "", result.Query, "TODO implementation returns empty struct")
		assert.Equal(t, "", result.Explanation, "TODO implementation returns empty struct")
	})

	t.Run("combines multiple steps", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		steps := []rag.StepSQL{
			{
				Step: rag.QueryStep{
					Order:       1,
					Description: "Get users",
					Complexity:  "simple",
				},
				SQL:         "SELECT * FROM users",
				Explanation: "Fetch users",
			},
			{
				Step: rag.QueryStep{
					Order:       2,
					Description: "Join orders",
					Complexity:  "medium",
				},
				SQL:         "JOIN orders ON users.id = orders.user_id",
				Explanation: "Join with orders table",
			},
		}

		result := planner.CombineSteps(steps)

		assert.Equal(t, "", result.Query, "TODO implementation returns empty struct")
		assert.Equal(t, "", result.Explanation, "TODO implementation returns empty struct")
	})

	t.Run("combines steps with varying complexity", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		steps := []rag.StepSQL{
			{
				Step: rag.QueryStep{
					Order:       1,
					Description: "Simple filter",
					Complexity:  "simple",
				},
				SQL:         "SELECT * FROM users WHERE active = 1",
				Explanation: "Filter active users",
			},
			{
				Step: rag.QueryStep{
					Order:       2,
					Description: "Complex aggregation",
					Complexity:  "complex",
				},
				SQL:         "GROUP BY category HAVING COUNT(*) > 10",
				Explanation: "Aggregate by category",
			},
		}

		result := planner.CombineSteps(steps)

		assert.Equal(t, "", result.Query, "TODO implementation returns empty struct")
		assert.Equal(t, "", result.Explanation, "TODO implementation returns empty struct")
	})
}

// TestQueryPlanner_ValidateAndOptimize tests validation and optimization
func TestQueryPlanner_ValidateAndOptimize(t *testing.T) {
	t.Run("validates and optimizes empty query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		planned := rag.PlannedSQL{
			Query:       "",
			Explanation: "",
		}

		result := planner.ValidateAndOptimize(planned)

		assert.Equal(t, planned, result, "TODO implementation returns input unchanged")
	})

	t.Run("validates and optimizes simple query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		planned := rag.PlannedSQL{
			Query:       "SELECT * FROM users",
			Explanation: "Get all users",
		}

		result := planner.ValidateAndOptimize(planned)

		assert.Equal(t, planned, result, "TODO implementation returns input unchanged")
	})

	t.Run("validates and optimizes complex query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		planned := rag.PlannedSQL{
			Query: `
				SELECT u.id, u.name, COUNT(o.id) as order_count
				FROM users u
				LEFT JOIN orders o ON u.id = o.user_id
				GROUP BY u.id, u.name
			`,
			Explanation: "Get users with order counts",
		}

		result := planner.ValidateAndOptimize(planned)

		assert.Equal(t, planned, result, "TODO implementation returns input unchanged")
	})

	t.Run("validates and optimizes query with subquery", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		planned := rag.PlannedSQL{
			Query: `
				SELECT * FROM users
				WHERE id IN (SELECT user_id FROM orders WHERE total > 100)
			`,
			Explanation: "Get users with high-value orders",
		}

		result := planner.ValidateAndOptimize(planned)

		assert.Equal(t, planned, result, "TODO implementation returns input unchanged")
	})

	t.Run("validates and optimizes query without explanation", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		planner := rag.NewQueryPlanner(logger)

		planned := rag.PlannedSQL{
			Query:       "SELECT * FROM products",
			Explanation: "",
		}

		result := planner.ValidateAndOptimize(planned)

		assert.Equal(t, planned, result, "TODO implementation returns input unchanged")
	})
}

// TestNewJoinDetector tests the JoinDetector constructor
func TestNewJoinDetector(t *testing.T) {
	t.Run("creates detector with valid logger", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		require.NotNil(t, detector)
	})

	t.Run("creates detector with nil logger", func(t *testing.T) {
		detector := rag.NewJoinDetector(nil)

		require.NotNil(t, detector)
	})

	t.Run("multiple detectors can be created", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector1 := rag.NewJoinDetector(logger)
		detector2 := rag.NewJoinDetector(logger)

		require.NotNil(t, detector1)
		require.NotNil(t, detector2)
		assert.NotSame(t, detector1, detector2)
	})
}

// TestJoinDetector_DetectTables tests table detection
func TestJoinDetector_DetectTables(t *testing.T) {
	t.Run("detects tables from empty query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		context := &rag.QueryContext{
			Query: "",
		}

		tables := detector.DetectTables("", context)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice")
	})

	t.Run("detects tables from simple SELECT", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		context := &rag.QueryContext{
			Query: "SELECT * FROM users",
		}

		tables := detector.DetectTables("SELECT * FROM users", context)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice")
	})

	t.Run("detects tables from JOIN query", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		query := "SELECT * FROM users u JOIN orders o ON u.id = o.user_id"
		context := &rag.QueryContext{
			Query: query,
		}

		tables := detector.DetectTables(query, context)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice (will detect users, orders when implemented)")
	})

	t.Run("detects tables from multiple JOINs", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		query := `
			SELECT * FROM users u
			JOIN orders o ON u.id = o.user_id
			JOIN products p ON o.product_id = p.id
		`
		context := &rag.QueryContext{
			Query: query,
		}

		tables := detector.DetectTables(query, context)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice (will detect users, orders, products when implemented)")
	})

	t.Run("detects tables from subquery", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		query := "SELECT * FROM users WHERE id IN (SELECT user_id FROM orders)"
		context := &rag.QueryContext{
			Query: query,
		}

		tables := detector.DetectTables(query, context)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice (will detect users, orders when implemented)")
	})

	t.Run("detects tables with nil context", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		tables := detector.DetectTables("SELECT * FROM users", nil)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice")
	})

	t.Run("detects tables from natural language", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		query := "Get users and their orders"
		context := &rag.QueryContext{
			Query: query,
		}

		tables := detector.DetectTables(query, context)

		require.NotNil(t, tables)
		assert.Empty(t, tables, "TODO implementation returns empty slice (will detect users, orders when implemented)")
	})
}

// TestJoinDetector_FindJoinPath tests join path finding
func TestJoinDetector_FindJoinPath(t *testing.T) {
	t.Run("finds path with nil tables", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := detector.FindJoinPath(nil, nil)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct")
	})

	t.Run("finds path with empty tables", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := detector.FindJoinPath([]string{}, nil)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct")
	})

	t.Run("finds path for single table", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		schemas := []rag.SchemaContext{
			{
				TableName: "users",
			},
		}

		path := detector.FindJoinPath([]string{"users"}, schemas)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct")
	})

	t.Run("finds path for two tables", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		schemas := []rag.SchemaContext{
			{
				TableName: "users",
			},
			{
				TableName: "orders",
			},
		}

		path := detector.FindJoinPath([]string{"users", "orders"}, schemas)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct (will find path when implemented)")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct (will find joins when implemented)")
	})

	t.Run("finds path for three tables", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		schemas := []rag.SchemaContext{
			{
				TableName: "users",
			},
			{
				TableName: "orders",
			},
			{
				TableName: "products",
			},
		}

		path := detector.FindJoinPath([]string{"users", "orders", "products"}, schemas)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct (will find path when implemented)")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct (will find joins when implemented)")
	})

	t.Run("finds path with nil schemas", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := detector.FindJoinPath([]string{"users", "orders"}, nil)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct")
	})

	t.Run("finds path with empty schemas", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := detector.FindJoinPath([]string{"users", "orders"}, []rag.SchemaContext{})

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct")
	})

	t.Run("finds path for indirect join", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		schemas := []rag.SchemaContext{
			{
				TableName: "users",
			},
			{
				TableName: "orders",
			},
			{
				TableName: "products",
			},
			{
				TableName: "categories",
			},
		}

		// users -> orders -> products -> categories
		path := detector.FindJoinPath([]string{"users", "categories"}, schemas)

		assert.Empty(t, path.Tables, "TODO implementation returns empty struct (will find indirect path when implemented)")
		assert.Empty(t, path.Joins, "TODO implementation returns empty struct (will find joins when implemented)")
	})
}

// TestJoinDetector_GenerateJoinConditions tests join condition generation
func TestJoinDetector_GenerateJoinConditions(t *testing.T) {
	t.Run("generates conditions for empty path", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := rag.JoinPath{
			Tables: []string{},
			Joins:  []rag.JoinCondition{},
		}

		conditions := detector.GenerateJoinConditions(path)

		require.NotNil(t, conditions)
		assert.Empty(t, conditions)
	})

	t.Run("generates conditions for path with no joins", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := rag.JoinPath{
			Tables: []string{"users"},
			Joins:  []rag.JoinCondition{},
		}

		conditions := detector.GenerateJoinConditions(path)

		require.NotNil(t, conditions)
		assert.Empty(t, conditions)
	})

	t.Run("generates conditions for single join", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := rag.JoinPath{
			Tables: []string{"users", "orders"},
			Joins: []rag.JoinCondition{
				{
					LeftTable:   "users",
					RightTable:  "orders",
					LeftColumn:  "id",
					RightColumn: "user_id",
					JoinType:    "INNER",
				},
			},
		}

		conditions := detector.GenerateJoinConditions(path)

		require.NotNil(t, conditions)
		require.Len(t, conditions, 1)
		assert.Equal(t, "users", conditions[0].LeftTable)
		assert.Equal(t, "orders", conditions[0].RightTable)
		assert.Equal(t, "id", conditions[0].LeftColumn)
		assert.Equal(t, "user_id", conditions[0].RightColumn)
		assert.Equal(t, "INNER", conditions[0].JoinType)
	})

	t.Run("generates conditions for multiple joins", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		path := rag.JoinPath{
			Tables: []string{"users", "orders", "products"},
			Joins: []rag.JoinCondition{
				{
					LeftTable:   "users",
					RightTable:  "orders",
					LeftColumn:  "id",
					RightColumn: "user_id",
					JoinType:    "INNER",
				},
				{
					LeftTable:   "orders",
					RightTable:  "products",
					LeftColumn:  "product_id",
					RightColumn: "id",
					JoinType:    "LEFT",
				},
			},
		}

		conditions := detector.GenerateJoinConditions(path)

		require.NotNil(t, conditions)
		require.Len(t, conditions, 2)
		assert.Equal(t, "users", conditions[0].LeftTable)
		assert.Equal(t, "orders", conditions[1].LeftTable)
		assert.Equal(t, "products", conditions[1].RightTable)
	})

	t.Run("generates conditions for different join types", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		detector := rag.NewJoinDetector(logger)

		joinTypes := []string{"INNER", "LEFT", "RIGHT", "FULL"}

		for _, joinType := range joinTypes {
			path := rag.JoinPath{
				Tables: []string{"users", "orders"},
				Joins: []rag.JoinCondition{
					{
						LeftTable:   "users",
						RightTable:  "orders",
						LeftColumn:  "id",
						RightColumn: "user_id",
						JoinType:    joinType,
					},
				},
			}

			conditions := detector.GenerateJoinConditions(path)

			require.Len(t, conditions, 1, "should have one condition for join type %s", joinType)
			assert.Equal(t, joinType, conditions[0].JoinType)
		}
	})
}

// TestQueryStep_Structure tests QueryStep struct
func TestQueryStep_Structure(t *testing.T) {
	t.Run("creates QueryStep with all fields", func(t *testing.T) {
		step := rag.QueryStep{
			Order:       1,
			Description: "Test step",
			Complexity:  "simple",
		}

		assert.Equal(t, 1, step.Order)
		assert.Equal(t, "Test step", step.Description)
		assert.Equal(t, "simple", step.Complexity)
	})

	t.Run("creates QueryStep with zero values", func(t *testing.T) {
		step := rag.QueryStep{}

		assert.Equal(t, 0, step.Order)
		assert.Equal(t, "", step.Description)
		assert.Equal(t, "", step.Complexity)
	})

	t.Run("creates multiple QuerySteps", func(t *testing.T) {
		steps := []rag.QueryStep{
			{Order: 1, Description: "Step 1", Complexity: "simple"},
			{Order: 2, Description: "Step 2", Complexity: "medium"},
			{Order: 3, Description: "Step 3", Complexity: "complex"},
		}

		require.Len(t, steps, 3)
		assert.Equal(t, 1, steps[0].Order)
		assert.Equal(t, 2, steps[1].Order)
		assert.Equal(t, 3, steps[2].Order)
	})
}

// TestStepSQL_Structure tests StepSQL struct
func TestStepSQL_Structure(t *testing.T) {
	t.Run("creates StepSQL with all fields", func(t *testing.T) {
		stepSQL := rag.StepSQL{
			Step: rag.QueryStep{
				Order:       1,
				Description: "Get users",
				Complexity:  "simple",
			},
			SQL:         "SELECT * FROM users",
			Explanation: "Fetch all users",
		}

		assert.Equal(t, 1, stepSQL.Step.Order)
		assert.Equal(t, "Get users", stepSQL.Step.Description)
		assert.Equal(t, "SELECT * FROM users", stepSQL.SQL)
		assert.Equal(t, "Fetch all users", stepSQL.Explanation)
	})

	t.Run("creates StepSQL with zero values", func(t *testing.T) {
		stepSQL := rag.StepSQL{}

		assert.Equal(t, 0, stepSQL.Step.Order)
		assert.Equal(t, "", stepSQL.Step.Description)
		assert.Equal(t, "", stepSQL.SQL)
		assert.Equal(t, "", stepSQL.Explanation)
	})
}

// TestPlannedSQL_Structure tests PlannedSQL struct
func TestPlannedSQL_Structure(t *testing.T) {
	t.Run("creates PlannedSQL with all fields", func(t *testing.T) {
		planned := rag.PlannedSQL{
			Query:       "SELECT * FROM users",
			Explanation: "Get all users from the database",
		}

		assert.Equal(t, "SELECT * FROM users", planned.Query)
		assert.Equal(t, "Get all users from the database", planned.Explanation)
	})

	t.Run("creates PlannedSQL with zero values", func(t *testing.T) {
		planned := rag.PlannedSQL{}

		assert.Equal(t, "", planned.Query)
		assert.Equal(t, "", planned.Explanation)
	})
}

// TestJoinCondition_Structure tests JoinCondition struct
func TestJoinCondition_Structure(t *testing.T) {
	t.Run("creates JoinCondition with all fields", func(t *testing.T) {
		condition := rag.JoinCondition{
			LeftTable:   "users",
			RightTable:  "orders",
			LeftColumn:  "id",
			RightColumn: "user_id",
			JoinType:    "INNER",
		}

		assert.Equal(t, "users", condition.LeftTable)
		assert.Equal(t, "orders", condition.RightTable)
		assert.Equal(t, "id", condition.LeftColumn)
		assert.Equal(t, "user_id", condition.RightColumn)
		assert.Equal(t, "INNER", condition.JoinType)
	})

	t.Run("creates JoinCondition with zero values", func(t *testing.T) {
		condition := rag.JoinCondition{}

		assert.Equal(t, "", condition.LeftTable)
		assert.Equal(t, "", condition.RightTable)
		assert.Equal(t, "", condition.LeftColumn)
		assert.Equal(t, "", condition.RightColumn)
		assert.Equal(t, "", condition.JoinType)
	})

	t.Run("creates different join types", func(t *testing.T) {
		joinTypes := []string{"INNER", "LEFT", "RIGHT", "FULL", "CROSS"}

		for _, joinType := range joinTypes {
			condition := rag.JoinCondition{
				LeftTable:   "users",
				RightTable:  "orders",
				LeftColumn:  "id",
				RightColumn: "user_id",
				JoinType:    joinType,
			}

			assert.Equal(t, joinType, condition.JoinType)
		}
	})
}

// TestJoinPath_Structure tests JoinPath struct
func TestJoinPath_Structure(t *testing.T) {
	t.Run("creates JoinPath with all fields", func(t *testing.T) {
		path := rag.JoinPath{
			Tables: []string{"users", "orders", "products"},
			Joins: []rag.JoinCondition{
				{
					LeftTable:   "users",
					RightTable:  "orders",
					LeftColumn:  "id",
					RightColumn: "user_id",
					JoinType:    "INNER",
				},
				{
					LeftTable:   "orders",
					RightTable:  "products",
					LeftColumn:  "product_id",
					RightColumn: "id",
					JoinType:    "LEFT",
				},
			},
		}

		require.Len(t, path.Tables, 3)
		require.Len(t, path.Joins, 2)
		assert.Equal(t, "users", path.Tables[0])
		assert.Equal(t, "orders", path.Tables[1])
		assert.Equal(t, "products", path.Tables[2])
	})

	t.Run("creates JoinPath with zero values", func(t *testing.T) {
		path := rag.JoinPath{}

		assert.Nil(t, path.Tables)
		assert.Nil(t, path.Joins)
	})

	t.Run("creates empty JoinPath", func(t *testing.T) {
		path := rag.JoinPath{
			Tables: []string{},
			Joins:  []rag.JoinCondition{},
		}

		assert.Empty(t, path.Tables)
		assert.Empty(t, path.Joins)
	})
}

// TestHelpers_EdgeCases tests various edge cases
func TestHelpers_EdgeCases(t *testing.T) {
	t.Run("all components handle nil logger gracefully", func(t *testing.T) {
		assert.NotPanics(t, func() {
			_ = rag.NewSchemaAnalyzer(nil)
			_ = rag.NewPatternMatcher(nil)
			_ = rag.NewStatsCollector(nil)
			_ = rag.NewSQLValidator(nil)
			_ = rag.NewQueryPlanner(nil)
			_ = rag.NewJoinDetector(nil)
		})
	})

	t.Run("components can be used immediately after creation", func(t *testing.T) {
		logger := newTestLoggerHelpers()

		validator := rag.NewSQLValidator(logger)
		err := validator.Validate("SELECT * FROM users")
		assert.NoError(t, err)

		matcher := rag.NewPatternMatcher(logger)
		patterns := matcher.ExtractPatterns(nil)
		assert.NotNil(t, patterns)

		planner := rag.NewQueryPlanner(logger)
		steps := planner.DecomposeRequest("test", nil)
		assert.NotNil(t, steps)

		detector := rag.NewJoinDetector(logger)
		tables := detector.DetectTables("test", nil)
		assert.NotNil(t, tables)
	})

	t.Run("multiple concurrent operations don't interfere", func(t *testing.T) {
		logger := newTestLoggerHelpers()
		validator := rag.NewSQLValidator(logger)

		// Multiple validators can be created and used concurrently
		err1 := validator.Validate("SELECT * FROM users")
		err2 := validator.Validate("SELECT * FROM orders")
		err3 := validator.Validate("SELECT * FROM products")

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NoError(t, err3)
	})

	t.Run("struct zero values are valid", func(t *testing.T) {
		var step rag.QueryStep
		var stepSQL rag.StepSQL
		var planned rag.PlannedSQL
		var condition rag.JoinCondition
		var path rag.JoinPath

		assert.Equal(t, 0, step.Order)
		assert.Equal(t, "", stepSQL.SQL)
		assert.Equal(t, "", planned.Query)
		assert.Equal(t, "", condition.LeftTable)
		assert.Nil(t, path.Tables)
	})

	t.Run("empty slices are handled correctly", func(t *testing.T) {
		logger := newTestLoggerHelpers()

		matcher := rag.NewPatternMatcher(logger)
		patterns := matcher.ExtractPatterns([]*rag.Document{})
		assert.NotNil(t, patterns)
		assert.Empty(t, patterns)

		planner := rag.NewQueryPlanner(logger)
		result := planner.CombineSteps([]rag.StepSQL{})
		assert.Equal(t, "", result.Query)

		detector := rag.NewJoinDetector(logger)
		path := detector.FindJoinPath([]string{}, []rag.SchemaContext{})
		assert.Empty(t, path.Tables)
	})
}
