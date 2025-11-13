package turso

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sql-studio/backend-go/internal/organization"
)

// OrganizationStore implements organization.Repository for Turso
type OrganizationStore struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewOrganizationStore creates a new Turso organization store
func NewOrganizationStore(db *sql.DB, logger *logrus.Logger) *OrganizationStore {
	return &OrganizationStore{
		db:     db,
		logger: logger,
	}
}

// ====================================================================
// Organization CRUD Operations
// ====================================================================

// Create creates a new organization
func (s *OrganizationStore) Create(ctx context.Context, org *organization.Organization) error {
	query := `
		INSERT INTO organizations (id, name, description, owner_id, created_at, updated_at, max_members, settings)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Generate ID if not set
	if org.ID == "" {
		org.ID = uuid.New().String()
	}

	// Set timestamps
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	// Marshal settings
	var settingsJSON []byte
	var err error
	if len(org.Settings) > 0 {
		settingsJSON, err = json.Marshal(org.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}
	}

	_, err = s.db.ExecContext(
		ctx,
		query,
		org.ID,
		org.Name,
		org.Description,
		org.OwnerID,
		org.CreatedAt.Unix(),
		org.UpdatedAt.Unix(),
		org.MaxMembers,
		settingsJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to create organization: %w", err)
	}

	// Add owner as member
	member := &organization.OrganizationMember{
		ID:             uuid.New().String(),
		OrganizationID: org.ID,
		UserID:         org.OwnerID,
		Role:           organization.RoleOwner,
		JoinedAt:       now,
	}

	if err := s.AddMember(ctx, member); err != nil {
		return fmt.Errorf("failed to add owner as member: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": org.ID,
		"name":            org.Name,
		"owner_id":        org.OwnerID,
	}).Info("Organization created")

	return nil
}

// GetByID retrieves an organization by ID
func (s *OrganizationStore) GetByID(ctx context.Context, id string) (*organization.Organization, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at, deleted_at, max_members, settings
		FROM organizations
		WHERE id = ? AND deleted_at IS NULL
	`

	var org organization.Organization
	var settingsJSON sql.NullString
	var deletedAt sql.NullInt64
	var createdAt, updatedAt int64

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Description,
		&org.OwnerID,
		&createdAt,
		&updatedAt,
		&deletedAt,
		&org.MaxMembers,
		&settingsJSON,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("organization not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query organization: %w", err)
	}

	// Convert timestamps
	org.CreatedAt = time.Unix(createdAt, 0)
	org.UpdatedAt = time.Unix(updatedAt, 0)
	if deletedAt.Valid {
		t := time.Unix(deletedAt.Int64, 0)
		org.DeletedAt = &t
	}

	// Unmarshal settings
	if settingsJSON.Valid && settingsJSON.String != "" {
		if err := json.Unmarshal([]byte(settingsJSON.String), &org.Settings); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal organization settings")
		}
	}

	// Get member count
	count, err := s.GetMemberCount(ctx, id)
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get member count")
	} else {
		org.MemberCount = count
	}

	return &org, nil
}

// GetByUserID retrieves all organizations a user is a member of
func (s *OrganizationStore) GetByUserID(ctx context.Context, userID string) ([]*organization.Organization, error) {
	query := `
		SELECT DISTINCT o.id, o.name, o.description, o.owner_id, o.created_at, o.updated_at, o.deleted_at, o.max_members, o.settings
		FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = ? AND o.deleted_at IS NULL
		ORDER BY o.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query organizations: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var orgs []*organization.Organization
	for rows.Next() {
		var org organization.Organization
		var settingsJSON sql.NullString
		var deletedAt sql.NullInt64
		var createdAt, updatedAt int64

		err := rows.Scan(
			&org.ID,
			&org.Name,
			&org.Description,
			&org.OwnerID,
			&createdAt,
			&updatedAt,
			&deletedAt,
			&org.MaxMembers,
			&settingsJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}

		// Convert timestamps
		org.CreatedAt = time.Unix(createdAt, 0)
		org.UpdatedAt = time.Unix(updatedAt, 0)
		if deletedAt.Valid {
			t := time.Unix(deletedAt.Int64, 0)
			org.DeletedAt = &t
		}

		// Unmarshal settings
		if settingsJSON.Valid && settingsJSON.String != "" {
			if err := json.Unmarshal([]byte(settingsJSON.String), &org.Settings); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal organization settings")
			}
		}

		// Get member count
		count, err := s.GetMemberCount(ctx, org.ID)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to get member count")
		} else {
			org.MemberCount = count
		}

		orgs = append(orgs, &org)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organizations: %w", err)
	}

	return orgs, nil
}

// Update updates an organization
func (s *OrganizationStore) Update(ctx context.Context, org *organization.Organization) error {
	query := `
		UPDATE organizations
		SET name = ?, description = ?, updated_at = ?, max_members = ?, settings = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	// Update timestamp
	org.UpdatedAt = time.Now()

	// Marshal settings
	var settingsJSON []byte
	var err error
	if len(org.Settings) > 0 {
		settingsJSON, err = json.Marshal(org.Settings)
		if err != nil {
			return fmt.Errorf("failed to marshal settings: %w", err)
		}
	}

	result, err := s.db.ExecContext(
		ctx,
		query,
		org.Name,
		org.Description,
		org.UpdatedAt.Unix(),
		org.MaxMembers,
		settingsJSON,
		org.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization not found or already deleted")
	}

	s.logger.WithField("organization_id", org.ID).Info("Organization updated")
	return nil
}

