package sync

import (
	"context"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockStore implements Store interface for testing
type MockStore struct {
	connections  map[string]*ConnectionTemplate // connID -> connection
	queries      map[string]*SavedQuery          // queryID -> query
	queryHistory map[string]*QueryHistory        // historyID -> history
	conflicts    map[string]*Conflict            // conflictID -> conflict
	syncMetadata map[string]*SyncMetadata        // userID-deviceID -> metadata
	syncLogs     []SyncLog
}

func NewMockStore() *MockStore {
	return &MockStore{
		connections:  make(map[string]*ConnectionTemplate),
		queries:      make(map[string]*SavedQuery),
		queryHistory: make(map[string]*QueryHistory),
		conflicts:    make(map[string]*Conflict),
		syncMetadata: make(map[string]*SyncMetadata),
		syncLogs:     []SyncLog{},
	}
}

func (m *MockStore) GetConnection(ctx context.Context, userID, connectionID string) (*ConnectionTemplate, error) {
	conn, exists := m.connections[connectionID]
	if !exists {
		return nil, assert.AnError
	}
	return conn, nil
}

func (m *MockStore) ListConnections(ctx context.Context, userID string, since time.Time) ([]ConnectionTemplate, error) {
	var result []ConnectionTemplate
	for _, conn := range m.connections {
		if conn.UserID == userID && conn.UpdatedAt.After(since) {
			result = append(result, *conn)
		}
	}
	return result, nil
}

func (m *MockStore) ListAccessibleConnections(ctx context.Context, userID string, orgIDs []string, since time.Time) ([]ConnectionTemplate, error) {
	var result []ConnectionTemplate

	orgMap := make(map[string]bool)
	for _, orgID := range orgIDs {
		orgMap[orgID] = true
	}

	for _, conn := range m.connections {
		// Include personal connections
		if conn.UserID == userID && (conn.Visibility == "personal" || conn.OrganizationID == nil) {
			if conn.UpdatedAt.After(since) {
				result = append(result, *conn)
			}
			continue
		}

		// Include shared connections in user's orgs
		if conn.Visibility == "shared" && conn.OrganizationID != nil {
			if orgMap[*conn.OrganizationID] && conn.UpdatedAt.After(since) {
				result = append(result, *conn)
			}
		}
	}

	return result, nil
}

func (m *MockStore) SaveConnection(ctx context.Context, userID string, conn *ConnectionTemplate) error {
	m.connections[conn.ID] = conn
	return nil
}

func (m *MockStore) DeleteConnection(ctx context.Context, userID, connectionID string) error {
	delete(m.connections, connectionID)
	return nil
}

func (m *MockStore) GetSavedQuery(ctx context.Context, userID, queryID string) (*SavedQuery, error) {
	query, exists := m.queries[queryID]
	if !exists {
		return nil, assert.AnError
	}
	return query, nil
}

func (m *MockStore) ListSavedQueries(ctx context.Context, userID string, since time.Time) ([]SavedQuery, error) {
	var result []SavedQuery
	for _, query := range m.queries {
		if query.UserID == userID && query.UpdatedAt.After(since) {
			result = append(result, *query)
		}
	}
	return result, nil
}

func (m *MockStore) ListAccessibleQueries(ctx context.Context, userID string, orgIDs []string, since time.Time) ([]SavedQuery, error) {
	var result []SavedQuery

	orgMap := make(map[string]bool)
	for _, orgID := range orgIDs {
		orgMap[orgID] = true
	}

	for _, query := range m.queries {
		// Include personal queries
		if query.UserID == userID && (query.Visibility == "personal" || query.OrganizationID == nil) {
			if query.UpdatedAt.After(since) {
				result = append(result, *query)
			}
			continue
		}

		// Include shared queries in user's orgs
		if query.Visibility == "shared" && query.OrganizationID != nil {
			if orgMap[*query.OrganizationID] && query.UpdatedAt.After(since) {
				result = append(result, *query)
			}
		}
	}

	return result, nil
}

func (m *MockStore) SaveQuery(ctx context.Context, userID string, query *SavedQuery) error {
	m.queries[query.ID] = query
	return nil
}

func (m *MockStore) DeleteQuery(ctx context.Context, userID, queryID string) error {
	delete(m.queries, queryID)
	return nil
}

func (m *MockStore) ListQueryHistory(ctx context.Context, userID string, since time.Time, limit int) ([]QueryHistory, error) {
	var result []QueryHistory
	count := 0
	for _, history := range m.queryHistory {
		if count >= limit {
			break
		}
		if history.ExecutedAt.After(since) {
			result = append(result, *history)
			count++
		}
	}
	return result, nil
}

func (m *MockStore) SaveQueryHistory(ctx context.Context, userID string, history *QueryHistory) error {
	m.queryHistory[history.ID] = history
	return nil
}

func (m *MockStore) SaveConflict(ctx context.Context, userID string, conflict *Conflict) error {
	m.conflicts[conflict.ID] = conflict
	return nil
}

func (m *MockStore) GetConflict(ctx context.Context, userID, conflictID string) (*Conflict, error) {
	conflict, exists := m.conflicts[conflictID]
	if !exists {
		return nil, assert.AnError
	}
	return conflict, nil
}

func (m *MockStore) ListConflicts(ctx context.Context, userID string, resolved bool) ([]Conflict, error) {
	var result []Conflict
	for _, conflict := range m.conflicts {
		if resolved && conflict.ResolvedAt != nil {
			result = append(result, *conflict)
		} else if !resolved && conflict.ResolvedAt == nil {
			result = append(result, *conflict)
		}
	}
	return result, nil
}

func (m *MockStore) ResolveConflict(ctx context.Context, userID, conflictID string, resolution ConflictResolutionStrategy) error {
	conflict, exists := m.conflicts[conflictID]
	if !exists {
		return assert.AnError
	}
	now := time.Now()
	conflict.ResolvedAt = &now
	conflict.Resolution = resolution
	return nil
}

func (m *MockStore) GetSyncMetadata(ctx context.Context, userID, deviceID string) (*SyncMetadata, error) {
	key := userID + "-" + deviceID
	metadata, exists := m.syncMetadata[key]
	if !exists {
		return nil, assert.AnError
	}
	return metadata, nil
}

func (m *MockStore) UpdateSyncMetadata(ctx context.Context, metadata *SyncMetadata) error {
	key := metadata.UserID + "-" + metadata.DeviceID
	m.syncMetadata[key] = metadata
	return nil
}

func (m *MockStore) SaveSyncLog(ctx context.Context, log *SyncLog) error {
	m.syncLogs = append(m.syncLogs, *log)
	return nil
}

func (m *MockStore) ListSyncLogs(ctx context.Context, userID string, limit int) ([]SyncLog, error) {
	var result []SyncLog
	count := 0
	for i := len(m.syncLogs) - 1; i >= 0 && count < limit; i-- {
		if m.syncLogs[i].UserID == userID {
			result = append(result, m.syncLogs[i])
			count++
		}
	}
	return result, nil
}
