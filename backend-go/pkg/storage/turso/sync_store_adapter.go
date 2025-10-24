package turso

import (
	"context"
	"database/sql"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/sync"
	"github.com/sql-studio/backend-go/pkg/storage"
)

// SyncStoreAdapter adapts TursoAppDataStore to sync.Store interface
type SyncStoreAdapter struct {
	appDataStore *TursoAppDataStore
	logger       *logrus.Logger
}

// NewSyncStoreAdapter creates a new sync store adapter
func NewSyncStoreAdapter(db *sql.DB, logger *logrus.Logger) *SyncStoreAdapter {
	return &SyncStoreAdapter{
		appDataStore: NewTursoAppDataStore(db, logger),
		logger:       logger,
	}
}

// Connections

func (a *SyncStoreAdapter) GetConnection(ctx context.Context, userID, connectionID string) (*sync.ConnectionTemplate, error) {
	conn, err := a.appDataStore.GetConnectionTemplate(ctx, connectionID)
	if err != nil {
		return nil, err
	}
	return a.convertFromTursoConnection(conn), nil
}

func (a *SyncStoreAdapter) ListConnections(ctx context.Context, userID string, since time.Time) ([]sync.ConnectionTemplate, error) {
	conns, err := a.appDataStore.GetConnectionTemplates(ctx, userID, false)
	if err != nil {
		return nil, err
	}

	result := make([]sync.ConnectionTemplate, 0, len(conns))
	for _, conn := range conns {
		if conn.UpdatedAt.After(since) {
			result = append(result, *a.convertFromTursoConnection(conn))
		}
	}
	return result, nil
}

func (a *SyncStoreAdapter) SaveConnection(ctx context.Context, userID string, conn *sync.ConnectionTemplate) error {
	tursoConn := a.convertToTursoConnection(userID, conn)
	return a.appDataStore.SaveConnectionTemplate(ctx, tursoConn)
}

func (a *SyncStoreAdapter) DeleteConnection(ctx context.Context, userID, connectionID string) error {
	return a.appDataStore.DeleteConnectionTemplate(ctx, connectionID)
}

// Saved Queries

func (a *SyncStoreAdapter) GetSavedQuery(ctx context.Context, userID, queryID string) (*sync.SavedQuery, error) {
	queries, err := a.appDataStore.GetSavedQueries(ctx, userID, false)
	if err != nil {
		return nil, err
	}

	for _, q := range queries {
		if q.ID == queryID {
			return a.convertFromTursoQuery(q), nil
		}
	}

	return nil, sql.ErrNoRows
}

func (a *SyncStoreAdapter) ListSavedQueries(ctx context.Context, userID string, since time.Time) ([]sync.SavedQuery, error) {
	queries, err := a.appDataStore.GetSavedQueries(ctx, userID, false)
	if err != nil {
		return nil, err
	}

	result := make([]sync.SavedQuery, 0, len(queries))
	for _, q := range queries {
		if q.UpdatedAt.After(since) {
			result = append(result, *a.convertFromTursoQuery(q))
		}
	}
	return result, nil
}

func (a *SyncStoreAdapter) SaveQuery(ctx context.Context, userID string, query *sync.SavedQuery) error {
	tursoQuery := a.convertToTursoQuery(userID, query)
	return a.appDataStore.SaveQuerySync(ctx, tursoQuery)
}

func (a *SyncStoreAdapter) DeleteQuery(ctx context.Context, userID, queryID string) error {
	return a.appDataStore.DeleteSavedQuery(ctx, queryID)
}

// Query History

func (a *SyncStoreAdapter) ListQueryHistory(ctx context.Context, userID string, since time.Time, limit int) ([]sync.QueryHistory, error) {
	// Build filters for history query
	filters := &storage.HistoryFilters{
		StartDate: &since,
		Limit:     limit,
	}

	history, err := a.appDataStore.GetQueryHistory(ctx, userID, filters)
	if err != nil {
		return nil, err
	}

	result := make([]sync.QueryHistory, 0, len(history))
	for _, h := range history {
		result = append(result, *a.convertFromTursoHistory(h))
	}
	return result, nil
}

func (a *SyncStoreAdapter) SaveQueryHistory(ctx context.Context, userID string, history *sync.QueryHistory) error {
	tursoHistory := a.convertToTursoHistory(userID, history)
	return a.appDataStore.SaveQueryHistory(ctx, tursoHistory)
}

// Conflicts (not yet implemented in Turso storage)

func (a *SyncStoreAdapter) SaveConflict(ctx context.Context, userID string, conflict *sync.Conflict) error {
	// TODO: Implement conflict storage when needed
	a.logger.WithField("conflict_id", conflict.ID).Warn("Conflict storage not yet implemented")
	return nil
}

func (a *SyncStoreAdapter) GetConflict(ctx context.Context, userID, conflictID string) (*sync.Conflict, error) {
	// TODO: Implement conflict retrieval when needed
	return nil, sql.ErrNoRows
}

