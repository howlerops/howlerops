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

// QuerySchedule represents a scheduled query execution
type QuerySchedule struct {
	ID                string                 `json:"id"`
	TemplateID        string                 `json:"template_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description,omitempty"`
	Frequency         string                 `json:"frequency"` // Cron expression
	Parameters        map[string]interface{} `json:"parameters,omitempty"`
	LastRunAt         *time.Time             `json:"last_run_at,omitempty"`
	NextRunAt         *time.Time             `json:"next_run_at,omitempty"`
	Status            string                 `json:"status"` // 'active', 'paused', 'failed'
	CreatedBy         string                 `json:"created_by"`
	OrganizationID    *string                `json:"organization_id,omitempty"`
	NotificationEmail string                 `json:"notification_email,omitempty"`
	ResultStorage     string                 `json:"result_storage"` // 'none', 's3', 'database'
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	DeletedAt         *time.Time             `json:"deleted_at,omitempty"`

	// Computed fields (joined data)
	Template *QueryTemplate `json:"template,omitempty"`
}

// ScheduleExecution represents a single execution of a schedule
type ScheduleExecution struct {
	ID            string    `json:"id"`
	ScheduleID    string    `json:"schedule_id"`
	ExecutedAt    time.Time `json:"executed_at"`
	Status        string    `json:"status"` // 'success', 'failed', 'timeout', 'cancelled'
	DurationMs    int       `json:"duration_ms"`
	RowsReturned  int       `json:"rows_returned"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	ResultPreview string    `json:"result_preview,omitempty"` // JSON: first 10 rows
}

// ScheduleFilters for querying schedules
type ScheduleFilters struct {
	TemplateID     *string
	OrganizationID *string
	CreatedBy      *string
	Status         *string
	Limit          int
	Offset         int
}

// ScheduleStore handles database operations for query schedules
type ScheduleStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewScheduleStore creates a new schedule store
func NewScheduleStore(db *sql.DB, logger *logrus.Logger) *ScheduleStore {
	return &ScheduleStore{
		db:     db,
		logger: logger,
	}
}

// Create creates a new query schedule
func (s *ScheduleStore) Create(ctx context.Context, schedule *QuerySchedule) error {
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}

	now := time.Now()
	schedule.CreatedAt = now
	schedule.UpdatedAt = now

	if schedule.Status == "" {
		schedule.Status = "active"
	}

	if schedule.ResultStorage == "" {
		schedule.ResultStorage = "none"
	}

	// Marshal parameters
	parametersJSON, err := json.Marshal(schedule.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO query_schedules (
			id, template_id, name, description, frequency, parameters,
			last_run_at, next_run_at, status, created_by, organization_id,
			notification_email, result_storage, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var lastRunAt, nextRunAt sql.NullInt64
	if schedule.LastRunAt != nil {
		lastRunAt.Valid = true
		lastRunAt.Int64 = schedule.LastRunAt.Unix()
	}
	if schedule.NextRunAt != nil {
		nextRunAt.Valid = true
		nextRunAt.Int64 = schedule.NextRunAt.Unix()
	}

	_, err = s.db.ExecContext(ctx, query,
		schedule.ID, schedule.TemplateID, schedule.Name, schedule.Description,
		schedule.Frequency, string(parametersJSON),
		lastRunAt, nextRunAt, schedule.Status,
		schedule.CreatedBy, schedule.OrganizationID,
		schedule.NotificationEmail, schedule.ResultStorage,
		schedule.CreatedAt.Unix(), schedule.UpdatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": schedule.ID,
		"name":        schedule.Name,
		"frequency":   schedule.Frequency,
		"created_by":  schedule.CreatedBy,
	}).Info("Schedule created")

	return nil
}

// GetByID retrieves a schedule by ID
func (s *ScheduleStore) GetByID(ctx context.Context, id string) (*QuerySchedule, error) {
	query := `
		SELECT id, template_id, name, description, frequency, parameters,
		       last_run_at, next_run_at, status, created_by, organization_id,
		       notification_email, result_storage, created_at, updated_at, deleted_at
		FROM query_schedules
		WHERE id = ?
	`

	var schedule QuerySchedule
	var parametersJSON sql.NullString
	var description, notificationEmail sql.NullString
	var orgID sql.NullString
	var lastRunAt, nextRunAt, deletedAt sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID, &schedule.TemplateID, &schedule.Name, &description,
		&schedule.Frequency, &parametersJSON,
		&lastRunAt, &nextRunAt, &schedule.Status,
		&schedule.CreatedBy, &orgID,
		&notificationEmail, &schedule.ResultStorage,
		&createdAt, &updatedAt, &deletedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query schedule: %w", err)
	}

	// Convert timestamps
	schedule.CreatedAt = time.Unix(createdAt, 0)
	schedule.UpdatedAt = time.Unix(updatedAt, 0)

	if lastRunAt.Valid {
		lastRun := time.Unix(lastRunAt.Int64, 0)
		schedule.LastRunAt = &lastRun
	}

	if nextRunAt.Valid {
		nextRun := time.Unix(nextRunAt.Int64, 0)
		schedule.NextRunAt = &nextRun
	}

	if deletedAt.Valid {
		deleted := time.Unix(deletedAt.Int64, 0)
		schedule.DeletedAt = &deleted
	}

	// Convert nullable fields
	if description.Valid {
		schedule.Description = description.String
	}

	if orgID.Valid {
		schedule.OrganizationID = &orgID.String
	}

	if notificationEmail.Valid {
		schedule.NotificationEmail = notificationEmail.String
	}

	// Unmarshal parameters
	if parametersJSON.Valid && parametersJSON.String != "" && parametersJSON.String != "null" {
		if err := json.Unmarshal([]byte(parametersJSON.String), &schedule.Parameters); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal parameters")
		}
	}

	return &schedule, nil
}

// List retrieves schedules with filters
func (s *ScheduleStore) List(ctx context.Context, filters ScheduleFilters) ([]*QuerySchedule, error) {
	query := `
		SELECT id, template_id, name, description, frequency, parameters,
		       last_run_at, next_run_at, status, created_by, organization_id,
		       notification_email, result_storage, created_at, updated_at
		FROM query_schedules
		WHERE deleted_at IS NULL
	`

	args := []interface{}{}

	// Apply filters
	if filters.TemplateID != nil {
		query += " AND template_id = ?"
		args = append(args, *filters.TemplateID)
	}

	if filters.OrganizationID != nil {
		query += " AND organization_id = ?"
		args = append(args, *filters.OrganizationID)
	}

	if filters.CreatedBy != nil {
		query += " AND created_by = ?"
		args = append(args, *filters.CreatedBy)
	}

	if filters.Status != nil {
		query += " AND status = ?"
		args = append(args, *filters.Status)
	}

	// Order by next run time
	query += " ORDER BY next_run_at ASC, created_at DESC"

	// Apply pagination
	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	} else {
		query += " LIMIT 100" // Default limit
	}

	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	return s.querySchedules(ctx, query, args...)
}

// GetDueSchedules returns all active schedules that need to be executed
func (s *ScheduleStore) GetDueSchedules(ctx context.Context) ([]*QuerySchedule, error) {
	now := time.Now().Unix()

	query := `
		SELECT id, template_id, name, description, frequency, parameters,
		       last_run_at, next_run_at, status, created_by, organization_id,
		       notification_email, result_storage, created_at, updated_at
		FROM query_schedules
		WHERE status = 'active'
		  AND deleted_at IS NULL
		  AND next_run_at IS NOT NULL
		  AND next_run_at <= ?
		ORDER BY next_run_at ASC
	`

	return s.querySchedules(ctx, query, now)
}

// Update updates a schedule
func (s *ScheduleStore) Update(ctx context.Context, schedule *QuerySchedule) error {
	schedule.UpdatedAt = time.Now()

	// Marshal parameters
	parametersJSON, err := json.Marshal(schedule.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE query_schedules
		SET name = ?, description = ?, frequency = ?, parameters = ?,
		    notification_email = ?, result_storage = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query,
		schedule.Name, schedule.Description, schedule.Frequency,
		string(parametersJSON), schedule.NotificationEmail,
		schedule.ResultStorage, schedule.UpdatedAt.Unix(),
		schedule.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found or already deleted")
	}

	s.logger.WithField("schedule_id", schedule.ID).Info("Schedule updated")
	return nil
}

// UpdateStatus updates only the status of a schedule
func (s *ScheduleStore) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `
		UPDATE query_schedules
		SET status = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, status, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found or already deleted")
	}

	s.logger.WithFields(logrus.Fields{
		"schedule_id": id,
		"status":      status,
	}).Info("Schedule status updated")

	return nil
}

