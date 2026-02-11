package export

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/jbeck018/howlerops/pkg/storage"
)

// ImportConflictStrategy defines how to handle conflicts during import
type ImportConflictStrategy string

const (
	// ConflictSkip skips items that already exist (by name)
	ConflictSkip ImportConflictStrategy = "skip"
	// ConflictOverwrite replaces existing items
	ConflictOverwrite ImportConflictStrategy = "overwrite"
	// ConflictRename creates new items with modified names
	ConflictRename ImportConflictStrategy = "rename"
	// ConflictMerge merges metadata/tags while keeping existing core data
	ConflictMerge ImportConflictStrategy = "merge"
)

// ConfigImportOptions configures the import behavior
type ConfigImportOptions struct {
	// What to do when an item with the same name exists
	ConflictStrategy ImportConflictStrategy `json:"conflict_strategy"`
	// Import connections
	ImportConnections bool `json:"import_connections"`
	// Import saved queries
	ImportSavedQueries bool `json:"import_saved_queries"`
	// Only import specific connection export IDs (empty = all)
	ConnectionExportIDs []string `json:"connection_export_ids,omitempty"`
	// Only import queries with specific tags (empty = all)
	QueryTags []string `json:"query_tags,omitempty"`
	// Override hosts with provided values (exportID -> host)
	HostOverrides map[string]string `json:"host_overrides,omitempty"`
	// Set all imported items as shared with this org
	ShareWithOrganization string `json:"share_with_organization,omitempty"`
	// Dry run - validate but don't actually import
	DryRun bool `json:"dry_run"`
}

// DefaultConfigImportOptions returns sensible defaults
func DefaultConfigImportOptions() ConfigImportOptions {
	return ConfigImportOptions{
		ConflictStrategy:   ConflictSkip,
		ImportConnections:  true,
		ImportSavedQueries: true,
		DryRun:             false,
	}
}

// ImportResult contains the results of an import operation
type ImportResult struct {
	// Whether the import was a dry run
	DryRun bool `json:"dry_run"`
	// Connections that were imported
	ConnectionsImported int `json:"connections_imported"`
	// Connections that were skipped
	ConnectionsSkipped int `json:"connections_skipped"`
	// Connection import errors
	ConnectionErrors []ImportError `json:"connection_errors,omitempty"`
	// Mapping of export IDs to new IDs (for linking queries)
	ConnectionIDMap map[string]string `json:"connection_id_map"`
	// Queries that were imported
	QueriesImported int `json:"queries_imported"`
	// Queries that were skipped
	QueriesSkipped int `json:"queries_skipped"`
	// Query import errors
	QueryErrors []ImportError `json:"query_errors,omitempty"`
	// New tags that were created
	NewTags []string `json:"new_tags,omitempty"`
	// New folders that were created
	NewFolders []string `json:"new_folders,omitempty"`
	// Items that need passwords set
	ConnectionsNeedingPasswords []PasswordRequired `json:"connections_needing_passwords"`
	// Warnings (non-fatal issues)
	Warnings []string `json:"warnings,omitempty"`
}

// ImportError represents an error during import
type ImportError struct {
	ExportID string `json:"export_id"`
	Name     string `json:"name"`
	Error    string `json:"error"`
}

// PasswordRequired indicates a connection that needs a password set
type PasswordRequired struct {
	NewConnectionID string `json:"new_connection_id"`
	ExportID        string `json:"export_id"`
	Name            string `json:"name"`
	Host            string `json:"host"`
	Database        string `json:"database"`
}

// ConfigImporter handles importing configurations
type ConfigImporter struct {
	logger *logrus.Logger
}

// NewConfigImporter creates a new config importer
func NewConfigImporter(logger *logrus.Logger) *ConfigImporter {
	return &ConfigImporter{
		logger: logger,
	}
}

// ConnectionStore interface for connection operations
type ConnectionStore interface {
	GetByName(ctx context.Context, userID, name string) (*storage.Connection, error)
	Create(ctx context.Context, conn *storage.Connection) error
	Update(ctx context.Context, conn *storage.Connection) error
}

// QueryStore interface for query operations
type QueryStore interface {
	GetByName(ctx context.Context, userID, name string) (*storage.SavedQuery, error)
	Create(ctx context.Context, query *storage.SavedQuery) error
	Update(ctx context.Context, query *storage.SavedQuery) error
}

// Import imports a config into the system
func (i *ConfigImporter) Import(
	ctx context.Context,
	config *ExportedConfig,
	userID string,
	connStore ConnectionStore,
	queryStore QueryStore,
	options ConfigImportOptions,
) (*ImportResult, error) {
	result := &ImportResult{
		DryRun:                      options.DryRun,
		ConnectionIDMap:             make(map[string]string),
		ConnectionsNeedingPasswords: []PasswordRequired{},
		Warnings:                    []string{},
	}

	// Validate config format
	if config.Format != ConfigExportFormat {
		return nil, fmt.Errorf("unsupported config format: %s", config.Format)
	}

	// Import connections first (queries reference them)
	if options.ImportConnections && len(config.Connections) > 0 {
		if err := i.importConnections(ctx, config, userID, connStore, options, result); err != nil {
			return result, fmt.Errorf("connection import failed: %w", err)
		}
	}

	// Import queries
	if options.ImportSavedQueries && len(config.SavedQueries) > 0 {
		if err := i.importQueries(ctx, config, userID, queryStore, options, result); err != nil {
			return result, fmt.Errorf("query import failed: %w", err)
		}
	}

	// Collect new tags and folders
	result.NewTags = config.Tags
	result.NewFolders = config.Folders

	return result, nil
}

