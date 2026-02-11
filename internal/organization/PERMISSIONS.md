# Organization Permission System

This document describes the comprehensive Role-Based Access Control (RBAC) system implemented for the organization module.

## Overview

The permission system provides fine-grained access control for organization operations, member management, and future resource access (connections, queries).

## Roles

The system defines three hierarchical roles:

### Owner
- Full access to all organization operations
- Can delete the organization
- Can perform all admin and member operations
- Automatically assigned to organization creator

### Admin
- Can manage organization settings (but cannot delete)
- Can invite, remove, and manage member roles
- Can view audit logs
- Can create, update, and delete connections and queries
- Cannot delete the organization

### Member
- Can view organization
- Can create their own connections and queries
- Can view connections and queries
- Cannot update/delete resources they don't own
- Cannot manage members or view audit logs

## Permission Matrix

| Permission | Owner | Admin | Member |
|------------|-------|-------|--------|
| `org:view` | ✓ | ✓ | ✓ |
| `org:update` | ✓ | ✓ | ✗ |
| `org:delete` | ✓ | ✗ | ✗ |
| `members:invite` | ✓ | ✓ | ✗ |
| `members:remove` | ✓ | ✓ | ✗ |
| `members:update_roles` | ✓ | ✓ | ✗ |
| `audit:view` | ✓ | ✓ | ✗ |
| `connections:view` | ✓ | ✓ | ✓ |
| `connections:create` | ✓ | ✓ | ✓ |
| `connections:update` | ✓ | ✓ | ✗* |
| `connections:delete` | ✓ | ✓ | ✗* |
| `queries:view` | ✓ | ✓ | ✓ |
| `queries:create` | ✓ | ✓ | ✓ |
| `queries:update` | ✓ | ✓ | ✗* |
| `queries:delete` | ✓ | ✓ | ✗* |

**Note:** Members can update/delete their own resources even if the permission shows ✗. Use `CanUpdateResource()` or `CanDeleteResource()` for resource-level checks.

## Implementation

### Permission Checking

All service methods check permissions using the `HasPermission()` function:

```go
member, err := s.repo.GetMember(ctx, orgID, userID)
if err != nil || member == nil {
    return nil, fmt.Errorf("not a member of this organization")
}

if !HasPermission(member.Role, PermUpdateOrganization) {
    // Log permission denial
    s.CreateAuditLog(ctx, &AuditLog{
        OrganizationID: &orgID,
        UserID:         userID,
        Action:         "permission_denied",
        ResourceType:   "organization",
        ResourceID:     &orgID,
        Details: map[string]interface{}{
            "permission": string(PermUpdateOrganization),
            "role":       string(member.Role),
            "attempted":  "update_organization",
        },
    })
    return nil, fmt.Errorf("insufficient permissions")
}
```

### Resource Ownership

For operations on resources (connections, queries), use resource ownership checks:

```go
// Check if user can update a resource
canUpdate := CanUpdateResource(member.Role, resource.OwnerID, userID)

// Check if user can delete a resource
canDelete := CanDeleteResource(member.Role, resource.OwnerID, userID)
```

**Rules:**
- Owners and Admins can update/delete any resource
- Members can only update/delete their own resources

### Audit Logging

All permission denials are automatically logged to the audit log with:
- The denied permission
- User's current role
- Attempted action
- Resource information (if applicable)

This provides complete visibility into security events.

## Usage Examples

### Check if user can invite members

```go
if HasPermission(member.Role, PermInviteMembers) {
    // User can invite members
}
```

### Check if user can update a connection

```go
// First check if they have the permission
if !HasPermission(member.Role, PermUpdateConnections) {
    // Then check if it's their own resource
    if !CanUpdateResource(member.Role, connection.OwnerID, userID) {
        return fmt.Errorf("insufficient permissions")
    }
}
```

### Get all permissions for a role

```go
permissions := GetPermissionsForRole(RoleAdmin)
// Returns: [PermViewOrganization, PermUpdateOrganization, ...]
```

### Validate a permission string

```go
if IsValidPermission(Permission("org:update")) {
    // Valid permission
}
```

## Service Methods Protected

The following service methods are protected by permission checks:

### Organization Operations
- `UpdateOrganization()` - Requires `PermUpdateOrganization`
- `DeleteOrganization()` - Requires `PermDeleteOrganization`

### Member Management
- `UpdateMemberRole()` - Requires `PermUpdateMemberRoles`
- `RemoveMember()` - Requires `PermRemoveMembers`

### Invitation Management
- `CreateInvitation()` - Requires `PermInviteMembers`
- `GetInvitations()` - Requires `PermInviteMembers`
- `RevokeInvitation()` - Requires `PermInviteMembers`

### Audit Logs
- `GetAuditLogs()` - Requires `PermViewAuditLogs`

## Testing

Comprehensive test coverage is provided in `permissions_test.go`:

- Permission matrix tests for all roles
- Resource ownership tests
- Permission utility tests
- Scenario-based tests

Run tests:
```bash
go test ./internal/organization -run Permission -v
```

## Special Rules

### Role Assignment Restrictions

1. **Cannot change owner's role**
   - The organization owner's role is immutable
   - Enforced in `UpdateMemberRole()`

2. **Admins cannot promote to owner**
   - Only owners can assign the owner role
   - Prevents unauthorized privilege escalation

3. **Admins can only invite members**
   - Admins cannot invite other admins
   - Only owners can invite admins

### Member Removal Restrictions

1. **Cannot remove owner**
   - Owner cannot be removed from organization
   - Organization must be deleted to remove owner

2. **Admins can only remove members**
   - Admins cannot remove other admins or owners
   - Prevents lateral privilege attacks

## Future Enhancements

The permission system is designed to be extensible. Future additions might include:

1. **Custom permissions per organization**
   - Allow organizations to define custom roles
   - Assign specific permissions to custom roles

2. **Resource-level permissions**
   - Fine-grained permissions per connection/query
   - Sharing resources with specific members

3. **Time-based permissions**
   - Temporary elevated access
   - Scheduled permission changes

4. **Permission inheritance**
   - Hierarchical resource structures
   - Inherited permissions from parent resources

## Migration Guide

If you're migrating from the old permission system:

1. Replace all `checkPermission()` calls with the new pattern:
   ```go
   // Old
   if err := s.checkPermission(ctx, orgID, userID, []OrganizationRole{RoleOwner, RoleAdmin}); err != nil {
       return err
   }

   // New
   member, err := s.repo.GetMember(ctx, orgID, userID)
   if err != nil || member == nil {
       return fmt.Errorf("not a member of this organization")
   }
   if !HasPermission(member.Role, PermYourPermission) {
       // Log and return error
   }
   ```

2. Add audit logging for permission denials
3. Update tests to use new permission functions
