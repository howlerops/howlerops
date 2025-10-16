package database

import (
	"context"
	"database/sql"
	"time"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	PostgreSQL DatabaseType = "postgresql"
	MySQL      DatabaseType = "mysql"
	SQLite     DatabaseType = "sqlite"
	MariaDB    DatabaseType = "mariadb"
)

// ConnectionConfig holds database connection configuration
type ConnectionConfig struct {
	Type              DatabaseType      `json:"type" validate:"required"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	Database          string            `json:"database" validate:"required"`
	Username          string            `json:"username"`
	Password          string            `json:"password"`
	SSLMode           string            `json:"ssl_mode"`
	ConnectionTimeout time.Duration     `json:"connection_timeout"`
	IdleTimeout       time.Duration     `json:"idle_timeout"`
	MaxConnections    int               `json:"max_connections"`
	MaxIdleConns      int               `json:"max_idle_connections"`
	Parameters        map[string]string `json:"parameters"`
}

// Connection represents a database connection with metadata
type Connection struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Config       ConnectionConfig  `json:"config"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Active       bool              `json:"active"`
	CreatedBy    string            `json:"created_by"`
	Tags         map[string]string `json:"tags"`
	Environments []string          `json:"environments,omitempty"` // Environment tags like "local", "dev", "prod"
}

// Pool represents a connection pool interface
type Pool interface {
	Get(ctx context.Context) (*sql.DB, error)
	Close() error
	Stats() PoolStats
	Ping(ctx context.Context) error
}

// PoolStats contains connection pool statistics
type PoolStats struct {
	OpenConnections   int           `json:"open_connections"`
	InUse             int           `json:"in_use"`
	Idle              int           `json:"idle"`
	WaitCount         int64         `json:"wait_count"`
	WaitDuration      time.Duration `json:"wait_duration"`
	MaxIdleClosed     int64         `json:"max_idle_closed"`
	MaxIdleTimeClosed int64         `json:"max_idle_time_closed"`
	MaxLifetimeClosed int64         `json:"max_lifetime_closed"`
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Columns  []string               `json:"columns"`
	Rows     [][]interface{}        `json:"rows"`
	RowCount int64                  `json:"row_count"`
	Affected int64                  `json:"affected"`
	Duration time.Duration          `json:"duration"`
	Error    error                  `json:"error,omitempty"`
	Editable *EditableQueryMetadata `json:"editable,omitempty"`
}

// TableInfo represents metadata about a database table
type TableInfo struct {
	Schema    string            `json:"schema"`
	Name      string            `json:"name"`
	Type      string            `json:"type"`
	Comment   string            `json:"comment"`
	CreatedAt *time.Time        `json:"created_at"`
	UpdatedAt *time.Time        `json:"updated_at"`
	RowCount  int64             `json:"row_count"`
	SizeBytes int64             `json:"size_bytes"`
	Owner     string            `json:"owner"`
	Metadata  map[string]string `json:"metadata"`
}

// ColumnInfo represents metadata about a table column
type ColumnInfo struct {
	Name               string            `json:"name"`
	DataType           string            `json:"data_type"`
	Nullable           bool              `json:"nullable"`
	DefaultValue       *string           `json:"default_value"`
	PrimaryKey         bool              `json:"primary_key"`
	Unique             bool              `json:"unique"`
	Indexed            bool              `json:"indexed"`
	Comment            string            `json:"comment"`
	OrdinalPosition    int               `json:"ordinal_position"`
	CharacterMaxLength *int64            `json:"character_maximum_length"`
	NumericPrecision   *int              `json:"numeric_precision"`
	NumericScale       *int              `json:"numeric_scale"`
	Metadata           map[string]string `json:"metadata"`
}

// IndexInfo represents metadata about a database index
type IndexInfo struct {
	Name     string            `json:"name"`
	Columns  []string          `json:"columns"`
	Unique   bool              `json:"unique"`
	Primary  bool              `json:"primary"`
	Type     string            `json:"type"`
	Method   string            `json:"method"`
	Metadata map[string]string `json:"metadata"`
}

