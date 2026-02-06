package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Service provides high-level catalog operations
type Service struct {
	store Store
}

// NewService creates a new catalog service
func NewService(store Store) *Service {
	return &Service{store: store}
}

// NewServiceWithSQLite creates a service with a SQLite store
func NewServiceWithSQLite(dataDir string, enableFTS5 bool) (*Service, error) {
	sqliteStore, err := NewSQLiteStore(dataDir)
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := sqliteStore.Initialize(ctx); err != nil {
		sqliteStore.Close()
		return nil, err
	}

	// Enable FTS5 if requested
	if enableFTS5 {
		if err := sqliteStore.EnableFTS5(ctx); err != nil {
			sqliteStore.Close()
			return nil, fmt.Errorf("enable FTS5: %w", err)
		}
	}

	return &Service{store: sqliteStore}, nil
}

// Close closes the service and underlying store
func (s *Service) Close() error {
	return s.store.Close()
}

// Passthrough methods for CRUD operations
// These provide direct access to store methods for Wails bindings

// CreateTableEntry creates a new table catalog entry
func (s *Service) CreateTableEntry(ctx context.Context, entry *TableCatalogEntry) error {
	return s.store.CreateTableEntry(ctx, entry)
}

// GetTableEntry retrieves a table catalog entry
func (s *Service) GetTableEntry(ctx context.Context, connectionID, schema, table string) (*TableCatalogEntry, error) {
	return s.store.GetTableEntry(ctx, connectionID, schema, table)
}

// UpdateTableEntry updates a table catalog entry
func (s *Service) UpdateTableEntry(ctx context.Context, entry *TableCatalogEntry) error {
	return s.store.UpdateTableEntry(ctx, entry)
}

// DeleteTableEntry deletes a table catalog entry by ID
func (s *Service) DeleteTableEntry(ctx context.Context, id string) error {
	return s.store.DeleteTableEntry(ctx, id)
}

// ListTableEntries lists all table entries for a connection
func (s *Service) ListTableEntries(ctx context.Context, connectionID string) ([]*TableCatalogEntry, error) {
	return s.store.ListTableEntries(ctx, connectionID)
}

// CreateColumnEntry creates a new column catalog entry
func (s *Service) CreateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error {
	return s.store.CreateColumnEntry(ctx, entry)
}

// GetColumnEntry retrieves a column catalog entry
func (s *Service) GetColumnEntry(ctx context.Context, tableID, column string) (*ColumnCatalogEntry, error) {
	return s.store.GetColumnEntry(ctx, tableID, column)
}

// UpdateColumnEntry updates a column catalog entry
func (s *Service) UpdateColumnEntry(ctx context.Context, entry *ColumnCatalogEntry) error {
	return s.store.UpdateColumnEntry(ctx, entry)
}

// ListColumnEntries lists all column entries for a table
func (s *Service) ListColumnEntries(ctx context.Context, tableID string) ([]*ColumnCatalogEntry, error) {
	return s.store.ListColumnEntries(ctx, tableID)
}

// CreateTag creates a new custom tag
func (s *Service) CreateTag(ctx context.Context, tag *CatalogTag) error {
	return s.store.CreateTag(ctx, tag)
}

// ListTags lists all tags for an organization
func (s *Service) ListTags(ctx context.Context, orgID *string) ([]*CatalogTag, error) {
	return s.store.ListTags(ctx, orgID)
}

// DeleteTag deletes a custom tag by ID
func (s *Service) DeleteTag(ctx context.Context, id string) error {
	return s.store.DeleteTag(ctx, id)
}

// SearchCatalog searches the catalog for tables and columns
func (s *Service) SearchCatalog(ctx context.Context, query string, filters *SearchFilters) (*SearchResults, error) {
	return s.store.SearchCatalog(ctx, query, filters)
}

// AssignSteward assigns a steward to a table by ID
func (s *Service) AssignSteward(ctx context.Context, tableID, userID string) error {
	// Get the table entry first, then update it
	tables, err := s.store.ListTableEntries(ctx, "")
	if err != nil {
		return err
	}

	for _, table := range tables {
		if table.ID == tableID {
			table.StewardUserID = &userID
			return s.store.UpdateTableEntry(ctx, table)
		}
	}
	return fmt.Errorf("table with ID %s not found", tableID)
}

