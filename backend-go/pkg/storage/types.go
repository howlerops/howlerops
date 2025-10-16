package storage

import (
	"time"
)

// Connection represents a database connection
type Connection struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Type              string            `json:"type"`
	Host              string            `json:"host,omitempty"`
	Port              int               `json:"port,omitempty"`
	DatabaseName      string            `json:"database_name,omitempty"`
	Username          string            `json:"username,omitempty"`
	PasswordEncrypted string            `json:"password_encrypted,omitempty"`
	SSLConfig         string            `json:"ssl_config,omitempty"`
	CreatedBy         string            `json:"created_by"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	TeamID            string            `json:"team_id,omitempty"`
	IsShared          bool              `json:"is_shared"`
	Environments      []string          `json:"environments,omitempty"` // Environment tags like "local", "dev", "prod"
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// MySQLVectorConfig represents MySQL vector store connection details
type MySQLVectorConfig struct {
    DSN        string `json:"dsn"`
    VectorSize int    `json:"vector_size"`
}

// ConnectionFilters represents filters for connection queries
type ConnectionFilters struct {
	TeamID       string
	CreatedBy    string
	Type         string
	IsShared     *bool
	Environments []string // Filter by environment tags
	Limit        int
	Offset       int
}

// SavedQuery represents a saved SQL query
type SavedQuery struct {
	ID           string            `json:"id"`
	Title        string            `json:"title"`
	Query        string            `json:"query"`
	Description  string            `json:"description,omitempty"`
	ConnectionID string            `json:"connection_id,omitempty"`
	CreatedBy    string            `json:"created_by"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	TeamID       string            `json:"team_id,omitempty"`
	IsShared     bool              `json:"is_shared"`
	Tags         []string          `json:"tags,omitempty"`
	Folder       string            `json:"folder,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// QueryFilters represents filters for saved query searches
type QueryFilters struct {
	TeamID       string
	CreatedBy    string
	ConnectionID string
	Folder       string
	Tags         []string
	IsShared     *bool
	Limit        int
	Offset       int
}

// QueryHistory represents a query execution record
type QueryHistory struct {
	ID           string    `json:"id"`
	Query        string    `json:"query"`
	ConnectionID string    `json:"connection_id"`
	ExecutedBy   string    `json:"executed_by"`
	ExecutedAt   time.Time `json:"executed_at"`
	DurationMS   int64     `json:"duration_ms"`
	RowsReturned int64     `json:"rows_returned"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
	TeamID       string    `json:"team_id,omitempty"`
	IsShared     bool      `json:"is_shared"`
}

// HistoryFilters represents filters for query history searches
type HistoryFilters struct {
	TeamID       string
	ExecutedBy   string
	ConnectionID string
	Success      *bool
	StartDate    *time.Time
	EndDate      *time.Time
	Limit        int
	Offset       int
}

// DocumentFilters represents filters for document searches
type DocumentFilters struct {
	ConnectionID string
	Type         string
	TeamID       string
	Limit        int
}

// SchemaCache represents cached schema information
type SchemaCache struct {
	ConnectionID string                 `json:"connection_id"`
	SchemaData   map[string]interface{} `json:"schema_data"`
	CachedAt     time.Time              `json:"cached_at"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// Team represents a team
type Team struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
}

// TeamMember represents a team member
type TeamMember struct {
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // admin, member, viewer
	JoinedAt  time.Time `json:"joined_at"`
	InvitedBy string    `json:"invited_by"`
}

// Mode represents the storage mode
type Mode int

const (
	ModeSolo Mode = iota
	ModeTeam
)

// String returns string representation of Mode
func (m Mode) String() string {
	switch m {
	case ModeSolo:
		return "solo"
	case ModeTeam:
		return "team"
	default:
		return "unknown"
	}
}
