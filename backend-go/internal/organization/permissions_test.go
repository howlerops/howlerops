package organization_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jbeck018/howlerops/backend-go/internal/organization"
)

// ====================================================================
// Permission Matrix Tests
// ====================================================================

func TestHasPermission_Owner(t *testing.T) {
	tests := []struct {
		name       string
		permission organization.Permission
		expected   bool
	}{
		{"view organization", organization.PermViewOrganization, true},
		{"update organization", organization.PermUpdateOrganization, true},
		{"delete organization", organization.PermDeleteOrganization, true},
		{"invite members", organization.PermInviteMembers, true},
		{"remove members", organization.PermRemoveMembers, true},
		{"update member roles", organization.PermUpdateMemberRoles, true},
		{"view audit logs", organization.PermViewAuditLogs, true},
		{"view connections", organization.PermViewConnections, true},
		{"create connections", organization.PermCreateConnections, true},
		{"update connections", organization.PermUpdateConnections, true},
		{"delete connections", organization.PermDeleteConnections, true},
		{"view queries", organization.PermViewQueries, true},
		{"create queries", organization.PermCreateQueries, true},
		{"update queries", organization.PermUpdateQueries, true},
		{"delete queries", organization.PermDeleteQueries, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := organization.HasPermission(organization.RoleOwner, tt.permission)
			assert.Equal(t, tt.expected, result, "Owner should have permission: %s", tt.permission)
		})
	}
}

func TestHasPermission_Admin(t *testing.T) {
	tests := []struct {
		name       string
		permission organization.Permission
		expected   bool
	}{
		{"view organization", organization.PermViewOrganization, true},
		{"update organization", organization.PermUpdateOrganization, true},
		{"delete organization", organization.PermDeleteOrganization, false}, // Admins cannot delete org
		{"invite members", organization.PermInviteMembers, true},
		{"remove members", organization.PermRemoveMembers, true},
		{"update member roles", organization.PermUpdateMemberRoles, true},
		{"view audit logs", organization.PermViewAuditLogs, true},
		{"view connections", organization.PermViewConnections, true},
		{"create connections", organization.PermCreateConnections, true},
		{"update connections", organization.PermUpdateConnections, true},
		{"delete connections", organization.PermDeleteConnections, true},
		{"view queries", organization.PermViewQueries, true},
		{"create queries", organization.PermCreateQueries, true},
		{"update queries", organization.PermUpdateQueries, true},
		{"delete queries", organization.PermDeleteQueries, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := organization.HasPermission(organization.RoleAdmin, tt.permission)
			assert.Equal(t, tt.expected, result, "Admin permission for %s should be %v", tt.permission, tt.expected)
		})
	}
}

func TestHasPermission_Member(t *testing.T) {
	tests := []struct {
		name       string
		permission organization.Permission
		expected   bool
	}{
		{"view organization", organization.PermViewOrganization, true},
		{"update organization", organization.PermUpdateOrganization, false},
		{"delete organization", organization.PermDeleteOrganization, false},
		{"invite members", organization.PermInviteMembers, false},
		{"remove members", organization.PermRemoveMembers, false},
		{"update member roles", organization.PermUpdateMemberRoles, false},
		{"view audit logs", organization.PermViewAuditLogs, false},
		{"view connections", organization.PermViewConnections, true},
		{"create connections", organization.PermCreateConnections, true},
		{"update connections", organization.PermUpdateConnections, false},
		{"delete connections", organization.PermDeleteConnections, false},
		{"view queries", organization.PermViewQueries, true},
		{"create queries", organization.PermCreateQueries, true},
		{"update queries", organization.PermUpdateQueries, false},
		{"delete queries", organization.PermDeleteQueries, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := organization.HasPermission(organization.RoleMember, tt.permission)
			assert.Equal(t, tt.expected, result, "Member permission for %s should be %v", tt.permission, tt.expected)
		})
	}
}

