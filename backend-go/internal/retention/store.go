package retention

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store handles persistence of retention policies and archive logs
type Store interface {
	// Retention policies
	CreatePolicy(ctx context.Context, policy *RetentionPolicy) error
	GetPolicy(ctx context.Context, orgID, resourceType string) (*RetentionPolicy, error)
	GetAllPolicies(ctx context.Context) ([]*RetentionPolicy, error)
	GetOrganizationPolicies(ctx context.Context, orgID string) ([]*RetentionPolicy, error)
	UpdatePolicy(ctx context.Context, policy *RetentionPolicy) error
	DeletePolicy(ctx context.Context, orgID, resourceType string) error

	// Archive logs
	CreateArchiveLog(ctx context.Context, log *ArchiveLog) error
	GetArchiveLogs(ctx context.Context, orgID string, since time.Time) ([]*ArchiveLog, error)

	// Data retrieval for archival
	GetOldQueryHistory(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error)
	GetOldAuditLogs(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error)
	GetOldConnections(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error)
	GetOldTemplates(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error)

	// Data deletion
	DeleteQueryHistory(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error)
	DeleteAuditLogs(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error)
	DeleteConnections(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error)
	DeleteTemplates(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error)

	// Statistics
	GetRetentionStats(ctx context.Context, orgID, resourceType string) (*RetentionStats, error)
}

type store struct {
	db *sql.DB
}

// NewStore creates a new retention store
func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreatePolicy(ctx context.Context, policy *RetentionPolicy) error {
	if policy.ID == "" {
		policy.ID = uuid.New().String()
	}
	now := time.Now()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = now
	}
	if policy.UpdatedAt.IsZero() {
		policy.UpdatedAt = now
	}

	query := `
		INSERT INTO data_retention_policies (
			id, organization_id, resource_type, retention_days,
			auto_archive, archive_location, created_at, updated_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		policy.ID,
		policy.OrganizationID,
		policy.ResourceType,
		policy.RetentionDays,
		policy.AutoArchive,
		policy.ArchiveLocation,
		policy.CreatedAt.Unix(),
		policy.UpdatedAt.Unix(),
		policy.CreatedBy,
	)

	return err
}

func (s *store) GetPolicy(ctx context.Context, orgID, resourceType string) (*RetentionPolicy, error) {
	query := `
		SELECT id, organization_id, resource_type, retention_days,
			auto_archive, archive_location, created_at, updated_at, created_by
		FROM data_retention_policies
		WHERE organization_id = ? AND resource_type = ?
	`

	var policy RetentionPolicy
	var createdAtUnix, updatedAtUnix int64
	var archiveLocation sql.NullString

	err := s.db.QueryRowContext(ctx, query, orgID, resourceType).Scan(
		&policy.ID,
		&policy.OrganizationID,
		&policy.ResourceType,
		&policy.RetentionDays,
		&policy.AutoArchive,
		&archiveLocation,
		&createdAtUnix,
		&updatedAtUnix,
		&policy.CreatedBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("policy not found")
		}
		return nil, err
	}

	policy.CreatedAt = time.Unix(createdAtUnix, 0)
	policy.UpdatedAt = time.Unix(updatedAtUnix, 0)
	if archiveLocation.Valid {
		policy.ArchiveLocation = archiveLocation.String
	}

	return &policy, nil
}

func (s *store) GetAllPolicies(ctx context.Context) ([]*RetentionPolicy, error) {
	query := `
		SELECT id, organization_id, resource_type, retention_days,
			auto_archive, archive_location, created_at, updated_at, created_by
		FROM data_retention_policies
		ORDER BY organization_id, resource_type
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPolicies(rows)
}

