package sync

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// TursoStore implements Store interface using Turso/SQLite
type TursoStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTursoStore creates a new Turso store
func NewTursoStore(dbURL, authToken string, logger *logrus.Logger) (*TursoStore, error) {
	// For Turso, use libsql driver with auth token in connection string
	// Format: libsql://[hostname]?authToken=[token]
	connStr := fmt.Sprintf("%s?authToken=%s", dbURL, authToken)

	// For local development, you can use sqlite3 driver
	// db, err := sql.Open("sqlite3", "./data/sync.db")

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &TursoStore{
		db:     db,
		logger: logger,
	}

	// Initialize schema
	if err := store.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return store, nil
}

// initSchema creates the necessary tables
func (s *TursoStore) initSchema() error {
	schema := `
	-- Connections table
	CREATE TABLE IF NOT EXISTS connections (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		host TEXT,
		port INTEGER,
		database_name TEXT NOT NULL,
		username TEXT,
		use_ssh BOOLEAN DEFAULT 0,
		ssh_host TEXT,
		ssh_port INTEGER,
		ssh_user TEXT,
		color TEXT,
		icon TEXT,
		metadata TEXT,
		organization_id TEXT,
		visibility TEXT DEFAULT 'personal',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		sync_version INTEGER DEFAULT 1,
		deleted_at TIMESTAMP,
		UNIQUE(user_id, id)
	);

	CREATE INDEX IF NOT EXISTS idx_connections_user_id ON connections(user_id);
	CREATE INDEX IF NOT EXISTS idx_connections_updated_at ON connections(updated_at);
	CREATE INDEX IF NOT EXISTS idx_connections_org_id ON connections(organization_id);

	-- Saved queries table
	CREATE TABLE IF NOT EXISTS saved_queries (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		name TEXT NOT NULL,
		description TEXT,
		query TEXT NOT NULL,
		connection_id TEXT,
		tags TEXT,
		favorite BOOLEAN DEFAULT 0,
		metadata TEXT,
		organization_id TEXT,
		visibility TEXT DEFAULT 'personal',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		sync_version INTEGER DEFAULT 1,
		deleted_at TIMESTAMP,
		UNIQUE(user_id, id)
	);

	CREATE INDEX IF NOT EXISTS idx_saved_queries_user_id ON saved_queries(user_id);
	CREATE INDEX IF NOT EXISTS idx_saved_queries_updated_at ON saved_queries(updated_at);
	CREATE INDEX IF NOT EXISTS idx_saved_queries_org_id ON saved_queries(organization_id);

	-- Query history table
	CREATE TABLE IF NOT EXISTS query_history (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		query TEXT NOT NULL,
		connection_id TEXT,
		executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		duration_ms INTEGER,
		rows_affected INTEGER,
		success BOOLEAN DEFAULT 1,
		error TEXT,
		metadata TEXT,
		sync_version INTEGER DEFAULT 1,
		UNIQUE(user_id, id)
	);

	CREATE INDEX IF NOT EXISTS idx_query_history_user_id ON query_history(user_id);
	CREATE INDEX IF NOT EXISTS idx_query_history_executed_at ON query_history(executed_at);

	-- Conflicts table
	CREATE TABLE IF NOT EXISTS conflicts (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		item_type TEXT NOT NULL,
		item_id TEXT NOT NULL,
		local_version TEXT,
		remote_version TEXT,
		detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP,
		resolution TEXT,
		UNIQUE(user_id, id)
	);

	CREATE INDEX IF NOT EXISTS idx_conflicts_user_id ON conflicts(user_id);
	CREATE INDEX IF NOT EXISTS idx_conflicts_resolved_at ON conflicts(resolved_at);

	-- Sync metadata table
	CREATE TABLE IF NOT EXISTS sync_metadata (
		user_id TEXT NOT NULL,
		device_id TEXT NOT NULL,
		last_sync_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		total_synced INTEGER DEFAULT 0,
		conflicts_count INTEGER DEFAULT 0,
		version TEXT,
		PRIMARY KEY(user_id, device_id)
	);

	-- Sync logs table
	CREATE TABLE IF NOT EXISTS sync_logs (
		id TEXT PRIMARY KEY,
		user_id TEXT NOT NULL,
		organization_id TEXT,
		action TEXT NOT NULL,
		resource_count INTEGER DEFAULT 0,
		conflict_count INTEGER DEFAULT 0,
		device_id TEXT NOT NULL,
		client_version TEXT,
		synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_sync_logs_user_id ON sync_logs(user_id);
	CREATE INDEX IF NOT EXISTS idx_sync_logs_synced_at ON sync_logs(synced_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// GetConnection retrieves a connection by ID
func (s *TursoStore) GetConnection(ctx context.Context, userID, connectionID string) (*ConnectionTemplate, error) {
	query := `
		SELECT id, name, type, host, port, database_name, username, use_ssh, ssh_host,
		       ssh_port, ssh_user, color, icon, metadata, created_at, updated_at, sync_version
		FROM connections
		WHERE user_id = ? AND id = ? AND deleted_at IS NULL
	`

	var conn ConnectionTemplate
	var metadataJSON sql.NullString
	var port, sshPort sql.NullInt64
	var host, username, sshHost, sshUser, color, icon sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, connectionID).Scan(
		&conn.ID, &conn.Name, &conn.Type, &host, &port, &conn.Database,
		&username, &conn.UseSSH, &sshHost, &sshPort, &sshUser,
		&color, &icon, &metadataJSON, &conn.CreatedAt, &conn.UpdatedAt, &conn.SyncVersion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("connection not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// Handle nullable fields
	if host.Valid {
		conn.Host = host.String
	}
	if port.Valid {
		conn.Port = int(port.Int64)
	}
	if username.Valid {
		conn.Username = username.String
	}
	if sshHost.Valid {
		conn.SSHHost = sshHost.String
	}
	if sshPort.Valid {
		conn.SSHPort = int(sshPort.Int64)
	}
	if sshUser.Valid {
		conn.SSHUser = sshUser.String
	}
	if color.Valid {
		conn.Color = color.String
	}
	if icon.Valid {
		conn.Icon = icon.String
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &conn.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal metadata")
		}
	}

	return &conn, nil
}

// ListConnections retrieves all connections for a user updated since a given time
func (s *TursoStore) ListConnections(ctx context.Context, userID string, since time.Time) ([]ConnectionTemplate, error) {
	query := `
		SELECT id, name, type, host, port, database_name, username, use_ssh, ssh_host,
		       ssh_port, ssh_user, color, icon, metadata, created_at, updated_at, sync_version
		FROM connections
		WHERE user_id = ? AND updated_at > ? AND deleted_at IS NULL
		ORDER BY updated_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list connections: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var connections []ConnectionTemplate
	for rows.Next() {
		var conn ConnectionTemplate
		var metadataJSON sql.NullString
		var port, sshPort sql.NullInt64
		var host, username, sshHost, sshUser, color, icon sql.NullString

		if err := rows.Scan(
			&conn.ID, &conn.Name, &conn.Type, &host, &port, &conn.Database,
			&username, &conn.UseSSH, &sshHost, &sshPort, &sshUser,
			&color, &icon, &metadataJSON, &conn.CreatedAt, &conn.UpdatedAt, &conn.SyncVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}

		// Handle nullable fields
		if host.Valid {
			conn.Host = host.String
		}
		if port.Valid {
			conn.Port = int(port.Int64)
		}
		if username.Valid {
			conn.Username = username.String
		}
		if sshHost.Valid {
			conn.SSHHost = sshHost.String
		}
		if sshPort.Valid {
			conn.SSHPort = int(sshPort.Int64)
		}
		if sshUser.Valid {
			conn.SSHUser = sshUser.String
		}
		if color.Valid {
			conn.Color = color.String
		}
		if icon.Valid {
			conn.Icon = icon.String
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &conn.Metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal metadata")
			}
		}

		connections = append(connections, conn)
	}

	return connections, nil
}

