package rag

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/sirupsen/logrus"
)

// DocumentType represents the type of document being embedded
type DocumentType string

const (
	DocumentTypeSchema      DocumentType = "schema"
	DocumentTypeQuery       DocumentType = "query"
	DocumentTypePlan        DocumentType = "plan"
	DocumentTypeResult      DocumentType = "result"
	DocumentTypeBusiness    DocumentType = "business"
	DocumentTypePerformance DocumentType = "performance"
	DocumentTypeMemory      DocumentType = "memory"
)

// Document represents an embedded document in the vector store
type Document struct {
	ID            string                 `json:"id"`
	ConnectionID  string                 `json:"connection_id"`
	Type          DocumentType           `json:"type"`
	Content       string                 `json:"content"`
	Embedding     []float32              `json:"embedding"`
	Metadata      map[string]interface{} `json:"metadata"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	AccessCount   int                    `json:"access_count"`
	LastAccessed  time.Time              `json:"last_accessed"`
	Score         float32                `json:"score,omitempty"`
}

// VectorStore defines the interface for vector storage operations
type VectorStore interface {
	// Initialize the vector store
	Initialize(ctx context.Context) error

	// Document operations
	IndexDocument(ctx context.Context, doc *Document) error
	BatchIndexDocuments(ctx context.Context, docs []*Document) error
	GetDocument(ctx context.Context, id string) (*Document, error)
	UpdateDocument(ctx context.Context, doc *Document) error
	DeleteDocument(ctx context.Context, id string) error

	// Search operations
	SearchSimilar(ctx context.Context, embedding []float32, k int, filter map[string]interface{}) ([]*Document, error)
	SearchByText(ctx context.Context, query string, k int, filter map[string]interface{}) ([]*Document, error)
	HybridSearch(ctx context.Context, query string, embedding []float32, k int) ([]*Document, error)

	// Collection management
	CreateCollection(ctx context.Context, name string, dimension int) error
	DeleteCollection(ctx context.Context, name string) error
	ListCollections(ctx context.Context) ([]string, error)

	// Statistics
	GetStats(ctx context.Context) (*VectorStoreStats, error)
	GetCollectionStats(ctx context.Context, collection string) (*CollectionStats, error)

	// Maintenance
	Optimize(ctx context.Context) error
	Backup(ctx context.Context, path string) error
	Restore(ctx context.Context, path string) error
}

// VectorStoreStats represents overall statistics
type VectorStoreStats struct {
	TotalDocuments   int64     `json:"total_documents"`
	TotalCollections int       `json:"total_collections"`
	StorageSize      int64     `json:"storage_size_bytes"`
	LastOptimized    time.Time `json:"last_optimized"`
	LastBackup       time.Time `json:"last_backup"`
}

// CollectionStats represents collection-specific statistics
type CollectionStats struct {
	Name          string    `json:"name"`
	DocumentCount int64     `json:"document_count"`
	Dimension     int       `json:"dimension"`
	IndexSize     int64     `json:"index_size_bytes"`
	LastUpdated   time.Time `json:"last_updated"`
}


// CollectionConfig defines collection configuration
type CollectionConfig struct {
	Name            string                 `json:"name"`
	VectorSize      int                    `json:"vector_size"`
	Distance        string                 `json:"distance"` // cosine, euclidean, dot
	OnDiskPayload   bool                   `json:"on_disk_payload"`
	OptimizersConfig map[string]interface{} `json:"optimizers_config"`
	WalConfig       map[string]interface{} `json:"wal_config"`
}

// VectorStoreConfig holds configuration for any vector store
type VectorStoreConfig struct {
    Type         string              `json:"type"`
    SQLiteConfig *SQLiteVectorConfig `json:"sqlite_config,omitempty"`
    MySQLConfig  *MySQLVectorConfig  `json:"mysql_config,omitempty"`
}

// NewVectorStore creates a new vector store based on configuration
func NewVectorStore(config *VectorStoreConfig, logger *logrus.Logger) (VectorStore, error) {
    storeType := strings.ToLower(config.Type)
    switch storeType {
    case "", "sqlite":
        if config.SQLiteConfig == nil {
            logger.Info("Creating default SQLite vector store config")
            config.SQLiteConfig = &SQLiteVectorConfig{
                Path:        "~/.howlerops/vectors.db",
                VectorSize:  1536,
                CacheSizeMB: 128,
                MMapSizeMB:  256,
                WALEnabled:  true,
                Timeout:     10 * time.Second,
            }
        }
        return NewSQLiteVectorStore(config.SQLiteConfig, logger)
    case "mysql":
        if config.MySQLConfig == nil {
            return nil, fmt.Errorf("mysql vector store requires configuration")
        }
        return NewMySQLVectorStore(config.MySQLConfig, logger)
    default:
        return nil, fmt.Errorf("unsupported vector store type: %s", storeType)
    }
}
