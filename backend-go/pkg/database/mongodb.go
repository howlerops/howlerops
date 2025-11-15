package database

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDBDatabase implements the Database interface for MongoDB
type MongoDBDatabase struct {
	client         *mongo.Client
	config         ConnectionConfig
	logger         *logrus.Logger
	stats          mongoConnectionStats
	structureCache *tableStructureCache
	mu             sync.RWMutex
}

// mongoConnectionStats tracks connection statistics for MongoDB
type mongoConnectionStats struct {
	requestCount  int64
	errorCount    int64
	lastRequestAt time.Time
}

// NewMongoDBDatabase creates a new MongoDB database instance
func NewMongoDBDatabase(config ConnectionConfig, logger *logrus.Logger) (*MongoDBDatabase, error) {
	m := &MongoDBDatabase{
		config:         config,
		logger:         logger,
		structureCache: newTableStructureCache(10 * time.Minute),
	}

	if err := m.Connect(context.Background(), config); err != nil {
		return nil, err
	}

	return m, nil
}

// Connect establishes a connection to MongoDB
func (m *MongoDBDatabase) Connect(ctx context.Context, config ConnectionConfig) error {
	m.config = config

	// Set default port if not specified
	if config.Port == 0 {
		config.Port = 27017
	}

	// Build connection URI
	uri := m.buildConnectionURI(config)

	// Set client options
	clientOpts := options.Client().ApplyURI(uri)

	// Configure connection timeout
	timeout := 30 * time.Second
	if config.ConnectionTimeout > 0 {
		timeout = config.ConnectionTimeout
	}
	if strings.EqualFold(os.Getenv("SQLSTUDIO_FAST_DB_TESTS"), "1") && timeout > 2*time.Second {
		timeout = 2 * time.Second
	}
	clientOpts.SetConnectTimeout(timeout)
	clientOpts.SetServerSelectionTimeout(timeout)

	// Configure connection pool
	maxPoolSize := uint64(25)
	if config.MaxConnections > 0 {
		maxPoolSize = uint64(config.MaxConnections)
	}
	clientOpts.SetMaxPoolSize(maxPoolSize)

	minPoolSize := uint64(5)
	if config.MaxIdleConns > 0 {
		minPoolSize = uint64(config.MaxIdleConns)
	}
	clientOpts.SetMinPoolSize(minPoolSize)

	// Configure idle timeout
	if config.IdleTimeout > 0 {
		clientOpts.SetMaxConnIdleTime(config.IdleTimeout)
	}

	// Configure authentication
	if config.Username != "" {
		credential := options.Credential{
			Username: config.Username,
			Password: config.Password,
		}

		// Check for authentication mechanism in parameters
		if config.Parameters != nil {
			if authMech, ok := config.Parameters["authMechanism"]; ok {
				credential.AuthMechanism = authMech
			}
			if authSource, ok := config.Parameters["authSource"]; ok {
				credential.AuthSource = authSource
			} else {
				// Default to admin database for authentication
				credential.AuthSource = "admin"
			}
		} else {
			credential.AuthSource = "admin"
		}

		clientOpts.SetAuth(credential)
	}

	// Configure TLS/SSL
	if config.SSLMode == "require" || config.SSLMode == "verify-full" {
		// #nosec G402 - InsecureSkipVerify controlled by user config, disabled in verify-full mode
		tlsConfig := &tls.Config{
			InsecureSkipVerify: config.SSLMode != "verify-full",
		}
		clientOpts.SetTLSConfig(tlsConfig)
	}

	// Create client
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test connection
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(ctx) // Best-effort disconnect on error
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	m.mu.Lock()
	if m.client != nil {
		if err := m.client.Disconnect(context.Background()); err != nil {
			log.Printf("Failed to disconnect existing MongoDB client: %v", err)
		}
	}
	m.client = client
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"host":     config.Host,
		"port":     config.Port,
		"database": config.Database,
	}).Info("MongoDB connection established successfully")

	return nil
}

