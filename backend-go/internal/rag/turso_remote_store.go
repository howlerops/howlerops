package rag

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/sirupsen/logrus"
)

// TursoRemoteVectorStore is a minimal VectorStore implementation for remote (libSQL/Turso) instances.
// It focuses on idempotent upserts for IndexDocument/BatchIndexDocuments; other operations are no-ops
// or return not supported. This is sufficient for adaptive local-first sync without changing behavior.
type TursoRemoteVectorStore struct {
    db     *sql.DB
    logger *logrus.Logger
}

func NewTursoRemoteVectorStore(db *sql.DB, logger *logrus.Logger) *TursoRemoteVectorStore {
    return &TursoRemoteVectorStore{db: db, logger: logger}
}

func (s *TursoRemoteVectorStore) Initialize(ctx context.Context) error {
    // Ensure tables exist (schema mirrors SQLite vector store)
    stmts := []string{
        `CREATE TABLE IF NOT EXISTS documents (
            id TEXT PRIMARY KEY,
            connection_id TEXT NOT NULL,
            type TEXT NOT NULL,
            content TEXT NOT NULL,
            metadata TEXT,
            created_at INTEGER NOT NULL,
            updated_at INTEGER NOT NULL,
            access_count INTEGER DEFAULT 0,
            last_accessed INTEGER
        )`,
        `CREATE TABLE IF NOT EXISTS embeddings (
            document_id TEXT PRIMARY KEY,
            embedding BLOB NOT NULL,
            dimension INTEGER NOT NULL
        )`,
    }
    for _, stmt := range stmts {
        if _, err := s.db.ExecContext(ctx, stmt); err != nil {
            return fmt.Errorf("turso init failed: %w", err)
        }
    }
    return nil
}

func (s *TursoRemoteVectorStore) IndexDocument(ctx context.Context, doc *Document) error {
    if doc == nil || doc.ID == "" {
        return fmt.Errorf("invalid document")
    }
    now := time.Now().Unix()
    if doc.CreatedAt.IsZero() {
        doc.CreatedAt = time.Now()
    }
    if doc.UpdatedAt.IsZero() {
        doc.UpdatedAt = doc.CreatedAt
    }
    metadataJSON, _ := json.Marshal(doc.Metadata)

    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Idempotent upsert by primary key
    if _, err := tx.ExecContext(ctx, `
        INSERT INTO documents (id, connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            content = excluded.content,
            metadata = excluded.metadata,
            updated_at = excluded.updated_at
    `, doc.ID, doc.ConnectionID, string(doc.Type), doc.Content, string(metadataJSON), doc.CreatedAt.Unix(), doc.UpdatedAt.Unix(), doc.AccessCount, now); err != nil {
        return err
    }

    if len(doc.Embedding) > 0 {
        if _, err := tx.ExecContext(ctx, `
            INSERT INTO embeddings (document_id, embedding, dimension)
            VALUES (?, ?, ?)
            ON CONFLICT(document_id) DO UPDATE SET
                embedding = excluded.embedding,
                dimension = excluded.dimension
        `, doc.ID, serializeEmbedding(doc.Embedding), len(doc.Embedding)); err != nil {
            return err
        }
    }

    return tx.Commit()
}

func (s *TursoRemoteVectorStore) BatchIndexDocuments(ctx context.Context, docs []*Document) error {
    if len(docs) == 0 {
        return nil
    }
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    now := time.Now().Unix()
    for _, doc := range docs {
        if doc == nil || doc.ID == "" {
            continue
        }
        if doc.CreatedAt.IsZero() {
            doc.CreatedAt = time.Now()
        }
        if doc.UpdatedAt.IsZero() {
            doc.UpdatedAt = doc.CreatedAt
        }
        metadataJSON, _ := json.Marshal(doc.Metadata)
        if _, err := tx.ExecContext(ctx, `
            INSERT INTO documents (id, connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
            ON CONFLICT(id) DO UPDATE SET
                content = excluded.content,
                metadata = excluded.metadata,
                updated_at = excluded.updated_at
        `, doc.ID, doc.ConnectionID, string(doc.Type), doc.Content, string(metadataJSON), doc.CreatedAt.Unix(), doc.UpdatedAt.Unix(), doc.AccessCount, now); err != nil {
            return err
        }
        if len(doc.Embedding) > 0 {
            if _, err := tx.ExecContext(ctx, `
                INSERT INTO embeddings (document_id, embedding, dimension)
                VALUES (?, ?, ?)
                ON CONFLICT(document_id) DO UPDATE SET
                    embedding = excluded.embedding,
                    dimension = excluded.dimension
            `, doc.ID, serializeEmbedding(doc.Embedding), len(doc.Embedding)); err != nil {
                return err
            }
        }
    }
    return tx.Commit()
}

// The following methods are placeholders to satisfy the interface.
// Remote store is used for write-sync only in current design.

func (s *TursoRemoteVectorStore) GetDocument(ctx context.Context, id string) (*Document, error) { return nil, fmt.Errorf("not supported") }
func (s *TursoRemoteVectorStore) UpdateDocument(ctx context.Context, doc *Document) error { return fmt.Errorf("not supported") }
func (s *TursoRemoteVectorStore) DeleteDocument(ctx context.Context, id string) error { return fmt.Errorf("not supported") }
func (s *TursoRemoteVectorStore) SearchSimilar(ctx context.Context, embedding []float32, k int, filter map[string]interface{}) ([]*Document, error) {
    return []*Document{}, nil
}
func (s *TursoRemoteVectorStore) SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*Document, error) {
    return []*Document{}, nil
}
func (s *TursoRemoteVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error) {
    return []*Document{}, nil
}
func (s *TursoRemoteVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error { return nil }
func (s *TursoRemoteVectorStore) DeleteCollection(ctx context.Context, name string) error { return nil }
func (s *TursoRemoteVectorStore) ListCollections(ctx context.Context) ([]string, error) { return []string{}, nil }
func (s *TursoRemoteVectorStore) GetStats(ctx context.Context) (*VectorStoreStats, error) { return &VectorStoreStats{}, nil }
func (s *TursoRemoteVectorStore) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
    return &CollectionStats{Name: collection}, nil
}
func (s *TursoRemoteVectorStore) Optimize(ctx context.Context) error { return nil }
func (s *TursoRemoteVectorStore) Backup(ctx context.Context, path string) error { return fmt.Errorf("not supported") }
func (s *TursoRemoteVectorStore) Restore(ctx context.Context, path string) error { return fmt.Errorf("not supported") }


