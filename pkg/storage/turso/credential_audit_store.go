package turso

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	// AuditLogBufferSize defines the channel buffer size for async logging
	AuditLogBufferSize = 1000
	// AuditLogBatchSize defines how many logs to batch insert at once
	AuditLogBatchSize = 100
	// AuditLogFlushInterval defines how often to flush pending logs
	AuditLogFlushInterval = 5 * time.Second
)

// CredentialAuditStore manages credential access audit logs
// Uses async logging via buffered channel to avoid blocking operations
type CredentialAuditStore struct {
	db      *sql.DB
	logger  *logrus.Logger
	logChan chan *CredentialAccessLog
	done    chan struct{}
}

// CredentialAccessLog represents a single audit log entry
type CredentialAccessLog struct {
	ID             string
	ConnectionID   string
	OrganizationID string
	UserID         string
	Action         string // "decrypt", "share", "unshare", "rotate", "view"
	Timestamp      time.Time
	IPAddress      string
	Success        bool
	ErrorMessage   string
}

// AuditLogFilters defines filters for querying audit logs
type AuditLogFilters struct {
	ConnectionID   string
	OrganizationID string
	UserID         string
	Action         string
	StartTime      *time.Time
	EndTime        *time.Time
	SuccessOnly    *bool
}

// NewCredentialAuditStore creates a new audit store with background logging
func NewCredentialAuditStore(db *sql.DB, logger *logrus.Logger) *CredentialAuditStore {
	return &CredentialAuditStore{
		db:      db,
		logger:  logger,
		logChan: make(chan *CredentialAccessLog, AuditLogBufferSize),
		done:    make(chan struct{}),
	}
}

// Start begins the background goroutine for async audit logging
func (s *CredentialAuditStore) Start() {
	go s.logWorker()
	s.logger.Info("Credential audit store started")
}

// logWorker is the background goroutine that drains the log channel
// and performs batch inserts to the database
func (s *CredentialAuditStore) logWorker() {
	ticker := time.NewTicker(AuditLogFlushInterval)
	defer ticker.Stop()

	batch := make([]*CredentialAccessLog, 0, AuditLogBatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}

		if err := s.batchInsert(batch); err != nil {
			s.logger.WithError(err).Error("Failed to insert audit logs")
		}

		// Clear batch
		batch = batch[:0]
	}

	for {
		select {
		case log := <-s.logChan:
			batch = append(batch, log)

			// Flush when batch is full
			if len(batch) >= AuditLogBatchSize {
				flush()
			}

		case <-ticker.C:
			// Periodic flush to ensure logs don't sit too long
			flush()

		case <-s.done:
			// Final flush on shutdown
			flush()

			// Drain any remaining logs
			for {
				select {
				case log := <-s.logChan:
					batch = append(batch, log)
					if len(batch) >= AuditLogBatchSize {
						flush()
					}
				default:
					flush()
					s.logger.Info("Credential audit store stopped")
					return
				}
			}
		}
	}
}