// SaveConnection saves or updates a connection
func (s *TursoStore) SaveConnection(ctx context.Context, userID string, conn *ConnectionTemplate) error {
	metadataJSON, _ := json.Marshal(conn.Metadata)

	query := `
		INSERT INTO connections (
			id, user_id, name, type, host, port, database_name, username,
			use_ssh, ssh_host, ssh_port, ssh_user, color, icon, metadata,
			created_at, updated_at, sync_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, id) DO UPDATE SET
			name = excluded.name,
			type = excluded.type,
			host = excluded.host,
			port = excluded.port,
			database_name = excluded.database_name,
			username = excluded.username,
			use_ssh = excluded.use_ssh,
			ssh_host = excluded.ssh_host,
			ssh_port = excluded.ssh_port,
			ssh_user = excluded.ssh_user,
			color = excluded.color,
			icon = excluded.icon,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at,
			sync_version = excluded.sync_version,
			deleted_at = NULL
	`

	_, err := s.db.ExecContext(ctx, query,
		conn.ID, userID, conn.Name, conn.Type, conn.Host, conn.Port, conn.Database,
		conn.Username, conn.UseSSH, conn.SSHHost, conn.SSHPort, conn.SSHUser,
		conn.Color, conn.Icon, string(metadataJSON),
		conn.CreatedAt, conn.UpdatedAt, conn.SyncVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to save connection: %w", err)
	}

	return nil
}

