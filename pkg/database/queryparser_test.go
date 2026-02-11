package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParseSimpleSelect tests the main query parser function
func TestParseSimpleSelect(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantSchema   string
		wantTable    string
		wantReason   string
		wantEditable bool
	}{
		// Empty and invalid queries
		{
			name:         "empty query",
			query:        "",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Empty query",
			wantEditable: false,
		},
		{
			name:         "whitespace only",
			query:        "   \t\n  ",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Empty query",
			wantEditable: false,
		},

		// Simple SELECT queries
		{
			name:         "simple select star",
			query:        "SELECT * FROM users",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "simple select columns",
			query:        "SELECT id, name, email FROM accounts",
			wantSchema:   "",
			wantTable:    "accounts",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select with trailing semicolon",
			query:        "SELECT * FROM products;",
			wantSchema:   "",
			wantTable:    "products",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select with leading/trailing whitespace",
			query:        "  \n  SELECT * FROM orders  \n  ",
			wantSchema:   "",
			wantTable:    "orders",
			wantReason:   "",
			wantEditable: true,
		},

		// Case insensitivity
		{
			name:         "lowercase select",
			query:        "select * from users",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "mixed case select",
			query:        "SeLeCt * FrOm UsErS",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},

		// Schema-qualified tables
		{
			name:         "schema.table",
			query:        "SELECT * FROM public.users",
			wantSchema:   "public",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "schema.table lowercase",
			query:        "SELECT * FROM myschema.mytable",
			wantSchema:   "myschema",
			wantTable:    "mytable",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "multiple dot notation",
			query:        "SELECT * FROM db.schema.table",
			wantSchema:   "db.schema",
			wantTable:    "table",
			wantReason:   "",
			wantEditable: true,
		},

		// Quoted identifiers
		{
			name:         "quoted table name",
			query:        `SELECT * FROM "my_table"`,
			wantSchema:   "",
			wantTable:    "my_table",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "quoted schema and table",
			query:        `SELECT * FROM "mySchema"."myTable"`,
			wantSchema:   "mySchema",
			wantTable:    "myTable",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "quoted table with underscores",
			query:        `SELECT * FROM "table_with_underscores"`,
			wantSchema:   "",
			wantTable:    "table_with_underscores",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "quoted identifiers with escaped quotes",
			query:        `SELECT * FROM "my""table"`,
			wantSchema:   "",
			wantTable:    `my"table`,
			wantReason:   "",
			wantEditable: true,
		},

		// ONLY keyword
		{
			name:         "select from ONLY table",
			query:        "SELECT * FROM ONLY users",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select from ONLY schema.table",
			query:        "SELECT * FROM ONLY public.accounts",
			wantSchema:   "public",
			wantTable:    "accounts",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "ONLY without table",
			query:        "SELECT * FROM ONLY",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Unable to determine target table",
			wantEditable: false,
		},

		// WHERE clause (should still be editable)
		{
			name:         "select with WHERE",
			query:        "SELECT * FROM users WHERE id = 1",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select with complex WHERE",
			query:        "SELECT * FROM accounts WHERE status = 'active' AND created_at > '2024-01-01'",
			wantSchema:   "",
			wantTable:    "accounts",
			wantReason:   "",
			wantEditable: true,
		},

		// ORDER BY, LIMIT, OFFSET (should still be editable)
		{
			name:         "select with ORDER BY",
			query:        "SELECT * FROM products ORDER BY name",
			wantSchema:   "",
			wantTable:    "products",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select with LIMIT",
			query:        "SELECT * FROM orders LIMIT 10",
			wantSchema:   "",
			wantTable:    "orders",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select with OFFSET",
			query:        "SELECT * FROM users OFFSET 20",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "select with ORDER BY, LIMIT, OFFSET",
			query:        "SELECT * FROM products ORDER BY price DESC LIMIT 10 OFFSET 5",
			wantSchema:   "",
			wantTable:    "products",
			wantReason:   "",
			wantEditable: true,
		},

		// Disallowed patterns
		{
			name:         "WITH clause (CTE)",
			query:        "WITH cte AS (SELECT * FROM users) SELECT * FROM cte",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Common table expressions are read-only",
			wantEditable: false,
		},
		{
			name:         "UNION operation",
			query:        "SELECT * FROM users UNION SELECT * FROM admins",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains UNION operations",
			wantEditable: false,
		},
		{
			name:         "UNION ALL operation",
			query:        "SELECT * FROM users UNION ALL SELECT * FROM admins",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains UNION operations",
			wantEditable: false,
		},
		{
			name:         "INTERSECT operation",
			query:        "SELECT id FROM users INTERSECT SELECT id FROM admins",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains INTERSECT operations",
			wantEditable: false,
		},
		{
			name:         "EXCEPT operation",
			query:        "SELECT id FROM users EXCEPT SELECT id FROM banned",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains EXCEPT operations",
			wantEditable: false,
		},
		{
			name:         "GROUP BY clause",
			query:        "SELECT COUNT(*) FROM users GROUP BY role",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains GROUP BY",
			wantEditable: false,
		},
		{
			name:         "HAVING clause",
			query:        "SELECT role, COUNT(*) FROM users GROUP BY role HAVING COUNT(*) > 5",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains GROUP BY",
			wantEditable: false,
		},
		{
			name:         "DISTINCT keyword",
			query:        "SELECT DISTINCT role FROM users",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query uses DISTINCT",
			wantEditable: false,
		},
		{
			name:         "RETURNING clause",
			query:        "INSERT INTO users (name) VALUES ('test') RETURNING id",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains RETURNING clause",
			wantEditable: false,
		},
		{
			name:         "FOR UPDATE clause",
			query:        "SELECT * FROM users WHERE id = 1 FOR UPDATE",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains FOR UPDATE",
			wantEditable: false,
		},

		// Subqueries in FROM
		{
			name:         "subquery in FROM",
			query:        "SELECT * FROM (SELECT * FROM users) AS u",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query selects from a subquery",
			wantEditable: false,
		},
		{
			name:         "complex subquery",
			query:        "SELECT * FROM (SELECT id, name FROM users WHERE active = true) AS active_users",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query selects from a subquery",
			wantEditable: false,
		},

		// Multiple tables (comma-separated)
		{
			name:         "multiple tables with comma",
			query:        "SELECT * FROM users, accounts",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query targets multiple tables",
			wantEditable: false,
		},
		{
			name:         "three tables with commas",
			query:        "SELECT * FROM users, accounts, orders",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query targets multiple tables",
			wantEditable: false,
		},

		// JOIN queries (should be editable for main table)
		{
			name:         "INNER JOIN",
			query:        "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "LEFT JOIN",
			query:        "SELECT * FROM users LEFT JOIN orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "RIGHT JOIN",
			query:        "SELECT * FROM users RIGHT JOIN orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "FULL OUTER JOIN",
			query:        "SELECT * FROM users FULL OUTER JOIN orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "JOIN with table alias",
			query:        "SELECT u.* FROM users u JOIN orders o ON u.id = o.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "JOIN with schema-qualified table",
			query:        "SELECT * FROM public.users JOIN public.orders ON users.id = orders.user_id",
			wantSchema:   "public",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "JOIN with quoted identifiers",
			query:        `SELECT * FROM "my_users" JOIN "my_orders" ON "my_users".id = "my_orders".user_id`,
			wantSchema:   "",
			wantTable:    "my_users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "multiple JOINs",
			query:        "SELECT * FROM users JOIN orders ON users.id = orders.user_id JOIN products ON orders.product_id = products.id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "JOIN with WHERE clause",
			query:        "SELECT * FROM users JOIN orders ON users.id = orders.user_id WHERE users.active = true",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},

		// Edge cases
		{
			name:         "no FROM clause",
			query:        "SELECT 1",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Unable to determine target table",
			wantEditable: false,
		},
		{
			name:         "FROM with no table",
			query:        "SELECT * FROM",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Unable to determine target table",
			wantEditable: false,
		},
		{
			name:         "table alias",
			query:        "SELECT * FROM users AS u",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "table alias without AS",
			query:        "SELECT * FROM users u",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "JOIN in FROM clause lowercase",
			query:        "SELECT * FROM users join orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "FROM ONLY with just ONLY",
			query:        "SELECT * FROM ONLY",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Unable to determine target table",
			wantEditable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table, reason, editable := parseSimpleSelect(tt.query)
			assert.Equal(t, tt.wantSchema, schema, "schema mismatch")
			assert.Equal(t, tt.wantTable, table, "table mismatch")
			assert.Equal(t, tt.wantReason, reason, "reason mismatch")
			assert.Equal(t, tt.wantEditable, editable, "editable mismatch")
		})
	}
}

