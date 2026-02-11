package multiquery

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJoinDetector_DetectsSimpleInnerJoin tests basic inner join detection
func TestJoinDetector_DetectsSimpleInnerJoin(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := `
		SELECT u.id, u.name, o.total
		FROM @db1.users u
		INNER JOIN @db2.orders o ON u.id = o.user_id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins, "Should detect INNER JOIN")
	assert.Len(t, parsed.RequiredConnections, 2)
	assert.ElementsMatch(t, []string{"db1", "db2"}, parsed.RequiredConnections)
}

// TestJoinDetector_DetectsLeftJoin tests left join detection
func TestJoinDetector_DetectsLeftJoin(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := "SELECT u.*, o.* FROM @prod.users u LEFT JOIN @staging.orders o ON u.id = o.user_id"

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins, "Should detect LEFT JOIN")
}

// TestJoinDetector_DetectsRightJoin tests right join detection
func TestJoinDetector_DetectsRightJoin(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := "SELECT * FROM @db1.customers c RIGHT JOIN @db2.orders o ON c.id = o.customer_id"

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins, "Should detect RIGHT JOIN")
}

// TestJoinDetector_DetectsCrossJoin tests cross join detection
func TestJoinDetector_DetectsCrossJoin(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := "SELECT * FROM @db1.users u CROSS JOIN @db2.products p"

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins, "Should detect CROSS JOIN")
}

// TestJoinDetector_DetectsMultipleJoins tests multiple joins in single query
func TestJoinDetector_DetectsMultipleJoins(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := `
		SELECT u.name, o.id, oi.product_id
		FROM @db1.users u
		JOIN @db2.orders o ON u.id = o.user_id
		JOIN @db3.order_items oi ON o.id = oi.order_id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins, "Should detect multiple JOINs")
	assert.Len(t, parsed.RequiredConnections, 3)
}

// TestJoinDetector_NoJoinDetectionWithoutJoinKeyword tests that queries without JOIN are correctly identified
func TestJoinDetector_NoJoinDetectionWithoutJoinKeyword(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "Simple SELECT",
			query: "SELECT * FROM @db1.users WHERE id = 5",
		},
		{
			name:  "Subquery without join",
			query: "SELECT * FROM (SELECT * FROM @db1.users) t WHERE t.active = 1",
		},
		{
			name:  "UNION without join",
			query: "SELECT * FROM @db1.users UNION SELECT * FROM @db2.customers",
		},
		{
			name:  "WHERE clause mentioning 'join'",
			query: "SELECT * FROM @db1.users WHERE user_type = 'join_user'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			require.NoError(t, err)
			assert.False(t, parsed.HasJoins, "Should not detect JOIN in: %s", tt.query)
		})
	}
}

// TestJoinDetector_ForeignKeyAwareness tests joining on known foreign key relationships
func TestJoinDetector_ForeignKeyAwareness(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	// E-commerce schema with FK relationships
	schema := CommonTestSchemas["e-commerce"]
	require.NotNil(t, schema)

	query := `
		SELECT p.name, oi.quantity, p.price
		FROM @db1.products p
		JOIN @db2.order_items oi ON p.id = oi.product_id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins)

	// Verify table structure info
	cols := schema.GetColumns("public", "products")
	assert.NotEmpty(t, cols)
	assert.True(t, len(cols) > 0, "Should have product columns")

	orderItemTable := schema.GetColumns("public", "order_items")
	assert.NotEmpty(t, orderItemTable)

	// Get foreign keys for order_items
	fks := schema.GetForeignKeys("public", "order_items")
	assert.Len(t, fks, 2, "Should have 2 foreign keys")

	// Verify FK to products exists
	productFKFound := false
	for _, fk := range fks {
		if fk.ReferencedTable == "products" {
			productFKFound = true
			assert.Equal(t, "fk_order_items_product", fk.Name)
		}
	}
	assert.True(t, productFKFound, "Should find FK to products table")
}

// TestJoinDetector_CircularForeignKeys tests circular FK relationships
func TestJoinDetector_CircularForeignKeys(t *testing.T) {
	schema := CommonTestSchemas["circular"]
	require.NotNil(t, schema)

	// Get foreign keys for both tables
	deptFKs := schema.GetForeignKeys("public", "departments")
	empFKs := schema.GetForeignKeys("public", "employees")

	// Verify circular references exist
	assert.Len(t, deptFKs, 1, "Department should have FK to employees")
	assert.Len(t, empFKs, 1, "Employee should have FK to departments")

	// Verify they reference each other
	assert.Equal(t, "employees", deptFKs[0].ReferencedTable)
	assert.Equal(t, "departments", empFKs[0].ReferencedTable)
}

// TestJoinDetector_NoForeignKeySchema tests schema without any foreign keys
func TestJoinDetector_NoForeignKeySchema(t *testing.T) {
	schema := CommonTestSchemas["no-fk"]
	require.NotNil(t, schema)

	// Verify no foreign keys exist
	logsFKs := schema.GetForeignKeys("public", "logs")
	eventsFKs := schema.GetForeignKeys("public", "events")

	assert.Empty(t, logsFKs, "Logs table should have no foreign keys")
	assert.Empty(t, eventsFKs, "Events table should have no foreign keys")
}

// TestJoinDetector_ComplexQueryAnalysis tests analysis of complex multi-database query
func TestJoinDetector_ComplexQueryAnalysis(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := `
		SELECT 
			u.id,
			u.name,
			o.id as order_id,
			o.total,
			oi.product_id,
			p.name as product_name,
			p.price
		FROM @prod.users u
		INNER JOIN @prod.orders o ON u.id = o.user_id
		INNER JOIN @staging.order_items oi ON o.id = oi.order_id
		LEFT JOIN @archive.products p ON oi.product_id = p.id
		WHERE u.created_at > '2024-01-01'
		ORDER BY o.created_at DESC
		LIMIT 100
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	assert.True(t, parsed.HasJoins)
	assert.Len(t, parsed.RequiredConnections, 3)
	assert.ElementsMatch(t, []string{"prod", "staging", "archive"}, parsed.RequiredConnections[:3])

	// Should suggest federated strategy for multi-DB joins
	assert.Equal(t, StrategyFederated, parsed.SuggestedStrategy)
}