// DeleteConnection soft deletes a connection
func (s *TursoStore) DeleteConnection(ctx context.Context, userID, connectionID string) error {
	query := `UPDATE connections SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ? AND id = ?`
	_, err := s.db.ExecContext(ctx, query, userID, connectionID)
	return err
}

// GetSavedQuery retrieves a saved query by ID
func (s *TursoStore) GetSavedQuery(ctx context.Context, userID, queryID string) (*SavedQuery, error) {
	query := `
		SELECT id, name, description, query, connection_id, tags, favorite, metadata,
		       created_at, updated_at, sync_version
		FROM saved_queries
		WHERE user_id = ? AND id = ? AND deleted_at IS NULL
	`

	var sq SavedQuery
	var description, connectionID, tagsJSON, metadataJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, queryID).Scan(
		&sq.ID, &sq.Name, &description, &sq.Query, &connectionID,
		&tagsJSON, &sq.Favorite, &metadataJSON,
		&sq.CreatedAt, &sq.UpdatedAt, &sq.SyncVersion,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("saved query not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get saved query: %w", err)
	}

	if description.Valid {
		sq.Description = description.String
	}
	if connectionID.Valid {
		sq.ConnectionID = connectionID.String
	}
	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &sq.Tags); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal tags JSON")
		}
	}
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &sq.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal metadata JSON")
		}
	}

	return &sq, nil
}

