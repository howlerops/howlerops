package analyzer

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// QueryAnalyzer analyzes SQL queries for optimization opportunities
type QueryAnalyzer struct {
	logger *logrus.Logger
}

// Schema represents database schema information
type Schema struct {
	Tables map[string]*Table `json:"tables"`
}

// Table represents a database table
type Table struct {
	Name    string             `json:"name"`
	Columns map[string]*Column `json:"columns"`
	Indexes map[string]*Index  `json:"indexes"`
	RowCount int64             `json:"row_count"`
}

// Column represents a database column
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Indexed  bool   `json:"indexed"`
}

// Index represents a database index
type Index struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Unique  bool     `json:"unique"`
	Primary bool     `json:"primary"`
}

// AnalysisResult contains the results of query analysis
type AnalysisResult struct {
	Suggestions []Suggestion `json:"suggestions"`
	Score       int          `json:"score"`       // 0-100
	Warnings    []Warning    `json:"warnings"`
	Complexity  string       `json:"complexity"`  // simple, moderate, complex
	EstimatedCost int        `json:"estimated_cost"` // relative cost estimate
}

// Suggestion represents an optimization suggestion
type Suggestion struct {
	Type        string `json:"type"`        // 'index', 'join', 'where', 'select', 'subquery'
	Severity    string `json:"severity"`    // 'info', 'warning', 'critical'
	Message     string `json:"message"`
	OriginalSQL string `json:"original_sql"`
	ImprovedSQL string `json:"improved_sql,omitempty"`
	Impact      string `json:"impact,omitempty"` // Expected performance impact
}

// Warning represents a potential issue
type Warning struct {
	Message  string `json:"message"`
	Severity string `json:"severity"` // 'low', 'medium', 'high'
}

// NewQueryAnalyzer creates a new query analyzer
func NewQueryAnalyzer(logger *logrus.Logger) *QueryAnalyzer {
	if logger == nil {
		logger = logrus.New()
	}
	return &QueryAnalyzer{logger: logger}
}

// Analyze analyzes a SQL query and provides optimization suggestions
func (a *QueryAnalyzer) Analyze(sql string, schema *Schema) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Suggestions: []Suggestion{},
		Warnings:    []Warning{},
		Score:       100, // Start with perfect score
		Complexity:  "simple",
	}

	// Parse the SQL query
	parsed, err := Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL: %w", err)
	}

	// Analyze based on query type
	switch parsed.Type {
	case "SELECT":
		a.analyzeSelectQuery(sql, parsed, schema, result)
	case "INSERT":
		a.analyzeInsertQuery(sql, parsed, schema, result)
	case "UPDATE":
		a.analyzeUpdateQuery(sql, parsed, schema, result)
	case "DELETE":
		a.analyzeDeleteQuery(sql, parsed, schema, result)
	}

	// Calculate complexity
	result.Complexity = a.calculateComplexity(parsed)

	// Estimate cost
	result.EstimatedCost = a.estimateCost(parsed, schema)

	// Ensure score is within bounds
	if result.Score < 0 {
		result.Score = 0
	}

	return result, nil
}

