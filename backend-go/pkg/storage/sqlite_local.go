package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/sql-studio/backend-go/internal/rag"
)

// LocalSQLiteStorage implements Storage interface for local solo mode
type LocalSQLiteStorage struct {
	db          *sql.DB
	vectorStore rag.VectorStore
	userID      string
	mode        Mode
	logger      *logrus.Logger
}

// LocalStorageConfig holds configuration for local storage
type LocalStorageConfig struct {
	DataDir    string
	Database   string
	VectorsDB  string
	UserID     string
	VectorSize int
}

// NewLocalStorage creates a new local SQLite storage
func NewLocalStorage(config *LocalStorageConfig, logger *logrus.Logger) (*LocalSQLiteStorage, error) {
	// Expand home directory
	dataDir := os.ExpandEnv(config.DataDir)
	if strings.HasPrefix(dataDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(home, dataDir[2:])
	}

	// Create data directory
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open main database
	dbPath := filepath.Join(dataDir, config.Database)
	db, err := sql.Open("sqlite3", fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL", dbPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Create vector store
	vectorDBPath := filepath.Join(dataDir, config.VectorsDB)
	vectorConfig := &rag.SQLiteVectorConfig{
		Path:        vectorDBPath,
		Extension:   "sqlite-vec",
		VectorSize:  config.VectorSize,
		CacheSizeMB: 128,
		MMapSizeMB:  256,
		WALEnabled:  true,
		Timeout:     10 * time.Second,
	}

	vectorStore, err := rag.NewSQLiteVectorStore(vectorConfig, logger)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	// Initialize vector store
	if err := vectorStore.Initialize(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize vector store: %w", err)
	}

	storage := &LocalSQLiteStorage{
		db:          db,
		vectorStore: vectorStore,
		userID:      config.UserID,
		mode:        ModeSolo,
		logger:      logger,
	}

	logger.WithFields(logrus.Fields{
		"data_dir": dataDir,
		"user_id":  config.UserID,
		"mode":     "solo",
	}).Info("Local storage initialized")

	return storage, nil
}

// Connection management

func (s *LocalSQLiteStorage) SaveConnection(ctx context.Context, conn *Connection) error {
	if conn.ID == "" {
		conn.ID = uuid.New().String()
	}

	now := time.Now()
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = now
	}
	conn.UpdatedAt = now

	// Encode environments into metadata for storage
	if conn.Metadata == nil {
		conn.Metadata = make(map[string]string)
	}
	if conn.Environments != nil && len(conn.Environments) > 0 {
		envsJSON, err := json.Marshal(conn.Environments)
		if err != nil {
			return fmt.Errorf("failed to marshal environments: %w", err)
		}
		conn.Metadata["environments"] = string(envsJSON)
	} else {
		delete(conn.Metadata, "environments")
	}

	metadataJSON, err := json.Marshal(conn.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO connections (id, name, type, host, port, database_name, username, password_encrypted,
		                        ssl_config, created_by, created_at, updated_at, team_id, is_shared, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			host = excluded.host,
			port = excluded.port,
			database_name = excluded.database_name,
			username = excluded.username,
			password_encrypted = excluded.password_encrypted,
			ssl_config = excluded.ssl_config,
			updated_at = excluded.updated_at,
			metadata = excluded.metadata
	`, conn.ID, conn.Name, conn.Type, conn.Host, conn.Port, conn.DatabaseName, conn.Username,
		conn.PasswordEncrypted, conn.SSLConfig, conn.CreatedBy, conn.CreatedAt.Unix(),
		conn.UpdatedAt.Unix(), conn.TeamID, conn.IsShared, string(metadataJSON))

	return err
}

func (s *LocalSQLiteStorage) GetConnections(ctx context.Context, filters *ConnectionFilters) ([]*Connection, error) {
	query := `SELECT id, name, type, host, port, database_name, username, password_encrypted, ssl_config,
	                 created_by, created_at, updated_at, team_id, is_shared, metadata
	          FROM connections WHERE 1=1`
	args := []interface{}{}

	if filters != nil {
		if filters.TeamID != "" {
			query += " AND team_id = ?"
			args = append(args, filters.TeamID)
		}
		if filters.CreatedBy != "" {
			query += " AND created_by = ?"
			args = append(args, filters.CreatedBy)
		}
		if filters.Type != "" {
			query += " AND type = ?"
			args = append(args, filters.Type)
		}
		if filters.IsShared != nil {
			query += " AND is_shared = ?"
			args = append(args, *filters.IsShared)
		}

		query += " ORDER BY created_at DESC"

		if filters.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filters.Limit)
		}
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query connections: %w", err)
	}
	defer rows.Close()

	var connections []*Connection
	for rows.Next() {
		conn, err := s.scanConnection(rows)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan connection")
			continue
		}
		
		// Apply environment filter if specified
		if filters != nil && len(filters.Environments) > 0 {
			hasMatch := false
			for _, filterEnv := range filters.Environments {
				for _, connEnv := range conn.Environments {
					if connEnv == filterEnv {
						hasMatch = true
						break
					}
				}
				if hasMatch {
					break
				}
			}
			if !hasMatch && len(conn.Environments) > 0 {
				// Skip this connection if it doesn't match any filter environment
				// But include connections with no environments (backward compatibility)
				continue
			}
		}
		
		connections = append(connections, conn)
	}

	return connections, nil
}

func (s *LocalSQLiteStorage) GetConnection(ctx context.Context, id string) (*Connection, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, host, port, database_name, username, password_encrypted, ssl_config,
		       created_by, created_at, updated_at, team_id, is_shared, metadata
		FROM connections WHERE id = ?
	`, id)

	return s.scanConnection(row)
}

func (s *LocalSQLiteStorage) UpdateConnection(ctx context.Context, conn *Connection) error {
	conn.UpdatedAt = time.Now()
	return s.SaveConnection(ctx, conn)
}

func (s *LocalSQLiteStorage) DeleteConnection(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM connections WHERE id = ?`, id)
	return err
}

func (s *LocalSQLiteStorage) GetAvailableEnvironments(ctx context.Context) ([]string, error) {
	connections, err := s.GetConnections(ctx, nil)
	if err != nil {
		return nil, err
	}

	// Collect unique environments
	envSet := make(map[string]bool)
	for _, conn := range connections {
		for _, env := range conn.Environments {
			envSet[env] = true
		}
	}

	// Convert to sorted slice
	environments := make([]string, 0, len(envSet))
	for env := range envSet {
		environments = append(environments, env)
	}

	return environments, nil
}

// Query management

func (s *LocalSQLiteStorage) SaveQuery(ctx context.Context, query *SavedQuery) error {
	if query.ID == "" {
		query.ID = uuid.New().String()
	}

	now := time.Now()
	if query.CreatedAt.IsZero() {
		query.CreatedAt = now
	}
	query.UpdatedAt = now

	tagsJSON, err := json.Marshal(query.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO saved_queries (id, title, query, description, connection_id, created_by, created_at,
		                          updated_at, team_id, is_shared, tags, folder)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			query = excluded.query,
			description = excluded.description,
			connection_id = excluded.connection_id,
			updated_at = excluded.updated_at,
			tags = excluded.tags,
			folder = excluded.folder
	`, query.ID, query.Title, query.Query, query.Description, query.ConnectionID, query.CreatedBy,
		query.CreatedAt.Unix(), query.UpdatedAt.Unix(), query.TeamID, query.IsShared, string(tagsJSON), query.Folder)

	return err
}

