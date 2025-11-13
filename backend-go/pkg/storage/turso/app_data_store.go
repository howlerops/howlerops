package turso

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/pkg/storage"
)

// TursoAppDataStore handles syncing of connections, queries, and history
// NOTE: Passwords are NOT stored in Turso - only connection metadata
type TursoAppDataStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTursoAppDataStore creates a new Turso app data store
func NewTursoAppDataStore(db *sql.DB, logger *logrus.Logger) *TursoAppDataStore {
	return &TursoAppDataStore{
		db:     db,
		logger: logger,
	}
}

// =============================================================================
// CONNECTION TEMPLATES (NO PASSWORDS!)
// =============================================================================

// ConnectionTemplate represents a connection template for sync (no passwords)
type ConnectionTemplate struct {
	ID           string            `json:"id"`
	UserID       string            `json:"user_id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Host         string            `json:"host,omitempty"`
	Port         int               `json:"port,omitempty"`
	DatabaseName string            `json:"database_name,omitempty"`
	Username     string            `json:"username,omitempty"`
	SSLConfig    string            `json:"ssl_config,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	SyncVersion  int               `json:"sync_version"`
	DeletedAt    *time.Time        `json:"deleted_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// SaveConnectionTemplate saves or updates a connection template
func (s *TursoAppDataStore) SaveConnectionTemplate(ctx context.Context, conn *ConnectionTemplate) error {
	if conn.ID == "" {
		conn.ID = uuid.New().String()
	}

	now := time.Now()
	if conn.CreatedAt.IsZero() {
		conn.CreatedAt = now
	}
	conn.UpdatedAt = now
	conn.SyncVersion++

	// Marshal metadata
	var metadataJSON []byte
	var err error
	if len(conn.Metadata) > 0 {
		metadataJSON, err = json.Marshal(conn.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Prepare deleted_at
	var deletedAt sql.NullInt64
	if conn.DeletedAt != nil {
		deletedAt.Valid = true
		deletedAt.Int64 = conn.DeletedAt.Unix()
	}

	query := `
		INSERT INTO connection_templates (
			id, user_id, name, type, host, port, database_name, username, ssl_config,
			created_at, updated_at, sync_version, deleted_at, metadata
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			host = excluded.host,
			port = excluded.port,
			database_name = excluded.database_name,
			username = excluded.username,
			ssl_config = excluded.ssl_config,
			updated_at = excluded.updated_at,
			sync_version = excluded.sync_version,
			deleted_at = excluded.deleted_at,
			metadata = excluded.metadata
	`

	_, err = s.db.ExecContext(ctx, query,
		conn.ID,
		conn.UserID,
		conn.Name,
		conn.Type,
		conn.Host,
		conn.Port,
		conn.DatabaseName,
		conn.Username,
		conn.SSLConfig,
		conn.CreatedAt.Unix(),
		conn.UpdatedAt.Unix(),
		conn.SyncVersion,
		deletedAt,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to save connection template: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id": conn.ID,
		"user_id":       conn.UserID,
		"sync_version":  conn.SyncVersion,
	}).Debug("Connection template saved")

	return nil
}

// GetConnectionTemplates retrieves connection templates for a user
func (s *TursoAppDataStore) GetConnectionTemplates(ctx context.Context, userID string, includeDeleted bool) ([]*ConnectionTemplate, error) {
	query := `
		SELECT id, user_id, name, type, host, port, database_name, username, ssl_config,
		       created_at, updated_at, sync_version, deleted_at, metadata
		FROM connection_templates
		WHERE user_id = ?
	`

	if !includeDeleted {
		query += " AND deleted_at IS NULL"
	}

	query += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query connection templates: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	var templates []*ConnectionTemplate

	for rows.Next() {
		var conn ConnectionTemplate
		var metadataJSON sql.NullString
		var deletedAt sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(
			&conn.ID,
			&conn.UserID,
			&conn.Name,
			&conn.Type,
			&conn.Host,
			&conn.Port,
			&conn.DatabaseName,
			&conn.Username,
			&conn.SSLConfig,
			&createdAt,
			&updatedAt,
			&conn.SyncVersion,
			&deletedAt,
			&metadataJSON,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan connection template row")
			continue
		}

		// Convert timestamps
		conn.CreatedAt = time.Unix(createdAt, 0)
		conn.UpdatedAt = time.Unix(updatedAt, 0)
		if deletedAt.Valid {
			t := time.Unix(deletedAt.Int64, 0)
			conn.DeletedAt = &t
		}

		// Unmarshal metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &conn.Metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal connection metadata")
				conn.Metadata = make(map[string]string)
			}
		} else {
			conn.Metadata = make(map[string]string)
		}

		templates = append(templates, &conn)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating connection template rows: %w", err)
	}

	return templates, nil
}

// GetConnectionTemplate retrieves a single connection template
func (s *TursoAppDataStore) GetConnectionTemplate(ctx context.Context, id string) (*ConnectionTemplate, error) {
	query := `
		SELECT id, user_id, name, type, host, port, database_name, username, ssl_config,
		       created_at, updated_at, sync_version, deleted_at, metadata
		FROM connection_templates
		WHERE id = ?
	`

	var conn ConnectionTemplate
	var metadataJSON sql.NullString
	var deletedAt sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&conn.ID,
		&conn.UserID,
		&conn.Name,
		&conn.Type,
		&conn.Host,
		&conn.Port,
		&conn.DatabaseName,
		&conn.Username,
		&conn.SSLConfig,
		&createdAt,
		&updatedAt,
		&conn.SyncVersion,
		&deletedAt,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("connection template not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query connection template: %w", err)
	}

	// Convert timestamps
	conn.CreatedAt = time.Unix(createdAt, 0)
	conn.UpdatedAt = time.Unix(updatedAt, 0)
	if deletedAt.Valid {
		t := time.Unix(deletedAt.Int64, 0)
		conn.DeletedAt = &t
	}

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &conn.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal connection metadata")
			conn.Metadata = make(map[string]string)
		}
	} else {
		conn.Metadata = make(map[string]string)
	}

	return &conn, nil
}

// DeleteConnectionTemplate soft deletes a connection template
func (s *TursoAppDataStore) DeleteConnectionTemplate(ctx context.Context, id string) error {
	now := time.Now()
	query := `UPDATE connection_templates SET deleted_at = ?, updated_at = ? WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, now.Unix(), now.Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete connection template: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("connection template not found")
	}

	return nil
}

