package export

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/pkg/storage"
)

// ConfigExportFormat is the version of the config export format
const ConfigExportFormat = "howlerops-config-v1"

// ConfigExportOptions specifies what to include in the export
type ConfigExportOptions struct {
	// Include connections (without passwords)
	IncludeConnections bool `json:"include_connections"`
	// Include saved queries
	IncludeSavedQueries bool `json:"include_saved_queries"`
	// Include query history (sanitized, no actual query results)
	IncludeQueryHistory bool `json:"include_query_history"`
	// Include only specific connection IDs (empty = all)
	ConnectionIDs []string `json:"connection_ids,omitempty"`
	// Include only queries with specific tags (empty = all)
	QueryTags []string `json:"query_tags,omitempty"`
	// Include organization-shared resources
	IncludeShared bool `json:"include_shared"`
	// Export metadata only (no actual query text for queries)
	MetadataOnly bool `json:"metadata_only"`
	// Anonymize hostnames (replace with placeholders)
	AnonymizeHosts bool `json:"anonymize_hosts"`
}

// DefaultConfigExportOptions returns sensible defaults
func DefaultConfigExportOptions() ConfigExportOptions {
	return ConfigExportOptions{
		IncludeConnections:  true,
		IncludeSavedQueries: true,
		IncludeQueryHistory: false,
		IncludeShared:       false, // Personal by default
		MetadataOnly:        false,
		AnonymizeHosts:      false,
	}
}

// ExportedConfig is the root structure for config export
type ExportedConfig struct {
	// Format version identifier
	Format string `json:"format"`
	// Export timestamp
	ExportedAt time.Time `json:"exported_at"`
	// Who exported this config (email, optional)
	ExportedBy string `json:"exported_by,omitempty"`
	// Application version
	AppVersion string `json:"app_version,omitempty"`
	// What was included in this export
	ExportOptions ConfigExportOptions `json:"export_options"`
	// Connections (without passwords)
	Connections []ExportedConnection `json:"connections,omitempty"`
	// Saved queries
	SavedQueries []ExportedSavedQuery `json:"saved_queries,omitempty"`
	// Query history (sanitized)
	QueryHistory []ExportedQueryHistory `json:"query_history,omitempty"`
	// All unique tags used across queries
	Tags []string `json:"tags,omitempty"`
	// All unique folders used
	Folders []string `json:"folders,omitempty"`
}

// ExportedConnection is a connection without sensitive data
type ExportedConnection struct {
	// Reference ID for linking queries
	ExportID string `json:"export_id"`
	// Original ID (for re-import mapping)
	OriginalID string `json:"original_id,omitempty"`
	// User-facing name
	Name string `json:"name"`
	// Database type (postgres, mysql, etc.)
	Type string `json:"type"`
	// Host (may be anonymized)
	Host string `json:"host"`
	// Port
	Port int `json:"port"`
	// Database name
	Database string `json:"database"`
	// Username (NOT password)
	Username string `json:"username"`
	// Environment (dev, staging, prod)
	Environment string `json:"environment,omitempty"`
	// SSL configuration keys (not values)
	SSLConfigKeys []string `json:"ssl_config_keys,omitempty"`
	// Custom metadata
	Metadata map[string]string `json:"metadata,omitempty"`
	// Whether this was shared in an org
	WasShared bool `json:"was_shared"`
	// Original creation time
	CreatedAt time.Time `json:"created_at"`
}

// ExportedSavedQuery is a saved query for export
type ExportedSavedQuery struct {
	// Reference ID for import
	ExportID string `json:"export_id"`
	// Original ID (for re-import)
	OriginalID string `json:"original_id,omitempty"`
	// Query name/title
	Name string `json:"name"`
	// Description
	Description string `json:"description,omitempty"`
	// SQL query text (omitted if metadata_only)
	Query string `json:"query,omitempty"`
	// Reference to ExportedConnection.ExportID
	ConnectionExportID string `json:"connection_export_id,omitempty"`
	// Connection type for reference
	ConnectionType string `json:"connection_type,omitempty"`
	// Folder path
	Folder string `json:"folder,omitempty"`
	// Tags
	Tags []string `json:"tags,omitempty"`
	// Custom metadata
	Metadata map[string]string `json:"metadata,omitempty"`
	// Favorite flag
	Favorite bool `json:"favorite"`
	// Whether this was shared
	WasShared bool `json:"was_shared"`
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	// Last update time
	UpdatedAt time.Time `json:"updated_at"`
}

// ExportedQueryHistory is sanitized query history
type ExportedQueryHistory struct {
	// Reference to connection
	ConnectionExportID string `json:"connection_export_id,omitempty"`
	// Execution time
	ExecutedAt time.Time `json:"executed_at"`
	// Duration in milliseconds
	DurationMS int `json:"duration_ms"`
	// Row count (not actual data)
	RowsReturned int `json:"rows_returned"`
	// Whether it succeeded
	Success bool `json:"success"`
	// Error message (if any)
	Error string `json:"error,omitempty"`
}

