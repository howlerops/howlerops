package rag

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// setupBenchmarkStore creates a vector store for benchmarking
func setupBenchmarkStore(b *testing.B, enableHNSW bool) *SQLiteVectorStore {
	b.Helper()

	// Create temp database
	dbPath := b.TempDir() + "/bench_vectors.db"

	config := &SQLiteVectorConfig{
		Path:        dbPath,
		CacheSizeMB: 128,
		MMapSizeMB:  256,
		WALEnabled:  true,
		Timeout:     10 * time.Second,
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in benchmarks

	store, err := NewSQLiteVectorStore(config, logger)
	if err != nil {
		b.Fatalf("Failed to create vector store: %v", err)
	}

	ctx := context.Background()
	if err := store.Initialize(ctx); err != nil {
		b.Fatalf("Failed to initialize vector store: %v", err)
	}

	// If HNSW not available and requested, skip benchmark
	if enableHNSW && !store.hnswAvailable {
		b.Skip("HNSW not available, skipping HNSW benchmark")
	}

	// Disable HNSW if requested
	if !enableHNSW {
		store.hnswAvailable = false
	}

	return store
}

// randomEmbedding generates a random embedding vector
func randomEmbedding(dim int) []float32 {
	embedding := make([]float32, dim)
	for i := range embedding {
		embedding[i] = rand.Float32()
	}
	return embedding
}

// generateTestDocuments creates test documents with embeddings
func generateTestDocuments(count int, dim int) []*Document {
	docs := make([]*Document, count)
	for i := 0; i < count; i++ {
		docs[i] = &Document{
			ConnectionID: "test-connection",
			Type:         DocumentTypeSchema,
			Content:      "Test document content for benchmarking",
			Embedding:    randomEmbedding(dim),
			Metadata: map[string]interface{}{
				"index": i,
			},
		}
	}
	return docs
}

// BenchmarkVectorSearchBruteForce benchmarks brute force vector search
func BenchmarkVectorSearchBruteForce(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	dim := 768

	for _, size := range sizes {
		b.Run(b.Name()+"_"+string(rune(size)), func(b *testing.B) {
			store := setupBenchmarkStore(b, false) // Disable HNSW
			ctx := context.Background()

			// Index documents
			docs := generateTestDocuments(size, dim)
			for _, doc := range docs {
				if err := store.IndexDocument(ctx, doc); err != nil {
					b.Fatalf("Failed to index document: %v", err)
				}
			}

			// Prepare query embedding
			queryEmbedding := randomEmbedding(dim)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.SearchSimilar(ctx, queryEmbedding, 10, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkVectorSearchHNSW benchmarks HNSW-accelerated vector search
func BenchmarkVectorSearchHNSW(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	dim := 768

	for _, size := range sizes {
		b.Run(b.Name()+"_"+string(rune(size)), func(b *testing.B) {
			store := setupBenchmarkStore(b, true) // Enable HNSW
			ctx := context.Background()

			// Index documents
			docs := generateTestDocuments(size, dim)
			for _, doc := range docs {
				if err := store.IndexDocument(ctx, doc); err != nil {
					b.Fatalf("Failed to index document: %v", err)
				}
			}

			// Prepare query embedding
			queryEmbedding := randomEmbedding(dim)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := store.SearchSimilar(ctx, queryEmbedding, 10, nil)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkIndexBuild benchmarks HNSW index build time
func BenchmarkIndexBuild(b *testing.B) {
	sizes := []int{100, 1000, 5000}
	dim := 768

	for _, size := range sizes {
		b.Run(b.Name()+"_"+string(rune(size)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				store := setupBenchmarkStore(b, true)
				ctx := context.Background()
				docs := generateTestDocuments(size, dim)
				b.StartTimer()

				// Index all documents (triggers HNSW index build)
				for _, doc := range docs {
					if err := store.IndexDocument(ctx, doc); err != nil {
						b.Fatalf("Failed to index document: %v", err)
					}
				}
			}
		})
	}
}

// BenchmarkBatchIndexing benchmarks batch insertion performance
func BenchmarkBatchIndexing(b *testing.B) {
	dim := 768
	batchSize := 100

	b.Run("BruteForce", func(b *testing.B) {
		store := setupBenchmarkStore(b, false)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			docs := generateTestDocuments(batchSize, dim)
			if err := store.BatchIndexDocuments(ctx, docs); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("HNSW", func(b *testing.B) {
		store := setupBenchmarkStore(b, true)
		ctx := context.Background()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			docs := generateTestDocuments(batchSize, dim)
			if err := store.BatchIndexDocuments(ctx, docs); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkSearchWithFilters benchmarks filtered searches
func BenchmarkSearchWithFilters(b *testing.B) {
	store := setupBenchmarkStore(b, true)
	ctx := context.Background()
	dim := 768

	// Index documents with different connection IDs
	for i := 0; i < 1000; i++ {
		doc := &Document{
			ConnectionID: "conn-" + string(rune(i%10)),
			Type:         DocumentTypeSchema,
			Content:      "Test document",
			Embedding:    randomEmbedding(dim),
		}
		if err := store.IndexDocument(ctx, doc); err != nil {
			b.Fatalf("Failed to index document: %v", err)
		}
	}

	queryEmbedding := randomEmbedding(dim)
	filter := map[string]interface{}{
		"connection_id": "conn-5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := store.SearchSimilar(ctx, queryEmbedding, 10, filter)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestMain ensures cleanup after benchmarks
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Cleanup
	os.Exit(code)
}
