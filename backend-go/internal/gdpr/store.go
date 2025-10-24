package gdpr

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store handles persistence of GDPR-related data
type Store interface {
	// Export requests
	CreateExportRequest(ctx context.Context, request *DataExportRequest) error
	GetExportRequest(ctx context.Context, requestID string) (*DataExportRequest, error)
	GetUserExportRequests(ctx context.Context, userID string) ([]*DataExportRequest, error)
	UpdateRequestStatus(ctx context.Context, requestID, status string) error
	UpdateRequestComplete(ctx context.Context, requestID, exportURL string) error
	UpdateRequestFailed(ctx context.Context, requestID, errorMessage string) error

	// User data retrieval
	GetUserData(ctx context.Context, userID string) (interface{}, error)
	GetConnections(ctx context.Context, userID string) ([]interface{}, error)
	GetQueries(ctx context.Context, userID string) ([]interface{}, error)
	GetQueryHistory(ctx context.Context, userID string) ([]interface{}, error)
	GetTemplates(ctx context.Context, userID string) ([]interface{}, error)
	GetSchedules(ctx context.Context, userID string) ([]interface{}, error)
	GetOrganizations(ctx context.Context, userID string) ([]interface{}, error)
	GetAuditLogs(ctx context.Context, userID string) ([]interface{}, error)

	// User data deletion
	DeleteConnections(ctx context.Context, userID string) (int64, error)
	DeleteQueries(ctx context.Context, userID string) (int64, error)
	DeleteQueryHistory(ctx context.Context, userID string) (int64, error)
	DeleteTemplates(ctx context.Context, userID string) (int64, error)
	DeleteSchedules(ctx context.Context, userID string) (int64, error)
	AnonymizeAuditLogs(ctx context.Context, userID string) (int64, error)
	DeleteUser(ctx context.Context, userID string) error
}

type store struct {
	db *sql.DB
}

// NewStore creates a new GDPR store
func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreateExportRequest(ctx context.Context, request *DataExportRequest) error {
	if request.ID == "" {
		request.ID = uuid.New().String()
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now()
	}

	query := `
		INSERT INTO data_export_requests (
			id, user_id, organization_id, request_type, status,
			export_url, requested_at, completed_at, error_message, metadata
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var completedAtUnix *int64
	if !request.CompletedAt.IsZero() {
		unix := request.CompletedAt.Unix()
		completedAtUnix = &unix
	}

	_, err := s.db.ExecContext(ctx, query,
		request.ID,
		request.UserID,
		request.OrganizationID,
		request.RequestType,
		request.Status,
		request.ExportURL,
		request.RequestedAt.Unix(),
		completedAtUnix,
		request.ErrorMessage,
		request.Metadata,
	)

	return err
}

func (s *store) GetExportRequest(ctx context.Context, requestID string) (*DataExportRequest, error) {
	query := `
		SELECT id, user_id, organization_id, request_type, status,
			export_url, requested_at, completed_at, error_message, metadata
		FROM data_export_requests
		WHERE id = ?
	`

	var request DataExportRequest
	var requestedAtUnix int64
	var completedAtUnix sql.NullInt64
	var orgID, exportURL, errorMsg, metadata sql.NullString

	err := s.db.QueryRowContext(ctx, query, requestID).Scan(
		&request.ID,
		&request.UserID,
		&orgID,
		&request.RequestType,
		&request.Status,
		&exportURL,
		&requestedAtUnix,
		&completedAtUnix,
		&errorMsg,
		&metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("export request not found")
		}
		return nil, err
	}

	request.RequestedAt = time.Unix(requestedAtUnix, 0)
	if completedAtUnix.Valid {
		request.CompletedAt = time.Unix(completedAtUnix.Int64, 0)
	}
	if orgID.Valid {
		request.OrganizationID = orgID.String
	}
	if exportURL.Valid {
		request.ExportURL = exportURL.String
	}
	if errorMsg.Valid {
		request.ErrorMessage = errorMsg.String
	}
	if metadata.Valid {
		request.Metadata = metadata.String
	}

	return &request, nil
}

func (s *store) GetUserExportRequests(ctx context.Context, userID string) ([]*DataExportRequest, error) {
	query := `
		SELECT id, user_id, organization_id, request_type, status,
			export_url, requested_at, completed_at, error_message, metadata
		FROM data_export_requests
		WHERE user_id = ?
		ORDER BY requested_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*DataExportRequest
	for rows.Next() {
		var request DataExportRequest
		var requestedAtUnix int64
		var completedAtUnix sql.NullInt64
		var orgID, exportURL, errorMsg, metadata sql.NullString

		err := rows.Scan(
			&request.ID,
			&request.UserID,
			&orgID,
			&request.RequestType,
			&request.Status,
			&exportURL,
			&requestedAtUnix,
			&completedAtUnix,
			&errorMsg,
			&metadata,
		)
		if err != nil {
			return nil, err
		}

		request.RequestedAt = time.Unix(requestedAtUnix, 0)
		if completedAtUnix.Valid {
			request.CompletedAt = time.Unix(completedAtUnix.Int64, 0)
		}
		if orgID.Valid {
			request.OrganizationID = orgID.String
		}
		if exportURL.Valid {
			request.ExportURL = exportURL.String
		}
		if errorMsg.Valid {
			request.ErrorMessage = errorMsg.String
		}
		if metadata.Valid {
			request.Metadata = metadata.String
		}

		requests = append(requests, &request)
	}

	return requests, rows.Err()
}

