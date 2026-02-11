package organization

// Permission represents an action that can be performed
type Permission string

const (
	// Organization permissions
	PermViewOrganization   Permission = "org:view"
	PermUpdateOrganization Permission = "org:update"
	PermDeleteOrganization Permission = "org:delete"

	// Member management permissions
	PermInviteMembers     Permission = "members:invite"
	PermRemoveMembers     Permission = "members:remove"
	PermUpdateMemberRoles Permission = "members:update_roles"

	// Audit log permissions
	PermViewAuditLogs Permission = "audit:view"

	// Connection permissions
	PermViewConnections   Permission = "connections:view"
	PermCreateConnections Permission = "connections:create"
	PermUpdateConnections Permission = "connections:update"
	PermDeleteConnections Permission = "connections:delete"

	// Query permissions
	PermViewQueries   Permission = "queries:view"
	PermCreateQueries Permission = "queries:create"
	PermUpdateQueries Permission = "queries:update"
	PermDeleteQueries Permission = "queries:delete"
)

// RolePermissions defines what each role can do
var RolePermissions = map[OrganizationRole][]Permission{
	RoleOwner: {
		// Owners can do everything
		PermViewOrganization,
		PermUpdateOrganization,
		PermDeleteOrganization,
		PermInviteMembers,
		PermRemoveMembers,
		PermUpdateMemberRoles,
		PermViewAuditLogs,
		PermViewConnections,
		PermCreateConnections,
		PermUpdateConnections,
		PermDeleteConnections,
		PermViewQueries,
		PermCreateQueries,
		PermUpdateQueries,
		PermDeleteQueries,
	},
	RoleAdmin: {
		// Admins can do most things except delete org
		PermViewOrganization,
		PermUpdateOrganization,
		PermInviteMembers,
		PermRemoveMembers,
		PermUpdateMemberRoles,
		PermViewAuditLogs,
		PermViewConnections,
		PermCreateConnections,
		PermUpdateConnections,
		PermDeleteConnections,
		PermViewQueries,
		PermCreateQueries,
		PermUpdateQueries,
		PermDeleteQueries,
	},
	RoleMember: {
		// Members can view and create their own resources
		PermViewOrganization,
		PermViewConnections,
		PermCreateConnections,
		PermViewQueries,
		PermCreateQueries,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role OrganizationRole, perm Permission) bool {
	perms, ok := RolePermissions[role]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// CanUpdateResource checks if a user can update a resource they don't own
// Owners and admins can update any resource
// Members can only update their own resources
func CanUpdateResource(role OrganizationRole, resourceOwnerID, userID string) bool {
	// Owners and admins can update any resource
	if role == RoleOwner || role == RoleAdmin {
		return true
	}
	// Members can only update their own resources
	return resourceOwnerID == userID
}

// CanDeleteResource checks if a user can delete a resource
// Same logic as update - owners and admins can delete any resource
// Members can only delete their own resources
func CanDeleteResource(role OrganizationRole, resourceOwnerID, userID string) bool {
	return CanUpdateResource(role, resourceOwnerID, userID)
}

// GetPermissionsForRole returns all permissions for a given role
func GetPermissionsForRole(role OrganizationRole) []Permission {
	perms, ok := RolePermissions[role]
	if !ok {
		return []Permission{}
	}
	// Return a copy to prevent modification
	result := make([]Permission, len(perms))
	copy(result, perms)
	return result
}

// IsValidPermission checks if a permission string is valid
func IsValidPermission(perm Permission) bool {
	validPerms := []Permission{
		PermViewOrganization,
		PermUpdateOrganization,
		PermDeleteOrganization,
		PermInviteMembers,
		PermRemoveMembers,
		PermUpdateMemberRoles,
		PermViewAuditLogs,
		PermViewConnections,
		PermCreateConnections,
		PermUpdateConnections,
		PermDeleteConnections,
		PermViewQueries,
		PermCreateQueries,
		PermUpdateQueries,
		PermDeleteQueries,
	}
	for _, valid := range validPerms {
		if perm == valid {
			return true
		}
	}
	return false
}
