# TypeScript Type Definitions

This directory contains comprehensive TypeScript type definitions for Howlerops features.

## Type Organization

### Core Types
- **`table.ts`** - Core table and query result types
- **`storage.ts`** - IndexedDB and local storage types
- **`ai.ts`** - AI provider and query types
- **`reports.ts`** - Report builder and visualization types
- **`websocket.ts`** - Real-time WebSocket communication

### Feature-Specific Types
- **`schema-diff.ts`** - Schema versioning, comparison, and migration
- **`catalog.ts`** - Data governance, PII detection, and cataloging
- **`organization.ts`** - Multi-tenant organization management
- **`sync.ts`** - Offline-first synchronization

### Utility Types
- **`branded.ts`** - Type-safe branded types (IDs, etc.)
- **`keyboard-shortcuts.ts`** - Keyboard binding configuration
- **`templates.ts`** - Template management
- **`tiers.ts`** - Feature tier management

## Schema Diff Types (`schema-diff.ts`)

Type-safe definitions for schema versioning and comparison features.

### Key Types

#### Snapshots
```typescript
import { SchemaSnapshot, SnapshotMetadata, createSnapshotId } from '@/types/schema-diff'

// Create a snapshot
const snapshotId = createSnapshotId('snap_123')

// Work with snapshot metadata
const metadata: SnapshotMetadata = {
  id: snapshotId,
  name: 'Pre-migration baseline',
  connectionId: 'conn_abc',
  databaseType: 'postgres',
  tableCount: 42,
  createdAt: new Date(),
  sizeBytes: 1024000
}
```

#### Schema Comparison
```typescript
import { SchemaDiff, DiffSummary, TableDiff } from '@/types/schema-diff'

// Compare two schemas
const diff: SchemaDiff = await CompareConnectionSchemas(sourceId, targetId)

// Access summary stats
console.log(`Tables added: ${diff.summary.tablesAdded}`)
console.log(`Columns modified: ${diff.summary.columnsModified}`)

// Examine table-level changes
diff.tables.forEach((tableDiff: TableDiff) => {
  if (tableDiff.status === 'modified') {
    console.log(`Table ${tableDiff.name} has changes:`)
    tableDiff.columns?.forEach(col => {
      if (col.status === 'modified') {
        console.log(`  - ${col.name}: ${col.oldType} → ${col.newType}`)
      }
    })
  }
})
```

#### Migration Scripts
```typescript
import { MigrationScript, MigrationPreview } from '@/types/schema-diff'

// Generate migration SQL
const script: MigrationScript = await GenerateMigrationSQL(
  sourceId,
  targetId,
  false // include destructive operations
)

// Preview migration with impact assessment
const preview: MigrationPreview = {
  script,
  warnings: [
    {
      severity: 'warning',
      objectType: 'column',
      objectName: 'users.email',
      message: 'Column will be dropped - data loss will occur',
      isDestructive: true
    }
  ],
  impact: {
    tablesAffected: 5,
    estimatedRowsAffected: 1500,
    isReversible: false,
    requiresDowntime: true,
    riskLevel: 'high'
  }
}
```

### Wails Integration

These types align with the Wails bindings in `wailsjs/go/models.ts`:

```typescript
// Wails class-based type
import { schemadiff } from '../wailsjs/go/models'
const wailsSnapshot = new schemadiff.SchemaSnapshot(data)

// Clean TypeScript interface
import { SchemaSnapshot } from '@/types/schema-diff'
const snapshot: SchemaSnapshot = data // No class instantiation needed
```

## Data Catalog Types (`catalog.ts`)

Type-safe definitions for data governance and cataloging features.

### Key Types

#### Table Cataloging
```typescript
import {
  TableCatalogEntry,
  CreateTableCatalogInput,
  createTableCatalogId
} from '@/types/catalog'

// Create a catalog entry
const input: CreateTableCatalogInput = {
  connectionId: 'conn_abc',
  schemaName: 'public',
  tableName: 'users',
  description: 'Application user accounts and profile data',
  tags: ['auth', 'pii', 'critical']
}

await CreateTableCatalogEntry(input)

// Work with catalog entries
const entry: TableCatalogEntry = await GetTableCatalogEntry(
  'conn_abc',
  'public',
  'users'
)

console.log(`Steward: ${entry.stewardUserId}`)
console.log(`Tags: ${entry.tags?.join(', ')}`)
```

#### PII Detection
```typescript
import { PIIType, PIIConfidence, ColumnCatalogEntry } from '@/types/catalog'

// Mark column as PII
await MarkColumnAsPII(
  tableId,
  'email',
  'email' as PIIType,
  0.95 // confidence score
)

// Check PII classifications
const columns = await ListColumnCatalogEntries(tableId)
const piiColumns = columns.filter(col => col.piiType)

piiColumns.forEach(col => {
  console.log(`${col.columnName}: ${col.piiType} (${col.piiConfidence})`)
})
```

#### Catalog Search
```typescript
import { SearchFilters, SearchResults } from '@/types/catalog'

// Search for tables and columns
const filters: SearchFilters = {
  connectionId: 'conn_abc',
  schemaPattern: 'public',
  tags: ['pii'],
  hasPII: true
}

const results: SearchResults = await SearchCatalog('user', filters)

results.results.forEach(result => {
  console.log(`${result.type}: ${result.tableName}.${result.columnName}`)
  console.log(`  Score: ${result.relevanceScore}`)
})
```

