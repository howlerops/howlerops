# Federated Query Integration Tests

This document describes the comprehensive integration test suite for federated queries, including join detection, multi-database query parsing, and DuckDB federation engine support.

## Overview

The test suite covers:

1. **Join Detection** - Testing detection and analysis of SQL joins with foreign key relationships
2. **Federated Query Parsing** - End-to-end parsing and validation of multi-database queries
3. **Strategy Selection** - Testing query execution strategy selection based on characteristics
4. **DuckDB Integration** - Testing DuckDB federation engine with build tag separation
5. **Schema Management** - Testing with realistic database schemas and FK relationships

## Test Files

### 1. `pkg/database/multiquery/testdata.go`

Shared test fixtures and schemas for all federated query tests.

**Components:**
- `TestSchemaBuilder` - Fluent API for building test schemas
- `TestTable`, `TestColumn`, `TestForeignKey` - Schema components
- `CommonTestSchemas` - Pre-built schemas:
  - `e-commerce` - E-commerce schema with FK relationships (users, products, orders, order_items)
  - `multi-db` - Schema for multi-database scenarios
  - `circular` - Schema with circular FK references (departments ↔ employees)
  - `no-fk` - Schema without any foreign keys

**Usage:**
```go
schema := CommonTestSchemas["e-commerce"]
fks := schema.GetForeignKeys("public", "order_items")
cols := schema.GetColumns("public", "products")
```

### 2. `pkg/database/multiquery/join_detector_test.go`

Tests for join detection and analysis with realistic schemas.

**Key Tests:**
- `TestJoinDetector_DetectsSimpleInnerJoin` - Basic INNER JOIN detection
- `TestJoinDetector_DetectsLeftJoin` - LEFT JOIN detection
- `TestJoinDetector_DetectsRightJoin` - RIGHT JOIN detection
- `TestJoinDetector_DetectsCrossJoin` - CROSS JOIN detection
- `TestJoinDetector_DetectsMultipleJoins` - Multiple joins in one query
- `TestJoinDetector_ForeignKeyAwareness` - FK relationship detection
- `TestJoinDetector_CircularForeignKeys` - Circular FK handling
- `TestJoinDetector_NoForeignKeySchema` - Schemas without FKs
- `TestJoinDetector_ComplexQueryAnalysis` - Complex multi-database joins
- `TestJoinDetector_TableAliasDetection` - Table alias handling
- `TestJoinDetector_AggregationWithJoins` - Aggregation detection with joins
- `TestJoinDetector_EdgeCases` - Edge cases (self-joins, special chars, etc.)

**Test Coverage:**
- 40+ join detection test cases
- FK relationship verification
- Multiple connection scenarios
- Real-world query complexity

### 3. `pkg/database/multiquery/federated_query_integration_test.go`

End-to-end federated query parsing and validation tests.

**Key Tests:**
- `TestFederatedQuery_ParseAndValidate` - Query parsing with 5 common scenarios
- `TestFederatedQuery_ExecutionStrategySelection` - Strategy selection (Auto/Federated)
- `TestFederatedQuery_MultiConnectionParsing` - Multi-connection query parsing
- `TestFederatedQuery_ValidationMaxConcurrentConnections` - Connection limit validation
- `TestFederatedQuery_SegmentationForParallelExecution` - Query segmentation for parallel execution
- `TestFederatedQuery_CrossDatabaseTypeJoin` - Cross-database-type joins (PostgreSQL + MySQL, etc.)
- `TestFederatedQuery_ComplexAggregationScenarios` - Complex aggregations and window functions
- `TestFederatedQuery_LargeScaleQuery` - 6-way database join parsing
- `TestFederatedQuery_SpecialCharacterHandling` - Hyphenated/underscored identifiers
- `TestFederatedQuery_ConfigValidation` - Configuration validation
- `BenchmarkFederatedQuery_Parsing` - Performance benchmark

**Test Coverage:**
- 30+ federated query test cases
- Table-driven tests for systematic coverage
- Strategy selection validation
- Configuration constraints

### 4. `pkg/federation/duckdb/integration_test.go`

DuckDB stub tests (runs without build tag, uses stub implementation).

**Tests:**
- `TestDuckDBStub_IsDisabledWithoutBuildTag` - Stub properly disabled
- `TestDuckDBStub_ExecuteQueryFails` - Query execution fails gracefully
- `TestDuckDBStub_IsNotInitialized` - Engine reports as not initialized
- `TestDuckDBStub_CloseIsNoOp` - Close is safe no-op
- `TestDuckDBStub_AllOperationsFail` - All operations fail consistently
- `TestDuckDB_FallbackBehavior` - Graceful fallback when unavailable
- `TestDuckDB_SafeMultipleCalls` - Multiple operations don't panic

**Coverage:**
- Verifies stub compliance with actual engine interface
- Tests graceful degradation when DuckDB unavailable
- Ensures safe multi-call behavior

