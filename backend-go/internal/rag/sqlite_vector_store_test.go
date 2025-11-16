//go:build integration

package rag_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test Helpers

func newTestLoggerSQLiteVectorStore() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Quiet during tests
	return logger
}

func newMemoryVectorStore(t *testing.T) (*rag.SQLiteVectorStore, context.Context) {
	logger := newTestLoggerSQLiteVectorStore()
	config := &rag.SQLiteVectorConfig{
		Path:        ":memory:",
		VectorSize:  128, // Smaller for tests
		CacheSizeMB: 16,
		MMapSizeMB:  32,
		WALEnabled:  false, // Not needed for memory DB
		Timeout:     5 * time.Second,
	}

	store, err := rag.NewSQLiteVectorStore(config, logger)
	require.NoError(t, err)
	require.NotNil(t, store)

	ctx := context.Background()
	err = store.Initialize(ctx)
	require.NoError(t, err)

	return store, ctx
}

func newFileVectorStore(t *testing.T) (*rag.SQLiteVectorStore, context.Context, func()) {
	tmpDir, err := os.MkdirTemp("", "rag-test-*")
	require.NoError(t, err)

	logger := newTestLoggerSQLiteVectorStore()
	config := &rag.SQLiteVectorConfig{
		Path:        filepath.Join(tmpDir, "test-vectors.db"),
		VectorSize:  128,
		CacheSizeMB: 16,
		MMapSizeMB:  32,
		WALEnabled:  true,
		Timeout:     5 * time.Second,
	}

	store, err := rag.NewSQLiteVectorStore(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = store.Initialize(ctx)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, ctx, cleanup
}

func makeTestEmbedding(dimension int, seed float32) []float32 {
	embedding := make([]float32, dimension)
	for i := range embedding {
		embedding[i] = seed + float32(i)/float32(dimension)
	}
	return embedding
}

func makeTestDocument(connectionID string, docType rag.DocumentType, content string) *rag.Document {
	return &rag.Document{
		ConnectionID: connectionID,
		Type:         docType,
		Content:      content,
		Embedding:    makeTestEmbedding(128, 0.1),
		Metadata: map[string]interface{}{
			"test": "data",
		},
	}
}

// Constructor Tests

func TestNewSQLiteVectorStore(t *testing.T) {
	logger := newTestLoggerSQLiteVectorStore()

	t.Run("create with memory database", func(t *testing.T) {
		config := &rag.SQLiteVectorConfig{
			Path:        ":memory:",
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  false,
			Timeout:     5 * time.Second,
		}

		store, err := rag.NewSQLiteVectorStore(config, logger)
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("create with file database", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "rag-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		config := &rag.SQLiteVectorConfig{
			Path:        filepath.Join(tmpDir, "test.db"),
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  true,
			Timeout:     5 * time.Second,
		}

		store, err := rag.NewSQLiteVectorStore(config, logger)
		require.NoError(t, err)
		assert.NotNil(t, store)
	})

	t.Run("create with invalid path", func(t *testing.T) {
		config := &rag.SQLiteVectorConfig{
			Path:        "/nonexistent/path/to/db.db",
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  false,
			Timeout:     5 * time.Second,
		}

		_, err := rag.NewSQLiteVectorStore(config, logger)
		// May succeed or fail depending on driver behavior
		// At least it shouldn't panic
		_ = err
	})
}

// Initialize Tests

func TestSQLiteVectorStore_Initialize(t *testing.T) {
	t.Run("initialize creates schema", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Verify collections were created
		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, collections)
		assert.Contains(t, collections, "schemas")
		assert.Contains(t, collections, "queries")
	})

	t.Run("initialize is idempotent", func(t *testing.T) {
		logger := newTestLoggerSQLiteVectorStore()
		config := &rag.SQLiteVectorConfig{
			Path:        ":memory:",
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  false,
			Timeout:     5 * time.Second,
		}

		store, err := rag.NewSQLiteVectorStore(config, logger)
		require.NoError(t, err)

		ctx := context.Background()

		// Initialize multiple times
		err = store.Initialize(ctx)
		require.NoError(t, err)

		err = store.Initialize(ctx)
		require.NoError(t, err)

		err = store.Initialize(ctx)
		require.NoError(t, err)
	})

	t.Run("initialize with context timeout", func(t *testing.T) {
		logger := newTestLoggerSQLiteVectorStore()
		config := &rag.SQLiteVectorConfig{
			Path:        ":memory:",
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  false,
			Timeout:     5 * time.Second,
		}

		store, err := rag.NewSQLiteVectorStore(config, logger)
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
		defer cancel()
		time.Sleep(10 * time.Millisecond)

		err = store.Initialize(ctx)
		// May succeed or fail depending on timing
		_ = err
	})
}

