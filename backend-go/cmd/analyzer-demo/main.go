package main

import (
	"fmt"
	"log"

	"github.com/jbeck018/howlerops/backend-go/internal/analyzer"
	"github.com/jbeck018/howlerops/backend-go/internal/autocomplete"
	"github.com/jbeck018/howlerops/backend-go/internal/nl2sql"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create sample schema
	schema := createSampleSchema()

	// Demo 1: Query Analysis
	fmt.Println("=== QUERY ANALYZER DEMO ===")
	demoQueryAnalyzer(schema, logger)

	// Demo 2: Natural Language to SQL
	fmt.Println("\n=== NATURAL LANGUAGE TO SQL DEMO ===")
	demoNL2SQL(logger)

	// Demo 3: Query Explainer
	fmt.Println("\n=== QUERY EXPLAINER DEMO ===")
	demoExplainer()

	// Demo 4: Autocomplete
	fmt.Println("\n=== AUTOCOMPLETE DEMO ===")
	demoAutocomplete(schema, logger)
}

func demoQueryAnalyzer(schema *analyzer.Schema, logger *logrus.Logger) {
	queryAnalyzer := analyzer.NewQueryAnalyzer(logger)

	queries := []struct {
		name string
		sql  string
	}{
		{
			name: "Poor Query",
			sql:  "SELECT * FROM users WHERE UPPER(email) = 'TEST@EXAMPLE.COM'",
		},
		{
			name: "Good Query",
			sql:  "SELECT id, name, email FROM users WHERE id = 1",
		},
		{
			name: "Complex Query",
			sql: `SELECT u.name, COUNT(o.id) as order_count
                  FROM users u
                  LEFT JOIN orders o ON u.id = o.user_id
                  WHERE u.status = 'active'
                  GROUP BY u.name
                  ORDER BY order_count DESC`,
		},
	}

	for _, q := range queries {
		fmt.Printf("Analyzing: %s\n", q.name)
		fmt.Printf("SQL: %s\n", q.sql)

		result, err := queryAnalyzer.Analyze(q.sql, schema)
		if err != nil {
			log.Printf("Error analyzing query: %v", err)
			continue
		}

		fmt.Printf("Score: %d/100\n", result.Score)
		fmt.Printf("Complexity: %s\n", result.Complexity)
		fmt.Printf("Estimated Cost: %d\n", result.EstimatedCost)

		if len(result.Warnings) > 0 {
			fmt.Println("Warnings:")
			for _, warning := range result.Warnings {
				fmt.Printf("  - [%s] %s\n", warning.Severity, warning.Message)
			}
		}

		if len(result.Suggestions) > 0 {
			fmt.Println("Suggestions:")
			for _, suggestion := range result.Suggestions {
				fmt.Printf("  - [%s/%s] %s\n", suggestion.Type, suggestion.Severity, suggestion.Message)
				if suggestion.Impact != "" {
					fmt.Printf("    Impact: %s\n", suggestion.Impact)
				}
			}
		}

		fmt.Println()
	}
}

func demoNL2SQL(logger *logrus.Logger) {
	converter := nl2sql.NewNL2SQLConverter(nil, logger)

	queries := []string{
		"show all users",
		"count orders from today",
		"find products where price greater than 100",
		"top 10 customers ordered by total_spent",
		"average price from products",
		"users where email contains gmail",
		"update users set status to active where id = 1",
		"delete orders where status is canceled",
	}

	for _, query := range queries {
		fmt.Printf("Natural Language: %s\n", query)

		result, err := converter.Convert(query)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			if result != nil && len(result.Suggestions) > 0 {
				fmt.Println("Suggestions:")
				for _, s := range result.Suggestions {
					fmt.Printf("  - %s\n", s)
				}
			}
		} else {
			fmt.Printf("Generated SQL: %s\n", result.SQL)
			fmt.Printf("Confidence: %.2f\n", result.Confidence)
			fmt.Printf("Template: %s\n", result.Template)
		}
		fmt.Println()
	}
}