### 5. `pkg/federation/duckdb/integration_duckdb_test.go`

Actual DuckDB engine tests (runs WITH `duckdb` build tag).

**Tests:**
- `TestDuckDBEngine_Initialize` - Engine initialization
- `TestDuckDBEngine_SimpleQuery` - SELECT 1 query execution
- `TestDuckDBEngine_ArrowResult` - Result structure and Arrow conversion
- `TestDuckDBEngine_ComplexQuery` - CTE and complex query execution
- `TestDuckDBEngine_Timeout` - Query timeout handling
- `TestDuckDBEngine_Close` - Engine cleanup
- `TestDuckDBCompiler_CompileSimpleSelect` - Query compilation
- `TestDuckDB_LoadExtensions` - DuckDB extension loading
- `BenchmarkDuckDBEngine_SimpleQuery` - Simple query performance
- `BenchmarkDuckDBEngine_ComplexQuery` - Complex query performance

**Coverage:**
- Only runs if DuckDB is available
- Tests actual DuckDB integration
- Performance benchmarks
- Extension loading verification

**Build Tag:**
```go
//go:build duckdb
```

Tests skip gracefully if DuckDB unavailable:
```go
if err != nil {
    skip.Skipf("DuckDB initialization failed: %v", err)
}
```

## Running the Tests

### All tests (without DuckDB):
```bash
go test -v ./pkg/database/multiquery/...
go test -v ./pkg/federation/duckdb/...
```

### Run specific test:
```bash
go test -v ./pkg/database/multiquery -run "TestJoinDetector_Detects"
```

### With DuckDB enabled (if binary available):
```bash
go test -tags duckdb -v ./pkg/federation/duckdb/...
```

### Run benchmarks:
```bash
go test -bench=. ./pkg/database/multiquery/...
go test -tags duckdb -bench=. ./pkg/federation/duckdb/...
```

### Coverage:
```bash
go test -cover ./pkg/database/multiquery/...
go test -cover ./pkg/federation/duckdb/...
```

## Test Statistics

### Total Test Cases: 80+
- Join detection: 40 tests
- Federated queries: 25+ tests
- DuckDB integration: 10+ tests
- DuckDB actual (with tag): 10+ tests

### Coverage Areas:
- Simple and complex joins
- Multi-database scenarios
- Foreign key relationships
- Aggregations and window functions
- Edge cases and error handling
- Configuration validation
- Performance characteristics
- Build tag conditional compilation

## Design Principles

### 1. Table-Driven Tests
Tests use table-driven patterns for systematic coverage:
```go
tests := []struct {
    name string
    query string
    expected bool
}{
    // ... test cases
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // ... assertions
    })
}
```

### 2. Realistic Schemas
Test schemas mirror real-world database structures:
- Foreign key relationships
- Multi-table joins
- Circular references
- Mixed cardinalities

### 3. No External Dependencies
- No actual database connections required
- All tests use mocks and fixtures
- Safe for CI/CD environments

### 4. Build Tag Separation
- DuckDB tests separated by build tag
- Stub implementation ensures tests pass without DuckDB
- Optional DuckDB tests run when binary available

### 5. Clear Failure Messages
Tests include descriptive assertions:
```go
assert.True(t, parsed.HasJoins, "Should detect INNER JOIN")
```

## CI/CD Integration

All tests pass in CI without DuckDB:
```yaml
test:
  script:
    - go test -v ./pkg/database/multiquery/...
    - go test -v ./pkg/federation/duckdb/...
```

Optional DuckDB tests only run when available:
```yaml
test-duckdb:
  script:
    - go test -tags duckdb -v ./pkg/federation/duckdb/...
  allow_failure: true
```

## Adding New Tests

### For Join Detection:
1. Add scenario to `CommonTestSchemas` or inline
2. Create test function in `join_detector_test.go`
3. Use table-driven pattern
4. Test both parsing and FK awareness

### For Federated Queries:
1. Add scenario to `CommonQueryScenarios` in testdata
2. Create integration test in `federated_query_integration_test.go`
3. Test parsing, validation, strategy selection
4. Use realistic schemas

### For DuckDB:
1. For general tests: use `integration_test.go` (always runs)
2. For DuckDB-specific: use `integration_duckdb_test.go` (with tag)
3. Use skip for optional features
4. Provide helpful error messages

## Key Insights

1. **FK Detection** - Tests verify that joins on FK columns are properly recognized
2. **Multi-DB Awareness** - Parser correctly identifies required connections
3. **Strategy Selection** - Complex queries correctly suggest federated strategy
4. **Graceful Degradation** - System safely handles missing DuckDB
5. **Edge Case Handling** - Circular FKs, self-joins, special characters handled
6. **Performance** - Benchmarks validate parsing performance

## Future Enhancements

Potential additions:
- Test actual data federation (with mock DB results)
- Test query result merging
- Test cross-database type coercion
- Stress tests with very large schemas
- Transaction handling tests
- Failure recovery tests
