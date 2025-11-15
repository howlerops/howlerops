package turso

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/jbeck018/howlerops/backend-go/internal/auth"
)

// TursoLoginAttemptStore implements auth.LoginAttemptStore interface for Turso
type TursoLoginAttemptStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTursoLoginAttemptStore creates a new Turso login attempt store
func NewTursoLoginAttemptStore(db *sql.DB, logger *logrus.Logger) *TursoLoginAttemptStore {
	return &TursoLoginAttemptStore{
		db:     db,
		logger: logger,
	}
}

// RecordAttempt records a login attempt
func (s *TursoLoginAttemptStore) RecordAttempt(ctx context.Context, attempt *auth.LoginAttempt) error {
	query := `
		INSERT INTO login_attempts (id, ip, username, timestamp, success)
		VALUES (?, ?, ?, ?, ?)
	`

	// Generate ID if not provided
	id := uuid.New().String()

	_, err := s.db.ExecContext(ctx, query,
		id,
		attempt.IP,
		attempt.Username,
		attempt.Timestamp.Unix(),
		attempt.Success,
	)

	if err != nil {
		return fmt.Errorf("failed to record login attempt: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"ip":       attempt.IP,
		"username": attempt.Username,
		"success":  attempt.Success,
	}).Debug("Login attempt recorded")

	return nil
}

// GetAttempts retrieves login attempts for an IP/username since a given time
func (s *TursoLoginAttemptStore) GetAttempts(ctx context.Context, ip, username string, since time.Time) ([]*auth.LoginAttempt, error) {
	// Build dynamic query based on filters
	query := `
		SELECT id, ip, username, timestamp, success
		FROM login_attempts
		WHERE timestamp >= ?
	`
	args := []interface{}{since.Unix()}

	if ip != "" {
		query += " AND ip = ?"
		args = append(args, ip)
	}

	if username != "" {
		query += " AND username = ?"
		args = append(args, username)
	}

	query += " ORDER BY timestamp DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query login attempts: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var attempts []*auth.LoginAttempt

	for rows.Next() {
		var attempt auth.LoginAttempt
		var id string
		var timestamp int64

		err := rows.Scan(
			&id,
			&attempt.IP,
			&attempt.Username,
			&timestamp,
			&attempt.Success,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan login attempt row")
			continue
		}

		// Convert timestamp
		attempt.Timestamp = time.Unix(timestamp, 0)

		attempts = append(attempts, &attempt)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating login attempt rows: %w", err)
	}

	return attempts, nil
}

// CleanupOldAttempts removes login attempts older than the specified time
func (s *TursoLoginAttemptStore) CleanupOldAttempts(ctx context.Context, before time.Time) error {
	query := `DELETE FROM login_attempts WHERE timestamp < ?`

	result, err := s.db.ExecContext(ctx, query, before.Unix())
	if err != nil {
		return fmt.Errorf("failed to cleanup old login attempts: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		s.logger.WithField("attempts_cleaned", rowsAffected).Info("Old login attempts cleaned up")
	}

	return nil
}

// GetFailedAttemptsCount returns the count of failed login attempts for an IP/username since a given time
func (s *TursoLoginAttemptStore) GetFailedAttemptsCount(ctx context.Context, ip, username string, since time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM login_attempts
		WHERE timestamp >= ? AND success = 0
	`
	args := []interface{}{since.Unix()}

	if ip != "" {
		query += " AND ip = ?"
		args = append(args, ip)
	}

	if username != "" {
		query += " AND username = ?"
		args = append(args, username)
	}

	var count int
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count failed attempts: %w", err)
	}

	return count, nil
}
