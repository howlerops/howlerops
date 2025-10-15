package rag

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SQLiteVectorConfig holds SQLite vector store configuration
type SQLiteVectorConfig struct {
	Path          string        `json:"path"`
	Extension     string        `json:"extension"`      // sqlite-vec or sqlite-vss
	VectorSize    int           `json:"vector_size"`
	CacheSizeMB   int           `json:"cache_size_mb"`
	MMapSizeMB    int           `json:"mmap_size_mb"`
	WALEnabled    bool          `json:"wal_enabled"`
	Timeout       time.Duration `json:"timeout"`
}

// SQLiteVectorStore implements VectorStore using SQLite
type SQLiteVectorStore struct {
	db          *sql.DB
	collections map[string]*CollectionConfig
	logger      *logrus.Logger
	config      *SQLiteVectorConfig
}

// NewSQLiteVectorStore creates a new SQLite vector store
func NewSQLiteVectorStore(config *SQLiteVectorConfig, logger *logrus.Logger) (*SQLiteVectorStore, error) {
	// Open SQLite database
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?cache=shared&mode=rwc", config.Path))
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	// Configure SQLite performance settings
	pragmas := []string{
		fmt.Sprintf("PRAGMA cache_size = -%d", config.CacheSizeMB*1024), // Negative means KB
		fmt.Sprintf("PRAGMA mmap_size = %d", config.MMapSizeMB*1024*1024),
		fmt.Sprintf("PRAGMA busy_timeout = %d", config.Timeout.Milliseconds()),
		"PRAGMA synchronous = NORMAL",
		"PRAGMA temp_store = MEMORY",
		"PRAGMA journal_size_limit = 67110000",
	}

	if config.WALEnabled {
		pragmas = append(pragmas, "PRAGMA journal_mode = WAL")
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			logger.WithError(err).Warnf("Failed to set pragma: %s", pragma)
		}
	}

	store := &SQLiteVectorStore{
		db:          db,
		collections: make(map[string]*CollectionConfig),
		logger:      logger,
		config:      config,
	}

	return store, nil
}

// Initialize initializes the vector store
func (s *SQLiteVectorStore) Initialize(ctx context.Context) error {
	// Migrations should be run separately via migration files
	// This method just loads existing configuration
	
	// Load existing collections
	rows, err := s.db.QueryContext(ctx, `
		SELECT name, vector_size, distance, document_count, created_at, updated_at
		FROM collections
	`)
	if err != nil {
		return fmt.Errorf("failed to load collections: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var name, distance string
		var vectorSize, docCount int
		var createdAt, updatedAt int64

		if err := rows.Scan(&name, &vectorSize, &distance, &docCount, &createdAt, &updatedAt); err != nil {
			s.logger.WithError(err).Warn("Failed to scan collection")
			continue
		}

		s.collections[name] = &CollectionConfig{
			Name:       name,
			VectorSize: vectorSize,
			Distance:   distance,
		}
	}

	s.logger.WithField("collections", len(s.collections)).Info("SQLite vector store initialized")
	return nil
}

// IndexDocument indexes a single document
func (s *SQLiteVectorStore) IndexDocument(ctx context.Context, doc *Document) error {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}

	now := time.Now().Unix()
	if doc.CreatedAt.IsZero() {
		doc.CreatedAt = time.Now()
	}
	doc.UpdatedAt = time.Now()

	// Serialize metadata to JSON
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Start transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert/update document
	_, err = tx.ExecContext(ctx, `
		INSERT INTO documents (id, connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at
	`, doc.ID, doc.ConnectionID, string(doc.Type), doc.Content, string(metadataJSON),
		doc.CreatedAt.Unix(), doc.UpdatedAt.Unix(), doc.AccessCount, now)

	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	// Insert/update embedding if present
	if len(doc.Embedding) > 0 {
		embeddingBytes := serializeEmbedding(doc.Embedding)
		_, err = tx.ExecContext(ctx, `
			INSERT INTO embeddings (document_id, embedding, dimension)
			VALUES (?, ?, ?)
			ON CONFLICT(document_id) DO UPDATE SET
				embedding = excluded.embedding,
				dimension = excluded.dimension
		`, doc.ID, embeddingBytes, len(doc.Embedding))

		if err != nil {
			return fmt.Errorf("failed to insert embedding: %w", err)
		}
	}

	// Update collection count
	collection := s.getCollectionForType(doc.Type)
	_, err = tx.ExecContext(ctx, `
		UPDATE collections
		SET document_count = document_count + 1, updated_at = ?
		WHERE name = ?
	`, now, collection)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to update collection count")
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"document_id": doc.ID,
		"type":        doc.Type,
		"collection":  collection,
	}).Debug("Document indexed")

	return nil
}

