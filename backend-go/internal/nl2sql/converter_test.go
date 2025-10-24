package nl2sql

import (
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNL2SQLConverter_Convert(t *testing.T) {
	logger := logrus.New()
	converter := NewNL2SQLConverter(nil, logger)

	tests := []struct {
		name           string
		input          string
		expectedSQL    string
		shouldContain  []string
		shouldNotError bool
	}{
		// Basic SELECT queries
		{
			name:           "Show all records",
			input:          "show users",
			expectedSQL:    "SELECT * FROM users LIMIT 100",
			shouldNotError: true,
		},
		{
			name:           "Get all with 'all' keyword",
			input:          "get all customers",
			expectedSQL:    "SELECT * FROM customers LIMIT 100",
			shouldNotError: true,
		},
		{
			name:           "List records",
			input:          "list products",
			expectedSQL:    "SELECT * FROM products LIMIT 100",
			shouldNotError: true,
		},

		// COUNT queries
		{
			name:           "Count records",
			input:          "count users",
			expectedSQL:    "SELECT COUNT(*) AS count FROM users",
			shouldNotError: true,
		},
		{
			name:           "How many records",
			input:          "how many orders are there",
			expectedSQL:    "SELECT COUNT(*) AS count FROM orders",
			shouldNotError: true,
		},

		// WHERE conditions with equality
		{
			name:           "Find with WHERE equals",
			input:          "find users where status is active",
			expectedSQL:    "SELECT * FROM users WHERE status = 'active'",
			shouldNotError: true,
		},
		{
			name:           "Get with numeric condition",
			input:          "get products where id = 123",
			expectedSQL:    "SELECT * FROM products WHERE id = 123",
			shouldNotError: true,
		},

		// LIKE conditions
		{
			name:           "Search with contains",
			input:          "find users where name contains john",
			expectedSQL:    "SELECT * FROM users WHERE name LIKE '%john%'",
			shouldNotError: true,
		},
		{
			name:           "Search with has",
			input:          "search products where title has phone",
			expectedSQL:    "SELECT * FROM products WHERE title LIKE '%phone%'",
			shouldNotError: true,
		},

		// Comparison operators
		{
			name:           "Greater than",
			input:          "find products where price greater than 100",
			expectedSQL:    "SELECT * FROM products WHERE price > 100",
			shouldNotError: true,
		},
		{
			name:           "Less than",
			input:          "get users where age less than 25",
			expectedSQL:    "SELECT * FROM users WHERE age < 25",
			shouldNotError: true,
		},

		// BETWEEN queries
		{
			name:           "Between range",
			input:          "find products where price between 10 and 100",
			expectedSQL:    "SELECT * FROM products WHERE price BETWEEN 10 AND 100",
			shouldNotError: true,
		},

		// ORDER BY queries
		{
			name:           "Order by ascending",
			input:          "show users ordered by name",
			expectedSQL:    "SELECT * FROM users ORDER BY name ASC",
			shouldNotError: true,
		},
		{
			name:           "Order by descending",
			input:          "list products sorted by price desc",
			expectedSQL:    "SELECT * FROM products ORDER BY price DESC",
			shouldNotError: true,
		},

		// TOP/LIMIT queries
		{
			name:           "Top N records",
			input:          "show top 10 users",
			expectedSQL:    "SELECT * FROM users LIMIT 10",
			shouldNotError: true,
		},
		{
			name:           "First N with order",
			input:          "get first 5 products ordered by price",
			expectedSQL:    "SELECT * FROM products ORDER BY price LIMIT 5",
			shouldNotError: true,
		},

		// Aggregate functions
		{
			name:           "Sum calculation",
			input:          "sum of amount in orders",
			expectedSQL:    "SELECT SUM(amount) AS total FROM orders",
			shouldNotError: true,
		},
		{
			name:           "Average calculation",
			input:          "average price from products",
			expectedSQL:    "SELECT AVG(price) AS average FROM products",
			shouldNotError: true,
		},
		{
			name:           "Maximum value",
			input:          "maximum price from products",
			expectedSQL:    "SELECT MAX(price) AS maximum FROM products",
			shouldNotError: true,
		},
		{
			name:           "Minimum value",
			input:          "minimum age in users",
			expectedSQL:    "SELECT MIN(age) AS minimum FROM users",
			shouldNotError: true,
		},

		// GROUP BY queries
		{
			name:           "Count with GROUP BY",
			input:          "count users grouped by status",
			expectedSQL:    "SELECT status, COUNT(*) AS count FROM users GROUP BY status",
			shouldNotError: true,
		},

		// DISTINCT queries
		{
			name:           "Select distinct values",
			input:          "show unique categories from products",
			expectedSQL:    "SELECT DISTINCT categories FROM products",
			shouldNotError: true,
		},

		// NULL checks
		{
			name:           "IS NULL check",
			input:          "find users where email is null",
			expectedSQL:    "SELECT * FROM users WHERE email IS NULL",
			shouldNotError: true,
		},
		{
			name:           "IS NOT NULL check",
			input:          "get products where image is not null",
			expectedSQL:    "SELECT * FROM products WHERE image IS NOT NULL",
			shouldNotError: true,
		},

		// IN clause
		{
			name:           "IN with numbers",
			input:          "find users where id in (1,2,3)",
			expectedSQL:    "SELECT * FROM users WHERE id IN (1,2,3)",
			shouldNotError: true,
		},
		{
			name:           "IN with strings",
			input:          "get products where category in (electronics,books)",
			expectedSQL:    "SELECT * FROM products WHERE category IN ('electronics', 'books')",
			shouldNotError: true,
		},

		// DELETE queries
		{
			name:           "Delete with condition",
			input:          "delete users where status is inactive",
			expectedSQL:    "DELETE FROM users WHERE status = 'inactive'",
			shouldNotError: true,
		},

		// UPDATE queries
		{
			name:           "Update with condition",
			input:          "update users set status to active where id = 1",
			expectedSQL:    "UPDATE users SET status = 'active' WHERE id = 1",
			shouldNotError: true,
		},

		// INSERT queries
		{
			name:          "Insert with values",
			input:         "insert user with name john and email john@example.com",
			shouldContain: []string{"INSERT INTO user", "name", "email", "john", "john@example.com"},
			shouldNotError: true,
		},

		// Date-based queries
		{
			name:          "Records from today",
			input:         "show orders from today",
			shouldContain: []string{"SELECT * FROM orders", "DATE(created_at) = CURDATE()"},
			shouldNotError: true,
		},
		{
			name:          "Records from this month",
			input:         "get users from this month",
			shouldContain: []string{"SELECT * FROM users", "MONTH(created_at)", "YEAR(created_at)"},
			shouldNotError: true,
		},

		// Select specific columns
		{
			name:           "Select specific columns",
			input:          "show name and email from users",
			expectedSQL:    "SELECT name, email FROM users",
			shouldNotError: true,
		},

		// Invalid queries
		{
			name:           "Unrecognized pattern",
			input:          "do something weird with data",
			shouldNotError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.input)

			if tt.shouldNotError {
				assert.NotNil(t, result)
				if tt.expectedSQL != "" {
					assert.Equal(t, tt.expectedSQL, result.SQL)
				}
				if len(tt.shouldContain) > 0 {
					for _, substr := range tt.shouldContain {
						assert.Contains(t, result.SQL, substr)
					}
				}
				assert.Greater(t, result.Confidence, 0.0)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestNL2SQLConverter_Patterns(t *testing.T) {
	logger := logrus.New()
	converter := NewNL2SQLConverter(nil, logger)

	// Test that we have at least 20 patterns as required
	patterns := converter.GetSupportedPatterns()
	assert.GreaterOrEqual(t, len(patterns), 20, "Should have at least 20 patterns")

	// Verify each pattern has required fields
	for _, pattern := range patterns {
		assert.NotEmpty(t, pattern["description"], "Pattern should have description")
		assert.NotEmpty(t, pattern["pattern"], "Pattern should have regex pattern")

		examples, ok := pattern["examples"].([]string)
		if ok {
			assert.NotEmpty(t, examples, "Pattern should have examples")
		}
	}
}

func TestNL2SQLConverter_CaseSensitivity(t *testing.T) {
	logger := logrus.New()
	converter := NewNL2SQLConverter(nil, logger)

	// Test that converter handles different cases
	inputs := []string{
		"SHOW USERS",
		"Show Users",
		"show users",
		"ShOw UsErS",
	}

	expectedSQL := "SELECT * FROM users LIMIT 100"

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result, err := converter.Convert(input)
			assert.NoError(t, err)
			assert.Equal(t, expectedSQL, result.SQL)
		})
	}
}

func TestNL2SQLConverter_Confidence(t *testing.T) {
	logger := logrus.New()
	converter := NewNL2SQLConverter(nil, logger)

	tests := []struct {
		name             string
		input            string
		minConfidence    float64
		maxConfidence    float64
	}{
		{
			name:          "Perfect match",
			input:         "show users",
			minConfidence: 0.8,
			maxConfidence: 1.0,
		},
		{
			name:          "Partial match with extra words",
			input:         "please show all the users records",
			minConfidence: 0.5,
			maxConfidence: 0.9,
		},
		{
			name:          "Complex query",
			input:         "find users where status is active",
			minConfidence: 0.7,
			maxConfidence: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := converter.Convert(tt.input)
			if result.SQL != "" {
				assert.GreaterOrEqual(t, result.Confidence, tt.minConfidence)
				assert.LessOrEqual(t, result.Confidence, tt.maxConfidence)
			}
		})
	}
}