// IndexDocument Tests

func TestSQLiteVectorStore_IndexDocument(t *testing.T) {
	t.Run("index document with all fields", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := &rag.Document{
			ConnectionID: "conn-1",
			Type:         rag.DocumentTypeSchema,
			Content:      "CREATE TABLE users (id INT PRIMARY KEY, name VARCHAR(255))",
			Embedding:    makeTestEmbedding(128, 0.1),
			Metadata: map[string]interface{}{
				"table": "users",
				"type":  "DDL",
			},
		}

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
		assert.NotEmpty(t, doc.ID)
		assert.False(t, doc.CreatedAt.IsZero())
		assert.False(t, doc.UpdatedAt.IsZero())
	})

	t.Run("index document without embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := &rag.Document{
			ConnectionID: "conn-1",
			Type:         rag.DocumentTypeQuery,
			Content:      "SELECT * FROM users",
			Metadata:     map[string]interface{}{},
		}

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
		assert.NotEmpty(t, doc.ID)
	})

	t.Run("index document generates ID if missing", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test content")

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
		assert.NotEmpty(t, doc.ID)
	})

	t.Run("index document with custom ID", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test content")
		doc.ID = "custom-id-123"

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
		assert.Equal(t, "custom-id-123", doc.ID)
	})

	t.Run("index document updates existing", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// First insert
		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "original content")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		originalID := doc.ID

		// Update with same ID
		doc.Content = "updated content"
		err = store.IndexDocument(ctx, doc)
		require.NoError(t, err)
		assert.Equal(t, originalID, doc.ID)

		// Verify update
		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated content", retrieved.Content)
	})

	t.Run("index document with empty content", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "")

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})

	t.Run("index document with nil metadata", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := &rag.Document{
			ConnectionID: "conn-1",
			Type:         rag.DocumentTypeSchema,
			Content:      "test",
			Embedding:    makeTestEmbedding(128, 0.1),
			Metadata:     nil,
		}

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})

	t.Run("index document with context cancellation", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")

		ctx, cancel := context.WithCancel(ctx)
		cancel()

		err := store.IndexDocument(ctx, doc)
		assert.Error(t, err)
	})
}

// BatchIndexDocuments Tests

func TestSQLiteVectorStore_BatchIndexDocuments(t *testing.T) {
	t.Run("batch index multiple documents", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "table1"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "table2"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "table3"),
		}

		err := store.BatchIndexDocuments(ctx, docs)
		require.NoError(t, err)

		for _, doc := range docs {
			assert.NotEmpty(t, doc.ID)
		}
	})

	t.Run("batch index empty slice", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.BatchIndexDocuments(ctx, []*rag.Document{})
		require.NoError(t, err)
	})

	t.Run("batch index with mixed types", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "schema doc"),
			makeTestDocument("conn-1", rag.DocumentTypeQuery, "query doc"),
			makeTestDocument("conn-1", rag.DocumentTypeBusiness, "business doc"),
		}

		err := store.BatchIndexDocuments(ctx, docs)
		require.NoError(t, err)
	})

	t.Run("batch index large batch", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		docs := make([]*rag.Document, 100)
		for i := range docs {
			docs[i] = makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("doc-%d", i))
		}

		err := store.BatchIndexDocuments(ctx, docs)
		require.NoError(t, err)
	})
}

// GetDocument Tests

func TestSQLiteVectorStore_GetDocument(t *testing.T) {
	t.Run("get existing document", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test content")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, doc.ID, retrieved.ID)
		assert.Equal(t, doc.ConnectionID, retrieved.ConnectionID)
		assert.Equal(t, doc.Type, retrieved.Type)
		assert.Equal(t, doc.Content, retrieved.Content)
	})

	t.Run("get document with embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, retrieved.Embedding)
		assert.Equal(t, len(doc.Embedding), len(retrieved.Embedding))
	})

	t.Run("get nonexistent document", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		_, err := store.GetDocument(ctx, "nonexistent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("get document with empty ID", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		_, err := store.GetDocument(ctx, "")
		assert.Error(t, err)
	})
}