// TestParseJoinQuery tests the JOIN query parser function
func TestParseJoinQuery(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantSchema   string
		wantTable    string
		wantReason   string
		wantEditable bool
	}{
		{
			name:         "simple JOIN",
			query:        "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "LEFT JOIN",
			query:        "SELECT * FROM accounts LEFT JOIN transactions ON accounts.id = transactions.account_id",
			wantSchema:   "",
			wantTable:    "accounts",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "schema-qualified JOIN",
			query:        "SELECT * FROM public.users JOIN public.orders ON users.id = orders.user_id",
			wantSchema:   "public",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "quoted identifiers in JOIN",
			query:        `SELECT * FROM "my_table" JOIN "other_table" ON "my_table".id = "other_table".ref_id`,
			wantSchema:   "",
			wantTable:    "my_table",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "JOIN with table alias",
			query:        "SELECT u.* FROM users u JOIN orders o ON u.id = o.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "no FROM clause",
			query:        "SELECT * JOIN orders",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "No FROM clause found",
			wantEditable: false,
		},
		{
			name:         "no JOIN in FROM clause",
			query:        "SELECT * FROM users",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "No JOIN found in FROM clause",
			wantEditable: false,
		},
		{
			name:         "empty FROM clause before JOIN",
			query:        "SELECT * FROM JOIN orders",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "No JOIN found in FROM clause",
			wantEditable: false,
		},
		{
			name:         "JOIN with trailing semicolon",
			query:        "SELECT * FROM users; JOIN orders ON users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table, reason, editable := parseJoinQuery(tt.query)
			assert.Equal(t, tt.wantSchema, schema, "schema mismatch")
			assert.Equal(t, tt.wantTable, table, "table mismatch")
			assert.Equal(t, tt.wantReason, reason, "reason mismatch")
			assert.Equal(t, tt.wantEditable, editable, "editable mismatch")
		})
	}
}

