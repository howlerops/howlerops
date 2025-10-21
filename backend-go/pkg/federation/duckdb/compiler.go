package duckdb

import (
	"fmt"
	"strings"
)

// ViewDefinition represents a synthetic view definition
type ViewDefinition struct {
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	Description       string             `json:"description"`
	Version           string             `json:"version"`
	Columns           []ColumnDefinition `json:"columns"`
	IR                QueryIR            `json:"ir"`
	Sources           []SourceDefinition `json:"sources"`
	CompiledDuckDBSQL string             `json:"compiledDuckDBSQL"`
	Options           ViewOptions        `json:"options"`
}

// ColumnDefinition represents a column in a synthetic view
type ColumnDefinition struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// SourceDefinition represents a source table for a synthetic view
type SourceDefinition struct {
	ConnectionIDOrName string `json:"connectionIdOrName"`
	Schema             string `json:"schema"`
	Table              string `json:"table"`
}

// ViewOptions contains configuration options for a synthetic view
type ViewOptions struct {
	RowLimitDefault int  `json:"rowLimitDefault"`
	MaterializeTemp bool `json:"materializeTemp"`
}

// QueryIR represents the intermediate representation of a query
type QueryIR struct {
	From    TableRef     `json:"from"`
	Joins   []Join       `json:"joins,omitempty"`
	Select  []SelectItem `json:"select"`
	Where   *Expression  `json:"where,omitempty"`
	OrderBy []OrderBy    `json:"orderBy,omitempty"`
	Limit   *int         `json:"limit,omitempty"`
	Offset  *int         `json:"offset,omitempty"`
}

// TableRef represents a table reference in the query
type TableRef struct {
	Schema     string `json:"schema"`
	Table      string `json:"table"`
	Alias      string `json:"alias,omitempty"`
	Connection string `json:"connection,omitempty"`
}

// Join represents a join in the query
type Join struct {
	Type  string     `json:"type"` // inner, left, right, full
	Table TableRef   `json:"table"`
	On    Expression `json:"on"`
}

// SelectItem represents a select item
type SelectItem struct {
	Column    string `json:"column"`
	Alias     string `json:"alias,omitempty"`
	Aggregate string `json:"aggregate,omitempty"`
}

// Expression represents a SQL expression
type Expression struct {
	Type       string       `json:"type"` // predicate, group, exists
	Column     string       `json:"column,omitempty"`
	Operator   string       `json:"operator,omitempty"`
	Value      interface{}  `json:"value,omitempty"`
	Not        bool         `json:"not,omitempty"`
	GroupOp    string       `json:"groupOp,omitempty"` // AND, OR for groups
	Conditions []Expression `json:"conditions,omitempty"`
	Subquery   *QueryIR     `json:"subquery,omitempty"`
}

// OrderBy represents an ORDER BY clause
type OrderBy struct {
	Column    string `json:"column"`
	Direction string `json:"direction"` // asc, desc
}

// Compiler compiles ViewDefinition to DuckDB SQL
type Compiler struct {
	connectionManager interface{}
}

// NewCompiler creates a new DuckDB SQL compiler
func NewCompiler(connectionManager interface{}) *Compiler {
	return &Compiler{
		connectionManager: connectionManager,
	}
}

// Compile compiles a ViewDefinition to DuckDB SQL using scanner functions
func (c *Compiler) Compile(viewDef *ViewDefinition) (string, error) {
	if viewDef.CompiledDuckDBSQL != "" {
		return viewDef.CompiledDuckDBSQL, nil
	}

	// Generate SELECT clause
	selectClause, err := c.compileSelect(viewDef.IR.Select)
	if err != nil {
		return "", fmt.Errorf("failed to compile SELECT: %w", err)
	}

	// Generate FROM clause with scanner function
	fromClause, err := c.compileFrom(viewDef.IR.From, viewDef.Sources)
	if err != nil {
		return "", fmt.Errorf("failed to compile FROM: %w", err)
	}

	// Generate JOIN clauses
	joinClauses, err := c.compileJoins(viewDef.IR.Joins, viewDef.Sources)
	if err != nil {
		return "", fmt.Errorf("failed to compile JOINs: %w", err)
	}

	// Generate WHERE clause
	whereClause, err := c.compileWhere(viewDef.IR.Where)
	if err != nil {
		return "", fmt.Errorf("failed to compile WHERE: %w", err)
	}

	// Generate ORDER BY clause
	orderByClause, err := c.compileOrderBy(viewDef.IR.OrderBy)
	if err != nil {
		return "", fmt.Errorf("failed to compile ORDER BY: %w", err)
	}

	// Generate LIMIT clause
	limitClause, err := c.compileLimit(viewDef.IR.Limit, viewDef.IR.Offset)
	if err != nil {
		return "", fmt.Errorf("failed to compile LIMIT: %w", err)
	}

	// Combine all clauses
	parts := []string{selectClause, fromClause}
	parts = append(parts, joinClauses...)
	if whereClause != "" {
		parts = append(parts, whereClause)
	}
	if orderByClause != "" {
		parts = append(parts, orderByClause)
	}
	if limitClause != "" {
		parts = append(parts, limitClause)
	}

	return strings.Join(parts, "\n"), nil
}