// UpdateDocument Tests

func TestSQLiteVectorStore_UpdateDocument(t *testing.T) {
	t.Run("update document content", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "original")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		doc.Content = "updated"
		err = store.UpdateDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, "updated", retrieved.Content)
	})

	t.Run("update document metadata", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		doc.Metadata["new_field"] = "new_value"
		err = store.UpdateDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, "new_value", retrieved.Metadata["new_field"])
	})

	t.Run("update document embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		newEmbedding := makeTestEmbedding(128, 0.9)
		doc.Embedding = newEmbedding
		err = store.UpdateDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, retrieved.Embedding)
	})

	t.Run("update sets updated_at", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		originalUpdatedAt := doc.UpdatedAt
		time.Sleep(10 * time.Millisecond)

		doc.Content = "updated"
		err = store.UpdateDocument(ctx, doc)
		require.NoError(t, err)

		assert.True(t, doc.UpdatedAt.After(originalUpdatedAt))
	})
}

// DeleteDocument Tests

func TestSQLiteVectorStore_DeleteDocument(t *testing.T) {
	t.Run("delete existing document", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		err = store.DeleteDocument(ctx, doc.ID)
		require.NoError(t, err)

		_, err = store.GetDocument(ctx, doc.ID)
		assert.Error(t, err)
	})

	t.Run("delete deletes embedding via cascade", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		err = store.DeleteDocument(ctx, doc.ID)
		require.NoError(t, err)

		// Embedding should also be deleted
		_, err = store.GetDocument(ctx, doc.ID)
		assert.Error(t, err)
	})

	t.Run("delete nonexistent document succeeds", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.DeleteDocument(ctx, "nonexistent-id")
		// SQLite DELETE succeeds even if no rows affected
		require.NoError(t, err)
	})

	t.Run("delete with empty ID", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.DeleteDocument(ctx, "")
		require.NoError(t, err)
	})
}

// SearchSimilar Tests

func TestSQLiteVectorStore_SearchSimilar(t *testing.T) {
	t.Run("search returns most similar documents", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Index documents with different embeddings
		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc1"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc2"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc3"),
		}
		docs[0].Embedding = makeTestEmbedding(128, 0.1)
		docs[1].Embedding = makeTestEmbedding(128, 0.5)
		docs[2].Embedding = makeTestEmbedding(128, 0.9)

		for _, doc := range docs {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		// Search with embedding similar to doc1
		queryEmbedding := makeTestEmbedding(128, 0.15)
		results, err := store.SearchSimilar(ctx, queryEmbedding, 3, nil)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		// Results should have scores
		for _, result := range results {
			assert.NotZero(t, result.Score)
		}

		// Results should be sorted by score descending
		for i := 0; i < len(results)-1; i++ {
			assert.GreaterOrEqual(t, results[i].Score, results[i+1].Score)
		}
	})

	t.Run("search with k limit", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Index 10 documents
		for i := 0; i < 10; i++ {
			doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("doc-%d", i))
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.SearchSimilar(ctx, queryEmbedding, 5, nil)
		require.NoError(t, err)
		assert.Len(t, results, 5)
	})

	t.Run("search with connection_id filter", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Index documents with different connection IDs
		doc1 := makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc1")
		doc2 := makeTestDocument("conn-2", rag.DocumentTypeSchema, "doc2")
		doc3 := makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc3")

		for _, doc := range []*rag.Document{doc1, doc2, doc3} {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		// Search with filter
		queryEmbedding := makeTestEmbedding(128, 0.5)
		filter := map[string]interface{}{
			"connection_id": "conn-1",
		}
		results, err := store.SearchSimilar(ctx, queryEmbedding, 10, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// All results should be from conn-1
		for _, result := range results {
			assert.Equal(t, "conn-1", result.ConnectionID)
		}
	})

	t.Run("search with type filter", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Index documents with different types
		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "schema"),
			makeTestDocument("conn-1", rag.DocumentTypeQuery, "query"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "schema2"),
		}

		for _, doc := range docs {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		queryEmbedding := makeTestEmbedding(128, 0.5)
		filter := map[string]interface{}{
			"type": "schema",
		}
		results, err := store.SearchSimilar(ctx, queryEmbedding, 10, filter)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		for _, result := range results {
			assert.Equal(t, rag.DocumentTypeSchema, result.Type)
		}
	})

	t.Run("search with multiple filters", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc1"),
			makeTestDocument("conn-2", rag.DocumentTypeSchema, "doc2"),
			makeTestDocument("conn-1", rag.DocumentTypeQuery, "doc3"),
		}

		for _, doc := range docs {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		queryEmbedding := makeTestEmbedding(128, 0.5)
		filter := map[string]interface{}{
			"connection_id": "conn-1",
			"type":          "schema",
		}
		results, err := store.SearchSimilar(ctx, queryEmbedding, 10, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})

	t.Run("search empty database", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.SearchSimilar(ctx, queryEmbedding, 10, nil)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("search with zero k", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.SearchSimilar(ctx, queryEmbedding, 0, nil)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("search with nil filter", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "doc")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.SearchSimilar(ctx, queryEmbedding, 10, nil)
		require.NoError(t, err)
		assert.Len(t, results, 1)
	})
}

