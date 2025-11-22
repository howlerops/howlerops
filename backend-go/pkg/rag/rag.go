package rag

import (
	"context"
	"time"

	internalrag "github.com/jbeck018/howlerops/backend-go/internal/rag"
	"github.com/sirupsen/logrus"
)

type (
	EmbeddingService   = internalrag.EmbeddingService
	EmbeddingProvider  = internalrag.EmbeddingProvider
	OllamaModelManager = internalrag.OllamaModelManager
	VectorStore        = internalrag.VectorStore
	DocumentType       = internalrag.DocumentType
	Document           = internalrag.Document
	CacheStats         = internalrag.CacheStats
	SchemaIndexer      = internalrag.SchemaIndexer
	ContextBuilder     = internalrag.ContextBuilder
	QueryContext       = internalrag.QueryContext
	OptimizationHint   = internalrag.OptimizationHint
	GeneratedSQL       = internalrag.GeneratedSQL
	SQLExplanation     = internalrag.SQLExplanation
	ExplanationStep    = internalrag.ExplanationStep
	OptimizedSQL       = internalrag.OptimizedSQL
	Improvement        = internalrag.Improvement
	SmartSQLGenerator  = internalrag.SmartSQLGenerator
	LLMProvider        = internalrag.LLMProvider
)

const (
	DocumentTypeSchema      DocumentType = internalrag.DocumentTypeSchema
	DocumentTypeQuery       DocumentType = internalrag.DocumentTypeQuery
	DocumentTypePlan        DocumentType = internalrag.DocumentTypePlan
	DocumentTypeResult      DocumentType = internalrag.DocumentTypeResult
	DocumentTypeBusiness    DocumentType = internalrag.DocumentTypeBusiness
	DocumentTypePerformance DocumentType = internalrag.DocumentTypePerformance
	DocumentTypeMemory      DocumentType = internalrag.DocumentTypeMemory
)

func NewOpenAIEmbeddingProvider(apiKey, model string, logger *logrus.Logger) EmbeddingProvider {
	return internalrag.NewOpenAIEmbeddingProvider(apiKey, model, logger)
}

func NewOllamaEmbeddingProvider(endpoint, model string, dimension int, logger *logrus.Logger) EmbeddingProvider {
	return internalrag.NewOllamaEmbeddingProvider(endpoint, model, dimension, logger)
}

func NewOllamaModelManager(endpoint string, logger *logrus.Logger) *OllamaModelManager {
	return internalrag.NewOllamaModelManager(endpoint, logger)
}

func NewONNXEmbeddingProvider(modelPath string, logger *logrus.Logger) EmbeddingProvider {
	return internalrag.NewONNXEmbeddingProvider(modelPath, logger)
}

func NewFallbackEmbeddingProvider(primary, fallback EmbeddingProvider) EmbeddingProvider {
	return internalrag.NewFallbackEmbeddingProvider(primary, fallback)
}

func NewEmbeddingService(provider EmbeddingProvider, logger *logrus.Logger) EmbeddingService {
	return internalrag.NewEmbeddingService(provider, logger)
}

func NewAdaptiveVectorStore(tier string, local VectorStore, remote VectorStore, syncEnabled bool) VectorStore {
	return internalrag.NewAdaptiveVectorStore(tier, local, remote, syncEnabled)
}

func StartSyncWorker(ctx context.Context, store VectorStore, interval time.Duration) {
	if s, ok := store.(interface {
		StartSyncWorker(context.Context, time.Duration)
	}); ok {
		s.StartSyncWorker(ctx, interval)
	}
}

func NewSchemaIndexer(store VectorStore, embeddings EmbeddingService, logger *logrus.Logger) *SchemaIndexer {
	return internalrag.NewSchemaIndexer(store, embeddings, logger)
}

func NewContextBuilder(
	vectorStore VectorStore,
	embeddingService EmbeddingService,
	logger *logrus.Logger,
) *ContextBuilder {
	return internalrag.NewContextBuilder(vectorStore, embeddingService, logger)
}

func NewSmartSQLGenerator(
	contextBuilder *ContextBuilder,
	llmProvider LLMProvider,
	logger *logrus.Logger,
) *SmartSQLGenerator {
	return internalrag.NewSmartSQLGenerator(contextBuilder, llmProvider, logger)
}
