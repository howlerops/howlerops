package analyzer

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestQueryAnalyzer_Analyze(t *testing.T) {
	logger := logrus.New()
	analyzer := NewQueryAnalyzer(logger)

	// Create a sample schema
	schema := &Schema{
		Tables: map[string]*Table{
			"users": {
				Name: "users",
				Columns: map[string]*Column{
					"id":    {Name: "id", Type: "INTEGER", Indexed: true},
					"name":  {Name: "name", Type: "VARCHAR(255)"},
					"email": {Name: "email", Type: "VARCHAR(255)", Indexed: true},
					"age":   {Name: "age", Type: "INTEGER"},
				},
				RowCount: 50000,
			},
			"orders": {
				Name: "orders",
				Columns: map[string]*Column{
					"id":      {Name: "id", Type: "INTEGER", Indexed: true},
					"user_id": {Name: "user_id", Type: "INTEGER", Indexed: true},
					"total":   {Name: "total", Type: "DECIMAL(10,2)"},
				},
				RowCount: 100000,
			},
		},
	}

	tests := []struct {
		name           string
		sql            string
		expectedScore  int
		minSuggestions int
		checkWarnings  bool
	}{
		{
			name:           "SELECT * usage",
			sql:            "SELECT * FROM users",
			expectedScore:  80, // -15 for SELECT *, -5 for missing LIMIT
			minSuggestions: 2,
		},
		{
			name:           "Missing index on WHERE",
			sql:            "SELECT * FROM users WHERE name = 'John'",
			expectedScore:  70, // -15 for SELECT *, -10 for missing index, -5 for missing LIMIT
			minSuggestions: 3,
		},
		{
			name:           "Function in WHERE clause",
			sql:            "SELECT id FROM users WHERE UPPER(name) = 'JOHN'",
			expectedScore:  70, // -15 for function in WHERE, -10 for missing index, -5 for missing LIMIT
			minSuggestions: 3,
		},
		{
			name:           "LIKE with leading wildcard",
			sql:            "SELECT * FROM users WHERE name LIKE '%john%'",
			expectedScore:  60, // -15 SELECT *, -10 LIKE wildcard, -10 missing index, -5 missing LIMIT
			minSuggestions: 4,
		},
		{
			name:           "UPDATE without WHERE",
			sql:            "UPDATE users SET name = 'Updated'",
			expectedScore:  70, // -30 for UPDATE without WHERE
			checkWarnings:  true,
		},
		{
			name:           "DELETE without WHERE",
			sql:            "DELETE FROM users",
			expectedScore:  70, // -30 for DELETE without WHERE
			checkWarnings:  true,
		},
		{
			name:          "Well-optimized query",
			sql:           "SELECT id, name FROM users WHERE id = 1",
			expectedScore: 95, // -5 for missing LIMIT
		},
		{
			name:          "Query with JOIN",
			sql:           "SELECT u.name, o.total FROM users u JOIN orders o ON u.id = o.user_id WHERE u.id = 1",
			expectedScore: 95, // Minor deductions possible
		},
		{
			name:           "Missing JOIN condition",
			sql:            "SELECT * FROM users, orders",
			expectedScore:  60, // -20 for cartesian product, -15 for SELECT *, -5 for missing LIMIT
			minSuggestions: 2,
			checkWarnings:  true,
		},
		{
			name:           "NOT IN with subquery",
			sql:            "SELECT * FROM users WHERE id NOT IN (SELECT user_id FROM orders)",
			expectedScore:  70, // -15 SELECT *, -10 NOT IN subquery, -5 missing LIMIT
			minSuggestions: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Analyze(tt.sql, schema)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			// Check score
			assert.GreaterOrEqual(t, result.Score, tt.expectedScore-5, "Score should be within range")
			assert.LessOrEqual(t, result.Score, tt.expectedScore+5, "Score should be within range")

			// Check suggestions count
			if tt.minSuggestions > 0 {
				assert.GreaterOrEqual(t, len(result.Suggestions), tt.minSuggestions,
					"Should have at least %d suggestions", tt.minSuggestions)
			}

			// Check for warnings
			if tt.checkWarnings {
				assert.Greater(t, len(result.Warnings), 0, "Should have warnings")
			}

			// Log result for debugging
			t.Logf("Query: %s", tt.sql)
			t.Logf("Score: %d", result.Score)
			t.Logf("Suggestions: %d", len(result.Suggestions))
			t.Logf("Warnings: %d", len(result.Warnings))
		})
	}
}

