# Elasticsearch and OpenSearch Database Connector Implementation

## Overview

This document describes the implementation of the Elasticsearch and OpenSearch database connectors for HowlerOps. Both search engines are supported through a single connector implementation, as they share compatible APIs.

## Implementation Summary

### Files Created

1. **`/pkg/database/elasticsearch.go`** (main implementation)
   - `ElasticsearchDatabase` struct - HTTP-based database connector
   - Implements all methods of the `Database` interface
   - Uses Elasticsearch `_sql` API for query execution
   - Supports both Elasticsearch and OpenSearch

2. **`/pkg/database/elasticsearch_test.go`** (unit tests)
   - Tests for core functionality
   - Tests for utility functions (base64 encoding, size parsing, etc.)
   - Validates data type mappings and identifier quoting

3. **`/pkg/database/examples/elasticsearch_example.go`** (usage examples)
   - Complete example demonstrating all connector features
   - Separate example for OpenSearch usage

### Files Modified

1. **`/pkg/database/manager.go`**
   - Updated `createDatabaseInstance()` to support Elasticsearch/OpenSearch
   - Updated `CreateDatabase()` in Factory
   - Updated `ValidateConfig()` to handle Elasticsearch configuration
   - Updated `GetDefaultConfig()` with Elasticsearch defaults

2. **`/README.md`**
   - Added Elasticsearch/OpenSearch to supported databases list
   - Added configuration examples for both databases

## Key Design Decisions

### 1. HTTP-Based Implementation

Unlike SQL databases that use the `database/sql` package and connection pools, Elasticsearch uses HTTP REST APIs. Key implementation details:

- Uses `net/http` client instead of SQL drivers
- Custom connection statistics tracking (no built-in pool)
- Timeout and TLS configuration managed directly
- Reuses HTTP connections through connection pooling in the HTTP transport

### 2. Authentication Support

The connector supports multiple authentication methods:

- **Basic Authentication**: Username/password (standard)
- **API Key Authentication**: Via `api_key` parameter (more secure)
- **No Authentication**: For development/testing

```go
// Basic auth
config.Username = "elastic"
config.Password = "password"

// API key auth (alternative)
config.Parameters["api_key"] = "base64-encoded-key"
```

### 3. SQL API Integration

Elasticsearch/OpenSearch provide a `_sql` API that allows SQL queries on indices:

- **Query Endpoint**: `POST /_sql?format=json`
- **Translation**: `POST /_sql/translate` (for EXPLAIN queries)
- **Cursor Support**: Pagination for large result sets

This allows seamless integration with HowlerOps' SQL-focused interface.

### 4. Schema Mapping

Elasticsearch doesn't have traditional databases/schemas, so we map concepts:

- **Schemas**: Return a single "default" schema
- **Tables**: Map to Elasticsearch indices
- **Columns**: Map to index field mappings
- **Rows**: Map to documents

### 5. Non-Editable Results

Elasticsearch indices are not directly editable through SQL updates. The connector:

- Always returns `Enabled: false` for editable metadata
- Provides clear reason: "Elasticsearch indices are not directly editable"
- Users must use Elasticsearch's native Update/Delete APIs for modifications

### 6. URL Format and Connection

Connection URL format: `http(s)://host:port`

- Default port: 9200
- HTTP/HTTPS determined by `ssl_mode` parameter
- TLS certificate verification configurable

### 7. Streaming Support

Large result sets are handled via Elasticsearch's cursor mechanism:

```go
// Initial query with fetch_size
POST /_sql?format=json
{
  "query": "SELECT * FROM index",
  "fetch_size": 1000
}

// Subsequent fetches
POST /_sql?format=json
{
  "cursor": "cursor-id-from-previous-response"
}
```

## Interface Implementation

All `Database` interface methods are implemented:

### Connection Management
- ✅ `Connect()` - Establishes HTTP client and tests connection
- ✅ `Disconnect()` - Closes idle HTTP connections
- ✅ `Ping()` - Tests connection by calling root endpoint
- ✅ `GetConnectionInfo()` - Returns cluster info from root endpoint

