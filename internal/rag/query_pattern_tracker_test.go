package rag

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockVectorStore for testing
type MockVectorStore struct {
	mock.Mock
}

func (m *MockVectorStore) Initialize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockVectorStore) IndexDocument(ctx context.Context, doc *Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockVectorStore) BatchIndexDocuments(ctx context.Context, docs []*Document) error {
	args := m.Called(ctx, docs)
	return args.Error(0)
}

func (m *MockVectorStore) GetDocument(ctx context.Context, id string) (*Document, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Document), args.Error(1)
}

func (m *MockVectorStore) UpdateDocument(ctx context.Context, doc *Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockVectorStore) DeleteDocument(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockVectorStore) StoreDocumentWithoutEmbedding(ctx context.Context, doc *Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockVectorStore) GetDocumentsBatch(ctx context.Context, ids []string) ([]*Document, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Document), args.Error(1)
}

func (m *MockVectorStore) UpdateDocumentMetadata(ctx context.Context, id string, metadata map[string]interface{}) error {
	args := m.Called(ctx, id, metadata)
	return args.Error(0)
}

func (m *MockVectorStore) SearchSimilar(ctx context.Context, embedding []float32, k int, filter map[string]interface{}) ([]*Document, error) {
	args := m.Called(ctx, embedding, k, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Document), args.Error(1)
}

func (m *MockVectorStore) SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*Document, error) {
	args := m.Called(ctx, query, k, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Document), args.Error(1)
}

func (m *MockVectorStore) HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error) {
	args := m.Called(ctx, query, embedding, k)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Document), args.Error(1)
}

func (m *MockVectorStore) CreateCollection(ctx context.Context, name string, dimension int) error {
	args := m.Called(ctx, name, dimension)
	return args.Error(0)
}

func (m *MockVectorStore) DeleteCollection(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockVectorStore) ListCollections(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockVectorStore) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*VectorStoreStats), args.Error(1)
}

func (m *MockVectorStore) GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error) {
	args := m.Called(ctx, collection)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*CollectionStats), args.Error(1)
}

func (m *MockVectorStore) Optimize(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockVectorStore) Backup(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func (m *MockVectorStore) Restore(ctx context.Context, path string) error {
	args := m.Called(ctx, path)
	return args.Error(0)
}

func TestQueryPatternTracker_TrackQuery(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	mockStore := new(MockVectorStore)
	tracker := NewQueryPatternTracker(mockStore, logger)

	ctx := context.Background()
	sql := "SELECT id, name FROM users WHERE status = 'active' AND age > 18"
	duration := 0.025
	connectionID := "conn-123"

	// Expect IndexDocument to be called
	mockStore.On("IndexDocument", ctx, mock.MatchedBy(func(doc *Document) bool {
		// Verify document structure
		assert.Equal(t, connectionID, doc.ConnectionID)
		assert.Equal(t, DocumentTypeQuery, doc.Type)
		assert.Contains(t, doc.Content, "Query pattern:")
		assert.Contains(t, doc.Content, "Tables: users")
		assert.Contains(t, doc.Content, "Filters: status, age")

		// Verify metadata
		metadata := doc.Metadata
		assert.NotNil(t, metadata["pattern"])
		assert.NotNil(t, metadata["tables"])
		assert.NotNil(t, metadata["where_columns"])
		assert.Equal(t, duration, metadata["avg_duration"])
		assert.Equal(t, 1, metadata["frequency"])

		return true
	})).Return(nil)

	err := tracker.TrackQuery(ctx, sql, duration, connectionID)
	require.NoError(t, err)

	mockStore.AssertExpectations(t)
}

func TestQueryPatternTracker_NormalizeToPattern(t *testing.T) {
	logger := logrus.New()
	tracker := NewQueryPatternTracker(nil, logger)

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "string literals",
			sql:      "SELECT * FROM users WHERE name = 'John Doe'",
			expected: "SELECT * FROM users WHERE name = '?'",
		},
		{
			name:     "numeric literals",
			sql:      "SELECT * FROM products WHERE price > 100",
			expected: "SELECT * FROM products WHERE price > ?",
		},
		{
			name:     "multiple values",
			sql:      "SELECT * FROM orders WHERE user_id = 123 AND status = 'pending'",
			expected: "SELECT * FROM orders WHERE user_id = ? AND status = '?'",
		},
		{
			name:     "whitespace normalization",
			sql:      "SELECT  *  FROM   users  WHERE   id  =  1",
			expected: "SELECT * FROM users WHERE id = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := tracker.normalizeToPattern(tt.sql)
			assert.Equal(t, tt.expected, pattern)
		})
	}
}

