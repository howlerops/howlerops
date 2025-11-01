/**
 * Integration Example: How to Use Sprint 3 Permission Components
 *
 * This file demonstrates how to integrate the new permission system
 * into your application. Copy relevant patterns into your actual components.
 */

import * as React from 'react'
import { useNavigate } from 'react-router-dom'
import { OrganizationSettingsPage } from './OrganizationSettingsPage'
import { InviteMemberModal } from './InviteMemberModal'
import { useOrganizationStore } from '@/store/organization-store'
import { useAuthStore } from '@/store/auth-store'
import type { CreateInvitationInput, AuditLogQueryParams, UpdateOrganizationInput, OrganizationRole } from '@/types/organization'
import { toast } from 'sonner'

/**
 * Example: Full Organization Settings Integration
 *
 * Shows how to wire up all the components with the organization store.
 */
export function OrganizationSettingsExample() {
  const navigate = useNavigate()
  const user = useAuthStore((state) => state.user)

  // Organization store
  const {
    currentOrg: getCurrentOrg,
    currentOrgMembers,
    updateOrganization,
    deleteOrganization,
    updateMemberRole,
    removeMember,
    fetchAuditLogs,
    createInvitation,
    currentOrgInvitations,
    revokeInvitation,
  } = useOrganizationStore()

  // Get the actual organization object
  const currentOrg = getCurrentOrg()

  // State
  const [showInviteModal, setShowInviteModal] = React.useState(false)
  const [loading, setLoading] = React.useState({
    organization: false,
    members: false,
    transfer: false,
  })

  if (!currentOrg || !user) {
    return <div>Loading...</div>
  }

  // Find current user's membership
  const currentMembership = currentOrgMembers.find((m) => m.user_id === user.id)
  const currentUserRole = currentMembership?.role

  if (!currentUserRole) {
    return <div>You are not a member of this organization</div>
  }

  // Handlers
  const handleUpdateOrganization = async (data: UpdateOrganizationInput) => {
    setLoading({ ...loading, organization: true })
    try {
      await updateOrganization(currentOrg.id, data)
      toast.success('Organization updated')
    } catch (error) {
      toast.error('Failed to update organization')
      throw error
    } finally {
      setLoading({ ...loading, organization: false })
    }
  }

  const handleDeleteOrganization = async () => {
    setLoading({ ...loading, organization: true })
    try {
      await deleteOrganization(currentOrg.id)
      toast.success('Organization deleted')
      navigate('/organizations')
    } catch (error) {
      toast.error('Failed to delete organization')
      throw error
    } finally {
      setLoading({ ...loading, organization: false })
    }
  }

  const handleTransferOwnership = async (newOwnerId: string, password: string) => {
    setLoading({ ...loading, transfer: true })
    try {
      // Call your API endpoint for transfer
      // await api.transferOwnership(currentOrg.id, newOwnerId, password)

      toast.success('Ownership transferred')
      navigate('/organizations')
    } catch (error) {
      toast.error('Failed to transfer ownership')
      throw error
    } finally {
      setLoading({ ...loading, transfer: false })
    }
  }

  const handleUpdateMemberRole = async (memberId: string, payload: { role: OrganizationRole }) => {
    setLoading({ ...loading, members: true })
    try {
      // Find member by ID to get user_id
      const member = currentOrgMembers.find((m) => m.id === memberId)
      if (!member) throw new Error('Member not found')

      await updateMemberRole(currentOrg.id, member.user_id, payload.role)
      toast.success('Member role updated')
    } catch (error) {
      toast.error('Failed to update member role')
      throw error
    } finally {
      setLoading({ ...loading, members: false })
    }
  }

  const handleRemoveMember = async (memberId: string) => {
    setLoading({ ...loading, members: true })
    try {
      const member = currentOrgMembers.find((m) => m.id === memberId)
      if (!member) throw new Error('Member not found')

      await removeMember(currentOrg.id, member.user_id)
      toast.success('Member removed')
    } catch (error) {
      toast.error('Failed to remove member')
      throw error
    } finally {
      setLoading({ ...loading, members: false })
    }
  }

  const handleFetchAuditLogs = async (params: AuditLogQueryParams) => {
    return await fetchAuditLogs(currentOrg.id, params)
  }

  const handleInviteMembers = async (invitations: CreateInvitationInput[]) => {
    try {
      // Send invitations
      for (const invitation of invitations) {
        await createInvitation(currentOrg.id, invitation)
      }

      toast.success(`Sent ${invitations.length} invitation(s)`)
      setShowInviteModal(false)
    } catch (error) {
      toast.error('Failed to send invitations')
      throw error
    }
  }

  const handleRevokeInvitation = async (invitationId: string) => {
    try {
      await revokeInvitation(currentOrg.id, invitationId)
      toast.success('Invitation revoked')
    } catch (error) {
      toast.error('Failed to revoke invitation')
      throw error
    }
  }

  return (
    <>
      <OrganizationSettingsPage
        organization={currentOrg}
        members={currentOrgMembers}
        currentUserId={user.id}
        currentUserRole={currentUserRole}
        onUpdateOrganization={handleUpdateOrganization}
        onDeleteOrganization={handleDeleteOrganization}
        onTransferOwnership={handleTransferOwnership}
        onUpdateMemberRole={handleUpdateMemberRole}
        onRemoveMember={handleRemoveMember}
        onFetchAuditLogs={handleFetchAuditLogs}
        onInviteClick={() => setShowInviteModal(true)}
        loading={loading}
      />

      <InviteMemberModal
        open={showInviteModal}
        onOpenChange={setShowInviteModal}
        onInvite={handleInviteMembers}
        pendingInvitations={currentOrgInvitations}
        onRevokeInvitation={handleRevokeInvitation}
        canInviteAdmin={true}
        currentUserRole={currentUserRole}
      />
    </>
  )
}

