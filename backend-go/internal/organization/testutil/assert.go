package testutil

import (
	"testing"

	"github.com/sql-studio/backend-go/internal/organization"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertOrganizationEqual asserts that two organizations are equal (comparing key fields)
func AssertOrganizationEqual(t *testing.T, expected, actual *organization.Organization) {
	t.Helper()
	require.NotNil(t, actual, "Organization should not be nil")
	assert.Equal(t, expected.Name, actual.Name, "Organization name should match")
	assert.Equal(t, expected.Description, actual.Description, "Organization description should match")
	assert.Equal(t, expected.OwnerID, actual.OwnerID, "Organization owner ID should match")
	assert.Equal(t, expected.MaxMembers, actual.MaxMembers, "Organization max members should match")
}

// AssertOrganizationNotNil asserts that an organization is not nil and has required fields
func AssertOrganizationNotNil(t *testing.T, org *organization.Organization) {
	t.Helper()
	require.NotNil(t, org, "Organization should not be nil")
	assert.NotEmpty(t, org.ID, "Organization ID should not be empty")
	assert.NotEmpty(t, org.Name, "Organization name should not be empty")
	assert.NotEmpty(t, org.OwnerID, "Organization owner ID should not be empty")
	assert.NotZero(t, org.CreatedAt, "Organization created_at should not be zero")
	assert.NotZero(t, org.UpdatedAt, "Organization updated_at should not be zero")
	assert.Nil(t, org.DeletedAt, "Organization deleted_at should be nil for active orgs")
}

// AssertMemberEqual asserts that two members are equal (comparing key fields)
func AssertMemberEqual(t *testing.T, expected, actual *organization.OrganizationMember) {
	t.Helper()
	require.NotNil(t, actual, "Member should not be nil")
	assert.Equal(t, expected.OrganizationID, actual.OrganizationID, "Member organization ID should match")
	assert.Equal(t, expected.UserID, actual.UserID, "Member user ID should match")
	assert.Equal(t, expected.Role, actual.Role, "Member role should match")
}

// AssertMemberHasRole asserts that a member has a specific role
func AssertMemberHasRole(t *testing.T, member *organization.OrganizationMember, expectedRole organization.OrganizationRole) {
	t.Helper()
	require.NotNil(t, member, "Member should not be nil")
	assert.Equal(t, expectedRole, member.Role, "Member role should be %s", expectedRole)
}

// AssertInvitationEqual asserts that two invitations are equal (comparing key fields)
func AssertInvitationEqual(t *testing.T, expected, actual *organization.OrganizationInvitation) {
	t.Helper()
	require.NotNil(t, actual, "Invitation should not be nil")
	assert.Equal(t, expected.OrganizationID, actual.OrganizationID, "Invitation organization ID should match")
	assert.Equal(t, expected.Email, actual.Email, "Invitation email should match")
	assert.Equal(t, expected.Role, actual.Role, "Invitation role should match")
	assert.Equal(t, expected.InvitedBy, actual.InvitedBy, "Invitation invited_by should match")
}

// AssertInvitationPending asserts that an invitation is pending (not expired, not accepted)
func AssertInvitationPending(t *testing.T, invitation *organization.OrganizationInvitation) {
	t.Helper()
	require.NotNil(t, invitation, "Invitation should not be nil")
	assert.False(t, invitation.IsExpired(), "Invitation should not be expired")
	assert.False(t, invitation.IsAccepted(), "Invitation should not be accepted")
	assert.Nil(t, invitation.AcceptedAt, "Invitation accepted_at should be nil")
}

// AssertInvitationExpired asserts that an invitation is expired
func AssertInvitationExpired(t *testing.T, invitation *organization.OrganizationInvitation) {
	t.Helper()
	require.NotNil(t, invitation, "Invitation should not be nil")
	assert.True(t, invitation.IsExpired(), "Invitation should be expired")
}

// AssertInvitationAccepted asserts that an invitation is accepted
func AssertInvitationAccepted(t *testing.T, invitation *organization.OrganizationInvitation) {
	t.Helper()
	require.NotNil(t, invitation, "Invitation should not be nil")
	assert.True(t, invitation.IsAccepted(), "Invitation should be accepted")
	assert.NotNil(t, invitation.AcceptedAt, "Invitation accepted_at should not be nil")
}

// AssertAuditLogCreated asserts that an audit log was created with expected fields
func AssertAuditLogCreated(t *testing.T, log *organization.AuditLog, expectedAction, expectedResourceType string) {
	t.Helper()
	require.NotNil(t, log, "Audit log should not be nil")
	assert.NotEmpty(t, log.ID, "Audit log ID should not be empty")
	assert.Equal(t, expectedAction, log.Action, "Audit log action should be %s", expectedAction)
	assert.Equal(t, expectedResourceType, log.ResourceType, "Audit log resource type should be %s", expectedResourceType)
	assert.NotZero(t, log.CreatedAt, "Audit log created_at should not be zero")
}

// AssertErrorContains asserts that an error contains a specific substring
func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	require.Error(t, err, "Expected an error")
	assert.Contains(t, err.Error(), substring, "Error should contain '%s'", substring)
}

// AssertNoError asserts that there is no error
func AssertNoError(t *testing.T, err error, message string) {
	t.Helper()
	assert.NoError(t, err, message)
}

// AssertSliceLength asserts that a slice has the expected length
func AssertSliceLength[T any](t *testing.T, slice []T, expectedLength int) {
	t.Helper()
	assert.Len(t, slice, expectedLength, "Slice should have length %d", expectedLength)
}

// AssertSliceNotEmpty asserts that a slice is not empty
func AssertSliceNotEmpty[T any](t *testing.T, slice []T) {
	t.Helper()
	assert.NotEmpty(t, slice, "Slice should not be empty")
}

// AssertSliceEmpty asserts that a slice is empty
func AssertSliceEmpty[T any](t *testing.T, slice []T) {
	t.Helper()
	assert.Empty(t, slice, "Slice should be empty")
}

// AssertOrganizationDeleted asserts that an organization is soft-deleted
func AssertOrganizationDeleted(t *testing.T, org *organization.Organization) {
	t.Helper()
	require.NotNil(t, org, "Organization should not be nil")
	assert.NotNil(t, org.DeletedAt, "Organization deleted_at should not be nil")
}

// AssertRoleIsValid asserts that a role is valid
func AssertRoleIsValid(t *testing.T, role organization.OrganizationRole) {
	t.Helper()
	assert.True(t, role.Validate(), "Role %s should be valid", role)
}

// AssertRoleIsInvalid asserts that a role is invalid
func AssertRoleIsInvalid(t *testing.T, role organization.OrganizationRole) {
	t.Helper()
	assert.False(t, role.Validate(), "Role %s should be invalid", role)
}