// Delete soft-deletes an organization
func (s *OrganizationStore) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE organizations
		SET deleted_at = ?
		WHERE id = ? AND deleted_at IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, time.Now().Unix(), id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("organization not found or already deleted")
	}

	s.logger.WithField("organization_id", id).Info("Organization deleted")
	return nil
}

// ====================================================================
// Member Management Operations
// ====================================================================

// AddMember adds a member to an organization
func (s *OrganizationStore) AddMember(ctx context.Context, member *organization.OrganizationMember) error {
	query := `
		INSERT INTO organization_members (id, organization_id, user_id, role, invited_by, joined_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	// Generate ID if not set
	if member.ID == "" {
		member.ID = uuid.New().String()
	}

	// Set joined_at if not set
	if member.JoinedAt.IsZero() {
		member.JoinedAt = time.Now()
	}

	var invitedBy interface{}
	if member.InvitedBy != nil {
		invitedBy = *member.InvitedBy
	}

	_, err := s.db.ExecContext(
		ctx,
		query,
		member.ID,
		member.OrganizationID,
		member.UserID,
		member.Role.String(),
		invitedBy,
		member.JoinedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to add member: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": member.OrganizationID,
		"user_id":         member.UserID,
		"role":            member.Role,
	}).Info("Member added to organization")

	return nil
}

// RemoveMember removes a member from an organization
func (s *OrganizationStore) RemoveMember(ctx context.Context, orgID, userID string) error {
	query := `
		DELETE FROM organization_members
		WHERE organization_id = ? AND user_id = ?
	`

	result, err := s.db.ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("member not found")
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
	}).Info("Member removed from organization")

	return nil
}

// GetMember retrieves a specific member
func (s *OrganizationStore) GetMember(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
	query := `
		SELECT om.id, om.organization_id, om.user_id, om.role, om.invited_by, om.joined_at,
			   u.id, u.email, u.username
		FROM organization_members om
		INNER JOIN users u ON om.user_id = u.id
		WHERE om.organization_id = ? AND om.user_id = ?
	`

	var member organization.OrganizationMember
	var user organization.UserInfo
	var invitedBy sql.NullString
	var joinedAt int64
	var roleStr string

	err := s.db.QueryRowContext(ctx, query, orgID, userID).Scan(
		&member.ID,
		&member.OrganizationID,
		&member.UserID,
		&roleStr,
		&invitedBy,
		&joinedAt,
		&user.ID,
		&user.Email,
		&user.Username,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("member not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query member: %w", err)
	}

	// Convert role string to type
	member.Role = organization.OrganizationRole(roleStr)
	member.JoinedAt = time.Unix(joinedAt, 0)
	if invitedBy.Valid {
		member.InvitedBy = &invitedBy.String
	}
	member.User = &user

	return &member, nil
}

// GetMembers retrieves all members of an organization
func (s *OrganizationStore) GetMembers(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
	query := `
		SELECT om.id, om.organization_id, om.user_id, om.role, om.invited_by, om.joined_at,
			   u.id, u.email, u.username
		FROM organization_members om
		INNER JOIN users u ON om.user_id = u.id
		WHERE om.organization_id = ?
		ORDER BY om.joined_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query members: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var members []*organization.OrganizationMember
	for rows.Next() {
		var member organization.OrganizationMember
		var user organization.UserInfo
		var invitedBy sql.NullString
		var joinedAt int64
		var roleStr string

		err := rows.Scan(
			&member.ID,
			&member.OrganizationID,
			&member.UserID,
			&roleStr,
			&invitedBy,
			&joinedAt,
			&user.ID,
			&user.Email,
			&user.Username,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan member: %w", err)
		}

		member.Role = organization.OrganizationRole(roleStr)
		member.JoinedAt = time.Unix(joinedAt, 0)
		if invitedBy.Valid {
			member.InvitedBy = &invitedBy.String
		}
		member.User = &user

		members = append(members, &member)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating members: %w", err)
	}

	return members, nil
}