// ConfigExporter handles exporting user configurations
type ConfigExporter struct {
	logger     *logrus.Logger
	appVersion string
}

// NewConfigExporter creates a new config exporter
func NewConfigExporter(logger *logrus.Logger, appVersion string) *ConfigExporter {
	return &ConfigExporter{
		logger:     logger,
		appVersion: appVersion,
	}
}

// Export creates an exported config from the given data
func (e *ConfigExporter) Export(
	ctx context.Context,
	userEmail string,
	connections []storage.Connection,
	queries []storage.SavedQuery,
	history []storage.QueryHistory,
	options ConfigExportOptions,
) (*ExportedConfig, error) {
	config := &ExportedConfig{
		Format:        ConfigExportFormat,
		ExportedAt:    time.Now().UTC(),
		ExportedBy:    userEmail,
		AppVersion:    e.appVersion,
		ExportOptions: options,
		Tags:          []string{},
		Folders:       []string{},
	}

	// Track unique tags and folders
	tagSet := make(map[string]bool)
	folderSet := make(map[string]bool)

	// Build connection export ID mapping
	connExportIDs := make(map[string]string)
	connTypes := make(map[string]string)

	// Export connections
	if options.IncludeConnections {
		for _, conn := range connections {
			// Skip if filtering by IDs and not in list
			if len(options.ConnectionIDs) > 0 && !contains(options.ConnectionIDs, conn.ID) {
				continue
			}
			// Skip shared if not including shared
			if conn.IsShared && !options.IncludeShared {
				continue
			}

			idSuffix := conn.ID
			if len(idSuffix) > 8 {
				idSuffix = idSuffix[:8]
			}
			exportID := fmt.Sprintf("conn_%s", idSuffix)
			connExportIDs[conn.ID] = exportID
			connTypes[conn.ID] = conn.Type

			exported := ExportedConnection{
				ExportID:    exportID,
				OriginalID:  conn.ID,
				Name:        conn.Name,
				Type:        conn.Type,
				Host:        conn.Host,
				Port:        conn.Port,
				Database:    conn.Database,
				Username:    conn.Username,
				Environment: conn.Environment,
				Metadata:    conn.Metadata,
				WasShared:   conn.IsShared,
				CreatedAt:   conn.CreatedAt,
			}

			// Anonymize host if requested
			if options.AnonymizeHosts {
				exported.Host = fmt.Sprintf("{{%s_HOST}}", conn.Name)
			}

			// Extract SSL config keys only (not values)
			if conn.SSLConfig != nil {
				exported.SSLConfigKeys = make([]string, 0, len(conn.SSLConfig))
				for k := range conn.SSLConfig {
					exported.SSLConfigKeys = append(exported.SSLConfigKeys, k)
				}
			}

			config.Connections = append(config.Connections, exported)
		}
	}

	// Export saved queries
	if options.IncludeSavedQueries {
		for _, query := range queries {
			// Skip shared if not including shared
			if query.IsShared && !options.IncludeShared {
				continue
			}

			// Filter by tags if specified
			if len(options.QueryTags) > 0 && !hasAnyTag(query.Tags, options.QueryTags) {
				continue
			}

			queryIDSuffix := query.ID
			if len(queryIDSuffix) > 8 {
				queryIDSuffix = queryIDSuffix[:8]
			}
			exportID := fmt.Sprintf("query_%s", queryIDSuffix)

			exported := ExportedSavedQuery{
				ExportID:           exportID,
				OriginalID:         query.ID,
				Name:               query.Name,
				Description:        query.Description,
				ConnectionExportID: connExportIDs[query.ConnectionID],
				ConnectionType:     connTypes[query.ConnectionID],
				Folder:             query.Folder,
				Tags:               query.Tags,
				Metadata:           query.Metadata,
				WasShared:          query.IsShared,
				CreatedAt:          query.CreatedAt,
				UpdatedAt:          query.UpdatedAt,
			}

			// Include query text unless metadata only
			if !options.MetadataOnly {
				exported.Query = query.Query
			}

			config.SavedQueries = append(config.SavedQueries, exported)

			// Track tags and folders
			for _, tag := range query.Tags {
				tagSet[tag] = true
			}
			if query.Folder != "" {
				folderSet[query.Folder] = true
			}
		}
	}

	// Export query history (sanitized)
	if options.IncludeQueryHistory {
		for _, h := range history {
			// Skip if connection wasn't exported
			if _, ok := connExportIDs[h.ConnectionID]; !ok {
				continue
			}

			exported := ExportedQueryHistory{
				ConnectionExportID: connExportIDs[h.ConnectionID],
				ExecutedAt:         h.ExecutedAt,
				DurationMS:         h.Duration,
				RowsReturned:       h.RowsReturned,
				Success:            h.Success,
				Error:              h.Error,
			}
			config.QueryHistory = append(config.QueryHistory, exported)
		}
	}

	// Convert tag and folder sets to slices
	for tag := range tagSet {
		config.Tags = append(config.Tags, tag)
	}
	for folder := range folderSet {
		config.Folders = append(config.Folders, folder)
	}

	return config, nil
}