// =============================================================================
// SAVED QUERIES
// =============================================================================

// SavedQuerySync represents a saved query for sync
type SavedQuerySync struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Title        string     `json:"title"`
	Query        string     `json:"query"`
	Description  string     `json:"description,omitempty"`
	ConnectionID string     `json:"connection_id,omitempty"`
	Folder       string     `json:"folder,omitempty"`
	Tags         []string   `json:"tags,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	SyncVersion  int        `json:"sync_version"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

// SaveQuerySync saves or updates a saved query
func (s *TursoAppDataStore) SaveQuerySync(ctx context.Context, query *SavedQuerySync) error {
	if query.ID == "" {
		query.ID = uuid.New().String()
	}

	now := time.Now()
	if query.CreatedAt.IsZero() {
		query.CreatedAt = now
	}
	query.UpdatedAt = now
	query.SyncVersion++

	// Marshal tags
	var tagsJSON []byte
	var err error
	if len(query.Tags) > 0 {
		tagsJSON, err = json.Marshal(query.Tags)
		if err != nil {
			return fmt.Errorf("failed to marshal tags: %w", err)
		}
	}

	// Prepare deleted_at
	var deletedAt sql.NullInt64
	if query.DeletedAt != nil {
		deletedAt.Valid = true
		deletedAt.Int64 = query.DeletedAt.Unix()
	}

	queryStr := `
		INSERT INTO saved_queries_sync (
			id, user_id, title, query, description, connection_id, folder, tags,
			created_at, updated_at, sync_version, deleted_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			title = excluded.title,
			query = excluded.query,
			description = excluded.description,
			connection_id = excluded.connection_id,
			folder = excluded.folder,
			tags = excluded.tags,
			updated_at = excluded.updated_at,
			sync_version = excluded.sync_version,
			deleted_at = excluded.deleted_at
	`

	_, err = s.db.ExecContext(ctx, queryStr,
		query.ID,
		query.UserID,
		query.Title,
		query.Query,
		query.Description,
		query.ConnectionID,
		query.Folder,
		string(tagsJSON),
		query.CreatedAt.Unix(),
		query.UpdatedAt.Unix(),
		query.SyncVersion,
		deletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save query: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"query_id":     query.ID,
		"user_id":      query.UserID,
		"sync_version": query.SyncVersion,
	}).Debug("Query saved")

	return nil
}

// GetSavedQueries retrieves saved queries for a user
func (s *TursoAppDataStore) GetSavedQueries(ctx context.Context, userID string, includeDeleted bool) ([]*SavedQuerySync, error) {
	queryStr := `
		SELECT id, user_id, title, query, description, connection_id, folder, tags,
		       created_at, updated_at, sync_version, deleted_at
		FROM saved_queries_sync
		WHERE user_id = ?
	`

	if !includeDeleted {
		queryStr += " AND deleted_at IS NULL"
	}

	queryStr += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, queryStr, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query saved queries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var queries []*SavedQuerySync

	for rows.Next() {
		var q SavedQuerySync
		var tagsJSON sql.NullString
		var deletedAt sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(
			&q.ID,
			&q.UserID,
			&q.Title,
			&q.Query,
			&q.Description,
			&q.ConnectionID,
			&q.Folder,
			&tagsJSON,
			&createdAt,
			&updatedAt,
			&q.SyncVersion,
			&deletedAt,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan saved query row")
			continue
		}

		// Convert timestamps
		q.CreatedAt = time.Unix(createdAt, 0)
		q.UpdatedAt = time.Unix(updatedAt, 0)
		if deletedAt.Valid {
			t := time.Unix(deletedAt.Int64, 0)
			q.DeletedAt = &t
		}

		// Unmarshal tags
		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &q.Tags); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal query tags")
				q.Tags = []string{}
			}
		} else {
			q.Tags = []string{}
		}

		queries = append(queries, &q)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating saved query rows: %w", err)
	}

	return queries, nil
}

