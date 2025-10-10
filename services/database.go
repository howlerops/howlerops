package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/sql-studio/backend-go/pkg/database"
)

// DatabaseService wraps the backend database manager for Wails
type DatabaseService struct {
	manager  *database.Manager
	logger   *logrus.Logger
	ctx      context.Context
	mu       sync.RWMutex
	streams  map[string]*QueryStream
	streamID int64
}

// QueryStream represents an active query stream
type QueryStream struct {
	ID         string
	Query      string
	Connection string
	Canceled   bool
	Results    chan [][]interface{}
	Errors     chan error
	Done       chan bool
	BatchSize  int
	TotalRows  int64
	cancel     context.CancelFunc
}

// StreamUpdate represents a streaming query update
type StreamUpdate struct {
	StreamID string          `json:"streamId"`
	Type     string          `json:"type"` // "data", "error", "complete"
	Data     [][]interface{} `json:"data,omitempty"`
	Error    string          `json:"error,omitempty"`
	Total    int64           `json:"total,omitempty"`
}

// NewDatabaseService creates a new database service for Wails
func NewDatabaseService(logger *logrus.Logger) *DatabaseService {
	manager := database.NewManager(logger)
	return &DatabaseService{
		manager: manager,
		logger:  logger,
		streams: make(map[string]*QueryStream),
	}
}

// SetContext sets the Wails context
func (s *DatabaseService) SetContext(ctx context.Context) {
	s.ctx = ctx
}

// CreateConnection creates a new database connection
func (s *DatabaseService) CreateConnection(config database.ConnectionConfig) (*database.Connection, error) {
	connection, err := s.manager.CreateConnection(s.ctx, config)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create database connection")
		return nil, err
	}

	// Emit connection created event
	runtime.EventsEmit(s.ctx, "connection:created", map[string]interface{}{
		"id":   connection.ID,
		"type": config.Type,
		"name": config.Database,
	})

	s.logger.WithFields(logrus.Fields{
		"connection_id": connection.ID,
		"type":          config.Type,
		"database":      config.Database,
	}).Info("Database connection created successfully")

	return connection, nil
}

// TestConnection tests a database connection
func (s *DatabaseService) TestConnection(config database.ConnectionConfig) error {
	return s.manager.TestConnection(s.ctx, config)
}

// ListConnections returns all active connections
func (s *DatabaseService) ListConnections() []string {
	return s.manager.ListConnections()
}

// RemoveConnection removes a database connection
func (s *DatabaseService) RemoveConnection(connectionID string) error {
	err := s.manager.RemoveConnection(connectionID)
	if err != nil {
		return err
	}

	// Emit connection removed event
	runtime.EventsEmit(s.ctx, "connection:removed", map[string]interface{}{
		"id": connectionID,
	})

	s.logger.WithField("connection_id", connectionID).Info("Database connection removed")
	return nil
}

// ExecuteQuery executes a SQL query
func (s *DatabaseService) ExecuteQuery(connectionID, query string, options *database.QueryOptions) (*database.QueryResult, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	// Apply default options if not provided
	if options == nil {
		options = &database.QueryOptions{
			Timeout:  30 * time.Second,
			ReadOnly: false,
			Limit:    1000,
		}
	}

	// Create context with timeout
	ctx := s.ctx
	if options.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(s.ctx, options.Timeout)
		defer cancel()
	}

	result, err := db.Execute(ctx, query)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("Query execution failed")
		return nil, err
	}

	// Emit query executed event
	runtime.EventsEmit(s.ctx, "query:executed", map[string]interface{}{
		"connectionId": connectionID,
		"duration":     result.Duration.String(),
		"rowCount":     result.RowCount,
		"affected":     result.Affected,
	})

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"duration":      result.Duration,
		"row_count":     result.RowCount,
		"affected":      result.Affected,
	}).Info("Query executed successfully")

	return result, nil
}

// UpdateRow persists modifications to a single row in the result set
func (s *DatabaseService) UpdateRow(connectionID string, params database.UpdateRowParams) error {
	err := s.manager.UpdateRow(s.ctx, connectionID, params)
	if err != nil {
		s.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"schema":        params.Schema,
			"table":         params.Table,
			"error":         err,
		}).Error("Failed to update row")
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"schema":        params.Schema,
		"table":         params.Table,
	}).Info("Row updated successfully")

	return nil
}

// ExecuteQueryStream executes a query with streaming results
func (s *DatabaseService) ExecuteQueryStream(connectionID, query string, batchSize int) (string, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return "", fmt.Errorf("connection not found: %w", err)
	}

	s.mu.Lock()
	s.streamID++
	streamID := fmt.Sprintf("stream_%d", s.streamID)

	// Create stream context with cancellation
	streamCtx, cancel := context.WithCancel(s.ctx)

	stream := &QueryStream{
		ID:         streamID,
		Query:      query,
		Connection: connectionID,
		Results:    make(chan [][]interface{}, 100),
		Errors:     make(chan error, 1),
		Done:       make(chan bool, 1),
		BatchSize:  batchSize,
		cancel:     cancel,
	}

	s.streams[streamID] = stream
	s.mu.Unlock()

	// Start streaming in goroutine
	go s.handleQueryStream(streamCtx, stream, db)

	return streamID, nil
}

