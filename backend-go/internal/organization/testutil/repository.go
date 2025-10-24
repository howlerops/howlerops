package testutil

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/sql-studio/backend-go/internal/organization"
)

// SQLiteRepository implements the organization.Repository interface using SQLite for testing
type SQLiteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository creates a new SQLite repository for testing
func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

// Create creates a new organization
func (r *SQLiteRepository) Create(ctx context.Context, org *organization.Organization) error {
	if org.ID == "" {
		org.ID = uuid.New().String()
	}
	org.CreatedAt = time.Now()
	org.UpdatedAt = time.Now()

	settings, _ := json.Marshal(org.Settings)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO organizations (id, name, description, owner_id, created_at, updated_at, max_members, settings)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		org.ID, org.Name, org.Description, org.OwnerID, org.CreatedAt, org.UpdatedAt, org.MaxMembers, string(settings),
	)
	if err != nil {
		return err
	}

	// Automatically add owner as a member
	member := &organization.OrganizationMember{
		ID:             uuid.New().String(),
		OrganizationID: org.ID,
		UserID:         org.OwnerID,
		Role:           organization.RoleOwner,
		JoinedAt:       time.Now(),
	}
	return r.AddMember(ctx, member)
}

// GetByID retrieves an organization by ID
func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (*organization.Organization, error) {
	var org organization.Organization
	var settings string
	var deletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, description, owner_id, created_at, updated_at, deleted_at, max_members, settings
		 FROM organizations WHERE id = ?`,
		id,
	).Scan(&org.ID, &org.Name, &org.Description, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt, &deletedAt, &org.MaxMembers, &settings)

	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		org.DeletedAt = &deletedAt.Time
	}

	if err := json.Unmarshal([]byte(settings), &org.Settings); err != nil {
		org.Settings = make(map[string]interface{})
	}

	// Get member count
	count, _ := r.GetMemberCount(ctx, id)
	org.MemberCount = count

	return &org, nil
}

// GetByUserID retrieves organizations for a user
func (r *SQLiteRepository) GetByUserID(ctx context.Context, userID string) ([]*organization.Organization, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DISTINCT o.id, o.name, o.description, o.owner_id, o.created_at, o.updated_at, o.deleted_at, o.max_members, o.settings
		 FROM organizations o
		 INNER JOIN organization_members m ON o.id = m.organization_id
		 WHERE m.user_id = ? AND o.deleted_at IS NULL
		 ORDER BY o.created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*organization.Organization
	for rows.Next() {
		var org organization.Organization
		var settings string
		var deletedAt sql.NullTime

		if err := rows.Scan(&org.ID, &org.Name, &org.Description, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt, &deletedAt, &org.MaxMembers, &settings); err != nil {
			return nil, err
		}

		if deletedAt.Valid {
			org.DeletedAt = &deletedAt.Time
		}

		if err := json.Unmarshal([]byte(settings), &org.Settings); err != nil {
			org.Settings = make(map[string]interface{})
		}

		count, _ := r.GetMemberCount(ctx, org.ID)
		org.MemberCount = count

		orgs = append(orgs, &org)
	}

	return orgs, nil
}

// Update updates an organization
func (r *SQLiteRepository) Update(ctx context.Context, org *organization.Organization) error {
	org.UpdatedAt = time.Now()
	settings, _ := json.Marshal(org.Settings)

	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations SET name = ?, description = ?, updated_at = ?, max_members = ?, settings = ?
		 WHERE id = ?`,
		org.Name, org.Description, org.UpdatedAt, org.MaxMembers, string(settings), org.ID,
	)
	return err
}

// Delete soft-deletes an organization
func (r *SQLiteRepository) Delete(ctx context.Context, id string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx,
		`UPDATE organizations SET deleted_at = ? WHERE id = ?`,
		now, id,
	)
	return err
}

// AddMember adds a member to an organization
func (r *SQLiteRepository) AddMember(ctx context.Context, member *organization.OrganizationMember) error {
	if member.ID == "" {
		member.ID = uuid.New().String()
	}
	if member.JoinedAt.IsZero() {
		member.JoinedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO organization_members (id, organization_id, user_id, role, invited_by, joined_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		member.ID, member.OrganizationID, member.UserID, member.Role, member.InvitedBy, member.JoinedAt,
	)
	return err
}

// RemoveMember removes a member from an organization
func (r *SQLiteRepository) RemoveMember(ctx context.Context, orgID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM organization_members WHERE organization_id = ? AND user_id = ?`,
		orgID, userID,
	)
	return err
}

// GetMember retrieves a specific member
func (r *SQLiteRepository) GetMember(ctx context.Context, orgID, userID string) (*organization.OrganizationMember, error) {
	var member organization.OrganizationMember
	var invitedBy sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, organization_id, user_id, role, invited_by, joined_at
		 FROM organization_members
		 WHERE organization_id = ? AND user_id = ?`,
		orgID, userID,
	).Scan(&member.ID, &member.OrganizationID, &member.UserID, &member.Role, &invitedBy, &member.JoinedAt)

	if err != nil {
		return nil, err
	}

	if invitedBy.Valid {
		member.InvitedBy = &invitedBy.String
	}

	return &member, nil
}

