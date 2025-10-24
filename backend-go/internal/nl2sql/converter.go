package nl2sql

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// NL2SQLConverter converts natural language queries to SQL
type NL2SQLConverter struct {
	schema    *Schema
	templates []QueryTemplate
	logger    *logrus.Logger
}

// Schema represents database schema for NL2SQL
type Schema struct {
	Tables map[string]*Table `json:"tables"`
}

// Table represents a database table
type Table struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// QueryTemplate defines a pattern and builder for NL to SQL conversion
type QueryTemplate struct {
	Pattern     string                                        // regex pattern
	Builder     func(matches []string, schema *Schema) string // SQL builder function
	Description string                                        // What this template does
	Examples    []string                                      // Example inputs
}

// ConversionResult contains the converted SQL and metadata
type ConversionResult struct {
	SQL         string   `json:"sql"`
	Confidence  float64  `json:"confidence"` // 0.0 to 1.0
	Template    string   `json:"template"`   // Which template was used
	Suggestions []string `json:"suggestions,omitempty"`
}

// NewNL2SQLConverter creates a new converter instance
func NewNL2SQLConverter(schema *Schema, logger *logrus.Logger) *NL2SQLConverter {
	if logger == nil {
		logger = logrus.New()
	}

	converter := &NL2SQLConverter{
		schema: schema,
		logger: logger,
	}

	converter.initializeTemplates()
	return converter
}

// Convert converts natural language to SQL
func (c *NL2SQLConverter) Convert(naturalLanguage string) (*ConversionResult, error) {
	// Clean and normalize input
	input := strings.TrimSpace(strings.ToLower(naturalLanguage))

	// Try each template
	for _, template := range c.templates {
		pattern := regexp.MustCompile(template.Pattern)
		if matches := pattern.FindStringSubmatch(input); len(matches) > 0 {
			sql := template.Builder(matches, c.schema)
			if sql != "" {
				return &ConversionResult{
					SQL:        sql,
					Confidence: calculateConfidence(input, matches),
					Template:   template.Description,
				}, nil
			}
		}
	}

	// If no template matches, provide suggestions
	suggestions := c.getSuggestions(input)
	return &ConversionResult{
		SQL:         "",
		Confidence:  0.0,
		Suggestions: suggestions,
	}, fmt.Errorf("could not convert to SQL - try rephrasing your query")
}