// TestExtractFromClause tests the FROM clause extraction function
func TestExtractFromClause(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "simple FROM",
			query: "SELECT * FROM users",
			want:  "users",
		},
		{
			name:  "FROM with WHERE",
			query: "SELECT * FROM users WHERE id = 1",
			want:  "users",
		},
		{
			name:  "FROM with ORDER BY",
			query: "SELECT * FROM products ORDER BY name",
			want:  "products",
		},
		{
			name:  "FROM with LIMIT",
			query: "SELECT * FROM orders LIMIT 10",
			want:  "orders",
		},
		{
			name:  "FROM with OFFSET",
			query: "SELECT * FROM accounts OFFSET 5",
			want:  "accounts",
		},
		{
			name:  "FROM with GROUP BY",
			query: "SELECT COUNT(*) FROM users GROUP BY role",
			want:  "users",
		},
		{
			name:  "FROM with RETURNING",
			query: "INSERT INTO users (name) VALUES ('test') RETURNING id",
			want:  "",
		},
		{
			name:  "FROM with FOR UPDATE",
			query: "SELECT * FROM users FOR UPDATE",
			want:  "users",
		},
		{
			name:  "FROM with UNION",
			query: "SELECT * FROM users UNION SELECT * FROM admins",
			want:  "users",
		},
		{
			name:  "FROM with INTERSECT",
			query: "SELECT * FROM users INTERSECT SELECT * FROM admins",
			want:  "users",
		},
		{
			name:  "FROM with EXCEPT",
			query: "SELECT * FROM users EXCEPT SELECT * FROM banned",
			want:  "users",
		},
		{
			name:  "no FROM clause",
			query: "SELECT 1",
			want:  "",
		},
		{
			name:  "schema-qualified table",
			query: "SELECT * FROM public.users WHERE active = true",
			want:  "public.users",
		},
		{
			name:  "quoted identifier",
			query: `SELECT * FROM "my_table" WHERE id = 1`,
			want:  `"my_table"`,
		},
		{
			name:  "table with alias",
			query: "SELECT * FROM users u WHERE u.id = 1",
			want:  "users u",
		},
		{
			name:  "multiple keywords",
			query: "SELECT * FROM products WHERE price > 100 ORDER BY price DESC LIMIT 10",
			want:  "products",
		},
		{
			name:  "lowercase from",
			query: "select * from users where id = 1",
			want:  "users",
		},
		{
			name:  "FROM with trailing whitespace",
			query: "SELECT * FROM users   WHERE id = 1",
			want:  "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFromClause(tt.query)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestSplitIdentifier tests the identifier splitting function
func TestSplitIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		wantSchema string
		wantTable  string
	}{
		{
			name:       "simple table",
			identifier: "users",
			wantSchema: "",
			wantTable:  "users",
		},
		{
			name:       "schema.table",
			identifier: "public.users",
			wantSchema: "public",
			wantTable:  "users",
		},
		{
			name:       "quoted table",
			identifier: `"my table"`,
			wantSchema: "",
			wantTable:  "my table",
		},
		{
			name:       "quoted schema and table",
			identifier: `"mySchema"."myTable"`,
			wantSchema: "mySchema",
			wantTable:  "myTable",
		},
		{
			name:       "three-part identifier",
			identifier: "db.schema.table",
			wantSchema: "db.schema",
			wantTable:  "table",
		},
		{
			name:       "empty identifier",
			identifier: "",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "whitespace only",
			identifier: "   ",
			wantSchema: "",
			wantTable:  "",
		},
		{
			name:       "uppercase table",
			identifier: "USERS",
			wantSchema: "",
			wantTable:  "users",
		},
		{
			name:       "mixed case schema.table",
			identifier: "MySchema.MyTable",
			wantSchema: "myschema",
			wantTable:  "mytable",
		},
		{
			name:       "quoted with escaped quotes",
			identifier: `"my""table"`,
			wantSchema: "",
			wantTable:  `my"table`,
		},
		{
			name:       "leading/trailing whitespace",
			identifier: "  public.users  ",
			wantSchema: "public",
			wantTable:  "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table := splitIdentifier(tt.identifier)
			assert.Equal(t, tt.wantSchema, schema, "schema mismatch")
			assert.Equal(t, tt.wantTable, table, "table mismatch")
		})
	}
}

