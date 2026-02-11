package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/jbeck018/howlerops/services"
)

// StorageService handles storage, migration, and synthetic view operations for Wails v3
type StorageService struct {
	deps             *SharedDeps
	storageMigration *services.StorageMigrationService
	syntheticViews   *storage.SyntheticViewStorage
	ctx              context.Context
}

// NewStorageService creates a new StorageService instance
func NewStorageService(
	deps *SharedDeps,
	storageMigration *services.StorageMigrationService,
	syntheticViews *storage.SyntheticViewStorage,
	ctx context.Context,
) *StorageService {
	return &StorageService{
		deps:             deps,
		storageMigration: storageMigration,
		syntheticViews:   syntheticViews,
		ctx:              ctx,
	}
}

// =============================================================================
// Synthetic Views Methods
// =============================================================================

// SaveSyntheticView creates or updates a synthetic view definition
func (s *StorageService) SaveSyntheticView(viewDef storage.ViewDefinition) (string, error) {
	if s.syntheticViews == nil {
		return "", fmt.Errorf("synthetic views storage not initialized")
	}

	if viewDef.ID == "" {
		viewDef.ID = fmt.Sprintf("view_%d", time.Now().UnixNano())
	}

	if viewDef.Version == "" {
		viewDef.Version = "1.0.0"
	}

	if err := s.syntheticViews.SaveSyntheticView(&viewDef); err != nil {
		return "", fmt.Errorf("failed to save synthetic view: %w", err)
	}

	s.deps.Logger.WithField("view_id", viewDef.ID).Info("Synthetic view saved")
	return viewDef.ID, nil
}

// ListSyntheticViews returns a list of all synthetic views
func (s *StorageService) ListSyntheticViews() ([]SyntheticViewSummary, error) {
	if s.syntheticViews == nil {
		return []SyntheticViewSummary{}, nil
	}

	views, err := s.syntheticViews.ListSyntheticViews()
	if err != nil {
		return nil, fmt.Errorf("failed to list synthetic views: %w", err)
	}

	summaries := make([]SyntheticViewSummary, 0, len(views))
	for _, view := range views {
		createdAt := ""
		if !view.CreatedAt.IsZero() {
			createdAt = view.CreatedAt.Format(time.RFC3339)
		}
		updatedAt := ""
		if !view.UpdatedAt.IsZero() {
			updatedAt = view.UpdatedAt.Format(time.RFC3339)
		}

		summaries = append(summaries, SyntheticViewSummary{
			ID:          view.ID,
			Name:        view.Name,
			Description: view.Description,
			Version:     view.Version,
			CreatedAt:   createdAt,
			UpdatedAt:   updatedAt,
		})
	}

	return summaries, nil
}

// GetSyntheticView retrieves a synthetic view by ID
func (s *StorageService) GetSyntheticView(id string) (*storage.ViewDefinition, error) {
	if s.syntheticViews == nil {
		return nil, fmt.Errorf("synthetic views storage not initialized")
	}

	view, err := s.syntheticViews.GetSyntheticView(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get synthetic view: %w", err)
	}

	return view, nil
}

// DeleteSyntheticView deletes a synthetic view by ID
func (s *StorageService) DeleteSyntheticView(id string) error {
	if s.syntheticViews == nil {
		return fmt.Errorf("synthetic views storage not initialized")
	}

	if err := s.syntheticViews.DeleteSyntheticView(id); err != nil {
		return fmt.Errorf("failed to delete synthetic view: %w", err)
	}

	s.deps.Logger.WithField("view_id", id).Info("Synthetic view deleted")
	return nil
}

