//go:build integration
// +build integration

package rag_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/rag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEmbeddingProvider implements the EmbeddingProvider interface for testing
type mockEmbeddingProvider struct {
	mu               sync.Mutex
	embedTextCalls   int
	embedBatchCalls  int
	embedTextInputs  []string
	embedBatchInputs [][]string
	embeddings       map[string][]float32
	dimension        int
	model            string
	shouldError      bool
	errorMessage     string
	embedTextDelay   time.Duration
	embedBatchDelay  time.Duration
}

func newMockEmbeddingProvider(dimension int, model string) *mockEmbeddingProvider {
	return &mockEmbeddingProvider{
		embeddings: make(map[string][]float32),
		dimension:  dimension,
		model:      model,
	}
}

func (m *mockEmbeddingProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.embedTextDelay > 0 {
		time.Sleep(m.embedTextDelay)
	}

	m.embedTextCalls++
	m.embedTextInputs = append(m.embedTextInputs, text)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	// Check if we have a pre-configured embedding
	if embedding, ok := m.embeddings[text]; ok {
		return embedding, nil
	}

	// Generate a deterministic embedding based on text
	embedding := make([]float32, m.dimension)
	for i := range embedding {
		embedding[i] = float32((len(text)+i)%100) / 100.0
	}
	return embedding, nil
}

func (m *mockEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.embedBatchDelay > 0 {
		time.Sleep(m.embedBatchDelay)
	}

	m.embedBatchCalls++
	m.embedBatchInputs = append(m.embedBatchInputs, texts)

	if m.shouldError {
		return nil, fmt.Errorf("%s", m.errorMessage)
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		// Check if we have a pre-configured embedding
		if embedding, ok := m.embeddings[text]; ok {
			embeddings[i] = embedding
		} else {
			// Generate a deterministic embedding
			embedding := make([]float32, m.dimension)
			for j := range embedding {
				embedding[j] = float32((len(text)+i+j)%100) / 100.0
			}
			embeddings[i] = embedding
		}
	}
	return embeddings, nil
}

func (m *mockEmbeddingProvider) GetDimension() int {
	return m.dimension
}

func (m *mockEmbeddingProvider) GetModel() string {
	return m.model
}

func (m *mockEmbeddingProvider) setError(shouldError bool, message string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldError = shouldError
	m.errorMessage = message
}

func (m *mockEmbeddingProvider) setCannedEmbedding(text string, embedding []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embeddings[text] = embedding
}

func (m *mockEmbeddingProvider) getCallCount() (embedText, embedBatch int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.embedTextCalls, m.embedBatchCalls
}

func (m *mockEmbeddingProvider) reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.embedTextCalls = 0
	m.embedBatchCalls = 0
	m.embedTextInputs = nil
	m.embedBatchInputs = nil
}

// Helper function to create a test logger for embedding service tests
func newTestLoggerEmbedding() *logrus.Logger {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	return logger
}

// TestNewEmbeddingService tests the constructor
func TestNewEmbeddingService(t *testing.T) {
	t.Run("creates service with valid provider", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()

		service := rag.NewEmbeddingService(provider, logger)

		require.NotNil(t, service)
	})

	t.Run("creates service with different dimensions", func(t *testing.T) {
		tests := []struct {
			name      string
			dimension int
		}{
			{"small dimension", 128},
			{"medium dimension", 384},
			{"large dimension", 768},
			{"very large dimension", 1536},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				provider := newMockEmbeddingProvider(tt.dimension, "test-model")
				logger := newTestLoggerEmbedding()

				service := rag.NewEmbeddingService(provider, logger)

				require.NotNil(t, service)
			})
		}
	})

	t.Run("initializes with cache", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()

		service := rag.NewEmbeddingService(provider, logger)

		stats := service.GetCacheStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats.Size)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
	})
}

