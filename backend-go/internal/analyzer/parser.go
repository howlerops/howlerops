package analyzer

import (
	"regexp"
	"strconv"
	"strings"
)

// ParsedQuery represents a parsed SQL query with extracted components
type ParsedQuery struct {
	Type         string   `json:"type"`          // SELECT, INSERT, UPDATE, DELETE
	Tables       []string `json:"tables"`        // Table names
	Columns      []string `json:"columns"`       // Column names in SELECT/INSERT
	WhereColumns []string `json:"where_columns"` // Columns in WHERE clause
	JoinColumns  []string `json:"join_columns"`  // Columns in JOIN conditions
	OrderBy      []string `json:"order_by"`      // Columns in ORDER BY
	GroupBy      []string `json:"group_by"`      // Columns in GROUP BY
	HasSubquery  bool     `json:"has_subquery"`  // Contains subquery
	HasWildcard  bool     `json:"has_wildcard"`  // Contains SELECT *
	HasDistinct  bool     `json:"has_distinct"`  // Has DISTINCT keyword
	Limit        int      `json:"limit"`         // LIMIT value if present
}

// Common SQL patterns
var (
	insertPattern   = regexp.MustCompile(`(?i)^INSERT\s+INTO\s+(\S+)\s*\(([^)]+)\)\s*VALUES`)
	updatePattern   = regexp.MustCompile(`(?i)^UPDATE\s+(\S+)\s+SET\s+(.+?)(\s+WHERE.+)?$`)
	deletePattern   = regexp.MustCompile(`(?i)^DELETE\s+FROM\s+(\S+)(\s+WHERE.+)?$`)
	tablePattern    = regexp.MustCompile(`(?i)FROM\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\s+(?:AS\s+)?[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	joinPattern     = regexp.MustCompile(`(?i)(?:INNER|LEFT|RIGHT|FULL|OUTER|CROSS)?\s*JOIN\s+([a-zA-Z_][a-zA-Z0-9_]*(?:\s+(?:AS\s+)?[a-zA-Z_][a-zA-Z0-9_]*)?)`)
	wherePattern    = regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+(?:GROUP|ORDER|LIMIT|$))`)
	columnPattern   = regexp.MustCompile(`(?i)([a-zA-Z_][a-zA-Z0-9_]*\.)?([a-zA-Z_][a-zA-Z0-9_]*)`)
	orderByPattern  = regexp.MustCompile(`(?i)ORDER\s+BY\s+([^)]+?)(?:\s+(?:LIMIT|$))`)
	groupByPattern  = regexp.MustCompile(`(?i)GROUP\s+BY\s+([^)]+?)(?:\s+(?:HAVING|ORDER|LIMIT|$))`)
	limitPattern    = regexp.MustCompile(`(?i)LIMIT\s+(\d+)`)
	subqueryPattern = regexp.MustCompile(`\([^)]*SELECT[^)]*\)`)
	functionPattern = regexp.MustCompile(`(?i)(COUNT|SUM|AVG|MAX|MIN|UPPER|LOWER|LENGTH|SUBSTR|CONCAT|COALESCE)\s*\(`)
)

// Parse analyzes a SQL query and extracts its components
func Parse(sql string) (*ParsedQuery, error) {
	sql = strings.TrimSpace(sql)
	parsed := &ParsedQuery{}

	// Determine query type
	upperSQL := strings.ToUpper(sql)
	switch {
	case strings.HasPrefix(upperSQL, "SELECT"):
		parsed.Type = "SELECT"
		parseSelectQuery(sql, parsed)
	case strings.HasPrefix(upperSQL, "INSERT"):
		parsed.Type = "INSERT"
		parseInsertQuery(sql, parsed)
	case strings.HasPrefix(upperSQL, "UPDATE"):
		parsed.Type = "UPDATE"
		parseUpdateQuery(sql, parsed)
	case strings.HasPrefix(upperSQL, "DELETE"):
		parsed.Type = "DELETE"
		parseDeleteQuery(sql, parsed)
	default:
		parsed.Type = "UNKNOWN"
	}

	// Check for subqueries
	parsed.HasSubquery = subqueryPattern.MatchString(sql)

	// Extract LIMIT
	if limitMatch := limitPattern.FindStringSubmatch(sql); len(limitMatch) > 1 {
		// Parse limit value
		trimmed := strings.TrimSpace(limitMatch[1])
		if limit, err := strconv.Atoi(trimmed); err == nil {
			parsed.Limit = limit
		}
	}

	return parsed, nil
}