// handleQueryStream handles the streaming query execution
func (s *DatabaseService) handleQueryStream(ctx context.Context, stream *QueryStream, db database.Database) {
	defer func() {
		s.mu.Lock()
		delete(s.streams, stream.ID)
		s.mu.Unlock()

		close(stream.Results)
		close(stream.Errors)
		close(stream.Done)
	}()

	callback := func(rows [][]interface{}) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case stream.Results <- rows:
			stream.TotalRows += int64(len(rows))

			// Emit streaming data
			update := StreamUpdate{
				StreamID: stream.ID,
				Type:     "data",
				Data:     rows,
				Total:    stream.TotalRows,
			}
			runtime.EventsEmit(s.ctx, "query:stream", update)
			return nil
		}
	}

	err := db.ExecuteStream(ctx, stream.Query, stream.BatchSize, callback)
	if err != nil {
		update := StreamUpdate{
			StreamID: stream.ID,
			Type:     "error",
			Error:    err.Error(),
		}
		runtime.EventsEmit(s.ctx, "query:stream", update)
		stream.Errors <- err
		return
	}

	// Emit completion
	update := StreamUpdate{
		StreamID: stream.ID,
		Type:     "complete",
		Total:    stream.TotalRows,
	}
	runtime.EventsEmit(s.ctx, "query:stream", update)
	stream.Done <- true
}

// CancelQueryStream cancels a streaming query
func (s *DatabaseService) CancelQueryStream(streamID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	stream, exists := s.streams[streamID]
	if !exists {
		return fmt.Errorf("stream not found: %s", streamID)
	}

	stream.Canceled = true
	stream.cancel()

	s.logger.WithField("stream_id", streamID).Info("Query stream canceled")
	return nil
}

// GetSchemas returns available schemas for a connection
func (s *DatabaseService) GetSchemas(connectionID string) ([]string, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	return db.GetSchemas(s.ctx)
}

// GetTables returns tables in a schema
func (s *DatabaseService) GetTables(connectionID, schema string) ([]database.TableInfo, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	return db.GetTables(s.ctx, schema)
}

// GetTableStructure returns the structure of a table
func (s *DatabaseService) GetTableStructure(connectionID, schema, table string) (*database.TableStructure, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	return db.GetTableStructure(s.ctx, schema, table)
}

// ExplainQuery returns query execution plan
func (s *DatabaseService) ExplainQuery(connectionID, query string) (string, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return "", fmt.Errorf("connection not found: %w", err)
	}

	return db.ExplainQuery(s.ctx, query)
}

// GetConnectionHealth returns health status for a connection
func (s *DatabaseService) GetConnectionHealth(connectionID string) (*database.HealthStatus, error) {
	return s.manager.GetConnectionHealth(s.ctx, connectionID)
}

// GetConnectionStats returns connection statistics
func (s *DatabaseService) GetConnectionStats() map[string]database.PoolStats {
	return s.manager.GetConnectionStats()
}

// HealthCheckAll checks health of all connections
func (s *DatabaseService) HealthCheckAll() map[string]*database.HealthStatus {
	return s.manager.HealthCheckAll(s.ctx)
}

// BeginTransaction starts a new transaction
func (s *DatabaseService) BeginTransaction(connectionID string) (string, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return "", fmt.Errorf("connection not found: %w", err)
	}

	_, err = db.BeginTransaction(s.ctx)
	if err != nil {
		return "", err
	}

	// TODO: Store transaction in a transactions map
	// For now, return a transaction ID
	txID := fmt.Sprintf("tx_%d", time.Now().UnixNano())

	s.logger.WithFields(logrus.Fields{
		"connection_id":  connectionID,
		"transaction_id": txID,
	}).Info("Transaction started")

	// Emit transaction started event
	runtime.EventsEmit(s.ctx, "transaction:started", map[string]interface{}{
		"connectionId":  connectionID,
		"transactionId": txID,
	})

	return txID, nil
}

// GetConnectionInfo returns connection information
func (s *DatabaseService) GetConnectionInfo(connectionID string) (map[string]interface{}, error) {
	db, err := s.manager.GetConnection(connectionID)
	if err != nil {
		return nil, fmt.Errorf("connection not found: %w", err)
	}

	return db.GetConnectionInfo(s.ctx)
}

// ValidateQuery validates SQL syntax (basic validation)
func (s *DatabaseService) ValidateQuery(query string) (bool, string) {
	// Basic validation - check for empty query
	if query == "" {
		return false, "Query cannot be empty"
	}

	// Add more sophisticated validation as needed
	// This could include SQL parsing, syntax checking, etc.

	return true, ""
}

// Close closes all database connections and cleans up resources
func (s *DatabaseService) Close() error {
	// Cancel all active streams
	s.mu.Lock()
	for _, stream := range s.streams {
		if !stream.Canceled {
			stream.cancel()
		}
	}
	s.streams = make(map[string]*QueryStream)
	s.mu.Unlock()

	// Close database manager
	return s.manager.Close()
}

// GetSupportedDatabaseTypes returns supported database types
func (s *DatabaseService) GetSupportedDatabaseTypes() []string {
	return []string{
		string(database.PostgreSQL),
		string(database.MySQL),
		string(database.MariaDB),
		string(database.SQLite),
	}
}

// GetDatabaseTypeInfo returns information about a database type
func (s *DatabaseService) GetDatabaseTypeInfo(dbType string) map[string]interface{} {
	factory := database.NewFactory(s.logger)
	config := factory.GetDefaultConfig(database.DatabaseType(dbType))

	return map[string]interface{}{
		"type":              string(config.Type),
		"defaultHost":       config.Host,
		"defaultPort":       config.Port,
		"requiresHost":      config.Type != database.SQLite,
		"requiresAuth":      config.Type != database.SQLite,
		"supportsSSL":       config.Type == database.PostgreSQL || config.Type == database.MySQL,
		"defaultParameters": config.Parameters,
	}
}
