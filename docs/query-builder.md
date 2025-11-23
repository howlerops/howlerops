# Visual Query Builder

A comprehensive visual query builder system that allows non-SQL users to construct database queries through an intuitive UI, with automatic SQL generation and validation.

## Overview

The Visual Query Builder provides:
- **No-code query construction** - Build queries without writing SQL
- **Mode switching** - Toggle between visual builder and SQL editor
- **Real-time validation** - Catch errors before running queries
- **Schema introspection** - Browse tables and columns from connected databases
- **SQL injection prevention** - All queries use parameterized values
- **Query preview** - Test queries before saving

## Architecture

### Frontend Components

#### 1. QueryBuilder (`query-builder.tsx`)
Main visual builder component with step-by-step query construction:

```typescript
<QueryBuilder
  state={builderState}
  onChange={setBuilderState}
  onGenerateSQL={handleSQL}
  disabled={false}
/>
```

**Features:**
- Data source & table selection
- Column selection with aggregations (COUNT, SUM, AVG, MIN, MAX)
- Join configuration (INNER, LEFT, RIGHT, FULL)
- Filter conditions with multiple operators
- GROUP BY selection
- ORDER BY sorting
- LIMIT controls
- Real-time validation
- Query preview execution

#### 2. QueryModeSwitcher (`query-mode-switcher.tsx`)
Handles switching between visual and SQL modes:

```typescript
<QueryModeSwitcher
  mode="builder"
  connectionId={connectionId}
  sql={sql}
  builderState={builderState}
  onChange={handleChanges}
/>
```

**Mode Switching Rules:**
- Visual → SQL: Always allowed, shows generated SQL
- SQL → Visual: Only if SQL was generated from visual builder
- Manual SQL edits: Locks out visual mode permanently

### Backend Services

#### 1. SQL Generation (`pkg/querybuilder/builder.go`)

Uses Squirrel library for safe SQL generation:

```go
qb := &QueryBuilder{
    DataSource: "conn-123",
    Table:      "users",
    Columns: []ColumnSelection{
        {Table: "users", Column: "id"},
        {Table: "users", Column: "email"},
    },
    Filters: []FilterCondition{
        {Column: "users.active", Operator: "=", Value: ptrStr("true")},
    },
    Limit: ptrInt(100),
}

sql, args, err := qb.ToSQL()
// Result: SELECT "users"."id", "users"."email" FROM "users" WHERE users.active = $1 LIMIT 100
// Args: ["true"]
```

**Key Features:**
- Parameterized queries (SQL injection prevention)
- Support for all major SQL operations
- Validation before generation
- Clean, readable SQL output

#### 2. Schema Introspection API (`pkg/querybuilder/handler.go`)

**Endpoints:**

```
GET /api/connections/{connectionId}/schema
```
Returns complete database schema including tables, columns, foreign keys.

```json
{
  "tables": [
    {
      "schema": "public",
      "name": "users",
      "rowCount": 15234,
      "columns": [
        {
          "name": "id",
          "dataType": "integer",
          "primaryKey": true,
          "nullable": false
        }
      ],
      "foreignKeys": []
    }
  ]
}
```

**SQL Generation API:**

```
POST /api/querybuilder/generate
```

Request:
```json
{
  "queryBuilder": {
    "dataSource": "conn-123",
    "table": "orders",
    "columns": [
      {"table": "orders", "column": "customer_id"},
      {"table": "orders", "column": "id", "aggregation": "count", "alias": "order_count"}
    ],
    "groupBy": ["orders.customer_id"]
  }
}
```

Response:
```json
{
  "sql": "SELECT \"orders\".\"customer_id\", COUNT(\"orders\".\"id\") AS order_count FROM \"orders\" GROUP BY orders.customer_id",
  "args": [],
  "parameters": 0
}
```

**Validation API:**

```
POST /api/querybuilder/validate
```

Returns validation errors and warnings:

```json
{
  "valid": false,
  "errors": [
    {
      "field": "columns",
      "message": "At least one column must be selected",
      "severity": "error"
    }
  ],
  "warnings": [
    {
      "field": "limit",
      "message": "Consider adding a LIMIT to improve query performance",
      "severity": "warning"
    }
  ]
}
```

**Query Preview API:**

```
POST /api/querybuilder/preview
```

Executes query with safety limits:

```json
{
  "connectionId": "conn-123",
  "sql": "SELECT * FROM users LIMIT 10",
  "limit": 10
}
```