func (s *LocalSQLiteStorage) GetQueries(ctx context.Context, filters *QueryFilters) ([]*SavedQuery, error) {
	query := `SELECT id, title, query, description, connection_id, created_by, created_at, updated_at,
	                 team_id, is_shared, tags, folder
	          FROM saved_queries WHERE 1=1`
	args := []interface{}{}

	if filters != nil {
		if filters.TeamID != "" {
			query += " AND team_id = ?"
			args = append(args, filters.TeamID)
		}
		if filters.CreatedBy != "" {
			query += " AND created_by = ?"
			args = append(args, filters.CreatedBy)
		}
		if filters.ConnectionID != "" {
			query += " AND connection_id = ?"
			args = append(args, filters.ConnectionID)
		}
		if filters.Folder != "" {
			query += " AND folder = ?"
			args = append(args, filters.Folder)
		}
		if filters.IsShared != nil {
			query += " AND is_shared = ?"
			args = append(args, *filters.IsShared)
		}

		query += " ORDER BY updated_at DESC"

		if filters.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filters.Limit)
		}
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query saved queries: %w", err)
	}
	defer rows.Close()

	var queries []*SavedQuery
	for rows.Next() {
		q, err := s.scanSavedQuery(rows)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan saved query")
			continue
		}
		queries = append(queries, q)
	}

	return queries, nil
}

