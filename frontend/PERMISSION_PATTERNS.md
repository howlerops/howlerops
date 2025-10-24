# Permission System Documentation

## Overview

Sprint 3 implements a comprehensive Role-Based Access Control (RBAC) system for the frontend. This document describes the permission patterns, components, and best practices.

## Architecture

### Permission Matrix

The permission system uses a strict matrix that **must match the backend RBAC exactly**:

| Permission | Owner | Admin | Member |
|-----------|-------|-------|--------|
| `org:view` | ✅ | ✅ | ✅ |
| `org:update` | ✅ | ✅ | ❌ |
| `org:delete` | ✅ | ❌ | ❌ |
| `members:invite` | ✅ | ✅ | ❌ |
| `members:remove` | ✅ | ✅ | ❌ |
| `members:update_roles` | ✅ | ✅ | ❌ |
| `audit:view` | ✅ | ✅ | ❌ |
| `connections:view` | ✅ | ✅ | ✅ |
| `connections:create` | ✅ | ✅ | ✅ |
| `connections:update` | ✅ | ✅ | Owner only* |
| `connections:delete` | ✅ | ✅ | Owner only* |
| `queries:view` | ✅ | ✅ | ✅ |
| `queries:create` | ✅ | ✅ | ✅ |
| `queries:update` | ✅ | ✅ | Owner only* |
| `queries:delete` | ✅ | ✅ | Owner only* |

*Members can only update/delete their own resources

### Role Hierarchy

```
Owner (Level 3)
  ├─ Full control of organization
  ├─ Can delete organization
  ├─ Can transfer ownership
  └─ Can promote members to owner

Admin (Level 2)
  ├─ Can manage members and settings
  ├─ Can view audit logs
  └─ Cannot delete organization or transfer ownership

Member (Level 1)
  ├─ Basic access to resources
  ├─ Can create connections and queries
  └─ Can only edit their own resources
```

## Core Components

### 1. usePermissions Hook

Location: `/frontend/src/hooks/usePermissions.ts`

**Usage:**
```typescript
import { usePermissions } from '@/hooks/usePermissions'

function MyComponent() {
  const { hasPermission, isOwner, canUpdateResource } = usePermissions()

  // Check specific permission
  if (hasPermission('members:invite')) {
    // Show invite button
  }

  // Check resource ownership
  if (canUpdateResource(connection.owner_id, currentUserId)) {
    // Allow editing
  }

  // Role checks
  if (isOwner) {
    // Show owner-only features
  }
}
```

**Available Methods:**
- `hasPermission(permission: Permission): boolean` - Check if user has a specific permission
- `canUpdateResource(resourceOwnerId: string, userId: string): boolean` - Check if user can update a resource
- `canDeleteResource(resourceOwnerId: string, userId: string): boolean` - Check if user can delete a resource
- `getPermissionTooltip(permission: Permission): string` - Get tooltip text for denied permission
- `userRole: OrganizationRole | null` - Current user's role
- `isOwner: boolean` - Convenience flag for owner check
- `isAdmin: boolean` - Convenience flag for admin check
- `isMember: boolean` - Convenience flag for member check

### 2. PermissionGate Component

Location: `/frontend/src/components/PermissionGate.tsx`

**Basic Usage:**
```typescript
import { PermissionGate } from '@/components/PermissionGate'

// Show content only if user has permission
<PermissionGate permission="members:invite">
  <Button>Invite Member</Button>
</PermissionGate>

// Show fallback if permission denied
<PermissionGate
  permission="org:delete"
  fallback={<p>Only owners can delete</p>}
>
  <Button variant="destructive">Delete</Button>
</PermissionGate>

// Show disabled button with tooltip
<PermissionGate
  permission="members:remove"
  showTooltip
  tooltipSide="top"
>
  <Button>Remove Member</Button>
</PermissionGate>
```

**Available Components:**

1. **PermissionGate** - Conditional rendering based on single permission
2. **MultiPermissionGate** - Requires multiple permissions (AND/OR logic)
3. **RoleGate** - Show content based on minimum role level
4. **PermissionButton** - Button with automatic permission-based disabling

### 3. RoleManagement Component

Location: `/frontend/src/components/organizations/RoleManagement.tsx`

Provides inline role management with:
- Dropdown for changing roles
- Confirmation dialog with role change preview
- Automatic permission checks
- Optimistic updates with rollback
- Loading states

**Usage:**
```typescript
import { RoleManagement } from '@/components/organizations/RoleManagement'

<RoleManagement
  member={member}
  currentUserRole={currentUserRole}
  onRoleChange={handleRoleChange}
/>
```

