package export

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/pkg/storage"
)

// Mock connection store
type mockConnectionStore struct {
	connections map[string]*storage.Connection
	byName      map[string]*storage.Connection
}

func newMockConnectionStore() *mockConnectionStore {
	return &mockConnectionStore{
		connections: make(map[string]*storage.Connection),
		byName:      make(map[string]*storage.Connection),
	}
}

func (m *mockConnectionStore) GetByName(ctx context.Context, userID, name string) (*storage.Connection, error) {
	key := fmt.Sprintf("%s:%s", userID, name)
	return m.byName[key], nil
}

func (m *mockConnectionStore) Create(ctx context.Context, conn *storage.Connection) error {
	m.connections[conn.ID] = conn
	key := fmt.Sprintf("%s:%s", conn.CreatedBy, conn.Name)
	m.byName[key] = conn
	return nil
}

func (m *mockConnectionStore) Update(ctx context.Context, conn *storage.Connection) error {
	m.connections[conn.ID] = conn
	return nil
}

// Mock query store
type mockQueryStore struct {
	queries map[string]*storage.SavedQuery
	byName  map[string]*storage.SavedQuery
}

func newMockQueryStore() *mockQueryStore {
	return &mockQueryStore{
		queries: make(map[string]*storage.SavedQuery),
		byName:  make(map[string]*storage.SavedQuery),
	}
}

func (m *mockQueryStore) GetByName(ctx context.Context, userID, name string) (*storage.SavedQuery, error) {
	key := fmt.Sprintf("%s:%s", userID, name)
	return m.byName[key], nil
}

func (m *mockQueryStore) Create(ctx context.Context, query *storage.SavedQuery) error {
	m.queries[query.ID] = query
	key := fmt.Sprintf("%s:%s", query.CreatedBy, query.Name)
	m.byName[key] = query
	return nil
}

func (m *mockQueryStore) Update(ctx context.Context, query *storage.SavedQuery) error {
	m.queries[query.ID] = query
	return nil
}

func TestConfigImporter_Import(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	importer := NewConfigImporter(logger)

	config := &ExportedConfig{
		Format:     ConfigExportFormat,
		ExportedAt: time.Now().UTC(),
		Connections: []ExportedConnection{
			{
				ExportID: "conn_prod",
				Name:     "Production DB",
				Type:     "postgres",
				Host:     "prod.example.com",
				Port:     5432,
				Database: "myapp",
				Username: "admin",
			},
			{
				ExportID: "conn_staging",
				Name:     "Staging DB",
				Type:     "mysql",
				Host:     "staging.example.com",
				Port:     3306,
				Database: "myapp_staging",
				Username: "stageuser",
			},
		},
		SavedQueries: []ExportedSavedQuery{
			{
				ExportID:           "query_users",
				Name:               "Get Active Users",
				Query:              "SELECT * FROM users WHERE active = true",
				ConnectionExportID: "conn_prod",
				Tags:               []string{"users", "reporting"},
				Folder:             "analytics",
			},
		},
		Tags:    []string{"users", "reporting"},
		Folders: []string{"analytics"},
	}

	t.Run("basic import", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 2, result.ConnectionsImported)
		assert.Equal(t, 1, result.QueriesImported)
		assert.Len(t, result.ConnectionsNeedingPasswords, 2)

		// Verify connections were created
		assert.Len(t, connStore.connections, 2)

		// Verify ID mapping
		assert.NotEmpty(t, result.ConnectionIDMap["conn_prod"])
		assert.NotEmpty(t, result.ConnectionIDMap["conn_staging"])
	})

	t.Run("dry run does not persist", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.DryRun = true
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.True(t, result.DryRun)
		assert.Equal(t, 2, result.ConnectionsImported)

		// Store should be empty since it was dry run
		assert.Len(t, connStore.connections, 0)
	})

	t.Run("skip existing connections", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		// Pre-create an existing connection
		existingConn := &storage.Connection{
			ID:        "existing-conn-id",
			Name:      "Production DB", // Same name as in config
			Type:      "postgres",
			CreatedBy: "user-1",
		}
		connStore.connections[existingConn.ID] = existingConn
		connStore.byName["user-1:Production DB"] = existingConn

		options := DefaultConfigImportOptions()
		options.ConflictStrategy = ConflictSkip
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ConnectionsImported)
		assert.Equal(t, 1, result.ConnectionsSkipped)

		// Existing connection should be mapped
		assert.Equal(t, "existing-conn-id", result.ConnectionIDMap["conn_prod"])
	})

	t.Run("rename duplicate connections", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		// Pre-create an existing connection
		existingConn := &storage.Connection{
			ID:        "existing-conn-id",
			Name:      "Production DB",
			CreatedBy: "user-1",
		}
		connStore.connections[existingConn.ID] = existingConn
		connStore.byName["user-1:Production DB"] = existingConn

		options := DefaultConfigImportOptions()
		options.ConflictStrategy = ConflictRename
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 2, result.ConnectionsImported)

		// Check that the renamed connection has "(imported" in the name
		var foundRenamed bool
		for _, conn := range connStore.connections {
			if conn.ID != "existing-conn-id" && conn.Type == "postgres" {
				assert.Contains(t, conn.Name, "(imported")
				foundRenamed = true
			}
		}
		assert.True(t, foundRenamed)
	})

	t.Run("host overrides", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.HostOverrides = map[string]string{
			"conn_prod": "new-host.example.com",
		}
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)

		// Find the production connection and verify host was overridden
		prodConnID := result.ConnectionIDMap["conn_prod"]
		prodConn := connStore.connections[prodConnID]
		assert.Equal(t, "new-host.example.com", prodConn.Host)

		// Staging should still have original host
		stagingConnID := result.ConnectionIDMap["conn_staging"]
		stagingConn := connStore.connections[stagingConnID]
		assert.Equal(t, "staging.example.com", stagingConn.Host)
	})

	t.Run("import only connections", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.ImportConnections = true
		options.ImportSavedQueries = false
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 2, result.ConnectionsImported)
		assert.Equal(t, 0, result.QueriesImported)
		assert.Len(t, queryStore.queries, 0)
	})

	t.Run("import only queries", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.ImportConnections = false
		options.ImportSavedQueries = true
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 0, result.ConnectionsImported)
		assert.Equal(t, 1, result.QueriesImported)

		// Query should have warning about missing connection reference
		assert.NotEmpty(t, result.Warnings)
	})

	t.Run("filter by connection export IDs", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.ConnectionExportIDs = []string{"conn_prod"}
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 1, result.ConnectionsImported)
		assert.Equal(t, 1, result.ConnectionsSkipped)
	})

	t.Run("filter queries by tags", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.QueryTags = []string{"nonexistent"}
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 0, result.QueriesImported)
		assert.Equal(t, 1, result.QueriesSkipped)
	})

	t.Run("share with organization", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		options.ShareWithOrganization = "org-123"
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 2, result.ConnectionsImported)

		// Check that all imported items are shared
		for _, conn := range connStore.connections {
			assert.True(t, conn.IsShared)
			assert.Equal(t, "org-123", conn.TeamID)
		}
		for _, query := range queryStore.queries {
			assert.True(t, query.IsShared)
			assert.Equal(t, "org-123", query.TeamID)
		}
	})

	t.Run("placeholder host warning", func(t *testing.T) {
		configWithPlaceholder := &ExportedConfig{
			Format: ConfigExportFormat,
			Connections: []ExportedConnection{
				{
					ExportID: "conn_1",
					Name:     "Test DB",
					Type:     "postgres",
					Host:     "{{Test DB_HOST}}",
					Port:     5432,
				},
			},
		}

		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		options := DefaultConfigImportOptions()
		result, err := importer.Import(context.Background(), configWithPlaceholder, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.NotEmpty(t, result.Warnings)
		assert.Contains(t, result.Warnings[0], "placeholder")
	})
}