// compileSelect compiles the SELECT clause
func (c *Compiler) compileSelect(selectItems []SelectItem) (string, error) {
	if len(selectItems) == 0 {
		return "SELECT *", nil
	}

	var items []string
	for _, item := range selectItems {
		column := c.quoteIdentifier(item.Column)

		if item.Aggregate != "" {
			column = fmt.Sprintf("%s(%s)", strings.ToUpper(item.Aggregate), column)
		}

		if item.Alias != "" {
			column += " AS " + c.quoteIdentifier(item.Alias)
		}

		items = append(items, column)
	}

	return "SELECT " + strings.Join(items, ", "), nil
}

// compileFrom compiles the FROM clause using scanner functions
func (c *Compiler) compileFrom(from TableRef, sources []SourceDefinition) (string, error) {
	// Find the source definition for this table
	var source *SourceDefinition
	for _, s := range sources {
		if s.Schema == from.Schema && s.Table == from.Table {
			source = &s
			break
		}
	}

	if source == nil {
		return "", fmt.Errorf("source not found for table %s.%s", from.Schema, from.Table)
	}

	// Get connection config to build DSN
	// For now, return a placeholder config - this would be implemented with actual connection manager
	config := &ScannerConfig{
		Type:     "postgres",
		Host:     "localhost",
		Port:     5432,
		Database: "database",
		Username: "user",
		Password: "password",
		SSLMode:  "prefer",
		Options:  make(map[string]string),
	}

	// Build scanner function call based on connection type
	scannerCall, err := c.buildScannerCall(config, source)
	if err != nil {
		return "", fmt.Errorf("failed to build scanner call: %w", err)
	}

	alias := from.Alias
	if alias == "" {
		alias = from.Table
	}

	return fmt.Sprintf("FROM %s AS %s", scannerCall, c.quoteIdentifier(alias)), nil
}

// buildScannerCall builds the appropriate scanner function call
func (c *Compiler) buildScannerCall(config interface{}, source *SourceDefinition) (string, error) {
	// Convert config to ScannerConfig
	scannerConfig, err := c.convertToScannerConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to convert config: %w", err)
	}

	// Build scanner call using ScannerBuilder
	builder := NewScannerBuilder()
	return builder.BuildScannerCall(scannerConfig, source.Schema, source.Table)
}

// convertToScannerConfig converts a connection config to ScannerConfig
func (c *Compiler) convertToScannerConfig(config interface{}) (*ScannerConfig, error) {
	// This is a simplified version - in practice, you'd extract the actual config fields
	// based on the connection type and build the appropriate ScannerConfig

	// For now, return a basic config that would be populated from the actual connection
	return &ScannerConfig{
		Type:     "postgres", // This would be determined from the actual connection
		Host:     "localhost",
		Port:     5432,
		Database: "database",
		Username: "user",
		Password: "password",
		SSLMode:  "prefer",
		Options:  make(map[string]string),
	}, nil
}

// compileJoins compiles JOIN clauses
func (c *Compiler) compileJoins(joins []Join, sources []SourceDefinition) ([]string, error) {
	var clauses []string

	for _, join := range joins {
		// Find source for join table
		var source *SourceDefinition
		for _, s := range sources {
			if s.Schema == join.Table.Schema && s.Table == join.Table.Table {
				source = &s
				break
			}
		}

		if source == nil {
			return nil, fmt.Errorf("source not found for join table %s.%s", join.Table.Schema, join.Table.Table)
		}

		// Build scanner call for join table
		// For now, return a placeholder config - this would be implemented with actual connection manager
		config := &ScannerConfig{
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "database",
			Username: "user",
			Password: "password",
			SSLMode:  "prefer",
			Options:  make(map[string]string),
		}

		scannerCall, err := c.buildScannerCall(config, source)
		if err != nil {
			return nil, fmt.Errorf("failed to build scanner call for join: %w", err)
		}

		// Compile ON condition
		onClause, err := c.compileExpression(join.On)
		if err != nil {
			return nil, fmt.Errorf("failed to compile join condition: %w", err)
		}

		alias := join.Table.Alias
		if alias == "" {
			alias = join.Table.Table
		}

		clause := fmt.Sprintf("%s JOIN %s AS %s ON %s",
			strings.ToUpper(join.Type), scannerCall, c.quoteIdentifier(alias), onClause)
		clauses = append(clauses, clause)
	}

	return clauses, nil
}

// compileWhere compiles the WHERE clause
func (c *Compiler) compileWhere(where *Expression) (string, error) {
	if where == nil {
		return "", nil
	}

	condition, err := c.compileExpression(*where)
	if err != nil {
		return "", err
	}

	return "WHERE " + condition, nil
}