// UpdateNextRun updates the next_run_at and last_run_at timestamps
func (s *ScheduleStore) UpdateNextRun(ctx context.Context, id string, lastRun, nextRun time.Time) error {
	query := `
		UPDATE query_schedules
		SET last_run_at = ?, next_run_at = ?, updated_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query,
		lastRun.Unix(), nextRun.Unix(), time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to update next run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found or already deleted")
	}

	return nil
}

// Delete soft-deletes a schedule
func (s *ScheduleStore) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE query_schedules
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found or already deleted")
	}

	s.logger.WithField("schedule_id", id).Info("Schedule deleted")
	return nil
}

// RecordExecution records the result of a schedule execution
func (s *ScheduleStore) RecordExecution(ctx context.Context, exec *ScheduleExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}

	query := `
		INSERT INTO schedule_executions (
			id, schedule_id, executed_at, status, duration_ms,
			rows_returned, error_message, result_preview
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		exec.ID, exec.ScheduleID, exec.ExecutedAt.Unix(),
		exec.Status, exec.DurationMs, exec.RowsReturned,
		exec.ErrorMessage, exec.ResultPreview,
	)

	if err != nil {
		return fmt.Errorf("failed to record execution: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"execution_id": exec.ID,
		"schedule_id":  exec.ScheduleID,
		"status":       exec.Status,
		"duration_ms":  exec.DurationMs,
	}).Info("Execution recorded")

	return nil
}