func (s *LocalSQLiteStorage) GetQuery(ctx context.Context, id string) (*SavedQuery, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, title, query, description, connection_id, created_by, created_at, updated_at,
		       team_id, is_shared, tags, folder
		FROM saved_queries WHERE id = ?
	`, id)

	return s.scanSavedQuery(row)
}

func (s *LocalSQLiteStorage) UpdateQuery(ctx context.Context, query *SavedQuery) error {
	query.UpdatedAt = time.Now()
	return s.SaveQuery(ctx, query)
}

func (s *LocalSQLiteStorage) DeleteQuery(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM saved_queries WHERE id = ?`, id)
	return err
}

// Query history

func (s *LocalSQLiteStorage) SaveQueryHistory(ctx context.Context, history *QueryHistory) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	if history.ExecutedAt.IsZero() {
		history.ExecutedAt = time.Now()
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO query_history (id, query, connection_id, executed_by, executed_at, duration_ms,
		                          rows_returned, success, error, team_id, is_shared)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, history.ID, history.Query, history.ConnectionID, history.ExecutedBy, history.ExecutedAt.Unix(),
		history.DurationMS, history.RowsReturned, history.Success, history.Error, history.TeamID, history.IsShared)

	return err
}

func (s *LocalSQLiteStorage) GetQueryHistory(ctx context.Context, filters *HistoryFilters) ([]*QueryHistory, error) {
	query := `SELECT id, query, connection_id, executed_by, executed_at, duration_ms, rows_returned,
	                 success, error, team_id, is_shared
	          FROM query_history WHERE 1=1`
	args := []interface{}{}

	if filters != nil {
		if filters.TeamID != "" {
			query += " AND team_id = ?"
			args = append(args, filters.TeamID)
		}
		if filters.ExecutedBy != "" {
			query += " AND executed_by = ?"
			args = append(args, filters.ExecutedBy)
		}
		if filters.ConnectionID != "" {
			query += " AND connection_id = ?"
			args = append(args, filters.ConnectionID)
		}
		if filters.Success != nil {
			query += " AND success = ?"
			args = append(args, *filters.Success)
		}
		if filters.StartDate != nil {
			query += " AND executed_at >= ?"
			args = append(args, filters.StartDate.Unix())
		}
		if filters.EndDate != nil {
			query += " AND executed_at <= ?"
			args = append(args, filters.EndDate.Unix())
		}

		query += " ORDER BY executed_at DESC"

		if filters.Limit > 0 {
			query += " LIMIT ?"
			args = append(args, filters.Limit)
		}
		if filters.Offset > 0 {
			query += " OFFSET ?"
			args = append(args, filters.Offset)
		}
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query history: %w", err)
	}
	defer rows.Close()

	var history []*QueryHistory
	for rows.Next() {
		h, err := s.scanQueryHistory(rows)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan query history")
			continue
		}
		history = append(history, h)
	}

	return history, nil
}