// compileOrderBy compiles the ORDER BY clause
func (c *Compiler) compileOrderBy(orderBy []OrderBy) (string, error) {
	if len(orderBy) == 0 {
		return "", nil
	}

	var items []string
	for _, item := range orderBy {
		column := c.quoteIdentifier(item.Column)
		direction := strings.ToUpper(item.Direction)
		items = append(items, fmt.Sprintf("%s %s", column, direction))
	}

	return "ORDER BY " + strings.Join(items, ", "), nil
}

// compileLimit compiles the LIMIT clause
func (c *Compiler) compileLimit(limit *int, offset *int) (string, error) {
	if limit == nil {
		return "", nil
	}

	if offset != nil && *offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", *limit, *offset), nil
	}

	return fmt.Sprintf("LIMIT %d", *limit), nil
}

// compileExpression compiles a SQL expression
func (c *Compiler) compileExpression(expr Expression) (string, error) {
	switch expr.Type {
	case "predicate":
		return c.compilePredicate(expr)
	case "group":
		return c.compileGroup(expr)
	case "exists":
		return c.compileExists(expr)
	default:
		return "", fmt.Errorf("unknown expression type: %s", expr.Type)
	}
}

// compilePredicate compiles a predicate expression
func (c *Compiler) compilePredicate(expr Expression) (string, error) {
	column := c.quoteIdentifier(expr.Column)
	operator := c.mapOperator(expr.Operator)
	value := c.formatValue(expr.Value, expr.Operator)

	condition := fmt.Sprintf("%s %s %s", column, operator, value)

	if expr.Not {
		condition = "NOT (" + condition + ")"
	}

	return condition, nil
}

// compileGroup compiles a group expression
func (c *Compiler) compileGroup(expr Expression) (string, error) {
	var conditions []string
	for _, condition := range expr.Conditions {
		compiled, err := c.compileExpression(condition)
		if err != nil {
			return "", err
		}
		conditions = append(conditions, compiled)
	}

	combined := "(" + strings.Join(conditions, " "+expr.GroupOp+" ") + ")"

	if expr.Not {
		combined = "NOT " + combined
	}

	return combined, nil
}

// compileExists compiles an EXISTS expression
func (c *Compiler) compileExists(expr Expression) (string, error) {
	if expr.Subquery == nil {
		return "", fmt.Errorf("EXISTS expression missing subquery")
	}

	// This would need to recursively compile the subquery
	// For now, return a placeholder
	subquery := "SELECT 1" // Placeholder

	result := "EXISTS (" + subquery + ")"
	if expr.Not {
		result = "NOT " + result
	}

	return result, nil
}

// mapOperator maps filter operators to SQL operators
func (c *Compiler) mapOperator(operator string) string {
	switch operator {
	case "equals":
		return "="
	case "not_equals":
		return "!="
	case "greater_than":
		return ">"
	case "greater_than_or_equals":
		return ">="
	case "less_than":
		return "<"
	case "less_than_or_equals":
		return "<="
	case "contains":
		return "ILIKE"
	case "not_contains":
		return "NOT ILIKE"
	case "starts_with":
		return "ILIKE"
	case "ends_with":
		return "ILIKE"
	case "in":
		return "IN"
	case "not_in":
		return "NOT IN"
	case "is_null":
		return "IS NULL"
	case "is_not_null":
		return "IS NOT NULL"
	case "between":
		return "BETWEEN"
	default:
		return operator
	}
}

// formatValue formats a value for SQL
func (c *Compiler) formatValue(value interface{}, operator string) string {
	if value == nil {
		return "NULL"
	}

	switch v := value.(type) {
	case string:
		// Handle special operators
		if operator == "contains" || operator == "not_contains" {
			return fmt.Sprintf("'%%%s%%'", strings.ReplaceAll(v, "'", "''"))
		}
		if operator == "starts_with" {
			return fmt.Sprintf("'%s%%'", strings.ReplaceAll(v, "'", "''"))
		}
		if operator == "ends_with" {
			return fmt.Sprintf("'%%%s'", strings.ReplaceAll(v, "'", "''"))
		}
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case []interface{}:
		if operator == "in" || operator == "not_in" {
			var values []string
			for _, item := range v {
				values = append(values, c.formatValue(item, ""))
			}
			return "(" + strings.Join(values, ", ") + ")"
		}
		if operator == "between" && len(v) == 2 {
			return fmt.Sprintf("%s AND %s", c.formatValue(v[0], ""), c.formatValue(v[1], ""))
		}
		return fmt.Sprintf("'%v'", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// quoteIdentifier quotes a SQL identifier
func (c *Compiler) quoteIdentifier(identifier string) string {
	return fmt.Sprintf("\"%s\"", strings.ReplaceAll(identifier, "\"", "\"\""))
}
