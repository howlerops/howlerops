/**
 * RoleManagement Component
 *
 * Inline role management for organization members.
 * Provides dropdown for changing roles with confirmation dialog.
 * Supports optimistic updates with automatic rollback on errors.
 */

import { AlertCircle,Check, Loader2, Shield } from 'lucide-react'
import * as React from 'react'
import { toast } from 'sonner'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { usePermissions } from '@/hooks/usePermissions'
import { cn } from '@/lib/utils'
import type { OrganizationMember } from '@/types/organization'
import { getRoleDisplayName,OrganizationRole } from '@/types/organization'

interface RoleManagementProps {
  member: OrganizationMember
  currentUserRole: OrganizationRole
  onRoleChange: (memberId: string, newRole: OrganizationRole) => Promise<void>
  disabled?: boolean
  className?: string
}

/**
 * RoleManagement - Dropdown for changing member roles
 *
 * Features:
 * - Only shows roles that current user can assign
 * - Owners can assign any role
 * - Admins can assign Admin or Member (not Owner)
 * - Shows confirmation dialog for role changes
 * - Optimistic updates with rollback
 * - Loading states and error handling
 */
export function RoleManagement({
  member,
  currentUserRole,
  onRoleChange,
  disabled = false,
  className,
}: RoleManagementProps) {
  const [showConfirmDialog, setShowConfirmDialog] = React.useState(false)
  const [pendingRole, setPendingRole] = React.useState<OrganizationRole | null>(
    null
  )
  const [isChanging, setIsChanging] = React.useState(false)
  const [error, setError] = React.useState<string | null>(null)

  const { hasPermission } = usePermissions()
  const canChangeRoles = hasPermission('members:update_roles')

  // Determine which roles can be assigned by current user
  const availableRoles = React.useMemo(() => {
    const roles: OrganizationRole[] = []

    if (currentUserRole === OrganizationRole.Owner) {
      // Owners can assign any role
      roles.push(
        OrganizationRole.Owner,
        OrganizationRole.Admin,
        OrganizationRole.Member
      )
    } else if (currentUserRole === OrganizationRole.Admin) {
      // Admins can assign Admin or Member (not Owner)
      roles.push(OrganizationRole.Admin, OrganizationRole.Member)
    }

    return roles
  }, [currentUserRole])

  const handleRoleSelect = (newRole: string) => {
    const role = newRole as OrganizationRole

    // Don't show confirmation if role hasn't actually changed
    if (role === member.role) {
      return
    }

    setPendingRole(role)
    setShowConfirmDialog(true)
  }

  const handleConfirmRoleChange = async () => {
    if (!pendingRole) return

    setIsChanging(true)
    setError(null)

    try {
      await onRoleChange(member.id, pendingRole)

      toast.success('Role updated', {
        description: `${member.user?.display_name || member.user?.username || member.user?.email} is now a ${getRoleDisplayName(pendingRole)}`,
      })

      setShowConfirmDialog(false)
      setPendingRole(null)
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Failed to update role'
      setError(errorMessage)

      toast.error('Failed to update role', {
        description: errorMessage,
      })
    } finally {
      setIsChanging(false)
    }
  }

  const handleCancelRoleChange = () => {
    setShowConfirmDialog(false)
    setPendingRole(null)
    setError(null)
  }

  // If user can't change roles, show badge instead of dropdown
  if (!canChangeRoles || disabled || availableRoles.length === 0) {
    const isCurrentUserOwner = currentUserRole === OrganizationRole.Owner
    const isMemberOwner = member.role === OrganizationRole.Owner

    const tooltipMessage = disabled
      ? 'Role changes are currently disabled'
      : !canChangeRoles
        ? 'You do not have permission to change roles'
        : isMemberOwner && !isCurrentUserOwner
          ? 'Only the owner can change the owner role'
          : 'You cannot change this member\'s role'

    return (
      <TooltipProvider>
        <Tooltip>
          <TooltipTrigger asChild>
            <div className={cn('inline-flex', className)}>
              <Badge
                variant={
                  member.role === OrganizationRole.Owner
                    ? 'default'
                    : member.role === OrganizationRole.Admin
                      ? 'secondary'
                      : 'outline'
                }
                className="cursor-not-allowed"
              >
                <Shield className="h-3 w-3 mr-1" />
                {getRoleDisplayName(member.role)}
              </Badge>
            </div>
          </TooltipTrigger>
          <TooltipContent>{tooltipMessage}</TooltipContent>
        </Tooltip>
      </TooltipProvider>
    )
  }

  const memberName =
    member.user?.display_name ||
    member.user?.username ||
    member.user?.email ||
    'Unknown User'

  return (
    <>
      <Select
        value={member.role}
        onValueChange={handleRoleSelect}
        disabled={isChanging}
      >
        <SelectTrigger className={cn('w-[140px]', className)}>
          <div className="flex items-center gap-2">
            <Shield className="h-3.5 w-3.5" />
            <SelectValue />
          </div>
        </SelectTrigger>
        <SelectContent>
          {availableRoles.map((role) => (
            <SelectItem key={role} value={role}>
              <div className="flex flex-col items-start">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{getRoleDisplayName(role)}</span>
                  {role === member.role && (
                    <Check className="h-3.5 w-3.5 text-primary" />
                  )}
                </div>
                <span className="text-xs text-muted-foreground">
                  {getRoleDescription(role)}
                </span>
              </div>
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      {/* Role Change Confirmation Dialog */}
      <Dialog open={showConfirmDialog} onOpenChange={handleCancelRoleChange}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Change Member Role</DialogTitle>
            <DialogDescription>
              Are you sure you want to change <strong>{memberName}</strong>'s
              role?
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-4">
            <div className="flex items-center justify-between p-3 rounded-lg border bg-muted/50">
              <span className="text-sm text-muted-foreground">Current Role</span>
              <Badge
                variant={
                  member.role === OrganizationRole.Owner
                    ? 'default'
                    : member.role === OrganizationRole.Admin
                      ? 'secondary'
                      : 'outline'
                }
              >
                {getRoleDisplayName(member.role)}
              </Badge>
            </div>

            <div className="flex items-center justify-center">
              <svg
                className="h-4 w-4 text-muted-foreground"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M19 14l-7 7m0 0l-7-7m7 7V3"
                />
              </svg>
            </div>

            <div className="flex items-center justify-between p-3 rounded-lg border bg-primary/5">
              <span className="text-sm font-medium">New Role</span>
              <Badge
                variant={
                  pendingRole === OrganizationRole.Owner
                    ? 'default'
                    : pendingRole === OrganizationRole.Admin
                      ? 'secondary'
                      : 'outline'
                }
              >
                {pendingRole && getRoleDisplayName(pendingRole)}
              </Badge>
            </div>

            {pendingRole === OrganizationRole.Owner && (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <strong>Important:</strong> Promoting this member to Owner will
                  give them full control of the organization, including the ability
                  to delete it or transfer ownership.
                </AlertDescription>
              </Alert>
            )}

            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </div>

          <DialogFooter>
            <Button
              variant="outline"
              onClick={handleCancelRoleChange}
              disabled={isChanging}
            >
              Cancel
            </Button>
            <Button onClick={handleConfirmRoleChange} disabled={isChanging}>
              {isChanging ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Updating...
                </>
              ) : (
                <>
                  <Check className="h-4 w-4 mr-2" />
                  Confirm Change
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}

/**
 * Get human-readable role description
 */
function getRoleDescription(role: OrganizationRole): string {
  switch (role) {
    case OrganizationRole.Owner:
      return 'Full control'
    case OrganizationRole.Admin:
      return 'Manage members & settings'
    case OrganizationRole.Member:
      return 'Basic access'
    default:
      return ''
  }
}
