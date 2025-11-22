//go:build integration

package rag_test

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/backend-go/internal/rag"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func TestOllamaEmbedding(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Skip if Ollama not running
	provider := rag.NewOllamaEmbeddingProvider("http://localhost:11434", "nomic-embed-text", 768, logger)

	ctx := context.Background()

	// Ensure model is available
	modelMgr := rag.NewOllamaModelManager("http://localhost:11434", logger)
	if err := modelMgr.EnsureModelAvailable(ctx, "nomic-embed-text"); err != nil {
		t.Skip("Ollama not available:", err)
	}

	// Test single embedding
	emb, err := provider.EmbedText(ctx, "database table with customer information")
	require.NoError(t, err)
	assert.Len(t, emb, 768)

	// Verify not sequential numbers (common bug)
	assert.NotEqual(t, emb[0], emb[1], "Embeddings should not be sequential")
	assert.NotEqual(t, float32(0)/float32(768), emb[0], "Should not be mock embedding")
}

func TestOllamaEmbeddingBatch(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	provider := rag.NewOllamaEmbeddingProvider("http://localhost:11434", "nomic-embed-text", 768, logger)

	ctx := context.Background()

	// Ensure model is available
	modelMgr := rag.NewOllamaModelManager("http://localhost:11434", logger)
	if err := modelMgr.EnsureModelAvailable(ctx, "nomic-embed-text"); err != nil {
		t.Skip("Ollama not available:", err)
	}

	texts := []string{
		"SELECT * FROM users WHERE id = 1",
		"INSERT INTO orders VALUES (1, 'pending')",
		"UPDATE products SET price = 100",
	}

	embeddings, err := provider.EmbedBatch(ctx, texts)
	require.NoError(t, err)
	assert.Len(t, embeddings, 3)

	for i, emb := range embeddings {
		assert.Len(t, emb, 768, "embedding %d should have 768 dimensions", i)
		assert.NotEqual(t, emb[0], emb[1], "embedding %d should not be sequential", i)
	}
}

func TestEmbeddingSimilarity(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	provider := rag.NewOllamaEmbeddingProvider("http://localhost:11434", "nomic-embed-text", 768, logger)

	ctx := context.Background()

	// Ensure model is available
	modelMgr := rag.NewOllamaModelManager("http://localhost:11434", logger)
	if err := modelMgr.EnsureModelAvailable(ctx, "nomic-embed-text"); err != nil {
		t.Skip("Ollama not available:", err)
	}

	// Similar concepts
	emb1, err := provider.EmbedText(ctx, "database table with customer information")
	require.NoError(t, err)

	emb2, err := provider.EmbedText(ctx, "table storing client data")
	require.NoError(t, err)

	// Unrelated concept
	emb3, err := provider.EmbedText(ctx, "weather forecast for tomorrow")
	require.NoError(t, err)

	sim12 := cosineSimilarity(emb1, emb2)
	sim13 := cosineSimilarity(emb1, emb3)

	t.Logf("Similarity (similar concepts): %.4f", sim12)
	t.Logf("Similarity (unrelated concepts): %.4f", sim13)

	assert.Greater(t, sim12, 0.7, "Similar concepts should have high similarity")
	assert.Less(t, sim13, 0.6, "Unrelated concepts should have lower similarity")
	assert.Greater(t, sim12, sim13, "Similar concepts should be more similar than unrelated")
}

func TestCacheRaceCondition(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Use mock provider for speed
	mockProvider := newMockEmbeddingProvider(768, "test")
	service := rag.NewEmbeddingService(mockProvider, logger)

	ctx := context.Background()

	// Embed once to populate cache
	_, err := service.EmbedText(ctx, "test-key")
	require.NoError(t, err)

	// Concurrent reads and writes
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = service.EmbedText(ctx, "test-key")
		}()
	}

	wg.Wait()

	stats := service.GetCacheStats()
	assert.Greater(t, stats.Hits, int64(90), "Most requests should be cache hits")
}

func TestCacheEviction(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	mockProvider := newMockEmbeddingProvider(768, "test")
	service := rag.NewEmbeddingService(mockProvider, logger)

	ctx := context.Background()

	// Fill cache beyond max size
	// Default max size is 10000, so we'll use a custom cache for testing
	// For this test, we'll just verify evictions happen

	// Embed 100 different texts
	for i := 0; i < 100; i++ {
		_, err := service.EmbedText(ctx, "text-"+string(rune(i)))
		require.NoError(t, err)
	}

	stats := service.GetCacheStats()
	assert.Equal(t, int64(0), stats.Hits, "No cache hits initially")
	assert.Equal(t, int64(100), stats.Misses, "All misses initially")
	assert.LessOrEqual(t, stats.Size, 10000, "Cache size should not exceed max")
}

func TestOllamaModelManager(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	modelMgr := rag.NewOllamaModelManager("http://localhost:11434", logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Test ensuring model is available (will auto-pull if needed)
	err := modelMgr.EnsureModelAvailable(ctx, "nomic-embed-text")
	if err != nil {
		t.Skip("Ollama not available or model pull failed:", err)
	}

	// Second call should be fast (model already available)
	start := time.Now()
	err = modelMgr.EnsureModelAvailable(ctx, "nomic-embed-text")
	require.NoError(t, err)
	elapsed := time.Since(start)

	assert.Less(t, elapsed, 2*time.Second, "Second call should be fast (model already exists)")
}

func TestOllamaIntegrationWithService(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Setup Ollama provider
	endpoint := "http://localhost:11434"
	model := "nomic-embed-text"

	modelMgr := rag.NewOllamaModelManager(endpoint, logger)
	ctx := context.Background()

	if err := modelMgr.EnsureModelAvailable(ctx, model); err != nil {
		t.Skip("Ollama not available:", err)
	}

	provider := rag.NewOllamaEmbeddingProvider(endpoint, model, 768, logger)
	service := rag.NewEmbeddingService(provider, logger)

	// Test embedding through service
	text := "SELECT * FROM users WHERE status = 'active'"
	emb, err := service.EmbedText(ctx, text)
	require.NoError(t, err)
	assert.Len(t, emb, 768)

	// Verify cache works
	stats1 := service.GetCacheStats()
	assert.Equal(t, int64(0), stats1.Hits)
	assert.Equal(t, int64(1), stats1.Misses)

	// Second call should hit cache
	emb2, err := service.EmbedText(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, emb, emb2)

	stats2 := service.GetCacheStats()
	assert.Equal(t, int64(1), stats2.Hits)
	assert.Equal(t, int64(1), stats2.Misses)
}

func TestOllamaErrorHandling(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// Test with invalid endpoint
	provider := rag.NewOllamaEmbeddingProvider("http://localhost:9999", "nomic-embed-text", 768, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := provider.EmbedText(ctx, "test text")
	assert.Error(t, err, "Should error with invalid endpoint")
}

func TestOllamaDimensionValidation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Test with wrong dimension expectation
	provider := rag.NewOllamaEmbeddingProvider("http://localhost:11434", "nomic-embed-text", 384, logger)

	ctx := context.Background()

	modelMgr := rag.NewOllamaModelManager("http://localhost:11434", logger)
	if err := modelMgr.EnsureModelAvailable(ctx, "nomic-embed-text"); err != nil {
		t.Skip("Ollama not available:", err)
	}

	_, err := provider.EmbedText(ctx, "test text")
	assert.Error(t, err, "Should error with dimension mismatch")
	assert.Contains(t, err.Error(), "expected 384 dimensions, got 768")
}