// buildConnectionURI builds a MongoDB connection URI
func (m *MongoDBDatabase) buildConnectionURI(config ConnectionConfig) string {
	host := config.Host
	port := config.Port
	if port == 0 {
		port = 27017
	}

	// Build URI based on configuration
	var uri string
	if config.Parameters != nil {
		if customURI, ok := config.Parameters["uri"]; ok {
			// Use custom URI if provided
			return customURI
		}
	}

	// Build standard URI
	scheme := "mongodb"
	if config.SSLMode == "require" || config.SSLMode == "verify-full" {
		scheme = "mongodb+srv"
		// For SRV records, don't include port
		uri = fmt.Sprintf("%s://%s", scheme, host)
	} else {
		uri = fmt.Sprintf("%s://%s:%d", scheme, host, port)
	}

	// Add database to URI if specified
	if config.Database != "" {
		uri = fmt.Sprintf("%s/%s", uri, config.Database)
	}

	// Add connection options
	options := []string{}
	if config.Parameters != nil {
		for key, value := range config.Parameters {
			// Skip special parameters
			if key == "uri" || key == "authMechanism" || key == "authSource" {
				continue
			}
			options = append(options, fmt.Sprintf("%s=%s", key, value))
		}
	}

	if len(options) > 0 {
		uri = fmt.Sprintf("%s?%s", uri, strings.Join(options, "&"))
	}

	return uri
}

// Disconnect closes the MongoDB connection
func (m *MongoDBDatabase) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := m.client.Disconnect(ctx); err != nil {
			return err
		}
		m.client = nil
	}

	m.logger.Info("MongoDB connection closed")
	return nil
}

// Ping tests the MongoDB connection
func (m *MongoDBDatabase) Ping(ctx context.Context) error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("not connected to MongoDB")
	}

	return client.Ping(ctx, readpref.Primary())
}

// GetConnectionInfo returns MongoDB connection information
func (m *MongoDBDatabase) GetConnectionInfo(ctx context.Context) (map[string]interface{}, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected to MongoDB")
	}

	info := make(map[string]interface{})

	// Get server version and build info
	db := client.Database("admin")
	var buildInfo bson.M
	err := db.RunCommand(ctx, bson.D{{Key: "buildInfo", Value: 1}}).Decode(&buildInfo)
	if err == nil {
		if version, ok := buildInfo["version"].(string); ok {
			info["version"] = version
		}
		if gitVersion, ok := buildInfo["gitVersion"].(string); ok {
			info["git_version"] = gitVersion
		}
	}

	// Get server status
	var serverStatus bson.M
	err = db.RunCommand(ctx, bson.D{{Key: "serverStatus", Value: 1}}).Decode(&serverStatus)
	if err == nil {
		if connections, ok := serverStatus["connections"].(bson.M); ok {
			if current, ok := connections["current"].(int32); ok {
				info["current_connections"] = current
			}
			if available, ok := connections["available"].(int32); ok {
				info["available_connections"] = available
			}
		}
		if repl, ok := serverStatus["repl"].(bson.M); ok {
			if setName, ok := repl["setName"].(string); ok {
				info["replica_set"] = setName
			}
			if ismaster, ok := repl["ismaster"].(bool); ok {
				info["is_primary"] = ismaster
			}
		}
	}

	// Get database name
	info["database"] = m.config.Database

	// Get host info
	var hostInfo bson.M
	err = db.RunCommand(ctx, bson.D{{Key: "hostInfo", Value: 1}}).Decode(&hostInfo)
	if err == nil {
		if system, ok := hostInfo["system"].(bson.M); ok {
			if hostname, ok := system["hostname"].(string); ok {
				info["hostname"] = hostname
			}
		}
	}

	return info, nil
}

// Execute runs a MongoDB query and returns the results
func (m *MongoDBDatabase) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	return m.ExecuteWithOptions(ctx, query, nil, args...)
}

// ExecuteWithOptions runs a MongoDB query with options and returns the results
func (m *MongoDBDatabase) ExecuteWithOptions(ctx context.Context, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	start := time.Now()
	m.stats.requestCount++
	m.stats.lastRequestAt = start

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		m.stats.errorCount++
		return &QueryResult{
			Error:    fmt.Errorf("not connected to MongoDB"),
			Duration: time.Since(start),
		}, fmt.Errorf("not connected to MongoDB")
	}

	// Parse and execute query
	result, err := m.parseAndExecute(ctx, client, query, opts, args...)
	if err != nil {
		m.stats.errorCount++
		return &QueryResult{
			Error:    err,
			Duration: time.Since(start),
		}, err
	}

	result.Duration = time.Since(start)

	// MongoDB collections are not directly editable via SQL-like interface
	metadata := newEditableMetadata(result.Columns)
	metadata.Reason = "MongoDB collections are not directly editable via SQL interface (use native MongoDB operations)"
	result.Editable = metadata

	return result, nil
}

