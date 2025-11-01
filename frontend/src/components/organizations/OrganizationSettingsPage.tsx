/**
 * OrganizationSettingsPage Component
 *
 * Comprehensive settings page with tabbed interface:
 * - General: Name, description, info
 * - Members: Team member management
 * - Audit Logs: Activity history (owner/admin only)
 * - Danger Zone: Transfer ownership, delete org (owner only)
 */

import * as React from 'react'
import {
  Users,
  FileText,
  AlertTriangle,
  Settings,
  Crown,
  Trash2,
} from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card } from '@/components/ui/card'
import { OrganizationSettings } from './OrganizationSettings'
import { MembersList } from './MembersList'
import { AuditLogViewer } from './AuditLogViewer'
import { TransferOwnershipModal } from './TransferOwnershipModal'
import { Button } from '@/components/ui/button'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { usePermissions } from '@/hooks/usePermissions'
import { RoleGate } from '@/components/PermissionGate'
import type {
  Organization,
  OrganizationMember,
  UpdateOrganizationInput,
  AuditLog,
  AuditLogQueryParams,
} from '@/types/organization'
import { OrganizationRole } from '@/types/organization'
import { cn } from '@/lib/utils'

interface OrganizationSettingsPageProps {
  organization: Organization
  members: OrganizationMember[]
  currentUserId: string
  currentUserRole: OrganizationRole
  onUpdateOrganization: (data: UpdateOrganizationInput) => Promise<void>
  onDeleteOrganization: () => Promise<void>
  onTransferOwnership: (newOwnerId: string, password: string) => Promise<void>
  onUpdateMemberRole: (memberId: string, data: { role: OrganizationRole }) => Promise<void>
  onRemoveMember: (memberId: string) => Promise<void>
  onFetchAuditLogs: (params: AuditLogQueryParams) => Promise<AuditLog[]>
  onInviteClick: () => void
  loading?: {
    organization?: boolean
    members?: boolean
    transfer?: boolean
  }
  error?: string | null
  className?: string
}

/**
 * OrganizationSettingsPage - Comprehensive settings with tabs
 */
export function OrganizationSettingsPage({
  organization,
  members,
  currentUserId,
  currentUserRole,
  onUpdateOrganization,
  onDeleteOrganization,
  onTransferOwnership,
  onUpdateMemberRole,
  onRemoveMember,
  onFetchAuditLogs,
  onInviteClick,
  loading = {},
  error = null,
  className,
}: OrganizationSettingsPageProps) {
  const [activeTab, setActiveTab] = React.useState('general')
  const [showTransferModal, setShowTransferModal] = React.useState(false)

  const { hasPermission, isOwner } = usePermissions()
  const canViewAudit = hasPermission('audit:view')

  const handleTransferSuccess = () => {
    setShowTransferModal(false)
    // Parent component should handle redirect to org list
  }

  return (
    <div className={cn('space-y-6', className)}>
      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <div>
        <h1 className="text-3xl font-bold tracking-tight mb-2">
          Organization Settings
        </h1>
        <p className="text-muted-foreground">
          Manage your organization, members, and preferences
        </p>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="general" className="flex items-center gap-2">
            <Settings className="h-4 w-4" />
            <span className="hidden sm:inline">General</span>
          </TabsTrigger>
          <TabsTrigger value="members" className="flex items-center gap-2">
            <Users className="h-4 w-4" />
            <span className="hidden sm:inline">Members</span>
          </TabsTrigger>
          <TabsTrigger
            value="audit"
            disabled={!canViewAudit}
            className="flex items-center gap-2"
          >
            <FileText className="h-4 w-4" />
            <span className="hidden sm:inline">Audit Logs</span>
          </TabsTrigger>
          <TabsTrigger
            value="danger"
            disabled={!isOwner}
            className="flex items-center gap-2"
          >
            <AlertTriangle className="h-4 w-4" />
            <span className="hidden sm:inline">Danger Zone</span>
          </TabsTrigger>
        </TabsList>

        {/* General Settings Tab */}
        <TabsContent value="general" className="space-y-6">
          <OrganizationSettings
            organization={organization}
            currentUserRole={currentUserRole}
            onUpdate={onUpdateOrganization}
            onDelete={onDeleteOrganization}
            onNavigateToMembers={() => setActiveTab('members')}
            loading={loading.organization}
            error={error}
          />
        </TabsContent>

        {/* Members Tab */}
        <TabsContent value="members" className="space-y-6">
          <MembersList
            members={members}
            currentUserId={currentUserId}
            currentUserRole={currentUserRole}
            onRoleChange={(memberId, data) => onUpdateMemberRole(memberId, data)}
            onRemoveMember={onRemoveMember}
            onInviteClick={onInviteClick}
            loading={loading.members}
            error={error}
          />
        </TabsContent>

        {/* Audit Logs Tab */}
        <TabsContent value="audit" className="space-y-6">
          <RoleGate minRole="admin">
            <AuditLogViewer
              organizationId={organization.id}
              members={members}
              onFetchLogs={onFetchAuditLogs}
            />
          </RoleGate>
        </TabsContent>

        {/* Danger Zone Tab */}
        <TabsContent value="danger" className="space-y-6">
          <RoleGate minRole="owner">
            <Card>
              <div className="p-6 space-y-6">
                <div>
                  <h2 className="text-2xl font-semibold text-destructive mb-2">
                    Danger Zone
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    Irreversible and destructive actions. Proceed with caution.
                  </p>
                </div>

                {/* Transfer Ownership */}
                <div className="p-4 rounded-lg border border-yellow-500/50 bg-yellow-500/5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Crown className="h-5 w-5 text-yellow-600 dark:text-yellow-500" />
                        <h3 className="font-semibold">Transfer Ownership</h3>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        Transfer complete control of this organization to another admin.
                        You will become an admin after the transfer.
                      </p>
                    </div>
                    <Button
                      variant="outline"
                      onClick={() => setShowTransferModal(true)}
                      className="flex-shrink-0"
                    >
                      <Crown className="h-4 w-4 mr-2" />
                      Transfer
                    </Button>
                  </div>
                </div>

                {/* Delete Organization */}
                <div className="p-4 rounded-lg border border-destructive/50 bg-destructive/5">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Trash2 className="h-5 w-5 text-destructive" />
                        <h3 className="font-semibold text-destructive">
                          Delete Organization
                        </h3>
                      </div>
                      <p className="text-sm text-muted-foreground">
                        Permanently delete this organization and all associated data
                        including members, connections, and queries. This action cannot
                        be undone.
                      </p>
                    </div>
                    <Button
                      variant="destructive"
                      onClick={onDeleteOrganization}
                      disabled={loading.organization}
                      className="flex-shrink-0"
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Delete
                    </Button>
                  </div>
                </div>

                {/* Warning */}
                <Alert variant="destructive">
                  <AlertTriangle className="h-4 w-4" />
                  <AlertDescription>
                    <strong>Warning:</strong> Actions in this section are permanent
                    and cannot be reversed. Make sure you understand the consequences
                    before proceeding.
                  </AlertDescription>
                </Alert>
              </div>
            </Card>
          </RoleGate>
        </TabsContent>
      </Tabs>

      {/* Transfer Ownership Modal */}
      <TransferOwnershipModal
        open={showTransferModal}
        onOpenChange={setShowTransferModal}
        organizationName={organization.name}
        members={members}
        onTransfer={onTransferOwnership}
        onSuccess={handleTransferSuccess}
        loading={loading.transfer}
        error={error}
      />
    </div>
  )
}