### Query Execution
- ✅ `Execute()` - Executes SQL via `_sql` API
- ✅ `ExecuteStream()` - Streams results using cursor pagination
- ✅ `ExplainQuery()` - Translates SQL to Query DSL
- ✅ `ComputeEditableMetadata()` - Returns non-editable metadata

### Schema Operations
- ✅ `GetSchemas()` - Returns ["default"]
- ✅ `GetTables()` - Lists indices via `_cat/indices`
- ✅ `GetTableStructure()` - Retrieves index mappings

### Transaction Management
- ✅ `BeginTransaction()` - Returns error (not supported)

### Data Modification
- ✅ `UpdateRow()` - Returns error (not supported via SQL)

### Utility Methods
- ✅ `GetDatabaseType()` - Returns `Elasticsearch` or `OpenSearch`
- ✅ `GetConnectionStats()` - Returns basic stats
- ✅ `QuoteIdentifier()` - Uses backticks (SQL standard)
- ✅ `GetDataTypeMappings()` - Returns ES data type mappings

## Data Type Mappings

Elasticsearch field types mapped to common types:

| Common Type | Elasticsearch Type |
|-------------|-------------------|
| string      | text              |
| keyword     | keyword           |
| int         | integer           |
| int64       | long              |
| float       | float             |
| float64     | double            |
| bool        | boolean           |
| time/date   | date              |
| json        | object            |
| geo         | geo_point         |
| binary      | binary            |
| ip          | ip                |

## Error Handling

The connector handles several error scenarios:

1. **Connection Errors**: HTTP timeout, unreachable host
2. **Authentication Errors**: Invalid credentials, missing API key
3. **Query Errors**: Invalid SQL, syntax errors
4. **Index Errors**: Non-existent index, mapping errors

All errors are properly wrapped with context for debugging.

## SSH Tunnel Support

Elasticsearch connector works with existing SSH tunnel infrastructure:

- Connection pool manager handles tunnel creation
- HTTP requests routed through localhost tunnel endpoint
- No changes needed to connector implementation

## Performance Considerations

1. **Connection Pooling**: HTTP client maintains connection pool automatically
2. **Streaming**: Large result sets use cursor-based pagination
3. **Batch Size**: Configurable `fetch_size` for streaming queries
4. **Timeout**: Configurable connection and query timeouts

## Usage Example

```go
// Create configuration
config := database.ConnectionConfig{
    Type:     database.Elasticsearch,
    Host:     "localhost",
    Port:     9200,
    Database: "my-cluster",
    Username: "elastic",
    Password: "password",
    SSLMode:  "disable",
}

// Create connector
es, err := database.NewElasticsearchDatabase(config, logger)
if err != nil {
    log.Fatal(err)
}
defer es.Disconnect()

// Execute SQL query
result, err := es.Execute(ctx, "SELECT * FROM my-index LIMIT 10")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Returned %d rows in %v\n", result.RowCount, result.Duration)
```

## Testing

Comprehensive unit tests cover:

- Database type identification
- Identifier quoting
- Data type mappings
- Size string parsing (kb, mb, gb, tb)
- Basic authentication encoding
- Editable metadata generation

All tests pass without requiring a running Elasticsearch instance.

## Future Enhancements

Potential improvements for future versions:

1. **Query DSL Support**: Direct Query DSL execution (bypass SQL)
2. **Bulk Operations**: Bulk insert/update support
3. **Index Creation**: DDL support for creating indices
4. **Aggregations**: Better support for complex aggregations
5. **Multi-Index Queries**: Improved handling of index patterns
6. **Authentication**: Support for more auth methods (SAML, OIDC)

## Compatibility

- **Elasticsearch**: Version 7.x and 8.x (SQL API)
- **OpenSearch**: Version 1.x and 2.x (SQL plugin)
- **Protocols**: HTTP/1.1, HTTPS
- **Authentication**: Basic Auth, API Key

## References

- [Elasticsearch SQL API](https://www.elastic.co/guide/en/elasticsearch/reference/current/sql-apis.html)
- [OpenSearch SQL](https://opensearch.org/docs/latest/search-plugins/sql/)
- [Elasticsearch Mapping](https://www.elastic.co/guide/en/elasticsearch/reference/current/mapping.html)