// GetMembers retrieves all members of an organization
func (r *SQLiteRepository) GetMembers(ctx context.Context, orgID string) ([]*organization.OrganizationMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.organization_id, m.user_id, m.role, m.invited_by, m.joined_at,
		        u.id, u.email, u.username, u.display_name
		 FROM organization_members m
		 INNER JOIN users u ON m.user_id = u.id
		 WHERE m.organization_id = ?
		 ORDER BY m.joined_at ASC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*organization.OrganizationMember
	for rows.Next() {
		var member organization.OrganizationMember
		var user organization.UserInfo
		var invitedBy sql.NullString
		var displayName sql.NullString

		if err := rows.Scan(
			&member.ID, &member.OrganizationID, &member.UserID, &member.Role, &invitedBy, &member.JoinedAt,
			&user.ID, &user.Email, &user.Username, &displayName,
		); err != nil {
			return nil, err
		}

		if invitedBy.Valid {
			member.InvitedBy = &invitedBy.String
		}

		if displayName.Valid {
			user.DisplayName = &displayName.String
		}

		member.User = &user
		members = append(members, &member)
	}

	return members, nil
}

// UpdateMemberRole updates a member's role
func (r *SQLiteRepository) UpdateMemberRole(ctx context.Context, orgID, userID string, role organization.OrganizationRole) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organization_members SET role = ? WHERE organization_id = ? AND user_id = ?`,
		role, orgID, userID,
	)
	return err
}

// GetMemberCount retrieves the member count for an organization
func (r *SQLiteRepository) GetMemberCount(ctx context.Context, orgID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM organization_members WHERE organization_id = ?`,
		orgID,
	).Scan(&count)
	return count, err
}

// CreateInvitation creates a new invitation
func (r *SQLiteRepository) CreateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	if invitation.ID == "" {
		invitation.ID = uuid.New().String()
	}
	if invitation.CreatedAt.IsZero() {
		invitation.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO organization_invitations (id, organization_id, email, role, invited_by, token, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		invitation.ID, invitation.OrganizationID, invitation.Email, invitation.Role,
		invitation.InvitedBy, invitation.Token, invitation.ExpiresAt, invitation.CreatedAt,
	)
	return err
}

