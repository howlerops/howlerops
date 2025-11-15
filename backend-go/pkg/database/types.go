package database

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// DatabaseType represents the type of database
type DatabaseType string

const (
	PostgreSQL    DatabaseType = "postgresql"
	MySQL         DatabaseType = "mysql"
	SQLite        DatabaseType = "sqlite"
	MariaDB       DatabaseType = "mariadb"
	Elasticsearch DatabaseType = "elasticsearch"
	OpenSearch    DatabaseType = "opensearch"
	ClickHouse    DatabaseType = "clickhouse"
	MongoDB       DatabaseType = "mongodb"
	TiDB          DatabaseType = "tidb"
)

var (
	// ErrDatabaseSwitchNotSupported indicates that a connector does not support runtime database switching.
	ErrDatabaseSwitchNotSupported = errors.New("database switching is not supported for this connection")
	// ErrDatabaseSwitchRequiresReconnect indicates that the connector must reconnect to switch databases.
	ErrDatabaseSwitchRequiresReconnect = errors.New("database switching requires a new connection")
)

// SSHAuthMethod represents the SSH authentication method
type SSHAuthMethod string

const (
	SSHAuthPassword   SSHAuthMethod = "password"
	SSHAuthPrivateKey SSHAuthMethod = "privatekey"
)

// SSHTunnelConfig holds SSH tunnel/bastion host configuration
type SSHTunnelConfig struct {
	Host                  string        `json:"host"`
	Port                  int           `json:"port"`
	User                  string        `json:"user"`
	AuthMethod            SSHAuthMethod `json:"auth_method"`
	Password              string        `json:"password,omitempty"`            // Deprecated: use SecretStore
	PrivateKey            string        `json:"private_key,omitempty"`         // Deprecated: use SecretStore
	PrivateKeyPath        string        `json:"private_key_path,omitempty"`    // Deprecated: use SecretStore
	PrivateKeyName        string        `json:"private_key_name,omitempty"`    // Reference to secret in SecretStore
	KnownHostsPath        string        `json:"known_hosts_path,omitempty"`    // Path to known_hosts file
	StrictHostKeyChecking bool          `json:"strict_host_key_checking"`      // Whether to verify host key
	Timeout               time.Duration `json:"timeout,omitempty"`             // Connection timeout
	KeepAliveInterval     time.Duration `json:"keep_alive_interval,omitempty"` // Keep-alive interval
	ConnectionID          string        `json:"connection_id,omitempty"`       // ID for loading secrets
}

// VPCConfig holds VPC/Private Link configuration for cloud providers
type VPCConfig struct {
	Provider   string            `json:"provider"`    // aws, gcp, azure
	EndpointID string            `json:"endpoint_id"` // VPC endpoint ID
	PrivateDNS string            `json:"private_dns"` // Private DNS name
	Parameters map[string]string `json:"parameters"`  // Provider-specific params
}

// ConnectionConfig holds database connection configuration
type ConnectionConfig struct {
	ID                string            `json:"id,omitempty"` // Optional stored connection ID for reconnecting
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

	// SSH Tunnel / Bastion Host configuration
	UseTunnel bool             `json:"use_tunnel"`
	SSHTunnel *SSHTunnelConfig `json:"ssh_tunnel,omitempty"`

	// VPC / Private Link configuration
	UseVPC    bool       `json:"use_vpc"`
	VPCConfig *VPCConfig `json:"vpc_config,omitempty"`
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
	RowCount int64                  `json:"row_count"` // Total rows in unpaginated query
	Affected int64                  `json:"affected"`
	Duration time.Duration          `json:"duration"`
	Error    error                  `json:"error,omitempty"`
	Editable *EditableQueryMetadata `json:"editable,omitempty"`
	// Pagination metadata
	TotalRows int64 `json:"total_rows,omitempty"` // NEW: Total rows available (for pagination)
	PagedRows int64 `json:"paged_rows,omitempty"` // NEW: Rows in this page
	HasMore   bool  `json:"has_more,omitempty"`   // NEW: More data available
	Offset    int   `json:"offset,omitempty"`     // NEW: Current offset
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
	ExecuteWithOptions(ctx context.Context, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error)
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
	InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error)
	DeleteRow(ctx context.Context, params DeleteRowParams) error

	// Database selection helpers
	ListDatabases(ctx context.Context) ([]string, error)
	SwitchDatabase(ctx context.Context, databaseName string) error

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
	Offset        int           `json:"offset"` // NEW: Pagination offset
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
	Name       string         `json:"name"`
	ResultName string         `json:"result_name"`
	DataType   string         `json:"data_type"`
	Editable   bool           `json:"editable"`
	PrimaryKey bool           `json:"primary_key"`
	ForeignKey *ForeignKeyRef `json:"foreign_key,omitempty"`
	HasDefault bool           `json:"has_default,omitempty"`
	DefaultVal interface{}    `json:"default_value,omitempty"`
	DefaultExp string         `json:"default_expression,omitempty"`
	AutoNumber bool           `json:"auto_number,omitempty"`
	TimeZone   bool           `json:"time_zone,omitempty"`
	Precision  *int           `json:"precision,omitempty"`
}

// ForeignKeyRef represents a foreign key reference for a column
type ForeignKeyRef struct {
	Table  string `json:"table"`
	Column string `json:"column"`
	Schema string `json:"schema,omitempty"`
}

// EditableQueryMetadata captures whether a query result set supports direct editing
type EditableQueryMetadata struct {
	Enabled      bool                  `json:"enabled"`
	Reason       string                `json:"reason,omitempty"`
	Schema       string                `json:"schema,omitempty"`
	Table        string                `json:"table,omitempty"`
	PrimaryKeys  []string              `json:"primary_keys,omitempty"`
	Columns      []EditableColumn      `json:"columns,omitempty"`
	Pending      bool                  `json:"pending,omitempty"`
	JobID        string                `json:"job_id,omitempty"`
	Capabilities *MutationCapabilities `json:"capabilities,omitempty"`
}

// MutationCapabilities describes which row-level operations are supported.
type MutationCapabilities struct {
	CanInsert bool   `json:"can_insert"`
	CanUpdate bool   `json:"can_update"`
	CanDelete bool   `json:"can_delete"`
	Reason    string `json:"reason,omitempty"`
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

// InsertRowParams describes the data required to insert a new row.
type InsertRowParams struct {
	Schema        string                 `json:"schema"`
	Table         string                 `json:"table"`
	Values        map[string]interface{} `json:"values"`
	OriginalQuery string                 `json:"original_query"`
	Columns       []string               `json:"columns"`
}

// DeleteRowParams describes the data required to delete an existing row.
type DeleteRowParams struct {
	Schema        string                 `json:"schema"`
	Table         string                 `json:"table"`
	PrimaryKey    map[string]interface{} `json:"primary_key"`
	OriginalQuery string                 `json:"original_query"`
	Columns       []string               `json:"columns"`
}
