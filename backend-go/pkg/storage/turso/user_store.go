package turso

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/jbeck018/howlerops/backend-go/internal/auth"
)

// TursoUserStore implements auth.UserStore interface for Turso
type TursoUserStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewTursoUserStore creates a new Turso user store
func NewTursoUserStore(db *sql.DB, logger *logrus.Logger) *TursoUserStore {
	return &TursoUserStore{
		db:     db,
		logger: logger,
	}
}

// GetUser retrieves a user by ID
func (s *TursoUserStore) GetUser(ctx context.Context, id string) (*auth.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login, metadata
		FROM users
		WHERE id = ?
	`

	var user auth.User
	var metadataJSON sql.NullString
	var lastLogin sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.Active,
		&createdAt,
		&updatedAt,
		&lastLogin,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Convert timestamps
	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if lastLogin.Valid {
		t := time.Unix(lastLogin.Int64, 0)
		user.LastLogin = &t
	}

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &user.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal user metadata")
			user.Metadata = make(map[string]string)
		}
	} else {
		user.Metadata = make(map[string]string)
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by username
func (s *TursoUserStore) GetUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login, metadata
		FROM users
		WHERE username = ?
	`

	var user auth.User
	var metadataJSON sql.NullString
	var lastLogin sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.Active,
		&createdAt,
		&updatedAt,
		&lastLogin,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Convert timestamps
	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if lastLogin.Valid {
		t := time.Unix(lastLogin.Int64, 0)
		user.LastLogin = &t
	}

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &user.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal user metadata")
			user.Metadata = make(map[string]string)
		}
	} else {
		user.Metadata = make(map[string]string)
	}

	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (s *TursoUserStore) GetUserByEmail(ctx context.Context, email string) (*auth.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login, metadata
		FROM users
		WHERE email = ?
	`

	var user auth.User
	var metadataJSON sql.NullString
	var lastLogin sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.Role,
		&user.Active,
		&createdAt,
		&updatedAt,
		&lastLogin,
		&metadataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	// Convert timestamps
	user.CreatedAt = time.Unix(createdAt, 0)
	user.UpdatedAt = time.Unix(updatedAt, 0)
	if lastLogin.Valid {
		t := time.Unix(lastLogin.Int64, 0)
		user.LastLogin = &t
	}

	// Unmarshal metadata
	if metadataJSON.Valid && metadataJSON.String != "" {
		if err := json.Unmarshal([]byte(metadataJSON.String), &user.Metadata); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal user metadata")
			user.Metadata = make(map[string]string)
		}
	} else {
		user.Metadata = make(map[string]string)
	}

	return &user, nil
}

// CreateUser creates a new user
func (s *TursoUserStore) CreateUser(ctx context.Context, user *auth.User) error {
	// Check if user already exists
	existingUser, err := s.GetUserByUsername(ctx, user.Username)
	if err == nil && existingUser != nil {
		return fmt.Errorf("username already taken")
	}

	existingUser, err = s.GetUserByEmail(ctx, user.Email)
	if err == nil && existingUser != nil {
		return fmt.Errorf("email already taken")
	}

	// Marshal metadata
	var metadataJSON []byte
	if len(user.Metadata) > 0 {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Prepare last login
	var lastLogin sql.NullInt64
	if user.LastLogin != nil {
		lastLogin.Valid = true
		lastLogin.Int64 = user.LastLogin.Unix()
	}

	query := `
		INSERT INTO users (id, username, email, password_hash, role, active, created_at, updated_at, last_login, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
		user.ID,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.Active,
		user.CreatedAt.Unix(),
		user.UpdatedAt.Unix(),
		lastLogin,
		string(metadataJSON),
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("User created successfully")

	return nil
}

// UpdateUser updates an existing user
func (s *TursoUserStore) UpdateUser(ctx context.Context, user *auth.User) error {
	// Marshal metadata
	var metadataJSON []byte
	var err error
	if len(user.Metadata) > 0 {
		metadataJSON, err = json.Marshal(user.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	// Prepare last login
	var lastLogin sql.NullInt64
	if user.LastLogin != nil {
		lastLogin.Valid = true
		lastLogin.Int64 = user.LastLogin.Unix()
	}

	query := `
		UPDATE users
		SET username = ?, email = ?, password_hash = ?, role = ?, active = ?,
		    updated_at = ?, last_login = ?, metadata = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
		user.Username,
		user.Email,
		user.Password,
		user.Role,
		user.Active,
		user.UpdatedAt.Unix(),
		lastLogin,
		string(metadataJSON),
		user.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	s.logger.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Debug("User updated successfully")

	return nil
}

// DeleteUser deletes a user
func (s *TursoUserStore) DeleteUser(ctx context.Context, id string) error {
	query := `DELETE FROM users WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	s.logger.WithField("user_id", id).Info("User deleted successfully")

	return nil
}

// ListUsers returns a list of users with pagination
func (s *TursoUserStore) ListUsers(ctx context.Context, limit, offset int) ([]*auth.User, error) {
	query := `
		SELECT id, username, email, password_hash, role, active, created_at, updated_at, last_login, metadata
		FROM users
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var users []*auth.User

	for rows.Next() {
		var user auth.User
		var metadataJSON sql.NullString
		var lastLogin sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Password,
			&user.Role,
			&user.Active,
			&createdAt,
			&updatedAt,
			&lastLogin,
			&metadataJSON,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan user row")
			continue
		}

		// Convert timestamps
		user.CreatedAt = time.Unix(createdAt, 0)
		user.UpdatedAt = time.Unix(updatedAt, 0)
		if lastLogin.Valid {
			t := time.Unix(lastLogin.Int64, 0)
			user.LastLogin = &t
		}

		// Unmarshal metadata
		if metadataJSON.Valid && metadataJSON.String != "" {
			if err := json.Unmarshal([]byte(metadataJSON.String), &user.Metadata); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal user metadata")
				user.Metadata = make(map[string]string)
			}
		} else {
			user.Metadata = make(map[string]string)
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, nil
}