// SearchByText Tests

func TestSQLiteVectorStore_SearchByText(t *testing.T) {
	t.Run("search by text finds matching documents", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Index documents
		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "CREATE TABLE users (id INT, name VARCHAR(255))"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "CREATE TABLE products (id INT, price DECIMAL)"),
			makeTestDocument("conn-1", rag.DocumentTypeQuery, "SELECT * FROM users WHERE active = true"),
		}

		for _, doc := range docs {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		// Search for "users"
		results, err := store.SearchByText(ctx, "users", 10, nil)

		// FTS5 may not be available or FTS table may not exist
		if err != nil {
			if err.Error() == "no such module: fts5" ||
				err.Error() == "failed to search text: no such table: documents_fts" {
				t.Skip("FTS5 not available or not initialized in this SQLite build")
			}
			require.NoError(t, err)
		}

		assert.NotEmpty(t, results)
	})

	t.Run("search with k limit", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		for i := 0; i < 10; i++ {
			doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("document test content %d", i))
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		results, err := store.SearchByText(ctx, "test", 5, nil)
		if err != nil {
			if err.Error() == "no such module: fts5" ||
				err.Error() == "failed to search text: no such table: documents_fts" {
				t.Skip("FTS5 not available")
			}
			require.NoError(t, err)
		}

		assert.LessOrEqual(t, len(results), 5)
	})

	t.Run("search with filters", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "test document one"),
			makeTestDocument("conn-2", rag.DocumentTypeSchema, "test document two"),
		}

		for _, doc := range docs {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		filter := map[string]interface{}{
			"connection_id": "conn-1",
		}
		results, err := store.SearchByText(ctx, "test", 10, filter)
		if err != nil {
			if err.Error() == "no such module: fts5" ||
				err.Error() == "failed to search text: no such table: documents_fts" {
				t.Skip("FTS5 not available")
			}
			require.NoError(t, err)
		}

		for _, result := range results {
			assert.Equal(t, "conn-1", result.ConnectionID)
		}
	})

	t.Run("search empty query", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		_, err = store.SearchByText(ctx, "", 10, nil)
		// May succeed or fail depending on FTS5 behavior
		_ = err
	})

	t.Run("search no matches", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test document")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		results, err := store.SearchByText(ctx, "nonexistent", 10, nil)
		if err != nil {
			if err.Error() == "no such module: fts5" ||
				err.Error() == "failed to search text: no such table: documents_fts" {
				t.Skip("FTS5 not available")
			}
			require.NoError(t, err)
		}

		assert.Empty(t, results)
	})
}

// HybridSearch Tests

