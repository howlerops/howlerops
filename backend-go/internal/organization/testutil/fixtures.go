package testutil

import (
	"time"

	"github.com/google/uuid"
	"github.com/sql-studio/backend-go/internal/organization"
)

// CreateTestUser creates a test user with default values
func CreateTestUser(email, username string) *organization.UserInfo {
	displayName := username
	return &organization.UserInfo{
		ID:          uuid.New().String(),
		Email:       email,
		Username:    username,
		DisplayName: &displayName,
	}
}

// CreateTestOrganization creates a test organization with default values
func CreateTestOrganization(name, ownerID string) *organization.Organization {
	return &organization.Organization{
		ID:          uuid.New().String(),
		Name:        name,
		Description: "Test organization for " + name,
		OwnerID:     ownerID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		DeletedAt:   nil,
		MaxMembers:  10,
		Settings:    make(map[string]interface{}),
	}
}

// CreateTestMember creates a test organization member
func CreateTestMember(orgID, userID string, role organization.OrganizationRole) *organization.OrganizationMember {
	return &organization.OrganizationMember{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		InvitedBy:      nil,
		JoinedAt:       time.Now(),
	}
}

// CreateTestMemberWithInviter creates a test organization member with an inviter
func CreateTestMemberWithInviter(orgID, userID, invitedByID string, role organization.OrganizationRole) *organization.OrganizationMember {
	return &organization.OrganizationMember{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		UserID:         userID,
		Role:           role,
		InvitedBy:      &invitedByID,
		JoinedAt:       time.Now(),
	}
}

// CreateTestInvitation creates a test organization invitation
func CreateTestInvitation(orgID, email, invitedBy string, role organization.OrganizationRole) *organization.OrganizationInvitation {
	return &organization.OrganizationInvitation{
		ID:             uuid.New().String(),
		OrganizationID: orgID,
		Email:          email,
		Role:           role,
		InvitedBy:      invitedBy,
		Token:          uuid.New().String(),
		ExpiresAt:      time.Now().Add(7 * 24 * time.Hour),
		AcceptedAt:     nil,
		CreatedAt:      time.Now(),
	}
}

// CreateExpiredInvitation creates an expired invitation for testing
func CreateExpiredInvitation(orgID, email, invitedBy string, role organization.OrganizationRole) *organization.OrganizationInvitation {
	inv := CreateTestInvitation(orgID, email, invitedBy, role)
	inv.ExpiresAt = time.Now().Add(-1 * time.Hour) // Expired 1 hour ago
	return inv
}

// CreateAcceptedInvitation creates an accepted invitation for testing
func CreateAcceptedInvitation(orgID, email, invitedBy string, role organization.OrganizationRole) *organization.OrganizationInvitation {
	inv := CreateTestInvitation(orgID, email, invitedBy, role)
	now := time.Now()
	inv.AcceptedAt = &now
	return inv
}

// CreateTestAuditLog creates a test audit log entry
func CreateTestAuditLog(orgID, userID, action, resourceType, resourceID string) *organization.AuditLog {
	ipAddress := "127.0.0.1"
	userAgent := "test-user-agent"

	return &organization.AuditLog{
		ID:             uuid.New().String(),
		OrganizationID: &orgID,
		UserID:         userID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     &resourceID,
		IPAddress:      &ipAddress,
		UserAgent:      &userAgent,
		Details:        make(map[string]interface{}),
		CreatedAt:      time.Now(),
	}
}

// OrganizationBuilder provides a fluent interface for building test organizations
type OrganizationBuilder struct {
	org *organization.Organization
}

// NewOrganizationBuilder creates a new organization builder
func NewOrganizationBuilder(name, ownerID string) *OrganizationBuilder {
	return &OrganizationBuilder{
		org: CreateTestOrganization(name, ownerID),
	}
}

// WithDescription sets the organization description
func (b *OrganizationBuilder) WithDescription(desc string) *OrganizationBuilder {
	b.org.Description = desc
	return b
}

// WithMaxMembers sets the maximum number of members
func (b *OrganizationBuilder) WithMaxMembers(max int) *OrganizationBuilder {
	b.org.MaxMembers = max
	return b
}

// WithSettings sets the organization settings
func (b *OrganizationBuilder) WithSettings(settings map[string]interface{}) *OrganizationBuilder {
	b.org.Settings = settings
	return b
}

// WithID sets a specific ID (useful for tests)
func (b *OrganizationBuilder) WithID(id string) *OrganizationBuilder {
	b.org.ID = id
	return b
}

// Build returns the built organization
func (b *OrganizationBuilder) Build() *organization.Organization {
	return b.org
}

// InvitationBuilder provides a fluent interface for building test invitations
type InvitationBuilder struct {
	inv *organization.OrganizationInvitation
}

// NewInvitationBuilder creates a new invitation builder
func NewInvitationBuilder(orgID, email, invitedBy string, role organization.OrganizationRole) *InvitationBuilder {
	return &InvitationBuilder{
		inv: CreateTestInvitation(orgID, email, invitedBy, role),
	}
}

// WithToken sets a specific token
func (b *InvitationBuilder) WithToken(token string) *InvitationBuilder {
	b.inv.Token = token
	return b
}

// WithExpiresAt sets the expiration time
func (b *InvitationBuilder) WithExpiresAt(expiresAt time.Time) *InvitationBuilder {
	b.inv.ExpiresAt = expiresAt
	return b
}

// AsExpired marks the invitation as expired
func (b *InvitationBuilder) AsExpired() *InvitationBuilder {
	b.inv.ExpiresAt = time.Now().Add(-1 * time.Hour)
	return b
}

// AsAccepted marks the invitation as accepted
func (b *InvitationBuilder) AsAccepted() *InvitationBuilder {
	now := time.Now()
	b.inv.AcceptedAt = &now
	return b
}

// WithOrganization attaches organization details
func (b *InvitationBuilder) WithOrganization(org *organization.Organization) *InvitationBuilder {
	b.inv.Organization = org
	return b
}

// Build returns the built invitation
func (b *InvitationBuilder) Build() *organization.OrganizationInvitation {
	return b.inv
}

// CreateTestOrganizationInput creates a test CreateOrganizationInput
func CreateTestOrganizationInput(name, description string) *organization.CreateOrganizationInput {
	return &organization.CreateOrganizationInput{
		Name:        name,
		Description: description,
	}
}

// CreateTestUpdateOrganizationInput creates a test UpdateOrganizationInput
func CreateTestUpdateOrganizationInput() *organization.UpdateOrganizationInput {
	return &organization.UpdateOrganizationInput{}
}

// CreateTestInvitationInput creates a test CreateInvitationInput
func CreateTestInvitationInput(email string, role organization.OrganizationRole) *organization.CreateInvitationInput {
	return &organization.CreateInvitationInput{
		Email: email,
		Role:  role,
	}
}

// CreateTestUpdateMemberRoleInput creates a test UpdateMemberRoleInput
func CreateTestUpdateMemberRoleInput(role organization.OrganizationRole) *organization.UpdateMemberRoleInput {
	return &organization.UpdateMemberRoleInput{
		Role: role,
	}
}
