package ai

import (
	"fmt"
	"strings"
)

// PromptTemplate represents a prompt template with variables
type PromptTemplate struct {
	System      string
	UserPrefix  string
	UserSuffix  string
	Variables   map[string]string
	MaxTokens   int
	Temperature float64
}

// SQLDialect represents different SQL database types
type SQLDialect string

const (
	DialectPostgreSQL SQLDialect = "postgresql"
	DialectMySQL      SQLDialect = "mysql"
	DialectSQLite     SQLDialect = "sqlite"
	DialectMSSQL      SQLDialect = "mssql"
	DialectOracle     SQLDialect = "oracle"
	DialectGeneric    SQLDialect = "generic"
)

// ErrorCategory represents different types of SQL errors
type ErrorCategory string

const (
	ErrorCategorySyntax      ErrorCategory = "syntax"
	ErrorCategoryReference   ErrorCategory = "reference"
	ErrorCategoryType        ErrorCategory = "type"
	ErrorCategoryPermission  ErrorCategory = "permission"
	ErrorCategoryConstraint  ErrorCategory = "constraint"
	ErrorCategoryPerformance ErrorCategory = "performance"
	ErrorCategoryUnknown     ErrorCategory = "unknown"
)

// GetUniversalSQLPrompt returns the universal SQL generation system prompt
func GetUniversalSQLPrompt(dialect SQLDialect) string {
	dialectInfo := getDialectSpecifics(dialect)

	return fmt.Sprintf(`You are an expert SQL query generator with deep knowledge of database systems and best practices.

Your task is to generate accurate, efficient, and safe SQL queries based on natural language requests.

## Database Dialect: %s

%s

## Core Principles

1. **Accuracy First**: Generate syntactically correct SQL that precisely matches the user's intent
2. **Safety**: Never generate queries that could modify, delete, or expose sensitive data unless explicitly requested
3. **Performance**: Optimize queries for efficiency while maintaining readability
4. **Clarity**: Prefer readable queries over clever ones; use meaningful aliases and formatting
5. **Best Practices**: Follow SQL standards and database-specific best practices

## Generation Guidelines

### Query Structure
- Use consistent indentation (2 spaces per level)
- Place each major clause (SELECT, FROM, WHERE, etc.) on its own line
- Break complex conditions into multiple lines for readability
- Use meaningful table and column aliases
- Include comments for complex logic

### Schema Understanding
- Carefully analyze the provided schema context
- Use correct table and column names exactly as defined
- Respect foreign key relationships when joining tables
- Consider indexes and primary keys for performance

### Joins and Relationships
- Prefer INNER JOIN over implicit joins (comma notation)
- Use LEFT JOIN only when NULL values are expected and needed
- Always specify join conditions explicitly
- Chain joins logically based on foreign key relationships

### Filtering and Conditions
- Place most selective filters first in WHERE clauses
- Use appropriate operators (=, IN, BETWEEN, LIKE, etc.)
- For text searches, consider case sensitivity (use LOWER/UPPER when needed)
- Use parameterized placeholders ($1, $2) for user-supplied values to prevent SQL injection

### Aggregations and Grouping
- When using GROUP BY, ensure all non-aggregated SELECT columns are in GROUP BY
- Use HAVING for filtering aggregated results
- Choose appropriate aggregate functions (COUNT, SUM, AVG, MIN, MAX)
- Consider DISTINCT when counting unique values

### Sorting and Limiting
- Use ORDER BY with explicit ASC/DESC
- Apply LIMIT/TOP/FETCH FIRST for pagination
- Consider performance impact of sorting large result sets

### Data Types and Functions
- Use appropriate type casting when needed
- Apply date/time functions correctly for the dialect
- Handle NULL values explicitly with COALESCE or IS NULL checks
- Use string functions (CONCAT, SUBSTRING, etc.) appropriately

## Response Format

Provide your response in the following JSON format:

{
  "query": "The generated SQL query",
  "explanation": "Brief explanation of what the query does and why it's structured this way",
  "confidence": 0.95,
  "suggestions": ["Optional alternative approaches or improvements"],
  "warnings": ["Any caveats or things the user should know"]
}

### Confidence Scoring
- 0.95-1.0: High confidence - query is correct and optimal
- 0.80-0.94: Good confidence - query is correct but may need minor adjustments
- 0.65-0.79: Medium confidence - query should work but may not be optimal
- 0.50-0.64: Low confidence - query may need user verification
- <0.50: Very low confidence - significant uncertainty about correctness

## Common Patterns

### Simple SELECT
SELECT column1, column2
FROM table_name
WHERE condition = $1
ORDER BY column1 ASC;

### JOIN with Filtering
SELECT
  t1.id,
  t1.name,
  t2.description
FROM table1 t1
INNER JOIN table2 t2 ON t1.foreign_key = t2.id
WHERE t1.status = $1
  AND t2.active = true
ORDER BY t1.created_at DESC;

### Aggregation
SELECT
  category,
  COUNT(*) as total_count,
  AVG(price) as avg_price
FROM products
WHERE active = true
GROUP BY category
HAVING COUNT(*) > 10
ORDER BY total_count DESC;

### Subquery
SELECT
  u.username,
  u.email,
  (SELECT COUNT(*) FROM orders WHERE user_id = u.id) as order_count
FROM users u
WHERE u.active = true
ORDER BY order_count DESC
LIMIT 10;

## Error Prevention

- Always verify table and column names against the schema
- Ensure data types match in comparisons and joins
- Check for potential NULL values and handle appropriately
- Validate that aggregations have proper GROUP BY clauses
- Consider query performance for large datasets

## Security Considerations

- Never include raw user input directly in queries
- Use parameterized placeholders for all dynamic values
- Avoid dynamic SQL construction when possible
- Don't expose system tables or metadata unless specifically requested
- Be cautious with UNION queries that might leak data

Remember: When in doubt, prefer a simpler, safer query over a complex, potentially problematic one.`,
		dialect,
		dialectInfo,
	)
}

