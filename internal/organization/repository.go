package organization

import (
	"context"
)

// Repository defines the interface for organization persistence
type Repository interface {
	// Organization CRUD
	Create(ctx context.Context, org *Organization) error
	GetByID(ctx context.Context, id string) (*Organization, error)
	GetByUserID(ctx context.Context, userID string) ([]*Organization, error)
	Update(ctx context.Context, org *Organization) error
	Delete(ctx context.Context, id string) error

	// Member management
	AddMember(ctx context.Context, member *OrganizationMember) error
	RemoveMember(ctx context.Context, orgID, userID string) error
	GetMember(ctx context.Context, orgID, userID string) (*OrganizationMember, error)
	GetMembers(ctx context.Context, orgID string) ([]*OrganizationMember, error)
	UpdateMemberRole(ctx context.Context, orgID, userID string, role OrganizationRole) error
	GetMemberCount(ctx context.Context, orgID string) (int, error)

	// Invitation management
	CreateInvitation(ctx context.Context, invitation *OrganizationInvitation) error
	GetInvitation(ctx context.Context, id string) (*OrganizationInvitation, error)
	GetInvitationByToken(ctx context.Context, token string) (*OrganizationInvitation, error)
	GetInvitationsByOrg(ctx context.Context, orgID string) ([]*OrganizationInvitation, error)
	GetInvitationsByEmail(ctx context.Context, email string) ([]*OrganizationInvitation, error)
	UpdateInvitation(ctx context.Context, invitation *OrganizationInvitation) error
	DeleteInvitation(ctx context.Context, id string) error

	// Audit logging
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	GetAuditLogs(ctx context.Context, orgID string, limit, offset int) ([]*AuditLog, error)
}