func (a *QueryAnalyzer) analyzeSelectQuery(sql string, parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	// Check for SELECT *
	if parsed.HasWildcard {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:        "select",
			Severity:    "warning",
			Message:     "Avoid using SELECT * - specify only the columns you need",
			OriginalSQL: sql,
			ImprovedSQL: a.generateImprovedSelectColumns(sql, parsed, schema),
			Impact:      "Reduces network traffic and improves query performance",
		})
		result.Score -= 15
	}

	// Check for missing WHERE clause on large tables
	if len(parsed.WhereColumns) == 0 && !parsed.HasDistinct {
		if a.isLargeTable(parsed.Tables, schema) {
			result.Warnings = append(result.Warnings, Warning{
				Message:  "Query on large table without WHERE clause may return excessive data",
				Severity: "high",
			})
			result.Score -= 20
		}
	}

	// Check for missing indexes on WHERE columns
	a.checkMissingIndexes(parsed, schema, result)

	// Check for inefficient WHERE clauses
	a.checkInefficientWhere(sql, parsed, result)

	// Check for missing JOIN conditions
	a.checkMissingJoinConditions(sql, parsed, result)

	// Check for subquery optimization opportunities
	a.checkSubqueryOptimization(sql, parsed, result)

	// Check for DISTINCT usage
	if parsed.HasDistinct {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "select",
			Severity: "info",
			Message:  "DISTINCT can be expensive - ensure it's necessary and consider using GROUP BY if aggregating",
			Impact:   "May require sorting entire result set",
		})
		result.Score -= 5
	}

	// Check for missing LIMIT on queries without aggregation
	if parsed.Limit == 0 && len(parsed.GroupBy) == 0 && !a.hasAggregateFunction(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:        "select",
			Severity:    "info",
			Message:     "Consider adding LIMIT to prevent returning excessive rows",
			OriginalSQL: sql,
			ImprovedSQL: sql + " LIMIT 100",
			Impact:      "Limits result set size and improves response time",
		})
		result.Score -= 5
	}

	// Check for OR conditions that could use IN
	a.checkOrToIn(sql, parsed, result)

	// Check for implicit type conversions
	a.checkImplicitConversions(sql, parsed, schema, result)

	// Check for correlated subqueries
	a.checkCorrelatedSubqueries(sql, parsed, result)
}

func (a *QueryAnalyzer) analyzeInsertQuery(sql string, parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	// Check for bulk insert opportunities
	if strings.Count(sql, "VALUES") == 1 {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "insert",
			Severity: "info",
			Message:  "Consider using bulk INSERT for multiple rows instead of individual INSERTs",
			Impact:   "Reduces round-trips to database and improves throughput",
		})
	}

	// Check if columns match table schema
	a.validateColumns(parsed, schema, result)
}

func (a *QueryAnalyzer) analyzeUpdateQuery(sql string, parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	// Check for UPDATE without WHERE
	if len(parsed.WhereColumns) == 0 {
		result.Warnings = append(result.Warnings, Warning{
			Message:  "UPDATE without WHERE clause will modify ALL rows",
			Severity: "high",
		})
		result.Score -= 30
	}

	// Check for missing indexes on WHERE columns
	a.checkMissingIndexes(parsed, schema, result)
}

func (a *QueryAnalyzer) analyzeDeleteQuery(sql string, parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	// Check for DELETE without WHERE
	if len(parsed.WhereColumns) == 0 {
		result.Warnings = append(result.Warnings, Warning{
			Message:  "DELETE without WHERE clause will remove ALL rows - use TRUNCATE for better performance if intentional",
			Severity: "high",
		})
		result.Score -= 30
	}

	// Check for missing indexes on WHERE columns
	a.checkMissingIndexes(parsed, schema, result)
}

func (a *QueryAnalyzer) checkMissingIndexes(parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	if schema == nil {
		return
	}

	checkedColumns := make(map[string]bool)

	// Check WHERE columns
	for _, col := range parsed.WhereColumns {
		if checkedColumns[col] {
			continue
		}
		checkedColumns[col] = true

		for _, tableName := range parsed.Tables {
			if table, ok := schema.Tables[tableName]; ok {
				if column, ok := table.Columns[col]; ok {
					if !column.Indexed && !a.isIndexed(tableName, col, schema) {
						result.Suggestions = append(result.Suggestions, Suggestion{
							Type:     "index",
							Severity: "warning",
							Message:  fmt.Sprintf("Column '%s.%s' used in WHERE clause is not indexed", tableName, col),
							Impact:   "Adding an index could significantly improve query performance",
						})
						result.Score -= 10
					}
				}
			}
		}
	}

	// Check JOIN columns
	for _, col := range parsed.JoinColumns {
		if checkedColumns[col] {
			continue
		}
		checkedColumns[col] = true

		for _, tableName := range parsed.Tables {
			if table, ok := schema.Tables[tableName]; ok {
				if column, ok := table.Columns[col]; ok {
					if !column.Indexed && !a.isIndexed(tableName, col, schema) {
						result.Suggestions = append(result.Suggestions, Suggestion{
							Type:     "index",
							Severity: "warning",
							Message:  fmt.Sprintf("Column '%s.%s' used in JOIN is not indexed", tableName, col),
							Impact:   "Adding an index could improve JOIN performance",
						})
						result.Score -= 10
					}
				}
			}
		}
	}

	// Check ORDER BY columns
	for _, col := range parsed.OrderBy {
		if checkedColumns[col] {
			continue
		}
		checkedColumns[col] = true

		for _, tableName := range parsed.Tables {
			if table, ok := schema.Tables[tableName]; ok {
				if column, ok := table.Columns[col]; ok {
					if !column.Indexed && !a.isIndexed(tableName, col, schema) {
						result.Suggestions = append(result.Suggestions, Suggestion{
							Type:     "index",
							Severity: "info",
							Message:  fmt.Sprintf("Column '%s.%s' used in ORDER BY is not indexed", tableName, col),
							Impact:   "Adding an index could improve sorting performance",
						})
						result.Score -= 5
					}
				}
			}
		}
	}
}