// BatchIndexDocuments indexes multiple documents
func (s *SQLiteVectorStore) BatchIndexDocuments(ctx context.Context, docs []*Document) error {
	if len(docs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().Unix()

	// Prepare statements
	docStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO documents (id, connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			content = excluded.content,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare document statement: %w", err)
	}
	defer docStmt.Close()

	embStmt, err := tx.PrepareContext(ctx, `
		INSERT INTO embeddings (document_id, embedding, dimension)
		VALUES (?, ?, ?)
		ON CONFLICT(document_id) DO UPDATE SET
			embedding = excluded.embedding,
			dimension = excluded.dimension
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare embedding statement: %w", err)
	}
	defer embStmt.Close()

	// Insert documents
	for _, doc := range docs {
		if doc.ID == "" {
			doc.ID = uuid.New().String()
		}

		if doc.CreatedAt.IsZero() {
			doc.CreatedAt = time.Now()
		}
		doc.UpdatedAt = time.Now()

		metadataJSON, err := json.Marshal(doc.Metadata)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to marshal metadata")
			continue
		}

		_, err = docStmt.ExecContext(ctx, doc.ID, doc.ConnectionID, string(doc.Type), doc.Content,
			string(metadataJSON), doc.CreatedAt.Unix(), doc.UpdatedAt.Unix(), doc.AccessCount, now)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to insert document")
			continue
		}

		if len(doc.Embedding) > 0 {
			embeddingBytes := serializeEmbedding(doc.Embedding)
			_, err = embStmt.ExecContext(ctx, doc.ID, embeddingBytes, len(doc.Embedding))
			if err != nil {
				s.logger.WithError(err).Warn("Failed to insert embedding")
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithField("count", len(docs)).Debug("Batch indexed documents")
	return nil
}

// SearchSimilar searches for similar documents using vector similarity
func (s *SQLiteVectorStore) SearchSimilar(ctx context.Context, embedding []float32, k int, filter map[string]interface{}) ([]*Document, error) {
	// Get all documents with embeddings
	query := `
		SELECT d.id, d.connection_id, d.type, d.content, d.metadata, d.created_at, d.updated_at, d.access_count, d.last_accessed, e.embedding
		FROM documents d
		INNER JOIN embeddings e ON d.id = e.document_id
	`

	args := []interface{}{}

	// Apply filters
	if filter != nil {
		whereClauses := []string{}
		if connID, ok := filter["connection_id"].(string); ok {
			whereClauses = append(whereClauses, "d.connection_id = ?")
			args = append(args, connID)
		}
		if docType, ok := filter["type"].(string); ok {
			whereClauses = append(whereClauses, "d.type = ?")
			args = append(args, docType)
		}

		if len(whereClauses) > 0 {
			query += " WHERE " + whereClauses[0]
			for i := 1; i < len(whereClauses); i++ {
				query += " AND " + whereClauses[i]
			}
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	// Calculate similarities
	type docWithScore struct {
		doc   *Document
		score float32
	}
	var results []docWithScore

	for rows.Next() {
		var id, connID, docType, content, metadataStr string
		var createdAt, updatedAt, lastAccessed int64
		var accessCount int
		var embeddingBytes []byte

		err := rows.Scan(&id, &connID, &docType, &content, &metadataStr, &createdAt, &updatedAt, &accessCount, &lastAccessed, &embeddingBytes)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan document")
			continue
		}

		docEmbedding := deserializeEmbedding(embeddingBytes)
		similarity := cosineSimilarity(embedding, docEmbedding)

		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			metadata = make(map[string]interface{})
		}

		doc := &Document{
			ID:           id,
			ConnectionID: connID,
			Type:         DocumentType(docType),
			Content:      content,
			Embedding:    docEmbedding,
			Metadata:     metadata,
			CreatedAt:    time.Unix(createdAt, 0),
			UpdatedAt:    time.Unix(updatedAt, 0),
			AccessCount:  accessCount,
			LastAccessed: time.Unix(lastAccessed, 0),
			Score:        similarity,
		}

		results = append(results, docWithScore{doc: doc, score: similarity})
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	// Limit to top k
	if len(results) > k {
		results = results[:k]
	}

	// Extract documents
	docs := make([]*Document, len(results))
	for i, r := range results {
		docs[i] = r.doc
	}

	return docs, nil
}

// SearchByText performs text-based search using FTS5
func (s *SQLiteVectorStore) SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*Document, error) {
	sqlQuery := `
		SELECT d.id, d.connection_id, d.type, d.content, d.metadata, d.created_at, d.updated_at, d.access_count, d.last_accessed
		FROM documents d
		INNER JOIN documents_fts fts ON d.id = fts.document_id
		WHERE documents_fts MATCH ?
	`

	args := []interface{}{query}

	// Apply additional filters
	if filter != nil {
		if connID, ok := filter["connection_id"].(string); ok {
			sqlQuery += " AND d.connection_id = ?"
			args = append(args, connID)
		}
		if docType, ok := filter["type"].(string); ok {
			sqlQuery += " AND d.type = ?"
			args = append(args, docType)
		}
	}

	sqlQuery += fmt.Sprintf(" ORDER BY rank LIMIT %d", k)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search text: %w", err)
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var id, connID, docType, content, metadataStr string
		var createdAt, updatedAt, lastAccessed int64
		var accessCount int

		err := rows.Scan(&id, &connID, &docType, &content, &metadataStr, &createdAt, &updatedAt, &accessCount, &lastAccessed)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan document")
			continue
		}

		var metadata map[string]interface{}
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			metadata = make(map[string]interface{})
		}

		doc := &Document{
			ID:           id,
			ConnectionID: connID,
			Type:         DocumentType(docType),
			Content:      content,
			Metadata:     metadata,
			CreatedAt:    time.Unix(createdAt, 0),
			UpdatedAt:    time.Unix(updatedAt, 0),
			AccessCount:  accessCount,
			LastAccessed: time.Unix(lastAccessed, 0),
		}

		docs = append(docs, doc)
	}

	return docs, nil
}

// HybridSearch combines vector and text search
func (s *SQLiteVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error) {
	// Perform both searches
	vectorResults, err := s.SearchSimilar(ctx, embedding, k*2, nil)
	if err != nil {
		return nil, err
	}

	textResults, err := s.SearchByText(ctx, query, k*2, nil)
	if err != nil {
		s.logger.WithError(err).Warn("Text search failed, using vector results only")
		textResults = []*Document{}
	}

	// Merge and deduplicate results
	seen := make(map[string]bool)
	var merged []*Document

	// Add vector results first (higher weight)
	for _, doc := range vectorResults {
		if !seen[doc.ID] {
			merged = append(merged, doc)
			seen[doc.ID] = true
		}
	}

	// Add text results
	for _, doc := range textResults {
		if !seen[doc.ID] {
			doc.Score = 0.5 // Lower score for text-only matches
			merged = append(merged, doc)
			seen[doc.ID] = true
		}
	}

	// Limit to k
	if len(merged) > k {
		merged = merged[:k]
	}

	return merged, nil
}

// GetDocument retrieves a document by ID
func (s *SQLiteVectorStore) GetDocument(ctx context.Context, id string) (*Document, error) {
	var connID, docType, content, metadataStr string
	var createdAt, updatedAt, lastAccessed int64
	var accessCount int

	err := s.db.QueryRowContext(ctx, `
		SELECT connection_id, type, content, metadata, created_at, updated_at, access_count, last_accessed
		FROM documents
		WHERE id = ?
	`, id).Scan(&connID, &docType, &content, &metadataStr, &createdAt, &updatedAt, &accessCount, &lastAccessed)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("document not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
		metadata = make(map[string]interface{})
	}

	doc := &Document{
		ID:           id,
		ConnectionID: connID,
		Type:         DocumentType(docType),
		Content:      content,
		Metadata:     metadata,
		CreatedAt:    time.Unix(createdAt, 0),
		UpdatedAt:    time.Unix(updatedAt, 0),
		AccessCount:  accessCount,
		LastAccessed: time.Unix(lastAccessed, 0),
	}

	// Try to get embedding
	var embeddingBytes []byte
	err = s.db.QueryRowContext(ctx, `SELECT embedding FROM embeddings WHERE document_id = ?`, id).Scan(&embeddingBytes)
	if err == nil {
		doc.Embedding = deserializeEmbedding(embeddingBytes)
	}

	return doc, nil
}

// UpdateDocument updates an existing document
func (s *SQLiteVectorStore) UpdateDocument(ctx context.Context, doc *Document) error {
	doc.UpdatedAt = time.Now()
	return s.IndexDocument(ctx, doc)
}

// DeleteDocument deletes a document
func (s *SQLiteVectorStore) DeleteDocument(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM documents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Embedding is deleted automatically via CASCADE
	s.logger.WithField("document_id", id).Debug("Document deleted")
	return nil
}

// CreateCollection creates a new collection
func (s *SQLiteVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	now := time.Now().Unix()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO collections (name, vector_size, distance, document_count, created_at, updated_at)
		VALUES (?, ?, ?, 0, ?, ?)
	`, name, dimension, "cosine", now, now)

	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	s.collections[name] = &CollectionConfig{
		Name:       name,
		VectorSize: dimension,
		Distance:   "cosine",
	}

	s.logger.WithField("collection", name).Info("Collection created")
	return nil
}

// DeleteCollection deletes a collection
func (s *SQLiteVectorStore) DeleteCollection(ctx context.Context, name string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM collections WHERE name = ?`, name)
	if err != nil {
		return fmt.Errorf("failed to delete collection: %w", err)
	}

	delete(s.collections, name)
	s.logger.WithField("collection", name).Info("Collection deleted")
	return nil
}

// ListCollections lists all collections
func (s *SQLiteVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT name FROM collections ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		names = append(names, name)
	}

	return names, nil
}

// GetStats returns vector store statistics
func (s *SQLiteVectorStore) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	var totalDocs int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM documents`).Scan(&totalDocs)
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}

	stats := &VectorStoreStats{
		TotalDocuments:   totalDocs,
		TotalCollections: len(s.collections),
		LastOptimized:    time.Now(), // TODO: Track this properly
	}

	return stats, nil
}

