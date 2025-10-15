# Migration Guide: Qdrant → SQLite

## Why We Migrated

HowlerOps moved from Qdrant to SQLite for vector storage to achieve:

✅ **Zero Dependencies** - No external services required  
✅ **Local-First** - Works completely offline  
✅ **Embedded** - Single-file database  
✅ **Simpler Setup** - No Docker containers  
✅ **Better Desktop Experience** - Fast, lightweight  
✅ **Team-Ready** - Future Turso integration for collaboration  

## Migration Status

**Current Status**: Qdrant code is deprecated but kept for reference.

**Migration Tool**: Available at `backend-go/cmd/migrate-vector-db/main.go`

**Note**: Most users won't need to migrate as fresh installations use SQLite by default.

## For New Users

**Skip this guide!** You're already using SQLite. Just run:

```bash
make init-local-db
```

## For Existing Qdrant Users

### Step 1: Backup Your Qdrant Data

```bash
# If using Docker Qdrant
docker exec qdrant ./qdrant-backup /backups/qdrant-backup-$(date +%Y%m%d)
```

### Step 2: Run Migration Tool

```bash
cd backend-go
go run cmd/migrate-vector-db/main.go \
  --from=qdrant \
  --to=sqlite \
  --qdrant-host=localhost \
  --qdrant-port=6333 \
  --sqlite-path=~/.howlerops/vectors.db \
  --batch-size=100
```

**Options**:
- `--dry-run` - Test without writing
- `--verbose` - Detailed logging
- `--batch-size` - Documents per batch (default: 100)

### Step 3: Verify Migration

```bash
# Check document counts
sqlite3 ~/.howlerops/vectors.db \
  "SELECT type, COUNT(*) as count FROM documents GROUP BY type;"

# Expected output:
# schema|150
# query|500
# business|25
# performance|75
```

### Step 4: Test RAG Functionality

1. Open HowlerOps
2. Go to Settings → AI
3. Test natural language query generation
4. Verify suggestions are relevant

### Step 5: Remove Qdrant (Optional)

Once verified, you can remove Qdrant:

```bash
# Stop Qdrant container
docker stop qdrant
docker rm qdrant

# Remove Qdrant data volume
docker volume rm qdrant-data

# Update docker-compose.yml (remove qdrant service)
```

## Manual Migration

If automatic migration doesn't work:

### Export from Qdrant

```python
# export_qdrant.py
from qdrant_client import QdrantClient

client = QdrantClient(host="localhost", port=6333)
collections = client.get_collections().collections

for collection in collections:
    print(f"Exporting {collection.name}...")
    points = client.scroll(
        collection_name=collection.name,
        limit=1000,
        with_vectors=True,
        with_payload=True,
    )
    
    # Save to JSON
    import json
    with open(f"{collection.name}.json", "w") as f:
        json.dump([{
            "id": point.id,
            "vector": point.vector,
            "payload": point.payload,
        } for point in points[0]], f)
```

### Import to SQLite

```go
// import_sqlite.go
package main

import (
    "context"
    "encoding/json"
    "io/ioutil"
    
    "github.com/sql-studio/backend-go/internal/rag"
)

func main() {
    // Initialize SQLite store
    store, _ := rag.NewSQLiteVectorStore(...)
    
    // Read exported JSON
    data, _ := ioutil.ReadFile("schemas.json")
    var documents []struct {
        ID      string
        Vector  []float32
        Payload map[string]interface{}
    }
    json.Unmarshal(data, &documents)
    
    // Import
    for _, doc := range documents {
        ragDoc := &rag.Document{
            ID:           doc.ID,
            Embedding:    doc.Vector,
            ConnectionID: doc.Payload["connection_id"].(string),
            Type:         rag.DocumentType(doc.Payload["type"].(string)),
            Content:      doc.Payload["content"].(string),
            Metadata:     doc.Payload,
        }
        store.IndexDocument(context.Background(), ragDoc)
    }
}
```

## Comparison: Qdrant vs SQLite

| Feature | Qdrant | SQLite |
|---------|---------|---------|
| **Setup** | Docker required | Embedded |
| **Dependencies** | External service | None |
| **Offline** | No | Yes |
| **Performance** | Excellent | Very Good |
| **Scalability** | Horizontal | Vertical |
| **Team Sync** | Complex | Turso (future) |
| **Backup** | Container volumes | Simple file copy |
| **Memory** | ~500MB | ~50MB |

## Troubleshooting

### Migration Fails

**Issue**: Connection refused to Qdrant

**Solution**: Ensure Qdrant is running:
```bash
docker ps | grep qdrant
```

**Issue**: Permission denied writing to SQLite

**Solution**: Check directory permissions:
```bash
ls -la ~/.howlerops/
chmod 700 ~/.howlerops/
```

### Missing Documents After Migration

**Issue**: Fewer documents in SQLite than Qdrant

**Solutions**:
1. Check migration logs for errors
2. Verify Qdrant collections list
3. Re-run with `--verbose` flag
4. Check SQLite database:
   ```bash
   sqlite3 ~/.howlerops/vectors.db "SELECT COUNT(*) FROM documents;"
   ```

### RAG Not Working After Migration

**Issue**: No results from vector search

**Solutions**:
1. Verify embeddings exist:
   ```sql
   SELECT COUNT(*) FROM embeddings;
   ```
2. Re-index documents:
   ```bash
   go run backend-go/cmd/reindex/main.go
   ```
3. Check AI service logs for errors

## Rollback Plan

If migration fails and you need Qdrant back:

### 1. Restore Qdrant

```bash
# Start Qdrant container
docker run -d \
  --name qdrant \
  -p 6333:6333 \
  -v qdrant-data:/qdrant/storage \
  qdrant/qdrant:latest

# Restore backup
docker exec qdrant ./qdrant-restore /backups/qdrant-backup-YYYYMMDD
```

### 2. Update Configuration

```yaml
# backend-go/configs/config.yaml
rag:
  vector_store:
    type: "qdrant"  # Change back to qdrant
    qdrant:
      host: "localhost"
      port: 6333
```

### 3. Restart Application

```bash
make dev
```

## Post-Migration Checklist

- [ ] All collections migrated
- [ ] Document counts match
- [ ] RAG/AI queries work
- [ ] Vector search returns relevant results
- [ ] Application starts without errors
- [ ] No Qdrant containers running
- [ ] SQLite files backed up
- [ ] Team informed (if applicable)

## Getting Help

If you encounter issues:

1. Check logs: `~/.howlerops/logs/howlerops.log`
2. Run diagnostics: `make doctor`
3. Create GitHub issue with:
   - Migration command used
   - Error messages
   - Document counts (before/after)
   - SQLite version: `sqlite3 --version`

## Future: Team Mode

Once Turso team mode is implemented, you'll be able to:

- Share vector data across team
- Sync RAG learnings
- Collaborate on query library
- Maintain local copies for offline work

Migration to team mode will be seamless - just enable in settings!

---

**Last Updated**: October 2025  
**Migration Tool Version**: 1.0.0  
**Recommended SQLite Version**: 3.45.0+