func demoExplainer() {
	queries := []string{
		"SELECT name, email FROM users WHERE status = 'active'",
		"UPDATE products SET price = price * 1.1 WHERE category = 'electronics'",
		"DELETE FROM orders WHERE created_at < '2023-01-01'",
		`SELECT u.name, COUNT(o.id) as order_count
         FROM users u
         LEFT JOIN orders o ON u.id = o.user_id
         GROUP BY u.name
         HAVING COUNT(o.id) > 5`,
	}

	for _, sql := range queries {
		fmt.Printf("SQL: %s\n", sql)

		explanation, err := analyzer.Explain(sql)
		if err != nil {
			log.Printf("Error explaining query: %v", err)
			continue
		}

		fmt.Printf("Explanation: %s\n\n", explanation)
	}
}

func demoAutocomplete(schema *analyzer.Schema, logger *logrus.Logger) {
	// Convert schema format
	acSchema := &autocomplete.Schema{
		Tables: make(map[string]*autocomplete.Table),
	}

	for tableName, table := range schema.Tables {
		acTable := &autocomplete.Table{
			Name:    tableName,
			Columns: make(map[string]string),
			Indexes: []string{},
		}

		for colName, col := range table.Columns {
			acTable.Columns[colName] = col.Type
			if col.Indexed {
				acTable.Indexes = append(acTable.Indexes, colName)
			}
		}

		acSchema.Tables[tableName] = acTable
	}

	service := autocomplete.NewAutocompleteService(acSchema, logger)

	testCases := []struct {
		sql       string
		cursorPos int
		context   string
	}{
		{
			sql:       "SELECT ",
			cursorPos: 7,
			context:   "after SELECT",
		},
		{
			sql:       "SELECT * FROM ",
			cursorPos: 14,
			context:   "after FROM",
		},
		{
			sql:       "SELECT * FROM users WHERE ",
			cursorPos: 26,
			context:   "after WHERE",
		},
		{
			sql:       "SELECT * FROM users u JOIN ",
			cursorPos: 27,
			context:   "after JOIN",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("Context: %s\n", tc.context)
		fmt.Printf("SQL: %s\n", tc.sql)

		suggestions, err := service.GetSuggestions(tc.sql, tc.cursorPos)
		if err != nil {
			log.Printf("Error getting suggestions: %v", err)
			continue
		}

		fmt.Printf("Suggestions (%d):\n", len(suggestions))
		for i, suggestion := range suggestions {
			if i >= 5 { // Show only first 5
				fmt.Printf("  ... and %d more\n", len(suggestions)-5)
				break
			}
			fmt.Printf("  - [%s] %s", suggestion.Type, suggestion.Text)
			if suggestion.Description != "" {
				fmt.Printf(" - %s", suggestion.Description)
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func createSampleSchema() *analyzer.Schema {
	return &analyzer.Schema{
		Tables: map[string]*analyzer.Table{
			"users": {
				Name: "users",
				Columns: map[string]*analyzer.Column{
					"id":         {Name: "id", Type: "INTEGER", Indexed: true},
					"name":       {Name: "name", Type: "VARCHAR(255)"},
					"email":      {Name: "email", Type: "VARCHAR(255)", Indexed: true},
					"status":     {Name: "status", Type: "VARCHAR(50)"},
					"created_at": {Name: "created_at", Type: "TIMESTAMP"},
				},
				RowCount: 50000,
			},
			"orders": {
				Name: "orders",
				Columns: map[string]*analyzer.Column{
					"id":         {Name: "id", Type: "INTEGER", Indexed: true},
					"user_id":    {Name: "user_id", Type: "INTEGER", Indexed: true},
					"total":      {Name: "total", Type: "DECIMAL(10,2)"},
					"status":     {Name: "status", Type: "VARCHAR(50)"},
					"created_at": {Name: "created_at", Type: "TIMESTAMP"},
				},
				RowCount: 100000,
			},
			"products": {
				Name: "products",
				Columns: map[string]*analyzer.Column{
					"id":          {Name: "id", Type: "INTEGER", Indexed: true},
					"name":        {Name: "name", Type: "VARCHAR(255)"},
					"price":       {Name: "price", Type: "DECIMAL(10,2)"},
					"category":    {Name: "category", Type: "VARCHAR(100)", Indexed: true},
					"description": {Name: "description", Type: "TEXT"},
				},
				RowCount: 5000,
			},
		},
	}
}
