/**
 * usePermissions Hook
 *
 * Provides role-based permission checking for organization features.
 * Permission matrix matches backend RBAC implementation exactly.
 *
 * Usage:
 * ```typescript
 * const { hasPermission, isOwner, canUpdateResource } = usePermissions(orgId)
 *
 * if (hasPermission('members:invite')) {
 *   // Show invite button
 * }
 *
 * if (canUpdateResource(connection.owner_id, currentUserId)) {
 *   // Allow editing connection
 * }
 * ```
 */

import { useOrganizationStore } from '@/store/organization-store'
import { useAuthStore } from '@/store/auth-store'
import { OrganizationRole } from '@/types/organization'

/**
 * Granular permission types matching backend permission matrix
 */
export type Permission =
  // Organization permissions
  | 'org:view'
  | 'org:update'
  | 'org:delete'
  // Member management
  | 'members:invite'
  | 'members:remove'
  | 'members:update_roles'
  // Audit logs
  | 'audit:view'
  // Connection management
  | 'connections:view'
  | 'connections:create'
  | 'connections:update'
  | 'connections:delete'
  // Query management
  | 'queries:view'
  | 'queries:create'
  | 'queries:update'
  | 'queries:delete'

/**
 * Permission matrix - defines what each role can do
 * MUST match backend permission matrix exactly
 */
const rolePermissions: Record<OrganizationRole, Permission[]> = {
  [OrganizationRole.Owner]: [
    // Organization
    'org:view',
    'org:update',
    'org:delete',
    // Members
    'members:invite',
    'members:remove',
    'members:update_roles',
    // Audit
    'audit:view',
    // Connections
    'connections:view',
    'connections:create',
    'connections:update',
    'connections:delete',
    // Queries
    'queries:view',
    'queries:create',
    'queries:update',
    'queries:delete',
  ],
  [OrganizationRole.Admin]: [
    // Organization
    'org:view',
    'org:update',
    // Members
    'members:invite',
    'members:remove',
    'members:update_roles',
    // Audit
    'audit:view',
    // Connections
    'connections:view',
    'connections:create',
    'connections:update',
    'connections:delete',
    // Queries
    'queries:view',
    'queries:create',
    'queries:update',
    'queries:delete',
  ],
  [OrganizationRole.Member]: [
    // Organization
    'org:view',
    // Connections
    'connections:view',
    'connections:create',
    // Queries
    'queries:view',
    'queries:create',
  ],
}

/**
 * Permission descriptions for tooltip/help text
 */
export const permissionDescriptions: Record<Permission, string> = {
  'org:view': 'View organization details',
  'org:update': 'Edit organization settings',
  'org:delete': 'Delete the organization',
  'members:invite': 'Invite new members',
  'members:remove': 'Remove members from organization',
  'members:update_roles': 'Change member roles',
  'audit:view': 'View audit logs',
  'connections:view': 'View database connections',
  'connections:create': 'Create new connections',
  'connections:update': 'Edit existing connections',
  'connections:delete': 'Delete connections',
  'queries:view': 'View queries',
  'queries:create': 'Create and save queries',
  'queries:update': 'Edit saved queries',
  'queries:delete': 'Delete queries',
}

export interface UsePermissionsReturn {
  /** Check if user has a specific permission */
  hasPermission: (permission: Permission) => boolean

  /** Check if user can update a specific resource (based on ownership) */
  canUpdateResource: (resourceOwnerId: string, userId: string) => boolean

  /** Check if user can delete a specific resource (based on ownership) */
  canDeleteResource: (resourceOwnerId: string, userId: string) => boolean

  /** Get tooltip text for a disabled permission */
  getPermissionTooltip: (permission: Permission) => string

  /** Current user's role in the organization */
  userRole: OrganizationRole | null

  /** Convenience flags */
  isOwner: boolean
  isAdmin: boolean
  isMember: boolean
}

/**
 * Main permissions hook
 *
 * @param orgId - Organization ID to check permissions for (defaults to current org)
 */
export function usePermissions(
  orgId?: string | null
): UsePermissionsReturn {
  const { organizations, currentOrgId, currentOrgMembers } = useOrganizationStore()
  const user = useAuthStore((state) => state.user)

  // Determine which org to check permissions for
  const effectiveOrgId = orgId !== undefined ? orgId : currentOrgId

  // Find the organization
  const org = organizations.find((o) => o.id === effectiveOrgId)

  // Find user's membership in the organization
  const membership = currentOrgMembers.find((m) => m.user_id === user?.id)
  const userRole = membership?.role || null

  /**
   * Check if user has a specific permission
   */
  const hasPermission = (permission: Permission): boolean => {
    if (!userRole || !org) return false
    const perms = rolePermissions[userRole]
    return perms.includes(permission)
  }

  /**
   * Check if user can update a resource
   * Owners and Admins can update any resource
   * Members can only update their own resources
   */
  const canUpdateResource = (
    resourceOwnerId: string,
    userId: string
  ): boolean => {
    if (!userRole) return false

    // Owners and Admins can update any resource
    if (
      userRole === OrganizationRole.Owner ||
      userRole === OrganizationRole.Admin
    ) {
      return true
    }

    // Members can only update their own resources
    return resourceOwnerId === userId
  }

  /**
   * Check if user can delete a resource
   * Same logic as update
   */
  const canDeleteResource = (
    resourceOwnerId: string,
    userId: string
  ): boolean => {
    return canUpdateResource(resourceOwnerId, userId)
  }

  /**
   * Get tooltip text explaining why a permission is denied
   */
  const getPermissionTooltip = (permission: Permission): string => {
    if (hasPermission(permission)) {
      return permissionDescriptions[permission]
    }

    if (!userRole) {
      return 'You must be a member of this organization'
    }

    const roleName = userRole.charAt(0).toUpperCase() + userRole.slice(1)
    return `${roleName}s cannot ${permissionDescriptions[permission].toLowerCase()}`
  }

  return {
    hasPermission,
    canUpdateResource,
    canDeleteResource,
    getPermissionTooltip,
    userRole,
    isOwner: userRole === OrganizationRole.Owner,
    isAdmin: userRole === OrganizationRole.Admin,
    isMember: userRole === OrganizationRole.Member,
  }
}

/**
 * Utility hook for simple permission checks
 *
 * Usage:
 * ```typescript
 * const canInvite = useCanPerform('members:invite')
 * ```
 */
export function useCanPerform(
  permission: Permission,
  orgId?: string | null
): boolean {
  const { hasPermission } = usePermissions(orgId)
  return hasPermission(permission)
}

/**
 * Hook for checking multiple permissions at once
 *
 * Usage:
 * ```typescript
 * const permissions = usePermissionCheck(['members:invite', 'members:remove'])
 * if (permissions.canInvite && permissions.canRemove) {
 *   // ...
 * }
 * ```
 */
export function usePermissionCheck(
  permissions: Permission[],
  orgId?: string | null
): Record<string, boolean> {
  const { hasPermission } = usePermissions(orgId)

  return permissions.reduce(
    (acc, perm) => {
      const key = perm.split(':')[1] || perm
      acc[`can${key.charAt(0).toUpperCase()}${key.slice(1)}`] =
        hasPermission(perm)
      return acc
    },
    {} as Record<string, boolean>
  )
}
