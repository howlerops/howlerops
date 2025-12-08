# Data Catalog Package

This package implements the Data Catalog feature for HowlerOps, providing metadata management for database tables and columns.

## Overview

The Data Catalog enables teams to:
- Document tables and columns with descriptions
- Assign data stewards/owners
- Tag data with classifications (PII, Sensitive, Internal, Public, Deprecated)
- Search across all catalog metadata using full-text search

## Architecture

The package follows a clean layered architecture:

```
types.go    - Domain models and constants
store.go    - SQLite repository (data access layer)
search.go   - Full-text search implementation (FTS5 with LIKE fallback)
service.go  - Business logic layer
```

## Components

### Types

- **TableCatalogEntry**: Metadata for database tables
- **ColumnCatalogEntry**: Metadata for table columns
- **CatalogTag**: Reusable tags (system and custom)
- **SearchResult**: Search result items
- **SearchFilters**: Query filtering options

### Store Interface

The Store interface provides CRUD operations for:
- Table entries (Create, Get, Update, Delete, List)
- Column entries (Create, Get, Update, List)
- Tags (Create, List, Delete)
- Search functionality

### Service Layer

The Service wraps the store with business logic:
- Input validation
- Tag validation
- PII detection import

## System Tags

Five predefined system tags are automatically created:

| Tag        | Color   | Description                     |
|------------|---------|--------------------------------|
| PII        | Red     | Personally Identifiable Information |
| Sensitive  | Orange  | Sensitive business data         |
| Internal   | Yellow  | Internal use only              |
| Public     | Green   | Safe for public access         |
| Deprecated | Gray    | Scheduled for removal          |

## Search Implementation

The package supports two search modes:

1. **FTS5 (Full-Text Search)**: High-performance search using SQLite's FTS5 extension
   - BM25 relevance scoring
   - Porter stemming for fuzzy matching
   - Prefix matching

2. **LIKE-based Fallback**: Used when FTS5 is not available
   - Pattern matching with relevance scoring
   - Works with all SQLite builds
   - Supports all the same filters

Search automatically detects which mode to use based on SQLite capabilities.

## Usage Example

```go
package main

import (
    "context"
    "database/sql"

    "github.com/jbeck018/howlerops/backend-go/internal/catalog"
    _ "github.com/mattn/go-sqlite3"
)

func main() {
    // Open database
    db, _ := sql.Open("sqlite3", "catalog.db")
    defer db.Close()

    // Create store and service
    store := catalog.NewStore(db)
    service := catalog.NewService(store)

    // Initialize (creates tables)
    ctx := context.Background()
    service.Initialize(ctx)

    // Create a table entry
    entry := &catalog.TableCatalogEntry{
        ConnectionID: "conn-123",
        SchemaName:   "public",
        TableName:    "customers",
        Description:  "Customer data including contact information",
        Tags:         []string{catalog.TagPII, catalog.TagSensitive},
        CreatedBy:    "user-1",
    }
    service.CreateTableEntry(ctx, entry)

    // Search catalog
    results, _ := service.SearchCatalog(ctx, "customer email", &catalog.SearchFilters{
        Tags:  []string{catalog.TagPII},
        Limit: 10,
    })

    // Results contain matching tables and columns
    for _, result := range results.Results {
        println(result.Type, result.TableName, result.ColumnName)
    }
}
```

## Database Schema

### catalog_table_entries
- Stores table-level metadata
- Unique constraint on (connection_id, schema_name, table_name)
- Indexed by connection, organization, steward, and update time

### catalog_column_entries
- Stores column-level metadata
- Links to parent table via foreign key
- Supports PII type and confidence tracking
- Unique constraint on (table_catalog_id, column_name)

### catalog_tags
- Stores both system and custom tags
- System tags cannot be deleted
- Organization-scoped custom tags

### catalog_fts (optional)
- FTS5 virtual table for full-text search
- Created automatically if FTS5 is available
- Indexes table names, column names, descriptions, and tags

## Integration with PII Detection

The service includes `ImportPIIDetections()` method to automatically populate catalog entries from PII detection results:

```go
detections := []catalog.PIIDetection{
    {
        SchemaName: "public",
        TableName:  "users",
        ColumnName: "email",
        PIIType:    "email",
        Confidence: 0.95,
    },
}

service.ImportPIIDetections(ctx, "conn-123", detections)
```

This automatically:
- Creates table entries if they don't exist
- Creates/updates column entries
- Adds PII tags to columns
- Records confidence scores

## Testing

Run tests with:
```bash
go test ./internal/catalog/ -v
```

All tests use in-memory SQLite databases and cover:
- CRUD operations for tables, columns, and tags
- Search functionality (both FTS5 and LIKE modes)
- PII integration
- Tag validation

## Future Enhancements

Potential additions for future versions:
- Data lineage tracking
- AI-generated descriptions
- Change history/versioning
- Custom metadata fields
- Quality scoring
