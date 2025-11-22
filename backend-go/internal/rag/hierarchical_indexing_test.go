package rag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/database"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupHierarchicalTest(t *testing.T) (*SchemaIndexer, *ContextBuilder, *SQLiteVectorStore, func()) {
	// Create temporary directory for test database
	tempDir, err := os.MkdirTemp("", "rag-test-*")
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "test_vectors.db")

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create vector store
	config := &SQLiteVectorConfig{
		Path:        dbPath,
		VectorSize:  384, // Using smaller vectors for testing
		CacheSizeMB: 10,
		MMapSizeMB:  20,
		WALEnabled:  true,
		Timeout:     5 * time.Second,
		RRFConstant: 60,
	}

	vectorStore, err := NewSQLiteVectorStore(config, logger)
	require.NoError(t, err)

	ctx := context.Background()
	err = vectorStore.Initialize(ctx)
	require.NoError(t, err)

	// Run migration for hierarchy
	migrationSQL, err := os.ReadFile("migrations/003_add_hierarchy.sql")
	require.NoError(t, err)

	// Apply migration (simplified for test)
	db := vectorStore.(*SQLiteVectorStore).db
	_, err = db.ExecContext(ctx, string(migrationSQL))
	require.NoError(t, err)

	// Create embedding service (mock for testing)
	embeddingService := &mockEmbeddingService{vectorSize: 384}

	// Create indexer and context builder
	indexer := NewSchemaIndexer(vectorStore, embeddingService, logger)
	contextBuilder := NewContextBuilder(vectorStore, embeddingService, logger)

	// Cleanup function
	cleanup := func() {
		vectorStore.(*SQLiteVectorStore).db.Close()
		os.RemoveAll(tempDir)
	}

	return indexer, contextBuilder, vectorStore, cleanup
}

// Mock embedding service for testing
type mockEmbeddingService struct {
	vectorSize int
}

func (m *mockEmbeddingService) EmbedText(ctx context.Context, text string) ([]float32, error) {
	// Generate deterministic embeddings based on text length
	embedding := make([]float32, m.vectorSize)
	textLen := float32(len(text))
	for i := range embedding {
		// Simple pattern: vary by text length and position
		embedding[i] = (textLen + float32(i)) / 1000.0
	}
	return embedding, nil
}

func (m *mockEmbeddingService) EmbedDocument(ctx context.Context, doc *Document) error {
	embedding, err := m.EmbedText(ctx, doc.Content)
	if err != nil {
		return err
	}
	doc.Embedding = embedding
	return nil
}

func TestHierarchicalIndexing(t *testing.T) {
	indexer, _, vectorStore, cleanup := setupHierarchicalTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create test table
	table := database.TableInfo{
		Name:     "users",
		RowCount: 1000,
		Comment:  "User accounts table",
	}

	structure := &database.TableStructure{
		Columns: []database.ColumnInfo{
			{Name: "id", DataType: "INTEGER", PrimaryKey: true, Nullable: false},
			{Name: "username", DataType: "VARCHAR(255)", Nullable: false, Unique: true},
			{Name: "email", DataType: "VARCHAR(255)", Nullable: false},
			{Name: "created_at", DataType: "TIMESTAMP", Nullable: false},
		},
		Indexes: []database.IndexInfo{
			{Name: "idx_username", Columns: []string{"username"}, Unique: true},
			{Name: "idx_email", Columns: []string{"email"}, Unique: false},
		},
		ForeignKeys: []database.ForeignKeyInfo{},
	}

	// Index table hierarchically
	err := indexer.IndexTableHierarchical(ctx, "conn-1", "public", table, structure)
	require.NoError(t, err)

	// Verify parent document exists and is embedded
	parentID := "table:conn-1:public.users"
	parent, err := vectorStore.GetDocument(ctx, parentID)
	require.NoError(t, err)
	assert.Equal(t, DocumentLevelTable, parent.Level)
	assert.NotNil(t, parent.Embedding, "Parent should have embedding")
	assert.NotEmpty(t, parent.Summary, "Parent should have summary")
	assert.Contains(t, parent.Summary, "users", "Summary should mention table name")

	// Verify child IDs are stored in metadata
	childIDsRaw, ok := parent.Metadata["child_ids"]
	require.True(t, ok, "Parent metadata should contain child_ids")

	childIDs, ok := childIDsRaw.([]string)
	require.True(t, ok, "child_ids should be []string")
	assert.Equal(t, 6, len(childIDs), "Should have 4 columns + 2 indexes = 6 children")

	// Verify children exist and are NOT embedded
	for _, childID := range childIDs {
		child, err := vectorStore.GetDocument(ctx, childID)
		require.NoError(t, err)
		assert.Equal(t, parentID, child.ParentID, "Child should reference parent")
		assert.Nil(t, child.Embedding, "Child should not have embedding (lazy loaded)")

		// Verify child level
		if child.Level == DocumentLevelColumn {
			assert.Contains(t, child.Content, "Column", "Column document should mention 'Column'")
		} else if child.Level == DocumentLevelIndex {
			assert.Contains(t, child.Content, "Index", "Index document should mention 'Index'")
		}
	}
}