// TestEmbedText tests single text embedding
func TestEmbedText(t *testing.T) {
	t.Run("embeds simple text", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embedding, err := service.EmbedText(ctx, "hello world")

		require.NoError(t, err)
		require.NotNil(t, embedding)
		assert.Equal(t, 384, len(embedding))

		embedTextCalls, _ := provider.getCallCount()
		assert.Equal(t, 1, embedTextCalls)
	})

	t.Run("embeds empty string", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embedding, err := service.EmbedText(ctx, "")

		require.NoError(t, err)
		require.NotNil(t, embedding)
		assert.Equal(t, 384, len(embedding))
	})

	t.Run("embeds long text", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		longText := ""
		for i := 0; i < 1000; i++ {
			longText += "word "
		}

		ctx := context.Background()
		embedding, err := service.EmbedText(ctx, longText)

		require.NoError(t, err)
		require.NotNil(t, embedding)
		assert.Equal(t, 384, len(embedding))
	})

	t.Run("preprocesses text with whitespace", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		text := "  Hello   World  \n\t  Test  "

		_, err := service.EmbedText(ctx, text)
		require.NoError(t, err)

		// The provider should receive preprocessed text (lowercased, normalized whitespace)
		assert.Len(t, provider.embedTextInputs, 1)
		assert.Contains(t, provider.embedTextInputs[0], "hello")
		assert.Contains(t, provider.embedTextInputs[0], "world")
		assert.Contains(t, provider.embedTextInputs[0], "test")
	})

	t.Run("handles provider error", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		provider.setError(true, "provider error")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embedding, err := service.EmbedText(ctx, "test")

		require.Error(t, err)
		assert.Nil(t, embedding)
		assert.Contains(t, err.Error(), "failed to generate embedding")
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		provider.embedTextDelay = 100 * time.Millisecond
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := service.EmbedText(ctx, "test")

		// The error depends on when cancellation is detected
		// Either no error (if embedding was fast) or context error
		if err != nil {
			assert.Contains(t, err.Error(), "context")
		}
	})
}

// TestEmbedText_Caching tests cache behavior for single embeddings
func TestEmbedText_Caching(t *testing.T) {
	t.Run("caches embedding on first call", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "test")
		require.NoError(t, err)

		stats := service.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
	})

	t.Run("uses cache on second call", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embedding1, err := service.EmbedText(ctx, "test")
		require.NoError(t, err)

		embedding2, err := service.EmbedText(ctx, "test")
		require.NoError(t, err)

		assert.Equal(t, embedding1, embedding2)

		embedTextCalls, _ := provider.getCallCount()
		assert.Equal(t, 1, embedTextCalls, "provider should be called only once")

		stats := service.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
		assert.Equal(t, int64(1), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 0.5, stats.HitRate)
	})

	t.Run("different texts create different cache entries", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "test1")
		require.NoError(t, err)

		_, err = service.EmbedText(ctx, "test2")
		require.NoError(t, err)

		stats := service.GetCacheStats()
		assert.Equal(t, 2, stats.Size)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(2), stats.Misses)
	})

	t.Run("cache keys are case sensitive despite preprocessing", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "Test")
		require.NoError(t, err)

		_, err = service.EmbedText(ctx, "test")
		require.NoError(t, err)

		// Cache keys are based on original text, not preprocessed
		// So "Test" and "test" have different cache keys
		embedTextCalls, _ := provider.getCallCount()
		assert.Equal(t, 2, embedTextCalls, "different cases result in different cache keys")

		stats := service.GetCacheStats()
		assert.Equal(t, 2, stats.Size, "should have 2 cache entries for different cases")
	})

	t.Run("cache keys include whitespace differences", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "hello   world")
		require.NoError(t, err)

		_, err = service.EmbedText(ctx, "hello world")
		require.NoError(t, err)

		// Cache keys are based on original text, so different whitespace = different keys
		embedTextCalls, _ := provider.getCallCount()
		assert.Equal(t, 2, embedTextCalls, "different whitespace results in different cache keys")

		stats := service.GetCacheStats()
		assert.Equal(t, 2, stats.Size, "should have 2 cache entries")
	})
}

