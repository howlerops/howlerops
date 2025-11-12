package autocomplete

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

// AutocompleteService provides context-aware SQL autocomplete suggestions
type AutocompleteService struct {
	schema *Schema
	logger *logrus.Logger
}

// Schema represents database schema for autocomplete
type Schema struct {
	Tables    map[string]*Table    `json:"tables"`
	Functions map[string]*Function `json:"functions"`
}

// Table represents a database table
type Table struct {
	Name    string            `json:"name"`
	Columns map[string]string `json:"columns"` // column_name -> data_type
	Indexes []string          `json:"indexes"`
}

// Function represents a SQL function
type Function struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Signature   string `json:"signature"`
}

// Suggestion represents an autocomplete suggestion
type Suggestion struct {
	Text        string `json:"text"`
	Type        string `json:"type"` // 'table', 'column', 'keyword', 'function', 'snippet'
	Description string `json:"description,omitempty"`
	InsertText  string `json:"insert_text,omitempty"` // What to actually insert (for snippets)
	Detail      string `json:"detail,omitempty"`      // Additional info like data type
}

// Context represents the current cursor context
type Context struct {
	Type         string   // 'after_select', 'after_from', 'after_where', 'after_join', etc.
	CurrentTable string   // Current table context
	Tables       []string // Tables mentioned in query
	Position     int      // Cursor position
}

// NewAutocompleteService creates a new autocomplete service
func NewAutocompleteService(schema *Schema, logger *logrus.Logger) *AutocompleteService {
	if logger == nil {
		logger = logrus.New()
	}

	// Initialize built-in SQL functions if not provided
	if schema.Functions == nil {
		schema.Functions = getBuiltInFunctions()
	}

	return &AutocompleteService{
		schema: schema,
		logger: logger,
	}
}

// GetSuggestions returns autocomplete suggestions based on the cursor position
func (s *AutocompleteService) GetSuggestions(sql string, cursorPos int) ([]Suggestion, error) {
	// Analyze context at cursor position
	context := s.analyzeContext(sql, cursorPos)

	// Get suggestions based on context
	suggestions := s.getSuggestionsForContext(context, sql)

	// Sort suggestions by relevance
	s.sortSuggestions(suggestions, context)

	return suggestions, nil
}

func (s *AutocompleteService) analyzeContext(sql string, cursorPos int) *Context {
	if cursorPos > len(sql) {
		cursorPos = len(sql)
	}

	sqlBeforeCursor := sql[:cursorPos]
	upperSQL := strings.ToUpper(sqlBeforeCursor)

	context := &Context{
		Position: cursorPos,
		Tables:   s.extractTables(sql),
	}

	// Determine context type based on the SQL before cursor
	lastKeyword := s.getLastKeyword(upperSQL)

	switch lastKeyword {
	case "SELECT":
		context.Type = "after_select"
	case "FROM":
		context.Type = "after_from"
	case "WHERE", "AND", "OR":
		context.Type = "after_where"
	case "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL":
		context.Type = "after_join"
	case "ON":
		context.Type = "after_on"
	case "GROUP":
		if strings.HasSuffix(upperSQL, "GROUP BY") || strings.HasSuffix(upperSQL, "GROUP BY ") {
			context.Type = "after_group_by"
		}
	case "ORDER":
		if strings.HasSuffix(upperSQL, "ORDER BY") || strings.HasSuffix(upperSQL, "ORDER BY ") {
			context.Type = "after_order_by"
		}
	case "INSERT":
		if strings.Contains(upperSQL, "INTO") {
			context.Type = "after_insert_into"
		}
	case "UPDATE":
		context.Type = "after_update"
	case "SET":
		context.Type = "after_set"
	case "VALUES":
		context.Type = "after_values"
	case "AS":
		context.Type = "after_as"
	default:
		// Check for table.column pattern
		if s.isTypingTableColumn(sqlBeforeCursor) {
			context.Type = "table_column"
			context.CurrentTable = s.extractCurrentTable(sqlBeforeCursor)
		} else if s.isStartOfStatement(upperSQL) {
			context.Type = "start_of_statement"
		} else if s.isAfterComma(sqlBeforeCursor) {
			// Determine what kind of comma-separated list we're in
			if strings.Contains(upperSQL, "SELECT") && !strings.Contains(upperSQL, "FROM") {
				context.Type = "select_list"
			} else if strings.Contains(upperSQL, "FROM") && !strings.Contains(upperSQL, "WHERE") {
				context.Type = "table_list"
			}
		} else {
			context.Type = "general"
		}
	}

	return context
}

