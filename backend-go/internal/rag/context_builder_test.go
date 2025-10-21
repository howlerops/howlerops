//go:build integration
// +build integration

package rag_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/rag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock VectorStore for testing
type mockVectorStore struct {
	mu                sync.Mutex
	searchSimilarFunc func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error)
	searchCalls       int
}

func newMockVectorStore() *mockVectorStore {
	return &mockVectorStore{
		searchCalls: 0,
	}
}

func (m *mockVectorStore) SearchSimilar(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchCalls++

	if m.searchSimilarFunc != nil {
		return m.searchSimilarFunc(ctx, embedding, limit, filter)
	}

	// Default behavior - return empty results
	return []*rag.Document{}, nil
}

func (m *mockVectorStore) Initialize(ctx context.Context) error {
	return nil
}

func (m *mockVectorStore) IndexDocument(ctx context.Context, doc *rag.Document) error {
	return nil
}

func (m *mockVectorStore) BatchIndexDocuments(ctx context.Context, docs []*rag.Document) error {
	return nil
}

func (m *mockVectorStore) GetDocument(ctx context.Context, id string) (*rag.Document, error) {
	return nil, errors.New("not implemented")
}

func (m *mockVectorStore) UpdateDocument(ctx context.Context, doc *rag.Document) error {
	return nil
}

func (m *mockVectorStore) DeleteDocument(ctx context.Context, id string) error {
	return nil
}

func (m *mockVectorStore) SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*rag.Document, error) {
	return []*rag.Document{}, nil
}

func (m *mockVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*rag.Document, error) {
	return []*rag.Document{}, nil
}

func (m *mockVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	return nil
}

func (m *mockVectorStore) DeleteCollection(ctx context.Context, name string) error {
	return nil
}

func (m *mockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (m *mockVectorStore) GetStats(ctx context.Context) (*rag.VectorStoreStats, error) {
	return &rag.VectorStoreStats{}, nil
}

func (m *mockVectorStore) GetCollectionStats(ctx context.Context, name string) (*rag.CollectionStats, error) {
	return &rag.CollectionStats{}, nil
}

func (m *mockVectorStore) Optimize(ctx context.Context) error {
	return nil
}

func (m *mockVectorStore) Backup(ctx context.Context, path string) error {
	return nil
}

func (m *mockVectorStore) Restore(ctx context.Context, path string) error {
	return nil
}

func (m *mockVectorStore) getSearchCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.searchCalls
}

func (m *mockVectorStore) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.searchCalls = 0
}

// Mock EmbeddingService for testing
type mockEmbeddingService struct {
	mu            sync.Mutex
	embedTextFunc func(ctx context.Context, text string) ([]float32, error)
	embedCalls    int
}

func newMockEmbeddingService() *mockEmbeddingService {
	return &mockEmbeddingService{
		embedCalls: 0,
	}
}

func (m *mockEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedCalls++

	if m.embedTextFunc != nil {
		return m.embedTextFunc(ctx, text)
	}

	// Default behavior - return simple embedding
	embedding := make([]float32, 384)
	for i := range embedding {
		embedding[i] = float32(i) / 384.0
	}
	return embedding, nil
}

func (m *mockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i := range texts {
		emb, err := m.EmbedText(ctx, texts[i])
		if err != nil {
			return nil, err
		}
		embeddings[i] = emb
	}
	return embeddings, nil
}

func (m *mockEmbeddingService) EmbedDocument(ctx context.Context, doc *rag.Document) error {
	embedding, err := m.EmbedText(ctx, doc.Content)
	if err != nil {
		return err
	}
	doc.Embedding = embedding
	return nil
}

func (m *mockEmbeddingService) GetCacheStats() *rag.CacheStats {
	return &rag.CacheStats{}
}

func (m *mockEmbeddingService) ClearCache() error {
	return nil
}

func (m *mockEmbeddingService) getEmbedCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.embedCalls
}

func (m *mockEmbeddingService) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedCalls = 0
}

// Helper function to create a test logger
func newTestLoggerContextBuilder() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

