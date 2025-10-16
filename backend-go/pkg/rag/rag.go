package rag

import (
	"github.com/sirupsen/logrus"
	internalrag "github.com/sql-studio/backend-go/internal/rag"
)

type (
	EmbeddingService  = internalrag.EmbeddingService
	EmbeddingProvider = internalrag.EmbeddingProvider
	DocumentType      = internalrag.DocumentType
	Document          = internalrag.Document
	CacheStats        = internalrag.CacheStats
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

func NewEmbeddingService(provider EmbeddingProvider, logger *logrus.Logger) EmbeddingService {
	return internalrag.NewEmbeddingService(provider, logger)
}