func TestMergeConflictStrategy(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	importer := NewConfigImporter(logger)

	t.Run("merge query tags", func(t *testing.T) {
		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		// Pre-create a query with some tags
		existingQuery := &storage.SavedQuery{
			ID:        "existing-query-id",
			Name:      "Existing Query",
			Query:     "SELECT 1",
			Tags:      []string{"existing-tag", "shared-tag"},
			Metadata:  map[string]string{"existing-key": "existing-value"},
			CreatedBy: "user-1",
		}
		queryStore.queries[existingQuery.ID] = existingQuery
		queryStore.byName["user-1:Existing Query"] = existingQuery

		config := &ExportedConfig{
			Format: ConfigExportFormat,
			SavedQueries: []ExportedSavedQuery{
				{
					ExportID: "query_1",
					Name:     "Existing Query", // Same name
					Query:    "SELECT 2",
					Tags:     []string{"new-tag", "shared-tag"}, // Partial overlap
					Metadata: map[string]string{
						"new-key":      "new-value",
						"existing-key": "should-not-override", // Should not override
					},
				},
			},
		}

		options := DefaultConfigImportOptions()
		options.ImportConnections = false
		options.ConflictStrategy = ConflictMerge
		result, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, options)

		require.NoError(t, err)
		assert.Equal(t, 1, result.QueriesImported)

		// Check merged tags (should have all unique tags)
		merged := queryStore.queries["existing-query-id"]
		assert.Contains(t, merged.Tags, "existing-tag")
		assert.Contains(t, merged.Tags, "shared-tag")
		assert.Contains(t, merged.Tags, "new-tag")

		// Check metadata merge (existing values should not be overwritten)
		assert.Equal(t, "existing-value", merged.Metadata["existing-key"])
		assert.Equal(t, "new-value", merged.Metadata["new-key"])
	})
}

func TestImportInvalidConfig(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	importer := NewConfigImporter(logger)

	t.Run("invalid format version", func(t *testing.T) {
		config := &ExportedConfig{
			Format: "invalid-format",
		}

		connStore := newMockConnectionStore()
		queryStore := newMockQueryStore()

		_, err := importer.Import(context.Background(), config, "user-1", connStore, queryStore, DefaultConfigImportOptions())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported config format")
	})
}
