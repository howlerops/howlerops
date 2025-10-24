package sync

import (
	"time"
)

// Time is a wrapper around time.Time for JSON marshaling
type Time struct {
	time.Time
}

// UnmarshalText implements the encoding.TextUnmarshaler interface
func (t *Time) UnmarshalText(data []byte) error {
	parsed, err := time.Parse(time.RFC3339, string(data))
	if err != nil {
		return err
	}
	t.Time = parsed
	return nil
}

// MarshalText implements the encoding.TextMarshaler interface
func (t Time) MarshalText() ([]byte, error) {
	return []byte(t.Format(time.RFC3339)), nil
}

// SyncItemType represents the type of item being synced
type SyncItemType string

const (
	SyncItemTypeConnection  SyncItemType = "connection"
	SyncItemTypeSavedQuery  SyncItemType = "saved_query"
	SyncItemTypeQueryHistory SyncItemType = "query_history"
)

// SyncAction represents the action performed on an item
type SyncAction string

const (
	SyncActionCreate SyncAction = "create"
	SyncActionUpdate SyncAction = "update"
	SyncActionDelete SyncAction = "delete"
)

// ConflictResolutionStrategy defines how conflicts should be resolved
type ConflictResolutionStrategy string

const (
	// Last write wins - use the most recently updated version
	ConflictResolutionLastWriteWins ConflictResolutionStrategy = "last_write_wins"

	// Keep both versions - create a copy for the conflicting version
	ConflictResolutionKeepBoth ConflictResolutionStrategy = "keep_both"

	// User choice required - flag for manual resolution
	ConflictResolutionUserChoice ConflictResolutionStrategy = "user_choice"
)

// ConnectionTemplate represents a database connection configuration
type ConnectionTemplate struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"` // mysql, postgres, sqlite, etc.
	Host           string            `json:"host,omitempty"`
	Port           int               `json:"port,omitempty"`
	Database       string            `json:"database"`
	Username       string            `json:"username,omitempty"`
	Password       string            `json:"-"` // Never sync passwords
	UseSSH         bool              `json:"use_ssh,omitempty"`
	SSHHost        string            `json:"ssh_host,omitempty"`
	SSHPort        int               `json:"ssh_port,omitempty"`
	SSHUser        string            `json:"ssh_user,omitempty"`
	Color          string            `json:"color,omitempty"`
	Icon           string            `json:"icon,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	SyncVersion    int               `json:"sync_version"`

	// Organization fields
	UserID         string  `json:"user_id"`                    // Owner of the connection
	OrganizationID *string `json:"organization_id,omitempty"` // NULL for personal, set for shared
	Visibility     string  `json:"visibility"`                 // "personal" or "shared"
}

// SavedQuery represents a user's saved query
type SavedQuery struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Description    string            `json:"description,omitempty"`
	Query          string            `json:"query"`
	ConnectionID   string            `json:"connection_id,omitempty"`
	Tags           []string          `json:"tags,omitempty"`
	Favorite       bool              `json:"favorite"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	SyncVersion    int               `json:"sync_version"`

	// Organization fields
	UserID         string  `json:"user_id"`                    // Owner of the query
	OrganizationID *string `json:"organization_id,omitempty"` // NULL for personal, set for shared
	Visibility     string  `json:"visibility"`                 // "personal" or "shared"
}