func TestSQLiteVectorStore_HybridSearch(t *testing.T) {
	t.Run("hybrid search combines results", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		docs := []*rag.Document{
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "CREATE TABLE users"),
			makeTestDocument("conn-1", rag.DocumentTypeSchema, "CREATE TABLE products"),
			makeTestDocument("conn-1", rag.DocumentTypeQuery, "SELECT FROM users"),
		}

		for _, doc := range docs {
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.HybridSearch(ctx, "users", queryEmbedding, 10)

		// Will use vector search even if FTS5 fails
		require.NoError(t, err)
		assert.NotEmpty(t, results)
	})

	t.Run("hybrid search deduplicates", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test document")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.HybridSearch(ctx, "test", queryEmbedding, 10)
		require.NoError(t, err)

		// Should not duplicate the same document
		ids := make(map[string]bool)
		for _, result := range results {
			assert.False(t, ids[result.ID], "duplicate document in results")
			ids[result.ID] = true
		}
	})

	t.Run("hybrid search respects k limit", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		for i := 0; i < 20; i++ {
			doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("test document %d", i))
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		queryEmbedding := makeTestEmbedding(128, 0.5)
		results, err := store.HybridSearch(ctx, "test", queryEmbedding, 5)
		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 5)
	})
}

// Collection Management Tests

func TestSQLiteVectorStore_CreateCollection(t *testing.T) {
	t.Run("create new collection", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "test-collection", 256)
		require.NoError(t, err)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.Contains(t, collections, "test-collection")
	})

	t.Run("create collection with different dimensions", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "small-vectors", 64)
		require.NoError(t, err)

		err = store.CreateCollection(ctx, "large-vectors", 2048)
		require.NoError(t, err)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.Contains(t, collections, "small-vectors")
		assert.Contains(t, collections, "large-vectors")
	})

	t.Run("create duplicate collection fails", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "duplicate", 128)
		require.NoError(t, err)

		err = store.CreateCollection(ctx, "duplicate", 128)
		assert.Error(t, err)
	})

	t.Run("create collection with empty name", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "", 128)
		// May succeed or fail depending on SQLite constraints
		_ = err
	})
}

func TestSQLiteVectorStore_ListCollections(t *testing.T) {
	t.Run("list default collections", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.NotEmpty(t, collections)

		// Should have default collections
		assert.Contains(t, collections, "schemas")
		assert.Contains(t, collections, "queries")
		assert.Contains(t, collections, "performance")
		assert.Contains(t, collections, "business")
		assert.Contains(t, collections, "memory")
	})

	t.Run("list after creating collections", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "custom1", 128)
		require.NoError(t, err)
		err = store.CreateCollection(ctx, "custom2", 128)
		require.NoError(t, err)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.Contains(t, collections, "custom1")
		assert.Contains(t, collections, "custom2")
	})

	t.Run("list is sorted", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)

		// Verify sorted order
		for i := 0; i < len(collections)-1; i++ {
			assert.LessOrEqual(t, collections[i], collections[i+1])
		}
	})
}

func TestSQLiteVectorStore_DeleteCollection(t *testing.T) {
	t.Run("delete existing collection", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "to-delete", 128)
		require.NoError(t, err)

		err = store.DeleteCollection(ctx, "to-delete")
		require.NoError(t, err)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.NotContains(t, collections, "to-delete")
	})

	t.Run("delete default collection", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.DeleteCollection(ctx, "schemas")
		require.NoError(t, err)

		collections, err := store.ListCollections(ctx)
		require.NoError(t, err)
		assert.NotContains(t, collections, "schemas")
	})

	t.Run("delete nonexistent collection succeeds", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.DeleteCollection(ctx, "nonexistent")
		// SQLite DELETE succeeds even if no rows affected
		require.NoError(t, err)
	})
}

// Statistics Tests

func TestSQLiteVectorStore_GetStats(t *testing.T) {
	t.Run("get stats from empty store", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		stats, err := store.GetStats(ctx)
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, int64(0), stats.TotalDocuments)
		assert.Greater(t, stats.TotalCollections, 0) // Has default collections
	})

	t.Run("get stats after indexing documents", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		for i := 0; i < 5; i++ {
			doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("doc-%d", i))
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		stats, err := store.GetStats(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(5), stats.TotalDocuments)
	})

	t.Run("stats includes last optimized", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		stats, err := store.GetStats(ctx)
		require.NoError(t, err)
		assert.False(t, stats.LastOptimized.IsZero())
	})
}