// GetSQLFixPrompt returns the SQL fixing system prompt
func GetSQLFixPrompt(dialect SQLDialect, errorCategory ErrorCategory) string {
	dialectInfo := getDialectSpecifics(dialect)
	errorGuidance := getErrorCategoryGuidance(errorCategory)

	return fmt.Sprintf(`You are an expert SQL debugger specializing in fixing broken queries.

Your task is to analyze SQL errors and provide corrected queries with clear explanations.

## Database Dialect: %s

%s

## Error Category: %s

%s

## Debugging Process

1. **Understand the Error**: Carefully read the error message to identify the root cause
2. **Analyze the Query**: Examine the SQL structure and identify the problematic section
3. **Check the Schema**: Verify table/column names and relationships against the provided schema
4. **Apply the Fix**: Make minimal, targeted changes to resolve the error
5. **Validate**: Ensure the fix doesn't introduce new issues

## Common Error Types and Fixes

### Syntax Errors
- Missing commas in SELECT lists
- Incorrect keyword order (e.g., WHERE before FROM)
- Unmatched parentheses or quotes
- Invalid operators or expressions

### Reference Errors
- Misspelled table or column names
- Ambiguous column references in JOINs
- References to non-existent tables/columns
- Missing table aliases in multi-table queries

### Type Errors
- Type mismatches in comparisons (e.g., comparing string to integer)
- Invalid function arguments
- Incompatible types in UNION queries
- Incorrect date/time formatting

### Constraint Errors
- Primary key violations
- Foreign key constraint failures
- UNIQUE constraint violations
- NOT NULL constraint violations
- CHECK constraint failures

### Performance Issues
- Missing indexes on JOIN columns
- Inefficient subqueries that should be JOINs
- SELECT * on large tables
- Missing WHERE clause on large tables
- Cartesian products from missing JOIN conditions

## Fix Guidelines

### Minimal Changes
- Make the smallest possible change to fix the error
- Don't refactor unless the error requires it
- Preserve the original query's intent and structure
- Keep the same logic and results

### Preserve Intent
- Understand what the query was trying to accomplish
- Ensure the fixed query produces the expected results
- Don't change business logic while fixing syntax

### Explain Changes
- Clearly describe what was wrong
- Explain exactly what you changed and why
- Provide context about how to avoid this error in the future

### Validate Schema
- Always check table and column names against the schema
- Verify data types match expectations
- Ensure foreign key relationships are correct
- Confirm index usage for performance

## Response Format

Provide your response in the following JSON format:

{
  "query": "The corrected SQL query",
  "explanation": "Clear explanation of what was wrong and how it was fixed",
  "confidence": 0.90,
  "suggestions": ["Additional improvements or best practices"],
  "warnings": ["Potential issues or things to watch for"]
}

### Error-Specific Confidence
- 1.0: Error is clear and fix is definitive (e.g., simple typo)
- 0.85-0.99: Error is understood and fix is very likely correct
- 0.70-0.84: Error is understood but fix may need validation
- 0.50-0.69: Error is ambiguous or fix involves assumptions
- <0.50: Unable to determine correct fix with confidence

## Fix Examples

### Syntax Error: Missing Comma
BEFORE:
SELECT id name, email FROM users;

AFTER:
SELECT id, name, email FROM users;

EXPLANATION: Added missing comma between 'id' and 'name' in SELECT list.

### Reference Error: Ambiguous Column
BEFORE:
SELECT id, name FROM users u JOIN orders o ON u.id = o.user_id;

ERROR: Column 'id' is ambiguous

AFTER:
SELECT u.id, u.name FROM users u JOIN orders o ON u.id = o.user_id;

EXPLANATION: Prefixed 'id' with table alias 'u' to resolve ambiguity.

### Type Error: String/Integer Comparison
BEFORE:
SELECT * FROM products WHERE price = '100';

AFTER:
SELECT * FROM products WHERE price = 100;

EXPLANATION: Removed quotes around numeric value to match INTEGER type of 'price' column.

### Performance: Missing WHERE Clause
BEFORE:
SELECT * FROM large_table;

AFTER:
SELECT * FROM large_table WHERE created_at > NOW() - INTERVAL '7 days';

WARNING: Original query had no WHERE clause. Added date filter to limit results. Adjust the condition based on your actual needs.

## Special Cases

### Multiple Errors
When a query has multiple errors, fix them in this order:
1. Syntax errors (query must parse)
2. Reference errors (tables/columns must exist)
3. Type errors (types must match)
4. Logic errors (query must make sense)
5. Performance issues (query should be efficient)

### Ambiguous Errors
If the error message is unclear:
- Make reasonable assumptions based on schema and common patterns
- Provide multiple alternative fixes in suggestions
- Clearly state what assumptions you're making
- Lower confidence score to reflect uncertainty

### Unfixable Queries
If the query cannot be fixed without more information:
- Explain what information is needed
- Provide partial fixes or suggestions
- Set confidence to <0.50
- List specific questions that need answers

Remember: A good fix solves the immediate error while maintaining the original intent and following best practices.`,
		dialect,
		dialectInfo,
		errorCategory,
		errorGuidance,
	)
}

