# Query Optimization Guide

SQL Studio's intelligent query optimization features help you write better, faster SQL queries through automatic analysis, natural language conversion, and smart autocomplete.

## Table of Contents

- [Query Analyzer](#query-analyzer)
- [Natural Language to SQL](#natural-language-to-sql)
- [Schema-Aware Autocomplete](#schema-aware-autocomplete)
- [Query Explainer](#query-explainer)
- [API Endpoints](#api-endpoints)
- [Frontend Integration](#frontend-integration)

## Query Analyzer

The query analyzer automatically detects common SQL anti-patterns and provides actionable optimization suggestions.

### Detected Anti-Patterns

1. **SELECT * Usage**
   - Problem: Retrieves unnecessary columns, increasing network traffic
   - Solution: Specify only needed columns
   - Score Impact: -15 points

2. **Missing Indexes**
   - Problem: Full table scans on WHERE/JOIN columns
   - Solution: Add indexes on frequently queried columns
   - Score Impact: -10 points per missing index

3. **Functions in WHERE Clause**
   - Problem: Prevents index usage (non-sargable predicates)
   - Solution: Move calculations to the right side or pre-compute values
   - Score Impact: -15 points
   - Example: `WHERE UPPER(name) = 'JOHN'` → `WHERE name = 'John'`

4. **Leading Wildcards in LIKE**
   - Problem: Cannot use indexes efficiently
   - Solution: Use full-text search or redesign query
   - Score Impact: -10 points
   - Example: `WHERE name LIKE '%john%'`

5. **NOT IN with Subqueries**
   - Problem: Poor performance with NULL values
   - Solution: Use NOT EXISTS or LEFT JOIN
   - Score Impact: -10 points

6. **Missing JOIN Conditions**
   - Problem: Creates Cartesian products
   - Solution: Add proper ON clauses
   - Score Impact: -20 points

7. **Correlated Subqueries**
   - Problem: Executes once per row (N+1 problem)
   - Solution: Use JOINs or window functions
   - Score Impact: -15 points

8. **UPDATE/DELETE without WHERE**
   - Problem: Modifies/removes ALL rows
   - Solution: Add WHERE clause or use TRUNCATE
   - Score Impact: -30 points

9. **Multiple OR Conditions**
   - Problem: Less efficient than IN clause
   - Solution: Use IN for multiple values
   - Score Impact: -5 points
   - Example: `WHERE status = 'A' OR status = 'B'` → `WHERE status IN ('A', 'B')`

10. **Missing LIMIT**
    - Problem: May return excessive rows
    - Solution: Add LIMIT clause
    - Score Impact: -5 points

### Query Scoring

Queries are scored from 0-100:
- **80-100**: Excellent - Well-optimized query
- **60-79**: Good - Minor improvements possible
- **Below 60**: Needs Improvement - Significant optimization opportunities

### Complexity Levels

- **Simple**: Basic queries with 1-2 tables
- **Moderate**: Queries with JOINs or simple subqueries
- **Complex**: Multiple JOINs, subqueries, or aggregations

## Natural Language to SQL

Convert plain English queries to SQL using pattern matching.

### Supported Patterns (20+)

#### Basic Queries
- `show users` → `SELECT * FROM users LIMIT 100`
- `count products` → `SELECT COUNT(*) AS count FROM products`
- `list orders` → `SELECT * FROM orders LIMIT 100`

#### Filtering
- `find users where status is active` → `SELECT * FROM users WHERE status = 'active'`
- `get products where price > 100` → `SELECT * FROM products WHERE price > 100`
- `show orders where total between 50 and 200` → `SELECT * FROM orders WHERE total BETWEEN 50 AND 200`

#### Search Patterns
- `find users where name contains john` → `SELECT * FROM users WHERE name LIKE '%john%'`
- `search products with description has phone` → `SELECT * FROM products WHERE description LIKE '%phone%'`

#### Sorting & Limiting
- `show users ordered by name` → `SELECT * FROM users ORDER BY name ASC`
- `top 10 products sorted by price desc` → `SELECT * FROM products ORDER BY price DESC LIMIT 10`
- `first 5 customers` → `SELECT * FROM customers LIMIT 5`

#### Aggregations
- `sum of amount in orders` → `SELECT SUM(amount) AS total FROM orders`
- `average price from products` → `SELECT AVG(price) AS average FROM products`
- `maximum salary in employees` → `SELECT MAX(salary) AS maximum FROM employees`
- `minimum age in users` → `SELECT MIN(age) AS minimum FROM users`

#### Grouping
- `count users grouped by status` → `SELECT status, COUNT(*) AS count FROM users GROUP BY status`
- `sum sales by category` → `SELECT category, SUM(sales) FROM products GROUP BY category`

#### Distinct Values
- `show unique categories from products` → `SELECT DISTINCT categories FROM products`
- `list distinct statuses in orders` → `SELECT DISTINCT status FROM orders`

#### NULL Handling
- `find users where email is null` → `SELECT * FROM users WHERE email IS NULL`
- `get products where image is not null` → `SELECT * FROM products WHERE image IS NOT NULL`

#### IN Clause
- `find users where id in (1,2,3)` → `SELECT * FROM users WHERE id IN (1,2,3)`
- `get products where category in (electronics,books)` → `SELECT * FROM products WHERE category IN ('electronics', 'books')`

#### Data Modification
- `delete users where status is inactive` → `DELETE FROM users WHERE status = 'inactive'`
- `update users set status to active where id = 1` → `UPDATE users SET status = 'active' WHERE id = 1`
- `insert user with name john and email john@example.com` → `INSERT INTO user (name, email) VALUES ('john', 'john@example.com')`

#### Date-based Queries
- `show orders from today` → `SELECT * FROM orders WHERE DATE(created_at) = CURDATE()`
- `get users from this month` → `SELECT * FROM users WHERE MONTH(created_at) = MONTH(CURDATE())`

### Confidence Scores

- **High (0.8-1.0)**: Exact pattern match
- **Medium (0.5-0.79)**: Partial match with some ambiguity
- **Low (0.0-0.49)**: Weak match, review suggested SQL

## Schema-Aware Autocomplete

Context-aware SQL autocomplete that understands your database schema.

### Context Detection

The autocomplete system detects your current context:

1. **After SELECT**: Suggests columns, functions, and wildcards
2. **After FROM**: Suggests table names
3. **After WHERE**: Suggests columns and operators
4. **After JOIN**: Suggests tables and join types
5. **After ORDER BY**: Suggests columns with sort options
6. **After GROUP BY**: Suggests groupable columns

### Suggestion Types

- **Tables**: Database tables with column count
- **Columns**: Table columns with data types
- **Keywords**: SQL keywords (SELECT, WHERE, JOIN, etc.)
- **Functions**: SQL functions with signatures
- **Snippets**: Complete query templates

### Smart Features

- Filters suggestions based on context
- Shows data types for columns
- Indicates indexed columns
- Provides function signatures
- Includes code snippets for common patterns

## Query Explainer

Translates SQL queries into plain English explanations.

### Simple Explanation

```sql
SELECT name, email FROM users WHERE status = 'active'
```

**Explanation**: "This query retrieves the 'name' and 'email' columns from the 'users' table. The results are filtered based on the 'status' column."

### Complex Explanation

Provides detailed breakdown including:
- Query type and complexity
- Tables involved
- Operations performed
- Potential warnings
- Optimization suggestions

## API Endpoints

### POST /api/query/analyze

Analyze a SQL query for optimization opportunities.

**Request:**
```json
{
  "sql": "SELECT * FROM users WHERE name = 'John'",
  "connection_id": "conn-123" // Optional
}
```

**Response:**
```json
{
  "suggestions": [
    {
      "type": "select",
      "severity": "warning",
      "message": "Avoid using SELECT * - specify only needed columns",
      "improved_sql": "SELECT id, name, email FROM users WHERE name = 'John'",
      "impact": "Reduces network traffic and improves performance"
    }
  ],
  "score": 75,
  "warnings": [],
  "complexity": "simple",
  "estimated_cost": 25
}
```

### POST /api/query/nl2sql

Convert natural language to SQL.

**Request:**
```json
{
  "query": "show top 10 users ordered by created date",
  "connection_id": "conn-123" // Optional
}
```

**Response:**
```json
{
  "sql": "SELECT * FROM users ORDER BY created_at DESC LIMIT 10",
  "confidence": 0.85,
  "template": "Select with ORDER BY and LIMIT"
}
```

### POST /api/query/autocomplete

Get autocomplete suggestions.

**Request:**
```json
{
  "sql": "SELECT * FROM u",
  "cursor": 16,
  "connection_id": "conn-123" // Optional
}
```

**Response:**
```json
{
  "suggestions": [
    {
      "text": "users",
      "type": "table",
      "description": "15 columns"
    },
    {
      "text": "user_roles",
      "type": "table",
      "description": "3 columns"
    }
  ]
}
```

### POST /api/query/explain

Explain a SQL query in plain English.

**Request:**
```json
{
  "sql": "SELECT COUNT(*) FROM orders WHERE total > 100",
  "verbose": false
}
```

**Response:**
```json
{
  "explanation": "This query retrieves the count of all rows from the 'orders' table. The results are filtered based on the 'total' column."
}
```

### GET /api/query/patterns

Get all supported natural language patterns.

**Response:**
```json
{
  "patterns": [
    {
      "description": "Select all from table",
      "examples": ["show users", "get all customers"],
      "pattern": "^(?:show|get|select|list)(?:all)? (\\w+)"
    }
  ],
  "total": 25
}
```

## Frontend Integration

### Using the Query Optimizer Component

```typescript
import { QueryOptimizer } from '@/components/query/QueryOptimizer'

function QueryEditor() {
  const [sql, setSql] = useState('')

  return (
    <div>
      <textarea
        value={sql}
        onChange={(e) => setSql(e.target.value)}
      />
      <QueryOptimizer
        sql={sql}
        connectionId="conn-123"
        isEnabled={true}
      />
    </div>
  )
}
```

### Using Natural Language Input

```typescript
import { NaturalLanguageInput } from '@/components/query/NaturalLanguageInput'

function QueryBuilder() {
  const handleSQLGenerated = (sql: string) => {
    // Use the generated SQL
    console.log('Generated SQL:', sql)
  }

  return (
    <NaturalLanguageInput
      onSQLGenerated={handleSQLGenerated}
      connectionId="conn-123"
    />
  )
}
```

## Best Practices

### Query Optimization

1. **Always specify columns** instead of using SELECT *
2. **Add indexes** on columns used in WHERE and JOIN clauses
3. **Avoid functions** in WHERE clauses
4. **Use EXISTS** instead of IN for subqueries
5. **Add LIMIT** to prevent excessive result sets
6. **Use proper JOIN types** (avoid CROSS JOINs)
7. **Batch operations** when possible
8. **Monitor query execution plans** in production

### Natural Language Queries

1. **Be specific** about what you want to retrieve
2. **Include table names** for clarity
3. **Specify conditions** clearly (e.g., "where status is active")
4. **Use numbers** for limits and comparisons
5. **Follow common patterns** for best results

### Performance Considerations

1. **Cache analysis results** for identical queries
2. **Debounce real-time analysis** (1-second delay)
3. **Limit autocomplete suggestions** to top 20 results
4. **Use connection pooling** for schema retrieval
5. **Implement request throttling** for API endpoints

## Testing

Run the comprehensive test suite:

```bash
# Run analyzer tests
go test ./internal/analyzer -v

# Run NL2SQL tests
go test ./internal/nl2sql -v

# Run autocomplete tests
go test ./internal/autocomplete -v

# Run with coverage
go test ./... -cover
```

## Monitoring

Key metrics to track:

- Query analysis response time
- NL2SQL conversion success rate
- Autocomplete suggestion relevance
- Most common anti-patterns detected
- User adoption of suggestions

## Future Enhancements

1. **Machine Learning Integration** (Phase 5)
   - Learn from user query patterns
   - Personalized suggestions
   - Anomaly detection

2. **Query Performance History**
   - Track query execution times
   - Identify performance regressions
   - Suggest index creation based on usage

3. **Advanced NL2SQL**
   - Support for complex joins
   - Multi-step query generation
   - Context from previous queries

4. **Visual Query Builder**
   - Drag-and-drop interface
   - Real-time optimization feedback
   - Query visualization