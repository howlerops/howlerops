package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jbeck018/howlerops/pkg/storage"
	"github.com/sirupsen/logrus"
)

// StorageMigrationService handles migration from IndexedDB to SQLite
type StorageMigrationService struct {
	storageManager *storage.Manager
	logger         *logrus.Logger
}

// NewStorageMigrationService creates a new storage migration service
func NewStorageMigrationService(sm *storage.Manager, logger *logrus.Logger) *StorageMigrationService {
	return &StorageMigrationService{
		storageManager: sm,
		logger:         logger,
	}
}

// MigrationStatus represents the current migration state
type MigrationStatus struct {
	SQLiteHasData    bool `json:"sqlite_has_data"`
	MigrationDone    bool `json:"migration_done"`
	ConnectionCount  int  `json:"connection_count"`
	QueryCount       int  `json:"query_count"`
	HistoryCount     int  `json:"history_count"`
	PreferencesCount int  `json:"preferences_count"`
}

// GetMigrationStatus checks if migration is needed
func (s *StorageMigrationService) GetMigrationStatus(ctx context.Context) (*MigrationStatus, error) {
	status := &MigrationStatus{}

	// Check if SQLite has data
	connections, err := s.storageManager.GetConnections(ctx, nil)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to check connections count")
	} else {
		status.ConnectionCount = len(connections)
	}

	queries, err := s.storageManager.GetQueries(ctx, nil)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to check queries count")
	} else {
		status.QueryCount = len(queries)
	}

	history, err := s.storageManager.GetQueryHistory(ctx, &storage.HistoryFilters{Limit: 1})
	if err != nil {
		s.logger.WithError(err).Warn("Failed to check history count")
	} else {
		status.HistoryCount = len(history)
	}

	// Check migration flag
	migrationDone, _ := s.storageManager.GetSetting(ctx, "migration_from_indexeddb_complete")
	status.MigrationDone = migrationDone == "true"

	// SQLite has data if any count is > 0
	status.SQLiteHasData = status.ConnectionCount > 0 || status.QueryCount > 0 || status.HistoryCount > 0

	return status, nil
}

