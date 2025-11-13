package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// Service manages database backups
type Service struct {
	db         *sql.DB
	store      Store
	backupPath string
	logger     *logrus.Logger
}

// NewService creates a new backup service
func NewService(db *sql.DB, store Store, backupPath string, logger *logrus.Logger) *Service {
	return &Service{
		db:         db,
		store:      store,
		backupPath: backupPath,
		logger:     logger,
	}
}

// CreateBackup initiates a database backup
func (s *Service) CreateBackup(ctx context.Context, opts *BackupOptions) (*DatabaseBackup, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(s.backupPath, 0750); err != nil {
		return nil, fmt.Errorf("create backup directory: %w", err)
	}

	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("backup_%s_%s.db", opts.BackupType, timestamp)
	filePath := filepath.Join(s.backupPath, filename)

	backup := &DatabaseBackup{
		BackupType:     opts.BackupType,
		Status:         "in_progress",
		FilePath:       filePath,
		TablesIncluded: opts.IncludeTables,
		StartedAt:      time.Now(),
	}

	err := s.store.CreateBackup(ctx, backup)
	if err != nil {
		return nil, fmt.Errorf("create backup record: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"backup_id":   backup.ID,
		"backup_type": opts.BackupType,
		"file_path":   filePath,
	}).Info("Starting database backup")

	// Perform backup asynchronously
	go s.performBackup(backup, opts)

	return backup, nil
}

// performBackup executes the backup operation
func (s *Service) performBackup(backup *DatabaseBackup, opts *BackupOptions) {
	ctx := context.Background()

	logger := s.logger.WithFields(logrus.Fields{
		"backup_id": backup.ID,
		"file_path": backup.FilePath,
	})

	// For SQLite, use VACUUM INTO for backup
	// For Turso/libSQL, this creates a consistent snapshot
	// #nosec G201 - SQLite VACUUM with validated path, not user input
	query := fmt.Sprintf("VACUUM INTO '%s'", backup.FilePath)

	logger.Info("Executing backup command")

	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		logger.WithError(err).Error("Backup failed")
		if updateErr := s.store.UpdateBackupFailed(ctx, backup.ID, err.Error()); updateErr != nil {
			logger.WithError(updateErr).Error("Failed to update backup status")
		}
		return
	}

	// Get file size
	info, err := os.Stat(backup.FilePath)
	if err != nil {
		logger.WithError(err).Error("Failed to stat backup file")
		if updateErr := s.store.UpdateBackupFailed(ctx, backup.ID, "Failed to get file size"); updateErr != nil {
			logger.WithError(updateErr).Error("Failed to update backup status")
		}
		return
	}

	fileSize := info.Size()

	logger.WithField("file_size_mb", fileSize/1024/1024).Info("Backup completed successfully")

	// Mark complete
	if err := s.store.UpdateBackupComplete(ctx, backup.ID, fileSize); err != nil {
		logger.WithError(err).Error("Failed to update backup status")
	}

	// Cleanup old backups if max_backups is set
	if opts.MaxBackups > 0 {
		if err := s.store.CleanupOldBackups(ctx, opts.MaxBackups); err != nil {
			logger.WithError(err).Error("Failed to cleanup old backups")
		}
	}
}

// GetBackup retrieves a backup by ID
func (s *Service) GetBackup(ctx context.Context, backupID string) (*DatabaseBackup, error) {
	return s.store.GetBackup(ctx, backupID)
}

// ListBackups lists recent backups
func (s *Service) ListBackups(ctx context.Context, limit int) ([]*DatabaseBackup, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListBackups(ctx, limit)
}

// DeleteBackup deletes a backup
func (s *Service) DeleteBackup(ctx context.Context, backupID string) error {
	backup, err := s.store.GetBackup(ctx, backupID)
	if err != nil {
		return err
	}

	// Delete physical file
	if err := os.Remove(backup.FilePath); err != nil && !os.IsNotExist(err) {
		s.logger.WithError(err).Warn("Failed to delete backup file")
	}

	// Delete database record
	return s.store.DeleteBackup(ctx, backupID)
}

// RestoreBackup restores from a backup
func (s *Service) RestoreBackup(ctx context.Context, opts *RestoreOptions) error {
	backup, err := s.store.GetBackup(ctx, opts.BackupID)
	if err != nil {
		return fmt.Errorf("get backup: %w", err)
	}

	// Verify backup file exists
	if _, err := os.Stat(backup.FilePath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backup.FilePath)
	}

	if backup.Status != "completed" {
		return fmt.Errorf("cannot restore from incomplete backup (status: %s)", backup.Status)
	}

	s.logger.WithFields(logrus.Fields{
		"backup_id": backup.ID,
		"file_path": backup.FilePath,
		"dry_run":   opts.DryRun,
	}).Warn("Restore requested")

	if opts.DryRun {
		s.logger.Info("Dry run - no actual restore performed")
		return nil
	}

	// Actual restore process:
	// For SQLite/Turso:
	// 1. Close all connections
	// 2. Copy backup file over current database file
	// 3. Reopen connections
	// This requires coordination with the application and typically a restart

	return fmt.Errorf("restore requires application restart - manual process required")
}

// GetBackupStats returns backup statistics
func (s *Service) GetBackupStats(ctx context.Context) (*BackupStats, error) {
	return s.store.GetBackupStats(ctx)
}

// StartScheduler starts the automatic backup scheduler
// Runs daily at 3 AM local time
func (s *Service) StartScheduler(ctx context.Context, opts *BackupOptions) {
	s.logger.Info("Starting backup scheduler")

	// Calculate next 3 AM
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), 3, 0, 0, 0, now.Location())
	if next.Before(now) {
		next = next.Add(24 * time.Hour)
	}

	// Wait until 3 AM
	duration := next.Sub(now)
	s.logger.WithField("next_run", next).Info("Scheduler will run at next scheduled time")

	timer := time.NewTimer(duration)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Backup scheduler stopped")
			return
		case <-timer.C:
			s.logger.Info("Running scheduled backup")

			_, err := s.CreateBackup(ctx, opts)
			if err != nil {
				s.logger.WithError(err).Error("Scheduled backup failed")
			}

			// Schedule next run in 24 hours
			timer.Reset(24 * time.Hour)
		}
	}
}

// VerifyBackup verifies the integrity of a backup
func (s *Service) VerifyBackup(ctx context.Context, backupID string) error {
	backup, err := s.store.GetBackup(ctx, backupID)
	if err != nil {
		return fmt.Errorf("get backup: %w", err)
	}

	// Check file exists
	info, err := os.Stat(backup.FilePath)
	if err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Check file size matches
	if backup.FileSize > 0 && info.Size() != backup.FileSize {
		return fmt.Errorf("file size mismatch: expected %d, got %d", backup.FileSize, info.Size())
	}

	// Try to open the database to verify it's valid
	testDB, err := sql.Open("sqlite3", backup.FilePath+"?mode=ro")
	if err != nil {
		return fmt.Errorf("failed to open backup database: %w", err)
	}
	defer func() { _ = testDB.Close() }() // Best-effort close

	// Run a simple query to verify integrity
	var count int
	err = testDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table'").Scan(&count)
	if err != nil {
		return fmt.Errorf("backup verification failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"backup_id": backupID,
		"tables":    count,
	}).Info("Backup verification successful")

	return nil
}