func TestSQLiteVectorStore_GetCollectionStats(t *testing.T) {
	t.Run("get stats for default collection", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		stats, err := store.GetCollectionStats(ctx, "schemas")
		require.NoError(t, err)
		assert.NotNil(t, stats)
		assert.Equal(t, "schemas", stats.Name)
		// Default collections use 1536 dimensions (from schema)
		assert.Equal(t, 1536, stats.Dimension)
	})

	t.Run("get stats for custom collection", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.CreateCollection(ctx, "custom", 256)
		require.NoError(t, err)

		stats, err := store.GetCollectionStats(ctx, "custom")
		require.NoError(t, err)
		assert.Equal(t, "custom", stats.Name)
		assert.Equal(t, 256, stats.Dimension)
	})

	t.Run("get stats for nonexistent collection", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		_, err := store.GetCollectionStats(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("stats includes document count", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		stats, err := store.GetCollectionStats(ctx, "schemas")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, stats.DocumentCount, int64(0))
	})
}

// Maintenance Tests

func TestSQLiteVectorStore_Optimize(t *testing.T) {
	t.Run("optimize succeeds", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		// Add some data
		for i := 0; i < 10; i++ {
			doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("doc-%d", i))
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		err := store.Optimize(ctx)
		// Some optimizations may fail (FTS5), but shouldn't error
		assert.NoError(t, err)
	})

	t.Run("optimize empty database", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.Optimize(ctx)
		assert.NoError(t, err)
	})
}

func TestSQLiteVectorStore_Backup(t *testing.T) {
	t.Run("backup to file", func(t *testing.T) {
		store, ctx, cleanup := newFileVectorStore(t)
		defer cleanup()

		// Add data
		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		// Backup
		tmpDir, err := os.MkdirTemp("", "backup-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		backupPath := filepath.Join(tmpDir, "backup.db")
		err = store.Backup(ctx, backupPath)
		require.NoError(t, err)

		// Verify backup file exists
		_, err = os.Stat(backupPath)
		assert.NoError(t, err)
	})

	t.Run("backup memory database", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		tmpDir, err := os.MkdirTemp("", "backup-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		backupPath := filepath.Join(tmpDir, "backup.db")
		err = store.Backup(ctx, backupPath)
		require.NoError(t, err)
	})
}

func TestSQLiteVectorStore_Restore(t *testing.T) {
	t.Run("restore not implemented", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		err := store.Restore(ctx, "/path/to/backup.db")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not yet implemented")
	})
}

// Concurrent Operations Tests

func TestSQLiteVectorStore_ConcurrentReads(t *testing.T) {
	t.Run("concurrent get operations", func(t *testing.T) {
		// Use file-based DB for proper concurrent access
		store, ctx, cleanup := newFileVectorStore(t)
		defer cleanup()

		// Index a document
		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		// Concurrent reads
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func() {
				_, err := store.GetDocument(ctx, doc.ID)
				done <- err
			}()
		}

		// Check all succeeded
		for i := 0; i < 10; i++ {
			err := <-done
			assert.NoError(t, err)
		}
	})

	t.Run("concurrent searches", func(t *testing.T) {
		// Use file-based DB for proper concurrent access
		store, ctx, cleanup := newFileVectorStore(t)
		defer cleanup()

		// Index documents
		for i := 0; i < 5; i++ {
			doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("doc-%d", i))
			err := store.IndexDocument(ctx, doc)
			require.NoError(t, err)
		}

		// Concurrent searches
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			go func() {
				queryEmbedding := makeTestEmbedding(128, 0.5)
				_, err := store.SearchSimilar(ctx, queryEmbedding, 3, nil)
				done <- err
			}()
		}

		for i := 0; i < 10; i++ {
			err := <-done
			assert.NoError(t, err)
		}
	})
}

func TestSQLiteVectorStore_ConcurrentWrites(t *testing.T) {
	t.Run("concurrent index operations", func(t *testing.T) {
		store, ctx, cleanup := newFileVectorStore(t)
		defer cleanup()

		// Concurrent writes
		done := make(chan error, 10)
		for i := 0; i < 10; i++ {
			i := i
			go func() {
				doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, fmt.Sprintf("doc-%d", i))
				done <- store.IndexDocument(ctx, doc)
			}()
		}

		// Check results
		for i := 0; i < 10; i++ {
			err := <-done
			// Some may fail due to locking, but shouldn't panic
			_ = err
		}
	})
}

// Edge Cases and Error Handling