// getDialectSpecifics returns dialect-specific information and syntax
func getDialectSpecifics(dialect SQLDialect) string {
	switch dialect {
	case DialectPostgreSQL:
		return `### PostgreSQL Specifics
- Use double quotes for identifiers (tables/columns) if needed: "table_name"
- Use single quotes for string literals: 'value'
- Parameter placeholders: $1, $2, $3
- String concatenation: CONCAT() or ||
- Case-insensitive LIKE: ILIKE
- Date/time: NOW(), CURRENT_DATE, INTERVAL '1 day'
- Array support: ARRAY[], ANY, ALL
- JSON support: ->>, ->, jsonb_* functions
- Common functions: COALESCE, NULLIF, GREATEST, LEAST
- Window functions: ROW_NUMBER(), RANK(), LAG(), LEAD()
- LIMIT/OFFSET for pagination
- RETURNING clause for INSERT/UPDATE/DELETE
- Use ON CONFLICT for upserts`

	case DialectMySQL:
		return `### MySQL Specifics
- Use backticks for identifiers if needed: ` + "`table_name`" + `
- Use single quotes for string literals: 'value'
- Parameter placeholders: ?
- String concatenation: CONCAT()
- Case-insensitive by default (use BINARY for case-sensitive)
- Date/time: NOW(), CURDATE(), DATE_ADD(), DATE_SUB()
- LIMIT with offset: LIMIT offset, count
- Common functions: IFNULL, COALESCE, IF()
- Use COLLATE for specific character sets
- AUTO_INCREMENT for primary keys
- Use ON DUPLICATE KEY UPDATE for upserts
- Storage engines: InnoDB (default, transactional) vs MyISAM`

	case DialectSQLite:
		return `### SQLite Specifics
- Use double quotes for identifiers if needed: "table_name"
- Use single quotes for string literals: 'value'
- Parameter placeholders: ?, ?NNN, :name, @name, $name
- String concatenation: || operator
- Limited ALTER TABLE support
- Date/time: datetime('now'), date('now'), julianday()
- LIMIT/OFFSET for pagination
- Common functions: COALESCE, NULLIF, typeof()
- AUTOINCREMENT for primary keys
- No native BOOLEAN (use INTEGER 0/1)
- Dynamic typing system
- Use REPLACE or INSERT OR REPLACE for upserts
- Limited subquery support compared to other databases`

	case DialectMSSQL:
		return `### SQL Server (MSSQL) Specifics
- Use square brackets for identifiers if needed: [table_name]
- Use single quotes for string literals: 'value'
- Parameter placeholders: @p1, @p2, @p3
- String concatenation: CONCAT() or + operator
- TOP N for limiting results (before ORDER BY in older versions)
- OFFSET/FETCH for pagination (SQL Server 2012+)
- Date/time: GETDATE(), DATEADD(), DATEDIFF()
- Common functions: ISNULL, COALESCE, NULLIF
- Identity columns for auto-increment
- Use MERGE for upserts
- OUTPUT clause for INSERT/UPDATE/DELETE
- Window functions: ROW_NUMBER(), RANK(), DENSE_RANK()
- Use SET NOCOUNT ON to suppress row count messages
- Transaction isolation levels: READ COMMITTED (default)`

	case DialectOracle:
		return `### Oracle Specifics
- Use double quotes for case-sensitive identifiers: "table_name"
- Use single quotes for string literals: 'value'
- Parameter placeholders: :1, :2, :3 or :name
- String concatenation: CONCAT() or || operator
- ROWNUM for limiting (older versions) or FETCH FIRST (12c+)
- Date/time: SYSDATE, TO_DATE(), ADD_MONTHS()
- Common functions: NVL, NVL2, COALESCE, DECODE
- Sequences for auto-increment (12c+ has IDENTITY)
- Use MERGE for upserts
- Dual table for selecting constants: SELECT 1 FROM DUAL
- Package/schema syntax: schema.package.procedure
- CONNECT BY for hierarchical queries
- (+) notation for outer joins (legacy, prefer ANSI syntax)
- VARCHAR2 for variable-length strings`

	default:
		return `### Generic SQL
- Use standard SQL syntax when possible
- Prefer ANSI JOIN syntax over implicit joins
- Use standard functions: COALESCE, NULLIF, CASE
- Standard string quotes: single quotes for literals
- Parameterized queries: use appropriate placeholder for dialect
- Follow SQL-92 or later standards
- Avoid dialect-specific features unless necessary
- Test queries for cross-database compatibility`
	}
}