// parseAndExecute parses a query and executes it
func (m *MongoDBDatabase) parseAndExecute(ctx context.Context, client *mongo.Client, query string, opts *QueryOptions, args ...interface{}) (*QueryResult, error) {
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	// Try to parse as simple SQL SELECT
	if strings.HasPrefix(upperQuery, "SELECT") {
		return m.executeSelectQuery(ctx, client, query, opts)
	}

	// Try to parse as MongoDB find command (JSON format)
	if strings.HasPrefix(query, "{") || strings.HasPrefix(query, "[") {
		return m.executeMongoQuery(ctx, client, query, opts)
	}

	return nil, fmt.Errorf("unsupported query format. Use SELECT syntax (e.g., SELECT * FROM collection WHERE field = 'value') or MongoDB JSON query (e.g., {\"find\": \"collection\", \"filter\": {}})")
}

// executeSelectQuery executes a SQL-like SELECT query
func (m *MongoDBDatabase) executeSelectQuery(ctx context.Context, client *mongo.Client, query string, opts *QueryOptions) (*QueryResult, error) {
	// Parse simple SELECT query
	// Format: SELECT * FROM collection [WHERE field = value] [LIMIT n]
	upperQuery := strings.ToUpper(query)

	// Extract collection name
	fromIndex := strings.Index(upperQuery, "FROM")
	if fromIndex == -1 {
		return nil, fmt.Errorf("invalid SELECT query: missing FROM clause")
	}

	remaining := strings.TrimSpace(query[fromIndex+4:])
	parts := strings.Fields(remaining)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid SELECT query: missing collection name")
	}

	collectionName := parts[0]
	collectionName = strings.Trim(collectionName, "`;\"")

	// Build filter
	filter := bson.M{}
	whereIndex := strings.Index(upperQuery, "WHERE")
	if whereIndex != -1 {
		// Simple WHERE parsing (field = value)
		whereClause := strings.TrimSpace(query[whereIndex+5:])
		// Remove LIMIT clause if present
		if limitIndex := strings.Index(strings.ToUpper(whereClause), "LIMIT"); limitIndex != -1 {
			whereClause = strings.TrimSpace(whereClause[:limitIndex])
		}

		// Very basic parser for field = value
		if strings.Contains(whereClause, "=") {
			eqParts := strings.SplitN(whereClause, "=", 2)
			if len(eqParts) == 2 {
				field := strings.TrimSpace(eqParts[0])
				value := strings.TrimSpace(eqParts[1])
				// Remove quotes from value
				value = strings.Trim(value, "'\"")
				filter[field] = value
			}
		}
	}

	// Extract LIMIT from query if not provided in opts
	queryLimit := int64(0)
	limitIndex := strings.Index(upperQuery, "LIMIT")
	if limitIndex != -1 {
		limitClause := strings.TrimSpace(query[limitIndex+5:])
		_, _ = fmt.Sscanf(limitClause, "%d", &queryLimit) // Best-effort parsing
	}

	// Execute query
	db := client.Database(m.config.Database)
	collection := db.Collection(collectionName)

	// Step 1: Get total count if pagination is requested
	var totalRows int64
	if opts != nil && opts.Limit > 0 {
		count, err := collection.CountDocuments(ctx, filter)
		if err != nil {
			m.logger.WithError(err).Warn("Failed to get total count for pagination")
			totalRows = 0
		} else {
			totalRows = count
		}
	}

	// Step 2: Apply pagination options
	findOptions := options.Find()
	if opts != nil && opts.Limit > 0 {
		// #nosec G115 - opts.Limit from config/API, reasonable values (<100k), well within int64 range
		findOptions.SetLimit(int64(opts.Limit))
		if opts.Offset > 0 {
			// #nosec G115 - opts.Offset from config/API, reasonable values (<100k), well within int64 range
			findOptions.SetSkip(int64(opts.Offset))
		}
	} else if queryLimit > 0 {
		// Use limit from query if no opts provided
		findOptions.SetLimit(queryLimit)
	} else {
		// Default limit
		findOptions.SetLimit(100)
	}

	// Step 3: Execute query
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to execute MongoDB find: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("Failed to close MongoDB cursor: %v", err)
		}
	}()

	// Step 4: Read all documents
	var documents []bson.M
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, fmt.Errorf("failed to read MongoDB results: %w", err)
	}

	// Step 5: Convert to QueryResult with normalization
	result := m.convertDocumentsToQueryResult(documents)

	// Step 6: Set pagination metadata
	if opts != nil && opts.Limit > 0 {
		result.TotalRows = totalRows
		result.PagedRows = int64(len(result.Rows))
		result.Offset = opts.Offset
		result.HasMore = (int64(opts.Offset) + result.PagedRows) < totalRows
	}

	return result, nil
}