// DeleteSavedQuery soft deletes a saved query
func (s *TursoAppDataStore) DeleteSavedQuery(ctx context.Context, id string) error {
	now := time.Now()
	query := `UPDATE saved_queries_sync SET deleted_at = ?, updated_at = ? WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, now.Unix(), now.Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete saved query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("saved query not found")
	}

	return nil
}

// =============================================================================
// QUERY HISTORY (SANITIZED)
// =============================================================================

// QueryHistorySync represents sanitized query history for sync
type QueryHistorySync struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	QuerySanitized string    `json:"query_sanitized"` // NO data literals!
	ConnectionID   string    `json:"connection_id,omitempty"`
	ExecutedAt     time.Time `json:"executed_at"`
	DurationMS     int64     `json:"duration_ms"`
	RowsReturned   int64     `json:"rows_returned"`
	Success        bool      `json:"success"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	SyncVersion    int       `json:"sync_version"`
}

// SaveQueryHistory saves a sanitized query history entry
func (s *TursoAppDataStore) SaveQueryHistory(ctx context.Context, history *QueryHistorySync) error {
	if history.ID == "" {
		history.ID = uuid.New().String()
	}

	if history.ExecutedAt.IsZero() {
		history.ExecutedAt = time.Now()
	}

	query := `
		INSERT INTO query_history_sync (
			id, user_id, query_sanitized, connection_id, executed_at,
			duration_ms, rows_returned, success, error_message, sync_version
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		history.ID,
		history.UserID,
		history.QuerySanitized,
		history.ConnectionID,
		history.ExecutedAt.Unix(),
		history.DurationMS,
		history.RowsReturned,
		history.Success,
		history.ErrorMessage,
		history.SyncVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to save query history: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"history_id": history.ID,
		"user_id":    history.UserID,
		"success":    history.Success,
	}).Debug("Query history saved")

	return nil
}

// GetQueryHistory retrieves query history for a user
func (s *TursoAppDataStore) GetQueryHistory(ctx context.Context, userID string, filters *storage.HistoryFilters) ([]*QueryHistorySync, error) {
	query := `
		SELECT id, user_id, query_sanitized, connection_id, executed_at,
		       duration_ms, rows_returned, success, error_message, sync_version
		FROM query_history_sync
		WHERE user_id = ?
	`
	args := []interface{}{userID}

	if filters != nil {
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
	}

	query += " ORDER BY executed_at DESC"

	if filters != nil {
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
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var history []*QueryHistorySync

	for rows.Next() {
		var h QueryHistorySync
		var executedAt int64

		err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.QuerySanitized,
			&h.ConnectionID,
			&executedAt,
			&h.DurationMS,
			&h.RowsReturned,
			&h.Success,
			&h.ErrorMessage,
			&h.SyncVersion,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan query history row")
			continue
		}

		// Convert timestamp
		h.ExecutedAt = time.Unix(executedAt, 0)

		history = append(history, &h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating query history rows: %w", err)
	}

	return history, nil
}

// =============================================================================
// SYNC METADATA
// =============================================================================

// SyncMetadata tracks last sync information
type SyncMetadata struct {
	UserID        string    `json:"user_id"`
	LastSyncAt    time.Time `json:"last_sync_at"`
	DeviceID      string    `json:"device_id,omitempty"`
	ClientVersion string    `json:"client_version,omitempty"`
}

// UpdateSyncMetadata updates the sync metadata for a user
func (s *TursoAppDataStore) UpdateSyncMetadata(ctx context.Context, metadata *SyncMetadata) error {
	query := `
		INSERT INTO sync_metadata (user_id, last_sync_at, device_id, client_version)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			last_sync_at = excluded.last_sync_at,
			device_id = excluded.device_id,
			client_version = excluded.client_version
	`

	_, err := s.db.ExecContext(ctx, query,
		metadata.UserID,
		metadata.LastSyncAt.Unix(),
		metadata.DeviceID,
		metadata.ClientVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to update sync metadata: %w", err)
	}

	return nil
}

// GetSyncMetadata retrieves sync metadata for a user
func (s *TursoAppDataStore) GetSyncMetadata(ctx context.Context, userID string) (*SyncMetadata, error) {
	query := `
		SELECT user_id, last_sync_at, device_id, client_version
		FROM sync_metadata
		WHERE user_id = ?
	`

	var metadata SyncMetadata
	var lastSyncAt int64

	err := s.db.QueryRowContext(ctx, query, userID).Scan(
		&metadata.UserID,
		&lastSyncAt,
		&metadata.DeviceID,
		&metadata.ClientVersion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sync metadata not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query sync metadata: %w", err)
	}

	metadata.LastSyncAt = time.Unix(lastSyncAt, 0)

	return &metadata, nil
}