// QueryHistory represents query execution history
type QueryHistory struct {
	ID           string            `json:"id"`
	Query        string            `json:"query"`
	ConnectionID string            `json:"connection_id"`
	ExecutedAt   time.Time         `json:"executed_at"`
	Duration     int64             `json:"duration_ms"` // milliseconds
	RowsAffected int64             `json:"rows_affected"`
	Success      bool              `json:"success"`
	Error        string            `json:"error,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	SyncVersion  int               `json:"sync_version"`
}

// SyncChange represents a change to be synced
type SyncChange struct {
	ID          string       `json:"id"`
	ItemType    SyncItemType `json:"item_type"`
	ItemID      string       `json:"item_id"`
	Action      SyncAction   `json:"action"`
	Data        interface{}  `json:"data"` // ConnectionTemplate, SavedQuery, or QueryHistory
	UpdatedAt   time.Time    `json:"updated_at"`
	SyncVersion int          `json:"sync_version"`
	DeviceID    string       `json:"device_id"`
	Checksum    string       `json:"checksum,omitempty"` // For data integrity
}

// SyncUploadRequest represents a request to upload local changes
type SyncUploadRequest struct {
	UserID     string       `json:"user_id"`
	DeviceID   string       `json:"device_id"`
	LastSyncAt time.Time    `json:"last_sync_at"`
	Changes    []SyncChange `json:"changes"`
}

// SyncUploadResponse represents the response after uploading changes
type SyncUploadResponse struct {
	Success    bool               `json:"success"`
	SyncedAt   time.Time          `json:"synced_at"`
	Conflicts  []Conflict         `json:"conflicts,omitempty"`
	Rejected   []RejectedChange   `json:"rejected,omitempty"`
	Message    string             `json:"message,omitempty"`
}

// SyncDownloadRequest represents a request to download remote changes
type SyncDownloadRequest struct {
	UserID   string    `json:"user_id"`
	DeviceID string    `json:"device_id"`
	Since    time.Time `json:"since"`
}

// SyncDownloadResponse represents the response with remote changes
type SyncDownloadResponse struct {
	Connections   []ConnectionTemplate `json:"connections"`
	SavedQueries  []SavedQuery         `json:"saved_queries"`
	QueryHistory  []QueryHistory       `json:"query_history"`
	Conflicts     []ConflictInfo       `json:"conflicts,omitempty"`
	SyncTimestamp time.Time            `json:"sync_timestamp"`
	HasMore       bool                 `json:"has_more"`
}

// ConflictInfo represents conflict metadata in sync response
type ConflictInfo struct {
	ResourceType string           `json:"resource_type"` // "connection" | "query"
	ResourceID   string           `json:"resource_id"`
	Metadata     ConflictMetadata `json:"metadata"`
}

// ConflictMetadata provides details about a conflict resolution
type ConflictMetadata struct {
	Resolution    string    `json:"resolution"`     // "client_wins" | "server_wins" | "manual_required"
	Reason        string    `json:"reason"`         // Human-readable reason
	ServerVersion int       `json:"server_version"` // Server's sync version
	ClientVersion int       `json:"client_version"` // Client's sync version
	ConflictedAt  time.Time `json:"conflicted_at"`  // When the conflict was detected
}

// Conflict represents a sync conflict
type Conflict struct {
	ID            string          `json:"id"`
	ItemType      SyncItemType    `json:"item_type"`
	ItemID        string          `json:"item_id"`
	LocalVersion  *ConflictVersion `json:"local_version"`
	RemoteVersion *ConflictVersion `json:"remote_version"`
	DetectedAt    time.Time       `json:"detected_at"`
	ResolvedAt    *time.Time      `json:"resolved_at,omitempty"`
	Resolution    ConflictResolutionStrategy `json:"resolution,omitempty"`
}

// ConflictVersion represents a version of data in a conflict
type ConflictVersion struct {
	Data        interface{} `json:"data"`
	UpdatedAt   time.Time   `json:"updated_at"`
	SyncVersion int         `json:"sync_version"`
	DeviceID    string      `json:"device_id"`
}

// RejectedChange represents a change that was rejected during sync
type RejectedChange struct {
	Change SyncChange `json:"change"`
	Reason string     `json:"reason"`
}

// ConflictResolutionRequest represents a request to resolve a conflict
type ConflictResolutionRequest struct {
	ConflictID string                     `json:"conflict_id"`
	Strategy   ConflictResolutionStrategy `json:"strategy"`
	ChosenVersion string                  `json:"chosen_version,omitempty"` // "local" or "remote"
}

// ConflictResolutionResponse represents the response after resolving a conflict
type ConflictResolutionResponse struct {
	Success    bool      `json:"success"`
	ResolvedAt time.Time `json:"resolved_at"`
	Message    string    `json:"message,omitempty"`
}

// SyncMetadata represents metadata about the sync process
type SyncMetadata struct {
	UserID         string    `json:"user_id"`
	DeviceID       string    `json:"device_id"`
	LastSyncAt     time.Time `json:"last_sync_at"`
	TotalSynced    int64     `json:"total_synced"`
	ConflictsCount int       `json:"conflicts_count"`
	Version        string    `json:"version"`
}
