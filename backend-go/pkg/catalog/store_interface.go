package catalog

import "context"

// Store defines the interface for catalog storage operations
type Store interface {
	// Initialize creates necessary tables and indexes
	Initialize(ctx context.Context) error

	// Close closes the storage connection
	Close() error

	// Table operations
	CreateTableEntry(ctx context.Context, entry *TableCatalogEntry) error
	GetTableEntry(ctx context.Context, connectionID, schema, table string) (*TableCatalogEntry, error)
	UpdateTableEntry(ctx context.Context, entry *TableCatalogEntry) error
	DeleteTableEntry(ctx context.Context, id string) error
	ListTableEntries(ctx context.Context, connectionID string) ([]*TableCatalogEntry, error)

	// Column operations
	CreateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error
	GetColumnEntry(ctx context.Context, tableID, column string) (*ColumnCatalogEntry, error)
	UpdateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error
	ListColumnEntries(ctx context.Context, tableID string) ([]*ColumnCatalogEntry, error)

	// Tag operations
	CreateTag(ctx context.Context, tag *CatalogTag) error
	ListTags(ctx context.Context, orgID *string) ([]*CatalogTag, error)
	DeleteTag(ctx context.Context, id string) error

	// Search operations
	SearchCatalog(ctx context.Context, query string, filters *SearchFilters) (*SearchResults, error)
}
