package export

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jbeck018/howlerops/pkg/storage"
)

func TestConfigExporter_Export(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	exporter := NewConfigExporter(logger, "1.0.0-test")

	connections := []storage.Connection{
		{
			ID:          "conn-123456789",
			Name:        "Production DB",
			Type:        "postgres",
			Host:        "prod.example.com",
			Port:        5432,
			Database:    "myapp",
			Username:    "admin",
			Password:    "secret123", // Should NOT appear in export
			Environment: "production",
			Metadata:    map[string]string{"team": "backend"},
			IsShared:    false,
			CreatedBy:   "user-1",
			CreatedAt:   time.Now().Add(-24 * time.Hour),
			UpdatedAt:   time.Now(),
		},
		{
			ID:        "conn-987654321",
			Name:      "Staging DB",
			Type:      "mysql",
			Host:      "staging.example.com",
			Port:      3306,
			Database:  "myapp_staging",
			Username:  "stageuser",
			IsShared:  true,
			TeamID:    "org-1",
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
	}

	queries := []storage.SavedQuery{
		{
			ID:           "query-111111111",
			Name:         "Get Active Users",
			Description:  "Retrieves all active users",
			Query:        "SELECT * FROM users WHERE active = true",
			ConnectionID: "conn-123456789",
			Folder:       "analytics",
			Tags:         []string{"users", "analytics", "reporting"},
			Metadata:     map[string]string{"author": "alice"},
			IsShared:     false,
			CreatedAt:    time.Now().Add(-12 * time.Hour),
			UpdatedAt:    time.Now(),
		},
	}

	history := []storage.QueryHistory{
		{
			ID:           "hist-1",
			Query:        "SELECT count(*) FROM users",
			ConnectionID: "conn-123456789",
			ExecutedAt:   time.Now().Add(-1 * time.Hour),
			Duration:     150,
			RowsReturned: 1,
			Success:      true,
		},
	}

	t.Run("basic export with defaults", func(t *testing.T) {
		options := DefaultConfigExportOptions()
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Equal(t, ConfigExportFormat, config.Format)
		assert.Equal(t, "test@example.com", config.ExportedBy)
		assert.Equal(t, "1.0.0-test", config.AppVersion)

		// Should only include personal connections (IncludeShared is false by default)
		assert.Len(t, config.Connections, 1)
		assert.Equal(t, "Production DB", config.Connections[0].Name)

		// Verify password is NOT in export
		jsonData, _ := config.ToJSON(false)
		assert.NotContains(t, string(jsonData), "secret123")

		// Should include queries
		assert.Len(t, config.SavedQueries, 1)
		assert.Equal(t, "Get Active Users", config.SavedQueries[0].Name)
		assert.Contains(t, config.SavedQueries[0].Query, "SELECT")

		// Should include tags
		assert.Contains(t, config.Tags, "users")
		assert.Contains(t, config.Tags, "analytics")

		// Should include folders
		assert.Contains(t, config.Folders, "analytics")
	})

	t.Run("export with shared resources", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections:  true,
			IncludeSavedQueries: true,
			IncludeShared:       true,
		}
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.Connections, 2)
	})

	t.Run("export with anonymized hosts", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections: true,
			AnonymizeHosts:     true,
		}
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.Connections, 1)
		assert.Contains(t, config.Connections[0].Host, "{{")
		assert.Contains(t, config.Connections[0].Host, "}}")
		assert.NotContains(t, config.Connections[0].Host, "prod.example.com")
	})

	t.Run("export metadata only", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections:  true,
			IncludeSavedQueries: true,
			MetadataOnly:        true,
		}
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.SavedQueries, 1)
		assert.Empty(t, config.SavedQueries[0].Query) // Query text should be empty
		assert.Equal(t, "Get Active Users", config.SavedQueries[0].Name)
	})

	t.Run("export with query history", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections:  true,
			IncludeSavedQueries: true,
			IncludeQueryHistory: true,
		}
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.QueryHistory, 1)
		assert.True(t, config.QueryHistory[0].Success)
	})

	t.Run("filter by connection IDs", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeConnections: true,
			ConnectionIDs:      []string{"conn-999999999"}, // Non-existent
		}
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.Connections, 0)
	})

	t.Run("filter by query tags", func(t *testing.T) {
		options := ConfigExportOptions{
			IncludeSavedQueries: true,
			QueryTags:           []string{"nonexistent"},
		}
		config, err := exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.SavedQueries, 0)

		// Now with matching tag
		options.QueryTags = []string{"users"}
		config, err = exporter.Export(context.Background(), "test@example.com", connections, queries, history, options)

		require.NoError(t, err)
		assert.Len(t, config.SavedQueries, 1)
	})
}

func TestExportedConfig_ToJSON(t *testing.T) {
	config := &ExportedConfig{
		Format:     ConfigExportFormat,
		ExportedAt: time.Now().UTC(),
		ExportedBy: "test@example.com",
		Connections: []ExportedConnection{
			{
				ExportID: "conn_12345678",
				Name:     "Test DB",
				Type:     "postgres",
				Host:     "localhost",
				Port:     5432,
			},
		},
	}

	t.Run("compact JSON", func(t *testing.T) {
		data, err := config.ToJSON(false)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "\n")
	})

	t.Run("pretty JSON", func(t *testing.T) {
		data, err := config.ToJSON(true)
		require.NoError(t, err)
		assert.Contains(t, string(data), "\n")
		assert.Contains(t, string(data), "  ")
	})
}

func TestParseConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		validConfig := `{
			"format": "howlerops-config-v1",
			"exported_at": "2024-01-01T00:00:00Z",
			"connections": [],
			"saved_queries": []
		}`

		config, err := ParseConfig([]byte(validConfig))
		require.NoError(t, err)
		assert.Equal(t, ConfigExportFormat, config.Format)
	})

	t.Run("invalid format version", func(t *testing.T) {
		invalidConfig := `{
			"format": "unknown-format-v999",
			"exported_at": "2024-01-01T00:00:00Z"
		}`

		_, err := ParseConfig([]byte(invalidConfig))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported config format")
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := ParseConfig([]byte("not valid json"))
		assert.Error(t, err)
	})
}

func TestValidateConfig(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := &ExportedConfig{
			Format: ConfigExportFormat,
			Connections: []ExportedConnection{
				{ExportID: "conn_1", Name: "DB1", Type: "postgres"},
			},
			SavedQueries: []ExportedSavedQuery{
				{ExportID: "query_1", Name: "Query1", ConnectionExportID: "conn_1"},
			},
		}

		issues := ValidateConfig(config)
		assert.Empty(t, issues)
	})

	t.Run("duplicate connection export IDs", func(t *testing.T) {
		config := &ExportedConfig{
			Format: ConfigExportFormat,
			Connections: []ExportedConnection{
				{ExportID: "conn_1", Name: "DB1", Type: "postgres"},
				{ExportID: "conn_1", Name: "DB2", Type: "mysql"}, // Duplicate
			},
		}

		issues := ValidateConfig(config)
		assert.NotEmpty(t, issues)
		assert.Contains(t, issues[0], "Duplicate connection")
	})

	t.Run("query references unknown connection", func(t *testing.T) {
		config := &ExportedConfig{
			Format:      ConfigExportFormat,
			Connections: []ExportedConnection{},
			SavedQueries: []ExportedSavedQuery{
				{ExportID: "query_1", Name: "Query1", ConnectionExportID: "nonexistent"},
			},
		}

		issues := ValidateConfig(config)
		assert.NotEmpty(t, issues)
		assert.Contains(t, issues[0], "unknown connection")
	})

	t.Run("connection missing name", func(t *testing.T) {
		config := &ExportedConfig{
			Format: ConfigExportFormat,
			Connections: []ExportedConnection{
				{ExportID: "conn_1", Name: "", Type: "postgres"},
			},
		}

		issues := ValidateConfig(config)
		assert.NotEmpty(t, issues)
		assert.Contains(t, issues[0], "no name")
	})
}

func TestExportRoundTrip(t *testing.T) {
	// Test that export -> JSON -> parse -> validate works correctly
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	exporter := NewConfigExporter(logger, "1.0.0")

	connections := []storage.Connection{
		{
			ID:       "conn-abc12345",
			Name:     "Test Connection",
			Type:     "postgres",
			Host:     "localhost",
			Port:     5432,
			Database: "testdb",
			Username: "testuser",
		},
	}

	queries := []storage.SavedQuery{
		{
			ID:           "query-xyz98765",
			Name:         "Test Query",
			Query:        "SELECT 1",
			ConnectionID: "conn-abc12345",
			Tags:         []string{"test"},
		},
	}

	options := DefaultConfigExportOptions()
	exported, err := exporter.Export(context.Background(), "test@test.com", connections, queries, nil, options)
	require.NoError(t, err)

	// Serialize
	jsonData, err := exported.ToJSON(true)
	require.NoError(t, err)

	// Parse back
	parsed, err := ParseConfig(jsonData)
	require.NoError(t, err)

	// Validate
	issues := ValidateConfig(parsed)
	assert.Empty(t, issues)

	// Verify content matches
	assert.Equal(t, exported.Format, parsed.Format)
	assert.Len(t, parsed.Connections, 1)
	assert.Equal(t, "Test Connection", parsed.Connections[0].Name)
	assert.Len(t, parsed.SavedQueries, 1)
	assert.Equal(t, "Test Query", parsed.SavedQueries[0].Name)
}

func TestExportSecurityNoPasswords(t *testing.T) {
	// Verify passwords are never in the exported JSON
	logger := logrus.New()
	exporter := NewConfigExporter(logger, "1.0.0")

	sensitivePassword := "super-secret-password-12345"
	connections := []storage.Connection{
		{
			ID:                "conn-security-test-12345",
			Name:              "Sensitive DB",
			Type:              "postgres",
			Host:              "secure.example.com",
			Password:          sensitivePassword,
			PasswordEncrypted: "encrypted-blob-here",
		},
	}

	config, err := exporter.Export(context.Background(), "test@test.com", connections, nil, nil, DefaultConfigExportOptions())
	require.NoError(t, err)

	// Check the struct
	assert.Len(t, config.Connections, 1)

	// Check the JSON
	jsonData, err := json.Marshal(config)
	require.NoError(t, err)

	jsonStr := string(jsonData)
	assert.NotContains(t, jsonStr, sensitivePassword)
	assert.NotContains(t, jsonStr, "encrypted-blob-here")
	assert.NotContains(t, jsonStr, "password")
}
