package quotas

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store handles quota and usage persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new quota store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetQuota retrieves quotas for an organization
func (s *Store) GetQuota(ctx context.Context, orgID string) (*OrganizationQuota, error) {
	query := `
		SELECT organization_id, max_connections, max_queries_per_day,
		       max_storage_mb, max_api_calls_per_hour, max_concurrent_queries,
		       max_team_members, features_enabled, created_at, updated_at
		FROM organization_quotas
		WHERE organization_id = ?
	`

	var quota OrganizationQuota
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, orgID).Scan(
		&quota.OrganizationID,
		&quota.MaxConnections,
		&quota.MaxQueriesPerDay,
		&quota.MaxStorageMB,
		&quota.MaxAPICallsPerHour,
		&quota.MaxConcurrentQueries,
		&quota.MaxTeamMembers,
		&quota.FeaturesEnabled,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default quotas
		return s.getDefaultQuota(orgID), nil
	}
	if err != nil {
		return nil, fmt.Errorf("query quota: %w", err)
	}

	quota.CreatedAt = time.Unix(createdAt, 0)
	quota.UpdatedAt = time.Unix(updatedAt, 0)

	return &quota, nil
}

// CreateQuota creates quota settings for an organization
func (s *Store) CreateQuota(ctx context.Context, quota *OrganizationQuota) error {
	query := `
		INSERT INTO organization_quotas (
			organization_id, max_connections, max_queries_per_day,
			max_storage_mb, max_api_calls_per_hour, max_concurrent_queries,
			max_team_members, features_enabled, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, query,
		quota.OrganizationID,
		quota.MaxConnections,
		quota.MaxQueriesPerDay,
		quota.MaxStorageMB,
		quota.MaxAPICallsPerHour,
		quota.MaxConcurrentQueries,
		quota.MaxTeamMembers,
		quota.FeaturesEnabled,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("create quota: %w", err)
	}

	return nil
}

// UpdateQuota updates quota settings
func (s *Store) UpdateQuota(ctx context.Context, quota *OrganizationQuota) error {
	query := `
		UPDATE organization_quotas
		SET max_connections = ?, max_queries_per_day = ?,
		    max_storage_mb = ?, max_api_calls_per_hour = ?,
		    max_concurrent_queries = ?, max_team_members = ?,
		    features_enabled = ?, updated_at = ?
		WHERE organization_id = ?
	`

	now := time.Now().Unix()
	result, err := s.db.ExecContext(ctx, query,
		quota.MaxConnections,
		quota.MaxQueriesPerDay,
		quota.MaxStorageMB,
		quota.MaxAPICallsPerHour,
		quota.MaxConcurrentQueries,
		quota.MaxTeamMembers,
		quota.FeaturesEnabled,
		now,
		quota.OrganizationID,
	)

	if err != nil {
		return fmt.Errorf("update quota: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("quota not found")
	}

	return nil
}

// GetTodayUsage retrieves usage for today
func (s *Store) GetTodayUsage(ctx context.Context, orgID string) (*OrganizationUsage, error) {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	return s.GetUsageForDate(ctx, orgID, startOfDay)
}

// GetUsageForDate retrieves usage for a specific date
func (s *Store) GetUsageForDate(ctx context.Context, orgID string, date time.Time) (*OrganizationUsage, error) {
	// Normalize to start of day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	dateUnix := startOfDay.Unix()

	query := `
		SELECT id, organization_id, usage_date, connections_count,
		       queries_count, storage_used_mb, api_calls_count,
		       concurrent_queries_peak, created_at, updated_at
		FROM organization_usage
		WHERE organization_id = ? AND usage_date = ?
	`

	var usage OrganizationUsage
	var usageDate, createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, orgID, dateUnix).Scan(
		&usage.ID,
		&usage.OrganizationID,
		&usageDate,
		&usage.ConnectionsCount,
		&usage.QueriesCount,
		&usage.StorageUsedMB,
		&usage.APICallsCount,
		&usage.ConcurrentQueriesPeak,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		// Create empty usage record
		return s.createEmptyUsage(ctx, orgID, startOfDay)
	}
	if err != nil {
		return nil, fmt.Errorf("query usage: %w", err)
	}

	usage.UsageDate = time.Unix(usageDate, 0)
	usage.CreatedAt = time.Unix(createdAt, 0)
	usage.UpdatedAt = time.Unix(updatedAt, 0)

	return &usage, nil
}

// IncrementUsage increments usage counter for a resource type
func (s *Store) IncrementUsage(ctx context.Context, orgID string, resourceType ResourceType, amount int) error {
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dateUnix := startOfDay.Unix()

	// Determine which column to increment
	var column string
	switch resourceType {
	case ResourceConnection:
		column = "connections_count"
	case ResourceQuery:
		column = "queries_count"
	case ResourceAPI:
		column = "api_calls_count"
	default:
		return fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	// Upsert usage record
	// #nosec G201 - column name from validated enum type, safe for SQL formatting
	query := fmt.Sprintf(`
		INSERT INTO organization_usage (
			id, organization_id, usage_date, %s, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(organization_id, usage_date) DO UPDATE
		SET %s = %s + ?, updated_at = ?
	`, column, column, column)

	nowUnix := now.Unix()
	_, err := s.db.ExecContext(ctx, query,
		uuid.New().String(),
		orgID,
		dateUnix,
		amount,
		nowUnix,
		nowUnix,
		amount,
		nowUnix,
	)

	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}

	return nil
}

// GetUsageHistory retrieves usage history for the last N days
func (s *Store) GetUsageHistory(ctx context.Context, orgID string, days int) ([]*OrganizationUsage, error) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -days)
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	query := `
		SELECT id, organization_id, usage_date, connections_count,
		       queries_count, storage_used_mb, api_calls_count,
		       concurrent_queries_peak, created_at, updated_at
		FROM organization_usage
		WHERE organization_id = ? AND usage_date >= ?
		ORDER BY usage_date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, startOfDay.Unix())
	if err != nil {
		return nil, fmt.Errorf("query usage history: %w", err)
	}
	defer rows.Close()

	var usageList []*OrganizationUsage
	for rows.Next() {
		var usage OrganizationUsage
		var usageDate, createdAt, updatedAt int64

		err := rows.Scan(
			&usage.ID,
			&usage.OrganizationID,
			&usageDate,
			&usage.ConnectionsCount,
			&usage.QueriesCount,
			&usage.StorageUsedMB,
			&usage.APICallsCount,
			&usage.ConcurrentQueriesPeak,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan usage: %w", err)
		}

		usage.UsageDate = time.Unix(usageDate, 0)
		usage.CreatedAt = time.Unix(createdAt, 0)
		usage.UpdatedAt = time.Unix(updatedAt, 0)

		usageList = append(usageList, &usage)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usage: %w", err)
	}

	return usageList, nil
}

