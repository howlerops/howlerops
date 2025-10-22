package storage

import (
	"context"
	"time"

	"github.com/sql-studio/backend-go/pkg/crypto"
)

// Redeclare Document locally to avoid import cycles
type Document struct {
	ID           string
	ConnectionID string
	Type         string
	Content      string
	Embedding    []float32
	Metadata     map[string]interface{}
	CreatedAt    time.Time
	UpdatedAt    time.Time
	AccessCount  int
	LastAccessed time.Time
	Score        float32
}

// Storage interface for all app data (DRY for local and team modes)
type Storage interface {
	// Connection management
	SaveConnection(ctx context.Context, conn *Connection) error
	GetConnections(ctx context.Context, filters *ConnectionFilters) ([]*Connection, error)
	GetConnection(ctx context.Context, id string) (*Connection, error)
	UpdateConnection(ctx context.Context, conn *Connection) error
	DeleteConnection(ctx context.Context, id string) error
	GetAvailableEnvironments(ctx context.Context) ([]string, error)
	
	// Query management
	SaveQuery(ctx context.Context, query *SavedQuery) error
	GetQueries(ctx context.Context, filters *QueryFilters) ([]*SavedQuery, error)
	GetQuery(ctx context.Context, id string) (*SavedQuery, error)
	UpdateQuery(ctx context.Context, query *SavedQuery) error
	DeleteQuery(ctx context.Context, id string) error
	
	// Query history
	SaveQueryHistory(ctx context.Context, history *QueryHistory) error
	GetQueryHistory(ctx context.Context, filters *HistoryFilters) ([]*QueryHistory, error)
	DeleteQueryHistory(ctx context.Context, id string) error
	
	// Vector/RAG operations
	IndexDocument(ctx context.Context, doc *Document) error
	SearchDocuments(ctx context.Context, embedding []float32, filters *DocumentFilters) ([]*Document, error)
	GetDocument(ctx context.Context, id string) (*Document, error)
	DeleteDocument(ctx context.Context, id string) error
	
	// Schema cache
	CacheSchema(ctx context.Context, connID string, schema *SchemaCache) error
	GetCachedSchema(ctx context.Context, connID string) (*SchemaCache, error)
	InvalidateSchemaCache(ctx context.Context, connID string) error
	
	// Settings
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	DeleteSetting(ctx context.Context, key string) error
	
	// Team operations (no-op in local mode)
	GetTeam(ctx context.Context) (*Team, error)
	GetTeamMembers(ctx context.Context) ([]*TeamMember, error)
	
	// Secret management
	GetSecretStore() crypto.SecretStore
	
	// Lifecycle
	Close() error
	
	// Mode information
	GetMode() Mode
	GetUserID() string
}