func TestQueryAnalyzer_DetectAntiPatterns(t *testing.T) {
	logger := logrus.New()
	analyzer := NewQueryAnalyzer(logger)

	antiPatterns := []struct {
		name        string
		sql         string
		patternType string
		shouldFind  bool
	}{
		{
			name:        "SELECT * anti-pattern",
			sql:         "SELECT * FROM users",
			patternType: "select",
			shouldFind:  true,
		},
		{
			name:        "Missing index pattern",
			sql:         "SELECT * FROM users WHERE name = 'test'",
			patternType: "index",
			shouldFind:  true,
		},
		{
			name:        "Function in WHERE",
			sql:         "SELECT id FROM users WHERE UPPER(email) = 'TEST@TEST.COM'",
			patternType: "where",
			shouldFind:  true,
		},
		{
			name:        "LIKE with leading wildcard",
			sql:         "SELECT * FROM products WHERE description LIKE '%phone%'",
			patternType: "where",
			shouldFind:  true,
		},
		{
			name:        "NOT IN subquery",
			sql:         "SELECT * FROM users WHERE id NOT IN (SELECT user_id FROM orders)",
			patternType: "where",
			shouldFind:  true,
		},
		{
			name:        "Cartesian product",
			sql:         "SELECT * FROM users, orders",
			patternType: "join",
			shouldFind:  true,
		},
		{
			name:        "Correlated subquery",
			sql:         "SELECT *, (SELECT COUNT(*) FROM orders o WHERE o.user_id = u.id) FROM users u",
			patternType: "subquery",
			shouldFind:  true,
		},
		{
			name:        "Multiple OR on same column",
			sql:         "SELECT * FROM users WHERE status = 'active' OR status = 'pending' OR status = 'verified'",
			patternType: "where",
			shouldFind:  true,
		},
		{
			name:        "UPDATE without WHERE",
			sql:         "UPDATE users SET status = 'inactive'",
			patternType: "warning",
			shouldFind:  true,
		},
		{
			name:        "DELETE without WHERE",
			sql:         "DELETE FROM orders",
			patternType: "warning",
			shouldFind:  true,
		},
	}

	for _, tt := range antiPatterns {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Analyze(tt.sql, nil)

			assert.NoError(t, err)
			assert.NotNil(t, result)

			if tt.shouldFind {
				if tt.patternType == "warning" {
					assert.Greater(t, len(result.Warnings), 0, "Should detect warning")
				} else {
					found := false
					for _, suggestion := range result.Suggestions {
						if suggestion.Type == tt.patternType {
							found = true
							break
						}
					}
					assert.True(t, found, "Should detect %s anti-pattern", tt.patternType)
				}
			}
		})
	}
}

func TestQueryAnalyzer_ComplexityCalculation(t *testing.T) {
	logger := logrus.New()
	analyzer := NewQueryAnalyzer(logger)

	tests := []struct {
		name               string
		sql                string
		expectedComplexity string
	}{
		{
			name:               "Simple query",
			sql:                "SELECT * FROM users WHERE id = 1",
			expectedComplexity: "simple",
		},
		{
			name:               "Moderate query with JOIN",
			sql:                "SELECT u.*, o.* FROM users u JOIN orders o ON u.id = o.user_id",
			expectedComplexity: "moderate",
		},
		{
			name: "Complex query with multiple JOINs and GROUP BY",
			sql: `SELECT u.name, COUNT(o.id), SUM(o.total)
                  FROM users u
                  JOIN orders o ON u.id = o.user_id
                  JOIN products p ON o.product_id = p.id
                  WHERE u.status = 'active'
                  GROUP BY u.name
                  ORDER BY COUNT(o.id) DESC`,
			expectedComplexity: "complex",
		},
		{
			name: "Complex query with subquery",
			sql: `SELECT * FROM users
                  WHERE id IN (SELECT user_id FROM orders WHERE total > 100)
                  ORDER BY created_at DESC`,
			expectedComplexity: "moderate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Analyze(tt.sql, nil)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tt.expectedComplexity, result.Complexity)
		})
	}
}

func TestQueryAnalyzer_SuggestionQuality(t *testing.T) {
	logger := logrus.New()
	analyzer := NewQueryAnalyzer(logger)

	sql := "SELECT * FROM users WHERE name = 'John'"
	result, err := analyzer.Analyze(sql, nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Check that suggestions have all required fields
	for _, suggestion := range result.Suggestions {
		assert.NotEmpty(t, suggestion.Type, "Suggestion should have type")
		assert.NotEmpty(t, suggestion.Severity, "Suggestion should have severity")
		assert.NotEmpty(t, suggestion.Message, "Suggestion should have message")

		// Severity should be valid
		validSeverities := []string{"info", "warning", "critical"}
		assert.Contains(t, validSeverities, suggestion.Severity)

		// Type should be valid
		validTypes := []string{"index", "join", "where", "select", "subquery", "insert"}
		assert.Contains(t, validTypes, suggestion.Type)
	}
}