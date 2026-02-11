package turso

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SavedQuery represents a saved query
type SavedQuery struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Query          string                 `json:"query"`
	ConnectionID   string                 `json:"connection_id,omitempty"`
	Tags           []string               `json:"tags,omitempty"`
	Favorite       bool                   `json:"favorite"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	SyncVersion    int                    `json:"sync_version"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	Visibility     string                 `json:"visibility"` // 'personal' or 'shared'
	CreatedBy      string                 `json:"created_by"`
}

// QueryStore handles database operations for saved queries
type QueryStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewQueryStore creates a new query store
func NewQueryStore(db *sql.DB, logger *logrus.Logger) *QueryStore {
	return &QueryStore{
		db:     db,
		logger: logger,
	}
}

// Create creates a new saved query
func (s *QueryStore) Create(ctx context.Context, query *SavedQuery) error {
	if query.ID == "" {
		query.ID = uuid.New().String()
	}

	now := time.Now()
	query.CreatedAt = now
	query.UpdatedAt = now
	query.SyncVersion = 1

	if query.Visibility == "" {
		query.Visibility = "personal"
	}

	tagsJSON, err := json.Marshal(query.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(query.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	querySQL := `
		INSERT INTO saved_queries_sync (
			id, user_id, name, description, query_text, connection_id, tags,
			favorite, metadata, created_at, updated_at, sync_version,
			organization_id, visibility, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, querySQL,
		query.ID, query.UserID, query.Name, query.Description, query.Query,
		query.ConnectionID, string(tagsJSON), query.Favorite, string(metadataJSON),
		query.CreatedAt.Unix(), query.UpdatedAt.Unix(), query.SyncVersion,
		query.OrganizationID, query.Visibility, query.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create saved query: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"query_id":   query.ID,
		"user_id":    query.UserID,
		"name":       query.Name,
		"visibility": query.Visibility,
	}).Info("Saved query created")

	return nil
}

// GetByID retrieves a saved query by ID
func (s *QueryStore) GetByID(ctx context.Context, id string) (*SavedQuery, error) {
	querySQL := `
		SELECT id, user_id, name, description, query_text, connection_id, tags,
		       favorite, metadata, created_at, updated_at, sync_version,
		       organization_id, visibility, created_by
		FROM saved_queries_sync
		WHERE id = ? AND deleted_at IS NULL
	`

	var query SavedQuery
	var tagsJSON, metadataJSON sql.NullString
	var orgID sql.NullString
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, querySQL, id).Scan(
		&query.ID, &query.UserID, &query.Name, &query.Description, &query.Query,
		&query.ConnectionID, &tagsJSON, &query.Favorite, &metadataJSON,
		&createdAt, &updatedAt, &query.SyncVersion,
		&orgID, &query.Visibility, &query.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("saved query not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query saved query: %w", err)
	}

	query.CreatedAt = time.Unix(createdAt, 0)
	query.UpdatedAt = time.Unix(updatedAt, 0)

	if orgID.Valid {
		query.OrganizationID = &orgID.String
	}

	if tagsJSON.Valid && tagsJSON.String != "" {
		if err := json.Unmarshal([]byte(tagsJSON.String), &query.Tags); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal tags")
		}
	}

	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &query.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal metadata")
		}
	}

	return &query, nil
}

// GetByUserID retrieves all queries for a user (personal only)
func (s *QueryStore) GetByUserID(ctx context.Context, userID string) ([]*SavedQuery, error) {
	querySQL := `
		SELECT id, user_id, name, description, query_text, connection_id, tags,
		       favorite, metadata, created_at, updated_at, sync_version,
		       organization_id, visibility, created_by
		FROM saved_queries_sync
		WHERE user_id = ? AND visibility = 'personal' AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	return s.queryQueries(ctx, querySQL, userID)
}

// GetQueriesByOrganization returns all queries in an organization
// Filters by visibility: 'shared' queries visible to all org members
func (s *QueryStore) GetQueriesByOrganization(ctx context.Context, orgID string) ([]*SavedQuery, error) {
	querySQL := `
		SELECT id, user_id, name, description, query_text, connection_id, tags,
		       favorite, metadata, created_at, updated_at, sync_version,
		       organization_id, visibility, created_by
		FROM saved_queries_sync
		WHERE organization_id = ? AND visibility = 'shared' AND deleted_at IS NULL
		ORDER BY created_at DESC
	`

	return s.queryQueries(ctx, querySQL, orgID)
}

// GetSharedQueries returns all shared queries accessible to a user
// Includes: personal queries + shared queries in user's organizations
func (s *QueryStore) GetSharedQueries(ctx context.Context, userID string) ([]*SavedQuery, error) {
	querySQL := `
		SELECT DISTINCT sq.id, sq.user_id, sq.name, sq.description, sq.query_text, sq.connection_id, sq.tags,
		       sq.favorite, sq.metadata, sq.created_at, sq.updated_at, sq.sync_version,
		       sq.organization_id, sq.visibility, sq.created_by
		FROM saved_queries_sync sq
		LEFT JOIN organization_members om ON sq.organization_id = om.organization_id
		WHERE (
			(sq.user_id = ? AND sq.visibility = 'personal')
			OR (sq.visibility = 'shared' AND om.user_id = ?)
		)
		AND sq.deleted_at IS NULL
		ORDER BY sq.created_at DESC
	`

	return s.queryQueries(ctx, querySQL, userID, userID)
}

// Update updates a saved query
func (s *QueryStore) Update(ctx context.Context, query *SavedQuery) error {
	query.UpdatedAt = time.Now()
	query.SyncVersion++

	tagsJSON, err := json.Marshal(query.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	metadataJSON, err := json.Marshal(query.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	querySQL := `
		UPDATE saved_queries_sync
		SET name = ?, description = ?, query_text = ?, connection_id = ?, tags = ?,
		    favorite = ?, metadata = ?, updated_at = ?, sync_version = ?,
		    organization_id = ?, visibility = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, querySQL,
		query.Name, query.Description, query.Query, query.ConnectionID, string(tagsJSON),
		query.Favorite, string(metadataJSON), query.UpdatedAt.Unix(), query.SyncVersion,
		query.OrganizationID, query.Visibility,
		query.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update saved query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("saved query not found or already deleted")
	}

	s.logger.WithField("query_id", query.ID).Info("Saved query updated")
	return nil
}

// UpdateQueryVisibility changes visibility (personal -> shared or vice versa)
// Only owner or admins can change visibility
func (s *QueryStore) UpdateQueryVisibility(ctx context.Context, queryID, userID string, visibility string) error {
	if visibility != "personal" && visibility != "shared" {
		return fmt.Errorf("invalid visibility: must be 'personal' or 'shared'")
	}

	// Get current query to verify ownership
	query, err := s.GetByID(ctx, queryID)
	if err != nil {
		return fmt.Errorf("failed to get query: %w", err)
	}

	// Verify user is the owner
	if query.CreatedBy != userID {
		return fmt.Errorf("only the creator can change visibility")
	}

	querySQL := `
		UPDATE saved_queries_sync
		SET visibility = ?, updated_at = ?, sync_version = sync_version + 1
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, querySQL, visibility, time.Now().Unix(), queryID)
	if err != nil {
		return fmt.Errorf("failed to update visibility: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("query not found or already deleted")
	}

	s.logger.WithFields(logrus.Fields{
		"query_id":   queryID,
		"visibility": visibility,
		"user_id":    userID,
	}).Info("Query visibility updated")

	return nil
}

// Delete soft-deletes a saved query
func (s *QueryStore) Delete(ctx context.Context, id string) error {
	querySQL := `
		UPDATE saved_queries_sync
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, querySQL, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete saved query: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("saved query not found or already deleted")
	}

	s.logger.WithField("query_id", id).Info("Saved query deleted")
	return nil
}

// queryQueries is a helper method to query and scan multiple queries
func (s *QueryStore) queryQueries(ctx context.Context, querySQL string, args ...interface{}) ([]*SavedQuery, error) {
	rows, err := s.db.QueryContext(ctx, querySQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query saved queries: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var queries []*SavedQuery
	for rows.Next() {
		var query SavedQuery
		var tagsJSON, metadataJSON sql.NullString
		var orgID sql.NullString
		var createdAt, updatedAt int64

		err := rows.Scan(
			&query.ID, &query.UserID, &query.Name, &query.Description, &query.Query,
			&query.ConnectionID, &tagsJSON, &query.Favorite, &metadataJSON,
			&createdAt, &updatedAt, &query.SyncVersion,
			&orgID, &query.Visibility, &query.CreatedBy,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan saved query: %w", err)
		}

		query.CreatedAt = time.Unix(createdAt, 0)
		query.UpdatedAt = time.Unix(updatedAt, 0)

		if orgID.Valid {
			query.OrganizationID = &orgID.String
		}

		if tagsJSON.Valid && tagsJSON.String != "" {
			if err := json.Unmarshal([]byte(tagsJSON.String), &query.Tags); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal tags")
			}
		}

		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &query.Metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal metadata")
			}
		}

		queries = append(queries, &query)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating queries: %w", err)
	}

	return queries, nil
}