// ExecuteSyntheticQuery executes a query against synthetic views
func (s *StorageService) ExecuteSyntheticQuery(sql string) (*QueryResponse, error) {
	if s.deps.DuckDBEngine == nil {
		return nil, fmt.Errorf("DuckDB federation engine not initialized")
	}

	// Check if query targets synthetic schema
	if !strings.Contains(strings.ToLower(sql), "synthetic.") {
		return nil, fmt.Errorf("query does not target synthetic schema")
	}

	// Check for DML operations (INSERT, UPDATE, DELETE, DDL)
	sqlLower := strings.ToLower(strings.TrimSpace(sql))
	if strings.HasPrefix(sqlLower, "insert") ||
		strings.HasPrefix(sqlLower, "update") ||
		strings.HasPrefix(sqlLower, "delete") ||
		strings.HasPrefix(sqlLower, "create") ||
		strings.HasPrefix(sqlLower, "alter") ||
		strings.HasPrefix(sqlLower, "drop") {
		return nil, fmt.Errorf("DML/DDL operations are not allowed on synthetic schema")
	}

	// Execute query with timeout
	timeout := 30 * time.Second
	result, err := s.deps.DuckDBEngine.ExecuteQuery(s.ctx, sql, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to execute synthetic query: %w", err)
	}

	// Convert result to QueryResponse format
	response := &QueryResponse{
		Columns:  result.Columns,
		Rows:     result.Rows,
		RowCount: int64(result.RowCount),
		Duration: result.Duration.String(),
	}

	return response, nil
}

// GetSyntheticSchema returns the schema information for synthetic views
func (s *StorageService) GetSyntheticSchema() (map[string]interface{}, error) {
	if s.syntheticViews == nil {
		return map[string]interface{}{
			"schema": "synthetic",
			"views":  []interface{}{},
		}, nil
	}

	schema, err := s.syntheticViews.GetSyntheticSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to get synthetic schema: %w", err)
	}

	return schema, nil
}

// =============================================================================
// Storage Migration API - Wails bindings for IndexedDB to SQLite migration
// =============================================================================

// StorageMigrationStatus returns the current migration status
func (s *StorageService) StorageMigrationStatus() (*services.MigrationStatus, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}
	return s.storageMigration.GetMigrationStatus(s.ctx)
}

// StorageImportConnections imports connections from IndexedDB format to SQLite
func (s *StorageService) StorageImportConnections(connectionsJSON string) (*services.ImportConnectionsResult, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	var connections []services.ConnectionImport
	if err := json.Unmarshal([]byte(connectionsJSON), &connections); err != nil {
		return nil, fmt.Errorf("failed to parse connections JSON: %w", err)
	}

	userID := "local-user"
	if s.deps.StorageManager != nil {
		userID = s.deps.StorageManager.GetUserID()
	}

	return s.storageMigration.ImportConnections(s.ctx, connections, userID)
}

// StorageImportQueries imports saved queries from IndexedDB format to SQLite
func (s *StorageService) StorageImportQueries(queriesJSON string) (*services.ImportQueriesResult, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	var queries []services.SavedQueryImport
	if err := json.Unmarshal([]byte(queriesJSON), &queries); err != nil {
		return nil, fmt.Errorf("failed to parse queries JSON: %w", err)
	}

	userID := "local-user"
	if s.deps.StorageManager != nil {
		userID = s.deps.StorageManager.GetUserID()
	}

	return s.storageMigration.ImportQueries(s.ctx, queries, userID)
}

// StorageImportHistory imports query history from IndexedDB format to SQLite
func (s *StorageService) StorageImportHistory(historyJSON string) (*services.ImportHistoryResult, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	var history []services.QueryHistoryImport
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		return nil, fmt.Errorf("failed to parse history JSON: %w", err)
	}

	userID := "local-user"
	if s.deps.StorageManager != nil {
		userID = s.deps.StorageManager.GetUserID()
	}

	return s.storageMigration.ImportQueryHistory(s.ctx, history, userID)
}

// StorageImportPreferences imports preferences from IndexedDB format to SQLite
func (s *StorageService) StorageImportPreferences(preferencesJSON string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}

	var preferences services.PreferencesImport
	if err := json.Unmarshal([]byte(preferencesJSON), &preferences); err != nil {
		return fmt.Errorf("failed to parse preferences JSON: %w", err)
	}

	return s.storageMigration.ImportPreferences(s.ctx, preferences)
}

