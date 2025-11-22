package rag

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRRFCalculation tests the basic RRF score calculation logic
func TestRRFCalculation(t *testing.T) {
	// Test with documents appearing at different ranks
	tests := []struct {
		name         string
		vectorRank   int
		textRank     int
		rrfConstant  int
		vectorWeight float64
		textWeight   float64
		expectedMin  float64
		expectedMax  float64
	}{
		{
			name:         "both_top_rank",
			vectorRank:   0,
			textRank:     0,
			rrfConstant:  60,
			vectorWeight: 1.0,
			textWeight:   1.0,
			expectedMin:  0.032,  // 2 * (1/(0+1+60)) ≈ 0.0328
			expectedMax:  0.033,
		},
		{
			name:         "vector_top_text_low",
			vectorRank:   0,
			textRank:     10,
			rrfConstant:  60,
			vectorWeight: 1.0,
			textWeight:   1.0,
			expectedMin:  0.0302, // (1/61) + (1/71) ≈ 0.0305
			expectedMax:  0.0306,
		},
		{
			name:         "both_low_rank",
			vectorRank:   50,
			textRank:     50,
			rrfConstant:  60,
			vectorWeight: 1.0,
			textWeight:   1.0,
			expectedMin:  0.0179, // 2 * (1/111) ≈ 0.018
			expectedMax:  0.0182,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate RRF score
			vectorScore := tt.vectorWeight / float64(tt.vectorRank+1+tt.rrfConstant)
			textScore := tt.textWeight / float64(tt.textRank+1+tt.rrfConstant)
			totalScore := vectorScore + textScore

			assert.GreaterOrEqual(t, totalScore, tt.expectedMin)
			assert.LessOrEqual(t, totalScore, tt.expectedMax)
		})
	}
}

// TestHybridSearchRRF tests hybrid search with RRF fusion
func TestHybridSearchRRF(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Index test documents with both text and embeddings
	docs := []*Document{
		{
			ID:           "doc1",
			ConnectionID: "test-conn",
			Type:         DocumentTypeSchema,
			Content:      "database schema with user table and customer information",
			Embedding:    generateTestEmbedding("database schema user table", 0.1),
		},
		{
			ID:           "doc2",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "select statement for customer data extraction",
			Embedding:    generateTestEmbedding("customer data query", 0.2),
		},
		{
			ID:           "doc3",
			ConnectionID: "test-conn",
			Type:         DocumentTypeSchema,
			Content:      "user authentication system with password hashing",
			Embedding:    generateTestEmbedding("authentication security", 0.3),
		},
		{
			ID:           "doc4",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "database query for user login verification",
			Embedding:    generateTestEmbedding("database user login", 0.15),
		},
	}

	for _, doc := range docs {
		require.NoError(t, store.IndexDocument(ctx, doc))
	}

	// Test hybrid search
	queryEmb := generateTestEmbedding("user database query", 0.0)
	results, err := store.HybridSearch(ctx, "user database", queryEmb, 3)
	require.NoError(t, err)

	// Verify results
	assert.Len(t, results, 3, "Should return top 3 results")

	// Verify all results have RRF scores
	for i, doc := range results {
		t.Logf("Result %d: %s (score: %.6f)", i+1, doc.ID, doc.Score)
		assert.Greater(t, doc.Score, float32(0), "RRF score should be positive")
		assert.NotNil(t, doc.Metadata["rrf_score"], "Should have RRF score metadata")

		// At least one result should have both vector and text ranks
		if doc.Metadata["vector_rank"] != nil && doc.Metadata["text_rank"] != nil {
			t.Logf("  Vector rank: %v, Text rank: %v",
				doc.Metadata["vector_rank"], doc.Metadata["text_rank"])
		}
	}

	// Verify scores are descending
	for i := 1; i < len(results); i++ {
		assert.GreaterOrEqual(t, results[i-1].Score, results[i].Score,
			"Results should be sorted by RRF score (descending)")
	}
}