// UpdateMemberRole updates a member's role
func (s *OrganizationStore) UpdateMemberRole(ctx context.Context, orgID, userID string, role organization.OrganizationRole) error {
	query := `
		UPDATE organization_members
		SET role = ?
		WHERE organization_id = ? AND user_id = ?
	`

	result, err := s.db.ExecContext(ctx, query, role.String(), orgID, userID)
	if err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("member not found")
	}

	s.logger.WithFields(logrus.Fields{
		"organization_id": orgID,
		"user_id":         userID,
		"new_role":        role,
	}).Info("Member role updated")

	return nil
}

// GetMemberCount returns the number of members in an organization
func (s *OrganizationStore) GetMemberCount(ctx context.Context, orgID string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM organization_members
		WHERE organization_id = ?
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}

	return count, nil
}

// ====================================================================
// Invitation Management Operations
// ====================================================================

// CreateInvitation creates a new invitation
func (s *OrganizationStore) CreateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	query := `
		INSERT INTO organization_invitations (id, organization_id, email, role, invited_by, token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Generate ID if not set
	if invitation.ID == "" {
		invitation.ID = uuid.New().String()
	}

	// Set created_at if not set
	if invitation.CreatedAt.IsZero() {
		invitation.CreatedAt = time.Now()
	}

	_, err := s.db.ExecContext(
		ctx,
		query,
		invitation.ID,
		invitation.OrganizationID,
		invitation.Email,
		invitation.Role.String(),
		invitation.InvitedBy,
		invitation.Token,
		invitation.ExpiresAt.Unix(),
		invitation.CreatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to create invitation: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"invitation_id":   invitation.ID,
		"organization_id": invitation.OrganizationID,
		"email":           invitation.Email,
	}).Info("Invitation created")

	return nil
}

// GetInvitation retrieves an invitation by ID
func (s *OrganizationStore) GetInvitation(ctx context.Context, id string) (*organization.OrganizationInvitation, error) {
	query := `
		SELECT id, organization_id, email, role, invited_by, token, expires_at, accepted_at, created_at
		FROM organization_invitations
		WHERE id = ?
	`

	var inv organization.OrganizationInvitation
	var expiresAt, createdAt int64
	var acceptedAt sql.NullInt64
	var roleStr string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&roleStr,
		&inv.InvitedBy,
		&inv.Token,
		&expiresAt,
		&acceptedAt,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invitation: %w", err)
	}

	inv.Role = organization.OrganizationRole(roleStr)
	inv.ExpiresAt = time.Unix(expiresAt, 0)
	inv.CreatedAt = time.Unix(createdAt, 0)
	if acceptedAt.Valid {
		t := time.Unix(acceptedAt.Int64, 0)
		inv.AcceptedAt = &t
	}

	return &inv, nil
}

// GetInvitationByToken retrieves an invitation by token
func (s *OrganizationStore) GetInvitationByToken(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
	query := `
		SELECT i.id, i.organization_id, i.email, i.role, i.invited_by, i.token, i.expires_at, i.accepted_at, i.created_at,
			   o.id, o.name, o.description, o.owner_id, o.created_at, o.updated_at, o.deleted_at, o.max_members, o.settings
		FROM organization_invitations i
		INNER JOIN organizations o ON i.organization_id = o.id
		WHERE i.token = ?
	`

	var inv organization.OrganizationInvitation
	var org organization.Organization
	var expiresAt, createdAt, orgCreatedAt, orgUpdatedAt int64
	var acceptedAt, orgDeletedAt sql.NullInt64
	var roleStr string
	var settingsJSON sql.NullString

	err := s.db.QueryRowContext(ctx, query, token).Scan(
		&inv.ID,
		&inv.OrganizationID,
		&inv.Email,
		&roleStr,
		&inv.InvitedBy,
		&inv.Token,
		&expiresAt,
		&acceptedAt,
		&createdAt,
		&org.ID,
		&org.Name,
		&org.Description,
		&org.OwnerID,
		&orgCreatedAt,
		&orgUpdatedAt,
		&orgDeletedAt,
		&org.MaxMembers,
		&settingsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invitation not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query invitation: %w", err)
	}

	inv.Role = organization.OrganizationRole(roleStr)
	inv.ExpiresAt = time.Unix(expiresAt, 0)
	inv.CreatedAt = time.Unix(createdAt, 0)
	if acceptedAt.Valid {
		t := time.Unix(acceptedAt.Int64, 0)
		inv.AcceptedAt = &t
	}

	// Set organization details
	org.CreatedAt = time.Unix(orgCreatedAt, 0)
	org.UpdatedAt = time.Unix(orgUpdatedAt, 0)
	if orgDeletedAt.Valid {
		t := time.Unix(orgDeletedAt.Int64, 0)
		org.DeletedAt = &t
	}
	if settingsJSON.Valid && settingsJSON.String != "" {
		if err := json.Unmarshal([]byte(settingsJSON.String), &org.Settings); err != nil {
			s.logger.WithError(err).Warn("Failed to unmarshal organization settings")
		}
	}
	inv.Organization = &org

	return &inv, nil
}

// GetInvitationsByOrg retrieves all invitations for an organization
func (s *OrganizationStore) GetInvitationsByOrg(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
	query := `
		SELECT id, organization_id, email, role, invited_by, token, expires_at, accepted_at, created_at
		FROM organization_invitations
		WHERE organization_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query invitations: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var invitations []*organization.OrganizationInvitation
	for rows.Next() {
		var inv organization.OrganizationInvitation
		var expiresAt, createdAt int64
		var acceptedAt sql.NullInt64
		var roleStr string

		err := rows.Scan(
			&inv.ID,
			&inv.OrganizationID,
			&inv.Email,
			&roleStr,
			&inv.InvitedBy,
			&inv.Token,
			&expiresAt,
			&acceptedAt,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}

		inv.Role = organization.OrganizationRole(roleStr)
		inv.ExpiresAt = time.Unix(expiresAt, 0)
		inv.CreatedAt = time.Unix(createdAt, 0)
		if acceptedAt.Valid {
			t := time.Unix(acceptedAt.Int64, 0)
			inv.AcceptedAt = &t
		}

		invitations = append(invitations, &inv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invitations: %w", err)
	}

	return invitations, nil
}