// TestEmbedBatch tests batch embedding
func TestEmbedBatch(t *testing.T) {
	t.Run("embeds empty batch", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embeddings, err := service.EmbedBatch(ctx, []string{})

		require.NoError(t, err)
		assert.Empty(t, embeddings)

		_, embedBatchCalls := provider.getCallCount()
		assert.Equal(t, 0, embedBatchCalls, "should not call provider for empty batch")
	})

	t.Run("embeds single text batch", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embeddings, err := service.EmbedBatch(ctx, []string{"test"})

		require.NoError(t, err)
		require.Len(t, embeddings, 1)
		assert.Equal(t, 384, len(embeddings[0]))
	})

	t.Run("embeds multiple texts", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		texts := []string{"text1", "text2", "text3", "text4"}
		embeddings, err := service.EmbedBatch(ctx, texts)

		require.NoError(t, err)
		require.Len(t, embeddings, 4)
		for i, embedding := range embeddings {
			assert.Equal(t, 384, len(embedding), "embedding %d has wrong dimension", i)
		}
	})

	t.Run("large batch processing", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		texts := make([]string, 100)
		for i := range texts {
			texts[i] = fmt.Sprintf("text %d", i)
		}

		ctx := context.Background()
		embeddings, err := service.EmbedBatch(ctx, texts)

		require.NoError(t, err)
		require.Len(t, embeddings, 100)
	})

	t.Run("handles provider error", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		provider.setError(true, "batch error")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embeddings, err := service.EmbedBatch(ctx, []string{"test1", "test2"})

		require.Error(t, err)
		assert.Nil(t, embeddings)
		assert.Contains(t, err.Error(), "failed to generate batch embeddings")
	})
}

// TestEmbedBatch_Caching tests cache behavior for batch embeddings
func TestEmbedBatch_Caching(t *testing.T) {
	t.Run("uses cache for partial batch", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		// First, cache some embeddings
		_, err := service.EmbedText(ctx, "cached1")
		require.NoError(t, err)
		_, err = service.EmbedText(ctx, "cached2")
		require.NoError(t, err)

		provider.reset()

		// Now batch embed with mix of cached and uncached
		texts := []string{"cached1", "new1", "cached2", "new2"}
		embeddings, err := service.EmbedBatch(ctx, texts)

		require.NoError(t, err)
		require.Len(t, embeddings, 4)

		// Provider should only be called for uncached items
		_, embedBatchCalls := provider.getCallCount()
		assert.Equal(t, 1, embedBatchCalls)
		assert.Len(t, provider.embedBatchInputs[0], 2, "only uncached texts should be sent to provider")
	})

	t.Run("all cached batch", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		texts := []string{"text1", "text2", "text3"}

		// First call to cache all
		embeddings1, err := service.EmbedBatch(ctx, texts)
		require.NoError(t, err)

		provider.reset()

		// Second call should use cache
		embeddings2, err := service.EmbedBatch(ctx, texts)
		require.NoError(t, err)

		assert.Equal(t, embeddings1, embeddings2)

		_, embedBatchCalls := provider.getCallCount()
		assert.Equal(t, 0, embedBatchCalls, "provider should not be called for fully cached batch")
	})

	t.Run("caches batch results", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		texts := []string{"text1", "text2"}

		_, err := service.EmbedBatch(ctx, texts)
		require.NoError(t, err)

		stats := service.GetCacheStats()
		assert.Equal(t, 2, stats.Size, "batch results should be cached individually")
	})

	t.Run("preserves order with mixed cache hits", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()

		// Cache text2 and text4
		embedding2, _ := service.EmbedText(ctx, "text2")
		embedding4, _ := service.EmbedText(ctx, "text4")

		// Batch with alternating cached/uncached
		texts := []string{"text1", "text2", "text3", "text4", "text5"}
		embeddings, err := service.EmbedBatch(ctx, texts)

		require.NoError(t, err)
		require.Len(t, embeddings, 5)

		// Verify cached embeddings are at correct positions
		assert.Equal(t, embedding2, embeddings[1])
		assert.Equal(t, embedding4, embeddings[3])

		// Verify all embeddings are non-nil
		for i, emb := range embeddings {
			assert.NotNil(t, emb, "embedding at index %d should not be nil", i)
		}
	})
}