func (s *store) GetOrganizationPolicies(ctx context.Context, orgID string) ([]*RetentionPolicy, error) {
	query := `
		SELECT id, organization_id, resource_type, retention_days,
			auto_archive, archive_location, created_at, updated_at, created_by
		FROM data_retention_policies
		WHERE organization_id = ?
		ORDER BY resource_type
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanPolicies(rows)
}

func (s *store) scanPolicies(rows *sql.Rows) ([]*RetentionPolicy, error) {
	var policies []*RetentionPolicy

	for rows.Next() {
		var policy RetentionPolicy
		var createdAtUnix, updatedAtUnix int64
		var archiveLocation sql.NullString

		err := rows.Scan(
			&policy.ID,
			&policy.OrganizationID,
			&policy.ResourceType,
			&policy.RetentionDays,
			&policy.AutoArchive,
			&archiveLocation,
			&createdAtUnix,
			&updatedAtUnix,
			&policy.CreatedBy,
		)
		if err != nil {
			return nil, err
		}

		policy.CreatedAt = time.Unix(createdAtUnix, 0)
		policy.UpdatedAt = time.Unix(updatedAtUnix, 0)
		if archiveLocation.Valid {
			policy.ArchiveLocation = archiveLocation.String
		}

		policies = append(policies, &policy)
	}

	return policies, rows.Err()
}

func (s *store) UpdatePolicy(ctx context.Context, policy *RetentionPolicy) error {
	policy.UpdatedAt = time.Now()

	query := `
		UPDATE data_retention_policies
		SET retention_days = ?, auto_archive = ?, archive_location = ?, updated_at = ?
		WHERE organization_id = ? AND resource_type = ?
	`

	_, err := s.db.ExecContext(ctx, query,
		policy.RetentionDays,
		policy.AutoArchive,
		policy.ArchiveLocation,
		policy.UpdatedAt.Unix(),
		policy.OrganizationID,
		policy.ResourceType,
	)

	return err
}

func (s *store) DeletePolicy(ctx context.Context, orgID, resourceType string) error {
	query := `DELETE FROM data_retention_policies WHERE organization_id = ? AND resource_type = ?`
	_, err := s.db.ExecContext(ctx, query, orgID, resourceType)
	return err
}

func (s *store) CreateArchiveLog(ctx context.Context, log *ArchiveLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO data_archive_log (
			id, organization_id, resource_type, records_archived,
			archive_location, archive_date, cutoff_date, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		log.ID,
		log.OrganizationID,
		log.ResourceType,
		log.RecordsArchived,
		log.ArchiveLocation,
		log.ArchiveDate.Unix(),
		log.CutoffDate.Unix(),
		log.CreatedAt.Unix(),
	)

	return err
}

func (s *store) GetArchiveLogs(ctx context.Context, orgID string, since time.Time) ([]*ArchiveLog, error) {
	query := `
		SELECT id, organization_id, resource_type, records_archived,
			archive_location, archive_date, cutoff_date, created_at
		FROM data_archive_log
		WHERE organization_id = ? AND archive_date >= ?
		ORDER BY archive_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, since.Unix())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*ArchiveLog
	for rows.Next() {
		var log ArchiveLog
		var archiveDateUnix, cutoffDateUnix, createdAtUnix int64

		err := rows.Scan(
			&log.ID,
			&log.OrganizationID,
			&log.ResourceType,
			&log.RecordsArchived,
			&log.ArchiveLocation,
			&archiveDateUnix,
			&cutoffDateUnix,
			&createdAtUnix,
		)
		if err != nil {
			return nil, err
		}

		log.ArchiveDate = time.Unix(archiveDateUnix, 0)
		log.CutoffDate = time.Unix(cutoffDateUnix, 0)
		log.CreatedAt = time.Unix(createdAtUnix, 0)

		logs = append(logs, &log)
	}

	return logs, rows.Err()
}

// Data retrieval methods (simplified - would need to match actual schema)
func (s *store) GetOldQueryHistory(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error) {
	// Note: This assumes a query_history table exists with organization_id and created_at fields
	query := `SELECT * FROM query_history WHERE organization_id = ? AND created_at < ?`
	return s.queryToMaps(ctx, query, orgID, cutoffDate.Unix())
}

func (s *store) GetOldAuditLogs(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error) {
	query := `SELECT * FROM audit_logs WHERE organization_id = ? AND timestamp < ?`
	return s.queryToMaps(ctx, query, orgID, cutoffDate.Unix())
}

func (s *store) GetOldConnections(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error) {
	query := `SELECT * FROM connections WHERE organization_id = ? AND created_at < ?`
	return s.queryToMaps(ctx, query, orgID, cutoffDate.Unix())
}

func (s *store) GetOldTemplates(ctx context.Context, orgID string, cutoffDate time.Time) ([]map[string]interface{}, error) {
	query := `SELECT * FROM query_templates WHERE organization_id = ? AND created_at < ?`
	return s.queryToMaps(ctx, query, orgID, cutoffDate.Unix())
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

func (s *store) DeleteQueryHistory(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error) {
	query := `DELETE FROM query_history WHERE organization_id = ? AND created_at < ?`
	result, err := s.db.ExecContext(ctx, query, orgID, cutoffDate.Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteAuditLogs(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error) {
	query := `DELETE FROM audit_logs WHERE organization_id = ? AND timestamp < ?`
	result, err := s.db.ExecContext(ctx, query, orgID, cutoffDate.Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteConnections(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error) {
	query := `DELETE FROM connections WHERE organization_id = ? AND created_at < ?`
	result, err := s.db.ExecContext(ctx, query, orgID, cutoffDate.Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) DeleteTemplates(ctx context.Context, orgID string, cutoffDate time.Time) (int64, error) {
	query := `DELETE FROM query_templates WHERE organization_id = ? AND created_at < ?`
	result, err := s.db.ExecContext(ctx, query, orgID, cutoffDate.Unix())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *store) GetRetentionStats(ctx context.Context, orgID, resourceType string) (*RetentionStats, error) {
	stats := &RetentionStats{
		ResourceType: resourceType,
	}

	var tableName, dateColumn string
	switch resourceType {
	case "query_history":
		tableName = "query_history"
		dateColumn = "created_at"
	case "audit_logs":
		tableName = "audit_logs"
		dateColumn = "timestamp"
	case "connections":
		tableName = "connections"
		dateColumn = "created_at"
	case "templates":
		tableName = "query_templates"
		dateColumn = "created_at"
	default:
		return nil, fmt.Errorf("unknown resource type: %s", resourceType)
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*), MIN(%s)
		FROM %s
		WHERE organization_id = ?
	`, dateColumn, tableName)

	var count int
	var oldestUnix sql.NullInt64
	err := s.db.QueryRowContext(ctx, query, orgID).Scan(&count, &oldestUnix)
	if err != nil {
		return nil, err
	}

	stats.TotalRecords = count
	if oldestUnix.Valid {
		stats.OldestRecord = time.Unix(oldestUnix.Int64, 0)
	}

	return stats, nil
}