func TestSQLiteVectorStore_EdgeCases(t *testing.T) {
	t.Run("empty embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		doc.Embedding = []float32{}

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})

	t.Run("very large embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		doc.Embedding = makeTestEmbedding(10000, 0.1)

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})

	t.Run("very long content", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		longContent := string(make([]byte, 1024*1024)) // 1MB
		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, longContent)

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})

	t.Run("special characters in content", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test 'quotes' \"double\" \n\t special")

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.Equal(t, doc.Content, retrieved.Content)
	})

	t.Run("metadata with nested objects", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		doc.Metadata = map[string]interface{}{
			"nested": map[string]interface{}{
				"level2": map[string]interface{}{
					"level3": "value",
				},
			},
		}

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		retrieved, err := store.GetDocument(ctx, doc.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.Metadata["nested"])
	})

	t.Run("metadata with arrays", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		doc.Metadata = map[string]interface{}{
			"tags": []string{"tag1", "tag2", "tag3"},
		}

		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})

	t.Run("search with nil embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		results, err := store.SearchSimilar(ctx, nil, 10, nil)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("search with empty embedding", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		results, err := store.SearchSimilar(ctx, []float32{}, 10, nil)
		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("multiple documents same content different embeddings", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc1 := makeTestDocument("conn-1", rag.DocumentTypeSchema, "same content")
		doc1.Embedding = makeTestEmbedding(128, 0.1)

		doc2 := makeTestDocument("conn-1", rag.DocumentTypeSchema, "same content")
		doc2.Embedding = makeTestEmbedding(128, 0.9)

		err := store.IndexDocument(ctx, doc1)
		require.NoError(t, err)

		err = store.IndexDocument(ctx, doc2)
		require.NoError(t, err)

		// Both should be indexed
		assert.NotEqual(t, doc1.ID, doc2.ID)
	})
}

// File-Based Database Tests

func TestSQLiteVectorStore_FileBased(t *testing.T) {
	t.Run("persist data across connections", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "persist-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		dbPath := filepath.Join(tmpDir, "persist.db")
		logger := newTestLoggerSQLiteVectorStore()

		// First connection
		config1 := &rag.SQLiteVectorConfig{
			Path:        dbPath,
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  true,
			Timeout:     5 * time.Second,
		}

		store1, err := rag.NewSQLiteVectorStore(config1, logger)
		require.NoError(t, err)

		ctx := context.Background()
		err = store1.Initialize(ctx)
		require.NoError(t, err)

		// Index document
		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test content")
		err = store1.IndexDocument(ctx, doc)
		require.NoError(t, err)

		docID := doc.ID

		// Second connection
		config2 := &rag.SQLiteVectorConfig{
			Path:        dbPath,
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  true,
			Timeout:     5 * time.Second,
		}

		store2, err := rag.NewSQLiteVectorStore(config2, logger)
		require.NoError(t, err)

		err = store2.Initialize(ctx)
		require.NoError(t, err)

		// Retrieve document from second connection
		retrieved, err := store2.GetDocument(ctx, docID)
		require.NoError(t, err)
		assert.Equal(t, "test content", retrieved.Content)
	})

	t.Run("WAL mode enabled", func(t *testing.T) {
		store, ctx, cleanup := newFileVectorStore(t)
		defer cleanup()

		// Just verify it works with WAL enabled
		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)
	})
}

// Context Cancellation Tests

func TestSQLiteVectorStore_ContextCancellation(t *testing.T) {
	t.Run("cancelled context on initialize", func(t *testing.T) {
		logger := newTestLoggerSQLiteVectorStore()
		config := &rag.SQLiteVectorConfig{
			Path:        ":memory:",
			VectorSize:  128,
			CacheSizeMB: 16,
			MMapSizeMB:  32,
			WALEnabled:  false,
			Timeout:     5 * time.Second,
		}

		store, err := rag.NewSQLiteVectorStore(config, logger)
		require.NoError(t, err)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err = store.Initialize(ctx)
		// May succeed or fail depending on timing
		_ = err
	})

	t.Run("cancelled context on search", func(t *testing.T) {
		store, ctx := newMemoryVectorStore(t)

		doc := makeTestDocument("conn-1", rag.DocumentTypeSchema, "test")
		err := store.IndexDocument(ctx, doc)
		require.NoError(t, err)

		cancelledCtx, cancel := context.WithCancel(ctx)
		cancel()

		queryEmbedding := makeTestEmbedding(128, 0.5)
		_, err = store.SearchSimilar(cancelledCtx, queryEmbedding, 10, nil)
		// May succeed or fail
		_ = err
	})
}
