package analyzer

import (
	"fmt"
	"strings"
)

// Explain generates a plain English explanation of what a SQL query does
func Explain(sql string) (string, error) {
	parsed, err := Parse(sql)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQL: %w", err)
	}

	var explanation strings.Builder

	switch parsed.Type {
	case "SELECT":
		explanation.WriteString(explainSelect(parsed, sql))
	case "INSERT":
		explanation.WriteString(explainInsert(parsed, sql))
	case "UPDATE":
		explanation.WriteString(explainUpdate(parsed, sql))
	case "DELETE":
		explanation.WriteString(explainDelete(parsed, sql))
	default:
		return "This is a complex SQL query that performs database operations.", nil
	}

	return explanation.String(), nil
}

func explainSelect(parsed *ParsedQuery, sql string) string {
	var parts []string

	// Start with the basic action
	intro := "This query retrieves "

	// Describe what is being selected
	if parsed.HasWildcard {
		intro += "all columns"
	} else if len(parsed.Columns) > 0 {
		if len(parsed.Columns) == 1 {
			intro += fmt.Sprintf("the '%s' column", parsed.Columns[0])
		} else if len(parsed.Columns) <= 3 {
			intro += fmt.Sprintf("the columns: %s", joinWithAnd(parsed.Columns))
		} else {
			intro += fmt.Sprintf("%d columns including %s", len(parsed.Columns), joinWithAnd(parsed.Columns[:2]))
		}
	} else {
		intro += "data"
	}

	// Describe the source tables
	if len(parsed.Tables) == 1 {
		intro += fmt.Sprintf(" from the '%s' table", parsed.Tables[0])
	} else if len(parsed.Tables) > 1 {
		intro += fmt.Sprintf(" from %d tables (%s)", len(parsed.Tables), joinWithAnd(parsed.Tables))
	}

	parts = append(parts, intro)

	// Explain JOINs
	if len(parsed.JoinColumns) > 0 {
		parts = append(parts, fmt.Sprintf("The tables are combined using JOIN conditions on: %s",
			joinWithAnd(parsed.JoinColumns)))
	}

	// Explain WHERE conditions
	if len(parsed.WhereColumns) > 0 {
		whereDesc := "The results are filtered "
		if len(parsed.WhereColumns) == 1 {
			whereDesc += fmt.Sprintf("based on the '%s' column", parsed.WhereColumns[0])
		} else {
			whereDesc += fmt.Sprintf("using conditions on: %s", joinWithAnd(parsed.WhereColumns))
		}
		parts = append(parts, whereDesc)
	}

	// Explain GROUP BY
	if len(parsed.GroupBy) > 0 {
		parts = append(parts, fmt.Sprintf("The results are grouped by: %s",
			joinWithAnd(parsed.GroupBy)))
	}

	// Explain ORDER BY
	if len(parsed.OrderBy) > 0 {
		parts = append(parts, fmt.Sprintf("The results are sorted by: %s",
			joinWithAnd(parsed.OrderBy)))
	}

	// Explain DISTINCT
	if parsed.HasDistinct {
		parts = append(parts, "Duplicate rows are removed from the results")
	}

	// Explain LIMIT
	if parsed.Limit > 0 {
		parts = append(parts, fmt.Sprintf("Only the first %d rows are returned", parsed.Limit))
	}

	// Explain subqueries
	if parsed.HasSubquery {
		parts = append(parts, "This query contains subqueries for complex filtering or data retrieval")
	}

	// Check for aggregations
	if hasAggregation(sql) {
		parts = append(parts, "Aggregate functions are used to calculate values like counts, sums, or averages")
	}

	return strings.Join(parts, ". ") + "."
}

func explainInsert(parsed *ParsedQuery, sql string) string {
	var parts []string

	if len(parsed.Tables) > 0 {
		intro := fmt.Sprintf("This query inserts new data into the '%s' table", parsed.Tables[0])
		parts = append(parts, intro)
	} else {
		parts = append(parts, "This query inserts new data into a table")
	}

	if len(parsed.Columns) > 0 {
		if len(parsed.Columns) == 1 {
			parts = append(parts, fmt.Sprintf("It sets the value for the '%s' column", parsed.Columns[0]))
		} else if len(parsed.Columns) <= 5 {
			parts = append(parts, fmt.Sprintf("It sets values for the columns: %s",
				joinWithAnd(parsed.Columns)))
		} else {
			parts = append(parts, fmt.Sprintf("It sets values for %d columns", len(parsed.Columns)))
		}
	}

	// Check for bulk insert
	if strings.Count(strings.ToUpper(sql), "VALUES") > 1 {
		parts = append(parts, "Multiple rows are being inserted in a single operation")
	}

	// Check for INSERT SELECT
	if strings.Contains(strings.ToUpper(sql), "SELECT") {
		parts = append(parts, "The data being inserted comes from another query")
	}

	return strings.Join(parts, ". ") + "."
}