func TestNL2SQLConverter_EdgeCases(t *testing.T) {
	logger := logrus.New()
	converter := NewNL2SQLConverter(nil, logger)

	tests := []struct {
		name          string
		input         string
		shouldContain string
	}{
		{
			name:          "Empty input",
			input:         "",
			shouldContain: "",
		},
		{
			name:          "Only whitespace",
			input:         "   ",
			shouldContain: "",
		},
		{
			name:          "Special characters in values",
			input:         "find users where name is O'Brien",
			shouldContain: "O'Brien",
		},
		{
			name:          "Multiple spaces",
			input:         "show    all     users",
			shouldContain: "SELECT * FROM users",
		},
		{
			name:          "Table name variations",
			input:         "show user", // singular
			shouldContain: "SELECT * FROM user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := converter.Convert(tt.input)
			if tt.shouldContain != "" {
				assert.Contains(t, result.SQL, tt.shouldContain)
			}
		})
	}
}

func TestNL2SQLConverter_Suggestions(t *testing.T) {
	logger := logrus.New()
	converter := NewNL2SQLConverter(nil, logger)

	// Test unmatched input returns suggestions
	result, err := converter.Convert("do something random with data")

	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.SQL)
	assert.NotEmpty(t, result.Suggestions)

	// Check that suggestions contain helpful examples
	suggestionsText := strings.Join(result.Suggestions, " ")
	assert.Contains(t, suggestionsText, "show")
	assert.Contains(t, suggestionsText, "count")
	assert.Contains(t, suggestionsText, "find")
}