func (c *NL2SQLConverter) initializeTemplates() {
	c.templates = []QueryTemplate{
		// Basic SELECT queries
		{
			Pattern:     `^(?:show|get|select|list|display|fetch|find)(?: all| the)? (\w+)(?:\s+(?:data|records|rows|entries))?$`,
			Description: "Select all from table",
			Examples:    []string{"show users", "get all customers", "list products"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				return fmt.Sprintf("SELECT * FROM %s LIMIT 100", table)
			},
		},

		// SELECT with specific columns
		{
			Pattern:     `^(?:show|get|select) (?:the )?(\w+)(?: and (\w+))?(?: and (\w+))? (?:from|of|in) (?:the )?(\w+)`,
			Description: "Select specific columns",
			Examples:    []string{"show name and email from users", "get id and title from posts"},
			Builder: func(matches []string, schema *Schema) string {
				columns := []string{matches[1]}
				if matches[2] != "" {
					columns = append(columns, matches[2])
				}
				if matches[3] != "" {
					columns = append(columns, matches[3])
				}
				table := normalizeTableName(matches[4])
				return fmt.Sprintf("SELECT %s FROM %s", strings.Join(columns, ", "), table)
			},
		},

		// COUNT queries
		{
			Pattern:     `^(?:count|how many)(?: the)? (\w+)(?: are there| exist| do we have)?`,
			Description: "Count records",
			Examples:    []string{"count users", "how many orders are there", "count the products"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				return fmt.Sprintf("SELECT COUNT(*) AS count FROM %s", table)
			},
		},

		// Find with WHERE condition (equality)
		{
			Pattern:     `^(?:find|get|show|select) (?:all )?(\w+) (?:where|with|having) (\w+) (?:is|equals?|=) ['"]?([^'"]+)['"]?`,
			Description: "Select with WHERE equals",
			Examples:    []string{"find users where status is active", "get products with category = electronics"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				value := matches[3]

				// Check if value is numeric
				if _, err := strconv.Atoi(value); err == nil {
					return fmt.Sprintf("SELECT * FROM %s WHERE %s = %s", table, column, value)
				}
				return fmt.Sprintf("SELECT * FROM %s WHERE %s = '%s'", table, column, value)
			},
		},

		// Find with LIKE condition
		{
			Pattern:     `^(?:find|search|get) (?:all )?(\w+) (?:where|with) (\w+) (?:contains?|has|includes?) ['"]?([^'"]+)['"]?`,
			Description: "Select with LIKE",
			Examples:    []string{"find users where name contains john", "search products with title has phone"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				value := matches[3]
				return fmt.Sprintf("SELECT * FROM %s WHERE %s LIKE '%%%s%%'", table, column, value)
			},
		},

		// Find with comparison operators
		{
			Pattern:     `^(?:find|get|show) (?:all )?(\w+) (?:where|with) (\w+) (?:is )?(greater than|less than|>=?|<=?|>) (\d+)`,
			Description: "Select with comparison",
			Examples:    []string{"find products where price greater than 100", "get users with age > 18"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				operator := normalizeOperator(matches[3])
				value := matches[4]
				return fmt.Sprintf("SELECT * FROM %s WHERE %s %s %s", table, column, operator, value)
			},
		},

		// Find between range
		{
			Pattern:     `^(?:find|get|show) (?:all )?(\w+) (?:where|with) (\w+) (?:is )?between (\d+) and (\d+)`,
			Description: "Select with BETWEEN",
			Examples:    []string{"find products where price between 10 and 100", "get orders with total between 50 and 200"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				min := matches[3]
				max := matches[4]
				return fmt.Sprintf("SELECT * FROM %s WHERE %s BETWEEN %s AND %s", table, column, min, max)
			},
		},

		// ORDER BY queries
		{
			Pattern:     `^(?:show|get|list) (?:all )?(\w+) (?:sorted|ordered) by (\w+)(?: (desc|descending|asc|ascending))?`,
			Description: "Select with ORDER BY",
			Examples:    []string{"show users ordered by name", "list products sorted by price desc"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				order := "ASC"
				if matches[3] != "" && strings.Contains(matches[3], "desc") {
					order = "DESC"
				}
				return fmt.Sprintf("SELECT * FROM %s ORDER BY %s %s", table, column, order)
			},
		},

		// TOP/LIMIT queries
		{
			Pattern:     `^(?:show|get|list) (?:the )?(?:top|first|last) (\d+) (\w+)(?: ordered by (\w+))?`,
			Description: "Select with LIMIT",
			Examples:    []string{"show top 10 users", "get first 5 products ordered by price"},
			Builder: func(matches []string, schema *Schema) string {
				limit := matches[1]
				table := normalizeTableName(matches[2])
				orderBy := ""
				if matches[3] != "" {
					orderBy = fmt.Sprintf(" ORDER BY %s", matches[3])
					if strings.Contains(matches[0], "last") {
						orderBy += " DESC"
					}
				}
				return fmt.Sprintf("SELECT * FROM %s%s LIMIT %s", table, orderBy, limit)
			},
		},

		// Aggregate queries - SUM
		{
			Pattern:     `^(?:what is |calculate |get )?(?:the )?(?:sum|total) (?:of )?(\w+)(?: for| from| in)? (?:the )?(\w+)`,
			Description: "Calculate SUM",
			Examples:    []string{"sum of amount in orders", "what is the total price from products", "calculate sum of quantity"},
			Builder: func(matches []string, schema *Schema) string {
				column := matches[1]
				table := normalizeTableName(matches[2])
				return fmt.Sprintf("SELECT SUM(%s) AS total FROM %s", column, table)
			},
		},

		// Aggregate queries - AVG
		{
			Pattern:     `^(?:what is |calculate |get )?(?:the )?average (?:of )?(\w+)(?: for| from| in)? (?:the )?(\w+)`,
			Description: "Calculate AVERAGE",
			Examples:    []string{"average price from products", "what is the average age of users"},
			Builder: func(matches []string, schema *Schema) string {
				column := matches[1]
				table := normalizeTableName(matches[2])
				return fmt.Sprintf("SELECT AVG(%s) AS average FROM %s", column, table)
			},
		},

		// Aggregate queries - MAX
		{
			Pattern:     `^(?:what is |find |get )?(?:the )?(?:max|maximum|highest) (\w+)(?: from| in)? (?:the )?(\w+)`,
			Description: "Find MAX value",
			Examples:    []string{"maximum price from products", "find highest salary in employees"},
			Builder: func(matches []string, schema *Schema) string {
				column := matches[1]
				table := normalizeTableName(matches[2])
				return fmt.Sprintf("SELECT MAX(%s) AS maximum FROM %s", column, table)
			},
		},

		// Aggregate queries - MIN
		{
			Pattern:     `^(?:what is |find |get )?(?:the )?(?:min|minimum|lowest) (\w+)(?: from| in)? (?:the )?(\w+)`,
			Description: "Find MIN value",
			Examples:    []string{"minimum price from products", "find lowest age in users"},
			Builder: func(matches []string, schema *Schema) string {
				column := matches[1]
				table := normalizeTableName(matches[2])
				return fmt.Sprintf("SELECT MIN(%s) AS minimum FROM %s", column, table)
			},
		},

		// GROUP BY queries
		{
			Pattern:     `^count (\w+) (?:grouped |group )?by (\w+)`,
			Description: "Count with GROUP BY",
			Examples:    []string{"count users grouped by status", "count orders by customer_id"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				return fmt.Sprintf("SELECT %s, COUNT(*) AS count FROM %s GROUP BY %s", column, table, column)
			},
		},

		// JOIN queries (simple)
		{
			Pattern:     `^(?:show|get) (\w+) (?:with|and) (?:their )?(\w+)`,
			Description: "Simple JOIN",
			Examples:    []string{"show users with their orders", "get products and categories"},
			Builder: func(matches []string, schema *Schema) string {
				table1 := normalizeTableName(matches[1])
				table2 := normalizeTableName(matches[2])

				// Try to infer the join condition
				fk := fmt.Sprintf("%s_id", singularize(table1))
				return fmt.Sprintf("SELECT * FROM %s JOIN %s ON %s.id = %s.%s",
					table1, table2, table1, table2, fk)
			},
		},

		// DISTINCT queries
		{
			Pattern:     `^(?:show|get|list) (?:all )?(?:unique|distinct) (\w+)(?: from| in)? (?:the )?(\w+)`,
			Description: "Select DISTINCT",
			Examples:    []string{"show unique categories from products", "list distinct statuses in orders"},
			Builder: func(matches []string, schema *Schema) string {
				column := matches[1]
				table := normalizeTableName(matches[2])
				return fmt.Sprintf("SELECT DISTINCT %s FROM %s", column, table)
			},
		},

		// NULL checks
		{
			Pattern:     `^(?:find|get|show) (\w+) (?:where|with) (\w+) is (?:null|empty|missing|not set)`,
			Description: "Select with NULL check",
			Examples:    []string{"find users where email is null", "get products with description is empty"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				return fmt.Sprintf("SELECT * FROM %s WHERE %s IS NULL", table, column)
			},
		},

		// NOT NULL checks
		{
			Pattern:     `^(?:find|get|show) (\w+) (?:where|with) (\w+) is not (?:null|empty|missing)`,
			Description: "Select with NOT NULL",
			Examples:    []string{"find users where email is not null", "get products with image is not empty"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				return fmt.Sprintf("SELECT * FROM %s WHERE %s IS NOT NULL", table, column)
			},
		},

		// IN queries
		{
			Pattern:     `^(?:find|get|show) (\w+) (?:where|with) (\w+) (?:is )?in \(([^)]+)\)`,
			Description: "Select with IN clause",
			Examples:    []string{"find users where id in (1,2,3)", "get products with category in (electronics,books)"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				values := matches[3]

				// Check if values are numeric
				if regexp.MustCompile(`^\d+(?:,\s*\d+)*$`).MatchString(values) {
					return fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", table, column, values)
				}

				// String values - add quotes
				parts := strings.Split(values, ",")
				quotedParts := []string{}
				for _, part := range parts {
					part = strings.TrimSpace(part)
					part = strings.Trim(part, "'\"")
					quotedParts = append(quotedParts, fmt.Sprintf("'%s'", part))
				}
				return fmt.Sprintf("SELECT * FROM %s WHERE %s IN (%s)", table, column, strings.Join(quotedParts, ", "))
			},
		},

		// DELETE queries
		{
			Pattern:     `^(?:delete|remove) (\w+) (?:where|with) (\w+) (?:is|equals?|=) ['"]?([^'"]+)['"]?`,
			Description: "Delete with WHERE",
			Examples:    []string{"delete users where status is inactive", "remove products with stock = 0"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				column := matches[2]
				value := matches[3]

				if _, err := strconv.Atoi(value); err == nil {
					return fmt.Sprintf("DELETE FROM %s WHERE %s = %s", table, column, value)
				}
				return fmt.Sprintf("DELETE FROM %s WHERE %s = '%s'", table, column, value)
			},
		},

		// UPDATE queries
		{
			Pattern:     `^(?:update|change|set|modify) (\w+) set (\w+) (?:to|=) ['"]?([^'"]+)['"]? (?:where|for) (\w+) (?:is|equals?|=) ['"]?([^'"]+)['"]?`,
			Description: "Update with WHERE",
			Examples:    []string{"update users set status to active where id = 1", "change products set price = 99 where sku is ABC123"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				setColumn := matches[2]
				setValue := matches[3]
				whereColumn := matches[4]
				whereValue := matches[5]

				// Format values based on type
				if _, err := strconv.Atoi(setValue); err != nil {
					setValue = fmt.Sprintf("'%s'", setValue)
				}
				if _, err := strconv.Atoi(whereValue); err != nil {
					whereValue = fmt.Sprintf("'%s'", whereValue)
				}

				return fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = %s",
					table, setColumn, setValue, whereColumn, whereValue)
			},
		},

		// INSERT queries
		{
			Pattern:     `^(?:insert|add|create)(?: a| new)? (\w+) with (.+)`,
			Description: "Insert new record",
			Examples:    []string{"insert user with name john and email john@example.com", "add product with title iPhone and price 999"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				fieldsStr := matches[2]

				// Parse field-value pairs
				fieldPattern := regexp.MustCompile(`(\w+)\s+['"]?([^,]+?)['"]?(?:\s+and\s+|\s*,\s*|$)`)
				fieldMatches := fieldPattern.FindAllStringSubmatch(fieldsStr, -1)

				if len(fieldMatches) == 0 {
					return ""
				}

				var columns []string
				var values []string

				for _, match := range fieldMatches {
					if len(match) > 2 {
						columns = append(columns, match[1])
						value := strings.TrimSpace(match[2])
						value = strings.TrimSuffix(value, "and")
						value = strings.TrimSpace(value)

						// Check if numeric
						if _, err := strconv.Atoi(value); err == nil {
							values = append(values, value)
						} else {
							values = append(values, fmt.Sprintf("'%s'", value))
						}
					}
				}

				if len(columns) > 0 {
					return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
						table,
						strings.Join(columns, ", "),
						strings.Join(values, ", "))
				}
				return ""
			},
		},

		// Date-based queries
		{
			Pattern:     `^(?:show|get|find) (?:all )?(\w+) from (today|yesterday|this week|this month|last month)`,
			Description: "Select with date filter",
			Examples:    []string{"show orders from today", "get users from this month"},
			Builder: func(matches []string, schema *Schema) string {
				table := normalizeTableName(matches[1])
				period := matches[2]

				dateColumn := "created_at" // Default date column
				dateCondition := ""

				switch period {
				case "today":
					dateCondition = "DATE(created_at) = CURDATE()"
				case "yesterday":
					dateCondition = "DATE(created_at) = DATE_SUB(CURDATE(), INTERVAL 1 DAY)"
				case "this week":
					dateCondition = "YEARWEEK(created_at) = YEARWEEK(CURDATE())"
				case "this month":
					dateCondition = "MONTH(created_at) = MONTH(CURDATE()) AND YEAR(created_at) = YEAR(CURDATE())"
				case "last month":
					dateCondition = "MONTH(created_at) = MONTH(DATE_SUB(CURDATE(), INTERVAL 1 MONTH))"
				}

				if dateCondition != "" {
					return fmt.Sprintf("SELECT * FROM %s WHERE %s", table, dateCondition)
				}
				return ""
			},
		},
	}
}