### 4. TransferOwnershipModal

Location: `/frontend/src/components/organizations/TransferOwnershipModal.tsx`

Owner-only modal for transferring organization ownership:
- Only shows admins as eligible recipients
- Requires password confirmation
- Shows clear warnings about consequences
- Includes acknowledgement checkbox

### 5. AuditLogViewer

Location: `/frontend/src/components/organizations/AuditLogViewer.tsx`

Audit log viewer for owners and admins:
- Filter by action type, user, date range
- Pagination (50 entries per page)
- Auto-refresh every 30 seconds
- Export to CSV
- Expandable rows for IP/user agent details

### 6. OrganizationSettingsPage

Location: `/frontend/src/components/organizations/OrganizationSettingsPage.tsx`

Comprehensive settings page with tabs:
- **General**: Organization info and settings
- **Members**: Team member management
- **Audit Logs**: Activity history (owner/admin only)
- **Danger Zone**: Transfer ownership, delete org (owner only)

## Implementation Patterns

### Pattern 1: Simple Permission Check

```typescript
const { hasPermission } = usePermissions()

return (
  <div>
    {hasPermission('members:invite') && (
      <Button onClick={handleInvite}>Invite</Button>
    )}
  </div>
)
```

### Pattern 2: Permission Gate with Tooltip

```typescript
<PermissionGate
  permission="members:invite"
  showTooltip
  tooltipSide="left"
>
  <Button onClick={handleInvite}>
    <UserPlus className="h-4 w-4 mr-2" />
    Invite Member
  </Button>
</PermissionGate>
```

### Pattern 3: Resource Ownership Check

```typescript
const { canUpdateResource } = usePermissions()
const currentUser = useAuthStore((state) => state.user)

const canEdit = canUpdateResource(connection.owner_id, currentUser?.id || '')

return (
  <Button
    onClick={handleEdit}
    disabled={!canEdit}
  >
    Edit
  </Button>
)
```

### Pattern 4: Role-Based Rendering

```typescript
import { RoleGate } from '@/components/PermissionGate'

<RoleGate minRole="admin">
  <AdminPanel />
</RoleGate>

<RoleGate minRole="owner">
  <DangerZone />
</RoleGate>
```

### Pattern 5: Multiple Permission Check

```typescript
import { MultiPermissionGate } from '@/components/PermissionGate'

// Require ALL permissions (AND logic)
<MultiPermissionGate permissions={['org:update', 'members:invite']}>
  <AdvancedSettings />
</MultiPermissionGate>

// Require ANY permission (OR logic)
<MultiPermissionGate
  permissions={['org:update', 'org:delete']}
  requireAny
>
  <SettingsPanel />
</MultiPermissionGate>
```

## Best Practices

### 1. Always Use Permission Checks

**Bad:**
```typescript
// Don't rely on role checks alone
{currentUserRole === 'owner' && <DeleteButton />}
```

**Good:**
```typescript
// Use permission-based checks
<PermissionGate permission="org:delete">
  <DeleteButton />
</PermissionGate>
```

### 2. Provide User Feedback

**Bad:**
```typescript
// Don't hide UI without explanation
{hasPermission('members:invite') && <InviteButton />}
```

**Good:**
```typescript
// Show disabled state with tooltip
<PermissionGate permission="members:invite" showTooltip>
  <InviteButton />
</PermissionGate>
```

### 3. Handle Resource Ownership

**Bad:**
```typescript
// Don't allow editing any resource
<Button onClick={handleEdit}>Edit</Button>
```

**Good:**
```typescript
// Check ownership for member-owned resources
const canEdit = canUpdateResource(resource.owner_id, currentUserId)
<Button onClick={handleEdit} disabled={!canEdit}>
  Edit
</Button>
```

### 4. Use Optimistic Updates

**Pattern:**
```typescript
const handleRoleChange = async (memberId: string, newRole: OrganizationRole) => {
  const originalRole = member.role

  // Optimistic update
  setMemberRole(newRole)

  try {
    await onRoleChange(memberId, { role: newRole })
    toast.success('Role updated')
  } catch (error) {
    // Rollback on error
    setMemberRole(originalRole)
    toast.error('Failed to update role')
  }
}
```

### 5. Confirmation for Destructive Actions

Always require confirmation for:
- Role changes
- Member removal
- Organization deletion
- Ownership transfer