func (s *AutocompleteService) getSuggestionsForContext(context *Context, sql string) []Suggestion {
	suggestions := []Suggestion{}

	switch context.Type {
	case "start_of_statement":
		suggestions = s.getStatementKeywords()

	case "after_select":
		suggestions = append(suggestions, s.getSelectSuggestions()...)
		if s.schema != nil {
			suggestions = append(suggestions, s.getColumnSuggestions("")...)
			suggestions = append(suggestions, s.getFunctionSuggestions()...)
		}

	case "after_from":
		suggestions = s.getTableSuggestions()

	case "after_where", "after_on":
		if s.schema != nil && len(context.Tables) > 0 {
			suggestions = s.getColumnSuggestionsForTables(context.Tables)
		}
		suggestions = append(suggestions, s.getOperatorKeywords()...)

	case "after_join":
		suggestions = s.getTableSuggestions()

	case "after_group_by", "after_order_by":
		if s.schema != nil && len(context.Tables) > 0 {
			suggestions = s.getColumnSuggestionsForTables(context.Tables)
		}

	case "after_insert_into":
		suggestions = s.getTableSuggestions()

	case "after_update":
		suggestions = s.getTableSuggestions()

	case "after_set":
		if s.schema != nil && context.CurrentTable != "" {
			suggestions = s.getColumnSuggestions(context.CurrentTable)
		}

	case "table_column":
		if context.CurrentTable != "" {
			suggestions = s.getColumnSuggestions(context.CurrentTable)
		}

	case "select_list":
		if s.schema != nil && len(context.Tables) > 0 {
			suggestions = s.getColumnSuggestionsForTables(context.Tables)
		}
		suggestions = append(suggestions, s.getFunctionSuggestions()...)

	case "table_list":
		suggestions = s.getTableSuggestions()

	default:
		// Provide general suggestions
		suggestions = s.getGeneralSuggestions(context)
	}

	// Add snippets for common patterns
	suggestions = append(suggestions, s.getSnippets(context)...)

	return suggestions
}

func (s *AutocompleteService) getStatementKeywords() []Suggestion {
	return []Suggestion{
		{Text: "SELECT", Type: "keyword", Description: "Retrieve data from tables"},
		{Text: "INSERT", Type: "keyword", Description: "Add new rows to a table"},
		{Text: "UPDATE", Type: "keyword", Description: "Modify existing rows"},
		{Text: "DELETE", Type: "keyword", Description: "Remove rows from a table"},
		{Text: "CREATE", Type: "keyword", Description: "Create new database objects"},
		{Text: "ALTER", Type: "keyword", Description: "Modify database objects"},
		{Text: "DROP", Type: "keyword", Description: "Delete database objects"},
		{Text: "TRUNCATE", Type: "keyword", Description: "Remove all rows from a table"},
	}
}

func (s *AutocompleteService) getSelectSuggestions() []Suggestion {
	return []Suggestion{
		{Text: "*", Type: "keyword", Description: "Select all columns"},
		{Text: "DISTINCT", Type: "keyword", Description: "Select unique rows"},
		{Text: "COUNT(*)", Type: "function", Description: "Count all rows"},
		{Text: "COUNT(DISTINCT", Type: "snippet", InsertText: "COUNT(DISTINCT ${1:column})", Description: "Count unique values"},
	}
}

func (s *AutocompleteService) getTableSuggestions() []Suggestion {
	suggestions := []Suggestion{}

	if s.schema == nil {
		return suggestions
	}

	for tableName, table := range s.schema.Tables {
		colCount := len(table.Columns)
		description := fmt.Sprintf("%d columns", colCount)

		suggestions = append(suggestions, Suggestion{
			Text:        tableName,
			Type:        "table",
			Description: description,
		})
	}

	return suggestions
}