func normalizeTableName(name string) string {
	// Handle common pluralization
	if strings.HasSuffix(name, "ies") {
		// categories -> category (but keep "series" as is)
		if name != "series" {
			return strings.TrimSuffix(name, "ies") + "y"
		}
	}

	// For now, keep the name as is (many databases use plural table names)
	return name
}

func normalizeOperator(op string) string {
	switch strings.ToLower(op) {
	case "greater than", "gt":
		return ">"
	case "less than", "lt":
		return "<"
	case "greater than or equal", "gte", ">=":
		return ">="
	case "less than or equal", "lte", "<=":
		return "<="
	default:
		return op
	}
}

func singularize(name string) string {
	// Simple singularization for common cases
	if strings.HasSuffix(name, "ies") {
		return strings.TrimSuffix(name, "ies") + "y"
	}
	if strings.HasSuffix(name, "es") {
		return strings.TrimSuffix(name, "es")
	}
	if strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss") {
		return strings.TrimSuffix(name, "s")
	}
	return name
}

func calculateConfidence(input string, matches []string) float64 {
	// Calculate confidence based on how much of the input was matched
	totalLen := len(strings.Join(matches[1:], ""))
	inputLen := len(strings.ReplaceAll(input, " ", ""))

	if inputLen == 0 {
		return 0.0
	}

	confidence := float64(totalLen) / float64(inputLen)

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	// Boost confidence if we matched the entire input
	if matches[0] == input {
		confidence = 1.0
	}

	return confidence
}