func TestHasPermission_InvalidRole(t *testing.T) {
	result := organization.HasPermission(organization.OrganizationRole("invalid"), organization.PermViewOrganization)
	assert.False(t, result, "Invalid role should not have any permissions")
}

// ====================================================================
// Resource Ownership Tests
// ====================================================================

func TestCanUpdateResource(t *testing.T) {
	tests := []struct {
		name            string
		role            organization.OrganizationRole
		resourceOwnerID string
		userID          string
		expected        bool
		description     string
	}{
		{
			"owner can update any resource",
			organization.RoleOwner,
			"other-user",
			"current-user",
			true,
			"Owner should be able to update any resource",
		},
		{
			"admin can update any resource",
			organization.RoleAdmin,
			"other-user",
			"current-user",
			true,
			"Admin should be able to update any resource",
		},
		{
			"member can update own resource",
			organization.RoleMember,
			"current-user",
			"current-user",
			true,
			"Member should be able to update their own resource",
		},
		{
			"member cannot update others resource",
			organization.RoleMember,
			"other-user",
			"current-user",
			false,
			"Member should not be able to update someone else's resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := organization.CanUpdateResource(tt.role, tt.resourceOwnerID, tt.userID)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestCanDeleteResource(t *testing.T) {
	tests := []struct {
		name            string
		role            organization.OrganizationRole
		resourceOwnerID string
		userID          string
		expected        bool
		description     string
	}{
		{
			"owner can delete any resource",
			organization.RoleOwner,
			"other-user",
			"current-user",
			true,
			"Owner should be able to delete any resource",
		},
		{
			"admin can delete any resource",
			organization.RoleAdmin,
			"other-user",
			"current-user",
			true,
			"Admin should be able to delete any resource",
		},
		{
			"member can delete own resource",
			organization.RoleMember,
			"current-user",
			"current-user",
			true,
			"Member should be able to delete their own resource",
		},
		{
			"member cannot delete others resource",
			organization.RoleMember,
			"other-user",
			"current-user",
			false,
			"Member should not be able to delete someone else's resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := organization.CanDeleteResource(tt.role, tt.resourceOwnerID, tt.userID)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

// ====================================================================
// Permission Utility Tests
// ====================================================================

func TestGetPermissionsForRole(t *testing.T) {
	tests := []struct {
		name        string
		role        organization.OrganizationRole
		expectedLen int
		description string
	}{
		{
			"owner permissions",
			organization.RoleOwner,
			15, // All 15 permissions
			"Owner should have all permissions",
		},
		{
			"admin permissions",
			organization.RoleAdmin,
			14, // All except delete org
			"Admin should have most permissions except delete org",
		},
		{
			"member permissions",
			organization.RoleMember,
			5, // View org, view/create connections, view/create queries
			"Member should have limited permissions",
		},
		{
			"invalid role",
			organization.OrganizationRole("invalid"),
			0,
			"Invalid role should have no permissions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := organization.GetPermissionsForRole(tt.role)
			assert.Len(t, perms, tt.expectedLen, tt.description)
		})
	}
}

func TestGetPermissionsForRole_ReturnsCopy(t *testing.T) {
	// Get permissions for owner
	perms1 := organization.GetPermissionsForRole(organization.RoleOwner)
	perms2 := organization.GetPermissionsForRole(organization.RoleOwner)

	// Modify first slice
	perms1[0] = organization.Permission("modified")

	// Second slice should be unchanged
	assert.NotEqual(t, perms1[0], perms2[0], "Modifying returned permissions should not affect subsequent calls")
}

func TestIsValidPermission(t *testing.T) {
	tests := []struct {
		name       string
		permission organization.Permission
		expected   bool
	}{
		{"valid - view org", organization.PermViewOrganization, true},
		{"valid - update org", organization.PermUpdateOrganization, true},
		{"valid - delete org", organization.PermDeleteOrganization, true},
		{"valid - invite members", organization.PermInviteMembers, true},
		{"valid - remove members", organization.PermRemoveMembers, true},
		{"valid - update member roles", organization.PermUpdateMemberRoles, true},
		{"valid - view audit logs", organization.PermViewAuditLogs, true},
		{"valid - view connections", organization.PermViewConnections, true},
		{"valid - create connections", organization.PermCreateConnections, true},
		{"valid - update connections", organization.PermUpdateConnections, true},
		{"valid - delete connections", organization.PermDeleteConnections, true},
		{"valid - view queries", organization.PermViewQueries, true},
		{"valid - create queries", organization.PermCreateQueries, true},
		{"valid - update queries", organization.PermUpdateQueries, true},
		{"valid - delete queries", organization.PermDeleteQueries, true},
		{"invalid - random string", organization.Permission("random:permission"), false},
		{"invalid - empty string", organization.Permission(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := organization.IsValidPermission(tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// ====================================================================
// Permission Scenario Tests
// ====================================================================

func TestPermissionScenario_MemberCannotManageOthers(t *testing.T) {
	// Scenario: A member tries to perform admin operations
	role := organization.RoleMember

	// Member cannot invite others
	assert.False(t, organization.HasPermission(role, organization.PermInviteMembers),
		"Members should not be able to invite others")

	// Member cannot remove others
	assert.False(t, organization.HasPermission(role, organization.PermRemoveMembers),
		"Members should not be able to remove others")

	// Member cannot update roles
	assert.False(t, organization.HasPermission(role, organization.PermUpdateMemberRoles),
		"Members should not be able to update roles")

	// Member cannot view audit logs
	assert.False(t, organization.HasPermission(role, organization.PermViewAuditLogs),
		"Members should not be able to view audit logs")
}

func TestPermissionScenario_AdminCanManageButNotDelete(t *testing.T) {
	// Scenario: An admin tries to perform operations
	role := organization.RoleAdmin

	// Admin can manage members
	assert.True(t, organization.HasPermission(role, organization.PermInviteMembers),
		"Admins should be able to invite members")
	assert.True(t, organization.HasPermission(role, organization.PermRemoveMembers),
		"Admins should be able to remove members")
	assert.True(t, organization.HasPermission(role, organization.PermUpdateMemberRoles),
		"Admins should be able to update member roles")

	// Admin can view audit logs
	assert.True(t, organization.HasPermission(role, organization.PermViewAuditLogs),
		"Admins should be able to view audit logs")

	// Admin can manage resources
	assert.True(t, organization.HasPermission(role, organization.PermUpdateConnections),
		"Admins should be able to update connections")
	assert.True(t, organization.HasPermission(role, organization.PermDeleteQueries),
		"Admins should be able to delete queries")

	// Admin CANNOT delete organization
	assert.False(t, organization.HasPermission(role, organization.PermDeleteOrganization),
		"Admins should NOT be able to delete the organization")
}

func TestPermissionScenario_OwnerHasAllPermissions(t *testing.T) {
	// Scenario: Owner should have all permissions
	role := organization.RoleOwner

	allPermissions := []organization.Permission{
		organization.PermViewOrganization,
		organization.PermUpdateOrganization,
		organization.PermDeleteOrganization,
		organization.PermInviteMembers,
		organization.PermRemoveMembers,
		organization.PermUpdateMemberRoles,
		organization.PermViewAuditLogs,
		organization.PermViewConnections,
		organization.PermCreateConnections,
		organization.PermUpdateConnections,
		organization.PermDeleteConnections,
		organization.PermViewQueries,
		organization.PermCreateQueries,
		organization.PermUpdateQueries,
		organization.PermDeleteQueries,
	}

	for _, perm := range allPermissions {
		assert.True(t, organization.HasPermission(role, perm),
			"Owner should have permission: %s", perm)
	}
}