func TestQueryPatternTracker_ExtractTables(t *testing.T) {
	logger := logrus.New()
	tracker := NewQueryPatternTracker(nil, logger)

	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "single table",
			sql:      "SELECT * FROM users",
			expected: []string{"users"},
		},
		{
			name:     "join query",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expected: []string{"users", "orders"},
		},
		{
			name:     "multiple joins",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id JOIN products ON orders.product_id = products.id",
			expected: []string{"users", "orders", "products"},
		},
		{
			name:     "duplicate tables",
			sql:      "SELECT * FROM users u1 JOIN users u2 ON u1.manager_id = u2.id",
			expected: []string{"users"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tables := tracker.extractTables(tt.sql)
			assert.ElementsMatch(t, tt.expected, tables)
		})
	}
}

func TestQueryPatternTracker_ExtractWhereColumns(t *testing.T) {
	logger := logrus.New()
	tracker := NewQueryPatternTracker(nil, logger)

	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "simple where",
			sql:      "SELECT * FROM users WHERE status = 'active'",
			expected: []string{"status"},
		},
		{
			name:     "multiple conditions",
			sql:      "SELECT * FROM users WHERE status = 'active' AND age > 18",
			expected: []string{"status", "age"},
		},
		{
			name:     "no where clause",
			sql:      "SELECT * FROM users",
			expected: []string{},
		},
		{
			name:     "where with group by",
			sql:      "SELECT status, COUNT(*) FROM users WHERE active = true GROUP BY status",
			expected: []string{"active"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := tracker.extractWhereColumns(tt.sql)
			assert.ElementsMatch(t, tt.expected, columns)
		})
	}
}

func TestQueryPatternTracker_ExtractJoinColumns(t *testing.T) {
	logger := logrus.New()
	tracker := NewQueryPatternTracker(nil, logger)

	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "simple join",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expected: []string{"users.id", "orders.user_id"},
		},
		{
			name:     "multiple joins",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id JOIN products ON orders.product_id = products.id",
			expected: []string{"users.id", "orders.user_id", "orders.product_id", "products.id"},
		},
		{
			name:     "no joins",
			sql:      "SELECT * FROM users",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			columns := tracker.extractJoinColumns(tt.sql)
			assert.ElementsMatch(t, tt.expected, columns)
		})
	}
}

func TestQueryPatternTracker_DescribePattern(t *testing.T) {
	logger := logrus.New()
	tracker := NewQueryPatternTracker(nil, logger)

	tests := []struct {
		name     string
		sql      string
		expected string
	}{
		{
			name:     "insert",
			sql:      "INSERT INTO users (name) VALUES ('John')",
			expected: "Insert data",
		},
		{
			name:     "update",
			sql:      "UPDATE users SET status = 'active'",
			expected: "Update records",
		},
		{
			name:     "delete",
			sql:      "DELETE FROM users WHERE id = 1",
			expected: "Delete records",
		},
		{
			name:     "simple select",
			sql:      "SELECT * FROM users",
			expected: "Simple select",
		},
		{
			name:     "select with join",
			sql:      "SELECT * FROM users JOIN orders ON users.id = orders.user_id",
			expected: "Query with joins",
		},
		{
			name:     "aggregate query",
			sql:      "SELECT status, COUNT(*) FROM users GROUP BY status",
			expected: "Aggregate query",
		},
		{
			name:     "filtered query",
			sql:      "SELECT * FROM users WHERE status = 'active'",
			expected: "Filtered query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			description := tracker.describePattern(tt.sql)
			assert.Equal(t, tt.expected, description)
		})
	}
}

func TestQueryPatternTracker_Unique(t *testing.T) {
	logger := logrus.New()
	tracker := NewQueryPatternTracker(nil, logger)

	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "empty",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tracker.unique(tt.input)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}