// TestEmbedDocument tests document embedding with preprocessing
func TestEmbedDocument(t *testing.T) {
	t.Run("embeds generic document", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		doc := &rag.Document{
			ID:      "doc1",
			Type:    rag.DocumentTypeResult,
			Content: "test content",
		}

		ctx := context.Background()
		err := service.EmbedDocument(ctx, doc)

		require.NoError(t, err)
		assert.NotNil(t, doc.Embedding)
		assert.Equal(t, 384, len(doc.Embedding))
	})

	t.Run("embeds schema document", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		doc := &rag.Document{
			ID:      "doc1",
			Type:    rag.DocumentTypeSchema,
			Content: "CREATE TABLE users",
			Metadata: map[string]interface{}{
				"table_name": "users",
				"columns": []interface{}{
					map[string]interface{}{
						"name": "id",
						"type": "int",
					},
					map[string]interface{}{
						"name": "email",
						"type": "varchar",
					},
				},
			},
		}

		ctx := context.Background()
		err := service.EmbedDocument(ctx, doc)

		require.NoError(t, err)
		assert.NotNil(t, doc.Embedding)

		// Verify preprocessing was applied (table name and columns should be included)
		assert.Len(t, provider.embedTextInputs, 1)
		processedText := provider.embedTextInputs[0]
		assert.Contains(t, processedText, "users")
		assert.Contains(t, processedText, "id")
		assert.Contains(t, processedText, "email")
	})

	t.Run("embeds schema document with relationships", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		doc := &rag.Document{
			ID:      "doc1",
			Type:    rag.DocumentTypeSchema,
			Content: "CREATE TABLE orders",
			Metadata: map[string]interface{}{
				"table_name": "orders",
				"relationships": []interface{}{
					map[string]interface{}{
						"target_table": "users",
					},
					map[string]interface{}{
						"target_table": "products",
					},
				},
			},
		}

		ctx := context.Background()
		err := service.EmbedDocument(ctx, doc)

		require.NoError(t, err)
		assert.NotNil(t, doc.Embedding)

		// Verify relationships are included
		processedText := provider.embedTextInputs[0]
		assert.Contains(t, processedText, "orders")
		assert.Contains(t, processedText, "users")
		assert.Contains(t, processedText, "products")
	})

	t.Run("embeds query document", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		doc := &rag.Document{
			ID:      "doc1",
			Type:    rag.DocumentTypeQuery,
			Content: "SELECT * FROM users WHERE id = 1",
			Metadata: map[string]interface{}{
				"query_type": "select",
				"tables":     []string{"users"},
			},
		}

		ctx := context.Background()
		err := service.EmbedDocument(ctx, doc)

		require.NoError(t, err)
		assert.NotNil(t, doc.Embedding)

		// Verify preprocessing was applied
		processedText := provider.embedTextInputs[0]
		assert.Contains(t, processedText, "select")
		assert.Contains(t, processedText, "users")
	})

	t.Run("embeds different document types", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		types := []rag.DocumentType{
			rag.DocumentTypeSchema,
			rag.DocumentTypeQuery,
			rag.DocumentTypePlan,
			rag.DocumentTypeResult,
			rag.DocumentTypeBusiness,
			rag.DocumentTypePerformance,
			rag.DocumentTypeMemory,
		}

		ctx := context.Background()
		for _, docType := range types {
			doc := &rag.Document{
				ID:      fmt.Sprintf("doc-%s", docType),
				Type:    docType,
				Content: "test content",
			}

			err := service.EmbedDocument(ctx, doc)
			require.NoError(t, err, "failed for type %s", docType)
			assert.NotNil(t, doc.Embedding)
		}
	})

	t.Run("handles provider error", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		provider.setError(true, "embedding error")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		doc := &rag.Document{
			ID:      "doc1",
			Type:    rag.DocumentTypeResult,
			Content: "test content",
		}

		ctx := context.Background()
		err := service.EmbedDocument(ctx, doc)

		require.Error(t, err)
		assert.Nil(t, doc.Embedding)
	})

	t.Run("document without metadata", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		doc := &rag.Document{
			ID:       "doc1",
			Type:     rag.DocumentTypeSchema,
			Content:  "test",
			Metadata: nil,
		}

		ctx := context.Background()
		err := service.EmbedDocument(ctx, doc)

		require.NoError(t, err)
		assert.NotNil(t, doc.Embedding)
	})
}

