package turso

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/auth"
)

// TursoSessionStore implements auth.SessionStore interface for Turso
type TursoSessionStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTursoSessionStore creates a new Turso session store
func NewTursoSessionStore(db *sql.DB, logger *logrus.Logger) *TursoSessionStore {
	return &TursoSessionStore{
		db:     db,
		logger: logger,
	}
}

// CreateSession creates a new session
func (s *TursoSessionStore) CreateSession(ctx context.Context, session *auth.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, refresh_token, expires_at, created_at, last_access, ip_address, user_agent, active)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		session.ID,
		session.UserID,
		session.Token,
		session.RefreshToken,
		session.ExpiresAt.Unix(),
		session.CreatedAt.Unix(),
		session.LastAccess.Unix(),
		session.IPAddress,
		session.UserAgent,
		session.Active,
	)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"user_id":    session.UserID,
		"ip_address": session.IPAddress,
	}).Debug("Session created successfully")

	return nil
}

// GetSession retrieves a session by token
func (s *TursoSessionStore) GetSession(ctx context.Context, token string) (*auth.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, last_access, ip_address, user_agent, active
		FROM sessions
		WHERE token = ?
	`

	var session auth.Session
	var expiresAt, createdAt, lastAccess int64

	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.RefreshToken,
		&expiresAt,
		&createdAt,
		&lastAccess,
		&session.IPAddress,
		&session.UserAgent,
		&session.Active,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Convert timestamps
	session.ExpiresAt = time.Unix(expiresAt, 0)
	session.CreatedAt = time.Unix(createdAt, 0)
	session.LastAccess = time.Unix(lastAccess, 0)

	return &session, nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func (s *TursoSessionStore) GetSessionByRefreshToken(ctx context.Context, refreshToken string) (*auth.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, last_access, ip_address, user_agent, active
		FROM sessions
		WHERE refresh_token = ?
	`

	var session auth.Session
	var expiresAt, createdAt, lastAccess int64

	err := s.db.QueryRowContext(ctx, query, refreshToken).Scan(
		&session.ID,
		&session.UserID,
		&session.Token,
		&session.RefreshToken,
		&expiresAt,
		&createdAt,
		&lastAccess,
		&session.IPAddress,
		&session.UserAgent,
		&session.Active,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("session not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query session: %w", err)
	}

	// Convert timestamps
	session.ExpiresAt = time.Unix(expiresAt, 0)
	session.CreatedAt = time.Unix(createdAt, 0)
	session.LastAccess = time.Unix(lastAccess, 0)

	return &session, nil
}

// UpdateSession updates an existing session
func (s *TursoSessionStore) UpdateSession(ctx context.Context, session *auth.Session) error {
	query := `
		UPDATE sessions
		SET user_id = ?, token = ?, refresh_token = ?, expires_at = ?, last_access = ?, ip_address = ?, user_agent = ?, active = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		session.UserID,
		session.Token,
		session.RefreshToken,
		session.ExpiresAt.Unix(),
		session.LastAccess.Unix(),
		session.IPAddress,
		session.UserAgent,
		session.Active,
		session.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	s.logger.WithFields(logrus.Fields{
		"session_id": session.ID,
		"user_id":    session.UserID,
	}).Debug("Session updated successfully")

	return nil
}

// DeleteSession deletes a session by token
func (s *TursoSessionStore) DeleteSession(ctx context.Context, token string) error {
	query := `DELETE FROM sessions WHERE token = ?`

	result, err := s.db.ExecContext(ctx, query, token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("session not found")
	}

	s.logger.WithField("token", token).Debug("Session deleted successfully")

	return nil
}

// DeleteUserSessions deletes all sessions for a user
func (s *TursoSessionStore) DeleteUserSessions(ctx context.Context, userID string) error {
	query := `DELETE FROM sessions WHERE user_id = ?`

	result, err := s.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":          userID,
		"sessions_deleted": rowsAffected,
	}).Debug("User sessions deleted successfully")

	return nil
}

// GetUserSessions returns all sessions for a user
func (s *TursoSessionStore) GetUserSessions(ctx context.Context, userID string) ([]*auth.Session, error) {
	query := `
		SELECT id, user_id, token, refresh_token, expires_at, created_at, last_access, ip_address, user_agent, active
		FROM sessions
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*auth.Session

	for rows.Next() {
		var session auth.Session
		var expiresAt, createdAt, lastAccess int64

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.Token,
			&session.RefreshToken,
			&expiresAt,
			&createdAt,
			&lastAccess,
			&session.IPAddress,
			&session.UserAgent,
			&session.Active,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan session row")
			continue
		}

		// Convert timestamps
		session.ExpiresAt = time.Unix(expiresAt, 0)
		session.CreatedAt = time.Unix(createdAt, 0)
		session.LastAccess = time.Unix(lastAccess, 0)

		sessions = append(sessions, &session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating session rows: %w", err)
	}

	return sessions, nil
}

// CleanupExpiredSessions removes expired sessions
func (s *TursoSessionStore) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now().Unix()

	query := `DELETE FROM sessions WHERE expires_at < ? OR active = 0`

	result, err := s.db.ExecContext(ctx, query, now)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected > 0 {
		s.logger.WithField("sessions_cleaned", rowsAffected).Info("Expired sessions cleaned up")
	}

	return nil
}
