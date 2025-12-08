# Schema Diff Tool - Product Requirements Document

## Executive Summary

The Schema Diff Tool enables side-by-side comparison of database schemas across connections, with visual diff highlighting, migration SQL generation, and snapshot functionality. This feature extends HowlerOps' existing schema visualization to support schema evolution tracking and migration workflows.

**Target Users**: Database administrators, developers managing schema migrations, DevOps engineers
**Priority**: Medium-High
**Estimated Effort**: 3-4 weeks

## Problem Statement

Database teams face challenges when:
1. Comparing schemas between environments (dev/staging/prod)
2. Tracking schema changes over time without formal migration tools
3. Generating migration scripts from ad-hoc schema differences
4. Understanding what changed between database versions

Current workflow requires manual comparison using external tools or writing custom scripts.

## Goals & Success Metrics

### Goals
1. Enable visual schema comparison between any two connections
2. Generate actionable migration SQL from detected differences
3. Provide schema snapshot functionality for historical comparison
4. Integrate seamlessly with existing schema visualizer

### Success Metrics
| Metric | Target |
|--------|--------|
| Schema comparison time | < 5 seconds for schemas under 200 tables |
| Migration SQL accuracy | 95% runnable without manual edits |
| User adoption | 40% of active users within 3 months |

## User Stories

### Primary Stories

**US-1**: As a database administrator, I want to compare schemas between production and staging so I can identify drift before deployments.

**US-2**: As a developer, I want to generate migration SQL from schema differences so I can quickly create migration files.

**US-3**: As a DevOps engineer, I want to save schema snapshots at deployment time so I can track schema evolution.

**US-4**: As a developer, I want to see visual diffs highlighting added/removed/modified objects so I can quickly understand changes.

### Secondary Stories

**US-5**: As a DBA, I want to compare a snapshot against a live connection to see what changed since the last deployment.

**US-6**: As a developer, I want to filter diff results by object type (tables, columns, indexes, foreign keys) to focus on relevant changes.

## Technical Requirements

### Backend (Go)

#### New Package: `pkg/schemadiff`

```
backend-go/pkg/schemadiff/
├── types.go          # Diff result types
├── comparator.go     # Schema comparison logic
├── snapshot.go       # Snapshot save/load
├── migration.go      # SQL generation
└── comparator_test.go
```

#### Core Types

```go
type SchemaDiff struct {
    SourceID    string          `json:"source_id"`
    TargetID    string          `json:"target_id"`
    Timestamp   time.Time       `json:"timestamp"`
    Summary     DiffSummary     `json:"summary"`
    Tables      []TableDiff     `json:"tables"`
    Duration    time.Duration   `json:"duration"`
}

type DiffSummary struct {
    TablesAdded     int `json:"tables_added"`
    TablesRemoved   int `json:"tables_removed"`
    TablesModified  int `json:"tables_modified"`
    ColumnsAdded    int `json:"columns_added"`
    ColumnsRemoved  int `json:"columns_removed"`
    ColumnsModified int `json:"columns_modified"`
    IndexesChanged  int `json:"indexes_changed"`
    FKsChanged      int `json:"fks_changed"`
}

type TableDiff struct {
    Schema      string        `json:"schema"`
    Name        string        `json:"name"`
    Status      DiffStatus    `json:"status"` // added, removed, modified, unchanged
    Columns     []ColumnDiff  `json:"columns,omitempty"`
    Indexes     []IndexDiff   `json:"indexes,omitempty"`
    ForeignKeys []FKDiff      `json:"foreign_keys,omitempty"`
}

type DiffStatus string
const (
    DiffAdded     DiffStatus = "added"
    DiffRemoved   DiffStatus = "removed"
    DiffModified  DiffStatus = "modified"
    DiffUnchanged DiffStatus = "unchanged"
)

type ColumnDiff struct {
    Name      string     `json:"name"`
    Status    DiffStatus `json:"status"`
    OldType   string     `json:"old_type,omitempty"`
    NewType   string     `json:"new_type,omitempty"`
    OldNull   *bool      `json:"old_nullable,omitempty"`
    NewNull   *bool      `json:"new_nullable,omitempty"`
    OldDefault *string   `json:"old_default,omitempty"`
    NewDefault *string   `json:"new_default,omitempty"`
}

type SchemaSnapshot struct {
    ID           string                        `json:"id"`
    Name         string                        `json:"name"`
    ConnectionID string                        `json:"connection_id"`
    DatabaseType database.DatabaseType         `json:"database_type"`
    Schemas      []string                      `json:"schemas"`
    Tables       map[string][]database.TableInfo `json:"tables"`
    Structures   map[string]*database.TableStructure `json:"structures"`
    CreatedAt    time.Time                     `json:"created_at"`
    Hash         string                        `json:"hash"`
}
```

#### Wails Bindings (app.go)