func (s *LocalSQLiteStorage) DeleteQueryHistory(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM query_history WHERE id = ?`, id)
	return err
}

// Vector/RAG operations (delegate to vector store)

func (s *LocalSQLiteStorage) IndexDocument(ctx context.Context, doc *Document) error {
	// Convert storage.Document to rag.Document
	ragDoc := &rag.Document{
		ID:           doc.ID,
		ConnectionID: doc.ConnectionID,
		Type:         rag.DocumentType(doc.Type),
		Content:      doc.Content,
		Embedding:    doc.Embedding,
		Metadata:     doc.Metadata,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
		AccessCount:  doc.AccessCount,
		LastAccessed: doc.LastAccessed,
	}
	return s.vectorStore.IndexDocument(ctx, ragDoc)
}

func (s *LocalSQLiteStorage) SearchDocuments(ctx context.Context, embedding []float32, filters *DocumentFilters) ([]*Document, error) {
	ragFilters := make(map[string]interface{})
	if filters != nil {
		if filters.ConnectionID != "" {
			ragFilters["connection_id"] = filters.ConnectionID
		}
		if filters.Type != "" {
			ragFilters["type"] = filters.Type
		}
	}

	limit := 10
	if filters != nil && filters.Limit > 0 {
		limit = filters.Limit
	}

	ragDocs, err := s.vectorStore.SearchSimilar(ctx, embedding, limit, ragFilters)
	if err != nil {
		return nil, err
	}

	// Convert rag.Document slice to storage.Document slice
	docs := make([]*Document, len(ragDocs))
	for i, ragDoc := range ragDocs {
		docs[i] = &Document{
			ID:           ragDoc.ID,
			ConnectionID: ragDoc.ConnectionID,
			Type:         string(ragDoc.Type),
			Content:      ragDoc.Content,
			Embedding:    ragDoc.Embedding,
			Metadata:     ragDoc.Metadata,
			CreatedAt:    ragDoc.CreatedAt,
			UpdatedAt:    ragDoc.UpdatedAt,
			AccessCount:  ragDoc.AccessCount,
			LastAccessed: ragDoc.LastAccessed,
			Score:        ragDoc.Score,
		}
	}
	return docs, nil
}

func (s *LocalSQLiteStorage) GetDocument(ctx context.Context, id string) (*Document, error) {
	ragDoc, err := s.vectorStore.GetDocument(ctx, id)
	if err != nil {
		return nil, err
	}

	return &Document{
		ID:           ragDoc.ID,
		ConnectionID: ragDoc.ConnectionID,
		Type:         string(ragDoc.Type),
		Content:      ragDoc.Content,
		Embedding:    ragDoc.Embedding,
		Metadata:     ragDoc.Metadata,
		CreatedAt:    ragDoc.CreatedAt,
		UpdatedAt:    ragDoc.UpdatedAt,
		AccessCount:  ragDoc.AccessCount,
		LastAccessed: ragDoc.LastAccessed,
	}, nil
}

func (s *LocalSQLiteStorage) DeleteDocument(ctx context.Context, id string) error {
	return s.vectorStore.DeleteDocument(ctx, id)
}

// Schema cache

func (s *LocalSQLiteStorage) CacheSchema(ctx context.Context, connID string, schema *SchemaCache) error {
	schemaJSON, err := json.Marshal(schema.SchemaData)
	if err != nil {
		return fmt.Errorf("failed to marshal schema data: %w", err)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO schema_cache (connection_id, schema_data, cached_at, expires_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(connection_id) DO UPDATE SET
			schema_data = excluded.schema_data,
			cached_at = excluded.cached_at,
			expires_at = excluded.expires_at
	`, connID, string(schemaJSON), schema.CachedAt.Unix(), schema.ExpiresAt.Unix())

	return err
}

func (s *LocalSQLiteStorage) GetCachedSchema(ctx context.Context, connID string) (*SchemaCache, error) {
	var schemaDataStr string
	var cachedAt, expiresAt int64

	err := s.db.QueryRowContext(ctx, `
		SELECT schema_data, cached_at, expires_at
		FROM schema_cache
		WHERE connection_id = ?
	`, connID).Scan(&schemaDataStr, &cachedAt, &expiresAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cached schema: %w", err)
	}

	// Check if expired
	if time.Now().Unix() > expiresAt {
		// Delete expired cache
		s.InvalidateSchemaCache(ctx, connID)
		return nil, nil
	}

	var schemaData map[string]interface{}
	if err := json.Unmarshal([]byte(schemaDataStr), &schemaData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema data: %w", err)
	}

	return &SchemaCache{
		ConnectionID: connID,
		SchemaData:   schemaData,
		CachedAt:     time.Unix(cachedAt, 0),
		ExpiresAt:    time.Unix(expiresAt, 0),
	}, nil
}

func (s *LocalSQLiteStorage) InvalidateSchemaCache(ctx context.Context, connID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM schema_cache WHERE connection_id = ?`, connID)
	return err
}

// Settings

func (s *LocalSQLiteStorage) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func (s *LocalSQLiteStorage) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, scope, user_id)
		VALUES (?, ?, 'user', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`, key, value, s.userID)
	return err
}

func (s *LocalSQLiteStorage) DeleteSetting(ctx context.Context, key string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	return err
}

// Team operations (no-op in local mode)