func (s *AutocompleteService) getColumnSuggestions(tableName string) []Suggestion {
	suggestions := []Suggestion{}

	if s.schema == nil {
		return suggestions
	}

	if tableName == "" {
		// Get columns from all tables
		for tName, table := range s.schema.Tables {
			for colName, colType := range table.Columns {
				suggestions = append(suggestions, Suggestion{
					Text:        colName,
					Type:        "column",
					Description: fmt.Sprintf("%s.%s", tName, colName),
					Detail:      colType,
				})
			}
		}
	} else {
		// Get columns from specific table
		if table, ok := s.schema.Tables[tableName]; ok {
			for colName, colType := range table.Columns {
				suggestions = append(suggestions, Suggestion{
					Text:        colName,
					Type:        "column",
					Description: fmt.Sprintf("%s.%s", tableName, colName),
					Detail:      colType,
				})
			}
		}
	}

	return suggestions
}

func (s *AutocompleteService) getColumnSuggestionsForTables(tables []string) []Suggestion {
	suggestions := []Suggestion{}
	seen := make(map[string]bool)

	for _, tableName := range tables {
		for _, suggestion := range s.getColumnSuggestions(tableName) {
			// Avoid duplicates
			if !seen[suggestion.Text] {
				suggestions = append(suggestions, suggestion)
				seen[suggestion.Text] = true
			}
		}
	}

	return suggestions
}

func (s *AutocompleteService) getFunctionSuggestions() []Suggestion {
	suggestions := []Suggestion{}

	if s.schema == nil || s.schema.Functions == nil {
		return s.getDefaultFunctionSuggestions()
	}

	for _, function := range s.schema.Functions {
		suggestions = append(suggestions, Suggestion{
			Text:        function.Name,
			Type:        "function",
			Description: function.Description,
			InsertText:  function.Signature,
		})
	}

	return suggestions
}

func (s *AutocompleteService) getDefaultFunctionSuggestions() []Suggestion {
	return []Suggestion{
		{Text: "COUNT", Type: "function", InsertText: "COUNT(${1:column})", Description: "Count rows"},
		{Text: "SUM", Type: "function", InsertText: "SUM(${1:column})", Description: "Sum values"},
		{Text: "AVG", Type: "function", InsertText: "AVG(${1:column})", Description: "Average value"},
		{Text: "MAX", Type: "function", InsertText: "MAX(${1:column})", Description: "Maximum value"},
		{Text: "MIN", Type: "function", InsertText: "MIN(${1:column})", Description: "Minimum value"},
		{Text: "LENGTH", Type: "function", InsertText: "LENGTH(${1:string})", Description: "String length"},
		{Text: "UPPER", Type: "function", InsertText: "UPPER(${1:string})", Description: "Convert to uppercase"},
		{Text: "LOWER", Type: "function", InsertText: "LOWER(${1:string})", Description: "Convert to lowercase"},
		{Text: "SUBSTR", Type: "function", InsertText: "SUBSTR(${1:string}, ${2:start}, ${3:length})", Description: "Extract substring"},
		{Text: "CONCAT", Type: "function", InsertText: "CONCAT(${1:str1}, ${2:str2})", Description: "Concatenate strings"},
		{Text: "COALESCE", Type: "function", InsertText: "COALESCE(${1:value}, ${2:default})", Description: "Return first non-null value"},
		{Text: "NOW", Type: "function", InsertText: "NOW()", Description: "Current timestamp"},
		{Text: "DATE", Type: "function", InsertText: "DATE(${1:datetime})", Description: "Extract date part"},
	}
}