// executeMongoQuery executes a native MongoDB query in JSON format
func (m *MongoDBDatabase) executeMongoQuery(ctx context.Context, client *mongo.Client, query string, opts *QueryOptions) (*QueryResult, error) {
	var command bson.M
	if err := json.Unmarshal([]byte(query), &command); err != nil {
		return nil, fmt.Errorf("failed to parse MongoDB JSON query: %w", err)
	}

	// Execute command
	db := client.Database(m.config.Database)
	var result bson.M
	if err := db.RunCommand(ctx, command).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to execute MongoDB command: %w", err)
	}

	// Check if it's a find command result
	if cursor, ok := result["cursor"].(bson.M); ok {
		if firstBatch, ok := cursor["firstBatch"].(bson.A); ok {
			documents := make([]bson.M, len(firstBatch))
			for i, doc := range firstBatch {
				if m, ok := doc.(bson.M); ok {
					documents[i] = m
				}
			}
			return m.convertDocumentsToQueryResult(documents), nil
		}
	}

	// For other commands, return the result as a single row
	return m.convertDocumentsToQueryResult([]bson.M{result}), nil
}

// convertDocumentsToQueryResult converts MongoDB documents to QueryResult
func (m *MongoDBDatabase) convertDocumentsToQueryResult(documents []bson.M) *QueryResult {
	if len(documents) == 0 {
		return &QueryResult{
			Columns:  []string{},
			Rows:     [][]interface{}{},
			RowCount: 0,
		}
	}

	// Collect all unique field names across all documents
	columnSet := make(map[string]bool)
	columnOrder := []string{}

	for _, doc := range documents {
		for key := range doc {
			if !columnSet[key] {
				columnSet[key] = true
				columnOrder = append(columnOrder, key)
			}
		}
	}

	// Convert documents to rows with normalization
	rows := make([][]interface{}, len(documents))
	for i, doc := range documents {
		row := make([]interface{}, len(columnOrder))
		for j, col := range columnOrder {
			if val, ok := doc[col]; ok {
				// Convert BSON value then normalize it
				convertedVal := m.convertBSONValue(val)
				row[j] = NormalizeValue(convertedVal)
			} else {
				row[j] = nil
			}
		}
		rows[i] = row
	}

	return &QueryResult{
		Columns:  columnOrder,
		Rows:     rows,
		RowCount: int64(len(rows)),
	}
}

// convertBSONValue converts BSON values to standard Go types
func (m *MongoDBDatabase) convertBSONValue(val interface{}) interface{} {
	switch v := val.(type) {
	case primitive.ObjectID:
		return v.Hex()
	case primitive.DateTime:
		return v.Time().Format(time.RFC3339)
	case primitive.Timestamp:
		return time.Unix(int64(v.T), 0).Format(time.RFC3339)
	case primitive.Binary:
		return fmt.Sprintf("Binary<%d bytes>", len(v.Data))
	case primitive.Regex:
		return fmt.Sprintf("/%s/%s", v.Pattern, v.Options)
	case primitive.JavaScript:
		return fmt.Sprintf("JavaScript: %s", v)
	case primitive.Decimal128:
		return v.String()
	case bson.M:
		// Convert nested document to JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	case bson.A:
		// Convert array to JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	case []interface{}:
		// Convert slice to JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	case map[string]interface{}:
		// Convert map to JSON string
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(jsonBytes)
	default:
		return v
	}
}