```go
func (a *App) CompareSchemas(sourceSessionID, targetSessionID string) (*schemadiff.SchemaDiff, error)
func (a *App) CompareWithSnapshot(sessionID, snapshotID string) (*schemadiff.SchemaDiff, error)
func (a *App) SaveSchemaSnapshot(sessionID, name string) (*schemadiff.SchemaSnapshot, error)
func (a *App) ListSchemaSnapshots() ([]schemadiff.SchemaSnapshot, error)
func (a *App) DeleteSchemaSnapshot(snapshotID string) error
func (a *App) GenerateMigrationSQL(diff *schemadiff.SchemaDiff, direction string) (string, error)
```

#### Schema Introspection Integration

Leverage existing `Database` interface methods:
- `GetSchemas(ctx)` - List schemas
- `GetTables(ctx, schema)` - List tables per schema
- `GetTableStructure(ctx, schema, table)` - Get columns, indexes, FKs

Use existing `SchemaCache` for performance optimization.

### Frontend (React/TypeScript)

#### New Components

```
frontend/src/components/schema-diff/
├── schema-diff-panel.tsx       # Main diff view container
├── connection-selector.tsx     # Source/target picker
├── diff-summary-card.tsx       # Summary statistics
├── table-diff-list.tsx         # Scrollable diff results
├── table-diff-item.tsx         # Individual table diff
├── column-diff-row.tsx         # Column-level changes
├── migration-preview.tsx       # Generated SQL viewer
├── snapshot-manager.tsx        # Snapshot list/actions
└── diff-filter-bar.tsx         # Filter controls
```

#### Types

```typescript
interface SchemaDiff {
  sourceId: string
  targetId: string
  timestamp: string
  summary: DiffSummary
  tables: TableDiff[]
  duration: number
}

interface DiffSummary {
  tablesAdded: number
  tablesRemoved: number
  tablesModified: number
  columnsAdded: number
  columnsRemoved: number
  columnsModified: number
  indexesChanged: number
  fksChanged: number
}

interface TableDiff {
  schema: string
  name: string
  status: 'added' | 'removed' | 'modified' | 'unchanged'
  columns?: ColumnDiff[]
  indexes?: IndexDiff[]
  foreignKeys?: FKDiff[]
}

interface SchemaSnapshot {
  id: string
  name: string
  connectionId: string
  databaseType: string
  createdAt: string
}
```

#### Integration Points

1. **Schema Visualizer** - Add "Compare" button to toolbar
2. **Connection Sidebar** - Add "Compare" context menu option
3. **Command Palette** - Add "Compare Schemas" command

## API Design

### Wails RPC Methods

Following existing patterns from `app.go`:

| Method | Parameters | Returns |
|--------|------------|---------|
| `CompareSchemas` | `sourceSessionID, targetSessionID string` | `*SchemaDiff, error` |
| `CompareWithSnapshot` | `sessionID, snapshotID string` | `*SchemaDiff, error` |
| `SaveSchemaSnapshot` | `sessionID, name string` | `*SchemaSnapshot, error` |
| `ListSchemaSnapshots` | none | `[]SchemaSnapshot, error` |
| `DeleteSchemaSnapshot` | `snapshotID string` | `error` |
| `GenerateMigrationSQL` | `diff *SchemaDiff, direction string` | `string, error` |

## UI/UX Wireframes

### Main Diff View
```
┌─────────────────────────────────────────────────────────────────┐
│ Schema Diff                                              [X]    │
├─────────────────────────────────────────────────────────────────┤
│ Source: [Production ▼]          Target: [Staging ▼]    [Compare]│
│                                  OR                             │
│ Compare with snapshot: [Select snapshot... ▼]                   │
├─────────────────────────────────────────────────────────────────┤
│ Summary: +3 tables  -1 table  ~5 modified  |  Duration: 1.2s   │
├─────────────────────────────────────────────────────────────────┤
│ Filter: [All ▼]  □ Tables  □ Columns  □ Indexes  □ FKs        │
├───────────────────────────────┬─────────────────────────────────┤
│ Tables                        │ Details                         │
│ ┌───────────────────────────┐ │ ┌─────────────────────────────┐ │
│ │ + users         (added)   │ │ │ Table: users                │ │
│ │ - old_logs      (removed) │ │ │ Status: Added               │ │
│ │ ~ orders        (modified)│ │ │                             │ │
│ │ ~ products      (modified)│ │ │ Columns:                    │ │
│ │   customers     (same)    │ │ │ + id         bigint PK     │ │
│ │                           │ │ │ + email      varchar(255)   │ │
│ │                           │ │ │ + created_at timestamp      │ │
│ └───────────────────────────┘ │ └─────────────────────────────┘ │
├───────────────────────────────┴─────────────────────────────────┤
│ [Save Snapshot]  [Generate Migration ▼]  [Export JSON]          │
└─────────────────────────────────────────────────────────────────┘
```

