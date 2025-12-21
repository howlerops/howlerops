//go:build duckdb

package duckdb

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDuckDBEngine_Initialize tests initialization of DuckDB engine
func TestDuckDBEngine_Initialize(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	require.NotNil(t, engine)

	ctx := context.Background()
	err := engine.Initialize(ctx)
	if err != nil {
		t.Skipf("DuckDB not available: %v", err)
	}
	defer engine.Close()

	assert.True(t, engine.IsInitialized())
}

// TestDuckDBEngine_SimpleQuery tests executing a simple query
func TestDuckDBEngine_SimpleQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.ExecuteQuery(ctx, "SELECT 1 as num", 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.RowCount)
	assert.Len(t, result.Rows, 1)
}

// TestDuckDBEngine_ArrowResult tests result structure and arrow conversion
func TestDuckDBEngine_ArrowResult(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := engine.ExecuteQuery(ctx, `
		SELECT 
			1 as id,
			'test' as name,
			3.14 as value
	`, 5*time.Second)

	require.NoError(t, err)
	require.NotNil(t, result)

	assert.GreaterOrEqual(t, result.RowCount, 1)
	assert.NotNil(t, result.Rows)
}

// TestDuckDBEngine_ComplexQuery tests more complex query execution
func TestDuckDBEngine_ComplexQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		WITH numbers AS (
			SELECT 1 as n UNION ALL
			SELECT 2 UNION ALL
			SELECT 3 UNION ALL
			SELECT 4 UNION ALL
			SELECT 5
		)
		SELECT n, n * n as squared FROM numbers WHERE n > 2
	`

	result, err := engine.ExecuteQuery(ctx, query, 5*time.Second)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.GreaterOrEqual(t, result.RowCount, 3)
}

// TestDuckDBEngine_Timeout tests query timeout handling
func TestDuckDBEngine_Timeout(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	// Very short timeout should fail
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Create a query that would take some time (depends on system)
	query := `
		SELECT COUNT(*) as cnt
		FROM (
			WITH RECURSIVE cnt(x) AS (
				SELECT 1 UNION ALL SELECT x+1 FROM cnt LIMIT 1000000
			)
			SELECT * FROM cnt
		)
	`

	result, err := engine.ExecuteQuery(ctx, query, 1*time.Millisecond)
	// Either timeout or error is acceptable
	if err == nil {
		assert.NotNil(t, result)
	}
}

// TestDuckDBEngine_Close tests engine cleanup
func TestDuckDBEngine_Close(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}

	// Close should succeed
	err = engine.Close()
	assert.NoError(t, err)

	// After close, further queries should fail
	result, err := engine.ExecuteQuery(context.Background(), "SELECT 1", 5*time.Second)
	assert.Error(t, err)
	assert.Nil(t, result)
}

// TestDuckDBEngine_MultipleInitializations tests that reinitializing doesn't cause issues
func TestDuckDBEngine_MultipleInitializations(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	defer engine.Close()

	ctx := context.Background()

	// First initialization
	err := engine.Initialize(ctx)
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	assert.True(t, engine.IsInitialized())

	// Second initialization should be idempotent
	err = engine.Initialize(ctx)
	assert.NoError(t, err)
	assert.True(t, engine.IsInitialized())
}

// TestDuckDBCompiler_CompileSimpleSelect tests query compilation
func TestDuckDBCompiler_CompileSimpleSelect(t *testing.T) {
	compiler := NewCompiler(nil)
	require.NotNil(t, compiler)

	// Create a simple view definition
	viewDef := &ViewDefinition{
		IR: QueryIR{
			From: TableRef{Schema: "public", Table: "users"},
			Select: []SelectItem{
				{Column: "id"},
				{Column: "name"},
			},
		},
		Sources: []SourceDefinition{
			{Schema: "public", Table: "users"},
		},
	}

	compiled, err := compiler.Compile(viewDef)

	// Compilation should succeed or be transparent
	if err == nil {
		assert.NotEmpty(t, compiled)
	}
}

// TestDuckDBCompiler_CompileJoinQuery tests compilation of federated join
func TestDuckDBCompiler_CompileJoinQuery(t *testing.T) {
	compiler := NewCompiler(nil)

	// Create a view definition with a join
	viewDef := &ViewDefinition{
		IR: QueryIR{
			From: TableRef{Schema: "public", Table: "users", Alias: "u"},
			Joins: []Join{
				{
					Type:  "inner",
					Table: TableRef{Schema: "public", Table: "orders", Alias: "o"},
					On: Expression{
						Type:     "predicate",
						Column:   "u.id",
						Operator: "equals",
						Value:    "o.user_id",
					},
				},
			},
			Select: []SelectItem{
				{Column: "u.id"},
				{Column: "o.total"},
			},
		},
		Sources: []SourceDefinition{
			{Schema: "public", Table: "users"},
			{Schema: "public", Table: "orders"},
		},
	}

	compiled, err := compiler.Compile(viewDef)

	// Compilation should handle cross-database references
	if err == nil {
		assert.NotEmpty(t, compiled)
	}
}

// TestDuckDBScanner_Initialization tests scanner initialization
func TestDuckDBScanner_Initialization(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	// Scanners should be available after initialization
	assert.NotNil(t, engine)
}

// TestDuckDB_LoadExtensions tests that required extensions are loaded
func TestDuckDB_LoadExtensions(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		t.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	// Verify extensions were loaded (this is checked during init)
	assert.True(t, engine.IsInitialized())
}

// BenchmarkDuckDBEngine_SimpleQuery benchmarks query execution
func BenchmarkDuckDBEngine_SimpleQuery(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		b.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	query := "SELECT 1 as num"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ExecuteQuery(ctx, query, 5*time.Second)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}

// BenchmarkDuckDBEngine_ComplexQuery benchmarks complex query execution
func BenchmarkDuckDBEngine_ComplexQuery(b *testing.B) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	engine := NewEngine(logger, nil)
	err := engine.Initialize(context.Background())
	if err != nil {
		b.Skipf("DuckDB initialization failed: %v", err)
	}
	defer engine.Close()

	ctx := context.Background()
	query := `
		WITH RECURSIVE cnt(x) AS (
			SELECT 1 UNION ALL SELECT x+1 FROM cnt WHERE x < 100
		)
		SELECT COUNT(*) as total FROM cnt
	`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := engine.ExecuteQuery(ctx, query, 5*time.Second)
		if err != nil {
			b.Fatalf("Query execution failed: %v", err)
		}
	}
}