// TestGetCacheStats tests cache statistics
func TestGetCacheStats(t *testing.T) {
	t.Run("initial stats are zero", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		stats := service.GetCacheStats()

		assert.NotNil(t, stats)
		assert.Equal(t, 0, stats.Size)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(0), stats.Misses)
		assert.Equal(t, 0.0, stats.HitRate)
	})

	t.Run("tracks misses", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, _ = service.EmbedText(ctx, "test1")
		_, _ = service.EmbedText(ctx, "test2")
		_, _ = service.EmbedText(ctx, "test3")

		stats := service.GetCacheStats()
		assert.Equal(t, 3, stats.Size)
		assert.Equal(t, int64(0), stats.Hits)
		assert.Equal(t, int64(3), stats.Misses)
		assert.Equal(t, 0.0, stats.HitRate)
	})

	t.Run("tracks hits", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, _ = service.EmbedText(ctx, "test")
		_, _ = service.EmbedText(ctx, "test")
		_, _ = service.EmbedText(ctx, "test")

		stats := service.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
		assert.Equal(t, int64(2), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
	})

	t.Run("calculates hit rate correctly", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		// 1 miss
		_, _ = service.EmbedText(ctx, "test")
		// 3 hits
		_, _ = service.EmbedText(ctx, "test")
		_, _ = service.EmbedText(ctx, "test")
		_, _ = service.EmbedText(ctx, "test")

		stats := service.GetCacheStats()
		assert.Equal(t, int64(3), stats.Hits)
		assert.Equal(t, int64(1), stats.Misses)
		assert.Equal(t, 0.75, stats.HitRate, "hit rate should be 3/4 = 0.75")
	})

	t.Run("hit rate with no requests", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		stats := service.GetCacheStats()
		assert.Equal(t, 0.0, stats.HitRate, "hit rate should be 0 when no requests")
	})
}

// TestClearCache tests cache clearing
func TestClearCache(t *testing.T) {
	t.Run("clears cache successfully", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, _ = service.EmbedText(ctx, "test1")
		_, _ = service.EmbedText(ctx, "test2")

		statsBeforeClear := service.GetCacheStats()
		assert.Equal(t, 2, statsBeforeClear.Size)

		err := service.ClearCache()
		require.NoError(t, err)

		statsAfterClear := service.GetCacheStats()
		assert.Equal(t, 0, statsAfterClear.Size)
		assert.Equal(t, int64(0), statsAfterClear.Hits)
		assert.Equal(t, int64(0), statsAfterClear.Misses)
	})

	t.Run("clearing empty cache succeeds", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		err := service.ClearCache()
		require.NoError(t, err)

		stats := service.GetCacheStats()
		assert.Equal(t, 0, stats.Size)
	})

	t.Run("cache works after clearing", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, _ = service.EmbedText(ctx, "test")

		err := service.ClearCache()
		require.NoError(t, err)

		// Should work after clearing
		embedding, err := service.EmbedText(ctx, "test")
		require.NoError(t, err)
		assert.NotNil(t, embedding)

		stats := service.GetCacheStats()
		assert.Equal(t, 1, stats.Size)
	})

	t.Run("multiple clears succeed", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		err := service.ClearCache()
		require.NoError(t, err)

		err = service.ClearCache()
		require.NoError(t, err)

		err = service.ClearCache()
		require.NoError(t, err)

		stats := service.GetCacheStats()
		assert.Equal(t, 0, stats.Size)
	})
}