// ExecuteStream executes a query and streams results in batches
func (m *MongoDBDatabase) ExecuteStream(ctx context.Context, query string, batchSize int, callback func([][]interface{}) error, args ...interface{}) error {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("not connected to MongoDB")
	}

	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	// Only support SELECT queries for streaming
	if !strings.HasPrefix(upperQuery, "SELECT") {
		return fmt.Errorf("streaming only supported for SELECT queries")
	}

	// Parse collection name
	fromIndex := strings.Index(upperQuery, "FROM")
	if fromIndex == -1 {
		return fmt.Errorf("invalid SELECT query: missing FROM clause")
	}

	remaining := strings.TrimSpace(query[fromIndex+4:])
	parts := strings.Fields(remaining)
	if len(parts) == 0 {
		return fmt.Errorf("invalid SELECT query: missing collection name")
	}

	collectionName := parts[0]
	collectionName = strings.Trim(collectionName, "`;\"")

	// Build filter (simplified)
	filter := bson.M{}
	whereIndex := strings.Index(upperQuery, "WHERE")
	if whereIndex != -1 {
		whereClause := strings.TrimSpace(query[whereIndex+5:])
		if limitIndex := strings.Index(strings.ToUpper(whereClause), "LIMIT"); limitIndex != -1 {
			whereClause = strings.TrimSpace(whereClause[:limitIndex])
		}

		if strings.Contains(whereClause, "=") {
			eqParts := strings.SplitN(whereClause, "=", 2)
			if len(eqParts) == 2 {
				field := strings.TrimSpace(eqParts[0])
				value := strings.TrimSpace(eqParts[1])
				value = strings.Trim(value, "'\"")
				filter[field] = value
			}
		}
	}

	// Execute query with cursor
	db := client.Database(m.config.Database)
	collection := db.Collection(collectionName)

	// #nosec G115 - batch size from config, reasonable values (<100k), well within int32 range
	opts := options.Find().SetBatchSize(int32(batchSize))
	cursor, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return fmt.Errorf("failed to execute MongoDB find: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Printf("Failed to close MongoDB cursor: %v", err)
		}
	}()

	// Determine columns from first batch
	var columnOrder []string
	batch := make([][]interface{}, 0, batchSize)

	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return err
		}

		// Build column list from first document
		if len(columnOrder) == 0 {
			for key := range doc {
				columnOrder = append(columnOrder, key)
			}
		}

		// Convert document to row
		row := make([]interface{}, len(columnOrder))
		for j, col := range columnOrder {
			if val, ok := doc[col]; ok {
				row[j] = m.convertBSONValue(val)
			} else {
				row[j] = nil
			}
		}

		batch = append(batch, row)

		// Send batch when full
		if len(batch) >= batchSize {
			if err := callback(batch); err != nil {
				return err
			}
			batch = batch[:0]
		}
	}

	// Send remaining rows
	if len(batch) > 0 {
		if err := callback(batch); err != nil {
			return err
		}
	}

	return cursor.Err()
}

// ExplainQuery returns the execution plan for a query
func (m *MongoDBDatabase) ExplainQuery(ctx context.Context, query string, args ...interface{}) (string, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return "", fmt.Errorf("not connected to MongoDB")
	}

	// For MongoDB, we need to execute explain on the query
	query = strings.TrimSpace(query)
	upperQuery := strings.ToUpper(query)

	if !strings.HasPrefix(upperQuery, "SELECT") {
		return "", fmt.Errorf("explain only supported for SELECT queries")
	}

	// Parse collection name
	fromIndex := strings.Index(upperQuery, "FROM")
	if fromIndex == -1 {
		return "", fmt.Errorf("invalid SELECT query: missing FROM clause")
	}

	remaining := strings.TrimSpace(query[fromIndex+4:])
	parts := strings.Fields(remaining)
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid SELECT query: missing collection name")
	}

	collectionName := parts[0]
	collectionName = strings.Trim(collectionName, "`;\"")

	// Build filter
	filter := bson.M{}

	// Execute explain
	db := client.Database(m.config.Database)
	command := bson.D{
		{Key: "explain", Value: bson.D{
			{Key: "find", Value: collectionName},
			{Key: "filter", Value: filter},
		}},
		{Key: "verbosity", Value: "executionStats"},
	}

	var result bson.M
	if err := db.RunCommand(ctx, command).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to explain query: %w", err)
	}

	// Pretty print the result
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonBytes), nil
}

