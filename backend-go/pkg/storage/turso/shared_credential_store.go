package turso

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SharedCredentialStore manages shared connection credentials encrypted with OEK
type SharedCredentialStore struct {
	db     *sql.DB
	logger *logrus.Logger
	// Prepared statements for performance
	stmtStore  *sql.Stmt
	stmtGet    *sql.Stmt
	stmtDelete *sql.Stmt
}

// SharedCredential represents a connection credential shared across an organization
type SharedCredential struct {
	ID                string
	ConnectionID      string
	OrganizationID    string
	EncryptedPassword string
	PasswordIV        string
	PasswordAuthTag   string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         string
}

// NewSharedCredentialStore creates a new shared credential store with prepared statements
func NewSharedCredentialStore(db *sql.DB, logger *logrus.Logger) (*SharedCredentialStore, error) {
	store := &SharedCredentialStore{
		db:     db,
		logger: logger,
	}

	var err error

	// Prepare store statement
	store.stmtStore, err = db.Prepare(`
		INSERT INTO shared_credentials (
			id, connection_id, organization_id, encrypted_password,
			password_iv, password_auth_tag, created_at, updated_at, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(connection_id, organization_id) DO UPDATE SET
			encrypted_password = excluded.encrypted_password,
			password_iv = excluded.password_iv,
			password_auth_tag = excluded.password_auth_tag,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare store statement: %w", err)
	}

	// Prepare get statement
	store.stmtGet, err = db.Prepare(`
		SELECT id, connection_id, organization_id, encrypted_password,
			password_iv, password_auth_tag, created_at, updated_at, created_by
		FROM shared_credentials
		WHERE connection_id = ? AND organization_id = ?
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get statement: %w", err)
	}

	// Prepare delete statement
	store.stmtDelete, err = db.Prepare(`
		DELETE FROM shared_credentials
		WHERE connection_id = ? AND organization_id = ?
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare delete statement: %w", err)
	}

	return store, nil
}

// Close closes all prepared statements
func (s *SharedCredentialStore) Close() error {
	var errs []error

	if s.stmtStore != nil {
		if err := s.stmtStore.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close store statement: %w", err))
		}
	}

	if s.stmtGet != nil {
		if err := s.stmtGet.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close get statement: %w", err))
		}
	}

	if s.stmtDelete != nil {
		if err := s.stmtDelete.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close delete statement: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing prepared statements: %v", errs)
	}

	return nil
}

// Store saves or updates a shared credential
// The password is encrypted with the organization's OEK before storage
func (s *SharedCredentialStore) Store(ctx context.Context, cred *SharedCredential) error {
	if cred.ID == "" {
		cred.ID = uuid.New().String()
	}

	now := time.Now().Unix()
	if cred.CreatedAt.IsZero() {
		cred.CreatedAt = time.Unix(now, 0)
	}
	cred.UpdatedAt = time.Unix(now, 0)

	_, err := s.stmtStore.ExecContext(ctx,
		cred.ID,
		cred.ConnectionID,
		cred.OrganizationID,
		cred.EncryptedPassword,
		cred.PasswordIV,
		cred.PasswordAuthTag,
		cred.CreatedAt.Unix(),
		cred.UpdatedAt.Unix(),
		cred.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to store shared credential: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id":   cred.ConnectionID,
		"organization_id": cred.OrganizationID,
		"created_by":      cred.CreatedBy,
	}).Info("Shared credential stored successfully")

	return nil
}

// Get retrieves a shared credential for a connection in an organization
func (s *SharedCredentialStore) Get(ctx context.Context, connID, orgID string) (*SharedCredential, error) {
	var cred SharedCredential
	var createdAt, updatedAt int64

	err := s.stmtGet.QueryRowContext(ctx, connID, orgID).Scan(
		&cred.ID,
		&cred.ConnectionID,
		&cred.OrganizationID,
		&cred.EncryptedPassword,
		&cred.PasswordIV,
		&cred.PasswordAuthTag,
		&createdAt,
		&updatedAt,
		&cred.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("shared credential not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve shared credential: %w", err)
	}

	cred.CreatedAt = time.Unix(createdAt, 0)
	cred.UpdatedAt = time.Unix(updatedAt, 0)

	return &cred, nil
}

// Delete removes a shared credential
func (s *SharedCredentialStore) Delete(ctx context.Context, connID, orgID string) error {
	result, err := s.stmtDelete.ExecContext(ctx, connID, orgID)
	if err != nil {
		return fmt.Errorf("failed to delete shared credential: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("shared credential not found")
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id":   connID,
		"organization_id": orgID,
	}).Info("Shared credential deleted successfully")

	return nil
}

// GetAllForOrganization retrieves all shared credentials for an organization
func (s *SharedCredentialStore) GetAllForOrganization(ctx context.Context, orgID string) ([]*SharedCredential, error) {
	query := `
		SELECT id, connection_id, organization_id, encrypted_password,
			password_iv, password_auth_tag, created_at, updated_at, created_by
		FROM shared_credentials
		WHERE organization_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shared credentials: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var credentials []*SharedCredential

	for rows.Next() {
		var cred SharedCredential
		var createdAt, updatedAt int64

		err := rows.Scan(
			&cred.ID,
			&cred.ConnectionID,
			&cred.OrganizationID,
			&cred.EncryptedPassword,
			&cred.PasswordIV,
			&cred.PasswordAuthTag,
			&createdAt,
			&updatedAt,
			&cred.CreatedBy,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan shared credential row")
			continue
		}

		cred.CreatedAt = time.Unix(createdAt, 0)
		cred.UpdatedAt = time.Unix(updatedAt, 0)

		credentials = append(credentials, &cred)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating credential rows: %w", err)
	}

	return credentials, nil
}

// Update updates an existing shared credential
func (s *SharedCredentialStore) Update(ctx context.Context, cred *SharedCredential) error {
	query := `
		UPDATE shared_credentials
		SET encrypted_password = ?, password_iv = ?, password_auth_tag = ?,
			updated_at = ?
		WHERE connection_id = ? AND organization_id = ?
	`

	now := time.Now().Unix()
	cred.UpdatedAt = time.Unix(now, 0)

	result, err := s.db.ExecContext(ctx, query,
		cred.EncryptedPassword,
		cred.PasswordIV,
		cred.PasswordAuthTag,
		now,
		cred.ConnectionID,
		cred.OrganizationID,
	)

	if err != nil {
		return fmt.Errorf("failed to update shared credential: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("shared credential not found")
	}

	s.logger.WithFields(logrus.Fields{
		"connection_id":   cred.ConnectionID,
		"organization_id": cred.OrganizationID,
	}).Info("Shared credential updated successfully")

	return nil
}

// GetByConnectionID retrieves all shared credentials for a specific connection across all organizations
// This is useful for cleanup when a connection is deleted
func (s *SharedCredentialStore) GetByConnectionID(ctx context.Context, connID string) ([]*SharedCredential, error) {
	query := `
		SELECT id, connection_id, organization_id, encrypted_password,
			password_iv, password_auth_tag, created_at, updated_at, created_by
		FROM shared_credentials
		WHERE connection_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, connID)
	if err != nil {
		return nil, fmt.Errorf("failed to query shared credentials by connection: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var credentials []*SharedCredential

	for rows.Next() {
		var cred SharedCredential
		var createdAt, updatedAt int64

		err := rows.Scan(
			&cred.ID,
			&cred.ConnectionID,
			&cred.OrganizationID,
			&cred.EncryptedPassword,
			&cred.PasswordIV,
			&cred.PasswordAuthTag,
			&createdAt,
			&updatedAt,
			&cred.CreatedBy,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan shared credential row")
			continue
		}

		cred.CreatedAt = time.Unix(createdAt, 0)
		cred.UpdatedAt = time.Unix(updatedAt, 0)

		credentials = append(credentials, &cred)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating credential rows: %w", err)
	}

	return credentials, nil
}
