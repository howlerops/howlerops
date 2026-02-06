package main

import (
	"context"
	"fmt"

	"github.com/jbeck018/howlerops/pkg/catalog"
)

// Data Catalog Service Wails Bindings
// These methods expose data catalog functionality to the frontend

// catalogService is lazily initialized when first needed
func (a *App) getCatalogService() (*catalog.Service, error) {
	if a.storageManager == nil {
		return nil, fmt.Errorf("storage manager not initialized")
	}

	// Create SQLite store using storage manager's data directory
	store, err := catalog.NewSQLiteStore(a.storageManager.GetDataDir())
	if err != nil {
		return nil, fmt.Errorf("failed to create catalog store: %w", err)
	}

	// Initialize the store
	if err := store.Initialize(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to initialize catalog store: %w", err)
	}

	return catalog.NewService(store), nil
}

// CreateTableCatalogEntry creates a new table entry in the catalog
func (a *App) CreateTableCatalogEntry(entry *catalog.TableCatalogEntry) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.CreateTableEntry(context.Background(), entry)
}

// GetTableCatalogEntry retrieves a table catalog entry
func (a *App) GetTableCatalogEntry(connectionID, schema, table string) (*catalog.TableCatalogEntry, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	return service.GetTableEntry(context.Background(), connectionID, schema, table)
}

// UpdateTableCatalogEntry updates a table catalog entry
func (a *App) UpdateTableCatalogEntry(entry *catalog.TableCatalogEntry) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.UpdateTableEntry(context.Background(), entry)
}

// DeleteTableCatalogEntry deletes a table catalog entry
func (a *App) DeleteTableCatalogEntry(id string) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.DeleteTableEntry(context.Background(), id)
}

// ListTableCatalogEntries lists all table entries for a connection
func (a *App) ListTableCatalogEntries(connectionID string) ([]*catalog.TableCatalogEntry, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	return service.ListTableEntries(context.Background(), connectionID)
}

// CreateColumnCatalogEntry creates a new column catalog entry
func (a *App) CreateColumnCatalogEntry(entry *catalog.ColumnCatalogEntry) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.CreateColumnEntry(context.Background(), entry)
}

// GetColumnCatalogEntry retrieves a column catalog entry
func (a *App) GetColumnCatalogEntry(tableID, column string) (*catalog.ColumnCatalogEntry, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	return service.GetColumnEntry(context.Background(), tableID, column)
}

// UpdateColumnCatalogEntry updates a column catalog entry
func (a *App) UpdateColumnCatalogEntry(entry *catalog.ColumnCatalogEntry) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.UpdateColumnEntry(context.Background(), entry)
}

// ListColumnCatalogEntries lists all column entries for a table
func (a *App) ListColumnCatalogEntries(tableID string) ([]*catalog.ColumnCatalogEntry, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	return service.ListColumnEntries(context.Background(), tableID)
}

// CreateCatalogTag creates a new custom tag
func (a *App) CreateCatalogTag(tag *catalog.CatalogTag) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.CreateTag(context.Background(), tag)
}

// ListCatalogTags lists all tags for an organization
func (a *App) ListCatalogTags(orgID *string) ([]*catalog.CatalogTag, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	return service.ListTags(context.Background(), orgID)
}

// DeleteCatalogTag deletes a custom tag
func (a *App) DeleteCatalogTag(id string) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.DeleteTag(context.Background(), id)
}

// SearchCatalog searches the catalog for tables and columns
func (a *App) SearchCatalog(query string, filters *catalog.SearchFilters) (*catalog.SearchResults, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	return service.SearchCatalog(context.Background(), query, filters)
}

// AssignTableSteward assigns a steward to a table
func (a *App) AssignTableSteward(tableID, userID string) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.AssignSteward(context.Background(), tableID, userID)
}

// MarkColumnAsPII marks a column as containing PII
func (a *App) MarkColumnAsPII(tableID, columnName, piiType string, confidence float64) error {
	service, err := a.getCatalogService()
	if err != nil {
		return err
	}

	return service.MarkColumnAsPII(context.Background(), tableID, columnName, piiType, confidence)
}

// GetCatalogStats returns statistics about the catalog
func (a *App) GetCatalogStats(connectionID string) (*CatalogStats, error) {
	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Get table entries
	tables, err := service.ListTableEntries(ctx, connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %w", err)
	}

	stats := &CatalogStats{
		TotalTables:  len(tables),
		TotalColumns: 0,
		TaggedTables: 0,
		PIIColumns:   0,
	}

	// Count columns and stats
	for _, table := range tables {
		if len(table.Tags) > 0 {
			stats.TaggedTables++
		}

		columns, err := service.ListColumnEntries(ctx, table.ID)
		if err != nil {
			continue // Skip on error
		}

		stats.TotalColumns += len(columns)
		for _, col := range columns {
			if col.PIIType != nil {
				stats.PIIColumns++
			}
		}
	}

	return stats, nil
}

// CatalogStats represents statistics about the catalog
type CatalogStats struct {
	TotalTables  int `json:"total_tables"`
	TotalColumns int `json:"total_columns"`
	TaggedTables int `json:"tagged_tables"`
	PIIColumns   int `json:"pii_columns"`
}

// SyncCatalogFromConnection syncs the catalog with the actual database schema
func (a *App) SyncCatalogFromConnection(connectionID string) (*CatalogSyncResult, error) {
	if a.databaseService == nil {
		return nil, fmt.Errorf("database service not initialized")
	}

	service, err := a.getCatalogService()
	if err != nil {
		return nil, err
	}

	ctx := context.Background()

	// Get database connection
	db, err := a.databaseService.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	result := &CatalogSyncResult{
		TablesAdded:   0,
		TablesUpdated: 0,
		ColumnsAdded:  0,
	}

	// Get all schemas
	schemas, err := db.GetSchemas(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	// Sync each schema's tables
	for _, schema := range schemas {
		tables, err := db.GetTables(ctx, schema)
		if err != nil {
			continue // Skip on error
		}

		for _, table := range tables {
			// Check if table already exists in catalog
			existing, err := service.GetTableEntry(ctx, connectionID, schema, table.Name)
			if err != nil {
				continue
			}

			if existing == nil {
				// Create new catalog entry
				entry := &catalog.TableCatalogEntry{
					ConnectionID: connectionID,
					SchemaName:   schema,
					TableName:    table.Name,
					CreatedBy:    "system-sync",
					Tags:         []string{},
				}
				if err := service.CreateTableEntry(ctx, entry); err != nil {
					continue
				}
				result.TablesAdded++

				// Sync columns
				structure, err := db.GetTableStructure(ctx, schema, table.Name)
				if err != nil {
					continue
				}

				for _, col := range structure.Columns {
					colEntry := &catalog.ColumnCatalogEntry{
						TableCatalogID: entry.ID,
						ColumnName:     col.Name,
						Tags:           []string{},
					}
					if err := service.CreateColumnEntry(ctx, colEntry); err == nil {
						result.ColumnsAdded++
					}
				}
			} else {
				result.TablesUpdated++
			}
		}
	}

	return result, nil
}

// CatalogSyncResult represents the result of a catalog sync operation
type CatalogSyncResult struct {
	TablesAdded   int `json:"tables_added"`
	TablesUpdated int `json:"tables_updated"`
	ColumnsAdded  int `json:"columns_added"`
}
