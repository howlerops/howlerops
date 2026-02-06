package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jbeck018/howlerops/pkg/database"
	"github.com/jbeck018/howlerops/pkg/rag"
	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/jbeck018/howlerops/services"
	"github.com/sirupsen/logrus"
)

// ConnectionService handles all database connection operations
type ConnectionService struct {
	deps              *SharedDeps
	storageManager    *storage.Manager
	credentialService *services.CredentialService
	embeddingService  rag.EmbeddingService
}

// NewConnectionService creates a new ConnectionService instance
func NewConnectionService(deps *SharedDeps, cs *services.CredentialService, embeddingService rag.EmbeddingService) *ConnectionService {
	return &ConnectionService{
		deps:              deps,
		storageManager:    deps.StorageManager,
		credentialService: cs,
		embeddingService:  embeddingService,
	}
}

// SaveConnection saves connection metadata
func (s *ConnectionService) SaveConnection(req ConnectionRequest) error {
	s.deps.Logger.WithFields(logrus.Fields{
		"id":       req.ID,
		"type":     req.Type,
		"host":     req.Host,
		"database": req.Database,
	}).Info("Saving connection metadata")

	// Convert request to storage connection
	conn := &storage.Connection{
		ID:           req.ID,
		Name:         req.Name,
		Type:         req.Type,
		Host:         req.Host,
		Port:         req.Port,
		DatabaseName: req.Database,
		Username:     req.Username,
		CreatedBy:    "local-user", // TODO: get from auth context
		Metadata:     make(map[string]string),
	}

	// Store password separately in secure storage
	if req.Password != "" {
		err := s.credentialService.StorePassword(req.ID, req.Password)
		if err != nil {
			s.deps.Logger.WithError(err).Warn("Failed to store password securely")
		}
		conn.PasswordEncrypted = "***" // Don't store plaintext
	}

	// Save to storage
	return s.storageManager.SaveConnection(context.Background(), conn)
}

