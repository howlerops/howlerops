# HowlerOps Storage Architecture

## Overview

HowlerOps uses a **local-first** storage architecture powered by SQLite for all application data, including connections, queries, query history, and vector embeddings for RAG (Retrieval-Augmented Generation).

## Design Principles

1. **Local-First**: Works completely offline, no internet required
2. **Zero External Dependencies**: No PostgreSQL, Redis, or Qdrant needed
3. **Embedded Storage**: SQLite databases stored in `~/.howlerops/`
4. **Team-Ready**: Architecture supports future team collaboration via Turso
5. **Encrypted Credentials**: Database passwords stored encrypted
6. **DRY Architecture**: Same code for local and team modes

## Architecture Diagram

```
┌─────────────────────────────────────────────────────┐
│                    Wails App (app.go)               │
├─────────────────────────────────────────────────────┤
│                  Storage Manager                    │
│              (backend-go/pkg/storage/)              │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────┐     ┌─────────────────────┐ │
│  │ LocalSQLiteStorage │     │ TursoTeamStorage   │ │
│  │  (Solo Mode)       │     │  (Team Mode)       │ │
│  └──────────────────┘     └─────────────────────┘ │
│          │                         │               │
│          ▼                         ▼               │
│  ┌──────────────────┐     ┌─────────────────────┐ │
│  │  local.db        │     │  team-replica.db    │ │
│  │  (Connections,   │     │  (Synced via        │ │
│  │   Queries,       │     │   Turso)            │ │
│  │   History)       │     │                     │ │
│  └──────────────────┘     └─────────────────────┘ │
│          │                                         │
│          ▼                                         │
│  ┌──────────────────┐                            │
│  │  vectors.db      │                            │
│  │  (Embeddings,    │                            │
│  │   RAG Data)      │                            │
│  └──────────────────┘                            │
│                                                     │
└─────────────────────────────────────────────────────┘
```

## Storage Components

### 1. Storage Manager (`backend-go/pkg/storage/manager.go`)

Central coordinator that:
- Manages storage mode (Solo vs Team)
- Switches between local and team storage
- Provides unified Storage interface
- Handles lifecycle (startup/shutdown)

```go
type Manager struct {
    mode       Mode // Solo or Team
    storage    Storage
    localStore *LocalSQLiteStorage
    teamStore  *TursoTeamStorage
    userID     string
    logger     *logrus.Logger
}
```

### 2. Storage Interface (`backend-go/pkg/storage/interface.go`)

Unified interface for all storage operations:

```go
type Storage interface {
    // Connection management
    SaveConnection(ctx context.Context, conn *Connection) error
    GetConnections(ctx context.Context, filters *ConnectionFilters) ([]*Connection, error)
    DeleteConnection(ctx context.Context, id string) error
    
    // Query management
    SaveQuery(ctx context.Context, query *SavedQuery) error
    GetQueryHistory(ctx context.Context, filters *HistoryFilters) ([]*QueryHistory, error)
    
    // Vector/RAG operations
    IndexDocument(ctx context.Context, doc *Document) error
    SearchDocuments(ctx context.Context, embedding []float32, filters *DocumentFilters) ([]*Document, error)
    
    // Settings & cache
    GetSetting(ctx context.Context, key string) (string, error)
    CacheSchema(ctx context.Context, connID string, schema *SchemaCache) error
    
    // Team operations
    GetTeam(ctx context.Context) (*Team, error)
    
    // Lifecycle
    Close() error
}
```

### 3. Local SQLite Storage (`backend-go/pkg/storage/sqlite_local.go`)

Solo mode implementation:
- **Database**: `~/.howlerops/local.db`
- **Vectors**: `~/.howlerops/vectors.db`
- **Features**:
  - Connection CRUD with encrypted passwords
  - Saved queries with tags/folders
  - Query history with performance metrics
  - Schema caching with TTL
  - Local-only settings (never synced)

### 4. SQLite Vector Store (`backend-go/internal/rag/sqlite_vector_store.go`)

RAG-specific vector database:
- **Embeddings**: Stored as BLOB (serialized float32 arrays)
- **FTS5**: Full-text search on document content
- **Hybrid Search**: Combines vector similarity + text search
- **Collections**: Separate namespaces (schemas, queries, business rules)

## Database Schemas

### Local Database (`local.db`)

**Connections**
```sql
CREATE TABLE connections (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    host TEXT,
    port INTEGER,
    database_name TEXT,
    username TEXT,
    password_encrypted TEXT,
    ssl_config TEXT,
    created_by TEXT,
    created_at INTEGER,
    updated_at INTEGER,
    team_id TEXT,
    is_shared BOOLEAN DEFAULT 0,
    metadata TEXT
);
```

**Saved Queries**
```sql
CREATE TABLE saved_queries (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    query TEXT NOT NULL,
    description TEXT,
    connection_id TEXT,
    created_by TEXT,
    created_at INTEGER,
    updated_at INTEGER,
    team_id TEXT,
    is_shared BOOLEAN DEFAULT 0,
    tags TEXT,  -- JSON array
    folder TEXT,
    FOREIGN KEY (connection_id) REFERENCES connections(id)
);
```

**Query History**
```sql
CREATE TABLE query_history (
    id TEXT PRIMARY KEY,
    query TEXT NOT NULL,
    connection_id TEXT,
    executed_by TEXT,
    executed_at INTEGER,
    duration_ms INTEGER,
    rows_returned INTEGER,
    success BOOLEAN,
    error TEXT,
    team_id TEXT,
    is_shared BOOLEAN DEFAULT 1
);
```

### Vector Database (`vectors.db`)