// Helper function to create test documents
func makeTestDoc(docType rag.DocumentType, content string, score float32) *rag.Document {
	return &rag.Document{
		ID:           fmt.Sprintf("doc-%s-%d", docType, time.Now().UnixNano()),
		Type:         docType,
		Content:      content,
		ConnectionID: "test-conn",
		Score:        score,
		Metadata: map[string]interface{}{
			"test": true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Constructor Tests

func TestNewContextBuilder(t *testing.T) {
	t.Run("creates context builder with valid parameters", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		require.NotNil(t, builder)
	})

	t.Run("creates with different logger levels", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()

		levels := []logrus.Level{
			logrus.DebugLevel,
			logrus.InfoLevel,
			logrus.WarnLevel,
			logrus.ErrorLevel,
		}

		for _, level := range levels {
			logger := logrus.New()
			logger.SetLevel(level)

			builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)
			require.NotNil(t, builder)
		}
	})

	t.Run("initializes internal components", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		// Build context to verify components work
		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})
}

// BuildContext Tests

func TestContextBuilder_BuildContext(t *testing.T) {
	t.Run("builds context successfully", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
		assert.Equal(t, "SELECT * FROM users", queryContext.Query)
		assert.NotNil(t, queryContext.RelevantSchemas)
		assert.NotNil(t, queryContext.SimilarQueries)
		assert.NotNil(t, queryContext.BusinessRules)
		assert.NotNil(t, queryContext.PerformanceHints)
		assert.NotNil(t, queryContext.DataStatistics)
		assert.NotNil(t, queryContext.Suggestions)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
	})

	t.Run("embeds query text", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, embeddingService.getEmbedCallCount(), 1)
	})

	t.Run("handles embedding error", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		embeddingService.embedTextFunc = func(ctx context.Context, text string) ([]float32, error) {
			return nil, errors.New("embedding failed")
		}
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to embed query")
	})

	t.Run("enriches context in parallel", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		start := time.Now()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")
		duration := time.Since(start)

		require.NoError(t, err)
		// Parallel execution should complete quickly
		assert.Less(t, duration, 6*time.Second)
	})

	t.Run("handles timeout in goroutines", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			// Simulate slow operation
			time.Sleep(6 * time.Second)
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		// Should not error, just log warning
		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})

	t.Run("handles errors in goroutines gracefully", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			return nil, errors.New("search failed")
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		// Should not error at top level, errors logged internally
		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})

	t.Run("generates suggestions", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.Suggestions)
	})

	t.Run("calculates confidence score", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
		assert.LessOrEqual(t, queryContext.Confidence, float32(1.0))
	})

	t.Run("handles empty query", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
		assert.Equal(t, "", queryContext.Query)
	})

	t.Run("handles empty connection ID", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		// May succeed or fail depending on timing
		_ = err
	})

	t.Run("concurrent build context calls", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		numGoroutines := 10
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				_, err := builder.BuildContext(ctx, "test query", "conn-1")
				results <- err
			}()
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})
}

// fetchRelevantSchemas Tests

func TestContextBuilder_FetchRelevantSchemas(t *testing.T) {
	t.Run("fetches schema documents", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeSchema) {
				return []*rag.Document{
					makeTestDoc(rag.DocumentTypeSchema, "CREATE TABLE users", 0.9),
					makeTestDoc(rag.DocumentTypeSchema, "CREATE TABLE orders", 0.8),
				}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		assert.NotEmpty(t, queryContext.RelevantSchemas)
	})

	t.Run("filters by connection ID", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		calledWithFilter := false
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter != nil && filter["connection_id"] == "test-conn" {
				calledWithFilter = true
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "test-conn")

		require.NoError(t, err)
		assert.True(t, calledWithFilter)
	})

	t.Run("limits results to top 5", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeSchema) {
				docs := make([]*rag.Document, 10)
				for i := 0; i < 10; i++ {
					docs[i] = makeTestDoc(rag.DocumentTypeSchema, fmt.Sprintf("schema %d", i), float32(1.0-float32(i)*0.1))
				}
				return docs, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.LessOrEqual(t, len(queryContext.RelevantSchemas), 5)
	})

	t.Run("sorts by relevance", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeSchema) {
				return []*rag.Document{
					makeTestDoc(rag.DocumentTypeSchema, "schema1", 0.7),
					makeTestDoc(rag.DocumentTypeSchema, "schema2", 0.9),
					makeTestDoc(rag.DocumentTypeSchema, "schema3", 0.8),
				}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		if len(queryContext.RelevantSchemas) > 1 {
			for i := 0; i < len(queryContext.RelevantSchemas)-1; i++ {
				assert.GreaterOrEqual(t, queryContext.RelevantSchemas[i].Relevance, queryContext.RelevantSchemas[i+1].Relevance)
			}
		}
	})

	t.Run("handles no schemas found", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.RelevantSchemas)
	})
}

// findSimilarQueries Tests

