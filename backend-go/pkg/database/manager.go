package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Manager manages multiple database connections
type Manager struct {
	connections map[string]Database
	mu          sync.RWMutex
	logger      *logrus.Logger
}

// NewManager creates a new database manager
func NewManager(logger *logrus.Logger) *Manager {
	return &Manager{
		connections: make(map[string]Database),
		logger:      logger,
	}
}

// CreateConnection creates a new database connection
func (m *Manager) CreateConnection(ctx context.Context, config ConnectionConfig) (*Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Create database instance based on type
	db, err := m.createDatabaseInstance(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create database instance: %w", err)
	}

	// Test the connection
	if err := db.Connect(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Generate connection ID
	connectionID := uuid.New().String()

	// Store the database instance
	m.connections[connectionID] = db

	// Create connection metadata
	connection := &Connection{
		ID:        connectionID,
		Config:    config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	m.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"type":          config.Type,
		"database":      config.Database,
	}).Info("Database connection created successfully")

	return connection, nil
}

// GetConnection retrieves a database connection by ID
func (m *Manager) GetConnection(connectionID string) (Database, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	db, exists := m.connections[connectionID]
	if !exists {
		return nil, fmt.Errorf("connection not found: %s", connectionID)
	}

	return db, nil
}

// ListConnections returns all active connections
func (m *Manager) ListConnections() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	connectionIDs := make([]string, 0, len(m.connections))
	for id := range m.connections {
		connectionIDs = append(connectionIDs, id)
	}

	return connectionIDs
}

// RemoveConnection removes a database connection
func (m *Manager) RemoveConnection(connectionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	db, exists := m.connections[connectionID]
	if !exists {
		return fmt.Errorf("connection not found: %s", connectionID)
	}

	// Close the connection
	if err := db.Disconnect(); err != nil {
		m.logger.WithFields(logrus.Fields{
			"connection_id": connectionID,
			"error":         err,
		}).Error("Failed to disconnect database")
	}

	// Remove from map
	delete(m.connections, connectionID)

	m.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
	}).Info("Database connection removed")

	return nil
}

// TestConnection tests a database connection configuration
func (m *Manager) TestConnection(ctx context.Context, config ConnectionConfig) error {
	// Create temporary database instance
	db, err := m.createDatabaseInstance(config)
	if err != nil {
		return fmt.Errorf("failed to create database instance: %w", err)
	}

	// Test connection
	if err := db.Connect(ctx, config); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Clean up
	if err := db.Disconnect(); err != nil {
		m.logger.WithError(err).Warn("Failed to disconnect test connection")
	}

	return nil
}

// GetConnectionHealth returns health status for a connection
func (m *Manager) GetConnectionHealth(ctx context.Context, connectionID string) (*HealthStatus, error) {
	db, err := m.GetConnection(connectionID)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = db.Ping(ctx)
	duration := time.Since(start)

	status := &HealthStatus{
		Timestamp:    time.Now(),
		ResponseTime: duration,
		Metrics:      make(map[string]string),
	}

	if err != nil {
		status.Status = "unhealthy"
		status.Message = fmt.Sprintf("Ping failed: %v", err)
	} else {
		status.Status = "healthy"
		status.Message = "Connection is healthy"

		// Add connection stats
		stats := db.GetConnectionStats()
		status.Metrics["open_connections"] = fmt.Sprintf("%d", stats.OpenConnections)
		status.Metrics["in_use"] = fmt.Sprintf("%d", stats.InUse)
		status.Metrics["idle"] = fmt.Sprintf("%d", stats.Idle)
	}

	return status, nil
}

// UpdateRow applies changes to a single row for the specified connection
func (m *Manager) UpdateRow(ctx context.Context, connectionID string, params UpdateRowParams) error {
	db, err := m.GetConnection(connectionID)
	if err != nil {
		return err
	}

	return db.UpdateRow(ctx, params)
}

// Close closes all database connections
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var lastErr error
	for id, db := range m.connections {
		if err := db.Disconnect(); err != nil {
			m.logger.WithFields(logrus.Fields{
				"connection_id": id,
				"error":         err,
			}).Error("Failed to close database connection")
			lastErr = err
		}
	}

	// Clear the map
	m.connections = make(map[string]Database)

	m.logger.Info("All database connections closed")
	return lastErr
}