func (s *LocalSQLiteStorage) GetTeam(ctx context.Context) (*Team, error) {
	return nil, nil // No team in solo mode
}

func (s *LocalSQLiteStorage) GetTeamMembers(ctx context.Context) ([]*TeamMember, error) {
	return nil, nil // No team in solo mode
}

// Lifecycle

func (s *LocalSQLiteStorage) Close() error {
	if err := s.db.Close(); err != nil {
		s.logger.WithError(err).Error("Failed to close database")
		return err
	}
	s.logger.Info("Local storage closed")
	return nil
}

// Mode information

func (s *LocalSQLiteStorage) GetMode() Mode {
	return s.mode
}

func (s *LocalSQLiteStorage) GetUserID() string {
	return s.userID
}

// Helper scanning functions

type scanner interface {
	Scan(dest ...interface{}) error
}

func (s *LocalSQLiteStorage) scanConnection(row scanner) (*Connection, error) {
	var id, name, connType, host, dbName, username, passwordEnc, sslConfig, createdBy, teamID, metadataStr string
	var port int
	var createdAt, updatedAt int64
	var isShared bool

	err := row.Scan(&id, &name, &connType, &host, &port, &dbName, &username, &passwordEnc,
		&sslConfig, &createdBy, &createdAt, &updatedAt, &teamID, &isShared, &metadataStr)
	if err != nil {
		return nil, err
	}

	var metadata map[string]string
	if metadataStr != "" {
		if err := json.Unmarshal([]byte(metadataStr), &metadata); err != nil {
			metadata = make(map[string]string)
		}
	}

	// Extract environments from metadata
	var environments []string
	if metadata != nil {
		if envsStr, ok := metadata["environments"]; ok && envsStr != "" {
			if err := json.Unmarshal([]byte(envsStr), &environments); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal environments")
			}
		}
	}

	return &Connection{
		ID:                id,
		Name:              name,
		Type:              connType,
		Host:              host,
		Port:              port,
		DatabaseName:      dbName,
		Username:          username,
		PasswordEncrypted: passwordEnc,
		SSLConfig:         sslConfig,
		CreatedBy:         createdBy,
		CreatedAt:         time.Unix(createdAt, 0),
		UpdatedAt:         time.Unix(updatedAt, 0),
		TeamID:            teamID,
		IsShared:          isShared,
		Environments:      environments,
		Metadata:          metadata,
	}, nil
}

func (s *LocalSQLiteStorage) scanSavedQuery(row scanner) (*SavedQuery, error) {
	var id, title, queryStr, description, connID, createdBy, teamID, tagsStr, folder string
	var createdAt, updatedAt int64
	var isShared bool

	err := row.Scan(&id, &title, &queryStr, &description, &connID, &createdBy,
		&createdAt, &updatedAt, &teamID, &isShared, &tagsStr, &folder)
	if err != nil {
		return nil, err
	}

	var tags []string
	if tagsStr != "" {
		if err := json.Unmarshal([]byte(tagsStr), &tags); err != nil {
			tags = []string{}
		}
	}

	return &SavedQuery{
		ID:           id,
		Title:        title,
		Query:        queryStr,
		Description:  description,
		ConnectionID: connID,
		CreatedBy:    createdBy,
		CreatedAt:    time.Unix(createdAt, 0),
		UpdatedAt:    time.Unix(updatedAt, 0),
		TeamID:       teamID,
		IsShared:     isShared,
		Tags:         tags,
		Folder:       folder,
	}, nil
}

func (s *LocalSQLiteStorage) scanQueryHistory(row scanner) (*QueryHistory, error) {
	var id, queryStr, connID, executedBy, errorStr, teamID string
	var executedAt, durationMS, rowsReturned int64
	var success, isShared bool

	err := row.Scan(&id, &queryStr, &connID, &executedBy, &executedAt, &durationMS,
		&rowsReturned, &success, &errorStr, &teamID, &isShared)
	if err != nil {
		return nil, err
	}

	return &QueryHistory{
		ID:           id,
		Query:        queryStr,
		ConnectionID: connID,
		ExecutedBy:   executedBy,
		ExecutedAt:   time.Unix(executedAt, 0),
		DurationMS:   durationMS,
		RowsReturned: rowsReturned,
		Success:      success,
		Error:        errorStr,
		TeamID:       teamID,
		IsShared:     isShared,
	}, nil
}