// GetInvitationsByEmail retrieves pending invitations for an email
func (s *OrganizationStore) GetInvitationsByEmail(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error) {
	query := `
		SELECT i.id, i.organization_id, i.email, i.role, i.invited_by, i.token, i.expires_at, i.accepted_at, i.created_at,
			   o.id, o.name, o.description, o.owner_id, o.created_at, o.updated_at, o.deleted_at, o.max_members, o.settings
		FROM organization_invitations i
		INNER JOIN organizations o ON i.organization_id = o.id
		WHERE i.email = ? AND i.accepted_at IS NULL AND i.expires_at > ?
		ORDER BY i.created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, email, time.Now().Unix())
	if err != nil {
		return nil, fmt.Errorf("failed to query invitations: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var invitations []*organization.OrganizationInvitation
	for rows.Next() {
		var inv organization.OrganizationInvitation
		var org organization.Organization
		var expiresAt, createdAt, orgCreatedAt, orgUpdatedAt int64
		var acceptedAt, orgDeletedAt sql.NullInt64
		var roleStr string
		var settingsJSON sql.NullString

		err := rows.Scan(
			&inv.ID,
			&inv.OrganizationID,
			&inv.Email,
			&roleStr,
			&inv.InvitedBy,
			&inv.Token,
			&expiresAt,
			&acceptedAt,
			&createdAt,
			&org.ID,
			&org.Name,
			&org.Description,
			&org.OwnerID,
			&orgCreatedAt,
			&orgUpdatedAt,
			&orgDeletedAt,
			&org.MaxMembers,
			&settingsJSON,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan invitation: %w", err)
		}

		inv.Role = organization.OrganizationRole(roleStr)
		inv.ExpiresAt = time.Unix(expiresAt, 0)
		inv.CreatedAt = time.Unix(createdAt, 0)
		if acceptedAt.Valid {
			t := time.Unix(acceptedAt.Int64, 0)
			inv.AcceptedAt = &t
		}

		// Set organization details
		org.CreatedAt = time.Unix(orgCreatedAt, 0)
		org.UpdatedAt = time.Unix(orgUpdatedAt, 0)
		if orgDeletedAt.Valid {
			t := time.Unix(orgDeletedAt.Int64, 0)
			org.DeletedAt = &t
		}
		if settingsJSON.Valid && settingsJSON.String != "" {
			if err := json.Unmarshal([]byte(settingsJSON.String), &org.Settings); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal organization settings")
			}
		}
		inv.Organization = &org

		invitations = append(invitations, &inv)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating invitations: %w", err)
	}

	return invitations, nil
}

// UpdateInvitation updates an invitation
func (s *OrganizationStore) UpdateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	query := `
		UPDATE organization_invitations
		SET accepted_at = ?
		WHERE id = ?
	`

	var acceptedAt interface{}
	if invitation.AcceptedAt != nil {
		acceptedAt = invitation.AcceptedAt.Unix()
	}

	result, err := s.db.ExecContext(ctx, query, acceptedAt, invitation.ID)
	if err != nil {
		return fmt.Errorf("failed to update invitation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	s.logger.WithField("invitation_id", invitation.ID).Info("Invitation updated")
	return nil
}

// DeleteInvitation deletes an invitation
func (s *OrganizationStore) DeleteInvitation(ctx context.Context, id string) error {
	query := `
		DELETE FROM organization_invitations
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete invitation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invitation not found")
	}

	s.logger.WithField("invitation_id", id).Info("Invitation deleted")
	return nil
}

