# SQLite + Turso Migration Implementation Status

## ✅ Completed Phases

### Phase 1: Foundation - SQLite Vector Store ✅
- **`backend-go/internal/rag/sqlite_vector_store.go`** - Full implementation
  - Vector similarity search with cosine similarity
  - FTS5 full-text search
  - Hybrid search (vector + text)
  - Document CRUD operations
  - Collection management
  - Optimization and backup methods
  
- **`backend-go/internal/rag/migrations/001_init_sqlite_vectors.sql`** - Schema
  - Documents table with metadata
  - Embeddings table (BLOB for float32 arrays)
  - FTS5 virtual tables with triggers
  - Collections metadata
  - Performance indexes

- **`backend-go/internal/rag/vector_store.go`** - Factory pattern updated
  - NewVectorStore() supports SQLite and Qdrant
  - Qdrant deprecated (moved to qdrant_deprecated.go)

### Phase 2: Storage Abstraction Layer ✅
- **`backend-go/pkg/storage/interface.go`** - Unified storage interface
- **`backend-go/pkg/storage/types.go`** - Data types
- **`backend-go/pkg/storage/sqlite_local.go`** - Local implementation
- **`backend-go/pkg/storage/manager.go`** - Storage manager
- **`backend-go/pkg/storage/migrations/001_init_local_storage.sql`** - Schema

### Phase 4: Configuration & Environment ✅
- **`backend-go/configs/config.yaml`** - Updated with storage config
- **`scripts/init-local-db.sh`** - Database initialization
- **`scripts/reset-local-db.sh`** - Database reset with backup
- **`Makefile`** - Added database management targets

---

## 🚧 Phase 3: Integration with Existing Code (IN PROGRESS)

### 3.1 Database Manager ✅
The multiquery functionality is already working with the databaseAdapter pattern in `backend-go/pkg/database/manager.go`.

### 3.2 App.go Integration (TODO)

**File**: `app.go`

Need to add storage manager and integrate with existing services:

```go
type App struct {
    ctx             context.Context
    logger          *logrus.Logger
    storageManager  *storage.Manager  // NEW
    databaseService *services.DatabaseService
    fileService     *services.FileService
    keyboardService *services.KeyboardService
    aiService       *ai.Service
}

func (a *App) startup(ctx context.Context) {
    // Initialize storage manager
    storageConfig := &storage.Config{
        Mode: storage.ModeSolo,
        Local: storage.LocalStorageConfig{
            DataDir:    "~/.howlerops",
            Database:   "local.db",
            VectorsDB:  "vectors.db",
            UserID:     "local-user",  // TODO: Get from OS user
            VectorSize: 1536,
        },
    }
    
    var err error
    a.storageManager, err = storage.NewManager(ctx, storageConfig, a.logger)
    if err != nil {
        a.logger.WithError(err).Fatal("Failed to initialize storage")
    }
}
```

Add Wails-exported methods for storage:

```go
// Connection Management
func (a *App) SaveConnection(conn *ConnectionRequest) error
func (a *App) GetConnections() ([]*Connection, error)
func (a *App) DeleteConnection(id string) error

// Query Management
func (a *App) SaveQuery(query *SavedQueryRequest) error
func (a *App) GetSavedQueries(filters *QueryFilters) ([]*SavedQuery, error)
func (a *App) GetQueryHistory(filters *HistoryFilters) ([]*QueryHistory, error)

// Team Operations (future)
func (a *App) JoinTeam(teamID string, authToken string) error
func (a *App) LeaveTeam() error
func (a *App) GetTeamInfo() (*TeamInfo, error)
func (a *App) SyncWithTeam() error
```

### 3.3 AI Service Integration (TODO)

**File**: `backend-go/pkg/ai/service.go` or `backend-go/internal/ai/service.go`

Update AI service to use storage manager for RAG operations:

```go
// In RAG context builder, use storage for vector operations
func (c *ContextBuilder) FetchSimilarQueries(ctx context.Context, embedding []float32) ([]*storage.Document, error) {
    filters := &storage.DocumentFilters{
        Type:  "query",
        Limit: c.config.MaxSimilarQueries,
    }
    return c.storage.SearchDocuments(ctx, embedding, filters)
}
```

---

## 📋 Phase 5: Dependencies & Build (TODO)

### 5.1 Go Dependencies ✅ (Mostly Complete)

**Current status**:
- ✅ `github.com/mattn/go-sqlite3 v1.14.18` - Already added
- ⏳ `github.com/tursodatabase/libsql-client-go` - For team mode (Phase 3.3)

### 5.2 SQLite Vector Extension (TODO)

**Option 1**: sqlite-vec (Recommended)
```bash
# scripts/install-sqlite-vec.sh
#!/bin/bash
set -e

echo "Installing sqlite-vec extension..."

# Check if extension already exists
if [ -f ~/.howlerops/extensions/vec0.so ]; then
    echo "sqlite-vec extension already installed"
    exit 0
fi

# Clone and build
TMP_DIR=$(mktemp -d)
cd $TMP_DIR
git clone https://github.com/asg017/sqlite-vec
cd sqlite-vec
make loadable

# Install
mkdir -p ~/.howlerops/extensions
cp vec0.so ~/.howlerops/extensions/
echo "✓ sqlite-vec extension installed"
```

**Option 2**: Use Go pure implementation (no C extension needed)
- Implement vector similarity search in Go
- Store embeddings as BLOBs
- Current implementation already does this!

