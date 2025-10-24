package backup

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Store handles persistence of backup metadata
type Store interface {
	CreateBackup(ctx context.Context, backup *DatabaseBackup) error
	GetBackup(ctx context.Context, backupID string) (*DatabaseBackup, error)
	ListBackups(ctx context.Context, limit int) ([]*DatabaseBackup, error)
	UpdateBackupProgress(ctx context.Context, backupID string, status string) error
	UpdateBackupComplete(ctx context.Context, backupID string, fileSize int64) error
	UpdateBackupFailed(ctx context.Context, backupID string, errorMessage string) error
	DeleteBackup(ctx context.Context, backupID string) error
	GetBackupStats(ctx context.Context) (*BackupStats, error)
	CleanupOldBackups(ctx context.Context, keepCount int) error
}

type store struct {
	db *sql.DB
}

// NewStore creates a new backup store
func NewStore(db *sql.DB) Store {
	return &store{db: db}
}

func (s *store) CreateBackup(ctx context.Context, backup *DatabaseBackup) error {
	if backup.ID == "" {
		backup.ID = uuid.New().String()
	}
	if backup.StartedAt.IsZero() {
		backup.StartedAt = time.Now()
	}

	tablesJSON, _ := json.Marshal(backup.TablesIncluded)

	query := `
		INSERT INTO database_backups (
			id, backup_type, status, file_path, file_size,
			tables_included, started_at, completed_at, error_message, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var completedAtUnix *int64
	if !backup.CompletedAt.IsZero() {
		unix := backup.CompletedAt.Unix()
		completedAtUnix = &unix
	}

	_, err := s.db.ExecContext(ctx, query,
		backup.ID,
		backup.BackupType,
		backup.Status,
		backup.FilePath,
		backup.FileSize,
		string(tablesJSON),
		backup.StartedAt.Unix(),
		completedAtUnix,
		backup.ErrorMessage,
		backup.CreatedBy,
	)

	return err
}

func (s *store) GetBackup(ctx context.Context, backupID string) (*DatabaseBackup, error) {
	query := `
		SELECT id, backup_type, status, file_path, file_size,
			tables_included, started_at, completed_at, error_message, created_by
		FROM database_backups
		WHERE id = ?
	`

	var backup DatabaseBackup
	var startedAtUnix int64
	var completedAtUnix sql.NullInt64
	var fileSize sql.NullInt64
	var tablesJSON, errorMsg, createdBy sql.NullString

	err := s.db.QueryRowContext(ctx, query, backupID).Scan(
		&backup.ID,
		&backup.BackupType,
		&backup.Status,
		&backup.FilePath,
		&fileSize,
		&tablesJSON,
		&startedAtUnix,
		&completedAtUnix,
		&errorMsg,
		&createdBy,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("backup not found")
		}
		return nil, err
	}

	backup.StartedAt = time.Unix(startedAtUnix, 0)
	if completedAtUnix.Valid {
		backup.CompletedAt = time.Unix(completedAtUnix.Int64, 0)
	}
	if fileSize.Valid {
		backup.FileSize = fileSize.Int64
	}
	if tablesJSON.Valid {
		json.Unmarshal([]byte(tablesJSON.String), &backup.TablesIncluded)
	}
	if errorMsg.Valid {
		backup.ErrorMessage = errorMsg.String
	}
	if createdBy.Valid {
		backup.CreatedBy = createdBy.String
	}

	return &backup, nil
}

func (s *store) ListBackups(ctx context.Context, limit int) ([]*DatabaseBackup, error) {
	query := `
		SELECT id, backup_type, status, file_path, file_size,
			tables_included, started_at, completed_at, error_message, created_by
		FROM database_backups
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var backups []*DatabaseBackup
	for rows.Next() {
		var backup DatabaseBackup
		var startedAtUnix int64
		var completedAtUnix sql.NullInt64
		var fileSize sql.NullInt64
		var tablesJSON, errorMsg, createdBy sql.NullString

		err := rows.Scan(
			&backup.ID,
			&backup.BackupType,
			&backup.Status,
			&backup.FilePath,
			&fileSize,
			&tablesJSON,
			&startedAtUnix,
			&completedAtUnix,
			&errorMsg,
			&createdBy,
		)
		if err != nil {
			return nil, err
		}

		backup.StartedAt = time.Unix(startedAtUnix, 0)
		if completedAtUnix.Valid {
			backup.CompletedAt = time.Unix(completedAtUnix.Int64, 0)
		}
		if fileSize.Valid {
			backup.FileSize = fileSize.Int64
		}
		if tablesJSON.Valid {
			json.Unmarshal([]byte(tablesJSON.String), &backup.TablesIncluded)
		}
		if errorMsg.Valid {
			backup.ErrorMessage = errorMsg.String
		}
		if createdBy.Valid {
			backup.CreatedBy = createdBy.String
		}

		backups = append(backups, &backup)
	}

	return backups, rows.Err()
}

func (s *store) UpdateBackupProgress(ctx context.Context, backupID string, status string) error {
	query := `UPDATE database_backups SET status = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, status, backupID)
	return err
}

func (s *store) UpdateBackupComplete(ctx context.Context, backupID string, fileSize int64) error {
	query := `
		UPDATE database_backups
		SET status = 'completed', file_size = ?, completed_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, fileSize, time.Now().Unix(), backupID)
	return err
}

func (s *store) UpdateBackupFailed(ctx context.Context, backupID string, errorMessage string) error {
	query := `
		UPDATE database_backups
		SET status = 'failed', error_message = ?, completed_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, errorMessage, time.Now().Unix(), backupID)
	return err
}

func (s *store) DeleteBackup(ctx context.Context, backupID string) error {
	query := `DELETE FROM database_backups WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, backupID)
	return err
}

func (s *store) GetBackupStats(ctx context.Context) (*BackupStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(file_size), 0) as total_size,
			MIN(started_at) as oldest,
			MAX(started_at) as latest,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as successful,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed
		FROM database_backups
	`

	var stats BackupStats
	var oldestUnix, latestUnix sql.NullInt64

	err := s.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalBackups,
		&stats.TotalSize,
		&oldestUnix,
		&latestUnix,
		&stats.SuccessfulBackups,
		&stats.FailedBackups,
	)

	if err != nil {
		return nil, err
	}

	if oldestUnix.Valid {
		stats.OldestBackup = time.Unix(oldestUnix.Int64, 0)
	}
	if latestUnix.Valid {
		stats.LatestBackup = time.Unix(latestUnix.Int64, 0)
	}

	return &stats, nil
}

func (s *store) CleanupOldBackups(ctx context.Context, keepCount int) error {
	// Delete old backups, keeping only the most recent 'keepCount' backups
	query := `
		DELETE FROM database_backups
		WHERE id NOT IN (
			SELECT id FROM database_backups
			ORDER BY started_at DESC
			LIMIT ?
		)
	`

	result, err := s.db.ExecContext(ctx, query, keepCount)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		// Could log this
		_ = rowsAffected
	}

	return nil
}