// ====================================================================
// Audit Log Operations
// ====================================================================

// CreateAuditLog creates a new audit log entry
func (s *OrganizationStore) CreateAuditLog(ctx context.Context, log *organization.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, organization_id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Generate ID if not set
	if log.ID == "" {
		log.ID = uuid.New().String()
	}

	// Set created_at if not set
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	// Marshal details
	var detailsJSON []byte
	var err error
	if len(log.Details) > 0 {
		detailsJSON, err = json.Marshal(log.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal details: %w", err)
		}
	}

	var orgID interface{}
	if log.OrganizationID != nil {
		orgID = *log.OrganizationID
	}

	var resourceID interface{}
	if log.ResourceID != nil {
		resourceID = *log.ResourceID
	}

	var ipAddress interface{}
	if log.IPAddress != nil {
		ipAddress = *log.IPAddress
	}

	var userAgent interface{}
	if log.UserAgent != nil {
		userAgent = *log.UserAgent
	}

	_, err = s.db.ExecContext(
		ctx,
		query,
		log.ID,
		orgID,
		log.UserID,
		log.Action,
		log.ResourceType,
		resourceID,
		ipAddress,
		userAgent,
		detailsJSON,
		log.CreatedAt.Unix(),
	)

	if err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs for an organization
func (s *OrganizationStore) GetAuditLogs(ctx context.Context, orgID string, limit, offset int) ([]*organization.AuditLog, error) {
	query := `
		SELECT id, organization_id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at
		FROM audit_logs
		WHERE organization_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`

	rows, err := s.db.QueryContext(ctx, query, orgID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer func() { if err := rows.Close(); err != nil { s.logger.WithError(err).Error("Failed to close rows") } }()

	var logs []*organization.AuditLog
	for rows.Next() {
		var log organization.AuditLog
		var organizationID, resourceID, ipAddress, userAgent sql.NullString
		var detailsJSON sql.NullString
		var createdAt int64

		err := rows.Scan(
			&log.ID,
			&organizationID,
			&log.UserID,
			&log.Action,
			&log.ResourceType,
			&resourceID,
			&ipAddress,
			&userAgent,
			&detailsJSON,
			&createdAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		log.CreatedAt = time.Unix(createdAt, 0)
		if organizationID.Valid {
			log.OrganizationID = &organizationID.String
		}
		if resourceID.Valid {
			log.ResourceID = &resourceID.String
		}
		if ipAddress.Valid {
			log.IPAddress = &ipAddress.String
		}
		if userAgent.Valid {
			log.UserAgent = &userAgent.String
		}

		// Unmarshal details
		if detailsJSON.Valid && detailsJSON.String != "" {
			if err := json.Unmarshal([]byte(detailsJSON.String), &log.Details); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal audit log details")
			}
		}

		logs = append(logs, &log)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}