func (c *NL2SQLConverter) getSuggestions(input string) []string {
	suggestions := []string{
		"Try phrases like:",
		"  - 'show all users'",
		"  - 'count products where price > 100'",
		"  - 'find orders from today'",
		"  - 'get top 10 customers ordered by total_spent'",
		"  - 'update users set status to active where id = 1'",
	}

	// Add context-specific suggestions based on input keywords
	if strings.Contains(input, "select") || strings.Contains(input, "show") {
		suggestions = append(suggestions,
			"  - 'show [table_name]'",
			"  - 'select [columns] from [table]'",
		)
	}

	if strings.Contains(input, "count") {
		suggestions = append(suggestions,
			"  - 'count [table_name]'",
			"  - 'count [table] grouped by [column]'",
		)
	}

	if strings.Contains(input, "update") {
		suggestions = append(suggestions,
			"  - 'update [table] set [column] to [value] where [condition]'",
		)
	}

	return suggestions
}

// GetSupportedPatterns returns all supported natural language patterns
func (c *NL2SQLConverter) GetSupportedPatterns() []map[string]interface{} {
	patterns := []map[string]interface{}{}

	for _, template := range c.templates {
		patterns = append(patterns, map[string]interface{}{
			"description": template.Description,
			"examples":    template.Examples,
			"pattern":     template.Pattern,
		})
	}

	return patterns
}