// GetInvitation retrieves an invitation by ID
func (r *SQLiteRepository) GetInvitation(ctx context.Context, id string) (*organization.OrganizationInvitation, error) {
	var inv organization.OrganizationInvitation
	var acceptedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT id, organization_id, email, role, invited_by, token, expires_at, accepted_at, created_at
		 FROM organization_invitations WHERE id = ?`,
		id,
	).Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.Token, &inv.ExpiresAt, &acceptedAt, &inv.CreatedAt)

	if err != nil {
		return nil, err
	}

	if acceptedAt.Valid {
		inv.AcceptedAt = &acceptedAt.Time
	}

	return &inv, nil
}

// GetInvitationByToken retrieves an invitation by token
func (r *SQLiteRepository) GetInvitationByToken(ctx context.Context, token string) (*organization.OrganizationInvitation, error) {
	var inv organization.OrganizationInvitation
	var acceptedAt sql.NullTime
	var org organization.Organization
	var settings string
	var orgDeletedAt sql.NullTime

	err := r.db.QueryRowContext(ctx,
		`SELECT i.id, i.organization_id, i.email, i.role, i.invited_by, i.token, i.expires_at, i.accepted_at, i.created_at,
		        o.id, o.name, o.description, o.owner_id, o.created_at, o.updated_at, o.deleted_at, o.max_members, o.settings
		 FROM organization_invitations i
		 INNER JOIN organizations o ON i.organization_id = o.id
		 WHERE i.token = ?`,
		token,
	).Scan(
		&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.Token, &inv.ExpiresAt, &acceptedAt, &inv.CreatedAt,
		&org.ID, &org.Name, &org.Description, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt, &orgDeletedAt, &org.MaxMembers, &settings,
	)

	if err != nil {
		return nil, err
	}

	if acceptedAt.Valid {
		inv.AcceptedAt = &acceptedAt.Time
	}

	if orgDeletedAt.Valid {
		org.DeletedAt = &orgDeletedAt.Time
	}

	if err := json.Unmarshal([]byte(settings), &org.Settings); err != nil {
		org.Settings = make(map[string]interface{})
	}

	inv.Organization = &org

	return &inv, nil
}

// GetInvitationsByOrg retrieves invitations for an organization
func (r *SQLiteRepository) GetInvitationsByOrg(ctx context.Context, orgID string) ([]*organization.OrganizationInvitation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, organization_id, email, role, invited_by, token, expires_at, accepted_at, created_at
		 FROM organization_invitations
		 WHERE organization_id = ? AND accepted_at IS NULL
		 ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*organization.OrganizationInvitation
	for rows.Next() {
		var inv organization.OrganizationInvitation
		var acceptedAt sql.NullTime

		if err := rows.Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.Token, &inv.ExpiresAt, &acceptedAt, &inv.CreatedAt); err != nil {
			return nil, err
		}

		if acceptedAt.Valid {
			inv.AcceptedAt = &acceptedAt.Time
		}

		invitations = append(invitations, &inv)
	}

	return invitations, nil
}

// GetInvitationsByEmail retrieves invitations for an email
func (r *SQLiteRepository) GetInvitationsByEmail(ctx context.Context, email string) ([]*organization.OrganizationInvitation, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, organization_id, email, role, invited_by, token, expires_at, accepted_at, created_at
		 FROM organization_invitations
		 WHERE email = ? AND accepted_at IS NULL AND expires_at > datetime('now')
		 ORDER BY created_at DESC`,
		email,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var invitations []*organization.OrganizationInvitation
	for rows.Next() {
		var inv organization.OrganizationInvitation
		var acceptedAt sql.NullTime

		if err := rows.Scan(&inv.ID, &inv.OrganizationID, &inv.Email, &inv.Role, &inv.InvitedBy, &inv.Token, &inv.ExpiresAt, &acceptedAt, &inv.CreatedAt); err != nil {
			return nil, err
		}

		if acceptedAt.Valid {
			inv.AcceptedAt = &acceptedAt.Time
		}

		invitations = append(invitations, &inv)
	}

	return invitations, nil
}

// UpdateInvitation updates an invitation
func (r *SQLiteRepository) UpdateInvitation(ctx context.Context, invitation *organization.OrganizationInvitation) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE organization_invitations SET accepted_at = ? WHERE id = ?`,
		invitation.AcceptedAt, invitation.ID,
	)
	return err
}

// DeleteInvitation deletes an invitation
func (r *SQLiteRepository) DeleteInvitation(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM organization_invitations WHERE id = ?`,
		id,
	)
	return err
}

// CreateAuditLog creates an audit log entry
func (r *SQLiteRepository) CreateAuditLog(ctx context.Context, log *organization.AuditLog) error {
	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	details, _ := json.Marshal(log.Details)

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO audit_logs (id, organization_id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.OrganizationID, log.UserID, log.Action, log.ResourceType, log.ResourceID,
		log.IPAddress, log.UserAgent, string(details), log.CreatedAt,
	)
	return err
}

// GetAuditLogs retrieves audit logs for an organization
func (r *SQLiteRepository) GetAuditLogs(ctx context.Context, orgID string, limit, offset int) ([]*organization.AuditLog, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, organization_id, user_id, action, resource_type, resource_id, ip_address, user_agent, details, created_at
		 FROM audit_logs
		 WHERE organization_id = ?
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`,
		orgID, limit, offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*organization.AuditLog
	for rows.Next() {
		var log organization.AuditLog
		var orgID, resourceID, ipAddress, userAgent sql.NullString
		var details string

		if err := rows.Scan(&log.ID, &orgID, &log.UserID, &log.Action, &log.ResourceType, &resourceID, &ipAddress, &userAgent, &details, &log.CreatedAt); err != nil {
			return nil, err
		}

		if orgID.Valid {
			log.OrganizationID = &orgID.String
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

		if err := json.Unmarshal([]byte(details), &log.Details); err != nil {
			log.Details = make(map[string]interface{})
		}

		logs = append(logs, &log)
	}

	return logs, nil
}

// Helper method for tests to verify database state
func (r *SQLiteRepository) ExecRaw(ctx context.Context, query string, args ...interface{}) error {
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

// Helper to query raw results
func (r *SQLiteRepository) QueryRaw(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return r.db.QueryContext(ctx, query, args...)
}