// TestJoinDetector_TableAliasDetection tests that table aliases are properly handled
func TestJoinDetector_TableAliasDetection(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := `
		SELECT u.name, o.total, oi.quantity
		FROM @db1.users AS u
		INNER JOIN @db2.orders AS o ON u.id = o.user_id
		INNER JOIN @db3.order_items AS oi ON o.id = oi.order_id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	assert.True(t, parsed.HasJoins)
	assert.Len(t, parsed.RequiredConnections, 3)

	// Verify table references were extracted correctly
	assert.Len(t, parsed.Tables, 3)
}

// TestJoinDetector_WithMockDatabase tests join analysis with mock database
func TestJoinDetector_WithMockDatabase(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Mock schema queries
	mock.ExpectQuery("SELECT CONSTRAINT_NAME, COLUMN_NAME.*").
		WillReturnRows(sqlmock.NewRows([]string{"constraint_name", "column_name"}).
			AddRow("fk_orders_user", "user_id"))

	mock.ExpectQuery("SELECT REFERENCED_TABLE_NAME, REFERENCED_COLUMN_NAME.*").
		WillReturnRows(sqlmock.NewRows([]string{"referenced_table_name", "referenced_column_name"}).
			AddRow("users", "id"))

	parser := NewQueryParser(&Config{Enabled: true}, logger)

	query := "SELECT u.name, o.total FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id"

	parsed, err := parser.Parse(query)
	require.NoError(t, err)
	assert.True(t, parsed.HasJoins)
}

// TestJoinDetector_AggregationWithJoins tests aggregation detection in joins
func TestJoinDetector_AggregationWithJoins(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	tests := []struct {
		name       string
		query      string
		expectAgg  bool
		expectJoin bool
	}{
		{
			name:       "COUNT with JOIN",
			query:      "SELECT u.id, COUNT(o.id) FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id GROUP BY u.id",
			expectAgg:  true,
			expectJoin: true,
		},
		{
			name:       "SUM with JOIN",
			query:      "SELECT u.name, SUM(o.total) FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id GROUP BY u.id",
			expectAgg:  true,
			expectJoin: true,
		},
		{
			name:       "MAX with LEFT JOIN",
			query:      "SELECT p.name, MAX(oi.quantity) FROM @db1.products p LEFT JOIN @db2.order_items oi ON p.id = oi.product_id GROUP BY p.id",
			expectAgg:  true,
			expectJoin: true,
		},
		{
			name:       "No aggregation with JOIN",
			query:      "SELECT u.name, o.total FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id",
			expectAgg:  false,
			expectJoin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			require.NoError(t, err)
			assert.Equal(t, tt.expectAgg, parsed.HasAggregation, "Aggregation detection mismatch")
			assert.Equal(t, tt.expectJoin, parsed.HasJoins, "Join detection mismatch")
		})
	}
}

// TestJoinDetector_SchemaConflictDetection tests detection of naming conflicts across databases
func TestJoinDetector_SchemaConflictDetection(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	// Query with tables named 'users' in multiple databases
	query := `
		SELECT a.id, b.id
		FROM @database1.users a
		JOIN @database2.users b ON a.id = b.id
	`

	parsed, err := parser.Parse(query)
	require.NoError(t, err)

	assert.Len(t, parsed.RequiredConnections, 2)
	// Both references to 'users' table should be captured
	tableNames := make(map[string]bool)
	for _, t := range parsed.Tables {
		tableNames[t] = true
	}
	assert.Len(t, tableNames, 1, "Should recognize both as 'users' table")
}

// TestJoinDetector_EdgeCases tests various edge cases
func TestJoinDetector_EdgeCases(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	parser := NewQueryParser(&Config{Enabled: true}, logger)

	tests := []struct {
		name       string
		query      string
		wantErr    bool
		expectJoin bool
	}{
		{
			name:       "Empty JOIN condition",
			query:      "SELECT * FROM @db1.a JOIN @db2.b",
			expectJoin: true,
		},
		{
			name:       "Self-join",
			query:      "SELECT * FROM @db1.users u1 JOIN @db1.users u2 ON u1.id = u2.parent_id",
			expectJoin: true,
		},
		{
			name:       "Very long table name",
			query:      "SELECT * FROM @db1.very_long_table_name_that_exceeds_normal_length JOIN @db2.another_very_long_table_name_here",
			expectJoin: true,
		},
		{
			name:       "Special characters in connection ID",
			query:      "SELECT * FROM @prod-us-east.users JOIN @prod-us-west.orders",
			expectJoin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := parser.Parse(tt.query)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expectJoin, parsed.HasJoins)
		})
	}
}