func (a *SyncStoreAdapter) ListConflicts(ctx context.Context, userID string, resolved bool) ([]sync.Conflict, error) {
	// TODO: Implement conflict listing when needed
	return []sync.Conflict{}, nil
}

func (a *SyncStoreAdapter) ResolveConflict(ctx context.Context, userID, conflictID string, resolution sync.ConflictResolutionStrategy) error {
	// TODO: Implement conflict resolution when needed
	return nil
}

// Metadata

func (a *SyncStoreAdapter) GetSyncMetadata(ctx context.Context, userID, deviceID string) (*sync.SyncMetadata, error) {
	metadata, err := a.appDataStore.GetSyncMetadata(ctx, userID)
	if err != nil {
		// Return default metadata if not found
		if err.Error() == "sync metadata not found" {
			return &sync.SyncMetadata{
				UserID:     userID,
				DeviceID:   deviceID,
				LastSyncAt: time.Time{},
			}, nil
		}
		return nil, err
	}

	return &sync.SyncMetadata{
		UserID:     metadata.UserID,
		DeviceID:   metadata.DeviceID,
		LastSyncAt: metadata.LastSyncAt,
		Version:    metadata.ClientVersion,
	}, nil
}

func (a *SyncStoreAdapter) UpdateSyncMetadata(ctx context.Context, metadata *sync.SyncMetadata) error {
	tursoMetadata := &SyncMetadata{
		UserID:        metadata.UserID,
		LastSyncAt:    metadata.LastSyncAt,
		DeviceID:      metadata.DeviceID,
		ClientVersion: metadata.Version,
	}
	return a.appDataStore.UpdateSyncMetadata(ctx, tursoMetadata)
}

// Conversion helpers

func (a *SyncStoreAdapter) convertFromTursoConnection(conn *ConnectionTemplate) *sync.ConnectionTemplate {
	return &sync.ConnectionTemplate{
		ID:          conn.ID,
		Name:        conn.Name,
		Type:        conn.Type,
		Host:        conn.Host,
		Port:        conn.Port,
		Database:    conn.DatabaseName,
		Username:    conn.Username,
		Metadata:    conn.Metadata,
		CreatedAt:   conn.CreatedAt,
		UpdatedAt:   conn.UpdatedAt,
		SyncVersion: conn.SyncVersion,
	}
}

func (a *SyncStoreAdapter) convertToTursoConnection(userID string, conn *sync.ConnectionTemplate) *ConnectionTemplate {
	return &ConnectionTemplate{
		ID:           conn.ID,
		UserID:       userID,
		Name:         conn.Name,
		Type:         conn.Type,
		Host:         conn.Host,
		Port:         conn.Port,
		DatabaseName: conn.Database,
		Username:     conn.Username,
		Metadata:     conn.Metadata,
		CreatedAt:    conn.CreatedAt,
		UpdatedAt:    conn.UpdatedAt,
		SyncVersion:  conn.SyncVersion,
	}
}

func (a *SyncStoreAdapter) convertFromTursoQuery(query *SavedQuerySync) *sync.SavedQuery {
	return &sync.SavedQuery{
		ID:           query.ID,
		Name:         query.Title,
		Description:  query.Description,
		Query:        query.Query,
		ConnectionID: query.ConnectionID,
		Tags:         query.Tags,
		CreatedAt:    query.CreatedAt,
		UpdatedAt:    query.UpdatedAt,
		SyncVersion:  query.SyncVersion,
	}
}

func (a *SyncStoreAdapter) convertToTursoQuery(userID string, query *sync.SavedQuery) *SavedQuerySync {
	return &SavedQuerySync{
		ID:           query.ID,
		UserID:       userID,
		Title:        query.Name,
		Description:  query.Description,
		Query:        query.Query,
		ConnectionID: query.ConnectionID,
		Tags:         query.Tags,
		CreatedAt:    query.CreatedAt,
		UpdatedAt:    query.UpdatedAt,
		SyncVersion:  query.SyncVersion,
	}
}

func (a *SyncStoreAdapter) convertFromTursoHistory(history *QueryHistorySync) *sync.QueryHistory {
	return &sync.QueryHistory{
		ID:           history.ID,
		Query:        history.QuerySanitized,
		ConnectionID: history.ConnectionID,
		ExecutedAt:   history.ExecutedAt,
		Duration:     history.DurationMS,
		RowsAffected: history.RowsReturned,
		Success:      history.Success,
		Error:        history.ErrorMessage,
		SyncVersion:  history.SyncVersion,
	}
}

func (a *SyncStoreAdapter) convertToTursoHistory(userID string, history *sync.QueryHistory) *QueryHistorySync {
	return &QueryHistorySync{
		ID:             history.ID,
		UserID:         userID,
		QuerySanitized: history.Query,
		ConnectionID:   history.ConnectionID,
		ExecutedAt:     history.ExecutedAt,
		DurationMS:     history.Duration,
		RowsReturned:   history.RowsAffected,
		Success:        history.Success,
		ErrorMessage:   history.Error,
		SyncVersion:    history.SyncVersion,
	}
}