// GetExecutionHistory returns execution history for a schedule
func (s *ScheduleStore) GetExecutionHistory(ctx context.Context, scheduleID string, limit int) ([]*ScheduleExecution, error) {
	query := `
		SELECT id, schedule_id, executed_at, status, duration_ms,
		       rows_returned, error_message, result_preview
		FROM schedule_executions
		WHERE schedule_id = ?
		ORDER BY executed_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, scheduleID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query executions: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var executions []*ScheduleExecution
	for rows.Next() {
		var exec ScheduleExecution
		var errorMessage, resultPreview sql.NullString
		var executedAt int64

		err := rows.Scan(
			&exec.ID, &exec.ScheduleID, &executedAt, &exec.Status,
			&exec.DurationMs, &exec.RowsReturned,
			&errorMessage, &resultPreview,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}

		exec.ExecutedAt = time.Unix(executedAt, 0)

		if errorMessage.Valid {
			exec.ErrorMessage = errorMessage.String
		}

		if resultPreview.Valid {
			exec.ResultPreview = resultPreview.String
		}

		executions = append(executions, &exec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating executions: %w", err)
	}

	return executions, nil
}

// GetExecutionStats returns statistics for a schedule
func (s *ScheduleStore) GetExecutionStats(ctx context.Context, scheduleID string) (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_executions,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful_executions,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_executions,
			AVG(duration_ms) as avg_duration_ms,
			MAX(executed_at) as last_execution
		FROM schedule_executions
		WHERE schedule_id = ?
	`

	var total, successful, failed int
	var avgDuration sql.NullFloat64
	var lastExecution sql.NullInt64

	err := s.db.QueryRowContext(ctx, query, scheduleID).Scan(
		&total, &successful, &failed, &avgDuration, &lastExecution,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get execution stats: %w", err)
	}

	stats := map[string]interface{}{
		"total_executions":      total,
		"successful_executions": successful,
		"failed_executions":     failed,
	}

	if avgDuration.Valid {
		stats["avg_duration_ms"] = avgDuration.Float64
	}

	if lastExecution.Valid {
		stats["last_execution"] = time.Unix(lastExecution.Int64, 0)
	}

	return stats, nil
}

// querySchedules is a helper method to query and scan multiple schedules
func (s *ScheduleStore) querySchedules(ctx context.Context, query string, args ...interface{}) ([]*QuerySchedule, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var schedules []*QuerySchedule
	for rows.Next() {
		var schedule QuerySchedule
		var parametersJSON sql.NullString
		var description, notificationEmail sql.NullString
		var orgID sql.NullString
		var lastRunAt, nextRunAt sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(
			&schedule.ID, &schedule.TemplateID, &schedule.Name, &description,
			&schedule.Frequency, &parametersJSON,
			&lastRunAt, &nextRunAt, &schedule.Status,
			&schedule.CreatedBy, &orgID,
			&notificationEmail, &schedule.ResultStorage,
			&createdAt, &updatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}

		// Convert timestamps
		schedule.CreatedAt = time.Unix(createdAt, 0)
		schedule.UpdatedAt = time.Unix(updatedAt, 0)

		if lastRunAt.Valid {
			lastRun := time.Unix(lastRunAt.Int64, 0)
			schedule.LastRunAt = &lastRun
		}

		if nextRunAt.Valid {
			nextRun := time.Unix(nextRunAt.Int64, 0)
			schedule.NextRunAt = &nextRun
		}

		// Convert nullable fields
		if description.Valid {
			schedule.Description = description.String
		}

		if orgID.Valid {
			schedule.OrganizationID = &orgID.String
		}

		if notificationEmail.Valid {
			schedule.NotificationEmail = notificationEmail.String
		}

		// Unmarshal parameters
		if parametersJSON.Valid && parametersJSON.String != "" && parametersJSON.String != "null" {
			if err := json.Unmarshal([]byte(parametersJSON.String), &schedule.Parameters); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal parameters")
			}
		}

		schedules = append(schedules, &schedule)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedules: %w", err)
	}

	return schedules, nil
}