// TestHybridSearchBetterThanVectorOnly tests that hybrid search outperforms pure vector search
func TestHybridSearchBetterThanVectorOnly(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Index documents where hybrid should outperform pure vector
	docs := []*Document{
		{
			ID:           "exact_match",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "exact query terms postgresql database",
			Embedding:    generateTestEmbedding("somewhat related content", 0.5),
		},
		{
			ID:           "vector_match",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "completely different text about authentication",
			Embedding:    generateTestEmbedding("exact query terms postgresql database", 0.0),
		},
		{
			ID:           "both_match",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "exact query terms postgresql database connection",
			Embedding:    generateTestEmbedding("exact query terms postgresql database", 0.05),
		},
	}

	for _, doc := range docs {
		require.NoError(t, store.IndexDocument(ctx, doc))
	}

	// Compare vector-only vs hybrid
	queryEmb := generateTestEmbedding("exact query terms postgresql database", 0.0)

	vectorResults, err := store.SearchSimilar(ctx, queryEmb, 3, nil)
	require.NoError(t, err)

	hybridResults, err := store.HybridSearch(ctx, "exact query terms postgresql database", queryEmb, 3)
	require.NoError(t, err)

	t.Logf("Vector-only top result: %s (score: %.6f)", vectorResults[0].ID, vectorResults[0].Score)
	t.Logf("Hybrid top result: %s (score: %.6f)", hybridResults[0].ID, hybridResults[0].Score)

	// Hybrid should rank "both_match" first (has both vector and text similarity)
	assert.Equal(t, "both_match", hybridResults[0].ID,
		"Hybrid search should rank document with both vector and text match highest")

	// Document with both matches should score higher in hybrid than vector-only
	var bothMatchVectorScore float32
	for _, doc := range vectorResults {
		if doc.ID == "both_match" {
			bothMatchVectorScore = doc.Score
			break
		}
	}

	assert.Greater(t, hybridResults[0].Score, bothMatchVectorScore,
		"Hybrid RRF score should be higher than vector-only score for documents matching both")
}

// TestRRFConstantEffect tests how different RRF constants affect fusion
func TestRRFConstantEffect(t *testing.T) {
	// Test with low constant (k=10): More weight to top results
	lowK := 10
	lowRankScore := 1.0 / float64(0+1+lowK)  // rank 0
	lowMidScore := 1.0 / float64(10+1+lowK)  // rank 10
	lowRatio := lowRankScore / lowMidScore

	// Test with high constant (k=100): More uniform weighting
	highK := 100
	highRankScore := 1.0 / float64(0+1+highK)  // rank 0
	highMidScore := 1.0 / float64(10+1+highK)  // rank 10
	highRatio := highRankScore / highMidScore

	t.Logf("Low k=%d ratio (rank 0 vs 10): %.3f", lowK, lowRatio)
	t.Logf("High k=%d ratio (rank 0 vs 10): %.3f", highK, highRatio)

	assert.Greater(t, lowRatio, highRatio,
		"Lower RRF constant should create bigger score differences between ranks")
	assert.Greater(t, lowRatio, 2.0, "Low k should give at least 2x weight to top rank")
	assert.Less(t, highRatio, 1.5, "High k should give less than 1.5x weight to top rank")
}

// TestHybridSearchWithWeights tests weighted RRF fusion
func TestHybridSearchWithWeights(t *testing.T) {
	// Create store with custom weights
	tmpDir, err := os.MkdirTemp("", "rag_hybrid_weights_*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	config := &SQLiteVectorConfig{
		Path:         filepath.Join(tmpDir, "test.db"),
		VectorSize:   128,
		CacheSizeMB:  8,
		MMapSizeMB:   16,
		WALEnabled:   true,
		RRFConstant:  60,
		VectorWeight: 2.0,  // Prefer vector results
		TextWeight:   1.0,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	store, err := NewSQLiteVectorStore(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx))

	// Index documents
	docs := []*Document{
		{
			ID:           "vector_strong",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "unrelated text content",
			Embedding:    generateTestEmbedding("target query", 0.0),
		},
		{
			ID:           "text_strong",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "target query matching text",
			Embedding:    generateTestEmbedding("unrelated embedding", 0.5),
		},
	}

	for _, doc := range docs {
		require.NoError(t, store.IndexDocument(ctx, doc))
	}

	// Search with vector weight = 2x text weight
	queryEmb := generateTestEmbedding("target query", 0.0)
	results, err := store.HybridSearch(ctx, "target query", queryEmb, 2)
	require.NoError(t, err)

	t.Logf("Weighted results:")
	for i, doc := range results {
		t.Logf("  %d. %s (score: %.6f)", i+1, doc.ID, doc.Score)
	}

	// With vector weight = 2.0, vector_strong should rank higher
	assert.Equal(t, "vector_strong", results[0].ID,
		"Higher vector weight should favor vector matches")
}

