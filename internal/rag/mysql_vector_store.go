package rag

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

// MySQLVectorConfig holds MySQL vector store configuration
type MySQLVectorConfig struct {
	DSN        string `json:"dsn"`
	VectorSize int    `json:"vector_size"`
}

// mysqlVectorSchema defines the tables used for the vector store
const mysqlVectorSchema = `
CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR(191) PRIMARY KEY,
    connection_id VARCHAR(191) NOT NULL,
    type VARCHAR(64) NOT NULL,
    content LONGTEXT NOT NULL,
    metadata JSON NULL,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,
    access_count BIGINT DEFAULT 0,
    last_accessed BIGINT DEFAULT 0
);

CREATE TABLE IF NOT EXISTS embeddings (
    document_id VARCHAR(191) PRIMARY KEY,
    embedding LONGBLOB NOT NULL,
    dimension INT NOT NULL,
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS collections (
    name VARCHAR(191) PRIMARY KEY,
    vector_size INT NOT NULL,
    distance VARCHAR(32) NOT NULL,
    document_count BIGINT DEFAULT 0,
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL
);
`

// MySQLVectorStore implements VectorStore using MySQL
type MySQLVectorStore struct {
	db     *sql.DB
	config *MySQLVectorConfig
	logger *logrus.Logger
}

// NewMySQLVectorStore creates a new MySQL vector store
func NewMySQLVectorStore(config *MySQLVectorConfig, logger *logrus.Logger) (*MySQLVectorStore, error) {
	if config.DSN == "" {
		return nil, fmt.Errorf("mysql vector store requires DSN")
	}
	if config.VectorSize == 0 {
		config.VectorSize = 1536
	}

	db, err := sql.Open("mysql", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection: %w", err)
	}

	store := &MySQLVectorStore{
		db:     db,
		config: config,
		logger: logger,
	}

	return store, nil
}

// Initialize creates tables and default collections
func (s *MySQLVectorStore) Initialize(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, mysqlVectorSchema); err != nil {
		return fmt.Errorf("failed to initialize mysql vector schema: %w", err)
	}

	stmt := `
INSERT INTO collections (name, vector_size, distance, document_count, created_at, updated_at)
VALUES (?, ?, ?, 0, ?, ?)
ON DUPLICATE KEY UPDATE vector_size = VALUES(vector_size)`

	now := time.Now().Unix()
	collections := []struct {
		Name     string
		Distance string
	}{
		{"schemas", "cosine"},
		{"queries", "cosine"},
		{"performance", "euclidean"},
		{"business", "cosine"},
		{"memory", "cosine"},
	}

	for _, col := range collections {
		if _, err := s.db.ExecContext(ctx, stmt, col.Name, s.config.VectorSize, col.Distance, now, now); err != nil {
			s.logger.WithError(err).Warn("Failed to ensure collection in MySQL vector store")
		}
	}

	return nil
}