func (i *ConfigImporter) importConnections(
	ctx context.Context,
	config *ExportedConfig,
	userID string,
	store ConnectionStore,
	options ConfigImportOptions,
	result *ImportResult,
) error {
	for _, exported := range config.Connections {
		// Filter by export IDs if specified
		if len(options.ConnectionExportIDs) > 0 && !contains(options.ConnectionExportIDs, exported.ExportID) {
			result.ConnectionsSkipped++
			continue
		}

		// Check for existing connection with same name
		existing, err := store.GetByName(ctx, userID, exported.Name)
		if err != nil {
			i.logger.WithError(err).WithField("name", exported.Name).Debug("Error checking for existing connection")
		}

		// Determine the host to use
		host := exported.Host
		if override, ok := options.HostOverrides[exported.ExportID]; ok {
			host = override
		}

		// Check if host is a placeholder that needs replacement
		if len(host) > 2 && host[:2] == "{{" && host[len(host)-2:] == "}}" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Connection %q has placeholder host %q - you'll need to set the actual host", exported.Name, host))
		}

		newConn := &storage.Connection{
			ID:          uuid.New().String(),
			Name:        exported.Name,
			Type:        exported.Type,
			Host:        host,
			Port:        exported.Port,
			Database:    exported.Database,
			Username:    exported.Username,
			Environment: exported.Environment,
			Metadata:    exported.Metadata,
			CreatedBy:   userID,
			IsShared:    options.ShareWithOrganization != "",
			TeamID:      options.ShareWithOrganization,
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}

		if existing != nil {
			switch options.ConflictStrategy {
			case ConflictSkip:
				result.ConnectionsSkipped++
				result.ConnectionIDMap[exported.ExportID] = existing.ID
				continue

			case ConflictOverwrite:
				newConn.ID = existing.ID
				newConn.CreatedAt = existing.CreatedAt
				if !options.DryRun {
					if err := store.Update(ctx, newConn); err != nil {
						result.ConnectionErrors = append(result.ConnectionErrors, ImportError{
							ExportID: exported.ExportID,
							Name:     exported.Name,
							Error:    fmt.Sprintf("failed to update: %v", err),
						})
						continue
					}
				}

			case ConflictRename:
				newConn.Name = fmt.Sprintf("%s (imported %s)", exported.Name, time.Now().Format("2006-01-02"))
				if !options.DryRun {
					if err := store.Create(ctx, newConn); err != nil {
						result.ConnectionErrors = append(result.ConnectionErrors, ImportError{
							ExportID: exported.ExportID,
							Name:     exported.Name,
							Error:    fmt.Sprintf("failed to create renamed: %v", err),
						})
						continue
					}
				}

			case ConflictMerge:
				// Merge metadata
				for k, v := range exported.Metadata {
					if existing.Metadata == nil {
						existing.Metadata = make(map[string]string)
					}
					if _, exists := existing.Metadata[k]; !exists {
						existing.Metadata[k] = v
					}
				}
				if !options.DryRun {
					if err := store.Update(ctx, existing); err != nil {
						result.ConnectionErrors = append(result.ConnectionErrors, ImportError{
							ExportID: exported.ExportID,
							Name:     exported.Name,
							Error:    fmt.Sprintf("failed to merge: %v", err),
						})
						continue
					}
				}
				result.ConnectionIDMap[exported.ExportID] = existing.ID
				result.ConnectionsImported++
				continue
			}
		} else {
			// New connection
			if !options.DryRun {
				if err := store.Create(ctx, newConn); err != nil {
					result.ConnectionErrors = append(result.ConnectionErrors, ImportError{
						ExportID: exported.ExportID,
						Name:     exported.Name,
						Error:    fmt.Sprintf("failed to create: %v", err),
					})
					continue
				}
			}
		}

		result.ConnectionIDMap[exported.ExportID] = newConn.ID
		result.ConnectionsImported++

		// Track that this connection needs a password
		result.ConnectionsNeedingPasswords = append(result.ConnectionsNeedingPasswords, PasswordRequired{
			NewConnectionID: newConn.ID,
			ExportID:        exported.ExportID,
			Name:            newConn.Name,
			Host:            newConn.Host,
			Database:        newConn.Database,
		})
	}

	return nil
}