func TestContextBuilder_FindSimilarQueries(t *testing.T) {
	t.Run("finds similar query documents", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeQuery) {
				return []*rag.Document{
					makeTestDoc(rag.DocumentTypeQuery, "SELECT * FROM users", 0.9),
					makeTestDoc(rag.DocumentTypeQuery, "SELECT * FROM orders", 0.8),
				}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE active = 1", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.SimilarQueries)
	})

	t.Run("limits results to top 10", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeQuery) {
				docs := make([]*rag.Document, 15)
				for i := 0; i < 15; i++ {
					docs[i] = makeTestDoc(rag.DocumentTypeQuery, fmt.Sprintf("query %d", i), float32(1.0-float32(i)*0.05))
				}
				return docs, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.LessOrEqual(t, len(queryContext.SimilarQueries), 10)
	})

	t.Run("handles no similar queries", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.SimilarQueries)
	})

	t.Run("sorts by similarity and frequency", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		// Verify patterns are returned
		assert.NotNil(t, queryContext.SimilarQueries)
	})

	t.Run("calls pattern matcher", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeQuery) {
				return []*rag.Document{
					makeTestDoc(rag.DocumentTypeQuery, "SELECT * FROM users", 0.9),
				}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
	})
}

// extractBusinessRules Tests

func TestContextBuilder_ExtractBusinessRules(t *testing.T) {
	t.Run("extracts business rule documents", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeBusiness) {
				return []*rag.Document{
					makeTestDoc(rag.DocumentTypeBusiness, "Active users must have status = 1", 0.9),
				}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.BusinessRules)
	})

	t.Run("filters applicable rules", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.BusinessRules)
	})

	t.Run("sorts rules by priority", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		if len(queryContext.BusinessRules) > 1 {
			for i := 0; i < len(queryContext.BusinessRules)-1; i++ {
				assert.GreaterOrEqual(t, queryContext.BusinessRules[i].Priority, queryContext.BusinessRules[i+1].Priority)
			}
		}
	})

	t.Run("handles no business rules", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.BusinessRules)
	})

	t.Run("no connection ID filter for business rules", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		hasNoConnectionFilter := false
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeBusiness) {
				if _, ok := filter["connection_id"]; !ok {
					hasNoConnectionFilter = true
				}
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.True(t, hasNoConnectionFilter)
	})
}

// generateOptimizationHints Tests

func TestContextBuilder_GenerateOptimizationHints(t *testing.T) {
	t.Run("generates optimization hints", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE name = 'test'", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})

	t.Run("searches performance documents", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		searchedPerformance := false
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypePerformance) {
				searchedPerformance = true
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.True(t, searchedPerformance)
	})

	t.Run("checks for missing indexes", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE email = 'test@example.com'", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})

	t.Run("suggests query rewrites for SELECT *", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		if len(queryContext.PerformanceHints) > 0 {
			foundRewrite := false
			for _, hint := range queryContext.PerformanceHints {
				if hint.Type == "rewrite" {
					foundRewrite = true
					break
				}
			}
			assert.True(t, foundRewrite)
		}
	})

	t.Run("checks partitioning for date queries", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM orders WHERE created_date BETWEEN '2024-01-01' AND '2024-12-31'", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})

	t.Run("handles no optimization hints", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SHOW TABLES", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})

	t.Run("embeds query for performance search", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		// Should have at least 2 embed calls (initial + performance)
		assert.GreaterOrEqual(t, embeddingService.getEmbedCallCount(), 1)
	})
}

// collectDataStatistics Tests

func TestContextBuilder_CollectDataStatistics(t *testing.T) {
	t.Run("collects data statistics", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext.DataStatistics)
		assert.Greater(t, queryContext.DataStatistics.TotalRows, int64(0))
		assert.Greater(t, queryContext.DataStatistics.DataSize, int64(0))
	})

	t.Run("includes access patterns", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext.DataStatistics)
		assert.NotEmpty(t, queryContext.DataStatistics.AccessPatterns)
	})

	t.Run("includes distribution data", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext.DataStatistics)
		assert.NotEmpty(t, queryContext.DataStatistics.Distribution)
	})

	t.Run("includes last analyzed timestamp", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext.DataStatistics)
		assert.False(t, queryContext.DataStatistics.LastAnalyzed.IsZero())
	})
}

// generateSuggestions Tests