func parseSelectQuery(sql string, parsed *ParsedQuery) {
	// Check for DISTINCT
	parsed.HasDistinct = regexp.MustCompile(`(?i)SELECT\s+DISTINCT`).MatchString(sql)

	// Check for SELECT *
	parsed.HasWildcard = strings.Contains(strings.ToUpper(sql), "SELECT *") ||
		regexp.MustCompile(`(?i)SELECT\s+[^,]+\.\*`).MatchString(sql)

	// Extract columns from SELECT clause
	if match := regexp.MustCompile(`(?i)SELECT\s+(?:DISTINCT\s+)?(.+?)\s+FROM`).FindStringSubmatch(sql); len(match) > 1 {
		columnsStr := match[1]
		if !parsed.HasWildcard {
			columns := splitColumns(columnsStr)
			for _, col := range columns {
				// Remove aliases and functions
				col = cleanColumn(col)
				if col != "" && col != "*" {
					parsed.Columns = append(parsed.Columns, col)
				}
			}
		}
	}

	// Extract tables
	extractTables(sql, parsed)

	// Extract WHERE columns
	extractWhereColumns(sql, parsed)

	// Extract JOIN columns
	extractJoinColumns(sql, parsed)

	// Extract ORDER BY columns
	if match := orderByPattern.FindStringSubmatch(sql); len(match) > 1 {
		orderCols := splitColumns(match[1])
		for _, col := range orderCols {
			col = cleanOrderByColumn(col)
			if col != "" {
				parsed.OrderBy = append(parsed.OrderBy, col)
			}
		}
	}

	// Extract GROUP BY columns
	if match := groupByPattern.FindStringSubmatch(sql); len(match) > 1 {
		groupCols := splitColumns(match[1])
		for _, col := range groupCols {
			col = cleanColumn(col)
			if col != "" {
				parsed.GroupBy = append(parsed.GroupBy, col)
			}
		}
	}
}

func parseInsertQuery(sql string, parsed *ParsedQuery) {
	if match := insertPattern.FindStringSubmatch(sql); len(match) > 2 {
		// Extract table name
		parsed.Tables = append(parsed.Tables, cleanTableName(match[1]))

		// Extract columns
		columnsStr := match[2]
		columns := splitColumns(columnsStr)
		for _, col := range columns {
			col = cleanColumn(col)
			if col != "" {
				parsed.Columns = append(parsed.Columns, col)
			}
		}
	}
}

func parseUpdateQuery(sql string, parsed *ParsedQuery) {
	if match := updatePattern.FindStringSubmatch(sql); len(match) > 1 {
		// Extract table name
		parsed.Tables = append(parsed.Tables, cleanTableName(match[1]))

		// Extract SET columns
		if len(match) > 2 {
			setClauses := strings.Split(match[2], ",")
			for _, clause := range setClauses {
				if parts := strings.Split(clause, "="); len(parts) > 0 {
					col := cleanColumn(parts[0])
					if col != "" {
						parsed.Columns = append(parsed.Columns, col)
					}
				}
			}
		}

		// Extract WHERE columns
		extractWhereColumns(sql, parsed)
	}
}

func parseDeleteQuery(sql string, parsed *ParsedQuery) {
	if match := deletePattern.FindStringSubmatch(sql); len(match) > 1 {
		// Extract table name
		parsed.Tables = append(parsed.Tables, cleanTableName(match[1]))

		// Extract WHERE columns
		extractWhereColumns(sql, parsed)
	}
}

func extractTables(sql string, parsed *ParsedQuery) {
	// Extract FROM tables
	if matches := tablePattern.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				table := cleanTableName(match[1])
				if table != "" && !contains(parsed.Tables, table) {
					parsed.Tables = append(parsed.Tables, table)
				}
			}
		}
	}

	// Extract JOIN tables
	if matches := joinPattern.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				table := cleanTableName(match[1])
				if table != "" && !contains(parsed.Tables, table) {
					parsed.Tables = append(parsed.Tables, table)
				}
			}
		}
	}
}

func extractWhereColumns(sql string, parsed *ParsedQuery) {
	if match := wherePattern.FindStringSubmatch(sql); len(match) > 1 {
		whereClause := match[1]
		// Extract column names from WHERE clause
		columns := extractColumnsFromClause(whereClause)
		for _, col := range columns {
			if col != "" && !contains(parsed.WhereColumns, col) {
				parsed.WhereColumns = append(parsed.WhereColumns, col)
			}
		}
	}
}

func extractJoinColumns(sql string, parsed *ParsedQuery) {
	// Look for ON clauses after JOINs
	onPattern := regexp.MustCompile(`(?i)ON\s+([^)]+?)(?:\s+(?:WHERE|GROUP|ORDER|LIMIT|JOIN|$))`)
	if matches := onPattern.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				columns := extractColumnsFromClause(match[1])
				for _, col := range columns {
					if col != "" && !contains(parsed.JoinColumns, col) {
						parsed.JoinColumns = append(parsed.JoinColumns, col)
					}
				}
			}
		}
	}
}

