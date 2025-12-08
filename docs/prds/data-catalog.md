# Data Catalog PRD

## 1. Executive Summary

Data Catalog is a metadata management feature for HowlerOps that enables teams to document, classify, and discover database assets across their organization. It provides table/column descriptions, data steward assignments, data classifications (PII, sensitive, etc.), and full-text search across all metadata—synced via Turso cloud for team collaboration.

**Target Users**: Data engineers, analysts, DBAs, and compliance officers who need to understand, govern, and discover data assets.

**Core Value**: Reduce time spent understanding unfamiliar schemas, improve data governance, and enable compliance with data privacy regulations.

## 2. Problem Statement

### Current Pain Points

1. **Tribal Knowledge**: Table and column meanings exist only in developers' heads or scattered documentation
2. **Data Discovery**: No way to search "which tables contain customer email?" across connections
3. **Governance Gap**: No tracking of data ownership or sensitivity classifications
4. **Compliance Risk**: PII exists in databases without clear identification or documentation
5. **Onboarding Friction**: New team members spend significant time understanding schema purposes

### Opportunity

HowlerOps already has:
- Schema visualization (frontend/src/components/schema-visualizer/)
- PII detection infrastructure (backend-go/internal/pii/)
- Organization/team structure (backend-go/internal/organization/)
- Cloud sync via Turso (backend-go/pkg/storage/turso/)

The Data Catalog leverages these existing patterns to add metadata management with minimal new infrastructure.

## 3. Goals & Success Metrics

### Primary Goals

1. Enable users to document any table or column with descriptions
2. Support data steward/owner assignment per table
3. Provide tag-based classification system (PII, sensitive, etc.)
4. Deliver full-text search across all catalog metadata
5. Sync catalog data across organization members

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Tables with descriptions | >50% within 30 days | Catalog coverage report |
| Search usage | >10 searches/user/week | Usage analytics |
| Time to find data | <30 seconds | User testing |
| PII coverage | 100% auto-detected fields tagged | Compliance audit |

### Non-Goals (V1)

- Data lineage tracking
- Automated description generation (AI)
- Data quality scoring
- Change history/versioning
- Custom metadata fields

## 4. User Stories

### Core Workflows

**US-1: Document a Table**
> As a data engineer, I want to add a description to a table so that other team members understand its purpose.

Acceptance Criteria:
- Can add/edit description from schema visualizer
- Description persists across sessions
- Description syncs to team members
- Maximum 2000 characters

**US-2: Classify Sensitive Data**
> As a compliance officer, I want to tag columns as PII so that we maintain GDPR compliance.

Acceptance Criteria:
- Can apply predefined tags (PII, sensitive, internal, public)
- Can create custom tags per organization
- Auto-suggested tags from PII detector
- Tags visible in schema visualizer

**US-3: Assign Data Steward**
> As a team lead, I want to assign ownership to tables so that team members know who to contact.

Acceptance Criteria:
- Can assign any org member as steward
- Steward visible in table details
- Can filter tables by steward
- Notification on steward assignment (optional)

**US-4: Search Catalog**
> As an analyst, I want to search for "customer email" and find all relevant tables/columns.

Acceptance Criteria:
- Full-text search across names, descriptions, tags
- Results ranked by relevance
- Filter by connection, schema, tag
- Results show table, column, description preview

**US-5: View Catalog in Schema Visualizer**
> As a developer, I want to see catalog metadata in the schema diagram.

Acceptance Criteria:
- Tags displayed as badges on table nodes
- Steward shown in table tooltip
- Description shown in table detail panel
- PII columns highlighted

## 5. Technical Requirements

### 5.1 Backend (Go)

#### New Package Structure

```
backend-go/internal/catalog/
├── types.go          # Domain models
├── store.go          # Repository interface
├── service.go        # Business logic
├── handlers.go       # HTTP handlers
└── search.go         # Search implementation
```

#### Domain Models

```go
// TableCatalogEntry represents catalog metadata for a table
type TableCatalogEntry struct {
    ID             string    `json:"id"`
    ConnectionID   string    `json:"connection_id"`
    SchemaName     string    `json:"schema_name"`
    TableName      string    `json:"table_name"`
    Description    string    `json:"description,omitempty"`
    StewardUserID  *string   `json:"steward_user_id,omitempty"`
    Tags           []string  `json:"tags,omitempty"`
    OrganizationID *string   `json:"organization_id,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
    CreatedBy      string    `json:"created_by"`
}

