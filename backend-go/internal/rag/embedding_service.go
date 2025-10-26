package rag

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// EmbeddingProvider defines the interface for embedding providers
type EmbeddingProvider interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	GetDimension() int
	GetModel() string
}

// EmbeddingService manages text embeddings
type EmbeddingService interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
	EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
	EmbedDocument(ctx context.Context, doc *Document) error
	GetCacheStats() *CacheStats
	ClearCache() error
}

// EmbeddingCache caches embeddings to avoid redundant API calls
type EmbeddingCache struct {
	cache   map[string]CachedEmbedding
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
	hits    int64
	misses  int64
}

// CachedEmbedding represents a cached embedding
type CachedEmbedding struct {
	Embedding   []float32 `json:"embedding"`
	CreatedAt   time.Time `json:"created_at"`
	AccessedAt  time.Time `json:"accessed_at"`
	AccessCount int       `json:"access_count"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	Size         int     `json:"size"`
	Hits         int64   `json:"hits"`
	Misses       int64   `json:"misses"`
	HitRate      float64 `json:"hit_rate"`
	EvictedCount int64   `json:"evicted_count"`
}

// DefaultEmbeddingService implements EmbeddingService
type DefaultEmbeddingService struct {
	provider EmbeddingProvider
	cache    *EmbeddingCache
	logger   *logrus.Logger

	// Different encoders for different content types
	schemaEncoder ModelEncoder
	queryEncoder  ModelEncoder
	textEncoder   ModelEncoder
}

// ModelEncoder represents a specific encoding model
type ModelEncoder struct {
	Name       string
	Dimension  int
	Provider   EmbeddingProvider
	Preprocess func(string) string
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(provider EmbeddingProvider, logger *logrus.Logger) *DefaultEmbeddingService {
	cache := &EmbeddingCache{
		cache:   make(map[string]CachedEmbedding),
		maxSize: 10000,
		ttl:     24 * time.Hour,
	}

	return &DefaultEmbeddingService{
		provider: provider,
		cache:    cache,
		logger:   logger,
	}
}

// EmbedText embeds a single text
func (es *DefaultEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Check cache first
	cacheKey := es.getCacheKey(text)
	if embedding, found := es.cache.Get(cacheKey); found {
		return embedding, nil
	}

	// Preprocess text
	processedText := es.preprocessText(text)

	// Generate embedding
	embedding, err := es.provider.EmbedText(ctx, processedText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Cache the result
	es.cache.Set(cacheKey, embedding)

	return embedding, nil
}

// EmbedBatch embeds multiple texts
func (es *DefaultEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	uncachedTexts := make([]string, 0)
	uncachedIndices := make([]int, 0)

	// Check cache for each text
	for i, text := range texts {
		cacheKey := es.getCacheKey(text)
		if embedding, found := es.cache.Get(cacheKey); found {
			results[i] = embedding
		} else {
			uncachedTexts = append(uncachedTexts, text)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Batch process uncached texts
	if len(uncachedTexts) > 0 {
		// Preprocess texts
		processedTexts := make([]string, len(uncachedTexts))
		for i, text := range uncachedTexts {
			processedTexts[i] = es.preprocessText(text)
		}

		// Generate embeddings
		embeddings, err := es.provider.EmbedBatch(ctx, processedTexts)
		if err != nil {
			return nil, fmt.Errorf("failed to generate batch embeddings: %w", err)
		}

		// Store results and cache
		for i, embedding := range embeddings {
			idx := uncachedIndices[i]
			results[idx] = embedding
			cacheKey := es.getCacheKey(texts[idx])
			es.cache.Set(cacheKey, embedding)
		}
	}

	return results, nil
}

// EmbedDocument embeds a document's content
func (es *DefaultEmbeddingService) EmbedDocument(ctx context.Context, doc *Document) error {
	// Choose encoder based on document type
	var text string
	switch doc.Type {
	case DocumentTypeSchema:
		text = es.preprocessSchemaContent(doc.Content, doc.Metadata)
	case DocumentTypeQuery:
		text = es.preprocessQueryContent(doc.Content, doc.Metadata)
	default:
		text = doc.Content
	}

	// Generate embedding
	embedding, err := es.EmbedText(ctx, text)
	if err != nil {
		return err
	}

	doc.Embedding = embedding
	return nil
}

// GetCacheStats returns cache statistics
func (es *DefaultEmbeddingService) GetCacheStats() *CacheStats {
	return es.cache.GetStats()
}

// ClearCache clears the embedding cache
func (es *DefaultEmbeddingService) ClearCache() error {
	es.cache.Clear()
	es.logger.Info("Embedding cache cleared")
	return nil
}

// preprocessText preprocesses text before embedding
func (es *DefaultEmbeddingService) preprocessText(text string) string {
	// Normalize whitespace
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")

	// Remove multiple spaces
	for strings.Contains(text, "  ") {
		text = strings.ReplaceAll(text, "  ", " ")
	}

	// Convert to lowercase for consistency
	text = strings.ToLower(text)

	return text
}

// preprocessSchemaContent preprocesses schema content for embedding
func (es *DefaultEmbeddingService) preprocessSchemaContent(content string, metadata map[string]interface{}) string {
	var parts []string

	// Add table name if present
	if tableName, ok := metadata["table_name"].(string); ok {
		parts = append(parts, fmt.Sprintf("table: %s", tableName))
	}

	// Add column information
	if columns, ok := metadata["columns"].([]interface{}); ok {
		for _, col := range columns {
			if colMap, ok := col.(map[string]interface{}); ok {
				if name, ok := colMap["name"].(string); ok {
					if dataType, ok := colMap["type"].(string); ok {
						parts = append(parts, fmt.Sprintf("column %s type %s", name, dataType))
					}
				}
			}
		}
	}

	// Add relationships
	if relationships, ok := metadata["relationships"].([]interface{}); ok {
		for _, rel := range relationships {
			if relMap, ok := rel.(map[string]interface{}); ok {
				if targetTable, ok := relMap["target_table"].(string); ok {
					parts = append(parts, fmt.Sprintf("relates to %s", targetTable))
				}
			}
		}
	}

	// Add the original content
	parts = append(parts, content)

	return es.preprocessText(strings.Join(parts, " "))
}

// preprocessQueryContent preprocesses query content for embedding
func (es *DefaultEmbeddingService) preprocessQueryContent(content string, metadata map[string]interface{}) string {
	var parts []string

	// Add query type if present
	if queryType, ok := metadata["query_type"].(string); ok {
		parts = append(parts, fmt.Sprintf("type: %s", queryType))
	}

	// Add tables involved
	if tables, ok := metadata["tables"].([]string); ok {
		parts = append(parts, fmt.Sprintf("tables: %s", strings.Join(tables, ", ")))
	}

	// Add the query
	parts = append(parts, content)

	return es.preprocessText(strings.Join(parts, " "))
}

// getCacheKey generates a cache key for text
func (es *DefaultEmbeddingService) getCacheKey(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

// Cache methods

// Get retrieves an embedding from cache
func (ec *EmbeddingCache) Get(key string) ([]float32, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if cached, found := ec.cache[key]; found {
		// Check if not expired
		if time.Since(cached.CreatedAt) <= ec.ttl {
			ec.hits++
			cached.AccessedAt = time.Now()
			cached.AccessCount++
			ec.cache[key] = cached
			return cached.Embedding, true
		}
		// Expired, remove it
		delete(ec.cache, key)
	}

	ec.misses++
	return nil, false
}

// Set stores an embedding in cache
func (ec *EmbeddingCache) Set(key string, embedding []float32) {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	// Check cache size and evict if necessary
	if len(ec.cache) >= ec.maxSize {
		ec.evictLRU()
	}

	ec.cache[key] = CachedEmbedding{
		Embedding:   embedding,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
	}
}

// evictLRU evicts the least recently used item
func (ec *EmbeddingCache) evictLRU() {
	var oldestKey string
	var oldestTime time.Time

	for key, cached := range ec.cache {
		if oldestKey == "" || cached.AccessedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = cached.AccessedAt
		}
	}

	if oldestKey != "" {
		delete(ec.cache, oldestKey)
	}
}

// GetStats returns cache statistics
func (ec *EmbeddingCache) GetStats() *CacheStats {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	total := ec.hits + ec.misses
	hitRate := 0.0
	if total > 0 {
		hitRate = float64(ec.hits) / float64(total)
	}

	return &CacheStats{
		Size:    len(ec.cache),
		Hits:    ec.hits,
		Misses:  ec.misses,
		HitRate: hitRate,
	}
}

// Clear clears the cache
func (ec *EmbeddingCache) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	ec.cache = make(map[string]CachedEmbedding)
	ec.hits = 0
	ec.misses = 0
}

// OpenAIEmbeddingProvider implements EmbeddingProvider using OpenAI
type OpenAIEmbeddingProvider struct {
	apiKey    string
	model     string
	dimension int
	logger    *logrus.Logger
}

// NewOpenAIEmbeddingProvider creates an OpenAI embedding provider
func NewOpenAIEmbeddingProvider(apiKey string, model string, logger *logrus.Logger) *OpenAIEmbeddingProvider {
	dimension := 1536 // Default for ada-002
	if strings.Contains(model, "3-small") {
		dimension = 1536
	} else if strings.Contains(model, "3-large") {
		dimension = 3072
	}

	return &OpenAIEmbeddingProvider{
		apiKey:    apiKey,
		model:     model,
		dimension: dimension,
		logger:    logger,
	}
}

// EmbedText generates embedding for text using OpenAI
func (p *OpenAIEmbeddingProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// This would call OpenAI API
	// For now, return mock embedding
	embedding := make([]float32, p.dimension)
	for i := range embedding {
		embedding[i] = float32(i) / float32(p.dimension)
	}
	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (p *OpenAIEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := p.EmbedText(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

// GetDimension returns the embedding dimension
func (p *OpenAIEmbeddingProvider) GetDimension() int {
	return p.dimension
}

// GetModel returns the model name
func (p *OpenAIEmbeddingProvider) GetModel() string {
	return p.model
}

// LocalEmbeddingProvider implements EmbeddingProvider using local models
type LocalEmbeddingProvider struct {
	modelPath string
	dimension int
	logger    *logrus.Logger
	// In production, this would include the actual model
}

// NewLocalEmbeddingProvider creates a local embedding provider
func NewLocalEmbeddingProvider(modelPath string, logger *logrus.Logger) *LocalEmbeddingProvider {
	return &LocalEmbeddingProvider{
		modelPath: modelPath,
		dimension: 384, // Default for sentence-transformers
		logger:    logger,
	}
}

// EmbedText generates embedding for text using local model
func (p *LocalEmbeddingProvider) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// This would use a local model like sentence-transformers
	// For now, return mock embedding
	embedding := make([]float32, p.dimension)
	for i := range embedding {
		embedding[i] = float32(i) / float32(p.dimension)
	}
	return embedding, nil
}

// EmbedBatch generates embeddings for multiple texts
func (p *LocalEmbeddingProvider) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embedding, err := p.EmbedText(ctx, text)
		if err != nil {
			return nil, err
		}
		embeddings[i] = embedding
	}
	return embeddings, nil
}

// GetDimension returns the embedding dimension
func (p *LocalEmbeddingProvider) GetDimension() int {
	return p.dimension
}

// GetModel returns the model name
func (p *LocalEmbeddingProvider) GetModel() string {
	return p.modelPath
}