// MarkColumnAsPII marks a column as containing PII
func (s *Service) MarkColumnAsPII(ctx context.Context, tableID, columnName, piiType string, confidence float64) error {
	entry, err := s.store.GetColumnEntry(ctx, tableID, columnName)
	if err != nil {
		return err
	}

	if entry == nil {
		// Create new entry with PII info
		entry = &ColumnCatalogEntry{
			TableCatalogID: tableID,
			ColumnName:     columnName,
			PIIType:        &piiType,
			PIIConfidence:  &confidence,
			Tags:           []string{},
		}
		return s.store.CreateColumnEntry(ctx, entry)
	}

	// Update existing entry
	entry.PIIType = &piiType
	entry.PIIConfidence = &confidence
	return s.store.UpdateColumnEntry(ctx, entry)
}

// UpdateTableDescription updates or creates a table catalog entry with description
func (s *Service) UpdateTableDescription(ctx context.Context, connID, schema, table, description, userID string) error {
	entry, err := s.store.GetTableEntry(ctx, connID, schema, table)
	if err != nil {
		return err
	}

	if entry != nil {
		// Update existing
		entry.Description = description
		return s.store.UpdateTableEntry(ctx, entry)
	}

	// Create new
	entry = &TableCatalogEntry{
		ConnectionID: connID,
		SchemaName:   schema,
		TableName:    table,
		Description:  description,
		Tags:         []string{},
		CreatedBy:    userID,
	}

	return s.store.CreateTableEntry(ctx, entry)
}

// UpdateColumnDescription updates or creates a column catalog entry with description
func (s *Service) UpdateColumnDescription(ctx context.Context, tableID, columnName, description string) error {
	entry, err := s.store.GetColumnEntry(ctx, tableID, columnName)
	if err != nil {
		return err
	}

	if entry != nil {
		// Update existing
		entry.Description = description
		return s.store.UpdateColumnEntry(ctx, entry)
	}

	// Create new
	entry = &ColumnCatalogEntry{
		TableCatalogID: tableID,
		ColumnName:     columnName,
		Description:    description,
		Tags:           []string{},
	}

	return s.store.CreateColumnEntry(ctx, entry)
}

// TagTable adds tags to a table (merges with existing tags)
func (s *Service) TagTable(ctx context.Context, connID, schema, table string, tags []string, userID string) error {
	entry, err := s.store.GetTableEntry(ctx, connID, schema, table)
	if err != nil {
		return err
	}

	if entry != nil {
		// Merge tags
		entry.Tags = mergeTags(entry.Tags, tags)
		return s.store.UpdateTableEntry(ctx, entry)
	}

	// Create new
	entry = &TableCatalogEntry{
		ConnectionID: connID,
		SchemaName:   schema,
		TableName:    table,
		Tags:         tags,
		CreatedBy:    userID,
	}

	return s.store.CreateTableEntry(ctx, entry)
}

// TagColumn adds tags to a column (merges with existing tags)
func (s *Service) TagColumn(ctx context.Context, tableID, columnName string, tags []string) error {
	entry, err := s.store.GetColumnEntry(ctx, tableID, columnName)
	if err != nil {
		return err
	}

	if entry != nil {
		// Merge tags
		entry.Tags = mergeTags(entry.Tags, tags)
		return s.store.UpdateColumnEntry(ctx, entry)
	}

	// Create new
	entry = &ColumnCatalogEntry{
		TableCatalogID: tableID,
		ColumnName:     columnName,
		Tags:           tags,
	}

	return s.store.CreateColumnEntry(ctx, entry)
}

// SetSteward assigns a steward to a table
func (s *Service) SetSteward(ctx context.Context, connID, schema, table, stewardUserID, userID string) error {
	entry, err := s.store.GetTableEntry(ctx, connID, schema, table)
	if err != nil {
		return err
	}

	if entry != nil {
		entry.StewardUserID = &stewardUserID
		return s.store.UpdateTableEntry(ctx, entry)
	}

	// Create new
	entry = &TableCatalogEntry{
		ConnectionID:  connID,
		SchemaName:    schema,
		TableName:     table,
		StewardUserID: &stewardUserID,
		Tags:          []string{},
		CreatedBy:     userID,
	}

	return s.store.CreateTableEntry(ctx, entry)
}

