package organization

import (
	"time"
)

// OrganizationRole represents a user's role in an organization
type OrganizationRole string

const (
	RoleOwner  OrganizationRole = "owner"
	RoleAdmin  OrganizationRole = "admin"
	RoleMember OrganizationRole = "member"
)

// Validate checks if the role is valid
func (r OrganizationRole) Validate() bool {
	switch r {
	case RoleOwner, RoleAdmin, RoleMember:
		return true
	default:
		return false
	}
}

// String returns the string representation
func (r OrganizationRole) String() string {
	return string(r)
}

// Organization represents a team/workspace
type Organization struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	OwnerID     string                 `json:"owner_id"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   *time.Time             `json:"deleted_at,omitempty"`
	MaxMembers  int                    `json:"max_members"`
	Settings    map[string]interface{} `json:"settings,omitempty"`

	// Computed fields (not in database)
	MemberCount int `json:"member_count,omitempty"`
}

// OrganizationMember represents a user's membership in an organization
type OrganizationMember struct {
	ID             string           `json:"id"`
	OrganizationID string           `json:"organization_id"`
	UserID         string           `json:"user_id"`
	Role           OrganizationRole `json:"role"`
	InvitedBy      *string          `json:"invited_by,omitempty"`
	JoinedAt       time.Time        `json:"joined_at"`

	// User details (joined from users table)
	User *UserInfo `json:"user,omitempty"`
}

// UserInfo contains basic user information for members
type UserInfo struct {
	ID          string  `json:"id"`
	Email       string  `json:"email"`
	Username    string  `json:"username"`
	DisplayName *string `json:"display_name,omitempty"`
}

// OrganizationInvitation represents an invitation to join an organization
type OrganizationInvitation struct {
	ID             string           `json:"id"`
	OrganizationID string           `json:"organization_id"`
	Email          string           `json:"email"`
	Role           OrganizationRole `json:"role"`
	InvitedBy      string           `json:"invited_by"`
	Token          string           `json:"token"`
	ExpiresAt      time.Time        `json:"expires_at"`
	AcceptedAt     *time.Time       `json:"accepted_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`

	// Organization details (joined from organizations table)
	Organization *Organization `json:"organization,omitempty"`
}

// IsExpired checks if the invitation has expired
func (i *OrganizationInvitation) IsExpired() bool {
	return time.Now().After(i.ExpiresAt)
}

// IsAccepted checks if the invitation has been accepted
func (i *OrganizationInvitation) IsAccepted() bool {
	return i.AcceptedAt != nil
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID             string                 `json:"id"`
	OrganizationID *string                `json:"organization_id,omitempty"`
	UserID         string                 `json:"user_id"`
	Action         string                 `json:"action"`
	ResourceType   string                 `json:"resource_type"`
	ResourceID     *string                `json:"resource_id,omitempty"`
	IPAddress      *string                `json:"ip_address,omitempty"`
	UserAgent      *string                `json:"user_agent,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
}

// CreateOrganizationInput represents the input for creating an organization
type CreateOrganizationInput struct {
	Name        string `json:"name" validate:"required,min=3,max=50"`
	Description string `json:"description" validate:"max=500"`
}

// UpdateOrganizationInput represents the input for updating an organization
type UpdateOrganizationInput struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=3,max=50"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=500"`
	MaxMembers  *int    `json:"max_members,omitempty" validate:"omitempty,min=1,max=1000"`
}

// CreateInvitationInput represents the input for creating an invitation
type CreateInvitationInput struct {
	Email string           `json:"email" validate:"required,email"`
	Role  OrganizationRole `json:"role" validate:"required,oneof=admin member"`
}

// UpdateMemberRoleInput represents the input for updating a member's role
type UpdateMemberRoleInput struct {
	Role OrganizationRole `json:"role" validate:"required,oneof=owner admin member"`
}