// GetCollectionStats returns collection-specific statistics
func (s *SQLiteVectorStore) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
	var name, distance string
	var vectorSize, docCount int
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, `
		SELECT name, vector_size, distance, document_count, created_at, updated_at
		FROM collections
		WHERE name = ?
	`, collection).Scan(&name, &vectorSize, &distance, &docCount, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("collection not found: %s", collection)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collection stats: %w", err)
	}

	return &CollectionStats{
		Name:          name,
		DocumentCount: int64(docCount),
		Dimension:     vectorSize,
		LastUpdated:   time.Unix(updatedAt, 0),
	}, nil
}

// Optimize optimizes the vector store
func (s *SQLiteVectorStore) Optimize(ctx context.Context) error {
	// Run VACUUM to reclaim space
	if _, err := s.db.ExecContext(ctx, `VACUUM`); err != nil {
		s.logger.WithError(err).Warn("Failed to VACUUM database")
	}

	// Optimize FTS5 index
	if _, err := s.db.ExecContext(ctx, `INSERT INTO documents_fts(documents_fts) VALUES('optimize')`); err != nil {
		s.logger.WithError(err).Warn("Failed to optimize FTS5 index")
	}

	// Analyze for query planner
	if _, err := s.db.ExecContext(ctx, `ANALYZE`); err != nil {
		s.logger.WithError(err).Warn("Failed to ANALYZE database")
	}

	s.logger.Info("Vector store optimization completed")
	return nil
}