// TestUnquoteIdentifierPart tests the identifier unquoting function
func TestUnquoteIdentifierPart(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{
			name:  "simple identifier",
			value: "users",
			want:  "users",
		},
		{
			name:  "uppercase identifier",
			value: "USERS",
			want:  "users",
		},
		{
			name:  "mixed case identifier",
			value: "MyTable",
			want:  "mytable",
		},
		{
			name:  "quoted identifier",
			value: `"myTable"`,
			want:  "myTable",
		},
		{
			name:  "quoted with underscores",
			value: `"my_table"`,
			want:  "my_table",
		},
		{
			name:  "quoted with escaped quotes",
			value: `"my""table"`,
			want:  `my"table`,
		},
		{
			name:  "quoted with multiple escaped quotes",
			value: `"my""special""table"`,
			want:  `my"special"table`,
		},
		{
			name:  "empty string",
			value: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			value: "   ",
			want:  "",
		},
		{
			name:  "leading/trailing whitespace",
			value: "  users  ",
			want:  "users",
		},
		{
			name:  "quoted with whitespace",
			value: `  "myTable"  `,
			want:  "myTable",
		},
		{
			name:  "identifier with underscore",
			value: "user_accounts",
			want:  "user_accounts",
		},
		{
			name:  "identifier with numbers",
			value: "table123",
			want:  "table123",
		},
		{
			name:  "quoted uppercase preserved",
			value: `"USERS"`,
			want:  "USERS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := unquoteIdentifierPart(tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestComplexQueries tests complex real-world query scenarios
func TestComplexQueries(t *testing.T) {
	tests := []struct {
		name         string
		query        string
		wantSchema   string
		wantTable    string
		wantReason   string
		wantEditable bool
	}{
		{
			name:         "complex WHERE with multiple conditions",
			query:        "SELECT id, name, email FROM users WHERE active = true AND role IN ('admin', 'user') AND created_at > '2024-01-01' ORDER BY created_at DESC",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "complex JOIN with multiple conditions",
			query:        "SELECT u.id, u.name, o.order_number FROM users u LEFT JOIN orders o ON u.id = o.user_id AND o.status = 'active' WHERE u.active = true ORDER BY u.created_at DESC LIMIT 100",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "aggregation query (not editable)",
			query:        "SELECT role, COUNT(*) as count, AVG(age) as avg_age FROM users GROUP BY role HAVING COUNT(*) > 10 ORDER BY count DESC",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query contains GROUP BY",
			wantEditable: false,
		},
		{
			name:         "CTE with JOIN (not editable)",
			query:        "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users JOIN orders ON active_users.id = orders.user_id",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Common table expressions are read-only",
			wantEditable: false,
		},
		{
			name:         "DISTINCT with JOIN (not editable)",
			query:        "SELECT DISTINCT u.role FROM users u JOIN orders o ON u.id = o.user_id",
			wantSchema:   "",
			wantTable:    "",
			wantReason:   "Query uses DISTINCT",
			wantEditable: false,
		},
		{
			name:         "schema-qualified with complex WHERE",
			query:        "SELECT * FROM public.users WHERE status = 'active' AND (role = 'admin' OR role = 'moderator') ORDER BY last_login DESC LIMIT 50 OFFSET 10",
			wantSchema:   "public",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
		{
			name:         "multiple JOINs (editable, first table)",
			query:        "SELECT u.*, o.order_number, p.name as product_name FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id WHERE u.active = true",
			wantSchema:   "",
			wantTable:    "users",
			wantReason:   "",
			wantEditable: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, table, reason, editable := parseSimpleSelect(tt.query)
			assert.Equal(t, tt.wantSchema, schema, "schema mismatch")
			assert.Equal(t, tt.wantTable, table, "table mismatch")
			assert.Equal(t, tt.wantReason, reason, "reason mismatch")
			assert.Equal(t, tt.wantEditable, editable, "editable mismatch")
		})
	}
}