// ConnectionImport represents a connection to import from IndexedDB
type ConnectionImport struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	Database     string   `json:"database"`
	Username     string   `json:"username"`
	SSLMode      string   `json:"ssl_mode"`
	Environments []string `json:"environments"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// ImportConnectionsResult contains the result of importing connections
type ImportConnectionsResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ImportConnections imports connections from IndexedDB format to SQLite
func (s *StorageMigrationService) ImportConnections(ctx context.Context, connections []ConnectionImport, userID string) (*ImportConnectionsResult, error) {
	result := &ImportConnectionsResult{
		Errors: []string{},
	}

	for _, conn := range connections {
		// Check if connection already exists
		existing, err := s.storageManager.GetConnection(ctx, conn.ID)
		if err == nil && existing != nil {
			result.Skipped++
			continue
		}

		// Parse timestamps
		createdAt := time.Now()
		updatedAt := time.Now()
		if conn.CreatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, conn.CreatedAt); err == nil {
				createdAt = parsed
			}
		}
		if conn.UpdatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, conn.UpdatedAt); err == nil {
				updatedAt = parsed
			}
		}

		// Build SSL config
		sslConfig := make(map[string]string)
		if conn.SSLMode != "" {
			sslConfig["mode"] = conn.SSLMode
		}

		// Create connection
		storageConn := &storage.Connection{
			ID:           conn.ID,
			Name:         conn.Name,
			Type:         conn.Type,
			Host:         conn.Host,
			Port:         conn.Port,
			DatabaseName: conn.Database,
			Username:     conn.Username,
			SSLConfig:    sslConfig,
			Environments: conn.Environments,
			CreatedBy:    userID,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		}

		if err := s.storageManager.SaveConnection(ctx, storageConn); err != nil {
			result.Errors = append(result.Errors, "Connection "+conn.Name+": "+err.Error())
			continue
		}

		result.Imported++
		s.logger.WithField("connection_id", conn.ID).Info("Imported connection from IndexedDB")
	}

	return result, nil
}

// SavedQueryImport represents a saved query to import from IndexedDB
type SavedQueryImport struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Name         string   `json:"name"` // Alias
	Description  string   `json:"description"`
	QueryText    string   `json:"query_text"`
	Query        string   `json:"query"` // Alias
	ConnectionID string   `json:"connection_id"`
	Folder       string   `json:"folder"`
	Tags         []string `json:"tags"`
	IsFavorite   bool     `json:"is_favorite"`
	CreatedAt    string   `json:"created_at"`
	UpdatedAt    string   `json:"updated_at"`
}

// ImportQueriesResult contains the result of importing queries
type ImportQueriesResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ImportQueries imports saved queries from IndexedDB format to SQLite
func (s *StorageMigrationService) ImportQueries(ctx context.Context, queries []SavedQueryImport, userID string) (*ImportQueriesResult, error) {
	result := &ImportQueriesResult{
		Errors: []string{},
	}

	for _, q := range queries {
		// Check if query already exists
		existing, err := s.storageManager.GetQuery(ctx, q.ID)
		if err == nil && existing != nil {
			result.Skipped++
			continue
		}

		// Parse timestamps
		createdAt := time.Now()
		updatedAt := time.Now()
		if q.CreatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, q.CreatedAt); err == nil {
				createdAt = parsed
			}
		}
		if q.UpdatedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, q.UpdatedAt); err == nil {
				updatedAt = parsed
			}
		}

		// Handle title/name alias
		title := q.Title
		if title == "" {
			title = q.Name
		}

		// Handle query/query_text alias
		queryText := q.QueryText
		if queryText == "" {
			queryText = q.Query
		}

		// Create query
		storageQuery := &storage.SavedQuery{
			ID:           q.ID,
			Title:        title,
			Description:  q.Description,
			Query:        queryText,
			ConnectionID: q.ConnectionID,
			Folder:       q.Folder,
			Tags:         q.Tags,
			CreatedBy:    userID,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		}

		if err := s.storageManager.SaveQuery(ctx, storageQuery); err != nil {
			result.Errors = append(result.Errors, "Query "+title+": "+err.Error())
			continue
		}

		result.Imported++
		s.logger.WithField("query_id", q.ID).Info("Imported saved query from IndexedDB")
	}

	return result, nil
}

// QueryHistoryImport represents a query history entry to import
type QueryHistoryImport struct {
	ID           string `json:"id"`
	ConnectionID string `json:"connection_id"`
	QueryText    string `json:"query_text"`
	Query        string `json:"query"` // Alias
	DurationMS   int    `json:"duration_ms"`
	RowCount     int    `json:"row_count"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	ExecutedAt   string `json:"executed_at"`
}

// ImportHistoryResult contains the result of importing history
type ImportHistoryResult struct {
	Imported int      `json:"imported"`
	Skipped  int      `json:"skipped"`
	Errors   []string `json:"errors"`
}

// ImportQueryHistory imports query history from IndexedDB format to SQLite
func (s *StorageMigrationService) ImportQueryHistory(ctx context.Context, history []QueryHistoryImport, userID string) (*ImportHistoryResult, error) {
	result := &ImportHistoryResult{
		Errors: []string{},
	}

	for _, h := range history {
		// Generate ID if empty
		historyID := h.ID
		if historyID == "" {
			historyID = uuid.New().String()
		}

		// Parse timestamp
		executedAt := time.Now()
		if h.ExecutedAt != "" {
			if parsed, err := time.Parse(time.RFC3339, h.ExecutedAt); err == nil {
				executedAt = parsed
			}
		}

		// Handle query/query_text alias
		queryText := h.QueryText
		if queryText == "" {
			queryText = h.Query
		}

		// Determine success
		success := h.Status == "success" || h.Status == ""

		// Create history entry
		storageHistory := &storage.QueryHistory{
			ID:           historyID,
			Query:        queryText,
			ConnectionID: h.ConnectionID,
			ExecutedAt:   executedAt,
			DurationMS:   h.DurationMS,
			RowsReturned: h.RowCount,
			Success:      success,
			Error:        h.ErrorMessage,
			ExecutedBy:   userID,
		}

		if err := s.storageManager.SaveQueryHistory(ctx, storageHistory); err != nil {
			result.Errors = append(result.Errors, "History entry: "+err.Error())
			continue
		}

		result.Imported++
	}

	s.logger.WithField("count", result.Imported).Info("Imported query history from IndexedDB")
	return result, nil
}