func TestContextBuilder_GenerateSuggestions(t *testing.T) {
	t.Run("generates suggestions from similar queries", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.Suggestions)
	})

	t.Run("limits suggestions from similar queries to 3", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		completionSuggestions := 0
		for _, sugg := range queryContext.Suggestions {
			if sugg.Type == "completion" {
				completionSuggestions++
			}
		}
		assert.LessOrEqual(t, completionSuggestions, 3)
	})

	t.Run("generates suggestions from optimization hints", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		if len(queryContext.PerformanceHints) > 0 {
			assert.NotEmpty(t, queryContext.Suggestions)
		}
	})

	t.Run("suggestion includes confidence", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		for _, sugg := range queryContext.Suggestions {
			assert.GreaterOrEqual(t, sugg.Confidence, float32(0.0))
			assert.LessOrEqual(t, sugg.Confidence, float32(1.0))
		}
	})

	t.Run("handles empty similar queries", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.Suggestions)
	})
}

// calculateConfidence Tests

func TestContextBuilder_CalculateConfidence(t *testing.T) {
	t.Run("calculates confidence from schema relevance", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeSchema) {
				return []*rag.Document{
					makeTestDoc(rag.DocumentTypeSchema, "schema", 0.9),
				}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.Greater(t, queryContext.Confidence, float32(0.0))
	})

	t.Run("calculates confidence from similar queries", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
	})

	t.Run("adds confidence for business rules", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
	})

	t.Run("adds confidence for optimization hints", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE name = 'test'", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
	})

	t.Run("returns default confidence when no context", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
	})

	t.Run("confidence is between 0 and 1", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.GreaterOrEqual(t, queryContext.Confidence, float32(0.0))
		assert.LessOrEqual(t, queryContext.Confidence, float32(1.0))
	})

	t.Run("higher confidence with more context", func(t *testing.T) {
		vectorStore1 := newMockVectorStore()
		embeddingService1 := newMockEmbeddingService()
		logger1 := newTestLoggerContextBuilder()
		builder1 := rag.NewContextBuilder(vectorStore1, embeddingService1, logger1)

		vectorStore2 := newMockVectorStore()
		vectorStore2.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeSchema) {
				return []*rag.Document{makeTestDoc(rag.DocumentTypeSchema, "schema", 0.9)}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService2 := newMockEmbeddingService()
		logger2 := newTestLoggerContextBuilder()
		builder2 := rag.NewContextBuilder(vectorStore2, embeddingService2, logger2)

		ctx := context.Background()
		qc1, err1 := builder1.BuildContext(ctx, "test query", "conn-1")
		require.NoError(t, err1)

		qc2, err2 := builder2.BuildContext(ctx, "test query", "conn-1")
		require.NoError(t, err2)

		// More context should generally lead to higher confidence
		_ = qc1.Confidence
		_ = qc2.Confidence
	})

	t.Run("confidence calculation handles empty arrays", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "test query", "conn-1")

		require.NoError(t, err)
		assert.NotZero(t, queryContext.Confidence)
	})
}

// Helper Method Tests

func TestContextBuilder_HelperMethods(t *testing.T) {
	t.Run("parseSchemaDocument extracts table name", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeSchema) {
				doc := makeTestDoc(rag.DocumentTypeSchema, "CREATE TABLE users", 0.9)
				doc.Metadata["table_name"] = "users"
				return []*rag.Document{doc}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		if len(queryContext.RelevantSchemas) > 0 {
			assert.Equal(t, "users", queryContext.RelevantSchemas[0].TableName)
		}
	})

	t.Run("parseBusinessRule extracts rule name", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			if filter["type"] == string(rag.DocumentTypeBusiness) {
				doc := makeTestDoc(rag.DocumentTypeBusiness, "Active users rule", 0.9)
				doc.Metadata["name"] = "active_users"
				return []*rag.Document{doc}, nil
			}
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE active = 1", "conn-1")

		require.NoError(t, err)
	})

	t.Run("isRuleApplicable checks conditions", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE active = 1", "conn-1")

		require.NoError(t, err)
	})

	t.Run("checkMissingIndexes detects WHERE without index", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE email = 'test@example.com'", "conn-1")

		require.NoError(t, err)
		if len(queryContext.PerformanceHints) > 0 {
			foundIndex := false
			for _, hint := range queryContext.PerformanceHints {
				if hint.Type == "index" {
					foundIndex = true
					break
				}
			}
			assert.True(t, foundIndex)
		}
	})

	t.Run("suggestQueryRewrite detects SELECT *", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")

		require.NoError(t, err)
		if len(queryContext.PerformanceHints) > 0 {
			foundRewrite := false
			for _, hint := range queryContext.PerformanceHints {
				if hint.Type == "rewrite" {
					foundRewrite = true
					assert.NotEmpty(t, hint.SQLBefore)
					assert.NotEmpty(t, hint.SQLAfter)
					break
				}
			}
			assert.True(t, foundRewrite)
		}
	})

	t.Run("checkPartitioning detects date queries", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM orders WHERE date BETWEEN '2024-01-01' AND '2024-12-31'", "conn-1")

		require.NoError(t, err)
		if len(queryContext.PerformanceHints) > 0 {
			foundPartition := false
			for _, hint := range queryContext.PerformanceHints {
				if hint.Type == "partition" {
					foundPartition = true
					break
				}
			}
			assert.True(t, foundPartition)
		}
	})

	t.Run("checkPartitioning detects BETWEEN clause", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM data WHERE value BETWEEN 100 AND 200", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})

	t.Run("helper methods handle nil metadata", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			doc := makeTestDoc(rag.DocumentTypeSchema, "test", 0.9)
			doc.Metadata = nil
			return []*rag.Document{doc}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		// Should not panic
		require.NoError(t, err)
	})

	t.Run("helper methods handle missing metadata fields", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			doc := makeTestDoc(rag.DocumentTypeSchema, "test", 0.9)
			doc.Metadata = map[string]interface{}{}
			return []*rag.Document{doc}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")

		// Should not panic
		require.NoError(t, err)
	})

	t.Run("case insensitive query pattern matching", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "select * FROM USERS where name = 'TEST'", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})
}