// CreateConnection creates a new database connection
func (s *ConnectionService) CreateConnection(req ConnectionRequest) (*ConnectionInfo, error) {
	s.deps.Logger.WithFields(logrus.Fields{
		"type":     req.Type,
		"host":     req.Host,
		"port":     req.Port,
		"database": req.Database,
		"username": req.Username,
	}).Info("Creating database connection")

	// Convert request to internal config
	config := database.ConnectionConfig{
		ID:               req.ID, // Pass stored connection ID for reconnecting
		Type:             database.DatabaseType(req.Type),
		Host:             req.Host,
		Port:             req.Port,
		Database:         req.Database,
		Username:         req.Username,
		Password:         req.Password,
		SSLMode:          req.SSLMode,
		PoolerCompatible: req.PoolerCompatible,
	}

	// Set default timeout if not provided
	if req.ConnectionTimeout > 0 {
		config.ConnectionTimeout = time.Duration(req.ConnectionTimeout) * time.Second
	} else {
		config.ConnectionTimeout = 30 * time.Second
	}

	// Set default parameters
	if config.Parameters == nil {
		config.Parameters = make(map[string]string)
	}
	for k, v := range req.Parameters {
		config.Parameters[k] = v
	}

	// Create connection
	connection, err := s.deps.DatabaseService.CreateConnection(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	// Kick off background schema indexing (table/column/relationship docs) if embeddings and storage are available
	if s.embeddingService != nil && s.storageManager != nil && connection != nil && connection.ID != "" {
		go func(connID string) {
			ctx := context.Background()
			// Build a shared SchemaIndexer from the active vector store
			vs := s.storageManager.GetVectorStore()
			vectorStore, _ := vs.(rag.VectorStore)
			if vectorStore == nil {
				return
			}
			indexer := rag.NewSchemaIndexer(vectorStore, s.embeddingService, s.deps.Logger)
			// Use the indexer's connection walker to index schema safely
			provider := &schemaProviderAdapter{dbsvc: s.deps.DatabaseService}
			if err := indexer.IndexConnection(ctx, provider, connID); err != nil {
				s.deps.Logger.WithError(err).WithField("connection_id", connID).Warn("Schema indexing failed")
			}
		}(connection.ID)
	}

	createdAt := ""
	if !connection.CreatedAt.IsZero() {
		createdAt = connection.CreatedAt.Format(time.RFC3339)
	}

	return &ConnectionInfo{
		ID:        connection.ID,
		Type:      string(connection.Config.Type),
		Host:      connection.Config.Host,
		Port:      connection.Config.Port,
		Database:  connection.Config.Database,
		Username:  connection.Config.Username,
		Active:    connection.Active,
		CreatedAt: createdAt,
	}, nil
}

// TestConnection tests a database connection
func (s *ConnectionService) TestConnection(req ConnectionRequest) error {
	s.deps.Logger.WithFields(logrus.Fields{
		"type":     req.Type,
		"host":     req.Host,
		"port":     req.Port,
		"database": req.Database,
		"username": req.Username,
	}).Info("Testing database connection")

	config := database.ConnectionConfig{
		Type:              database.DatabaseType(req.Type),
		Host:              req.Host,
		Port:              req.Port,
		Database:          req.Database,
		Username:          req.Username,
		Password:          req.Password,
		SSLMode:           req.SSLMode,
		ConnectionTimeout: time.Duration(req.ConnectionTimeout) * time.Second,
		Parameters:        req.Parameters,
	}

	if config.ConnectionTimeout == 0 {
		config.ConnectionTimeout = 10 * time.Second
	}

	err := s.deps.DatabaseService.TestConnection(config)
	if err != nil {
		s.deps.Logger.WithFields(logrus.Fields{
			"type":     req.Type,
			"host":     req.Host,
			"port":     req.Port,
			"database": req.Database,
			"sslmode":  req.SSLMode,
		}).WithError(err).Error("Database connection test failed")
	} else {
		s.deps.Logger.WithField("type", req.Type).Info("Database connection test successful")
	}
	return err
}

// ListConnections returns all active connections
func (s *ConnectionService) ListConnections() ([]string, error) {
	return s.deps.DatabaseService.ListConnections(), nil
}

// RemoveConnection removes a database connection
func (s *ConnectionService) RemoveConnection(connectionID string) error {
	s.deps.Logger.WithField("connection_id", connectionID).Info("Removing database connection")
	return s.deps.DatabaseService.RemoveConnection(connectionID)
}

// ListConnectionDatabases returns the databases available for a connection
func (s *ConnectionService) ListConnectionDatabases(connectionID string) (*ListDatabasesResponse, error) {
	databases, err := s.deps.DatabaseService.ListDatabases(connectionID)
	if err != nil {
		return &ListDatabasesResponse{ //nolint:nilerr // error embedded in response
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &ListDatabasesResponse{
		Success:   true,
		Databases: databases,
	}, nil
}

// SwitchConnectionDatabase switches the active database for a connection
func (s *ConnectionService) SwitchConnectionDatabase(req SwitchDatabaseRequest) (*SwitchDatabaseResponse, error) {
	result, err := s.deps.DatabaseService.SwitchDatabase(req.ConnectionID, req.Database)
	if err != nil {
		return &SwitchDatabaseResponse{ //nolint:nilerr // error embedded in response
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &SwitchDatabaseResponse{
		Success:     true,
		Database:    result.Database,
		Reconnected: result.Reconnected,
	}, nil
}

// GetConnectionHealth returns the health status of a connection
func (s *ConnectionService) GetConnectionHealth(connectionID string) (*HealthStatus, error) {
	status, err := s.deps.DatabaseService.GetConnectionHealth(connectionID)
	if err != nil {
		return nil, err
	}
	return convertHealthStatus(status), nil
}

// GetDatabaseVersion returns the database version
func (s *ConnectionService) GetDatabaseVersion(connectionID string) (string, error) {
	info, err := s.deps.DatabaseService.GetConnectionInfo(connectionID)
	if err != nil {
		return "", err
	}

	if version, ok := info["version"].(string); ok {
		return version, nil
	}

	return "Unknown", nil
}

// GetConnectionStats returns connection pool statistics
func (s *ConnectionService) GetConnectionStats() map[string]database.PoolStats {
	return s.deps.DatabaseService.GetConnectionStats()
}

// HealthCheckAll performs health checks on all connections
func (s *ConnectionService) HealthCheckAll() map[string]*HealthStatus {
	raw := s.deps.DatabaseService.HealthCheckAll()
	results := make(map[string]*HealthStatus, len(raw))
	for key, status := range raw {
		results[key] = convertHealthStatus(status)
	}
	return results
}

// convertHealthStatus converts database.HealthStatus to app.HealthStatus
func convertHealthStatus(status *database.HealthStatus) *HealthStatus {
	if status == nil {
		return &HealthStatus{}
	}

	timestamp := ""
	if !status.Timestamp.IsZero() {
		timestamp = status.Timestamp.Format(time.RFC3339)
	}

	metrics := status.Metrics
	if metrics == nil {
		metrics = make(map[string]string)
	}

	return &HealthStatus{
		Status:       status.Status,
		Message:      status.Message,
		Timestamp:    timestamp,
		ResponseTime: status.ResponseTime.Milliseconds(),
		Metrics:      metrics,
	}
}

// GetSupportedDatabaseTypes returns supported database types
func (s *ConnectionService) GetSupportedDatabaseTypes() []string {
	return s.deps.DatabaseService.GetSupportedDatabaseTypes()
}

// GetDatabaseTypeInfo returns information about a database type
func (s *ConnectionService) GetDatabaseTypeInfo(dbType string) map[string]interface{} {
	return s.deps.DatabaseService.GetDatabaseTypeInfo(dbType)
}

// InvalidateSchemaCache invalidates the cached schema for a specific connection
func (s *ConnectionService) InvalidateSchemaCache(connectionID string) error {
	s.deps.Logger.WithField("connection", connectionID).Info("Invalidating schema cache")
	s.deps.DatabaseService.InvalidateSchemaCache(connectionID)
	return nil
}

// InvalidateAllSchemas invalidates all cached schemas
func (s *ConnectionService) InvalidateAllSchemas() error {
	s.deps.Logger.Info("Invalidating all schema caches")
	s.deps.DatabaseService.InvalidateAllSchemas()
	return nil
}

// RefreshSchema forces a refresh of the schema for a connection
func (s *ConnectionService) RefreshSchema(connectionID string) error {
	s.deps.Logger.WithField("connection", connectionID).Info("Refreshing schema")
	return s.deps.DatabaseService.RefreshSchema(context.Background(), connectionID)
}

// GetSchemaCacheStats returns statistics about the schema cache
func (s *ConnectionService) GetSchemaCacheStats() map[string]interface{} {
	return s.deps.DatabaseService.GetSchemaCacheStats()
}

// GetConnectionCount returns the number of active database connections
func (s *ConnectionService) GetConnectionCount() int {
	return s.deps.DatabaseService.GetConnectionCount()
}

// GetConnectionIDs returns a list of all connection IDs
func (s *ConnectionService) GetConnectionIDs() []string {
	return s.deps.DatabaseService.GetConnectionIDs()
}

// GetAvailableEnvironments returns all unique environment tags across connections
func (s *ConnectionService) GetAvailableEnvironments() ([]string, error) {
	if s.storageManager == nil {
		return []string{}, nil
	}
	return s.storageManager.GetAvailableEnvironments(context.Background())
}