```typescript
<Dialog open={showConfirm} onOpenChange={setShowConfirm}>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>Confirm Action</DialogTitle>
      <DialogDescription>
        Are you sure you want to {action}?
      </DialogDescription>
    </DialogHeader>
    <DialogFooter>
      <Button variant="outline" onClick={handleCancel}>
        Cancel
      </Button>
      <Button variant="destructive" onClick={handleConfirm}>
        Confirm
      </Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

## Accessibility

All permission-controlled UI must follow these accessibility guidelines:

### 1. Disabled Elements

```typescript
<Button
  disabled={!hasPermission('members:invite')}
  aria-label={
    hasPermission('members:invite')
      ? 'Invite member'
      : 'You do not have permission to invite members'
  }
>
  Invite
</Button>
```

### 2. Tooltips

```typescript
<TooltipProvider>
  <Tooltip>
    <TooltipTrigger asChild>
      <Button disabled={!canEdit}>Edit</Button>
    </TooltipTrigger>
    <TooltipContent>
      {canEdit
        ? 'Edit this resource'
        : 'You can only edit your own resources'}
    </TooltipContent>
  </Tooltip>
</TooltipProvider>
```

### 3. Keyboard Navigation

Ensure all interactive elements are keyboard accessible:
- Tab navigation works correctly
- Enter/Space activate buttons
- Escape closes dialogs
- Focus management in modals

## Testing Checklist

When implementing permission-based features, verify:

- [ ] Owner can perform all actions
- [ ] Admin can perform admin-level actions
- [ ] Member can only perform basic actions
- [ ] Members can only edit their own resources
- [ ] Disabled elements show helpful tooltips
- [ ] Confirmation dialogs appear for destructive actions
- [ ] Optimistic updates rollback on errors
- [ ] Loading states display during async operations
- [ ] Error messages are user-friendly
- [ ] Keyboard navigation works correctly
- [ ] Screen reader support is adequate

## Common Issues

### Issue: Permission check returns false incorrectly

**Cause:** User's role not loaded from organization store

**Solution:**
```typescript
// Ensure organization members are loaded
React.useEffect(() => {
  if (currentOrgId) {
    fetchMembers(currentOrgId)
  }
}, [currentOrgId, fetchMembers])
```

### Issue: Tooltip doesn't show on disabled element

**Cause:** Disabled elements don't trigger pointer events

**Solution:**
```typescript
// Wrap disabled element in div
<TooltipProvider>
  <Tooltip>
    <TooltipTrigger asChild>
      <div className="inline-flex">
        <Button disabled>Action</Button>
      </div>
    </TooltipTrigger>
    <TooltipContent>Reason for disabled state</TooltipContent>
  </Tooltip>
</TooltipProvider>
```

### Issue: Permission gate always hides content

**Cause:** Organization context not set

**Solution:**
```typescript
// Ensure organization is selected
const { currentOrgId, switchOrganization } = useOrganizationStore()

if (!currentOrgId && organizations.length > 0) {
  switchOrganization(organizations[0].id)
}
```

## Migration Guide

### Updating Existing Components

1. **Add Permission Imports**
```typescript
import { usePermissions } from '@/hooks/usePermissions'
import { PermissionGate } from '@/components/PermissionGate'
```

2. **Replace Role Checks**
```typescript
// Before
const canInvite = role === 'owner' || role === 'admin'

// After
const { hasPermission } = usePermissions()
const canInvite = hasPermission('members:invite')
```

3. **Wrap UI Elements**
```typescript
// Before
{canInvite && <Button>Invite</Button>}

// After
<PermissionGate permission="members:invite" showTooltip>
  <Button>Invite</Button>
</PermissionGate>
```

4. **Add Tooltips**
```typescript
// Add helpful tooltips for disabled states
<PermissionGate
  permission="members:remove"
  showTooltip
  tooltipMessage="Only owners and admins can remove members"
>
  <Button>Remove</Button>
</PermissionGate>
```

## Related Files

- **Types**: `/frontend/src/types/organization.ts`
- **Store**: `/frontend/src/store/organization-store.ts`
- **Hooks**: `/frontend/src/hooks/usePermissions.ts`
- **Components**:
  - `/frontend/src/components/PermissionGate.tsx`
  - `/frontend/src/components/organizations/RoleManagement.tsx`
  - `/frontend/src/components/organizations/TransferOwnershipModal.tsx`
  - `/frontend/src/components/organizations/AuditLogViewer.tsx`
  - `/frontend/src/components/organizations/OrganizationSettingsPage.tsx`

## Backend Integration

The frontend permission matrix must stay in sync with the backend. See:
- Backend RBAC: `/backend-go/internal/rbac/permissions.go`
- Permission middleware: `/backend-go/internal/middleware/rbac.go`
- API documentation: `/backend-go/API_DOCUMENTATION.md`
