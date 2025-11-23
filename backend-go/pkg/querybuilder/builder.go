package querybuilder

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
)

// ToSQL generates a parameterized SQL query from the QueryBuilder state
// Returns the SQL string, arguments slice, and any error
func (qb *QueryBuilder) ToSQL() (string, []interface{}, error) {
	if err := qb.Validate(); err != nil {
		return "", nil, fmt.Errorf("validation failed: %w", err)
	}

	// Start with PostgreSQL placeholder format ($1, $2, ...)
	// This can be adapted for other databases as needed
	builder := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// Build SELECT clause
	selectBuilder := builder.Select()

	for _, col := range qb.Columns {
		columnExpr := qb.buildColumnExpression(col)
		if col.Alias != "" {
			selectBuilder = selectBuilder.Column(fmt.Sprintf("%s AS %s", columnExpr, quoteIdentifier(col.Alias)))
		} else {
			selectBuilder = selectBuilder.Column(columnExpr)
		}
	}

	// Add FROM clause
	selectBuilder = selectBuilder.From(quoteIdentifier(qb.Table))

	// Add JOIN clauses
	for _, join := range qb.Joins {
		joinClause := qb.buildJoinClause(join)
		switch strings.ToUpper(join.Type) {
		case "INNER":
			selectBuilder = selectBuilder.InnerJoin(joinClause)
		case "LEFT":
			selectBuilder = selectBuilder.LeftJoin(joinClause)
		case "RIGHT":
			selectBuilder = selectBuilder.RightJoin(joinClause)
		case "FULL":
			// Squirrel doesn't have native FULL JOIN, use JoinClause
			selectBuilder = selectBuilder.JoinClause(fmt.Sprintf("FULL OUTER JOIN %s", joinClause))
		}
	}

	// Add WHERE clause
	if len(qb.Filters) > 0 {
		whereExpr, err := qb.buildWhereClause()
		if err != nil {
			return "", nil, fmt.Errorf("failed to build WHERE clause: %w", err)
		}
		selectBuilder = selectBuilder.Where(whereExpr)
	}

	// Add GROUP BY clause
	if len(qb.GroupBy) > 0 {
		for _, col := range qb.GroupBy {
			selectBuilder = selectBuilder.GroupBy(col)
		}
	}

	// Add ORDER BY clause
	for _, order := range qb.OrderBy {
		orderExpr := fmt.Sprintf("%s %s", order.Column, strings.ToUpper(order.Direction))
		selectBuilder = selectBuilder.OrderBy(orderExpr)
	}

	// Add LIMIT clause
	if qb.Limit != nil && *qb.Limit > 0 {
		selectBuilder = selectBuilder.Limit(uint64(*qb.Limit))
	}

	// Add OFFSET clause
	if qb.Offset != nil && *qb.Offset > 0 {
		selectBuilder = selectBuilder.Offset(uint64(*qb.Offset))
	}

	// Generate the SQL
	sql, args, err := selectBuilder.ToSql()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate SQL: %w", err)
	}

	return sql, args, nil
}

// buildColumnExpression creates the column expression with optional aggregation
func (qb *QueryBuilder) buildColumnExpression(col ColumnSelection) string {
	columnRef := fmt.Sprintf("%s.%s", quoteIdentifier(col.Table), quoteIdentifier(col.Column))

	if col.Aggregation == nil {
		return columnRef
	}

	aggFunc := strings.ToUpper(*col.Aggregation)
	switch aggFunc {
	case "COUNT_DISTINCT":
		return fmt.Sprintf("COUNT(DISTINCT %s)", columnRef)
	case "COUNT", "SUM", "AVG", "MIN", "MAX":
		return fmt.Sprintf("%s(%s)", aggFunc, columnRef)
	default:
		// Fallback: no aggregation
		return columnRef
	}
}