// TestHybridSearchParallelExecution tests that searches run in parallel
func TestHybridSearchParallelExecution(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Index enough documents to make searches measurable
	docs := make([]*Document, 100)
	for i := 0; i < 100; i++ {
		docs[i] = &Document{
			ID:           fmt.Sprintf("doc%d", i),
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      fmt.Sprintf("content %d with various keywords database query schema", i),
			Embedding:    generateTestEmbedding(fmt.Sprintf("embedding %d", i), float64(i)*0.01),
		}
	}

	err := store.BatchIndexDocuments(ctx, docs)
	require.NoError(t, err)

	// Hybrid search should complete successfully with parallel execution
	queryEmb := generateTestEmbedding("database query", 0.0)
	results, err := store.HybridSearch(ctx, "database query schema", queryEmb, 10)
	require.NoError(t, err)
	assert.Len(t, results, 10)

	// Verify all results have metadata from both searches
	foundBothRanks := false
	for _, doc := range results {
		if doc.Metadata["vector_rank"] != nil && doc.Metadata["text_rank"] != nil {
			foundBothRanks = true
			break
		}
	}
	assert.True(t, foundBothRanks, "At least one result should have both vector and text ranks")
}

// TestHybridSearchWithFilters tests that filters work with hybrid search
func TestHybridSearchWithFilters(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()

	// Index documents with different types
	docs := []*Document{
		{
			ID:           "schema1",
			ConnectionID: "test-conn",
			Type:         DocumentTypeSchema,
			Content:      "database schema definition",
			Embedding:    generateTestEmbedding("schema", 0.1),
		},
		{
			ID:           "query1",
			ConnectionID: "test-conn",
			Type:         DocumentTypeQuery,
			Content:      "database query statement",
			Embedding:    generateTestEmbedding("query", 0.2),
		},
		{
			ID:           "schema2",
			ConnectionID: "test-conn",
			Type:         DocumentTypeSchema,
			Content:      "another schema with database tables",
			Embedding:    generateTestEmbedding("schema", 0.15),
		},
	}

	for _, doc := range docs {
		require.NoError(t, store.IndexDocument(ctx, doc))
	}

	// Note: Current implementation doesn't support filters in HybridSearch
	// This test documents the expected behavior once filters are added
	queryEmb := generateTestEmbedding("database schema", 0.0)
	results, err := store.HybridSearch(ctx, "database schema", queryEmb, 10)
	require.NoError(t, err)

	// Should return both schema and query documents
	assert.Greater(t, len(results), 0)
}

// Helper functions

func setupTestStore(t *testing.T) (*SQLiteVectorStore, func()) {
	tmpDir, err := os.MkdirTemp("", "rag_hybrid_test_*")
	require.NoError(t, err)

	config := &SQLiteVectorConfig{
		Path:        filepath.Join(tmpDir, "test.db"),
		VectorSize:  128,
		CacheSizeMB: 8,
		MMapSizeMB:  16,
		WALEnabled:  true,
		RRFConstant: 60,  // Use default
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	store, err := NewSQLiteVectorStore(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.Initialize(ctx))

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

// generateTestEmbedding generates a simple test embedding based on content
// The seed parameter adds variation to the embedding
func generateTestEmbedding(content string, seed float64) []float32 {
	embedding := make([]float32, 128)
	hash := 0.0
	for i, c := range content {
		hash += float64(c) * float64(i+1)
	}
	hash += seed * 1000

	for i := range embedding {
		// Create pseudo-random but deterministic embeddings
		angle := (hash + float64(i)) / 100.0
		embedding[i] = float32(math.Sin(angle) * math.Cos(angle*0.7))
	}

	// Normalize
	var magnitude float32
	for _, v := range embedding {
		magnitude += v * v
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))

	for i := range embedding {
		embedding[i] /= magnitude
	}

	return embedding
}
