package rag

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockEmbeddingService for testing
type MockEmbeddingService struct{}

func (m *MockEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Return dummy embedding
	return make([]float32, 384), nil
}

func (m *MockEmbeddingService) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	// Return dummy embeddings for batch
	embeddings := make([][]float32, len(texts))
	for i := range embeddings {
		embeddings[i] = make([]float32, 384)
	}
	return embeddings, nil
}

func (m *MockEmbeddingService) EmbedDocument(ctx context.Context, doc *Document) error {
	doc.Embedding = make([]float32, 384)
	return nil
}

func (m *MockEmbeddingService) ClearCache() error {
	// No-op for testing
	return nil
}

func (m *MockEmbeddingService) GetCacheStats() *CacheStats {
	return &CacheStats{
		Size:    0,
		Hits:    0,
		Misses:  0,
		HitRate: 0.0,
	}
}

func TestSchemaIndexer_WithEnrichment_Integration(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create mock vector store
	mockStore := new(MockVectorStore)
	mockEmbedding := &MockEmbeddingService{}

	// Create indexer with enricher
	enricher := NewSchemaEnricher(db, logger)
	indexer := NewSchemaIndexer(mockStore, mockEmbedding, logger).WithEnricher(enricher)

	// Test data
	connID := "test-conn"
	schemaName := "public"
	table := database.TableInfo{
		Name:     "users",
		Type:     "TABLE",
		Comment:  "User accounts",
		RowCount: 1000,
	}

	structure := &database.TableStructure{
		Columns: []database.ColumnInfo{
			{
				Name:       "id",
				DataType:   "integer",
				Nullable:   false,
				PrimaryKey: true,
			},
			{
				Name:     "status",
				DataType: "varchar",
				Nullable: false,
			},
			{
				Name:     "age",
				DataType: "integer",
				Nullable: true,
			},
		},
	}

	// Mock enrichment queries for "id" column (numeric, but also PK so not enriched much)
	sqlMock.ExpectQuery("SELECT COUNT\\(DISTINCT id\\) FROM public.users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1000))
	sqlMock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.users WHERE id IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	sqlMock.ExpectQuery("SELECT MIN\\(id\\), MAX\\(id\\), AVG\\(id\\)").
		WillReturnRows(sqlmock.NewRows([]string{"min", "max", "avg"}).AddRow(1, 1000, 500.5))

	// Mock enrichment queries for "status" column (categorical)
	sqlMock.ExpectQuery("SELECT COUNT\\(DISTINCT status\\) FROM public.users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))
	sqlMock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.users WHERE status IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	sqlMock.ExpectQuery("SELECT status, COUNT\\(\\*\\) as cnt").
		WillReturnRows(
			sqlmock.NewRows([]string{"status", "cnt"}).
				AddRow("active", 700).
				AddRow("inactive", 200).
				AddRow("pending", 100),
		)

	// Mock enrichment queries for "age" column (numeric)
	sqlMock.ExpectQuery("SELECT COUNT\\(DISTINCT age\\) FROM public.users").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(80))
	sqlMock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM public.users WHERE age IS NULL").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
	sqlMock.ExpectQuery("SELECT MIN\\(age\\), MAX\\(age\\), AVG\\(age\\)").
		WillReturnRows(sqlmock.NewRows([]string{"min", "max", "avg"}).AddRow(18, 95, 42.5))

	// Expect table document
	mockStore.On("IndexDocument", mock.Anything, mock.MatchedBy(func(doc *Document) bool {
		return doc.Type == DocumentTypeSchema &&
			doc.Metadata["subtype"] == "table" &&
			doc.Metadata["table_name"] == "users"
	})).Return(nil)

	// Expect enriched column documents
	var capturedDocs []*Document
	mockStore.On("IndexDocument", mock.Anything, mock.MatchedBy(func(doc *Document) bool {
		if doc.Type == DocumentTypeSchema && doc.Metadata["subtype"] == "column" {
			capturedDocs = append(capturedDocs, doc)
			return true
		}
		return false
	})).Return(nil)

	ctx := context.Background()
	err = indexer.IndexTableDetails(ctx, connID, schemaName, table, structure)
	require.NoError(t, err)

	// Verify all SQL expectations were met
	require.NoError(t, sqlMock.ExpectationsWereMet())

	// Verify we captured enriched column documents
	require.Len(t, capturedDocs, 3, "Should have 3 column documents")

	// Find and verify the "status" column document (categorical)
	var statusDoc *Document
	for _, doc := range capturedDocs {
		if doc.Metadata["column"] == "status" {
			statusDoc = doc
			break
		}
	}
	require.NotNil(t, statusDoc, "Should have status column document")

	// Verify enrichment data in content
	assert.Contains(t, statusDoc.Content, "examples:")
	assert.Contains(t, statusDoc.Content, "active")
	assert.Contains(t, statusDoc.Content, "distinct_values: 3")

	// Verify enrichment data in metadata
	assert.Equal(t, int64(3), statusDoc.Metadata["distinct_count"])
	assert.Equal(t, int64(0), statusDoc.Metadata["null_count"])
	assert.NotNil(t, statusDoc.Metadata["sample_values"])
	assert.NotNil(t, statusDoc.Metadata["top_values"])

	// Find and verify the "age" column document (numeric)
	var ageDoc *Document
	for _, doc := range capturedDocs {
		if doc.Metadata["column"] == "age" {
			ageDoc = doc
			break
		}
	}
	require.NotNil(t, ageDoc, "Should have age column document")

	// Verify numeric enrichment
	assert.Contains(t, ageDoc.Content, "range: 18 to 95")
	assert.Contains(t, ageDoc.Content, "distinct_values: 80")
	assert.Equal(t, int64(18), ageDoc.Metadata["min_value"])
	assert.Equal(t, int64(95), ageDoc.Metadata["max_value"])
	assert.Equal(t, 42.5, ageDoc.Metadata["avg_value"])

	mockStore.AssertExpectations(t)
}

func TestSchemaIndexer_WithoutEnrichment(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create mock vector store
	mockStore := new(MockVectorStore)
	mockEmbedding := &MockEmbeddingService{}

	// Create indexer WITHOUT enricher
	indexer := NewSchemaIndexer(mockStore, mockEmbedding, logger)

	// Test data
	connID := "test-conn"
	schemaName := "public"
	table := database.TableInfo{
		Name:     "users",
		Type:     "TABLE",
		RowCount: 1000,
	}

	structure := &database.TableStructure{
		Columns: []database.ColumnInfo{
			{
				Name:     "status",
				DataType: "varchar",
				Nullable: false,
			},
		},
	}

	// Expect table document
	mockStore.On("IndexDocument", mock.Anything, mock.MatchedBy(func(doc *Document) bool {
		return doc.Type == DocumentTypeSchema &&
			doc.Metadata["subtype"] == "table"
	})).Return(nil)

	// Expect basic column document (without enrichment)
	var capturedDoc *Document
	mockStore.On("IndexDocument", mock.Anything, mock.MatchedBy(func(doc *Document) bool {
		if doc.Type == DocumentTypeSchema && doc.Metadata["subtype"] == "column" {
			capturedDoc = doc
			return true
		}
		return false
	})).Return(nil)

	ctx := context.Background()
	err := indexer.IndexTableDetails(ctx, connID, schemaName, table, structure)
	require.NoError(t, err)

	// Verify column document has no enrichment
	require.NotNil(t, capturedDoc)
	assert.NotContains(t, capturedDoc.Content, "examples:")
	assert.NotContains(t, capturedDoc.Content, "range:")
	assert.Nil(t, capturedDoc.Metadata["distinct_count"])
	assert.Nil(t, capturedDoc.Metadata["sample_values"])

	mockStore.AssertExpectations(t)
}
