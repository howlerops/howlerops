package storage

import "time"

// Connection represents a database connection configuration
type Connection struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	Host              string            `json:"host"`
	Port              int               `json:"port"`
	Database          string            `json:"database"`
	DatabaseName      string            `json:"database_name"` // Alias for Database
	Username          string            `json:"username"`
	Password          string            `json:"password,omitempty"`
	PasswordEncrypted string            `json:"password_encrypted,omitempty"` // Encrypted password
	Environment       string            `json:"environment"`
	Environments      []string          `json:"environments"` // Multiple environments this connection supports
	SSLConfig         map[string]string `json:"ssl_config,omitempty"`
	Metadata          map[string]string `json:"metadata"`
	TeamID            string            `json:"team_id,omitempty"`
	CreatedBy         string            `json:"created_by,omitempty"`
	IsShared          bool              `json:"is_shared"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

// ConnectionFilters defines filters for querying connections
type ConnectionFilters struct {
	Environment  string
	Environments []string
	Type         string
	TeamID       string
	CreatedBy    string
	IsShared     *bool
	Limit        int
	Offset       int
}

// SavedQuery represents a saved SQL query
type SavedQuery struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Title        string            `json:"title"` // Alias for Name
	Description  string            `json:"description"`
	Query        string            `json:"query"`
	ConnectionID string            `json:"connection_id"`
	Folder       string            `json:"folder"`
	Tags         []string          `json:"tags"`
	Metadata     map[string]string `json:"metadata"`
	TeamID       string            `json:"team_id,omitempty"`
	CreatedBy    string            `json:"created_by,omitempty"`
	IsShared     bool              `json:"is_shared"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// QueryFilters defines filters for querying saved queries
type QueryFilters struct {
	ConnectionID string
	Folder       string
	Tags         []string
	TeamID       string
	CreatedBy    string
	IsShared     *bool
	Limit        int
	Offset       int
}

// QueryHistory represents a record of an executed query
type QueryHistory struct {
	ID           string    `json:"id"`
	Query        string    `json:"query"`
	ConnectionID string    `json:"connection_id"`
	ExecutedAt   time.Time `json:"executed_at"`
	Duration     int       `json:"duration_ms"`
	DurationMS   int       `json:"duration"` // Alias for Duration
	RowsAffected int       `json:"rows_affected"`
	RowsReturned int       `json:"rows_returned"` // Alias for RowsAffected
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	ExecutedBy   string    `json:"executed_by,omitempty"`
	TeamID       string    `json:"team_id,omitempty"`
	IsShared     bool      `json:"is_shared"`
}

// HistoryFilters defines filters for querying query history
type HistoryFilters struct {
	ConnectionID string
	Success      *bool
	StartDate    *time.Time
	EndDate      *time.Time
	TeamID       string
	ExecutedBy   string
	Limit        int
	Offset       int
}

// DocumentFilters defines filters for document search
type DocumentFilters struct {
	ConnectionID string
	Type         string
	Limit        int
}

// SchemaCache represents cached database schema information
type SchemaCache struct {
	ConnectionID string                 `json:"connection_id"`
	Schema       map[string]interface{} `json:"schema"`
	SchemaData   string                 `json:"schema_data"` // JSON-encoded schema
	CachedAt     time.Time              `json:"cached_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// Team represents a team in team mode
type Team struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// TeamMember represents a member of a team
type TeamMember struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"`
	JoinedAt  time.Time `json:"joined_at"`
}

// Mode represents the storage mode (local or team)
type Mode string

const (
	ModeLocal Mode = "local"
	ModeTeam  Mode = "team"
	ModeSolo  Mode = "solo" // Alias for ModeLocal for backward compatibility
)

// MySQLVectorConfig represents MySQL vector configuration
type MySQLVectorConfig struct {
	Host       string
	Port       int
	Database   string
	Username   string
	Password   string
	DSN        string // Data Source Name (connection string)
	VectorSize int    // Dimension of vector embeddings
}