// createDatabaseInstance creates a database instance based on type
func (m *Manager) createDatabaseInstance(config ConnectionConfig) (Database, error) {
	switch config.Type {
	case PostgreSQL:
		return NewPostgresDatabase(config, m.logger)
	case MySQL, MariaDB:
		return NewMySQLDatabase(config, m.logger)
	case SQLite:
		return NewSQLiteDatabase(config, m.logger)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// GetConnectionStats returns statistics for all connections
func (m *Manager) GetConnectionStats() map[string]PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for id, db := range m.connections {
		stats[id] = db.GetConnectionStats()
	}

	return stats
}

// HealthCheckAll checks health of all connections
func (m *Manager) HealthCheckAll(ctx context.Context) map[string]*HealthStatus {
	m.mu.RLock()
	connectionIDs := make([]string, 0, len(m.connections))
	for id := range m.connections {
		connectionIDs = append(connectionIDs, id)
	}
	m.mu.RUnlock()

	results := make(map[string]*HealthStatus)
	var wg sync.WaitGroup

	for _, id := range connectionIDs {
		wg.Add(1)
		go func(connectionID string) {
			defer wg.Done()
			status, err := m.GetConnectionHealth(ctx, connectionID)
			if err != nil {
				status = &HealthStatus{
					Status:    "error",
					Message:   fmt.Sprintf("Failed to check health: %v", err),
					Timestamp: time.Now(),
					Metrics:   make(map[string]string),
				}
			}
			results[connectionID] = status
		}(id)
	}

	wg.Wait()
	return results
}

// Factory provides factory methods for creating database instances
type Factory struct {
	logger *logrus.Logger
}

// NewFactory creates a new database factory
func NewFactory(logger *logrus.Logger) *Factory {
	return &Factory{
		logger: logger,
	}
}

// CreateDatabase creates a database instance based on configuration
func (f *Factory) CreateDatabase(config ConnectionConfig) (Database, error) {
	switch config.Type {
	case PostgreSQL:
		return NewPostgresDatabase(config, f.logger)
	case MySQL, MariaDB:
		return NewMySQLDatabase(config, f.logger)
	case SQLite:
		return NewSQLiteDatabase(config, f.logger)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", config.Type)
	}
}

// ValidateConfig validates a database configuration
func (f *Factory) ValidateConfig(config ConnectionConfig) error {
	if config.Type == "" {
		return fmt.Errorf("database type is required")
	}

	if config.Database == "" {
		return fmt.Errorf("database name is required")
	}

	switch config.Type {
	case PostgreSQL, MySQL, MariaDB:
		if config.Host == "" {
			return fmt.Errorf("host is required for %s", config.Type)
		}
		if config.Port <= 0 {
			return fmt.Errorf("valid port is required for %s", config.Type)
		}
		if config.Username == "" {
			return fmt.Errorf("username is required for %s", config.Type)
		}
	case SQLite:
		// SQLite only needs database file path
		if config.Database == "" {
			return fmt.Errorf("database file path is required for SQLite")
		}
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	return nil
}

// GetDefaultConfig returns default configuration for a database type
func (f *Factory) GetDefaultConfig(dbType DatabaseType) ConnectionConfig {
	config := ConnectionConfig{
		Type:              dbType,
		ConnectionTimeout: 30 * time.Second,
		IdleTimeout:       5 * time.Minute,
		MaxConnections:    25,
		MaxIdleConns:      5,
		Parameters:        make(map[string]string),
	}

	switch dbType {
	case PostgreSQL:
		config.Host = "localhost"
		config.Port = 5432
		config.SSLMode = "prefer"
	case MySQL, MariaDB:
		config.Host = "localhost"
		config.Port = 3306
		config.Parameters["parseTime"] = "true"
		config.Parameters["loc"] = "UTC"
	case SQLite:
		config.Database = ":memory:"
		config.Parameters["cache"] = "shared"
		config.Parameters["mode"] = "rwc"
	}

	return config
}

// GetSupportedTypes returns list of supported database types
func (f *Factory) GetSupportedTypes() []DatabaseType {
	return []DatabaseType{
		PostgreSQL,
		MySQL,
		MariaDB,
		SQLite,
	}
}