func explainUpdate(parsed *ParsedQuery, sql string) string {
	var parts []string

	if len(parsed.Tables) > 0 {
		intro := fmt.Sprintf("This query modifies existing data in the '%s' table", parsed.Tables[0])
		parts = append(parts, intro)
	} else {
		parts = append(parts, "This query modifies existing data in a table")
	}

	if len(parsed.Columns) > 0 {
		if len(parsed.Columns) == 1 {
			parts = append(parts, fmt.Sprintf("It updates the '%s' column", parsed.Columns[0]))
		} else if len(parsed.Columns) <= 5 {
			parts = append(parts, fmt.Sprintf("It updates the columns: %s",
				joinWithAnd(parsed.Columns)))
		} else {
			parts = append(parts, fmt.Sprintf("It updates %d columns", len(parsed.Columns)))
		}
	}

	if len(parsed.WhereColumns) > 0 {
		whereDesc := "Only rows that match conditions on "
		if len(parsed.WhereColumns) == 1 {
			whereDesc += fmt.Sprintf("the '%s' column are updated", parsed.WhereColumns[0])
		} else {
			whereDesc += fmt.Sprintf("these columns are updated: %s", joinWithAnd(parsed.WhereColumns))
		}
		parts = append(parts, whereDesc)
	} else {
		parts = append(parts, "⚠️ ALL rows in the table will be updated (no WHERE clause)")
	}

	return strings.Join(parts, ". ") + "."
}

func explainDelete(parsed *ParsedQuery, sql string) string {
	var parts []string

	if len(parsed.Tables) > 0 {
		intro := fmt.Sprintf("This query removes rows from the '%s' table", parsed.Tables[0])
		parts = append(parts, intro)
	} else {
		parts = append(parts, "This query removes rows from a table")
	}

	if len(parsed.WhereColumns) > 0 {
		whereDesc := "Only rows that match conditions on "
		if len(parsed.WhereColumns) == 1 {
			whereDesc += fmt.Sprintf("the '%s' column are deleted", parsed.WhereColumns[0])
		} else {
			whereDesc += fmt.Sprintf("these columns are deleted: %s", joinWithAnd(parsed.WhereColumns))
		}
		parts = append(parts, whereDesc)
	} else {
		parts = append(parts, "⚠️ ALL rows in the table will be deleted (no WHERE clause)")
	}

	return strings.Join(parts, ". ") + "."
}

// ExplainComplex provides more detailed explanations for complex queries
func ExplainComplex(sql string) (map[string]interface{}, error) {
	parsed, err := Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	result := map[string]interface{}{
		"summary":     "",
		"type":        parsed.Type,
		"tables":      parsed.Tables,
		"complexity":  calculateQueryComplexity(parsed),
		"operations":  []string{},
		"warnings":    []string{},
		"suggestions": []string{},
	}

	// Generate summary
	summary, _ := Explain(sql)
	result["summary"] = summary

	// List operations
	ops := []string{}
	if parsed.HasWildcard {
		ops = append(ops, "Select all columns")
	}
	if len(parsed.WhereColumns) > 0 {
		ops = append(ops, fmt.Sprintf("Filter on %d columns", len(parsed.WhereColumns)))
	}
	if len(parsed.JoinColumns) > 0 {
		ops = append(ops, fmt.Sprintf("Join on %d columns", len(parsed.JoinColumns)))
	}
	if len(parsed.GroupBy) > 0 {
		ops = append(ops, "Group results")
	}
	if len(parsed.OrderBy) > 0 {
		ops = append(ops, "Sort results")
	}
	if parsed.HasDistinct {
		ops = append(ops, "Remove duplicates")
	}
	if parsed.HasSubquery {
		ops = append(ops, "Execute subquery")
	}
	result["operations"] = ops

	// Add warnings
	warnings := []string{}
	if parsed.Type == "DELETE" && len(parsed.WhereColumns) == 0 {
		warnings = append(warnings, "This will delete ALL rows")
	}
	if parsed.Type == "UPDATE" && len(parsed.WhereColumns) == 0 {
		warnings = append(warnings, "This will update ALL rows")
	}
	if parsed.HasWildcard {
		warnings = append(warnings, "Using SELECT * may retrieve unnecessary data")
	}
	if len(parsed.Tables) > 3 {
		warnings = append(warnings, "Multiple table joins may impact performance")
	}
	result["warnings"] = warnings

	// Add suggestions
	suggestions := []string{}
	if parsed.HasWildcard {
		suggestions = append(suggestions, "Specify only needed columns instead of SELECT *")
	}
	if parsed.Limit == 0 && parsed.Type == "SELECT" {
		suggestions = append(suggestions, "Consider adding LIMIT to prevent large result sets")
	}
	if len(parsed.WhereColumns) == 0 && parsed.Type == "SELECT" {
		suggestions = append(suggestions, "Add WHERE clause to filter results")
	}
	result["suggestions"] = suggestions

	return result, nil
}

func calculateQueryComplexity(parsed *ParsedQuery) string {
	score := 0

	// Calculate complexity score
	score += len(parsed.Tables)
	score += len(parsed.JoinColumns) * 2
	if parsed.HasSubquery {
		score += 3
	}
	if len(parsed.GroupBy) > 0 {
		score += 2
	}
	if parsed.HasDistinct {
		score += 1
	}

	switch {
	case score <= 2:
		return "simple"
	case score <= 5:
		return "moderate"
	default:
		return "complex"
	}
}

func hasAggregation(sql string) bool {
	upperSQL := strings.ToUpper(sql)
	aggregates := []string{"COUNT(", "SUM(", "AVG(", "MAX(", "MIN(", "GROUP_CONCAT("}
	for _, agg := range aggregates {
		if strings.Contains(upperSQL, agg) {
			return true
		}
	}
	return false
}

func joinWithAnd(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	// For more than 2 items
	result := strings.Join(items[:len(items)-1], ", ")
	result += ", and " + items[len(items)-1]
	return result
}