func extractColumnsFromClause(clause string) []string {
	var columns []string

	// Remove string literals
	clause = regexp.MustCompile(`'[^']*'`).ReplaceAllString(clause, "")
	clause = regexp.MustCompile(`"[^"]*"`).ReplaceAllString(clause, "")

	// Find column references
	if matches := columnPattern.FindAllStringSubmatch(clause, -1); len(matches) > 0 {
		for _, match := range matches {
			col := match[2]
			// Skip SQL keywords and functions
			if !isSQLKeyword(col) && !isFunction(col) {
				columns = append(columns, col)
			}
		}
	}

	return columns
}

func splitColumns(columnsStr string) []string {
	var columns []string
	var current strings.Builder
	parenCount := 0

	for _, ch := range columnsStr {
		switch ch {
		case '(':
			parenCount++
			current.WriteRune(ch)
		case ')':
			parenCount--
			current.WriteRune(ch)
		case ',':
			if parenCount == 0 {
				columns = append(columns, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		columns = append(columns, strings.TrimSpace(current.String()))
	}

	return columns
}

func cleanColumn(col string) string {
	col = strings.TrimSpace(col)

	// Remove AS alias
	if idx := regexp.MustCompile(`(?i)\s+AS\s+`).FindStringIndex(col); idx != nil {
		col = col[:idx[0]]
	}

	// Remove space-separated alias
	if parts := strings.Fields(col); len(parts) > 1 {
		// Check if the last part is not a keyword
		if !isSQLKeyword(parts[len(parts)-1]) {
			col = parts[0]
		}
	}

	// Remove table prefix if present
	if idx := strings.LastIndex(col, "."); idx != -1 {
		col = col[idx+1:]
	}

	// Remove function calls
	if idx := strings.Index(col, "("); idx != -1 {
		// This might be a function, try to extract column from it
		innerContent := extractFromFunction(col)
		if innerContent != "" {
			col = innerContent
		} else {
			return ""
		}
	}

	return col
}

func cleanOrderByColumn(col string) string {
	col = strings.TrimSpace(col)

	// Remove ASC/DESC
	col = regexp.MustCompile(`(?i)\s+(ASC|DESC)$`).ReplaceAllString(col, "")

	return cleanColumn(col)
}

func cleanTableName(table string) string {
	table = strings.TrimSpace(table)

	// Remove AS alias
	if idx := regexp.MustCompile(`(?i)\s+AS\s+`).FindStringIndex(table); idx != nil {
		table = table[:idx[0]]
	}

	// Remove space-separated alias
	if parts := strings.Fields(table); len(parts) > 1 {
		table = parts[0]
	}

	return table
}

func extractFromFunction(funcCall string) string {
	// Try to extract column name from function call like COUNT(id)
	if match := regexp.MustCompile(`\(([^)]+)\)`).FindStringSubmatch(funcCall); len(match) > 1 {
		inner := strings.TrimSpace(match[1])
		if inner != "*" && !strings.Contains(inner, " ") {
			return cleanColumn(inner)
		}
	}
	return ""
}

func isSQLKeyword(word string) bool {
	keywords := []string{
		"AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "IS", "NULL",
		"TRUE", "FALSE", "ASC", "DESC", "ALL", "ANY", "SOME",
	}
	upper := strings.ToUpper(word)
	for _, kw := range keywords {
		if upper == kw {
			return true
		}
	}
	return false
}

func isFunction(word string) bool {
	functions := []string{
		"COUNT", "SUM", "AVG", "MAX", "MIN", "UPPER", "LOWER",
		"LENGTH", "SUBSTR", "SUBSTRING", "CONCAT", "COALESCE", "CAST",
	}
	upper := strings.ToUpper(word)
	for _, fn := range functions {
		if upper == fn {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// HasFunction checks if a SQL expression contains function calls
func HasFunction(expr string) bool {
	return functionPattern.MatchString(expr)
}

// ExtractTableAliases extracts table aliases from a query
func ExtractTableAliases(sql string) map[string]string {
	aliases := make(map[string]string)

	// Pattern for table AS alias or table alias
	aliasPattern := regexp.MustCompile(`(?i)([a-zA-Z_][a-zA-Z0-9_]*)\s+(?:AS\s+)?([a-zA-Z_][a-zA-Z0-9_]*)\s*(?:,|JOIN|WHERE|GROUP|ORDER|LIMIT|$)`)

	if matches := aliasPattern.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 2 {
				table := match[1]
				alias := match[2]
				if !isSQLKeyword(alias) {
					aliases[alias] = table
				}
			}
		}
	}

	return aliases
}