func (i *ConfigImporter) importQueries(
	ctx context.Context,
	config *ExportedConfig,
	userID string,
	store QueryStore,
	options ConfigImportOptions,
	result *ImportResult,
) error {
	for _, exported := range config.SavedQueries {
		// Filter by tags if specified
		if len(options.QueryTags) > 0 && !hasAnyTag(exported.Tags, options.QueryTags) {
			result.QueriesSkipped++
			continue
		}

		// Check for existing query with same name
		existing, err := store.GetByName(ctx, userID, exported.Name)
		if err != nil {
			i.logger.WithError(err).WithField("name", exported.Name).Debug("Error checking for existing query")
		}

		// Map connection reference
		connectionID := ""
		if exported.ConnectionExportID != "" {
			if mappedID, ok := result.ConnectionIDMap[exported.ConnectionExportID]; ok {
				connectionID = mappedID
			} else {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Query %q references unknown connection %q", exported.Name, exported.ConnectionExportID))
			}
		}

		newQuery := &storage.SavedQuery{
			ID:           uuid.New().String(),
			Name:         exported.Name,
			Description:  exported.Description,
			Query:        exported.Query,
			ConnectionID: connectionID,
			Folder:       exported.Folder,
			Tags:         exported.Tags,
			Metadata:     exported.Metadata,
			CreatedBy:    userID,
			IsShared:     options.ShareWithOrganization != "",
			TeamID:       options.ShareWithOrganization,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if existing != nil {
			switch options.ConflictStrategy {
			case ConflictSkip:
				result.QueriesSkipped++
				continue

			case ConflictOverwrite:
				newQuery.ID = existing.ID
				newQuery.CreatedAt = existing.CreatedAt
				if !options.DryRun {
					if err := store.Update(ctx, newQuery); err != nil {
						result.QueryErrors = append(result.QueryErrors, ImportError{
							ExportID: exported.ExportID,
							Name:     exported.Name,
							Error:    fmt.Sprintf("failed to update: %v", err),
						})
						continue
					}
				}

			case ConflictRename:
				newQuery.Name = fmt.Sprintf("%s (imported %s)", exported.Name, time.Now().Format("2006-01-02"))
				if !options.DryRun {
					if err := store.Create(ctx, newQuery); err != nil {
						result.QueryErrors = append(result.QueryErrors, ImportError{
							ExportID: exported.ExportID,
							Name:     exported.Name,
							Error:    fmt.Sprintf("failed to create renamed: %v", err),
						})
						continue
					}
				}

			case ConflictMerge:
				// Merge tags (dedupe)
				tagSet := make(map[string]bool)
				for _, t := range existing.Tags {
					tagSet[t] = true
				}
				for _, t := range exported.Tags {
					tagSet[t] = true
				}
				mergedTags := make([]string, 0, len(tagSet))
				for t := range tagSet {
					mergedTags = append(mergedTags, t)
				}
				existing.Tags = mergedTags

				// Merge metadata
				for k, v := range exported.Metadata {
					if existing.Metadata == nil {
						existing.Metadata = make(map[string]string)
					}
					if _, exists := existing.Metadata[k]; !exists {
						existing.Metadata[k] = v
					}
				}

				if !options.DryRun {
					if err := store.Update(ctx, existing); err != nil {
						result.QueryErrors = append(result.QueryErrors, ImportError{
							ExportID: exported.ExportID,
							Name:     exported.Name,
							Error:    fmt.Sprintf("failed to merge: %v", err),
						})
						continue
					}
				}
				result.QueriesImported++
				continue
			}
		} else {
			// New query
			if !options.DryRun {
				if err := store.Create(ctx, newQuery); err != nil {
					result.QueryErrors = append(result.QueryErrors, ImportError{
						ExportID: exported.ExportID,
						Name:     exported.Name,
						Error:    fmt.Sprintf("failed to create: %v", err),
					})
					continue
				}
			}
		}

		result.QueriesImported++
	}

	return nil
}

// ValidateConfig performs validation on a config before import
func ValidateConfig(config *ExportedConfig) []string {
	var issues []string

	if config.Format != ConfigExportFormat {
		issues = append(issues, fmt.Sprintf("Unknown format: %s", config.Format))
	}

	// Check for duplicate export IDs
	connIDs := make(map[string]bool)
	for _, c := range config.Connections {
		if connIDs[c.ExportID] {
			issues = append(issues, fmt.Sprintf("Duplicate connection export_id: %s", c.ExportID))
		}
		connIDs[c.ExportID] = true

		if c.Name == "" {
			issues = append(issues, fmt.Sprintf("Connection %s has no name", c.ExportID))
		}
		if c.Type == "" {
			issues = append(issues, fmt.Sprintf("Connection %s has no type", c.ExportID))
		}
	}

	queryIDs := make(map[string]bool)
	for _, q := range config.SavedQueries {
		if queryIDs[q.ExportID] {
			issues = append(issues, fmt.Sprintf("Duplicate query export_id: %s", q.ExportID))
		}
		queryIDs[q.ExportID] = true

		if q.Name == "" {
			issues = append(issues, fmt.Sprintf("Query %s has no name", q.ExportID))
		}

		// Verify connection reference exists
		if q.ConnectionExportID != "" && !connIDs[q.ConnectionExportID] {
			issues = append(issues, fmt.Sprintf("Query %s references unknown connection: %s", q.ExportID, q.ConnectionExportID))
		}
	}

	return issues
}