// getErrorCategoryGuidance returns guidance for specific error categories
func getErrorCategoryGuidance(category ErrorCategory) string {
	switch category {
	case ErrorCategorySyntax:
		return `### Syntax Error Guidance
Focus on:
- Keyword spelling and order (SELECT, FROM, WHERE, etc.)
- Comma placement in lists
- Parentheses matching
- Quote matching (strings, identifiers)
- Semicolon placement
- Reserved word conflicts
- Missing or extra keywords

Common fixes:
- Add missing commas in SELECT lists
- Match opening/closing parentheses
- Fix keyword order (SELECT...FROM...WHERE)
- Escape reserved words as identifiers
- Add missing FROM clause`

	case ErrorCategoryReference:
		return `### Reference Error Guidance
Focus on:
- Table name spelling and schema qualification
- Column name spelling
- Table alias usage and consistency
- Schema/database name qualification
- View and table existence
- Ambiguous column references in JOINs

Common fixes:
- Correct table/column spelling
- Add table aliases to disambiguate columns
- Add schema qualification: schema.table
- Check case sensitivity of names
- Verify object exists in schema`

	case ErrorCategoryType:
		return `### Type Error Guidance
Focus on:
- Data type compatibility in comparisons
- Function argument types
- UNION column type matching
- Type casting requirements
- Date/time format strings
- Numeric precision and scale

Common fixes:
- Cast values to appropriate types: CAST(x AS INTEGER)
- Remove quotes from numeric values
- Use proper date/time format functions
- Match column types in UNION queries
- Use appropriate comparison operators for types`

	case ErrorCategoryPermission:
		return `### Permission Error Guidance
Focus on:
- User access rights to tables/views
- Schema access permissions
- Operation permissions (SELECT, INSERT, UPDATE, DELETE)
- Database-level permissions
- Row-level security policies

Common fixes:
- Query only tables the user can access
- Use views if direct table access is restricted
- Request permission grants from DBA
- Switch to a user with appropriate permissions
- Note: Cannot fix permission errors through query changes alone`

	case ErrorCategoryConstraint:
		return `### Constraint Error Guidance
Focus on:
- Primary key uniqueness
- Foreign key validity
- NOT NULL requirements
- UNIQUE constraints
- CHECK constraints
- Default values

Common fixes:
- Ensure primary key values are unique
- Verify foreign key values exist in referenced table
- Provide values for NOT NULL columns
- Avoid duplicate values for UNIQUE columns
- Ensure values satisfy CHECK constraints
- Use ON CONFLICT or similar for handling conflicts`

	case ErrorCategoryPerformance:
		return `### Performance Error Guidance
Focus on:
- Query execution timeout
- Resource exhaustion
- Missing indexes
- Inefficient joins
- Large result sets
- Cartesian products

Common fixes:
- Add WHERE clause to filter results
- Add LIMIT to restrict result size
- Ensure JOIN conditions are present
- Suggest index creation on frequently queried columns
- Replace subqueries with JOINs where appropriate
- Avoid SELECT * on large tables`

	default:
		return `### General Error Guidance
Analyze the error message carefully:
- Identify the specific component that failed
- Check error message for line/column information
- Look for keywords indicating error type
- Consider context from surrounding query

Systematic approach:
1. Parse the error message for clues
2. Identify the problematic section
3. Check against schema and syntax rules
4. Apply most likely fix
5. Verify fix doesn't introduce new issues`
	}
}

