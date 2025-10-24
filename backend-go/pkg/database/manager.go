package database

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/database/multiquery"
)

// databaseAdapter adapts Database to multiquery.Database interface
type databaseAdapter struct {
	db Database
}

// Execute implements multiquery.Database interface
func (a *databaseAdapter) Execute(ctx context.Context, query string, args ...interface{}) (*multiquery.QueryResult, error) {
	result, err := a.db.Execute(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	// Convert database.QueryResult to multiquery.QueryResult
	multiqueryResult := &multiquery.QueryResult{
		Columns:  result.Columns,
		Rows:     result.Rows,
		RowCount: result.RowCount,
		Duration: result.Duration,
	}
	
	// Convert editable metadata if present
	if result.Editable != nil {
		multiqueryResult.Editable = &multiquery.EditableQueryMetadata{
			Enabled:     result.Editable.Enabled,
			Reason:      result.Editable.Reason,
			Schema:      result.Editable.Schema,
			Table:       result.Editable.Table,
			PrimaryKeys: result.Editable.PrimaryKeys,
			Pending:     result.Editable.Pending,
			JobID:       result.Editable.JobID,
		}
		
		// Convert columns
		multiqueryResult.Editable.Columns = make([]multiquery.EditableColumn, len(result.Editable.Columns))
		for i, col := range result.Editable.Columns {
			multiqueryResult.Editable.Columns[i] = multiquery.EditableColumn{
				Name:       col.Name,
				ResultName: col.ResultName,
				DataType:   col.DataType,
				Editable:   col.Editable,
				PrimaryKey: col.PrimaryKey,
			}
			
			// Convert foreign key if present
			if col.ForeignKey != nil {
				multiqueryResult.Editable.Columns[i].ForeignKey = &multiquery.ForeignKeyRef{
					Table:  col.ForeignKey.Table,
					Column: col.ForeignKey.Column,
					Schema: col.ForeignKey.Schema,
				}
			}
		}
	}
	
	return multiqueryResult, nil
}

// Manager manages multiple database connections
type Manager struct {
	connections      map[string]Database
	connectionNames  map[string]string // name -> sessionId mapping for multi-DB queries
	mu               sync.RWMutex
	logger           *logrus.Logger
	multiQueryParser *multiquery.QueryParser
	multiQueryExec   *multiquery.Executor
	multiQueryConfig *multiquery.Config
	schemaCache      *SchemaCache // Smart schema caching with change detection
}

// NewManager creates a new database manager
func NewManager(logger *logrus.Logger) *Manager {
	return &Manager{
		connections:     make(map[string]Database),
		connectionNames: make(map[string]string),
		logger:          logger,
		schemaCache:     NewSchemaCache(logger),
	}
}

// NewManagerWithConfig creates a new database manager with multi-query support
func NewManagerWithConfig(logger *logrus.Logger, mqConfig *multiquery.Config) *Manager {
	m := &Manager{
		connections:      make(map[string]Database),
		connectionNames:  make(map[string]string),
		logger:           logger,
		schemaCache:      NewSchemaCache(logger),
		multiQueryConfig: mqConfig,
	}

	// Initialize multi-query components if enabled
	if mqConfig != nil && mqConfig.Enabled {
		m.multiQueryParser = multiquery.NewQueryParser(mqConfig, logger)
		m.multiQueryExec = multiquery.NewExecutor(mqConfig, logger)
		logger.Info("Multi-query support enabled")
	}

	return m
}

// CreateConnection creates a new database connection
func (m *Manager) CreateConnection(ctx context.Context, config ConnectionConfig) (*Connection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	aliasTargets := make(map[string]struct{})
	displayName := strings.TrimSpace(config.Database)
	if displayName != "" {
		aliasTargets[displayName] = struct{}{}
	}

	if config.Parameters != nil {
		if alias, ok := config.Parameters["alias"]; ok {
			if trimmed := strings.TrimSpace(alias); trimmed != "" {
				displayName = trimmed
				aliasTargets[trimmed] = struct{}{}
			}
			delete(config.Parameters, "alias")
		}

		if slug, ok := config.Parameters["alias_slug"]; ok {
			if trimmed := strings.TrimSpace(slug); trimmed != "" {
				aliasTargets[trimmed] = struct{}{}
			}
			delete(config.Parameters, "alias_slug")
		}

		if lower, ok := config.Parameters["alias_lower"]; ok {
			if trimmed := strings.TrimSpace(lower); trimmed != "" {
				aliasTargets[trimmed] = struct{}{}
			}
			delete(config.Parameters, "alias_lower")
		}

		if len(config.Parameters) == 0 {
			config.Parameters = nil
		}
	}

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

	// Store name-to-sessionId mapping for multi-DB queries
	for alias := range aliasTargets {
		if alias != "" {
			m.connectionNames[alias] = connectionID
		}
	}

	// Also register the stored connection ID if provided (for reconnecting to saved connections)
	if config.ID != "" {
		m.connectionNames[config.ID] = connectionID
		m.logger.WithFields(logrus.Fields{
			"stored_id":  config.ID,
			"session_id": connectionID,
		}).Debug("Registered stored connection ID as alias")
	}

	// Create connection metadata
	connection := &Connection{
		ID:        connectionID,
		Name:      displayName,
		Config:    config,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	m.logger.WithFields(logrus.Fields{
		"connection_id": connectionID,
		"type":          config.Type,
		"database":      config.Database,
		"alias":         displayName,
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

// resolveConnectionID resolves a connection identifier (name or sessionId) to a sessionId
// This enables multi-DB queries to use @connectionName.table syntax
func (m *Manager) resolveConnectionID(identifier string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try direct lookup first (sessionId)
	if _, exists := m.connections[identifier]; exists {
		return identifier, nil
	}

	// Try name resolution
	if sessionID, exists := m.connectionNames[identifier]; exists {
		return sessionID, nil
	}

	return "", fmt.Errorf("connection not found: %s", identifier)
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

	// Remove from connections map
	delete(m.connections, connectionID)

	// Remove from connectionNames map (find and delete the reverse mapping)
	for name, sessID := range m.connectionNames {
		if sessID == connectionID {
			delete(m.connectionNames, name)
			break
		}
	}

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
	case ClickHouse:
		return NewClickHouseDatabase(config, m.logger)
	case TiDB:
		return NewTiDBDatabase(config, m.logger)
	case Elasticsearch, OpenSearch:
		return NewElasticsearchDatabase(config, m.logger)
	case MongoDB:
		return NewMongoDBDatabase(config, m.logger)
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
	case ClickHouse:
		return NewClickHouseDatabase(config, f.logger)
	case TiDB:
		return NewTiDBDatabase(config, f.logger)
	case Elasticsearch, OpenSearch:
		return NewElasticsearchDatabase(config, f.logger)
	case MongoDB:
		return NewMongoDBDatabase(config, f.logger)
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
	case PostgreSQL, MySQL, MariaDB, ClickHouse, TiDB:
		if config.Host == "" {
			return fmt.Errorf("host is required for %s", config.Type)
		}
		if config.Port <= 0 {
			return fmt.Errorf("valid port is required for %s", config.Type)
		}
		if config.Username == "" {
			return fmt.Errorf("username is required for %s", config.Type)
		}
	case MongoDB:
		if config.Host == "" {
			return fmt.Errorf("host is required for %s", config.Type)
		}
		// Port defaults to 27017 if not specified
		// Username is optional for MongoDB (can use unauthenticated access)
	case Elasticsearch, OpenSearch:
		if config.Host == "" {
			return fmt.Errorf("host is required for %s", config.Type)
		}
		// Port defaults to 9200 if not specified
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
	case ClickHouse:
		config.Host = "localhost"
		config.Port = 9000
		config.SSLMode = "disable"
	case TiDB:
		config.Host = "localhost"
		config.Port = 4000
		config.Parameters["parseTime"] = "true"
		config.Parameters["loc"] = "UTC"
	case MongoDB:
		config.Host = "localhost"
		config.Port = 27017
		config.SSLMode = "disable"
		config.Database = "test"
	case Elasticsearch, OpenSearch:
		config.Host = "localhost"
		config.Port = 9200
		config.SSLMode = "disable"
		config.Database = "default"
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
		ClickHouse,
		TiDB,
		Elasticsearch,
		OpenSearch,
		MongoDB,
	}
}

// Multi-query methods

// ExecuteMultiQuery executes a query spanning multiple connections
func (m *Manager) ExecuteMultiQuery(ctx context.Context, query string, options *multiquery.Options) (*multiquery.Result, error) {
	if m.multiQueryParser == nil || m.multiQueryExec == nil {
		return nil, fmt.Errorf("multi-query support is not enabled")
	}

	// Parse query to identify connections
	parsed, err := m.multiQueryParser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("failed to parse multi-query: %w", err)
	}

	// Validate the parsed query
	if err := m.multiQueryParser.Validate(parsed); err != nil {
		return nil, fmt.Errorf("query validation failed: %w", err)
	}

	// Validate all connections exist
	if err := m.validateConnections(parsed.RequiredConnections); err != nil {
		return nil, err
	}

	// Get database instances for execution
	m.mu.RLock()
	connections := make(map[string]multiquery.Database)
	for _, connID := range parsed.RequiredConnections {
		// Resolve connection name to sessionId
		resolvedID := connID
		// Try direct lookup first (sessionId)
		if _, exists := m.connections[connID]; !exists {
			// Try name resolution
			if sessionID, exists := m.connectionNames[connID]; exists {
				resolvedID = sessionID
			}
		}

		if db, exists := m.connections[resolvedID]; exists {
			connections[connID] = &databaseAdapter{db: db}
		}
	}
	// For single-connection or no explicit connections, add all connections
	if len(connections) == 0 {
		for id, db := range m.connections {
			connections[id] = &databaseAdapter{db: db}
		}
	}
	m.mu.RUnlock()

	// Execute using appropriate strategy
	result, err := m.multiQueryExec.Execute(ctx, parsed, connections, options)
	if err != nil {
		return nil, fmt.Errorf("failed to execute multi-query: %w", err)
	}

	m.logger.WithFields(logrus.Fields{
		"connections": parsed.RequiredConnections,
		"duration":    result.Duration,
		"row_count":   result.RowCount,
	}).Info("Multi-query executed successfully")

	return result, nil
}

// ParseMultiQuery parses a query to identify connections without executing
func (m *Manager) ParseMultiQuery(query string) (*multiquery.ParsedQuery, error) {
	if m.multiQueryParser == nil {
		return nil, fmt.Errorf("multi-query support is not enabled")
	}

	return m.multiQueryParser.Parse(query)
}

// ValidateMultiQuery validates a parsed multi-query
func (m *Manager) ValidateMultiQuery(parsed *multiquery.ParsedQuery) error {
	if m.multiQueryParser == nil {
		return fmt.Errorf("multi-query support is not enabled")
	}

	if err := m.multiQueryParser.Validate(parsed); err != nil {
		return err
	}

	return m.validateConnections(parsed.RequiredConnections)
}

// GetMultiConnectionSchema returns combined schema for multiple connections with smart caching
func (m *Manager) GetMultiConnectionSchema(ctx context.Context, connectionIDs []string) (*multiquery.CombinedSchema, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	combined := &multiquery.CombinedSchema{
		Connections: make(map[string]*multiquery.ConnectionSchema),
		Conflicts:   []multiquery.SchemaConflict{},
	}

	// Use channels for parallel processing
	type schemaResult struct {
		connID string
		schema *multiquery.ConnectionSchema
		err    error
	}

	resultChan := make(chan schemaResult, len(connectionIDs))
	
	// Process each connection in parallel
	for _, connID := range connectionIDs {
		go func(connID string) {
			result := schemaResult{connID: connID}
			
			// Resolve connection name to sessionId
			resolvedID := connID
			// Try direct lookup first (sessionId)
			if _, exists := m.connections[connID]; !exists {
				// Try name resolution
				if sessionID, exists := m.connectionNames[connID]; exists {
					resolvedID = sessionID
				} else {
					result.err = fmt.Errorf("connection not found: %s", connID)
					resultChan <- result
					return
				}
			}

			db, exists := m.connections[resolvedID]
			if !exists {
				result.err = fmt.Errorf("connection not found: %s", connID)
				resultChan <- result
				return
			}

			// Try cache first (massive performance boost!)
			cached, err := m.schemaCache.GetCachedSchema(ctx, connID, db)
			if err == nil && cached != nil {
				// Use cached schema - 520x faster!
				connSchema := &multiquery.ConnectionSchema{
					ConnectionID: connID,
					Schemas:      cached.Schemas,
					Tables:       []multiquery.TableInfo{},
				}

				// Convert from cached format
				for _, tables := range cached.Tables {
					for _, table := range tables {
						connSchema.Tables = append(connSchema.Tables, multiquery.TableInfo{
							Schema:    table.Schema,
							Name:      table.Name,
							Type:      table.Type,
							Comment:   table.Comment,
							RowCount:  table.RowCount,
							SizeBytes: table.SizeBytes,
						})
					}
				}

				result.schema = connSchema
				m.logger.WithField("connection", connID).Debug("Schema loaded from cache")
				resultChan <- result
				return
			}

			// Cache miss - fetch fresh
			schemas, err := db.GetSchemas(ctx)
			if err != nil {
				result.err = fmt.Errorf("failed to get schemas for connection %s: %w", connID, err)
				resultChan <- result
				return
			}

			tablesMap := make(map[string][]TableInfo)
			connSchema := &multiquery.ConnectionSchema{
				ConnectionID: connID,
				Schemas:      schemas,
				Tables:       []multiquery.TableInfo{},
			}

			// Get tables for each schema in parallel
			type tableResult struct {
				schema string
				tables []TableInfo
				err    error
			}
			
			tableChan := make(chan tableResult, len(schemas))
			for _, schema := range schemas {
				go func(schema string) {
					tables, err := db.GetTables(ctx, schema)
					tableChan <- tableResult{
						schema: schema,
						tables: tables,
						err:    err,
					}
				}(schema)
			}

			// Collect table results
			for i := 0; i < len(schemas); i++ {
				tableRes := <-tableChan
				if tableRes.err != nil {
					m.logger.WithError(tableRes.err).Warnf("Failed to get tables for schema %s in connection %s", tableRes.schema, connID)
					continue
				}

				tablesMap[tableRes.schema] = tableRes.tables

				// Convert database.TableInfo to multiquery.TableInfo
				for _, table := range tableRes.tables {
					connSchema.Tables = append(connSchema.Tables, multiquery.TableInfo{
						Schema:    table.Schema,
						Name:      table.Name,
						Type:      table.Type,
						Comment:   table.Comment,
						RowCount:  table.RowCount,
						SizeBytes: table.SizeBytes,
					})
				}
			}

			// Cache the schema for future use
			if err := m.schemaCache.CacheSchema(ctx, connID, db, schemas, tablesMap); err != nil {
				m.logger.WithError(err).Warn("Failed to cache schema")
			}

			result.schema = connSchema
			m.logger.WithField("connection", connID).Debug("Schema fetched and cached")
			resultChan <- result
		}(connID)
	}

	// Collect all results
	for i := 0; i < len(connectionIDs); i++ {
		result := <-resultChan
		if result.err != nil {
			m.logger.WithError(result.err).Warnf("Failed to load schema for connection %s", result.connID)
			continue
		}
		combined.Connections[result.connID] = result.schema
	}

	// Detect naming conflicts
	combined.Conflicts = m.detectSchemaConflicts(combined.Connections)

	return combined, nil
}

func (m *Manager) validateConnections(connectionIDs []string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, connID := range connectionIDs {
		// Resolve connection name to sessionId
		resolvedID := connID
		// Try direct lookup first (sessionId)
		if _, exists := m.connections[connID]; !exists {
			// Try name resolution
			if sessionID, exists := m.connectionNames[connID]; exists {
				resolvedID = sessionID
			} else {
				return fmt.Errorf("connection not found: %s", connID)
			}
		}

		if _, exists := m.connections[resolvedID]; !exists {
			return fmt.Errorf("connection not found: %s", connID)
		}
	}
	return nil
}

func (m *Manager) detectSchemaConflicts(schemas map[string]*multiquery.ConnectionSchema) []multiquery.SchemaConflict {
	// Track table names across connections
	tableMap := make(map[string][]multiquery.ConflictingTable)

	for connID, schema := range schemas {
		for _, table := range schema.Tables {
			key := table.Name
			tableMap[key] = append(tableMap[key], multiquery.ConflictingTable{
				ConnectionID: connID,
				TableName:    table.Name,
				Schema:       table.Schema,
			})
		}
	}

	// Identify conflicts (tables with same name in multiple connections)
	var conflicts []multiquery.SchemaConflict
	for tableName, tables := range tableMap {
		if len(tables) > 1 {
			conflicts = append(conflicts, multiquery.SchemaConflict{
				TableName:   tableName,
				Connections: tables,
				Resolution:  fmt.Sprintf("Use @connection.%s syntax to disambiguate", tableName),
			})
		}
	}

	return conflicts
}