Response:
```json
{
  "sql": "SELECT * FROM users LIMIT 10",
  "estimatedRows": 10,
  "columns": ["id", "name", "email"],
  "rows": [[1, "John", "john@example.com"]],
  "totalRows": 10,
  "executionTimeMs": 45
}
```

## Type Definitions

### QueryBuilderState

```typescript
interface QueryBuilderState {
  dataSource: string      // connectionId
  table: string
  columns: ColumnSelection[]
  joins: JoinDefinition[]
  filters: FilterCondition[]
  groupBy: string[]       // ["table.column"]
  orderBy: OrderByClause[]
  limit?: number
  offset?: number
}
```

### ColumnSelection

```typescript
interface ColumnSelection {
  table: string
  column: string
  alias?: string
  aggregation?: 'count' | 'sum' | 'avg' | 'min' | 'max' | 'count_distinct'
}
```

### FilterCondition

```typescript
interface FilterCondition {
  id: string
  column: string          // "table.column"
  operator: FilterOperator
  value?: unknown
  valueTo?: unknown       // for BETWEEN
  combinator?: 'AND' | 'OR'
}

type FilterOperator =
  | '=' | '!=' | '>' | '<' | '>=' | '<='
  | 'LIKE' | 'NOT LIKE'
  | 'IN' | 'NOT IN'
  | 'IS NULL' | 'IS NOT NULL'
  | 'BETWEEN'
```

### JoinDefinition

```typescript
interface JoinDefinition {
  type: 'INNER' | 'LEFT' | 'RIGHT' | 'FULL'
  table: string
  alias?: string
  on: {
    left: string   // "table.column"
    right: string  // "table.column"
  }
}
```

## Usage Examples

### Basic Query

```typescript
const state: QueryBuilderState = {
  dataSource: 'conn-123',
  table: 'users',
  columns: [
    { table: 'users', column: 'id' },
    { table: 'users', column: 'name' },
    { table: 'users', column: 'email' }
  ],
  joins: [],
  filters: [],
  groupBy: [],
  orderBy: [],
  limit: 100
}

// Generated SQL:
// SELECT "users"."id", "users"."name", "users"."email" FROM "users" LIMIT 100
```

### Query with Aggregation

```typescript
const state: QueryBuilderState = {
  dataSource: 'conn-123',
  table: 'orders',
  columns: [
    { table: 'orders', column: 'customer_id' },
    { table: 'orders', column: 'id', aggregation: 'count', alias: 'total_orders' },
    { table: 'orders', column: 'total', aggregation: 'sum', alias: 'revenue' }
  ],
  joins: [],
  filters: [],
  groupBy: ['orders.customer_id'],
  orderBy: [{ column: 'revenue', direction: 'DESC' }],
  limit: 50
}

// Generated SQL:
// SELECT "orders"."customer_id", COUNT("orders"."id") AS total_orders, SUM("orders"."total") AS revenue
// FROM "orders"
// GROUP BY orders.customer_id
// ORDER BY revenue DESC
// LIMIT 50
```

### Query with Join and Filters

```typescript
const state: QueryBuilderState = {
  dataSource: 'conn-123',
  table: 'orders',
  columns: [
    { table: 'customers', column: 'name' },
    { table: 'orders', column: 'id', aggregation: 'count', alias: 'order_count' }
  ],
  joins: [
    {
      type: 'INNER',
      table: 'customers',
      on: { left: 'orders.customer_id', right: 'customers.id' }
    }
  ],
  filters: [
    {
      id: 'f1',
      column: 'customers.active',
      operator: '=',
      value: 'true'
    },
    {
      id: 'f2',
      column: 'orders.created_at',
      operator: 'BETWEEN',
      value: '2024-01-01',
      valueTo: '2024-12-31',
      combinator: 'AND'
    }
  ],
  groupBy: ['customers.name'],
  orderBy: [{ column: 'order_count', direction: 'DESC' }],
  limit: 20
}

// Generated SQL:
// SELECT "customers"."name", COUNT("orders"."id") AS order_count
// FROM "orders"
// INNER JOIN "customers" ON orders.customer_id = customers.id
// WHERE customers.active = $1 AND orders.created_at BETWEEN $2 AND $3
// GROUP BY customers.name
// ORDER BY order_count DESC
// LIMIT 20
```

## Security Features

### SQL Injection Prevention

All user inputs are parameterized:

