/**
 * Members List Component
 *
 * Table view of organization members with role management.
 * Shows member details, allows role changes, and member removal (owner/admin only).
 */

import { Loader2, Shield, Trash2, User,UserPlus } from 'lucide-react'
import * as React from 'react'

import { PermissionGate } from '@/components/PermissionGate'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { usePermissions } from '@/hooks/usePermissions'
import { cn } from '@/lib/utils'
import type {
  OrganizationMember,
  OrganizationRole,
  UpdateMemberRoleInput,
} from '@/types/organization'
import {
  canRemoveMembers,
  formatRelativeTime,
  getRoleBadgeVariant,
  getRoleDisplayName,
  OrganizationRole as OrgRole,
} from '@/types/organization'

import { RoleManagement } from './RoleManagement'

interface MembersListProps {
  members: OrganizationMember[]
  currentUserId: string
  currentUserRole: OrganizationRole
  onRoleChange: (memberId: string, data: UpdateMemberRoleInput) => Promise<void>
  onRemoveMember: (memberId: string) => Promise<void>
  onInviteClick: () => void
  loading?: boolean
  error?: string | null
  className?: string
}

export function MembersList({
  members,
  currentUserId,
  currentUserRole,
  onRoleChange,
  onRemoveMember,
  onInviteClick,
  loading = false,
  error = null,
  className,
}: MembersListProps) {
  const [removingMemberId, setRemovingMemberId] = React.useState<string | null>(null)

  const { hasPermission } = usePermissions()
  const canInvite = hasPermission('members:invite')
  const canManage = hasPermission('members:remove')

  const handleRoleChange = async (memberId: string, newRole: OrganizationRole) => {
    try {
      await onRoleChange(memberId, { role: newRole })
    } catch (err) {
      console.error('Failed to change role:', err)
    }
  }

  const handleRemove = async () => {
    if (!removingMemberId) return

    try {
      await onRemoveMember(removingMemberId)
      setRemovingMemberId(null)
    } catch (err) {
      console.error('Failed to remove member:', err)
    }
  }

  if (loading && members.length === 0) {
    return (
      <div className={cn('flex items-center justify-center py-12', className)}>
        <div className="flex flex-col items-center gap-3">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          <p className="text-sm text-muted-foreground">Loading members...</p>
        </div>
      </div>
    )
  }

  const memberToRemove = members.find((m) => m.id === removingMemberId)

  return (
    <div className={className}>
      {error && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex items-center justify-between mb-6">
        <div>
          <h2 className="text-2xl font-semibold">Team Members</h2>
          <p className="text-sm text-muted-foreground mt-1">
            {members.length} {members.length === 1 ? 'member' : 'members'}
          </p>
        </div>
        <PermissionGate
          permission="members:invite"
          showTooltip={!canInvite}
          tooltipSide="left"
        >
          <Button onClick={onInviteClick}>
            <UserPlus className="h-4 w-4 mr-2" />
            Invite Member
          </Button>
        </PermissionGate>
      </div>

      <div className="rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Member</TableHead>
              <TableHead>Role</TableHead>
              <TableHead>Joined</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {members.map((member) => {
              const isCurrentUser = member.user_id === currentUserId
              const canRemove = canManage && !isCurrentUser && member.role !== OrgRole.Owner

              return (
                <TableRow key={member.id}>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="flex-shrink-0 w-8 h-8 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
                        <User className="h-4 w-4 text-white" />
                      </div>
                      <div className="min-w-0">
                        <div className="font-medium truncate">
                          {member.user?.display_name || member.user?.username || 'Unknown User'}
                          {isCurrentUser && (
                            <span className="text-muted-foreground ml-2">(You)</span>
                          )}
                        </div>
                        <div className="text-sm text-muted-foreground truncate">
                          {member.user?.email}
                        </div>
                      </div>
                    </div>
                  </TableCell>

                  <TableCell>
                    {!isCurrentUser ? (
                      <RoleManagement
                        member={member}
                        currentUserRole={currentUserRole}
                        onRoleChange={handleRoleChange}
                      />
                    ) : (
                      <Badge variant={getRoleBadgeVariant(member.role)}>
                        <Shield className="h-3 w-3 mr-1" />
                        {getRoleDisplayName(member.role)}
                      </Badge>
                    )}
                  </TableCell>

                  <TableCell>
                    <TooltipProvider>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="text-sm text-muted-foreground cursor-default">
                            {formatRelativeTime(new Date(member.joined_at))}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          {new Date(member.joined_at).toLocaleDateString('en-US', {
                            year: 'numeric',
                            month: 'long',
                            day: 'numeric',
                            hour: '2-digit',
                            minute: '2-digit',
                          })}
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </TableCell>

                  <TableCell className="text-right">
                    {canRemove ? (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setRemovingMemberId(member.id)}
                        aria-label={`Remove ${member.user?.email}`}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    ) : (
                      <TooltipProvider>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <div className="inline-flex">
                              <Button variant="ghost" size="sm" disabled>
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
                          </TooltipTrigger>
                          <TooltipContent>
                            {isCurrentUser
                              ? 'You cannot remove yourself'
                              : member.role === OrgRole.Owner
                                ? 'Cannot remove the owner'
                                : 'Insufficient permissions'}
                          </TooltipContent>
                        </Tooltip>
                      </TooltipProvider>
                    )}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      </div>

      {/* Remove Member Confirmation Dialog */}
      <Dialog open={!!removingMemberId} onOpenChange={() => setRemovingMemberId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove Member</DialogTitle>
            <DialogDescription>
              Are you sure you want to remove{' '}
              <strong>{memberToRemove?.user?.email}</strong> from this organization? They will
              lose access to all organization resources.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRemovingMemberId(null)}>
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRemove}>
              <Trash2 className="h-4 w-4 mr-2" />
              Remove Member
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

// Mobile-friendly version
export function MembersListMobile({
  members,
  currentUserId,
  currentUserRole,
  onRemoveMember,
  onInviteClick,
  loading = false,
  error = null,
  className,
}: MembersListProps) {
  const [_expandedId, setExpandedId] = React.useState<string | null>(null)
  const [removingMemberId, setRemovingMemberId] = React.useState<string | null>(null)

  const canInvite = canRemoveMembers(currentUserRole)

  const handleRemove = async () => {
    if (!removingMemberId) return

    try {
      await onRemoveMember(removingMemberId)
      setRemovingMemberId(null)
      setExpandedId(null)
    } catch (err) {
      console.error('Failed to remove member:', err)
    }
  }

  if (loading && members.length === 0) {
    return (
      <div className={cn('flex items-center justify-center py-8', className)}>
        <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  const memberToRemove = members.find((m) => m.id === removingMemberId)

  return (
    <div className={className}>
      {error && (
        <Alert variant="destructive" className="mb-4">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold">{members.length} Members</h2>
        {canInvite && (
          <Button onClick={onInviteClick} size="sm">
            <UserPlus className="h-4 w-4 mr-2" />
            Invite
          </Button>
        )}
      </div>

      <div className="space-y-2">
        {members.map((member) => {
          const isCurrentUser = member.user_id === currentUserId
          return (
            <div key={member.id} className="rounded-lg border bg-card p-3">
              <div className="flex items-center gap-3">
                <div className="flex-shrink-0 w-10 h-10 rounded-full bg-gradient-to-br from-blue-500 to-purple-500 flex items-center justify-center">
                  <User className="h-5 w-5 text-white" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="font-medium text-sm truncate">
                    {member.user?.display_name || member.user?.username}
                    {isCurrentUser && <span className="text-muted-foreground ml-1">(You)</span>}
                  </div>
                  <div className="text-xs text-muted-foreground truncate">
                    {member.user?.email}
                  </div>
                </div>
                <Badge variant={getRoleBadgeVariant(member.role)} className="text-xs">
                  {getRoleDisplayName(member.role)}
                </Badge>
              </div>
            </div>
          )
        })}
      </div>

      {/* Remove Member Confirmation Dialog */}
      <Dialog open={!!removingMemberId} onOpenChange={() => setRemovingMemberId(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove Member</DialogTitle>
            <DialogDescription>
              Remove <strong>{memberToRemove?.user?.email}</strong>?
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setRemovingMemberId(null)} size="sm">
              Cancel
            </Button>
            <Button variant="destructive" onClick={handleRemove} size="sm">
              Remove
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
