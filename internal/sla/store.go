package sla

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
)

// Store handles SLA metrics persistence
type Store struct {
	db *sql.DB
}

// NewStore creates a new SLA store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// LogRequest logs a request for SLA calculation
func (s *Store) LogRequest(ctx context.Context, log *RequestLog) error {
	query := `
		INSERT INTO request_log (
			id, organization_id, endpoint, method,
			response_time_ms, status_code, success, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		log.ID,
		log.OrganizationID,
		log.Endpoint,
		log.Method,
		log.ResponseTimeMS,
		log.StatusCode,
		log.Success,
		log.CreatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("log request: %w", err)
	}

	return nil
}

// GetRequestsForDay retrieves all requests for a specific day
func (s *Store) GetRequestsForDay(ctx context.Context, orgID string, date time.Time) ([]*RequestLog, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, organization_id, endpoint, method,
		       response_time_ms, status_code, success, created_at
		FROM request_log
		WHERE organization_id = ? AND created_at >= ? AND created_at < ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, startOfDay.Unix(), endOfDay.Unix())
	if err != nil {
		return nil, fmt.Errorf("query requests: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var requests []*RequestLog
	for rows.Next() {
		var r RequestLog
		var createdAt int64

		err := rows.Scan(
			&r.ID,
			&r.OrganizationID,
			&r.Endpoint,
			&r.Method,
			&r.ResponseTimeMS,
			&r.StatusCode,
			&r.Success,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan request: %w", err)
		}

		r.CreatedAt = time.Unix(createdAt, 0)
		requests = append(requests, &r)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate requests: %w", err)
	}

	return requests, nil
}

// SaveSLAMetrics saves calculated SLA metrics
func (s *Store) SaveSLAMetrics(ctx context.Context, metrics *SLAMetrics) error {
	// Upsert metrics
	query := `
		INSERT INTO sla_metrics (
			id, organization_id, metric_date, uptime_percentage,
			avg_response_time_ms, error_rate, p95_response_time_ms,
			p99_response_time_ms, total_requests, failed_requests,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(organization_id, metric_date) DO UPDATE
		SET uptime_percentage = ?,
		    avg_response_time_ms = ?,
		    error_rate = ?,
		    p95_response_time_ms = ?,
		    p99_response_time_ms = ?,
		    total_requests = ?,
		    failed_requests = ?,
		    updated_at = ?
	`

	now := time.Now().Unix()
	metricDateUnix := metrics.MetricDate.Unix()

	_, err := s.db.ExecContext(ctx, query,
		metrics.ID,
		metrics.OrganizationID,
		metricDateUnix,
		metrics.UptimePercentage,
		metrics.AvgResponseTimeMS,
		metrics.ErrorRate,
		metrics.P95ResponseTimeMS,
		metrics.P99ResponseTimeMS,
		metrics.TotalRequests,
		metrics.FailedRequests,
		now,
		now,
		// ON CONFLICT UPDATE values
		metrics.UptimePercentage,
		metrics.AvgResponseTimeMS,
		metrics.ErrorRate,
		metrics.P95ResponseTimeMS,
		metrics.P99ResponseTimeMS,
		metrics.TotalRequests,
		metrics.FailedRequests,
		now,
	)

	if err != nil {
		return fmt.Errorf("save SLA metrics: %w", err)
	}

	return nil
}

// GetMetricsForPeriod retrieves SLA metrics for a date range
func (s *Store) GetMetricsForPeriod(ctx context.Context, orgID string, startDate, endDate time.Time) ([]*SLAMetrics, error) {
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())

	query := `
		SELECT id, organization_id, metric_date, uptime_percentage,
		       avg_response_time_ms, error_rate, p95_response_time_ms,
		       p99_response_time_ms, total_requests, failed_requests,
		       created_at, updated_at
		FROM sla_metrics
		WHERE organization_id = ? AND metric_date >= ? AND metric_date <= ?
		ORDER BY metric_date ASC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, startOfDay.Unix(), endOfDay.Unix())
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var metricsList []*SLAMetrics
	for rows.Next() {
		var m SLAMetrics
		var metricDate, createdAt, updatedAt int64

		err := rows.Scan(
			&m.ID,
			&m.OrganizationID,
			&metricDate,
			&m.UptimePercentage,
			&m.AvgResponseTimeMS,
			&m.ErrorRate,
			&m.P95ResponseTimeMS,
			&m.P99ResponseTimeMS,
			&m.TotalRequests,
			&m.FailedRequests,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan metrics: %w", err)
		}

		m.MetricDate = time.Unix(metricDate, 0)
		m.CreatedAt = time.Unix(createdAt, 0)
		m.UpdatedAt = time.Unix(updatedAt, 0)

		metricsList = append(metricsList, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metrics: %w", err)
	}

	return metricsList, nil
}

// CleanupOldLogs removes request logs older than retention period
func (s *Store) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)

	query := `DELETE FROM request_log WHERE created_at < ?`

	result, err := s.db.ExecContext(ctx, query, cutoffDate.Unix())
	if err != nil {
		return fmt.Errorf("cleanup old logs: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows > 0 {
		fmt.Printf("Cleaned up %d old request logs\n", rows)
	}

	return nil
}

// GetLatestMetrics retrieves the most recent SLA metrics for an organization
func (s *Store) GetLatestMetrics(ctx context.Context, orgID string, limit int) ([]*SLAMetrics, error) {
	query := `
		SELECT id, organization_id, metric_date, uptime_percentage,
		       avg_response_time_ms, error_rate, p95_response_time_ms,
		       p99_response_time_ms, total_requests, failed_requests,
		       created_at, updated_at
		FROM sla_metrics
		WHERE organization_id = ?
		ORDER BY metric_date DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, limit)
	if err != nil {
		return nil, fmt.Errorf("query latest metrics: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var metricsList []*SLAMetrics
	for rows.Next() {
		var m SLAMetrics
		var metricDate, createdAt, updatedAt int64

		err := rows.Scan(
			&m.ID,
			&m.OrganizationID,
			&metricDate,
			&m.UptimePercentage,
			&m.AvgResponseTimeMS,
			&m.ErrorRate,
			&m.P95ResponseTimeMS,
			&m.P99ResponseTimeMS,
			&m.TotalRequests,
			&m.FailedRequests,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan metrics: %w", err)
		}

		m.MetricDate = time.Unix(metricDate, 0)
		m.CreatedAt = time.Unix(createdAt, 0)
		m.UpdatedAt = time.Unix(updatedAt, 0)

		metricsList = append(metricsList, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metrics: %w", err)
	}

	return metricsList, nil
}

// CreateRequestLog creates a new request log entry (helper for LogRequest)
func CreateRequestLog(orgID, endpoint, method string, responseTimeMS, statusCode int) *RequestLog {
	success := statusCode >= 200 && statusCode < 500 // 5xx errors are failures
	return &RequestLog{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Endpoint:       endpoint,
		Method:         method,
		ResponseTimeMS: responseTimeMS,
		StatusCode:     statusCode,
		Success:        success,
		CreatedAt:      time.Now(),
	}
}