// ListSavedQueries retrieves all saved queries for a user updated since a given time
func (s *TursoStore) ListSavedQueries(ctx context.Context, userID string, since time.Time) ([]SavedQuery, error) {
	query := `
		SELECT id, name, description, query, connection_id, tags, favorite, metadata,
		       created_at, updated_at, sync_version
		FROM saved_queries
		WHERE user_id = ? AND updated_at > ? AND deleted_at IS NULL
		ORDER BY updated_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to list saved queries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var queries []SavedQuery
	for rows.Next() {
		var sq SavedQuery
		var description, connectionID, tagsJSON, metadataJSON sql.NullString

		if err := rows.Scan(
			&sq.ID, &sq.Name, &description, &sq.Query, &connectionID,
			&tagsJSON, &sq.Favorite, &metadataJSON,
			&sq.CreatedAt, &sq.UpdatedAt, &sq.SyncVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan saved query: %w", err)
		}

		if description.Valid {
			sq.Description = description.String
		}
		if connectionID.Valid {
			sq.ConnectionID = connectionID.String
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &sq.Tags); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal tags JSON")
			}
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &sq.Metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal metadata JSON")
			}
		}

		queries = append(queries, sq)
	}

	return queries, nil
}

// SaveQuery saves or updates a saved query
func (s *TursoStore) SaveQuery(ctx context.Context, userID string, query *SavedQuery) error {
	tagsJSON, _ := json.Marshal(query.Tags)
	metadataJSON, _ := json.Marshal(query.Metadata)

	querySQL := `
		INSERT INTO saved_queries (
			id, user_id, name, description, query, connection_id, tags, favorite,
			metadata, created_at, updated_at, sync_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, id) DO UPDATE SET
			name = excluded.name,
			description = excluded.description,
			query = excluded.query,
			connection_id = excluded.connection_id,
			tags = excluded.tags,
			favorite = excluded.favorite,
			metadata = excluded.metadata,
			updated_at = excluded.updated_at,
			sync_version = excluded.sync_version,
			deleted_at = NULL
	`

	_, err := s.db.ExecContext(ctx, querySQL,
		query.ID, userID, query.Name, query.Description, query.Query,
		query.ConnectionID, string(tagsJSON), query.Favorite, string(metadataJSON),
		query.CreatedAt, query.UpdatedAt, query.SyncVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to save query: %w", err)
	}

	return nil
}

// DeleteQuery soft deletes a saved query
func (s *TursoStore) DeleteQuery(ctx context.Context, userID, queryID string) error {
	query := `UPDATE saved_queries SET deleted_at = CURRENT_TIMESTAMP WHERE user_id = ? AND id = ?`
	_, err := s.db.ExecContext(ctx, query, userID, queryID)
	return err
}

// ListQueryHistory retrieves query history for a user
func (s *TursoStore) ListQueryHistory(ctx context.Context, userID string, since time.Time, limit int) ([]QueryHistory, error) {
	query := `
		SELECT id, query, connection_id, executed_at, duration_ms, rows_affected,
		       success, error, metadata, sync_version
		FROM query_history
		WHERE user_id = ? AND executed_at > ?
		ORDER BY executed_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list query history: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var history []QueryHistory
	for rows.Next() {
		var qh QueryHistory
		var connectionID, errorMsg, metadataJSON sql.NullString

		if err := rows.Scan(
			&qh.ID, &qh.Query, &connectionID, &qh.ExecutedAt, &qh.Duration,
			&qh.RowsAffected, &qh.Success, &errorMsg, &metadataJSON, &qh.SyncVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan query history: %w", err)
		}

		if connectionID.Valid {
			qh.ConnectionID = connectionID.String
		}
		if errorMsg.Valid {
			qh.Error = errorMsg.String
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			_ = json.Unmarshal([]byte(metadataJSON.String), &qh.Metadata) // Best-effort unmarshal
		}

		history = append(history, qh)
	}

	return history, nil
}

// SaveQueryHistory saves query history
func (s *TursoStore) SaveQueryHistory(ctx context.Context, userID string, history *QueryHistory) error {
	metadataJSON, _ := json.Marshal(history.Metadata)

	query := `
		INSERT INTO query_history (
			id, user_id, query, connection_id, executed_at, duration_ms,
			rows_affected, success, error, metadata, sync_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, id) DO NOTHING
	`

	_, err := s.db.ExecContext(ctx, query,
		history.ID, userID, history.Query, history.ConnectionID, history.ExecutedAt,
		history.Duration, history.RowsAffected, history.Success, history.Error,
		string(metadataJSON), history.SyncVersion,
	)

	if err != nil {
		return fmt.Errorf("failed to save query history: %w", err)
	}

	return nil
}

