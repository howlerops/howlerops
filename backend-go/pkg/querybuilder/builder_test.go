package querybuilder

import (
	"strings"
	"testing"
)

func TestQueryBuilder_ToSQL_BasicSelect(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
			{Table: "users", Column: "name"},
			{Table: "users", Column: "email"},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	expectedSQL := `SELECT "users"."id", "users"."name", "users"."email" FROM "users"`
	if sql != expectedSQL {
		t.Errorf("Expected SQL:\n%s\nGot:\n%s", expectedSQL, sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_WithAggregation(t *testing.T) {
	countAgg := "count"
	avgAgg := "avg"

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "orders",
		Columns: []ColumnSelection{
			{Table: "orders", Column: "customer_id"},
			{Table: "orders", Column: "id", Aggregation: &countAgg, Alias: "order_count"},
			{Table: "orders", Column: "total", Aggregation: &avgAgg, Alias: "avg_total"},
		},
		GroupBy: []string{"orders.customer_id"},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "COUNT(") {
		t.Errorf("Expected COUNT aggregation in SQL: %s", sql)
	}

	if !strings.Contains(sql, "AVG(") {
		t.Errorf("Expected AVG aggregation in SQL: %s", sql)
	}

	if !strings.Contains(sql, "GROUP BY") {
		t.Errorf("Expected GROUP BY clause in SQL: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_WithFilters(t *testing.T) {
	valueActive := "true"
	valueName := "John"

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
			{Table: "users", Column: "name"},
		},
		Filters: []FilterCondition{
			{ID: "f1", Column: "users.active", Operator: "=", Value: &valueActive},
			{ID: "f2", Column: "users.name", Operator: "LIKE", Value: &valueName},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "WHERE") {
		t.Errorf("Expected WHERE clause in SQL: %s", sql)
	}

	// Should have parameterized values
	if len(args) < 2 {
		t.Errorf("Expected at least 2 args, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_WithJoin(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "orders",
		Columns: []ColumnSelection{
			{Table: "orders", Column: "id"},
			{Table: "customers", Column: "name"},
		},
		Joins: []JoinDefinition{
			{
				Type:  "INNER",
				Table: "customers",
				On:    JoinOn{Left: "orders.customer_id", Right: "customers.id"},
			},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "INNER JOIN") {
		t.Errorf("Expected INNER JOIN in SQL: %s", sql)
	}

	if !strings.Contains(sql, "ON") {
		t.Errorf("Expected ON clause in SQL: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_WithOrderBy(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
			{Table: "users", Column: "name"},
		},
		OrderBy: []OrderByClause{
			{Column: "users.name", Direction: "ASC"},
			{Column: "users.id", Direction: "DESC"},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "ORDER BY") {
		t.Errorf("Expected ORDER BY clause in SQL: %s", sql)
	}

	if !strings.Contains(sql, "ASC") {
		t.Errorf("Expected ASC in SQL: %s", sql)
	}

	if !strings.Contains(sql, "DESC") {
		t.Errorf("Expected DESC in SQL: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_WithLimit(t *testing.T) {
	limit := 100

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
		},
		Limit: &limit,
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "LIMIT") {
		t.Errorf("Expected LIMIT clause in SQL: %s", sql)
	}

	if len(args) != 0 {
		t.Errorf("Expected 0 args, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_ComplexQuery(t *testing.T) {
	countAgg := "count"
	sumAgg := "sum"
	valueActive := "true"
	limit := 50

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "orders",
		Columns: []ColumnSelection{
			{Table: "customers", Column: "name"},
			{Table: "orders", Column: "id", Aggregation: &countAgg, Alias: "total_orders"},
			{Table: "orders", Column: "total", Aggregation: &sumAgg, Alias: "revenue"},
		},
		Joins: []JoinDefinition{
			{
				Type:  "INNER",
				Table: "customers",
				On:    JoinOn{Left: "orders.customer_id", Right: "customers.id"},
			},
		},
		Filters: []FilterCondition{
			{ID: "f1", Column: "customers.active", Operator: "=", Value: &valueActive},
		},
		GroupBy: []string{"customers.name"},
		OrderBy: []OrderByClause{
			{Column: "revenue", Direction: "DESC"},
		},
		Limit: &limit,
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	// Check all parts are present
	requiredParts := []string{"SELECT", "COUNT(", "SUM(", "FROM", "INNER JOIN", "WHERE", "GROUP BY", "ORDER BY", "LIMIT"}
	for _, part := range requiredParts {
		if !strings.Contains(sql, part) {
			t.Errorf("Expected %s in SQL: %s", part, sql)
		}
	}

	if len(args) < 1 {
		t.Errorf("Expected at least 1 arg for filter, got %d", len(args))
	}
}

func TestQueryBuilder_Validate_MissingTable(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
		},
	}

	err := qb.Validate()
	if err == nil {
		t.Error("Expected validation error for missing table")
	}
}

func TestQueryBuilder_Validate_NoColumns(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns:    []ColumnSelection{},
	}

	err := qb.Validate()
	if err == nil {
		t.Error("Expected validation error for no columns")
	}
}

func TestQueryBuilder_Validate_AggregationWithoutGroupBy(t *testing.T) {
	countAgg := "count"

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "orders",
		Columns: []ColumnSelection{
			{Table: "orders", Column: "customer_id"}, // Non-aggregated
			{Table: "orders", Column: "id", Aggregation: &countAgg},
		},
		// Missing GroupBy for non-aggregated columns
	}

	err := qb.Validate()
	if err == nil {
		t.Error("Expected validation error for aggregation without GROUP BY")
	}
}

func TestQueryBuilder_Validate_FilterMissingValue(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
		},
		Filters: []FilterCondition{
			{ID: "f1", Column: "users.name", Operator: "="},
			// Missing Value
		},
	}

	err := qb.Validate()
	if err == nil {
		t.Error("Expected validation error for filter missing value")
	}
}

func TestQueryBuilder_GetValidationErrors(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "",
		Columns:    []ColumnSelection{},
	}

	result := qb.GetValidationErrors()

	if result.Valid {
		t.Error("Expected validation to fail")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}

	// Check specific error messages
	foundTableError := false
	foundColumnsError := false

	for _, err := range result.Errors {
		if strings.Contains(err.Message, "Table") {
			foundTableError = true
		}
		if strings.Contains(err.Message, "column") {
			foundColumnsError = true
		}
	}

	if !foundTableError {
		t.Error("Expected table validation error")
	}

	if !foundColumnsError {
		t.Error("Expected columns validation error")
	}
}

func TestQueryBuilder_buildColumnExpression(t *testing.T) {
	tests := []struct {
		name     string
		column   ColumnSelection
		expected string
	}{
		{
			name:     "Simple column",
			column:   ColumnSelection{Table: "users", Column: "id"},
			expected: `"users"."id"`,
		},
		{
			name: "COUNT aggregation",
			column: ColumnSelection{
				Table:       "users",
				Column:      "id",
				Aggregation: stringPtr("count"),
			},
			expected: `COUNT("users"."id")`,
		},
		{
			name: "COUNT DISTINCT aggregation",
			column: ColumnSelection{
				Table:       "users",
				Column:      "email",
				Aggregation: stringPtr("count_distinct"),
			},
			expected: `COUNT(DISTINCT "users"."email")`,
		},
		{
			name: "SUM aggregation",
			column: ColumnSelection{
				Table:       "orders",
				Column:      "total",
				Aggregation: stringPtr("sum"),
			},
			expected: `SUM("orders"."total")`,
		},
	}

	qb := &QueryBuilder{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := qb.buildColumnExpression(tt.column)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestQueryBuilder_ToSQL_BetweenOperator(t *testing.T) {
	valueFrom := "2024-01-01"
	valueTo := "2024-12-31"

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "orders",
		Columns: []ColumnSelection{
			{Table: "orders", Column: "id"},
		},
		Filters: []FilterCondition{
			{
				ID:       "f1",
				Column:   "orders.created_at",
				Operator: "BETWEEN",
				Value:    &valueFrom,
				ValueTo:  &valueTo,
			},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "BETWEEN") {
		t.Errorf("Expected BETWEEN in SQL: %s", sql)
	}

	if len(args) != 2 {
		t.Errorf("Expected 2 args for BETWEEN, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_IsNullOperator(t *testing.T) {
	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
		},
		Filters: []FilterCondition{
			{ID: "f1", Column: "users.deleted_at", Operator: "IS NULL"},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "NULL") {
		t.Errorf("Expected NULL check in SQL: %s", sql)
	}

	// IS NULL doesn't require parameterized values
	if len(args) != 0 {
		t.Errorf("Expected 0 args for IS NULL, got %d", len(args))
	}
}

func TestQueryBuilder_ToSQL_InOperator(t *testing.T) {
	valueList := "1,2,3,4,5"

	qb := &QueryBuilder{
		DataSource: "conn-1",
		Table:      "users",
		Columns: []ColumnSelection{
			{Table: "users", Column: "id"},
		},
		Filters: []FilterCondition{
			{ID: "f1", Column: "users.id", Operator: "IN", Value: &valueList},
		},
	}

	sql, args, err := qb.ToSQL()
	if err != nil {
		t.Fatalf("ToSQL failed: %v", err)
	}

	if !strings.Contains(sql, "IN") {
		t.Errorf("Expected IN operator in SQL: %s", sql)
	}

	// Should have 5 parameterized values
	if len(args) != 5 {
		t.Errorf("Expected 5 args for IN operator, got %d", len(args))
	}
}