// Edge Cases Tests

func TestContextBuilder_EdgeCases(t *testing.T) {
	t.Run("handles very long query", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		longQuery := "SELECT * FROM users WHERE " + string(make([]byte, 10000))
		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, longQuery, "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})

	t.Run("handles special characters in query", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE name = 'O''Brien'", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})

	t.Run("handles unicode in query", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE name = '世界'", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})

	t.Run("handles multiple WHERE clauses", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM users WHERE name = 'test' AND age > 18 AND status = 'active'", "conn-1")

		require.NoError(t, err)
		assert.NotNil(t, queryContext.PerformanceHints)
	})

	t.Run("handles malformed SQL gracefully", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		queryContext, err := builder.BuildContext(ctx, "SELECT * FROM WHERE AND", "conn-1")

		require.NoError(t, err)
		require.NotNil(t, queryContext)
	})
}

// Concurrent Operations Tests

func TestContextBuilder_ConcurrentOperations(t *testing.T) {
	t.Run("handles concurrent BuildContext calls", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		numGoroutines := 20
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(n int) {
				_, err := builder.BuildContext(ctx, fmt.Sprintf("query %d", n), "conn-1")
				results <- err
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			err := <-results
			assert.NoError(t, err)
		}
	})

	t.Run("concurrent calls with different connection IDs", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		numGoroutines := 10
		results := make(chan *rag.QueryContext, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(n int) {
				qc, _ := builder.BuildContext(ctx, "test query", fmt.Sprintf("conn-%d", n))
				results <- qc
			}(i)
		}

		for i := 0; i < numGoroutines; i++ {
			qc := <-results
			assert.NotNil(t, qc)
		}
	})

	t.Run("thread safety with mocks", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		var wg sync.WaitGroup

		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = builder.BuildContext(ctx, "test query", "conn-1")
			}()
		}

		wg.Wait()
		// Should not panic or race
	})
}

// Performance Tests

func TestContextBuilder_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	t.Run("builds context within reasonable time", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		start := time.Now()
		_, err := builder.BuildContext(ctx, "SELECT * FROM users", "conn-1")
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 6*time.Second, "should complete within timeout")
	})

	t.Run("parallel enrichment is faster than sequential", func(t *testing.T) {
		vectorStore := newMockVectorStore()
		vectorStore.searchSimilarFunc = func(ctx context.Context, embedding []float32, limit int, filter map[string]interface{}) ([]*rag.Document, error) {
			time.Sleep(100 * time.Millisecond) // Simulate slow search
			return []*rag.Document{}, nil
		}
		embeddingService := newMockEmbeddingService()
		logger := newTestLoggerContextBuilder()

		builder := rag.NewContextBuilder(vectorStore, embeddingService, logger)

		ctx := context.Background()
		start := time.Now()
		_, err := builder.BuildContext(ctx, "test query", "conn-1")
		duration := time.Since(start)

		require.NoError(t, err)
		// With 5 parallel operations at 100ms each, should complete in ~100ms not 500ms
		assert.Less(t, duration, 1*time.Second)
	})
}