// ComputeEditableMetadata returns metadata indicating MongoDB collections are not directly editable
func (m *MongoDBDatabase) ComputeEditableMetadata(ctx context.Context, query string, columns []string) (*EditableQueryMetadata, error) {
	metadata := newEditableMetadata(columns)
	metadata.Reason = "MongoDB collections are not directly editable via SQL interface (use native MongoDB operations)"
	metadata.Capabilities = &MutationCapabilities{
		CanInsert: false,
		CanUpdate: false,
		CanDelete: false,
		Reason:    metadata.Reason,
	}
	return metadata, nil
}

// GetSchemas returns list of databases in MongoDB
func (m *MongoDBDatabase) GetSchemas(ctx context.Context) ([]string, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected to MongoDB")
	}

	databases, err := client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list databases: %w", err)
	}

	// Filter out system databases
	filtered := make([]string, 0, len(databases))
	for _, db := range databases {
		if db != "admin" && db != "local" && db != "config" {
			filtered = append(filtered, db)
		}
	}

	return filtered, nil
}

// ListDatabases returns the available MongoDB databases
func (m *MongoDBDatabase) ListDatabases(ctx context.Context) ([]string, error) {
	return m.GetSchemas(ctx)
}

// SwitchDatabase is not supported via the SQL interface
func (m *MongoDBDatabase) SwitchDatabase(ctx context.Context, databaseName string) error {
	return ErrDatabaseSwitchNotSupported
}

// GetTables returns list of collections in a database
func (m *MongoDBDatabase) GetTables(ctx context.Context, schema string) ([]TableInfo, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected to MongoDB")
	}

	// Use schema as database name
	dbName := schema
	if dbName == "" {
		dbName = m.config.Database
	}

	db := client.Database(dbName)
	collections, err := db.ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}

	tables := make([]TableInfo, 0, len(collections))
	for _, collName := range collections {
		// Get collection stats
		var stats bson.M
		err := db.RunCommand(ctx, bson.D{
			{Key: "collStats", Value: collName},
		}).Decode(&stats)

		table := TableInfo{
			Schema:   dbName,
			Name:     collName,
			Type:     "COLLECTION",
			Metadata: make(map[string]string),
		}

		if err == nil {
			if count, ok := stats["count"].(int32); ok {
				table.RowCount = int64(count)
			} else if count, ok := stats["count"].(int64); ok {
				table.RowCount = count
			}

			if size, ok := stats["size"].(int32); ok {
				table.SizeBytes = int64(size)
			} else if size, ok := stats["size"].(int64); ok {
				table.SizeBytes = size
			}

			if storageSize, ok := stats["storageSize"].(int64); ok {
				table.Metadata["storage_size"] = fmt.Sprintf("%d", storageSize)
			} else if storageSize, ok := stats["storageSize"].(int32); ok {
				table.Metadata["storage_size"] = fmt.Sprintf("%d", storageSize)
			}

			if avgObjSize, ok := stats["avgObjSize"].(int64); ok {
				table.Metadata["avg_obj_size"] = fmt.Sprintf("%d", avgObjSize)
			} else if avgObjSize, ok := stats["avgObjSize"].(int32); ok {
				table.Metadata["avg_obj_size"] = fmt.Sprintf("%d", avgObjSize)
			}

			if capped, ok := stats["capped"].(bool); ok {
				table.Metadata["capped"] = fmt.Sprintf("%v", capped)
			}

			if nindexes, ok := stats["nindexes"].(int32); ok {
				table.Metadata["index_count"] = fmt.Sprintf("%d", nindexes)
			}
		}

		tables = append(tables, table)
	}

	return tables, nil
}