func (a *QueryAnalyzer) checkInefficientWhere(sql string, parsed *ParsedQuery, result *AnalysisResult) {
	upperSQL := strings.ToUpper(sql)

	// Check for functions in WHERE clause (non-sargable)
	whereMatch := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+(?:GROUP|ORDER|LIMIT|$))`).FindStringSubmatch(sql)
	if len(whereMatch) > 1 {
		whereClause := whereMatch[1]
		if HasFunction(whereClause) {
			result.Suggestions = append(result.Suggestions, Suggestion{
				Type:     "where",
				Severity: "warning",
				Message:  "Functions in WHERE clause prevent index usage (non-sargable)",
				Impact:   "Move calculations to the right side of comparisons or pre-compute values",
			})
			result.Score -= 15
		}
	}

	// Check for LIKE with leading wildcard
	if regexp.MustCompile(`(?i)LIKE\s+['"]%`).MatchString(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "where",
			Severity: "warning",
			Message:  "LIKE with leading wildcard (LIKE '%...') prevents index usage",
			Impact:   "Consider full-text search or redesigning the query",
		})
		result.Score -= 10
	}

	// Check for NOT IN with subquery
	if strings.Contains(upperSQL, "NOT IN") && strings.Contains(upperSQL, "SELECT") {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "where",
			Severity: "warning",
			Message:  "NOT IN with subquery can be inefficient - consider NOT EXISTS or LEFT JOIN",
			Impact:   "NOT EXISTS often performs better than NOT IN",
		})
		result.Score -= 10
	}

	// Check for != or <> operators
	if regexp.MustCompile(`(?i)(!=|<>)`).MatchString(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "where",
			Severity: "info",
			Message:  "Inequality operators (!=, <>) may not use indexes efficiently",
			Impact:   "Consider restructuring with IN or range conditions if possible",
		})
		result.Score -= 5
	}
}

func (a *QueryAnalyzer) checkMissingJoinConditions(sql string, parsed *ParsedQuery, result *AnalysisResult) {
	upperSQL := strings.ToUpper(sql)

	// Check for CROSS JOIN or missing ON clause
	if strings.Contains(upperSQL, "CROSS JOIN") {
		result.Warnings = append(result.Warnings, Warning{
			Message:  "CROSS JOIN creates Cartesian product - ensure this is intentional",
			Severity: "medium",
		})
		result.Score -= 15
	}

	// Check for implicit cross join (comma-separated tables without WHERE)
	if len(parsed.Tables) > 1 && strings.Contains(upperSQL, "FROM") {
		if !strings.Contains(upperSQL, "JOIN") && len(parsed.WhereColumns) == 0 {
			result.Warnings = append(result.Warnings, Warning{
				Message:  "Multiple tables without JOIN conditions may create Cartesian product",
				Severity: "high",
			})
			result.Score -= 20
		}
	}

	// Check for multiple JOINs without proper conditions
	joinCount := strings.Count(upperSQL, "JOIN") - strings.Count(upperSQL, "CROSS JOIN")
	onCount := strings.Count(upperSQL, " ON ")
	if joinCount > 0 && onCount < joinCount {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "join",
			Severity: "critical",
			Message:  "Missing ON conditions for some JOINs",
			Impact:   "Add proper JOIN conditions to avoid Cartesian products",
		})
		result.Score -= 20
	}
}