// Backup creates a backup of the vector store
func (s *SQLiteVectorStore) Backup(ctx context.Context, path string) error {
	// SQLite backup using VACUUM INTO
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`VACUUM INTO '%s'`, path))
	if err != nil {
		return fmt.Errorf("failed to backup database: %w", err)
	}

	s.logger.WithField("path", path).Info("Backup created")
	return nil
}

// Restore restores from a backup
func (s *SQLiteVectorStore) Restore(ctx context.Context, path string) error {
	// Close current database
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}

	// Copy backup file to current path
	// This is OS-specific, implement using appropriate file operations
	return fmt.Errorf("restore not yet implemented")
}

// Helper functions

func (s *SQLiteVectorStore) getCollectionForType(docType DocumentType) string {
	switch docType {
	case DocumentTypeSchema:
		return "schemas"
	case DocumentTypeQuery, DocumentTypePlan:
		return "queries"
	case DocumentTypePerformance:
		return "performance"
	case DocumentTypeBusiness:
		return "business"
	default:
		return "queries"
	}
}

// serializeEmbedding converts float32 slice to bytes
func serializeEmbedding(embedding []float32) []byte {
	bytes := make([]byte, len(embedding)*4)
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(v))
	}
	return bytes
}

// deserializeEmbedding converts bytes to float32 slice
func deserializeEmbedding(bytes []byte) []float32 {
	embedding := make([]float32, len(bytes)/4)
	for i := range embedding {
		bits := binary.LittleEndian.Uint32(bytes[i*4:])
		embedding[i] = math.Float32frombits(bits)
	}
	return embedding
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

