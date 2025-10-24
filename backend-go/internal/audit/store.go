package audit

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store handles persistence of detailed audit logs
type Store interface {
	CreateDetailedLog(ctx context.Context, log *AuditLogDetailed) error
	CreateDetailedLogs(ctx context.Context, logs []*AuditLogDetailed) error
	GetChangeHistory(ctx context.Context, tableName, recordID string) (*ChangeHistory, error)
	GetFieldHistory(ctx context.Context, tableName, recordID, fieldName string) ([]FieldChange, error)
	GetPIIAccessLogs(ctx context.Context, userID string, since time.Time) ([]PIIAccessLog, error)
	LogPIIAccess(ctx context.Context, log *PIIAccessLog) error
}

type store struct {
	db *sql.DB
}

// NewStore creates a new audit store
func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreateDetailedLog(ctx context.Context, log *AuditLogDetailed) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO audit_logs_detailed (
			id, audit_log_id, table_name, record_id, field_name,
			old_value, new_value, field_type, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		log.ID,
		log.AuditLogID,
		log.TableName,
		log.RecordID,
		log.FieldName,
		log.OldValue,
		log.NewValue,
		log.FieldType,
		log.CreatedAt.Unix(),
	)

	return err
}

func (s *store) CreateDetailedLogs(ctx context.Context, logs []*AuditLogDetailed) error {
	if len(logs) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO audit_logs_detailed (
			id, audit_log_id, table_name, record_id, field_name,
			old_value, new_value, field_type, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, log := range logs {
		if log.ID == "" {
			log.ID = uuid.New().String()
		}
		if log.CreatedAt.IsZero() {
			log.CreatedAt = time.Now()
		}

		_, err = stmt.ExecContext(ctx,
			log.ID,
			log.AuditLogID,
			log.TableName,
			log.RecordID,
			log.FieldName,
			log.OldValue,
			log.NewValue,
			log.FieldType,
			log.CreatedAt.Unix(),
		)
		if err != nil {
			return fmt.Errorf("insert log: %w", err)
		}
	}

	return tx.Commit()
}

func (s *store) GetChangeHistory(ctx context.Context, tableName, recordID string) (*ChangeHistory, error) {
	query := `
		SELECT
			ald.field_name,
			ald.old_value,
			ald.new_value,
			ald.field_type,
			ald.created_at,
			al.user_id,
			al.id as audit_id
		FROM audit_logs_detailed ald
		INNER JOIN audit_logs al ON ald.audit_log_id = al.id
		WHERE ald.table_name = ? AND ald.record_id = ?
		ORDER BY ald.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tableName, recordID)
	if err != nil {
		return nil, fmt.Errorf("query change history: %w", err)
	}
	defer rows.Close()

	history := &ChangeHistory{
		TableName: tableName,
		RecordID:  recordID,
		Fields:    make(map[string][]FieldChange),
	}

	for rows.Next() {
		var fc FieldChange
		var createdAtUnix int64
		var userID string

		err := rows.Scan(
			&fc.FieldName,
			&fc.OldValue,
			&fc.NewValue,
			&fc.FieldType,
			&createdAtUnix,
			&userID,
			&fc.AuditID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		fc.ChangedAt = time.Unix(createdAtUnix, 0)
		fc.ChangedBy = userID

		history.Fields[fc.FieldName] = append(history.Fields[fc.FieldName], fc)
	}

	return history, rows.Err()
}

func (s *store) GetFieldHistory(ctx context.Context, tableName, recordID, fieldName string) ([]FieldChange, error) {
	query := `
		SELECT
			ald.old_value,
			ald.new_value,
			ald.field_type,
			ald.created_at,
			al.user_id,
			al.id as audit_id
		FROM audit_logs_detailed ald
		INNER JOIN audit_logs al ON ald.audit_log_id = al.id
		WHERE ald.table_name = ? AND ald.record_id = ? AND ald.field_name = ?
		ORDER BY ald.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tableName, recordID, fieldName)
	if err != nil {
		return nil, fmt.Errorf("query field history: %w", err)
	}
	defer rows.Close()

	var changes []FieldChange
	for rows.Next() {
		var fc FieldChange
		var createdAtUnix int64
		var userID string

		err := rows.Scan(
			&fc.OldValue,
			&fc.NewValue,
			&fc.FieldType,
			&createdAtUnix,
			&userID,
			&fc.AuditID,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		fc.FieldName = fieldName
		fc.ChangedAt = time.Unix(createdAtUnix, 0)
		fc.ChangedBy = userID

		changes = append(changes, fc)
	}

	return changes, rows.Err()
}

func (s *store) GetPIIAccessLogs(ctx context.Context, userID string, since time.Time) ([]PIIAccessLog, error) {
	// This would query a separate PII access log table
	// For now, we'll return PII-type changes from detailed logs
	query := `
		SELECT
			al.user_id,
			ald.table_name,
			ald.field_name,
			ald.record_id,
			CASE
				WHEN ald.new_value IS NOT NULL THEN 'write'
				ELSE 'read'
			END as access_type,
			ald.created_at
		FROM audit_logs_detailed ald
		INNER JOIN audit_logs al ON ald.audit_log_id = al.id
		WHERE ald.field_type = 'pii'
		AND al.user_id = ?
		AND ald.created_at >= ?
		ORDER BY ald.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID, since.Unix())
	if err != nil {
		return nil, fmt.Errorf("query pii access: %w", err)
	}
	defer rows.Close()

	var logs []PIIAccessLog
	for rows.Next() {
		var log PIIAccessLog
		var accessedAtUnix int64

		err := rows.Scan(
			&log.UserID,
			&log.TableName,
			&log.FieldName,
			&log.RecordID,
			&log.AccessType,
			&accessedAtUnix,
		)
		if err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		log.AccessedAt = time.Unix(accessedAtUnix, 0)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

func (s *store) LogPIIAccess(ctx context.Context, log *PIIAccessLog) error {
	// Create a detailed audit log entry for PII access
	detailedLog := &AuditLogDetailed{
		ID:        uuid.New().String(),
		TableName: log.TableName,
		RecordID:  log.RecordID,
		FieldName: log.FieldName,
		FieldType: "pii",
		CreatedAt: log.AccessedAt,
	}

	// Note: This would ideally link to a parent audit_log entry
	// For now, we'll create a standalone entry
	return s.CreateDetailedLog(ctx, detailedLog)
}