func (s *store) UpdateRequestStatus(ctx context.Context, requestID, status string) error {
	query := `UPDATE data_export_requests SET status = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, status, requestID)
	return err
}

func (s *store) UpdateRequestComplete(ctx context.Context, requestID, exportURL string) error {
	query := `
		UPDATE data_export_requests
		SET status = 'completed', export_url = ?, completed_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, exportURL, time.Now().Unix(), requestID)
	return err
}

func (s *store) UpdateRequestFailed(ctx context.Context, requestID, errorMessage string) error {
	query := `
		UPDATE data_export_requests
		SET status = 'failed', error_message = ?, completed_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, errorMessage, time.Now().Unix(), requestID)
	return err
}

// Data retrieval methods
func (s *store) GetUserData(ctx context.Context, userID string) (interface{}, error) {
	query := `SELECT * FROM users WHERE id = ?`
	return s.queryToMaps(ctx, query, userID)
}

func (s *store) GetConnections(ctx context.Context, userID string) ([]interface{}, error) {
	query := `SELECT * FROM connections WHERE user_id = ?`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) GetQueries(ctx context.Context, userID string) ([]interface{}, error) {
	query := `SELECT * FROM queries WHERE user_id = ?`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) GetQueryHistory(ctx context.Context, userID string) ([]interface{}, error) {
	query := `SELECT * FROM query_history WHERE user_id = ?`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) GetTemplates(ctx context.Context, userID string) ([]interface{}, error) {
	query := `SELECT * FROM query_templates WHERE user_id = ?`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) GetSchedules(ctx context.Context, userID string) ([]interface{}, error) {
	query := `SELECT * FROM scheduled_queries WHERE user_id = ?`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) GetOrganizations(ctx context.Context, userID string) ([]interface{}, error) {
	query := `
		SELECT o.* FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = ?
	`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) GetAuditLogs(ctx context.Context, userID string) ([]interface{}, error) {
	query := `SELECT * FROM audit_logs WHERE user_id = ?`
	maps, err := s.queryToMaps(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	result := make([]interface{}, len(maps))
	for i, m := range maps {
		result[i] = m
	}
	return result, nil
}

func (s *store) queryToMaps(ctx context.Context, query string, args ...interface{}) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, rows.Err()
}

// Deletion methods
func (s *store) DeleteConnections(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM connections WHERE user_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteQueries(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM queries WHERE user_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteQueryHistory(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM query_history WHERE user_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteTemplates(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM query_templates WHERE user_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteSchedules(ctx context.Context, userID string) (int64, error) {
	query := `DELETE FROM scheduled_queries WHERE user_id = ?`
	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) AnonymizeAuditLogs(ctx context.Context, userID string) (int64, error) {
	// Replace user info with anonymized data but keep logs for compliance
	query := `
		UPDATE audit_logs
		SET user_id = '[DELETED]',
		    metadata = json_set(metadata, '$.anonymized', true, '$.anonymized_at', ?)
		WHERE user_id = ?
	`
	result, err := s.db.ExecContext(ctx, query, time.Now().Unix(), userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteUser(ctx context.Context, userID string) error {
	query := `DELETE FROM users WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, userID)
	return err
}