func (s *AutocompleteService) getOperatorKeywords() []Suggestion {
	return []Suggestion{
		{Text: "=", Type: "keyword", Description: "Equal to"},
		{Text: "!=", Type: "keyword", Description: "Not equal to"},
		{Text: "<>", Type: "keyword", Description: "Not equal to"},
		{Text: ">", Type: "keyword", Description: "Greater than"},
		{Text: "<", Type: "keyword", Description: "Less than"},
		{Text: ">=", Type: "keyword", Description: "Greater than or equal"},
		{Text: "<=", Type: "keyword", Description: "Less than or equal"},
		{Text: "LIKE", Type: "keyword", Description: "Pattern matching"},
		{Text: "NOT LIKE", Type: "keyword", Description: "Negative pattern matching"},
		{Text: "IN", Type: "keyword", Description: "Value in list"},
		{Text: "NOT IN", Type: "keyword", Description: "Value not in list"},
		{Text: "BETWEEN", Type: "keyword", Description: "Value in range"},
		{Text: "IS NULL", Type: "keyword", Description: "Check for NULL"},
		{Text: "IS NOT NULL", Type: "keyword", Description: "Check for non-NULL"},
		{Text: "AND", Type: "keyword", Description: "Logical AND"},
		{Text: "OR", Type: "keyword", Description: "Logical OR"},
		{Text: "NOT", Type: "keyword", Description: "Logical NOT"},
	}
}

func (s *AutocompleteService) getSnippets(context *Context) []Suggestion {
	snippets := []Suggestion{}

	switch context.Type {
	case "start_of_statement":
		snippets = append(snippets, []Suggestion{
			{
				Text:        "SELECT with WHERE",
				Type:        "snippet",
				InsertText:  "SELECT ${1:columns}\nFROM ${2:table}\nWHERE ${3:condition}",
				Description: "Basic SELECT query with WHERE clause",
			},
			{
				Text:        "INSERT statement",
				Type:        "snippet",
				InsertText:  "INSERT INTO ${1:table} (${2:columns})\nVALUES (${3:values})",
				Description: "Insert new row",
			},
			{
				Text:        "UPDATE with WHERE",
				Type:        "snippet",
				InsertText:  "UPDATE ${1:table}\nSET ${2:column} = ${3:value}\nWHERE ${4:condition}",
				Description: "Update existing rows",
			},
			{
				Text:        "CREATE TABLE",
				Type:        "snippet",
				InsertText:  "CREATE TABLE ${1:table_name} (\n    ${2:id} INTEGER PRIMARY KEY,\n    ${3:column} ${4:datatype}\n)",
				Description: "Create new table",
			},
		}...)

	case "after_select":
		snippets = append(snippets, []Suggestion{
			{
				Text:        "CASE WHEN",
				Type:        "snippet",
				InsertText:  "CASE\n    WHEN ${1:condition} THEN ${2:result}\n    ELSE ${3:default}\nEND",
				Description: "Conditional expression",
			},
		}...)

	case "after_where":
		snippets = append(snippets, []Suggestion{
			{
				Text:        "IN list",
				Type:        "snippet",
				InsertText:  "${1:column} IN (${2:value1}, ${3:value2})",
				Description: "Check if value is in list",
			},
			{
				Text:        "BETWEEN range",
				Type:        "snippet",
				InsertText:  "${1:column} BETWEEN ${2:min} AND ${3:max}",
				Description: "Check if value is in range",
			},
		}...)
	}

	return snippets
}

func (s *AutocompleteService) getGeneralSuggestions(context *Context) []Suggestion {
	suggestions := []Suggestion{}

	// Add common keywords
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "LEFT JOIN", "RIGHT JOIN",
		"GROUP BY", "ORDER BY", "HAVING", "LIMIT", "OFFSET",
		"UNION", "UNION ALL", "AS", "ON", "AND", "OR", "NOT",
		"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
	}

	for _, keyword := range keywords {
		suggestions = append(suggestions, Suggestion{
			Text: keyword,
			Type: "keyword",
		})
	}

	// Add tables if we have schema
	if s.schema != nil {
		suggestions = append(suggestions, s.getTableSuggestions()...)
	}

	return suggestions
}

func (s *AutocompleteService) sortSuggestions(suggestions []Suggestion, context *Context) {
	sort.Slice(suggestions, func(i, j int) bool {
		// Prioritize by type based on context
		typePriority := map[string]int{
			"column":   1,
			"table":    2,
			"keyword":  3,
			"function": 4,
			"snippet":  5,
		}

		// Adjust priorities based on context
		switch context.Type {
		case "after_from", "after_join":
			typePriority["table"] = 1
			typePriority["column"] = 5
		case "after_select":
			typePriority["column"] = 1
			typePriority["function"] = 2
		}

		iPriority := typePriority[suggestions[i].Type]
		jPriority := typePriority[suggestions[j].Type]

		if iPriority != jPriority {
			return iPriority < jPriority
		}

		// If same type, sort alphabetically
		return suggestions[i].Text < suggestions[j].Text
	})
}

