package multiquery

// TestSchemaBuilder provides helper methods to build test schemas with foreign key relationships
type TestSchemaBuilder struct {
	tables map[string]*TestTable
}

// TestTable represents a table with columns and constraints
type TestTable struct {
	Schema      string
	Name        string
	Columns     []*TestColumn
	PrimaryKey  []string
	ForeignKeys []*TestForeignKey
}

// TestColumn represents a column definition
type TestColumn struct {
	Name     string
	DataType string
	Nullable bool
	Indexed  bool
}

// TestForeignKey represents a foreign key constraint
type TestForeignKey struct {
	Name              string
	Columns           []string
	ReferencedSchema  string
	ReferencedTable   string
	ReferencedColumns []string
	OnDelete          string
	OnUpdate          string
}

// NewTestSchemaBuilder creates a new schema builder
func NewTestSchemaBuilder() *TestSchemaBuilder {
	return &TestSchemaBuilder{
		tables: make(map[string]*TestTable),
	}
}

// AddTable adds a table to the schema
func (b *TestSchemaBuilder) AddTable(schema, name string) *TestTableBuilder {
	return &TestTableBuilder{
		builder: b,
		table: &TestTable{
			Schema:      schema,
			Name:        name,
			Columns:     make([]*TestColumn, 0),
			ForeignKeys: make([]*TestForeignKey, 0),
		},
	}
}

// TestTableBuilder provides fluent API for building tables
type TestTableBuilder struct {
	builder *TestSchemaBuilder
	table   *TestTable
}

// AddColumn adds a column to the table
func (tb *TestTableBuilder) AddColumn(name, dataType string) *TestColumnBuilder {
	return &TestColumnBuilder{
		builder: tb,
		column:  &TestColumn{Name: name, DataType: dataType},
	}
}

// TestColumnBuilder provides fluent API for building columns
type TestColumnBuilder struct {
	builder *TestTableBuilder
	column  *TestColumn
}

// Nullable marks the column as nullable
func (cb *TestColumnBuilder) Nullable() *TestColumnBuilder {
	cb.column.Nullable = true
	return cb
}

// Indexed marks the column as indexed
func (cb *TestColumnBuilder) Indexed() *TestColumnBuilder {
	cb.column.Indexed = true
	return cb
}

// PrimaryKey marks the column as part of the primary key
func (cb *TestColumnBuilder) PrimaryKey() *TestTableBuilder {
	cb.builder.table.Columns = append(cb.builder.table.Columns, cb.column)
	cb.builder.table.PrimaryKey = append(cb.builder.table.PrimaryKey, cb.column.Name)
	return cb.builder
}

// Then adds the column and returns the table builder for chaining
func (cb *TestColumnBuilder) Then() *TestTableBuilder {
	cb.builder.table.Columns = append(cb.builder.table.Columns, cb.column)
	return cb.builder
}

// PrimaryKeys sets the primary key columns
func (tb *TestTableBuilder) PrimaryKeys(columns ...string) *TestTableBuilder {
	tb.table.PrimaryKey = columns
	return tb
}

// AddForeignKey adds a foreign key constraint
func (tb *TestTableBuilder) AddForeignKey(name string, columns []string, refSchema, refTable string, refColumns []string) *TestTableBuilder {
	fk := &TestForeignKey{
		Name:              name,
		Columns:           columns,
		ReferencedSchema:  refSchema,
		ReferencedTable:   refTable,
		ReferencedColumns: refColumns,
		OnDelete:          "CASCADE",
		OnUpdate:          "CASCADE",
	}
	tb.table.ForeignKeys = append(tb.table.ForeignKeys, fk)
	return tb
}

// Build completes the table definition and adds it to the schema
func (tb *TestTableBuilder) Build() *TestSchemaBuilder {
	key := tb.table.Schema + "." + tb.table.Name
	tb.builder.tables[key] = tb.table
	return tb.builder
}

// Schema returns a combined schema for use in tests
func (b *TestSchemaBuilder) Schema() map[string]*TestTable {
	return b.tables
}