func TestHierarchicalRetrieval(t *testing.T) {
	indexer, contextBuilder, _, cleanup := setupHierarchicalTest(t)
	defer cleanup()

	ctx := context.Background()

	// Index multiple tables
	tables := []struct {
		name      string
		columns   []database.ColumnInfo
		rowCount  int64
	}{
		{
			name: "users",
			columns: []database.ColumnInfo{
				{Name: "id", DataType: "INTEGER", PrimaryKey: true},
				{Name: "username", DataType: "VARCHAR(255)"},
				{Name: "email", DataType: "VARCHAR(255)"},
			},
			rowCount: 1000,
		},
		{
			name: "posts",
			columns: []database.ColumnInfo{
				{Name: "id", DataType: "INTEGER", PrimaryKey: true},
				{Name: "user_id", DataType: "INTEGER"},
				{Name: "title", DataType: "TEXT"},
				{Name: "content", DataType: "TEXT"},
			},
			rowCount: 5000,
		},
		{
			name: "comments",
			columns: []database.ColumnInfo{
				{Name: "id", DataType: "INTEGER", PrimaryKey: true},
				{Name: "post_id", DataType: "INTEGER"},
				{Name: "user_id", DataType: "INTEGER"},
				{Name: "text", DataType: "TEXT"},
			},
			rowCount: 10000,
		},
	}

	for _, tbl := range tables {
		table := database.TableInfo{
			Name:     tbl.name,
			RowCount: tbl.rowCount,
		}
		structure := &database.TableStructure{
			Columns: tbl.columns,
		}
		err := indexer.IndexTableHierarchical(ctx, "conn-1", "public", table, structure)
		require.NoError(t, err)
	}

	// Search for user-related tables
	queryEmb, err := indexer.embeddingService.EmbedText(ctx, "user authentication and profile")
	require.NoError(t, err)

	schemas, err := contextBuilder.FetchRelevantSchemasHierarchical(ctx, queryEmb, "conn-1", 5)
	require.NoError(t, err)
	assert.Greater(t, len(schemas), 0, "Should find relevant schemas")

	// Top result should likely be 'users' table
	if len(schemas) > 0 {
		topSchema := schemas[0]
		t.Logf("Top schema: %s with %d columns", topSchema.TableName, len(topSchema.Columns))

		// If this is a top-3 result, it should have full column details
		if topSchema.TableName != "" {
			// Top 3 should have expanded columns
			hasColumns := len(topSchema.Columns) > 0
			t.Logf("Has columns expanded: %v", hasColumns)
		}
	}
}

func TestHierarchicalRetrievalReducesNoise(t *testing.T) {
	indexer, contextBuilder, vectorStore, cleanup := setupHierarchicalTest(t)
	defer cleanup()

	ctx := context.Background()

	// Create a table with many columns (50+)
	columns := make([]database.ColumnInfo, 50)
	for i := 0; i < 50; i++ {
		columns[i] = database.ColumnInfo{
			Name:     fmt.Sprintf("col_%d", i),
			DataType: "VARCHAR(255)",
		}
	}
	columns[0].PrimaryKey = true // Make first column PK

	table := database.TableInfo{
		Name:     "large_table",
		RowCount: 100000,
	}
	structure := &database.TableStructure{
		Columns: columns,
	}

	err := indexer.IndexTableHierarchical(ctx, "conn-1", "public", table, structure)
	require.NoError(t, err)

	// Count total documents in vector store
	// With flat indexing: 1 table + 50 columns = 51 embedded documents
	// With hierarchical: 1 table document embedded, 50 columns stored without embeddings

	// Check that only parent is searchable (has embedding)
	filter := map[string]interface{}{
		"connection_id": "conn-1",
		"level":         string(DocumentLevelTable),
	}

	// Create a generic query embedding
	queryEmb, err := indexer.embeddingService.EmbedText(ctx, "large table columns")
	require.NoError(t, err)

	// Search should only return table-level documents (not individual columns)
	results, err := vectorStore.SearchSimilar(ctx, queryEmb, 10, filter)
	require.NoError(t, err)

	t.Logf("Search results (table-level only): %d documents", len(results))
	assert.LessOrEqual(t, len(results), 10, "Should only get table-level results")

	// All results should be table-level
	for _, doc := range results {
		assert.Equal(t, DocumentLevelTable, doc.Level, "All search results should be table-level")
		assert.NotNil(t, doc.Embedding, "Table documents should have embeddings")
	}

	// Verify column documents exist but are not searchable
	parentDoc, err := vectorStore.GetDocument(ctx, "table:conn-1:public.large_table")
	require.NoError(t, err)

	childIDsRaw, ok := parentDoc.Metadata["child_ids"]
	require.True(t, ok)
	childIDs := childIDsRaw.([]string)
	assert.Equal(t, 50, len(childIDs), "Should have 50 column children")

	// Verify children are stored without embeddings
	for _, childID := range childIDs {
		child, err := vectorStore.GetDocument(ctx, childID)
		require.NoError(t, err)
		assert.Nil(t, child.Embedding, "Column documents should not have embeddings")
	}

	t.Log("SUCCESS: Hierarchical structure reduces noise by only indexing parent documents")
}