// StorageCompleteMigration marks the IndexedDB to SQLite migration as complete
func (s *StorageService) StorageCompleteMigration() error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}
	return s.storageMigration.CompleteMigration(s.ctx)
}

// =============================================================================
// SQLite Storage CRUD API - Direct SQLite access for frontend
// =============================================================================

// SQLiteConnection represents a connection for the frontend (without encrypted fields)
type SQLiteConnection struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Host         string            `json:"host"`
	Port         int               `json:"port"`
	Database     string            `json:"database"`
	Username     string            `json:"username"`
	SSLConfig    map[string]string `json:"ssl_config"`
	Environments []string          `json:"environments"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
}

// SQLiteGetConnections returns all connections from SQLite
func (s *StorageService) SQLiteGetConnections() ([]SQLiteConnection, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	connections, err := s.storageMigration.GetAllConnections(s.ctx)
	if err != nil {
		return nil, err
	}

	result := make([]SQLiteConnection, len(connections))
	for i, conn := range connections {
		result[i] = SQLiteConnection{
			ID:           conn.ID,
			Name:         conn.Name,
			Type:         conn.Type,
			Host:         conn.Host,
			Port:         conn.Port,
			Database:     conn.DatabaseName,
			Username:     conn.Username,
			SSLConfig:    conn.SSLConfig,
			Environments: conn.Environments,
			CreatedAt:    conn.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    conn.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// SQLiteGetConnection returns a single connection by ID
func (s *StorageService) SQLiteGetConnection(id string) (*SQLiteConnection, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	conn, err := s.storageMigration.GetConnection(s.ctx, id)
	if err != nil {
		return nil, err
	}
	if conn == nil {
		return nil, nil
	}

	return &SQLiteConnection{
		ID:           conn.ID,
		Name:         conn.Name,
		Type:         conn.Type,
		Host:         conn.Host,
		Port:         conn.Port,
		Database:     conn.DatabaseName,
		Username:     conn.Username,
		SSLConfig:    conn.SSLConfig,
		Environments: conn.Environments,
		CreatedAt:    conn.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    conn.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// SQLiteSavedQuery represents a saved query for the frontend
type SQLiteSavedQuery struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Query        string   `json:"query"`
	ConnectionID string   `json:"connection_id"`
	Folder       string   `json:"folder"`
	Tags         []string `json:"tags"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// SQLiteGetQueries returns all saved queries from SQLite
