package schemadiff

import (
	"time"

	"github.com/jbeck018/howlerops/backend-go/pkg/database"
)

// DiffStatus represents the status of a schema object in a diff
type DiffStatus string

const (
	DiffAdded     DiffStatus = "added"
	DiffRemoved   DiffStatus = "removed"
	DiffModified  DiffStatus = "modified"
	DiffUnchanged DiffStatus = "unchanged"
)

// SchemaDiff represents the complete diff between two database schemas
type SchemaDiff struct {
	SourceID  string        `json:"source_id"`
	TargetID  string        `json:"target_id"`
	Timestamp time.Time     `json:"timestamp"`
	Summary   DiffSummary   `json:"summary"`
	Tables    []TableDiff   `json:"tables"`
	Duration  time.Duration `json:"duration"`
}

// DiffSummary provides aggregate statistics about the diff
type DiffSummary struct {
	TablesAdded     int `json:"tables_added"`
	TablesRemoved   int `json:"tables_removed"`
	TablesModified  int `json:"tables_modified"`
	ColumnsAdded    int `json:"columns_added"`
	ColumnsRemoved  int `json:"columns_removed"`
	ColumnsModified int `json:"columns_modified"`
	IndexesChanged  int `json:"indexes_changed"`
	FKsChanged      int `json:"fks_changed"`
}

// TableDiff represents the diff for a single table
type TableDiff struct {
	Schema      string       `json:"schema"`
	Name        string       `json:"name"`
	Status      DiffStatus   `json:"status"`
	Columns     []ColumnDiff `json:"columns,omitempty"`
	Indexes     []IndexDiff  `json:"indexes,omitempty"`
	ForeignKeys []FKDiff     `json:"foreign_keys,omitempty"`
}

// ColumnDiff represents the diff for a single column
type ColumnDiff struct {
	Name       string     `json:"name"`
	Status     DiffStatus `json:"status"`
	OldType    string     `json:"old_type,omitempty"`
	NewType    string     `json:"new_type,omitempty"`
	OldNull    *bool      `json:"old_nullable,omitempty"`
	NewNull    *bool      `json:"new_nullable,omitempty"`
	OldDefault *string    `json:"old_default,omitempty"`
	NewDefault *string    `json:"new_default,omitempty"`
}

// IndexDiff represents the diff for a single index
type IndexDiff struct {
	Name       string     `json:"name"`
	Status     DiffStatus `json:"status"`
	OldColumns []string   `json:"old_columns,omitempty"`
	NewColumns []string   `json:"new_columns,omitempty"`
	OldUnique  *bool      `json:"old_unique,omitempty"`
	NewUnique  *bool      `json:"new_unique,omitempty"`
	OldMethod  string     `json:"old_method,omitempty"`
	NewMethod  string     `json:"new_method,omitempty"`
}

// FKDiff represents the diff for a single foreign key
type FKDiff struct {
	Name          string     `json:"name"`
	Status        DiffStatus `json:"status"`
	OldColumns    []string   `json:"old_columns,omitempty"`
	NewColumns    []string   `json:"new_columns,omitempty"`
	OldRefTable   string     `json:"old_ref_table,omitempty"`
	NewRefTable   string     `json:"new_ref_table,omitempty"`
	OldRefColumns []string   `json:"old_ref_columns,omitempty"`
	NewRefColumns []string   `json:"new_ref_columns,omitempty"`
	OldOnDelete   string     `json:"old_on_delete,omitempty"`
	NewOnDelete   string     `json:"new_on_delete,omitempty"`
	OldOnUpdate   string     `json:"old_on_update,omitempty"`
	NewOnUpdate   string     `json:"new_on_update,omitempty"`
}

// SchemaSnapshot represents a saved point-in-time snapshot of a database schema
type SchemaSnapshot struct {
	ID           string                              `json:"id"`
	Name         string                              `json:"name"`
	ConnectionID string                              `json:"connection_id"`
	DatabaseType database.DatabaseType               `json:"database_type"`
	Schemas      []string                            `json:"schemas"`
	Tables       map[string][]database.TableInfo     `json:"tables"`
	Structures   map[string]*database.TableStructure `json:"structures"`
	CreatedAt    time.Time                           `json:"created_at"`
	Hash         string                              `json:"hash"`
}

// SnapshotMetadata provides lightweight snapshot info for listing
type SnapshotMetadata struct {
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	ConnectionID string                `json:"connection_id"`
	DatabaseType database.DatabaseType `json:"database_type"`
	TableCount   int                   `json:"table_count"`
	CreatedAt    time.Time             `json:"created_at"`
	SizeBytes    int64                 `json:"size_bytes"`
}