func (a *QueryAnalyzer) checkSubqueryOptimization(sql string, parsed *ParsedQuery, result *AnalysisResult) {
	if !parsed.HasSubquery {
		return
	}

	upperSQL := strings.ToUpper(sql)

	// Check for IN with subquery that could be JOIN
	if regexp.MustCompile(`(?i)WHERE\s+\S+\s+IN\s*\(\s*SELECT`).MatchString(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "subquery",
			Severity: "info",
			Message:  "IN subquery might be rewritten as JOIN for better performance",
			Impact:   "JOINs are often optimized better than subqueries",
		})
		result.Score -= 5
	}

	// Check for subquery in SELECT list (N+1 problem)
	if regexp.MustCompile(`(?i)SELECT\s+.*\(\s*SELECT`).MatchString(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "subquery",
			Severity: "warning",
			Message:  "Subquery in SELECT list executes for each row (N+1 problem)",
			Impact:   "Consider using JOIN or window functions instead",
		})
		result.Score -= 15
	}

	// Check for EXISTS that could be simplified
	if strings.Contains(upperSQL, "EXISTS") {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "subquery",
			Severity: "info",
			Message:  "EXISTS subquery - ensure it's more efficient than JOIN for your use case",
			Impact:   "EXISTS can be efficient for semi-joins but verify performance",
		})
	}
}

func (a *QueryAnalyzer) checkOrToIn(sql string, parsed *ParsedQuery, result *AnalysisResult) {
	// Check for multiple OR conditions on same column
	orPattern := regexp.MustCompile(`(?i)(\w+)\s*=\s*['"]?\w+['"]?\s+OR\s+\1\s*=`)
	if orPattern.MatchString(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "where",
			Severity: "info",
			Message:  "Multiple OR conditions on same column can be rewritten using IN",
			Impact:   "IN clause is often more readable and can be optimized better",
		})
		result.Score -= 5
	}
}

func (a *QueryAnalyzer) checkImplicitConversions(sql string, parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	if schema == nil {
		return
	}

	// Check for string literals compared to numeric columns
	if regexp.MustCompile(`(?i)WHERE\s+\w+\s*=\s*'[0-9]+['"]`).MatchString(sql) {
		result.Suggestions = append(result.Suggestions, Suggestion{
			Type:     "where",
			Severity: "warning",
			Message:  "Potential implicit type conversion - ensure data types match",
			Impact:   "Implicit conversions can prevent index usage",
		})
		result.Score -= 10
	}
}

func (a *QueryAnalyzer) checkCorrelatedSubqueries(sql string, parsed *ParsedQuery, result *AnalysisResult) {
	// Simple detection of correlated subqueries (references to outer query)
	subqueryPattern := regexp.MustCompile(`\(([^)]*SELECT[^)]*)\)`)
	if matches := subqueryPattern.FindAllStringSubmatch(sql, -1); len(matches) > 0 {
		for _, match := range matches {
			subquery := match[1]
			// Check if subquery references outer tables
			for _, table := range parsed.Tables {
				if strings.Contains(subquery, table+".") {
					result.Suggestions = append(result.Suggestions, Suggestion{
						Type:     "subquery",
						Severity: "warning",
						Message:  "Correlated subquery detected - executes once per row",
						Impact:   "Consider rewriting as JOIN or using window functions",
					})
					result.Score -= 15
					break
				}
			}
		}
	}
}