// ExportWithPasswords creates an encrypted export that includes passwords
// This requires a user-provided passphrase for encryption
func (e *ConfigExporter) ExportWithPasswords(
	ctx context.Context,
	userEmail string,
	connections []storage.Connection,
	queries []storage.SavedQuery,
	options ConfigExportOptions,
	passphrase string,
) (*EncryptedConfigExport, error) {
	// Build credentials with passwords
	var credentials []ExportedCredential
	connExportIDs := make(map[string]string)
	connTypes := make(map[string]string)

	for _, conn := range connections {
		// Apply same filtering as regular export
		if len(options.ConnectionIDs) > 0 && !contains(options.ConnectionIDs, conn.ID) {
			continue
		}
		if conn.IsShared && !options.IncludeShared {
			continue
		}

		idSuffix := conn.ID
		if len(idSuffix) > 8 {
			idSuffix = idSuffix[:8]
		}
		exportID := fmt.Sprintf("conn_%s", idSuffix)
		connExportIDs[conn.ID] = exportID
		connTypes[conn.ID] = conn.Type

		host := conn.Host
		if options.AnonymizeHosts {
			host = fmt.Sprintf("{{%s_HOST}}", conn.Name)
		}

		cred := ExportedCredential{
			ExportedConnection: ExportedConnection{
				ExportID:    exportID,
				OriginalID:  conn.ID,
				Name:        conn.Name,
				Type:        conn.Type,
				Host:        host,
				Port:        conn.Port,
				Database:    conn.Database,
				Username:    conn.Username,
				Environment: conn.Environment,
				Metadata:    conn.Metadata,
				WasShared:   conn.IsShared,
				CreatedAt:   conn.CreatedAt,
			},
			Password: conn.Password, // Include password in encrypted export
		}

		if conn.SSLConfig != nil {
			cred.SSLConfigKeys = make([]string, 0, len(conn.SSLConfig))
			for k := range conn.SSLConfig {
				cred.SSLConfigKeys = append(cred.SSLConfigKeys, k)
			}
		}

		credentials = append(credentials, cred)
	}

	// Build saved queries (same as regular export)
	var savedQueries []ExportedSavedQuery
	tagSet := make(map[string]bool)
	folderSet := make(map[string]bool)

	if options.IncludeSavedQueries {
		for _, query := range queries {
			if query.IsShared && !options.IncludeShared {
				continue
			}
			if len(options.QueryTags) > 0 && !hasAnyTag(query.Tags, options.QueryTags) {
				continue
			}

			queryIDSuffix := query.ID
			if len(queryIDSuffix) > 8 {
				queryIDSuffix = queryIDSuffix[:8]
			}
			exportID := fmt.Sprintf("query_%s", queryIDSuffix)

			exported := ExportedSavedQuery{
				ExportID:           exportID,
				OriginalID:         query.ID,
				Name:               query.Name,
				Description:        query.Description,
				ConnectionExportID: connExportIDs[query.ConnectionID],
				ConnectionType:     connTypes[query.ConnectionID],
				Folder:             query.Folder,
				Tags:               query.Tags,
				Metadata:           query.Metadata,
				WasShared:          query.IsShared,
				CreatedAt:          query.CreatedAt,
				UpdatedAt:          query.UpdatedAt,
			}

			if !options.MetadataOnly {
				exported.Query = query.Query
			}

			savedQueries = append(savedQueries, exported)

			for _, tag := range query.Tags {
				tagSet[tag] = true
			}
			if query.Folder != "" {
				folderSet[query.Folder] = true
			}
		}
	}

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	folders := make([]string, 0, len(folderSet))
	for folder := range folderSet {
		folders = append(folders, folder)
	}

	// Create payload
	payload := &FullExportPayload{
		Credentials:  credentials,
		SavedQueries: savedQueries,
		Tags:         tags,
		Folders:      folders,
		ExportedAt:   time.Now().UTC(),
		ExportedBy:   userEmail,
		AppVersion:   e.appVersion,
	}

	// Encrypt with passphrase
	return EncryptExport(payload, passphrase)
}

// ToJSON serializes the config to JSON
func (c *ExportedConfig) ToJSON(pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(c, "", "  ")
	}
	return json.Marshal(c)
}

// ParseConfig parses a JSON config
func ParseConfig(data []byte) (*ExportedConfig, error) {
	var config ExportedConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("invalid config format: %w", err)
	}

	if config.Format != ConfigExportFormat {
		return nil, fmt.Errorf("unsupported config format: %s (expected %s)", config.Format, ConfigExportFormat)
	}

	return &config, nil
}

// helper functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func hasAnyTag(queryTags, filterTags []string) bool {
	for _, qt := range queryTags {
		for _, ft := range filterTags {
			if qt == ft {
				return true
			}
		}
	}
	return false
}