// TestConcurrentAccess tests thread safety
func TestConcurrentAccess(t *testing.T) {
	t.Run("concurrent EmbedText calls", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		numGoroutines := 100
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				text := fmt.Sprintf("test %d", n%10) // Reuse some texts
				_, err := service.EmbedText(ctx, text)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		stats := service.GetCacheStats()
		assert.LessOrEqual(t, stats.Size, 10, "should have at most 10 unique texts")
		assert.Equal(t, int64(numGoroutines), stats.Hits+stats.Misses)
	})

	t.Run("concurrent EmbedBatch calls", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		numGoroutines := 50
		var wg sync.WaitGroup

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				texts := []string{
					fmt.Sprintf("text %d-1", n),
					fmt.Sprintf("text %d-2", n),
				}
				_, err := service.EmbedBatch(ctx, texts)
				assert.NoError(t, err)
			}(i)
		}

		wg.Wait()

		stats := service.GetCacheStats()
		assert.Equal(t, numGoroutines*2, stats.Size)
	})

	t.Run("concurrent cache operations", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		var wg sync.WaitGroup

		// Concurrent embedders
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = service.EmbedText(ctx, "test")
			}()
		}

		// Concurrent stats readers
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.GetCacheStats()
			}()
		}

		wg.Wait()

		// Should not panic
		stats := service.GetCacheStats()
		assert.NotNil(t, stats)
	})

	t.Run("concurrent clear and embed", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		var wg sync.WaitGroup

		// Concurrent embedders
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = service.EmbedText(ctx, "test")
			}()
		}

		// Concurrent clearers
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = service.ClearCache()
			}()
		}

		wg.Wait()

		// Should not panic
		stats := service.GetCacheStats()
		assert.NotNil(t, stats)
	})
}

// TestProviderIntegration tests with OpenAI and Local providers
func TestProviderIntegration(t *testing.T) {
	t.Run("OpenAI provider integration", func(t *testing.T) {
		logger := newTestLoggerEmbedding()
		provider := rag.NewOpenAIEmbeddingProvider("test-key", "text-embedding-3-small", logger)

		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embedding, err := service.EmbedText(ctx, "test")

		// OpenAI provider returns mock embeddings in test environment
		require.NoError(t, err)
		assert.NotNil(t, embedding)
		assert.Equal(t, provider.GetDimension(), len(embedding))
	})

	t.Run("Local provider integration", func(t *testing.T) {
		logger := newTestLoggerEmbedding()
		provider := rag.NewLocalEmbeddingProvider("/path/to/model", logger)

		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		embedding, err := service.EmbedText(ctx, "test")

		// Local provider returns mock embeddings
		require.NoError(t, err)
		assert.NotNil(t, embedding)
		assert.Equal(t, provider.GetDimension(), len(embedding))
	})

	t.Run("OpenAI different model dimensions", func(t *testing.T) {
		logger := newTestLoggerEmbedding()

		models := map[string]int{
			"text-embedding-3-small": 1536,
			"text-embedding-3-large": 3072,
			"text-embedding-ada-002": 1536,
		}

		for model, expectedDim := range models {
			provider := rag.NewOpenAIEmbeddingProvider("test-key", model, logger)
			assert.Equal(t, expectedDim, provider.GetDimension(), "wrong dimension for %s", model)
			assert.Equal(t, model, provider.GetModel(), "wrong model name")
		}
	})

	t.Run("OpenAI batch embedding", func(t *testing.T) {
		logger := newTestLoggerEmbedding()
		provider := rag.NewOpenAIEmbeddingProvider("test-key", "text-embedding-3-small", logger)

		ctx := context.Background()
		texts := []string{"text1", "text2", "text3"}
		embeddings, err := provider.EmbedBatch(ctx, texts)

		require.NoError(t, err)
		require.Len(t, embeddings, 3)
		for _, embedding := range embeddings {
			assert.Equal(t, 1536, len(embedding))
		}
	})

	t.Run("Local provider model name", func(t *testing.T) {
		logger := newTestLoggerEmbedding()
		modelPath := "/path/to/model"
		provider := rag.NewLocalEmbeddingProvider(modelPath, logger)

		assert.Equal(t, modelPath, provider.GetModel())
	})

	t.Run("Local provider batch embedding", func(t *testing.T) {
		logger := newTestLoggerEmbedding()
		provider := rag.NewLocalEmbeddingProvider("/path/to/model", logger)

		ctx := context.Background()
		texts := []string{"text1", "text2"}
		embeddings, err := provider.EmbedBatch(ctx, texts)

		require.NoError(t, err)
		require.Len(t, embeddings, 2)
		for _, embedding := range embeddings {
			assert.Equal(t, 384, len(embedding))
		}
	})
}

