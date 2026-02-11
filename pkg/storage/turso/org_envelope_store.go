package turso

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// OEKEncryptedData represents encrypted organization envelope key data
type OEKEncryptedData struct {
	Ciphertext string // Base64-encoded ciphertext
	IV         string // Base64-encoded IV/nonce
	AuthTag    string // Base64-encoded auth tag
}

// OrgEnvelopeKeyStore manages organization envelope keys (OEK)
// Each organization member has their own copy of the OEK, encrypted with their master key
type OrgEnvelopeKeyStore struct {
	db     *sql.DB
	logger *logrus.Logger
	// Prepared statements for performance
	stmtStore  *sql.Stmt
	stmtGet    *sql.Stmt
	stmtDelete *sql.Stmt
}

// StoredOEK represents an encrypted OEK for a user in an organization
type StoredOEK struct {
	ID             string
	OrganizationID string
	UserID         string
	EncryptedOEK   string
	OEKIV          string
	OEKAuthTag     string
	KeyVersion     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// NewOrgEnvelopeKeyStore creates a new OEK store with prepared statements
func NewOrgEnvelopeKeyStore(db *sql.DB, logger *logrus.Logger) (*OrgEnvelopeKeyStore, error) {
	store := &OrgEnvelopeKeyStore{
		db:     db,
		logger: logger,
	}

	var err error

	// Prepare store statement
	store.stmtStore, err = db.Prepare(`
		INSERT INTO org_envelope_keys (
			id, organization_id, user_id, encrypted_oek, oek_iv, oek_auth_tag,
			key_version, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(organization_id, user_id) DO UPDATE SET
			encrypted_oek = excluded.encrypted_oek,
			oek_iv = excluded.oek_iv,
			oek_auth_tag = excluded.oek_auth_tag,
			key_version = excluded.key_version,
			updated_at = excluded.updated_at
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare store statement: %w", err)
	}

	// Prepare get statement
	store.stmtGet, err = db.Prepare(`
		SELECT id, organization_id, user_id, encrypted_oek, oek_iv, oek_auth_tag,
			key_version, created_at, updated_at
		FROM org_envelope_keys
		WHERE organization_id = ? AND user_id = ?
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare get statement: %w", err)
	}

	// Prepare delete statement
	store.stmtDelete, err = db.Prepare(`
		DELETE FROM org_envelope_keys
		WHERE organization_id = ? AND user_id = ?
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare delete statement: %w", err)
	}

	return store, nil
}

// Close closes all prepared statements
func (s *OrgEnvelopeKeyStore) Close() error {
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

// StoreOEKForUser stores the encrypted OEK for a specific user in an organization
// The OEK is encrypted with the user's master key before storage
func (s *OrgEnvelopeKeyStore) StoreOEKForUser(ctx context.Context, orgID, userID string, encrypted *OEKEncryptedData) error {
	id := uuid.New().String()
	now := time.Now().Unix()

	_, err := s.stmtStore.ExecContext(ctx,
		id,
		orgID,
		userID,
		encrypted.Ciphertext,
		encrypted.IV,
		encrypted.AuthTag,
		1, // Initial version
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to store OEK for user: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("OEK stored successfully for user")

	return nil
}

// GetOEKForUser retrieves the encrypted OEK for a specific user in an organization
func (s *OrgEnvelopeKeyStore) GetOEKForUser(ctx context.Context, orgID, userID string) (*StoredOEK, error) {
	var oek StoredOEK
	var createdAt, updatedAt int64

	err := s.stmtGet.QueryRowContext(ctx, orgID, userID).Scan(
		&oek.ID,
		&oek.OrganizationID,
		&oek.UserID,
		&oek.EncryptedOEK,
		&oek.OEKIV,
		&oek.OEKAuthTag,
		&oek.KeyVersion,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("OEK not found for user in organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve OEK: %w", err)
	}

	oek.CreatedAt = time.Unix(createdAt, 0)
	oek.UpdatedAt = time.Unix(updatedAt, 0)

	return &oek, nil
}

// GetAnyOEKForOrg retrieves any member's OEK from the organization
// This is used when adding a new member - we decrypt an existing member's OEK,
// then re-encrypt it with the new member's master key
func (s *OrgEnvelopeKeyStore) GetAnyOEKForOrg(ctx context.Context, orgID string) (*StoredOEK, error) {
	query := `
		SELECT id, organization_id, user_id, encrypted_oek, oek_iv, oek_auth_tag,
			key_version, created_at, updated_at
		FROM org_envelope_keys
		WHERE organization_id = ?
		LIMIT 1
	`

	var oek StoredOEK
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, orgID).Scan(
		&oek.ID,
		&oek.OrganizationID,
		&oek.UserID,
		&oek.EncryptedOEK,
		&oek.OEKIV,
		&oek.OEKAuthTag,
		&oek.KeyVersion,
		&createdAt,
		&updatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no OEK found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve OEK: %w", err)
	}

	oek.CreatedAt = time.Unix(createdAt, 0)
	oek.UpdatedAt = time.Unix(updatedAt, 0)

	return &oek, nil
}

// DeleteOEKForUser removes the OEK when a member leaves the organization
func (s *OrgEnvelopeKeyStore) DeleteOEKForUser(ctx context.Context, orgID, userID string) error {
	result, err := s.stmtDelete.ExecContext(ctx, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete OEK: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OEK not found for user")
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("OEK deleted successfully")

	return nil
}

// GetAllOEKsForOrg retrieves all OEKs for an organization
// This is used during key rotation when the OEK itself needs to be changed
func (s *OrgEnvelopeKeyStore) GetAllOEKsForOrg(ctx context.Context, orgID string) ([]*StoredOEK, error) {
	query := `
		SELECT id, organization_id, user_id, encrypted_oek, oek_iv, oek_auth_tag,
			key_version, created_at, updated_at
		FROM org_envelope_keys
		WHERE organization_id = ?
		ORDER BY created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query OEKs: %w", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close rows")
		}
	}()

	var oeks []*StoredOEK

	for rows.Next() {
		var oek StoredOEK
		var createdAt, updatedAt int64

		err := rows.Scan(
			&oek.ID,
			&oek.OrganizationID,
			&oek.UserID,
			&oek.EncryptedOEK,
			&oek.OEKIV,
			&oek.OEKAuthTag,
			&oek.KeyVersion,
			&createdAt,
			&updatedAt,
		)

		if err != nil {
			s.logger.WithError(err).Warn("Failed to scan OEK row")
			continue
		}

		oek.CreatedAt = time.Unix(createdAt, 0)
		oek.UpdatedAt = time.Unix(updatedAt, 0)

		oeks = append(oeks, &oek)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating OEK rows: %w", err)
	}

	return oeks, nil
}

// UpdateOEKForUser updates the OEK for a user (used during key rotation)
func (s *OrgEnvelopeKeyStore) UpdateOEKForUser(ctx context.Context, orgID, userID string, encrypted *OEKEncryptedData, newVersion int) error {
	query := `
		UPDATE org_envelope_keys
		SET encrypted_oek = ?, oek_iv = ?, oek_auth_tag = ?,
			key_version = ?, updated_at = ?
		WHERE organization_id = ? AND user_id = ?
	`

	now := time.Now().Unix()

	result, err := s.db.ExecContext(ctx, query,
		encrypted.Ciphertext,
		encrypted.IV,
		encrypted.AuthTag,
		newVersion,
		now,
		orgID,
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to update OEK: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("OEK not found for user")
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
		"new_version":     newVersion,
	}).Info("OEK updated successfully")

	return nil
}