func (a *QueryAnalyzer) isLargeTable(tables []string, schema *Schema) bool {
	if schema == nil {
		return false
	}

	for _, tableName := range tables {
		if table, ok := schema.Tables[tableName]; ok {
			// Consider table large if it has more than 10,000 rows
			if table.RowCount > 10000 {
				return true
			}
		}
	}
	return false
}

func (a *QueryAnalyzer) isIndexed(tableName, columnName string, schema *Schema) bool {
	if schema == nil {
		return false
	}

	if table, ok := schema.Tables[tableName]; ok {
		// Check if column is part of any index
		for _, index := range table.Indexes {
			for _, col := range index.Columns {
				if col == columnName {
					return true
				}
			}
		}
	}
	return false
}

func (a *QueryAnalyzer) hasAggregateFunction(sql string) bool {
	aggregatePattern := regexp.MustCompile(`(?i)(COUNT|SUM|AVG|MAX|MIN|GROUP_CONCAT)\s*\(`)
	return aggregatePattern.MatchString(sql)
}

func (a *QueryAnalyzer) generateImprovedSelectColumns(sql string, parsed *ParsedQuery, schema *Schema) string {
	if schema == nil || len(parsed.Tables) == 0 {
		return ""
	}

	// Get the first table's columns as example
	tableName := parsed.Tables[0]
	if table, ok := schema.Tables[tableName]; ok {
		var columns []string
		for colName := range table.Columns {
			columns = append(columns, colName)
			if len(columns) >= 5 { // Limit to first 5 columns as example
				columns = append(columns, "...")
				break
			}
		}

		improved := regexp.MustCompile(`(?i)SELECT\s+\*`).ReplaceAllString(
			sql,
			fmt.Sprintf("SELECT %s", strings.Join(columns, ", ")),
		)
		return improved
	}

	return ""
}

func (a *QueryAnalyzer) calculateComplexity(parsed *ParsedQuery) string {
	score := 0

	// Add complexity points
	score += len(parsed.Tables) * 2        // Each table adds complexity
	score += len(parsed.JoinColumns) * 3   // JOINs are complex
	if parsed.HasSubquery {
		score += 5
	}
	if len(parsed.GroupBy) > 0 {
		score += 3
	}
	if parsed.HasDistinct {
		score += 2
	}

	switch {
	case score <= 5:
		return "simple"
	case score <= 15:
		return "moderate"
	default:
		return "complex"
	}
}

func (a *QueryAnalyzer) estimateCost(parsed *ParsedQuery, schema *Schema) int {
	cost := 10 // Base cost

	// Estimate based on operations
	cost += len(parsed.Tables) * 20

	if parsed.HasWildcard {
		cost += 10
	}

	if len(parsed.JoinColumns) > 0 {
		cost += len(parsed.JoinColumns) * 30
	}

	if parsed.HasSubquery {
		cost += 50
	}

	if len(parsed.OrderBy) > 0 {
		cost += 20
	}

	if parsed.HasDistinct {
		cost += 30
	}

	if len(parsed.GroupBy) > 0 {
		cost += 25
	}

	// Adjust for indexes
	if schema != nil {
		indexedWhereColumns := 0
		for _, col := range parsed.WhereColumns {
			for _, table := range parsed.Tables {
				if a.isIndexed(table, col, schema) {
					indexedWhereColumns++
					break
				}
			}
		}
		// Reduce cost for indexed columns
		cost -= indexedWhereColumns * 15
	}

	if cost < 10 {
		cost = 10
	}

	return cost
}

func (a *QueryAnalyzer) validateColumns(parsed *ParsedQuery, schema *Schema, result *AnalysisResult) {
	if schema == nil || len(parsed.Tables) == 0 {
		return
	}

	tableName := parsed.Tables[0]
	if table, ok := schema.Tables[tableName]; ok {
		for _, col := range parsed.Columns {
			if _, ok := table.Columns[col]; !ok {
				result.Warnings = append(result.Warnings, Warning{
					Message:  fmt.Sprintf("Column '%s' not found in table '%s'", col, tableName),
					Severity: "high",
				})
				result.Score -= 20
			}
		}
	}
}