// GetTableStructure returns detailed structure information for a collection
func (m *MongoDBDatabase) GetTableStructure(ctx context.Context, schema, table string) (*TableStructure, error) {
	// Check cache first
	if structure, ok := m.structureCache.get(schema, table); ok {
		return structure, nil
	}

	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected to MongoDB")
	}

	dbName := schema
	if dbName == "" {
		dbName = m.config.Database
	}

	db := client.Database(dbName)
	collection := db.Collection(table)

	structure := &TableStructure{
		Table: TableInfo{
			Schema: dbName,
			Name:   table,
			Type:   "COLLECTION",
		},
		Columns:     make([]ColumnInfo, 0),
		Indexes:     make([]IndexInfo, 0),
		ForeignKeys: make([]ForeignKeyInfo, 0),
		Triggers:    make([]string, 0),
		Statistics:  make(map[string]string),
	}

	// Get collection stats
	var stats bson.M
	err := db.RunCommand(ctx, bson.D{
		{Key: "collStats", Value: table},
	}).Decode(&stats)
	if err == nil {
		if count, ok := stats["count"].(int32); ok {
			structure.Table.RowCount = int64(count)
		} else if count, ok := stats["count"].(int64); ok {
			structure.Table.RowCount = count
		}

		if size, ok := stats["size"].(int64); ok {
			structure.Table.SizeBytes = size
		} else if size, ok := stats["size"].(int32); ok {
			structure.Table.SizeBytes = int64(size)
		}
	}

	// Infer schema from sample documents (first 100)
	opts := options.Find().SetLimit(100)
	cursor, err := collection.Find(ctx, bson.M{}, opts)
	if err == nil {
		defer func() {
			if err := cursor.Close(ctx); err != nil {
				log.Printf("Failed to close MongoDB cursor: %v", err)
			}
		}()

		fieldTypes := make(map[string]map[string]int) // field -> type -> count
		fieldOrder := []string{}
		fieldSet := make(map[string]bool)

		for cursor.Next(ctx) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				continue
			}

			for key, val := range doc {
				if !fieldSet[key] {
					fieldSet[key] = true
					fieldOrder = append(fieldOrder, key)
					fieldTypes[key] = make(map[string]int)
				}

				// Infer type
				dataType := m.inferBSONType(val)
				fieldTypes[key][dataType]++
			}
		}

		// Build column info from inferred schema
		for i, field := range fieldOrder {
			// Determine most common type for this field
			maxCount := 0
			dataType := "string"
			for t, count := range fieldTypes[field] {
				if count > maxCount {
					maxCount = count
					dataType = t
				}
			}

			col := ColumnInfo{
				Name:            field,
				DataType:        dataType,
				Nullable:        true, // MongoDB fields are always nullable
				OrdinalPosition: i + 1,
				Metadata:        make(map[string]string),
			}

			// Mark _id as primary key
			if field == "_id" {
				col.PrimaryKey = true
				col.Nullable = false
			}

			structure.Columns = append(structure.Columns, col)
		}
	}

	// Get indexes
	indexCursor, err := collection.Indexes().List(ctx)
	if err == nil {
		defer func() {
			if err := indexCursor.Close(ctx); err != nil {
				log.Printf("Failed to close MongoDB index cursor: %v", err)
			}
		}()

		for indexCursor.Next(ctx) {
			var indexDoc bson.M
			if err := indexCursor.Decode(&indexDoc); err != nil {
				continue
			}

			idx := IndexInfo{
				Metadata: make(map[string]string),
			}

			if name, ok := indexDoc["name"].(string); ok {
				idx.Name = name
			}

			if key, ok := indexDoc["key"].(bson.M); ok {
				for field := range key {
					idx.Columns = append(idx.Columns, field)
				}
			}

			if unique, ok := indexDoc["unique"].(bool); ok {
				idx.Unique = unique
			}

			// _id index is the primary index
			if idx.Name == "_id_" {
				idx.Primary = true
			}

			structure.Indexes = append(structure.Indexes, idx)
		}
	}

	// Cache the structure
	m.structureCache.set(schema, table, structure)

	return structure, nil
}

// inferBSONType infers the MongoDB/BSON type from a value
func (m *MongoDBDatabase) inferBSONType(val interface{}) string {
	if val == nil {
		return "null"
	}

	switch v := val.(type) {
	case primitive.ObjectID:
		return "objectId"
	case string:
		return "string"
	case int, int32, int64:
		return "int"
	case float32, float64:
		return "double"
	case bool:
		return "bool"
	case primitive.DateTime, time.Time:
		return "date"
	case primitive.Timestamp:
		return "timestamp"
	case primitive.Binary:
		return "binData"
	case primitive.Regex:
		return "regex"
	case primitive.JavaScript:
		return "javascript"
	case primitive.Decimal128:
		return "decimal"
	case bson.M, map[string]interface{}:
		return "object"
	case bson.A, []interface{}:
		return "array"
	default:
		// Use reflection for other types
		t := reflect.TypeOf(v)
		if t != nil {
			switch t.Kind() {
			case reflect.Slice, reflect.Array:
				return "array"
			case reflect.Map, reflect.Struct:
				return "object"
			default:
				return "string"
			}
		}
		return "string"
	}
}