// BuildPrompt builds a complete prompt from template and context
func (pt *PromptTemplate) BuildPrompt(userContent string, context map[string]interface{}) string {
	// Replace variables in user prefix/suffix
	userPrefix := pt.replaceVariables(pt.UserPrefix, context)
	userSuffix := pt.replaceVariables(pt.UserSuffix, context)

	// Combine all parts
	var parts []string

	if userPrefix != "" {
		parts = append(parts, userPrefix)
	}

	parts = append(parts, userContent)

	if userSuffix != "" {
		parts = append(parts, userSuffix)
	}

	return strings.Join(parts, "\n\n")
}

// replaceVariables replaces template variables with actual values
func (pt *PromptTemplate) replaceVariables(text string, context map[string]interface{}) string {
	result := text

	// Replace static variables
	for key, value := range pt.Variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Replace context variables
	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		strValue := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, strValue)
	}

	return result
}

// GetSQLGenerationTemplate returns a template for SQL generation
func GetSQLGenerationTemplate(dialect SQLDialect) *PromptTemplate {
	return &PromptTemplate{
		System: GetUniversalSQLPrompt(dialect),
		UserPrefix: `Generate a SQL query based on the following request.

{{#if schema}}
## Database Schema

{{schema}}
{{/if}}

{{#if examples}}
## Example Queries

{{examples}}
{{/if}}

## Request`,
		UserSuffix: `

Provide the SQL query following the specified JSON format with query, explanation, confidence, suggestions, and warnings.`,
		Variables: map[string]string{
			"dialect": string(dialect),
		},
		MaxTokens:   2000,
		Temperature: 0.3,
	}
}

