package querybuilder

// QueryBuilder represents the visual query builder state
type QueryBuilder struct {
	DataSource string            `json:"dataSource"` // connectionId
	Table      string            `json:"table"`
	Columns    []ColumnSelection `json:"columns"`
	Joins      []JoinDefinition  `json:"joins"`
	Filters    []FilterCondition `json:"filters"`
	GroupBy    []string          `json:"groupBy"`
	OrderBy    []OrderByClause   `json:"orderBy"`
	Limit      *int              `json:"limit,omitempty"`
	Offset     *int              `json:"offset,omitempty"`
}

// ColumnSelection represents a column to select with optional aggregation
type ColumnSelection struct {
	Table       string  `json:"table"`
	Column      string  `json:"column"`
	Alias       string  `json:"alias,omitempty"`
	Aggregation *string `json:"aggregation,omitempty"` // count, sum, avg, min, max, count_distinct
}

// JoinDefinition represents a table join
type JoinDefinition struct {
	Type  string `json:"type"` // INNER, LEFT, RIGHT, FULL
	Table string `json:"table"`
	Alias string `json:"alias,omitempty"`
	On    JoinOn `json:"on"`
}

// JoinOn represents the ON condition for a join
type JoinOn struct {
	Left  string `json:"left"`  // format: "table.column"
	Right string `json:"right"` // format: "table.column"
}

// FilterCondition represents a WHERE condition
type FilterCondition struct {
	ID         string  `json:"id"`
	Column     string  `json:"column"`   // format: "table.column"
	Operator   string  `json:"operator"` // =, !=, >, <, >=, <=, LIKE, NOT LIKE, IN, NOT IN, IS NULL, IS NOT NULL, BETWEEN
	Value      *string `json:"value,omitempty"`
	ValueTo    *string `json:"valueTo,omitempty"`    // for BETWEEN
	Combinator *string `json:"combinator,omitempty"` // AND, OR (only for filters after first)
}

// OrderByClause represents an ORDER BY clause
type OrderByClause struct {
	Column    string `json:"column"`    // format: "table.column"
	Direction string `json:"direction"` // ASC, DESC
}

// ValidationError represents a query validation error
type ValidationError struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // error, warning
}

// ValidationResult holds query validation results
type ValidationResult struct {
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// GeneratedSQL represents the output of SQL generation
type GeneratedSQL struct {
	SQL        string        `json:"sql"`
	Args       []interface{} `json:"args"`
	Parameters int           `json:"parameters"` // Count of parameterized values
}