// Helper functions

func (s *AutocompleteService) getLastKeyword(sql string) string {
	// Get the last SQL keyword before cursor
	keywords := []string{
		"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER", "FULL",
		"ON", "GROUP", "ORDER", "BY", "HAVING", "LIMIT", "OFFSET", "AS",
		"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE", "AND", "OR",
	}

	sql = strings.TrimSpace(sql)
	words := strings.Fields(sql)

	for i := len(words) - 1; i >= 0; i-- {
		word := strings.ToUpper(words[i])
		for _, keyword := range keywords {
			if word == keyword {
				return keyword
			}
		}
	}

	return ""
}

func (s *AutocompleteService) extractTables(sql string) []string {
	tables := []string{}
	upperSQL := strings.ToUpper(sql)

	// Simple extraction of table names after FROM and JOIN
	patterns := []string{
		`FROM\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
		`JOIN\s+([a-zA-Z_][a-zA-Z0-9_]*)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(upperSQL, -1)
		for _, match := range matches {
			if len(match) > 1 {
				// Get the original case table name
				tableStart := strings.Index(strings.ToUpper(sql), match[1])
				if tableStart >= 0 {
					tableName := sql[tableStart : tableStart+len(match[1])]
					tables = append(tables, tableName)
				}
			}
		}
	}

	return tables
}

func (s *AutocompleteService) isTypingTableColumn(sql string) bool {
	// Check if user is typing table.column pattern
	sql = strings.TrimSpace(sql)
	return regexp.MustCompile(`\w+\.$`).MatchString(sql)
}

func (s *AutocompleteService) extractCurrentTable(sql string) string {
	// Extract table name from "table." pattern
	matches := regexp.MustCompile(`(\w+)\.$`).FindStringSubmatch(sql)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (s *AutocompleteService) isStartOfStatement(sql string) bool {
	sql = strings.TrimSpace(sql)
	return sql == "" || regexp.MustCompile(`;\s*$`).MatchString(sql)
}

func (s *AutocompleteService) isAfterComma(sql string) bool {
	sql = strings.TrimSpace(sql)
	return strings.HasSuffix(sql, ",")
}

func getBuiltInFunctions() map[string]*Function {
	return map[string]*Function{
		"COUNT": {
			Name:        "COUNT",
			Description: "Count the number of rows",
			Signature:   "COUNT(${1:expression})",
		},
		"SUM": {
			Name:        "SUM",
			Description: "Calculate sum of values",
			Signature:   "SUM(${1:column})",
		},
		"AVG": {
			Name:        "AVG",
			Description: "Calculate average value",
			Signature:   "AVG(${1:column})",
		},
		"MAX": {
			Name:        "MAX",
			Description: "Find maximum value",
			Signature:   "MAX(${1:column})",
		},
		"MIN": {
			Name:        "MIN",
			Description: "Find minimum value",
			Signature:   "MIN(${1:column})",
		},
		"COALESCE": {
			Name:        "COALESCE",
			Description: "Return first non-null value",
			Signature:   "COALESCE(${1:value1}, ${2:value2})",
		},
		"LENGTH": {
			Name:        "LENGTH",
			Description: "Get string length",
			Signature:   "LENGTH(${1:string})",
		},
		"UPPER": {
			Name:        "UPPER",
			Description: "Convert to uppercase",
			Signature:   "UPPER(${1:string})",
		},
		"LOWER": {
			Name:        "LOWER",
			Description: "Convert to lowercase",
			Signature:   "LOWER(${1:string})",
		},
		"SUBSTR": {
			Name:        "SUBSTR",
			Description: "Extract substring",
			Signature:   "SUBSTR(${1:string}, ${2:start}, ${3:length})",
		},
		"NOW": {
			Name:        "NOW",
			Description: "Current timestamp",
			Signature:   "NOW()",
		},
		"DATE": {
			Name:        "DATE",
			Description: "Extract date",
			Signature:   "DATE(${1:datetime})",
		},
	}
}