// ForeignKeyInfo represents metadata about a foreign key constraint
type ForeignKeyInfo struct {
	Name              string   `json:"name"`
	Columns           []string `json:"columns"`
	ReferencedTable   string   `json:"referenced_table"`
	ReferencedSchema  string   `json:"referenced_schema"`
	ReferencedColumns []string `json:"referenced_columns"`
	OnDelete          string   `json:"on_delete"`
	OnUpdate          string   `json:"on_update"`
}

// Database interface defines the contract for database operations
type Database interface {
	// Connection management
	Connect(ctx context.Context, config ConnectionConfig) error
	Disconnect() error
	Ping(ctx context.Context) error
	GetConnectionInfo(ctx context.Context) (map[string]interface{}, error)

	// Query execution
	Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error)
	ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error
	ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error)
	ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error)

	// Schema operations
	GetSchemas(ctx context.Context) ([]string, error)
	GetTables(ctx context.Context, schema string) ([]TableInfo, error)
	GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error)

	// Transaction management
	BeginTransaction(ctx context.Context) (Transaction, error)

	// Data modification helpers
	UpdateRow(ctx context.Context, params UpdateRowParams) error

	// Utility methods
	GetDatabaseType() DatabaseType
	GetConnectionStats() PoolStats
	QuoteIdentifier(identifier string) string
	GetDataTypeMappings() map[string]string
}

// Transaction interface for database transactions
type Transaction interface {
	Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error)
	Commit() error
	Rollback() error
}

// TableStructure represents the complete structure of a table
type TableStructure struct {
	Table       TableInfo         `json:"table"`
	Columns     []ColumnInfo      `json:"columns"`
	Indexes     []IndexInfo       `json:"indexes"`
	ForeignKeys []ForeignKeyInfo  `json:"foreign_keys"`
	Triggers    []string          `json:"triggers"`
	Statistics  map[string]string `json:"statistics"`
}

// QueryOptions contains options for query execution
type QueryOptions struct {
	Timeout       time.Duration `json:"timeout"`
	ReadOnly      bool          `json:"read_only"`
	Limit         int           `json:"limit"`
	IncludeSchema bool          `json:"include_schema"`
	StreamingMode bool          `json:"streaming_mode"`
	BatchSize     int           `json:"batch_size"`
}

// StreamCallback is a function type for handling streaming query results
type StreamCallback func(rows [][]interface{}) error

// HealthStatus represents the health of a database connection
type HealthStatus struct {
	Status       string            `json:"status"`
	Message      string            `json:"message"`
	Timestamp    time.Time         `json:"timestamp"`
	ResponseTime time.Duration     `json:"response_time"`
	Metrics      map[string]string `json:"metrics"`
}

// EditableColumn describes a column returned in a query and whether it is editable
type EditableColumn struct {
	Name       string `json:"name"`
	ResultName string `json:"result_name"`
	DataType   string `json:"data_type"`
	Editable   bool   `json:"editable"`
	PrimaryKey bool   `json:"primary_key"`
}

// EditableQueryMetadata captures whether a query result set supports direct editing
type EditableQueryMetadata struct {
	Enabled     bool             `json:"enabled"`
	Reason      string           `json:"reason,omitempty"`
	Schema      string           `json:"schema,omitempty"`
	Table       string           `json:"table,omitempty"`
	PrimaryKeys []string         `json:"primary_keys,omitempty"`
	Columns     []EditableColumn `json:"columns,omitempty"`
	Pending     bool             `json:"pending,omitempty"`
	JobID       string           `json:"job_id,omitempty"`
}

// UpdateRowParams describes the data required to persist edits back to the source table
type UpdateRowParams struct {
	Schema        string                 `json:"schema"`
	Table         string                 `json:"table"`
	PrimaryKey    map[string]interface{} `json:"primary_key"`
	Values        map[string]interface{} `json:"values"`
	OriginalQuery string                 `json:"original_query"`
	Columns       []string               `json:"columns"`
}