### 5.3 Update Makefile ✅

Already added targets for database management.

---

## 📊 Phase 6: Migration & Testing (TODO)

### 6.1 Migration Tool from Qdrant

**File**: `backend-go/cmd/migrate-vector-db/main.go` (NEW)

```go
package main

import (
    "context"
    "flag"
    "fmt"
    "log"
    
    "github.com/sql-studio/backend-go/internal/rag"
)

func main() {
    from := flag.String("from", "qdrant", "Source vector store (qdrant)")
    to := flag.String("to", "sqlite", "Target vector store (sqlite)")
    batchSize := flag.Int("batch", 100, "Batch size for migration")
    flag.Parse()

    ctx := context.Background()
    
    // Initialize source store (Qdrant)
    sourceConfig := &rag.VectorStoreConfig{
        Type: *from,
        QdrantConfig: &rag.QdrantConfig{
            Host: "localhost",
            Port: 6333,
        },
    }
    
    // Initialize target store (SQLite)
    targetConfig := &rag.VectorStoreConfig{
        Type: *to,
        SQLiteConfig: &rag.SQLiteVectorConfig{
            Path: "~/.howlerops/vectors.db",
            VectorSize: 1536,
        },
    }
    
    // TODO: Implement migration logic
    // 1. List all collections in source
    // 2. For each collection:
    //    a. Fetch all documents in batches
    //    b. Write to target store
    //    c. Verify counts match
    
    fmt.Println("Migration complete!")
}
```

### 6.2 Unit Tests

Create test files:
- `backend-go/pkg/storage/sqlite_local_test.go`
- `backend-go/internal/rag/sqlite_vector_store_test.go`
- `backend-go/pkg/storage/manager_test.go`

### 6.3 Integration Tests

**File**: `backend-go/integration_test.go`

Test scenarios:
- Local mode: CRUD operations on connections, queries, history
- Vector search: Index documents and search by similarity
- AI+RAG: Generate SQL using vector context
- Multi-database queries with local storage

---

## 📚 Phase 7: Documentation (TODO)

### 7.1 User Documentation

**Files to create**:

1. **`docs/STORAGE_ARCHITECTURE.md`** - Architecture overview
   - Local-first design
   - SQLite for everything
   - Team mode architecture (future)

2. **`docs/TEAM_MODE_SETUP.md`** - Team collaboration guide
   - How to set up Turso
   - Joining a team
   - Sync settings

3. **`docs/SQLITE_VECTOR_GUIDE.md`** - Technical details
   - How vector search works
   - Embedding storage
   - Performance tuning

4. **`docs/MIGRATION_FROM_QDRANT.md`** - Migration guide
   - Why we moved to SQLite
   - Migration tool usage
   - Rollback procedure

### 7.2 Update Existing Docs

**Files to update**:
- `docs/PART_2_AI_RAG_INTEGRATION.md` - Change Qdrant → SQLite
- `README.md` - Update setup instructions
- `AI_SETUP_GUIDE.md` - Update for storage-backed AI

---

## 🎯 Next Steps (Priority Order)

1. **Test full Wails build** to ensure no compilation errors
2. **Integrate storage with app.go** (Phase 3.2)
3. **Create migration tool** (Phase 6.1)  
4. **Write basic tests** (Phase 6.2)
5. **Update documentation** (Phase 7)
6. **Implement Turso team mode** (Phase 2.3 from original plan)

---

## 🔍 Testing Checklist

- [ ] Initialize local databases with `make init-local-db`
- [ ] Test connection CRUD operations
- [ ] Test query saving and history
- [ ] Test vector indexing and search
- [ ] Test AI query generation with RAG
- [ ] Test backup and restore
- [ ] Build desktop app with Wails
- [ ] Test app startup and shutdown
- [ ] Verify no data loss on restart

---

## ⚠️ Known Issues / TODO

1. **sqlite-vec extension**: Currently using pure Go implementation (BLOB storage). Consider adding actual extension for performance.

2. **Encryption**: Password encryption in connections table needs proper implementation (currently just stored as "encrypted").

3. **User ID generation**: Need to generate stable user ID from OS username or machine ID.

4. **Team mode**: Complete implementation pending (Turso integration).

5. **Migration from Qdrant**: Tool skeleton exists, needs full implementation.

6. **Schema cache TTL**: Need to implement automatic cache invalidation.

7. **Query history cleanup**: Add retention policy for old history records.

---

## 📈 Success Metrics

- ✅ SQLite vector store compiles and runs
- ✅ Storage layer compiles successfully
- ✅ Configuration loads correctly
- ✅ Databases initialize without errors
- ⏳ Full Wails app builds
- ⏳ App starts and connects to storage
- ⏳ AI/RAG works with SQLite backend
- ⏳ No external dependencies required for solo mode
- ⏳ All tests pass

---

## 🚀 Quick Start (Current Status)

```bash
# Initialize databases
make init-local-db

# Build and run (when integration complete)
make dev

# Backup databases
make backup-local-db

# Reset (WARNING: deletes data!)
make reset-local-db
```

**Data Location**: `~/.howlerops/`
- `local.db` - Connections, queries, settings
- `vectors.db` - Vector embeddings and RAG data
- `backups/` - Database backups
- `extensions/` - SQLite extensions (future)

---

*Last Updated: $(date)*