// buildJoinClause creates the JOIN clause string
func (qb *QueryBuilder) buildJoinClause(join JoinDefinition) string {
	tableName := quoteIdentifier(join.Table)
	if join.Alias != "" {
		tableName = fmt.Sprintf("%s AS %s", tableName, quoteIdentifier(join.Alias))
	}

	onClause := fmt.Sprintf("%s = %s", join.On.Left, join.On.Right)
	return fmt.Sprintf("%s ON %s", tableName, onClause)
}

// buildWhereClause constructs the WHERE clause with proper combinators
func (qb *QueryBuilder) buildWhereClause() (sq.Sqlizer, error) {
	if len(qb.Filters) == 0 {
		return nil, nil
	}

	// Group filters by combinator (AND/OR)
	// For simplicity, we'll build a single expression tree
	// More complex logic can handle nested AND/OR groups

	var conditions []sq.Sqlizer

	for i, filter := range qb.Filters {
		condition, err := qb.buildFilterCondition(filter)
		if err != nil {
			return nil, fmt.Errorf("failed to build filter %d: %w", i, err)
		}

		// For the first filter or filters with AND combinator, add to AND group
		// For OR combinator, we need to handle differently
		// Simplified: assume all filters are ANDed (or use combinator per filter)
		conditions = append(conditions, condition)
	}

	// Combine all conditions with AND by default
	// TODO: Enhance to support mixed AND/OR logic with proper grouping
	return sq.And(conditions), nil
}

// buildFilterCondition creates a single filter condition
func (qb *QueryBuilder) buildFilterCondition(filter FilterCondition) (sq.Sqlizer, error) {
	column := filter.Column
	operator := strings.ToUpper(filter.Operator)

	switch operator {
	case "=":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for = operator", filter.ID)
		}
		return sq.Eq{column: *filter.Value}, nil

	case "!=", "<>":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for != operator", filter.ID)
		}
		return sq.NotEq{column: *filter.Value}, nil

	case ">":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for > operator", filter.ID)
		}
		return sq.Gt{column: *filter.Value}, nil

	case "<":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for < operator", filter.ID)
		}
		return sq.Lt{column: *filter.Value}, nil

	case ">=":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for >= operator", filter.ID)
		}
		return sq.GtOrEq{column: *filter.Value}, nil

	case "<=":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for <= operator", filter.ID)
		}
		return sq.LtOrEq{column: *filter.Value}, nil

	case "LIKE":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for LIKE operator", filter.ID)
		}
		return sq.Like{column: fmt.Sprintf("%%%s%%", *filter.Value)}, nil

	case "NOT LIKE":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for NOT LIKE operator", filter.ID)
		}
		return sq.NotLike{column: fmt.Sprintf("%%%s%%", *filter.Value)}, nil

	case "IN":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for IN operator", filter.ID)
		}
		// Parse comma-separated values
		values := strings.Split(*filter.Value, ",")
		trimmedValues := make([]interface{}, len(values))
		for i, v := range values {
			trimmedValues[i] = strings.TrimSpace(v)
		}
		return sq.Eq{column: trimmedValues}, nil

	case "NOT IN":
		if filter.Value == nil {
			return nil, fmt.Errorf("filter %s: value required for NOT IN operator", filter.ID)
		}
		values := strings.Split(*filter.Value, ",")
		trimmedValues := make([]interface{}, len(values))
		for i, v := range values {
			trimmedValues[i] = strings.TrimSpace(v)
		}
		return sq.NotEq{column: trimmedValues}, nil

	case "IS NULL":
		return sq.Eq{column: nil}, nil

	case "IS NOT NULL":
		return sq.NotEq{column: nil}, nil

	case "BETWEEN":
		if filter.Value == nil || filter.ValueTo == nil {
			return nil, fmt.Errorf("filter %s: two values required for BETWEEN operator", filter.ID)
		}
		// Use raw SQL for BETWEEN as squirrel doesn't have a native BETWEEN type
		return sq.Expr(fmt.Sprintf("%s BETWEEN ? AND ?", column), *filter.Value, *filter.ValueTo), nil

	default:
		return nil, fmt.Errorf("filter %s: unsupported operator %s", filter.ID, operator)
	}
}

