# Schema Enrichment Usage Guide

## Overview

Schema enrichment adds runtime statistics, sample values, and query patterns to schema documents, improving RAG context quality for natural language queries.

## Components

### 1. SchemaEnricher

Collects runtime statistics from database columns:

```go
import (
    "github.com/jbeck018/howlerops/backend-go/internal/rag"
)

// Create enricher with database connection
enricher := rag.NewSchemaEnricher(db, logger)

// Enrich a single column
stats, err := enricher.EnrichColumn(ctx, "public", "users", "status", "varchar")

// Stats include:
// - Distinct count
// - Null count
// - Sample values (for categorical)
// - Min/max/avg (for numeric)
// - Top values with frequencies (for categorical)
```

### 2. QueryPatternTracker

Tracks successful queries to learn common patterns:

```go
tracker := rag.NewQueryPatternTracker(vectorStore, logger)

// Track a successful query
err := tracker.TrackQuery(
    ctx,
    "SELECT * FROM users WHERE status = 'active' AND age > 18",
    0.025, // duration in seconds
    "conn-123",
)

// Creates indexed document with:
// - Normalized pattern (values replaced with placeholders)
// - Tables used
// - WHERE columns
// - JOIN columns
// - Query description
```

### 3. Enhanced Schema Indexing

Use enrichment during schema indexing:

```go
// Create indexer with enrichment
enricher := rag.NewSchemaEnricher(db, logger)
indexer := rag.NewSchemaIndexer(vectorStore, embeddingService, logger).
    WithEnricher(enricher)

// Index table with enrichment
err := indexer.IndexTableDetails(ctx, connID, schema, table, structure)

// Column documents now include:
// - "examples: active, inactive, pending" in content
// - "range: 18 to 95" in content (for numeric)
// - "distinct_values: 3" in content
// - Full statistics in metadata
```

## Example Enriched Documents

### Categorical Column (status)

**Before enrichment:**
```
Content: "column public.users.status type varchar"
Metadata: {
  "column": "status",
  "data_type": "varchar"
}
```

**After enrichment:**
```
Content: "column public.users.status type varchar examples: active, inactive, pending distinct_values: 3"
Metadata: {
  "column": "status",
  "data_type": "varchar",
  "distinct_count": 3,
  "null_count": 0,
  "sample_values": ["active", "inactive", "pending"],
  "top_values": {
    "active": 700,
    "inactive": 200,
    "pending": 100
  }
}
```

### Numeric Column (age)

**Before enrichment:**
```
Content: "column public.users.age type integer"
Metadata: {
  "column": "age",
  "data_type": "integer"
}
```

**After enrichment:**
```
Content: "column public.users.age type integer range: 18 to 95 distinct_values: 80"
Metadata: {
  "column": "age",
  "data_type": "integer",
  "distinct_count": 80,
  "null_count": 10,
  "min_value": 18,
  "max_value": 95,
  "avg_value": 42.5
}
```

### Query Pattern

**Tracked query:**
```sql
SELECT id, name FROM users WHERE status = 'active' AND age > 18
```

**Indexed as:**
```
Content: "Query pattern: Filtered query\nTables: users\nFilters: status, age"
Metadata: {
  "pattern": "SELECT id, name FROM users WHERE status = '?' AND age > ?",
  "tables": ["users"],
  "where_columns": ["status", "age"],
  "avg_duration": 0.025,
  "frequency": 1,
  "last_used": "2025-01-22T10:30:00Z"
}
```

## Benefits for RAG

### 1. Better Context for Value Queries

**User:** "Show me active users"

**Without enrichment:** RAG must guess what "active" means
**With enrichment:** RAG knows `status` column contains "active" value

### 2. Numeric Range Understanding

**User:** "Find adult users"

**Without enrichment:** RAG doesn't know valid age range
**With enrichment:** RAG sees age ranges from 18-95, understands "adult" context

### 3. Query Pattern Learning

**User:** "Users by status"

**Without enrichment:** No prior patterns to reference
**With enrichment:** RAG finds similar pattern "SELECT * FROM users WHERE status = ?"

### 4. Cardinality Hints

**User:** "Group by status"

**Without enrichment:** Unknown if status is suitable for grouping
**With enrichment:** RAG sees only 3 distinct values, perfect for GROUP BY

## Performance Considerations

### Enrichment Cost

- **Categorical columns** (< 50 distinct): ~3 queries (COUNT DISTINCT, NULL count, TOP 10)
- **Numeric columns**: ~3 queries (COUNT DISTINCT, NULL count, MIN/MAX/AVG)
- **Text columns** (high cardinality): ~3 queries (COUNT DISTINCT, NULL count, 5 samples)

### Optimization Strategies

1. **Limit enrichment to important tables** (most queried first)
2. **Cache enrichment results** (refresh periodically)
3. **Enrich incrementally** (start with columns used in WHERE clauses)
4. **Skip large tables** if row count > threshold

### Example: Selective Enrichment

```go
// Only enrich tables with < 10M rows
if table.RowCount < 10_000_000 {
    indexer = indexer.WithEnricher(enricher)
} else {
    // Skip enrichment for very large tables
    logger.Info("Skipping enrichment for large table", "table", table.Name)
}
```

## Testing

All enrichment features are fully tested:

```bash
# Run schema enrichment tests
go test -v -run TestSchemaEnricher ./internal/rag/schema_enrichment_test.go ./internal/rag/schema_enrichment.go

# Run query pattern tracker tests
go test -v -run TestQueryPatternTracker ./internal/rag/

# Results:
# ✅ Categorical column enrichment
# ✅ Numeric column enrichment
# ✅ Text column enrichment
# ✅ Type detection (categorical vs numeric)
# ✅ Partial failure handling
# ✅ Query pattern extraction
# ✅ Pattern normalization
```

## Integration Checklist

- [x] SchemaEnricher collects column statistics
- [x] QueryPatternTracker records query patterns
- [x] SchemaIndexer supports enrichment via WithEnricher()
- [x] Enriched data embedded in document content
- [x] Statistics stored in metadata
- [x] Tests verify all functionality
- [x] Graceful degradation on errors
- [x] Documentation and examples

## Future Enhancements

1. **Incremental updates** - Only re-enrich changed columns
2. **Smart sampling** - Adaptive sample sizes based on cardinality
3. **Correlation detection** - Find related columns automatically
4. **Usage-based prioritization** - Enrich frequently queried columns first
5. **Distributed caching** - Share enrichment across instances
