/**
 * PermissionGate Component
 *
 * Conditionally renders children based on user permissions.
 * Provides a clean way to show/hide UI elements based on RBAC.
 *
 * Usage:
 * ```typescript
 * // Show button only if user has permission
 * <PermissionGate permission="members:invite">
 *   <Button>Invite Member</Button>
 * </PermissionGate>
 *
 * // Show fallback message if no permission
 * <PermissionGate
 *   permission="org:delete"
 *   fallback={<p>Only owners can delete organizations</p>}
 * >
 *   <Button variant="destructive">Delete Organization</Button>
 * </PermissionGate>
 *
 * // Custom tooltip for disabled state
 * <PermissionGate
 *   permission="members:remove"
 *   showTooltip
 *   tooltipSide="top"
 * >
 *   <Button>Remove Member</Button>
 * </PermissionGate>
 * ```
 */

import * as React from 'react'
import { usePermissions, type Permission } from '@/hooks/usePermissions'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface PermissionGateProps {
  /** Permission required to show children */
  permission: Permission

  /** Organization ID (defaults to current organization) */
  orgId?: string | null

  /** Content to show if permission is denied */
  fallback?: React.ReactNode

  /** Children to render if permission is granted */
  children: React.ReactNode

  /** If true, wraps children in tooltip when permission denied */
  showTooltip?: boolean

  /** Custom tooltip message (defaults to permission description) */
  tooltipMessage?: string

  /** Tooltip placement */
  tooltipSide?: 'top' | 'bottom' | 'left' | 'right'
}

/**
 * PermissionGate - Show/hide content based on permissions
 */
export function PermissionGate({
  permission,
  orgId,
  fallback,
  children,
  showTooltip = false,
  tooltipMessage,
  tooltipSide = 'top',
}: PermissionGateProps) {
  const { hasPermission, getPermissionTooltip } = usePermissions(orgId)

  const hasAccess = hasPermission(permission)

  // If user has permission, render children normally
  if (hasAccess) {
    return <>{children}</>
  }

  // If no permission and tooltip requested, show disabled with tooltip
  if (showTooltip) {
    const message = tooltipMessage || getPermissionTooltip(permission)

    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            {/* Clone children and add disabled prop if it's a button/input */}
            <div className="inline-flex">
              {React.Children.map(children, (child) => {
                if (React.isValidElement(child)) {
                  return React.cloneElement(child as React.ReactElement<any>, {
                    disabled: true,
                    'aria-label': message,
                  })
                }
                return child
              })}
            </div>
          </TooltipTrigger>
          <TooltipContent side={tooltipSide}>
            {message}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  // If fallback provided, show it
  if (fallback) {
    return <>{fallback}</>
  }

  // Otherwise, render nothing
  return null
}

/**
 * PermissionButton - Button that's automatically disabled without permission
 *
 * Usage:
 * ```typescript
 * <PermissionButton permission="members:invite" onClick={handleInvite}>
 *   Invite Member
 * </PermissionButton>
 * ```
 */
interface PermissionButtonProps
  extends React.ComponentPropsWithoutRef<'button'> {
  permission: Permission
  orgId?: string | null
  tooltipSide?: 'top' | 'bottom' | 'left' | 'right'
}

export function PermissionButton({
  permission,
  orgId,
  tooltipSide = 'top',
  children,
  disabled,
  ...props
}: PermissionButtonProps) {
  const { hasPermission, getPermissionTooltip } = usePermissions(orgId)

  const hasAccess = hasPermission(permission)
  const isDisabled = disabled || !hasAccess

  if (!hasAccess) {
    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <button {...props} disabled={true}>
              {children}
            </button>
          </TooltipTrigger>
          <TooltipContent side={tooltipSide}>
            {getPermissionTooltip(permission)}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  return (
    <button {...props} disabled={isDisabled}>
      {children}
    </button>
  )
}

/**
 * MultiPermissionGate - Requires multiple permissions (AND logic)
 *
 * Usage:
 * ```typescript
 * <MultiPermissionGate permissions={['org:update', 'members:invite']}>
 *   <AdvancedSettings />
 * </MultiPermissionGate>
 * ```
 */
interface MultiPermissionGateProps {
  permissions: Permission[]
  orgId?: string | null
  fallback?: React.ReactNode
  children: React.ReactNode
  /** If true, requires ANY permission (OR logic). Default is ALL (AND logic) */
  requireAny?: boolean
}

export function MultiPermissionGate({
  permissions,
  orgId,
  fallback,
  children,
  requireAny = false,
}: MultiPermissionGateProps) {
  const { hasPermission } = usePermissions(orgId)

  const hasAccess = requireAny
    ? permissions.some((perm) => hasPermission(perm))
    : permissions.every((perm) => hasPermission(perm))

  if (hasAccess) {
    return <>{children}</>
  }

  return fallback ? <>{fallback}</> : null
}

/**
 * RoleGate - Show content based on role level
 *
 * Usage:
 * ```typescript
 * // Show only to owners
 * <RoleGate minRole="owner">
 *   <DangerZone />
 * </RoleGate>
 *
 * // Show to admins and owners
 * <RoleGate minRole="admin">
 *   <MemberManagement />
 * </RoleGate>
 * ```
 */
interface RoleGateProps {
  minRole: 'owner' | 'admin' | 'member'
  orgId?: string | null
  fallback?: React.ReactNode
  children: React.ReactNode
}

export function RoleGate({
  minRole,
  orgId,
  fallback,
  children,
}: RoleGateProps) {
  const { userRole } = usePermissions(orgId)

  const roleHierarchy = {
    owner: 3,
    admin: 2,
    member: 1,
  }

  const hasAccess =
    userRole && roleHierarchy[userRole] >= roleHierarchy[minRole]

  if (hasAccess) {
    return <>{children}</>
  }

  return fallback ? <>{fallback}</> : null
}
