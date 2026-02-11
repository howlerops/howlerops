package rag

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

// BenchmarkHybridSearchRRF benchmarks hybrid search with RRF fusion
func BenchmarkHybridSearchRRF(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
			store, cleanup := setupBenchStore(b, size)
			defer cleanup()

			ctx := context.Background()
			query := "database table schema query"
			embedding := generateTestEmbedding(query, 0.0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.HybridSearch(ctx, query, embedding, 10)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkVectorSearchOnly benchmarks pure vector search for comparison
func BenchmarkVectorSearchOnly(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
			store, cleanup := setupBenchStore(b, size)
			defer cleanup()

			ctx := context.Background()
			embedding := generateTestEmbedding("database table schema query", 0.0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.SearchSimilar(ctx, embedding, 10, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTextSearchOnly benchmarks pure text search for comparison
func BenchmarkTextSearchOnly(b *testing.B) {
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("docs_%d", size), func(b *testing.B) {
			store, cleanup := setupBenchStore(b, size)
			defer cleanup()

			ctx := context.Background()
			query := "database table schema query"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.SearchByText(ctx, query, 10, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkParallelSearches compares sequential vs parallel search execution
func BenchmarkParallelSearches(b *testing.B) {
	store, cleanup := setupBenchStore(b, 10000)
	defer cleanup()

	ctx := context.Background()
	query := "database table schema"
	embedding := generateTestEmbedding(query, 0.0)

	b.Run("Sequential", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = store.SearchSimilar(ctx, embedding, 10, nil)
			_, _ = store.SearchByText(ctx, query, 10, nil)
		}
	})

	b.Run("Parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = store.HybridSearch(ctx, query, embedding, 10)
		}
	})
}

// BenchmarkRRFConstantVariations benchmarks different RRF constant values
func BenchmarkRRFConstantVariations(b *testing.B) {
	rrfConstants := []int{20, 60, 100}

	for _, constant := range rrfConstants {
		b.Run(fmt.Sprintf("k_%d", constant), func(b *testing.B) {
			tmpDir, err := os.MkdirTemp("", "rag_bench_rrf_*")
			if err != nil {
				b.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			config := &SQLiteVectorConfig{
				Path:        filepath.Join(tmpDir, "bench.db"),
				VectorSize:  128,
				CacheSizeMB: 64,
				MMapSizeMB:  128,
				WALEnabled:  true,
				RRFConstant: constant,
			}

			logger := logrus.New()
			logger.SetLevel(logrus.ErrorLevel)

			store, err := NewSQLiteVectorStore(config, logger)
			if err != nil {
				b.Fatal(err)
			}

			ctx := context.Background()
			if err := store.Initialize(ctx); err != nil {
				b.Fatal(err)
			}

			// Index documents
			docs := make([]*Document, 1000)
			for i := 0; i < 1000; i++ {
				docs[i] = &Document{
					ID:           fmt.Sprintf("doc%d", i),
					ConnectionID: "bench-conn",
					Type:         DocumentTypeQuery,
					Content:      fmt.Sprintf("database query %d with schema table information", i),
					Embedding:    generateTestEmbedding(fmt.Sprintf("query %d", i), float64(i)*0.01),
				}
			}

			if err := store.BatchIndexDocuments(ctx, docs); err != nil {
				b.Fatal(err)
			}

			query := "database schema query"
			embedding := generateTestEmbedding(query, 0.0)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.HybridSearch(ctx, query, embedding, 10)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRRFFusion benchmarks just the RRF fusion step (without search)
func BenchmarkRRFFusion(b *testing.B) {
	store := &SQLiteVectorStore{
		rrfConstant:  60,
		vectorWeight: 1.0,
		textWeight:   1.0,
		logger:       logrus.New(),
	}

	// Generate sample search results
	vectorResults := make([]*Document, 30)
	textResults := make([]*Document, 30)

	for i := 0; i < 30; i++ {
		vectorResults[i] = &Document{
			ID:           fmt.Sprintf("doc%d", i),
			ConnectionID: "test",
			Type:         DocumentTypeQuery,
			Content:      fmt.Sprintf("content %d", i),
			Score:        float32(1.0 / (float64(i) + 1)),
		}
		textResults[i] = &Document{
			ID:           fmt.Sprintf("doc%d", (i*2)%30), // Some overlap
			ConnectionID: "test",
			Type:         DocumentTypeQuery,
			Content:      fmt.Sprintf("content %d", (i*2)%30),
			Score:        float32(1.0 / (float64(i) + 1)),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.fuseWithRRF(vectorResults, textResults, 10)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCandidateCount benchmarks different candidate counts for RRF
func BenchmarkCandidateCount(b *testing.B) {
	store, cleanup := setupBenchStore(b, 1000)
	defer cleanup()

	ctx := context.Background()
	query := "database schema"
	embedding := generateTestEmbedding(query, 0.0)

	// Test with different multipliers (k * multiplier candidates)
	for _, k := range []int{10, 30, 50} {
		b.Run(fmt.Sprintf("k_%d", k), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				// Manually execute with different candidate counts
				vectorResults, _ := store.SearchSimilar(ctx, embedding, k, nil)
				textResults, _ := store.SearchByText(ctx, query, k, nil)
				_, err := store.fuseWithRRF(vectorResults, textResults, 10)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// Helper functions

func setupBenchStore(b *testing.B, docCount int) (*SQLiteVectorStore, func()) {
	tmpDir, err := os.MkdirTemp("", "rag_bench_*")
	if err != nil {
		b.Fatal(err)
	}

	config := &SQLiteVectorConfig{
		Path:        filepath.Join(tmpDir, "bench.db"),
		VectorSize:  128,
		CacheSizeMB: 64,
		MMapSizeMB:  128,
		WALEnabled:  true,
		RRFConstant: 60,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	store, err := NewSQLiteVectorStore(config, logger)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()
	if err := store.Initialize(ctx); err != nil {
		b.Fatal(err)
	}

	// Index test documents
	docs := make([]*Document, docCount)
	for i := 0; i < docCount; i++ {
		docs[i] = &Document{
			ID:           fmt.Sprintf("doc%d", i),
			ConnectionID: "bench-conn",
			Type:         DocumentTypeQuery,
			Content:      fmt.Sprintf("database query %d with schema table information", i),
			Embedding:    generateTestEmbedding(fmt.Sprintf("query %d", i), float64(i)*0.01),
		}
	}

	if err := store.BatchIndexDocuments(ctx, docs); err != nil {
		b.Fatal(err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

// generateBenchEmbedding generates a test embedding for benchmarks
func generateBenchEmbedding(content string, seed float64) []float32 {
	embedding := make([]float32, 128)
	hash := 0.0
	for i, c := range content {
		hash += float64(c) * float64(i+1)
	}
	hash += seed * 1000

	for i := range embedding {
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