**Documents**
```sql
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    connection_id TEXT,
    type TEXT,  -- 'schema', 'query', 'plan', 'business', 'performance'
    content TEXT,
    metadata TEXT,  -- JSON
    created_at INTEGER,
    updated_at INTEGER,
    access_count INTEGER DEFAULT 0,
    last_accessed INTEGER
);
```

**Embeddings**
```sql
CREATE TABLE embeddings (
    document_id TEXT PRIMARY KEY,
    embedding BLOB,  -- Serialized float32 array
    dimension INTEGER,
    FOREIGN KEY (document_id) REFERENCES documents(id)
);
```

**Full-Text Search**
```sql
CREATE VIRTUAL TABLE documents_fts USING fts5(
    document_id UNINDEXED,
    content,
    tokenize = 'porter unicode61'
);
```

## Solo Mode vs Team Mode

### Solo Mode (Default)

- **Storage**: Local SQLite only (`~/.howlerops/`)
- **Sync**: None - fully offline
- **Users**: Single user
- **Performance**: Fastest (no network)
- **Data**: Stays on local machine

**Use Cases**:
- Personal projects
- Offline work
- Maximum privacy
- Fast iterations

### Team Mode (Future)

- **Storage**: Local SQLite + Turso cloud replica
- **Sync**: Automatic background sync
- **Users**: Multiple team members
- **Performance**: Good (local reads, background writes)
- **Data**: Shared connections, queries, RAG learnings

**Use Cases**:
- Team collaboration
- Shared query library
- Collective knowledge base
- Cross-device sync

**Team Mode Architecture**:
```
┌──────────────┐     Sync      ┌──────────────┐
│   Local      │ ◄──────────► │    Turso     │
│ SQLite DB    │   (libsql)   │   Cloud DB   │
└──────────────┘               └──────────────┘
       ▲                              ▲
       │                              │
   App reads                      Team writes
   (instant)                      (background)
```

## Data Flow Examples

### Saving a Query

```
1. User saves query in UI
2. Frontend calls: SaveQuery(query)
3. app.go → storageManager.SaveQuery()
4. storageManager → storage.SaveQuery()
5. LocalSQLiteStorage writes to local.db
6. Success returned to UI
```

### AI Query Generation with RAG

```
1. User types natural language query
2. Frontend calls: GenerateSQLFromNaturalLanguage(prompt)
3. app.go → aiService.GenerateSQL()
4. AI service → storageManager.SearchDocuments(embedding)
5. StorageManager → vectorStore.SearchSimilar()
6. SQLite vector search finds similar queries
7. Context passed to LLM
8. Generated SQL returned to UI
```

## Performance Optimizations

1. **WAL Mode**: Enabled for concurrent reads during writes
2. **Connection Pooling**: Reuse SQLite connections
3. **Schema Caching**: Reduce database round-trips
4. **Batch Operations**: Bulk inserts for vector data
5. **Indexes**: Strategic indexes on frequently queried columns
6. **PRAGMA Settings**: Optimized for performance
   ```sql
   PRAGMA cache_size = -131072;  -- 128MB cache
   PRAGMA mmap_size = 268435456;  -- 256MB mmap
   PRAGMA journal_mode = WAL;
   PRAGMA synchronous = NORMAL;
   ```

## Backup & Recovery

### Automatic Backups

- Created before database resets
- Stored in `~/.howlerops/backups/`
- Timestamped: `local_20251014_103045.db`

### Manual Backup

```bash
make backup-local-db
```

### Restore from Backup

```bash
cp ~/.howlerops/backups/local_20251014_103045.db ~/.howlerops/local.db
cp ~/.howlerops/backups/vectors_20251014_103045.db ~/.howlerops/vectors.db
```

## Security

### Credential Encryption

Connection passwords are encrypted before storage:
- **Algorithm**: AES-256-GCM
- **Key Derivation**: PBKDF2
- **Storage**: `password_encrypted` column

### Local-Only Secrets

AI API keys and other secrets stored in `local_credentials` table:
- **Never synced** to team storage
- **Machine-specific**: Stays on local device
- **Encrypted at rest**: Same encryption as passwords

## Monitoring & Debugging

### Check Storage Status

```go
// In app
stats, err := app.storageManager.GetStorage().GetStats(ctx)
// Returns: mode, user_id, document_count, etc.
```

### View Database Contents

```bash
# Local database
sqlite3 ~/.howlerops/local.db ".schema"
sqlite3 ~/.howlerops/local.db "SELECT * FROM connections;"

# Vector database
sqlite3 ~/.howlerops/vectors.db "SELECT COUNT(*) FROM documents;"
sqlite3 ~/.howlerops/vectors.db "SELECT type, COUNT(*) FROM documents GROUP BY type;"
```

### Logs

```bash
# App logs show storage operations
tail -f ~/.howlerops/logs/howlerops.log | grep storage
```

## Migration from Other Databases

If migrating from PostgreSQL/Qdrant:

1. **Export Data**: Use existing tools to export
2. **Transform**: Convert to SQLite schema
3. **Import**: Use migration tool or SQL INSERT statements
4. **Verify**: Compare row counts

See: [MIGRATION_FROM_QDRANT.md](./MIGRATION_FROM_QDRANT.md)

## Future Enhancements

- [ ] Distributed SQLite with LiteFS (alternative to Turso)
- [ ] S3 backup sync for enterprise
- [ ] Compression for large query results
- [ ] Retention policies for old history
- [ ] Multi-tenancy support
- [ ] Audit logging
- [ ] Data export/import tools

## References

- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [Turso Documentation](https://docs.turso.tech/)
- [sqlite-vec Extension](https://github.com/asg017/sqlite-vec)
- [SQLite Best Practices](https://www.sqlite.org/bestpractice.html)