// GetForeignKeys returns foreign keys for a table
func (b *TestSchemaBuilder) GetForeignKeys(schema, table string) []*TestForeignKey {
	key := schema + "." + table
	if t, ok := b.tables[key]; ok {
		return t.ForeignKeys
	}
	return []*TestForeignKey{}
}

// GetColumns returns columns for a table
func (b *TestSchemaBuilder) GetColumns(schema, table string) []*TestColumn {
	key := schema + "." + table
	if t, ok := b.tables[key]; ok {
		return t.Columns
	}
	return []*TestColumn{}
}

// CommonTestSchemas provides pre-built schemas for testing
var CommonTestSchemas = map[string]*TestSchemaBuilder{
	"e-commerce": buildECommerceSchema(),
	"multi-db":   buildMultiDatabaseSchema(),
	"circular":   buildCircularRefSchema(),
	"no-fk":      buildNoForeignKeySchema(),
}

// buildECommerceSchema creates a typical e-commerce schema
func buildECommerceSchema() *TestSchemaBuilder {
	builder := NewTestSchemaBuilder()

	// Users table
	builder.AddTable("public", "users").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("email", "varchar(255)").Indexed().Then().
		AddColumn("name", "varchar(255)").Then().
		AddColumn("created_at", "timestamp").Then().
		Build()

	// Products table
	builder.AddTable("public", "products").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("name", "varchar(255)").Indexed().Then().
		AddColumn("price", "decimal(10,2)").Then().
		AddColumn("category_id", "bigint").Then().
		AddColumn("created_at", "timestamp").Then().
		PrimaryKeys("id").
		AddForeignKey("fk_products_category", []string{"category_id"}, "public", "categories", []string{"id"}).
		Build()

	// Categories table
	builder.AddTable("public", "categories").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("name", "varchar(255)").Indexed().Then().
		AddColumn("parent_id", "bigint").Nullable().Then().
		Build()

	// Orders table
	builder.AddTable("public", "orders").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("user_id", "bigint").Then().
		AddColumn("created_at", "timestamp").Then().
		AddColumn("total", "decimal(10,2)").Then().
		AddForeignKey("fk_orders_user", []string{"user_id"}, "public", "users", []string{"id"}).
		Build()

	// Order items table
	builder.AddTable("public", "order_items").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("order_id", "bigint").Then().
		AddColumn("product_id", "bigint").Then().
		AddColumn("quantity", "int").Then().
		AddColumn("unit_price", "decimal(10,2)").Then().
		AddForeignKey("fk_order_items_order", []string{"order_id"}, "public", "orders", []string{"id"}).
		AddForeignKey("fk_order_items_product", []string{"product_id"}, "public", "products", []string{"id"}).
		Build()

	return builder
}

// buildMultiDatabaseSchema creates schemas across multiple databases
func buildMultiDatabaseSchema() *TestSchemaBuilder {
	builder := NewTestSchemaBuilder()

	// Customers in accounts DB
	builder.AddTable("public", "customers").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("name", "varchar(255)").Then().
		AddColumn("account_id", "bigint").Then().
		Build()

	// Accounts info (separate database)
	builder.AddTable("public", "accounts").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("company_name", "varchar(255)").Then().
		AddColumn("account_type", "varchar(50)").Then().
		Build()

	// Transactions in analytics DB
	builder.AddTable("public", "transactions").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("customer_id", "bigint").Indexed().Then().
		AddColumn("amount", "decimal(12,2)").Then().
		AddColumn("created_at", "timestamp").Then().
		Build()

	return builder
}

// buildCircularRefSchema creates a schema with circular foreign key references
func buildCircularRefSchema() *TestSchemaBuilder {
	builder := NewTestSchemaBuilder()

	// Department table (references manager from employees)
	builder.AddTable("public", "departments").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("name", "varchar(255)").Then().
		AddColumn("manager_id", "bigint").Nullable().Then().
		AddForeignKey("fk_dept_manager", []string{"manager_id"}, "public", "employees", []string{"id"}).
		Build()

	// Employee table (references department)
	builder.AddTable("public", "employees").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("name", "varchar(255)").Then().
		AddColumn("department_id", "bigint").Then().
		AddForeignKey("fk_emp_dept", []string{"department_id"}, "public", "departments", []string{"id"}).
		Build()

	return builder
}