// ColumnCatalogEntry represents catalog metadata for a column
type ColumnCatalogEntry struct {
    ID              string    `json:"id"`
    TableCatalogID  string    `json:"table_catalog_id"`
    ColumnName      string    `json:"column_name"`
    Description     string    `json:"description,omitempty"`
    Tags            []string  `json:"tags,omitempty"`
    PIIType         *string   `json:"pii_type,omitempty"`
    PIIConfidence   *float64  `json:"pii_confidence,omitempty"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

// CatalogTag represents a reusable tag
type CatalogTag struct {
    ID             string  `json:"id"`
    Name           string  `json:"name"`
    Color          string  `json:"color"`
    Description    string  `json:"description,omitempty"`
    OrganizationID *string `json:"organization_id,omitempty"`
    IsSystem       bool    `json:"is_system"`
}
```

#### Store Interface

```go
type CatalogStore interface {
    // Table entries
    CreateTableEntry(ctx context.Context, entry *TableCatalogEntry) error
    GetTableEntry(ctx context.Context, connectionID, schema, table string) (*TableCatalogEntry, error)
    UpdateTableEntry(ctx context.Context, entry *TableCatalogEntry) error
    DeleteTableEntry(ctx context.Context, id string) error
    ListTableEntries(ctx context.Context, connectionID string) ([]*TableCatalogEntry, error)

    // Column entries
    CreateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error
    GetColumnEntry(ctx context.Context, tableID, column string) (*ColumnCatalogEntry, error)
    UpdateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error
    ListColumnEntries(ctx context.Context, tableID string) ([]*ColumnCatalogEntry, error)

    // Tags
    CreateTag(ctx context.Context, tag *CatalogTag) error
    ListTags(ctx context.Context, orgID *string) ([]*CatalogTag, error)
    DeleteTag(ctx context.Context, id string) error

    // Search
    SearchCatalog(ctx context.Context, query string, filters SearchFilters) (*SearchResults, error)
}
```

### 5.2 Frontend (React/TypeScript)

#### New Components

```
frontend/src/components/catalog/
├── catalog-panel.tsx           # Side panel for catalog editing
├── catalog-search.tsx          # Search interface
├── catalog-tag-badge.tsx       # Tag display component
├── steward-selector.tsx        # User picker for steward
└── catalog-table-overlay.tsx   # Overlay for schema visualizer
```

#### Types

```typescript
interface TableCatalogEntry {
  id: string;
  connectionId: string;
  schemaName: string;
  tableName: string;
  description?: string;
  stewardUserId?: string;
  stewardUser?: UserInfo;
  tags: string[];
  organizationId?: string;
  createdAt: string;
  updatedAt: string;
}

interface ColumnCatalogEntry {
  id: string;
  tableCatalogId: string;
  columnName: string;
  description?: string;
  tags: string[];
  piiType?: string;
  piiConfidence?: number;
}

interface CatalogSearchResult {
  type: 'table' | 'column';
  connectionId: string;
  schemaName: string;
  tableName: string;
  columnName?: string;
  description?: string;
  tags: string[];
  relevanceScore: number;
}
```

### 5.3 Cloud Sync (Turso)

#### Schema Migration

```sql
-- Migration: Add catalog tables

CREATE TABLE IF NOT EXISTS catalog_table_entries (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL,
    schema_name TEXT NOT NULL,
    table_name TEXT NOT NULL,
    description TEXT,
    steward_user_id TEXT REFERENCES users(id) ON DELETE SET NULL,
    tags TEXT, -- JSON array
    organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    created_by TEXT NOT NULL,
    sync_version INTEGER DEFAULT 1,
    deleted_at INTEGER,
    UNIQUE(connection_id, schema_name, table_name)
);

CREATE INDEX IF NOT EXISTS idx_catalog_tables_connection ON catalog_table_entries(connection_id);
CREATE INDEX IF NOT EXISTS idx_catalog_tables_org ON catalog_table_entries(organization_id);
CREATE INDEX IF NOT EXISTS idx_catalog_tables_steward ON catalog_table_entries(steward_user_id);
CREATE INDEX IF NOT EXISTS idx_catalog_tables_updated ON catalog_table_entries(updated_at);

CREATE TABLE IF NOT EXISTS catalog_column_entries (
    id TEXT PRIMARY KEY,
    table_catalog_id TEXT NOT NULL REFERENCES catalog_table_entries(id) ON DELETE CASCADE,
    column_name TEXT NOT NULL,
    description TEXT,
    tags TEXT, -- JSON array
    pii_type TEXT,
    pii_confidence REAL,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    sync_version INTEGER DEFAULT 1,
    UNIQUE(table_catalog_id, column_name)
);

CREATE INDEX IF NOT EXISTS idx_catalog_columns_table ON catalog_column_entries(table_catalog_id);
CREATE INDEX IF NOT EXISTS idx_catalog_columns_pii ON catalog_column_entries(pii_type);

CREATE TABLE IF NOT EXISTS catalog_tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#6366f1',
    description TEXT,
    organization_id TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    is_system BOOLEAN DEFAULT 0,
    created_at INTEGER NOT NULL,
    UNIQUE(name, organization_id)
);

-- System tags (seeded)
INSERT OR IGNORE INTO catalog_tags (id, name, color, description, is_system, created_at)
VALUES
    ('tag-pii', 'PII', '#ef4444', 'Personally Identifiable Information', 1, strftime('%s', 'now')),
    ('tag-sensitive', 'Sensitive', '#f97316', 'Sensitive business data', 1, strftime('%s', 'now')),
    ('tag-internal', 'Internal', '#eab308', 'Internal use only', 1, strftime('%s', 'now')),
    ('tag-public', 'Public', '#22c55e', 'Safe for public access', 1, strftime('%s', 'now')),
    ('tag-deprecated', 'Deprecated', '#6b7280', 'Scheduled for removal', 1, strftime('%s', 'now'));

-- Full-text search virtual table
CREATE VIRTUAL TABLE IF NOT EXISTS catalog_fts USING fts5(
    table_name,
    column_name,
    description,
    tags,
    content='',
    tokenize='porter'
);
```

## 6. API Design

### REST Endpoints

Following existing patterns from `organization/handlers.go`:

```
# Table Catalog
POST   /api/catalog/tables                          # Create table entry
GET    /api/catalog/tables?connection_id=...        # List entries for connection
GET    /api/catalog/tables/{id}                     # Get single entry
PUT    /api/catalog/tables/{id}                     # Update entry
DELETE /api/catalog/tables/{id}                     # Delete entry

# Column Catalog
POST   /api/catalog/tables/{tableId}/columns        # Create column entry
GET    /api/catalog/tables/{tableId}/columns        # List columns for table
GET    /api/catalog/columns/{id}                    # Get single column
PUT    /api/catalog/columns/{id}                    # Update column
DELETE /api/catalog/columns/{id}                    # Delete column

# Tags
GET    /api/catalog/tags                            # List all tags
POST   /api/catalog/tags                            # Create custom tag
DELETE /api/catalog/tags/{id}                       # Delete custom tag

# Search
GET    /api/catalog/search?q=...&connection=...&tags=...  # Search catalog

# Bulk Operations
POST   /api/catalog/tables/bulk                     # Bulk create/update
POST   /api/catalog/import-pii                      # Import PII detections as tags
```

### Request/Response Examples

**Create Table Entry**
```json
POST /api/catalog/tables
{
  "connection_id": "conn-123",
  "schema_name": "public",
  "table_name": "customers",
  "description": "Core customer data including contact information and preferences",
  "steward_user_id": "user-456",
  "tags": ["PII", "sensitive"]
}

Response: 201 Created
{
  "id": "cat-789",
  "connection_id": "conn-123",
  "schema_name": "public",
  "table_name": "customers",
  "description": "Core customer data...",
  "steward_user_id": "user-456",
  "steward_user": {
    "id": "user-456",
    "email": "jane@example.com",
    "username": "jdoe"
  },
  "tags": ["PII", "sensitive"],
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

**Search Catalog**
```json
GET /api/catalog/search?q=customer+email&tags=PII

Response: 200 OK
{
  "results": [
    {
      "type": "column",
      "connection_id": "conn-123",
      "schema_name": "public",
      "table_name": "customers",
      "column_name": "email",
      "description": "Customer primary email address",
      "tags": ["PII", "email"],
      "relevance_score": 0.95
    },
    {
      "type": "table",
      "connection_id": "conn-123",
      "schema_name": "public",
      "table_name": "customer_emails",
      "description": "Secondary email addresses for customers",
      "tags": ["PII"],
      "relevance_score": 0.82
    }
  ],
  "total": 2,
  "query": "customer email"
}
```

## 7. Data Model

### Entity Relationship

```
┌─────────────────┐     ┌──────────────────────┐
│  connections    │     │  catalog_table_      │
│  (existing)     │────<│  entries             │
└─────────────────┘     └──────────────────────┘
                                │
                                │ 1:N
                                ▼
                        ┌──────────────────────┐
                        │  catalog_column_     │
                        │  entries             │
                        └──────────────────────┘

┌─────────────────┐     ┌──────────────────────┐
│  organizations  │────<│  catalog_table_      │
│  (existing)     │     │  entries             │
└─────────────────┘     └──────────────────────┘
        │
        │ 1:N
        ▼
┌─────────────────┐
│  catalog_tags   │
│  (custom)       │
└─────────────────┘

┌─────────────────┐     ┌──────────────────────┐
│  users          │────<│  catalog_table_      │
│  (existing)     │     │  entries (steward)   │
└─────────────────┘     └──────────────────────┘
```

### Sync Strategy

Following existing patterns from `sync_store_adapter.go`:

1. **Versioning**: Each entry has `sync_version` incremented on update
2. **Soft Delete**: `deleted_at` timestamp for deletion sync
3. **Conflict Resolution**: Last-write-wins based on `updated_at`
4. **Scope**: Organization-level entries sync to all org members

## 8. Search and Discovery

### Search Implementation

Using SQLite FTS5 (already available in Turso):

```go
func (s *CatalogStore) SearchCatalog(ctx context.Context, query string, filters SearchFilters) (*SearchResults, error) {
    // Build FTS query
    ftsQuery := buildFTSQuery(query)

    sql := `
        SELECT
            t.id,
            t.connection_id,
            t.schema_name,
            t.table_name,
            NULL as column_name,
            t.description,
            t.tags,
            bm25(catalog_fts) as score
        FROM catalog_fts f
        JOIN catalog_table_entries t ON f.rowid = t.rowid
        WHERE catalog_fts MATCH ?
        AND (? IS NULL OR t.connection_id = ?)
        AND (? IS NULL OR t.organization_id = ?)
        AND t.deleted_at IS NULL

        UNION ALL

        SELECT
            c.id,
            t.connection_id,
            t.schema_name,
            t.table_name,
            c.column_name,
            c.description,
            c.tags,
            bm25(catalog_fts) as score
        FROM catalog_fts f
        JOIN catalog_column_entries c ON f.rowid = c.rowid
        JOIN catalog_table_entries t ON c.table_catalog_id = t.id
        WHERE catalog_fts MATCH ?
        AND (? IS NULL OR t.connection_id = ?)
        AND t.deleted_at IS NULL

        ORDER BY score DESC
        LIMIT ?
    `
    // Execute and map results...
}
```

### Search Features

1. **Full-text**: Searches table names, column names, descriptions, tags
2. **Fuzzy matching**: Porter stemming via FTS5 tokenizer
3. **Filters**: Connection, schema, tag, steward
4. **Ranking**: BM25 relevance scoring
5. **Highlighting**: Match snippets in results

## 9. Integration with Schema Visualizer

### Changes to TableNode Component

```typescript
// frontend/src/components/schema-visualizer/table-node.tsx

interface TableNodeData extends TableConfig {
  // ... existing fields
  catalogEntry?: TableCatalogEntry;
}

function TableNodeComponent({ data }: NodeProps<TableNodeData>) {
  const { catalogEntry } = data;

  return (
    <div className="...">
      {/* Header with tags */}
      <div className="flex items-center gap-1">
        {catalogEntry?.tags.map(tag => (
          <CatalogTagBadge key={tag} tag={tag} size="sm" />
        ))}
      </div>

      {/* Steward indicator */}
      {catalogEntry?.stewardUser && (
        <div className="text-xs text-muted-foreground">
          Steward: {catalogEntry.stewardUser.username}
        </div>
      )}

      {/* Description tooltip */}
      {catalogEntry?.description && (
        <TooltipContent>{catalogEntry.description}</TooltipContent>
      )}
    </div>
  );
}
```

### PII Integration

Leverage existing PII detector to auto-populate catalog:

```go
func (s *CatalogService) ImportPIIDetections(ctx context.Context, connectionID string) error {
    // Get PII fields from detector
    piiFields, err := s.piiDetector.GetRegisteredPIIFields(ctx)
    if err != nil {
        return err
    }

    // Create/update catalog entries with PII tags
    for _, field := range piiFields {
        entry, _ := s.store.GetColumnEntry(ctx, connectionID, field.TableName, field.FieldName)
        if entry == nil {
            entry = &ColumnCatalogEntry{
                ColumnName: field.FieldName,
                PIIType:    &field.PIIType,
                PIIConfidence: &field.ConfidenceScore,
                Tags: []string{"PII", field.PIIType},
            }
        } else {
            entry.PIIType = &field.PIIType
            entry.PIIConfidence = &field.ConfidenceScore
            entry.Tags = appendUnique(entry.Tags, "PII", field.PIIType)
        }
        s.store.UpsertColumnEntry(ctx, entry)
    }
    return nil
}
```

## 10. Implementation Plan

### Phase 1: Core Infrastructure (Week 1)

1. Database schema migration
2. Backend types, store, basic CRUD
3. REST API endpoints
4. Unit tests for store

### Phase 2: Search & Tags (Week 2)

1. FTS5 search implementation
2. Tag management (system + custom)
3. Search API with filters
4. Integration tests

### Phase 3: Frontend UI (Week 3)

1. Catalog panel component
2. Search interface
3. Tag badges in schema visualizer
4. Steward selector

### Phase 4: Integration & Polish (Week 4)

1. PII detector integration
2. Cloud sync testing
3. Organization-level visibility
4. Performance optimization
5. Documentation

### Milestones

| Milestone | Date | Deliverable |
|-----------|------|-------------|
| M1 | Week 1 | CRUD API working |
| M2 | Week 2 | Search functional |
| M3 | Week 3 | UI complete |
| M4 | Week 4 | Production ready |

## 11. Risks and Mitigations

### Technical Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| FTS5 performance on large catalogs | Medium | Low | Pagination, index optimization |
| Sync conflicts with multiple editors | Medium | Medium | Last-write-wins + audit log |
| Schema visualizer performance with overlays | High | Low | Lazy load catalog data |

### Product Risks

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Low adoption | High | Medium | Integrate into existing workflows |
| Stale documentation | Medium | High | Auto-suggest descriptions, reminders |
| Tag proliferation | Low | Medium | System tags + org-level custom |

### Security Considerations

1. **Authorization**: Catalog entries inherit connection/org permissions
2. **Audit Trail**: All changes logged via existing audit_logs table
3. **Data Exposure**: Descriptions should not contain credentials
4. **PII Handling**: Catalog metadata does not store actual PII values

## 12. Future Considerations

Out of scope for V1 but worth planning for:

1. **AI-Generated Descriptions**: Use LLM to suggest descriptions based on column names/data patterns
2. **Data Lineage**: Track where data comes from and where it flows
3. **Data Quality**: Add quality rules and scoring
4. **Change History**: Version descriptions and track changes over time
5. **Glossary**: Organization-wide data dictionary
6. **Compliance Reports**: Generate GDPR/CCPA data maps

## Appendix A: Existing Code References

- Organization patterns: `backend-go/internal/organization/`
- PII detection: `backend-go/internal/pii/`
- Turso storage: `backend-go/pkg/storage/turso/`
- Schema visualizer: `frontend/src/components/schema-visualizer/`
- Table node: `frontend/src/components/schema-visualizer/table-node.tsx`

## Appendix B: System Tag Definitions

| Tag | Color | Description |
|-----|-------|-------------|
| PII | Red (#ef4444) | Personally Identifiable Information |
| Sensitive | Orange (#f97316) | Sensitive business data |
| Internal | Yellow (#eab308) | Internal use only |
| Public | Green (#22c55e) | Safe for public access |
| Deprecated | Gray (#6b7280) | Scheduled for removal |