### Migration Preview Modal
```
┌─────────────────────────────────────────────────────────────────┐
│ Generated Migration SQL                                   [X]   │
├─────────────────────────────────────────────────────────────────┤
│ Direction: [Forward (Source → Target) ▼]                        │
├─────────────────────────────────────────────────────────────────┤
│ -- Migration generated by HowlerOps                             │
│ -- Source: Production  Target: Staging                          │
│ -- Generated: 2024-01-15 10:30:00                               │
│                                                                 │
│ -- Add table: users                                             │
│ CREATE TABLE users (                                            │
│   id BIGINT PRIMARY KEY,                                        │
│   email VARCHAR(255) NOT NULL,                                  │
│   created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP                │
│ );                                                              │
│                                                                 │
│ -- Drop table: old_logs                                         │
│ DROP TABLE IF EXISTS old_logs;                                  │
│                                                                 │
│ -- Modify table: orders                                         │
│ ALTER TABLE orders ADD COLUMN tracking_id VARCHAR(50);          │
├─────────────────────────────────────────────────────────────────┤
│ [Copy to Clipboard]  [Open in Query Editor]  [Download .sql]    │
└─────────────────────────────────────────────────────────────────┘
```

## Database Support Matrix

| Database | Schema Introspection | Migration SQL | Snapshot | Notes |
|----------|---------------------|---------------|----------|-------|
| PostgreSQL | Full | Full | Yes | Primary target |
| MySQL | Full | Full | Yes | Including MariaDB |
| SQLite | Full | Full | Yes | Single-file snapshots |
| ClickHouse | Full | Partial | Yes | No ALTER constraints |
| MongoDB | Partial | N/A | Yes | Collection/index only |
| TiDB | Full | Full | Yes | MySQL-compatible |
| Elasticsearch | Partial | N/A | Yes | Mapping comparison only |
| DuckDB | Full | Partial | Yes | Limited ALTER support |
| Turso | Full | Full | Yes | SQLite-compatible |

### Database-Specific Considerations

**PostgreSQL/MySQL/TiDB**:
- Full DDL generation including constraints, defaults, comments
- Index method preservation (btree, hash, gin, etc.)
- Sequence/auto-increment handling

**SQLite**:
- Limited ALTER TABLE (recreate table for column modifications)
- No schema namespace (single namespace)

**MongoDB/Elasticsearch**:
- Schema-less; compare inferred structure from sample documents
- Index and mapping comparison only
- No DDL migration generation

**ClickHouse**:
- Engine-specific DDL (MergeTree family)
- No foreign key constraints
- Partition-aware comparison

## Implementation Plan

### Phase 1: Core Backend (Week 1)
- [ ] Create `pkg/schemadiff` package structure
- [ ] Implement `Comparator` for table/column/index/FK diffing
- [ ] Add snapshot save/load with JSON storage
- [ ] Wire up Wails bindings

### Phase 2: Migration Generation (Week 2)
- [ ] PostgreSQL migration generator
- [ ] MySQL migration generator
- [ ] SQLite migration generator (with table recreation)
- [ ] Add direction support (forward/reverse)

### Phase 3: Frontend UI (Week 2-3)
- [ ] Schema diff panel component
- [ ] Connection/snapshot selector
- [ ] Diff result display with tree view
- [ ] Migration preview modal
- [ ] Snapshot manager

### Phase 4: Integration & Polish (Week 3-4)
- [ ] Schema visualizer integration
- [ ] Command palette commands
- [ ] Keyboard shortcuts
- [ ] Error handling and edge cases
- [ ] Performance optimization for large schemas

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Large schema comparison timeout | High | Medium | Implement streaming comparison, progress indicator |
| Migration SQL errors | High | Medium | Add preview mode, warn on destructive operations |
| Cross-database type mapping | Medium | High | Document type equivalence, allow manual override |
| Snapshot storage growth | Low | Medium | Add snapshot retention policy, compression |
| Schema-less DB comparison | Medium | Low | Document limitations, compare indexes only |

### Technical Risks

1. **Type System Differences**: PostgreSQL `TEXT` vs MySQL `LONGTEXT` - mitigate with type normalization layer
2. **Constraint Naming**: Auto-generated constraint names differ between DBs - compare by structure not name
3. **Case Sensitivity**: MySQL table names case-sensitive on Linux - normalize comparisons

## Out of Scope (v1)

- Real-time schema sync/replication
- Automatic migration execution
- Schema diff history/versioning
- Multi-database diff (3+ sources)
- Trigger/procedure comparison
- Row-level data diff

## Dependencies

- Existing `Database` interface and implementations
- `SchemaCache` for performance
- `TableStructure` type from `pkg/database/types.go`
- React Flow (for potential visual diff mode)
