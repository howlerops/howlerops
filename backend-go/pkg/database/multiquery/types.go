package multiquery

import (
	"context"
	"time"
)

// ExecutionStrategy defines how multi-database queries are executed
type ExecutionStrategy string

const (
	// StrategyAuto automatically selects the best strategy
	StrategyAuto ExecutionStrategy = "auto"
	// StrategyFederated executes queries separately and merges results in Go
	StrategyFederated ExecutionStrategy = "federated"
	// StrategyPushDown pushes the query to a database with federation support
	StrategyPushDown ExecutionStrategy = "push_down"
)

// Config holds configuration for multi-query operations
type Config struct {
	Enabled                bool              `yaml:"enabled"`
	MaxConcurrentConns     int               `yaml:"max_concurrent_connections"`
	DefaultStrategy        ExecutionStrategy `yaml:"default_strategy"`
	Timeout                time.Duration     `yaml:"timeout"`
	MaxResultRows          int               `yaml:"max_result_rows"`
	EnableCrossTypeQueries bool              `yaml:"enable_cross_type_queries"`
	BatchSize              int               `yaml:"batch_size"`
	MergeBufferSize        int               `yaml:"merge_buffer_size"`
	ParallelExecution      bool              `yaml:"parallel_execution"`
	RequireExplicitConns   bool              `yaml:"require_explicit_connections"`
	AllowedOperations      []string          `yaml:"allowed_operations"`
}

// Options holds options for executing a multi-query
type Options struct {
	Timeout  time.Duration
	Strategy ExecutionStrategy
	Limit    int
}

// ParsedQuery represents a parsed multi-database query
type ParsedQuery struct {
	OriginalSQL         string
	RequiredConnections []string
	Segments            []QuerySegment
	Tables              []string
	SuggestedStrategy   ExecutionStrategy
	HasJoins            bool
	HasAggregation      bool
}

// QuerySegment represents a query segment for a specific connection
type QuerySegment struct {
	ConnectionID string
	SQL          string
	Tables       []TableRef
	IsSubquery   bool
}

// TableRef represents a table reference with connection info
type TableRef struct {
	ConnectionID string
	Schema       string
	Table        string
	Alias        string
}

// Result represents the result of a multi-database query
type Result struct {
	Columns         []string
	Rows            [][]interface{}
	RowCount        int64
	Duration        time.Duration
	ConnectionsUsed []string
	Strategy        ExecutionStrategy
	Editable        *EditableQueryMetadata
}

// CombinedSchema represents schema information from multiple connections
type CombinedSchema struct {
	Connections map[string]*ConnectionSchema
	Conflicts   []SchemaConflict
}

// ConnectionSchema represents schema info for a single connection
type ConnectionSchema struct {
	ConnectionID string
	Schemas      []string
	Tables       []TableInfo
}

// SchemaConflict represents a table name conflict between connections
type SchemaConflict struct {
	TableName   string
	Connections []ConflictingTable
	Resolution  string
}

// ConflictingTable represents a table in a conflict
type ConflictingTable struct {
	ConnectionID string
	TableName    string
	Schema       string
}

// ConnectionRef represents a @connection reference in SQL
type ConnectionRef struct {
	Alias  string
	Schema string
	Table  string
	Line   int
	Column int
}

// Database is a minimal interface for database operations needed by multiquery
// This avoids import cycles with the main database package
type Database interface {
	Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error)
}

// QueryResult represents a minimal query result (avoiding import cycles)
type QueryResult struct {
	Columns  []string
	Rows     [][]interface{}
	RowCount int64
	Duration time.Duration
	Editable *EditableQueryMetadata
}

// TableInfo represents minimal table info (avoiding import cycles)
type TableInfo struct {
	Schema    string
	Name      string
	Type      string
	Comment   string
	RowCount  int64
	SizeBytes int64
}

// EditableQueryMetadata represents metadata for editable queries
type EditableQueryMetadata struct {
	Enabled    bool
	Reason     string
	Schema     string
	Table      string
	PrimaryKeys []string
	Columns    []EditableColumn
	Pending    bool
	JobID      string
}

// EditableColumn represents an editable column
type EditableColumn struct {
	Name        string
	ResultName  string
	DataType    string
	Editable    bool
	PrimaryKey  bool
	ForeignKey  *ForeignKeyRef
}

// ForeignKeyRef represents foreign key information
type ForeignKeyRef struct {
	Table  string
	Column string
	Schema string
}