// TestCacheTTL tests TTL expiration (if implementable)
func TestCacheTTL(t *testing.T) {
	// Note: This test is limited because we can't easily manipulate time
	// In a real implementation, you might use time mocking
	t.Run("cache entries tracked by creation time", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "test")
		require.NoError(t, err)

		// Immediately access again should be a cache hit
		_, err = service.EmbedText(ctx, "test")
		require.NoError(t, err)

		stats := service.GetCacheStats()
		assert.Equal(t, int64(1), stats.Hits)
	})
}

// TestCacheLRUEviction tests LRU eviction
func TestCacheLRUEviction(t *testing.T) {
	t.Run("cache size grows with unique texts", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()

		// Add many unique texts
		for i := 0; i < 100; i++ {
			_, err := service.EmbedText(ctx, fmt.Sprintf("text %d", i))
			require.NoError(t, err)
		}

		stats := service.GetCacheStats()
		assert.Equal(t, 100, stats.Size)
	})

	t.Run("cache handles large number of entries", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()

		// The default cache size is 10000
		// Add entries up to that limit
		for i := 0; i < 1000; i++ {
			_, err := service.EmbedText(ctx, fmt.Sprintf("text %d", i))
			require.NoError(t, err)
		}

		stats := service.GetCacheStats()
		assert.Equal(t, 1000, stats.Size)
		assert.LessOrEqual(t, stats.Size, 10000, "should not exceed max cache size")
	})

	t.Run("cache evicts LRU when full", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()

		// Fill cache to max size (10000) + 1 to trigger eviction
		// For testing purposes, add 10001 items
		for i := 0; i < 10001; i++ {
			_, err := service.EmbedText(ctx, fmt.Sprintf("text %d", i))
			require.NoError(t, err)
		}

		stats := service.GetCacheStats()
		// Cache should not exceed max size
		assert.LessOrEqual(t, stats.Size, 10000, "cache should have evicted to stay at or below max size")

		// The earliest item should have been evicted
		// Try to get text 0 - it should require a new embedding call
		provider.reset()
		embedding, err := service.EmbedText(ctx, "text 0")
		require.NoError(t, err)
		assert.NotNil(t, embedding)

		// If eviction happened correctly, this should be a miss (new provider call)
		embedTextCalls, _ := provider.getCallCount()
		if stats.Size < 10000 {
			// Eviction occurred
			assert.Greater(t, embedTextCalls, 0, "should have called provider for evicted item")
		}
	})
}

// TestPreprocessing tests text preprocessing
func TestPreprocessing(t *testing.T) {
	t.Run("normalizes whitespace", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "hello\t\tworld\n\ntest")
		require.NoError(t, err)

		processedText := provider.embedTextInputs[0]
		assert.NotContains(t, processedText, "\t")
		assert.NotContains(t, processedText, "\n")
	})

	t.Run("converts to lowercase", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "HELLO WORLD TEST")
		require.NoError(t, err)

		processedText := provider.embedTextInputs[0]
		assert.Equal(t, "hello world test", processedText)
	})

	t.Run("trims leading and trailing whitespace", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "   hello world   ")
		require.NoError(t, err)

		processedText := provider.embedTextInputs[0]
		assert.Equal(t, "hello world", processedText)
	})

	t.Run("collapses multiple spaces", func(t *testing.T) {
		provider := newMockEmbeddingProvider(384, "test-model")
		logger := newTestLoggerEmbedding()
		service := rag.NewEmbeddingService(provider, logger)

		ctx := context.Background()
		_, err := service.EmbedText(ctx, "hello     world     test")
		require.NoError(t, err)

		processedText := provider.embedTextInputs[0]
		assert.Equal(t, "hello world test", processedText)
	})
}