func (s *StorageService) SQLiteGetQueries() ([]SQLiteSavedQuery, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	queries, err := s.storageMigration.GetAllQueries(s.ctx)
	if err != nil {
		return nil, err
	}

	result := make([]SQLiteSavedQuery, len(queries))
	for i, q := range queries {
		result[i] = SQLiteSavedQuery{
			ID:           q.ID,
			Title:        q.Title,
			Description:  q.Description,
			Query:        q.Query,
			ConnectionID: q.ConnectionID,
			Folder:       q.Folder,
			Tags:         q.Tags,
			CreatedAt:    q.CreatedAt.Format(time.RFC3339),
			UpdatedAt:    q.UpdatedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// SQLiteGetQuery returns a single query by ID
func (s *StorageService) SQLiteGetQuery(id string) (*SQLiteSavedQuery, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	q, err := s.storageMigration.GetQuery(s.ctx, id)
	if err != nil {
		return nil, err
	}
	if q == nil {
		return nil, nil
	}

	return &SQLiteSavedQuery{
		ID:           q.ID,
		Title:        q.Title,
		Description:  q.Description,
		Query:        q.Query,
		ConnectionID: q.ConnectionID,
		Folder:       q.Folder,
		Tags:         q.Tags,
		CreatedAt:    q.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    q.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// SQLiteQueryHistory represents a query history entry for the frontend
type SQLiteQueryHistory struct {
	ID           string `json:"id"`
	ConnectionID string `json:"connection_id"`
	Query        string `json:"query"`
	DurationMS   int    `json:"duration_ms"`
	RowCount     int    `json:"row_count"`
	Success      bool   `json:"success"`
	Error        string `json:"error"`
	ExecutedAt   string `json:"executed_at"`
}

// SQLiteGetQueryHistory returns query history from SQLite
func (s *StorageService) SQLiteGetQueryHistory(connectionID string, limit int) ([]SQLiteQueryHistory, error) {
	if s.storageMigration == nil {
		return nil, fmt.Errorf("storage migration service not initialized")
	}

	if limit <= 0 {
		limit = 100
	}

	history, err := s.storageMigration.GetQueryHistory(s.ctx, connectionID, limit)
	if err != nil {
		return nil, err
	}

	result := make([]SQLiteQueryHistory, len(history))
	for i, h := range history {
		result[i] = SQLiteQueryHistory{
			ID:           h.ID,
			ConnectionID: h.ConnectionID,
			Query:        h.Query,
			DurationMS:   h.DurationMS,
			RowCount:     h.RowsReturned,
			Success:      h.Success,
			Error:        h.Error,
			ExecutedAt:   h.ExecutedAt.Format(time.RFC3339),
		}
	}

	return result, nil
}

// SQLiteGetSetting retrieves a setting from SQLite
func (s *StorageService) SQLiteGetSetting(key string) (string, error) {
	if s.storageMigration == nil {
		return "", fmt.Errorf("storage migration service not initialized")
	}
	return s.storageMigration.GetSetting(s.ctx, key)
}

// SQLiteSetSetting saves a setting to SQLite
func (s *StorageService) SQLiteSetSetting(key, value string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}
	return s.storageMigration.SetSetting(s.ctx, key, value)
}

// SQLiteSaveConnection saves or updates a connection in SQLite
// Accepts JSON string and parses it into storage.Connection
func (s *StorageService) SQLiteSaveConnection(connectionJSON string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}

	var conn storage.Connection
	if err := json.Unmarshal([]byte(connectionJSON), &conn); err != nil {
		return fmt.Errorf("invalid connection JSON: %w", err)
	}

	// Set timestamps if not provided
	now := time.Now()
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = now
	}
	conn.UpdatedAt = now

	return s.storageMigration.SaveConnection(s.ctx, &conn)
}

// SQLiteDeleteConnection removes a connection from SQLite by ID
func (s *StorageService) SQLiteDeleteConnection(id string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}
	return s.storageMigration.DeleteConnection(s.ctx, id)
}

// SQLiteSaveQuery saves or updates a query in SQLite
// Accepts JSON string and parses it into storage.SavedQuery
func (s *StorageService) SQLiteSaveQuery(queryJSON string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}

	var query storage.SavedQuery
	if err := json.Unmarshal([]byte(queryJSON), &query); err != nil {
		return fmt.Errorf("invalid query JSON: %w", err)
	}

	// Set timestamps if not provided
	now := time.Now()
	if query.CreatedAt.IsZero() {
		query.CreatedAt = now
	}
	query.UpdatedAt = now

	return s.storageMigration.SaveQuery(s.ctx, &query)
}

// SQLiteDeleteQuery removes a saved query from SQLite by ID
func (s *StorageService) SQLiteDeleteQuery(id string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}
	return s.storageMigration.DeleteQuery(s.ctx, id)
}

// SQLiteSaveQueryHistory saves a query history entry to SQLite
// Accepts JSON string and parses it into storage.QueryHistory
func (s *StorageService) SQLiteSaveQueryHistory(historyJSON string) error {
	if s.storageMigration == nil {
		return fmt.Errorf("storage migration service not initialized")
	}

	var history storage.QueryHistory
	if err := json.Unmarshal([]byte(historyJSON), &history); err != nil {
		return fmt.Errorf("invalid history JSON: %w", err)
	}

	// Set executed timestamp if not provided
	if history.ExecutedAt.IsZero() {
		history.ExecutedAt = time.Now()
	}

	return s.storageMigration.SaveQueryHistory(s.ctx, &history)
}
