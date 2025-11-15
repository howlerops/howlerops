<!-- 13ef2ff0-5b8a-4768-9f08-2e64066ab335 7e577699-449b-4b0e-8ef8-c85999e68864 -->
# Database Extension Master Plan

## Overview

Extend Howlerops from 9 to 20 supported databases (excluding Oracle due to licensing), achieving feature parity with Beekeeper Studio while maintaining our superior SSH tunnel and VPC networking capabilities.

## Database Categories & Priority

### Tier 1: Enterprise & Cloud Data Warehouses (High Business Value)

1. **SQL Server** - Microsoft enterprise standard, Azure integration
2. **Amazon Redshift** - AWS data warehouse (PostgreSQL wire protocol)
3. **Google BigQuery** - GCP data warehouse, serverless analytics
4. **Snowflake** - Multi-cloud data warehouse

### Tier 2: Modern Distributed & Analytics Databases

5. **CockroachDB** - Distributed SQL (PostgreSQL compatible)
6. **DuckDB** - Embedded analytics database
7. **Trino/Presto** - Distributed SQL query engine

### Tier 3: Specialized Databases

8. **Redis** - In-memory key-value store with SQL-like commands
9. **Cassandra** - NoSQL wide-column store
10. **LibSQL** - Turso's SQLite fork with sync capabilities
11. **Firebird** - Open-source RDBMS

## Implementation Strategy

Each database requires changes across 6 layers:

### Layer 1: Protocol Definition (`proto/database.proto`)

- Add database type enum constant
- Ensure compatibility with existing RPC methods

### Layer 2: Backend Type System (`backend-go/pkg/database/types.go`)

- Add DatabaseType constant
- Add to factory method switches

### Layer 3: Backend Driver Implementation (`backend-go/pkg/database/<db>.go`)

- Implement Database interface (15 methods)
- Connection pooling with SSH tunnel support
- Schema introspection queries
- Query execution with streaming
- Editable metadata computation
- Connection health checks

### Layer 4: Connection Pool & DSN (`backend-go/pkg/database/pool.go`)

- Add DSN builder method
- Configure driver-specific parameters
- Handle SSL/TLS modes
- Default ports and settings

### Layer 5: Frontend Type System (`frontend/src/types/storage.ts`)

- Add to DatabaseType union
- Update connection store types

### Layer 6: Frontend UI (`frontend/src/components/connection-manager.tsx`)

- Add database option to selector
- Configure default ports
- Add database-specific configuration fields
- Update validation logic

## Feature Parity Checklist (Per Database)

### Core Features

- [ ] Basic connectivity (host, port, database, credentials)
- [ ] SSL/TLS configuration (disable, prefer, require, verify-ca, verify-full)
- [ ] SSH tunnel support (password & private key auth)
- [ ] VPC/Private Link configuration
- [ ] Connection pooling with configurable limits
- [ ] Connection timeout and keepalive settings

### Query Operations

- [ ] Execute queries with streaming support
- [ ] Query explain/analyze
- [ ] Transaction support (begin, commit, rollback)
- [ ] Editable result sets (where applicable)
- [ ] Batch size configuration for large results

### Schema Operations

- [ ] List schemas/databases
- [ ] List tables with metadata (row count, size, created/updated times)
- [ ] Get table structure (columns, types, constraints)
- [ ] List indexes with details
- [ ] List foreign keys
- [ ] Quote identifier handling

### Monitoring

- [ ] Connection health checks
- [ ] Connection pool statistics
- [ ] Query performance metrics

## Go Driver Dependencies

Add to `go.mod`:

```go
github.com/denisenkom/go-mssqldb v0.12.3          // SQL Server
github.com/go-redis/redis/v8 v8.11.5              // Redis
github.com/gocql/gocql v1.6.0                     // Cassandra
github.com/jackc/pgx/v5 v5.5.0                    // Redshift (PostgreSQL wire)
github.com/marcboeker/go-duckdb v1.5.6            // DuckDB
github.com/nakagami/firebirdsql v0.9.10           // Firebird
github.com/prestodb/presto-go-client v0.0.0       // Presto/Trino
github.com/snowflakedb/gosnowflake v1.7.1         // Snowflake
github.com/tursodatabase/libsql-client-go v0.0.0  // LibSQL
cloud.google.com/go/bigquery v1.57.1              // BigQuery
```

## Testing Requirements

For each database:

- Unit tests for connection, query, schema operations
- Integration tests with real database instances (Docker)
- SSH tunnel connection tests
- Error handling and reconnection tests
- Concurrent connection tests

## Documentation Requirements

For each database:

- Update `docs/DATABASE_CONNECTORS.md` with connection examples
- Add to `docs/user-guides/FAQ.md` supported database list
- Create `docs/databases/<database>-guide.md` with:
  - Connection string format
  - SSL/TLS configuration
  - SSH tunnel examples
  - Common troubleshooting
  - Version compatibility matrix

## Validation Steps

Before marking each database complete:

1. Run `go test ./backend-go/pkg/database/<db>_test.go -v`
2. Run `make proto` to regenerate bindings
3. Run `cd frontend && npm run typecheck`
4. Run `cd frontend && npm run lint`
5. Test connection in UI with real database
6. Test SSH tunnel connection
7. Verify schema browsing works
8. Execute sample queries
9. Test connection pool statistics
10. Update and verify documentation

## Success Metrics

- **Coverage**: 20 databases supported (vs Beekeeper's 19)
- **Feature Parity**: 100% feature match across all databases
- **Quality**: ≥80% test coverage for all new code
- **Documentation**: Complete setup guide for each database
- **Performance**: Connection pooling efficiency matches existing databases

### To-dos

- [ ] Implement SQL Server support with full feature parity
- [ ] Implement Amazon Redshift support (PostgreSQL wire protocol)
- [ ] Implement Google BigQuery support with OAuth and service account auth
- [ ] Implement Snowflake support with multi-cloud capabilities
- [ ] Implement CockroachDB support (PostgreSQL compatible)
- [ ] Implement DuckDB embedded analytics support
- [ ] Implement Trino/Presto distributed query engine support
- [ ] Implement Redis in-memory database support
- [ ] Implement Apache Cassandra NoSQL support
- [ ] Implement LibSQL (Turso) support with sync capabilities
- [ ] Implement Firebird RDBMS support
- [ ] Complete integration testing and validation for all new databases
- [ ] Update all documentation with new database support details