// buildNoForeignKeySchema creates a schema without any foreign keys
func buildNoForeignKeySchema() *TestSchemaBuilder {
	builder := NewTestSchemaBuilder()

	// Logs table - no foreign keys
	builder.AddTable("public", "logs").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("level", "varchar(10)").Then().
		AddColumn("message", "text").Then().
		AddColumn("timestamp", "timestamp").Then().
		Build()

	// Events table - no foreign keys
	builder.AddTable("public", "events").
		AddColumn("id", "bigint").PrimaryKey().
		AddColumn("event_type", "varchar(100)").Then().
		AddColumn("data", "json").Nullable().Then().
		AddColumn("created_at", "timestamp").Then().
		Build()

	return builder
}

// MockDatabaseResult represents a mock database query result
type MockDatabaseResult struct {
	Columns  []string
	Rows     [][]interface{}
	RowCount int64
}

// NewMockDatabaseResult creates a new mock result
func NewMockDatabaseResult(columns []string) *MockDatabaseResult {
	return &MockDatabaseResult{
		Columns: columns,
		Rows:    make([][]interface{}, 0),
	}
}

// AddRow adds a row to the mock result
func (r *MockDatabaseResult) AddRow(values ...interface{}) *MockDatabaseResult {
	r.Rows = append(r.Rows, values)
	r.RowCount = int64(len(r.Rows))
	return r
}

// QueryScenario represents a complete query testing scenario
type QueryScenario struct {
	Name             string
	Query            string
	Schema           *TestSchemaBuilder
	ExpectedConns    []string
	HasJoins         bool
	HasAggregation   bool
	ShouldError      bool
	ErrorMessage     string
	ExpectedStrategy ExecutionStrategy
}

// CommonQueryScenarios provides pre-built query scenarios
var CommonQueryScenarios = []QueryScenario{
	{
		Name:             "Simple single database query",
		Query:            "SELECT * FROM users WHERE id = ?",
		ExpectedConns:    []string{},
		HasJoins:         false,
		HasAggregation:   false,
		ExpectedStrategy: StrategyAuto,
	},
	{
		Name:             "Cross-database join",
		Query:            "SELECT u.name, o.total FROM @accounts.users u JOIN @orders.orders o ON u.id = o.user_id",
		ExpectedConns:    []string{"accounts", "orders"},
		HasJoins:         true,
		HasAggregation:   false,
		ExpectedStrategy: StrategyFederated,
	},
	{
		Name:             "Three-way join across databases",
		Query:            "SELECT u.name, p.name, oi.quantity FROM @db1.users u JOIN @db2.orders o ON u.id = o.user_id JOIN @db3.order_items oi ON o.id = oi.order_id",
		ExpectedConns:    []string{"db1", "db2", "db3"},
		HasJoins:         true,
		HasAggregation:   false,
		ExpectedStrategy: StrategyFederated,
	},
	{
		Name:             "Aggregation with group by",
		Query:            "SELECT user_id, COUNT(*) as order_count FROM @orders.orders GROUP BY user_id",
		ExpectedConns:    []string{"orders"},
		HasJoins:         false,
		HasAggregation:   true,
		ExpectedStrategy: StrategyAuto,
	},
	{
		Name:             "Cross-database with aggregation",
		Query:            "SELECT u.country, COUNT(o.id) as total_orders FROM @accounts.users u LEFT JOIN @orders.orders o ON u.id = o.user_id GROUP BY u.country",
		ExpectedConns:    []string{"accounts", "orders"},
		HasJoins:         true,
		HasAggregation:   true,
		ExpectedStrategy: StrategyFederated,
	},
}