// batchInsert performs a batch insert of audit logs
func (s *CredentialAuditStore) batchInsert(logs []*CredentialAccessLog) error {
	if len(logs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			s.logger.WithError(err).Error("Failed to rollback transaction")
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO credential_audit_logs (
			id, connection_id, organization_id, user_id, action,
			timestamp, ip_address, success, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, log := range logs {
		if log.ID == "" {
			log.ID = uuid.New().String()
		}

		_, err := stmt.ExecContext(ctx,
			log.ID,
			log.ConnectionID,
			log.OrganizationID,
			log.UserID,
			log.Action,
			log.Timestamp.Unix(),
			log.IPAddress,
			log.Success,
			log.ErrorMessage,
		)

		if err != nil {
			return fmt.Errorf("failed to insert audit log: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.logger.WithField("count", len(logs)).Debug("Batch inserted audit logs")
	return nil
}

// LogAccess sends an audit log to the channel (non-blocking)
// If the channel is full, it drops the log and logs a warning
func (s *CredentialAuditStore) LogAccess(log *CredentialAccessLog) {
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	select {
	case s.logChan <- log:
		// Successfully queued
	default:
		// Channel full - log warning and drop
		s.logger.WithFields(logrus.Fields{
			"connection_id":   log.ConnectionID,
			"organization_id": log.OrganizationID,
			"user_id":         log.UserID,
			"action":          log.Action,
		}).Warn("Audit log channel full, dropping log entry")
	}
}

// GetLogs retrieves audit logs with filtering and pagination
func (s *CredentialAuditStore) GetLogs(ctx context.Context, filters AuditLogFilters, limit, offset int) ([]*CredentialAccessLog, error) {
	query := `
		SELECT id, connection_id, organization_id, user_id, action,
			timestamp, ip_address, success, error_message
		FROM credential_audit_logs
		WHERE 1=1
	`

	args := make([]interface{}, 0)

	if filters.ConnectionID != "" {
		query += " AND connection_id = ?"
		args = append(args, filters.ConnectionID)
	}

	if filters.OrganizationID != "" {
		query += " AND organization_id = ?"
		args = append(args, filters.OrganizationID)
	}

	if filters.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filters.UserID)
	}

	if filters.Action != "" {
		query += " AND action = ?"
		args = append(args, filters.Action)
	}

	if filters.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, filters.StartTime.Unix())
	}

	if filters.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, filters.EndTime.Unix())
	}

	if filters.SuccessOnly != nil {
		query += " AND success = ?"
		args = append(args, *filters.SuccessOnly)
	}

	query += " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var logs []*CredentialAccessLog

	for rows.Next() {
		var log CredentialAccessLog
		var timestamp int64
		var errorMsg sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.ConnectionID,
			&log.OrganizationID,
			&log.UserID,
			&log.Action,
			&timestamp,
			&log.IPAddress,
			&log.Success,
			&errorMsg,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan audit log row")
			continue
		}

		log.Timestamp = time.Unix(timestamp, 0)
		if errorMsg.Valid {
			log.ErrorMessage = errorMsg.String
		}

		logs = append(logs, &log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit log rows: %w", err)
	}

	return logs, nil
}

// GetLogCount returns the total count of logs matching the filters
func (s *CredentialAuditStore) GetLogCount(ctx context.Context, filters AuditLogFilters) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM credential_audit_logs
		WHERE 1=1
	`

	args := make([]interface{}, 0)

	if filters.ConnectionID != "" {
		query += " AND connection_id = ?"
		args = append(args, filters.ConnectionID)
	}

	if filters.OrganizationID != "" {
		query += " AND organization_id = ?"
		args = append(args, filters.OrganizationID)
	}

	if filters.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, filters.UserID)
	}

	if filters.Action != "" {
		query += " AND action = ?"
		args = append(args, filters.Action)
	}

	if filters.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, filters.StartTime.Unix())
	}

	if filters.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, filters.EndTime.Unix())
	}

	if filters.SuccessOnly != nil {
		query += " AND success = ?"
		args = append(args, *filters.SuccessOnly)
	}

	var count int64
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	return count, nil
}

// DeleteOldLogs removes audit logs older than the specified duration
// This should be called periodically to prevent unbounded growth
func (s *CredentialAuditStore) DeleteOldLogs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Unix()

	query := `DELETE FROM credential_audit_logs WHERE timestamp < ?`

	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old audit logs: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		s.logger.WithFields(logrus.Fields{
			"deleted_count": rowsAffected,
			"older_than":    olderThan.String(),
		}).Info("Deleted old audit logs")
	}

	return rowsAffected, nil
}

// Close stops the background worker and flushes pending logs
func (s *CredentialAuditStore) Close() error {
	close(s.done)
	// Give the worker time to finish
	time.Sleep(100 * time.Millisecond)
	return nil
}