// IndexDocument inserts or updates a document and its embedding
func (s *MySQLVectorStore) IndexDocument(ctx context.Context, doc *Document) error {
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Best-effort rollback

	now := time.Now().Unix()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	if doc.UpdatedAt.IsZero() {
		doc.UpdatedAt = doc.CreatedAt
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO documents (id, connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    content = VALUES(content),
    metadata = VALUES(metadata),
    updated_at = VALUES(updated_at)
`, doc.ID, doc.ConnectionID, string(doc.Type), doc.Content, string(metadataJSON), doc.CreatedAt.Unix(), doc.UpdatedAt.Unix(), doc.AccessCount, now)
	if err != nil {
		return fmt.Errorf("failed to upsert document: %w", err)
	}

	if len(doc.Embedding) > 0 {
		embeddingBytes := serializeEmbedding(doc.Embedding)
		_, err = tx.ExecContext(ctx, `
INSERT INTO embeddings (document_id, embedding, dimension)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE
    embedding = VALUES(embedding),
    dimension = VALUES(dimension)
`, doc.ID, embeddingBytes, len(doc.Embedding))
		if err != nil {
			return fmt.Errorf("failed to upsert embedding: %w", err)
		}
	}

	collection := s.collectionForType(doc.Type)
	_, err = tx.ExecContext(ctx, `
UPDATE collections
SET document_count = document_count + 1, updated_at = ?
WHERE name = ?`, now, collection)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to update collection count")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BatchIndexDocuments indexes documents one by one
func (s *MySQLVectorStore) BatchIndexDocuments(ctx context.Context, docs []*Document) error {
	for _, doc := range docs {
		if err := s.IndexDocument(ctx, doc); err != nil {
			return err
		}
	}
	return nil
}

// SearchSimilar performs a cosine similarity search in memory
func (s *MySQLVectorStore) SearchSimilar(ctx context.Context, embedding []float32, k int, filter map[string]interface{}) ([]*Document, error) {
	if len(embedding) == 0 {
		return []*Document{}, nil
	}

	queryBuilder := strings.Builder{}
	args := []interface{}{}

	queryBuilder.WriteString(`
SELECT d.id, d.connection_id, d.type, d.content, d.metadata, d.created_at, d.updated_at, d.access_count, d.last_accessed, e.embedding
FROM documents d
INNER JOIN embeddings e ON d.id = e.document_id
`)

	conditions := []string{}
	if filter != nil {
		if conn, ok := filter["connection_id"].(string); ok && conn != "" {
			conditions = append(conditions, "d.connection_id = ?")
			args = append(args, conn)
		}
		if docType, ok := filter["type"].(string); ok && docType != "" {
			conditions = append(conditions, "d.type = ?")
			args = append(args, docType)
		}
	}

	if len(conditions) > 0 {
		queryBuilder.WriteString("WHERE ")
		queryBuilder.WriteString(strings.Join(conditions, " AND "))
	}

	queryBuilder.WriteString(" ORDER BY d.updated_at DESC LIMIT ?")
	args = append(args, max(k*5, 50))

	rows, err := s.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	results := []*Document{}
	for rows.Next() {
		var (
			id, connectionID, docType, content string
			metadataJSON                       sql.NullString
			createdAt, updatedAt, accessCount  int64
			lastAccessed                       sql.NullInt64
			embeddingBytes                     []byte
		)

		if err := rows.Scan(&id, &connectionID, &docType, &content, &metadataJSON, &createdAt, &updatedAt, &accessCount, &lastAccessed, &embeddingBytes); err != nil {
			return nil, fmt.Errorf("failed to scan document row: %w", err)
		}

		docEmbedding := deserializeEmbedding(embeddingBytes)
		score := cosineSimilarity(embedding, docEmbedding)

		metadata := map[string]interface{}{}
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal metadata")
			}
		}

		doc := &Document{
			ID:           id,
			ConnectionID: connectionID,
			Type:         DocumentType(docType),
			Content:      content,
			Embedding:    docEmbedding,
			Metadata:     metadata,
			CreatedAt:    time.Unix(createdAt, 0),
			UpdatedAt:    time.Unix(updatedAt, 0),
			AccessCount:  int(accessCount),
			Score:        score,
		}
		if lastAccessed.Valid {
			doc.LastAccessed = time.Unix(lastAccessed.Int64, 0)
		}
		results = append(results, doc)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > k {
		results = results[:k]
	}
	return results, nil
}

// SearchByText performs a LIKE search on content
func (s *MySQLVectorStore) SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*Document, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*Document{}, nil
	}

	queryBuilder := strings.Builder{}
	args := []interface{}{}

	queryBuilder.WriteString(`
SELECT id, connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed
FROM documents
WHERE content LIKE ?`)
	args = append(args, "%"+query+"%")

	if filter != nil {
		if conn, ok := filter["connection_id"].(string); ok && conn != "" {
			queryBuilder.WriteString(" AND connection_id = ?")
			args = append(args, conn)
		}
		if docType, ok := filter["type"].(string); ok && docType != "" {
			queryBuilder.WriteString(" AND type = ?")
			args = append(args, docType)
		}
	}

	queryBuilder.WriteString(" ORDER BY updated_at DESC LIMIT ?")
	args = append(args, k)

	rows, err := s.db.QueryContext(ctx, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search documents by text: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	var results []*Document
	for rows.Next() {
		var (
			id, connectionID, docType, content string
			metadataJSON                       sql.NullString
			createdAt, updatedAt, accessCount  int64
			lastAccessed                       sql.NullInt64
		)

		if err := rows.Scan(&id, &connectionID, &docType, &content, &metadataJSON, &createdAt, &updatedAt, &accessCount, &lastAccessed); err != nil {
			return nil, fmt.Errorf("failed to scan text search row: %w", err)
		}

		metadata := map[string]interface{}{}
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal metadata for text search")
			}
		}

		doc := &Document{
			ID:           id,
			ConnectionID: connectionID,
			Type:         DocumentType(docType),
			Content:      content,
			Metadata:     metadata,
			CreatedAt:    time.Unix(createdAt, 0),
			UpdatedAt:    time.Unix(updatedAt, 0),
			AccessCount:  int(accessCount),
		}
		if lastAccessed.Valid {
			doc.LastAccessed = time.Unix(lastAccessed.Int64, 0)
		}

		results = append(results, doc)
	}

	return results, nil
}

// HybridSearch combines vector and text search
func (s *MySQLVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error) {
	vectorResults, err := s.SearchSimilar(ctx, embedding, k*2, nil)
	if err != nil {
		return nil, err
	}

	textResults, err := s.SearchByText(ctx, query, k, nil)
	if err != nil {
		s.logger.WithError(err).Warn("Text search failed, returning vector results only")
		return vectorResults, nil
	}

	seen := make(map[string]bool)
	results := make([]*Document, 0, k)

	for _, doc := range vectorResults {
		if !seen[doc.ID] {
			results = append(results, doc)
			seen[doc.ID] = true
		}
		if len(results) >= k {
			return results, nil
		}
	}

	for _, doc := range textResults {
		if !seen[doc.ID] {
			doc.Score = 0.5
			results = append(results, doc)
			seen[doc.ID] = true
		}
		if len(results) >= k {
			break
		}
	}

	return results, nil
}

// GetDocument returns a document by ID
func (s *MySQLVectorStore) GetDocument(ctx context.Context, id string) (*Document, error) {
	var (
		connectionID, docType, content string
		metadataJSON                   sql.NullString
		createdAt, updatedAt           int64
		accessCount                    int64
		lastAccessed                   sql.NullInt64
	)

	err := s.db.QueryRowContext(ctx, `
SELECT connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed
FROM documents
WHERE id = ?`, id).Scan(&connectionID, &docType, &content, &metadataJSON, &createdAt, &updatedAt, &accessCount, &lastAccessed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	metadata := map[string]interface{}{}
	doc := &Document{
		ID:           id,
		ConnectionID: connectionID,
		Type:         DocumentType(docType),
		Content:      content,
		Metadata:     metadata,
		CreatedAt:    time.Unix(createdAt, 0),
		UpdatedAt:    time.Unix(updatedAt, 0),
		AccessCount:  int(accessCount),
	}
	if lastAccessed.Valid {
		doc.LastAccessed = time.Unix(lastAccessed.Int64, 0)
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal metadata for document")
		}
	}

	doc.Metadata = metadata

	var embeddingBytes []byte
	if err := s.db.QueryRowContext(ctx, `SELECT embedding FROM embeddings WHERE document_id = ?`, id).Scan(&embeddingBytes); err == nil {
		doc.Embedding = deserializeEmbedding(embeddingBytes)
	}

	return doc, nil
}

// UpdateDocument delegates to IndexDocument
func (s *MySQLVectorStore) UpdateDocument(ctx context.Context, doc *Document) error {
	return s.IndexDocument(ctx, doc)
}

// DeleteDocument removes a document
func (s *MySQLVectorStore) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	return err
}

// CreateCollection is supported for compatibility
func (s *MySQLVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO collections (name, vector_size, distance, document_count, created_at, updated_at)
VALUES (?, ?, 'cosine', 0, ?, ?)
ON DUPLICATE KEY UPDATE vector_size = VALUES(vector_size)
`, name, dimension, time.Now().Unix(), time.Now().Unix())
	return err
}

// DeleteCollection removes a collection
func (s *MySQLVectorStore) DeleteCollection(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE name = ?`, name)
	return err
}

// ListCollections returns all collections
func (s *MySQLVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM collections`)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan collection name: %w", err)
		}
		names = append(names, name)
	}
	return names, nil
}