// SaveConflict saves a conflict
func (s *TursoStore) SaveConflict(ctx context.Context, userID string, conflict *Conflict) error {
	localJSON, _ := json.Marshal(conflict.LocalVersion)
	remoteJSON, _ := json.Marshal(conflict.RemoteVersion)

	query := `
		INSERT INTO conflicts (
			id, user_id, item_type, item_id, local_version, remote_version, detected_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		conflict.ID, userID, conflict.ItemType, conflict.ItemID,
		string(localJSON), string(remoteJSON), conflict.DetectedAt,
	)

	return err
}

// GetConflict retrieves a conflict by ID
func (s *TursoStore) GetConflict(ctx context.Context, userID, conflictID string) (*Conflict, error) {
	query := `
		SELECT id, item_type, item_id, local_version, remote_version, detected_at,
		       resolved_at, resolution
		FROM conflicts
		WHERE user_id = ? AND id = ?
	`

	var conflict Conflict
	var localJSON, remoteJSON string
	var resolvedAt sql.NullTime
	var resolution sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, conflictID).Scan(
		&conflict.ID, &conflict.ItemType, &conflict.ItemID,
		&localJSON, &remoteJSON, &conflict.DetectedAt,
		&resolvedAt, &resolution,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conflict not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get conflict: %w", err)
	}

	_ = json.Unmarshal([]byte(localJSON), &conflict.LocalVersion)   // Best-effort unmarshal
	_ = json.Unmarshal([]byte(remoteJSON), &conflict.RemoteVersion) // Best-effort unmarshal

	if resolvedAt.Valid {
		conflict.ResolvedAt = &resolvedAt.Time
	}
	if resolution.Valid {
		conflict.Resolution = ConflictResolutionStrategy(resolution.String)
	}

	return &conflict, nil
}

// ListConflicts retrieves all conflicts for a user
func (s *TursoStore) ListConflicts(ctx context.Context, userID string, resolved bool) ([]Conflict, error) {
	var query string
	if resolved {
		query = `SELECT id, item_type, item_id, local_version, remote_version, detected_at, resolved_at, resolution
		         FROM conflicts WHERE user_id = ? AND resolved_at IS NOT NULL`
	} else {
		query = `SELECT id, item_type, item_id, local_version, remote_version, detected_at, resolved_at, resolution
		         FROM conflicts WHERE user_id = ? AND resolved_at IS NULL`
	}

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list conflicts: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	var conflicts []Conflict
	for rows.Next() {
		var conflict Conflict
		var localJSON, remoteJSON string
		var resolvedAt sql.NullTime
		var resolution sql.NullString

		if err := rows.Scan(
			&conflict.ID, &conflict.ItemType, &conflict.ItemID,
			&localJSON, &remoteJSON, &conflict.DetectedAt,
			&resolvedAt, &resolution,
		); err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		_ = json.Unmarshal([]byte(localJSON), &conflict.LocalVersion)   // Best-effort unmarshal
		_ = json.Unmarshal([]byte(remoteJSON), &conflict.RemoteVersion) // Best-effort unmarshal

		if resolvedAt.Valid {
			conflict.ResolvedAt = &resolvedAt.Time
		}
		if resolution.Valid {
			conflict.Resolution = ConflictResolutionStrategy(resolution.String)
		}

		conflicts = append(conflicts, conflict)
	}

	return conflicts, nil
}

// ResolveConflict marks a conflict as resolved
func (s *TursoStore) ResolveConflict(ctx context.Context, userID, conflictID string, resolution ConflictResolutionStrategy) error {
	query := `
		UPDATE conflicts
		SET resolved_at = CURRENT_TIMESTAMP, resolution = ?
		WHERE user_id = ? AND id = ?
	`

	_, err := s.db.ExecContext(ctx, query, string(resolution), userID, conflictID)
	return err
}

// GetSyncMetadata retrieves sync metadata
func (s *TursoStore) GetSyncMetadata(ctx context.Context, userID, deviceID string) (*SyncMetadata, error) {
	query := `
		SELECT user_id, device_id, last_sync_at, total_synced, conflicts_count, version
		FROM sync_metadata
		WHERE user_id = ? AND device_id = ?
	`

	var metadata SyncMetadata
	var version sql.NullString

	err := s.db.QueryRowContext(ctx, query, userID, deviceID).Scan(
		&metadata.UserID, &metadata.DeviceID, &metadata.LastSyncAt,
		&metadata.TotalSynced, &metadata.ConflictsCount, &version,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("sync metadata not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync metadata: %w", err)
	}

	if version.Valid {
		metadata.Version = version.String
	}

	return &metadata, nil
}

// UpdateSyncMetadata updates sync metadata
func (s *TursoStore) UpdateSyncMetadata(ctx context.Context, metadata *SyncMetadata) error {
	query := `
		INSERT INTO sync_metadata (user_id, device_id, last_sync_at, total_synced, conflicts_count, version)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id, device_id) DO UPDATE SET
			last_sync_at = excluded.last_sync_at,
			total_synced = total_synced + excluded.total_synced,
			conflicts_count = excluded.conflicts_count,
			version = excluded.version
	`

	_, err := s.db.ExecContext(ctx, query,
		metadata.UserID, metadata.DeviceID, metadata.LastSyncAt,
		metadata.TotalSynced, metadata.ConflictsCount, metadata.Version,
	)

	return err
}

// ListAccessibleConnections retrieves connections user has access to (personal + shared in orgs)
func (s *TursoStore) ListAccessibleConnections(ctx context.Context, userID string, orgIDs []string, since time.Time) ([]ConnectionTemplate, error) {
	// Build query with dynamic org filter
	query := `
		SELECT id, name, type, host, port, database_name, username, use_ssh, ssh_host,
		       ssh_port, ssh_user, color, icon, metadata, user_id, organization_id,
		       visibility, created_at, updated_at, sync_version
		FROM connections
		WHERE deleted_at IS NULL AND updated_at > ? AND (
			(user_id = ? AND (visibility = 'personal' OR organization_id IS NULL))
	`

	args := []interface{}{since, userID}

	// Add OR clause for each organization
	if len(orgIDs) > 0 {
		query += " OR (organization_id IN ("
		for i := range orgIDs {
			if i > 0 {
				query += ","
			}
			query += "?"
			args = append(args, orgIDs[i])
		}
		query += ") AND visibility = 'shared')"
	}

	query += ") ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accessible connections: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	var connections []ConnectionTemplate
	for rows.Next() {
		var conn ConnectionTemplate
		var metadataJSON sql.NullString
		var port, sshPort sql.NullInt64
		var host, username, sshHost, sshUser, color, icon, organizationID sql.NullString

		if err := rows.Scan(
			&conn.ID, &conn.Name, &conn.Type, &host, &port, &conn.Database,
			&username, &conn.UseSSH, &sshHost, &sshPort, &sshUser,
			&color, &icon, &metadataJSON, &conn.UserID, &organizationID,
			&conn.Visibility, &conn.CreatedAt, &conn.UpdatedAt, &conn.SyncVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan connection: %w", err)
		}

		// Handle nullable fields
		if host.Valid {
			conn.Host = host.String
		}
		if port.Valid {
			conn.Port = int(port.Int64)
		}
		if username.Valid {
			conn.Username = username.String
		}
		if sshHost.Valid {
			conn.SSHHost = sshHost.String
		}
		if sshPort.Valid {
			conn.SSHPort = int(sshPort.Int64)
		}
		if sshUser.Valid {
			conn.SSHUser = sshUser.String
		}
		if color.Valid {
			conn.Color = color.String
		}
		if icon.Valid {
			conn.Icon = icon.String
		}
		if organizationID.Valid {
			orgID := organizationID.String
			conn.OrganizationID = &orgID
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &conn.Metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal metadata")
			}
		}

		connections = append(connections, conn)
	}

	return connections, nil
}

// ListAccessibleQueries retrieves queries user has access to (personal + shared in orgs)
func (s *TursoStore) ListAccessibleQueries(ctx context.Context, userID string, orgIDs []string, since time.Time) ([]SavedQuery, error) {
	// Build query with dynamic org filter
	query := `
		SELECT id, name, description, query, connection_id, tags, favorite, metadata,
		       user_id, organization_id, visibility, created_at, updated_at, sync_version
		FROM saved_queries
		WHERE deleted_at IS NULL AND updated_at > ? AND (
			(user_id = ? AND (visibility = 'personal' OR organization_id IS NULL))
	`

	args := []interface{}{since, userID}

	// Add OR clause for each organization
	if len(orgIDs) > 0 {
		query += " OR (organization_id IN ("
		for i := range orgIDs {
			if i > 0 {
				query += ","
			}
			query += "?"
			args = append(args, orgIDs[i])
		}
		query += ") AND visibility = 'shared')"
	}

	query += ") ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list accessible queries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			// Log error but don't return it as defer executes after return
		}
	}()

	var queries []SavedQuery
	for rows.Next() {
		var sq SavedQuery
		var description, connectionID, tagsJSON, metadataJSON, organizationID sql.NullString

		if err := rows.Scan(
			&sq.ID, &sq.Name, &description, &sq.Query, &connectionID,
			&tagsJSON, &sq.Favorite, &metadataJSON,
			&sq.UserID, &organizationID, &sq.Visibility,
			&sq.CreatedAt, &sq.UpdatedAt, &sq.SyncVersion,
		); err != nil {
			return nil, fmt.Errorf("failed to scan saved query: %w", err)
		}

		if description.Valid {
			sq.Description = description.String
		}
		if connectionID.Valid {
			sq.ConnectionID = connectionID.String
		}
		if organizationID.Valid {
			orgID := organizationID.String
			sq.OrganizationID = &orgID
		}
		if tagsJSON.Valid && tagsJSON.String != "" {
			_ = json.Unmarshal([]byte(tagsJSON.String), &sq.Tags) // Best-effort unmarshal
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			_ = json.Unmarshal([]byte(metadataJSON.String), &sq.Metadata) // Best-effort unmarshal
		}

		queries = append(queries, sq)
	}

	return queries, nil
}

// SaveSyncLog saves a sync log entry
func (s *TursoStore) SaveSyncLog(ctx context.Context, log *SyncLog) error {
	query := `
		INSERT INTO sync_logs (
			id, user_id, organization_id, action, resource_count, conflict_count,
			device_id, client_version, synced_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		log.ID, log.UserID, log.OrganizationID, log.Action,
		log.ResourceCount, log.ConflictCount, log.DeviceID,
		log.ClientVersion, log.SyncedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save sync log: %w", err)
	}

	return nil
}

// ListSyncLogs retrieves sync logs for a user
func (s *TursoStore) ListSyncLogs(ctx context.Context, userID string, limit int) ([]SyncLog, error) {
	query := `
		SELECT id, user_id, organization_id, action, resource_count, conflict_count,
		       device_id, client_version, synced_at
		FROM sync_logs
		WHERE user_id = ?
		ORDER BY synced_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list sync logs: %w", err)
	}
	defer func() { _ = rows.Close() }() // Best-effort close

	var logs []SyncLog
	for rows.Next() {
		var log SyncLog
		var orgID, clientVersion sql.NullString

		if err := rows.Scan(
			&log.ID, &log.UserID, &orgID, &log.Action,
			&log.ResourceCount, &log.ConflictCount, &log.DeviceID,
			&clientVersion, &log.SyncedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan sync log: %w", err)
		}

		if orgID.Valid {
			orgIDStr := orgID.String
			log.OrganizationID = &orgIDStr
		}
		if clientVersion.Valid {
			log.ClientVersion = clientVersion.String
		}

		logs = append(logs, log)
	}

	return logs, nil
}

// Close closes the database connection
func (s *TursoStore) Close() error {
	return s.db.Close()
}