// quoteIdentifier wraps an identifier in double quotes for PostgreSQL
// Adjust this for other databases (e.g., backticks for MySQL)
func quoteIdentifier(identifier string) string {
	// Don't quote if already quoted or if it contains a dot (table.column reference)
	if strings.Contains(identifier, ".") || strings.HasPrefix(identifier, "\"") {
		return identifier
	}
	return fmt.Sprintf(`"%s"`, identifier)
}

// Validate checks if the QueryBuilder state is valid
func (qb *QueryBuilder) Validate() error {
	errors := qb.GetValidationErrors()
	if !errors.Valid {
		var messages []string
		for _, err := range errors.Errors {
			messages = append(messages, err.Message)
		}
		return fmt.Errorf("validation errors: %s", strings.Join(messages, "; "))
	}
	return nil
}

// GetValidationErrors returns detailed validation errors
func (qb *QueryBuilder) GetValidationErrors() ValidationResult {
	result := ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []ValidationError{},
	}

	// Must have table selected
	if qb.Table == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "table",
			Message:  "Table must be selected",
			Severity: "error",
		})
	}

	// Must have at least one column
	if len(qb.Columns) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "columns",
			Message:  "At least one column must be selected",
			Severity: "error",
		})
	}

	// All columns must have a column name
	for i, col := range qb.Columns {
		if col.Column == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("columns[%d]", i),
				Message:  fmt.Sprintf("Column %d is missing column name", i+1),
				Severity: "error",
			})
		}
	}

	// Check GROUP BY requirements when using aggregations
	hasAggregations := false
	nonAggregatedColumns := []string{}

	for _, col := range qb.Columns {
		if col.Aggregation != nil {
			hasAggregations = true
		} else {
			nonAggregatedColumns = append(nonAggregatedColumns, fmt.Sprintf("%s.%s", col.Table, col.Column))
		}
	}

	if hasAggregations && len(nonAggregatedColumns) > 0 && len(qb.GroupBy) == 0 {
		result.Errors = append(result.Errors, ValidationError{
			Field:    "groupBy",
			Message:  "When using aggregations, non-aggregated columns must be in GROUP BY",
			Severity: "error",
		})
	}

	// Validate filters
	for i, filter := range qb.Filters {
		if filter.Column == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("filters[%d]", i),
				Message:  fmt.Sprintf("Filter %d is missing column", i+1),
				Severity: "error",
			})
		}

		needsValue := !strings.Contains(strings.ToUpper(filter.Operator), "NULL")
		if needsValue && filter.Value == nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("filters[%d]", i),
				Message:  fmt.Sprintf("Filter %d is missing value for operator %s", i+1, filter.Operator),
				Severity: "error",
			})
		}

		if strings.ToUpper(filter.Operator) == "BETWEEN" && filter.ValueTo == nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("filters[%d]", i),
				Message:  fmt.Sprintf("Filter %d is missing second value for BETWEEN operator", i+1),
				Severity: "error",
			})
		}
	}

	// Validate joins
	for i, join := range qb.Joins {
		if join.Table == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("joins[%d]", i),
				Message:  fmt.Sprintf("Join %d is missing table", i+1),
				Severity: "error",
			})
		}
		if join.On.Left == "" || join.On.Right == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("joins[%d]", i),
				Message:  fmt.Sprintf("Join %d is missing ON condition", i+1),
				Severity: "error",
			})
		}
	}

	// Validate ORDER BY
	for i, order := range qb.OrderBy {
		if order.Column == "" {
			result.Errors = append(result.Errors, ValidationError{
				Field:    fmt.Sprintf("orderBy[%d]", i),
				Message:  fmt.Sprintf("Sort %d is missing column", i+1),
				Severity: "error",
			})
		}
	}

	// Add warnings
	if qb.Limit == nil || *qb.Limit > 1000 {
		result.Warnings = append(result.Warnings, ValidationError{
			Field:    "limit",
			Message:  "Consider adding a LIMIT to improve query performance",
			Severity: "warning",
		})
	}

	result.Valid = len(result.Errors) == 0

	return result
}