/**
 * Example: Using Permission Checks in Custom Components
 */
export function CustomComponentWithPermissions() {
  const { hasPermission, canUpdateResource, isOwner } = usePermissions()
  const user = useAuthStore((state) => state.user)

  return (
    <div className="space-y-4">
      {/* Simple permission check */}
      {hasPermission('members:invite') && (
        <Button>Invite Members</Button>
      )}

      {/* Permission gate with tooltip */}
      <PermissionGate permission="org:delete" showTooltip>
        <Button variant="destructive">Delete Organization</Button>
      </PermissionGate>

      {/* Resource ownership check */}
      <ResourceList
        items={resources}
        canEdit={(resource) =>
          canUpdateResource(resource.owner_id, user?.id || '')
        }
      />

      {/* Role-based rendering */}
      <RoleGate minRole="admin">
        <AdminDashboard />
      </RoleGate>

      {/* Owner-only section */}
      {isOwner && (
        <DangerZone />
      )}
    </div>
  )
}

/**
 * Example: Permission-Aware Table Actions
 */
export function ConnectionTable({ connections }: { connections: Connection[] }) {
  const { canUpdateResource, canDeleteResource } = usePermissions()
  const user = useAuthStore((state) => state.user)

  return (
    <Table>
      <TableBody>
        {connections.map((conn) => {
          const canEdit = canUpdateResource(conn.owner_id, user?.id || '')
          const canDelete = canDeleteResource(conn.owner_id, user?.id || '')

          return (
            <TableRow key={conn.id}>
              <TableCell>{conn.name}</TableCell>
              <TableCell className="text-right">
                <PermissionGate
                  permission="connections:update"
                  showTooltip
                >
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => handleEdit(conn)}
                    disabled={!canEdit}
                  >
                    Edit
                  </Button>
                </PermissionGate>

                <PermissionGate
                  permission="connections:delete"
                  showTooltip
                >
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => handleDelete(conn)}
                    disabled={!canDelete}
                  >
                    Delete
                  </Button>
                </PermissionGate>
              </TableCell>
            </TableRow>
          )
        })}
      </TableBody>
    </Table>
  )
}

/**
 * Example: Multi-Permission Check
 */
export function AdvancedSettings() {
  return (
    <MultiPermissionGate
      permissions={['org:update', 'members:update_roles']}
    >
      <Card>
        <CardHeader>
          <CardTitle>Advanced Settings</CardTitle>
          <CardDescription>
            Configure advanced organization features
          </CardDescription>
        </CardHeader>
        <CardContent>
          {/* Advanced settings content */}
        </CardContent>
      </Card>
    </MultiPermissionGate>
  )
}

/**
 * Example: Permission-Based Navigation
 */
export function OrganizationNav() {
  const { hasPermission } = usePermissions()

  return (
    <nav>
      <NavLink to="/org/overview">Overview</NavLink>

      {hasPermission('org:view') && (
        <NavLink to="/org/settings">Settings</NavLink>
      )}

      {hasPermission('members:invite') && (
        <NavLink to="/org/members">Members</NavLink>
      )}

      {hasPermission('audit:view') && (
        <NavLink to="/org/audit">Audit Logs</NavLink>
      )}
    </nav>
  )
}

// Type imports for examples
import { usePermissions } from '@/hooks/usePermissions'
import { PermissionGate, RoleGate, MultiPermissionGate } from '@/components/PermissionGate'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableRow, TableCell } from '@/components/ui/table'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'

// Mock types for example
interface Connection {
  id: string
  name: string
  owner_id: string
}

interface Resource {
  id: string
  owner_id: string
}

function ResourceList({ items, canEdit }: { items: Resource[]; canEdit: (r: Resource) => boolean }) {
  return null
}

function AdminDashboard() {
  return null
}

function DangerZone() {
  return null
}

function NavLink({ to, children }: { to: string; children: React.ReactNode }) {
  return <a href={to}>{children}</a>
}

function handleEdit(conn: Connection) {}
function handleDelete(conn: Connection) {}
const resources: Resource[] = []