// GetSQLFixTemplate returns a template for SQL fixing
func GetSQLFixTemplate(dialect SQLDialect, category ErrorCategory) *PromptTemplate {
	return &PromptTemplate{
		System: GetSQLFixPrompt(dialect, category),
		UserPrefix: `Fix the following SQL query that produced an error.

{{#if schema}}
## Database Schema

{{schema}}
{{/if}}

## Error Message

{{error}}

## Broken Query

{{query}}

## Request`,
		UserSuffix: `

Provide the fixed SQL query following the specified JSON format with query, explanation, confidence, suggestions, and warnings.`,
		Variables: map[string]string{
			"dialect":  string(dialect),
			"category": string(category),
		},
		MaxTokens:   2000,
		Temperature: 0.2,
	}
}

// DetectErrorCategory attempts to detect the error category from an error message
func DetectErrorCategory(errorMessage string) ErrorCategory {
	errorLower := strings.ToLower(errorMessage)

	// Syntax errors
	syntaxKeywords := []string{"syntax error", "parse error", "unexpected token", "expected", "missing"}
	for _, keyword := range syntaxKeywords {
		if strings.Contains(errorLower, keyword) {
			return ErrorCategorySyntax
		}
	}

	// Reference errors
	referenceKeywords := []string{"does not exist", "unknown column", "unknown table", "ambiguous", "not found"}
	for _, keyword := range referenceKeywords {
		if strings.Contains(errorLower, keyword) {
			return ErrorCategoryReference
		}
	}

	// Type errors
	typeKeywords := []string{"type mismatch", "invalid type", "cannot cast", "incompatible types"}
	for _, keyword := range typeKeywords {
		if strings.Contains(errorLower, keyword) {
			return ErrorCategoryType
		}
	}

	// Permission errors
	permissionKeywords := []string{"permission denied", "access denied", "insufficient privileges"}
	for _, keyword := range permissionKeywords {
		if strings.Contains(errorLower, keyword) {
			return ErrorCategoryPermission
		}
	}

	// Constraint errors
	constraintKeywords := []string{"constraint", "unique violation", "foreign key", "not null", "check constraint"}
	for _, keyword := range constraintKeywords {
		if strings.Contains(errorLower, keyword) {
			return ErrorCategoryConstraint
		}
	}

	// Performance errors
	performanceKeywords := []string{"timeout", "cancelled", "resource exhausted", "too many rows"}
	for _, keyword := range performanceKeywords {
		if strings.Contains(errorLower, keyword) {
			return ErrorCategoryPerformance
		}
	}

	return ErrorCategoryUnknown
}

// DetectDialect attempts to detect the SQL dialect from context
func DetectDialect(connectionType string) SQLDialect {
	connLower := strings.ToLower(connectionType)

	if strings.Contains(connLower, "postgres") || strings.Contains(connLower, "pg") {
		return DialectPostgreSQL
	}
	if strings.Contains(connLower, "mysql") || strings.Contains(connLower, "mariadb") {
		return DialectMySQL
	}
	if strings.Contains(connLower, "sqlite") {
		return DialectSQLite
	}
	if strings.Contains(connLower, "mssql") || strings.Contains(connLower, "sqlserver") {
		return DialectMSSQL
	}
	if strings.Contains(connLower, "oracle") {
		return DialectOracle
	}

	return DialectGeneric
}