```go
// User input: "'; DROP TABLE users; --"
filter := FilterCondition{
    Column:   "users.email",
    Operator: "=",
    Value:    ptrStr("'; DROP TABLE users; --"),
}

// Generated SQL (SAFE):
// WHERE users.email = $1
// Args: ["'; DROP TABLE users; --"]
```

### Schema Validation

Before execution, queries are validated against the actual database schema:
- Column names must exist in selected tables
- Join conditions reference valid columns
- Data types match operators
- Aggregations used correctly with GROUP BY

### Query Limits

- Preview queries: Max 100 rows
- Auto-timeout: 30 seconds
- Read-only mode for previews
- Connection isolation

## Performance Optimization

### Caching

- **Schema cache**: Database schemas cached for 15 minutes
- **Debounced SQL generation**: 300ms delay after state changes
- **Lazy loading**: Table/column lists loaded on demand
- **Memoized components**: Prevent unnecessary re-renders

### Query Optimization

- **Index hints**: Suggests indexes based on filters and joins
- **Join order**: Optimizes join sequence for performance
- **Limit enforcement**: Warns when LIMIT not set
- **Execution plans**: EXPLAIN available for complex queries

## Testing

### Unit Tests

```bash
cd backend-go
go test ./pkg/querybuilder/...
```

**Coverage:**
- Basic SELECT queries
- Aggregations (COUNT, SUM, AVG, MIN, MAX, COUNT DISTINCT)
- All filter operators (=, !=, >, <, >=, <=, LIKE, IN, NULL, BETWEEN)
- Join types (INNER, LEFT, RIGHT, FULL)
- ORDER BY and LIMIT
- Validation rules
- Error handling

### Integration Tests

Test complete workflow:
1. Schema introspection
2. Query building
3. SQL generation
4. Validation
5. Preview execution
6. Result rendering

## Common Patterns

### Daily Metrics Report

```typescript
{
  table: 'events',
  columns: [
    { table: 'events', column: 'date', aggregation: 'count_distinct', alias: 'days' },
    { table: 'events', column: 'user_id', aggregation: 'count_distinct', alias: 'users' },
    { table: 'events', column: 'id', aggregation: 'count', alias: 'events' }
  ],
  filters: [
    {
      column: 'events.created_at',
      operator: '>=',
      value: '2024-01-01'
    }
  ],
  groupBy: ['DATE(events.created_at)']
}
```

### Top N Analysis

```typescript
{
  table: 'products',
  columns: [
    { table: 'products', column: 'name' },
    { table: 'sales', column: 'revenue', aggregation: 'sum', alias: 'total_revenue' }
  ],
  joins: [
    {
      type: 'INNER',
      table: 'sales',
      on: { left: 'products.id', right: 'sales.product_id' }
    }
  ],
  groupBy: ['products.name'],
  orderBy: [{ column: 'total_revenue', direction: 'DESC' }],
  limit: 10
}
```

### Period-over-Period Comparison

Use window functions (via custom SQL mode) after building base query visually.

## Troubleshooting

### "Cannot switch to visual mode"
**Cause**: SQL was manually edited
**Solution**: Create a new component with visual builder

### "GROUP BY required" error
**Cause**: Using aggregations with non-aggregated columns
**Solution**: Add non-aggregated columns to GROUP BY or aggregate them

### "Schema not loading"
**Cause**: Connection error or timeout
**Solution**: Check connection status, verify credentials, check network

### Slow preview queries
**Cause**: Large tables without filters
**Solution**: Add WHERE filters, reduce selected columns, add LIMIT

## Future Enhancements

- [ ] Subquery support
- [ ] Window functions in visual mode
- [ ] Query templates library
- [ ] AI-assisted query suggestions
- [ ] Visual EXPLAIN plan viewer
- [ ] Multi-database federation queries
- [ ] Saved query snippets
- [ ] Collaborative query building
- [ ] Version history for queries
- [ ] Query performance insights

## API Reference

See [API Documentation](./api-reference.md) for detailed endpoint specs.

## Contributing

When adding new features to the query builder:

1. Add types to `frontend/src/types/reports.ts`
2. Update `QueryBuilderState` interface
3. Implement UI in `query-builder.tsx`
4. Add SQL generation logic to `backend-go/pkg/querybuilder/builder.go`
5. Add validation rules to `Validate()` function
6. Write unit tests
7. Update this documentation

## License

Copyright © 2024 HowlerOps. All rights reserved.