// Search performs catalog search (uses FTS5 if available, falls back to LIKE)
func (s *Service) Search(ctx context.Context, query string, filters *SearchFilters) (*SearchResults, error) {
	// Try FTS5 search first
	if sqliteStore, ok := s.store.(*SQLiteStore); ok {
		results, err := sqliteStore.SearchCatalogFTS5(ctx, query, filters)
		if err == nil {
			return results, nil
		}
		// Fall back to regular search if FTS5 fails
	}

	return s.store.SearchCatalog(ctx, query, filters)
}

// GetStats returns catalog coverage statistics
func (s *Service) GetStats(ctx context.Context, connectionID string) (*CatalogStats, error) {
	tables, err := s.store.ListTableEntries(ctx, connectionID)
	if err != nil {
		return nil, err
	}

	stats := &CatalogStats{
		TotalTables:      len(tables),
		DocumentedTables: 0,
		TotalColumns:     0,
		DocumentedCols:   0,
		TagDistribution:  make(map[string]int),
	}

	for _, table := range tables {
		if table.Description != "" {
			stats.DocumentedTables++
		}

		// Count tags
		for _, tag := range table.Tags {
			stats.TagDistribution[tag]++
		}

		// Get columns for this table
		columns, err := s.store.ListColumnEntries(ctx, table.ID)
		if err != nil {
			continue
		}

		stats.TotalColumns += len(columns)
		for _, col := range columns {
			if col.Description != "" {
				stats.DocumentedCols++
			}
			for _, tag := range col.Tags {
				stats.TagDistribution[tag]++
			}
		}
	}

	return stats, nil
}

// ExportCatalog exports catalog as JSON
func (s *Service) ExportCatalog(ctx context.Context, connectionID string) ([]byte, error) {
	tables, err := s.store.ListTableEntries(ctx, connectionID)
	if err != nil {
		return nil, err
	}

	// Load columns for each table
	for _, table := range tables {
		columns, err := s.store.ListColumnEntries(ctx, table.ID)
		if err != nil {
			return nil, err
		}
		table.Columns = columns
	}

	tags, err := s.store.ListTags(ctx, nil)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"version":     "1.0",
		"exported_at": time.Now().Format(time.RFC3339),
		"connection":  connectionID,
		"tables":      tables,
		"tags":        tags,
	}

	return json.MarshalIndent(export, "", "  ")
}

// ImportCatalog imports catalog from JSON
func (s *Service) ImportCatalog(ctx context.Context, data []byte) error {
	var importData struct {
		Version    string                `json:"version"`
		Connection string                `json:"connection"`
		Tables     []*TableCatalogEntry  `json:"tables"`
		Tags       []*CatalogTag         `json:"tags"`
		Columns    []*ColumnCatalogEntry `json:"columns,omitempty"`
	}

	if err := json.Unmarshal(data, &importData); err != nil {
		return fmt.Errorf("unmarshal import data: %w", err)
	}

	// Import custom tags
	for _, tag := range importData.Tags {
		if !tag.IsSystem {
			if err := s.store.CreateTag(ctx, tag); err != nil {
				// Ignore duplicates
				if !isDuplicateError(err) {
					return err
				}
			}
		}
	}

	// Import tables
	for _, table := range importData.Tables {
		if err := s.store.CreateTableEntry(ctx, table); err != nil {
			if !isDuplicateError(err) {
				return err
			}
		}

		// Import columns for this table
		if table.Columns != nil {
			for _, col := range table.Columns {
				if err := s.store.CreateColumnEntry(ctx, col); err != nil {
					if !isDuplicateError(err) {
						return err
					}
				}
			}
		}
	}

	return nil
}

// Helper types and functions

// CatalogStats provides catalog coverage statistics
type CatalogStats struct {
	TotalTables      int            `json:"total_tables"`
	DocumentedTables int            `json:"documented_tables"`
	TotalColumns     int            `json:"total_columns"`
	DocumentedCols   int            `json:"documented_columns"`
	TagDistribution  map[string]int `json:"tag_distribution"`
}

// TableCatalogEntry extension to include columns for export
type tableCatalogWithColumns struct {
	*TableCatalogEntry
	Columns []*ColumnCatalogEntry `json:"columns,omitempty"`
}

// mergeTags combines two tag lists, removing duplicates
func mergeTags(existing, new []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(existing)+len(new))

	for _, tag := range existing {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	for _, tag := range new {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}

	return result
}

// isDuplicateError checks if an error is a duplicate constraint error
func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "already exists") || contains(errStr, "UNIQUE constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findInString(s, substr))
}

func findInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