// PreferencesImport represents preferences to import
type PreferencesImport map[string]interface{}

// ImportPreferences imports preferences from IndexedDB format to SQLite
func (s *StorageMigrationService) ImportPreferences(ctx context.Context, preferences PreferencesImport) error {
	for key, value := range preferences {
		valueJSON, err := json.Marshal(value)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to marshal preference %s", key)
			continue
		}

		if err := s.storageManager.SetSetting(ctx, "pref_"+key, string(valueJSON)); err != nil {
			s.logger.WithError(err).Warnf("Failed to save preference %s", key)
			continue
		}
	}

	s.logger.Info("Imported preferences from IndexedDB")
	return nil
}

// CompleteMigration marks the migration as complete
func (s *StorageMigrationService) CompleteMigration(ctx context.Context) error {
	if err := s.storageManager.SetSetting(ctx, "migration_from_indexeddb_complete", "true"); err != nil {
		return err
	}
	if err := s.storageManager.SetSetting(ctx, "migration_timestamp", time.Now().Format(time.RFC3339)); err != nil {
		return err
	}
	s.logger.Info("IndexedDB to SQLite migration marked complete")
	return nil
}

// SQLite CRUD operations for frontend

// GetAllConnections returns all connections from SQLite
func (s *StorageMigrationService) GetAllConnections(ctx context.Context) ([]*storage.Connection, error) {
	return s.storageManager.GetConnections(ctx, nil)
}

// GetConnection returns a single connection by ID
func (s *StorageMigrationService) GetConnection(ctx context.Context, id string) (*storage.Connection, error) {
	return s.storageManager.GetConnection(ctx, id)
}

// SaveConnection saves a connection to SQLite
func (s *StorageMigrationService) SaveConnection(ctx context.Context, conn *storage.Connection) error {
	return s.storageManager.SaveConnection(ctx, conn)
}

// DeleteConnection deletes a connection from SQLite
func (s *StorageMigrationService) DeleteConnection(ctx context.Context, id string) error {
	return s.storageManager.DeleteConnection(ctx, id)
}

// GetAllQueries returns all saved queries from SQLite
func (s *StorageMigrationService) GetAllQueries(ctx context.Context) ([]*storage.SavedQuery, error) {
	return s.storageManager.GetQueries(ctx, nil)
}

// GetQuery returns a single query by ID
func (s *StorageMigrationService) GetQuery(ctx context.Context, id string) (*storage.SavedQuery, error) {
	return s.storageManager.GetQuery(ctx, id)
}

// SaveQuery saves a query to SQLite
func (s *StorageMigrationService) SaveQuery(ctx context.Context, query *storage.SavedQuery) error {
	return s.storageManager.SaveQuery(ctx, query)
}

// DeleteQuery deletes a query from SQLite
func (s *StorageMigrationService) DeleteQuery(ctx context.Context, id string) error {
	return s.storageManager.DeleteQuery(ctx, id)
}

// GetQueryHistory returns query history from SQLite
func (s *StorageMigrationService) GetQueryHistory(ctx context.Context, connectionID string, limit int) ([]*storage.QueryHistory, error) {
	filters := &storage.HistoryFilters{
		ConnectionID: connectionID,
		Limit:        limit,
	}
	return s.storageManager.GetQueryHistory(ctx, filters)
}

// SaveQueryHistory saves a history entry to SQLite
func (s *StorageMigrationService) SaveQueryHistory(ctx context.Context, history *storage.QueryHistory) error {
	return s.storageManager.SaveQueryHistory(ctx, history)
}

// GetSetting retrieves a setting from SQLite
func (s *StorageMigrationService) GetSetting(ctx context.Context, key string) (string, error) {
	return s.storageManager.GetSetting(ctx, key)
}

// SetSetting saves a setting to SQLite
func (s *StorageMigrationService) SetSetting(ctx context.Context, key, value string) error {
	return s.storageManager.SetSetting(ctx, key, value)
}