// GetStats returns minimal stats for the store
func (s *MySQLVectorStore) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	var totalDocs int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&totalDocs)
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %w", err)
	}
	return &VectorStoreStats{
		TotalDocuments:   totalDocs,
		TotalCollections: 5,
	}, nil
}

// GetCollectionStats returns stats for a collection
func (s *MySQLVectorStore) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
	var (
		count, vectorSize, updatedAt int64
	)
	err := s.db.QueryRowContext(ctx, `
SELECT document_count, vector_size, updated_at FROM collections WHERE name = ?`, collection).
		Scan(&count, &vectorSize, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	return &CollectionStats{
		Name:          collection,
		DocumentCount: count,
		Dimension:     int(vectorSize),
		LastUpdated:   time.Unix(updatedAt, 0),
	}, nil
}

// Optimize is a no-op for MySQL
func (s *MySQLVectorStore) Optimize(ctx context.Context) error {
	return nil
}

// Backup is not implemented for MySQL
func (s *MySQLVectorStore) Backup(ctx context.Context, path string) error {
	return fmt.Errorf("backup not implemented for MySQL vector store")
}

// Restore is not implemented for MySQL
func (s *MySQLVectorStore) Restore(ctx context.Context, path string) error {
	return fmt.Errorf("restore not implemented for MySQL vector store")
}

// collectionForType maps document type to collection name
func (s *MySQLVectorStore) collectionForType(docType DocumentType) string {
	switch docType {
	case DocumentTypeSchema:
		return "schemas"
	case DocumentTypeQuery, DocumentTypePlan:
		return "queries"
	case DocumentTypePerformance:
		return "performance"
	case DocumentTypeBusiness:
		return "business"
	case DocumentTypeMemory:
		return "memory"
	default:
		return "queries"
	}
}

// max returns the maximum of two ints
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// StoreDocumentWithoutEmbedding stores a document without embedding (stub for MySQL)
func (s *MySQLVectorStore) StoreDocumentWithoutEmbedding(ctx context.Context, doc *Document) error {
	return fmt.Errorf("hierarchical indexing not yet implemented for MySQL")
}

// GetDocumentsBatch retrieves multiple documents (stub for MySQL)
func (s *MySQLVectorStore) GetDocumentsBatch(ctx context.Context, ids []string) ([]*Document, error) {
	return nil, fmt.Errorf("batch retrieval not yet implemented for MySQL")
}

// UpdateDocumentMetadata updates document metadata (stub for MySQL)
func (s *MySQLVectorStore) UpdateDocumentMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	return fmt.Errorf("metadata update not yet implemented for MySQL")
}