// BeginTransaction starts a new transaction (MongoDB 4.0+ with replica sets)
func (m *MongoDBDatabase) BeginTransaction(ctx context.Context) (Transaction, error) {
	m.mu.RLock()
	client := m.client
	m.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("not connected to MongoDB")
	}

	session, err := client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start MongoDB session: %w", err)
	}

	if err := session.StartTransaction(); err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start MongoDB transaction: %w", err)
	}

	return &MongoDBTransaction{
		session: session,
		db:      m,
		ctx:     ctx,
	}, nil
}

// UpdateRow is not supported for MongoDB via SQL interface
func (m *MongoDBDatabase) UpdateRow(ctx context.Context, params UpdateRowParams) error {
	return fmt.Errorf("direct row updates are not supported in MongoDB via SQL interface (use native MongoDB update operations)")
}

// InsertRow is not supported for MongoDB via SQL interface
func (m *MongoDBDatabase) InsertRow(ctx context.Context, params InsertRowParams) (map[string]interface{}, error) {
	return nil, fmt.Errorf("direct row inserts are not supported in MongoDB via SQL interface (use native MongoDB insert operations)")
}

// DeleteRow is not supported for MongoDB via SQL interface
func (m *MongoDBDatabase) DeleteRow(ctx context.Context, params DeleteRowParams) error {
	return fmt.Errorf("direct row deletes are not supported in MongoDB via SQL interface (use native MongoDB delete operations)")
}

// GetDatabaseType returns the database type
func (m *MongoDBDatabase) GetDatabaseType() DatabaseType {
	return MongoDB
}

// GetConnectionStats returns connection statistics
func (m *MongoDBDatabase) GetConnectionStats() PoolStats {
	// MongoDB driver doesn't expose detailed pool stats like sql.DB
	// Return basic stats
	return PoolStats{
		OpenConnections: 1,
		InUse:           1,
		Idle:            0,
	}
}

// QuoteIdentifier quotes an identifier for MongoDB
func (m *MongoDBDatabase) QuoteIdentifier(identifier string) string {
	// MongoDB doesn't require quoting like SQL, but we use backticks for consistency
	return fmt.Sprintf("`%s`", strings.ReplaceAll(identifier, "`", "``"))
}

// GetDataTypeMappings returns MongoDB-specific data type mappings
func (m *MongoDBDatabase) GetDataTypeMappings() map[string]string {
	return map[string]string{
		"string":     "string",
		"int":        "int",
		"int64":      "long",
		"float":      "double",
		"float64":    "double",
		"bool":       "bool",
		"time":       "date",
		"date":       "date",
		"json":       "object",
		"object":     "object",
		"array":      "array",
		"objectId":   "objectId",
		"binary":     "binData",
		"timestamp":  "timestamp",
		"decimal":    "decimal",
		"regex":      "regex",
		"javascript": "javascript",
	}
}

// MongoDBTransaction implements the Transaction interface for MongoDB
type MongoDBTransaction struct {
	session mongo.Session
	db      *MongoDBDatabase
	ctx     context.Context
}

// Execute runs a query within the transaction
func (t *MongoDBTransaction) Execute(ctx context.Context, query string, args ...interface{}) (*QueryResult, error) {
	// Execute within session context
	return t.db.Execute(mongo.NewSessionContext(ctx, t.session), query, args...)
}

// Commit commits the transaction
func (t *MongoDBTransaction) Commit() error {
	if err := t.session.CommitTransaction(t.ctx); err != nil {
		return err
	}
	t.session.EndSession(t.ctx)
	return nil
}

// Rollback rolls back the transaction
func (t *MongoDBTransaction) Rollback() error {
	if err := t.session.AbortTransaction(t.ctx); err != nil {
		return err
	}
	t.session.EndSession(t.ctx)
	return nil
}