#### Tags
```typescript
import { CatalogTag, CreateTagInput, createCatalogTagId } from '@/types/catalog'

// Create a custom tag
const tagInput: CreateTagInput = {
  name: 'Sensitive',
  color: '#FF6B6B',
  description: 'Contains sensitive business data'
}

await CreateCatalogTag(tagInput)

// List all tags
const tags: CatalogTag[] = await ListCatalogTags({})
const systemTags = tags.filter(tag => tag.isSystem)
const customTags = tags.filter(tag => !tag.isSystem)
```

#### Statistics
```typescript
import { CatalogStats, PIIStats } from '@/types/catalog'

// Get catalog coverage stats
const stats: CatalogStats = await GetCatalogStats('conn_abc')

console.log(`Coverage: ${stats.coveragePercentage}%`)
console.log(`PII Coverage: ${stats.piiCoveragePercentage}%`)
console.log(`Stewarded: ${stats.stewardedTables}/${stats.totalTables}`)
```

### Wails Integration

```typescript
// Wails class-based type
import { catalog } from '../wailsjs/go/models'
const wailsEntry = new catalog.TableCatalogEntry(data)

// Clean TypeScript interface
import { TableCatalogEntry } from '@/types/catalog'
const entry: TableCatalogEntry = data // Direct usage
```

## Type Safety Features

### Branded Types
Both modules use branded types for compile-time ID safety:

```typescript
// These are incompatible at compile time
const snapshotId: SnapshotId = createSnapshotId('snap_123')
const catalogId: TableCatalogId = createTableCatalogId('tbl_456')

// ❌ Type error - cannot assign SnapshotId to TableCatalogId
const wrong: TableCatalogId = snapshotId

// ✅ Type-safe - IDs cannot be confused
const snapshot = getSnapshot(snapshotId)
const catalog = getCatalog(catalogId)
```

### Exhaustive Type Checking
Discriminated unions enable exhaustive switch statements:

```typescript
import { DiffStatus } from '@/types/schema-diff'

function handleDiffStatus(status: DiffStatus) {
  switch (status) {
    case 'added':
      return 'New item'
    case 'removed':
      return 'Deleted item'
    case 'modified':
      return 'Changed item'
    case 'unchanged':
      return 'No changes'
    // TypeScript enforces all cases are handled
  }
}
```

### Optional Chaining Safety
All optional fields are properly typed:

```typescript
import { TableDiff } from '@/types/schema-diff'

function processTableDiff(diff: TableDiff) {
  // ✅ Safe optional chaining
  const columnCount = diff.columns?.length ?? 0
  const indexCount = diff.indexes?.length ?? 0

  // ✅ Type-safe array operations
  diff.columns?.forEach(col => {
    console.log(col.name) // col is typed as ColumnDiff
  })
}
```

## Best Practices

### 1. Use Branded Types for IDs
```typescript
// ❌ Avoid plain strings for IDs
function getSnapshot(id: string) { ... }

// ✅ Use branded types
function getSnapshot(id: SnapshotId) { ... }
```

### 2. Prefer Interfaces Over Wails Classes
```typescript
// ❌ Avoid Wails class constructors in UI code
const entry = new catalog.TableCatalogEntry(data)

// ✅ Use TypeScript interfaces directly
const entry: TableCatalogEntry = data
```

### 3. Leverage Type Inference
```typescript
// ❌ Redundant type annotations
const filters: SearchFilters = {
  connectionId: 'conn_abc' as string,
  hasPII: true as boolean
}

// ✅ Let TypeScript infer
const filters: SearchFilters = {
  connectionId: 'conn_abc',
  hasPII: true
}
```

### 4. Use Discriminated Unions
```typescript
// ✅ Type-safe result handling
type SearchResult = {
  type: 'table'
  tableName: string
} | {
  type: 'column'
  tableName: string
  columnName: string
}

function displayResult(result: SearchResult) {
  if (result.type === 'table') {
    // TypeScript knows columnName doesn't exist here
    console.log(result.tableName)
  } else {
    // TypeScript knows columnName exists here
    console.log(`${result.tableName}.${result.columnName}`)
  }
}
```

## Migration from Wails Types

If migrating existing code from Wails class-based types:

```typescript
// Before (Wails classes)
import { schemadiff, catalog } from '../wailsjs/go/models'

const snapshot = new schemadiff.SchemaSnapshot(data)
const entry = new catalog.TableCatalogEntry(data)

// After (TypeScript interfaces)
import { SchemaSnapshot } from '@/types/schema-diff'
import { TableCatalogEntry } from '@/types/catalog'

const snapshot: SchemaSnapshot = data
const entry: TableCatalogEntry = data
```

Benefits:
- ✅ No runtime overhead from class instantiation
- ✅ Better tree-shaking and bundle size
- ✅ Cleaner, more idiomatic TypeScript
- ✅ Easier to work with in React components
- ✅ Better autocomplete and IntelliSense