// createEmptyUsage creates an empty usage record for a date
func (s *Store) createEmptyUsage(ctx context.Context, orgID string, date time.Time) (*OrganizationUsage, error) {
	usage := &OrganizationUsage{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UsageDate:      date,
	}

	query := `
		INSERT INTO organization_usage (
			id, organization_id, usage_date, connections_count,
			queries_count, storage_used_mb, api_calls_count,
			concurrent_queries_peak, created_at, updated_at
		) VALUES (?, ?, ?, 0, 0, 0, 0, 0, ?, ?)
	`

	now := time.Now().Unix()
	_, err := s.db.ExecContext(ctx, query,
		usage.ID,
		orgID,
		date.Unix(),
		now,
		now,
	)

	if err != nil {
		return nil, fmt.Errorf("create empty usage: %w", err)
	}

	return usage, nil
}

// getDefaultQuota returns default quota settings
func (s *Store) getDefaultQuota(orgID string) *OrganizationQuota {
	return &OrganizationQuota{
		OrganizationID:       orgID,
		MaxConnections:       10,
		MaxQueriesPerDay:     1000,
		MaxStorageMB:         100,
		MaxAPICallsPerHour:   1000,
		MaxConcurrentQueries: 5,
		MaxTeamMembers:       5,
		FeaturesEnabled:      "basic",
	}
}
